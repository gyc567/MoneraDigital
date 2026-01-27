# OpenSpec: Add Consistent JWT Token Verification to Protected Endpoints

**Status:** Proposal
**Author:** Claude Code
**Date:** 2026-01-27
**Priority:** High
**Affects:** API Security, Address Management, Authentication

---

## 1. Problem Statement

The application has **inconsistent JWT token verification across protected API endpoints**, creating a security vulnerability where some endpoints verify tokens at the Vercel layer while others skip verification and rely solely on the Go backend.

### Current Behavior

**Verified (Correct):**
- `/api/auth/me` - **Verifies JWT token** before proxying to Go backend
- Pattern: Uses `verifyToken(req)` to validate Authorization header

**Not Verified (Vulnerable):**
- `/api/addresses/index.ts` - GET/POST requests **skip token verification**
- `/api/addresses/[...path].ts` - DELETE/POST requests **skip token verification**
- Pattern: Only forwards Authorization header without validation

### Impact

When unverified endpoints proxy requests to the Go backend without client-side token validation:

1. **No early rejection** of requests without Authorization header
2. **All errors are 401 Unauthorized from Go backend**, not caught at proxy layer
3. **Inconsistent user experience** - some endpoints return 401 immediately, others wait for backend response
4. **Delayed error detection** - network latency before discovering missing token
5. **Potential for confusion** - developers assume all protected endpoints verify tokens

### Real-World Scenario

A frontend user loses their session token:

1. Calls `GET /api/auth/me` → Vercel returns `401: Authentication required` (fast, local)
2. Calls `GET /api/addresses` → Vercel proxies to Go → Go returns `401: Unauthorized` (slow, from backend)
3. Calls `DELETE /api/addresses/123` → Vercel proxies to Go → Go returns `401: Unauthorized` (slow, from backend)

The inconsistency causes confusion and potential client-side retry logic failures.

---

## 2. Root Cause Analysis

### Why This Inconsistency Exists

1. **`/api/auth/me` (Current Pattern)**
   ```typescript
   // Verifies token at proxy layer
   const user = verifyToken(req);
   if (!user) {
     return res.status(401).json({ code: 'MISSING_TOKEN', message: 'Authentication required' });
   }
   ```
   - Best practice: Early validation, consistent error responses
   - Issue: Only endpoint following this pattern

2. **`/api/addresses/*` (Legacy Pattern)**
   ```typescript
   // No verification - just forward header
   headers: {
     ...(req.headers.authorization ? { 'Authorization': req.headers.authorization } : {}),
   },
   ```
   - Assumes Go backend handles all validation
   - Works, but inconsistent with `/api/auth/me` approach

3. **Other 2FA Endpoints**
   - `/api/auth/2fa/setup.ts` - No verification (protected operation)
   - `/api/auth/2fa/enable.ts` - No verification (protected operation)
   - `/api/auth/2fa/disable.ts` - No verification (protected operation)
   - `/api/auth/2fa/status.ts` - No verification (protected operation)

### Why This is a Problem

From the **Vercel proxy layer perspective**:

| Endpoint | Verified at Proxy | Benefit |
|----------|------------------|---------|
| `/api/auth/me` | ✅ Yes | Fast 401, consistent with API contract |
| `/api/addresses` | ❌ No | Inconsistent, slow 401 response |
| `/api/addresses/[...path]` | ❌ No | Inconsistent, slow 401 response |
| `/api/auth/2fa/*` | ❌ No | Should be verified for protected operations |

The **root cause** is lack of standardized middleware composition. Each endpoint manually builds its request, and some remember to call `verifyToken()` while others don't.

---

## 3. Proposed Solution

### 3.1 Core Strategy

**Apply the proven `/api/auth/me` pattern to ALL protected endpoints:**

1. Verify JWT token at Vercel proxy layer (not just forwarding)
2. Return `401` with consistent error structure
3. Only proxy to Go backend if token is valid
4. Maintain backward compatibility with Go backend

### 3.2 Implementation Approach

#### Phase 1: Protected Endpoints Requiring Authentication

Add token verification to:

1. **`/api/addresses/index.ts`** (GET/POST)
   - Required for: retrieving user's addresses, creating new addresses

2. **`/api/addresses/[...path].ts`** (DELETE/POST/PUT/PATCH)
   - Required for: deleting, verifying, setting primary address

3. **`/api/auth/2fa/setup.ts`** (POST)
   - Required for: generating TOTP secret (authenticated user only)

4. **`/api/auth/2fa/enable.ts`** (POST)
   - Required for: enabling TOTP (authenticated user only)

5. **`/api/auth/2fa/disable.ts`** (POST)
   - Required for: disabling TOTP (authenticated user only)

6. **`/api/auth/2fa/status.ts`** (GET)
   - Required for: checking 2FA status (authenticated user only)

#### Phase 2: Public Endpoints (No Changes)

These remain unchanged - they don't require authentication:

1. `/api/auth/login` - Public, no token needed
2. `/api/auth/register` - Public, no token needed
3. `/api/auth/2fa/verify-login` - Session-based (uses sessionId, not JWT)

