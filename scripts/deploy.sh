#!/bin/bash

set -e

cd "$(dirname "$0")/.."

echo "🚀 Starting Docker deployment process..."

echo "📦 Pulling latest Docker image..."
docker pull ghcr.io/gyc567/monera-digital:latest

echo "🔄 Updating docker-compose.yml..."
sed -i 's|image: ghcr.io/gyc567/monera-digital:.*|image: ghcr.io/gyc567/monera-digital:latest|' docker-compose.yml

echo "🛑 Stopping existing containers..."
docker compose down || true

echo "🚀 Starting containers..."
docker compose up -d

echo "⏳ Waiting for application to be healthy..."
sleep 10

echo "✅ Verifying deployment..."
if curl -sf http://localhost:5000/health > /dev/null 2>&1; then
    echo "✅ Deployment successful! Application is healthy."
else
    echo "⚠️ Application may need more time to start. Checking logs..."
    docker compose logs --tail=50
fi

echo "📋 Running containers:"
docker compose ps

echo "🧹 Cleaning up old Docker images..."
docker rmi $(docker images -q -f "reference=ghcr.io/gyc567/monera-digital:*" --filter "before=ghcr.io/gyc567/monera-digital:latest" 2>/dev/null) || true

echo "✨ Deployment complete!"
