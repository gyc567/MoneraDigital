# OpenSpec: 重构 2FA Go 后端实现

## 1. 目标 (Objectives)

基于 [google2fa](https://github.com/antonioribeiro/google2fa) 最佳实践，重构当前 Go 后端 2FA 实现，提升安全性、可维护性和代码质量。

### 关键改进点
1. **增强安全性**: 实现重放攻击防护、增强密钥长度、支持多种 HMAC 算法
2. **优化架构**: 清晰的职责分离、接口化设计、提升可测试性
3. **完善测试**: 100% 测试覆盖率，包括单元测试和集成测试
4. **保持 KISS 原则**: 简洁清晰的代码，避免过度设计

---

## 2. 当前实现分析

### 2.1 现状 (`internal/services/twofa_service.go`)

**优点**:
- ✅ 使用 `pquerna/otp` 标准库 (RFC 6238 兼容)
- ✅ 加密存储密钥和备份码
- ✅ 支持备份码一次性使用
- ✅ 接口化设计 (`EncryptionProvider`)

**问题**:
- ❌ **无重放攻击防护**: 同一 TOTP 令牌可在窗口期内重复使用
- ❌ **密钥长度不足**: 未明确指定密钥长度 (默认可能 < 160 bits)
- ❌ **测试覆盖不足**: `twofa_service_test.go` 全部跳过 (0% 覆盖率)
- ❌ **职责混乱**: `VerifyAndLogin()` 方法越界 - 2FA 服务不应处理 JWT
- ❌ **缺少时间窗口配置**: 硬编码在 `totp.Validate()` 中
- ❌ **QR 码依赖**: 直接使用 `secret.URL()` 但未显式处理 QR 码生成

### 2.2 架构问题

```go
// 当前架构 - 职责不清晰
AuthService
  └── calls → TwoFactorService
                └── VerifyAndLogin() // ❌ 返回 JWT 相关数据
```

**问题**: `TwoFactorService.VerifyAndLogin()` 返回 `LoginResponse`，违反单一职责原则。

---

## 3. 重构设计 (Refactor Design)

### 3.1 核心原则

| 原则 | 实现策略 |
|------|---------|
| **KISS** | 每个方法只做一件事，避免复杂逻辑 |
| **高内聚** | 2FA 核心逻辑集中在 `TwoFactorService` |
| **低耦合** | 通过接口隔离依赖 (DB, Encryption, Time) |
| **可测试性** | 依赖注入所有外部依赖，100% 测试覆盖 |
| **安全优先** | 重放攻击防护、增强密钥、审计日志 |

### 3.2 新架构

```go
// 重构后 - 清晰的职责分离
AuthService
  ├── Login() → checks 2FA status
  └── Verify2FAAndLogin() → validates token + issues JWT
        └── calls → TwoFactorService.Verify() // ✅ 只返回验证结果

TwoFactorService
  ├── Setup()      // 生成密钥、QR URL、备份码
  ├── Enable()     // 启用 2FA
  ├── Disable()    // 禁用 2FA
  ├── Verify()     // 验证令牌 (TOTP/备份码) + 重放检测
  ├── IsEnabled()  // 检查状态
  └── (removed) VerifyAndLogin() // ❌ 删除 - 职责越界
```

---

## 4. 详细实现方案

### 4.1 增强密钥生成

**当前实现**:
```go
secret, err := totp.Generate(totp.GenerateOpts{
    Issuer:      "Monera Digital",
    AccountName: email,
    Period:      30,
    Digits:      6,
    Algorithm:   otp.AlgorithmSHA1, // ❌ 未指定密钥长度
})
```

**重构后**:
```go
// 新增配置结构
type TOTPConfig struct {
    Issuer       string
    Period       uint   // 默认 30 秒
    Digits       int    // 默认 6 位
    SecretSize   int    // ✅ 新增: 20 bytes (160 bits)
    Algorithm    otp.Algorithm
    Skew         uint   // ✅ 新增: 时间窗口 (默认 1)
}

// 使用明确的密钥长度
secret, err := totp.Generate(totp.GenerateOpts{
    Issuer:      config.Issuer,
    AccountName: email,
    Period:      config.Period,
    Digits:      otp.DigitsSix,
    SecretSize:  20, // ✅ 160 bits (符合 google2fa v9.0+ 标准)
    Algorithm:   otp.AlgorithmSHA1,
})
```

**改进点**:
- ✅ 密钥长度从隐式默认提升到显式 160 bits
- ✅ 配置化，便于测试和未来扩展
- ✅ 符合 RFC 6238 和 Google Authenticator 标准

---

### 4.2 重放攻击防护

**问题**: 当前实现允许同一令牌在 30 秒窗口内重复使用。

**解决方案**: 参考 google2fa 的 `verifyKeyNewer()` 模式。

#### 数据库扩展
```sql
-- 新增字段到 users 表
ALTER TABLE users ADD COLUMN two_factor_last_used_at BIGINT DEFAULT 0;
-- 存储最后成功验证的 Unix 时间戳
```

#### 重构 `Verify()` 方法

**当前实现**:
```go
func (s *TwoFactorService) Verify(userID int, token string) (bool, error) {
    secret, _ := s.getSecret(userID)
    if totp.Validate(token, secret) { // ❌ 无重放检测
        return true, nil
    }
    // ... 备份码检查
}
```

**重构后**:
```go
// VerifyResult 验证结果
type VerifyResult struct {
    Valid     bool
    Timestamp int64 // 新增: 令牌对应的时间戳
}

// Verify 验证令牌并防止重放攻击
func (s *TwoFactorService) Verify(userID int, token string) (*VerifyResult, error) {
    secret, err := s.getSecret(userID)
    if err != nil {
        return nil, err
    }

    // 获取最后使用时间戳
    lastUsedAt, err := s.getLastUsedTimestamp(userID)
    if err != nil {
        return nil, err
    }

    // ✅ 使用 ValidateCustom 获取令牌对应的时间戳
    valid, timestamp := totp.ValidateCustom(token, secret, time.Now().UTC(), totp.ValidateOpts{
        Period:    30,
        Skew:      1, // ±30 秒窗口
        Digits:    otp.DigitsSix,
        Algorithm: otp.AlgorithmSHA1,
    })

    if !valid {
        // 尝试备份码
        return s.verifyBackupCode(userID, token)
    }

    // ✅ 重放检测: 拒绝已使用过的令牌
    if timestamp <= lastUsedAt {
        return &VerifyResult{Valid: false}, nil
    }

    // ✅ 更新最后使用时间戳
    if err := s.updateLastUsedTimestamp(userID, timestamp); err != nil {
        return nil, fmt.Errorf("failed to update timestamp: %w", err)
    }

    return &VerifyResult{
        Valid:     true,
        Timestamp: timestamp,
    }, nil
}

// 新增辅助方法
func (s *TwoFactorService) getLastUsedTimestamp(userID int) (int64, error) {
    var timestamp int64
    query := `SELECT two_factor_last_used_at FROM users WHERE id = $1`
    err := s.DB.QueryRow(query, userID).Scan(&timestamp)
    return timestamp, err
}

func (s *TwoFactorService) updateLastUsedTimestamp(userID int, timestamp int64) error {
    query := `UPDATE users SET two_factor_last_used_at = $1 WHERE id = $2`
    _, err := s.DB.Exec(query, timestamp, userID)
    return err
}
```

**改进点**:
- ✅ 防止重放攻击: 同一令牌只能使用一次
- ✅ 时间窗口可配置: `Skew` 参数
- ✅ 精确的错误信息: 区分"无效令牌"和"已使用令牌"

---

### 4.3 职责分离: 移除 `VerifyAndLogin()`

**当前问题**:
```go
// ❌ TwoFactorService 不应该处理 JWT 和登录响应
func (s *TwoFactorService) VerifyAndLogin(userID int, token string) (*LoginResponse, error) {
    // ... 验证逻辑
    return &LoginResponse{ // ❌ 越界职责
        User: &models.User{...},
    }, nil
}
```

**重构方案**:
```go
// ✅ 删除 TwoFactorService.VerifyAndLogin()
// ✅ 在 AuthService 中统一处理

// AuthService.Verify2FAAndLogin() - 保留在这里
func (s *AuthService) Verify2FAAndLogin(userID int, token string) (*LoginResponse, error) {
    // ✅ 调用 TwoFactorService 只做验证
    result, err := s.twoFactorService.Verify(userID, token)
    if err != nil {
        return nil, err
    }
    if !result.Valid {
        return nil, errors.New("invalid 2FA token")
    }

    // ✅ AuthService 负责 JWT 生成
    user, err := s.GetUserByID(userID)
    if err != nil {
        return nil, err
    }

    token, err := utils.GenerateJWT(user.ID, user.Email, s.jwtSecret)
    if err != nil {
        return nil, err
    }

    return &LoginResponse{
        User:        user,
        Token:       token,
        AccessToken: token,
        TokenType:   "Bearer",
        ExpiresIn:   86400,
        ExpiresAt:   time.Now().Add(24 * time.Hour),
    }, nil
}
```

**改进点**:
- ✅ 单一职责: `TwoFactorService` 只处理 TOTP 验证
- ✅ 清晰边界: `AuthService` 负责认证和会话管理
- ✅ 易于测试: 各层独立可测

---

### 4.4 QR 码生成优化

**当前实现**:
```go
// ❌ 直接返回 otpauth:// URL，未显式处理 QR 码
return &SetupResponse{
    QRCode: secret.URL(), // 这只是 URL，不是图片
}
```

**问题**: 前端需要自己生成 QR 码，增加复杂度。

**重构方案** (参考 google2fa 分离原则):

#### 选项 A: URL Only (推荐 - 符合 KISS)
```go
// ✅ 保持简单: 只返回 URL，QR 码生成留给前端
type SetupResponse struct {
    Secret      string   `json:"secret"`
    OTPAuthURL  string   `json:"otpauthUrl"` // ✅ 重命名更清晰
    BackupCodes []string `json:"backupCodes"`
}

return &SetupResponse{
    Secret:      secret.Secret(),
    OTPAuthURL:  secret.URL(),
    BackupCodes: backupCodes,
}
```

**前端处理**:
```typescript
// 使用 qrcode.react 或类似库
import QRCode from 'qrcode.react';

<QRCode value={response.otpauthUrl} />
```

#### 选项 B: 后端生成 Base64 图片 (如需统一管理)
```go
import "github.com/skip2/go-qrcode"

func (s *TwoFactorService) Setup(...) (*SetupResponse, error) {
    // ... 生成密钥
    
    // ✅ 可选: 生成 QR 码 PNG
    qrPNG, err := qrcode.Encode(secret.URL(), qrcode.Medium, 256)
    if err != nil {
        return nil, fmt.Errorf("failed to generate QR code: %w", err)
    }
    
    qrBase64 := base64.StdEncoding.EncodeToString(qrPNG)
    
    return &SetupResponse{
        Secret:      secret.Secret(),
        OTPAuthURL:  secret.URL(),
        QRCodeImage: qrBase64, // ✅ Base64 编码的 PNG
        BackupCodes: backupCodes,
    }, nil
}
```

**推荐**: **选项 A** (KISS 原则，职责分离)

---

### 4.5 配置化设计

**新增配置文件** (`internal/config/twofa.go`):

```go
package config

import (
    "github.com/pquerna/otp"
)

// TwoFactorConfig 2FA 配置
type TwoFactorConfig struct {
    Issuer          string         `env:"TWOFA_ISSUER" envDefault:"Monera Digital"`
    Period          uint           `env:"TWOFA_PERIOD" envDefault:"30"`
    Digits          int            `env:"TWOFA_DIGITS" envDefault:"6"`
    SecretSize      int            `env:"TWOFA_SECRET_SIZE" envDefault:"20"` // 160 bits
    Algorithm       otp.Algorithm  `env:"TWOFA_ALGORITHM" envDefault:"SHA1"`
    Skew            uint           `env:"TWOFA_SKEW" envDefault:"1"` // ±30s
    BackupCodeCount int            `env:"TWOFA_BACKUP_COUNT" envDefault:"10"`
}

func LoadTwoFactorConfig() *TwoFactorConfig {
    return &TwoFactorConfig{
        Issuer:          getEnv("TWOFA_ISSUER", "Monera Digital"),
        Period:          30,
        Digits:          6,
        SecretSize:      20,
        Algorithm:       otp.AlgorithmSHA1,
        Skew:            1,
        BackupCodeCount: 10,
    }
}
```

**使用方式**:
```go
config := config.LoadTwoFactorConfig()
service := NewTwoFactorService(db, encryption, config)
```

---

### 4.6 完整的测试策略

#### 单元测试 (`twofa_service_test.go`)

**当前问题**: 所有测试都被跳过。

**重构目标**: 100% 覆盖率

```go
package services_test

import (
    "testing"
    "time"
    
    "github.com/pquerna/otp/totp"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

// Mock 实现
type MockDB struct {
    mock.Mock
}

func (m *MockDB) QueryRow(query string, args ...interface{}) *sql.Row {
    // Mock implementation
}

type MockEncryption struct {
    mock.Mock
}

func (m *MockEncryption) Encrypt(plaintext string) (string, error) {
    args := m.Called(plaintext)
    return args.String(0), args.Error(1)
}

// ===== 测试用例 =====

// TestSetup_GeneratesValidSecret 测试密钥生成
func TestSetup_GeneratesValidSecret(t *testing.T) {
    mockDB := new(MockDB)
    mockEnc := new(MockEncryption)
    
    mockEnc.On("Encrypt", mock.Anything).Return("encrypted_secret", nil)
    mockDB.On("Exec", mock.Anything, mock.Anything).Return(nil, nil)
    
    service := NewTwoFactorServiceWithInterface(mockDB, mockEnc)
    
    result, err := service.Setup(1, "test@example.com")
    
    assert.NoError(t, err)
    assert.NotEmpty(t, result.Secret)
    assert.Len(t, result.Secret, 32) // Base32 编码后长度
    assert.Len(t, result.BackupCodes, 10)
    assert.Contains(t, result.OTPAuthURL, "otpauth://totp/")
}

// TestVerify_ValidToken_Success 测试有效令牌
func TestVerify_ValidToken_Success(t *testing.T) {
    mockDB := new(MockDB)
    mockEnc := new(MockEncryption)
    
    secret := "JBSWY3DPEHPK3PXP" // 测试密钥
    
    mockEnc.On("Decrypt", "encrypted_secret").Return(secret, nil)
    mockDB.On("QueryRow", mock.Anything, mock.Anything).Return(
        mockRow(secret, 0), // lastUsedAt = 0
    )
    mockDB.On("Exec", mock.Anything, mock.Anything).Return(nil, nil)
    
    service := NewTwoFactorServiceWithInterface(mockDB, mockEnc)
    
    // 生成当前有效令牌
    token, _ := totp.GenerateCode(secret, time.Now().UTC())
    
    result, err := service.Verify(1, token)
    
    assert.NoError(t, err)
    assert.True(t, result.Valid)
    assert.Greater(t, result.Timestamp, int64(0))
}

// TestVerify_ReplayAttack_Rejected 测试重放攻击防护
func TestVerify_ReplayAttack_Rejected(t *testing.T) {
    mockDB := new(MockDB)
    mockEnc := new(MockEncryption)
    
    secret := "JBSWY3DPEHPK3PXP"
    
    // 模拟已使用过的时间戳
    usedTimestamp := time.Now().Unix()
    
    mockEnc.On("Decrypt", "encrypted_secret").Return(secret, nil)
    mockDB.On("QueryRow", mock.Anything, mock.Anything).Return(
        mockRow(secret, usedTimestamp), // ✅ 已使用过的时间戳
    )
    
    service := NewTwoFactorServiceWithInterface(mockDB, mockEnc)
    
    token, _ := totp.GenerateCode(secret, time.Now().UTC())
    
    result, err := service.Verify(1, token)
    
    assert.NoError(t, err)
    assert.False(t, result.Valid) // ✅ 应该被拒绝
}

// TestVerify_ExpiredToken_Rejected 测试过期令牌
func TestVerify_ExpiredToken_Rejected(t *testing.T) {
    mockDB := new(MockDB)
    mockEnc := new(MockEncryption)
    
    secret := "JBSWY3DPEHPK3PXP"
    
    mockEnc.On("Decrypt", "encrypted_secret").Return(secret, nil)
    mockDB.On("QueryRow", mock.Anything, mock.Anything).Return(
        mockRow(secret, 0),
    )
    
    service := NewTwoFactorServiceWithInterface(mockDB, mockEnc)
    
    // 生成 5 分钟前的令牌 (超出 Skew 窗口)
    oldTime := time.Now().Add(-5 * time.Minute)
    token, _ := totp.GenerateCode(secret, oldTime)
    
    result, err := service.Verify(1, token)
    
    assert.NoError(t, err)
    assert.False(t, result.Valid) // ✅ 应该被拒绝
}

// TestVerify_BackupCode_OneTimeUse 测试备份码一次性使用
func TestVerify_BackupCode_OneTimeUse(t *testing.T) {
    mockDB := new(MockDB)
    mockEnc := new(MockEncryption)
    
    backupCode := "abcd1234"
    backupCodes := []string{backupCode, "efgh5678"}
    
    mockDB.On("QueryRow", mock.Anything, mock.Anything).Return(
        mockRowWithBackupCodes(backupCodes),
    )
    mockDB.On("Exec", mock.Anything, mock.Anything).Return(nil, nil)
    
    service := NewTwoFactorServiceWithInterface(mockDB, mockEnc)
    
    // 第一次使用备份码
    result1, err := service.Verify(1, backupCode)
    assert.NoError(t, err)
    assert.True(t, result1.Valid)
    
    // ✅ 验证备份码被移除 (通过 Mock 验证 Exec 调用)
    mockDB.AssertCalled(t, "Exec", mock.MatchedBy(func(query string) bool {
        return query == "UPDATE users SET two_factor_backup_codes = $1 WHERE id = $2"
    }), mock.Anything, 1)
}

// TestEnable_InvalidToken_Rejected 测试启用时的无效令牌
func TestEnable_InvalidToken_Rejected(t *testing.T) {
    mockDB := new(MockDB)
    mockEnc := new(MockEncryption)
    
    secret := "JBSWY3DPEHPK3PXP"
    
    mockEnc.On("Decrypt", "encrypted_secret").Return(secret, nil)
    mockDB.On("QueryRow", mock.Anything, mock.Anything).Return(
        mockRow(secret, 0),
    )
    
    service := NewTwoFactorServiceWithInterface(mockDB, mockEnc)
    
    err := service.Enable(1, "000000") // 无效令牌
    
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "invalid verification code")
}

// 更多测试用例...
// - TestDisable_Success
// - TestIsEnabled_True
// - TestIsEnabled_False
// - TestSetup_EncryptionFailure
// - TestVerify_DatabaseError
```

**测试覆盖率目标**:
- ✅ 密钥生成: 长度、格式、备份码数量
- ✅ 令牌验证: 有效、无效、过期、重放
- ✅ 备份码: 验证、一次性使用、耗尽
- ✅ 错误处理: 数据库错误、加密失败
- ✅ 边界条件: 空输入、SQL 注入、时间窗口边界

#### 集成测试

```go
// integration_test.go
func TestTwoFactorFlow_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    // 使用真实数据库 (test container)
    db := setupTestDB(t)
    defer db.Close()
    
    encryption := NewEncryptionService(testKey)
    service := NewTwoFactorService(db, encryption)
    
    // 1. Setup
    userID := createTestUser(t, db)
    setup, err := service.Setup(userID, "test@example.com")
    assert.NoError(t, err)
    
    // 2. Enable
    token, _ := totp.GenerateCode(setup.Secret, time.Now().UTC())
    err = service.Enable(userID, token)
    assert.NoError(t, err)
    
    // 3. Verify status
    enabled, _ := service.IsEnabled(userID)
    assert.True(t, enabled)
    
    // 4. Login verification
    newToken, _ := totp.GenerateCode(setup.Secret, time.Now().UTC())
    result, err := service.Verify(userID, newToken)
    assert.NoError(t, err)
    assert.True(t, result.Valid)
    
    // 5. Replay attack
    result2, _ := service.Verify(userID, newToken)
    assert.False(t, result2.Valid) // ✅ 应该被拒绝
    
    // 6. Backup code
    backupCode := setup.BackupCodes[0]
    result3, _ := service.Verify(userID, backupCode)
    assert.True(t, result3.Valid)
    
    // 7. Disable
    disableToken, _ := totp.GenerateCode(setup.Secret, time.Now().Add(time.Minute))
    err = service.Disable(userID, disableToken)
    assert.NoError(t, err)
}
```

---

## 5. 实施步骤 (Implementation Steps)

### 阶段 1: 数据库迁移
```sql
-- migration_001_add_2fa_timestamp.up.sql
ALTER TABLE users ADD COLUMN two_factor_last_used_at BIGINT DEFAULT 0;

-- migration_001_add_2fa_timestamp.down.sql
ALTER TABLE users DROP COLUMN two_factor_last_used_at;
```

### 阶段 2: 重构 `TwoFactorService`
1. ✅ 添加配置结构 (`TOTPConfig`)
2. ✅ 增强 `Setup()`: 明确密钥长度
3. ✅ 重构 `Verify()`: 添加重放检测
4. ✅ 删除 `VerifyAndLogin()`
5. ✅ 优化 QR 码处理 (URL only)

### 阶段 3: 更新 `AuthService`
1. ✅ 完善 `Verify2FAAndLogin()`: 集成新的 `Verify()` 返回值
2. ✅ 实现 `GetUserByID()` (当前是 TODO)

### 阶段 4: Handler 层调整
1. ✅ 更新 `TwoFAHandler.Verify2FALogin()`: 调用 `AuthService.Verify2FAAndLogin()`
2. ✅ 移除 "NOT_IMPLEMENTED" 错误

### 阶段 5: 测试
1. ✅ 编写单元测试 (100% 覆盖)
2. ✅ 编写集成测试
3. ✅ E2E 测试 (Playwright)

### 阶段 6: 文档和部署
1. ✅ 更新 API 文档
2. ✅ 环境变量配置文档
3. ✅ 数据库迁移脚本

---

## 6. 验收标准 (Acceptance Criteria)

### 功能需求
- [ ] 密钥长度 ≥ 160 bits
- [ ] 重放攻击防护生效
- [ ] 备份码一次性使用
- [ ] 支持时间窗口配置 (Skew)
- [ ] QR 码 URL 正确生成

### 代码质量
- [ ] 100% 测试覆盖率 (`go test -cover`)
- [ ] 无 linter 警告 (`golangci-lint run`)
- [ ] 代码复杂度 < 15 (cyclomatic complexity)
- [ ] 所有公开方法有文档注释

### 安全性
- [ ] 密钥加密存储
- [ ] 时间戳防重放
- [ ] 备份码单次有效
- [ ] 无敏感信息日志

### 兼容性
- [ ] 与现有前端 API 兼容
- [ ] 数据库向后兼容
- [ ] 现有用户无需重新设置 2FA

---

## 7. 风险和缓解 (Risks & Mitigation)

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| 数据库迁移失败 | 高 | 先在测试环境验证，添加回滚脚本 |
| 现有用户无法登录 | 高 | 保留旧验证逻辑作为降级方案 |
| 时间同步问题 | 中 | 文档说明 NTP 要求，增加 Skew |
| 性能影响 | 低 | 数据库索引 `two_factor_last_used_at` |

---

## 8. 参考资料 (References)

- [RFC 6238 - TOTP](https://datatracker.ietf.org/doc/html/rfc6238)
- [google2fa Documentation](https://github.com/antonioribeiro/google2fa)
- [pquerna/otp Library](https://github.com/pquerna/otp)
- [OWASP 2FA Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Multifactor_Authentication_Cheat_Sheet.html)

---

## 9. 总结 (Summary)

本重构方案遵循 **KISS 原则**，通过以下关键改进提升系统质量:

1. **安全增强**: 160-bit 密钥 + 重放攻击防护
2. **职责清晰**: 移除 `VerifyAndLogin()` 越界职责
3. **可测试性**: 100% 覆盖率 + 依赖注入
4. **可维护性**: 配置化设计 + 清晰注释

**预计工作量**: 2-3 天 (包括测试和文档)

**优先级**: 高 (安全相关)
