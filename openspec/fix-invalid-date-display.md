# 修复地址列表日期显示为 "Invalid Date" 的问题

## 问题描述

用户 `gyc567@gmail.com` 访问地址列表页面时，看到：
- 已添加: Invalid Date
- 已验证: Invalid Date

## 问题分析

### 根本原因

后端返回的 JSON 字段名与前端期望的不一致：

**后端模型** (`internal/models/models.go`):
```go
type WithdrawalAddress struct {
    CreatedAt  time.Time    `json:"createdAt"`     // camelCase ✅
    VerifiedAt sql.NullTime `json:"verified_at"`   // snake_case ❌
}
```

**前端期望** (`src/pages/dashboard/Addresses.tsx`):
```typescript
interface WithdrawalAddress {
    created_at: string;  // snake_case ❌
    verified_at: string | null;  // snake_case ❌
}
```

**问题**：字段命名不统一，导致前端解析失败。

### 规范要求
根据 AGENTS.md，**所有 API JSON 字段必须使用 camelCase**。

## 修复方案

### 统一使用 camelCase

1. 修改 `internal/dto/address.go` - DTO 字段使用 camelCase
2. 修改 `src/pages/dashboard/Addresses.tsx` - 前端接口使用 camelCase
3. 使用 DTO 转换模型数据，确保 API 响应格式一致

## 实施步骤

1. ✅ 更新 DTO 字段名为 camelCase
2. ✅ 更新前端接口为 camelCase
3. ✅ 修改 handler 使用 DTO 转换
4. ✅ 添加单元测试验证转换逻辑
5. ✅ 运行回归测试

## 测试策略

### 新增测试
- `TestConvertAddressToDTO` - 测试模型到 DTO 的转换
- `TestWithdrawalAddressResponse_JSONFormat` - 测试 JSON 格式 (camelCase)
- `TestWithdrawalAddressResponse_NullVerifiedAt` - 测试 null 值处理

### 回归测试
```bash
go test ./internal/...
# 结果: 全部通过
```

## 设计原则

1. **KISS**: 使用 DTO 模式，简单直接
2. **高内聚低耦合**: DTO 专门负责 API 格式，与模型分离
3. **测试覆盖**: 新增 3 个单元测试，覆盖所有场景
4. **隔离性**: 只修改 DTO 和 handler，不影响其他功能
5. **规范统一**: 严格遵循 camelCase API JSON 规范
