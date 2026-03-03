# ==============================================
# Push Images to Docker Hub (PowerShell)
# ==============================================

param(
    [string]$Version = "latest"
)

# Load environment variables
if (Test-Path .env) {
    Get-Content .env | ForEach-Object {
        if ($_ -match '^([^#][^=]+)=(.*)$') {
            [Environment]::SetEnvironmentVariable($matches[1], $matches[2])
        }
    }
}

# Check required variables
if (-not $env:DOCKERHUB_USERNAME) {
    Write-Host "[!] DOCKERHUB_USERNAME not set in .env" -ForegroundColor Red
    exit 1
}

Write-Host "[*] Pushing Grafikarsa Images to Docker Hub..." -ForegroundColor Green
Write-Host "[v] Version: $Version" -ForegroundColor Cyan
Write-Host ""

# Login to Docker Hub
Write-Host "[1/3] Logging in to Docker Hub..." -ForegroundColor Cyan
docker login

if ($LASTEXITCODE -ne 0) {
    Write-Host "[ERROR] Docker login failed" -ForegroundColor Red
    exit 1
}

# Push backend
Write-Host "[2/3] Pushing backend image..." -ForegroundColor Cyan
docker push "$($env:DOCKERHUB_USERNAME)/grafikarsa-backend:$Version"

if ($Version -ne "latest") {
    docker push "$($env:DOCKERHUB_USERNAME)/grafikarsa-backend:latest"
}

if ($LASTEXITCODE -eq 0) {
    Write-Host "[OK] Backend image pushed" -ForegroundColor Green
} else {
    Write-Host "[ERROR] Backend push failed" -ForegroundColor Red
    exit 1
}

# Push frontend
Write-Host "[3/3] Pushing frontend image..." -ForegroundColor Cyan
docker push "$($env:DOCKERHUB_USERNAME)/grafikarsa-web:$Version"

if ($Version -ne "latest") {
    docker push "$($env:DOCKERHUB_USERNAME)/grafikarsa-web:latest"
}

if ($LASTEXITCODE -eq 0) {
    Write-Host "[OK] Frontend image pushed" -ForegroundColor Green
} else {
    Write-Host "[ERROR] Frontend push failed" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "SUCCESS: All images pushed successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "Images:" -ForegroundColor Cyan
Write-Host "   - Backend: $($env:DOCKERHUB_USERNAME)/grafikarsa-backend:$Version"
Write-Host "   - Frontend: $($env:DOCKERHUB_USERNAME)/grafikarsa-web:$Version"
Write-Host ""
