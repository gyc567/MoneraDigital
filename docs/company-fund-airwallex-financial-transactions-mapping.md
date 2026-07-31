# Airwallex Financial Transactions Mapping 覆盖与 Phase 2 契约

> 状态：实现与证据文档（2026-07-31）。本文记录当前代码对 Financial
> Transactions 的严格映射、Phase 2 关系协调，以及仍必须 fail-closed 的
> Provider evidence gap。

## 1. 背景

公司资金的 Airwallex 数据流：

```
Airwallex Financial Transactions（REST 权威）
  ← reconciler 定时 pull（airwallex_reconciliation）
  ← webhook 唤醒 lookback（修复 orphan，whsec_ 验签）
→ company_fund_provider_events（event_type=FINANCIAL_TRANSACTION_SNAPSHOT）
→ normalize（mapping → classification）
→ company_fund_provider_transaction_facts
→ company_fund_transactions
```

normalize 的核心是 **mapping**：把 Airwallex Financial Transaction（`source_type`/`transaction_type`/`status`/...）映射到公司资金的 classification（`MovementKind`/`Direction`/`AmountField`/...），决定如何入账。**未匹配 mapping 的事件 DEAD_LETTER**（fail-closed，避免错误入账）。

## 2. 当前实现

### 2.1 runtime config mapping 机制（`internal/companyfund/airwallex_runtime_config.go`）

- **显式 rule，无 fallback**（`airwallex_normalizer.go:23` 注释："deliberately no fallback rule"）
- rule key = `providerAccountKey + transactionType + sourceType + currency + status`（精确匹配，`airwallex_runtime_config.go:321-349`）
- rule 数上限 256（`maxAirwallexRuntimeRuleCount`）
- 通过 `AIRWALLEX_FINANCIAL_TRANSACTIONS_RUNTIME_CONFIG` 环境变量配置（JSON）

### 2.2 rule classification 字段 + 枚举

| 字段 | 枚举 | runtime config 约束 |
|------|------|---------------------|
| Action | `APPLY` / `IGNORE` / `QUARANTINE` | 三者均需 exact rule；未知 tuple fail-closed |
| MovementKind | `PRINCIPAL` / `FEE` / `REVERSAL` / `ADJUSTMENT` / `CONVERSION` | 复杂类型必须声明 evidence-pinned `relationship` |
| Direction | `INFLOW` / `OUTFLOW` / `INTERNAL_TRANSFER` | 三者之一（`:485`）|
| TransferMode | `SINGLE` / `BATCH` | — |
| AmountField | `AMOUNT` / `FEE` / `NET` | — |
| ExpectedSign | `POSITIVE` / `NEGATIVE` | — |
| OccurredAtField | `CREATED_AT` / `SETTLED_AT` | — |

### 2.3 relationship resolver 现状

`AirwallexProviderEventNormalizer` 有 3 个 resolver 接口（`mappings` / `relationships` / `counterparties`，`airwallex_provider_event_normalizer.go:114-116`）：

- 三个 resolver 均由严格 runtime rule 驱动；不存在 wildcard 或生产默认映射。
- FEE、CONVERSION、REVERSAL 必须配置 `relationship.strategy`、
  `evidence_reference` 和正数 SLA。
- `已解析`关系可立即生成 movement；精确父交易、原交易或换汇对腿尚未到达时，
  只保存 fact 与 `company_fund_ledger_tasks`，由 PostgreSQL lease/retry 恢复。
- `source_id` 只保存为 provider fact 和关系证据，**不参与 movement identity**。
- 动态对手方不属于 relationship resolver；`PAYOUT` 由独立的 Transfer
  beneficiary resolver 补充，其他动态类型仍 fail-closed/留空。

## 3. Airwallex Financial Transaction 枚举（官方文档）

