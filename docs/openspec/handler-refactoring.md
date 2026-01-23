# Handler Refactoring OpenSpec

> **目标**: 提升代码质量，满足 KISS、高内聚低耦合、100% 测试覆盖率要求
> 
> **范围**: `internal/handlers/` 和 `internal/container/`
> 
> **版本**: v1.0
> 
> **作者**: Sisyphus Architecture Team

---

## 📋 目录

1. [背景](#背景)
2. [架构原则](#架构原则)
3. [设计决策](#设计决策)
4. [实现计划](#实现计划)
5. [测试策略](#测试策略)

---

## 背景

### 当前问题

1. **代码重复**: `twofa_handler.go` 中 4 个方法有相同用户验证逻辑
2. **类型断言不安全**: 缺少类型检查保护
3. **错误响应不统一**: 多种错误格式混用
4. **Container 膨胀**: 参数过多，违反单一职责
5. **零测试覆盖**: 新代码无测试

### 预期收益

| 改进项 | 当前 | 目标 | 收益 |
|--------|------|------|------|
| 代码重复 | ~30% | 0% | 减少维护成本 |
| 测试覆盖 | 0% | 100% | 质量保证 |
| 类型安全 | 警告 | 编译期检查 | 稳定性 |
| 错误格式 | 3种 | 1种 | 可预测性 |

---

## 架构原则

### 1. KISS (Keep It Simple, Stupid)

- 每个文件 < 200 行
- 每个函数 < 20 行
- 单一职责

### 2. 高内聚低耦合

- Handler 只负责 HTTP 解析和响应
- Service 负责业务逻辑
- 依赖通过接口注入

### 3. 组合优于继承

```go
// ❌ 避免: 继承
type TwoFAHandler struct {
    BaseHandler  // 继承
}

// ✅ 推荐: 组合
type TwoFAHandler struct {
    base *BaseHandler  // 组合
}
```

---

## 设计决策

### 决策 1: BaseHandler 提取

```go
// internal/handlers/base.go

package handlers

import (
    "errors"
    "net/http"

    "github.com/gin-gonic/gin"
)

// BaseHandler 封装通用的 HTTP 处理逻辑
type BaseHandler struct{}

// getUserID 从上下文中获取用户 ID
func (h *BaseHandler) getUserID(c *gin.Context) (int, bool) {
    userID, exists := c.Get("userID")
    if !exists {
        return 0, false
    }
    id, ok := userID.(int)
    return id, ok
}

// getUserEmail 从上下文中获取用户邮箱
func (h *BaseHandler) getUserEmail(c *gin.Context) (string, bool) {
    email, exists := c.Get("email")
    if !exists {
        return "", false
    }
    emailStr, ok := email.(string)
    return emailStr, ok
}

// requireUserID 确保用户已认证
func (h *BaseHandler) requireUserID(c *gin.Context) (int, bool) {
    id, ok := h.getUserID(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{
            "error":   "Unauthorized",
            "code":    "AUTH_REQUIRED",
        })
        return 0, false
    }
    return id, true
}

// bindTokenRequest 绑定并验证 TOTP token 请求
func (h *BaseHandler) bindTokenRequest(c *gin.Context) (string, bool) {
    var req struct {
        Token string `json:"token" binding:"required"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "Token is required",
            "code":    "INVALID_REQUEST",
        })
        return "", false
    }

    if len(req.Token) != 6 {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "Token must be 6 digits",
            "code":    "INVALID_TOKEN_FORMAT",
        })
        return "", false
    }

    return req.Token, true
}

// successResponse 返回成功响应
func (h *BaseHandler) successResponse(c *gin.Context, data interface{}) {
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data":    data,
    })
}

// errorResponse 返回错误响应
func (h *BaseHandler) errorResponse(c *gin.Context, status int, code string, message string) {
    c.JSON(status, gin.H{
        "success": false,
        "error": gin.H{
            "code":    code,
            "message": message,
        },
    })
}
```

### 决策 2: TwoFAHandler 重构

```go
// internal/handlers/twofa_handler.go

package handlers

import (
    "net/http"

    "github.com/gin-gonic/gin"

    "monera-digital/internal/services"
)

