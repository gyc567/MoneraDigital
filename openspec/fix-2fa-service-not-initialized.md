# 修复 2FA Service 未初始化错误

## 问题描述

用户 `gyc567@gmail.com` 在验证地址时遇到错误：
```
Failed to verify 2FA: two factor service not initialized
```

## 问题分析

### 错误来源
错误来自 `internal/services/auth.go:248-253`:
```go
func (s *AuthService) Verify2FA(userID int, token string) (bool, error) {
	if s.twoFactorService == nil {
		return false, errors.New("two factor service not initialized")
	}
	return s.twoFactorService.Verify(userID, token)
}
```

### 根本原因
在 `internal/container/container.go` 中，依赖注入顺序有误：

```go
// 问题代码
func NewContainer(db *sql.DB, jwtSecret string, opts ...ContainerOption) *Container {
	// 1. 创建 AuthService
	c.AuthService = services.NewAuthService(db, jwtSecret)
	
	// 2. 尝试注入 TwoFAService（此时还是 nil！）
	if c.TwoFAService != nil {
		c.AuthService.SetTwoFactorService(c.TwoFAService)
	}
	
	// 3. 应用选项函数才创建 TwoFAService
	for _, opt := range opts {
		opt(c)
	}
}
```

**问题**：`TwoFAService` 是在选项函数 `WithEncryption` 中创建的，但注入发生在选项函数应用之前。

## 修复方案

### 修复内容

调整依赖注入顺序：先应用选项函数创建 `TwoFAService`，再注入到 `AuthService`。

```go
// 修复后代码
func NewContainer(db *sql.DB, jwtSecret string, opts ...ContainerOption) *Container {
	// 1. 创建 AuthService
	c.AuthService = services.NewAuthService(db, jwtSecret)
	
	// 2. 应用选项函数创建 TwoFAService
	for _, opt := range opts {
		opt(c)
	}
	
	// 3. 注入 TwoFAService（现在已初始化）
	if c.TwoFAService != nil {
		c.AuthService.SetTwoFactorService(c.TwoFAService)
	}
}
```

## 实施步骤

1. ✅ 修改 `internal/container/container.go` 调整注入顺序
2. ✅ 添加单元测试验证修复
3. ✅ 运行回归测试确保不影响其他功能

## 测试策略

### 单元测试
- `TestNewContainer_WithEncryption_TwoFAServiceInjected` - 验证注入顺序
- `TestWithEncryption_CreatesTwoFAService` - 验证选项函数行为
- `TestContainer_Verify_IncludesTwoFAService` - 验证容器验证

### 回归测试
```bash
go test ./internal/...
# 结果: PASS
```

## 设计原则

1. **KISS**: 只调整代码顺序，不增加复杂度
2. **高内聚低耦合**: 保持 Container 的依赖注入职责单一
3. **测试覆盖**: 新增 3 个单元测试，100% 覆盖修复代码
4. **隔离性**: 只修改 Container 初始化逻辑，不影响其他功能
