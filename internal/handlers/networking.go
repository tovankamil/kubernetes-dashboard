package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"k8s-dashboard/internal/k8s"
	"k8s-dashboard/internal/models"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetServicesHandler handles /api/services?namespace=...
func GetServicesHandler(w http.ResponseWriter, r *http.Request) {
	client, err := k8s.GetManager().GetClient()
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	ns := r.URL.Query().Get("namespace")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	services, err := client.CoreV1().Services(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]models.ServiceInfo, 0, len(services.Items))
	for _, s := range services.Items {
		var ports []string
		for _, p := range s.Spec.Ports {
			if p.NodePort != 0 {
				ports = append(ports, fmt.Sprintf("%d:%d/%s", p.Port, p.NodePort, p.Protocol))
			} else {
				ports = append(ports, fmt.Sprintf("%d/%s", p.Port, p.Protocol))
			}
		}

		var extIP string
		if len(s.Status.LoadBalancer.Ingress) > 0 {
			if s.Status.LoadBalancer.Ingress[0].IP != "" {
				extIP = s.Status.LoadBalancer.Ingress[0].IP
			} else {
				extIP = s.Status.LoadBalancer.Ingress[0].Hostname
			}
		}
		if extIP == "" && len(s.Spec.ExternalIPs) > 0 {
			extIP = strings.Join(s.Spec.ExternalIPs, ",")
		}
		if extIP == "" {
			extIP = "<none>"
		}

		result = append(result, models.ServiceInfo{
			Name:       s.Name,
			Namespace:  s.Namespace,
			Type:       string(s.Spec.Type),
			ClusterIP:  s.Spec.ClusterIP,
			ExternalIP: extIP,
			Ports:      ports,
			Selector:   s.Spec.Selector,
			Age:        k8s.FormatAge(s.CreationTimestamp),
		})
	}

	WriteJSON(w, http.StatusOK, result)
}

// GetIngressesHandler handles /api/ingresses?namespace=...
func GetIngressesHandler(w http.ResponseWriter, r *http.Request) {
	client, err := k8s.GetManager().GetClient()
	if err != nil {
		WriteError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	ns := r.URL.Query().Get("namespace")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	ingresses, err := client.NetworkingV1().Ingresses(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make([]models.IngressInfo, 0, len(ingresses.Items))
	for _, ing := range ingresses.Items {
		var hosts []string
		var endpoints []string

		for _, rule := range ing.Spec.Rules {
			if rule.Host != "" {
				hosts = append(hosts, rule.Host)
			}
			if rule.HTTP != nil {
				for _, path := range rule.HTTP.Paths {
					endpoints = append(endpoints, path.Path)
				}
			}
		}

		className := "<none>"
		if ing.Spec.IngressClassName != nil {
			className = *ing.Spec.IngressClassName
		}

		result = append(result, models.IngressInfo{
			Name:      ing.Name,
			Namespace: ing.Namespace,
			Class:     className,
			Hosts:     hosts,
			Endpoints: endpoints,
			Age:       k8s.FormatAge(ing.CreationTimestamp),
		})
	}

	WriteJSON(w, http.StatusOK, result)
}

