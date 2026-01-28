# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**MoneraDigital** is an institutional-grade digital asset platform offering secure, transparent static finance and lending solutions. It's a full-stack application with React frontend (TypeScript, Vite) and Golang backend.

**Key Stack:**
- Frontend: React 18, TypeScript, Vite, Tailwind CSS, Radix UI (51 components)
- Backend: **Golang (Go)** - Mandatory for all backend interfaces, database access, and operations.
- Testing: Vitest, Playwright
- i18n: English and Chinese support via i18next

---

## Common Development Commands

### Development
```bash
npm run dev          # Start Vite dev server (http://localhost:8080)
npm run lint         # Run ESLint checks
```

### Build & Preview
```bash
npm run build        # Production build with optimized chunks
npm run build:dev    # Development build
npm run preview      # Preview built application locally
```

### Testing
```bash
npm run test         # Run unit tests with Vitest (JSDOM environment)
npm run test:ui      # Interactive test UI in browser
```

### Single Test Execution
```bash
npm run test -- src/lib/auth-service.test.ts          # Specific file
npm run test -- --reporter=verbose                      # Verbose output
npm run test -- --watch                                 # Watch mode
```

---

## New Feature Development Rules

**Mandatory rules for developing new features:**

1.  **Technology Stack**
     - **Frontend**: TypeScript
     - **Backend**: Golang (Go) - **MUST** be used for all backend interfaces, database access, and operations.

2.  **Architecture Principle: Backend-Only Business Logic**
     - **Frontend API Route (`api/`)**: MUST be pure HTTP proxies only - NO business logic
     - **Go Backend (`internal/`)**: MUST handle ALL business logic, database operations, and authentication
     - **Frontend Service Layer (`src/lib/`)**: MUST NOT contain direct database access or authentication logic
     - **All API calls**: Frontend → Vercel API (proxy only) → Go Backend (business logic)
     - **Exception**: Simple UI utilities and form validation are allowed in frontend

3.  **Design Principles**
     - **KISS**: Keep code clean and simple.
     - **Architecture**: High Cohesion, Low Coupling. Use streamlined design patterns.
     - **Single Source of Truth**: Go backend is the only source for business logic.

4.  **Testing**
     - **Requirement**: All new functional code must be tested.
     - **Coverage**: Maintain **100% test coverage**.

5.  **Isolation**
     - Changes must **not** affect unrelated functions.

6.  **Proposal Process**
     - Use **openspec** to generate proposals for new features.

## Bug Fixing Rules

**Mandatory rules for fixing bugs:**

1.  **Technology Stack**
     - **Frontend**: TypeScript
     - **Backend**: Golang (Go) - **MUST** be used for all backend interfaces, database access, and operations.

2.  **Architecture Principle: Backend-Only Business Logic**
     - **Frontend API Routes (`api/`)**: MUST be pure HTTP proxies only - NO business logic
     - **Go Backend (`internal/`)**: MUST handle ALL business logic, database operations, and authentication
     - **Frontend Service Layer (`src/lib/`)**: MUST NOT contain direct database access or authentication logic
     - **All API calls**: Frontend → Vercel API (proxy only) → Go Backend (business logic)
     - **Exception**: Simple UI utilities and form validation are allowed in frontend

3.  **Design Principles**
     - **KISS**: Keep code clean and simple.
     - **Architecture**: High Cohesion, Low Coupling. Use concise design patterns.
     - **Single Source of Truth**: Go backend is the only source for business logic.

4.  **Testing**
     - **Methodology**: Test-Driven Development (TDD) - **write tests first**.
     - **Requirement**: All new functional code must be tested.
     - **Coverage**: Maintain **100% test coverage**.
     - **Regression**: Perform regression testing after fixes.

5.  **Isolation**
     - Changes must **not** affect unrelated functions.

6.  **Proposal Process**
     - Use **openspec** to generate new bug proposals.

---

## High-Level Architecture

### Frontend Structure (`src/`)

**Pages** (`src/pages/`):
- `Index.tsx` - Landing page (hero, features, call-to-actions)
- `Login.tsx` - Two-step authentication (email/password + 2FA token/backup code)
- `Register.tsx` - User registration with email verification
- `dashboard/` - Protected pages (Overview, Assets, Lending, Addresses, Withdraw, Security)

**Components** (`src/components/`):
- `DashboardLayout.tsx` - Protected layout wrapper (checks token, redirects to login)
- `DashboardSidebar.tsx` - Collapsible navigation menu with i18n support
- `ui/` - Shadcn/Radix UI primitives (51 pre-built components)
- Feature components (Hero, Features, Stats, etc.)

