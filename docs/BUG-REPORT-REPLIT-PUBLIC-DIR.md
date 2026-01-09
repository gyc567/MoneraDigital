# Bug Report: Replit Missing Public Directory

## OpenAPI Specification Format

```yaml
openapi: 3.0.0
info:
  title: MoneraDigital Replit Configuration Bug
  version: 1.0.0
  description: |
    Bug report for Replit deployment error: "Could not find public directory"
    This document tracks the issue and resolution for the missing public directory
    in the Replit environment configuration.

paths:
  /deployment/replit:
    post:
      summary: Deploy application to Replit
      description: |
        Deployment process that fails due to missing public directory configuration
      tags:
        - Deployment
      responses:
        '500':
          description: Deployment failed - public directory not found
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/DeploymentError'
        '200':
          description: Deployment successful
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/DeploymentSuccess'

components:
  schemas:
    DeploymentError:
      type: object
      properties:
        error:
          type: string
          example: "Could not find public directory"
        status:
          type: string
          enum: [FAILED]
        timestamp:
          type: string
          format: date-time
        details:
          type: object
          properties:
            missingDirectory:
              type: string
              example: "public"
            expectedPath:
              type: string
              example: "/workspace/public"
            rootCause:
              type: string
              description: |
                Replit's web module expects a public directory for static file serving,
                but Vite-based React projects don't require this by default.

    DeploymentSuccess:
      type: object
      properties:
        status:
          type: string
          enum: [SUCCESS]
        message:
          type: string
          example: "Application deployed successfully"
        frontendUrl:
          type: string
          format: uri
        backendUrl:
          type: string
          format: uri
```

## Issue Summary

**Title**: Replit Deployment Fails - Missing Public Directory

**Severity**: High (Blocks deployment)

**Environment**: Replit with Node.js 20 and Go 1.21

**Error Message**: `Could not find public directory`

## Root Cause Analysis

The `.replit` configuration file specifies the `web` module, which expects a `public` directory for serving static files. However:

1. The project uses Vite as the build tool
2. Vite outputs built files to `dist/` directory, not `public/`
3. No `public/` directory exists in the project root
4. Replit's web module cannot find the expected directory and fails

## Current Configuration

**File**: `.replit`
```toml
modules = ["nodejs-20", "go-1.21", "web"]
run = "bash scripts/start-replit.sh"

[nix]
channel = "stable-25_05"

[[ports]]
localPort = 8080
externalPort = 80
```

## Solution

Create an empty `public/` directory to satisfy Replit's web module requirements. This directory will:
- Serve as a placeholder for static assets
- Allow Replit's web module to initialize successfully
- Not interfere with Vite's build process (Vite uses `dist/` for output)

## Implementation Steps

1. Create `public/` directory
2. Add `.gitkeep` file to preserve directory in git
3. Test Replit deployment
4. Commit changes

## Files Modified

- `public/.gitkeep` (new file)

## Testing

After fix:
1. Push to Replit
2. Verify no "Could not find public directory" error
3. Confirm frontend loads on port 8080
4. Confirm backend loads on port 8081
5. Verify API proxy works correctly

## Related Issues

- Vite configuration: `vite.config.ts` (uses `dist/` for output)
- Replit configuration: `.replit` (specifies web module)
- Startup script: `scripts/start-replit.sh` (runs Vite dev server)
