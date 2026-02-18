#!/bin/bash

set -e

cd "$(dirname "$0")/.."

echo "🚀 Deploying to Vercel..."

vercel --prod --yes

echo "✅ Deployment complete!"
