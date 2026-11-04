package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"k8s-dashboard/internal/k8s"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AIConfig struct {
	Provider string `json:"provider"` // "gemini", "openai", "builtin"
	APIKey   string `json:"apiKey"`
	Model    string `json:"model"`
}

var (
	aiMu     sync.RWMutex
	aiConfig = AIConfig{
		Provider: "gemini",
		Model:    "gemini-2.5-flash",
		APIKey:   os.Getenv("GEMINI_API_KEY"),
	}
)

type AIChatRequest struct {
	Prompt               string `json:"prompt"`
	AttachClusterContext bool   `json:"attachClusterContext"`
}

type AIChatResponse struct {
	Reply     string `json:"reply"`
	Provider  string `json:"provider"`
	HasAPIKey bool   `json:"hasApiKey"`
}

// GetAIConfigHandler handles GET /api/ai/config
func GetAIConfigHandler(w http.ResponseWriter, r *http.Request) {
	aiMu.RLock()
	defer aiMu.RUnlock()

	hasKey := aiConfig.APIKey != ""
	maskedKey := ""
	if hasKey {
		if len(aiConfig.APIKey) > 8 {
			maskedKey = aiConfig.APIKey[:4] + "..." + aiConfig.APIKey[len(aiConfig.APIKey)-4:]
		} else {
			maskedKey = "****"
		}
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"provider":  aiConfig.Provider,
		"model":     aiConfig.Model,
		"hasKey":    hasKey,
		"maskedKey": maskedKey,
	})
}

// SaveAIConfigHandler handles POST /api/ai/config
func SaveAIConfigHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Provider string `json:"provider"`
		APIKey   string `json:"apiKey"`
		Model    string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	aiMu.Lock()
	defer aiMu.Unlock()

	if req.Provider != "" {
		aiConfig.Provider = req.Provider
	}
	if req.APIKey != "" {
		aiConfig.APIKey = req.APIKey
	}
	if req.Model != "" {
		aiConfig.Model = req.Model
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Konfigurasi AI berhasil disimpan",
	})
}

// AIChatHandler handles POST /api/ai/chat
func AIChatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req AIChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Prompt) == "" {
		WriteError(w, http.StatusBadRequest, "Prompt tidak boleh kosong")
		return
	}

	aiMu.RLock()
	cfg := aiConfig
	aiMu.RUnlock()

	// Gather live cluster context
	clusterContext := ""
	if req.AttachClusterContext {
		clusterContext = gatherClusterContext()
	}

	systemPrompt := `Anda adalah asisten AI ahli Kubernetes (K8s Copilot) dan arsitektur server Linux yang terintegrasi langsung di dalam Kubernetes Master Dashboard.
Tugas Anda:
1. Membantu pengguna memecahkan error, mendiagnosa masalah node/pod, memberikan rekomendasi konfigurasi, dan membuatkan manifest YAML atau perintah kubectl yang siap dijalankan.
2. Selalu berikan jawaban yang jelas, akurat, terstruktur, dan gunakan Markdown dengan blok kode untuk YAML atau perintah shell bash/kubectl.
3. Gunakan bahasa Indonesia yang ramah, profesional, dan to the point.`

	if clusterContext != "" {
		systemPrompt += "\n\nBerikut adalah INFORMASI LIVE CLUSTER KUBERNETES SAAT INI:\n" + clusterContext
	}

	// 1. If Gemini API Key is available, call Gemini API
	if cfg.APIKey != "" && (cfg.Provider == "gemini" || cfg.Provider == "") {
		reply, err := callGeminiAPI(cfg.APIKey, cfg.Model, systemPrompt, req.Prompt)
		if err == nil && reply != "" {
			WriteJSON(w, http.StatusOK, AIChatResponse{
				Reply:     reply,
				Provider:  "Google Gemini (" + cfg.Model + ")",
				HasAPIKey: true,
			})
			return
		}
	}

	// 2. Fallback: Intelligent Built-in Diagnostic Engine
	reply := generateBuiltinResponse(req.Prompt, clusterContext)
	WriteJSON(w, http.StatusOK, AIChatResponse{
		Reply:     reply,
		Provider:  "K8s Built-in Expert Engine",
		HasAPIKey: cfg.APIKey != "",
	})
}