**Services** (`src/lib/`):
- Core business logic services that are **reused by both API endpoints and frontend**
  - `auth-service.ts` - Registration, login, password hashing, JWT generation
  - `two-factor-service.ts` - TOTP setup, verification, backup codes (encrypted)
  - `lending-service.ts` - Lending applications, position management, APY calculations
  - `address-whitelist-service.ts` - Address verification (24-hour email tokens), management
  - `withdrawal-service.ts` - Withdrawal processing and transaction tracking
  - `email-service.ts` - Email notifications and verification tokens
  - `encryption.ts` - AES-256-GCM encryption (2FA secrets, backup codes)
  - `auth-middleware.ts` - JWT verification (used in API handlers)
  - `redirect-validator.ts` - Open redirect attack mitigation (whitelist validation)
  - `rate-limit.ts` - Redis-based rate limiting (5 requests/60s per IP)
- Utilities: `db.ts` (Drizzle instance), `redis.ts` (Upstash client), `logger.ts` (Pino), `utils.ts` (cn function)

**Database** (`src/db/`):
- `schema.ts` - Drizzle schema definitions (users, lending_positions, withdrawal_addresses, address_verifications, withdrawals)
- PostgreSQL native enums for status fields (lending_status, address_type, withdrawal_status)
- Foreign key relationships enforced

**Internationalization** (`src/i18n/`):
- `config.ts` - i18next setup (localStorage language persistence)
- `locales/en.json` - English translations (all UI copy)
- `locales/zh.json` - Chinese translations

### Backend Structure (`api/`)

**Route Organization** (Vercel Serverless Functions):
- **`api/[...route].ts`** - **统一路由处理器（唯一 Serverless Function）**
- `api/__route__.test.ts` - 路由测试

**重要规则**: Vercel Hobby 计划限制最多 12 个 Serverless Functions。项目必须使用统一的路由架构：

```
api/
├── [...route].ts          # 统一路由处理器（唯一 Serverless Function）
└── __route__.test.ts      # 路由测试
```

**禁止这样做**（会导致超过 12 个函数限制）：
```
api/
├── auth/
│   ├── login.ts          # ❌ 单独的函数
│   ├── register.ts       # ❌ 单独的函数
│   └── logout.ts         # ❌ 单独的函数
├── 2fa/
│   ├── setup.ts          # ❌ 单独的函数
│   └── enable.ts         # ❌ 单独的函数
└── ... (更多文件)
```

**统一路由配置** (`ROUTE_CONFIG`):
```typescript
const ROUTE_CONFIG: Record<string, RouteConfig> = {
  // Auth endpoints
  'POST /auth/login': { requiresAuth: false, backendPath: '/api/auth/login' },
  'POST /auth/register': { requiresAuth: false, backendPath: '/api/auth/register' },
  
  // 2FA endpoints
  'POST /auth/2fa/setup': { requiresAuth: true, backendPath: '/api/auth/2fa/setup' },
  'POST /auth/2fa/enable': { requiresAuth: true, backendPath: '/api/auth/2fa/enable' },
  
  // ... 其他路由
};
```

**Pattern**: 统一路由处理器的工作流程：
1. 解析请求路径和方法
2. 在 `ROUTE_CONFIG` 中查找匹配的路由
3. 检查认证要求（如果需要）
4. 转发请求到 Go 后端
5. 返回后端响应

**添加新路由的步骤**：
1. 在 `ROUTE_CONFIG` 中添加配置（不需要创建新文件）
2. 在 Go 后端 `internal/routes/routes.go` 中添加对应处理器
3. 在 `api/__route__.test.ts` 中添加测试

---

## Database Schema Quick Reference

**users** table
- id (PK), email (unique), password (hashed)
- twoFactorSecret, twoFactorEnabled, twoFactorBackupCodes (encrypted)
- createdAt

**lending_positions** table
- id (PK), userId (FK), asset, amount, durationDays
- apy, status (enum), accruedYield, startDate, endDate

**withdrawal_addresses** table
- id (PK), userId (FK), address, addressType (enum: BTC|ETH|USDC|USDT)
- label, isVerified, isPrimary, createdAt, verifiedAt, deactivatedAt

**address_verifications** table
- id (PK), addressId (FK), token (unique), expiresAt, verifiedAt

**withdrawals** table
- id (PK), userId (FK), fromAddressId (FK), amount, asset, toAddress
- status (enum), txHash, createdAt, completedAt, failureReason

---

## Authentication & Security Patterns

### Authentication Flow
1. **Register**: Email + password → bcryptjs hash → stored in PostgreSQL
2. **Login**: Email + password verification → JWT token (24h) → localStorage
3. **2FA Setup**: Generate TOTP secret → show QR code → verify with TOTP → store encrypted secret + 10 backup codes
4. **2FA Verification**: Validate TOTP token or use backup code (one-time use)

