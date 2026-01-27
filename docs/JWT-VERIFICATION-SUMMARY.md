# JWT Token Verification Fix - Executive Summary

## Problem

The `/api/addresses` endpoint group does not verify JWT tokens before proxying requests to the Go backend, while `/api/auth/me` does. This creates:

1. **Inconsistent error handling** - Some endpoints return 401 immediately, others after backend round-trip
2. **Security gap** - No client-side validation for protected operations
3. **Poor user experience** - Slower 401 responses due to backend latency
4. **Code inconsistency** - Different patterns across similar endpoints

## Solution

Add JWT token verification to all protected endpoints using the proven pattern from `/api/auth/me`:

```typescript
import { verifyToken } from '../../src/lib/auth-middleware.js';

const user = verifyToken(req);
if (!user) {
  return res.status(401).json({ code: 'MISSING_TOKEN', message: 'Authentication required' });
}
```

## Scope

**6 files to modify:**
1. `/api/addresses/index.ts` - GET/POST endpoints
2. `/api/addresses/[...path].ts` - DELETE/POST endpoints
3. `/api/auth/2fa/setup.ts` - Protected 2FA setup
4. `/api/auth/2fa/enable.ts` - Protected 2FA enable
5. `/api/auth/2fa/disable.ts` - Protected 2FA disable
6. `/api/auth/2fa/status.ts` - Protected 2FA status

**Unaffected:**
- Public endpoints (login, register, verify-login)
- Frontend code
- Go backend

## Impact

| Metric | Before | After |
|--------|--------|-------|
| Protected endpoints with verification | 1 | 7 |
| Consistency | Low | High |
| 401 latency | ~100-500ms | <1ms |
| Test coverage | Partial | 100% |

## Implementation

**Effort:** ~2.5 hours
**Risk Level:** Low (isolated changes, extensive tests)
**Testing:** Full test coverage with 100% new code coverage
**Rollback:** Simple git revert if needed

## Key Files

- **Full Proposal:** `/docs/OPENSPEC-JWT-VERIFICATION.md`
- **Code Changes:** 6 API endpoint files
- **Tests:** Update `/api/addresses/addresses.test.ts` + create `/api/auth/2fa/2fa-token-verification.test.ts`

## Success Criteria

- [ ] All protected endpoints verify JWT tokens
- [ ] 401 responses consistent across all endpoints
- [ ] All tests pass with 100% coverage
- [ ] No performance regression
- [ ] No regressions in existing functionality
- [ ] Zero breaking changes for frontend

## Next Steps

1. Read full OpenSpec proposal: `/docs/OPENSPEC-JWT-VERIFICATION.md`
2. Review and provide feedback
3. Approve implementation plan
4. Execute Phase 1: Write tests first (TDD)
5. Execute Phase 2: Implement changes
6. Execute Phase 3: Deploy and monitor

---

**Document Location:** `/Users/eric/dreame/code/MoneraDigital/docs/OPENSPEC-JWT-VERIFICATION.md`
