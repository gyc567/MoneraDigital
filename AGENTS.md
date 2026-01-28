# AGENTS.md

> Context and instructions for AI coding agents working on Monera Digital.

## Project Overview

**Monera Digital** is an institutional-grade digital asset platform offering secure, transparent static finance and lending solutions. It is a full-stack application.

## Tech Stack

- **Frontend**: React 18, TypeScript, Vite, Tailwind CSS, Shadcn UI (Radix Primitives)
- **Backend**: Golang (Go) - **Mandatory for all interfaces, database access, and operations.**
- **Database**: PostgreSQL (Neon)
- **State/Cache**: Redis (Upstash)
- **Testing**: Vitest (Unit/Integration), Playwright (E2E)
- **Language**: TypeScript (Frontend), Go (Backend)

---

## Build, Lint & Test Commands

### Core Commands

```bash
# Install dependencies
npm install

# Development server (port 8080)
npm run dev

# Production build
npm run build

# Linting
npm run lint
npm run lint -- --fix  # Auto-fix issues
```

### Testing

```bash
# Run all tests
npm test

# Run specific test file (most common pattern)
npm test -- src/lib/auth-service.test.ts

# Run tests with coverage
npm test -- --coverage

# Run tests in watch mode (development)
npm test

# Run tests matching pattern
npm test -- --testNamePattern="login"

# E2E tests with Playwright
npm run test:e2e
```

---

## Code Style Guidelines

### Imports

- Use **absolute imports** with `@/` prefix for internal modules
- Use **relative imports** (`./*`) only for same-directory co-located files
- Group imports in this order:
  1. Built-in/Node imports (`node:*`, `path`, `fs`)
  2. Third-party packages (alphabetically)
  3. Absolute imports from `@/` (alphabetically)
  4. Relative imports (alphabetically)

```typescript
// ✅ Correct
import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { Card, CardContent } from "@/components/ui/card";
import { db } from "./db.js";
import { users } from "../db/schema.js";

// ❌ Avoid
import { db } from './db.js';
import { useState } from "react";
import { Card } from '@/components/ui/card';
```

### Formatting

- **Line length**: Soft limit at 100 characters
- **Indentation**: 2 spaces (not tabs)
- **Semicolons**: Always use semicolons
- **Quotes**: Double quotes for strings, single quotes for JSX
- **Trailing commas**: Use in multi-line objects/arrays

```typescript
// ✅ Correct
const user = {
  id: 1,
  email: "user@example.com",
};

// ❌ Avoid
const user = { id: 1, email: "user@example.com" };
```

### TypeScript

- **Explicit types** for function parameters and return types
- **No `any`** - use `unknown` or proper types instead
- **Use interfaces** for object shapes, types for unions/primitives
- **Avoid type assertions** (`as any`, `as const`) - use proper typing
- **Use `zod`** for runtime validation of external inputs

```typescript
// ✅ Correct
interface User {
  id: number;
  email: string;
}

async function getUser(id: number): Promise<User | null> {
  // implementation
}

// ❌ Avoid
function getUser(id: any): any {
  // implementation
}
```

### Naming Conventions

| Type | Convention | Examples |
|------|------------|----------|
| Variables | camelCase | `userId`, `isLoading` |
| Constants | UPPER_SNAKE_CASE | `JWT_SECRET`, `MAX_RETRY_COUNT` |
| Functions | camelCase (verb-first) | `getUser()`, `fetchWithdrawalHistory()` |
| Classes/PInterfaces | PascalCase | `AuthService`, `WithdrawalAddress` |
| Components | PascalCase | `DashboardLayout`, `WithdrawPage` |
| Files | kebab-case | `auth-service.ts`, `withdrawal-history.tsx` |
| Database tables | snake_case | `withdrawal_addresses`, `lending_positions` |

### Error Handling

- **Never leave errors uncaught** - always handle or rethrow
- **No empty catch blocks** (`catch(e) {}` is forbidden)
- **Use descriptive error messages** with context
- **Log errors** with structured context using logger
- **Throw Error instances**, not strings

```typescript
// ✅ Correct
try {
  await db.insert(users).values({ email, password });
} catch (error: any) {
  logger.error({ error: error.message, email }, 'Registration failed');
  throw new Error('User registration failed');
}

// ❌ Avoid
try {
  // ...
} catch (e) {}  // Never empty!
```

