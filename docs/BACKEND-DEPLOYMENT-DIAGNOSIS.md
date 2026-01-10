# 前后端集成问题诊断报告

## 问题描述

部署到Vercel后，前端发送API请求返回错误：
```
POST https://www.moneradigital.com/api/auth/register 405 (Method Not Allowed)
Non-JSON response received
Registration error: Error: 注册失败
```

## 诊断结果

### ✅ 已验证正常
- Vercel前端部署成功 (GET / 返回200)
- Vercel配置正确（已添加API代理规则）
- 前端代码正确（使用/api/auth/register）

### ❌ 发现的问题
- 后端API端点返回404（无论直接访问还是通过Vercel代理）
- 后端Go服务器的API路由未正确启动
- 后端可能没有正确初始化或启动失败

## 根本原因分析

### 问题1: 后端Go服务器可能未正确启动
**症状**: API端点返回404
**原因**:
- 后端构建失败
- 后端启动失败
- 后端数据库连接失败
- 后端配置缺失

### 问题2: 环境变量配置不完整
**症状**: 后端无法连接数据库或初始化
**原因**:
- DATABASE_URL未设置或无效
- JWT_SECRET未设置
- PORT配置错误

### 问题3: 后端依赖缺失
**症状**: 后端构建或运行时出错
**原因**:
- Go依赖未安装
- 数据库驱动缺失
- 第三方库版本不匹配

## 解决方案

### 步骤1: 检查后端日志
在Replit上查看后端日志文件：
```bash
cat backend.log
cat backend.out
```

### 步骤2: 验证环境变量
确保Replit上设置了以下环境变量：
```
DATABASE_URL=postgresql://...  # PostgreSQL连接字符串
JWT_SECRET=...                 # 最少32字节的密钥
PORT=8081                      # 后端端口
```

### 步骤3: 手动启动后端进行调试
```bash
# 在Replit上运行
export PORT=8081
export DATABASE_URL="your-database-url"
export JWT_SECRET="your-jwt-secret"
go run cmd/server/main.go
```

### 步骤4: 验证后端API端点
```bash
# 测试注册端点
curl -X POST https://monera-digital--gyc567.replit.app/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"TestPassword123!"}'

# 应该返回201或400（而不是404）
```

## 推荐的修复步骤

### 方案A: 在Replit上重新启动后端（推荐）
1. 登录Replit
2. 打开MoneraDigital项目
3. 检查.env文件中的环境变量
4. 运行: `npm run dev` 或 `bash scripts/start-replit.sh`
5. 查看后端日志
6. 修复任何错误

### 方案B: 部署后端到Vercel（长期解决方案）
1. 将后端API端点部署到Vercel Serverless Functions
2. 更新前端API代理指向Vercel后端
3. 优点：前后端都在Vercel，更稳定
4. 缺点：需要重新配置

### 方案C: 使用Docker容器（企业级解决方案）
1. 为后端创建Dockerfile
2. 部署到云服务（AWS, GCP, Azure等）
3. 配置前端代理指向后端
4. 优点：更灵活，更可靠
5. 缺点：需要更多配置

## 检查清单

- [ ] 检查Replit上的后端日志
- [ ] 验证环境变量是否正确设置
- [ ] 确认数据库连接字符串有效
- [ ] 测试后端API端点是否返回正确的状态码
- [ ] 验证前端是否能通过Vercel代理访问后端
- [ ] 测试完整的注册和登陆流程

## 预期结果

修复后，API请求应该返回：
```
POST /api/auth/register
状态码: 201 (成功) 或 400 (验证错误)
响应体: {"message": "User created successfully", "user": {...}}
```

而不是：
```
状态码: 404 或 405
响应体: 空或错误信息
```
