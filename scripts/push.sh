#!/bin/bash

# ==============================================
# Push Images to Docker Hub
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

# Get version tag
VERSION=${1:-latest}

echo "📤 Pushing Grafikarsa Images to Docker Hub..."
echo "📦 Version: $VERSION"
echo ""

# Login to Docker Hub
echo "🔐 Logging in to Docker Hub..."
docker login

# Push backend
echo "📤 Pushing backend image..."
docker push ${DOCKERHUB_USERNAME}/grafikarsa-backend:${VERSION}
if [ "$VERSION" != "latest" ]; then
    docker push ${DOCKERHUB_USERNAME}/grafikarsa-backend:latest
fi
echo "✅ Backend image pushed"

# Push frontend
echo "📤 Pushing frontend image..."
docker push ${DOCKERHUB_USERNAME}/grafikarsa-web:${VERSION}
if [ "$VERSION" != "latest" ]; then
    docker push ${DOCKERHUB_USERNAME}/grafikarsa-web:latest
fi
echo "✅ Frontend image pushed"

echo ""
echo "✅ All images pushed successfully!"
echo ""
echo "🔗 Images:"
echo "   - Backend: ${DOCKERHUB_USERNAME}/grafikarsa-backend:${VERSION}"
echo "   - Frontend: ${DOCKERHUB_USERNAME}/grafikarsa-web:${VERSION}"
echo ""
