# GEMINI.md

This file provides context and guidance for Gemini when working with the Monera Digital codebase.

## Project Overview

**Monera Digital** is an institutional-grade digital asset platform focused on static finance and lending solutions.
It is a **full-stack application** utilizing a TypeScript frontend and a **Golang backend**.

**Primary Stack:**
*   **Frontend:** React 18, TypeScript, Vite, Tailwind CSS, Shadcn/Radix UI.
*   **Backend:** **Golang (Go)** - Mandatory for all interfaces, database access, and operations.
*   **External Core System:** **Monnaire Core API** - Core account management (integrated via Go backend only).
*   **Database:** PostgreSQL (Neon).
*   **Caching/State:** Redis (Upstash) for rate limiting and session management.
*   **Testing:** Vitest (Unit/Integration), Playwright (E2E).

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Architecture                          │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│   Frontend (React)                                          │
│   └── src/lib/*-service.ts    →  HTTP API Calls             │
│                                    │                        │
│                                    ▼                        │
│   API Routes (Vercel)                                       │
│   └── api/[...route].ts       →  Proxy to Go Backend        │
│                                    │                        │
│                                    ▼                        │
│   Go Backend (internal/)                                    │
│   ├── internal/handlers/      →  HTTP Handlers              │
│   ├── internal/services/      →  Business Logic             │
│   ├── internal/repository/    →  Database Access (PostgreSQL)│
│   └── internal/handlers/core/ →  Core API Integration       │
│                                    │                        │
│                          ┌─────────┴──────────┐             │
│                          ▼                    ▼             │
│                  PostgreSQL (Neon)    Monnaire Core API     │
│                                                  (External) │
└─────────────────────────────────────────────────────────────┘
```

### Critical Architecture Rules

1. **Frontend ONLY calls `/api/*`**
   - ✅ Frontend → `/api/withdrawals` → Go Backend
   - ❌ Frontend → Direct database access
   - ❌ Frontend → Direct Core API access

2. **Go Backend handles ALL external integrations**
   - ✅ Go Backend → PostgreSQL (database)
   - ✅ Go Backend → Monnaire Core API (external system)
   - Go backend is the **only** layer that can access these resources

3. **Service Layer Pattern**
   - Frontend services (`src/lib/*-service.ts`) are **API clients only**
   - They contain NO business logic, only HTTP calls
   - Business logic is in `internal/services/` (Go)

## Key Directory Structure

*   **`src/`**: Frontend source code and shared business logic.
    *   `src/components/`: React components (UI primitives in `ui/`, feature components elsewhere).
    *   `src/pages/`: Route components (Login, Register, Dashboard views).
    *   `src/lib/`: Frontend service layer and utilities.
    *   `src/i18n/`: Internationalization (English/Chinese).
*   **`internal/` & `cmd/`**: **Primary Backend Code**. All API handlers, business logic, and database operations reside here.
*   **`api/`**: **统一 Serverless Function（唯一文件）**
    *   `api/[...route].ts`: 统一路由处理器，所有 API 请求通过此单一入口
    *   `api/__route__.test.ts`: 路由测试
*   **`docs/`**: Extensive project documentation (Architecture, PRDs, Security).

### 统一 Serverless Function 架构（重要）

**Vercel Hobby 计划限制**: 最多 12 个 Serverless Functions。

**必须使用统一路由架构**:
```
api/
├── [...route].ts          # 统一路由处理器（唯一 Serverless Function）
└── __route__.test.ts      # 路由测试
```

**禁止这样做**（会导致部署失败）:
```
api/
├── auth/
│   ├── login.ts          # ❌ 单独的函数 - 会导致超过 12 个限制
│   ├── register.ts       # ❌ 单独的函数
│   └── logout.ts         # ❌ 单独的函数
```

### JSON 命名规范（关键）

**所有 API 请求/响应字段必须使用驼峰命名（camelCase）**：

✅ **正确**:
```json
{
  "userId": 1,
  "accessToken": "xxx",
  "refreshToken": "xxx",
  "requires2FA": true
}
```

❌ **错误**:
```json
{
  "user_id": 1,
  "access_token": "xxx",
  "refresh_token": "xxx",
  "requires_2fa": true
}
```

**Go 结构体标签规范**:
```go
type LoginResponse struct {
    UserID      int    `json:"userId"`       // JSON: camelCase
    AccessToken string `json:"accessToken"`  // JSON: camelCase
    // db tag uses snake_case for database columns
}
```
├── 2fa/
│   ├── setup.ts          # ❌ 单独的函数
│   └── enable.ts         # ❌ 单独的函数
└── ... (更多文件)
```

**所有路由在 `api/[...route].ts` 中集中配置**:
```typescript
const ROUTE_CONFIG: Record<string, RouteConfig> = {
  'POST /auth/login': { requiresAuth: false, backendPath: '/api/auth/login' },
  'POST /auth/register': { requiresAuth: false, backendPath: '/api/auth/register' },
  // ... 其他路由
};
```

**添加新 API 端点**: 只需在 `ROUTE_CONFIG` 中添加配置，**不要创建新文件**。

## Development Workflow

### Commands
*   **Start Dev Server:** `npm run dev` (Vite, defaults to port 8080).
*   **Build:** `npm run build` (Production build to `dist/`).
*   **Test:** `npm test` (Runs Vitest).
*   **Lint:** `npm run lint`.

### Architecture & Patterns

1.  **Architecture Principle: Backend-Only Business Logic**:
    *   **Frontend API Routes (`api/`)**: MUST be pure HTTP proxies only - NO business logic
    *   **Go Backend (`internal/`)**: MUST handle ALL business logic, database operations, and authentication
    *   **Frontend Service Layer (`src/lib/`)**: MUST NOT contain direct database access or authentication logic
    *   **All API calls**: Frontend → Vercel API (proxy only) → Go Backend (business logic)
    *   **Exception**: Simple UI utilities and form validation are allowed in frontend

2.  **Authentication**:
    *   JWT-based. Tokens are stored in `localStorage` (known limitation/choice).
    *   Flow: Login → Vercel API (proxy) → Go Backend → JWT → LocalStorage → Headers in API calls.
    *   Includes 2FA (TOTP) support.

3.  **Service Layer Pattern (`src/lib/`)**:
    *   **Frontend services are API clients only** - they call `/api/*` endpoints
    *   NO business logic in frontend services
    *   NO direct database access
    *   Example: `auth-service.ts` calls `/api/auth/login`, not database

4.  **Go Backend Structure**:
    *   `internal/handlers/` - HTTP request handlers
    *   `internal/services/` - Business logic and database operations
    *   `internal/handlers/core/` - Monnaire Core API integration
    *   `internal/repository/` - Database access layer
    *   `internal/migration/` - Database migrations

5.  **External Integrations** (Go Backend only):
    *   **Monnaire Core API**: Core account management, KYC
    *   **PostgreSQL**: Application data storage
    *   **Redis**: Caching and rate limiting

6.  **UI/UX**:
    *   **Naming Convention:** "Lending" features are exposed to the user as **"Fixed Deposit"** (Sidebar, Hero buttons), though internal code variables often still use `lending`.
    *   **Theme:** Shadcn UI + Tailwind.

4.  **UI/UX**:
    *   **Naming Convention:** "Lending" features are exposed to the user as **"Fixed Deposit"** (Sidebar, Hero buttons), though internal code variables often still use `lending`.
    *   **Theme:** Shadcn UI + Tailwind.

## Important Context (Recent Changes)

*   **Fixed Deposit vs. Lending:** As of Jan 2026, the UI term "Lending" has been renamed to "Fixed Deposit" to clarify the product offering. The internal routes (`/dashboard/lending`) and API paths (`api/lending`) remain unchanged.
*   **Architecture Unification (Jan 2026):** 
    *   Removed Drizzle ORM from frontend
    *   Frontend services now ONLY make API calls (no direct database access)
    *   All database and Core API access centralized in Go backend
    *   Architecture: Frontend → API → Go Backend → Core API/Database
*   **Mandatory Go Backend:** The project has shifted to a strict **Golang backend** architecture. All new features and bug fixes regarding backend logic, database interactions, Core API integration, and API interfaces **must** be implemented in Go (residing in `internal/` and `cmd/`). Node.js/Vercel functions should only be used for frontend-specific proxying if absolutely necessary, but core logic belongs in Go.

## New Feature Development Rules

**Mandatory rules for developing new features:**

1.  **Technology Stack**
     - **Frontend**: TypeScript
     - **Backend**: Golang (Go) - **MUST** be used for all backend interfaces, database access, and operations.

2.  **Architecture Principle: Backend-Only Business Logic**
     - **Frontend API Route (`api/`)**: MUST be pure HTTP proxies only - NO business logic
     - **Go Backend (`internal/`)**: MUST handle ALL business logic, database operations, Core API integration, and authentication
     - **Frontend Service Layer (`src/lib/`)**: MUST NOT contain direct database access, Core API access, or business logic - ONLY API calls to `/api/*`
     - **All API calls**: Frontend → Vercel API (proxy only) → Go Backend → Core API/Database
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
