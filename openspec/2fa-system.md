# OpenSpec: 谷歌双因素认证 (2FA) 系统

## 1. 目标
通过引入基于时间的一次性密码 (TOTP) 增强账户安全性，支持谷歌验证码 (Google Authenticator)。

## 2. 功能设计
### 2.1 数据库扩展 (`users` 表)
- `two_factor_secret`: 存储加密后的 TOTP 密钥。
- `two_factor_enabled`: 布尔值，标识是否开启 2FA。

### 2.2 后端 API
- `POST /api/auth/2fa/setup`: 生成新的 TOTP 密钥并返回 QR Code 数据链接（Base64）。
- `POST /api/auth/2fa/enable`: 验证用户输入的第一个验证码，验证通过后正式开启 2FA。
- `POST /api/auth/2fa/disable`: 关闭 2FA。
- 修改 `POST /api/auth/login`: 如果用户开启了 2FA，登录第一步仅返回临时状态，要求进入第二步验证。

### 2.3 业务逻辑 (`TwoFactorService`)
- 使用 `otplib` 处理 TOTP 生成与校验。
- 使用 `qrcode` 生成前端展示的二维码。

## 3. 设计原则 (KISS)
- **解耦**: 2FA 逻辑独立于核心登录逻辑，通过 `requires2FA` 标志位进行流程分发。
- **内聚**: 所有密钥生成、验证码校验逻辑集中在 `TwoFactorService.ts`。

## 4. 验证标准
- 100% 覆盖密钥生成与校验逻辑的单元测试。
- 模拟非法验证码、过期验证码的拒绝场景。