### 3.3 Code Changes

#### Change 1: Update `/api/addresses/index.ts`

**Before:**
```typescript
export default async function handler(req: VercelRequest, res: VercelResponse) {
  const backendUrl = process.env.BACKEND_URL;

  if (!backendUrl) {
    return res.status(500).json({
      error: 'Server configuration error',
      message: 'Backend URL not configured',
    });
  }

  // No token verification - just forward
  if (req.method !== 'GET' && req.method !== 'POST') {
    return res.status(405).json({ error: 'Method Not Allowed' });
  }

  try {
    const url = `${backendUrl}/api/addresses`;

    const options: RequestInit = {
      method: req.method,
      headers: {
        'Content-Type': 'application/json',
        ...(req.headers.authorization ? { 'Authorization': req.headers.authorization } : {}),
      },
    };

    if (req.method === 'POST') {
      options.body = JSON.stringify(req.body);
    }

    const response = await fetch(url, options);
    const data = await response.json().catch(() => ({}));

    return res.status(response.status).json(data);
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    logger.error({ error: errorMessage }, 'Addresses proxy error');
    return res.status(500).json({
      error: 'Internal Server Error',
      message: 'Failed to connect to backend service',
    });
  }
}
```

**After:**
```typescript
import type { VercelRequest, VercelResponse } from '@vercel/node';
import { verifyToken } from '../../src/lib/auth-middleware.js';
import logger from '../../src/lib/logger.js';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  const backendUrl = process.env.BACKEND_URL;

  if (!backendUrl) {
    return res.status(500).json({
      error: 'Server configuration error',
      message: 'Backend URL not configured',
    });
  }

  // Verify JWT token - NEW
  const user = verifyToken(req);
  if (!user) {
    return res.status(401).json({ code: 'MISSING_TOKEN', message: 'Authentication required' });
  }

  if (req.method !== 'GET' && req.method !== 'POST') {
    return res.status(405).json({ error: 'Method Not Allowed' });
  }

  try {
    const url = `${backendUrl}/api/addresses`;

    const options: RequestInit = {
      method: req.method,
      headers: {
        'Content-Type': 'application/json',
        'Authorization': req.headers.authorization || '',
      },
    };

    if (req.method === 'POST') {
      options.body = JSON.stringify(req.body);
    }

    const response = await fetch(url, options);
    const data = await response.json().catch(() => ({}));

    return res.status(response.status).json(data);
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    logger.error({ error: errorMessage }, 'Addresses proxy error');
    return res.status(500).json({
      error: 'Internal Server Error',
      message: 'Failed to connect to backend service',
    });
  }
}
```

**Changes:**
- Line 2: Import `verifyToken` from auth-middleware
- Lines 18-22: Add token verification check, return 401 if missing/invalid

#### Change 2: Update `/api/addresses/[...path].ts`

**Before:**
```typescript
export default async function handler(req: VercelRequest, res: VercelResponse) {
  const backendUrl = process.env.BACKEND_URL;
  const { path } = req.query;

  if (!backendUrl) {
    return res.status(500).json({
      error: 'Server configuration error',
      message: 'Backend URL not configured',
    });
  }

  // No token verification
  try {
    const pathStr = Array.isArray(path) ? path.join('/') : path;
    const url = `${backendUrl}/api/addresses/${pathStr}`;

    const options: RequestInit = {
      method: req.method,
      headers: {
        'Content-Type': 'application/json',
        ...(req.headers.authorization ? { 'Authorization': req.headers.authorization } : {}),
      },
    };

    if (req.body && (req.method === 'POST' || req.method === 'PUT' || req.method === 'PATCH')) {
      options.body = JSON.stringify(req.body);
    }

    const response = await fetch(url, options);
    const data = await response.json().catch(() => ({}));

    return res.status(response.status).json(data);
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    logger.error({ error: errorMessage }, 'Addresses sub-path proxy error');
    return res.status(500).json({
      error: 'Internal Server Error',
      message: 'Failed to connect to backend service',
    });
  }
}
```

**After:**
```typescript
import type { VercelRequest, VercelResponse } from '@vercel/node';
import { verifyToken } from '../../src/lib/auth-middleware.js';
import logger from '../../src/lib/logger.js';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  const backendUrl = process.env.BACKEND_URL;
  const { path } = req.query;

  if (!backendUrl) {
    return res.status(500).json({
      error: 'Server configuration error',
      message: 'Backend URL not configured',
    });
  }

  // Verify JWT token - NEW
  const user = verifyToken(req);
  if (!user) {
    return res.status(401).json({ code: 'MISSING_TOKEN', message: 'Authentication required' });
  }

  try {
    const pathStr = Array.isArray(path) ? path.join('/') : path;
    const url = `${backendUrl}/api/addresses/${pathStr}`;

    const options: RequestInit = {
      method: req.method,
      headers: {
        'Content-Type': 'application/json',
        'Authorization': req.headers.authorization || '',
      },
    };

    if (req.body && (req.method === 'POST' || req.method === 'PUT' || req.method === 'PATCH')) {
      options.body = JSON.stringify(req.body);
    }

    const response = await fetch(url, options);
    const data = await response.json().catch(() => ({}));

    return res.status(response.status).json(data);
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    logger.error({ error: errorMessage }, 'Addresses sub-path proxy error');
    return res.status(500).json({
      error: 'Internal Server Error',
      message: 'Failed to connect to backend service',
    });
  }
}
```

