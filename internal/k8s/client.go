package k8s

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// Manager manages Kubernetes client connection and state
type Manager struct {
	mu           sync.RWMutex
	client       *kubernetes.Clientset
	restConfig   *rest.Config
	connected    bool
	connectedVia string
	serverURL    string
	version      string
	lastError    string
}

var (
	defaultManager *Manager
	once           sync.Once
)

// GetManager returns the singleton manager instance
func GetManager() *Manager {
	once.Do(func() {
		defaultManager = &Manager{}
	})
	return defaultManager
}

// InitDefault attempts to auto-detect and connect to a Kubernetes cluster
func (m *Manager) InitDefault(customKubeconfigPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var config *rest.Config
	var err error
	var via string

	// 1. Try In-Cluster (if running in a pod)
	config, err = rest.InClusterConfig()
	if err == nil {
		via = "In-Cluster ServiceAccount"
	} else {
		// 2. Try custom provided kubeconfig path
		var kubeconfig string
		if customKubeconfigPath != "" {
			kubeconfig = customKubeconfigPath
			via = fmt.Sprintf("Flag Kubeconfig (%s)", customKubeconfigPath)
		} else if envPath := os.Getenv("KUBECONFIG"); envPath != "" {
			kubeconfig = envPath
			via = fmt.Sprintf("KUBECONFIG Env (%s)", envPath)
		} else {
			// Check standard user home .kube/config
			if home := homedir.HomeDir(); home != "" {
				candidate := filepath.Join(home, ".kube", "config")
				if _, statErr := os.Stat(candidate); statErr == nil {
					kubeconfig = candidate
					via = candidate
				}
			}
			// If on Linux master node, also check /etc/kubernetes/admin.conf
			if kubeconfig == "" {
				adminConf := "/etc/kubernetes/admin.conf"
				if _, statErr := os.Stat(adminConf); statErr == nil {
					kubeconfig = adminConf
					via = adminConf
				}
			}
		}

		if kubeconfig != "" {
			config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		} else {
			err = fmt.Errorf("no kubeconfig found in standard paths")
		}
	}

	if err != nil {
		m.connected = false
		m.lastError = fmt.Sprintf("Koneksi gagal: %v", err)
		return err
	}

	return m.applyConfigLocked(config, via)
}

// ConnectWithKubeconfigRaw connects using raw kubeconfig yaml string
func (m *Manager) ConnectWithKubeconfigRaw(content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	clientConfig, err := clientcmd.NewClientConfigFromBytes([]byte(content))
	if err != nil {
		m.connected = false
		m.lastError = fmt.Sprintf("Format kubeconfig invalid: %v", err)
		return err
	}

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		m.connected = false
		m.lastError = fmt.Sprintf("Gagal parse rest config: %v", err)
		return err
	}

	return m.applyConfigLocked(restConfig, "Uploaded Kubeconfig")
}

// ConnectWithToken connects using Server URL + Bearer Token
func (m *Manager) ConnectWithToken(serverURL, token string, insecure bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	restConfig := &rest.Config{
		Host:        serverURL,
		BearerToken: token,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: insecure,
		},
		Timeout: 10 * time.Second,
	}

	return m.applyConfigLocked(restConfig, fmt.Sprintf("Token Auth (%s)", serverURL))
}

// applyConfigLocked creates clientset and validates connection
func (m *Manager) applyConfigLocked(config *rest.Config, via string) error {
	// Set reasonable timeout
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		m.connected = false
		m.lastError = fmt.Sprintf("Gagal membuat clientset: %v", err)
		return err
	}

	// Ping Kubernetes API server to verify connection
	v, err := clientset.Discovery().ServerVersion()
	if err != nil {
		m.connected = false
		m.client = clientset
		m.restConfig = config
		m.serverURL = config.Host
		m.lastError = fmt.Sprintf("Terhubung tapi API Server tidak merespon: %v", err)
		return err
	}

	m.client = clientset
	m.restConfig = config
	m.connected = true
	m.connectedVia = via
	m.serverURL = config.Host
	m.version = v.GitVersion
	m.lastError = ""

	return nil
}

// GetClient returns current clientset and error if not connected
func (m *Manager) GetClient() (*kubernetes.Clientset, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.connected || m.client == nil {
		if m.lastError != "" {
			return nil, fmt.Errorf("cluster not connected: %s", m.lastError)
		}
		return nil, fmt.Errorf("cluster not connected (please configure kubeconfig or token)")
	}
	return m.client, nil
}

// GetStatus returns the current connection status
func (m *Manager) GetStatus() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"connected":    m.connected,
		"connectedVia": m.connectedVia,
		"serverURL":    m.serverURL,
		"version":      m.version,
		"error":        m.lastError,
	}
}

// FormatAge converts timestamp to readable string (e.g. 2d, 4h, 15m)
func FormatAge(t metav1.Time) string {
	if t.IsZero() {
		return "-"
	}
	d := time.Since(t.Time)
	if d.Hours() >= 24 {
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%dd", days)
	}
	if d.Hours() >= 1 {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	if d.Minutes() >= 1 {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%ds", int(d.Seconds()))
}
