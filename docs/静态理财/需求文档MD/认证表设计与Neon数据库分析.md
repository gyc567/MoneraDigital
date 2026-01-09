# 用户认证表设计与 Neon 数据库分析报告

**分析日期**: 2026-01-09
**分析对象**: MoneraDigital 项目认证模块
**数据库**: Neon PostgreSQL

---

## 一、核心认证表结构

### 1.1 主用户表：`users` (Drizzle ORM)

**文件位置**: `src/db/schema.ts`

```typescript
export const users = pgTable('users', {
  id: serial('id').primaryKey(),                              // 用户ID (自增主键)
  email: text('email').notNull().unique(),                    // 邮箱 (唯一约束)
  password: text('password').notNull(),                       // 密码哈希 (bcryptjs, 10轮盐)
  twoFactorSecret: text('two_factor_secret'),                 // 2FA密钥 (TOTP, 加密存储)
  twoFactorEnabled: boolean('two_factor_enabled')
    .default(false).notNull(),                                // 2FA启用标志
  twoFactorBackupCodes: text('two_factor_backup_codes'),      // 备份码 (加密JSON数组)
  createdAt: timestamp('created_at').defaultNow().notNull(),  // 注册时间
});
```

**表的作用**: 存储用户基本信息和身份验证凭证

| 字段名 | 数据类型 | 约束 | 说明 |
|--------|---------|------|------|
| `id` | SERIAL | PRIMARY KEY | 自增整数主键 |
| `email` | TEXT | NOT NULL, UNIQUE | 用户邮箱（登陆用户名） |
| `password` | TEXT | NOT NULL | bcryptjs 哈希密码 |
| `twoFactorSecret` | TEXT | NULL | TOTP 密钥（加密） |
| `twoFactorEnabled` | BOOLEAN | DEFAULT false | 是否启用2FA |
| `twoFactorBackupCodes` | TEXT | NULL | 10个恢复码（加密JSON） |
| `createdAt` | TIMESTAMP | DEFAULT NOW() | 账户创建时间 |

---

## 二、注册和登陆流程使用的表

### 2.1 用户注册流程

```
用户提交注册表单 (email, password)
    ↓
validate_schema (Zod 验证)
    ├─ email 格式检查
    └─ password 最少6个字符
    ↓
bcryptjs.hash(password, 10) → 生成哈希
    ↓
INSERT INTO users (email, password, created_at)
    ↓
PostgreSQL 唯一约束检查 (email UNIQUE)
    ├─ 如果重复 → Error: "User already exists" (code 23505)
    └─ 如果成功 → RETURNING { id, email }
```

**涉及表**: `users` 表

**关键操作**:
```typescript
// src/lib/auth-service.ts: register 方法
const hashedPassword = await bcrypt.hash(password, 10);

const [user] = await db.insert(users).values({
  email: validated.email,
  password: hashedPassword,
}).returning({
  id: users.id,
  email: users.email,
});
```

**错误处理**:
- PostgreSQL 错误代码 23505 (UNIQUE constraint violation) → 邮箱已存在
- 其他异常 → 注册失败

---

### 2.2 用户登陆流程

```
用户提交登陆表单 (email, password)
    ↓
validate_schema (Zod 验证)
    ↓
SELECT * FROM users WHERE email = ?
    ├─ 查询不到 → Error: "Invalid email or password"
    └─ 查询到用户继续
    ↓
bcryptjs.compare(password, user.password)
    ├─ 密码不匹配 → Error: "Invalid email or password"
    └─ 密码匹配继续
    ↓
检查 user.twoFactorEnabled 标志
    ├─ 如果 = true → 返回 { requires2FA: true, userId }
    │                (跳转到2FA验证)
    └─ 如果 = false → 继续
    ↓
jwt.sign({ userId, email }, JWT_SECRET, { expiresIn: '24h' })
    ↓
返回 { user: { id, email }, token }
```

**涉及表**: `users` 表

