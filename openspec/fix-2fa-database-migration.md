# 2FA 数据库 Migration 修复提案

## 问题描述

数据库中缺少 2FA 相关字段，后台报错：
```
ERROR: relation "two_factor_secret" does not exist (SQLSTATE 42P01)
```

## 根本原因分析

1. **Migration 004 和 005 使用函数形式** - 不符合 Migration 接口规范
2. **Migration 未被注册** - `cmd/migrate/main.go` 只注册了 001-003
3. **Migration 006 已被删除** - 之前删除了 `pending_login_sessions` 表

## 当前 Migration 状态

| 文件 | 类型 | 注册状态 | 问题 |
|------|------|---------|------|
| 001_create_users_table.go | struct | ✅ 已注册 | 正常 |
| 002_create_lending_positions_table.go | struct | ✅ 已注册 | 正常 |
| 003_create_withdrawal_tables.go | struct | ✅ 已注册 | 正常 |
| 004_add_two_factor_columns.go | 函数 | ❌ 未注册 | 不符合接口 |
| 005_add_two_factor_timestamp.go | 函数 | ❌ 未注册 | 不符合接口 |
| 007_update_wallet_requests_table.go | struct | ❌ 未注册 | 未注册 |

## 修复方案

### 方案：重构 004 和 005 为标准 Migration 接口

将函数形式的 migrations 重构为符合 `Migration` 接口的 struct 形式，并在 `cmd/migrate/main.go` 中注册。

## 实施步骤

1. **重构 004_add_two_factor_columns.go**
   - 创建 `AddTwoFactorColumnsMigration` struct
   - 实现 `Version()`, `Description()`, `Up()`, `Down()` 方法

2. **重构 005_add_two_factor_timestamp.go**
   - 创建 `AddTwoFactorTimestampMigration` struct
   - 实现 `Version()`, `Description()`, `Up()`, `Down()` 方法

3. **更新 cmd/migrate/main.go**
   - 注册 004, 005, 007 migrations

4. **运行测试**
   - 验证 migration 可以正常执行
   - 验证 rollback 可以正常执行

## 代码变更

### 1. 004_add_two_factor_columns.go
```go
type AddTwoFactorColumnsMigration struct{}

func (m *AddTwoFactorColumnsMigration) Version() string {
    return "004"
}

func (m *AddTwoFactorColumnsMigration) Description() string {
    return "Add two factor columns to users table"
}

func (m *AddTwoFactorColumnsMigration) Up(db *sql.DB) error {
    // 添加 two_factor_secret, two_factor_backup_codes, two_factor_enabled 列
}

func (m *AddTwoFactorColumnsMigration) Down(db *sql.DB) error {
    // 删除列
}
```

### 2. cmd/migrate/main.go
```go
migrator.Register(&migrations.CreateUsersTable{})
migrator.Register(&migrations.CreateLendingPositionsTable{})
migrator.Register(&migrations.CreateWithdrawalTables{})
migrator.Register(&migrations.AddTwoFactorColumnsMigration{})
migrator.Register(&migrations.AddTwoFactorTimestampMigration{})
migrator.Register(&migrations.UpdateWalletRequestsTable{})
```

## 测试策略

1. 测试 migration 可以正常应用
2. 测试 rollback 可以正常执行
3. 验证数据库表结构正确
4. 验证 2FA 功能正常工作

## 回滚策略

如果需要回滚：
1. 执行 `migrator.Rollback()` 回滚到之前版本
2. 或者手动删除新增的列

## 安全考虑

- Migration 使用 `IF EXISTS` / `IF NOT EXISTS` 避免重复执行错误
- Down 方法确保可以安全回滚
- 所有操作都是幂等的
