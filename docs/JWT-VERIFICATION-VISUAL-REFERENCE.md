# JWT Token Verification - Visual Reference

## Architecture Diagram

### Before (Current State)

```
Client Request
    |
    v
Vercel Proxy Layer
    |
    +--- /api/auth/me
    |    [✓ Verify Token]
    |    └─> 401 (fast)
    |        or
    |    └─> Forward to Go
    |
    +--- /api/addresses/*
    |    [✗ No Verification]
    |    └─> Forward to Go
         └─> 401 from Go (slow)
    |
    +--- /api/auth/2fa/*
         [✗ No Verification]
         └─> Forward to Go
             └─> 401 from Go (slow)
```

### After (Proposed State)

```
Client Request
    |
    v
Vercel Proxy Layer
    |
    +--- /api/auth/me
    |    [✓ Verify Token]
    |    └─> 401 (fast) or Forward to Go
    |
    +--- /api/addresses/*
    |    [✓ Verify Token] ← NEW
    |    └─> 401 (fast) or Forward to Go
    |
    +--- /api/auth/2fa/*
         [✓ Verify Token] ← NEW
         └─> 401 (fast) or Forward to Go
```

## Error Response Timeline

### Before: Slow 401 (No Verification)

```
Timeline:
0ms     Client sends request without token
        |
        +-- Vercel receives request
        |
        +-- Vercel SKIPS verification
        |
        +-- Vercel forwards to Go backend (~50-200ms)
        |
50-200ms +-- Go backend checks Authorization header
        |
        +-- Go backend returns 401
        |
100-250ms +-- Client receives 401 error
         └─> Slow response with backend latency
```

### After: Fast 401 (With Verification)

```
Timeline:
0ms     Client sends request without token
        |
        +-- Vercel receives request
        |
        +-- Vercel verifies token (<1ms)
        |
<1ms    +-- Token invalid/missing
        |
        +-- Vercel returns 401
        |
<1ms    └─> Client receives 401 error
         └─> Fast response, no backend call
```

## Code Changes at a Glance

### Generic Protected Endpoint Template

```typescript
// BEFORE (Vulnerable Pattern)
export default async function handler(req: VercelRequest, res: VercelResponse) {
  // ... method and config checks ...

  try {
    const response = await fetch(backendUrl, {
      headers: {
        'Authorization': req.headers.authorization || '', // Just forward
      },
    });
    // ...
  } catch (error) {
    // ...
  }
}
```

```typescript
// AFTER (Secure Pattern)
import { verifyToken } from '../../src/lib/auth-middleware.js';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  // ... method and config checks ...

  // ✓ NEW: Verify token before backend call
  const user = verifyToken(req);
  if (!user) {
    return res.status(401).json({ code: 'MISSING_TOKEN', message: 'Authentication required' });
  }

  try {
    const response = await fetch(backendUrl, {
      headers: {
        'Authorization': req.headers.authorization || '',
      },
    });
    // ...
  } catch (error) {
    // ...
  }
}
```

## File Changes Matrix

```
File                               | Status | Changes
-----------------------------------+--------+---------
/api/addresses/index.ts            | Modify | +import +verify (6 lines)
/api/addresses/[...path].ts        | Modify | +import +verify (6 lines)
/api/auth/2fa/setup.ts             | Modify | +import +verify +logger (7 lines)
/api/auth/2fa/enable.ts            | Modify | +import +verify +logger (7 lines)
/api/auth/2fa/disable.ts           | Modify | +import +verify +logger (7 lines)
/api/auth/2fa/status.ts            | Modify | +import +verify +logger (7 lines)
/api/addresses/addresses.test.ts   | Update | +token verification tests
/api/auth/2fa/2fa-token-v...ts     | Create | New test file
```

## Test Coverage Map

```
Test Type           | File                              | Coverage
--------------------+-----------------------------------+----------
Unit Tests          | addresses.test.ts                 | 100%
Unit Tests          | 2fa-token-verification.test.ts    | 100%
Integration Tests   | addresses.test.ts                 | 100%
Regression Tests    | Existing test suite               | No changes
```

## Endpoint Classification

### Public Endpoints (No Changes)

```
/api/auth/login              [Public]
/api/auth/register           [Public]
/api/auth/2fa/verify-login   [Session-based]
```

### Protected Endpoints (Add Verification)

```
/api/auth/me                 [Protected] ✓ Already verifies
/api/addresses               [Protected] ✗ Need to add
/api/addresses/[id]          [Protected] ✗ Need to add
/api/addresses/[id]/verify   [Protected] ✗ Need to add
/api/addresses/[id]/primary  [Protected] ✗ Need to add
/api/auth/2fa/setup          [Protected] ✗ Need to add
/api/auth/2fa/enable         [Protected] ✗ Need to add
/api/auth/2fa/disable        [Protected] ✗ Need to add
/api/auth/2fa/status         [Protected] ✗ Need to add
```