// TwoFAHandler 处理 2FA 相关 HTTP 请求
type TwoFAHandler struct {
    base         *BaseHandler
    twofaService *services.TwoFactorService
}

// NewTwoFAHandler 创建 TwoFAHandler
func NewTwoFAHandler(twofa *services.TwoFactorService) *TwoFAHandler {
    return &TwoFAHandler{
        base:         &BaseHandler{},
        twofaService: twofa,
    }
}

// Setup2FA 生成 2FA 密钥、QR 码和备用码
// POST /api/auth/2fa/setup
func (h *TwoFAHandler) Setup2FA(c *gin.Context) {
    userID, ok := h.base.requireUserID(c)
    if !ok {
        return
    }

    email, ok := h.base.getUserEmail(c)
    if !ok {
        h.base.errorResponse(c, http.StatusBadRequest, "INVALID_EMAIL", "User email not found")
        return
    }

    setup, err := h.twofaService.Setup(userID, email)
    if err != nil {
        h.base.errorResponse(c, http.StatusInternalServerError, "SETUP_FAILED", err.Error())
        return
    }

    h.base.successResponse(c, gin.H{
        "secret":       setup.Secret,
        "qrCodeUrl":    setup.QRCode,
        "backupCodes":  setup.BackupCodes,
        "message":      "2FA setup successful",
    })
}

// Enable2FA 启用 2FA (需要验证 TOTP)
// POST /api/auth/2fa/enable
func (h *TwoFAHandler) Enable2FA(c *gin.Context) {
    userID, ok := h.base.requireUserID(c)
    if !ok {
        return
    }

    token, ok := h.base.bindTokenRequest(c)
    if !ok {
        return
    }

    if err := h.twofaService.Enable(userID, token); err != nil {
        h.base.errorResponse(c, http.StatusBadRequest, "ENABLE_FAILED", err.Error())
        return
    }

    h.base.successResponse(c, gin.H{
        "enabled": true,
        "message": "2FA enabled successfully",
    })
}

// Disable2FA 禁用 2FA
// POST /api/auth/2fa/disable
func (h *TwoFAHandler) Disable2FA(c *gin.Context) {
    userID, ok := h.base.requireUserID(c)
    if !ok {
        return
    }

    token, ok := h.base.bindTokenRequest(c)
    if !ok {
        return
    }

    if err := h.twofaService.Disable(userID, token); err != nil {
        h.base.errorResponse(c, http.StatusBadRequest, "DISABLE_FAILED", err.Error())
        return
    }

    h.base.successResponse(c, gin.H{
        "enabled": false,
        "message": "2FA disabled successfully",
    })
}

