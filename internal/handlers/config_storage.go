package handlers

import (
	"context"
	"net/http"
	"time"

	"k8s-dashboard/internal/k8s"
	"k8s-dashboard/internal/models"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetConfigMapsHandler handles /api/configmaps?namespace=...
func GetConfigMapsHandler(w http.ResponseWriter, r *http.Request) {
	client, err := k8s.GetManager().GetClient()
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	ns := r.URL.Query().Get("namespace")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	cms, err := client.CoreV1().ConfigMaps(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]models.ConfigResourceInfo, 0, len(cms.Items))
	for _, cm := range cms.Items {
		var keys []string
		for k := range cm.Data {
			keys = append(keys, k)
		}
		for k := range cm.BinaryData {
			keys = append(keys, k)
		}
		result = append(result, models.ConfigResourceInfo{
			Name:      cm.Name,
			Namespace: cm.Namespace,
			Type:      "ConfigMap",
			KeysCount: len(keys),
			Keys:      keys,
			Age:       k8s.FormatAge(cm.CreationTimestamp),
		})
	}

	WriteJSON(w, http.StatusOK, result)
}

// GetSecretsHandler handles /api/secrets?namespace=...
func GetSecretsHandler(w http.ResponseWriter, r *http.Request) {
	client, err := k8s.GetManager().GetClient()
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	ns := r.URL.Query().Get("namespace")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	secrets, err := client.CoreV1().Secrets(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]models.ConfigResourceInfo, 0, len(secrets.Items))
	for _, sec := range secrets.Items {
		var keys []string
		for k := range sec.Data {
			keys = append(keys, k)
		}
		result = append(result, models.ConfigResourceInfo{
			Name:      sec.Name,
			Namespace: sec.Namespace,
			Type:      string(sec.Type),
			KeysCount: len(keys),
			Keys:      keys,
			Age:       k8s.FormatAge(sec.CreationTimestamp),
		})
	}

	WriteJSON(w, http.StatusOK, result)
}

// GetPersistentVolumesHandler handles /api/storage/pv
func GetPersistentVolumesHandler(w http.ResponseWriter, r *http.Request) {
	client, err := k8s.GetManager().GetClient()
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	pvs, err := client.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]models.StorageInfo, 0, len(pvs.Items))
	for _, pv := range pvs.Items {
		var accessModes []string
		for _, am := range pv.Spec.AccessModes {
			accessModes = append(accessModes, string(am))
		}

		cap := pv.Spec.Capacity.Storage().String()
		result = append(result, models.StorageInfo{
			Name:         pv.Name,
			Namespace:    "-",
			Status:       string(pv.Status.Phase),
			Capacity:     cap,
			AccessModes:  accessModes,
			StorageClass: pv.Spec.StorageClassName,
			Age:          k8s.FormatAge(pv.CreationTimestamp),
		})
	}

	WriteJSON(w, http.StatusOK, result)
}

// GetPersistentVolumeClaimsHandler handles /api/storage/pvc?namespace=...
func GetPersistentVolumeClaimsHandler(w http.ResponseWriter, r *http.Request) {
	client, err := k8s.GetManager().GetClient()
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	ns := r.URL.Query().Get("namespace")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	pvcs, err := client.CoreV1().PersistentVolumeClaims(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]models.StorageInfo, 0, len(pvcs.Items))
	for _, pvc := range pvcs.Items {
		var accessModes []string
		for _, am := range pvc.Spec.AccessModes {
			accessModes = append(accessModes, string(am))
		}

		cap := pvc.Status.Capacity.Storage().String()
		sc := ""
		if pvc.Spec.StorageClassName != nil {
			sc = *pvc.Spec.StorageClassName
		}

		result = append(result, models.StorageInfo{
			Name:         pvc.Name,
			Namespace:    pvc.Namespace,
			Status:       string(pvc.Status.Phase),
			Volume:       pvc.Spec.VolumeName,
			Capacity:     cap,
			AccessModes:  accessModes,
			StorageClass: sc,
			Age:          k8s.FormatAge(pvc.CreationTimestamp),
		})
	}

	WriteJSON(w, http.StatusOK, result)
}

