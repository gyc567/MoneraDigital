# 产品需求文档：用户提现功能

## 前言
> **核心目标**：为用户提供从资产提现申请到区块链转账完成的全链路功能。通过与 Safeheron 托管服务的深度集成，确保用户资金安全提出，支持多链提币，提供完整的提现进度追踪和异常处理机制，提升用户资金流动性和应急支取体验。

---

## 一、 版本信息

| 版本号 | 创建日期 | 审核人 |
| :--- | :--- | :--- |
| V1.0.0 | 2026-01-09 | 待定 |

---

## 二、 变更日志

| 时间 | 版本号 | 变更人 | 主要变更内容 |
| :--- | :--- | :--- | :--- |
| 2026-01-09 | V1.0.0 | Claude | 文档初始化：定义用户提现完整流程，整合 Safeheron 提币接口 |

---

## 三、 文档说明

### 名词解释

| 术语 / 缩略词 | 说明 |
| :--- | :--- |
| **Safeheron** | 数字资产托管服务商，负责资金安全保管和区块链提币执行。 |
| **提现 (Withdrawal)** | 用户将理财账户资金转出至指定区块链钱包地址的操作。 |
| **提币地址（提现地址）** | 用户在某条公链上的钱包地址（如 USDT-TRC20 地址），作为提币的目标接收地址。 |
| **链上确认** | 区块链网络确认交易已上链的状态，通常需要等待若干个区块确认。 |
| **白名单地址** | 用户预先添加的可信任提币地址，首次提现至新地址需要身份验证。 |
| **矿工费 / 手续费** | 区块链网络费用 + 平台提现手续费的总和，由用户承担或平台补贴。 |
| **T+0 / T+1** | T+0 表示同日提币，T+1 表示次日到账（取决于链和网络状况）。 |

---

## 四、 需求背景

### 产品 / 数据现状
用户在理财账户积累了收益或本金后，需要灵活支取资金。目前系统尚无提现功能，导致用户资金流动性受限，制约产品的市场竞争力。

### 用户调研
*   **痛点**：
    *   用户担心资金被锁定，无法在紧急情况下支取。
    *   提现过程复杂，涉及多个确认步骤，用户害怕出错。
*   **期望**：
    *   提现流程简洁快速，最好一键提现。
    *   能实时查看提现状态和到账时间。
    *   支持多条公链提币，选择灵活。

### 竞品分析

| 主要信息 | 关键结论 |
| :--- | :--- |
| **某头部理财 APP** | 支持 T+0 提现，但通过平台垫资实现，风险高。提现页面实时显示矿工费，用户感知强。 |
| **竞品 B** | 支持白名单管理，首次提现至新地址需要短信/邮件二次验证，安全但操作步骤多。 |
| **竞品 C** | 提现至自有钱包是免费的，但提现至第三方地址收 0.5% 手续费，激励用户自有钱包绑定。 |

**结论**：采用"白名单 + 二次验证"的安全模式，支持多链提币，提现手续费透明可见，首次新地址提现需身份验证，但后续可快速提现。

---

## 五、 需求范围

1.  **资产概览与提现入口**：用户在资产页面看到可提现余额，点击"提现"进入流程。
2.  **提币地址管理**：支持添加、编辑、删除提币地址，实现白名单管理。
3.  **提现申请**：输入提现金额、选择提币链和地址，计算手续费，展示到账额度。
4.  **身份安全验证**：首次新地址提现需要二次验证（短信/邮件/谷歌认证器）。
5.  **Safeheron 提币接口**：后端调用 Safeheron 的 Coin Out（提币）API。
6.  **提现订单追踪**：用户可查看提现历史、订单状态、区块链交易哈希。
7.  **异常处理与重试**：网络中断、余额变化、地址非法等异常场景的友好提示。

---

## 六、 功能详细说明

### 1. 业务流程图（冻结机制简化方案）

> **简化主流程**：
> 用户进入资产页面 -> 查看可提现余额 -> 点击"提现"按钮 -> **冻结金额** -> **身份验证（如需）** -> **调用 Safeheron API** -> 扣款创建订单 -> 前端展示提现处理中 -> **轮询查询订单状态** -> 交易上链 -> **提现完成** -> 用户邮件/APP 通知。
>
> **异常处理流程**：
> - Safeheron 调用失败 → **立即解冻** → 用户可重试（不需要等待，本金恢复立即可用）
> - 超时 > 1 小时 → **定时任务自动解冻** → 用户重新申请
> - 地址错误/不可用 → 提示用户重新选择地址并重试

**流程图（ASCII）**：
```
┌─────────────────┐
│ 用户点击提现    │
└────────┬────────┘
         ↓
    ┌─────────────────────────────┐
    │ 冻结金额                    │
    │ frozen_balance += amount     │
    │ 用户看到"冻结中..."          │
    └────────┬────────────────────┘
             ↓
    ┌─────────────────────────────┐
    │ 身份验证（如需）             │
    │ 首次新地址需二次验证         │
    └────────┬────────────────────┘
             ↓
    ┌─────────────────────────────┐
    │ 调用 Safeheron API          │
    │ (30秒超时)                  │
    └────────┬────────────────────┘
             ↓
      ┌──────┴──────┐
      ↓             ↓
  ┌─成功─┐    ┌─失败─┐
  │      │    │      │
  ↓      ↓    ↓      ↓
┌──────────────┐  ┌──────────────────┐
│ 扣款 + 创建  │  │ 立即解冻          │
│ 订单（事务） │  │ 用户可立即重试    │
│ 清除冻结     │  │ (无需等待)        │
└───────┬──────┘  └──────┬───────────┘
        ↓                ↓
    用户看到订单      显示失败原因
    号和hash           提供重试按钮
        ↓
    轮询Safeheron
    查询订单状态
        ↓
    交易上链确认
        ↓
    提现完成
        ↓
    邮件通知用户
```

