# Build script for Windows and Linux AMD64
Write-Host "Compiling K8s Dashboard for Windows..." -ForegroundColor Cyan
go build -ldflags="-s -w" -o k8s-dashboard.exe .

Write-Host "Compiling K8s Dashboard for Linux (VPS)..." -ForegroundColor Cyan
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -ldflags="-s -w" -o k8s-dashboard-linux .
$env:GOOS = ""
$env:GOARCH = ""

Write-Host "Build complete!" -ForegroundColor Green
Write-Host "Windows executable: .\k8s-dashboard.exe"
Write-Host "Linux executable:   .\k8s-dashboard-linux"