## Request/Response Flow

### Protected Endpoint with Valid Token

```
1. Client sends request
   POST /api/addresses
   Authorization: Bearer eyJhbGc...

2. Vercel proxy receives request
   ↓
3. Verify JWT token
   ✓ Valid token → Extract user info
   ↓
4. Forward to Go backend
   POST http://go-backend/api/addresses
   Authorization: Bearer eyJhbGc...
   ↓
5. Go backend processes request
   ↓
6. Return response to client
   Status: 200
   Body: { id: 1, address: "0x..." }
```

### Protected Endpoint with Missing Token

```
1. Client sends request (missing Authorization header)
   POST /api/addresses
   (no Authorization header)

2. Vercel proxy receives request
   ↓
3. Verify JWT token
   ✗ No token → Return error immediately
   ↓
4. Return 401 to client (NO backend call)
   Status: 401
   Body: { code: "MISSING_TOKEN", message: "Authentication required" }
```

### Protected Endpoint with Invalid Token

```
1. Client sends request
   POST /api/addresses
   Authorization: Bearer invalid.token.here

2. Vercel proxy receives request
   ↓
3. Verify JWT token
   ✗ Invalid token → Return error immediately
   ↓
4. Return 401 to client (NO backend call)
   Status: 401
   Body: { code: "MISSING_TOKEN", message: "Authentication required" }
```

## Performance Impact

### Latency Improvement

```
Operation: User calls /api/addresses without token

Before Implementation:
├─ Vercel processes: 10ms
├─ Network to backend: 50-150ms
├─ Backend validation: 20ms
└─ Total: ~80-180ms ✗

After Implementation:
├─ Vercel verification: <1ms
└─ Total: <1ms ✓

Improvement: 80-180x faster for 401 responses
```

### Network Calls Reduction

```
Scenario: User loses session (100 requests/minute)

Before:
├─ 100 requests reach Vercel
└─ ~80 unauthorized requests forwarded to Go (~80 backend calls)

After:
├─ 100 requests reach Vercel
└─ ~80 unauthorized requests rejected at Vercel (~0 backend calls)

Reduction: ~80 unnecessary backend calls per minute
```

## Testing Scenarios

```
Test Case                               | Expected | Implementation
----------------------------------------+----------+-----------------
No Authorization header                 | 401      | verifyToken() returns null
Authorization: InvalidFormat            | 401      | verifyToken() returns null
Authorization: Bearer invalid.jwt.here  | 401      | jwt.verify() throws
Authorization: Bearer [expired token]   | 401      | jwt.verify() throws
Authorization: Bearer [valid token]     | Proxy    | Forward to backend
```

## Implementation Checklist

```
Phase 1: Preparation
[ ] Review this OpenSpec proposal
[ ] Understand token verification logic
[ ] Set up test infrastructure

Phase 2: Implementation (6 endpoints)
[ ] /api/addresses/index.ts
[ ] /api/addresses/[...path].ts
[ ] /api/auth/2fa/setup.ts
[ ] /api/auth/2fa/enable.ts
[ ] /api/auth/2fa/disable.ts
[ ] /api/auth/2fa/status.ts

Phase 3: Testing
[ ] Write comprehensive tests
[ ] Run full test suite (100% coverage)
[ ] Verify no regressions

Phase 4: Deployment
[ ] Deploy to staging
[ ] Manual testing
[ ] Monitor error rates
[ ] Deploy to production
[ ] Monitor for 24 hours

Phase 5: Validation
[ ] All endpoints reject unauthenticated requests
[ ] All endpoints accept valid tokens
[ ] Error responses consistent
[ ] Performance metrics normal
```

## Error Message Consistency

### Current Inconsistency

```
/api/auth/me (with verification):
Status: 401
{
  "code": "MISSING_TOKEN",
  "message": "Authentication required"
}

/api/addresses (no verification):
Status: 200 (proxied)
(waits for Go backend)

/api/auth/2fa/setup (no verification):
Status: 200 (proxied)
(waits for Go backend)
```

### After Fix (Consistent)

```
All protected endpoints:
Status: 401
{
  "code": "MISSING_TOKEN",
  "message": "Authentication required"
}

(Same format, same latency)
```

## Rollback Path

```
If issues occur:

Current State          Rollback
├─ Commit hash ABC     → git revert ABC
├─ Deploy new version  → git push
└─ Verify rollback     → All working (no verification again)

Rollback time: ~5 minutes
```

---

**Full Details:** See `/docs/OPENSPEC-JWT-VERIFICATION.md`
