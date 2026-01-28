# OpenSpec: 统一 Serverless Function 架构

## 1. 目标

解决 Vercel Hobby 计划限制（最多 12 个 Serverless Functions）的问题，通过统一的路由处理器将所有 API 请求集中到一个 Serverless Function 中。

## 2. 问题背景

Vercel Hobby 计划限制每个部署最多 12 个 Serverless Functions。当项目需要多个 API 端点时，这个限制很容易被突破。

### 2.1 常见错误
```
No more than 12 Serverless Functions can be added to a Deployment 
on the Hobby plan. Create a team (Pro plan) to deploy more.
```

## 3. 解决方案

### 3.1 统一路由架构

使用单一入口文件 `api/[...route].ts` 处理所有 API 请求：

```
api/
├── [...route].ts          # 统一路由处理器（唯一 Serverless Function）
└── __route__.test.ts      # 路由测试
```

### 3.2 路由配置

所有路由在 `ROUTE_CONFIG` 中集中配置：

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

### 3.3 动态路由支持

支持动态路由参数，如 `/addresses/:id`：

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

## 4. 设计原则 (KISS)

### 4.1 单一职责
- **API 路由层**: 只负责请求转发和认证检查
- **后端服务层**: 处理所有业务逻辑
- **前端**: 只负责调用 API，不处理业务逻辑

### 4.2 高内聚低耦合
- 所有路由配置集中在一个文件中
- 路由处理器与业务逻辑完全分离
- 新增 API 端点只需修改配置，无需新增文件

### 4.3 可扩展性
- 通过配置即可添加新路由
- 支持动态路由参数
- 支持多种 HTTP 方法

## 5. 技术规范

### 5.1 文件结构

```
/Users/eric/dreame/code/MoneraDigital/
├── api/
│   ├── [...route].ts          # 统一路由处理器
│   └── __route__.test.ts      # 路由测试
├── src/
│   └── lib/
│       └── auth-middleware.ts # 认证中间件
├── internal/                   # Go 后端服务
│   ├── routes/
│   │   └── routes.go          # 后端路由定义
│   └── handlers/
│       └── handlers.go        # 后端处理器
└── vercel.json                # Vercel 配置
```

### 5.2 Vercel 配置

```json
{
  "framework": "vite",
  "buildCommand": "npm run build",
  "outputDirectory": "dist",
  "rewrites": [
    {
      "source": "/api/(.*)",
      "destination": "/api/[...route]?route=$1"
    },
    {
      "source": "/(.*)",
      "destination": "/index.html"
    }
  ]
}
```

### 5.3 环境变量

```
# Development
BACKEND_URL=http://localhost:8081

# Production
BACKEND_URL=https://your-backend-url.com
```

## 6. 添加新路由的步骤

1. **在 `ROUTE_CONFIG` 中添加配置**:
```typescript
'POST /new/endpoint': { 
  requiresAuth: true, 
  backendPath: '/api/new/endpoint' 
}
```

2. **在后端添加对应处理器**:
```go
// internal/routes/routes.go
protected.POST("/new/endpoint", h.NewEndpointHandler)
```

3. **添加测试**:
```typescript
// api/__route__.test.ts
it('should route POST /new/endpoint correctly', async () => {
  // 测试代码
});
```

## 7. 验证标准

- [x] 只有一个 Serverless Function 文件
- [x] 所有 API 请求通过统一入口
- [x] 支持动态路由
- [x] 支持认证检查
- [x] 100% 测试覆盖
- [x] 不影响 Hobby 计划部署

## 8. 注意事项

### 8.1 避免创建多个 API 文件
❌ 不要这样做：
```
api/
├── auth/
│   ├── login.ts
│   ├── register.ts
│   └── logout.ts
├── 2fa/
│   ├── setup.ts
│   ├── enable.ts
│   └── disable.ts
└── ... (更多文件)
```

✅ 应该这样做：
```
api/
├── [...route].ts          # 唯一文件
└── __route__.test.ts
```

### 8.2 冷启动优化
- 统一函数减少了冷启动次数
- 共享代码和资源
- 更好的性能表现

## 9. 参考资料

- [Vercel Serverless Functions Limits](https://vercel.com/docs/concepts/limits/overview#serverless-functions)
- [Vercel Hobby Plan](https://vercel.com/pricing)
