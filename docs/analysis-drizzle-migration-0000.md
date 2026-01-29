# Drizzle Migration 脚本分析报告

## 文件信息

| 属性 | 值 |
|------|-----|
| **文件名** | `0000_black_human_fly.sql` |
| **位置** | `drizzle/0000_black_human_fly.sql` |
| **生成工具** | Drizzle Kit |
| **来源** | `src/db/schema.ts` |
| **行数** | 368 行 |
| **用途** | 初始数据库 Schema 创建 |

---

## 脚本作用

### 1. 核心功能
这是 **Drizzle ORM** 生成的初始数据库迁移脚本，用于：
- 创建 PostgreSQL 数据库的完整表结构
- 定义 ENUM 类型
- 建立表之间的关系（外键约束）
- 设置主键、唯一约束、默认值

### 2. 生成方式
```bash
# 由以下命令生成
npx drizzle-kit generate
# 或
npx drizzle-kit push
```

### 3. 数据来源
根据 `drizzle.config.ts` 配置，从 `src/db/schema.ts` 生成：
```typescript
export default defineConfig({
  schema: './src/db/schema.ts',  // 源文件
  out: './drizzle',              // 输出目录
  dialect: 'postgresql',
});
```

---

## 脚本内容详解

### 一、ENUM 类型定义（第 1-5 行）

```sql
CREATE TYPE "public"."address_type" AS ENUM('BTC', 'ETH', 'USDC', 'USDT');
CREATE TYPE "public"."deposit_status" AS ENUM('PENDING', 'CONFIRMED', 'FAILED');
CREATE TYPE "public"."lending_status" AS ENUM('ACTIVE', 'COMPLETED', 'TERMINATED');
CREATE TYPE "public"."wallet_creation_status" AS ENUM('CREATING', 'SUCCESS', 'FAILED');
CREATE TYPE "public"."withdrawal_status" AS ENUM('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED');
```

| ENUM 名称 | 用途 | 对应代码 |
|-----------|------|---------|
| `address_type` | 地址类型（币种） | `addressTypeEnum` |
| `deposit_status` | 充值状态 | `depositStatusEnum` |
| `lending_status` | 借贷状态 | `lendingStatusEnum` |
| `wallet_creation_status` | 钱包创建状态 | `walletCreationStatusEnum` |
| `withdrawal_status` | 提现状态 | `withdrawalStatusEnum` |

### 二、表结构创建（第 6-361 行）

共创建 **26 张表**，分为以下几类：

#### 1. 核心用户与认证（1 张）
| 表名 | 行号 | 说明 |
|------|------|------|
| `users` | 183-192 | 用户认证表，包含 2FA 字段 |

**关键字段**:
- `two_factor_secret` - TOTP 密钥
- `two_factor_enabled` - 2FA 启用状态
- `two_factor_backup_codes` - 备用验证码

#### 2. 账户与资金（3 张）
| 表名 | 行号 | 说明 |
|------|------|------|
| `account` | 37-47 | 统一账户表（新架构） |
| `account_journal` | 24-35 | 账户资金流水 |
| `account_adjustment` | 6-22 | 账户调整记录 |

#### 3. 理财（Wealth）（4 张）
| 表名 | 行号 | 说明 |
|------|------|------|
| `wealth_product` | 260-274 | 理财产品 |
| `wealth_order` | 217-238 | 理财订单 |
| `wealth_interest_record` | 208-215 | 利息记录 |
| `wealth_product_approval` | 240-258 | 产品审批流程 |

#### 4. 借贷（Legacy）（1 张）
| 表名 | 行号 | 说明 |
|------|------|------|
| `lending_positions` | 110-121 | 旧版借贷仓位 |

#### 5. 钱包与地址（3 张）
| 表名 | 行号 | 说明 |
|------|------|------|
| `wallet_creation_requests` | 194-206 | 钱包创建请求（旧） |
| `withdrawal_addresses` | 290-301 | 提现地址 |
| `address_verifications` | 49-56 | 地址验证 |

#### 6. 提现（5 张）
| 表名 | 行号 | 说明 |
|------|------|------|
| `withdrawals` | 344-360 | 提现记录（旧） |
| `withdrawal_order` | 313-331 | 提现订单（新） |
| `withdrawal_request` | 333-342 | 提现请求 |
| `withdrawal_address_whitelist` | 276-288 | 地址白名单 |
| `withdrawal_freeze_log` | 303-311 | 提现冻结日志 |
| `withdrawal_verification` | - | 提现验证（代码中有，SQL中缺失） |

#### 7. 充值（1 张）
| 表名 | 行号 | 说明 |
|------|------|------|
| `deposits` | 82-95 | 充值记录 |

#### 8. 转账（1 张）
| 表名 | 行号 | 说明 |
|------|------|------|
| `transfer_record` | 170-181 | 账户间转账 |

#### 9. 幂等性控制（1 张）
| 表名 | 行号 | 说明 |
|------|------|------|
| `idempotency_record` | 97-108 | 幂等性记录 |

