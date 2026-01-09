#!/bin/bash

# Build the backend to ensure it compiles
echo "Building Backend..."
go build -o server_bin cmd/server/main.go

# Start Backend in background
echo "Starting Backend on port 8081..."
export PORT=8081
./server_bin > backend.log 2>&1 &
BACKEND_PID=$!

# Wait for backend to initialize
sleep 2

# Start Frontend (Vite)
# Vite config is set to listen on 0.0.0.0:8080 or ::
echo "Starting Frontend..."
npm run dev

# Cleanup
kill $BACKEND_PID
