#!/bin/bash

# ====================================================================
# Monera Digital - 前端启动脚本 (Vite + React)
# ====================================================================
# 功能:
#   - 检查 npm 依赖
#   - 启动 Vite 开发服务器 (端口 5001)
#   - 自动代理 /api 请求到后端
# ====================================================================

set -e

# 颜色
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}启动 Vite 前端开发服务器...${NC}"

# 检查 Node.js
if ! command -v node &> /dev/null; then
    echo "错误: Node.js 未安装"
    exit 1
fi

echo "Node.js 版本: $(node -v)"

# 安装依赖 (如果需要)
if [ ! -d "node_modules" ]; then
    echo "安装 npm 依赖..."
    npm install
fi

# 停止端口 5001 上的现有进程
if lsof -i :5001 > /dev/null 2>&1; then
    echo "停止端口 5001 上的现有进程..."
    kill $(lsof -t -i :5001) 2>/dev/null || true
    sleep 1
fi

# 启动 Vite
echo "启动前端 (端口 5001, Vite 配置默认)..."
npm run dev &

# 等待启动
sleep 5

# 检查
if curl -s -o /dev/null -w "%{http_code}" http://localhost:5001/ 2>/dev/null | grep -q "200"; then
    echo -e "${GREEN}✅ 前端启动成功: http://localhost:5001${NC}"
    echo ""
    echo "API 请求将自动代理到 http://localhost:8081"
else
    echo "前端可能还在启动中，请稍后访问 http://localhost:5001"
fi
