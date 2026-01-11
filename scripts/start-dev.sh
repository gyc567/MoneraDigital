#!/bin/bash

# ====================================================================
# Monera Digital - 本地开发环境启动脚本
# ====================================================================
# 用法:
#   ./scripts/start-backend.sh        # 只启动后端
#   ./scripts/start-frontend.sh       # 只启动前端
#   ./scripts/start-dev.sh            # 同时启动前后端
# ====================================================================

set -e  # 遇到错误立即退出

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $(date '+%H:%M:%S') - $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $(date '+%H:%M:%S') - $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $(date '+%H:%M:%S') - $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%H:%M:%S') - $1"
}

# 检查 .env 文件
check_env() {
    if [ ! -f ".env" ]; then
        log_error ".env 文件不存在!"
        log_info "请运行: cp .env.example .env"
        exit 1
    fi
    
    # 检查 DATABASE_URL
    if ! grep -q "DATABASE_URL" .env; then
        log_error ".env 文件中缺少 DATABASE_URL 配置!"
        exit 1
    fi
    
    # 确保使用 development 数据库
    if grep -q "/neondb?" .env; then
        log_warn "检测到使用 neondb 数据库，正在切换到 development..."
        sed -i '' 's|/neondb?|/development?|g' .env
        log_success "已切换到 development 数据库"
    fi
    
    log_success ".env 配置检查通过"
}

# 停止现有进程
stop_existing() {
    log_info "停止现有进程..."
    
    # 停止 Go 服务器
    if lsof -i :8081 > /dev/null 2>&1; then
        kill $(lsof -t -i :8081) 2>/dev/null || true
        log_info "已停止端口 8081 上的进程"
    fi
    
    # 停止 Vite 开发服务器 (端口 5001)
    if lsof -i :5001 > /dev/null 2>&1; then
        kill $(lsof -t -i :5001) 2>/dev/null || true
        log_info "已停止端口 5001 上的进程"
    fi
    
    sleep 1
}

# 启动后端
start_backend() {
    log_info "启动 Go 后端服务器..."
    
    # 检查 Go 是否安装
    if ! command -v go &> /dev/null; then
        log_error "Go 未安装，请先安装 Go 1.21+"
        exit 1
    fi
    
    # 构建后端
    log_info "编译 Go 后端..."
    go build -o /tmp/monera-server ./cmd/server/main.go
    
    # 导出环境变量
    export PORT=8081
    export GIN_MODE=debug
    
    # 启动服务器
    log_info "启动服务器 (端口 8081, 数据库: development)..."
    /tmp/monera-server &
    
    # 等待服务器启动
    sleep 3
    
    # 检查服务器是否运行
    if curl -s http://localhost:8081/api/docs > /dev/null 2>&1; then
        log_success "Go 后端服务器已启动: http://localhost:8081"
    else
        log_error "Go 后端服务器启动失败"
        exit 1
    fi
}

# 启动前端
start_frontend() {
    log_info "启动 Vite 前端开发服务器..."
    
    # 检查 Node.js 是否安装
    if ! command -v node &> /dev/null; then
        log_error "Node.js 未安装，请先安装 Node.js 20+"
        exit 1
    fi
    
    # 安装依赖 (如果需要)
    if [ ! -d "node_modules" ]; then
        log_info "安装 npm 依赖..."
        npm install
    fi
    
    log_info "启动前端 (端口 5001)..."
    npm run dev -- --port 5001 &
    
    # 等待服务器启动
    sleep 5
    
    # 检查服务器是否运行
    if curl -s -o /dev/null -w "%{http_code}" http://localhost:5001/ 2>/dev/null | grep -q "200"; then
        log_success "Vite 前端服务器已启动: http://localhost:5001"
    else
        log_warn "前端服务器可能还在启动中，请稍后访问 http://localhost:5001"
    fi
}

# 主函数
main() {
    echo -e "${BLUE}"
    echo "╔════════════════════════════════════════════════════════╗"
    echo "║          Monera Digital 本地开发环境启动脚本            ║"
    echo "╚════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    
    # 解析参数
    case "${1:-all}" in
        backend)
            check_env
            start_backend
            ;;
        frontend)
            start_frontend
            ;;
        all|"")
            check_env
            stop_existing
            start_backend
            start_frontend
            echo ""
            log_success "=========================================="
            log_success "开发环境已完全启动!"
            log_success "=========================================="
            log_info "前端: http://localhost:5001"
            log_info "后端: http://localhost:8081"
            log_info "API:  http://localhost:8081/api/auth/login"
            echo ""
            log_warn "按 Ctrl+C 停止所有服务"
            ;;
        help|-h|--help)
            echo "用法: $0 [命令]"
            echo ""
            echo "命令:"
            echo "  all       启动前后端 (默认)"
            echo "  backend   只启动后端"
            echo "  frontend  只启动前端"
            echo "  help      显示帮助信息"
            echo ""
            ;;
        *)
            log_error "未知命令: $1"
            echo "运行 $0 help 查看帮助"
            exit 1
            ;;
    esac
}

main "$@"
