# 架构统一：移除 Drizzle，禁止前端直接访问数据库

## 背景

当前项目存在两套数据库访问方式：
1. **前端/Drizzle** - 通过 `src/db/schema.ts` 和 `drizzle-orm` 直接访问数据库
2. **后端/Go** - 通过 `internal/migration/` 和 `internal/repository/` 访问数据库

这违反了单一职责原则，造成维护困难。

## 目标

统一架构：
- ✅ 删除 Drizzle 相关文件和依赖
- ✅ 禁止前端直接访问数据库
- ✅ 所有数据库操作通过 Go 后端 API
- ✅ 前端只负责 UI 展示和 API 调用

## 架构变更

### 变更前（混合架构）

```
┌─────────────────────────────────────────┐
│              Frontend (React)            │
│  ┌─────────────────────────────────┐    │
│  │  src/lib/withdrawal-service.ts  │────┼──► Drizzle ORM ──► Database
│  │  (直接操作数据库)                │    │
│  └─────────────────────────────────┘    │
│                    │                     │
│                    ▼                     │
│  ┌─────────────────────────────────┐    │
│  │  API Calls (/api/*)             │────┼──► Go Backend ──► Database
│  └─────────────────────────────────┘    │
└─────────────────────────────────────────┘
```

### 变更后（统一架构）

```
┌─────────────────────────────────────────┐
│              Frontend (React)            │
│  ┌─────────────────────────────────┐    │
│  │  API Calls Only (/api/*)        │────┼──► Go Backend ──► Database
│  │  禁止直接数据库访问              │    │
│  └─────────────────────────────────┘    │
└─────────────────────────────────────────┘
```

## 实施计划

### Phase 1: 删除 Drizzle 文件

| 文件/目录 | 操作 | 说明 |
|-----------|------|------|
| `drizzle/0000_black_human_fly.sql` | 删除 | 自动生成文件 |
| `drizzle/meta/` | 删除 | Drizzle 元数据 |
| `src/db/schema.ts` | 删除 | Drizzle schema 定义 |
| `src/db/index.ts` | 删除 | Drizzle 数据库连接 |
| `drizzle.config.ts` | 删除 | Drizzle 配置 |
| `drizzle-orm` | npm uninstall | 依赖包 |
| `drizzle-kit` | npm uninstall | 依赖包 |

### Phase 2: 移除前端数据库访问代码

| 文件 | 变更 |
|------|------|
| `src/lib/withdrawal-service.ts` | 删除 Drizzle 操作，改为 API 调用 |
| `src/lib/db.ts` | 删除（如果存在） |

### Phase 3: 更新前端代码

所有数据库操作改为调用 Go 后端 API：

```typescript
// 变更前（直接操作数据库）
import { db } from '@/db';
import { withdrawals } from '@/db/schema';

const result = await db.insert(withdrawals).values({...});

// 变更后（API 调用）
const response = await fetch('/api/withdrawals', {
  method: 'POST',
  body: JSON.stringify({...})
});
```

### Phase 4: 更新文档

- 更新 `AGENTS.md` 架构说明
- 更新 `CLAUDE.md` 开发规范
- 删除 Drizzle 相关文档

## 代码变更详情

### 1. 删除文件列表

```bash
# Drizzle 文件
rm -rf drizzle/0000_black_human_fly.sql
rm -rf drizzle/meta/
rm -f drizzle.config.ts

# 前端数据库代码
rm -rf src/db/

# 依赖
npm uninstall drizzle-orm drizzle-kit
```

### 2. 修改文件列表

| 文件 | 修改内容 |
|------|---------|
| `package.json` | 移除 `drizzle-orm`, `drizzle-kit` |
| `src/lib/withdrawal-service.ts` | 删除 Drizzle 导入和数据库操作 |
| `AGENTS.md` | 更新架构说明 |
| `CLAUDE.md` | 更新开发规范 |

### 3. 保留的文件

| 文件 | 原因 |
|------|------|
| `drizzle/monera_complete_schema.sql` | 作为数据库文档保留 |

## 设计原则验证

### KISS (Keep It Simple, Stupid)
- ✅ 单一数据库访问层（Go 后端）
- ✅ 前端只负责 UI 和 API 调用
- ✅ 移除重复的 ORM 工具

### 高内聚低耦合
- ✅ 数据库操作集中在 Go 后端
- ✅ 前端与数据库解耦
- ✅ 通过 API 契约通信

### 单一职责
- ✅ Go 后端：业务逻辑 + 数据库操作
- ✅ 前端：用户界面 + API 调用

## 测试策略

1. **构建测试** - 确保前端能正常构建
2. **API 测试** - 验证所有 API 调用正常
3. **集成测试** - 验证前后端联调正常
4. **回归测试** - 确保不影响其他功能

## 回滚方案

如果需要回滚：
1. 从 git 历史恢复删除的文件
2. 重新安装 npm 依赖
3. 恢复修改的文件

## 影响范围

| 模块 | 影响 | 说明 |
|------|------|------|
| 前端数据库操作 | 删除 | 改为 API 调用 |
| Go 后端 | 无变化 | 本来就是 API 方式 |
| 数据库 | 无变化 | 只是访问方式改变 |
| 构建流程 | 简化 | 移除 Drizzle 构建步骤 |

## 验收标准

- [ ] 所有 Drizzle 文件已删除
- [ ] 前端代码无直接数据库访问
- [ ] 所有数据库操作通过 Go API
- [ ] 前端构建成功
- [ ] 所有测试通过
- [ ] 文档已更新
