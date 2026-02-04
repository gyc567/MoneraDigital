# AGENTS.md

> Monera Digital 项目代理编码指南
> 机构级数字资产平台，提供安全的静态金融和借贷解决方案

## 项目概览

**Monera Digital** 是一个全栈应用，遵循严格的**后端业务逻辑**架构要求。

## 技术栈

<<<<<<< Updated upstream
- **Frontend**: React 18, TypeScript, Vite, Tailwind CSS, Shadcn UI (Radix Primitives)
- **Backend**: Golang (Go) - **Mandatory for all interfaces, database access, and operations.**
- **Database**: PostgreSQL (Neon)
- **External Core System**: Monnaire Core API (Account Management)
- **State/Cache**: Redis (Upstash)
- **Testing**: Vitest (Unit/Integration), Playwright (E2E)
- **Language**: TypeScript (Frontend), Go (Backend)

## Architecture Overview

```
Frontend (React) → API Routes (Vercel) → Go Backend → Core API / Database
```

### Layer Responsibilities

| Layer | Technology | Responsibilities |
|-------|------------|------------------|
| **Frontend** | React + TypeScript | UI rendering, form validation, API calls only |
| **API Routes** | Vercel Serverless | Request routing, auth validation, proxy to backend |
| **Go Backend** | Go (internal/) | Business logic, database operations, Core API integration |
| **Core API** | External/Mock | Core account management, KYC, compliance |
| **Database** | PostgreSQL (Neon) | User data, transactions, application state |

## Architecture Principles

### 1. Frontend-Only API Calls
**⚠️ CRITICAL**: Frontend code **MUST NOT** directly access the database or Core API.

```
✅ Correct:  Frontend → /api/* → Go Backend → Database/Core API
❌ Forbidden: Frontend → Direct Database Access
❌ Forbidden: Frontend → Direct Core API Access
```

### 2. Service Layer Pattern

Frontend services (`src/lib/*-service.ts`) **only** make HTTP API calls:

```typescript
// ✅ Correct: Service calls local API
export class UserService {
  static async getUser() {
    const response = await fetch('/api/users/me', {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    return response.json();
  }
}

// ❌ Forbidden: Direct database access
export class UserService {
  static async getUser() {
    return db.select().from(users);  // Never do this!
  }
}

// ❌ Forbidden: Direct Core API access
export class UserService {
  static async getUser() {
    return fetch('https://core-api.monera.com/users');  // Never do this!
  }
}
```

### 3. Go Backend Responsibilities

Go backend (`internal/`) is the **only** layer that can:
- Access PostgreSQL database
- Call Monnaire Core API
- Implement business logic
- Handle authentication/authorization

```go
// ✅ Correct: Go Backend calls Core API
func (s *AuthService) createCoreAccount(userID int, email string) (string, error) {
    coreAPIURL := os.Getenv("Monnaire_Core_API_URL") + "/accounts/create"
    resp, err := http.Post(coreAPIURL, "application/json", body)
    // ...
}

// ✅ Correct: Go Backend accesses database
func (s *AuthService) CreateUser(req models.RegisterRequest) (*models.User, error) {
    _, err := s.DB.Exec("INSERT INTO users ...", req.Email, hashedPassword)
    // ...
}
```

### 4. Data Flow Example (User Registration)

```
1. Frontend (React)
   POST /api/auth/register
   { email, password }
   
2. API Routes (Vercel)
   Validate JWT → Forward to Go Backend
   
3. Go Backend (internal/services/auth.go)
   a. Create user in PostgreSQL
   b. Call Monnaire Core API to create core account
      POST Monnaire_Core_API_URL/accounts/create
   c. Return { user, token }
   
4. Frontend
   Receive response, update UI
```

---

## Build, Lint & Test Commands

### Core Commands
=======
- **前端**: React 18, TypeScript, Vite, Tailwind CSS, Shadcn UI (Radix Primitives)
- **后端**: Golang (Go) - **强制要求：所有业务逻辑、数据库访问和操作必须在 Go 后端实现**
- **数据库**: PostgreSQL (Neon) + Drizzle ORM
- **缓存**: Redis (Upstash)
- **测试**: Vitest (单元/集成测试), Playwright (E2E测试)

