package models

import "time"

// ClusterOverview contains summary metrics of the cluster
type ClusterOverview struct {
	KubernetesVersion  string            `json:"kubernetesVersion"`
	APIServerURL       string            `json:"apiServerUrl"`
	TotalNodes         int               `json:"totalNodes"`
	ReadyNodes         int               `json:"readyNodes"`
	TotalPods          int               `json:"totalPods"`
	RunningPods        int               `json:"runningPods"`
	PendingPods        int               `json:"pendingPods"`
	FailedPods         int               `json:"failedPods"`
	TotalNamespaces    int               `json:"totalNamespaces"`
	TotalDeployments   int               `json:"totalDeployments"`
	TotalServices      int               `json:"totalServices"`
	WarningEventsCount int               `json:"warningEventsCount"`
	CPUCapacity        string            `json:"cpuCapacity"`
	MemoryCapacity     string            `json:"memoryCapacity"`
	CPUAllocatable     string            `json:"cpuAllocatable"`
	MemoryAllocatable  string            `json:"memoryAllocatable"`
	ConnectedVia       string            `json:"connectedVia"`
	ControlPlaneHealth map[string]string `json:"controlPlaneHealth"`
}

// NodeInfo represents a Kubernetes node
type NodeInfo struct {
	Name             string            `json:"name"`
	Status           string            `json:"status"` // Ready, NotReady, SchedulingDisabled
	Roles            []string          `json:"roles"`
	InternalIP       string            `json:"internalIP"`
	ExternalIP       string            `json:"externalIP"`
	KubeletVersion   string            `json:"kubeletVersion"`
	OSImage          string            `json:"osImage"`
	KernelVersion    string            `json:"kernelVersion"`
	ContainerRuntime string            `json:"containerRuntime"`
	CPUCapacity      string            `json:"cpuCapacity"`
	MemoryCapacity   string            `json:"memoryCapacity"`
	PodsCount        int               `json:"podsCount"`
	Unschedulable    bool              `json:"unschedulable"`
	Age              string            `json:"age"`
	Conditions       map[string]string `json:"conditions"`
}

// PodInfo represents a Kubernetes pod
type PodInfo struct {
	Name         string   `json:"name"`
	Namespace    string   `json:"namespace"`
	Status       string   `json:"status"` // Running, Pending, CrashLoopBackOff, etc.
	Ready        string   `json:"ready"`  // e.g. "1/1"
	Restarts     int32    `json:"restarts"`
	Node         string   `json:"node"`
	IP           string   `json:"ip"`
	Containers   []string `json:"containers"`
	Age          string   `json:"age"`
	CreationTime string   `json:"creationTime"`
}

// DeploymentInfo represents a Kubernetes Deployment
type DeploymentInfo struct {
	Name              string   `json:"name"`
	Namespace         string   `json:"namespace"`
	Replicas          int32    `json:"replicas"`
	ReadyReplicas     int32    `json:"readyReplicas"`
	UpdatedReplicas   int32    `json:"updatedReplicas"`
	AvailableReplicas int32    `json:"availableReplicas"`
	Images            []string `json:"images"`
	Age               string   `json:"age"`
}

// DaemonSetInfo represents a Kubernetes DaemonSet
type DaemonSetInfo struct {
	Name            string   `json:"name"`
	Namespace       string   `json:"namespace"`
	DesiredNumber   int32    `json:"desiredNumber"`
	CurrentNumber   int32    `json:"currentNumber"`
	ReadyNumber     int32    `json:"readyNumber"`
	AvailableNumber int32    `json:"availableNumber"`
	Images          []string `json:"images"`
	Age             string   `json:"age"`
}

// StatefulSetInfo represents a Kubernetes StatefulSet
type StatefulSetInfo struct {
	Name          string   `json:"name"`
	Namespace     string   `json:"namespace"`
	Replicas      int32    `json:"replicas"`
	ReadyReplicas int32    `json:"readyReplicas"`
	Images        []string `json:"images"`
	Age           string   `json:"age"`
}

// ServiceInfo represents a Kubernetes Service
type ServiceInfo struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Type        string            `json:"type"`
	ClusterIP   string            `json:"clusterIP"`
	ExternalIP  string            `json:"externalIP"`
	Ports       []string          `json:"ports"`
	Selector    map[string]string `json:"selector"`
	Age         string            `json:"age"`
}

// IngressInfo represents an Ingress resource
type IngressInfo struct {
	Name      string   `json:"name"`
	Namespace string   `json:"namespace"`
	Class     string   `json:"class"`
	Hosts     []string `json:"hosts"`
	Endpoints []string `json:"endpoints"`
	Age       string   `json:"age"`
}

// ConfigResourceInfo represents ConfigMap or Secret
type ConfigResourceInfo struct {
	Name      string   `json:"name"`
	Namespace string   `json:"namespace"`
	Type      string   `json:"type"`
	KeysCount int      `json:"keysCount"`
	Keys      []string `json:"keys"`
	Age       string   `json:"age"`
}

// StorageInfo represents PV or PVC
type StorageInfo struct {
	Name         string   `json:"name"`
	Namespace    string   `json:"namespace"`
	Status       string   `json:"status"`
	Volume       string   `json:"volume"`
	Capacity     string   `json:"capacity"`
	AccessModes  []string `json:"accessModes"`
	StorageClass string   `json:"storageClass"`
	Age          string   `json:"age"`
}

// EventInfo represents a Kubernetes Event
type EventInfo struct {
	Type          string    `json:"type"` // Warning, Normal
	Reason        string    `json:"reason"`
	Object        string    `json:"object"`
	Message       string    `json:"message"`
	Namespace     string    `json:"namespace"`
	Count         int32     `json:"count"`
	Age           string    `json:"age"`
	LastTimestamp time.Time `json:"lastTimestamp"`
}

// NamespaceInfo represents a Kubernetes Namespace
type NamespaceInfo struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Age    string `json:"age"`
}

// ConnectionConfig holds connection settings
type ConnectionConfig struct {
	Mode         string `json:"mode"` // "auto", "kubeconfig", "token"
	Server       string `json:"server"`
	Token        string `json:"token"`
	Insecure     bool   `json:"insecure"`
	KubeconfigContent string `json:"kubeconfigContent"`
}

