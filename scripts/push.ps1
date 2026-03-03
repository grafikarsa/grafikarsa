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
    Write-Host "❌ DOCKERHUB_USERNAME not set in .env" -ForegroundColor Red
    exit 1
}

Write-Host "📤 Pushing Grafikarsa Images to Docker Hub..." -ForegroundColor Green
Write-Host "📦 Version: $Version" -ForegroundColor Cyan
Write-Host ""

# Login to Docker Hub
Write-Host "🔐 Logging in to Docker Hub..." -ForegroundColor Cyan
docker login

if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Docker login failed" -ForegroundColor Red
    exit 1
}

# Push backend
Write-Host "📤 Pushing backend image..." -ForegroundColor Cyan
docker push "$env:DOCKERHUB_USERNAME/grafikarsa-backend:$Version"

if ($Version -ne "latest") {
    docker push "$env:DOCKERHUB_USERNAME/grafikarsa-backend:latest"
}

if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ Backend image pushed" -ForegroundColor Green
} else {
    Write-Host "❌ Backend push failed" -ForegroundColor Red
    exit 1
}

# Push frontend
Write-Host "📤 Pushing frontend image..." -ForegroundColor Cyan
docker push "$env:DOCKERHUB_USERNAME/grafikarsa-web:$Version"

if ($Version -ne "latest") {
    docker push "$env:DOCKERHUB_USERNAME/grafikarsa-web:latest"
}

if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ Frontend image pushed" -ForegroundColor Green
} else {
    Write-Host "❌ Frontend push failed" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "✅ All images pushed successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "🔗 Images:" -ForegroundColor Cyan
Write-Host "   - Backend: $env:DOCKERHUB_USERNAME/grafikarsa-backend:$Version"
Write-Host "   - Frontend: $env:DOCKERHUB_USERNAME/grafikarsa-web:$Version"
Write-Host ""