### Component Patterns

- **File naming**: PascalCase for components (`DashboardLayout.tsx`)
- **Props interface**: `ComponentNameProps` suffix
- **Default export** for page components
- **Named exports** for reusable components
- **Use Shadcn UI** components from `@/components/ui/*`
- **Use `lucide-react`** for icons
- **Use `sonner`** for toasts (not alert())

```typescript
// ✅ Correct
interface WithdrawPageProps {
  userId: number;
  asset?: string;
}

export const WithdrawPage = ({ userId, asset }: WithdrawPageProps) => {
  return <div>...</div>;
};

// Default export for pages
export default WithdrawPage;
```

### Service Layer Patterns

- **Location**: `src/lib/` for shared business logic
- **File naming**: kebab-case with `-service` suffix
- **Class-based** with static methods for stateless services
- **Zod schema** for input validation
- **Logger** for structured logging
- **Named exports** for services

```typescript
// ✅ Correct
import { z } from 'zod';
import { db } from './db.js';
import logger from './logger.js';

export const withdrawalSchema = z.object({
  addressId: z.number().int().positive(),
  amount: z.string(),
});

export class WithdrawalService {
  static async initiateWithdrawal(userId: number, data: z.infer<typeof withdrawalSchema>) {
    // implementation
  }
}
```

### Database (Drizzle ORM)

- **Table naming**: snake_case, plural (`users`, `withdrawal_addresses`)
- **Column naming**: snake_case (`created_at`, `user_id`)
- **Foreign keys**: `tableName_id` suffix (`user_id`, `withdrawal_id`)
- **Enums**: snake_case values (`'PENDING'`, `'COMPLETED'`)
- **Types**: Use `$inferSelect` and `$inferInsert`

```typescript
// ✅ Correct
export const withdrawals = pgTable('withdrawals', {
  id: serial('id').primaryKey(),
  userId: integer('user_id').references(() => users.id).notNull(),
  status: withdrawalStatusEnum('status').default('PENDING').notNull(),
});

export type Withdrawal = typeof withdrawals.$inferSelect;
export type NewWithdrawal = typeof withdrawals.$inferInsert;
```

---

## Directory Structure

```
src/
├── api/                    # Vercel Serverless Functions (endpoints)
├── components/
│   ├── ui/                 # Shadcn/Radix UI components
│   └── DashboardLayout.tsx # Layout components
├── db/
│   ├── schema.ts           # Drizzle schema definitions
│   └── migrations/         # Database migrations
├── lib/                    # Core service layer (business logic)
├── pages/                  # Route pages (React Router)
│   └── dashboard/          # Dashboard pages
├── hooks/                  # Custom React hooks
└── i18n/                   # Internationalization
```

---

## State Management

| Type | When to Use | Implementation |
|------|-------------|----------------|
| Server state | API data, caching | React Query (`@tanstack/react-query`) |
| UI state | Form inputs, dialogs | `useState` / `useReducer` |
| Auth state | JWT tokens, user info | `localStorage` + React Context |
| Navigation | Route params, URL state | React Router (`useSearchParams`) |

---

## API Routes

### 统一 Serverless Function 架构（强制）

**重要**: Vercel Hobby 计划限制最多 **12 个 Serverless Functions**。
项目必须使用统一的路由架构，所有 API 请求通过单一入口处理。

**正确的文件结构**:
```
api/
├── [...route].ts          # 统一路由处理器（唯一 Serverless Function）
└── __route__.test.ts      # 路由测试
```

**禁止的文件结构**（会导致超过 12 个函数限制）:
```
api/
├── auth/
│   ├── login.ts          # ❌ 单独的 Serverless Function
│   ├── register.ts       # ❌ 单独的 Serverless Function
│   └── logout.ts         # ❌ 单独的 Serverless Function
├── 2fa/
│   ├── setup.ts          # ❌ 单独的 Serverless Function
│   └── enable.ts         # ❌ 单独的 Serverless Function
└── ... (更多文件)
```

### 统一路由配置

所有路由在 `api/[...route].ts` 的 `ROUTE_CONFIG` 中集中配置：