**Changes:**
- Lines 1-2: Add imports for `verifyToken` and `logger`
- Lines 17-21: Add token verification check, return 401 if missing/invalid

#### Change 3: Update `/api/auth/2fa/setup.ts`

**Before:**
```typescript
export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method Not Allowed' });
  }

  const backendUrl = process.env.BACKEND_URL;

  if (!backendUrl) {
    return res.status(500).json({
      error: 'Server configuration error',
      message: 'Backend URL not configured. Please set BACKEND_URL environment variable.'
    });
  }

  try {
    // No token verification
    const response = await fetch(`${backendUrl}/api/auth/2fa/setup`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': req.headers.authorization || '',
      },
    });

    const data = await response.json();

    return res.status(response.status).json(data);
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    console.error('Proxy error:', errorMessage);
    return res.status(500).json({
      error: 'Internal Server Error',
      message: 'Failed to connect to backend service'
    });
  }
}
```

**After:**
```typescript
import type { VercelRequest, VercelResponse } from '@vercel/node';
import { verifyToken } from '../../src/lib/auth-middleware.js';
import logger from '../../src/lib/logger.js';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method Not Allowed' });
  }

  const backendUrl = process.env.BACKEND_URL;

  if (!backendUrl) {
    return res.status(500).json({
      error: 'Server configuration error',
      message: 'Backend URL not configured. Please set BACKEND_URL environment variable.'
    });
  }

  // Verify JWT token - NEW
  const user = verifyToken(req);
  if (!user) {
    return res.status(401).json({ code: 'MISSING_TOKEN', message: 'Authentication required' });
  }

  try {
    const response = await fetch(`${backendUrl}/api/auth/2fa/setup`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': req.headers.authorization || '',
      },
    });

    const data = await response.json();

    return res.status(response.status).json(data);
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    logger.error({ error: errorMessage }, 'Setup 2FA proxy error');
    return res.status(500).json({
      error: 'Internal Server Error',
      message: 'Failed to connect to backend service'
    });
  }
}
```

**Changes:**
- Lines 1-3: Add imports
- Lines 21-25: Add token verification
- Line 39: Replace `console.error` with `logger.error`

#### Change 4: Update `/api/auth/2fa/enable.ts`

**Before:**
```typescript
export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method Not Allowed' });
  }

  const backendUrl = process.env.BACKEND_URL;

  if (!backendUrl) {
    return res.status(500).json({
      error: 'Server configuration error',
      message: 'Backend URL not configured. Please set BACKEND_URL environment variable.'
    });
  }

  try {
    // No token verification
    const response = await fetch(`${backendUrl}/api/auth/2fa/enable`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': req.headers.authorization || '',
      },
      body: JSON.stringify(req.body),
    });

    const data = await response.json();

    return res.status(response.status).json(data);
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    console.error('Proxy error:', errorMessage);
    return res.status(500).json({
      error: 'Internal Server Error',
      message: 'Failed to connect to backend service'
    });
  }
}
```

**After:**
```typescript
import type { VercelRequest, VercelResponse } from '@vercel/node';
import { verifyToken } from '../../src/lib/auth-middleware.js';
import logger from '../../src/lib/logger.js';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method Not Allowed' });
  }

  const backendUrl = process.env.BACKEND_URL;

  if (!backendUrl) {
    return res.status(500).json({
      error: 'Server configuration error',
      message: 'Backend URL not configured. Please set BACKEND_URL environment variable.'
    });
  }

  // Verify JWT token - NEW
  const user = verifyToken(req);
  if (!user) {
    return res.status(401).json({ code: 'MISSING_TOKEN', message: 'Authentication required' });
  }

  try {
    const response = await fetch(`${backendUrl}/api/auth/2fa/enable`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': req.headers.authorization || '',
      },
      body: JSON.stringify(req.body),
    });

    const data = await response.json();

    return res.status(response.status).json(data);
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    logger.error({ error: errorMessage }, 'Enable 2FA proxy error');
    return res.status(500).json({
      error: 'Internal Server Error',
      message: 'Failed to connect to backend service'
    });
  }
}
```

**Changes:**
- Lines 1-3: Add imports
- Lines 21-25: Add token verification
- Line 43: Replace `console.error` with `logger.error`

#### Change 5: Update `/api/auth/2fa/disable.ts`