**关键操作**:
```typescript
// src/lib/auth-service.ts: login 方法
const [user] = await db.select().from(users)
  .where(eq(users.email, validated.email));

const isValid = await bcrypt.compare(password, user.password);

if (user.twoFactorEnabled) {
  return { requires2FA: true, userId: user.id };
}

const token = jwt.sign({ userId: user.id, email: user.email },
                        JWT_SECRET,
                        { expiresIn: '24h' });
```

**返回值**:
- 成功（无2FA）: `{ user: { id, email }, token }`
- 成功（需2FA）: `{ requires2FA: true, userId }`
- 失败: 抛出错误

---

### 2.3 2FA 验证与二次登陆流程

```
用户输入2FA码 (TOTP 6位数字)
    ↓
SELECT * FROM users WHERE id = ?
    ↓
otplib.authenticator.check(token, user.twoFactorSecret)
    ├─ 验证失败 → Error: "Invalid verification code"
    └─ 验证成功继续
    ↓
jwt.sign({ userId, email }, JWT_SECRET, { expiresIn: '24h' })
    ↓
返回 { user: { id, email }, token }
```

**涉及表**: `users` 表 (读取 `twoFactorSecret`)

**关键操作**:
```typescript
// src/lib/auth-service.ts: verify2FAAndLogin 方法
const [user] = await db.select().from(users)
  .where(eq(users.id, userId));

const { authenticator } = await import('otplib');
const isValid = authenticator.check(token, user.twoFactorSecret);

if (!isValid) {
  throw new Error('Invalid verification code');
}

const jwtToken = jwt.sign({ userId: user.id, email: user.email },
                           JWT_SECRET,
                           { expiresIn: '24h' });
```

---

## 三、相关数据库表

除了 `users` 表，系统还定义了以下相关表（来自 `src/db/schema.ts`）:

### 3.1 借贷相关表

```typescript
export const lendingPositions = pgTable('lending_positions', {
  id: serial('id').primaryKey(),
  userId: integer('user_id').references(() => users.id).notNull(),
  asset: text('asset').notNull(),
  amount: numeric('amount', { precision: 20, scale: 8 }).notNull(),
  durationDays: integer('duration_days').notNull(),
  apy: numeric('apy', { precision: 5, scale: 2 }).notNull(),
  status: lendingStatusEnum('status').default('ACTIVE').notNull(),
  accruedYield: numeric('accrued_yield', { precision: 20, scale: 8 }),
  startDate: timestamp('start_date').defaultNow().notNull(),
  endDate: timestamp('end_date').notNull(),
});
```

**目的**: 记录用户的理财产品申购订单

---

### 3.2 提现地址相关表

```typescript
export const withdrawalAddresses = pgTable('withdrawal_addresses', {
  id: serial('id').primaryKey(),
  userId: integer('user_id').references(() => users.id).notNull(),
  address: text('address').notNull(),
  addressType: addressTypeEnum('address_type').notNull(),  // BTC/ETH/USDC/USDT
  label: text('label').notNull(),
  isVerified: boolean('is_verified').default(false).notNull(),
  isPrimary: boolean('is_primary').default(false).notNull(),
  createdAt: timestamp('created_at').defaultNow().notNull(),
  verifiedAt: timestamp('verified_at'),
  deactivatedAt: timestamp('deactivated_at'),
});

export const addressVerifications = pgTable('address_verifications', {
  id: serial('id').primaryKey(),
  addressId: integer('address_id').references(() => withdrawalAddresses.id),
  token: text('token').notNull().unique(),
  expiresAt: timestamp('expires_at').notNull(),
  verifiedAt: timestamp('verified_at'),
});
```

**目的**: 用户管理和验证提现地址（支持多条公链）

---

### 3.3 提现交易表

```typescript
export const withdrawals = pgTable('withdrawals', {
  id: serial('id').primaryKey(),
  userId: integer('user_id').references(() => users.id).notNull(),
  fromAddressId: integer('from_address_id')
    .references(() => withdrawalAddresses.id).notNull(),
  amount: numeric('amount', { precision: 20, scale: 8 }).notNull(),
  asset: text('asset').notNull(),
  toAddress: text('to_address').notNull(),
  status: withdrawalStatusEnum('status').default('PENDING').notNull(),
  txHash: text('tx_hash'),
  createdAt: timestamp('created_at').defaultNow().notNull(),
  completedAt: timestamp('completed_at'),
  failureReason: text('failure_reason'),
});
```

