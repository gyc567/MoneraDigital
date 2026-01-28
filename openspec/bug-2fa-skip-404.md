# OpenSpec: 2FA Skip 404 错误修复

## 1. 问题描述

**用户报告**: 用户 `gyc567@gmail.com` 登录访问 `https://www.moneradigital.com/login`，点击 "Skip For Now" 按钮时报错：

```
POST https://www.moneradigital.com/api/auth/2fa/skip 404 (Not Found)
```

## 2. 根因分析

### 2.1 前端代码分析
- 文件: `src/pages/Login.tsx`
- 函数: `handleSkip2FA()` 第126-155行
- 请求路径: `/api/auth/2fa/skip`
- 请求方法: `POST`
- 请求体: `{ userId: tempUserId }`

前端代码正确，问题出在后端路由或API网关配置上。

### 2.2 API路由配置分析
- 文件: `api/[...route].ts`
- 路由表第35行已配置: `'POST /auth/2fa/skip': { requiresAuth: false, backendPath: '/api/auth/2fa/skip' }`
- 统一路由器应该正确转发此请求到Go后端

### 2.3 Go后端路由分析
- 文件: `internal/routes/routes.go`
- 第59行已配置: `auth.POST("/2fa/skip", h.Skip2FALogin)`
- 处理器: `internal/handlers/handlers.go` 第214-250行 `Skip2FALogin` 函数
- 服务层: `internal/services/auth.go` 第224-252行 `Skip2FAAndLogin` 函数

### 2.4 问题定位

经过分析，发现代码层面都已经实现，但可能存在以下问题：

1. **Vercel部署问题**: 生产环境的Vercel函数可能没有正确部署或缓存问题
2. **路由匹配问题**: 统一路由器中的路由匹配逻辑可能对某些路径处理不正确
3. **后端服务未启动**: Go后端服务可能没有正确启动或监听

## 3. 修复方案

### 3.1 验证路由配置
确认 `api/[...route].ts` 中的路由配置正确无误。

### 3.2 增强路由调试
在统一路由器中添加更详细的日志记录，帮助排查404问题。

### 3.3 验证后端服务
确保Go后端服务正确启动，并且路由注册成功。

## 4. 设计原则 (KISS)

- **解耦**: 前端只负责调用API，不处理业务逻辑
- **内聚**: 2FA跳过逻辑集中在后端 `AuthService.Skip2FAAndLogin`
- **简单**: 保持现有架构不变，仅修复404问题

## 5. 测试要求

- 100% 覆盖 `Skip2FALogin` 处理器的单元测试
- 100% 覆盖 `Skip2FAAndLogin` 服务层的单元测试
- 集成测试验证端到端流程

## 6. 验证标准

- [ ] 用户点击 "Skip For Now" 按钮不再出现404错误
- [ ] 用户成功登录并跳转到仪表板
- [ ] 已启用2FA的用户点击跳过应返回错误提示
- [ ] 所有测试通过，覆盖率100%
