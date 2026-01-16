# Core Account System API - OpenSpec 功能提案

## 文档信息

| 属性 | 值 |
|------|-----|
| 版本 | 1.0.0 |
| 日期 | 2026-01-16 |
| 作者 | Monera Digital Engineering Team |
| 状态 | 待审批 |
| API版本 | v1 |

---

## 1. 执行摘要

本文档定义了核心账户系统（Core Account System）的API接口规范，用于Monera Digital平台的账户注册和管理功能。该系统将作为外部核心账户系统的Mock实现，用于开发、测试和演示目的。

**主要功能:**
- 账户创建（Account Creation）
- 账户查询（Account Query）
- 账户状态管理（Account Status Management）
- KYC验证集成（KYC Verification）

---

## 2. 概述

### 2.1 背景

Monera Digital是一个机构级数字资产平台，需要与核心账户系统进行集成以实现：
- 用户开户流程
- 账户信息同步
- 身份验证（KYC）
- 账户状态管理

由于核心账户系统尚未就绪，本文档定义了Mock接口规范，以便前端和后端开发可以并行进行。

### 2.2 范围

本文档涵盖以下API接口：
- `POST /api/core/accounts/create` - 创建账户
- `GET /api/core/accounts/:accountId` - 查询账户详情
- `PUT /api/core/accounts/:accountId/status` - 更新账户状态
- `POST /api/core/accounts/:accountId/kyc/submit` - 提交KYC信息
- `GET /api/core/accounts/:accountId/kyc/status` - 查询KYC状态

### 2.3 术语定义

| 术语 | 定义 |
|------|------|
| Account ID | 核心账户系统的唯一账户标识符 |
| External ID | 外部系统（如Monera Digital）的用户标识 |
| KYC | Know Your Customer，客户身份验证流程 |
| Status | 账户状态（CREATING, ACTIVE, SUSPENDED, CLOSED） |

---

## 3. API接口规范

### 3.1 通用信息

#### 3.1.1 Base URL

```
开发环境: http://localhost:8081/api/core
生产环境: https://core-api.monera.digital/api/v1
```

#### 3.1.2 认证方式

所有API请求需要包含以下Header：
```
X-Core-API-Key: YOUR_API_KEY
X-Request-ID: UUID-v4
X-Timestamp: Unix Timestamp (seconds)
X-Signature: HMAC-SHA256签名
```

#### 3.1.3 通用响应格式

```json
{
  "success": true,
  "data": {
    // 响应数据
  },
  "meta": {
    "requestId": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": 1705392000
  }
}
```

#### 3.1.4 错误响应格式

```json
{
  "success": false,
  "error": {
    "code": "ACCOUNT_CREATION_FAILED",
    "message": "账户创建失败",
    "details": {
      "field": "email",
      "reason": "Email格式无效"
    }
  },
  "meta": {
    "requestId": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": 1705392000
  }
}
```

#### 3.1.5 错误码定义

| 错误码 | HTTP状态码 | 说明 |
|--------|------------|------|
| SUCCESS | 200 | 成功 |
| ACCOUNT_NOT_FOUND | 404 | 账户不存在 |
| ACCOUNT_CREATION_FAILED | 400 | 账户创建失败 |
| INVALID_REQUEST | 400 | 请求参数无效 |
| UNAUTHORIZED | 401 | 未授权 |
| FORBIDDEN | 403 | 禁止访问 |
| RATE_LIMIT_EXCEEDED | 429 | 超过速率限制 |
| INTERNAL_ERROR | 500 | 内部错误 |

---

### 3.2 创建账户接口

#### 3.2.1 接口信息

| 属性 | 值 |
|------|-----|
| 端点 | `POST /accounts/create` |
| 方法 | POST |
| Content-Type | application/json |
| 认证 | Required |

#### 3.2.2 请求参数

```json
{
  "externalId": "user_12345",
  "accountType": "INDIVIDUAL",
  "profile": {
    "email": "user@example.com",
    "phone": "+8613800138000",
    "firstName": "John",
    "lastName": "Doe",
    "dateOfBirth": "1990-01-15",
    "nationality": "CN",
    "address": {
      "street": "123 Main Street",
      "city": "Shanghai",
      "state": "Shanghai",
      "postalCode": "200000",
      "country": "CN"
    }
  },
  "metadata": {
    "source": "web",
    "referralCode": "REF123"
  }
}
```