### Key Security Features
- **JWT**: `jsonwebtoken` library for token generation/verification (24h expiry)
- **Password Hashing**: bcryptjs (v2.4.3)
- **TOTP**: otplib library for authenticator support
- **Encryption**: AES-256-GCM (2FA secrets, backup codes stored encrypted)
- **Open Redirect Protection**: `redirect-validator.ts` whitelist-based validation for returnTo URLs
- **Rate Limiting**: Redis-backed, 5 requests/60s per IP
- **Address Whitelisting**: 24-hour email verification before enabling withdrawal

### Token Storage & Refresh
- **Current**: Token + user object in localStorage (XSS vulnerability - consider HttpOnly cookies in future)
- **No automatic refresh**: Token expires at 24h, user must re-login
- This is a known limitation for long-session scenarios

---

## State Management

**React Query (TanStack)**: Server state (API data caching/refetching)
- Used for: `/api/lending/positions`, address listings, withdrawal history
- Configured in `vite.config.ts` (manual chunking: `vendor-charts`)

**useState**: Component-level UI state
- Form inputs, loading states, UI toggles
- Used in Login (email/password/2FA), Lending (application form), withdrawals

**localStorage**: Persistent client state
- Stores: JWT token, user object (email, id)
- Managed directly (no Zustand/Redux)

**No centralized global state**: No Redux, Zustand, or Jotai
- Consider adding for future features if local state becomes complex

---

## Routing & Protected Pages

**React Router v6** (`src/App.tsx` route setup):
```
/                          → Index (public landing)
/login                     → Login (public)
/register                  → Register (public)
/dashboard                 → DashboardLayout (protected)
  /dashboard               → Overview
  /dashboard/assets        → Assets
  /dashboard/lending       → Lending
  /dashboard/addresses     → Addresses (whitelist)
  /dashboard/withdraw      → Withdraw
  /dashboard/security      → Security (2FA settings)
*                          → NotFound (404)
```

**Protection Strategy**: `DashboardLayout.tsx` checks for token in localStorage. If missing, redirects to `/login?returnTo=/dashboard/...`. No centralized route guard component (check `ProtectedRoute` pattern in docs if implementing).

---

## Build System & Optimization

**Vite Config** (`vite.config.ts`):
- Dev server: port 8080, IPv6 enabled
- React SWC compiler for fast builds
- **Manual chunk splitting** (important for cache busting):
  - `vendor-ui` - Radix UI + Lucide React
  - `vendor-charts` - Recharts
  - `vendor-core` - Other node_modules
  - Main app bundle
- Path alias: `@/` → `./src/`

**Build Output**: Optimized chunks in `dist/` with gzip compression

---

## Testing Structure

**Unit Testing** (Vitest with JSDOM):
- Tests co-located with source files (e.g., `auth-service.test.ts` next to `auth-service.ts`)
- `vitest.setup.ts` provides global configuration
- Run single test: `npm run test -- src/lib/auth-service.test.ts`

**Integration Tests** (`tests/` directory):
- `api-auth.test.ts` - API endpoint tests
- `auth.spec.ts` - Auth flow specification

**E2E Testing** (Playwright):
- Configuration in `playwright.config.ts`
- Run: `npm run test:e2e` (if script exists in package.json)

---

## Environment Variables

**Required**:
```
DATABASE_URL         # PostgreSQL Neon connection string
JWT_SECRET          # Secret for JWT signing/verification (minimum 32 bytes)
ENCRYPTION_KEY      # 32-byte hex string for AES-256-GCM (64 hex chars)
```

**Optional**:
```
UPSTASH_REDIS_REST_URL          # Redis connection (for rate limiting, session management)
UPSTASH_REDIS_REST_TOKEN        # Redis auth token
```

**Local Development**: Copy `.env.example` to `.env` and fill in values

---

## Code Organization Principles

### Service Layer Pattern
All business logic lives in `src/lib/` services. Services are:
- **Reused by both API endpoints and frontend** (single source of truth)
- **Dependency-injected**: Accept database, Redis, logger as parameters
- **Type-safe**: Use Zod for validation, TypeScript for type inference

**Example Structure**:
```typescript
// src/lib/lending-service.ts
export async function applyForLending(
  db: Database,
  userId: string,
  params: LendingApplicationParams
): Promise<LendingPosition> {
  // Business logic here
}

// api/lending/apply.ts (uses same service)
const position = await applyForLending(db, userId, body);
```

### API Handler Pattern
```typescript
// api/endpoint.ts
1. Check auth (via auth-middleware.ts)
2. Validate request (Zod)
3. Call service layer
4. Return response with proper status code
5. Catch errors and return structured error object
```

