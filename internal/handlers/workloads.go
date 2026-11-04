package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"k8s-dashboard/internal/k8s"
	"k8s-dashboard/internal/models"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetPodsHandler handles /api/pods?namespace=...
func GetPodsHandler(w http.ResponseWriter, r *http.Request) {
	client, err := k8s.GetManager().GetClient()
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	ns := r.URL.Query().Get("namespace")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	pods, err := client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]models.PodInfo, 0, len(pods.Items))
	for _, p := range pods.Items {
		var restarts int32
		var readyContainers int
		var containers []string

		for _, cs := range p.Status.ContainerStatuses {
			restarts += cs.RestartCount
			if cs.Ready {
				readyContainers++
			}
			containers = append(containers, cs.Name)
		}

		status := string(p.Status.Phase)
		// Check for detailed container waiting reasons like CrashLoopBackOff or ErrImagePull
		for _, cs := range p.Status.ContainerStatuses {
			if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
				status = cs.State.Waiting.Reason
				break
			}
		}

		result = append(result, models.PodInfo{
			Name:         p.Name,
			Namespace:    p.Namespace,
			Status:       status,
			Ready:        fmt.Sprintf("%d/%d", readyContainers, len(p.Spec.Containers)),
			Restarts:     restarts,
			Node:         p.Spec.NodeName,
			IP:           p.Status.PodIP,
			Containers:   containers,
			Age:          k8s.FormatAge(p.CreationTimestamp),
			CreationTime: p.CreationTimestamp.Format(time.RFC3339),
		})
	}

	WriteJSON(w, http.StatusOK, result)
}

// GetPodLogsHandler handles /api/pods/logs?namespace=...&name=...&container=...&tail=...
func GetPodLogsHandler(w http.ResponseWriter, r *http.Request) {
	client, err := k8s.GetManager().GetClient()
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	ns := r.URL.Query().Get("namespace")
	name := r.URL.Query().Get("name")
	container := r.URL.Query().Get("container")
	tailStr := r.URL.Query().Get("tail")

	if ns == "" || name == "" {
		WriteError(w, http.StatusBadRequest, "Missing namespace or name query parameter")
		return
	}

	var tailLines int64 = 100
	if tailStr != "" {
		if t, err := strconv.ParseInt(tailStr, 10, 64); err == nil && t > 0 {
			tailLines = t
		}
	}

	opts := &corev1.PodLogOptions{
		TailLines: &tailLines,
	}
	if container != "" {
		opts.Container = container
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := client.CoreV1().Pods(ns).GetLogs(name, opts)
	podLogs, err := req.Stream(ctx)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get logs: %v", err))
		return
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, fmt.Sprintf("Failed reading logs: %v", err))
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{
		"pod":       name,
		"namespace": ns,
		"logs":      buf.String(),
	})
}

// DeletePodHandler handles DELETE /api/pods?namespace=...&name=...
func DeletePodHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete && r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	ns := r.URL.Query().Get("namespace")
	name := r.URL.Query().Get("name")
	if ns == "" || name == "" {
		WriteError(w, http.StatusBadRequest, "Missing namespace or name parameter")
		return
	}

	client, err := k8s.GetManager().GetClient()
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	err = client.CoreV1().Pods(ns).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Pod %s/%s deleted", ns, name),
	})
}

// GetDeploymentsHandler handles /api/deployments?namespace=...
func GetDeploymentsHandler(w http.ResponseWriter, r *http.Request) {
	client, err := k8s.GetManager().GetClient()
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	ns := r.URL.Query().Get("namespace")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	deployments, err := client.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]models.DeploymentInfo, 0, len(deployments.Items))
	for _, d := range deployments.Items {
		var images []string
		for _, c := range d.Spec.Template.Spec.Containers {
			images = append(images, c.Image)
		}

		var replicas int32
		if d.Spec.Replicas != nil {
			replicas = *d.Spec.Replicas
		}

		result = append(result, models.DeploymentInfo{
			Name:              d.Name,
			Namespace:         d.Namespace,
			Replicas:          replicas,
			ReadyReplicas:     d.Status.ReadyReplicas,
			UpdatedReplicas:   d.Status.UpdatedReplicas,
			AvailableReplicas: d.Status.AvailableReplicas,
			Images:            images,
			Age:               k8s.FormatAge(d.CreationTimestamp),
		})
	}

	WriteJSON(w, http.StatusOK, result)
}