// Verify2FA 验证 TOTP 或备用码
// POST /api/auth/2fa/verify
func (h *TwoFAHandler) Verify2FA(c *gin.Context) {
    userID, ok := h.base.requireUserID(c)
    if !ok {
        return
    }

    token, ok := h.base.bindTokenRequest(c)
    if !ok {
        return
    }

    valid, err := h.twofaService.Verify(userID, token)
    if err != nil {
        h.base.errorResponse(c, http.StatusBadRequest, "VERIFY_FAILED", err.Error())
        return
    }

    if !valid {
        h.base.errorResponse(c, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid verification code")
        return
    }

    h.base.successResponse(c, gin.H{
        "valid":   true,
        "message": "Token is valid",
    })
}

// Get2FAStatus 获取 2FA 启用状态
// GET /api/auth/2fa/status
func (h *TwoFAHandler) Get2FAStatus(c *gin.Context) {
    userID, ok := h.base.requireUserID(c)
    if !ok {
        return
    }

    enabled, err := h.twofaService.IsEnabled(userID)
    if err != nil {
        h.base.errorResponse(c, http.StatusInternalServerError, "STATUS_FAILED", err.Error())
        return
    }

    h.base.successResponse(c, gin.H{
        "enabled": enabled,
    })
}
```

### 决策 3: Container Options Pattern

```go
// internal/container/container.go

package container

import (
    "database/sql"
    "log"

    "monera-digital/internal/cache"
    "monera-digital/internal/middleware"
    "monera-digital/internal/repository"
    "monera-digital/internal/repository/postgres"
    "monera-digital/internal/services"
)

// ContainerOption 配置选项
type ContainerOption func(*Container)

// WithEncryption 配置加密服务
func WithEncryption(key string) ContainerOption {
    return func(c *Container) {
        encryptionService, err := services.NewEncryptionService(key)
        if err != nil {
            log.Printf("Warning: Failed to initialize encryption service: %v", err)
            return
        }
        c.EncryptionService = encryptionService
        c.TwoFAService = services.NewTwoFactorService(c.DB, encryptionService)
    }
}

// Container 依赖注入容器
type Container struct {
    // 基础设施
    DB *sql.DB

    // 缓存
    TokenBlacklist *cache.TokenBlacklist
    RateLimiter    *middleware.RateLimiter

    // 仓储
    Repository *repository.Repository

    // 服务
    AuthService       *services.AuthService
    LendingService    *services.LendingService
    AddressService    *services.AddressService
    WithdrawalService *services.WithdrawalService
    DepositService    *services.DepositService
    WalletService     *services.WalletService
    EncryptionService *services.EncryptionService
    TwoFAService      *services.TwoFactorService

    // 中间件
    RateLimitMiddleware *middleware.PerEndpointRateLimiter
}

// NewContainer 创建容器
func NewContainer(db *sql.DB, jwtSecret string, opts ...ContainerOption) *Container {
    c := &Container{DB: db}

    // 初始化缓存
    c.TokenBlacklist = cache.NewTokenBlacklist()
    c.RateLimiter = middleware.NewRateLimiter(5, 60)

    // 初始化仓储
    c.Repository = &repository.Repository{
        User:       postgres.NewUserRepository(db),
        Deposit:    postgres.NewDepositRepository(db),
        Wallet:     postgres.NewWalletRepository(db),
        Account:    postgres.NewAccountRepository(db),
        Address:    postgres.NewAddressRepository(db),
        Withdrawal: postgres.NewWithdrawalRepository(db),
    }

    // 初始化服务
    c.AuthService = services.NewAuthService(db, jwtSecret)
    c.AuthService.SetTokenBlacklist(c.TokenBlacklist)

    c.LendingService = services.NewLendingService(db)
    c.AddressService = services.NewAddressService(c.Repository.Address)
    c.WithdrawalService = services.NewWithdrawalService(db, c.Repository, services.NewSafeheronService())
    c.DepositService = services.NewDepositService(c.Repository.Deposit)
    c.WalletService = services.NewWalletService(c.Repository.Wallet)

    // 应用选项
    for _, opt := range opts {
        opt(c)
    }

    // 初始化中间件
    c.RateLimitMiddleware = middleware.NewPerEndpointRateLimiter()
    c.RateLimitMiddleware.AddEndpoint("/api/auth/register", 5, 60)
    c.RateLimitMiddleware.AddEndpoint("/api/auth/login", 5, 60)
    c.RateLimitMiddleware.AddEndpoint("/api/auth/refresh", 10, 60)

    return c
}

// Close 关闭资源
func (c *Container) Close() error {
    if c.TokenBlacklist != nil {
        c.TokenBlacklist.Close()
    }
    if c.DB != nil {
        return c.DB.Close()
    }
    return nil
}

// Verify 验证依赖
func (c *Container) Verify() error {
    if err := c.DB.Ping(); err != nil {
        return err
    }
    // 验证必要服务
    services := []struct {
        name  string
        value interface{}
    }{
        {"AuthService", c.AuthService},
        {"LendingService", c.LendingService},
        {"AddressService", c.AddressService},
        {"WithdrawalService", c.WithdrawalService},
        {"DepositService", c.DepositService},
        {"WalletService", c.WalletService},
    }
    for _, s := range services {
        if s.value == nil {
            log.Printf("%s not initialized", s.name)
        }
    }
    return nil
}
```

---

## 实现计划

### Phase 1: BaseHandler 提取 (1天)

- [ ] 创建 `internal/handlers/base.go`
- [ ] 迁移通用方法
- [ ] 更新所有 Handler 使用 BaseHandler

### Phase 2: TwoFAHandler 重构 (1天)

- [ ] 重构 `twofa_handler.go`
- [ ] 使用 BaseHandler 减少重复
- [ ] 统一错误响应格式

### Phase 3: Container Options Pattern (0.5天)

- [ ] 重构 `container.go`
- [ ] 使用 Options Pattern
- [ ] 更新 `main.go`

### Phase 4: 测试覆盖 (2天)

- [ ] 为 BaseHandler 添加测试
- [ ] 为 TwoFAHandler 添加测试
- [ ] 验证测试覆盖率 100%

---

## 测试策略

### 测试用例

```go
// internal/handlers/twofa_handler_test.go

package handlers

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    "github.com/gin-gonic/gin"
    "monera-digital/internal/services"
)

