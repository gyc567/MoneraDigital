# OpenSpec: 客户借款与收益模块 (Lending System)

## 1. 目标
实现核心的“借出资产赚取收益”功能，包括引导式申请、资产管理、收益实时预览及风险披露。

## 2. 功能设计
### 2.1 数据库 Schema (`lending_positions`)
- `id`: 唯一标识
- `user_id`: 关联用户
- `asset`: 币种 (BTC, ETH, USDT 等)
- `amount`: 借出金额 (Decimal)
- `duration_days`: 期限 (30, 90, 180, 360 天)
- `apy`: 约定的年化收益率
- `status`: 状态 (ACTIVE, COMPLETED, TERMINATED)
- `accrued_yield`: 累计产生收益
- `start_date`: 开始时间
- `end_date`: 预计到期时间

### 2.2 后端 API
- `POST /api/lending/apply`: 提交借出申请，计算 APY 并存入数据库。
- `GET /api/lending/positions`: 获取用户的借贷列表。
- `POST /api/lending/terminate`: 申请提前终止（计算手续费）。

### 2.3 前端组件
- `LendingApplication`: 引导式表单，包含实时收益计算器。
- `LendingManager`: 展示活跃和历史仓位。
- `RiskDisclosure`: 嵌入式的风险警示模块。

## 3. 设计原则 (KISS & Clean)
- **高内聚**: 所有利息计算公式封装在 `LendingService.ts`。
- **低耦合**: 借贷系统与用户系统仅通过 `user_id` 关联，不依赖复杂的中间件。
- **模式**: 使用单例 Service 模式处理业务逻辑。

## 4. 验证标准
- 100% 覆盖核心利息计算逻辑的单元测试。
- 模拟多币种、多期限的申请流程。
- 确保前端在提交申请后实时刷新仪表盘资产状态。
