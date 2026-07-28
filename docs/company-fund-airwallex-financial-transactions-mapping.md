# Airwallex Financial Transactions Mapping 覆盖梳理

> 状态：梳理文档（2026-07-28）。梳理当前 Airwallex Financial Transactions normalize mapping 的实现、覆盖与 gap，规划补全路径。**本文档不引入代码变更。**

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
| Action | `APPLY` / `IGNORE` / `QUARANTINE` | 必须 `APPLY`（`airwallex_runtime_config.go:479`）|
| MovementKind | `PRINCIPAL` / `FEE` / `REVERSAL` / `ADJUSTMENT` / `CONVERSION` | 拒绝 `FEE`/`REVERSAL`/`CONVERSION`（`:472`，需 dedicated resolver）|
| Direction | `INFLOW` / `OUTFLOW` / `INTERNAL_TRANSFER` | 三者之一（`:485`）|
| TransferMode | `SINGLE` / `BATCH` | — |
| AmountField | `AMOUNT` / `FEE` / `NET` | — |
| ExpectedSign | `POSITIVE` / `NEGATIVE` | — |
| OccurredAtField | `CREATED_AT` / `SETTLED_AT` | — |

### 2.3 dedicated resolver 现状

`AirwallexProviderEventNormalizer` 有 3 个 resolver 接口（`mappings` / `relationships` / `counterparties`，`airwallex_provider_event_normalizer.go:114-116`）：

- 当前三个 resolver **全部由 runtime config 实现**（静态 rule）
- **dedicated resolver 未实现**（FEE/CONVERSION/REVERSAL 的 relationship + 动态 counterparty）
- runtime config 的 relationship resolver 返回空关系（`airwallex_runtime_config.go:288` 注释："intentionally leaves linked movement kinds quarantined until a dedicated, evidence-backed relation resolver exists"）

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

## 4. 当前覆盖

stage `AIRWALLEX_FINANCIAL_TRANSACTIONS_RUNTIME_CONFIG`（`mapping-sandbox-v1`）：**6 条 rule**：

- 5 条 `ADJUSTMENT SETTLED INFLOW`（CNY/EUR/GBP/SGD/USD）— sandbox 2026-07-27 初始余额调整，`evidence_reference: sandbox-2026-07-27-adjustment-settled-*`
- 1 条 `PAYOUT SETTLED OUTFLOW`（SGD）— 2026-07-28 验收（transfer `P260728-3824YVV`，1000 SGD；`amount_field=AMOUNT`、`expected_sign=NEGATIVE`、`occurred_at_field=SETTLED_AT`），fact id=26 + transaction id=19 已入账，`evidence_reference: sandbox-2026-07-28-payout-settled-sgd`

## 5. 覆盖矩阵 + gap

### 5.1 简单类型（可加 runtime config rule，当前未覆盖）

这些 `source_type` 的 MovementKind ∈ {`PRINCIPAL`, `ADJUSTMENT`}、Action=`APPLY`，符合 runtime config 约束，只需 evidence-backed rule：

| source_type | 方向 | 公司资金场景 | 优先级 |
|-------------|------|-------------|--------|
| `DEPOSIT` | inflow | 收款入账 | 高 |
| `PAYOUT` | outflow | 付款（1000 SGD 场景）| 高 |
| `PAYMENT_ATTEMPT` | inflow | 收款业务 | 高 |
| `TRANSFER_IN` | inflow | 内部转入 | 中 |
| `TRANSFER_OUT` | outflow | 内部转出 | 中 |
| `TRANSFER` | internal | 内部调拨 | 中 |
| `YIELD` | inflow | 利息收益 | 中 |
| `WITHHOLDING_TAX` | outflow | 预扣税 | 中 |
| `REFUND` | outflow | 退款 | 低 |
| `CHARGE` | outflow | 扣费 | 低 |

### 5.2 复杂类型（runtime config 不支持，需 dedicated resolver）

MovementKind ∈ {`FEE`, `REVERSAL`, `CONVERSION`}，runtime config 明确拒绝（`airwallex_runtime_config.go:472`）。需开发 dedicated relationship resolver：

| source_type / transaction_type | MovementKind | 关联需求 |
|--------------------------------|--------------|----------|
| `FEE` | `FEE` | `fee.source_id` → 父交易 |
| `CONVERSION`（`CONVERSION_SELL` + `CONVERSION_BUY`）| `CONVERSION` | 买卖腿配对（conversion group）|
| `*_REVERSAL`（`PAYOUT_REVERSAL` / `REFUND_REVERSAL` / `CONVERSION_REVERSAL` / `TRANSFER_*_REVERSAL`）| `REVERSAL` | `reversal.source_id` → 原交易 |

### 5.3 动态 counterparty gap

runtime config 的 counterparty resolver 返回 rule 内静态 counterparty（`airwallex_runtime_config.go:300-307`）。`ADJUSTMENT` 等内部调整无对手方（或自己）。但 **`PAYOUT` / `DEPOSIT` / `PAYMENT` 的对手方是动态的**（每笔不同收款人/付款人）：

- 当前 rule 无 counterparty 字段（stage config rules 未配 counterparty）
- 动态对手方需 dedicated counterparty resolver（从 transaction field 提取，如 `source_id` / `funding_source_id` / 扩展字段）
- 涉及 KYT / 合规（对手方筛查）

