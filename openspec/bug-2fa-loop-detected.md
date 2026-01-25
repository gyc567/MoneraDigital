# OpenSpec: 修复 2FA Setup 循环重定向导致 508 Loop Detected

## 1. 问题描述

### 错误现象
- **用户操作**: 登录后点击"启用 2FA"
- **报错信息**:
  ```
  Failed to load resource: the server responded with a status of 401 ()
  /api/auth/2fa/setup:1  Failed to load resource: the server responded with a status of 508 ()
  index-xxx.js:1236 2FA Setup error: SyntaxError: Unexpected token 'I', "Infinite l"... is not valid JSON
  ```
- **根本表现**: 返回字符串 "Infinite Loop Detected" 而非有效的 JSON 响应

### 影响范围
- **受影响的端点**: `/api/auth/2fa/setup`
- **关联端点**: `/api/auth/2fa/enable`, `/api/auth/2fa/disable`, `/api/auth/2fa/status`
- **用户体验**: 无法完成 2FA 设置流程

---

## 2. 原因分析 (3 个可能原因逐一排除)

### 原因 1: API 路由文件缺失 ❌ 排除
- **检查**: 确认 `api/auth/2fa/setup.ts` 是否存在
- **结果**: 文件 **不存在**
- **状态**: ✅ 找到问题，已创建

### 原因 2: Vercel 重定向配置循环 ❌ 排除
- **检查**: 分析 `vercel.json` rewrite 配置
- **配置**:
  ```json
  {
    "source": "/api/auth/2fa/setup",
    "destination": "https://www.moneradigital.com/api/auth/2fa/setup"
  }
  ```
- **问题**: 跨域名重定向导致无限循环
- **状态**: ✅ 找到问题，需修复

### 原因 3: 前端 API 调用逻辑问题 ✅ 排除
- **检查**: `Security.tsx` 中的 fetch 调用
- **代码**:
  ```typescript
  const res = await fetch("/api/auth/2fa/setup", {
    method: "POST",
    headers: { Authorization: `Bearer ${authToken}` }
  });
  ```
- **结果**: 前端调用正确，问题在服务端
- **状态**: ✅ 确认无问题

---

## 3. 根本原因

### 直接原因
1. **API 路由文件缺失**: `api/auth/2fa/setup.ts` 等关键文件不存在
2. **Vercel 重定向循环**: `vercel.json` 配置导致跨域名无限重定向
3. **依赖外部域名**: 所有请求都尝试转发到 `www.moneradigital.com`，形成循环

### 影响链路
```
前端 POST /api/auth/2fa/setup
  ↓ Vercel rewrite
  → https://www.moneradigital.com/api/auth/2fa/setup
  ↓ (循环) 再次 rewrite
  → 返回 "Infinite Loop Detected"
  ↓
  前端收到 508 + 字符串
  ↓ JSON.parse() 失败
  → SyntaxError: Unexpected token 'I'
```

---

## 4. 解决方案

### 4.1 创建缺失的 API 路由文件

| 文件 | 用途 |
|------|------|
| `api/auth/2fa/setup.ts` | 初始化 2FA，获取 QR 码 |
| `api/auth/2fa/enable.ts` | 启用 2FA，验证第一个 TOTP |
| `api/auth/2fa/disable.ts` | 禁用 2FA |
| `api/auth/2fa/status.ts` | 查询 2FA 状态 |

### 4.2 API 路由实现模式

```typescript
import type { VercelRequest, VercelResponse } from '@vercel/node';
import { verifyToken } from '../../../src/lib/auth-middleware.js';

const BACKEND_URL = process.env.VITE_API_BASE_URL || 'http://localhost:8081';

export default async function handler(req: VercelRequest, res: VercelResponse) {
  if (req.method !== 'POST') {
    return res.status(405).json({ error: 'Method Not Allowed' });
  }

  // 验证 JWT 令牌
  const user = verifyToken(req);
  if (!user) {
    return res.status(401).json({ code: 'AUTH_REQUIRED', message: 'Authentication required' });
  }

  try {
    // 纯转发到 Go 后端
    const response = await fetch(`${BACKEND_URL}/api/auth/2fa/setup`, {
      method: 'POST',
      headers: {
        'Authorization': req.headers.authorization || '',
        'Content-Type': 'application/json',
      },
    });

    const data = await response.json();
    return res.status(response.status).json(data);
  } catch (error: unknown) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    console.error('2FA Setup proxy error:', errorMessage);
    return res.status(500).json({ error: 'Internal Server Error' });
  }
}
```

### 4.3 修复 vercel.json (可选，保留本地路由优先)

当前配置保留，因为本地路由已创建，Vercel 会优先匹配文件路由。

---

## 5. 验证标准

### 5.1 单元验证
- [ ] `api/auth/2fa/setup.ts` 文件存在
- [ ] `api/auth/2fa/enable.ts` 文件存在
- [ ] `api/auth/2fa/disable.ts` 文件存在
- [ ] `api/auth/2fa/status.ts` 文件存在
- [ ] 文件无 TypeScript 编译错误

### 5.2 集成测试
- [ ] 本地环境: `POST /api/auth/2fa/setup` 返回 200 或 401
- [ ] 本地环境: `POST /api/auth/2fa/setup` 返回正确的 JSON 响应
- [ ] 前端 2FA 设置流程可正常完成

### 5.3 回归测试
- [ ] `/api/auth/me` 正常
- [ ] `/api/auth/login` 正常
- [ ] `/api/auth/register` 正常

---

## 6. 执行步骤

### Step 1: 创建 API 路由文件
- [x] 创建 `api/auth/2fa/setup.ts`
- [x] 创建 `api/auth/2fa/enable.ts`
- [x] 创建 `api/auth/2fa/disable.ts`
- [x] 创建 `api/auth/2fa/status.ts`

### Step 2: 修复 import 路径
- [x] 修正 `api/auth/2fa/*.ts` 中的 auth-middleware 导入路径

### Step 3: 运行验证
- [ ] 运行 TypeScript 编译检查
- [ ] 运行本地 API 测试

### Step 4: 部署验证
- [ ] 部署到 Vercel
- [ ] 验证生产环境 2FA 功能正常

---

## 7. 预防措施

1. **API 路由完整性检查**: 添加集成测试，确保所有前端调用的 API 端点都有对应实现
2. **Vercel 重定向测试**: 添加 CI 检查，防止跨域名重定向循环
3. **错误处理增强**: 确保 API 错误返回有效的 JSON，而非原始字符串
