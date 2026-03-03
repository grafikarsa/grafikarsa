#!/bin/bash

# ==============================================
# Build Production Images
# ==============================================

set -e

# Load environment variables
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

# Check required variables
if [ -z "$DOCKERHUB_USERNAME" ]; then
    echo "❌ DOCKERHUB_USERNAME not set in .env"
    exit 1
fi

# Get version tag (default to git commit hash)
VERSION=${1:-$(git rev-parse --short HEAD)}

echo "🏗️  Building Grafikarsa Production Images..."
echo "📦 Version: $VERSION"
echo ""

# Build backend
echo "🔨 Building backend image..."
docker build \
    -t ${DOCKERHUB_USERNAME}/grafikarsa-backend:${VERSION} \
    -t ${DOCKERHUB_USERNAME}/grafikarsa-backend:latest \
    --target production \
    ./apps/backend

echo "✅ Backend image built"

# Build frontend
echo "🔨 Building frontend image..."
docker build \
    -t ${DOCKERHUB_USERNAME}/grafikarsa-web:${VERSION} \
    -t ${DOCKERHUB_USERNAME}/grafikarsa-web:latest \
    --target production \
    --build-arg NEXT_PUBLIC_API_URL=${NEXT_PUBLIC_API_URL} \
    --build-arg NEXT_PUBLIC_APP_URL=${NEXT_PUBLIC_APP_URL} \
    ./apps/web

echo "✅ Frontend image built"

echo ""
echo "✅ All images built successfully!"
echo ""
echo "📝 Next steps:"
echo "   - Test images:  docker compose -f docker-compose.prod.yml up"
echo "   - Push images:  ./scripts/push.sh $VERSION"
echo ""
