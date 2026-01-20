# OpenSpec: 用户提现功能 (User Withdrawal Feature)

> 版本: 1.0.0  
> 状态: 已实现 (Implemented)  
> 最后更新: 2024-01-20

## 1. 概述

本规范描述了 Monera Digital 平台的用户提现功能实现，包括前端页面、后端 API 和数据模型。

### 1.1 功能范围
- 用户发起提现请求
- 选择提现地址和链类型
- 计算和显示网络手续费
- 提现历史记录
- 2FA 安全验证（新地址首次提现）

### 1.2 技术栈
| 层级 | 技术 |
|------|------|
| 前端 | React 18, TypeScript, Tailwind CSS |
| 后端 | Go, Gin Framework |
| 数据库 | PostgreSQL (Neon) |
| ORM | Drizzle ORM |
| 外部服务 | Safeheron (Mock) |

## 2. 数据模型

### 2.1 核心表

```sql
-- 提现订单表
CREATE TABLE withdrawal_order (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  amount NUMERIC(65, 30) NOT NULL,
  network_fee NUMERIC(65, 30),
  platform_fee NUMERIC(65, 30),
  actual_amount NUMERIC(65, 30),
  chain_type TEXT NOT NULL,
  coin_type TEXT NOT NULL,
  to_address TEXT NOT NULL,
  safeheron_order_id TEXT,
  transaction_hash TEXT,
  status TEXT DEFAULT 'PENDING',
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
  sent_at TIMESTAMP WITH TIME ZONE,
  confirmed_at TIMESTAMP WITH TIME ZONE,
  completed_at TIMESTAMP WITH TIME ZONE,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);
```

### 2.2 状态流转

```
PENDING → PROCESSING → SENT → CONFIRMING → CONFIRMED → COMPLETED
                                      ↓
                               FAILED (任何阶段)
```

## 3. API 规范

### 3.1 API Endpoints

| 方法 | 路径 | 描述 | 认证 |
|------|------|------|------|
| GET | `/api/withdrawals` | 获取提现历史 | ✅ |
| POST | `/api/withdrawals` | 创建提现订单 | ✅ |
| GET | `/api/withdrawals/fees` | 估算手续费 | ✅ |
| GET | `/api/withdrawals/:id` | 获取单个提现详情 | ✅ |

### 3.2 请求/响应格式

#### 创建提现请求
```json
POST /api/withdrawals
Content-Type: application/json
Authorization: Bearer <token>

{
  "address_id": 123,
  "amount": "100.00",
  "asset": "USDT"
}
```

#### 响应
```json
{
  "message": "Withdrawal created",
  "order": {
    "id": 456,
    "amount": "100.00",
    "status": "PENDING",
    "created_at": "2024-01-20T12:00:00Z"
  }
}
```

#### 手续费估算
```json
GET /api/withdrawals/fees?asset=USDT&amount=100&chain=Ethereum

{
  "fee": "2.00",
  "receivedAmount": "98.00"
}
```

## 4. 前端实现

### 4.1 页面组件

```
src/pages/dashboard/
├── Withdraw.tsx          # 提现页面主组件
├── Deposit.tsx           # 存款页面
└── ...
```

### 4.2 组件结构

```typescript
// 状态定义
interface WithdrawalAddress {
  id: number;
  wallet_address: string;
  chain_type: string;
  address_alias: string;
  verified: boolean;
}

interface Withdrawal {
  id: number;
  status: WithdrawalStatus;
  amount: string;
  coin_type: string;
  to_address: string;
  transaction_hash: string | null;
  network_fee: string | null;
  actual_amount: string | null;
  created_at: string;
  completed_at: string | null;
}
```

### 4.3 侧边栏菜单

```typescript
const menuItems = [
  // ...
  { icon: ArrowDownCircle, label: t("dashboard.nav.deposit"), href: "/dashboard/deposit" },
  { icon: ArrowLeftRight, label: t("dashboard.nav.lending"), href: "/dashboard/lending" },
  { icon: ArrowUpCircle, label: t("dashboard.nav.withdraw"), href: "/dashboard/withdraw" },
  // ...
];
```