**目的**: 跟踪用户提现订单和区块链交易状态

---

## 四、Neon 云数据库配置

### 4.1 数据库连接信息

**文件位置**: `.env`

```env
DATABASE_URL="postgresql://neondb_owner:npg_4zuq7JQNWFDB@ep-bold-cloud-adfpuk12-pooler.c-2.us-east-1.aws.neon.tech/neondb?sslmode=require"
```

**配置解析**:

| 配置项 | 值 | 说明 |
|--------|-----|------|
| **数据库系统** | PostgreSQL | 关系型数据库 |
| **服务商** | Neon (AWS) | 无服务器 PostgreSQL |
| **用户** | neondb_owner | 数据库所有者 |
| **密码** | npg_4zuq7JQNWFDB | 连接密码 |
| **主机** | ep-bold-cloud-adfpuk12-pooler.c-2.us-east-1.aws.neon.tech | Pooler 连接端点 |
| **数据库名** | neondb | 项目数据库 |
| **地区** | us-east-1 | AWS 美东地区 |
| **SSL 模式** | require | 必须使用 SSL 加密连接 |

### 4.2 数据库连接配置

**文件位置**: `src/lib/db.ts`

```typescript
const connectionString = process.env.DATABASE_URL;

export const client = postgres(connectionString || '', {
  ssl: 'require',           // SSL 加密
  max: 1                    // 最多1个连接（资源受限环境）
});

export const db = drizzle(client, { schema });
```

**特点**:
- 使用 `postgres.js` 驱动（轻量级）
- SSL 必需（生产环境安全要求）
- 单连接池（Neon Serverless 特性）

---

## 五、Drizzle ORM 配置

### 5.1 Drizzle 配置文件

**文件位置**: `drizzle.config.ts`

```typescript
export default defineConfig({
  schema: './src/db/schema.ts',          // 模式定义位置
  out: './drizzle',                      // 迁移文件输出目录
  dialect: 'postgresql',                 // 数据库方言
  dbCredentials: {
    url: process.env.DATABASE_URL!,      // 连接字符串
  },
});
```

### 5.2 ORM 类型安全

Drizzle 提供完整的 TypeScript 类型推导：

```typescript
// 类型自动推导
export type User = typeof users.$inferSelect;    // 查询结果类型
export type NewUser = typeof users.$inferInsert; // 插入输入类型
```

---

## 六、认证相关的 API 端点

### 6.1 API 列表

**文件位置**: `api/auth/`

| 端点 | 方法 | 功能 | 实现文件 |
|------|------|------|--------|
| `/api/auth/register` | POST | 用户注册 | `api/auth/register.ts` |
| `/api/auth/login` | POST | 用户登陆 | `api/auth/login.ts` |
| `/api/auth/me` | GET | 获取当前用户信息 | `api/auth/me.ts` |
| `/api/auth/2fa/setup` | POST | 设置2FA | `api/auth/2fa/setup.ts` |
| `/api/auth/2fa/enable` | POST | 启用2FA | `api/auth/2fa/enable.ts` |
| `/api/auth/2fa/verify-login` | POST | 2FA登陆验证 | `api/auth/2fa/verify-login.ts` |

### 6.2 注册端点实现

```typescript
// POST /api/auth/register
Handler: async function handler(req: VercelRequest, res: VercelResponse) {
  // 1. Rate limiting: 5 req/60s per IP
  const isAllowed = await rateLimit(ip, 5, 60000);

  // 2. Extract email & password
  const { email, password } = req.body;

  // 3. Call AuthService.register
  const user = await AuthService.register(email, password);

  // 4. Return 201 Created
  return res.status(201).json({ message: 'User created successfully', user });
}
```

**返回示例**:
```json
{
  "message": "User created successfully",
  "user": {
    "id": 42,
    "email": "user@example.com"
  }
}
```

### 6.3 登陆端点实现