**Before:**
```typescript
export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method Not Allowed' });
  }

  const backendUrl = process.env.BACKEND_URL;

  if (!backendUrl) {
    return res.status(500).json({
      error: 'Server configuration error',
      message: 'Backend URL not configured. Please set BACKEND_URL environment variable.'
    });
  }

  try {
    // No token verification
    const response = await fetch(`${backendUrl}/api/auth/2fa/disable`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': req.headers.authorization || '',
      },
      body: JSON.stringify(req.body),
    });

    const data = await response.json();

    return res.status(response.status).json(data);
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    console.error('Proxy error:', errorMessage);
    return res.status(500).json({
      error: 'Internal Server Error',
      message: 'Failed to connect to backend service'
    });
  }
}
```

**After:**
```typescript
import type { VercelRequest, VercelResponse } from '@vercel/node';
import { verifyToken } from '../../src/lib/auth-middleware.js';
import logger from '../../src/lib/logger.js';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method Not Allowed' });
  }

  const backendUrl = process.env.BACKEND_URL;

  if (!backendUrl) {
    return res.status(500).json({
      error: 'Server configuration error',
      message: 'Backend URL not configured. Please set BACKEND_URL environment variable.'
    });
  }

  // Verify JWT token - NEW
  const user = verifyToken(req);
  if (!user) {
    return res.status(401).json({ code: 'MISSING_TOKEN', message: 'Authentication required' });
  }

  try {
    const response = await fetch(`${backendUrl}/api/auth/2fa/disable`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': req.headers.authorization || '',
      },
      body: JSON.stringify(req.body),
    });

    const data = await response.json();

    return res.status(response.status).json(data);
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    logger.error({ error: errorMessage }, 'Disable 2FA proxy error');
    return res.status(500).json({
      error: 'Internal Server Error',
      message: 'Failed to connect to backend service'
    });
  }
}
```

**Changes:**
- Lines 1-3: Add imports
- Lines 21-25: Add token verification
- Line 43: Replace `console.error` with `logger.error`

#### Change 6: Update `/api/auth/2fa/status.ts`

**Before:**
```typescript
export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'GET') {
    return res.status(405).json({ error: 'Method Not Allowed' });
  }

  const backendUrl = process.env.BACKEND_URL;

  if (!backendUrl) {
    return res.status(500).json({
      error: 'Server configuration error',
      message: 'Backend URL not configured. Please set BACKEND_URL environment variable.'
    });
  }

  try {
    // No token verification
    const response = await fetch(`${backendUrl}/api/auth/2fa/status`, {
      method: 'GET',
      headers: {
        'Authorization': req.headers.authorization || '',
        'Content-Type': 'application/json',
      },
    });

    const data = await response.json();

    return res.status(response.status).json(data);
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    console.error('Proxy error:', errorMessage);
    return res.status(500).json({
      error: 'Internal Server Error',
      message: 'Failed to connect to backend service'
    });
  }
}
```

**After:**
```typescript
import type { VercelRequest, VercelResponse } from '@vercel/node';
import { verifyToken } from '../../src/lib/auth-middleware.js';
import logger from '../../src/lib/logger.js';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'GET') {
    return res.status(405).json({ error: 'Method Not Allowed' });
  }

  const backendUrl = process.env.BACKEND_URL;

  if (!backendUrl) {
    return res.status(500).json({
      error: 'Server configuration error',
      message: 'Backend URL not configured. Please set BACKEND_URL environment variable.'
    });
  }

  // Verify JWT token - NEW
  const user = verifyToken(req);
  if (!user) {
    return res.status(401).json({ code: 'MISSING_TOKEN', message: 'Authentication required' });
  }

  try {
    const response = await fetch(`${backendUrl}/api/auth/2fa/status`, {
      method: 'GET',
      headers: {
        'Authorization': req.headers.authorization || '',
        'Content-Type': 'application/json',
      },
    });

    const data = await response.json();

    return res.status(response.status).json(data);
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    logger.error({ error: errorMessage }, 'Get 2FA status proxy error');
    return res.status(500).json({
      error: 'Internal Server Error',
      message: 'Failed to connect to backend service'
    });
  }
}
```

**Changes:**
- Lines 1-3: Add imports
- Lines 19-23: Add token verification
- Line 39: Replace `console.error` with `logger.error`

### 3.4 Summary of Changes

| File | Change | Lines Changed |
|------|--------|---------------|
| `/api/addresses/index.ts` | Add token verification | +2 (import), +4 (verify) = 6 lines |
| `/api/addresses/[...path].ts` | Add token verification | +2 (import), +4 (verify) = 6 lines |
| `/api/auth/2fa/setup.ts` | Add token verification, improve logging | +2 (import), +4 (verify), 1 (logger) = 7 lines |
| `/api/auth/2fa/enable.ts` | Add token verification, improve logging | +2 (import), +4 (verify), 1 (logger) = 7 lines |
| `/api/auth/2fa/disable.ts` | Add token verification, improve logging | +2 (import), +4 (verify), 1 (logger) = 7 lines |
| `/api/auth/2fa/status.ts` | Add token verification, improve logging | +2 (import), +4 (verify), 1 (logger) = 7 lines |

**Total: ~41 lines changed across 6 files**

---

## 4. Test Strategy

### 4.1 Test Categories

