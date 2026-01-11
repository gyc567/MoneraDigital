#!/bin/bash

# ====================================================================
# Monera Digital - 后端启动脚本 (Go + Neon development 数据库)
# ====================================================================
# 功能:
#   - 检查 .env 配置
#   - 确保连接 development 数据库
#   - 编译并启动 Go 服务器
# ====================================================================

set -e

# 颜色
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}启动 Go 后端服务器...${NC}"

# 检查 .env
if [ ! -f ".env" ]; then
    echo "错误: .env 文件不存在"
    exit 1
fi

# 确保使用 development 数据库
if grep -q "/neondb?" .env 2>/dev/null; then
    echo "切换到 development 数据库..."
    sed -i '' 's|/neondb?|/development?|g' .env
fi

# 检查 DATABASE_URL
if ! grep -q "DATABASE_URL" .env; then
    echo "错误: .env 中缺少 DATABASE_URL"
    exit 1
fi

echo "当前数据库: $(grep DATABASE_URL .env | sed 's/.*\///' | sed 's/\?.*//')"

# 构建
echo "编译 Go 后端..."
go build -o /tmp/monera-server ./cmd/server/main.go

# 启动
export PORT=8081
export GIN_MODE=debug

echo "启动服务器 (端口 8081)..."
/tmp/monera-server &
SERVER_PID=$!

# 等待启动
sleep 3

# 检查
if curl -s http://localhost:8081/api/docs > /dev/null 2>&1; then
    echo -e "${GREEN}✅ 后端启动成功: http://localhost:8081${NC}"
    echo "PID: $SERVER_PID"
else
    echo "错误: 后端启动失败"
    exit 1
fi
