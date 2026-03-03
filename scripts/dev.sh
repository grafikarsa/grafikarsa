#!/bin/bash

# ==============================================
# Development Helper Script
# ==============================================

set -e

echo "🚀 Starting Grafikarsa Development Environment..."

# Check if .env exists
if [ ! -f .env ]; then
    echo "⚠️  .env file not found. Copying from .env.example..."
    cp .env.example .env
    echo "✅ Please edit .env file with your configuration"
    exit 1
fi

# Start services
echo "📦 Starting Docker containers..."
docker compose up -d

# Wait for services to be healthy
echo "⏳ Waiting for services to be ready..."
sleep 10

# Check if database needs initialization
echo "🗄️  Checking database..."
if docker exec grafikarsa-db-dev psql -U grafikarsa -d grafikarsa -c "SELECT 1 FROM users LIMIT 1;" 2>/dev/null; then
    echo "✅ Database already initialized"
else
    echo "📥 Importing database schema..."
    if [ -f db/db.sql ]; then
        docker exec -i grafikarsa-db-dev psql -U grafikarsa -d grafikarsa < db/db.sql
        echo "✅ Database schema imported"
    else
        echo "⚠️  Database schema file not found at db/db.sql"
    fi
fi

# Show status
echo ""
echo "✅ Development environment is ready!"
echo ""
echo "📍 Services:"
echo "   - Frontend:      http://localhost:3000"
echo "   - Backend API:   http://localhost:8080"
echo "   - MinIO Console: http://localhost:9001"
echo "   - Database:      localhost:5432"
echo ""
echo "📝 Useful commands:"
echo "   - View logs:     docker compose logs -f"
echo "   - Stop services: docker compose down"
echo "   - Restart:       docker compose restart"
echo ""
