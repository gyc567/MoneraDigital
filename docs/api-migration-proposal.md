# API Migration Proposal: TypeScript → Go Backend

## Executive Summary

**Current State**: 8 Vercel Serverless Functions deployed at `/api/auth/*` and `/api/redemption`  
**Target State**: Migrate all endpoints to Go backend (`internal/`)  
**Goal**: Eliminate Vercel Serverless Function limit issues, consolidate backend logic

---

## 1. Current API Inventory Analysis

### 1.1 TypeScript API Endpoints

| Endpoint | Method | Status | Complexity |
|----------|--------|--------|------------|
| `/api/auth/login` | POST | ✅ Active | Medium |
| `/api/auth/register` | POST | ✅ Active | Low |
| `/api/auth/me` | GET | ✅ Active | Low |
| `/api/auth/2fa/setup` | POST | ✅ Active | Medium |
| `/api/auth/2fa/enable` | POST | ✅ Active | Medium |
| `/api/auth/2fa/disable` | POST | ✅ Active | Medium |
| `/api/auth/2fa/verify-login` | POST | ✅ Active | Medium |
| `/api/redemption` | GET/POST | ✅ Active | Medium |

### 1.2 Go Backend Handler Coverage

| Handler | Status | Implementation Notes |
|---------|--------|----------------------|
| `Login` | ✅ Full | Implemented in `handlers.go:55-91` |
| `Register` | ✅ Full | Implemented in `handlers.go:93-119` |
| `GetMe` | ✅ Full | Implemented in `handlers.go:121-130` |
| `Setup2FA` | ⚠️ Stub | Returns `{"message": "Setup 2FA endpoint"}` |
| `Enable2FA` | ⚠️ Stub | Returns `{"message": "Enable 2FA endpoint"}` |
| `Verify2FALogin` | ⚠️ Stub | Returns `{"message": "Verify 2FA login endpoint"}` |

### 1.3 Go AuthService Coverage

| Method | Status | Implementation Notes |
|--------|--------|----------------------|
| `Login` | ✅ Full | `services/auth.go:137-180` |
| `Register` | ✅ Full | `services/auth.go:52-88` |
| `Verify2FAAndLogin` | ⚠️ Stub | Returns empty `LoginResponse{}` |

---

## 2. Migration Classification

### 2.1 Ready for Migration (No Changes Needed)

| Endpoint | Migration Effort | Risk |
|----------|------------------|------|
| `/api/auth/login` | Low | Low |
| `/api/auth/register` | Low | Low |
| `/api/auth/me` | Low | Low |

**Reason**: Go handlers and services are fully implemented and match TypeScript API behavior.

### 2.2 Requires Implementation Work

| Endpoint | Migration Effort | Risk | Dependencies |
|----------|------------------|------|--------------|
| `/api/auth/2fa/setup` | Medium | Medium | otplib, QRCode |
| `/api/auth/2fa/enable` | Medium | Medium | otplib |
| `/api/auth/2fa/disable` | Medium | Medium | otplib |
| `/api/auth/2fa/verify-login` | Medium | Medium | otplib |
| `/api/redemption` | High | Medium | PostgreSQL migration |

### 2.3 2FA Dependencies

**TypeScript Implementation** (`src/lib/two-factor-service.ts`):
- `otplib` - TOTP generation/verification
- `qrcode` - QR code generation
- `crypto` - Backup code generation
- `encryption.ts` - Secret encryption

**Go Alternatives**:
- `github.com/pquimby/otp` - TOTP library
- `github.com/skip2/go-qrcode` - QR code generation
- `crypto` (stdlib) - Backup codes

---

## 3. Detailed Migration Plan

### Phase 1: Production-Ready Endpoints (Week 1)

#### 3.1 `/api/auth/login` → Go

**TypeScript Behavior**:
```typescript
POST /api/auth/login
Body: { email, password }
Response: { user, token } OR { requires2FA, userId }
```

**Go Implementation**: `handlers.go:55-91`
```go
func (h *Handler) Login(c *gin.Context) {
    // ✅ Fully implemented
}
```