#### 10. 对账与风控（5 张）
| 表名 | 行号 | 说明 |
|------|------|------|
| `reconciliation_log` | 159-168 | 对账日志 |
| `reconciliation_alert_log` | 134-146 | 对账告警 |
| `reconciliation_error_log` | 148-157 | 对账错误 |
| `manual_review_queue` | 123-132 | 人工审核队列 |
| `business_freeze_status` | 74-80 | 全局冻结状态 |

#### 11. 审计（2 张）
| 表名 | 行号 | 说明 |
|------|------|------|
| `audit_trail` | 58-72 | 审计日志 |
| `admin_users` | - | 管理员用户（代码中有，SQL中缺失） |

### 三、外键约束（第 362-368 行）

```sql
ALTER TABLE "address_verifications" ADD CONSTRAINT ... FOREIGN KEY ("address_id") REFERENCES "withdrawal_addresses";
ALTER TABLE "deposits" ADD CONSTRAINT ... FOREIGN KEY ("user_id") REFERENCES "users";
ALTER TABLE "lending_positions" ADD CONSTRAINT ... FOREIGN KEY ("user_id") REFERENCES "users";
ALTER TABLE "wallet_creation_requests" ADD CONSTRAINT ... FOREIGN KEY ("user_id") REFERENCES "users";
ALTER TABLE "withdrawal_addresses" ADD CONSTRAINT ... FOREIGN KEY ("user_id") REFERENCES "users";
ALTER TABLE "withdrawals" ADD CONSTRAINT ... FOREIGN KEY ("user_id") REFERENCES "users";
ALTER TABLE "withdrawals" ADD CONSTRAINT ... FOREIGN KEY ("from_address_id") REFERENCES "withdrawal_addresses";
```

---

## 使用场景

### 1. 新环境初始化
```bash
# 首次部署时使用
psql $DATABASE_URL -f drizzle/0000_black_human_fly.sql
```

### 2. Drizzle Kit 工作流
```bash
# 1. 修改 src/db/schema.ts
# 2. 生成新的 migration
npx drizzle-kit generate

# 3. 应用到数据库
npx drizzle-kit push
```

### 3. 与 Go Migration 的关系
| 类型 | 用途 | 工具 |
|------|------|------|
| Drizzle Migration | 前端/Node.js 开发 | Drizzle Kit |
| Go Migration | 后端/生产部署 | Go migrator |

**注意**: 项目中有两套 migration 系统，需要保持同步！

---

## 与 Go Migrations 的对比

| 特性 | Drizzle (0000_black_human_fly.sql) | Go Migrations |
|------|-----------------------------------|---------------|
| **目的** | 前端开发快速迭代 | 后端生产环境部署 |
| **格式** | 单文件完整 Schema | 多文件增量迁移 |
| **工具** | Drizzle Kit | 自定义 Go migrator |
| **执行** | `drizzle-kit push` | `./migrate` |
| **回滚** | 不支持 | 支持 |
| **版本控制** | `_journal.json` | `migrations` 表 |

### 需要同步的变更

当前 Drizzle schema 与 Go migrations 的差异：

| 字段/表 | Drizzle | Go Migration | 状态 |
|---------|---------|--------------|------|
| `users.two_factor_last_used_at` | ❌ 缺失 | ✅ 005 | ⚠️ 不同步 |
| `users.updated_at` | ❌ 缺失 | ✅ 001 | ⚠️ 不同步 |
| `wallet_creation_request.product_code` | ❌ 缺失 | ✅ 007 | ⚠️ 不同步 |
| `wallet_creation_request.currency` | ❌ 缺失 | ✅ 007 | ⚠️ 不同步 |
| `pending_login_sessions` 表 | ✅ 存在 | ❌ 已删除 | ⚠️ 不同步 |

---

## 建议

### 1. 同步策略
建议统一使用 **Go Migration** 作为生产环境的唯一 migration 工具：
- 修改 `src/db/schema.ts` 添加缺失字段
- 重新生成 Drizzle migration
- 或者：直接使用 Go migrator 执行所有迁移

### 2. 更新 Drizzle Schema
```typescript
// src/db/schema.ts
export const users = pgTable('users', {
  // ... 现有字段
  twoFactorLastUsedAt: bigint('two_factor_last_used_at', { mode: 'number' }).default(0),
  updatedAt: timestamp('updated_at').defaultNow().notNull(),
});

export const walletCreationRequest = pgTable('wallet_creation_request', {
  // ... 现有字段
  productCode: varchar('product_code', { length: 50 }).default(''),
  currency: varchar('currency', { length: 20 }).default(''),
});
```

### 3. 重新生成
```bash
npx drizzle-kit generate
```

---

## 总结

`0000_black_human_fly.sql` 是 Drizzle ORM 生成的初始数据库 Schema，用于：
1. **前端开发** - 快速创建本地数据库
2. **类型生成** - Drizzle 根据此生成 TypeScript 类型
3. **开发环境** - 与 Go Migration 并行使用

**生产环境应使用 Go Migration 系统**，因为它支持：
- 增量迁移
- 回滚操作
- 版本控制
- 多环境部署