// MockTwoFactorService 用于测试
type MockTwoFactorService struct {
    SetupFunc    func(userID int, email string) (*services.SetupResponse, error)
    EnableFunc   func(userID int, token string) error
    DisableFunc  func(userID int, token string) error
    VerifyFunc   func(userID int, token string) (bool, error)
    IsEnabledFunc func(userID int) (bool, error)
}

func TestTwoFAHandler_Setup2FA_Success(t *testing.T) {
    gin.SetMode(gin.TestMode)

    // Arrange
    mockService := &MockTwoFactorService{
        SetupFunc: func(userID int, email string) (*services.SetupResponse, error) {
            return &services.SetupResponse{
                Secret:      "JBSWY3DPEHPK3PXP",
                QRCode:      "otpauth://totp/Monera Digital:test@example.com?secret=JBSWY3DPEHPK3PXP",
                BackupCodes: []string{"abc123", "def456"},
            }, nil
        },
    }

    handler := &TwoFAHandler{
        base:         &BaseHandler{},
        twofaService: mockService,
    }

    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    c.Set("userID", 1)
    c.Set("email", "test@example.com")

    // Act
    handler.Setup2FA(c)

    // Assert
    if w.Code != http.StatusOK {
        t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
    }

    var response map[string]interface{}
    json.Unmarshal(w.Body.Bytes(), &response)

    if response["success"] != true {
        t.Error("Expected success to be true")
    }
}

func TestTwoFAHandler_Enable2FA_InvalidTokenFormat(t *testing.T) {
    gin.SetMode(gin.TestMode)

    handler := &TwoFAHandler{
        base:         &BaseHandler{},
        twofaService: &MockTwoFactorService{},
    }

    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    c.Set("userID", 1)
    c.Set("email", "test@example.com")
    c.Request = httptest.NewRequest("POST", "/api/auth/2fa/enable",
        strings.NewReader(`{"token": "123"}`))
    c.Request.Header.Set("Content-Type", "application/json")

    // Act
    handler.Enable2FA(c)

    // Assert
    if w.Code != http.StatusBadRequest {
        t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
    }
}

func TestTwoFAHandler_Get2FAStatus_Unauthorized(t *testing.T) {
    gin.SetMode(gin.TestMode)

    handler := &TwoFAHandler{
        base:         &BaseHandler{},
        twofaService: &MockTwoFactorService{},
    }

    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    // 不设置 userID

    // Act
    handler.Get2FAStatus(c)

    // Assert
    if w.Code != http.StatusUnauthorized {
        t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
    }
}
```

### 测试覆盖目标

| 文件 | 方法数 | 目标测试数 | 覆盖类型 |
|------|--------|------------|----------|
| `base.go` | 6 | 18 | 单元测试 |
| `twofa_handler.go` | 5 | 25 | 单元测试 + 集成测试 |

---

## 验收标准

- [ ] 所有新建文件行数 < 200 行
- [ ] 重复代码率 < 5%
- [ ] 测试覆盖率 100%
- [ ] 无新的 linter 警告
- [ ] 所有现有功能不受影响

---

## 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 重构引入 bug | 高 | 逐步重构，每步验证 |
| 测试编写耗时 | 中 | 使用 table-driven 测试 |
| 性能下降 | 低 | 基准测试验证 |

---

## 相关文档

- [2FA Handler 架构审计](audit/2fa-handler-audit.md)
- [2FA Service 设计文档](../openspec/auth-migration-and-2fa-implementation.md)

---

*文档版本: v1.0*
*创建日期: 2026-01-23*