```typescript
// POST /api/auth/login
Handler: async function handler(req: VercelRequest, res: VercelResponse) {
  // 1. Rate limiting: 5 req/60s per IP
  const isAllowed = await rateLimit(ip, 5, 60000);

  // 2. Extract email & password
  const { email, password } = req.body;

  // 3. Call AuthService.login
  const result = await AuthService.login(email, password);

  // 4. Return 200 OK with token or 2FA requirement
  return res.status(200).json(result);
}
```

**返回示例 (无2FA)**:
```json
{
  "user": {
    "id": 42,
    "email": "user@example.com"
  },
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**返回示例 (需2FA)**:
```json
{
  "requires2FA": true,
  "userId": 42
}
```

---

## 七、安全性实现

### 7.1 密码安全

| 安全措施 | 实现方式 | 强度 |
|---------|---------|------|
| **密码哈希** | bcryptjs (10 salt rounds) | 强 🟢 |
| **密码验证** | bcryptjs.compare() | 强 🟢 |
| **最短密码** | 6 个字符 | 中等 🟡 |

### 7.2 用户枚举防护

```typescript
// 登陆失败时，返回通用错误，不透露用户是否存在
throw new Error('Invalid email or password'); // 相同的消息
```

### 7.3 2FA 安全

- **TOTP 算法**: otplib (基于 RFC 6238)
- **备份码**: 10 个加密恢复码
- **密钥存储**: AES-256 加密

### 7.4 Token 安全

| 安全特性 | 配置 |
|---------|------|
| **算法** | HS256 |
| **密钥长度** | 64 字符 hex |
| **过期时间** | 24 小时 |
| **传输** | Bearer token in Authorization header |

### 7.5 速率限制

```typescript
// src/lib/rate-limit.ts
// 限制: 5 requests per 60 seconds per IP
// 后端: Redis (如可用) 或 in-memory Map