#### Unit Tests: Token Verification

Test that all protected endpoints now verify tokens properly.

**File:** `/api/addresses/addresses.test.ts` (existing, update)

Add test cases:

```typescript
describe('JWT Token Verification', () => {
  describe('index.ts', () => {
    it('should return 401 when Authorization header is missing', async () => {
      process.env.BACKEND_URL = 'http://localhost:8081';
      const handler = await import('./index.js').then(m => m.default);

      const req = {
        method: 'GET',
        headers: {}, // No Authorization header
      } as any;

      const res = {
        status: vi.fn().mockReturnThis(),
        json: vi.fn().mockReturnThis(),
      } as any;

      await handler(req, res);

      expect(res.status).toHaveBeenCalledWith(401);
      expect(res.json).toHaveBeenCalledWith({
        code: 'MISSING_TOKEN',
        message: 'Authentication required'
      });
    });

    it('should return 401 when Authorization header is invalid', async () => {
      process.env.BACKEND_URL = 'http://localhost:8081';
      const handler = await import('./index.js').then(m => m.default);

      const req = {
        method: 'GET',
        headers: {
          authorization: 'InvalidToken123' // Missing "Bearer " prefix
        },
      } as any;

      const res = {
        status: vi.fn().mockReturnThis(),
        json: vi.fn().mockReturnThis(),
      } as any;

      await handler(req, res);

      expect(res.status).toHaveBeenCalledWith(401);
      expect(res.json).toHaveBeenCalledWith({
        code: 'MISSING_TOKEN',
        message: 'Authentication required'
      });
    });

    it('should return 401 when Bearer token is invalid (malformed JWT)', async () => {
      process.env.BACKEND_URL = 'http://localhost:8081';
      const handler = await import('./index.js').then(m => m.default);

      const req = {
        method: 'GET',
        headers: {
          authorization: 'Bearer invalid.jwt.token'
        },
      } as any;

      const res = {
        status: vi.fn().mockReturnThis(),
        json: vi.fn().mockReturnThis(),
      } as any;

      await handler(req, res);

      expect(res.status).toHaveBeenCalledWith(401);
      expect(res.json).toHaveBeenCalledWith({
        code: 'MISSING_TOKEN',
        message: 'Authentication required'
      });
    });

    it('should proxy to backend when valid token is present', async () => {
      process.env.BACKEND_URL = 'http://localhost:8081';
      process.env.JWT_SECRET = 'test-secret-key-at-least-32-bytes-long-1234567890';

      // Create a valid JWT token
      const jwt = await import('jsonwebtoken');
      const token = jwt.default.sign(
        { userId: 1, email: 'user@example.com' },
        process.env.JWT_SECRET,
        { expiresIn: '24h' }
      );

      const handler = await import('./index.js').then(m => m.default);

      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: vi.fn().mockResolvedValue({ addresses: [] }),
      });

      const req = {
        method: 'GET',
        headers: {
          authorization: `Bearer ${token}`
        },
      } as any;

      const res = {
        status: vi.fn().mockReturnThis(),
        json: vi.fn().mockReturnThis(),
      } as any;

      await handler(req, res);

      // Should call backend, not return 401
      expect(global.fetch).toHaveBeenCalled();
      expect(res.status).toHaveBeenCalledWith(200);
    });
  });

  describe('[...path].ts', () => {
    it('should return 401 when Authorization header is missing', async () => {
      process.env.BACKEND_URL = 'http://localhost:8081';
      const handler = await import('./[...path].js').then(m => m.default);

      const req = {
        method: 'DELETE',
        query: { path: ['123'] },
        headers: {}, // No Authorization header
      } as any;

      const res = {
        status: vi.fn().mockReturnThis(),
        json: vi.fn().mockReturnThis(),
      } as any;

      await handler(req, res);

      expect(res.status).toHaveBeenCalledWith(401);
      expect(res.json).toHaveBeenCalledWith({
        code: 'MISSING_TOKEN',
        message: 'Authentication required'
      });
    });

    it('should proxy to backend when valid token is present', async () => {
      process.env.BACKEND_URL = 'http://localhost:8081';
      process.env.JWT_SECRET = 'test-secret-key-at-least-32-bytes-long-1234567890';

      const jwt = await import('jsonwebtoken');
      const token = jwt.default.sign(
        { userId: 1, email: 'user@example.com' },
        process.env.JWT_SECRET,
        { expiresIn: '24h' }
      );

      const handler = await import('./[...path].js').then(m => m.default);

      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: vi.fn().mockResolvedValue({ message: 'Deleted' }),
      });

      const req = {
        method: 'DELETE',
        query: { path: ['123'] },
        headers: {
          authorization: `Bearer ${token}`
        },
      } as any;

      const res = {
        status: vi.fn().mockReturnThis(),
        json: vi.fn().mockReturnThis(),
      } as any;

      await handler(req, res);

      expect(global.fetch).toHaveBeenCalled();
      expect(res.status).toHaveBeenCalledWith(200);
    });
  });
});
```

#### Integration Tests: 2FA Endpoints