// ScaleDeploymentHandler handles POST /api/deployments/scale
func ScaleDeploymentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Namespace string `json:"namespace"`
		Name      string `json:"name"`
		Replicas  int32  `json:"replicas"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	client, err := k8s.GetManager().GetClient()
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	scale, err := client.AppsV1().Deployments(req.Namespace).GetScale(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		WriteError(w, http.StatusNotFound, err.Error())
		return
	}

	scale.Spec.Replicas = req.Replicas
	_, err = client.AppsV1().Deployments(req.Namespace).UpdateScale(ctx, req.Name, scale, metav1.UpdateOptions{})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success":  true,
		"message":  fmt.Sprintf("Deployment %s scaled to %d replicas", req.Name, req.Replicas),
		"replicas": req.Replicas,
	})
}

// RestartDeploymentHandler handles POST /api/deployments/restart
func RestartDeploymentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Namespace string `json:"namespace"`
		Name      string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	client, err := k8s.GetManager().GetClient()
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	deployment, err := client.AppsV1().Deployments(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		WriteError(w, http.StatusNotFound, err.Error())
		return
	}

	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = make(map[string]string)
	}
	deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	_, err = client.AppsV1().Deployments(req.Namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Deployment %s rollout restart initiated", req.Name),
	})
}

// GetDaemonSetsHandler handles /api/daemonsets?namespace=...
func GetDaemonSetsHandler(w http.ResponseWriter, r *http.Request) {
	client, err := k8s.GetManager().GetClient()
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	ns := r.URL.Query().Get("namespace")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	dsList, err := client.AppsV1().DaemonSets(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]models.DaemonSetInfo, 0, len(dsList.Items))
	for _, ds := range dsList.Items {
		var images []string
		for _, c := range ds.Spec.Template.Spec.Containers {
			images = append(images, c.Image)
		}
		result = append(result, models.DaemonSetInfo{
			Name:            ds.Name,
			Namespace:       ds.Namespace,
			DesiredNumber:   ds.Status.DesiredNumberScheduled,
			CurrentNumber:   ds.Status.CurrentNumberScheduled,
			ReadyNumber:     ds.Status.NumberReady,
			AvailableNumber: ds.Status.NumberAvailable,
			Images:          images,
			Age:             k8s.FormatAge(ds.CreationTimestamp),
		})
	}

	WriteJSON(w, http.StatusOK, result)
}

// GetStatefulSetsHandler handles /api/statefulsets?namespace=...
func GetStatefulSetsHandler(w http.ResponseWriter, r *http.Request) {
	client, err := k8s.GetManager().GetClient()
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	ns := r.URL.Query().Get("namespace")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	stsList, err := client.AppsV1().StatefulSets(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]models.StatefulSetInfo, 0, len(stsList.Items))
	for _, sts := range stsList.Items {
		var images []string
		for _, c := range sts.Spec.Template.Spec.Containers {
			images = append(images, c.Image)
		}
		var replicas int32
		if sts.Spec.Replicas != nil {
			replicas = *sts.Spec.Replicas
		}
		result = append(result, models.StatefulSetInfo{
			Name:          sts.Name,
			Namespace:     sts.Namespace,
			Replicas:      replicas,
			ReadyReplicas: sts.Status.ReadyReplicas,
			Images:        images,
			Age:           k8s.FormatAge(sts.CreationTimestamp),
		})
	}

	WriteJSON(w, http.StatusOK, result)
}

