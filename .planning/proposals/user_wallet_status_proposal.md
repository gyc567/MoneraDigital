# UserWallet 状态管理改进提案

## 概述

为 `user_wallets` 表增加状态管理功能，支持钱包地址的冻结、注销等操作。

## 变更内容

### 1. 数据库变更

#### 移除 NOT NULL 约束
- `request_id` 字段从 `NOT NULL` 改为 `NULL`
- 原因：允许手动添加的钱包地址（不关联 request_id）

#### 新增状态字段
```sql
status VARCHAR(20) NOT NULL DEFAULT 'NORMAL'
CHECK (status IN ('NORMAL', 'FROZEN', 'CANCELLED'))
```

### 2. 枚举类型定义

```go
type UserWalletStatus string

const (
    UserWalletStatusNormal    UserWalletStatus = "NORMAL"
    UserWalletStatusFrozen    UserWalletStatus = "FROZEN"
    UserWalletStatusCancelled UserWalletStatus = "CANCELLED"
)
```

### 3. 架构原则

- **KISS**: 简单状态枚举，无复杂状态机
- **高内聚**: 状态逻辑集中在 UserWallet 模型
- **低耦合**: 状态变更通过 Repository 接口，不影响其他模块

### 4. 实现范围

| 文件 | 变更 |
|------|------|
| `internal/models/models.go` | 添加 UserWalletStatus 类型，更新 UserWallet 结构体 |
| `internal/migration/migrations/009_user_wallet_status.go` | 新增迁移文件 |
| `internal/repository/repository.go` | 添加 UpdateUserWalletStatus 方法 |
| `internal/repository/postgres/wallet.go` | 实现状态更新方法 |
| `internal/services/wallet.go` | 同步逻辑更新 |
| `drizzle/monera_complete_schema.sql` | 更新 schema 文档 |
| `internal/*/mock_*.go` | 更新所有 Mock 实现 |

### 5. 测试要求

- 所有新增方法 100% 测试覆盖率
- 不影响现有测试

### 6. 风险评估

- **低风险**: 仅新增字段和枚举，不影响现有功能
- **向后兼容**: 新增字段有默认值，不影响现有数据