**超时处理（> 1 小时未完成）**：
```
定时任务每小时执行一次
  ↓
发现未完成订单 > 1 小时
  ↓
自动解冻金额
  ↓
标记订单为 TIMEOUT
  ↓
用户可重新申请提现
```

### 2. 功能模块详述

#### 模块一：资产中心与提现入口

| 序号 | 功能点 | 详细说明 | 交互逻辑 |
| :--- | :--- | :--- | :--- |
| 1.1 | **理财账户余额展示** | 在用户资产页面顶部展示：<br>1. **总资产（USDT）**<br>2. **昨日收益（USDT）**<br>3. **可提现余额（USDT）**：不含冻结中的待赎回订单。 | 数据实时更新，支持隐藏显示。 |
| 1.2 | **提现按钮状态** | 按钮状态流转：<br>- **可提现**：可提现余额 > 0，按钮高亮可点。<br>- **余额为 0**：按钮灰显，提示"暂无可提现余额"。<br>- **提现中**：用户有正在处理的提现订单，按钮显示"提现处理中"，点击进入订单详情。 | 底部固定或右下浮窗。 |
| 1.3 | **提现订单快捷入口** | 在资产页面下方显示最近 3 条提现订单（最新优先）。<br>显示字段：**金额、状态、申请时间、预计到账时间**。 | 点击订单卡片进入详情页，可查看交易哈希和区块链确认进度。 |

#### 模块二：提币地址白名单管理

| 序号 | 功能点 | 详细说明 | 交互逻辑 |
| :--- | :--- | :--- | :--- |
| 2.1 | **地址列表展示** | 用户进入"地址管理"页面，展示已保存的所有提币地址。<br>每个地址卡片显示：<br>- **地址别名**（自定义，如"我的 TRC20 钱包"）<br>- **区块链类型**（TRC20/ERC20/BEP20 等）<br>- **完整地址**（部分脱敏，显示前 6 位和后 4 位）<br>- **添加时间**<br>- **操作按钮**：【编辑】【删除】 | 点击地址卡片可展开查看完整地址。 |
| 2.2 | **新增地址** | 用户点击"添加地址"按钮，进入表单：<br>1. **地址别名**（必填，如 "Binance 提现地址"）<br>2. **选择区块链**（下拉，TRC20/ERC20/BEP20/... ）<br>3. **输入地址**（必填，长字符串）<br>4. **地址验证**（前端正则校验，后端严格校验）<br>5. **提交按钮** | 新地址需要二次验证（见模块四）。 |
| 2.3 | **编辑地址** | 用户只能修改**别名**，不能修改链类型和地址本身（为了防止误修改）。 | 如需更改链或地址，需删除后重新添加。 |
| 2.4 | **删除地址** | 用户点击【删除】后，弹出确认框。<br>确认后，地址被标记为 soft delete（逻辑删除），不再显示在列表中。<br>历史订单中仍可查看该地址（数据溯源）。 | 防止意外删除，需要二次确认。 |
| 2.5 | **地址分组与标签** | （未来优化）支持给地址添加标签（如"交易所"、"个人钱包"），便于快速分类。 | 暂不实现。 |

#### 模块三：提现申请与确认

| 序号 | 功能点 | 详细说明 | 交互逻辑 |
| :--- | :--- | :--- | :--- |
| 3.1 | **提现申请页导航** | 用户点击"提现"按钮后进入该页面。<br>**页面标题**：申请提现<br>**核心元素**：<br>- 理财账户可提现余额展示<br>- 选择提币链和地址的下拉菜单<br>- 金额输入框<br>- 手续费计算展示<br>- 到账金额展示<br>- 【全部提现】快速按钮<br>- 【确认提现】提交按钮 | 采用单页表单设计，输入实时校验和计算。 |
| 3.2 | **提币链和地址选择** | **第一步：选择提币链**<br>- 下拉菜单列出该地址支持的所有链（如 TRC20、ERC20）<br>- 显示对应矿工费估算<br><br>**第二步：选择提币地址**<br>- 如无已保存地址，提示"暂无地址，请先添加"<br>- 如有已保存地址，列表展示，用户选择一个<br>- 支持快速新增地址（弹窗） | 链和地址选择会影响手续费计算，实时更新。 |
| 3.3 | **提现金额输入** | **限制说明**：<br>- 只能输入数字，支持小数。<br>- 最小提现额：100 USDT（业务配置）<br>- 最大提现额：可提现余额<br>- 单日提现限额：500,000 USDT（业务配置）<br>- 单笔最大额：100,000 USDT（业务配置）<br><br>**校验逻辑**：<br>- 输入值 < 最小额 -> 红字提示"最小提现 100 USDT"<br>- 输入值 > 可提现余额 -> 红字提示"余额不足"<br>- 输入值 > 单笔上限 -> 红字提示"单笔最多 100,000 USDT"<br>- 单日累计 + 新提现 > 日限额 -> 红字提示"今日剩余可提现 X USDT" | 实时校验，禁止提交非法金额。 |
| 3.4 | **手续费计算与展示** | **公式**：<br>```<br>矿工费 = 区块链固定费 (由 Safeheron 返回)\n平台手续费 = 提现金额 × 手续费率 (0.5%)\n总手续费 = 矿工费 + 平台手续费\n\n到账金额 = 提现金额 - 总手续费\n```<br><br>**示例**（提现 1000 USDT 到 TRC20）：<br>- 矿工费：1 USDT（Safeheron 估算）<br>- 平台手续费：1000 × 0.5% = 5 USDT<br>- 总手续费：6 USDT<br>- 到账金额：994 USDT<br><br>**展示设计**：<br>- 分行展示各项费用<br>- 到账金额加粗高亮，让用户清晰了解实际到账额<br>- 支持悬停显示手续费说明 | 输入金额后实时计算，不含确认环节再计算。 |
| 3.5 | **【全部提现】快速按钮** | 用户点击后，自动计算：<br>```<br>最大提现金额 = 可提现余额 - 总手续费\n```<br>自动填充至金额输入框，用户看到实际到账金额。 | 便利按钮，避免用户手工计算。 |
| 3.6 | **订单确认摘要** | 在用户点击"确认提现"前，显示确认框，包含：<br>- **提现金额**（大字体）<br>- **手续费**（小字，可展开详情）<br>- **到账金额**（绿色高亮）<br>- **提币地址别名 + 脱敏地址**<br>- **预计到账时间**："通常 1-2 小时"（视网络状况）<br>- **勾选框**："我已确认上述信息，同意提现" | 再次确认，降低误操作风险。 |