来源：[Airwallex Financial Transactions API](https://www.airwallex.com/docs/api/finance/financial_transactions)

- **`source_type`**（22 种）：`PAYOUT`, `CONVERSION`, `DEPOSIT`, `ADJUSTMENT`, `FEE`, `PAYMENT_ATTEMPT`, `REFUND`, `DISPUTE`, `CHARGE`, `TRANSFER`, `YIELD`, `BATCH_PAYOUT`, `CARD_PURCHASE`, `CARD_REFUND`, `PURCHASE`, `REFUND_REVERSAL`, `REPAYMENT`, `TRANSFER_OUT`, `TRANSFER_IN`, `TRANSFER_OUT_REVERSAL`, `TRANSFER_IN_REVERSAL`, `WITHHOLDING_TAX`
- **`transaction_type`**（40+ 种）：`PAYOUT`/`PAYOUT_FAILURE`/`PAYOUT_REVERSAL`, `DEPOSIT`, `ADJUSTMENT`, `FEE`, `CONVERSION_SELL`/`CONVERSION_BUY`/`CONVERSION_REVERSAL`, `TRANSFER`/`TRANSFER_IN`/`TRANSFER_OUT`/`..._REVERSAL`, `PAYMENT`, `REFUND`/`REFUND_REVERSAL`/`REFUND_FAILURE`, `DISPUTE`/`DISPUTE_REVERSAL`/`DISPUTE_LOST`, `YIELD`, `WITHHOLDING_TAX`/`..._REVERSAL`, `ISSUING_*`, `DD_CREDIT`/`DD_DEBIT`, `DC_CREDIT`/`DC_DEBIT`, `PAYMENT_RESERVE_HOLD`/`RELEASE`, `PREPAYMENT`/`PREPAYMENT_RELEASE`, `REPAYMENT_FUNDING`/`REPAYMENT_DEDUCTION`, `MOR_PAYOUT`/`MOR_PAYOUT_REVERSAL`, `PURCHASE`
- **`status`**：`PENDING` / `SETTLED` / `CANCELLED`
- **无显式 direction 字段**：inflow/outflow 靠 `source_type`/`transaction_type` 推断
- **monetary 字段**：`amount`（总额）/ `net`（净额）/ `fee`（手续费）/ `currency` / `client_rate` / `currency_pair`
- **temporal 字段**：`created_at` / `estimated_settled_at` / `settled_at`

方向语义（推断）：

- **Inflow**：`DEPOSIT`, `DD_CREDIT`, `DC_CREDIT`, `TRANSFER_IN`, `PAYMENT`, `CONVERSION_BUY`, `REFUND_REVERSAL`, `PAYOUT_REVERSAL`, `PAYOUT_FAILURE`, `YIELD`, `CARD_REFUND`, `PAYMENT_RESERVE_RELEASE`, `PREPAYMENT_RELEASE`, ...
- **Outflow**：`PAYOUT`, `BATCH_PAYOUT`, `CARD_PURCHASE`, `PURCHASE`, `DD_DEBIT`, `DC_DEBIT`, `TRANSFER_OUT`, `CONVERSION_SELL`, `FEE`, `REFUND`, `DISPUTE`, `CHARGE`, `WITHHOLDING_TAX`, `ISSUING_AUTHORISATION_HOLD`, `ISSUING_CAPTURE`, `PREPAYMENT`, `REPAYMENT_DEDUCTION`, ...
- **Neutral / 双向**：`TRANSFER`, `CONVERSION`, `ADJUSTMENT`（方向依赖上下文）

## 4. Stage 覆盖快照

截至 2026-07-31，stage `AIRWALLEX_FINANCIAL_TRANSACTIONS_RUNTIME_CONFIG`
（`mapping-sandbox-v1`）包含以下简单类型规则；其他环境仍必须根据对应环境的
Provider evidence 生成配置，不能直接复制 Sandbox 清单：

- 5 条 `ADJUSTMENT SETTLED INFLOW`（CNY/EUR/GBP/SGD/USD）— sandbox 2026-07-27 初始余额调整，`evidence_reference: sandbox-2026-07-27-adjustment-settled-*`
- 1 条 `PAYOUT SETTLED OUTFLOW`（SGD）— 2026-07-28 验收（transfer `P260728-3824YVV`，1000 SGD；`amount_field=AMOUNT`、`expected_sign=NEGATIVE`、`occurred_at_field=CREATED_AT`），fact id=26 + transaction id=19 已入账，`evidence_reference: sandbox-2026-07-28-payout-settled-sgd`
- 1 条 `PAYOUT SETTLED OUTFLOW`（USD）— 2026-07-31 验收（SWIFT transfer
  `P260731-RATI18U`，1000 USD；`amount_field=AMOUNT`、`expected_sign=NEGATIVE`、
  `occurred_at_field=CREATED_AT`），fact id=42 + transaction id=34 已入账，
  `evidence_reference: sandbox-2026-07-31-payout-settled-usd-swift`

## 5. 覆盖矩阵 + gap

### 5.1 简单类型（通过 runtime config rule 按币种精确覆盖）

这些 `source_type` 的 MovementKind ∈ {`PRINCIPAL`, `ADJUSTMENT`}、Action=`APPLY`，符合 runtime config 约束，只需 evidence-backed rule：

| source_type | 方向 | 公司资金场景 | 优先级 |
|-------------|------|-------------|--------|
| `DEPOSIT` | inflow | 收款入账 | 高 |
| `PAYOUT` | outflow | 付款（已验证 1000 SGD、1000 USD SWIFT 场景）| 高 |
| `PAYMENT_ATTEMPT` | inflow | 收款业务 | 高 |
| `TRANSFER_IN` | inflow | 内部转入 | 中 |
| `TRANSFER_OUT` | outflow | 内部转出 | 中 |
| `TRANSFER` | internal | 内部调拨 | 中 |
| `YIELD` | inflow | 利息收益 | 中 |
| `WITHHOLDING_TAX` | outflow | 预扣税 | 中 |
| `REFUND` | outflow | 退款 | 低 |
| `CHARGE` | outflow | 扣费 | 低 |

### 5.2 复杂类型（已实现，启用仍需 evidence）

| 类型 | 策略 | 入账语义 |
|------|------|----------|
| 独立 `FEE` | `SOURCE_ID_EXACT_PARENT` / `SOURCE_ID_GROUP_ONLY` / `BATCH_ID_GROUP_ONLY` | 精确证据才写 parent；组证据只写 group；自动分类为配置的“经营支出 / 手续费 / 是” |
| `CONVERSION_SELL` + `CONVERSION_BUY` | `SOURCE_ID_CONVERSION_GROUP` | 同一已验证 conversion identifier 恰好一买一卖；同一 Airwallex 账户的不同币种余额；`INTERNAL_TRANSFER` |
| `*_REVERSAL` | `SOURCE_ID_REVERSAL_TARGET` | 原交易尚未到达则等待；到达后保存独立 reversal 并继承有效分类 |

复杂类型的 task payload 只保存 allowlisted normalized proposal 和 SHA-256
digest，不保存原始 Provider payload。等待任务有 lease、attempt、next-attempt、
SLA、terminal state 和安全错误码，进程重启可恢复。

### 5.3 动态 counterparty

`ADJUSTMENT` 等内部调整可能没有外部对手方。动态外部对手方按类型独立解析：

- `PAYOUT`：exact runtime rule 命中后，以 Financial Transaction 的
  `source_id` 调用官方 `GET /api/v1/transfers/{id}`；只提取 beneficiary 的完整
  `account_number`（无本地账号时使用完整 `iban`）和 `account_name`。不读取或
  保存本期不展示的其他 beneficiary 银行/个人字段。
- `source_id` 只用于已确认的 Transfer 资源查询，仍不参与 movement identity。
- 客户端按数据库中的当前 Airwallex account scope 创建，避免跨账户查询。
- 网络、5xx、408/409/425/429 等暂时失败进入耐久重试；404、格式错误或缺少完整
  账号按永久失败处理，避免无限请求。任何失败都不会用空响应覆盖已有有效对手方。
- 已处理且加密快照仍在 retention 内、完整账号为空的历史 PAYOUT，会被有界、
  `SKIP LOCKED` 地重新排队并通过同一 upsert 路径补齐；扫描到空批次后本进程停止
  该发布期回填。重复处理仍只有一条 movement。
- `DEPOSIT` / `PAYMENT` 等类型尚未取得已验证的对应资源契约，当前不推断对手方。
- 本次只负责 Provider 对手方展示事实，不新增 KYT / 合规判断。

## 6. 补全路径（分阶段）

### Phase 1：runtime config rules（简单类型，config-only，无代码）
- sandbox 触发各 `source_type`（`DEPOSIT`/`PAYOUT`/`PAYMENT`/`TRANSFER`/`YIELD`/`TAX`）
- 查 Financial Transaction 真实字段（`source_type`/`transaction_type`/`status`/`amount`/`occurred_at`）
- 配 evidence-backed rule（`AIRWALLEX_FINANCIAL_TRANSACTIONS_RUNTIME_CONFIG`，每条带 `evidence_reference`）
- 验证各类型入账（facts 创建）

### Phase 2：复杂流水关系与财务语义（代码已完成）

- 独立 FEE movement、精确/组级关系、默认自动分类与历史分批回填。
- CONVERSION 一买一卖配对；孤儿腿耐久等待；完成前不进入财务明细；完成后
  两腿可审计但自动排除外部/经营汇总；本期不生成 FX gain/loss。
- REVERSAL 独立 movement、原交易关联、分类/经营属性/汇总语义继承与反向净额。
- 管理后台直接修改共享数据库会触发 `MANUAL` 所有权；人工修改或人工清空后，
  provider replay、自动规则和 reversal 传播都不覆盖。
- 受控 schema 版本：migration `062`。

### Phase 3：dedicated counterparty resolver（部分完成）

- [x] `PAYOUT`：Financial Transaction `source_id` → Transfer beneficiary；
  完整账号/IBAN、账户名；历史 retention 内回填。
- [ ] `DEPOSIT` / `PAYMENT`：取得 Sandbox/官方资源关联证据后分别实现，禁止猜测。
- [ ] KYT / 合规集成（独立范围）。

## 7. evidence-backed 设计原则

每个 rule 必须基于 sandbox 实测的 Financial Transaction（`evidence_reference` 字段）。**不能凭文档猜测** `amount_field` / `occurred_at_field` / `classification`，因为：

- Airwallex 不同 `source_type` 的 amount 语义可能不同（`amount` vs `net` vs `fee`）
- `occurred_at` 可能用 `created_at` 或 `settled_at`
- `status` 流转（`PENDING` → `SETTLED`）决定何时入账
- direction 推断需实测确认（无显式 direction 字段）

## 8. 已解决阻塞

**1000 SGD payout（`P260728-3824YVV`）** — 2026-07-28 解决。配 `PAYOUT SGD SETTLED OUTFLOW` rule 后，fact id=26 + transaction id=19 已入账（原 DEAD_LETTER 的 provider event 已在 stage 重处理成功）。issue #72 记录。

**1000 USD SWIFT payout（`P260731-RATI18U`）** — 2026-07-31
解决。Financial Transaction `ftx_1qAED__GPQ6P_CmaCvcYSA` 已是 `SETTLED`，
金额字段为 `amount=-1000`、`fee=0`、`net=-1000`，而 Transfer 资源当时仍为
`PROCESSING`。配精确 `PAYOUT USD SETTLED OUTFLOW` rule 后，原 DEAD_LETTER
event id=1105 经受控重处理成为 PROCESSED，fact id=42 + transaction id=34
入账，完整 beneficiary account/name 与 Transfer API 精确一致，重复 movement
数量为 1。该证据同时确认公司资金账务以 Financial Transactions 状态为准，
不等待 Transfer 生命周期变为终态。

Airwallex Console 对这笔付款展示 15.58 USD transfer fee，收款人得到 984.42
USD，但截至验收窗口，Financial Transactions 仅返回上述 `PAYOUT -1000 USD`，
没有独立 `FEE` item，且该 item 的 `fee=0`、`net=-1000`。因此当前只登记 1000
USD 的公司余额支出，不根据 Console 展示自行推导第二条手续费流水；若 Provider
后续产生独立 `FEE` 事实，再由 FEE exact rule 按权威事实入账。

## 9. 关联

- webhook 验签支持：issue #72
- stage sandbox 验收：PR #67 / #68 / #69
- stage→main release：PR #70
- 域语言 / ADR：`CONTEXT.md`、`docs/adr/`

## 10. Sandbox evidence（2026-07-29 复核）

Phase 1 原计划在 sandbox 触发各 simple type 收集 evidence，实测以下限制：

| 类型 | sandbox 可触发 | 限制 |
|------|---------------|------|
| `ADJUSTMENT` | ✅ 已配 5 币种 | — |
| `PAYOUT`/SGD | ✅ 已配（1000 SGD 入账）| 用户通过 Console UI 发款；Financial Transaction `source_id` 可通过官方 `GET /api/v1/transfers/{id}` 查询 Transfer，Sandbox 实测 200，并返回 beneficiary 的完整 `account_number`、`account_name`、`bank_name` |
| `PAYOUT`/USD | ✅ 已配（1000 USD SWIFT 入账）| Financial Transaction 已 `SETTLED` 时 Transfer 仍可为 `PROCESSING`；完整 beneficiary account/name 已通过 Transfer API 精确回填；Console 展示的收款侧 fee 不得在 Provider 未返回独立 `FEE` item 时自行拆账 |
| `PAYOUT` 其他币种 | ❌ | 同上 + 无该币种 exact evidence |
| `DEPOSIT` / `YIELD` / `WITHHOLDING_TAX` | ❌ | sandbox 不模拟真实业务（汇款/利息/税务）|
| `TRANSFER` / `TRANSFER_IN` / `TRANSFER_OUT` | ❌ | 单账号，无 linked accounts |
| `CONVERSION`（Phase 2）| ✅ | `/api/v1/fx/conversions/create`；SELL/BUY 两腿共享 `source_id=conversion_id`，SELL 为负、BUY 为正，币对与两腿币种一致。可配置 `SOURCE_ID_CONVERSION_GROUP`。|
| `FEE`（Phase 2）| ✅ 有独立样本 | `source_type=FEE`、`transaction_type=FEE`、SETTLED；`amount` 为负、`fee` 字段为零；`source_id` 在当前 Financial Transactions 列表中不等于某条父 item ID。因此当前证据只允许 `SOURCE_ID_GROUP_ONLY`，不得配置 exact parent。|
| `PAYOUT_REVERSAL`（Phase 2）| ✅ 有独立样本 | SETTLED、金额为正，存在 `source_id`，部分记录有 `batch_id`；但 `source_id` 未命中当前 Financial Transaction item ID，尚不足以配置 `SOURCE_ID_REVERSAL_TARGET`，继续 fail-closed。|

**Airwallex API `page_num` 是 0-based**（`page_num=0` 是第一页）。本地
`go run ./cmd/airwallex_sandbox_smoke -evidence` 只输出脱敏后的结构证据，不输出
Provider ID、凭据或 payload。任何新增 exact-parent / exact-reversal rule 都必须先用
可复核的跨资源 identity evidence 更新本节，不能按金额、时间、顺序或相似 ID 猜测。

Console "Send test event" 验签（`client-secret-key` 路径）已修复并 TDD 覆盖（commit `615a53e`）。
