# Drizzle Migration 文件说明

## 0000_black_human_fly.sql 是自动生成的吗？

**是的**，这个文件是由 Drizzle Kit 自动生成的。

### 生成方式

```bash
# 方式1: 生成迁移文件（不执行）
npx drizzle-kit generate

# 方式2: 生成并推送到数据库
npx drizzle-kit push
```

### 生成流程

```
src/db/schema.ts ──► drizzle-kit generate ──► drizzle/0000_black_human_fly.sql
                                              drizzle/meta/0000_snapshot.json
                                              drizzle/meta/_journal.json
```

---

## 可以删除吗？

### 简短回答

**可以删除，但有条件**：
- ✅ 如果项目**不使用 Drizzle ORM** 进行数据库操作
- ✅ 如果**只用 Go Migration** 管理数据库
- ❌ 如果前端/Node.js 代码使用 Drizzle ORM 操作数据库

---

## 为什么不能随意删除？

### 1. Drizzle ORM 依赖这些文件

如果前端代码使用 Drizzle ORM：

```typescript
// 例如：src/db/schema.ts
import { drizzle } from 'drizzle-orm/node-postgres';
import * as schema from './schema';

const db = drizzle(client, { schema });

// 查询用户
const users = await db.select().from(schema.users);
```

**Drizzle Kit 需要这些文件来**：
- 生成 TypeScript 类型
- 追踪 schema 变更
- 生成新的迁移

### 2. 文件关联性

```
drizzle/
├── 0000_black_human_fly.sql          # 迁移 SQL
├── meta/
│   ├── 0000_snapshot.json            # Schema 快照（关联！）
│   └── _journal.json                 # 迁移日志（关联！）
└── monera_complete_schema.sql        # 手动维护（独立）
```

**删除 0000_black_human_fly.sql 会导致**：
- `_journal.json` 记录的迁移丢失
- `0000_snapshot.json` 与实际不符
- 下次 `drizzle-kit generate` 可能出错

### 3. 前端开发依赖

如果团队使用 Drizzle 进行开发：

```bash
# 新成员克隆项目后
npm install
npx drizzle-kit push     # 需要 0000_black_human_fly.sql
npm run dev
```

---

## 正确的处理方式

### 方案一：完全弃用 Drizzle（推荐如果只用 Go）

如果项目**只用 Go 后端**，不使用 Drizzle ORM：

```bash
# 1. 删除 Drizzle 相关文件
rm -rf drizzle/
rm drizzle.config.ts
rm src/db/schema.ts

# 2. 删除 package.json 中的依赖
npm uninstall drizzle-orm drizzle-kit

# 3. 只用 Go Migration
go build -o migrate ./cmd/migrate
./migrate
```

### 方案二：保留 Drizzle 用于开发

保留 Drizzle，但明确分工：

| 环境 | 工具 | 文件 |
|------|------|------|
| 开发环境 | Drizzle Kit | `drizzle/0000_*.sql` |
| 生产环境 | Go Migration | `internal/migration/` |
| 文档参考 | 手动维护 | `drizzle/monera_complete_schema.sql` |

### 方案三：重新生成（如果已删除）

如果不小心删除了，可以重新生成：

```bash
# 1. 确保 src/db/schema.ts 存在
# 2. 重新生成
npx drizzle-kit generate

# 这会创建新的迁移文件（编号可能不同）
# 例如：0001_xxxxxx.sql
```

---

## 当前项目情况分析

### 检查是否使用 Drizzle

```bash
# 检查 package.json
grep -E "drizzle" package.json

# 检查前端代码
grep -r "from 'drizzle-orm'" src/
grep -r "from '@/db/schema'" src/
```

### 检查结果

```bash
$ grep -r "drizzle-orm" src/
src/db/schema.ts:import { pgTable, ... } from 'drizzle-orm/pg-core';
src/db/index.ts:import { drizzle } from 'drizzle-orm/node-postgres';

$ grep -r "@/db/schema" src/
src/lib/withdrawal-service.ts:import { withdrawals, withdrawalAddresses } from '../db/schema.js';
```

**结论**: 项目确实使用 Drizzle ORM！

### 使用 Drizzle 的文件

| 文件 | 用途 |
|------|------|
| `src/db/schema.ts` | Schema 定义 |
| `src/db/index.ts` | 数据库连接 |
| `src/lib/withdrawal-service.ts` | 使用 Drizzle 查询 |

---

## 建议操作

### 不要删除！

因为：
1. ✅ 前端代码使用 Drizzle ORM
2. ✅ `src/lib/withdrawal-service.ts` 依赖 `src/db/schema.ts`
3. ✅ `src/db/schema.ts` 与 `drizzle/` 迁移文件关联

### 正确的维护方式

```bash
# 1. 保持文件完整
# 不要删除 drizzle/0000_black_human_fly.sql

# 2. 如果需要更新，修改 src/db/schema.ts
# 然后重新生成
npx drizzle-kit generate

# 3. 生产环境使用 Go Migration
./migrate
```

### 文件分工

| 文件 | 用途 | 维护方式 |
|------|------|---------|
| `drizzle/0000_black_human_fly.sql` | Drizzle 迁移 | 自动生成，勿手动修改 |
| `drizzle/monera_complete_schema.sql` | 完整文档 | 手动维护 |
| `internal/migration/migrations/*.go` | 生产迁移 | Go 代码 |

---

## 总结

| 问题 | 答案 |
|------|------|
| 是自动生成的吗？ | ✅ 是的 |
| 可以删除吗？ | ❌ 不建议（项目使用 Drizzle） |
| 如何更新？ | 修改 `src/db/schema.ts` 后重新生成 |
| 生产环境用哪个？ | Go Migrations |

**最终建议**: 保留 `0000_black_human_fly.sql`，它是 Drizzle ORM 工作流的一部分。
