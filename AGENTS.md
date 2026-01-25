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

- **Location**: `api/` directory
- **Naming**: File-based routing (`api/auth/login.ts` → `/api/auth/login`)
- **Method handlers**: Check `req.method` in handler
- **Auth**: Use `verifyToken()` from auth middleware
- **Response**: Return JSON with `res.status().json()`

```typescript
// ✅ Correct
import type { VercelRequest, VercelResponse } from '@vercel/node';
import { verifyToken } from '../../src/lib/auth-middleware.js';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method === 'GET') {
    const user = verifyToken(req);
    if (!user) return res.status(401).json({ error: 'Unauthorized' });
    // handle GET
  }
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