#### 模块四：身份安全验证 - **P0 修复：二次验证机制**

| 序号 | 功能点 | 详细说明 | 交互逻辑 |
| :--- | :--- | :--- | :--- |
| 4.1 | **验证规则判定** | 系统判断是否需要二次验证：<br>**情况 1：首次提现至新地址**<br>- 检查该地址是否在 `withdrawal_address_whitelist` 表中<br>- 如不存在 -> **必须** 进行二次验证<br>- 如存在且已验证 -> 跳过验证<br><br>**情况 2：24 小时内单笔提现 > 50,000 USDT**<br>- 需要进行二次验证<br><br>**情况 3：用户安全设置要求**<br>- 如用户开启"所有提现都验证"，每笔都验证 | 多层次安全机制，保护高额提现。 |
| 4.2 | **验证方式选择** | 用户可选择以下验证方式（至少一种）：<br>1. **短信验证**：发送 6 位验证码至用户注册手机<br>2. **邮件验证**：发送链接至用户注册邮箱<br>3. **Google Authenticator**：如用户已绑定<br>4. **安全问题**：用户在注册时设定的问题<br><br>**推荐顺序**：短信 > 邮件 > 谷歌认证器 | 用户选择后立即发送验证码。 |
| 4.3 | **验证码发送与输入** | **发送逻辑**：<br>- 前端点击"发送验证码"后，按钮进入冷却状态（60 秒倒计时）<br>- 防止频繁点击<br><br>**输入验证**：<br>- 6 位数字，有效期 10 分钟<br>- 输入错误 3 次后，账户临时锁定 5 分钟<br>- 提示"尝试次数过多，请稍后再试" | 严格的安全机制，防止暴力破解。 |
| 4.4 | **验证成功后的处理** | 验证码正确后：<br>1. 在 `withdrawal_address_whitelist` 表中标记该地址为**已验证**（verified = 1）<br>2. 记录验证时间<br>3. 关闭验证弹窗，自动进行下一步（调用提币接口）<br>4. 后续 24 小时内相同地址无需再验证 | 一经验证，后续快速操作。 |
| 4.5 | **验证超时处理** | 如用户 10 分钟内未输入验证码：<br>1. 验证码失效<br>2. 提示"验证码已失效，请重新申请"<br>3. 用户需点击"重新申请"，重新开始流程 | 安全考量，限制验证有效期。 |

#### 模块五：Safeheron 提币接口对接 - **冻结机制 + 数据库事务简化方案**