Create a new test file: `/api/auth/2fa/2fa-token-verification.test.ts`

```typescript
import { describe, it, expect, vi, beforeEach } from 'vitest';
import jwt from 'jsonwebtoken';

describe('/api/auth/2fa - Token Verification', () => {
  beforeEach(() => {
    const originalEnv = { ...process.env };
    process.env = { ...originalEnv };
    vi.clearAllMocks();
  });

  async function createValidToken() {
    const JWT_SECRET = process.env.JWT_SECRET || 'test-secret-key-at-least-32-bytes-long-1234567890';
    return jwt.sign(
      { userId: 1, email: 'user@example.com' },
      JWT_SECRET,
      { expiresIn: '24h' }
    );
  }

  describe('setup.ts', () => {
    it('should return 401 when token is missing', async () => {
      process.env.BACKEND_URL = 'http://localhost:8081';
      process.env.JWT_SECRET = 'test-secret-key-at-least-32-bytes-long-1234567890';

      const handler = await import('./setup.js').then(m => m.default);

      const req = {
        method: 'POST',
        headers: {},
        body: {},
      } as any;

      const res = {
        status: vi.fn().mockReturnThis(),
        json: vi.fn().mockReturnThis(),
      } as any;

      await handler(req, res);

      expect(res.status).toHaveBeenCalledWith(401);
      expect(res.json).toHaveBeenCalledWith({
        code: 'MISSING_TOKEN',
        message: 'Authentication required'
      });
    });

    it('should proxy to backend when valid token provided', async () => {
      process.env.BACKEND_URL = 'http://localhost:8081';
      process.env.JWT_SECRET = 'test-secret-key-at-least-32-bytes-long-1234567890';

      const token = await createValidToken();
      const handler = await import('./setup.js').then(m => m.default);

      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: vi.fn().mockResolvedValue({ secret: 'xxx', otpauth: 'otpauth://...' }),
      });

      const req = {
        method: 'POST',
        headers: { authorization: `Bearer ${token}` },
        body: {},
      } as any;

      const res = {
        status: vi.fn().mockReturnThis(),
        json: vi.fn().mockReturnThis(),
      } as any;

      await handler(req, res);

      expect(global.fetch).toHaveBeenCalled();
      expect(res.status).toHaveBeenCalledWith(200);
    });
  });

  describe('enable.ts', () => {
    it('should return 401 when token is missing', async () => {
      process.env.BACKEND_URL = 'http://localhost:8081';
      process.env.JWT_SECRET = 'test-secret-key-at-least-32-bytes-long-1234567890';

      const handler = await import('./enable.js').then(m => m.default);

      const req = {
        method: 'POST',
        headers: {},
        body: { token: '123456' },
      } as any;

      const res = {
        status: vi.fn().mockReturnThis(),
        json: vi.fn().mockReturnThis(),
      } as any;

      await handler(req, res);

      expect(res.status).toHaveBeenCalledWith(401);
    });
  });

  describe('disable.ts', () => {
    it('should return 401 when token is missing', async () => {
      process.env.BACKEND_URL = 'http://localhost:8081';
      process.env.JWT_SECRET = 'test-secret-key-at-least-32-bytes-long-1234567890';

      const handler = await import('./disable.js').then(m => m.default);

      const req = {
        method: 'POST',
        headers: {},
        body: {},
      } as any;

      const res = {
        status: vi.fn().mockReturnThis(),
        json: vi.fn().mockReturnThis(),
      } as any;

      await handler(req, res);

      expect(res.status).toHaveBeenCalledWith(401);
    });
  });

  describe('status.ts', () => {
    it('should return 401 when token is missing', async () => {
      process.env.BACKEND_URL = 'http://localhost:8081';
      process.env.JWT_SECRET = 'test-secret-key-at-least-32-bytes-long-1234567890';

      const handler = await import('./status.js').then(m => m.default);

      const req = {
        method: 'GET',
        headers: {},
      } as any;

      const res = {
        status: vi.fn().mockReturnThis(),
        json: vi.fn().mockReturnThis(),
      } as any;

      await handler(req, res);

      expect(res.status).toHaveBeenCalledWith(401);
    });

    it('should proxy to backend when valid token provided', async () => {
      process.env.BACKEND_URL = 'http://localhost:8081';
      process.env.JWT_SECRET = 'test-secret-key-at-least-32-bytes-long-1234567890';

      const token = await createValidToken();
      const handler = await import('./status.js').then(m => m.default);

      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: vi.fn().mockResolvedValue({ enabled: true }),
      });

      const req = {
        method: 'GET',
        headers: { authorization: `Bearer ${token}` },
      } as any;

      const res = {
        status: vi.fn().mockReturnThis(),
        json: vi.fn().mockReturnThis(),
      } as any;

      await handler(req, res);

      expect(global.fetch).toHaveBeenCalled();
      expect(res.status).toHaveBeenCalledWith(200);
    });
  });
});
```

### 4.2 Test Coverage Requirements

