package handlers

import (
	"context"
	"net/http"
	"sort"
	"time"

	"k8s-dashboard/internal/k8s"
	"k8s-dashboard/internal/models"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetEventsHandler handles /api/events?namespace=...
func GetEventsHandler(w http.ResponseWriter, r *http.Request) {
	client, err := k8s.GetManager().GetClient()
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	ns := r.URL.Query().Get("namespace")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	events, err := client.CoreV1().Events(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]models.EventInfo, 0, len(events.Items))
	for _, ev := range events.Items {
		lastTime := ev.LastTimestamp.Time
		if lastTime.IsZero() {
			lastTime = ev.EventTime.Time
		}
		if lastTime.IsZero() {
			lastTime = ev.CreationTimestamp.Time
		}

		result = append(result, models.EventInfo{
			Type:          ev.Type,
			Reason:        ev.Reason,
			Object:        fmtObject(ev.InvolvedObject.Kind, ev.InvolvedObject.Name),
			Message:       ev.Message,
			Namespace:     ev.Namespace,
			Count:         ev.Count,
			Age:           k8s.FormatAge(ev.LastTimestamp),
			LastTimestamp: lastTime,
		})
	}

	// Sort events descending by LastTimestamp
	sort.Slice(result, func(i, j int) bool {
		return result[i].LastTimestamp.After(result[j].LastTimestamp)
	})

	// Limit to latest 100 events
	if len(result) > 100 {
		result = result[:100]
	}

	WriteJSON(w, http.StatusOK, result)
}

func fmtObject(kind, name string) string {
	if kind == "" {
		return name
	}
	if name == "" {
		return kind
	}
	return kind + "/" + name
}

