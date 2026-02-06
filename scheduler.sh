#!/bin/bash

#===============================================================================
# Monera Digital 利息调度器启动脚本
# 
# 功能: 独立启动利息调度器服务
#
# 使用方法:
#   ./scheduler.sh              # 前台运行
#   ./scheduler.sh start        # 后台运行
#   ./scheduler.sh stop         # 停止服务
#   ./scheduler.sh status       # 查看状态
#   ./scheduler.sh log          # 查看日志
#
# 环境变量:
#   DATABASE_URL     - 数据库连接字符串 (必需)
#   ENV              - 环境 (development/production)
#   LOG_LEVEL        - 日志级别 (debug/info/warn/error)
#
# 示例:
#   DATABASE_URL="postgresql://user:pass@localhost:5432/monera" ./scheduler.sh start
#
# Crontab 定时任务示例 (每天凌晨运行):
#   0 0 * * * cd /path/to/project && ./scheduler.sh start >> /var/log/scheduler.log 2>&1
#===============================================================================

set -e

# 配置
APP_NAME="monera-scheduler"
APP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_FILE="/var/log/${APP_NAME}.log"
PID_FILE="/var/run/${APP_NAME}.pid"

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "[${GREEN}INFO${NC}] $(date '+%Y-%m-%d %H:%M:%S') - $1"
    echo -e "[INFO] $(date '+%Y-%m-%d %H:%M:%S') - $1" >> "$LOG_FILE" 2>/dev/null || true
}

log_warn() {
    echo -e "[${YELLOW}WARN${NC}] $(date '+%Y-%m-%d %H:%M:%S') - $1"
    echo -e "[WARN] $(date '+%Y-%m-%d %H:%M:%S') - $1" >> "$LOG_FILE" 2>/dev/null || true
}

log_error() {
    echo -e "[${RED}ERROR${NC}] $(date '+%Y-%m-%d %H:%M:%S') - $1"
    echo -e "[ERROR] $(date '+%Y-%m-%d %H:%M:%S') - $1" >> "$LOG_FILE" 2>/dev/null || true
}

# 检查环境
check_env() {
    if [ -z "$DATABASE_URL" ]; then
        log_error "DATABASE_URL 环境变量未设置"
        echo "请设置 DATABASE_URL 环境变量:"
        echo "  export DATABASE_URL=\"postgresql://user:pass@localhost:5432/monera\""
        echo ""
        echo "或者在运行时指定:"
        echo "  DATABASE_URL=\"...\" $0 start"
        exit 1
    fi
    log_info "数据库连接已配置"
}

# 检查进程是否运行
is_running() {
    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE")
        if kill -0 "$PID" 2>/dev/null; then
            return 0
        fi
        rm -f "$PID_FILE"
    fi
    return 1
}

# 启动服务
start() {
    check_env

    if is_running; then
        log_warn "服务已在运行中 (PID: $(cat $PID_FILE))"
        return 0
    fi

    log_info "启动利息调度器..."

    # 确保日志目录存在
    mkdir -p "$(dirname "$LOG_FILE")" 2>/dev/null || true

    # 设置日志级别
    export LOG_LEVEL="${LOG_LEVEL:-info}"

    # 启动进程
    cd "$APP_DIR"
    nohup go run cmd/scheduler/main.go >> "$LOG_FILE" 2>&1 &
    PID=$!
    echo $PID > "$PID_FILE"

    sleep 2

    if kill -0 "$PID" 2>/dev/null; then
        log_info "服务已启动 (PID: $PID)"
        log_info "日志文件: $LOG_FILE"
        echo "运行 $0 status 查看状态"
    else
        log_error "服务启动失败"
        rm -f "$PID_FILE"
        exit 1
    fi
}

# 停止服务
stop() {
    if ! is_running; then
        log_warn "服务未运行"
        return 0
    fi

    PID=$(cat "$PID_FILE")
    log_info "停止服务 (PID: $PID)..."

    # 发送SIGTERM信号
    kill "$PID" 2>/dev/null

    # 等待进程结束
    TIMEOUT=10
    while [ $TIMEOUT -gt 0 ]; do
        if ! kill -0 "$PID" 2>/dev/null; then
            rm -f "$PID_FILE"
            log_info "服务已停止"
            return 0
        fi
        sleep 1
        TIMEOUT=$((TIMEOUT - 1))
    done

    # 强制终止
    log_warn "进程未响应，发送SIGKILL..."
    kill -9 "$PID" 2>/dev/null
    rm -f "$PID_FILE"
    log_info "服务已强制停止"
}

# 查看状态
status() {
    if is_running; then
        PID=$(cat "$PID_FILE")
        echo -e "${GREEN}●${NC} 服务运行中 (PID: $PID)"
        
        # 显示进程信息
        if command -v ps >/dev/null 2>&1; then
            echo ""
            echo "进程信息:"
            ps -p "$PID" -o pid,ppid,cmd,etime 2>/dev/null || true
        fi
        
        # 显示最后几条日志
        if [ -f "$LOG_FILE" ]; then
            echo ""
            echo "最近日志:"
            tail -5 "$LOG_FILE" 2>/dev/null || true
        fi
    else
        echo -e "${RED}●${NC} 服务未运行"
        echo "使用 $0 start 启动服务"
    fi
}

# 查看日志
log() {
    if [ -f "$LOG_FILE" ]; then
        tail -f "$LOG_FILE"
    else
        log_error "日志文件不存在: $LOG_FILE"
    fi
}

# 一键测试运行 (只运行一次然后退出)
test_run() {
    check_env
    log_info "测试运行..."
    cd "$APP_DIR"
    go run cmd/scheduler/main.go
}

# 帮助信息
help() {
    echo "Monera Digital 利息调度器管理脚本"
    echo ""
    echo "使用方法: $0 <命令> [选项]"
    echo ""
    echo "命令:"
    echo "  start     启动服务 (后台运行)"
    echo "  stop      停止服务"
    echo "  status    查看服务状态"
    echo "  log       查看实时日志"
    echo "  test      测试运行 (前台运行一次)"
    echo "  help      显示帮助信息"
    echo ""
    echo "环境变量:"
    echo "  DATABASE_URL  数据库连接字符串 (必需)"
    echo "  ENV          环境 (development/production)"
    echo "  LOG_LEVEL    日志级别 (debug/info/warn/error)"
    echo ""
    echo "示例:"
    echo "  $0 start"
    echo "  DATABASE_URL=\"...\" $0 start"
    echo "  $0 status"
    echo "  $0 log"
}

# 主入口
case "${1:-help}" in
    start)
        start
        ;;
    stop)
        stop
        ;;
    status)
        status
        ;;
    log)
        log
        ;;
    test|test_run)
        test_run
        ;;
    help|--help|-h)
        help
        ;;
    *)
        echo "未知命令: $1"
        echo "使用 $0 help 查看帮助"
        exit 1
        ;;
esac