```typescript
const ROUTE_CONFIG: Record<string, RouteConfig> = {
  // Auth endpoints
  'POST /auth/login': { requiresAuth: false, backendPath: '/api/auth/login' },
  'POST /auth/register': { requiresAuth: false, backendPath: '/api/auth/register' },
  'GET /auth/me': { requiresAuth: true, backendPath: '/api/auth/me' },
  
  // 2FA endpoints
  'POST /auth/2fa/setup': { requiresAuth: true, backendPath: '/api/auth/2fa/setup' },
  'POST /auth/2fa/enable': { requiresAuth: true, backendPath: '/api/auth/2fa/enable' },
  'POST /auth/2fa/disable': { requiresAuth: true, backendPath: '/api/auth/2fa/disable' },
  'GET /auth/2fa/status': { requiresAuth: true, backendPath: '/api/auth/2fa/status' },
  'POST /auth/2fa/verify-login': { requiresAuth: false, backendPath: '/api/auth/2fa/verify-login' },
  'POST /auth/2fa/skip': { requiresAuth: false, backendPath: '/api/auth/2fa/skip' },
  
  // Address endpoints
  'GET /addresses': { requiresAuth: true, backendPath: '/api/addresses' },
  'POST /addresses': { requiresAuth: true, backendPath: '/api/addresses' },
  
  // ... 其他路由
};
```

### 添加新 API 端点的步骤

1. **在 `ROUTE_CONFIG` 中添加配置**（不需要创建新文件）:
```typescript
'POST /new/endpoint': { 
  requiresAuth: true, 
  backendPath: '/api/new/endpoint' 
}
```

2. **在 Go 后端添加处理器**:
```go
// internal/routes/routes.go
protected.POST("/new/endpoint", h.NewEndpointHandler)
```

3. **在 `api/__route__.test.ts` 中添加测试**:
```typescript
it('should route POST /new/endpoint correctly', async () => {
  // 测试代码
});
```

### 动态路由支持

支持动态路由参数（如 `/addresses/:id`）:

```typescript
// Handle dynamic address routes: /addresses/123, /addresses/123/verify, etc.
if (path.startsWith('/addresses/')) {
  const isValidAddressRoute =
    /^\/addresses\/[\w-]+(\/verify|\/primary)?$/.test(path) &&
    (method === 'DELETE' || method === 'POST' || method === 'PUT' || method === 'PATCH');
  
  if (isValidAddressRoute) {
    return {
      found: true,
      config: { requiresAuth: true, backendPath: '' },
      backendPath: `/api${path}`,
    };
  }
}
```

### 旧的 API 路由模式（已废弃）

以下模式不再使用，仅作为参考：

```typescript
// ❌ 废弃：每个端点一个文件
// api/auth/login.ts
import type { VercelRequest, VercelResponse } from '@vercel/node';
export default async function handler(req: VercelRequest, res: VercelResponse) {
  // ...
}
```

---

## Developer Environment Tips

- **Setup**: Copy `.env.example` to `.env`, then run `npm install`
- **Port**: Vite dev server runs on port 8080 by default
- **Database**: Use `npm run db:push` to sync schema changes
- **Tests**: Run `npm test` before committing

---

## Key Files to Know

| File | Purpose |
|------|---------|
| `src/db/schema.ts` | Database schema definitions |
| `src/lib/auth-service.ts` | Authentication business logic |
| `src/lib/withdrawal-service.ts` | Withdrawal business logic |
| `api/` | API route handlers |
| `vite.config.ts` | Build configuration |
| `eslint.config.js` | Linting rules |
| `tailwind.config.ts` | Tailwind CSS configuration |

---

## Quick Reference

| Category | Rule |
|----------|------|
| Imports | Absolute `@/` for internal, relative for co-located |
| Types | Explicit, no `any`, use Zod for validation |
| Errors | Always handle, never empty catch, log with context |
| Components | Props interface, default export pages |
| Services | Static methods, Zod schema, structured logging |
| Files | kebab-case for non-components, PascalCase for components |
| Tests | Run single file with `npm test -- <path>` |

---

## Go Code Conventions (Legacy Backend in `internal/`)

- **Package**: lowercase, no underscore, no plural
- **Files**: lowercase + underscore, ≤ 300 lines
- **Context**: First param, for I/O only
- **Errors**: Always handle, no panic for business errors
- **Logging**: Handler layer only, structured logging
- **Tools**: Run `gofmt -w .` and `go vet ./...` before committing

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