func gatherClusterContext() string {
	mgr := k8s.GetManager()
	client, err := mgr.GetClient()
	if err != nil {
		return "Status: Cluster belum terhubung (" + err.Error() + ")"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var sb strings.Builder
	status := mgr.GetStatus()
	sb.WriteString(fmt.Sprintf("- Versi K8s: %v\n- API Server: %v\n", status["version"], status["serverURL"]))

	// Nodes summary
	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err == nil {
		sb.WriteString(fmt.Sprintf("- Total Nodes: %d\n", len(nodes.Items)))
		for _, n := range nodes.Items {
			ready := "NotReady"
			for _, c := range n.Status.Conditions {
				if c.Type == "Ready" && c.Status == "True" {
					ready = "Ready"
				}
			}
			var roles []string
			for l := range n.Labels {
				if strings.HasPrefix(l, "node-role.kubernetes.io/") {
					roles = append(roles, strings.TrimPrefix(l, "node-role.kubernetes.io/"))
				}
			}
			sb.WriteString(fmt.Sprintf("  * Node %s (Roles: %s, Status: %s, Kubelet: %s)\n", n.Name, strings.Join(roles, ","), ready, n.Status.NodeInfo.KubeletVersion))
		}
	}

	// Warning events
	events, err := client.CoreV1().Events("").List(ctx, metav1.ListOptions{
		FieldSelector: "type=Warning",
		Limit:         5,
	})
	if err == nil && len(events.Items) > 0 {
		sb.WriteString("- Peringatan/Warning Events Terkini:\n")
		for _, e := range events.Items {
			sb.WriteString(fmt.Sprintf("  * [%s] %s: %s\n", e.Reason, e.InvolvedObject.Name, e.Message))
		}
	}

	return sb.String()
}

func callGeminiAPI(apiKey, model, systemPrompt, userPrompt string) (string, error) {
	if model == "" {
		model = "gemini-2.5-flash"
	}
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)

	payload := map[string]interface{}{
		"system_instruction": map[string]interface{}{
			"parts": []map[string]interface{}{
				{"text": systemPrompt},
			},
		},
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]interface{}{
					{"text": userPrompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature": 0.4,
			"maxOutputTokens": 2048,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gemini api error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(respBytes, &geminiResp); err != nil {
		return "", err
	}

	if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
		return geminiResp.Candidates[0].Content.Parts[0].Text, nil
	}

	return "", fmt.Errorf("no response candidates received")
}

func generateBuiltinResponse(prompt, clusterContext string) string {
	p := strings.ToLower(prompt)

	if strings.Contains(p, "notready") || strings.Contains(p, "worker") && (strings.Contains(p, "merah") || strings.Contains(p, "error")) {
		return `### 🔍 Analisis Node Worker NotReady

Node worker yang berstatus **NotReady** biasanya disebabkan oleh salah satu dari faktor berikut:

1. **Swap Aktif**:
   Kubernetes mewajibkan swap dinonaktifkan secara permanen.
   ```bash
   sudo swapoff -a
   sudo sed -i '/ swap / s/^\(.*\)$/#\1/g' /etc/fstab
   ```

2. **Kubelet Mati atau Gagal Menghubungi Master**:
   Cek status service kubelet di worker:
   ```bash
   sudo systemctl status kubelet
   sudo journalctl -u kubelet -n 50 --no-pager
   ```

3. **CNI Network Belum Terpasang / Crash**:
   Periksa pod CNI (Calico/Flannel) di master node:
   ```bash
   kubectl get pods -n kube-system -o wide
   ```

4. **Sertifikat SAN / IP Mismatch**:
   Jika worker berada di beda network/data center, pastikan perintah join mengarah ke **IP Public Master** dan IP tersebut sudah didaftarkan pada certs SAN master.`
	}

	if strings.Contains(p, "join") || strings.Contains(p, "tambah worker") || strings.Contains(p, "integrasi worker") {
		return `### 🚀 Panduan Menghubungkan Worker Node Baru

1. **Di Master Node**, buat token join baru:
   ` + "```bash\nsudo kubeadm token create --print-join-command\n```" + `

2. **Di Worker Node**, pastikan Swap mati, containerd berjalan, dan kubelet versi sama sudah terpasang:
   ` + "```bash\nsudo swapoff -a\nsudo systemctl restart containerd\n```" + `

3. **Eksekusi Join**:
   *Jika worker berada di luar LAN (beda data center), ganti IP private menjadi **IP Public Master**:*
   ` + "```bash\nsudo kubeadm join <IP_PUBLIC_MASTER>:6443 --token <token> --discovery-token-ca-cert-hash sha256:<hash>\n```"
	}

	if strings.Contains(p, "scale") || strings.Contains(p, "replika") {
		return `### ⚙️ Cara Scale Deployment

Anda dapat mengubah jumlah replica langsung dari UI di tab **Workloads > Deployments > Scale**, atau via terminal:

` + "```bash\nkubectl scale deployment <nama-deployment> --replicas=<jumlah> -n <namespace>\n```" + `

Contoh:
` + "```bash\nkubectl scale deployment nginx-app --replicas=3 -n default\n```"
	}

	if strings.Contains(p, "nginx") || strings.Contains(p, "deploy") || strings.Contains(p, "yaml") {
		return `### 📦 Manifest YAML Deployment & Service Nginx

Berikut manifest standar siap pakai:

` + "```yaml\napiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: nginx-deployment\n  namespace: default\n  labels:\n    app: nginx\nspec:\n  replicas: 2\n  selector:\n    matchLabels:\n      app: nginx\n  template:\n    metadata:\n      labels:\n        app: nginx\n    spec:\n      containers:\n      - name: nginx\n        image: nginx:latest\n        ports:\n        - containerPort: 80\n---\napiVersion: v1\nkind: Service\nmetadata:\n  name: nginx-service\n  namespace: default\nspec:\n  type: NodePort\n  selector:\n    app: nginx\n  ports:\n  - port: 80\n    targetPort: 80\n    nodePort: 30080\n```" + `

Terapkan dengan:
` + "```bash\nkubectl apply -f nginx.yaml\n```"
	}

	// General response
	return `### 🤖 K8s Assistant Diagnostic Report

Halo! Saya telah memeriksa kondisi cluster Anda:

` + clusterContext + `

**Rekomendasi Diagnosa Cepat:**
1. Untuk mengecek event peringatan di seluruh cluster:
   ` + "```bash\nkubectl get events -A --sort-by='.metadata.creationTimestamp' | grep -i warning\n```" + `
2. Untuk melihat log pod tertentu yang mengalami crash:
   ` + "```bash\nkubectl logs <nama-pod> -n <namespace> --previous\n```" + `
3. Untuk melihat kondisi mendalam sebuah node:
   ` + "```bash\nkubectl describe node <nama-node>\n```" + `

> [!TIP]
> *Anda juga dapat memasukkan **Google Gemini API Key** di menu pengaturan AI untuk membuka kemampuan penalaran LLM penuh bagi troubleshooting otomatis & perbaikan server.*`
}