| 序号 | 功能点 | 详细说明 | 交互逻辑 |
| :--- | :--- | :--- | :--- |
| 5.1 | **冻结机制详解** | **核心流程**：冻结 → 调用 Safeheron → 确认 → 解冻<br><br>**冻结的三个保护作用**：<br>1. **防止重复提现**：用户多次点击同一提现按钮时，已冻结的金额无法再冻结<br>2. **防止余额被其他操作扣除**：冻结期间，该金额对其他操作（申购等）不可见<br>3. **自动恢复机制**：Safeheron 调用失败时立即解冻，无需人工干预<br><br>**冻结逻辑**：<br>```sql\n-- 第一步：冻结金额\nUPDATE account\nSET frozen_balance = frozen_balance + amount,\n    version = version + 1\nWHERE user_id = :uid AND type = 'WEALTH'\nAND balance >= amount  -- 验证可用余额足够\n```<br><br>**可用余额定义**：`available_balance = balance - frozen_balance`<br>用户在 UI 看到的"可用余额"已扣除冻结金额。 | 冻结在用户点击"确认提现"后立即执行，防止并发操作。 |
| 5.2 | **调用 Safeheron 提币 API** | **请求信息**：<br>- `wallet_id`：用户的 Safeheron 钱包 ID<br>- `to_address`：目标提币地址<br>- `amount`：提现金额（不含冻结金额标记）<br>- `coin_type`：币种（USDT）<br>- `chain`：区块链（TRC20/ERC20 等）<br>- `request_id`：幂等性 ID（UUID）<br><br>**响应信息**：<br>- `transaction_hash`：区块链交易哈希<br>- `safeheron_order_id`：Safeheron 内部订单 ID<br>- `status`：状态（SENT/CONFIRMED/FAILED）<br>- `network_fee`：矿工费实际值<br>- `created_at`：交易创建时间<br><br>**超时处理**：<br>- Safeheron API 超时或 HTTP 5xx：自动回退，保留冻结状态（1小时后定时任务自动解冻）<br>- 用户可重试提现（使用原 request_id 进行幂等检查） | 后端同步调用，30 秒超时。Safeheron 本身也支持幂等性检查。 |
| 5.3 | **提币成功 - 扣款与订单创建（事务）** | 当 Safeheron 返回 SENT 或 CONFIRMED 状态时：<br><br>```sql\n-- 事务开始 (REPEATABLE_READ 隔离级别)\nBEGIN TRANSACTION\n\n-- 1. 从账户扣款（从冻结金额转移到已扣款）\nUPDATE account\nSET frozen_balance = frozen_balance - amount,\n    balance = balance - amount,\n    version = version + 1\nWHERE user_id = :uid AND type = 'WEALTH'\nAND frozen_balance >= amount\n\n-- 2. 创建提现订单\nINSERT INTO withdrawal_order (\n  user_id, amount, network_fee, platform_fee, actual_amount,\n  to_address, chain, coin_type,\n  safeheron_order_id, transaction_hash,\n  status = 'SENT',  -- 已发送至链\n  created_at, updated_at\n) VALUES (...)\n\n-- 3. 记录账务流水\nINSERT INTO account_journal (\n  serial_no, user_id, account_id, amount, balance_snapshot,\n  biz_type = 'WITHDRAWAL_SUCCESS', ref_id = order_id, created_at\n) VALUES (...)\n\n-- 4. 更新提现请求状态\nUPDATE withdrawal_request\nSET status = 'SUCCESS', updated_at = NOW()\nWHERE request_id = :request_id\n\nCOMMIT TRANSACTION\n```<br><br>**关键保证**：<br>- 扣款和订单创建原子性，确保账户一致<br>- 幂等性：相同 request_id 再次调用时跳过，直接返回已有 order_id | 事务保证账户一致，无需分布式锁。 |
| 5.4 | **提币失败 - 立即解冻** | 若 Safeheron API 返回失败：<br><br>```sql\n-- 立即解冻（不需要事务，直接恢复）\nUPDATE account\nSET frozen_balance = frozen_balance - amount,\n    version = version + 1\nWHERE user_id = :uid AND type = 'WEALTH'\n\n-- 更新提现请求为失败\nUPDATE withdrawal_request\nSET status = 'FAILED', error_code = :error_code,\n    error_message = :error_msg, updated_at = NOW()\nWHERE request_id = :request_id\n```<br><br>**用户反馈**：<br>- 关闭 Loading，弹窗提示具体失败原因（如"钱包暂停提币"、"地址被冻结"）<br>- 提供【重试】按钮（使用原 request_id，后端会检查是否需要重新冻结）<br>- 如重试 3 次失败，提示"请联系客服处理"<br><br>**幂等性处理**：同一 request_id 失败后，用户可重新提交（后端会重新冻结） | 失败立即恢复，无需等待定时任务，用户体验最优。 |
| 5.5 | **提币超时 - 自动解冻（定时任务）** | 若冻结超过 1 小时仍未完成（可能是 Safeheron 故障或网络中断）：<br><br>```sql\n-- 每小时执行一次（CRON: 0 * * * * *）\nUPDATE account\nSET frozen_balance = frozen_balance - amount,\n    version = version + 1\nWHERE id IN (\n  SELECT account_id FROM withdrawal_order\n  WHERE status IN ('PENDING', 'VERIFYING')\n  AND created_at < DATE_SUB(NOW(), INTERVAL 1 HOUR)\n)\n\n-- 标记订单为超时（用户可重试）\nUPDATE withdrawal_order\nSET status = 'TIMEOUT'\nWHERE status IN ('PENDING', 'VERIFYING')\nAND created_at < DATE_SUB(NOW(), INTERVAL 1 HOUR)\n\n-- 发送用户通知\nSEND_NOTIFICATION(user_id, '您的提现申请已超时，本金已恢复，请重新申请')\n```<br><br>**用户体验**：<br>- 用户可在提现历史中看到"已超时"状态<br>- 点击"重试"按钮会重新发起申请（新 request_id）<br>- 本金自动回到账户，无需客服干预 | 全自动处理，无需人工干预，用户友好。 |
| 5.6 | **提币成功反馈** | API 返回成功后（扣款完成）：<br>1. 关闭 Loading 遮罩<br>2. Toast 提示"提现请求已提交，请稍候确认"<br>3. 自动进入"提现详情页"，显示订单信息和区块链确认进度<br>4. 发送邮件通知用户 | 用户需要看到确切的订单号和交易哈希。 |

#### 模块六：提现订单追踪与进度查看

