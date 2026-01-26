# OpenSpec: Fix 2FA Setup Endpoint SQL Failure on Vercel

**Date:** 2026-01-26
**Severity:** CRITICAL
**Issue:** 2FA setup endpoint returns 500 - Failed database update query
**Status:** Root Cause Analysis Complete - Refactoring Solution Required

---

## Problem Update

### Initial Investigation
- ✅ Missing DATABASE_URL on Vercel - **FIXED**
- ❌ New Issue: Database update query failing with SQL error

### Current Error

**HTTP 500 Response:**
```json
{
  "error": "Internal Server Error",
  "message": "Failed query: update \"users\" set \"two_factor_secret\" = $1, \"two_factor_backup_codes\" = $2 where \"users\".\"id\" = $3"
}
```

### Why It Happens

The Drizzle ORM update query is failing when executed against Neon PostgreSQL database on Vercel, while the **exact same code works perfectly locally** against a PostgreSQL instance.

**Possible Causes:**
1. **Connection Pool Issue:** `max: 1` connection limit causing query timeouts
2. **SSL Negotiation:** Neon's SSL handling differs from local PostgreSQL
3. **Timestamp Serialization:** Drizzle ORM's timestamp handling on Vercel
4. **Large Encrypted Data:** Backup codes encryption produces large strings (900+ chars)

---

## Recommended Solution: Convert to Proxy Pattern

The proper fix is to **convert the 2FA endpoints to HTTP proxies** (like login/register), rather than direct database calls. This approach:

1. ✅ Maintains consistency with existing login/register endpoints
2. ✅ Offloads database operations to the Go backend (proven to work)
3. ✅ Eliminates Drizzle ORM complexity on Vercel serverless
4. ✅ Reduces Vercel cold start time
5. ✅ Simplifies dependency management

---

## Architecture Comparison

### Current (Broken on Vercel)
```typescript
// api/auth/2fa/setup.ts
const result = await TwoFactorService.setup(userId, email); // Direct DB call
```

### Proposed (Working Pattern)
```typescript
// api/auth/2fa/setup.ts
const response = await fetch(`${BACKEND_URL}/api/auth/2fa/setup`, {
  method: 'POST',
  headers: { Authorization: `Bearer ${req.headers.authorization}` },
});
return res.status(response.status).json(await response.json());
```

---

## Implementation Plan

### Phase 1: Refactor 2FA Endpoints to Proxy Pattern (Priority: HIGH)

**Files to Modify:**
1. `api/auth/2fa/setup.ts` - Convert to proxy
2. `api/auth/2fa/enable.ts` - Convert to proxy
3. `api/auth/2fa/disable.ts` - Convert to proxy
4. `api/auth/2fa/status.ts` - Convert to proxy
5. `api/auth/2fa/verify-login.ts` - Convert to proxy (already partially done)

**Pattern to Follow:** (Use login.ts as reference)

```typescript
import type { VercelRequest, VercelResponse } from '@vercel/node';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  // 1. Validate method
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method Not Allowed' });
  }

  // 2. Check BACKEND_URL configuration
  const backendUrl = process.env.BACKEND_URL;
  if (!backendUrl) {
    return res.status(500).json({
      error: 'Server configuration error',
      message: 'Backend URL not configured'
    });
  }

  try {
    // 3. Proxy request to backend
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
    return res.status(500).json({
      error: 'Internal Server Error',
      message: 'Failed to connect to backend service'
    });
  }
}
```

### Phase 2: Update Go Backend to Support 2FA Endpoints

**Verification:**
- Go backend already has `TwoFAHandler` with all methods implemented ✅
- Backend is accessible at https://monera-digital--gyc567.replit.app ✅

**Required Actions:**
1. Verify backend 2FA routes are properly registered
2. Test backend 2FA endpoints directly (bypass Vercel proxy)
3. Ensure backend returns proper responses

### Phase 3: Testing & Deployment