## 6. 补全路径（分阶段）

### Phase 1：runtime config rules（简单类型，config-only，无代码）
- sandbox 触发各 `source_type`（`DEPOSIT`/`PAYOUT`/`PAYMENT`/`TRANSFER`/`YIELD`/`TAX`）
- 查 Financial Transaction 真实字段（`source_type`/`transaction_type`/`status`/`amount`/`occurred_at`）
- 配 evidence-backed rule（`AIRWALLEX_FINANCIAL_TRANSACTIONS_RUNTIME_CONFIG`，每条带 `evidence_reference`）
- 验证各类型入账（facts 创建）

### Phase 2：dedicated relationship resolver（复杂类型，代码开发）
- `FEE` resolver：fee → 父交易关联
- `CONVERSION` resolver：sell + buy 腿配对（conversion group）
- `REVERSAL` resolver：reversal → 原交易关联
- 实现 `AirwallexProviderEventRelationshipResolver`（非 runtime config 路径）

### Phase 3：dedicated counterparty resolver（动态对手方，代码开发）
- 从 transaction field 提取对手方
- KYT / 合规集成
- 覆盖 `PAYOUT` / `DEPOSIT` / `PAYMENT` 动态对手方

## 7. evidence-backed 设计原则

每个 rule 必须基于 sandbox 实测的 Financial Transaction（`evidence_reference` 字段）。**不能凭文档猜测** `amount_field` / `occurred_at_field` / `classification`，因为：

- Airwallex 不同 `source_type` 的 amount 语义可能不同（`amount` vs `net` vs `fee`）
- `occurred_at` 可能用 `created_at` 或 `settled_at`
- `status` 流转（`PENDING` → `SETTLED`）决定何时入账
- direction 推断需实测确认（无显式 direction 字段）

## 8. 已解决阻塞

**1000 SGD payout（`P260728-3824YVV`）** — 2026-07-28 解决。配 `PAYOUT SGD SETTLED OUTFLOW` rule 后，fact id=26 + transaction id=19 已入账（原 DEAD_LETTER 的 provider event 已在 stage 重处理成功）。issue #72 记录。

## 9. 关联

- webhook 验签支持：issue #72
- stage sandbox 验收：PR #67 / #68 / #69
- stage→main release：PR #70
- 域语言 / ADR：`CONTEXT.md`、`docs/adr/`

## 10. sandbox 触发限制（2026-07-28 穷尽验证）

Phase 1 原计划在 sandbox 触发各 simple type 收集 evidence，实测以下限制：

| 类型 | sandbox 可触发 | 限制 |
|------|---------------|------|
| `ADJUSTMENT` | ✅ 已配 5 币种 | — |
| `PAYOUT`/SGD | ✅ 已配（1000 SGD 入账）| 用户通过 Console UI 发款；API `payouts/beneficiaries` + `payouts/transfers/{id}` 返回 404（账号无 Payouts 产品权限）|
| `PAYOUT` 其他币种 | ❌ | 同上 + 无 beneficiary |
| `DEPOSIT` / `YIELD` / `WITHHOLDING_TAX` | ❌ | sandbox 不模拟真实业务（汇款/利息/税务）|
| `TRANSFER` / `TRANSFER_IN` / `TRANSFER_OUT` | ❌ | 单账号，无 linked accounts |
| `CONVERSION`（Phase 2）| ✅ 已触发 evidence | 正确路径 `/api/v1/fx/conversions/create`（带 `fx` 前缀）；sell+buy 两腿 `source_id` 共享 conversion_id（**配对 key 确认**），SELL `amount=-1`/BUY `amount=+1.29`，`currency_pair=USDSGD`。amendment：`/api/v1/fx/conversion_amendments/create`（`type=CANCEL`）|
| `FEE`（Phase 2）| ❌ | sandbox PAYOUT/CONVERSION 均 `fee=0`，无独立 `FEE` source_type（sandbox 不收手续费），需生产 evidence |
| `REVERSAL`（Phase 2）| ❌ | sandbox conversion 立即 SETTLED，amendment `CANCEL` 被拒（`cannot amend in state SETTLED`）；需生产 evidence |

**Airwallex API `page_num` 是 0-based**（`page_num=0` 是第一页）。用 1-based 假设查询会得到空结果或残缺数据——曾因此误判为"数据不稳定"，实际数据始终稳定。sandbox 实测稳定 8 条（5 `ADJUSTMENT` + 1 `PAYOUT`/SGD + 2 `CONVERSION`，stage DB 前置 6 条全部 `PROCESSED`），**没有 `DEPOSIT`/`YIELD`/`TAX`/`TRANSFER`/`FEE`/`REVERSAL` 类型**。剩余 simple type 的 evidence-backed rule 须在**真实业务环境**（生产 Airwallex 账号产生真实 `DEPOSIT`/`YIELD`/`TAX`/`TRANSFER`）后补配，不能凭文档猜测 `amount_field` / `occurred_at_field` / 方向。

Console "Send test event" 验签（`client-secret-key` 路径）已修复并 TDD 覆盖（commit `615a53e`）。