### 4.4 国际化键

```json
{
  "dashboard": {
    "nav": {
      "withdraw": "Withdraw"
    },
    "withdraw": {
      "title": "Withdraw Funds",
      "selectAddress": "Select Withdrawal Address",
      "selectDescription": "Choose a verified address to withdraw funds to",
      "withdrawAmount": "Withdrawal Amount",
      "confirmTitle": "Confirm Withdrawal"
    }
  }
}
```

## 5. 后端实现

### 5.1 服务层

```go
// internal/services/withdrawal_service.go

type WithdrawalService struct {
  repo     *repository.Repository
  safeheron ISafeheronService
  db       *sql.DB
}

func (s *WithdrawalService) CreateWithdrawal(ctx context.Context, userID int, req models.CreateWithdrawalRequest) (*models.WithdrawalOrder, error) {
  // 1. Validate Input
  // 2. Get Account
  // 3. Check Balance
  // 4. Get Address details
  // 5. Freeze Balance
  // 6. Call Safeheron
  // 7. Create Order
}

func (s *WithdrawalService) EstimateFee(ctx context.Context, asset, chain, amount string) (string, string, error) {
  // Mock implementation
  return networkFee, receivedAmount, nil
}
```

### 5.2 Mock Safeheron 服务

```go
// internal/services/safeheron_service.go

type SafeheronService struct{}

func (s *SafeheronService) Withdraw(ctx context.Context, req SafeheronWithdrawalRequest) (*SafeheronWithdrawalResponse, error) {
  // Mock implementation - returns success
  return &SafeheronWithdrawalResponse{
    SafeheronOrderID: "mock-sh-id",
    TxHash:           "mock-tx-hash",
    NetworkFee:       "1.0",
  }, nil
}
```

### 5.3 Handler 层

```go
// internal/handlers/handlers.go

func (h *Handler) CreateWithdrawal(c *gin.Context) {
  var req models.CreateWithdrawalRequest
  if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
  }

  userID, _ := c.Get("userID")
  order, err := h.WithdrawalService.CreateWithdrawal(c.Request.Context(), userID.(int), req)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
  }

  c.JSON(http.StatusCreated, gin.H{"message": "Withdrawal created", "order": order})
}
```

## 6. 安全考虑

### 6.1 2FA 验证
- 新地址首次提现需要 2FA 验证
- 验证通过后才能发起提现

### 6.2 余额检查
- 提现金额不能超过可用余额
- 提现前先冻结相应金额

### 6.3 地址验证
- 只有已验证的地址才能用于提现

## 7. 测试用例

### 7.1 单元测试
- 手续费计算
- 余额检查逻辑
- 状态转换

### 7.2 E2E 测试
- 用户登录
- 发起提现
- 查看提现历史
- 2FA 验证流程

## 8. 已验证功能

| 功能 | 状态 | 说明 |
|------|------|------|
| 侧边栏菜单 | ✅ 已实现 | ArrowUpCircle 图标已添加 |
| 提现页面 | ✅ 已实现 | 完整的 UI 和逻辑 |
| API 路由 | ✅ 已配置 | /api/withdrawals/* |
| 服务层 | ✅ 已实现 | WithdrawalService |
| Mock 服务 | ✅ 已实现 | SafeheronService |

## 9. 待改进项

### 9.1 高优先级
- [ ] 完善数据库迁移（确保表结构同步）
- [ ] 添加提现确认邮件通知

### 9.2 中优先级
- [ ] 提现限额功能
- [ ] 提现手续费实时计算
- [ ] 批量提现支持

### 9.3 低优先级
- [ ] 提现二维码生成
- [ ] 提现到账时间预测

## 10. 参考文档

- [Drizzle ORM 文档](https://orm.drizzle.team/)
- [Safeheron API 文档](https://docs.safeheron.com/)
- [React Query 最佳实践](https://tanstack.com/query/latest)

---

> 本文档由 AI 自动生成，如有更新请同步修改。