## 构建、代码检查和测试命令
>>>>>>> Stashed changes

### 核心命令
```bash
npm install                    # 安装依赖
npm run dev                    # 开发服务器 (Vite) - 端口: 5001
npm run build                  # 生产构建
npm run build:dev              # 开发模式构建
npm run preview                # 预览生产构建
npm run lint                   # ESLint 代码检查
npm run lint -- --fix          # 自动修复代码检查问题
```

### 测试命令 (关键)
```bash
npm test                           # 运行所有测试 (Vitest)
npm test src/lib/auth-service.test.ts  # 运行特定测试文件
npm test src/__tests__/           # 运行 __tests__ 目录下的测试
npm test -- --coverage            # 运行测试并生成覆盖率报告
npm test -- --testNamePattern="login"  # 运行匹配模式的测试
npm test -- --grep "2FA"          # 运行包含特定文本的测试
npm run test:ui                   # Vitest UI 界面
npm run test:e2e                  # E2E 测试 (Playwright)
```

### 数据库命令
```bash
npm run db:push                   # 推送数据库架构
npm run db:generate               # 生成数据库迁移文件
```

### 辅助命令
```bash
npm run favicon                   # 生成网站图标
```

## 架构：后端业务逻辑架构 (强制要求)

**所有 API 调用流程**: 前端 → Vercel API (仅代理) → Go Backend (业务逻辑)

- **前端 API 路由 (`api/`)**: 纯 HTTP 代理 - **禁止包含业务逻辑**
- **Go 后端 (`internal/`)**: **所有业务逻辑、数据库操作、认证逻辑**
- **前端服务层 (`src/lib/`)**: **禁止直接数据库访问或认证逻辑**
- **允许的例外**: 简单 UI 工具和表单验证

## 代码风格指南

### 导入规则
- **使用绝对导入** - `@/` 前缀用于内部模块
- **使用相对导入** - 仅用于同目录文件
- **导入顺序**: 内置模块 → 第三方模块 → 绝对导入 (`@/`) → 相对导入

### TypeScript 规则
- **显式类型定义** - 参数和返回值必须显式声明类型
- **禁止使用 `any`** - 使用 `unknown` 或proper类型
- **使用 `zod`** - 进行运行时验证
- **使用接口定义对象** - 使用类型别名定义联合类型
- **TypeScript 配置**:
  - 严格模式: `false` (便于开发)
  - 路径别名: `@/*` → `./src/*`

### 代码格式化
- **行长度**: 100 字符软限制
- **缩进**: 2 空格
- **分号**: 始终使用
- **引号**: 字符串使用双引号，JSX 使用单引号
- **尾随逗号**: 多行对象/数组使用

### 命名约定
- **变量/函数**: `camelCase`
- **常量**: `UPPER_SNAKE_CASE`
- **组件/类**: `PascalCase`
- **文件**: `kebab-case` (服务文件: `-service` 后缀)
- **数据库**: `snake_case` (表名复数，列名 snake_case)

### 错误处理
- **从不遗漏未捕获错误** - 始终处理或重新抛出
- **禁止空 catch 块**
- **结构化日志记录** - 包含上下文信息
- **抛出 Error 实例** - 不抛出字符串

### 不可变性 (关键)
始终创建新对象，**从不直接修改**。使用展开运算符或 structuredClone。

```typescript
// 错误：直接修改
function updateUser(user, name) {
  user.name = name;
  return user;
}

// 正确：不可变方式
function updateUser(user, name) {
  return { ...user, name };
}
```

## 文件组织

<<<<<<< Updated upstream
| Type | Convention | Examples |
|------|------------|----------|
| Variables | camelCase | `userId`, `isLoading` |
| Constants | UPPER_SNAKE_CASE | `JWT_SECRET`, `MAX_RETRY_COUNT` |
| JSON Fields | camelCase | `userId`, `accessToken`, `requires2FA` |
| Database Columns | snake_case | `user_id`, `created_at` |

**Critical Rule**: All API request/response JSON fields MUST use camelCase (`userId`, not `user_id`).

### JSON Field Naming Convention