#### 3.2.3 参数说明

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| externalId | string | 是 | 外部系统用户ID，最大50字符 |
| accountType | string | 是 | 账户类型: INDIVIDUAL, CORPORATE |
| profile.email | string | 是 | 邮箱地址 |
| profile.phone | string | 否 | 电话号码 |
| profile.firstName | string | 是 | 名 |
| profile.lastName | string | 是 | 姓 |
| profile.dateOfBirth | string | 否 | 出生日期 (YYYY-MM-DD) |
| profile.nationality | string | 是 | 国籍代码 (ISO 3166-1 alpha-2) |
| profile.address | object | 否 | 邮寄地址 |
| metadata | object | 否 | 元数据 |

#### 3.2.4 响应示例

**成功响应 (201 Created):**
```json
{
  "success": true,
  "data": {
    "accountId": "core_550e8400e29b41d4a716446655440000",
    "externalId": "user_12345",
    "accountType": "INDIVIDUAL",
    "status": "CREATING",
    "profile": {
      "email": "user@example.com",
      "phone": "+8613800138000"
    },
    "createdAt": "2026-01-16T09:09:09Z",
    "updatedAt": "2026-01-16T09:09:09Z"
  },
  "meta": {
    "requestId": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": 1705392000
  }
}
```

---

### 3.3 查询账户详情接口

#### 3.3.1 接口信息

| 属性 | 值 |
|------|-----|
| 端点 | `GET /accounts/:accountId` |
| 方法 | GET |
| 认证 | Required |

#### 3.3.2 路径参数

| 参数 | 类型 | 说明 |
|------|------|------|
| accountId | string | 核心账户系统生成的账户ID |

#### 3.3.3 响应示例

**成功响应 (200 OK):**
```json
{
  "success": true,
  "data": {
    "accountId": "core_550e8400e29b41d4a716446655440000",
    "externalId": "user_12345",
    "accountType": "INDIVIDUAL",
    "status": "ACTIVE",
    "profile": {
      "email": "user@example.com",
      "phone": "+8613800138000",
      "firstName": "John",
      "lastName": "Doe",
      "dateOfBirth": "1990-01-15",
      "nationality": "CN"
    },
    "kycStatus": "VERIFIED",
    "walletIds": ["wallet_abc123", "wallet_def456"],
    "createdAt": "2026-01-16T09:09:09Z",
    "updatedAt": "2026-01-16T09:10:09Z"
  },
  "meta": {
    "requestId": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": 1705392000
  }
}
```

---

### 3.4 更新账户状态接口

#### 3.4.1 接口信息

| 属性 | 值 |
|------|-----|
| 端点 | `PUT /accounts/:accountId/status` |
| 方法 | PUT |
| 认证 | Required |

#### 3.4.2 请求参数

```json
{
  "status": "ACTIVE",
  "reason": "KYC verification completed"
}
```

#### 3.4.3 状态枚举

| 状态 | 说明 |
|------|------|
| CREATING | 创建中 |
| PENDING_KYC | 等待KYC验证 |
| ACTIVE | 激活可用 |
| SUSPENDED | 暂停使用 |
| CLOSED | 已关闭 |
| REJECTED | 被拒绝 |

---

### 3.5 KYC提交接口

#### 3.5.1 接口信息

| 属性 | 值 |
|------|-----|
| 端点 | `POST /accounts/:accountId/kyc/submit` |
| 方法 | POST |
| 认证 | Required |

#### 3.5.2 请求参数

```json
{
  "documentType": "PASSPORT",
  "documentNumber": "G12345678",
  "documentFrontImage": "base64_encoded_image",
  "documentBackImage": "base64_encoded_image",
  "selfieImage": "base64_encoded_image",
  "addressProof": {
    "type": "UTILITY_BILL",
    "documentImage": "base64_encoded_image"
  }
}
```

---

### 3.6 KYC状态查询接口

#### 3.6.1 接口信息

| 属性 | 值 |
|------|-----|
| 端点 | `GET /accounts/:accountId/kyc/status` |
| 方法 | GET |
| 认证 | Required |

#### 3.6.2 响应示例

