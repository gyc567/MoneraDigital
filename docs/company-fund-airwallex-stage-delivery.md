# Airwallex company-fund — Stage 交付与 Phase 2 验收清单

> 状态：Phase 2 待上线版本 `790fc2b` 已于 2026-07-29 通过本机门禁和 Stage
> 验收。Stage 装配 `airwallex_reconciliation=true airwallex_webhook=true
> airwallex_ledger=true`；两次 Sandbox REST 重放均 `seen=42 created=0`，
> 数据保持 `facts=19 txns=19`。REVERSAL 样本仍没有 exact target evidence，
> 因此继续 fail closed，不是已支持的自动入账类型。
>
> 目标环境：测试环境（stage 分支 → `13.159.187.219` /
> `test-mdapi.moneradigital.com`）。
> 本清单不覆盖 production（#35 仍 OPEN，未经明确授权不得合 main）。

## 1. 功能边界（交付口径）

| 路径 | 作用 | 会不会直接入账 |
| --- | --- | --- |
| REST Financial Transactions | 权威事实源：snapshot → encrypted inbox → worker → facts/txns | 是（仅 `FINANCIAL_TRANSACTION_SNAPSHOT` + 命中 runtime rule） |
| Webhook `POST /api/webhooks/airwallex` | HMAC 验签 → 加密入库 → wake REST lookback | **否**。非 snapshot envelope 一律 `IGNORED`（审计 + 唤醒） |
| 映射规则 | 严格 JSON `AIRWALLEX_FINANCIAL_TRANSACTIONS_RUNTIME_CONFIG` | exact account/type/source/currency/status；复杂类型还必须有 evidence-pinned relationship |
| Phase 2 task | PostgreSQL durable lease/retry/SLA | 关系未到时只存 fact/task；关系满足后才登记 movement |

单账户 fail-closed：共享数据库中必须恰好有一条 `CURRENT`（即 enabled）
`AIRWALLEX` 账户。REST 每次调用从 Account Registry 取得该账户并创建独立
`x-login-as` scope；切换或纠错命令完成并刷新 Registry 后，无需修改环境变量或
重启服务。`AIRWALLEX_LOGIN_AS` 仅保留为旧部署兼容项，不再决定 Company Fund
采集账户。

## 2. 本地交付门禁（开发机）

前置：

- 仓库根目录 `.env` 含 sandbox 四件套（smoke 会 `godotenv.Overload(".env")`，以 `.env` 为准）：`AIRWALLEX_BASE_URL` / `CLIENT_ID` / `API_KEY` / `LOGIN_AS`
- `DATABASE_URL` 指向 monera_local（`MONERA_DATABASE_URL` 若设置则优先覆盖）
- `secrets/airwallex-sandbox-runtime.json`（enabled + 与 LOGIN_AS 对齐的 ADJUSTMENT 规则）
- `secrets/company-fund-payload.key`（64 hex AES key）
- monera_local 中恰好一条 enabled AIRWALLEX `company_fund_accounts`，`provider_account_key = LOGIN_AS`

```bash
# 1) 只读：login + list + 内存 normalize
go run ./cmd/airwallex_sandbox_smoke

# 2) REST 入账路径
go run ./cmd/airwallex_sandbox_smoke -persist

# 3) Webhook 验签入库 + worker IGNORED + 证明不改 ledger
go run ./cmd/airwallex_sandbox_smoke -webhook

# 4) 全门禁
go run ./cmd/airwallex_sandbox_smoke -persist -webhook

# 5) 脱敏 evidence 结构（只读，不输出 Provider ID/payload/凭据）
go run ./cmd/airwallex_sandbox_smoke -evidence

# 6) migration 与真实 PostgreSQL Phase 2 集成测试
go run ./cmd/migrate -dry-run
RUN_COMPANY_FUND_AIRWALLEX_PHASE2_INTEGRATION=1 \
  go test ./internal/companyfund \
  -run 'TestAirwallexPhase2.*PostgresIntegration' -count=1 -v
```

期望：

- dry-run：`normalize summary` 无 UNKNOWN disposition
- `-persist`：`reconcile ok` + worker drain + `facts/txns > 0`（sandbox 有 SETTLED ADJUSTMENT 时）
- `-webhook`：handler 200、replay 200、event 恰好 1 行、`event_state=IGNORED`、facts/txns 计数不变
- 相关单测：

```bash
go test ./internal/companyfund/ -count=1 \
  -run 'Airwallex|FinalizeCompanyFundSyncRun|OwnedProviderPayload|ProviderEventWorker_ProcessNextIgnores'
go test ./internal/handlers/ -count=1 -run 'CompanyFundAirwallexWebhook'
go test ./... -count=1
go vet ./...
```

## 3. Stage 环境变量（与 `.env.example` 对齐）

在 stage 服务器 `/opt/monera-digital/.env`（systemd EnvironmentFile；历史路径 `/home/ec2-user/monera` 已废弃）写入：

