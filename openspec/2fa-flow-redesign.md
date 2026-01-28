# OpenSpec: 2FA 流程重新设计

## 1. 目标

重新设计 2FA 验证流程：
- **登录时**: 不强制要求 2FA 验证，所有用户直接登录
- **提现时**: 强制要求 2FA 验证
- **验证提现白名单时**: 强制要求 2FA 验证

## 2. 当前问题

当前流程：
1. 用户登录时，如果启用了 2FA，需要输入 2FA 验证码
2. 这导致用户体验不佳，每次登录都需要 2FA

期望流程：
1. 用户登录时，直接完成登录，不需要 2FA
2. 用户进行敏感操作（提现、验证白名单）时，才需要 2FA

## 3. 功能设计

### 3.1 登录流程变更

**当前**: 
```
登录 → 检查 2FA → 需要 2FA → 输入 2FA 验证码 → 完成登录
```

**新流程**:
```
登录 → 直接完成登录（无论是否启用 2FA）
```

### 3.2 敏感操作 2FA 验证

**提现流程**:
```
发起提现 → 检查 2FA → 需要 2FA → 输入 2FA 验证码 → 完成提现
```

**验证白名单流程**:
```
验证地址 → 检查 2FA → 需要 2FA → 输入 2FA 验证码 → 完成验证
```

### 3.3 API 变更

#### 登录 API (POST /api/auth/login)
- **变更**: 移除 2FA 检查，直接返回 JWT token
- **响应**: 不再返回 `requires2FA` 字段

#### 提现 API (POST /api/withdrawals)
- **新增**: 2FA 验证步骤
- **请求体**: 添加 `twoFactorToken` 字段
- **错误**: 如果 2FA 验证失败，返回 401

#### 验证白名单 API (POST /api/addresses/:id/verify)
- **变更**: 如果用户启用 2FA，强制要求 2FA 验证
- **请求体**: 已有 `token` 字段，用于 2FA 验证

## 4. 业务逻辑

### 4.1 AuthService.Login

```go
func (s *AuthService) Login(req models.LoginRequest) (*LoginResponse, error) {
    // 验证用户和密码
    // ...
    
    // 不再检查 2FA，直接生成 JWT
    token, err := utils.GenerateJWT(user.ID, user.Email, s.jwtSecret)
    // ...
    
    return &LoginResponse{
        User:        &user,
        AccessToken: token,
        // ...
    }, nil
}
```

### 4.2 WithdrawalService.CreateWithdrawal

```go
func (s *WithdrawalService) CreateWithdrawal(ctx context.Context, userID int, req models.CreateWithdrawalRequest, twoFactorToken string) (*WithdrawalOrder, error) {
    // 检查用户是否启用 2FA
    user, err := s.AuthService.GetUserByID(userID)
    if err != nil {
        return nil, err
    }
    
    // 如果启用 2FA，验证 2FA token
    if user.TwoFactorEnabled {
        valid, err := s.AuthService.Verify2FA(userID, twoFactorToken)
        if err != nil || !valid {
            return nil, errors.New("invalid 2FA code")
        }
    }
    
    // 创建提现订单
    // ...
}
```

### 4.3 AddressService.VerifyAddress

```go
func (s *AddressService) VerifyAddress(ctx context.Context, userID int, addressID int, token string) error {
    // 检查用户是否启用 2FA
    user, err := s.AuthService.GetUserByID(userID)
    if err != nil {
        return err
    }
    
    // 如果启用 2FA，验证 2FA token
    if user.TwoFactorEnabled {
        valid, err := s.AuthService.Verify2FA(userID, token)
        if err != nil || !valid {
            return errors.New("invalid 2FA code")
        }
    } else {
        // 验证邮箱验证码
        // ...
    }
    
    // 验证地址
    // ...
}
```

## 5. 设计原则 (KISS)

### 5.1 简化登录流程
- 登录时不再检查 2FA，直接返回 JWT
- 提升用户体验，减少登录步骤

### 5.2 集中 2FA 验证
- 2FA 验证集中在敏感操作（提现、白名单验证）
- 保持安全性，不影响用户体验

### 5.3 向后兼容
- 保留 2FA 设置和验证功能
- 用户可以选择启用或禁用 2FA

## 6. 验证标准

- [ ] 用户登录时不需要 2FA 验证
- [ ] 用户提现时需要 2FA 验证（如果启用 2FA）
- [ ] 用户验证白名单时需要 2FA 验证（如果启用 2FA）
- [ ] 所有测试通过，覆盖率 100%
- [ ] 不影响其他无关功能
