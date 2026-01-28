# OpenSpec: 2FA Skip 404 错误修复（生产环境）

## 1. 问题描述

**用户报告**: 用户 `gyc567@gmail.com` 登录访问 `https://www.moneradigital.com/login`，点击 "Skip For Now" 按钮时报错：

```
POST https://www.moneradigital.com/api/auth/2fa/skip 404 (Not Found)
```

## 2. 根因分析

### 2.1 三个可能的原因排查

#### 原因1: 生产环境代码未更新 ❌
- **检查**: Git 日志显示代码已提交
- **检查**: Vercel 部署显示最新代码已部署
- **结论**: 不是这个问题

#### 原因2: Vercel部署配置问题 ❌
- **检查**: Vercel 只有一个 Serverless Function `api/[...route]`
- **检查**: 前端 API 路由返回正确的 404 消息（"No route found"）
- **结论**: Vercel 配置正确，不是这个问题

#### 原因3: 后端服务未更新 ✅
- **检查**: 直接访问 Replit 后端 `https://monera-digital--gyc567.replit.app/api/auth/2fa/skip`
- **结果**: 返回 404 "API endpoint not found"
- **结论**: **Replit 后端运行的是旧代码，没有 `Skip2FALogin` 路由**

### 2.2 问题确认

```bash
# 测试 Replit 后端（旧代码）
curl -X POST https://monera-digital--gyc567.replit.app/api/auth/2fa/skip
# 返回: {"error":"API endpoint not found"} HTTP 404

# 测试本地后端（新代码）
curl -X POST http://localhost:8081/api/auth/2fa/skip
# 返回: {"access_token":"..."} HTTP 200
```

## 3. 修复方案

### 3.1 方案A: 更新 Replit 后端（推荐）

在 Replit 上重新部署最新的 Go 后端代码。

**步骤**:
1. 在 Replit 上拉取最新代码
2. 重新构建 Go 后端
3. 重启服务

### 3.2 方案B: 切换到 Vercel 部署的后端

将 `BACKEND_URL` 指向 Vercel 部署的后端服务。

**缺点**: 需要额外的后端部署

### 3.3 方案C: 使用统一的 Vercel + Go 部署

将 Go 后端作为 Vercel 的 Serverless Function 部署。

**缺点**: 需要修改架构

## 4. 实施方案（方案A）

### 4.1 Replit 部署步骤

```bash
# 1. 在 Replit 控制台执行
cd ~/MoneraDigital
git pull origin main

# 2. 构建后端
go build -o server ./cmd/server

# 3. 重启服务
killall server
./server &
```

### 4.2 验证部署

```bash
# 测试 2FA skip 端点
curl -X POST https://monera-digital--gyc567.replit.app/api/auth/2fa/skip \
  -H "Content-Type: application/json" \
  -d '{"userId": 1}'

# 应该返回 200 和 access_token
```

## 5. 预防措施

### 5.1 自动化部署

建议设置 GitHub Actions 自动部署到 Replit：

```yaml
name: Deploy to Replit
on:
  push:
    branches: [ main ]
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Trigger Replit Deploy
        run: |
          curl -X POST ${{ secrets.REPLIT_DEPLOY_HOOK }}
```

### 5.2 健康检查

添加部署后的健康检查：

```bash
#!/bin/bash
# health-check.sh

ENDPOINTS=(
  "/health"
  "/api/auth/login"
  "/api/auth/2fa/skip"
)

for endpoint in "${ENDPOINTS[@]}"; do
  status=$(curl -s -o /dev/null -w "%{http_code}" "https://monera-digital--gyc567.replit.app$endpoint")
  if [ "$status" != "200" ] && [ "$status" != "401" ]; then
    echo "❌ $endpoint failed with status $status"
    exit 1
  fi
  echo "✅ $endpoint OK"
done
```

## 6. 设计原则

### KISS
- 直接更新 Replit 后端，保持架构简单
- 不引入额外的复杂度

### 高内聚低耦合
- 前端和后端独立部署
- 通过环境变量配置后端地址

### 100% 测试覆盖
- 后端已有完整的测试覆盖
- 部署后运行健康检查

### 不影响其他功能
- 只更新后端代码
- 保持前端不变

## 7. 验证清单

- [ ] Replit 后端代码已更新到最新
- [ ] Go 后端构建成功
- [ ] 服务重启成功
- [ ] 2FA skip 端点返回 200
- [ ] 生产环境功能正常

## 8. 总结

问题根源：**Replit 后端运行的是旧代码**，没有包含 `Skip2FALogin` 路由。

解决方案：**在 Replit 上重新部署最新的 Go 后端代码**。
