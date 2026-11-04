package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"k8s-dashboard/internal/handlers"
	"k8s-dashboard/internal/k8s"
	"k8s-dashboard/web"
)

func main() {
	port := flag.String("port", "8080", "Port to listen on")
	host := flag.String("host", "0.0.0.0", "Host address to bind to")
	kubeconfig := flag.String("kubeconfig", "", "Path to kubeconfig file")
	flag.Parse()

	// Check environment variable PORT if present
	if envPort := os.Getenv("PORT"); envPort != "" {
		*port = envPort
	}

	log.Println("==================================================")
	log.Println("   KUBERNETES MASTER CONTROL DASHBOARD (GO)")
	log.Println("==================================================")

	// Attempt initial Kubernetes connection
	mgr := k8s.GetManager()
	if err := mgr.InitDefault(*kubeconfig); err != nil {
		log.Printf("[INFO] Belum terhubung otomatis ke cluster: %v", err)
		log.Println("[INFO] Anda dapat menghubungkan cluster melalui halaman UI / Settings.")
	} else {
		status := mgr.GetStatus()
		log.Printf("[SUCCESS] Terhubung ke Kubernetes Cluster %s via %s", status["version"], status["connectedVia"])
	}

	mux := http.NewServeMux()

	// API Routes
	mux.HandleFunc("/api/status", handlers.GetStatusHandler)
	mux.HandleFunc("/api/connect", handlers.ConnectHandler)
	mux.HandleFunc("/api/overview", handlers.GetOverviewHandler)
	mux.HandleFunc("/api/namespaces", handlers.GetNamespacesHandler)
	mux.HandleFunc("/api/nodes", handlers.GetNodesHandler)
	mux.HandleFunc("/api/nodes/cordon", handlers.CordonNodeHandler)
	mux.HandleFunc("/api/pods", handlers.GetPodsHandler)
	mux.HandleFunc("/api/pods/logs", handlers.GetPodLogsHandler)
	mux.HandleFunc("/api/pods/delete", handlers.DeletePodHandler)
	mux.HandleFunc("/api/deployments", handlers.GetDeploymentsHandler)
	mux.HandleFunc("/api/deployments/scale", handlers.ScaleDeploymentHandler)
	mux.HandleFunc("/api/deployments/restart", handlers.RestartDeploymentHandler)
	mux.HandleFunc("/api/daemonsets", handlers.GetDaemonSetsHandler)
	mux.HandleFunc("/api/statefulsets", handlers.GetStatefulSetsHandler)
	mux.HandleFunc("/api/services", handlers.GetServicesHandler)
	mux.HandleFunc("/api/ingresses", handlers.GetIngressesHandler)
	mux.HandleFunc("/api/configmaps", handlers.GetConfigMapsHandler)
	mux.HandleFunc("/api/secrets", handlers.GetSecretsHandler)
	mux.HandleFunc("/api/storage/pv", handlers.GetPersistentVolumesHandler)
	mux.HandleFunc("/api/storage/pvc", handlers.GetPersistentVolumeClaimsHandler)
	mux.HandleFunc("/api/events", handlers.GetEventsHandler)

	// AI Copilot Routes
	mux.HandleFunc("/api/ai/chat", handlers.AIChatHandler)
	mux.HandleFunc("/api/ai/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handlers.SaveAIConfigHandler(w, r)
		} else {
			handlers.GetAIConfigHandler(w, r)
		}
	})

	// Static assets & SPA fallback
	fileServer := http.FileServer(web.GetFileSystem())
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// If requesting an API that doesn't exist, return 404 JSON
		if strings.HasPrefix(r.URL.Path, "/api/") {
			handlers.WriteError(w, http.StatusNotFound, "API endpoint not found")
			return
		}
		fileServer.ServeHTTP(w, r)
	})

	// CORS and Logger Middleware
	handler := loggingAndCORSMiddleware(mux)

	addr := fmt.Sprintf("%s:%s", *host, *port)
	log.Printf("[INFO] Dashboard UI siap diakses di: http://%s (atau http://localhost:%s)", addr, *port)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server gagal berjalan: %v", err)
	}
}

func loggingAndCORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