- **New code coverage:** 100% for all verification logic
- **Existing tests:** Must continue to pass (backward compatibility)
- **Edge cases:**
  - Missing Authorization header
  - Invalid Bearer token format
  - Expired JWT tokens
  - Malformed JWT tokens
  - Valid tokens (happy path)

### 4.3 Testing Checklist

```
Token Verification Tests:
- [ ] Missing Authorization header → 401
- [ ] Invalid Bearer format → 401
- [ ] Expired token → 401
- [ ] Malformed JWT → 401
- [ ] Valid token → proxies to backend
- [ ] Correct error message in 401 response

Backend Proxying Tests:
- [ ] Authorization header forwarded correctly
- [ ] Request body preserved
- [ ] Response status codes forwarded
- [ ] Response data forwarded
- [ ] Network errors handled

Integration Tests:
- [ ] All 6 protected endpoints verify tokens
- [ ] Existing 2FA functionality still works
- [ ] Address management operations work with valid token
- [ ] 401 errors consistent across all endpoints
```

---

## 5. Deployment Considerations

### 5.1 Pre-Deployment Checklist

Before deploying to production:

1. **All tests pass:**
   ```bash
   npm run test
   ```
   - Unit tests: 100% coverage for new code
   - Integration tests: All scenarios covered
   - No breaking changes to existing tests

2. **Type safety:**
   ```bash
   npx tsc --noEmit
   ```
   - No TypeScript errors
   - Proper imports and exports

3. **Code review:**
   - Security: Token validation logic correct
   - Consistency: All endpoints follow same pattern
   - Performance: No additional network calls

4. **Environment variables confirmed:**
   - `JWT_SECRET` set to minimum 32 bytes
   - `BACKEND_URL` properly configured
   - Both production and staging validated

### 5.2 Deployment Strategy

**Phased Rollout (Recommended):**

1. **Phase 1: Staging Environment**
   - Deploy to staging with the 6 changes
   - Run full test suite
   - Manual testing: Create address, verify address, delete address
   - Verify 2FA endpoints work correctly
   - Confirm no regressions

2. **Phase 2: Production Canary (Optional)**
   - Deploy to 5-10% of production traffic
   - Monitor error rates: no increase in 401s
   - Monitor latency: no performance impact
   - Monitor logs: correct error messages appearing

3. **Phase 3: Full Production**
   - Deploy to 100% of production traffic
   - Monitor for 24-48 hours
   - Verify no increase in failed requests
   - Check error logs for unexpected patterns

### 5.3 Monitoring Plan

After deployment, monitor these metrics:

**Error Rates:**
- Track 401 Unauthorized responses by endpoint
- Verify they occur at Vercel layer (not Go backend)
- Alert if 401 rate increases unexpectedly

**Performance:**
- Response time should be faster (less backend calls needed)
- No additional latency introduced

**User Experience:**
- Frontend should handle 401 errors consistently
- Session refresh flows should work as expected

---

## 6. Rollback Plan

If issues are discovered after deployment:

### 6.1 Rollback Procedure

**Immediate Rollback (If Critical Issues):**

1. Revert commits:
   ```bash
   git revert <commit-hash>
   ```

2. Re-deploy previous version:
   ```bash
   git push
   # Vercel auto-deploys
   ```

3. Verify rollback successful:
   - Endpoints responding normally
   - No 401 errors at proxy layer
   - All functionality working

**Timeline:** Can be reverted within 5 minutes

### 6.2 Issue Scenarios & Recovery

| Scenario | Symptom | Solution |
|----------|---------|----------|
| 401s for valid tokens | Users getting logged out | Check JWT_SECRET is correct, redeploy |
| No token verification | Addresses accessible without auth | Rollback immediately |
| Broken address operations | 405 errors from proxy | Check method handling logic, verify update |
| Performance degradation | Slow 401 responses | Check logger performance, remove verbose logging |

### 6.3 Validation After Rollback

After rolling back, verify:

1. Addresses API works without token verification (reverted behavior)
2. /api/auth/me still verifies tokens (unaffected)
3. No error messages in logs about token verification
4. Performance metrics return to baseline

---

## 7. Acceptance Criteria

### 7.1 Functional Requirements

- [ ] All 6 protected endpoints verify JWT tokens at Vercel layer
- [ ] Missing token returns 401 with code `MISSING_TOKEN`
- [ ] Invalid token returns 401 with code `MISSING_TOKEN`
- [ ] Valid token is forwarded to Go backend without modification
- [ ] All endpoints follow identical verification pattern
- [ ] No unprotected endpoints are affected

### 7.2 Quality Requirements

- [ ] 100% test coverage for verification logic
- [ ] All existing tests continue to pass
- [ ] No console.log statements (use logger)
- [ ] No hardcoded values
- [ ] No mutation of request objects
- [ ] Consistent error response format

### 7.3 Performance Requirements

- [ ] Response time faster or equal (less backend overhead)
- [ ] No additional network calls introduced
- [ ] Logger calls don't impact performance

### 7.4 Security Requirements

