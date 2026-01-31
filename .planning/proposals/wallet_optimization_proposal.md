# Wallet API 优化提案

## 概述

优化钱包创建流程，解决两个问题：
1. 前端冗余的地址获取调用
2. wallet/create 端点安全问题

## 问题分析

### 问题 1: 前端冗余调用

**现状**: 创建钱包成功后，前端立即调用 `getAddressMutation` 再次获取地址。

**原因**: `WalletService.CreateWallet()` 已将地址保存到 `user_wallets` 表，地址应直接从 `walletInfo` 获取，无需二次调用。

**影响**: 额外的网络请求，增加延迟和服务器负载。

### 问题 2: 认证安全问题

**现状**: `wallet/create` 在 `public` 组，无需认证即可调用。

**风险**: 未登录用户可创建钱包，可能导致：
- 垃圾钱包数据
- 用户身份混淆
- 安全漏洞

**建议**: 移到 `protected` 组，要求登录。

## 变更内容

### 1. 前端优化 (AccountOpening.tsx)

```typescript
// 移除前
onSuccess: () => {
  queryClient.invalidateQueries({ queryKey: ["walletInfo"] });
  getAddressMutation.mutate();  // 冗余调用
}

// 修改后
onSuccess: () => {
  queryClient.invalidateQueries({ queryKey: ["walletInfo"] });
  // 直接使用 walletInfo 中的地址，无需额外请求
}
```

### 2. 路由安全 (routes.go)

```go
// 现状
wallet := public.Group("/wallet")
{
    wallet.POST("/create", h.CreateWallet)
}

// 修改后
wallet := protected.Group("/wallet")
{
    wallet.POST("/create", h.CreateWallet)
}
```

### 3. API 路由配置 (api/[...route].ts)

```typescript
// 现状
'POST /wallet/create': { requiresAuth: false, backendPath: '/api/wallet/create' }

// 修改后
'POST /wallet/create': { requiresAuth: true, backendPath: '/api/wallet/create' }
```

## 架构原则

- **KISS**: 简单移除冗余代码
- **高内聚**: 认证逻辑集中在 middleware
- **低耦合**: 前端直接使用已有数据源

## 测试要求

- 前端测试：验证 `walletInfo` 数据正确显示
- 后端测试：验证未认证请求被拒绝 (401)
- 不影响现有功能

## 风险评估

- **低风险**: 仅代码清理和认证增强
- **兼容性问题**: 已有未登录用户调用需处理