await rateLimit(ip, 5, 60000); // 返回 boolean
```

---

## 八、数据库表汇总（Neon 中的所有表）

### 8.1 现有表（TypeScript Drizzle 定义）

| 表名 | 目的 | 用户表关联 |
|------|------|----------|
| **users** | 用户认证和基本信息 | PK: id |
| **lendingPositions** | 理财产品申购订单 | FK: user_id → users.id |
| **withdrawalAddresses** | 提现地址白名单 | FK: user_id → users.id |
| **addressVerifications** | 地址验证令牌 | FK: address_id → withdrawalAddresses.id |
| **withdrawals** | 提现交易历史 | FK: user_id → users.id, from_address_id |

### 8.2 待实现表（来自 SQL 建表脚本）

基于 `docs/静态理财/需求文档MD/数据库建表脚本.sql`，以下表需要创建或迁移到 Neon:

**资产账户域**:
- `account` - 用户账户 (FUND/WEALTH)
- `account_journal` - 资金流水（不可变）

**理财业务域**:
- `wealth_product` - 理财产品配置
- `wealth_order` - 理财订单
- `wealth_interest_record` - 每日计息/发放记录

**幂等性与防护**:
- `idempotency_record` - 幂等性记录
- `wallet_creation_request` - Safeheron 钱包创建
- `transfer_record` - 划转记录
- `withdrawal_address_whitelist` - 提币地址白名单（重定义）
- `withdrawal_order` - 提现订单（扩展）
- `withdrawal_freeze_log` - 冻结/解冻日志

**权限与审核**:
- `wealth_product_approval` - 产品审核工作流
- `account_adjustment` - 账户调账
- `audit_trail` - 审计日志

**对账与监控**:
- `reconciliation_log` - 对账日志
- `reconciliation_alert_log` - 告警日志
- `reconciliation_error_log` - 错误日志
- `manual_review_queue` - 人工审查队列
- `business_freeze_status` - 业务冻结状态

---

## 九、数据库迁移工具

### 9.1 Go 迁移运行器

**文件位置**: `cmd/db_migration/main.go`

**功能**:
- 读取 `DATABASE_URL` 环境变量
- 执行 SQL 迁移脚本（原子事务）
- 连接 Neon PostgreSQL
- 返回执行结果

**使用方式**:
```bash
go run cmd/db_migration/main.go
# 或编译后运行
./db_migration
```

### 9.2 SQL 脚本位置

**主脚本**: `docs/静态理财/需求文档MD/数据库建表脚本.sql`

**内容**:
- 22 个核心表定义
- 5 个便利视图
- 初始化数据 (系统账户, 业务状态)
- 完整的索引和约束

---

## 十、现状总结

### ✅ 已实现

| 功能 | 位置 | 状态 |
|------|------|------|
| 用户注册 | `api/auth/register.ts` | ✅ 完成 |
| 用户登陆 | `api/auth/login.ts` | ✅ 完成 |
| JWT 认证 | `src/lib/auth-service.ts` | ✅ 完成 |
| 2FA 设置 | `api/auth/2fa/setup.ts` | ✅ 完成 |
| 速率限制 | `src/lib/rate-limit.ts` | ✅ 完成 |
| Drizzle ORM | `src/lib/db.ts`, `src/db/schema.ts` | ✅ 完成 |
| Neon 连接 | `.env` + `src/lib/db.ts` | ✅ 完成 |

### ⏳ 待实现

| 功能 | 优先级 | 说明 |
|------|--------|------|
| 理财账户系统 | P0 | `account`, `wealth_order` 等表 |
| 提现功能完整实现 | P0 | 冻结机制, 自动解冻, 对账 |
| 权限和审核工作流 | P1 | RBAC, 三级审批 |
| 对账和监控 | P1 | 定时任务, 告警系统 |
| Go 后端重写 | P2 | 当前仅框架，服务未实现 |

---

## 十一、建议行动计划

### 立即行动（第1天）

1. **验证 Neon 数据库连接**
   ```bash
   npm test  # 运行 auth-service.test.ts
   ```

2. **检查数据库迁移工具**
   ```bash
   cd cmd/db_migration && go run main.go
   ```

3. **初始化 TypeScript 表到 Neon**
   ```bash
   npx drizzle-kit push:pg
   ```

### 第2-3天：实现理财系统

1. 在 `src/db/schema.ts` 中定义新表
   - `account`, `account_journal`
   - `wealth_product`, `wealth_order`, `wealth_interest_record`

2. 创建 Drizzle 迁移
   ```bash
   npx drizzle-kit generate:pg
   ```

3. 推送到 Neon
   ```bash
   npx drizzle-kit push:pg
   ```

### 第4-5天：实现提现和对账

1. 迁移 SQL 建表脚本中的所有表到 Drizzle 定义
2. 实现定时任务 (冻结自动解冻, 每日对账)
3. 编写测试用例验证幂等性

---

## 附录 A：表关系图

```
users (认证核心)
  ├─ lendingPositions (一对多)
  ├─ withdrawalAddresses (一对多)
  │   └─ addressVerifications (一对多)
  ├─ withdrawals (一对多)
  │   └─ (future) withdrawal_freeze_log
  │
  ├─ (future) account (一对多, FUND + WEALTH)
  │   └─ account_journal (不可变流水)
  │
  ├─ (future) wealth_order (一对多, 理财订单)
  │   ├─ wealth_product (多对一, 产品配置)
  │   └─ wealth_interest_record (一对多, 计息记录)
  │
  └─ (future) audit_trail, reconciliation_* (审计和对账)
```

---

## 附录 B：技术栈版本

```json
{
  "Database": "PostgreSQL (Neon Serverless)",
  "ORM": "drizzle-orm",
  "Driver": "postgres ^3.4.7",
  "PasswordHashing": "bcryptjs ^2.4.3",
  "JWT": "jsonwebtoken ^9.0.3",
  "2FA": "otplib ^12.0.1",
  "QRCode": "qrcode ^1.5.4",
  "Validation": "zod",
  "Logger": "pino ^10.1.0"
}
```

---

**报告完成**: 2026-01-09
**文档位置**: `/Users/eric/dreame/code/MoneraDigital/docs/静态理财/需求文档MD/认证表设计与Neon数据库分析.md`
