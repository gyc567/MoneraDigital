# OpenSpec: 2FA Skip 401 错误修复

## 1. 问题描述

**用户报告**: 用户 `gyc567@gmail.com` 登录访问 `https://www.moneradigital.com/login`，点击 "Skip For Now" 按钮时报错：

```
POST https://www.moneradigital.com/api/auth/2fa/skip 401 (Unauthorized)
```

**注意**: 之前修复了 404 错误，现在出现了 401 错误。

## 2. 问题分析

### 2.1 后端测试

```bash
# 直接测试后端 API
curl -X POST https://www.moneradigital.com/api/auth/2fa/skip \
  -H "Content-Type: application/json" \
  -d '{"userId": 1}'
```

**结果**: HTTP 200 + access_token ✅

这说明后端 API 工作正常。

### 2.2 401 错误原因分析

后端 `Skip2FALogin` 处理器在以下情况返回 401：

1. **用户不存在**: `GetUserByID` 返回错误
2. **用户已启用 2FA**: `user.TwoFactorEnabled == true`

```go
func (s *AuthService) Skip2FAAndLogin(userID int) (*LoginResponse, error) {
    user, err := s.GetUserByID(userID)
    if err != nil {
        return nil, fmt.Errorf("user not found: %w", err)  // → 401
    }
    
    if user.TwoFactorEnabled {
        return nil, errors.New("cannot skip 2FA as it is enabled for this account")  // → 401
    }
    // ...
}
```

### 2.3 可能的原因

1. **前端发送的 userId 不正确**: `tempUserId` 可能为 null 或 undefined
2. **用户已启用 2FA**: 但登录流程却进入了 2FA 验证页面
3. **登录响应中没有 userId**: 前端无法获取正确的 userId

## 3. 修复方案

### 3.1 添加前端调试日志

在 `handleSkip2FA` 函数中添加日志，确认 `tempUserId` 的值：

```typescript
const handleSkip2FA = async () => {
  console.log('Skip 2FA clicked, tempUserId:', tempUserId);  // 添加调试日志
  if (!tempUserId) {
    console.error('tempUserId is null or undefined');  // 添加错误日志
    return;
  }
  // ...
};
```

### 3.2 改进错误处理

在前端显示更详细的错误信息：

```typescript
if (!response.ok) {
  console.error('Skip 2FA failed:', data);  // 添加调试日志
  throw new Error(data.error || data.message || 'Skip failed');
}
```

### 3.3 后端改进错误信息

让后端返回更详细的错误信息：

```go
if err != nil {
  c.JSON(http.StatusUnauthorized, gin.H{
    "error": err.Error(),
    "code": "SKIP_2FA_FAILED",
    "userId": req.UserID,
  })
  return
}
```

## 4. 实施步骤

### 步骤1: 添加前端调试

修改 `src/pages/Login.tsx`：

1. 在 `handleSkip2FA` 开头添加 `console.log`
2. 在错误处理中添加 `console.error`
3. 显示更详细的错误信息给用户

### 步骤2: 改进后端错误响应

修改 `internal/handlers/handlers.go` 的 `Skip2FALogin`：

1. 返回更详细的错误信息
2. 添加错误代码

### 步骤3: 部署和测试

1. 部署前端到 Vercel
2. 部署后端到 Replit
3. 让用户再次测试并提供浏览器控制台日志

## 5. 设计原则

### KISS
- 添加简单的调试日志
- 改进错误信息
- 不引入复杂逻辑

### 高内聚低耦合
- 前端负责显示错误
- 后端负责提供详细错误信息
- 保持现有架构不变

### 100% 测试覆盖
- 添加测试验证错误处理
- 测试各种错误场景

### 不影响其他功能
- 只添加日志和错误信息
- 不改变业务逻辑

## 6. 验证清单

- [ ] 前端调试日志已添加
- [ ] 后端错误信息已改进
- [ ] 部署完成
- [ ] 用户测试通过
- [ ] 浏览器控制台日志收集

## 7. 下一步

需要用户提供以下信息：
1. 浏览器控制台日志（按 F12 → Console）
2. Network 标签页中 `/api/auth/2fa/skip` 请求的详细信息
