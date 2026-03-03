# ==============================================
# Development Helper Script (PowerShell)
# ==============================================

Write-Host "[*] Starting Grafikarsa Development Environment..." -ForegroundColor Green

# Check if .env exists
if (-not (Test-Path .env)) {
    Write-Host "[!] .env file not found. Copying from .env.example..." -ForegroundColor Yellow
    Copy-Item .env.example .env
    Write-Host "[OK] Please edit .env file with your configuration" -ForegroundColor Green
    exit 1
}

# Start services
Write-Host "[*] Starting Docker containers..." -ForegroundColor Cyan
docker compose up -d

# Wait for services to be healthy
Write-Host "[*] Waiting for services to be ready..." -ForegroundColor Cyan
Start-Sleep -Seconds 10

# Check if database needs initialization
Write-Host "[*] Checking database..." -ForegroundColor Cyan
$dbCheck = docker exec grafikarsa-db-dev psql -U grafikarsa -d grafikarsa -c "SELECT 1 FROM users LIMIT 1;" 2>$null

if ($LASTEXITCODE -eq 0) {
    Write-Host "[OK] Database already initialized" -ForegroundColor Green
} else {
    Write-Host "[*] Importing database schema..." -ForegroundColor Cyan
    if (Test-Path db/db.sql) {
        Get-Content db/db.sql | docker exec -i grafikarsa-db-dev psql -U grafikarsa -d grafikarsa
        Write-Host "[OK] Database schema imported" -ForegroundColor Green
    } else {
        Write-Host "[!] Database schema file not found at db/db.sql" -ForegroundColor Yellow
    }
}

# Show status
Write-Host ""
Write-Host "SUCCESS: Development environment is ready!" -ForegroundColor Green
Write-Host ""
Write-Host "Services:" -ForegroundColor Cyan
Write-Host "   - Frontend:      http://localhost:3000"
Write-Host "   - Backend API:   http://localhost:8080"
Write-Host "   - MinIO Console: http://localhost:9001"
Write-Host "   - Database:      localhost:5432"
Write-Host ""
Write-Host "Useful commands:" -ForegroundColor Cyan
Write-Host "   - View logs:     docker compose logs -f"
Write-Host "   - Stop services: docker compose down"
Write-Host "   - Restart:       docker compose restart"
Write-Host ""
