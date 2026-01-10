#!/bin/bash
set -e

echo "Installing npm dependencies..."
npm install

echo "Building frontend..."
npm run build

echo "Installing Go dependencies..."
go mod download

echo "Building Go backend..."
go build -o server ./cmd/server/main.go

echo "Build complete!"
