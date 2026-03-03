#!/bin/bash

# =============================================================================
# GRAFIKARSA MONOREPO - API Setup & Test Script
# =============================================================================
# Usage: ./scripts/run_tests.sh
#        ./scripts/run_tests.sh --setup-only
#        ./scripts/run_tests.sh --test-only
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BACKEND_DIR="$PROJECT_ROOT/apps/backend"

SETUP_ONLY=false
TEST_ONLY=false
SKIP_SEED=false

# Parse arguments
for arg in "$@"; do
    case $arg in
        --setup-only) SETUP_ONLY=true ;;
        --test-only) TEST_ONLY=true ;;
        --skip-seed) SKIP_SEED=true ;;
    esac
done

echo ""
echo "═══════════════════════════════"
echo "  GRAFIKARSA API SETUP & TEST"
echo "═══════════════════════════════"
echo ""

# Check if backend directory exists
if [ ! -f "$BACKEND_DIR/go.mod" ]; then
    echo "❌ Backend directory not found at $BACKEND_DIR"
    exit 1
fi

if [ "$TEST_ONLY" = false ]; then
    # Step 1: Check Docker
    echo "[1/5] Checking Docker..."
    if command -v docker &> /dev/null; then
        echo "      ✅ Docker is available"
    else
        echo "      ❌ Docker is not installed or not running"
        exit 1
    fi

    # Step 2: Start Docker services
    echo "[2/5] Starting Docker services (db, minio)..."
    cd "$PROJECT_ROOT"
    docker compose up -d db minio
    echo "      Waiting for services to be ready..."
    sleep 5

    # Step 3: Build dbcli (if it exists)
    echo "[3/5] Building database CLI..."
    cd "$BACKEND_DIR"
    if [ -d "$BACKEND_DIR/cmd/dbcli" ]; then
        go build -o bin/dbcli ./cmd/dbcli
        if [ $? -eq 0 ]; then
            echo "      ✅ Built successfully"

            # Step 4: Setup database
            echo "[4/5] Setting up database..."
            DB_INPUT="1\ny\n"
            if [ "$SKIP_SEED" = false ]; then
                DB_INPUT="${DB_INPUT}5\n1\n"
            fi
            DB_INPUT="${DB_INPUT}0\n"
            echo -e "$DB_INPUT" | ./bin/dbcli
            echo "      ✅ Database setup complete"
        else
            echo "      ⚠️  Failed to build dbcli, importing schema directly..."
            if [ -f "$PROJECT_ROOT/db/db.sql" ]; then
                docker exec -i grafikarsa-db-dev psql -U grafikarsa -d grafikarsa < "$PROJECT_ROOT/db/db.sql"
                echo "      ✅ Database schema imported"
            else
                echo "      ⚠️  Schema file not found"
            fi
        fi
    else
        echo "      dbcli not found, importing schema directly..."
        if [ -f "$PROJECT_ROOT/db/db.sql" ]; then
            docker exec -i grafikarsa-db-dev psql -U grafikarsa -d grafikarsa < "$PROJECT_ROOT/db/db.sql"
            echo "      ✅ Database schema imported"
        else
            echo "      ⚠️  Schema file not found at $PROJECT_ROOT/db/db.sql"
        fi
    fi

    # Step 5: Build API
    echo "[5/5] Building API server..."
    cd "$BACKEND_DIR"
    go build -o bin/api ./cmd/api
    echo "      ✅ Built successfully"
fi

if [ "$SETUP_ONLY" = true ]; then
    echo ""
    echo "✅ Setup complete! To start the API server, run:"
    echo "  ./apps/backend/bin/api"
    echo ""
    echo "Then run tests with:"
    echo "  ./scripts/api_test.sh"
    exit 0
fi

# Start API server in background
echo ""
echo "Starting API server..."
cd "$BACKEND_DIR"
./bin/api &
API_PID=$!
echo "API server started (PID: $API_PID)"
echo "Waiting for server to be ready..."
sleep 3

# Run tests
echo ""
echo "Running API tests..."
echo ""

TEST_RESULT=0
"$SCRIPT_DIR/api_test.sh" || TEST_RESULT=$?

# Stop API server
echo ""
echo "Stopping API server..."
kill $API_PID 2>/dev/null || true
wait $API_PID 2>/dev/null || true
echo "API server stopped"

exit $TEST_RESULT