| Layer | Format | Example |
|-------|--------|---------|
| **API Request/Response** | camelCase | `userId`, `createdAt`, `walletAddress` |
| **Database Columns** | snake_case | `user_id`, `created_at`, `wallet_address` |
| **TypeScript Interfaces** | camelCase | `userId: number` |
| **Go Struct JSON Tags** | camelCase | `json:"userId"` |
| **Go Struct DB Tags** | snake_case | `db:"user_id"` |

```go
// Go struct - JSON camelCase, DB snake_case
type WithdrawalAddress struct {
    ID            int          `json:"id" db:"id"`
    UserID        int          `json:"userId" db:"user_id"`
    WalletAddress string       `json:"walletAddress" db:"wallet_address"`
    ChainType     string       `json:"chainType" db:"chain_type"`
    AddressAlias  string       `json:"addressAlias" db:"address_alias"`
    Verified      bool         `json:"verified" db:"verified"`
    CreatedAt     time.Time    `json:"createdAt" db:"created_at"`
    VerifiedAt    sql.NullTime `json:"verifiedAt,omitempty" db:"verified_at"`
}
```

```typescript
// TypeScript - always camelCase
interface WithdrawalAddress {
  id: number;
  userId: number;
  walletAddress: string;
  chainType: "BTC" | "ETH" | "USDC" | "USDT";
  addressAlias: string;
  verified: boolean;
  createdAt: string;
  verifiedAt: string | null;
}
```