| 序号 | 功能点 | 详细说明 | 交互逻辑 |
| :--- | :--- | :--- | :--- |
| 6.1 | **提现订单详情页** | 用户进入该页面可查看：<br>1. **订单基本信息**：<br>   - 订单号 (order_id)<br>   - 提现金额<br>   - 手续费（矿工费 + 平台费）<br>   - 到账金额<br>   - 申请时间<br><br>2. **提币信息**：<br>   - 目标地址（脱敏显示）<br>   - 区块链（TRC20/ERC20）<br>   - 交易哈希（可点击复制）<br>   - 交易哈希链接（跳转到区块浏览器）<br><br>3. **订单状态时间轴**：<br>   - 📝 申请中 (T 日)<br>   - ✅ 已发送 (T 日 14:30)<br>   - ⏳ 链上确认中 (T 日 14:35-14:45，依链速度）<br>   - ✅ 已确认 (T+1 日 10:00)<br><br>4. **操作按钮**：<br>   - 【复制地址】<br>   - 【查看交易】（跳转区块浏览器）<br>   - 【返回资产】 | 时间轴要清晰易读，让用户了解进度。 |
| 6.2 | **订单状态枚举** | 系统维护订单的完整生命周期：<br><br>| 状态 | 说明 |<br>|-----|------|<br>| PENDING | 待提交（用户填表但未确认）|<br>| VERIFYING | 身份验证中 |<br>| SENT | 已发送至链（区块链已收到）|<br>| CONFIRMING | 链上确认中（等待 N 个块确认）|<br>| CONFIRMED | 已确认（通常 6-12 个块确认）|<br>| COMPLETED | 已完成（最终状态）|<br>| FAILED | 提币失败（可重试或客服处理）|<br>| CANCELLED | 已取消（用户主动或超时自动取消）| | 订单始终处于某个确定状态。 |
| 6.3 | **订单历史列表** | 用户在"资产 -> 提现历史"可查看所有提现订单：<br>- 列表展示：**金额、状态、申请时间、完成时间**<br>- 按时间倒序排列<br>- 支持筛选（全部、处理中、已完成、已失败）<br>- 点击订单卡片进入详情页 | 便于用户追踪历史提现。 |
| 6.4 | **状态实时轮询更新** | 前端采用轮询机制查询订单状态：<br>- **初始轮询间隔**：5 秒<br>- **状态不变时逐步延长**：10 秒 -> 30 秒 -> 60 秒<br>- **状态变化时立即刷新**：如 SENT -> CONFIRMING<br>- **最终状态时停止轮询**：COMPLETED 或 FAILED<br><br>**可选方案**：使用 WebSocket 推送，实时通知状态变化 | 用户体验流畅，不刷新也能看到最新状态。 |
| 6.5 | **到账延迟提示** | 若订单长时间未确认（超过预计到账时间）：<br>1. 在订单详情页显示警告："这笔提现处理时间较长，可能因网络拥堵，请耐心等待"<br>2. 提供【客服帮助】入口<br>3. 邮件通知用户，提供交易哈希便于查询 | 关键的用户体验提升点。 |

#### 模块七：异常场景与边界条件

| 场景 | 处理方案 |
| :--- | :--- |
| **用户在提现过程中提现金额被其他操作扣除** | 在提币 API 调用前再次校验账户余额，如不足则提示"可提现余额已变化，请重新填写"，返回上一步。 |
| **用户输入的提币地址非法** | 前端正则校验，后端 Safeheron 也会校验，失败时提示"地址格式错误，请检查"。 |
| **用户输入错误的提币链** | 校验地址和链是否匹配（如 USDT-TRC20 地址必须选 TRC20 链），不匹配则提示"地址与选中链不符"。 |
| **提现至已被冻结的地址** | Safeheron 服务端会拒绝，返回错误，系统提示"该地址暂不可提现，请联系客服"。 |
| **日/月/年提现额度超限** | 前端输入时实时计算剩余额度，超限时禁用输入和按钮，提示"今日剩余可提现 X USDT"。 |
| **KYC 身份认证过期** | 提现时检查 user.kyc_status，过期则拦截，弹窗提示"请重新进行 KYC 认证后再提现"，提供【立即认证】链接。 |
| **Safeheron 服务不可用** | 提币接口超时或 HTTP 500，系统记录错误并告知用户"系统繁忙，请稍后重试"，提供【重试】按钮。 |
| **网络中断导致用户重复点击** | 依靠前端 request_id 去重 + 后端幂等检查，确保只进行一次提币。 |
| **交易在链上被拒绝** | Safeheron 会捕获链上错误并回调，系统自动标记订单为 FAILED，告知用户"交易失败，原因：[具体原因]"。 |

---

## 七、 非功能需求

1.  **安全性**：
    *   与 Safeheron 的通信必须使用服务端签名，严禁在前端直接调用 Safeheron API。
    *   提现地址必须存储加密状态（数据库加密）。
    *   二次验证的验证码不能在日志中记录，防止泄露。
    *   提现操作需记录审计日志，包括操作者、时间、金额、地址。
    *   实施 IP 限制：用户在异常 IP 首次提现需额外验证。
    *   实施风控规则：单日/单周/单月提现额度上限，异常金额自动告警。

2.  **性能**：
    *   提现申请提交到 Safeheron 返回成功期望在 **3 秒** 内完成。
    *   订单详情页加载期望在 **1 秒** 内。
    *   轮询更新期望延迟不超过 **5 秒**。
    *   支持批量查询历史订单，每页 20 条，支持分页。

3.  **可用性**：
    *   所有错误提示必须是用户友好的自然语言，不要技术术语。
    *   提供图文并茂的帮助文档，解释提现流程、矿工费、链确认等概念。
    *   支持 H5、iOS、Android 端样式一致。
    *   支持离线浏览历史订单（缓存本地）。

4.  **合规性**：
    *   提现记录必须保存 7 年以上，便于审计和税务申报。
    *   提现操作需留存审计日志，与对账单一致。
    *   用户协议中需明确说明：Safeheron 托管的资金不属于平台资产，用户对自有钱包地址和助记词负责。

---

## 八、 埋点设计

| 事件ID (Event ID) | 事件名称 | 参数名 (Key) | 参数说明 | 触发时机 |
| :--- | :--- | :--- | :--- | :--- |
| `click_withdraw` | 点击提现 | `source_page` | 来源页面 (assets/detail) | 用户点击提现按钮时 |
| `add_withdrawal_address` | 添加提币地址 | `chain_type` | 区块链类型（TRC20/ERC20 等） | 用户新增地址成功后 |
| `withdraw_address_verified` | 提币地址已验证 | `address_id` | 地址 ID，`verification_method` | 用户通过二次验证后 |
| `withdrawal_fee_calculated` | 提现手续费已计算 | `amount`, `fee`, `net_amount` | 提现金额、手续费、实际到账 | 用户输入金额后（实时） |
| `withdrawal_submit` | 提现订单已提交 | `order_id`, `amount`, `chain` | 订单 ID、金额、链类型 | 用户点击"确认提现"后 |
| `withdrawal_safeheron_api` | Safeheron 提币 API 调用 | `result` (success/fail), `error_code` | 调用结果、错误码 | 后端调用 Safeheron API 时 |
| `withdrawal_status_change` | 提现订单状态变化 | `order_id`, `old_status`, `new_status` | 订单 ID、旧状态、新状态 | 订单状态更新时（轮询或回调） |
| `withdrawal_complete` | 提现已完成 | `order_id`, `amount`, `duration_seconds` | 订单 ID、金额、从申请到完成的耗时 | 订单状态变为 COMPLETED 时 |
| `withdrawal_failed` | 提现已失败 | `order_id`, `fail_reason` | 订单 ID、失败原因 | 订单状态变为 FAILED 时 |
| `withdrawal_cancel` | 提现已取消 | `order_id`, `cancel_reason` | 订单 ID、取消原因 | 用户手动取消或自动超时时 |

---

## 九、 数据库设计

### 9.1 账户表更新（新增冻结余额字段）

在现有的 `account` 表中添加冻结余额相关字段：

```sql
-- 更新现有 account 表（理财账户系统设计中的账户表）
ALTER TABLE account ADD COLUMN (
  frozen_balance DECIMAL(32,16) DEFAULT 0 COMMENT '冻结余额（提现等待中）'
);

-- 添加生成列计算可用余额（可选，用于快速查询）
ALTER TABLE account ADD COLUMN (
  available_balance GENERATED ALWAYS AS (balance - frozen_balance) STORED COMMENT '可用余额 = 余额 - 冻结金额'
);

-- 添加索引以支持快速查询冻结信息
ALTER TABLE account ADD INDEX idx_frozen_balance (user_id, frozen_balance);
```

### 9.2 新增表结构

```sql
-- 1. 提币地址白名单表
CREATE TABLE withdrawal_address_whitelist (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT NOT NULL,

  address_alias VARCHAR(255) NOT NULL,  -- 用户自定义别名
  chain_type VARCHAR(32) NOT NULL,      -- TRC20/ERC20/BEP20
  wallet_address VARCHAR(255) NOT NULL, -- 实际钱包地址（加密存储）

  verified TINYINT DEFAULT 0,            -- 是否已验证（0/1）
  verified_at DATETIME,                  -- 验证时间
  verification_method VARCHAR(32),       -- 验证方式（SMS/EMAIL/GOOGLE）

  is_deleted TINYINT DEFAULT 0,          -- 软删除
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  INDEX idx_user_id (user_id),
  INDEX idx_chain_type (chain_type),
  UNIQUE KEY uk_user_address (user_id, wallet_address)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 2. 提现请求记录表（幂等性）
CREATE TABLE withdrawal_request (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT NOT NULL,

  request_id VARCHAR(64) NOT NULL UNIQUE, -- UUID，用于幂等性

  status ENUM('PROCESSING', 'SUCCESS', 'FAILED') DEFAULT 'PROCESSING',
  error_code VARCHAR(64),
  error_message TEXT,

  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_request_id (request_id),
  INDEX idx_user_id (user_id),
  INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 3. 提现订单表
CREATE TABLE withdrawal_order (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT NOT NULL,

  amount DECIMAL(32,16) NOT NULL,         -- 提现金额
  network_fee DECIMAL(32,16),              -- 矿工费
  platform_fee DECIMAL(32,16),             -- 平台手续费
  actual_amount DECIMAL(32,16),            -- 实际到账金额

  chain_type VARCHAR(32) NOT NULL,        -- TRC20/ERC20/BEP20
  coin_type VARCHAR(32) NOT NULL,         -- USDT/USDC 等
  to_address VARCHAR(255) NOT NULL,       -- 目标地址（加密）

  safeheron_order_id VARCHAR(64),         -- Safeheron 订单 ID
  transaction_hash VARCHAR(255),          -- 区块链交易哈希

  status ENUM(
    'PENDING',      -- 待提交
    'VERIFYING',    -- 验证中
    'SENT',         -- 已发送
    'CONFIRMING',   -- 确认中
    'CONFIRMED',    -- 已确认
    'COMPLETED',    -- 已完成
    'FAILED',       -- 失败
    'CANCELLED',    -- 已取消
    'TIMEOUT'       -- 已超时（冻结已解冻）
  ) DEFAULT 'PENDING',

  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  sent_at DATETIME,                       -- 发送至链的时间
  confirmed_at DATETIME,                  -- 链上确认时间
  completed_at DATETIME,                  -- 完成时间
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  INDEX idx_user_id (user_id),
  INDEX idx_status (status),
  INDEX idx_created_at (created_at),
  UNIQUE KEY uk_safeheron_order (safeheron_order_id),
  KEY idx_transaction_hash (transaction_hash)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 4. 提现二次验证记录表
CREATE TABLE withdrawal_verification (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT NOT NULL,
  withdrawal_order_id BIGINT NOT NULL,

  verification_method VARCHAR(32) NOT NULL, -- SMS/EMAIL/GOOGLE
  verification_code VARCHAR(255),           -- 加密存储

  attempts INT DEFAULT 0,                    -- 尝试次数
  max_attempts INT DEFAULT 3,                -- 最大尝试次数

  verified TINYINT DEFAULT 0,                -- 是否通过
  verified_at DATETIME,

  expires_at DATETIME,                       -- 验证码过期时间

  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_user_id (user_id),
  INDEX idx_order_id (withdrawal_order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 5. 冻结日志表（监控冻结/解冻过程）
CREATE TABLE withdrawal_freeze_log (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT NOT NULL,
  order_id BIGINT NOT NULL,

  amount DECIMAL(32,16) NOT NULL,         -- 冻结金额
  frozen_at DATETIME,                     -- 冻结时间
  released_at DATETIME,                   -- 解冻时间
  reason VARCHAR(64),                     -- 解冻原因：SUCCESS / FAILED / TIMEOUT

  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_user_id (user_id),
  INDEX idx_order_id (order_id),
  INDEX idx_reason (reason),
  INDEX idx_frozen_at (frozen_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 9.3 字段加密说明
- `wallet_address`：存储时需用对称加密（AES-256），查询时解密对比
- `verification_code`：短期高强度加密或 Hash，比对时需要额外安全校验
- `transaction_hash`：可明文存储（区块链公开数据）

## 十、 定时任务配置

```
-- 每小时执行一次：自动解冻超时订单（>1小时未完成）
CRON: 0 * * * * *
CALL auto_release_frozen_balance();

-- 每小时执行一次：同步 Safeheron 订单状态
CRON: 0 * * * * *
CALL sync_withdrawal_order_status();

-- 每日凌晨 1 点：日对账（本地订单 vs Safeheron 订单）
CRON: 0 1 * * * *
CALL daily_withdrawal_reconciliation();

-- 每小时执行一次：冻结余额健康检查（告警）
CRON: 0 * * * * *
CALL check_frozen_balance_health();

-- 每日凌晨 3 点：清理过期的幂等性记录（7天前的已完成请求）
CRON: 0 3 * * * *
DELETE FROM withdrawal_request
WHERE status IN ('SUCCESS', 'FAILED')
AND created_at < DATE_SUB(NOW(), INTERVAL 7 DAY);
```

### 10.1 冻结解冻定时任务详解

```sql
-- 自动解冻超时订单（>1小时未完成）
CREATE PROCEDURE auto_release_frozen_balance()
BEGIN
  DECLARE done INT DEFAULT FALSE;
  DECLARE order_id_var BIGINT;
  DECLARE user_id_var BIGINT;
  DECLARE amount_var DECIMAL(32,16);

  DECLARE order_cursor CURSOR FOR
    SELECT id, user_id, amount FROM withdrawal_order
    WHERE status IN ('PENDING', 'VERIFYING', 'SENT')
    AND created_at < DATE_SUB(NOW(), INTERVAL 1 HOUR);

  DECLARE CONTINUE HANDLER FOR NOT FOUND SET done = TRUE;

  OPEN order_cursor;

  release_loop: LOOP
    FETCH order_cursor INTO order_id_var, user_id_var, amount_var;
    IF done THEN LEAVE release_loop; END IF;

    -- 1. 更新账户，解冻金额
    UPDATE account
    SET frozen_balance = frozen_balance - amount_var,
        version = version + 1
    WHERE user_id = user_id_var AND type = 'WEALTH';

    -- 2. 标记订单为超时
    UPDATE withdrawal_order
    SET status = 'TIMEOUT', updated_at = NOW()
    WHERE id = order_id_var;

    -- 3. 记录冻结日志
    INSERT INTO withdrawal_freeze_log
    (user_id, order_id, amount, frozen_at, released_at, reason, created_at)
    VALUES (user_id_var, order_id_var, amount_var,
            (SELECT created_at FROM withdrawal_order WHERE id = order_id_var),
            NOW(), 'TIMEOUT', NOW());

    -- 4. 发送用户通知
    INSERT INTO user_notification (user_id, title, message, type)
    VALUES (user_id_var, '提现超时',
            '您的提现申请已超时，本金已恢复到账户，请重新申请',
            'WITHDRAWAL_TIMEOUT');
  END LOOP;

  CLOSE order_cursor;
END;
```

### 10.2 冻结余额健康检查

```sql
-- 监控冻结余额是否异常
CREATE PROCEDURE check_frozen_balance_health()
BEGIN
  DECLARE total_balance DECIMAL(32,16);
  DECLARE total_frozen DECIMAL(32,16);
  DECLARE frozen_ratio DECIMAL(5,2);

  -- 计算冻结率
  SELECT COALESCE(SUM(frozen_balance), 0) INTO total_frozen
  FROM account WHERE type = 'WEALTH';

  SELECT COALESCE(SUM(balance), 0) INTO total_balance
  FROM account WHERE type = 'WEALTH';

  IF total_balance > 0 THEN
    SET frozen_ratio = (total_frozen / total_balance) * 100;
  ELSE
    SET frozen_ratio = 0;
  END IF;

  -- 告警：冻结余额 > 总余额的 10%（说明解冻有问题）
  IF frozen_ratio > 10 THEN
    INSERT INTO system_alert_log
    (alert_type, severity, message, alert_data)
    VALUES ('FROZEN_BALANCE_HIGH', 'WARNING',
            CONCAT('冻结余额过高，比例: ', frozen_ratio, '%'),
            JSON_OBJECT('total_frozen', total_frozen, 'total_balance', total_balance));
  END IF;
END;
```

### 10.3 对账任务简化（无需两阶段提交）

```sql
-- 日对账：对比本地订单与 Safeheron 订单
CREATE PROCEDURE daily_withdrawal_reconciliation()
BEGIN
  -- 1. 检查所有已发送但未确认的订单
  SELECT * FROM withdrawal_order
  WHERE status IN ('SENT', 'CONFIRMING')
  AND created_at < DATE_SUB(NOW(), INTERVAL 24 HOUR)
  INTO @stale_orders;

  -- 2. 对于每笔订单，查询 Safeheron 确认状态
  -- （使用 safeheron_order_id 查询 Safeheron API）

  -- 3. 如果发现不一致（本地 SENT，但 Safeheron 已 FAILED）
  --    则更新本地订单状态，并解冻金额（如需）

  -- 4. 如果发现差异，记录告警供人工审查
  INSERT INTO reconciliation_alert_log
  (alert_type, order_id, description, created_at)
  VALUES ('WITHDRAWAL_STATUS_MISMATCH', @order_id, @description, NOW());
END;
```

---

## 十一、 项目规划（简化方案）

基于**冻结机制 + 数据库事务**的简化方案，开发时间从"2天"改为"5天"，更加现实可控。

*   **阶段一（1 天）**：数据库设计、Safeheron API 调试、冻结机制数据流设计。
*   **阶段二（2 天）**：后端实现（地址管理、冻结/解冻逻辑、幂等性、二次验证）。
*   **阶段三（1.5 天）**：前端实现（提现表单、地址管理、订单追踪、实时进度更新）。
*   **阶段四（1.5 天）**：联调测试、冻结机制验证、异常场景覆盖、对账测试。
*   **阶段五（1 天）**：灰度上线、监控告警配置、冻结余额健康检查。

**总计：5 天（40小时），相比原来的 10 天大幅简化**

### 关键改进说明

**从"复杂分布式事务"改为"简单冻结机制"的优势**：

| 维度 | 原方案（两阶段提交） | 新方案（冻结机制） |
|-----|------------------|------------------|
| **复杂度** | 极高（需分布式框架） | 低（单数据库事务） |
| **开发时间** | 2 周 | 5 天 |
| **可靠性** | 较高但实现复杂 | 高（无外部依赖） |
| **运维成本** | 高（需监控分布式锁） | 低（冻结自动解冻） |
| **用户体验** | 好（提币快） | 好（冻结快速反馈） |
| **容错能力** | 中（需手动处理） | 高（自动解冻超时） |

---

## 附录

### A. 用户协议补充条款

**第 X 条 提现风险声明**

用户确认理解并同意以下条款：
1. 用户提供的钱包地址和提币链必须由用户自行验证，平台对因用户提供错误地址导致的资金损失不负责任。
2. 用户的资金由 Safeheron 托管，Safeheron 按照其服务协议对资金进行保管。用户如对 Safeheron 有异议，应直接向 Safeheron 提出。
3. 区块链确认时间取决于网络拥堵情况，平台无法保证到账时间，但承诺在 24 小时内提交至链。
4. 提现手续费包括矿工费和平台费用，由用户承担。
5. 用户不得使用平台进行洗钱、传销等违法活动，否则平台有权冻结账户和上报有关部门。

### B. FAQ 文档

**Q1：为什么首次提现至新地址需要验证？**
A：这是安全措施，防止黑客盗用账号转走资金。一旦新地址通过验证，后续提现更便捷。

**Q2：提现手续费包括什么？**
A：包括区块链矿工费（由网络状况决定）和平台费（提现金额的 0.5%）。

**Q3：提现要多久到账？**
A：通常 1-2 小时，视区块链网络拥堵程度而定。可在订单详情查看交易哈希，在区块浏览器上查询确认进度。

**Q4：如果提现失败了怎么办？**
A：系统会自动回滚，资金返回账户。您可重新申请，或联系客服帮助。

---

**文档完成日期**：2026-01-09
**下一版本**：V1.1（支持定时提现、提现优化）
