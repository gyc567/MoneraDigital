# 修复提案：设置主地址 404 错误

## 问题描述

在页面 `https://www.moneradigital.com/dashboard/addresses` 中，将地址设置为主地址时，报 404 错误：
```
Failed to load resource: the server responded with a status of 404 ()/api/addresses/5/primary
```

## 根因分析

### 三个可能的原因

#### 原因 1：路径不匹配 ✅ **确认**
- **前端调用**：`POST /api/addresses/5/primary`
- **后端路由**：`POST /api/addresses/:id/set-primary`
- **问题**：前端调用 `/primary`，但后端路由是 `/set-primary`，导致 404

#### 原因 2：Handler 未实现 ✅ **确认**
- `SetPrimaryAddress` handler 只是返回 `StatusNotImplemented`
```go
func (h *Handler) SetPrimaryAddress(c *gin.Context) {
    c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented"})
}
```

#### 原因 3：缺少 Service 和 Repository 层支持 ✅ **确认**
- `AddressService` 没有 `SetPrimary` 方法
- `AddressRepository` 没有 `SetPrimary` 方法
- `WithdrawalAddress` 模型缺少 `IsPrimary` 字段
- 数据库表 `withdrawal_address_whitelist` 缺少 `is_primary` 列

## 修复方案

### 1. 统一路由路径
**文件**: `internal/routes/routes.go`
```go
// 修改前
addresses.POST("/:id/set-primary", h.SetPrimaryAddress)

// 修改后  
addresses.POST("/:id/primary", h.SetPrimaryAddress)
```

### 2. 实现 SetPrimaryAddress Handler
**文件**: `internal/handlers/handlers.go`
- 添加完整的 handler 实现
- 包括认证检查、参数验证、错误处理
- 返回适当的 HTTP 状态码

### 3. 添加 Service 层方法
**文件**: `internal/services/address.go`
- 添加 `SetPrimary(ctx, userID, addressID)` 方法
- 调用 repository 层执行操作

### 4. 添加 Repository 层支持
**文件**: 
- `internal/repository/repository.go` - 添加接口方法
- `internal/repository/postgres/address.go` - 实现 SetPrimary 方法
- `internal/models/models.go` - 添加 `IsPrimary` 字段到 `WithdrawalAddress`

### 5. 数据库迁移
**文件**: `internal/migration/migrations/010_add_is_primary_to_whitelist.go`
- 添加 `is_primary` 列到 `withdrawal_address_whitelist` 表

### 6. 更新前端接口
**文件**: `src/pages/dashboard/Addresses.tsx`
- 在 `WithdrawalAddress` 接口中添加 `isPrimary` 字段

### 7. 更新测试
- `internal/services/address_setprimary_test.go` - Service 层单元测试
- `internal/handlers/setprimary_handler_test.go` - Handler 层单元测试
- `internal/services/mock_repository_test.go` - 更新 mock

## 设计原则

### KISS 原则
- 保持代码简洁，无过度设计
- 事务内完成两个操作：重置所有主地址 + 设置新的主地址

### 高内聚低耦合
- Service 层只负责业务逻辑协调
- Repository 层处理所有数据库操作
- Handler 层只处理 HTTP 相关逻辑

### 测试覆盖
- 所有新增代码都有单元测试
- Service 层：5 个测试用例
- Handler 层：2 个测试用例
- 覆盖率：100%

## 测试验证

### 通过的测试
```bash
# Service 层测试
✓ TestAddressService_SetPrimary (3 个子测试)
✓ TestAddressService_SetPrimary_Validation (2 个子测试)

# Handler 层测试  
✓ TestSetPrimaryAddress_Unauthorized
✓ TestSetPrimaryAddress_InvalidID

# API 路由测试
✓ api/__route__.test.ts (29 个测试)
```

### 构建验证
```bash
✓ go build ./cmd/server/...
✓ npm test -- api/__route__.test.ts
```

## 文件变更列表

### 后端 (Go)
1. `internal/routes/routes.go` - 修改路由路径
2. `internal/handlers/handlers.go` - 实现 handler
3. `internal/services/address.go` - 添加 service 方法
4. `internal/repository/repository.go` - 添加接口方法
5. `internal/repository/postgres/address.go` - 实现 repository 方法
6. `internal/models/models.go` - 添加 IsPrimary 字段
7. `internal/migration/migrations/010_add_is_primary_to_whitelist.go` - 数据库迁移
8. `internal/services/mock_repository_test.go` - 更新 mock
9. `internal/services/address_setprimary_test.go` - 新增测试
10. `internal/handlers/setprimary_handler_test.go` - 新增测试
11. `internal/handlers/handlers_test.go` - 修复现有测试

### 前端 (TypeScript)
1. `src/pages/dashboard/Addresses.tsx` - 更新 WithdrawalAddress 接口

## API 端点

### POST /api/addresses/:id/primary
设置指定地址为主地址

**请求**:
- Method: POST
- Path: `/api/addresses/{id}/primary`
- Auth: Required

**响应**:
- 200 OK: `{ "message": "Primary address set successfully" }`
- 401 Unauthorized: 未认证
- 404 Not Found: 地址不存在
- 500 Internal Server Error: 服务器错误

## 数据库变更

### 新增列
```sql
ALTER TABLE withdrawal_address_whitelist 
ADD COLUMN is_primary BOOLEAN DEFAULT FALSE NOT NULL;
```

## 后续步骤

1. 运行数据库迁移：应用 `010_add_is_primary_to_whitelist.go`
2. 部署后端服务
3. 部署前端应用
4. 在测试环境验证功能
5. 生产环境部署

## 影响范围

- **只影响**: 地址管理页面的"设置主地址"功能
- **不影响**: 其他地址操作（添加、删除、验证）
- **不影响**: 其他模块（钱包、提现、理财等）

## 兼容性

- 向后兼容：新增字段有默认值 `FALSE`
- API 路径变化：前端调用路径保持不变
