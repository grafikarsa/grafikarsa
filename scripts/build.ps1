# ==============================================
# Build Production Images (PowerShell)
# ==============================================

param(
    [string]$Version = (git rev-parse --short HEAD 2>$null)
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

if (-not $Version) {
    $Version = "latest"
}

Write-Host "[*] Building Grafikarsa Production Images..." -ForegroundColor Green
Write-Host "[v] Version: $Version" -ForegroundColor Cyan
Write-Host ""

# Build backend
Write-Host "[1/2] Building backend image..." -ForegroundColor Cyan
docker build `
    -t "$($env:DOCKERHUB_USERNAME)/grafikarsa-backend:$Version" `
    -t "$($env:DOCKERHUB_USERNAME)/grafikarsa-backend:latest" `
    --target production `
    ./apps/backend

if ($LASTEXITCODE -eq 0) {
    Write-Host "[OK] Backend image built" -ForegroundColor Green
} else {
    Write-Host "[ERROR] Backend build failed" -ForegroundColor Red
    exit 1
}

# Build frontend
Write-Host "[2/2] Building frontend image..." -ForegroundColor Cyan
docker build `
    -t "$($env:DOCKERHUB_USERNAME)/grafikarsa-web:$Version" `
    -t "$($env:DOCKERHUB_USERNAME)/grafikarsa-web:latest" `
    --target production `
    --build-arg NEXT_PUBLIC_API_URL="$($env:NEXT_PUBLIC_API_URL)" `
    --build-arg NEXT_PUBLIC_APP_URL="$($env:NEXT_PUBLIC_APP_URL)" `
    ./apps/web

if ($LASTEXITCODE -eq 0) {
    Write-Host "[OK] Frontend image built" -ForegroundColor Green
} else {
    Write-Host "[ERROR] Frontend build failed" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "SUCCESS: All images built successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Cyan
Write-Host "   - Test images:  docker compose -f docker-compose.prod.yml up"
Write-Host "   - Push images:  .\scripts\push.ps1 $Version"
Write-Host ""