**Migration Steps**:
1. Update frontend to call Go backend (`http://localhost:8081/api/auth/login`)
2. Remove `api/auth/login.ts`
3. Test 2FA flow

**Frontend Changes Required**:
- Update `src/pages/Login.tsx:81-85` to point to Go backend URL
- Add environment variable for Go backend URL

---

#### 3.2 `/api/auth/register` → Go

**TypeScript Behavior**:
```typescript
POST /api/auth/register
Body: { email, password }
Response: { message, user: { id, email } }
```

**Go Implementation**: `handlers.go:93-119`
```go
func (h *Handler) Register(c *gin.Context) {
    // ✅ Fully implemented
}
```

**Migration Steps**:
1. Update frontend to call Go backend
2. Remove `api/auth/register.ts`
3. Test registration flow

---

#### 3.3 `/api/auth/me` → Go

**TypeScript Behavior**:
```typescript
GET /api/auth/me
Headers: Authorization: Bearer <token>
Response: { id, email, twoFactorEnabled }
```

**Go Implementation**: `handlers.go:121-130`
```go
func (h *Handler) GetMe(c *gin.Context) {
    // ✅ Fully implemented
}
```

**Migration Steps**:
1. Update frontend API client
2. Remove `api/auth/me.ts`
3. Test authenticated user endpoint

---

### Phase 2: 2FA Implementation (Week 2)

#### 3.4 `/api/auth/2fa/setup` → Go

**TypeScript Behavior**:
```typescript
POST /api/auth/2fa/setup
Headers: Authorization: Bearer <token>
Response: { secret, qrCodeUrl, backupCodes, otpauth }
```

**Go Implementation Required**:
```go
// New handler in handlers.go
func (h *Handler) Setup2FA(c *gin.Context) {
    // TODO: Implement
    // 1. Generate secret using otp library
    // 2. Generate QR code
    // 3. Generate backup codes
    // 4. Encrypt and store in PostgreSQL
}
```

**New Service Required**:
```go
// services/two_factor_service.go
type TwoFactorService struct {
    DB          *sql.DB
    Encryption  *encryption.Service
}

func (s *TwoFactorService) Setup(userID int, email string) (*Setup2FAResponse, error)
func (s *TwoFactorService) Enable(userID int, token string) error
func (s *TwoFactorService) Disable(userID int, token string) error
func (s *TwoFactorService) Verify(userID int, token string) (bool, error)
```

**Required Go Dependencies**:
```bash
go get github.com/pquimby/otp@v1.2.1
go get github.com/skip2/go-qrcode@v1.3.0
```

---

#### 3.5 `/api/auth/2fa/enable` → Go

**TypeScript Behavior**:
```typescript
POST /api/auth/2fa/enable
Headers: Authorization: Bearer <token>
Body: { token }
Response: { success: true }
```

**Go Implementation Required**:
```go
func (h *Handler) Enable2FA(c *gin.Context) {
    // 1. Get user ID from context
    // 2. Verify TOTP token
    // 3. Update user.two_factor_enabled = true
}
```

---

#### 3.6 `/api/auth/2fa/disable` → Go

**TypeScript Behavior**:
```typescript
POST /api/auth/2fa/disable
Headers: Authorization: Bearer <token>
Body: { token }
Response: { success: true }
```

**Go Implementation Required**:
```go
func (h *Handler) Disable2FA(c *gin.Context) {
    // 1. Get user ID from context
    // 2. Verify TOTP or backup code
    // 3. Clear two_factor_secret, two_factor_enabled, two_factor_backup_codes
}
```

---

#### 3.7 `/api/auth/2fa/verify-login` → Go

**TypeScript Behavior**:
```typescript
POST /api/auth/2fa/verify-login
Body: { userId, token }
Response: { user, token }
```

**Go Implementation Required**:
```go
func (h *Handler) Verify2FALogin(c *gin.Context) {
    // 1. Verify TOTP token
    // 2. Generate JWT
    // 3. Return { user, token }
}
```

**Update AuthService**:
```go
func (s *AuthService) Verify2FAAndLogin(userID int, token string) (*LoginResponse, error) {
    // Currently returns empty struct - needs full implementation
}
```

