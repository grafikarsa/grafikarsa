# =============================================================================
# GRAFIKARSA MONOREPO - Docker Hub Publish Script
# =============================================================================
# Usage: .\scripts\docker-publish.ps1 -Version "1.0.0"
#        .\scripts\docker-publish.ps1 -Version "1.0.0" -Username "yourusername"
# =============================================================================

param(
    [Parameter(Mandatory=$true)]
    [string]$Version,
    
    [string]$Username = "",
    [switch]$SkipLatest,
    [switch]$BuildOnly
)

$ErrorActionPreference = "Stop"
$ProjectRoot = Split-Path -Parent $PSScriptRoot

# Load environment variables from root .env
$envFile = Join-Path $ProjectRoot ".env"
if (Test-Path $envFile) {
    Get-Content $envFile | ForEach-Object {
        if ($_ -match '^([^#][^=]+)=(.*)$') {
            [Environment]::SetEnvironmentVariable($matches[1].Trim(), $matches[2].Trim())
        }
    }
}

# Use param or env var
if (-not $Username) {
    $Username = $env:DOCKERHUB_USERNAME
}

if (-not $Username) {
    Write-Host "ERROR: DOCKERHUB_USERNAME not set. Provide -Username or set in .env" -ForegroundColor Red
    exit 1
}

# Validate version format
if ($Version -notmatch '^\d+\.\d+\.\d+$') {
    Write-Host "ERROR: Version must be in format X.Y.Z (e.g., 1.0.0)" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "GRAFIKARSA DOCKER PUBLISH" -ForegroundColor Magenta
Write-Host "=========================" -ForegroundColor Magenta
Write-Host ""

$BackendImage = "$Username/grafikarsa-backend"
$WebImage = "$Username/grafikarsa-web"

Write-Host "Backend Image: ${BackendImage}:${Version}" -ForegroundColor Cyan
Write-Host "Web Image:     ${WebImage}:${Version}" -ForegroundColor Cyan
Write-Host ""

# Change to project root
Set-Location $ProjectRoot

# Step 1: Build backend image
Write-Host "[1/4] Building backend image..." -ForegroundColor Yellow
docker build `
    -t "${BackendImage}:${Version}" `
    -t "${BackendImage}:latest" `
    --target production `
    ./apps/backend

if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Backend build failed" -ForegroundColor Red
    exit 1
}
Write-Host "      Backend build successful!" -ForegroundColor Green

# Step 2: Build web image
Write-Host "[2/4] Building web image..." -ForegroundColor Yellow
docker build `
    -t "${WebImage}:${Version}" `
    -t "${WebImage}:latest" `
    --target production `
    --build-arg NEXT_PUBLIC_API_URL="$($env:NEXT_PUBLIC_API_URL)" `
    --build-arg NEXT_PUBLIC_APP_URL="$($env:NEXT_PUBLIC_APP_URL)" `
    ./apps/web

if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Web build failed" -ForegroundColor Red
    exit 1
}
Write-Host "      Web build successful!" -ForegroundColor Green

if ($BuildOnly) {
    Write-Host ""
    Write-Host "Build complete! (--BuildOnly flag set, skipping push)" -ForegroundColor Green
    Write-Host ""
    Write-Host "To push manually:" -ForegroundColor Cyan
    Write-Host "  docker push ${BackendImage}:${Version}" -ForegroundColor White
    Write-Host "  docker push ${WebImage}:${Version}" -ForegroundColor White
    exit 0
}

# Step 3: Login check
Write-Host "[3/4] Checking Docker Hub login..." -ForegroundColor Yellow
$loginCheck = docker info 2>&1 | Select-String "Username"
if (-not $loginCheck) {
    Write-Host "      You need to login to Docker Hub first" -ForegroundColor Yellow
    docker login
    if ($LASTEXITCODE -ne 0) {
        Write-Host "ERROR: Docker login failed" -ForegroundColor Red
        exit 1
    }
}

# Step 4: Push images
Write-Host "[4/4] Pushing images to Docker Hub..." -ForegroundColor Yellow

docker push "${BackendImage}:${Version}"
if ($LASTEXITCODE -ne 0) { Write-Host "ERROR: Failed to push backend" -ForegroundColor Red; exit 1 }

docker push "${WebImage}:${Version}"
if ($LASTEXITCODE -ne 0) { Write-Host "ERROR: Failed to push web" -ForegroundColor Red; exit 1 }

if (-not $SkipLatest) {
    docker push "${BackendImage}:latest"
    docker push "${WebImage}:latest"
}

Write-Host ""
Write-Host "SUCCESS! Images published to Docker Hub" -ForegroundColor Green
Write-Host ""
Write-Host "Pull commands:" -ForegroundColor Cyan
Write-Host "  docker pull ${BackendImage}:${Version}" -ForegroundColor White
Write-Host "  docker pull ${WebImage}:${Version}" -ForegroundColor White
Write-Host ""
