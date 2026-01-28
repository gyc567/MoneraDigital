# Bug 修复提案：登录时 JSON 解析错误

## 问题描述

用户登录时报错：
```
vendor-core-D9WIbsMN.js:38 Uncaught SyntaxError: "undefined" is not valid JSON
    at JSON.parse (<anonymous>)
    at index-SXlQ7EAX.js:1240:29826
```

## 根本原因分析

1. **Login.tsx 第 80 行**:
   ```typescript
   localStorage.setItem("user", JSON.stringify(data.user));
   ```
   如果 `data.user` 是 `undefined`，`JSON.stringify(undefined)` 返回 `"undefined"`（字符串）

2. **Header.tsx 第 18 行** 和 **DashboardLayout.tsx 第 19 行**:
   ```typescript
   setUser(JSON.parse(savedUser));
   ```
   尝试解析 `"undefined"` 字符串，导致 JSON 解析错误

## 修复方案

### 方案 1: 前端防御性编程（推荐）

在存储和解析 user 数据时添加空值检查：

1. **Login.tsx**: 只在 `data.user` 存在时存储
2. **Header.tsx**: 添加 try-catch 和空值检查
3. **DashboardLayout.tsx**: 添加 try-catch 和空值检查

### 方案 2: 后端保证 User 字段

确保后端始终返回 User 字段（已在代码中实现，但可能被 API 路由修改）

## 实施计划

1. 修复 Login.tsx 添加空值检查
2. 修复 Header.tsx 添加错误处理
3. 修复 DashboardLayout.tsx 添加错误处理
4. 添加单元测试验证修复
5. 运行回归测试

## 代码变更

### Login.tsx
```typescript
// 修复前
localStorage.setItem("user", JSON.stringify(data.user));

// 修复后
if (data.user) {
  localStorage.setItem("user", JSON.stringify(data.user));
}
```

### Header.tsx & DashboardLayout.tsx
```typescript
// 修复前
setUser(JSON.parse(savedUser));

// 修复后
if (savedUser && savedUser !== "undefined") {
  try {
    setUser(JSON.parse(savedUser));
  } catch (e) {
    console.error("Failed to parse user data:", e);
    localStorage.removeItem("user");
  }
}
```

## 测试策略

1. 测试正常登录流程
2. 测试 user 数据缺失的情况
3. 测试 user 数据损坏的情况
4. 验证所有使用 localStorage user 的组件

## 回归测试

- [ ] 登录功能正常
- [ ] Header 显示用户邮箱
- [ ] DashboardLayout 正常加载
- [ ] 刷新页面后用户状态保持