---

### Phase 3: Redemption Service (Week 3)

#### 3.8 `/api/redemption` → Go

**TypeScript Behavior**:
```typescript
GET /api/redemption?id=<id>
POST /api/redemption
Body: { userId, productId, principal, autoRenew }
Response: Redemption record
```

**Current Go Implementation**: `internal/redemption/service.go`
- Uses in-memory repository (`NewInMemoryRedemptionRepository()`)
- PostgreSQL repository exists (`internal/redemption/repository.go`)

**Migration Steps**:
1. Update Go handler to use PostgreSQL repository instead of in-memory
2. Verify PostgreSQL schema for redemption tables
3. Test complete redemption flow
4. Remove TypeScript endpoint

**Required PostgreSQL Schema**:
```sql
-- Already exists in migrations?
-- Check internal/migration/migrations/
```

---

## 4. Frontend Integration Strategy

### 4.1 API Client Configuration

**Current**: Direct path references
```typescript
fetch("/api/auth/login", ...)
```

**Target**: Environment-based configuration
```typescript
const API_BASE = import.meta.env.VITE_API_URL || "http://localhost:8081";
fetch(`${API_BASE}/api/auth/login`, ...)
```

**Environment Variables Required**:
```env
# .env
VITE_API_URL=http://localhost:8081  # Development
# VITE_API_URL=https://api.moneradigital.com  # Production
```

### 4.2 Vite Proxy Configuration

**Current** (`vite.config.ts`):
```typescript
proxy: {
  "/api": {
    target: "http://localhost:8081",
    changeOrigin: true,
  },
}
```

**Updated for Production**:
- Remove proxy for production builds
- Use environment variable for API URL

---

## 5. Rollout Strategy

### 5.1 Blue-Green Deployment

1. **Go Backend** runs on `http://localhost:8081`
2. **Frontend** configured to use Go backend via environment
3. **Testing**: Canary deploy to subset of users
4. **Monitoring**: Error rates, latency, user feedback
5. **Full Migration**: Once validated, remove TypeScript endpoints

### 5.2 Rollback Plan

1. Keep TypeScript endpoints as fallback
2. Use feature flag to control backend selection
3. Immediate rollback via environment configuration

---

## 6. File Deletion清单

After successful migration, delete:

```
api/auth/login.ts          # Migrated
api/auth/register.ts       # Migrated
api/auth/me.ts             # Migrated
api/auth/2fa/setup.ts      # Migrated
api/auth/2fa/enable.ts     # Migrated
api/auth/2fa/disable.ts    # Migrated
api/auth/2fa/verify-login.ts # Migrated
api/redemption.ts          # Migrated
```

---

## 7. Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| 2FA TOTP compatibility | High | Test tokens across devices |
| JWT token format mismatch | High | Ensure identical payload structure |
| Database schema differences | Medium | Verify PostgreSQL schema matches |
| Performance regression | Medium | Benchmark before/after |
| Rollback complexity | Medium | Keep TypeScript endpoints as fallback |

---

## 8. Success Criteria

- [ ] All 8 endpoints migrated to Go
- [ ] Zero downtime deployment
- [ ] Error rates < 0.1%
- [ ] API latency < 200ms (p95)
- [ ] All 2FA flows tested
- [ ] All user registration/login tested

---

## 9. Effort Estimation

| Phase | Tasks | Estimated Effort |
|-------|-------|------------------|
| Phase 1 | Login, Register, Me | 2 hours |
| Phase 2 | 2FA (4 endpoints) | 8 hours |
| Phase 3 | Redemption | 4 hours |
| Testing | Integration tests | 4 hours |
| **Total** | | **~18 hours** |

---

## 10. Next Steps

1. **Approval**: Review and approve migration plan
2. **Phase 1**: Implement login/register/me migration
3. **Testing**: Validate Phase 1 in staging
4. **Phase 2**: Implement 2FA services
5. **Phase 3**: Migrate redemption service
6. **Final Cleanup**: Remove TypeScript endpoints

---

*Generated: 2026-01-22*
*Project: Monera Digital*
*Author: Sisyphus AI Agent*
