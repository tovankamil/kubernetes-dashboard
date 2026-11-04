# Kubernetes Master Control Dashboard (Golang)

Dashboard visual berbasis web modern dan ringan untuk mengelola dan memonitor cluster **Kubernetes Master Node**, dibangun menggunakan **Golang** dan **Tailwind CSS**.

Seluruh asset frontend (HTML, CSS, JS) di-*embed* langsung ke dalam binary Go (`embed.FS`), sehingga menghasilkan **satu file binary mandiri tanpa dependensi eksternal** (*single zero-dependency executable*).

---

## Fitur Utama

1. **Cluster Overview & Audit**:
   - Status Control Plane & Master Node
   - Metrik resource cluster: CPU & Memory Capacity vs Allocatable
   - Penghitung status Pods (Running, Pending, Failed)
   - Jumlah Deployment, Service, dan Namespace
2. **Node & Master Management**:
   - Status Ready / NotReady, role node (control-plane, worker)
   - Versi Kubelet, OS Image, Kernel, dan Container Runtime
   - Kapasitas Pods dan Resource per node
   - **Cordon & Uncordon node** langsung dari UI (mencegah penjadwalan pod baru saat maintenance)
3. **Workload Management**:
   - **Pods**: Filter namespace, status badge, jumlah restart, node host, IP, dan umur pod.
   - **Live Pod Logs Viewer**: Terminal log viewer dengan auto-refresh live, container selector, filter baris (tail 50-1000), dan tombol Copy.
   - **Delete / Restart Pod**: Hapus pod agar di-*recreate* otomatis oleh ReplicaSet.
   - **Deployments**: Status replica (Ready vs Desired), container images.
   - **Interactive Scale Deployment**: Naikkan atau turunkan jumlah replica secara langsung dari UI.
   - **Rollout Restart Deployment**: Trigger rolling restart aman tanpa downtime.
   - **DaemonSets & StatefulSets**: Monitoring status dan replika secara lengkap.
4. **Networking**:
   - **Services (SVC)**: ClusterIP, NodePort, LoadBalancer, port mappings, dan selector.
   - **Ingress**: Ingress class, host routing, dan endpoint paths.
5. **Storage & Configuration**:
   - **ConfigMaps**: Daftar configmap, jumlah keys, dan metadata.
   - **Secrets**: Daftar secret, tipe secret, dan keys count.
   - **Persistent Volumes (PV) & Claims (PVC)**: Kapasitas, access mode, status bound, dan storage class.
6. **Cluster Events (Real-Time Audit)**:
   - Menampilkan peringatan (`Warning`) dan event (`Normal`) untuk diagnosa instan terhadap *crash*, *failed scheduling*, *OOMKilled*, atau masalah CNI jaringan.
7. **Koneksi Fleksibel**:
   - **Auto-Detect**: Otomatis mendeteksi `~/.kube/config`, `/etc/kubernetes/admin.conf`, atau In-Cluster ServiceAccount.
   - **Paste Kubeconfig**: Tempel file konfigurasi kubeconfig langsung melalui UI.
   - **Remote Token**: Hubungkan via URL API Server (misal: `https://103.172.205.41:6443`) dan Bearer Token dengan opsi lewati TLS self-signed.

---

## Cara Menjalankan

### A. Dijalankan di Lokal (Windows)

1. Jalankan aplikasi:
   ```powershell
   go run main.go
   ```
   Atau compile menjadi file executable `.exe`:
   ```powershell
   go build -o k8s-dashboard.exe .
   .\k8s-dashboard.exe
   ```
2. Buka browser di [http://localhost:8080](http://localhost:8080).
3. Di menu **Connection & Config**, tempel isi `kubeconfig` dari VPS Anda atau masukkan URL API Server VPS (`https://103.172.205.41:6443`) beserta Token.

---

### B. Dijalankan Langsung di VPS Linux (Direkomendasikan)

Jika dijalankan langsung di VPS master node, dashboard akan **otomatis mendeteksi** `/etc/kubernetes/admin.conf` atau `~/.kube/config` tanpa perlu setup tambahan.

1. **Compile untuk Linux AMD64 (dari Windows)**:
   ```powershell
   $env:GOOS="linux"; $env:GOARCH="amd64"; go build -o k8s-dashboard-linux .
   ```
2. **Unggah file binary ke VPS**:
   ```bash
   scp k8s-dashboard-linux tovan@103.172.205.41:/home/tovan/
   ```
3. **Login ke VPS & Jalankan**:
   ```bash
   ssh tovan@103.172.205.41
   chmod +x k8s-dashboard-linux
   
   # Jalankan di background (port 8080)
   sudo ./k8s-dashboard-linux -port 8080 &
   ```
4. Buka di browser: `http://103.172.205.41:8080` (pastikan port 8080 diizinkan di firewall VPS / `ufw allow 8080/tcp`).

---

## Perintah Audit Tambahan untuk Master Node

Jalankan perintah ini di terminal VPS untuk memverifikasi kesehatan cluster:

```bash
# 1. Cek status node & control plane
kubectl get nodes -o wide

# 2. Cek pod core Kubernetes (etcd, apiserver, scheduler, cni)
kubectl get pods -n kube-system -o wide

# 3. Cek resource utilization
kubectl top nodes
kubectl top pods -A
```

