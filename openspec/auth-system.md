# OpenSpec: 用户注册与登录系统

## 1. 目标
实现一个简洁、高内聚、低耦合的用户身份验证系统，包括注册和登录功能。

## 2. 技术栈
- **后端**: Vercel Serverless Functions (TypeScript)
- **数据库**: Neon PostgreSQL
- **加密**: `bcryptjs` (密码哈希)
- **身份验证**: `jsonwebtoken` (JWT)
- **测试**: `vitest` (100% 逻辑覆盖率)

## 3. 数据库模式 (Schema)
```sql
CREATE TABLE IF NOT EXISTS users (
  id SERIAL PRIMARY KEY,
  email TEXT UNIQUE NOT NULL,
  password TEXT NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

## 4. API 设计
- `POST /api/auth/register`: 注册新用户。
- `POST /api/auth/login`: 用户登录并返回 JWT。

## 5. 实现计划
1.  **基础设施**:
    - 配置数据库连接池。
    - 编写数据库初始化脚本。
2.  **后端逻辑**:
    - `src/lib/auth-service.ts`: 处理核心业务逻辑（哈希、验证、JWT 生成）。
    - `api/auth/register.ts`: 注册接口。
    - `api/auth/login.ts`: 登录接口。
3.  **前端页面**:
    - `src/pages/Register.tsx`: 注册页面。
    - `src/pages/Login.tsx`: 登录页面。
4.  **测试**:
    - 编写 `auth-service.test.ts` 确保 100% 覆盖业务逻辑。

## 6. 设计原则
- **KISS**: 不使用过度复杂的 ORM，直接使用 `postgres.js` 进行高效查询。
- **高内聚**: 认证逻辑集中在 `auth-service.ts`。
- **低耦合**: API 层仅负责 HTTP 交互，业务逻辑解耦。