- [ ] JWT_SECRET used correctly (no hardcoding)
- [ ] Token verification cannot be bypassed
- [ ] Error messages don't leak sensitive data
- [ ] No token stored in response body

### 7.5 Documentation Requirements

- [ ] Code changes documented with comments
- [ ] Error codes documented in code
- [ ] Test cases clearly explain what they test
- [ ] This OpenSpec document kept updated

---

## 8. Implementation Timeline

| Phase | Task | Duration | Owner |
|-------|------|----------|-------|
| 1 | Create test file with all test cases | 30 min | TDD Guide |
| 2 | Implement changes to 6 endpoints | 45 min | Developer |
| 3 | Run full test suite | 15 min | Developer |
| 4 | Code review for security | 20 min | Security Reviewer |
| 5 | Integration testing (staging) | 30 min | QA |
| 6 | Deploy to production | 10 min | DevOps |
| 7 | Monitor for 24 hours | Ongoing | DevOps |

**Total Timeline:** ~2.5 hours implementation + 24 hours monitoring

---

## 9. Known Risks & Mitigations

### Risk 1: JWT_SECRET Not Set Correctly
**Impact:** All endpoints return 401, user locked out
**Probability:** Low (env var required at startup)
**Mitigation:** Pre-deployment environment check, test in staging first

### Risk 2: Backward Compatibility with Frontend
**Impact:** Frontend doesn't handle 401s from addresses endpoints
**Probability:** Very Low (401 already handled by auth/me)
**Mitigation:** Frontend already handles 401s, no changes needed

### Risk 3: Go Backend Changes Authorization Header Expectation
**Impact:** 401 errors from backend even with valid token
**Probability:** Very Low (header forwarded unchanged)
**Mitigation:** Request from backend team about their expectations

### Risk 4: Token Verification Adds Latency
**Impact:** Slower response times
**Probability:** Very Low (JWT verification is <1ms)
**Mitigation:** Monitor performance metrics, optimize if needed

---

## 10. Success Metrics

**After deployment, verify:**

1. **Token Verification Working:**
   - [ ] 401 response when no token provided
   - [ ] 401 response when invalid token provided
   - [ ] Successful proxy when valid token provided

2. **Consistency Achieved:**
   - [ ] All 6 protected endpoints follow same pattern
   - [ ] Error responses identical format
   - [ ] Same logging approach across endpoints

3. **No Regressions:**
   - [ ] All existing tests pass
   - [ ] Address operations work normally
   - [ ] 2FA operations work normally
   - [ ] Public endpoints (login, register) unaffected

4. **Performance Maintained:**
   - [ ] Response times same or faster
   - [ ] No increase in error rates
   - [ ] No increase in CPU/memory usage

---

## 11. Future Improvements

After this fix is deployed, consider:

1. **Middleware Factory Pattern**
   - Create reusable middleware factory for protected endpoints
   - Reduce code duplication across handlers

2. **Rate Limiting at Vercel Layer**
   - Apply rate limiting before proxying to backend
   - Reduce unnecessary backend calls

3. **Request/Response Logging**
   - Add structured logging for all proxy operations
   - Helps with debugging and monitoring

4. **HttpOnly Cookies**
   - Consider replacing token in localStorage with HttpOnly cookies
   - Reduces XSS attack surface

---

## Appendix A: File Change Summary

### Files to Modify (6 total)

1. `/api/addresses/index.ts` - Add token verification
2. `/api/addresses/[...path].ts` - Add token verification
3. `/api/auth/2fa/setup.ts` - Add token verification + logging
4. `/api/auth/2fa/enable.ts` - Add token verification + logging
5. `/api/auth/2fa/disable.ts` - Add token verification + logging
6. `/api/auth/2fa/status.ts` - Add token verification + logging

### Files to Create (1 total)

1. `/api/auth/2fa/2fa-token-verification.test.ts` - New test file

### Files to Update (1 total)

1. `/api/addresses/addresses.test.ts` - Add token verification tests

---

## Appendix B: Error Response Format

**Standard Error Response (401 Unauthorized):**

```json
{
  "code": "MISSING_TOKEN",
  "message": "Authentication required"
}
```

**Backend Error Response (for comparison):**

```json
{
  "error": "Unauthorized",
  "message": "Invalid token"
}
```

Note: Frontend should treat both as 401 Unauthorized and redirect to login.

---

## Appendix C: Testing Commands

```bash
# Run all tests
npm run test

# Run specific test file
npm run test -- api/addresses/addresses.test.ts

# Run with coverage
npm run test -- --coverage

# Run in watch mode
npm run test -- --watch

# Run 2FA tests specifically
npm run test -- api/auth/2fa/2fa-token-verification.test.ts
```

---

## Sign-Off

**Status:** Ready for Implementation
**Created:** 2026-01-27
**Last Updated:** 2026-01-27

**Reviewers Needed:**
- [ ] Security Reviewer - Verify token validation logic
- [ ] Code Reviewer - Verify code quality and consistency
- [ ] Backend Team - Confirm Go backend compatibility
- [ ] DevOps - Confirm deployment plan and monitoring

