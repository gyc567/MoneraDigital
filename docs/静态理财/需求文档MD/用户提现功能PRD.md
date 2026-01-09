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

### 1. 业务流程图

> **主流程**：
> 用户进入资产页面 -> 查看可提现余额 -> 点击"提现"按钮 -> **进入提现申请页** -> 选择提币链 & 地址 -> 输入提现金额 -> **计算手续费** -> 确认订单 -> **身份验证** (如需) -> 提交提现请求 -> **调用 Safeheron API** -> **订单创建成功** -> 前端展示提现处理中 -> **轮询查询订单状态** -> 交易上链 -> **提现完成** -> 用户邮件/APP 通知。

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

#### 模块五：Safeheron 提币接口对接 - **P0 修复：幂等性与异常处理**

| 序号 | 功能点 | 详细说明 | 交互逻辑 |
| :--- | :--- | :--- | :--- |
| 5.1 | **幂等性保护** | **前端生成 request_id (UUID)**，每次提现请求携带。<br><br>**后端逻辑**：<br>1. 检查 `withdrawal_request` 表，查询是否已有相同 request_id 的记录<br>2. **如已存在**：<br>   - 状态为 SUCCESS：直接返回已有订单信息（交易哈希等）<br>   - 状态为 PROCESSING：返回"提现处理中，请稍候"<br>   - 状态为 FAILED：允许用户重新提交（新 request_id）<br>3. **如不存在**：创建新的 PROCESSING 记录，继续流程<br><br>**目的**：防止网络中断或用户重复点击导致的重复提币 | 必须保证幂等性，避免重复转账。 |
| 5.2 | **调用 Safeheron 提币 API** | **请求信息**：<br>- `wallet_id`：用户的 Safeheron 钱包 ID（来自账户开户）<br>- `to_address`：目标提币地址<br>- `amount`：提现金额<br>- `coin_type`：币种（USDT）<br>- `chain`：区块链（TRC20/ERC20 等）<br>- `request_id`：幂等性 ID<br><br>**响应信息**：<br>- `transaction_hash`：区块链交易哈希<br>- `safeheron_order_id`：Safeheron 内部订单 ID<br>- `status`：状态（SENT/CONFIRMED/FAILED）<br>- `network_fee`：矿工费实际值<br>- `created_at`：交易创建时间 | 与 Safeheron 服务端签名，前端无法直接调用。 |
| 5.3 | **订单创建与存储** | 提币 API 返回成功后，在本地数据库创建订单记录：<br><br>```sql\nINSERT INTO withdrawal_order (\n  user_id, amount, fee_amount, actual_amount,\n  to_address, chain, coin_type,\n  safeheron_order_id, transaction_hash,\n  status = 'SENT',  -- 已发送至链\n  created_at, updated_at\n)\n```<br><br>同时：<br>- 从用户理财账户扣款（需要二阶段提交）<br>- 更新 `withdrawal_request` 表为 SUCCESS<br>- 记录账务流水（biz_type = WITHDRAWAL_SUCCESS） | 双重记录，确保数据完整性。 |
| 5.4 | **提币失败处理** | 若 Safeheron API 返回失败或超时：<br><br>1. **记录失败日志**：<br>   - 更新 `withdrawal_request` 表为 FAILED<br>   - 记录错误信息和错误代码<br><br>2. **恢复用户余额**：<br>   - 撤销扣款操作（回滚）<br>   - 用户账户余额恢复<br><br>3. **用户反馈**：<br>   - 关闭 Loading，弹窗提示具体失败原因（如 "钱包暂停提币"、"金额超出限额"）<br>   - 提供【重试】按钮（可使用新 request_id 重试，或原 request_id）<br>   - 如重试 3 次失败，提示"请联系客服处理" | 失败不应导致资金丢失，必须可追踪。 |
| 5.5 | **提币成功反馈** | API 返回成功后：<br>1. 关闭 Loading 遮罩<br>2. Toast 提示"提现请求已提交，请稍候确认"<br>3. 自动进入"提现详情页"，显示订单信息和区块链确认进度<br>4. 发送邮件通知用户 | 用户需要看到确切的订单号和交易哈希。 |

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

### 9.1 新增表结构

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
    'CANCELLED'     -- 已取消
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
```

### 9.2 字段加密说明
- `wallet_address`：存储时需用对称加密（AES-256），查询时解密对比
- `verification_code`：短期高强度加密或 Hash，比对时需要额外安全校验
- `transaction_hash`：可明文存储（区块链公开数据）

---

## 十、 定时任务配置

```
-- 每小时执行一次：同步 Safeheron 订单状态
CRON: 0 * * * * *
CALL sync_withdrawal_order_status();

-- 每日凌晨 2 点：检查超时订单（未完成超过 24 小时）
CRON: 0 2 * * * *
CALL check_withdrawal_timeout_orders();

-- 每日生成对账单
CRON: 0 1 * * * *
CALL generate_withdrawal_reconciliation_report();

-- 定期清理过期验证码（每 10 分钟）
CRON: */10 * * * * *
CALL cleanup_expired_verification_codes();
```

---

## 十一、 项目规划

*   **阶段一（2 天）**：数据库设计、Safeheron API 调试、UI 原型设计。
*   **阶段二（3 天）**：后端实现（地址管理、订单创建、幂等性、安全验证）。
*   **阶段三（2 天）**：前端实现（提现表单、地址管理、订单追踪）。
*   **阶段四（2 天）**：联调测试、异常场景覆盖、安全审查。
*   **阶段五（1 天）**：灰度上线、监控告警配置。

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
