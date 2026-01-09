# Bug Report: Replit Deployment 404 Errors

**Priority**: High
**Status**: Open
**Date**: 2026-01-09

## 1. Issue Description

The application deployed on Replit (`https://monera-digital--gyc567.replit.app`) returns `404 Not Found` for all backend API endpoints (`/api/auth/login`, `/api/auth/me`).

## 2. Root Cause Analysis

1.  **Deployment Configuration**: The current `.replit` file only defines a "Frontend" workflow that runs `npm run dev`.
2.  **Missing Backend**: The Go backend is never started in the Replit environment.
3.  **Static Deployment**: The `[deployment]` section specifies `deploymentTarget = "static"`, which serves only the built frontend files (`dist`). This completely bypasses the possibility of running a dynamic Go backend side-by-side.

## 3. Proposed Fix

Reconfigure Replit to run as a **dynamic web application** that executes both the Go backend and the Vite frontend (proxying to the backend).

**Changes Required**:
1.  **Modify `.replit`**:
    -   Change deployment target (if applicable) or configure the `run` command to start both servers.
    -   Use a startup script (e.g., `scripts/start-replit.sh`) to launch the Go backend in the background and the Frontend in the foreground.
    -   Ensure `go run cmd/server/main.go` runs on a dedicated port (8081).
    -   Ensure `npm run dev` runs on the exposed port (8080/5000) and proxies to 8081.

2.  **Dependencies**: Ensure `go` is available in the environment (it is listed in `modules`).

## 4. Implementation Details

Create `scripts/start-replit.sh`:
```bash
#!/bin/bash
# Start Backend
echo "Starting Backend on port 8081..."
PORT=8081 go run cmd/server/main.go &

# Wait for backend
sleep 2

# Start Frontend
echo "Starting Frontend..."
npm run dev
```

Update `.replit`:
```toml
run = "bash scripts/start-replit.sh"
```
