# Monera Digital 架构 V2 - Core API 集成

## 架构概述

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              架构分层                                     │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────────────┐    │
│  │   Frontend   │────▶│  API Routes  │────▶│   Go Backend         │    │
│  │   (React)    │     │  (Vercel)    │     │   (internal/)        │    │
│  └──────────────┘     └──────────────┘     └──────────────────────┘    │
│                                                     │                   │
│                                                     ▼                   │
│                                          ┌──────────────────────┐      │
│                                          │   Monnaire_Core_     │      │
│                                          │   API_URL            │      │
│                                          │   (外部核心账户系统)  │      │
│                                          └──────────────────────┘      │
│                                                     │                   │
│                                                     ▼                   │
│                                          ┌──────────────────────┐      │
│                                          │   PostgreSQL         │      │
│                                          │   (Neon)             │      │
│                                          └──────────────────────┘      │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

## 架构说明

### 1. Frontend (React + TypeScript)
- **职责**: 用户界面、表单验证、API 调用
- **禁止**: 直接访问数据库、直接访问 Core API
- **只能**: 调用 `/api/*` 端点

### 2. API Routes (Vercel Serverless)
- **职责**: 统一路由入口、认证校验、请求转发
- **禁止**: 业务逻辑、数据库操作
- **只能**: 代理请求到 Go Backend

### 3. Go Backend (internal/)
- **职责**: 业务逻辑、数据库操作、外部 API 调用
- **可以**: 
  - 操作 PostgreSQL 数据库
  - 调用 Monnaire_Core_API_URL
  - 处理业务规则

### 4. Monnaire_Core_API_URL (外部系统)
- **职责**: 核心账户管理、KYC、合规
- **由 Go Backend 调用**: 不是前端直接调用
- **当前实现**: Mock API (`/api/core/*`)
- **生产环境**: 外部真实 Core Account System

## 数据流示例

### 用户注册流程

```
1. Frontend
   POST /api/auth/register
   { email, password }
        │
        ▼
2. API Routes (Vercel)
   验证 JWT → 转发到 Go Backend
        │
        ▼
3. Go Backend (internal/services/auth.go)
   a. 创建用户到 PostgreSQL
   b. 调用 Monnaire_Core_API_URL 创建 Core Account
      POST /api/core/accounts/create
      { externalId: userID, accountType: "INDIVIDUAL" }
        │
        ▼
4. Core API (Mock/External)
   创建核心账户
   返回 accountId
        │
        ▼
5. Go Backend
   返回 { user, token } 给 Frontend
```

### 提现流程

```
1. Frontend
   POST /api/withdrawals
   { addressId, amount, asset, twoFactorToken }
        │
        ▼
2. API Routes (Vercel)
   验证 JWT → 转发到 Go Backend
        │
        ▼
3. Go Backend (internal/handlers/handlers.go)
   a. 验证 2FA
   b. 检查余额 (PostgreSQL)
   c. 创建提现订单
   d. 调用 Core API 冻结资金 (可选)
        │
        ▼
4. Core API (Mock/External)
   处理资金冻结/划转
        │
        ▼
5. Go Backend
   返回 { order } 给 Frontend
```

## 代码规范

### Frontend 规范
```typescript
// ✅ 正确：调用 API
const response = await fetch('/api/withdrawals', {
  method: 'POST',
  headers: { 'Authorization': `Bearer ${token}` },
  body: JSON.stringify(data)
});

// ❌ 禁止：直接访问数据库
import { db } from '@/db';
await db.insert(withdrawals).values(data);

// ❌ 禁止：直接调用 Core API
await fetch('https://core-api.monera.com/accounts/create');
```

### Go Backend 规范
```go
// ✅ 正确：调用 Core API
func (s *AuthService) createCoreAccount(userID int, email string) (string, error) {
    coreAPIURL := os.Getenv("Monnaire_Core_API_URL") + "/accounts/create"
    resp, err := http.Post(coreAPIURL, "application/json", body)
    // ...
}

// ✅ 正确：操作数据库
func (s *AuthService) CreateUser(req models.RegisterRequest) (*models.User, error) {
    _, err := s.DB.Exec("INSERT INTO users ...", req.Email, hashedPassword)
    // ...
}
```

## 环境变量

```bash
# Frontend (.env)
VITE_API_URL=/api  # 只调用本地 API

# Go Backend (.env)
DATABASE_URL=postgresql://...        # PostgreSQL 连接
Monnaire_Core_API_URL=http://...     # Core API 地址
JWT_SECRET=...                       # JWT 密钥
```

## 部署架构

```
┌─────────────────────────────────────────┐
│           Vercel (Frontend + API)       │
│  ┌─────────────┐    ┌──────────────┐   │
│  │  React App  │    │ API Routes   │   │
│  │  (Static)   │    │ (Serverless) │   │
│  └─────────────┘    └──────────────┘   │
└─────────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────┐
│        Go Backend (Replit/Server)       │
│  ┌─────────────────────────────────┐    │
│  │  - Business Logic               │    │
│  │  - Database Access (PostgreSQL) │    │
│  │  - Core API Integration         │    │
│  └─────────────────────────────────┘    │
└─────────────────────────────────────────┘
                   │
         ┌─────────┴─────────┐
         ▼                   ▼
┌──────────────┐    ┌─────────────────┐
│  PostgreSQL  │    │  Core API       │
│  (Neon)      │    │  (External)     │
└──────────────┘    └─────────────────┘
```

## 关键文件

| 层级 | 文件 | 说明 |
|------|------|------|
| Frontend | `src/lib/*-service.ts` | API 客户端 |
| API Routes | `api/[...route].ts` | 统一路由 |
| Go Backend | `internal/services/*.go` | 业务逻辑 |
| Go Backend | `internal/handlers/*.go` | HTTP 处理器 |
| Core API | `internal/handlers/core/*.go` | Core API Mock/Client |

## 总结

1. **Frontend** 只调用 `/api/*`，不直接访问数据库或 Core API
2. **API Routes** 只做代理，无业务逻辑
3. **Go Backend** 处理所有业务，包括数据库和 Core API 调用
4. **Core API** 是外部系统，由 Go Backend 调用
