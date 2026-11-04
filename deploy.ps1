# Helper script untuk deploy ke VPS dari terminal Anda
param(
    [string]$ServerIP = "103.172.205.41",
    [string]$User = "tovan"
)

Write-Host "=== Deploy Kubernetes Dashboard ke VPS ($ServerIP) ===" -ForegroundColor Cyan

# 1. Pastikan binary linux tersedia
if (-not (Test-Path "k8s-dashboard-linux")) {
    Write-Host "Mengompilasi binary Linux..." -ForegroundColor Yellow
    $env:GOOS = "linux"
    $env:GOARCH = "amd64"
    go build -ldflags="-s -w" -o k8s-dashboard-linux .
    $env:GOOS = ""
    $env:GOARCH = ""
}

# 2. Upload binary ke VPS
Write-Host "`n1. Mengunggah k8s-dashboard-linux ke $User@$ServerIP..." -ForegroundColor Green
Write-Host "(Masukkan kata sandi VPS Anda saat diminta oleh OpenSSH)" -ForegroundColor DarkGray
scp k8s-dashboard-linux "${User}@${ServerIP}:/home/${User}/"

if ($LASTEXITCODE -ne 0) {
    Write-Host "Gagal mengunggah file ke VPS. Periksa koneksi jaringan atau kata sandi Anda." -ForegroundColor Red
    exit $LASTEXITCODE
}

# 3. Jalankan di VPS
Write-Host "`n2. Mengaktifkan dashboard di VPS..." -ForegroundColor Green
ssh -t "${User}@${ServerIP}" "chmod +x /home/${User}/k8s-dashboard-linux && sudo nohup /home/${User}/k8s-dashboard-linux -port 8080 > /home/${User}/dashboard.log 2>&1 & echo 'Dashboard berjalan di background pada port 8080!'"

Write-Host "`n=== Selesai! Buka browser di: http://${ServerIP}:8080 ===" -ForegroundColor Cyan

