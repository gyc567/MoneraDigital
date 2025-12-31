# OpenSpec: 修复 Vercel 注册接口 500 错误

## 1. 问题描述
用户在尝试注册新用户时，`POST /api/auth/register` 返回 500 错误。
错误代码：`FUNCTION_INVOCATION_FAILED`
现象：前端收到 "A server error has occurred"，后端函数无法正常执行。

## 2. 根本原因分析 (假设)
- **Top-level Exception**: `AuthService` 在导入时检查 `JWT_SECRET`，如果环境变量未定义则抛出异常。
- **Dependency Mismatch**: `bcryptjs` 版本号异常 (`3.0.3`)，可能导致库无法加载。
- **Path Resolution**: API 目录下的 Serverless 函数引用 `src` 目录下的逻辑时可能存在路径解析问题。

## 3. 解决方案计划
1.  **依赖修复**: 将 `bcryptjs` 回退到稳定的 `2.4.3` 版本，并确保类型定义匹配。
2.  **代码健壮性**:
    - 在 `AuthService` 中将环境变量检查从“顶层抛错”改为“运行时检查”或提供默认值（仅限开发环境）。
    - 统一 API 导入路径。
3.  **日志增强**: 在 API 处理函数中增加更详细的错误捕获，确保即使发生错误也能返回 JSON 格式的错误信息而非 Vercel 的通用 500 页面。
4.  **环境检查**: 提示用户在 Vercel Dashboard 中检查环境变量配置。

## 4. 验证标准
- 本地 `npm run build` 通过。
- 模拟 API 请求不再产生 `FUNCTION_INVOCATION_FAILED`。
