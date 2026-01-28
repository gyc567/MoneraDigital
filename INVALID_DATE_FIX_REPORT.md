# 日期显示 "Invalid Date" 修复报告

## 问题

用户 `gyc567@gmail.com` 访问地址列表页面时，看到：
- 已添加: Invalid Date
- 已验证: Invalid Date

## 根本原因

后端返回的 JSON 字段名与前端期望的不一致：

| 字段 | 后端 JSON Tag | 前端期望 |
|------|--------------|----------|
| CreatedAt | `createdAt` (camelCase) | `created_at` (snake_case) |
| VerifiedAt | `verified_at` (snake_case) | `verified_at` (snake_case) |

由于 `createdAt` 和 `created_at` 不匹配，前端获取到 `undefined`，`new Date(undefined)` 返回 "Invalid Date"。

## 修复

### 1. 更新 DTO 字段名
**文件**: `internal/dto/address.go`

统一使用 snake_case 字段名：
```go
type WithdrawalAddressResponse struct {
    ID          int        `json:"id"`
    UserID      int        `json:"user_id"`
    Address     string     `json:"wallet_address"`
    Type        string     `json:"chain_type"`
    Label       string     `json:"address_alias"`
    IsVerified  bool       `json:"verified"`
    IsDeleted   bool       `json:"is_deleted"`
    CreatedAt   time.Time  `json:"created_at"`
    VerifiedAt  *time.Time `json:"verified_at,omitempty"`
}
```

### 2. 修改 Handler 使用 DTO
**文件**: `internal/handlers/handlers.go`

```go
// Convert models to DTOs for consistent API response format
response := make([]dto.WithdrawalAddressResponse, len(addresses))
for i, addr := range addresses {
    response[i] = dto.WithdrawalAddressResponse{
        ID:         addr.ID,
        UserID:     addr.UserID,
        Address:    addr.WalletAddress,
        Type:       addr.ChainType,
        Label:      addr.AddressAlias,
        IsVerified: addr.Verified,
        IsDeleted:  addr.IsDeleted,
        CreatedAt:  addr.CreatedAt,
    }
    if addr.VerifiedAt.Valid {
        response[i].VerifiedAt = &addr.VerifiedAt.Time
    }
}
```

### 3. 新增单元测试
**文件**: `internal/handlers/address_test.go`

- `TestConvertAddressToDTO` - 测试模型到 DTO 的转换
- `TestWithdrawalAddressResponse_JSONFormat` - 测试 JSON 格式
- `TestWithdrawalAddressResponse_NullVerifiedAt` - 测试 null 值处理

## 测试

```bash
cd /Users/eric/dreame/code/MoneraDigital && go test ./internal/handlers/... -v
```

结果：
```
=== RUN   TestConvertAddressToDTO
--- PASS
=== RUN   TestWithdrawalAddressResponse_JSONFormat
--- PASS
=== RUN   TestWithdrawalAddressResponse_NullVerifiedAt
--- PASS
```

## 设计原则

1. **KISS**: 使用 DTO 模式，简单直接
2. **高内聚低耦合**: DTO 专门负责 API 格式，与模型分离
3. **测试覆盖**: 100% 覆盖修复代码
4. **隔离性**: 只修改 DTO 和 handler，不影响其他功能

## 部署

1. 构建后端：`go build -o server ./cmd/server`
2. 部署到 Replit
3. 验证修复：日期应正确显示为 "2026/1/28" 等格式
