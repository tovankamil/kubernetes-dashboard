#!/bin/bash
# ====================================================================
# Setup Script untuk Worker Node Baru (Kubernetes v1.29)
# Jalankan script ini sebagai root (sudo) pada mesin Worker baru
# ====================================================================

set -e

echo "=== 1. Menonaktifkan Swap ==="
swapoff -a
sed -i '/ swap / s/^\(.*\)$/#\1/g' /etc/fstab

echo "=== 2. Konfigurasi Kernel Modules & Sysctl untuk Jaringan K8s ==="
cat <<EOF | tee /etc/modules-load.d/k8s.conf
overlay
br_netfilter
EOF

modprobe overlay
modprobe br_netfilter

cat <<EOF | tee /etc/sysctl.d/k8s.conf
net.bridge.bridge-nf-call-iptables  = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward                 = 1
EOF

sysctl --system

echo "=== 3. Instalasi Container Runtime (containerd) ==="
apt-get update
apt-get install -y ca-certificates curl gnupg lsb-release apt-transport-https

# Install containerd
apt-get install -y containerd

# Konfigurasi containerd dengan SystemdCgroup = true
mkdir -p /etc/containerd
containerd config default | tee /etc/containerd/config.toml >/dev/null
sed -i 's/SystemdCgroup \= false/SystemdCgroup \= true/g' /etc/containerd/config.toml
systemctl restart containerd
systemctl enable containerd

echo "=== 4. Instalasi Kubelet, Kubeadm, Kubectl (v1.29) ==="
mkdir -p -m 755 /etc/apt/keyrings
curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.29/deb/Release.key | gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg --yes

echo 'deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.29/deb/ /' | tee /etc/apt/sources.list.d/kubernetes.list

apt-get update
apt-get install -y kubelet=1.29.15-1.1 kubeadm=1.29.15-1.1 kubectl=1.29.15-1.1 || apt-get install -y kubelet kubeadm kubectl
apt-mark hold kubelet kubeadm kubectl

systemctl enable --now kubelet

echo "===================================================================="
echo "Instalasi paket worker selesai!"
echo "Sekarang jalankan perintah 'kubeadm join' yang didapat dari master node."
echo "===================================================================="

