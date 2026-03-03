# =============================================================================
# GRAFIKARSA MONOREPO - API Setup & Test Script
# =============================================================================
# Usage: .\scripts\run_tests.ps1
#        .\scripts\run_tests.ps1 -SetupOnly
#        .\scripts\run_tests.ps1 -TestOnly

param(
    [switch]$SetupOnly,
    [switch]$TestOnly,
    [switch]$SkipSeed
)

$ErrorActionPreference = "Stop"
$ProjectRoot = Split-Path -Parent $PSScriptRoot
$BackendDir = Join-Path $ProjectRoot "apps\backend"

Write-Host ""
Write-Host "GRAFIKARSA API SETUP & TEST" -ForegroundColor Magenta
Write-Host "============================" -ForegroundColor Magenta
Write-Host ""

# Check if backend directory exists
if (-not (Test-Path (Join-Path $BackendDir "go.mod"))) {
    Write-Host "ERROR: Backend directory not found at $BackendDir" -ForegroundColor Red
    exit 1
}

if (-not $TestOnly) {
    # Step 1: Check Docker
    Write-Host "[1/5] Checking Docker..." -ForegroundColor Cyan
    try {
        $null = docker --version
        Write-Host "      Docker is available" -ForegroundColor Green
    } catch {
        Write-Host "      ERROR: Docker is not installed or not running" -ForegroundColor Red
        exit 1
    }

    # Step 2: Start Docker services
    Write-Host "[2/5] Starting Docker services (db, minio)..." -ForegroundColor Cyan
    Set-Location $ProjectRoot
    docker compose up -d db minio
    if ($LASTEXITCODE -ne 0) {
        Write-Host "      ERROR: Failed to start Docker services" -ForegroundColor Red
        exit 1
    }
    Write-Host "      Waiting for services to be ready..." -ForegroundColor Gray
    Start-Sleep -Seconds 5

    # Step 3: Build dbcli (if it exists)
    Write-Host "[3/5] Building database CLI..." -ForegroundColor Cyan
    Set-Location $BackendDir
    if (Test-Path (Join-Path $BackendDir "cmd\dbcli")) {
        go build -o bin/dbcli.exe ./cmd/dbcli
        if ($LASTEXITCODE -ne 0) {
            Write-Host "      WARNING: Failed to build dbcli, skipping db setup via CLI" -ForegroundColor Yellow
        } else {
            Write-Host "      Built successfully" -ForegroundColor Green

            # Step 4: Setup database
            Write-Host "[4/5] Setting up database..." -ForegroundColor Cyan
            $dbcliInput = "1`ny`n"
            if (-not $SkipSeed) {
                $dbcliInput += "5`n1`n"
            }
            $dbcliInput += "0`n"
            $dbcliInput | .\bin\dbcli.exe
            Write-Host "      Database setup complete" -ForegroundColor Green
        }
    } else {
        Write-Host "      dbcli not found, importing schema directly..." -ForegroundColor Yellow
        $schemaFile = Join-Path $ProjectRoot "db\db.sql"
        if (Test-Path $schemaFile) {
            Get-Content $schemaFile | docker exec -i grafikarsa-db-dev psql -U grafikarsa -d grafikarsa
            Write-Host "      Database schema imported" -ForegroundColor Green
        } else {
            Write-Host "      WARNING: Schema file not found at $schemaFile" -ForegroundColor Yellow
        }
    }

    # Step 5: Build API
    Write-Host "[5/5] Building API server..." -ForegroundColor Cyan
    Set-Location $BackendDir
    go build -o bin/api.exe ./cmd/api
    if ($LASTEXITCODE -ne 0) {
        Write-Host "      ERROR: Failed to build API" -ForegroundColor Red
        exit 1
    }
    Write-Host "      Built successfully" -ForegroundColor Green
}

if ($SetupOnly) {
    Write-Host ""
    Write-Host "Setup complete! To start the API server, run:" -ForegroundColor Green
    Write-Host "  .\apps\backend\bin\api.exe" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Then run tests with:" -ForegroundColor Green
    Write-Host "  .\scripts\api_test.ps1" -ForegroundColor Yellow
    exit 0
}

# Start API server in background
Write-Host ""
Write-Host "Starting API server..." -ForegroundColor Cyan
Set-Location $BackendDir
$apiProcess = Start-Process -FilePath ".\bin\api.exe" -PassThru -WindowStyle Hidden
Write-Host "API server started (PID: $($apiProcess.Id))" -ForegroundColor Green
Write-Host "Waiting for server to be ready..." -ForegroundColor Gray
Start-Sleep -Seconds 3

# Run tests
Write-Host ""
Write-Host "Running API tests..." -ForegroundColor Cyan
Write-Host ""

try {
    & "$PSScriptRoot\api_test.ps1"
    $testResult = $LASTEXITCODE
} finally {
    # Stop API server
    Write-Host ""
    Write-Host "Stopping API server..." -ForegroundColor Cyan
    Stop-Process -Id $apiProcess.Id -Force -ErrorAction SilentlyContinue
    Write-Host "API server stopped" -ForegroundColor Green
}

exit $testResult
