# OpenSpec: 用户提现系统

## 1. 目标
实现一个完整的用户提现系统，包括前端页面、后端API、安全验证和Safeheron集成。保持KISS设计原则，高内聚、低耦合，确保100%测试覆盖率。

## 2. 技术栈
- **前端**: React + TypeScript + Vite
- **后端**: Vercel Serverless Functions (TypeScript)
- **数据库**: Neon PostgreSQL + Drizzle ORM
- **安全**: JWT + 2FA (OTP/TOTP)
- **外部API**: Safeheron API (Coin Out)
- **测试**: Vitest (100% 逻辑覆盖率)

## 3. 数据库模式 (Schema)
现有表结构已包含withdrawals、withdrawal_addresses、address_verifications。

新增字段：
```sql
-- 添加手续费和到账金额字段
ALTER TABLE withdrawals ADD COLUMN fee_amount numeric(20,8);
ALTER TABLE withdrawals ADD COLUMN received_amount numeric(20,8);
ALTER TABLE withdrawals ADD COLUMN safeheron_tx_id text;
ALTER TABLE withdrawals ADD COLUMN chain text;
```

## 4. API 设计
### 现有API (需要扩展)
- `GET /api/withdrawals` - 获取提现历史
- `POST /api/withdrawals` - 发起提现

### 新增API
- `GET /api/withdrawals/fees?asset={asset}&amount={amount}&chain={chain}` - 计算手续费
- `POST /api/withdrawals/verify-2fa` - 二次验证 (新地址首次提现)
- `GET /api/safeheron/wallets` - 获取Safeheron钱包信息
- `POST /api/safeheron/coin-out` - 调用Safeheron Coin Out API

## 5. Safeheron 集成
### 认证
- 使用API Key + Secret进行HMAC签名认证
- 请求头包含: X-API-Key, X-Timestamp, X-Signature

### Coin Out API
```typescript
interface SafeheronCoinOutRequest {
  vaultId: string;
  assetId: string;
  amount: string;
  toAddress: string;
  note?: string;
}

interface SafeheronCoinOutResponse {
  txId: string;
  status: 'INIT' | 'PROCESSING' | 'COMPLETED' | 'FAILED';
}
```

## 6. 实现计划
### 6.1 基础设施
- 配置Safeheron API密钥环境变量
- 实现Safeheron客户端服务 (`src/lib/safeheron-service.ts`)
- 添加手续费计算逻辑
- 扩展数据库迁移脚本

### 6.2 后端逻辑
- `src/lib/safeheron-service.ts` - Safeheron API客户端
- `src/lib/withdrawal-fee-service.ts` - 手续费计算服务
- `src/lib/two-factor-verification-service.ts` - 二次验证服务
- 扩展 `withdrawal-service.ts` - 集成真实处理逻辑
- API路由扩展 - 手续费计算、二次验证

### 6.3 前端页面
- 修改 `src/pages/dashboard/Assets.tsx` - 添加提现入口按钮
- 扩展 `src/pages/dashboard/Withdraw.tsx` - 添加手续费显示、到账金额、链选择
- 添加二次验证模态框
- 提现状态实时更新

### 6.4 安全验证
- 新地址首次提现需要2FA验证
- 支持SMS/Email/Google Authenticator
- 验证失败后锁定账户一定时间

### 6.5 异常处理
- 网络超时重试机制
- 余额不足提示
- 地址格式验证
- Safeheron API错误处理

### 6.6 测试
- `safeheron-service.test.ts` - Safeheron API集成测试
- `withdrawal-fee-service.test.ts` - 手续费计算测试
- `two-factor-verification-service.test.ts` - 二次验证测试
- 扩展 `withdrawal-service.test.ts` - 完整提现流程测试
- E2E测试覆盖完整用户提现流程

## 7. 设计原则
- **KISS**: 每个服务职责单一，避免过度抽象
- **高内聚**: 相关功能集中在一个服务中
- **低耦合**: 服务间通过明确接口通信
- **100%测试覆盖率**: 所有业务逻辑都有单元测试
- **安全优先**: 所有用户输入验证，敏感操作需要额外验证

## 8. 验收标准
- [ ] 用户可以在资产页面点击"提现"进入提现流程
- [ ] 支持添加/编辑/删除提现地址，实现白名单管理
- [ ] 提现时显示手续费和预计到账金额
- [ ] 新地址首次提现需要二次验证
- [ ] 成功集成Safeheron Coin Out API
- [ ] 提现历史显示完整状态和区块链交易哈希
- [ ] 所有异常场景有友好提示
- [ ] 所有新增代码100%测试覆盖
- [ ] 不影响现有功能