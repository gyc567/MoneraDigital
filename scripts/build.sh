#!/bin/bash
set -e

cd "$(dirname "$0")/.."

echo "Current directory: $(pwd)"

echo "Installing npm dependencies..."
npm install

echo "Building frontend..."
npm run build

echo "Installing Go dependencies..."
go mod download

echo "Building Go backend..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o server ./cmd/server/main.go

echo "Verifying server binary..."
ls -la server
chmod +x server

echo "Build complete!"
