# ==============================================
# Deploy to Production Server (PowerShell)
# ==============================================
# Usage: .\scripts\deploy.ps1
#        .\scripts\deploy.ps1 -Version "abc1234"

param(
    [string]$Version = "latest"
)

$ErrorActionPreference = "Stop"
$ProjectRoot = Split-Path -Parent $PSScriptRoot

# Load environment variables
$envFile = Join-Path $ProjectRoot ".env"
if (Test-Path $envFile) {
    Get-Content $envFile | ForEach-Object {
        if ($_ -match '^([^#][^=]+)=(.*)$') {
            [Environment]::SetEnvironmentVariable($matches[1].Trim(), $matches[2].Trim())
        }
    }
}

# Configuration
$SSH_HOST = if ($env:SSH_HOST) { $env:SSH_HOST } else { "YOUR_SERVER_IP" }
$SSH_PORT = if ($env:SSH_PORT) { $env:SSH_PORT } else { "22" }
$SSH_USER = if ($env:SSH_USER) { $env:SSH_USER } else { "deploy" }
$DEPLOY_PATH = "/opt/grafikarsa"

Write-Host ""
Write-Host "🚀 Deploying Grafikarsa to Production..." -ForegroundColor Magenta
Write-Host ""

# Check if SSH config is set
if ($SSH_HOST -eq "YOUR_SERVER_IP") {
    Write-Host "❌ Please set SSH_HOST, SSH_PORT, and SSH_USER in .env" -ForegroundColor Red
    exit 1
}

Write-Host "📦 Version: $Version" -ForegroundColor Cyan
Write-Host "🖥️  Server: ${SSH_USER}@${SSH_HOST}:${SSH_PORT}" -ForegroundColor Cyan
Write-Host ""

# SSH to server and deploy
Write-Host "📡 Connecting to server..." -ForegroundColor Yellow

$sshScript = @"
set -e
echo '📂 Navigating to deployment directory...'
cd $DEPLOY_PATH
echo '📥 Pulling latest images...'
export IMAGE_TAG=$Version
docker compose -f docker-compose.deploy.yml pull
echo '🔄 Restarting services...'
docker compose -f docker-compose.deploy.yml up -d
echo '🧹 Cleaning up old images...'
docker image prune -f
echo '✅ Deployment complete!'
"@

ssh -p $SSH_PORT "${SSH_USER}@${SSH_HOST}" $sshScript

if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Deployment failed!" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "✅ Deployment successful!" -ForegroundColor Green
Write-Host ""
Write-Host "📝 Verify deployment:" -ForegroundColor Cyan
Write-Host "   ssh -p $SSH_PORT ${SSH_USER}@${SSH_HOST} 'docker ps'" -ForegroundColor White
Write-Host ""
