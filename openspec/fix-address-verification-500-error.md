# 修复地址验证 500 错误提案

## 问题描述

用户 `gyc567@gmail.com` 在验证地址时遇到 500 错误：
```
POST https://www.moneradigital.com/api/addresses/12/verify 500 (Internal Server Error)
```

用户已启用 2FA，使用 Google Authenticator，secret key: `YO5CXNI64PL3ZDCUFPIFWJWMCHWECV6O`

## 问题分析

### 根本原因

在 `internal/services/twofa_service.go` 的 `getSecret` 函数中，当查询 `two_factor_secret` 列时：

1. 如果数据库中的值为 NULL，`Scan` 方法会将 `encryptedSecret` 设置为空字符串
2. 而不是返回 `sql.ErrNoRows` 错误
3. 随后尝试解密空字符串会导致错误

```go
// 问题代码
var encryptedSecret string
err := s.DB.QueryRow(query, userID).Scan(&encryptedSecret)
// 当 two_factor_secret 为 NULL 时，encryptedSecret 变成空字符串
// 而不是返回 sql.ErrNoRows
```

### 修复方案

使用 `sql.NullString` 来正确处理 NULL 值：

```go
// 修复后代码
var encryptedSecret sql.NullString
err := s.DB.QueryRow(query, userID).Scan(&encryptedSecret)
if !encryptedSecret.Valid || encryptedSecret.String == "" {
    return "", fmt.Errorf("2FA not set up")
}
```

## 修复内容

### 1. 修复 `getSecret` 函数
文件：`internal/services/twofa_service.go:191-210`

- 使用 `sql.NullString` 替代 `string`
- 添加 NULL 和空字符串检查

### 2. 修复 `getBackupCodes` 函数
文件：`internal/services/twofa_service.go:212-230`

- 同样使用 `sql.NullString` 处理可能为 NULL 的备份码

### 3. 增强错误日志
文件：`internal/handlers/handlers.go:377-432`

- 在 2FA 验证失败时添加详细日志
- 在地址验证失败时添加详细日志

## 测试

所有现有测试通过：
- `go test ./internal/services/...` - PASS
- `go test ./internal/handlers/...` - PASS

## 部署

1. 构建后端：`go build -o server ./cmd/server`
2. 部署到生产环境
3. 监控日志确认问题已解决

## 验证

用户需要：
1. 打开 Google Authenticator
2. 获取当前 6 位 TOTP 码
3. 在地址验证对话框中输入
4. 验证应该成功，不再返回 500 错误