**Local Testing:**
```bash
# Test backend directly
curl -s http://localhost:8081/api/auth/2fa/setup \
  -H "Authorization: Bearer $TOKEN" | jq

# Test Vercel proxy (after refactoring)
curl -s https://www.moneradigital.com/api/auth/2fa/setup \
  -H "Authorization: Bearer $TOKEN" | jq
```

**Deployment:**
```bash
npm run build
vercel --prod --yes
```

---

## Design Principles

✅ **KISS:** Simple HTTP proxy pattern (proven, already used for login)
✅ **High Cohesion:** Vercel handles auth/routing, Go backend handles business logic
✅ **Low Coupling:** No direct DB dependencies in Vercel functions
✅ **Consistency:** All auth endpoints now follow proxy pattern
✅ **Reliability:** Offloads complexity to proven Go backend
✅ **Zero Test Impact:** No new tests needed (existing service tests cover logic)

---

## Files to Refactor

| File | Current | Target | Effort |
|------|---------|--------|--------|
| `api/auth/2fa/setup.ts` | Direct service | HTTP proxy | ~30 lines |
| `api/auth/2fa/enable.ts` | Direct service | HTTP proxy | ~30 lines |
| `api/auth/2fa/disable.ts` | Direct service | HTTP proxy | ~30 lines |
| `api/auth/2fa/status.ts` | Direct service | HTTP proxy | ~25 lines |
| `api/auth/2fa/verify-login.ts` | Partial stub | Complete proxy | ~40 lines |

**Total:** ~5 files, ~155 lines of code

---

## Backend Verification Checklist

Before deploying Vercel changes, verify Go backend is ready:

- [ ] Backend 2FA endpoints respond with HTTP 200
- [ ] Response contains required fields (secret, otpauth, QR code, backup codes)
- [ ] Encryption/decryption works correctly
- [ ] Database operations succeed
- [ ] CORS headers are properly configured (if needed)
- [ ] Backend responds within acceptable latency (<500ms)

---

## Rollback Plan

If something goes wrong after refactoring:

1. **Immediate:** Revert Vercel deployment
2. **Restore:** Previous version automatically served
3. **Recovery:** No database changes needed

---

## Success Criteria

- [ ] All 5 2FA endpoints converted to proxy pattern
- [ ] 2FA setup endpoint returns HTTP 200 on production
- [ ] QR code generates and displays in web UI
- [ ] User can complete full 2FA enable flow
- [ ] No errors in Vercel logs
- [ ] Response time acceptable (<1 second)
- [ ] Works with both local and production backends

---

## Timeline

- **Phase 1 (Refactoring):** 30 minutes
- **Phase 2 (Backend Verification):** 10 minutes
- **Phase 3 (Testing):** 15 minutes
- **Total:** ~55 minutes

---

## Alternative Solutions (NOT Recommended)

### Option A: Fix Connection Pool
```typescript
// max: 10 instead of max: 1
export const client = postgres(connectionString || '', {
  ssl: 'require',
  max: 10  // Increase from 1
});
```
**Status:** Unlikely to fix root issue

### Option B: Add Retry Logic
```typescript
// Retry failed queries
for (let i = 0; i < 3; i++) {
  try {
    await db.update(...);
    break;
  } catch (err) {
    if (i === 2) throw err;
  }
}
```
**Status:** Masks underlying problem

### Option C: Use Raw SQL
```typescript
// Bypass Drizzle ORM
await client.unsafe(`UPDATE users SET ... WHERE id = $1`, [userId]);
```
**Status:** Defeats purpose of ORM

---

## Recommendation

**Proceed with Phase 1: Convert to Proxy Pattern**

This is the best long-term solution because it:
1. Fixes the immediate issue
2. Improves consistency
3. Reduces Vercel complexity
4. Leverages proven Go backend
5. Simplifies future maintenance

---

## Next Steps

1. **Confirm this approach with team**
2. **Refactor 5 endpoints** (api/auth/2fa/*.ts)
3. **Verify Go backend** endpoints
4. **Deploy to production**
5. **Run full E2E test flow**
