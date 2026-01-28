# Drizzle Migration vs Complete Schema 对比分析

## 文件概述

| 属性 | `0000_black_human_fly.sql` | `monera_complete_schema.sql` |
|------|---------------------------|------------------------------|
| **用途** | Drizzle ORM 生成的迁移文件 | 手动维护的完整 Schema 文档 |
| **生成方式** | `drizzle-kit generate` | 手工编写 |
| **主要使用者** | 前端/Node.js 开发 | DBA/后端开发/文档 |
| **更新频率** | 每次修改 `schema.ts` | 定期手动同步 |
| **表数量** | 26 张 | 29 张 |
| **索引数量** | 较少 | 完整（40+ 个） |

---

## 核心关系

```
src/db/schema.ts
       ↓
   [Drizzle Kit]
       ↓
0000_black_human_fly.sql (自动生成的迁移)
       ↓
   [人工整理/补充]
       ↓
monera_complete_schema.sql (完整文档)
```

**关系说明**:
- `0000_black_human_fly.sql` 是 **Drizzle 的迁移输出**
- `monera_complete_schema.sql` 是 **人工维护的完整 Schema**
- 两者应该保持一致，但存在差异

---

## 详细差异对比

### 一、表数量差异

| 表名 | 0000_black_human_fly.sql | monera_complete_schema.sql | 说明 |
|------|-------------------------|---------------------------|------|
| `users` | ✅ | ✅ | 都有 |
| `account` | ✅ | ✅ | 都有 |
| `account_journal` | ✅ | ✅ | 都有 |
| `account_adjustment` | ✅ | ✅ | 都有 |
| `wealth_product` | ✅ | ✅ | 都有 |
| `wealth_order` | ✅ | ✅ | 都有 |
| `wealth_interest_record` | ✅ | ✅ | 都有 |
| `wealth_product_approval` | ✅ | ✅ | 都有 |
| `lending_positions` | ✅ | ✅ | 都有 |
| `deposits` | ✅ | ✅ | 都有 |
| `withdrawals` | ✅ | ✅ | 都有 |
| `withdrawal_order` | ✅ | ✅ | 都有 |
| `withdrawal_request` | ✅ | ✅ | 都有 |
| `withdrawal_addresses` | ✅ | ✅ | 都有 |
| `withdrawal_address_whitelist` | ✅ | ✅ | 都有 |
| `withdrawal_freeze_log` | ✅ | ✅ | 都有 |
| `address_verifications` | ✅ | ✅ | 都有 |
| `wallet_creation_requests` | ✅ | ✅ | 都有（旧表） |
| `idempotency_record` | ✅ | ✅ | 都有 |
| `transfer_record` | ✅ | ✅ | 都有 |
| `reconciliation_log` | ✅ | ✅ | 都有 |
| `reconciliation_alert_log` | ✅ | ✅ | 都有 |
| `reconciliation_error_log` | ✅ | ✅ | 都有 |
| `manual_review_queue` | ✅ | ✅ | 都有 |
| `business_freeze_status` | ✅ | ✅ | 都有 |
| `audit_trail` | ✅ | ✅ | 都有 |
| `admin_users` | ❌ | ✅ | **只在完整 Schema** |
| `wallet_creation_request` | ❌ | ✅ | **只在完整 Schema**（新表） |
| `withdrawal_verification` | ❌ | ✅ | **只在完整 Schema** |

**总结**: 
- `0000_black_human_fly.sql`: 26 张表
- `monera_complete_schema.sql`: 29 张表
- **缺失 3 张表**: `admin_users`, `wallet_creation_request`, `withdrawal_verification`

### 二、索引差异

#### 0000_black_human_fly.sql 的索引
- 仅包含表定义中的主键和唯一约束
- 少量外键索引
- **缺少性能优化索引**

#### monera_complete_schema.sql 的索引（40+ 个）

**users 表索引**:
```sql
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_two_factor_last_used_at ON users(two_factor_last_used_at) WHERE two_factor_enabled = TRUE;
```

**account 表索引**:
```sql
CREATE INDEX IF NOT EXISTS idx_account_user_id ON account(user_id);
CREATE INDEX IF NOT EXISTS idx_account_type ON account(type);
CREATE INDEX IF NOT EXISTS idx_account_frozen_balance ON account(user_id, frozen_balance);
CREATE INDEX IF NOT EXISTS idx_account_updated_at ON account(updated_at);
CREATE UNIQUE INDEX IF NOT EXISTS uk_user_type_currency ON account(user_id, type, currency);
```

**wealth_order 表索引**:
```sql
CREATE INDEX IF NOT EXISTS idx_wealth_order_user_id ON wealth_order(user_id);
CREATE INDEX IF NOT EXISTS idx_wealth_order_product_id ON wealth_order(product_id);
CREATE INDEX IF NOT EXISTS idx_wealth_order_end_date ON wealth_order(end_date);
CREATE INDEX IF NOT EXISTS idx_wealth_order_status ON wealth_order(status);
CREATE INDEX IF NOT EXISTS idx_wealth_order_renewed_from ON wealth_order(renewed_from_order_id);
```

