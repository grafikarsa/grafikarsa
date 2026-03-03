#!/bin/bash

# =============================================================================
# GRAFIKARSA MONOREPO - Docker Hub Publish Script
# =============================================================================
# Usage: ./scripts/docker-publish.sh 1.0.0
#        ./scripts/docker-publish.sh 1.0.0 yourusername
#        ./scripts/docker-publish.sh 1.0.0 --build-only
# =============================================================================

set -e

VERSION=${1:?"Usage: $0 <version> [username] [--build-only]"}
USERNAME=${2:-""}
BUILD_ONLY=false

# Check for --build-only flag
for arg in "$@"; do
    if [ "$arg" == "--build-only" ]; then
        BUILD_ONLY=true
    fi
done

# Load environment variables
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

# Use param or env var
if [ -z "$USERNAME" ] || [ "$USERNAME" == "--build-only" ]; then
    USERNAME=$DOCKERHUB_USERNAME
fi

if [ -z "$USERNAME" ]; then
    echo "❌ DOCKERHUB_USERNAME not set. Provide as argument or set in .env"
    exit 1
fi

# Validate version format
if ! echo "$VERSION" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$'; then
    echo "❌ Version must be in format X.Y.Z (e.g., 1.0.0)"
    exit 1
fi

echo ""
echo "═══════════════════════════"
echo "  GRAFIKARSA DOCKER PUBLISH"
echo "═══════════════════════════"
echo ""

BACKEND_IMAGE="$USERNAME/grafikarsa-backend"
WEB_IMAGE="$USERNAME/grafikarsa-web"

echo "Backend Image: ${BACKEND_IMAGE}:${VERSION}"
echo "Web Image:     ${WEB_IMAGE}:${VERSION}"
echo ""

# Step 1: Build backend image
echo "[1/4] Building backend image..."
docker build \
    -t "${BACKEND_IMAGE}:${VERSION}" \
    -t "${BACKEND_IMAGE}:latest" \
    --target production \
    ./apps/backend

echo "      ✅ Backend build successful!"

# Step 2: Build web image
echo "[2/4] Building web image..."
docker build \
    -t "${WEB_IMAGE}:${VERSION}" \
    -t "${WEB_IMAGE}:latest" \
    --target production \
    --build-arg NEXT_PUBLIC_API_URL=${NEXT_PUBLIC_API_URL} \
    --build-arg NEXT_PUBLIC_APP_URL=${NEXT_PUBLIC_APP_URL} \
    ./apps/web

echo "      ✅ Web build successful!"

if [ "$BUILD_ONLY" = true ]; then
    echo ""
    echo "✅ Build complete! (--build-only flag set, skipping push)"
    echo ""
    echo "To push manually:"
    echo "  docker push ${BACKEND_IMAGE}:${VERSION}"
    echo "  docker push ${WEB_IMAGE}:${VERSION}"
    exit 0
fi

# Step 3: Login check
echo "[3/4] Checking Docker Hub login..."
if ! docker info 2>&1 | grep -q "Username"; then
    echo "      You need to login to Docker Hub first"
    docker login
fi

# Step 4: Push images
echo "[4/4] Pushing images to Docker Hub..."

docker push "${BACKEND_IMAGE}:${VERSION}"
docker push "${WEB_IMAGE}:${VERSION}"
docker push "${BACKEND_IMAGE}:latest"
docker push "${WEB_IMAGE}:latest"

echo ""
echo "✅ SUCCESS! Images published to Docker Hub"
echo ""
echo "Pull commands:"
echo "  docker pull ${BACKEND_IMAGE}:${VERSION}"
echo "  docker pull ${WEB_IMAGE}:${VERSION}"
echo ""
