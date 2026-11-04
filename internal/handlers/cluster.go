package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"k8s-dashboard/internal/k8s"
	"k8s-dashboard/internal/models"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WriteJSON sends a JSON response with status code
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// WriteError sends a JSON error response
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]string{"error": message})
}

// GetStatusHandler handles /api/status
func GetStatusHandler(w http.ResponseWriter, r *http.Request) {
	mgr := k8s.GetManager()
	WriteJSON(w, http.StatusOK, mgr.GetStatus())
}

// ConnectHandler handles /api/connect
func ConnectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req models.ConnectionConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	mgr := k8s.GetManager()
	var err error

	switch req.Mode {
	case "token":
		if req.Server == "" || req.Token == "" {
			WriteError(w, http.StatusBadRequest, "Server URL and Token are required")
			return
		}
		err = mgr.ConnectWithToken(req.Server, req.Token, req.Insecure)
	case "kubeconfig":
		if req.KubeconfigContent == "" {
			WriteError(w, http.StatusBadRequest, "Kubeconfig content is required")
			return
		}
		err = mgr.ConnectWithKubeconfigRaw(req.KubeconfigContent)
	case "auto":
		err = mgr.InitDefault("")
	default:
		WriteError(w, http.StatusBadRequest, "Invalid connection mode")
		return
	}

	if err != nil {
		WriteError(w, http.StatusBadGateway, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"status":  mgr.GetStatus(),
	})
}

// GetOverviewHandler handles /api/overview
func GetOverviewHandler(w http.ResponseWriter, r *http.Request) {
	mgr := k8s.GetManager()
	client, err := mgr.GetClient()
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	overview := models.ClusterOverview{
		ControlPlaneHealth: make(map[string]string),
	}

	status := mgr.GetStatus()
	overview.KubernetesVersion = fmt.Sprintf("%v", status["version"])
	overview.APIServerURL = fmt.Sprintf("%v", status["serverURL"])
	overview.ConnectedVia = fmt.Sprintf("%v", status["connectedVia"])

	// 1. Fetch Nodes
	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err == nil {
		overview.TotalNodes = len(nodes.Items)
		var totalCPU, totalMem, allocCPU, allocMem int64
		for _, node := range nodes.Items {
			for _, cond := range node.Status.Conditions {
				if cond.Type == "Ready" && cond.Status == "True" {
					overview.ReadyNodes++
				}
			}
			totalCPU += node.Status.Capacity.Cpu().MilliValue()
			totalMem += node.Status.Capacity.Memory().Value()
			allocCPU += node.Status.Allocatable.Cpu().MilliValue()
			allocMem += node.Status.Allocatable.Memory().Value()
		}
		overview.CPUCapacity = fmt.Sprintf("%.2f Cores", float64(totalCPU)/1000.0)
		overview.MemoryCapacity = fmt.Sprintf("%.2f GiB", float64(totalMem)/(1024*1024*1024))
		overview.CPUAllocatable = fmt.Sprintf("%.2f Cores", float64(allocCPU)/1000.0)
		overview.MemoryAllocatable = fmt.Sprintf("%.2f GiB", float64(allocMem)/(1024*1024*1024))
	}

	// 2. Fetch Pods
	pods, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err == nil {
		overview.TotalPods = len(pods.Items)
		for _, pod := range pods.Items {
			switch pod.Status.Phase {
			case "Running":
				overview.RunningPods++
			case "Pending":
				overview.PendingPods++
			case "Failed":
				overview.FailedPods++
			}
		}
	}

	// 3. Fetch Namespaces
	namespaces, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err == nil {
		overview.TotalNamespaces = len(namespaces.Items)
	}

	// 4. Fetch Deployments
	deployments, err := client.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err == nil {
		overview.TotalDeployments = len(deployments.Items)
	}

	// 5. Fetch Services
	services, err := client.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if err == nil {
		overview.TotalServices = len(services.Items)
	}

	// 6. Fetch Warning Events count
	events, err := client.CoreV1().Events("").List(ctx, metav1.ListOptions{
		FieldSelector: "type=Warning",
	})
	if err == nil {
		overview.WarningEventsCount = len(events.Items)
	}

	WriteJSON(w, http.StatusOK, overview)
}

// GetNamespacesHandler handles /api/namespaces
func GetNamespacesHandler(w http.ResponseWriter, r *http.Request) {
	client, err := k8s.GetManager().GetClient()
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	list, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]models.NamespaceInfo, 0, len(list.Items))
	for _, ns := range list.Items {
		result = append(result, models.NamespaceInfo{
			Name:   ns.Name,
			Status: string(ns.Status.Phase),
			Age:    k8s.FormatAge(ns.CreationTimestamp),
		})
	}

	WriteJSON(w, http.StatusOK, result)
}

