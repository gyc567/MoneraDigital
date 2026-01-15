# 钱包管理与充值 API 接口规范 (对接核心账户系统)

## 概述

本文档定义了 **账户开通**（创建钱包）和 **充值**（入金）功能的 REST API 接口标准。这些接口由 Monera Digital 后端 (`internal/`) 提供，底层与**核心账户系统** (Core Account System) 对接。

通过分析现有数据库架构，确认系统目前仅包含 `withdrawals` (提现) 表，缺乏存储入金记录的机制。因此，**必须新增 `deposits` 表** 来独立管理充值账目，以确保资金流向（进/出）的清晰分离和审计。

## 1. 支持的网络与资产

系统目前支持主流公链及稳定币充值。

### 1.1 支持的公链 (Chains)
*   **Ethereum (ETH)**: 主网
*   **Ethereum Layer 2**: Arbitrum, Optimism
*   **Tron (TRX)**: 波场网络
*   **Solana (SOL)**: Solana 网络
*   **Avalanche (AVAX)**: C-Chain
*   **BNB Smart Chain (BSC)**: BEP20
*   **OKTC**: OKT Chain
*   **Base**: Coinbase L2

### 1.2 支持的资产 (Assets)
*   **USDT**: Tether USD
*   **USDC**: USD Coin

## 2. 账户开通（创建钱包）

"账户开通"流程在核心账户系统中为用户注册并分配专属的充值地址。由于支持多链，系统可能会为不同链分配相同或不同的地址。

### 2.1 创建/激活钱包
发起在核心账户系统的开户请求。

*   **接口地址：** `POST /api/wallet/create`
*   **认证方式：** `Bearer Token` (用户)
*   **请求体：** `None` (默认开通所有支持的链)
*   **响应 (200 OK):**
    ```json
    {
      "requestId": "req_core_123456",
      "status": "CREATING" // 或 "SUCCESS"
    }
    ```
*   **响应 (400/409):**
    *   `409 Conflict`: "Wallet already exists or creation in progress"

### 2.2 获取钱包信息
查询用户在核心系统的钱包状态及各链的充值地址。

*   **接口地址：** `GET /api/wallet/info`
*   **认证方式：** `Bearer Token` (用户)
*   **响应 (200 OK):**
    ```json
    {
      "status": "SUCCESS", // "NONE", "CREATING", "SUCCESS", "FAILED"
      "walletId": "uid_10001", // 核心系统中的用户ID或钱包ID
      "addresses": [
        {
          "chain": "ETH",
          "address": "0x1234567890abcdef1234567890abcdef12345678"
        },
        {
          "chain": "TRON",
          "address": "TJCnKsPa7y5okkXvQAidZBzqx3QyQ6sxMW"
        },
        // ... 其他链地址
      ]
    }
    ```

## 3. 充值（入金）

允许用户查询和追踪他们的充值记录。

### 3.1 获取充值历史
获取该用户的所有充值入账记录。

*   **接口地址：** `GET /api/deposits`
*   **认证方式：** `Bearer Token` (用户)
*   **查询参数：**
    *   `limit`: (可选) 默认 20
    *   `offset`: (可选) 默认 0
    *   `asset`: (可选) 筛选币种 (e.g., USDT)
    *   `chain`: (可选) 筛选链 (e.g., TRON)
    *   `status`: (可选) 筛选状态 (e.g., CONFIRMED)
*   **响应 (200 OK):**
    ```json
    {
      "total": 10,
      "deposits": [
        {
          "id": 101,
          "txHash": "0xabc...",
          "amount": "5000.000000",
          "asset": "USDT",
          "chain": "ETH",
          "fromAddress": "0xsender...",
          "toAddress": "0xuserWallet...",
          "status": "CONFIRMED", // "PENDING", "CONFIRMED", "FAILED"
          "confirmations": 12,
          "createdAt": "2026-01-13T10:00:00Z",
          "confirmedAt": "2026-01-13T10:05:00Z"
        }
      ]
    }
    ```

### 3.2 核心系统充值 Webhook
接收来自核心账户系统的资金入账通知。**注意：此接口由核心系统调用，而非前端。**

*   **接口地址：** `POST /api/webhooks/core/deposit`
*   **认证方式：** `X-Core-Signature` 签名验证 (HMAC-SHA256)。
*   **请求体：**
    ```json
    {
      "eventId": "evt_987654",
      "eventType": "DEPOSIT_CONFIRMED",
      "data": {
        "userId": "user_123", // Monera 系统的用户ID
        "externalUserId": "10001", // 核心系统内部ID
        "txHash": "0x...",
        "amount": "1000.00",
        "asset": "USDT",
        "chain": "ETH",
        "fromAddress": "...",
        "toAddress": "...",
        "blockNumber": 12345678,
        "timestamp": 1600000000
      }
    }
    ```
*   **响应 (200 OK):** `{ "success": true }`

## 4. 数据库结构更新

### 4.1 `deposits` 表 (新增)
**分析结论：** 现有数据库仅包含 `withdrawals` (出金) 表，无法复用。入金 (Deposit) 是外部触发的被动事件，数据结构与出金不同（无审核拒绝状态，需记录确认数等）。为了保证账目清晰，必须新增此表。

| 字段名 | 类型 | 说明 |
| :--- | :--- | :--- |
| `id` | `SERIAL` | 主键 |
| `user_id` | `INTEGER` | 外键，关联 `users.id` |
| `tx_hash` | `TEXT` | 链上交易哈希 (唯一索引) |
| `amount` | `NUMERIC` | 充值金额 |
| `asset` | `TEXT` | 资产符号 (USDT, USDC) |
| `chain` | `TEXT` | 链名称 (ETH, TRON, SOL, BSC, AVAX, OP, ARB, BASE, OKTC) |
| `status` | `ENUM` | 'PENDING', 'CONFIRMED', 'FAILED' |
| `from_address` | `TEXT` | 发送方地址 |
| `to_address` | `TEXT` | 接收方地址 |
| `created_at` | `TIMESTAMP` | 记录创建时间 |
| `confirmed_at` | `TIMESTAMP` | 确认时间 |

### 4.2 `wallet_creation_requests` 表 (复用现有)
复用现有表，利用 `addresses` 字段存储多链地址。

*   `addresses` 字段 (TEXT/JSON) 用于存储地址映射。
*   `status` 为 `SUCCESS` 表示钱包已开通。

**JSON 结构:**
```json
{
  "ETH": "0x...",
  "TRON": "T..."
}
```

## 5. 实施计划

1.  **数据库 (Database):**
    *   创建 `deposits` 表迁移脚本。
    *   执行迁移。
2.  **后端 (Backend - Go):**
    *   **Models/DTOs**: 定义 `Deposit` 结构体。
    *   **WalletService**: 实现 `CreateWallet` (调用核心API) 和 `GetWalletInfo` (读DB)。
    *   **DepositService**: 实现 `GetDeposits` (读DB) 和 `HandleWebhook` (写DB)。
    *   **Routes**: 注册路由。
3.  **前端 (Frontend - React):**
    *   **AccountOpening.tsx**: 对接开户接口。
    *   **Assets.tsx**: 展示多链地址和充值记录。