```bash
COMPANY_FUND_ENABLED=true
COMPANY_FUND_START_BACKGROUND_WORKERS=true
COMPANY_FUND_PAYLOAD_KEY=<64-hex>
COMPANY_FUND_PAYLOAD_KEY_VERSION=payload-v1
COMPANY_FUND_PAYLOAD_RETENTION=48h
COMPANY_FUND_PAYLOAD_LEGAL_HOLD=false

# Stage 若继续打 sandbox，BASE_URL 用 sandbox；切正式网前必须换 credential + runtime rules + 账户行
AIRWALLEX_BASE_URL=https://api.sandbox.airwallex.com
AIRWALLEX_CLIENT_ID=<stage>
AIRWALLEX_API_KEY=<stage>
# 旧部署兼容项；新账户范围由数据库 CURRENT 状态决定
AIRWALLEX_LOGIN_AS=
AIRWALLEX_API_VERSION=2026-07-17

AIRWALLEX_WEBHOOK_VERSION=event-v1
AIRWALLEX_WEBHOOK_SECRET=<console webhook secret>
AIRWALLEX_WEBHOOK_TIMESTAMP_TOLERANCE=5m
COMPANY_FUND_AIRWALLEX_WEBHOOK_LOOKBACK=24h
AIRWALLEX_FINANCIAL_TRANSACTIONS_PAGE_SIZE=100
AIRWALLEX_FINANCIAL_TRANSACTIONS_MAX_PAGES=100

# Phase 2 durable task + finance policy（code 必须对应已存在且 enabled 的分类）
COMPANY_FUND_AIRWALLEX_LEDGER_TASK_LEASE_OWNER=
COMPANY_FUND_AIRWALLEX_LEDGER_TASK_LEASE_DURATION=1m
COMPANY_FUND_AIRWALLEX_LEDGER_TASK_RETRY_DELAY=1m
COMPANY_FUND_AIRWALLEX_LEDGER_TASK_SLA=168h
COMPANY_FUND_AIRWALLEX_LEDGER_MAINTENANCE_BATCH=100
COMPANY_FUND_AIRWALLEX_FEE_CATEGORY_LEVEL1_CODE=<stable-level-1-code>
COMPANY_FUND_AIRWALLEX_FEE_CATEGORY_LEVEL2_CODE=<stable-level-2-code>
COMPANY_FUND_AIRWALLEX_FEE_CLASSIFICATION_POLICY_VERSION=airwallex-fee-v1
COMPANY_FUND_AIRWALLEX_REVERSAL_POLICY_VERSION=airwallex-reversal-v1

# 必须是单行 JSON；event_version 必须 == AIRWALLEX_WEBHOOK_VERSION
AIRWALLEX_FINANCIAL_TRANSACTIONS_RUNTIME_CONFIG={"enabled":true,"api_version":"2026-07-17","schema_version":"schema-v1","event_version":"event-v1","mapping_version":"mapping-stage-v1","fact_version":1,"rules":[...]}
```

注意：

- `AIRWALLEX_WEBHOOK_VERSION` ≠ runtime `event_version` → container **关闭 webhook**（打 log，不暴露 handler）
- runtime JSON 非法 / enabled=false → Airwallex REST+worker normalizer 关闭
- Phase 2 分类 code/数据库行缺失、停用或父子关系错误 → relationship processor
  仍工作，FEE movement 保留且分类任务可重试；不按中文名或 ID 猜测，不静默创建分类
- 密钥与 runtime JSON **不得**进 git；`secrets/` 已 gitignore

## 4. Stage 数据与 Console

1. **账户行**（MGT 或受控 SQL，仅一条 enabled）：

```sql
-- 形状示意；真实列名以 schema 为准，执行前在 stage 只读确认
SELECT id, channel, is_enabled, provider_account_key
FROM company_fund_accounts
WHERE channel = 'AIRWALLEX';
-- 期望：恰好 1 行 enabled=true 且 lifecycle=CURRENT
```

2. **Airwallex Console（sandbox 或 stage 对应环境）**
   - Webhook URL：`https://test-mdapi.moneradigital.com/api/webhooks/airwallex`
   - 直打 Go 后端（Caddy 反代到 `:8086`），**不要**走 Vercel
   - Secret → `AIRWALLEX_WEBHOOK_SECRET`
   - 订阅任意业务事件即可（用于 wake）；入账不依赖 envelope 类型

3. **部署**
   - 当前 artifact 通过 `monera-migrate -print-release-sequence` 声明本次迁移：
     `063`（要求数据库已存在 `062` provenance）
   - standard deploy 使用 direct `MIGRATION_DATABASE_URL`，逐个执行：
     `EXPECTED_MIGRATION_CEILING=<version> monera-migrate -exact-version <version>`
   - 任一步失败都必须在安装新 server 前停止；已应用版本按 provenance
     安全跳过
   - 含本功能的 commit 进 `stage` 后走 standard 一条龙
   - 启动 log 应出现类似：
     `company-fund runtime assembled: ... airwallex_reconciliation=true airwallex_webhook=true airwallex_ledger=true ...`
   - 任一为 false：按第 3 节对照缺项，不要靠重启碰运气