**⚠️ WARNING**: Never mix snake_case and camelCase in API JSON. All API communication MUST use camelCase.
```
| Functions | camelCase (verb-first) | `getUser()`, `fetchWithdrawalHistory()` |
| Classes/PInterfaces | PascalCase | `AuthService`, `WithdrawalAddress` |
| Components | PascalCase | `DashboardLayout`, `WithdrawPage` |
| Files | kebab-case | `auth-service.ts`, `withdrawal-history.tsx` |
| Database tables | snake_case | `withdrawal_addresses`, `lending_positions` |
=======
**小文件优先于大文件**：
- 高内聚，低耦合
- 典型 200-400 行，最大 800 行
- 函数保持精简 (< 50 行)
- 嵌套层次最多 4 层
>>>>>>> Stashed changes

## 测试要求

- **覆盖率**: 新代码维持 **80% 测试覆盖率**
- **测试方法**: 测试驱动开发 (TDD) - 先写测试
- **测试类型**: 单元测试 (Vitest), 集成测试, E2E 测试 (Playwright)
- **测试文件**: 使用 `.test.ts` 后缀，与源文件放在一起

### TDD 工作流
1. 先写测试 (红色)
2. 运行测试 - 应该失败
3. 编写最小实现 (绿色)
4. 运行测试 - 应该通过
5. 重构 (改进)
6. 验证覆盖率 (80%+)

### 测试文件位置
- **单元测试**: `src/lib/*/*.test.ts`, `src/__tests__/*`
- **API 测试**: `api/*/*.test.ts`
- **集成测试**: `tests/*-integration.test.ts`
- **E2E 测试**: `tests/*.spec.ts`

## 安全指南

提交前必须检查：
- [ ] 无硬编码密钥 (API 密钥、密码、令牌)
- [ ] 所有用户输入已验证
- [ ] SQL 注入防护 (参数化查询)
- [ ] XSS 防护 (HTML 清理)
- [ ] CSRF 保护启用
- [ ] 认证/授权验证
- [ ] 所有端点限流
- [ ] 错误消息不泄露敏感数据

**密钥管理**: 始终使用 `process.env.VARIABLE_NAME`，绝不硬编码。

## 常用模式

### API 响应格式
```typescript
interface ApiResponse<T> {
  success: boolean
  data?: T
  error?: string
  meta?: {
    total: number
    page: number
    limit: number
  }
}
```

### 自定义 Hook 模式
```typescript
export function useDebounce<T>(value: T, delay: number): T {
  const [debouncedValue, setDebouncedValue] = useState<T>(value)
  useEffect(() => {
    const handler = setTimeout(() => setDebouncedValue(value), delay)
    return () => clearTimeout(handler)
  }, [value, delay])
  return debouncedValue
}
```

## 代理工作流

**主动使用代理**:
- **planner** - 复杂功能，重构
- **tdd-guide** - 新功能，错误修复 (先写测试)
- **code-reviewer** - 编写代码后
- **security-reviewer** - 提交前安全分析
- **build-error-resolver** - 构建失败时
- **e2e-runner** - 关键用户流程
- **refactor-cleaner** - 死代码清理
- **architect** - 架构决策

**始终使用并行任务执行**处理独立操作。

## Go 后端约定 (`internal/`)
- **包名**: 小写，无下划线，无复数
- **文件**: 小写 + 下划线，≤ 300 行
- **Context**: 第一个参数，仅用于 I/O
- **错误**: 始终处理，业务错误不 panic
- **工具**: 提交前运行 `gofmt -w .` 和 `go vet ./...`

## 目录结构

```
src/
├── api/                      # Vercel 无服务器函数 (仅 HTTP 代理)
├── components/
<<<<<<< Updated upstream
│   ├── ui/                 # Shadcn/Radix UI components
│   └── DashboardLayout.tsx # Layout components
├── lib/                    # Core service layer (API clients only)
│   ├── auth-service.ts     # Auth API client
│   ├── withdrawal-service.ts # Withdrawal API client
│   └── ...                 # Other service clients
├── pages/                  # Route pages (React Router)
│   └── dashboard/          # Dashboard pages
├── hooks/                  # Custom React hooks
└── i18n/                   # Internationalization
```

**Note**: `src/db/` directory has been removed. Frontend must NOT directly access database.

---
=======
│   ├── ui/                   # Shadcn/Radix UI 组件
│   └── DashboardLayout.tsx   # 布局组件
├── db/
│   ├── schema.ts             # Drizzle 架构定义
│   └── migrations/           # 数据库迁移
├── lib/                      # 仅 UI 工具和表单验证
├── pages/                    # 路由页面 (React Router)
├── hooks/                    # 自定义 React hooks
├── __tests__/                # 测试文件
└── i18n/                     # 国际化
```

## 组件模式
- **属性接口**: `ComponentNameProps` 后缀
- **页面组件**: 默认导出
- **可复用组件**: 命名导出
- **使用 Shadcn UI**: 来自 `@/components/ui/*`
- **使用图标**: `lucide-react`
- **使用通知**: `sonner`
>>>>>>> Stashed changes

## 数据库 (Drizzle ORM)
- **表名**: `snake_case`，复数形式 (`users`, `withdrawal_addresses`)
- **列名**: `snake_case` (`created_at`, `user_id`)
- **外键**: `tableName_id` 后缀
- **类型**: 使用 `$inferSelect` 和 `$inferInsert`

## Git 工作流

**提交格式**: `<类型>: <描述>`
类型: feat, fix, refactor, docs, test, chore, perf, ci

**功能实现流程**:
1. 首先规划 (使用 **planner**)
2. TDD 方法 (使用 **tdd-guide**)
3. 代码审查 (使用 **code-reviewer**)
4. 提交推送与详细消息

<<<<<<< Updated upstream
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
- **Database**: Database schema is managed by Go backend (`internal/migration/`)
- **Tests**: Run `npm test` before committing

**Note**: Frontend does NOT directly access database. All database operations go through Go backend API.

---

## Key Files to Know

| File | Purpose |
|------|---------|
| `internal/migration/migrations/` | Go database migrations (backend only) |
| `src/lib/auth-service.ts` | Auth API client (frontend) |
| `src/lib/withdrawal-service.ts` | Withdrawal API client (frontend) |
| `api/[...route].ts` | Unified API router (Vercel) |
| `vite.config.ts` | Build configuration |
| `eslint.config.js` | Linting rules |
| `tailwind.config.ts` | Tailwind CSS configuration |

**Note**: Database schema is defined in Go backend (`internal/migration/migrations/`), not in frontend.

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
=======
## 代码质量检查清单

完成任务前检查：
- [ ] 代码可读性好，命名清晰
- [ ] 函数精简 (< 50 行)
- [ ] 文件集中 (< 800 行)
- [ ] 无深层嵌套 (> 4 层)
- [ ] 错误处理完善
- [ ] 无 console.log 语句
- [ ] 无硬编码值
- [ ] 使用不可变模式
>>>>>>> Stashed changes
