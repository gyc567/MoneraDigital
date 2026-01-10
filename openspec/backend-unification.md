# OpenSpec: 后端统一到 Go 架构方案

## 1. 目标

将 Monera Digital 后端统一到 Go + Gin 架构，消除双后端并存：
- 单一后端架构（Go）
- KISS 设计原则
- 高内聚、低耦合
- 100% 测试覆盖率
- 零回归风险

## 2. 问题分析

### 2.1 现状：双后端并存

```
前端 → Go后端(8081) + Vercel Serverless(api/)
         ↓
    PostgreSQL (Neon)
```

### 2.2 核心问题

| 问题 | 影响 |
|------|------|
| 功能重复 | 两套实现，维护成本 x2 |
| 逻辑不一致 | 业务行为可能不同 |
| 配置分散 | 环境变量管理混乱 |
| 部署复杂 | 两个部署目标 |

## 3. 架构设计

### 3.1 分层原则

```
┌─────────────────────────────────────┐
│  HTTP层 (handlers)                  │ 高内聚、低耦合
├─────────────────────────────────────┤
│  服务层 (services)                  │ 独立领域
├─────────────────────────────────────┤
│  数据层 (db)                        │ 纯数据库操作
├─────────────────────────────────────┤
│  模型层 (models)                    │ 纯数据定义
└─────────────────────────────────────┘
```

### 3.2 目录结构

```
cmd/server/
  main.go              # 入口
internal/
  config/
    config.go          # 配置加载
  db/
    db.go              # 数据库连接池
  errors/
    errors.go          # 统一错误定义
  handlers/
    auth.go            # 认证处理器
    lending.go         # 借贷处理器
    address.go         # 地址处理器
    withdrawal.go      # 提现处理器
    response.go        # 统一响应
  middleware/
    cors.go            # CORS中间件
    auth.go            # JWT认证
  models/
    models.go          # 数据模型
  services/
    auth.go            # 认证服务
    lending.go         # 借贷服务
    address.go         # 地址服务
    withdrawal.go      # 提现服务
```

## 4. 实施计划

### 阶段一：基础设施加固 (Day 1)

#### 4.1.1 数据库连接池

```go
// internal/db/db.go
type Config struct {
    MaxOpenConns    int
    MaxIdleConns    int
    ConnMaxLifetime time.Duration
}

func InitDB(databaseURL string, cfg Config) (*sql.DB, error) {
    db, err := sql.Open("postgres", databaseURL)
    db.SetMaxOpenConns(cfg.MaxOpenConns)      // 默认 25
    db.SetMaxIdleConns(cfg.MaxIdleConns)      // 默认 5
    db.SetConnMaxLifetime(cfg.ConnMaxLifetime) // 默认 5分钟
    return db, nil
}
```

#### 4.1.2 CORS 安全配置

```go
// internal/middleware/cors.go
func NewCORS(allowedOrigins []string) gin.HandlerFunc {
    origins := make(map[string]bool)
    for _, o := range allowedOrigins {
        origins[o] = true
    }
    return func(c *gin.Context) {
        origin := c.GetHeader("Origin")
        if origins[origin] {
            c.Header("Access-Control-Allow-Origin", origin)
            c.Header("Access-Control-Allow-Credentials", "true")
        }
        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
        }
    }
}
```

### 阶段二：安全加固 (Day 2-3)

#### 4.2.1 JWT Secret 管理

```go
// internal/config/config.go
func Load() (*Config, error) {
    jwtSecret := viper.GetString("JWT_SECRET")
    if viper.GetString("ENV") == "production" {
        if len(jwtSecret) < 32 {
            return nil, fmt.Errorf("JWT_SECRET must be at least 32 chars")
        }
    }
    return &Config{JWTSecret: jwtSecret}, nil
}
```

#### 4.2.2 统一错误处理

```go
// internal/errors/errors.go
type AppError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
}

var (
    ErrInvalidCredentials = &AppError{Code: 401, Message: "Invalid email or password"}
    ErrUnauthorized       = &AppError{Code: 401, Message: "Unauthorized"}
    ErrNotFound           = &AppError{Code: 404, Message: "Resource not found"}
    ErrInvalidInput       = &AppError{Code: 400, Message: "Invalid input data"}
)
```

### 阶段三：核心服务实现 (Day 4-7)

#### 4.3.1 Auth Service

```go
// internal/services/auth.go
type AuthService struct {
    db        *sql.DB
    jwtSecret string
}

func (s *AuthService) Register(email, password string) (*User, error) {
    // 1. 验证输入
    if len(password) < 8 {
        return nil, errors.New("password must be at least 8 characters")
    }
    // 2. 检查邮箱是否存在
    // 3. bcrypt 密码哈希
    // 4. 插入数据库
}

func (s *AuthService) Login(email, password string) (*LoginResponse, error) {
    // 1. 查询用户
    // 2. bcrypt 验证密码
    // 3. 检查 2FA
    // 4. 生成 JWT
}
```

