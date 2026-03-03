#!/bin/bash

# ==============================================
# Deploy to Production Server
# ==============================================

set -e

# Configuration
SSH_HOST=${SSH_HOST:-"YOUR_SERVER_IP"}
SSH_PORT=${SSH_PORT:-"22"}
SSH_USER=${SSH_USER:-"deploy"}
DEPLOY_PATH="/opt/grafikarsa"

echo "🚀 Deploying Grafikarsa to Production..."
echo ""

# Check if SSH config is set
if [ "$SSH_HOST" == "YOUR_SERVER_IP" ]; then
    echo "❌ Please set SSH_HOST, SSH_PORT, and SSH_USER in .env or as environment variables"
    exit 1
fi

# Get version tag
VERSION=${1:-latest}

echo "📦 Version: $VERSION"
echo "🖥️  Server: $SSH_USER@$SSH_HOST:$SSH_PORT"
echo ""

# SSH to server and deploy
echo "📡 Connecting to server..."
ssh -p $SSH_PORT $SSH_USER@$SSH_HOST << EOF
    set -e
    
    echo "📂 Navigating to deployment directory..."
    cd $DEPLOY_PATH
    
    echo "📥 Pulling latest images..."
    export IMAGE_TAG=$VERSION
    docker compose -f docker-compose.deploy.yml pull
    
    echo "🔄 Restarting services..."
    docker compose -f docker-compose.deploy.yml up -d
    
    echo "🧹 Cleaning up old images..."
    docker image prune -f
    
    echo "✅ Deployment complete!"
EOF

echo ""
echo "✅ Deployment successful!"
echo ""
echo "📝 Verify deployment:"
echo "   ssh -p $SSH_PORT $SSH_USER@$SSH_HOST 'docker ps'"
echo ""
