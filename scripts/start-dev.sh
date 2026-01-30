#!/bin/bash

# ====================================================================
# Monera Digital - 本地开发环境启动脚本 (优化版)
# ====================================================================
# 用法:
#   ./scripts/start-dev.sh              # 启动前后端 (默认)
#   ./scripts/start-dev.sh start        # 启动前后端
#   ./scripts/start-dev.sh restart      # 重启前后端
#   ./scripts/start-dev.sh stop         # 停止所有服务
#   ./scripts/start-dev.sh status       # 查看服务状态
#   ./scripts/start-dev.sh backend      # 只启动后端
#   ./scripts/start-dev.sh frontend     # 只启动前端
#   ./scripts/start-dev.sh help         # 显示帮助
# ====================================================================

# 注意：不使用 set -e，因为需要处理后台进程的正常退出

# 获取脚本所在目录并切换到项目根目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_DIR"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 端口定义
BACKEND_PORT=8081
FRONTEND_PORT=5001

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

# 取消代理设置（解决 localhost 访问问题）
unset_proxy() {
    unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY
}

# 检查服务是否运行
is_backend_running() {
    curl -s -o /dev/null -w "%{http_code}" "http://localhost:${BACKEND_PORT}/health" 2>/dev/null | grep -q "200"
}

is_frontend_running() {
    curl -s -o /dev/null -w "%{http_code}" "http://localhost:${FRONTEND_PORT}/" 2>/dev/null | grep -q "200"
}

# 验证服务健康状态
verify_backend() {
    log_info "验证后端服务..."
    local max_attempts=10
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if is_backend_running; then
            log_success "后端服务健康检查通过: http://localhost:${BACKEND_PORT}"
            return 0
        fi
        sleep 1
        ((attempt++))
    done
    
    log_error "后端服务健康检查失败"
    return 1
}

verify_frontend() {
    log_info "验证前端服务..."
    local max_attempts=10
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if is_frontend_running; then
            log_success "前端服务健康检查通过: http://localhost:${FRONTEND_PORT}"
            return 0
        fi
        sleep 1
        ((attempt++))
    done
    
    log_error "前端服务健康检查失败"
    return 1
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
    
    log_success ".env 配置检查通过"
}

# 停止现有进程
stop_existing() {
    log_info "停止现有进程..."
    
    # 停止 Go 服务器
    if lsof -i :${BACKEND_PORT} > /dev/null 2>&1; then
        kill $(lsof -t -i :${BACKEND_PORT}) 2>/dev/null || true
        log_info "已停止端口 ${BACKEND_PORT} 上的进程"
        sleep 1
    fi
    
    # 停止 Vite 开发服务器
    if lsof -i :${FRONTEND_PORT} > /dev/null 2>&1; then
        kill $(lsof -t -i :${FRONTEND_PORT}) 2>/dev/null || true
        log_info "已停止端口 ${FRONTEND_PORT} 上的进程"
        sleep 1
    fi
    
    # 额外清理：停止任何残留的 node 和 go 进程
    pkill -f "vite" 2>/dev/null || true
    pkill -f "monera-server" 2>/dev/null || true
    
    log_success "进程清理完成"
}

# 停止单个服务
stop_backend() {
    if lsof -i :${BACKEND_PORT} > /dev/null 2>&1; then
        kill $(lsof -t -i :${BACKEND_PORT}) 2>/dev/null || true
        log_info "已停止后端服务"
        sleep 1
    else
        log_info "后端服务未运行"
    fi
}

stop_frontend() {
    if lsof -i :${FRONTEND_PORT} > /dev/null 2>&1; then
        kill $(lsof -t -i :${FRONTEND_PORT}) 2>/dev/null || true
        log_info "已停止前端服务"
        sleep 1
    else
        log_info "前端服务未运行"
    fi
}

# 启动后端
start_backend() {
    # 检查是否已运行
    if is_backend_running; then
        log_warn "后端服务已在运行 (端口 ${BACKEND_PORT})"
        verify_backend
        return 0
    fi
    
    log_info "启动 Go 后端服务器..."
    
    # 取消代理设置
    unset_proxy
    
    # 检查 Go 是否安装
    if ! command -v go &> /dev/null; then
        log_error "Go 未安装，请先安装 Go 1.21+"
        exit 1
    fi
    
    # 构建后端
    log_info "编译 Go 后端..."
    if ! go build -o /tmp/monera-server ./cmd/server/main.go; then
        log_error "Go 后端编译失败"
        exit 1
    fi
    
    # 导出环境变量
    export PORT=${BACKEND_PORT}
    export GIN_MODE=debug
    export MONNAIRE_CORE_API_URL=http://198.13.57.142:8080
    # 从 .env 文件读取 JWT_SECRET，确保前后端使用相同的密钥
    export JWT_SECRET=$(grep "JWT_SECRET=" .env | cut -d '=' -f2-)

    # 启动服务器
    log_info "启动服务器 (端口 ${BACKEND_PORT}, 数据库: neondb, Core API: http://198.13.57.142:8080)..."
    /tmp/monera-server &
    
    # 验证服务
    if verify_backend; then
        return 0
    else
        log_error "Go 后端服务器启动失败"
        return 1
    fi
}

