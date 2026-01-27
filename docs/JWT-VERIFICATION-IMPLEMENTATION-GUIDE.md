# JWT Token Verification Implementation Guide

Quick reference for implementing the OpenSpec proposal.

## Pattern to Apply

Every protected endpoint should follow this structure:

```typescript
import type { VercelRequest, VercelResponse } from '@vercel/node';
import { verifyToken } from '../../src/lib/auth-middleware.js';
import logger from '../../src/lib/logger.js';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  // 1. Check HTTP method
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method Not Allowed' });
  }

  // 2. Verify backend URL
  const backendUrl = process.env.BACKEND_URL;
  if (!backendUrl) {
    return res.status(500).json({
      error: 'Server configuration error',
      message: 'Backend URL not configured. Please set BACKEND_URL environment variable.'
    });
  }

  // 3. Verify JWT token (NEW - lines 18-21)
  const user = verifyToken(req);
  if (!user) {
    return res.status(401).json({ code: 'MISSING_TOKEN', message: 'Authentication required' });
  }

  try {
    // 4. Proxy to backend
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

## Endpoint Checklist

Apply this pattern to each endpoint:

### ✅ /api/addresses/index.ts
- [x] Import verifyToken
- [x] Import logger
- [x] Add token verification check
- [x] Return 401 if no token
- [x] Replace console.error with logger.error

### ✅ /api/addresses/[...path].ts
- [x] Import verifyToken
- [x] Import logger
- [x] Add token verification check
- [x] Return 401 if no token
- [x] Replace console.error with logger.error (if present)

### ✅ /api/auth/2fa/setup.ts
- [x] Import verifyToken
- [x] Import logger
- [x] Add token verification check
- [x] Return 401 if no token
- [x] Replace console.error with logger.error

### ✅ /api/auth/2fa/enable.ts
- [x] Import verifyToken
- [x] Import logger
- [x] Add token verification check
- [x] Return 401 if no token
- [x] Replace console.error with logger.error

### ✅ /api/auth/2fa/disable.ts
- [x] Import verifyToken
- [x] Import logger
- [x] Add token verification check
- [x] Return 401 if no token
- [x] Replace console.error with logger.error

### ✅ /api/auth/2fa/status.ts
- [x] Import verifyToken
- [x] Import logger
- [x] Add token verification check
- [x] Return 401 if no token
- [x] Replace console.error with logger.error

## Testing Checklist

### Unit Tests - Token Verification

For each endpoint, add tests:

- [ ] Missing Authorization header → 401
- [ ] Invalid Bearer format → 401
- [ ] Expired token → 401
- [ ] Malformed JWT → 401
- [ ] Valid token → calls backend

### Integration Tests

- [ ] All endpoints reject unauthenticated requests
- [ ] All endpoints accept valid tokens
- [ ] Error response format consistent: `{ code: 'MISSING_TOKEN', message: 'Authentication required' }`
- [ ] Backend Authorization header forwarded unchanged

### Regression Tests

- [ ] Existing tests still pass
- [ ] Address operations work with auth
- [ ] 2FA operations work with auth
- [ ] Public endpoints (login, register) unaffected

## Command Reference

```bash
# Run all tests
npm run test

# Run specific test
npm run test -- api/addresses/addresses.test.ts

# Run with coverage
npm run test -- --coverage

# Check TypeScript
npx tsc --noEmit

# Lint
npm run lint
```

## Error Response Format

**Correct (with verification):**
```json
{
  "code": "MISSING_TOKEN",
  "message": "Authentication required"
}
```

**Status:** `401 Unauthorized`

## Key Points

1. **Token verification happens at Vercel layer** (before backend call)
2. **Error response is immediate** (no backend latency)
3. **All protected endpoints use same pattern** (consistency)
4. **No changes to public endpoints** (login, register)
5. **No changes to Go backend** (backward compatible)

## Verification Steps

After implementing each file:

1. Check imports are correct
2. Verify token check is before any backend calls
3. Verify 401 response format matches expected
4. Run tests: `npm run test`
5. Check TypeScript: `npx tsc --noEmit`

## Files Changed Summary

```
api/addresses/index.ts                          +6 lines (import + verify)
api/addresses/[...path].ts                      +6 lines (import + verify)
api/auth/2fa/setup.ts                           +7 lines (import + verify + logger)
api/auth/2fa/enable.ts                          +7 lines (import + verify + logger)
api/auth/2fa/disable.ts                         +7 lines (import + verify + logger)
api/auth/2fa/status.ts                          +7 lines (import + verify + logger)
api/addresses/addresses.test.ts                 +Tests for verification
api/auth/2fa/2fa-token-verification.test.ts     +New test file

Total: ~41 lines of implementation + comprehensive tests
```

## Deployment Steps

1. **Locally:**
   ```bash
   npm run test              # All tests pass
   npx tsc --noEmit         # No TS errors
   ```

2. **Staging:**
   - Deploy changes
   - Run full test suite
   - Manual testing: addresses, 2FA operations
   - Verify 401 responses appear immediately

3. **Production:**
   - Deploy to staging first
   - Monitor 24 hours
   - Deploy to production
   - Monitor error rates and latency

## Rollback

If issues arise:

```bash
git revert <commit-hash>
git push
# Vercel auto-deploys reverted version
```

---

**Full Details:** See `/docs/OPENSPEC-JWT-VERIFICATION.md`