## 5. Stage 验收标准

> **已知修复（2026-07-27）**：webhook lookback 用 `time.Now()` 生成亚秒窗口时，
> PostgreSQL `timestamptz` 只存微秒，而 `MatchesWindow` 用 `time.Equal` 比较。
> Create 后 Claim 前窗口对不上会留下 `PENDING attempts=0` orphan，REST 入账静默失败。
> 修复：`CompanyFundSyncRunInput.canonical()` 与 Airwallex reconciler `validateInput`
> 统一 `UTC().Truncate(time.Microsecond)`。部署含此修复的 binary 后，新 lookback
> 应能 SUCCEEDED；旧 orphan 的 window_key 含纳秒 hash，可忽略或手工标 SKIPPED。


- [x] 进程 log：`airwallex_reconciliation=true` 且 `airwallex_webhook=true`（2026-07-27 15:55:57 UTC）
- [x] `POST /api/webhooks/airwallex` 健康：签名探测 200；坏签 400；公网可达
- [x] webhook lookback REST：sync_run `80884` SUCCEEDED attempts=1 seen=5 created=5 → facts=5 txns=5
- [x] 非 snapshot webhook 行 `event_state=IGNORED`，不增加 ledger
- [x] 重复 webhook delivery 幂等（同一 `provider_event_id` 不双插）
- [x] 相关 Go 单测绿；PR #68 合 stage 并 standard 部署成功
- [x] 修复前 orphan PENDING 80791–80794 已手工 SKIPPED（window_key 含纳秒 hash，不可重放）

### Phase 2 Stage 证据（2026-07-29）

- [x] exact dry-run 显示 `061` / `062` applied；standard deploy 按 `061 → 062`
      顺序检查并安全跳过，之后才安装并启动 `790fc2b`
- [x] FEE category 为 `operating-expense / transaction-fee`，均 enabled，
      level=1/2 且父子关系正确
- [x] 启动日志显示 ledger task processor 已装配，且无 payload、Token、API key、
      Webhook secret 或 DATABASE_URL
- [x] 3 组 Sandbox CONVERSION 共 6 条，两腿均 `COMPLETE`、默认汇总零条，
      不生成 FX gain/loss
- [x] 4 条 FEE 仅用 `SOURCE_ID_GROUP_ONLY`，无虚构 parent；全部自动分类为
      `operating-expense / transaction-fee / 计入经营收支`
- [x] 3 条 PAYOUT_REVERSAL 样本只有 source reference、没有 target reference，
      保持 quarantine/fail-closed
- [x] 两次 REST reconciliation 均 `seen=42 created=0`，facts/txns 始终
      `19/19`；19 个 movement key 全部唯一，10 个 task identity 全部唯一
- [x] 财务手工改类与手工清空后均为 `MANUAL`，自动规则更新 0 行；事务
      rollback 后 4 条 FEE 均恢复 `AUTO_RULE`
- [x] 任务共 10 条，全部 `COMPLETED`、最大 attempts=1，无
      WAITING/LEASED/DEAD_LETTER 或热循环
- [x] 公网 `/health`、`/api/health` 均 200；Console test-event 签名路径 200、
      坏签 400，test event 不写入 provider inbox
- [ ] 回滚时先停 worker；代码回退前评估 migration `062` Down 对新关系/分类审计数据的影响

### Account Lifecycle Stage 证据（migration 063）

- [ ] standard deploy 使用 exact migration `063`，完成后才安装新 server
- [ ] 错误 Account ID 校验失败且不改变候选生命周期
- [ ] 正确 Sandbox Account ID 校验成功并保存脱敏身份摘要
- [ ] `CURRENT → PAUSED → CURRENT`，暂停路径不依赖 Provider
- [ ] 身份纠正后 REST/Webhook 重放不重复流水
- [ ] 旧账户仍有未完成 provider event 时 cutover 失败关闭
- [ ] 若 Sandbox 有第二个真实 Account ID，验证 `CURRENT + CANDIDATE`
      原子切换；否则记录 Provider evidence gap
- [ ] MGT 浏览器验收权限、二次确认、异步命令状态、流水关系与分类来源
- [ ] 运行日志无异常、无密钥或数据库连接串泄露

## 6. 明确不在本交付内

- 同时启用多个 Airwallex `CURRENT` 账户
- 无 exact identity evidence 的 FEE parent / REVERSAL original 映射
- 动态业务对手方解析与 KYT
- CONVERSION 汇兑损益 movement
- production 切换与 PR #35 合并
- 把 sandbox runtime JSON 或 webhook secret 提交进仓库

## 7. 本地工具索引

| 工具 | 用途 |
| --- | --- |
| `cmd/airwallex_sandbox_smoke` | 本机 sandbox 门禁（dry-run / -persist / -webhook） |
| `internal/companyfund/*airwallex*` | 客户端、验签、reconciler、normalizer、runtime |
| `internal/handlers/company_fund_airwallex_webhook_handler.go` | HTTP 边界 |
| `internal/container/company_fund.go` | 运行时装配与 fail-closed 开关 |
