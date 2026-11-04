package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"k8s-dashboard/internal/k8s"
	"k8s-dashboard/internal/models"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetNodesHandler handles /api/nodes
func GetNodesHandler(w http.ResponseWriter, r *http.Request) {
	client, err := k8s.GetManager().GetClient()
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Also count pods per node
	pods, _ := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	podCountMap := make(map[string]int)
	if pods != nil {
		for _, pod := range pods.Items {
			if pod.Spec.NodeName != "" {
				podCountMap[pod.Spec.NodeName]++
			}
		}
	}

	result := make([]models.NodeInfo, 0, len(nodes.Items))
	for _, node := range nodes.Items {
		info := models.NodeInfo{
			Name:             node.Name,
			KubeletVersion:   node.Status.NodeInfo.KubeletVersion,
			OSImage:          node.Status.NodeInfo.OSImage,
			KernelVersion:    node.Status.NodeInfo.KernelVersion,
			ContainerRuntime: node.Status.NodeInfo.ContainerRuntimeVersion,
			CPUCapacity:      node.Status.Capacity.Cpu().String(),
			MemoryCapacity:   node.Status.Capacity.Memory().String(),
			PodsCount:        podCountMap[node.Name],
			Unschedulable:    node.Spec.Unschedulable,
			Age:              k8s.FormatAge(node.CreationTimestamp),
			Conditions:       make(map[string]string),
		}

		// Roles
		var roles []string
		for label := range node.Labels {
			if strings.HasPrefix(label, "node-role.kubernetes.io/") {
				role := strings.TrimPrefix(label, "node-role.kubernetes.io/")
				if role != "" {
					roles = append(roles, role)
				}
			}
		}
		if len(roles) == 0 {
			roles = []string{"worker"}
		}
		info.Roles = roles

		// Addresses
		for _, addr := range node.Status.Addresses {
			if addr.Type == "InternalIP" {
				info.InternalIP = addr.Address
			} else if addr.Type == "ExternalIP" {
				info.ExternalIP = addr.Address
			}
		}

		// Conditions & Status
		info.Status = "NotReady"
		for _, cond := range node.Status.Conditions {
			info.Conditions[string(cond.Type)] = string(cond.Status)
			if cond.Type == "Ready" {
				if cond.Status == "True" {
					info.Status = "Ready"
				}
			}
		}
		if node.Spec.Unschedulable {
			info.Status += ",SchedulingDisabled"
		}

		result = append(result, info)
	}

	WriteJSON(w, http.StatusOK, result)
}

// CordonNodeHandler handles /api/nodes/cordon
func CordonNodeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		NodeName      string `json:"nodeName"`
		Unschedulable bool   `json:"unschedulable"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.NodeName == "" {
		WriteError(w, http.StatusBadRequest, "Invalid request body: nodeName required")
		return
	}

	client, err := k8s.GetManager().GetClient()
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	node, err := client.CoreV1().Nodes().Get(ctx, req.NodeName, metav1.GetOptions{})
	if err != nil {
		WriteError(w, http.StatusNotFound, err.Error())
		return
	}

	node.Spec.Unschedulable = req.Unschedulable
	_, err = client.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	action := "cordoned"
	if !req.Unschedulable {
		action = "uncordoned"
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Node successfully " + action,
	})
}