**总结**: 完整 Schema 包含大量性能优化索引，Drizzle 生成的文件缺少这些。

### 三、字段差异

#### users 表

| 字段 | 0000_black_human_fly.sql | monera_complete_schema.sql |
|------|-------------------------|---------------------------|
| `id` | ✅ | ✅ |
| `email` | ✅ | ✅ |
| `password` | ✅ | ✅ |
| `two_factor_secret` | ✅ | ✅ |
| `two_factor_enabled` | ✅ | ✅ |
| `two_factor_backup_codes` | ✅ | ✅ |
| `created_at` | ✅ | ✅ |
| `two_factor_last_used_at` | ❌ | ✅ |
| `updated_at` | ❌ | ✅ |

**差异**: 完整 Schema 多了 2 个字段（来自 Go Migration 005）

#### wallet_creation_request 表

| 字段 | 0000_black_human_fly.sql | monera_complete_schema.sql |
|------|-------------------------|---------------------------|
| `product_code` | ❌ | ✅ |
| `currency` | ❌ | ✅ |

**差异**: 完整 Schema 多了 2 个字段（来自 Go Migration 007）

### 四、语法差异

| 特性 | 0000_black_human_fly.sql | monera_complete_schema.sql |
|------|-------------------------|---------------------------|
| **表名引号** | 双引号 `"users"` | 无双引号 `users` |
| **IF NOT EXISTS** | ❌ 不使用 | ✅ 使用 |
| **字段引号** | 双引号 `"email"` | 无双引号 `email` |
| **注释** | ❌ 无 | ✅ 详细注释 |
| **外键位置** | 文件末尾集中定义 | 表定义中内联 |

### 五、其他差异

#### 视图（Views）
- **0000_black_human_fly.sql**: ❌ 无视图
- **monera_complete_schema.sql**: ✅ 包含 `v_account_available`

#### 注释（Comments）
- **0000_black_human_fly.sql**: ❌ 无注释
- **monera_complete_schema.sql**: ✅ 所有表和字段都有详细注释

---

## 使用场景对比

### 何时使用 `0000_black_human_fly.sql`？

| 场景 | 原因 |
|------|------|
| 前端开发环境初始化 | Drizzle Kit 自动生成，与 TypeScript 类型同步 |
| 快速原型开发 | 与 `src/db/schema.ts` 保持一致 |
| Drizzle ORM 操作 | Drizzle 需要此文件进行类型推断 |
| 开发环境数据库重置 | `npx drizzle-kit push` 使用此文件 |

### 何时使用 `monera_complete_schema.sql`？

| 场景 | 原因 |
|------|------|
| 生产环境部署 | 包含完整索引和优化 |
| DBA 审查 | 包含详细注释和文档 |
| 数据库文档 | 人工维护，易于阅读 |
| 性能调优 | 包含所有性能索引 |
| 新环境初始化 | 包含所有表和字段 |

---

## 同步建议

### 方案一：统一使用 Go Migration（推荐生产环境）

```bash
# 1. 使用 Go migrator 执行所有迁移
go build -o migrate ./cmd/migrate
./migrate

# 2. 导出当前 Schema 作为文档
pg_dump --schema-only $DATABASE_URL > schema_backup.sql
```

### 方案二：同步 Drizzle Schema

更新 `src/db/schema.ts`，添加缺失的表和字段：

```typescript
// 添加缺失的字段
export const users = pgTable('users', {
  // ... 现有字段
  twoFactorLastUsedAt: bigint('two_factor_last_used_at', { mode: 'number' }).default(0),
  updatedAt: timestamp('updated_at').defaultNow().notNull(),
});

// 添加缺失的表
export const adminUsers = pgTable('admin_users', {
  id: serial('id').primaryKey(),
  username: text('username').notNull().unique(),
  passwordHash: text('password_hash').notNull(),
  createdAt: timestamp('created_at').defaultNow(),
});

export const withdrawalVerification = pgTable('withdrawal_verification', {
  // ... 字段定义
});
```

然后重新生成：
```bash
npx drizzle-kit generate
```

### 方案三：维护两个文件（当前做法）

- `0000_black_human_fly.sql` - 用于 Drizzle/前端开发
- `monera_complete_schema.sql` - 用于文档和 DBA 参考

定期人工同步差异。

---

## 总结

| 维度 | 结论 |
|------|------|
| **关系** | `monera_complete_schema.sql` 是 `0000_black_human_fly.sql` 的超集 |
| **完整性** | 完整 Schema 更全（29 vs 26 张表，40+ 索引） |
| **时效性** | 两者都需要更新以反映最新的 Go Migrations |
| **推荐用法** | 生产用 Go Migration，开发用 Drizzle，文档用完整 Schema |

**关键问题**: 两个文件都与 Go Migrations 不同步，需要统一维护策略。