#### 4.3.2 Lending Service

```go
// internal/services/lending.go
func (s *LendingService) CalculateAPY(asset string, durationDays int) decimal.Decimal {
    baseRates := map[string]decimal.Decimal{
        "BTC":  decimal.NewFromFloat(4.5),
        "ETH":  decimal.NewFromFloat(5.2),
        "USDT": decimal.NewFromFloat(8.5),
    }
    multiplier := 1.0
    if durationDays >= 360 { multiplier = 1.5 }
    return baseRates[asset].Mul(decimal.NewFromFloat(multiplier)).Round(2)
}

func (s *LendingService) ApplyForLending(userID int, asset string, amount decimal.Decimal, durationDays int) (*LendingPosition, error) {
    if durationDays < 1 || durationDays > 365 {
        return nil, errors.New("duration must be between 1 and 365 days")
    }
    apy := s.CalculateAPY(asset, durationDays)
    // 插入数据库
}
```

#### 4.3.3 Address Service

```go
// internal/services/address.go
func (s *AddressService) AddAddress(userID int, address, addressType, label string) (*WithdrawalAddress, error) {
    if err := validateAddress(addressType, address); err != nil {
        return nil, err
    }
    // 插入数据库
    // 生成验证 token
    return &addr, nil
}

func validateAddress(addressType, address string) error {
    switch addressType {
    case "ETH":
        if len(address) != 42 || address[:2] != "0x" {
            return errors.New("invalid ETH address format")
        }
    }
    return nil
}
```

#### 4.3.4 Withdrawal Service

```go
// internal/services/withdrawal.go
func (s *WithdrawalService) CreateWithdrawal(userID int, addressID int, amount decimal.Decimal, asset string) (*Withdrawal, error) {
    // 验证地址所有权和验证状态
    var addr WithdrawalAddress
    err := s.db.QueryRow("SELECT user_id, address, is_verified FROM withdrawal_addresses WHERE id = $1", addressID).Scan(&addr.UserID, &addr.Address, &addr.IsVerified)
    if addr.UserID != userID {
        return nil, errors.New("address does not belong to user")
    }
    if !addr.IsVerified {
        return nil, errors.New("address must be verified before withdrawal")
    }
    // 事务创建提现
}
```

### 阶段四：Handlers 层实现 (Day 8-10)

```go
// internal/handlers/auth.go
type AuthHandler struct {
    authService *services.AuthService
}

func (h *AuthHandler) Login(c *gin.Context) {
    var req LoginRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        Error(c, errors.ErrInvalidInput)
        return
    }
    resp, err := h.authService.Login(req.Email, req.Password)
    if err != nil {
        Error(c, &errors.AppError{Code: 401, Message: err.Error()})
        return
    }
    Success(c, resp)
}
```

### 阶段五：删除 Vercel API (Day 11)

```bash
rm -rf api/
```

### 阶段六：验证和测试 (Day 12)

## 5. 测试策略

### 5.1 测试覆盖率要求

| 模块 | 覆盖率 |
|------|--------|
| services | 100% |
| handlers | 100% |
| middleware | 100% |
| errors | 100% |
| **整体** | **100%** |

### 5.2 测试示例

```go
// internal/services/auth_test.go
func TestAuthService_Register_Success(t *testing.T) {}
func TestAuthService_Register_InvalidEmail(t *testing.T) {}
func TestAuthService_Register_ShortPassword(t *testing.T) {}
func TestAuthService_Register_EmailExists(t *testing.T) {}
func TestAuthService_Login_Success(t *testing.T) {}
func TestAuthService_Login_InvalidCredentials(t *testing.T) {}
```

## 6. 代码统计

| 模块 | 代码行数 | 测试用例数 |
|------|---------|-----------|
| db | 50 | 3 |
| errors | 40 | 2 |
| middleware | 80 | 3 |
| services | 400 | 20 |
| handlers | 200 | 10 |
| **总计** | **770** | **38** |

## 7. 实施路线图

| 阶段 | 内容 | 时间 |
|------|------|------|
| P0 | 基础设施加固 | Day 1 |
| P1 | 安全加固 | Day 2-3 |
| P2 | 核心服务实现 | Day 4-7 |
| P3 | Handlers实现 | Day 8-10 |
| P4 | 删除Vercel API | Day 11 |
| P5 | 验证和测试 | Day 12 |

## 8. 风险控制

1. **零回归风险**：每次提交前运行完整测试套件
2. **回滚方案**：保留 Vercel API 备份 1 周
3. **灰度发布**：逐步切换流量

## 9. 预期收益

- 降低 50% 维护成本
- 提升代码一致性
- 增强安全性
- 简化部署流程