```json
{
  "success": true,
  "data": {
    "accountId": "core_550e8400e29b41d4a716446655440000",
    "kycStatus": "VERIFIED",
    "kycLevel": 2,
    "verificationDate": "2026-01-16T10:00:00Z",
    "expiresAt": "2027-01-16T10:00:00Z",
    "documents": [
      {
        "type": "PASSPORT",
        "status": "APPROVED",
        "submittedAt": "2026-01-16T09:30:00Z",
        "verifiedAt": "2026-01-16T10:00:00Z"
      }
    ]
  },
  "meta": {
    "requestId": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": 1705392000
  }
}
```

---

## 4. 数据模型

### 4.1 账户模型

```typescript
interface CoreAccount {
  accountId: string;
  externalId: string;
  accountType: 'INDIVIDUAL' | 'CORPORATE';
  status: AccountStatus;
  profile: AccountProfile;
  kycStatus: KYCStatus;
  walletIds: string[];
  metadata: Record<string, any>;
  createdAt: string;
  updatedAt: string;
}

type AccountStatus = 
  | 'CREATING'
  | 'PENDING_KYC'
  | 'ACTIVE'
  | 'SUSPENDED'
  | 'CLOSED'
  | 'REJECTED';

type KYCStatus = 
  | 'NOT_SUBMITTED'
  | 'PENDING'
  | 'IN_REVIEW'
  | 'VERIFIED'
  | 'REJECTED';
```

---

## 5. 安全性

### 5.1 认证流程

所有API请求需要使用HMAC-SHA256进行签名：

```
Signature = HMAC-SHA256(
  API_SECRET_KEY,
  HTTP_METHOD + "\n" +
  REQUEST_PATH + "\n" +
  TIMESTAMP + "\n" +
  REQUEST_BODY
)
```

### 5.2 速率限制

| 端点 | 限制 |
|------|------|
| 创建账户 | 10次/分钟/IP |
| 查询账户 | 60次/分钟/IP |
| KYC提交 | 5次/分钟/IP |

---

## 6. 实施计划

### 6.1 Phase 1: Mock实现（当前阶段）

1. 创建Mock HTTP Handler
2. 实现内存存储
3. 添加基本验证
4. 单元测试覆盖

### 6.2 Phase 2: 集成测试

1. 与后端服务集成
2. E2E测试验证
3. 性能测试

### 6.3 Phase 3: 生产就绪

1. 添加数据库持久化
2. 实现完整签名验证
3. 添加监控和日志
4. 负载测试

---

## 7. 测试用例

### 7.1 单元测试

| 用例ID | 测试场景 | 预期结果 |
|--------|----------|----------|
| TC001 | 有效请求创建账户 | 返回201，账户状态为CREATING |
| TC002 | 无效邮箱格式 | 返回400，错误码INVALID_REQUEST |
| TC003 | 重复externalId | 返回400，错误码ACCOUNT_EXISTS |
| TC004 | 查询不存在的账户 | 返回404，错误码ACCOUNT_NOT_FOUND |
| TC005 | 有效的状态更新 | 返回200，状态更新成功 |

### 7.2 E2E测试

| 用例ID | 测试场景 | 预期结果 |
|--------|----------|----------|
| E2E001 | 完整注册流程 | 账户创建成功，状态变为ACTIVE |
| E2E002 | KYC提交和验证 | KYC状态变为VERIFIED |
| E2E003 | 并发账户创建 | 所有请求正确处理 |

---

## 8. 附录

### 8.1 Mock实现配置

```yaml
core_api:
  enabled: true
  port: 8081
  base_path: /api/core
  storage: memory  # memory, postgres, redis
  
  mock:
    delay_ms: 100  # 模拟网络延迟
    error_rate: 0  # 模拟错误率 (0-1)
    
  rate_limit:
    create_account: 10/minute
    query_account: 60/minute
```

### 8.2 版本历史

| 版本 | 日期 | 作者 | 变更说明 |
|------|------|------|----------|
| 1.0.0 | 2026-01-16 | Monera Team | 初始版本 |

---

## 9. 审批

| 角色 | 姓名 | 签字 | 日期 |
|------|------|------|------|
| 产品负责人 | | | |
| 技术负责人 | | | |
| QA负责人 | | | |

---

*本文档由Monera Digital Engineering Team生成*
*如有问题，请联系: engineering@monera.digital*