### Component Patterns
- **Pages**: Full-page components in `src/pages/`
- **Layout Components**: DashboardLayout, Header, Footer in `src/components/`
- **UI Components**: Reusable Radix/Shadcn components in `src/components/ui/`
- **Custom Hooks**: React logic in `src/hooks/`
- **Form Validation**: React Hook Form + Zod for type safety

---

## Common Modification Scenarios

### Adding a New API Endpoint
1. Create service in `src/lib/` (reusable logic)
2. Create handler in `api/` directory
3. Add Zod validation schema
4. Import and use service
5. Add tests alongside service

### Adding a New Protected Page
1. Create page component in `src/pages/`
2. Add route to `src/App.tsx` under DashboardLayout
3. Create service logic in `src/lib/` if needed
4. Create corresponding API endpoints
5. Use React Query for data fetching

### Adding 2FA or Security Features
- Reference `two-factor-service.ts` and `api/auth/2fa/` for patterns
- Use `encryption.ts` for sensitive data
- Add email verification flows (see `address-whitelist-service.ts`)

### Modifying Database Schema
1. Update `src/db/schema.ts` (Drizzle definitions)
2. Generate migration: `npx drizzle-kit generate:pg` (if using migrations)
3. Update type-inferred models
4. Update services that use the table
5. Add tests

### Adding i18n Strings
1. Add key to `src/i18n/locales/en.json`
2. Add translation to `src/i18n/locales/zh.json`
3. Use in component: `const { t } = useTranslation(); t('key')`

---

## Known Architectural Decisions & Limitations

### Why No Global State Manager (Redux/Zustand)?
- Current complexity is manageable with useState + localStorage + React Query
- Can be added later if feature complexity increases
- Services layer keeps business logic separate from state

### Why Token in localStorage?
- Simpler implementation than HttpOnly cookies for Vercel Serverless
- **Tradeoff**: Vulnerable to XSS attacks
- **Mitigation**: Use Content Security Policy, avoid inline scripts
- **Future improvement**: Consider HttpOnly cookies + CSRF tokens

### Why No Automatic Token Refresh?
- 24-hour JWT expiry is acceptable for current use cases
- Implement refresh tokens if longer sessions needed
- See `POST /api/auth/refresh` pattern in other repos

### Why Manual Middleware Composition?
- Vercel Functions don't support Express middleware chains
- Each handler composes middleware it needs
- Benefits: Fine-grained control, simple debugging

### Why Go Backend in `internal/`?
- Legacy parallel backend for specific workloads
- Not actively used in current frontend
- Can be removed if no longer needed

---

## Performance & Optimization Tips

1. **React Query Caching**: Leverage automatic refetching for dashboard data
2. **Code Splitting**: Vite automatically splits vendor chunks (see config)
3. **Lazy Loading**: Use React.lazy() for dashboard pages not always needed
4. **Image Optimization**: Use `<img>` with srcset or next-gen formats
5. **Redis Caching**: Rate limit middleware uses Redis—extend for other data
6. **Database Indexes**: Add indexes on frequently queried columns (userId, email, status)

---

## Debugging Tips

### View API Requests
- Open browser DevTools → Network tab
- Watch for auth failures (401) or validation errors (400)
- Check `Authorization` header is present

### Check Authentication State
- Open DevTools → Console: `localStorage.getItem('token')`
- Verify JWT payload: `jwtDecode(token)` (install jwt-decode if needed)

### Trace Service Logic
- Services use Pino logger (check `logger.ts`)
- Add `logger.info()` calls in service methods
- Check logs in Vercel deployment or local console

### Database Issues
- Verify DATABASE_URL is correct: `psql $DATABASE_URL -c "SELECT 1"`
- Check Drizzle schema types: `npx drizzle-kit introspect:pg`
- View logs in PostgreSQL Neon dashboard

---

## Deployment Notes

**Frontend**: Deployed to Vercel (static files + serverless functions)
- `npm run build` generates `/dist` folder
- `.vercelignore` excludes unnecessary files
- API routes in `/api` become serverless endpoints

**Environment Setup**:
1. Add DATABASE_URL, JWT_SECRET, ENCRYPTION_KEY to Vercel environment
2. Run `npm run build` locally to verify
3. Commit to `main` branch to trigger automatic deployment

---

## Resources & References

- **Architecture Audit**: See `docs/ARCHITECT-AUDIT-REPORT.md` (comprehensive code review)
- **Security Fixes**: See `docs/SECURITY-FIXES.md` (open redirect mitigation details)
- **Protected Routes**: See `docs/PROTECTED-ROUTES-IMPLEMENTATION.md` (centralized auth guard pattern)
- **Implementation Summary**: See `docs/IMPLEMENTATION-SUMMARY.md` (recent feature summaries)
- **React Router v6 Docs**: https://reactrouter.com/
- **Drizzle ORM Docs**: https://orm.drizzle.team/
- **Vite Docs**: https://vitejs.dev/