# 启动前端
start_frontend() {
    # 检查是否已运行
    if is_frontend_running; then
        log_warn "前端服务已在运行 (端口 ${FRONTEND_PORT})"
        verify_frontend
        return 0
    fi
    
    log_info "启动 Vite 前端开发服务器..."
    
    # 取消代理设置
    unset_proxy
    
    # 检查 Node.js 是否安装
    if ! command -v node &> /dev/null; then
        log_error "Node.js 未安装，请先安装 Node.js 20+"
        exit 1
    fi
    
    # 安装依赖 (如果需要)
    if [ ! -d "node_modules" ]; then
        log_info "安装 npm 依赖..."
        npm install || {
            log_error "npm install 失败"
            exit 1
        }
    fi
    
    log_info "启动前端 (端口 ${FRONTEND_PORT}, Vite 配置默认)..."
    npm run dev &
    
    # 验证服务
    if verify_frontend; then
        return 0
    else
        log_warn "前端服务启动中，请稍后访问 http://localhost:${FRONTEND_PORT}"
        return 1
    fi
}

# 查看服务状态
show_status() {
    echo -e "${CYAN}════════════════════════════════════════════════════════${NC}"
    echo -e "${CYAN}             Monera Digital 服务状态                   ${NC}"
    echo -e "${CYAN}════════════════════════════════════════════════════════${NC}"
    echo ""
    
    # 检查后端
    if is_backend_running; then
        echo -e "  ${GREEN}●${NC} 后端服务    : 运行中 (http://localhost:${BACKEND_PORT})"
    else
        echo -e "  ${RED}●${NC} 后端服务    : 未运行"
    fi
    
    # 检查前端
    if is_frontend_running; then
        echo -e "  ${GREEN}●${NC} 前端服务    : 运行中 (http://localhost:${FRONTEND_PORT})"
    else
        echo -e "  ${RED}●${NC} 前端服务    : 未运行"
    fi
    
    echo ""
    
    # 总体状态
    if is_backend_running && is_frontend_running; then
        echo -e "${GREEN}所有服务运行正常 ✓${NC}"
    elif is_backend_running || is_frontend_running; then
        echo -e "${YELLOW}部分服务运行中${NC}"
    else
        echo -e "${RED}所有服务已停止${NC}"
    fi
    echo ""
}

# 启动所有服务
start_all() {
    check_env
    
    echo -e "${BLUE}"
    echo "╔════════════════════════════════════════════════════════╗"
    echo "║          Monera Digital 本地开发环境启动脚本            ║"
    echo "╚════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    
    # 取消代理设置
    unset_proxy
    
    start_backend
    start_frontend
    
    echo ""
    echo -e "${GREEN}════════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}              开发环境已完全启动!                       ${NC}"
    echo -e "${GREEN}════════════════════════════════════════════════════════${NC}"
    echo ""
    log_info "前端: http://localhost:${FRONTEND_PORT}"
    log_info "后端: http://localhost:${BACKEND_PORT}"
    log_info "API:  http://localhost:${BACKEND_PORT}/api/auth/login"
    echo ""
    log_warn "按 Ctrl+C 停止所有服务"
    echo ""
    
    # 等待所有后台进程
    trap '' SIGINT SIGTERM
    wait
}

# 重启所有服务
restart_all() {
    echo -e "${BLUE}"
    echo "╔════════════════════════════════════════════════════════╗"
    echo "║          Monera Digital 本地开发环境重启脚本            ║"
    echo "╚════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    
    unset_proxy
    
    log_info "正在重启服务..."
    stop_existing
    sleep 2
    
    start_all
}

# 主函数
main() {
    # 解析参数
    case "${1:-start}" in
        start)
            start_all
            ;;
        restart)
            restart_all
            ;;
        stop)
            echo -e "${BLUE}╔════════════════════════════════════════════════════════╗${NC}"
            echo -e "${BLUE}║              停止所有服务                             ║${NC}"
            echo -e "${BLUE}╚════════════════════════════════════════════════════════╝${NC}"
            unset_proxy
            stop_existing
            log_success "所有服务已停止"
            ;;
        status)
            show_status
            ;;
        backend)
            check_env
            start_backend
            ;;
        frontend)
            start_frontend
            ;;
        help|-h|--help|"")
            echo -e "${BLUE}"
            echo "╔════════════════════════════════════════════════════════╗"
            echo "║          Monera Digital 开发环境管理脚本               ║"
            echo "╚════════════════════════════════════════════════════════╝"
            echo -e "${NC}"
            echo ""
            echo "用法: $0 [命令]"
            echo ""
            echo "命令:"
            echo "  start    启动前后端 (默认)"
            echo "  restart  重启前后端"
            echo "  stop     停止所有服务"
            echo "  status   查看服务状态"
            echo "  backend  只启动后端"
            echo "  frontend 只启动前端"
            echo "  help     显示帮助信息"
            echo ""
            echo "示例:"
            echo "  $0              # 启动所有服务"
            echo "  $0 restart      # 重启所有服务"
            echo "  $0 stop         # 停止所有服务"
            echo "  $0 status       # 查看状态"
            echo "  $0 backend      # 只启动后端"
            echo "  $0 frontend     # 只启动前端"
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
