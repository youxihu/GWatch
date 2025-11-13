#!/bin/bash

# 测试脚本：启动一个Server和两个Client模式（测试多客户端聚合）
# 使用方法: ./test_client_server.sh

set -e  # 遇到错误立即退出

# 获取脚本所在目录的父目录（项目根目录）
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT" || exit 1

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 全局变量
CLIENT1_PID=""
CLIENT2_PID=""
SERVER_PID=""
EXECUTABLE="bin/Gwatch"
CLIENT1_LOG="/tmp/gwatch_client1.out"
CLIENT2_LOG="/tmp/gwatch_client2.out"
SERVER_LOG="/tmp/gwatch_server.out"

# 清理函数
cleanup() {
    echo ""
    echo -e "${YELLOW}=========================================="
    echo "正在清理进程..."
    echo "==========================================${NC}"
    
    if [ ! -z "$CLIENT1_PID" ] && kill -0 $CLIENT1_PID 2>/dev/null; then
        echo -e "${CYAN}停止 Client 1 (PID: $CLIENT1_PID)...${NC}"
        kill $CLIENT1_PID 2>/dev/null || true
        wait $CLIENT1_PID 2>/dev/null || true
    fi
    
    if [ ! -z "$CLIENT2_PID" ] && kill -0 $CLIENT2_PID 2>/dev/null; then
        echo -e "${CYAN}停止 Client 2 (PID: $CLIENT2_PID)...${NC}"
        kill $CLIENT2_PID 2>/dev/null || true
        wait $CLIENT2_PID 2>/dev/null || true
    fi
    
    if [ ! -z "$SERVER_PID" ] && kill -0 $SERVER_PID 2>/dev/null; then
        echo -e "${CYAN}停止 Server (PID: $SERVER_PID)...${NC}"
        kill $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
    fi
    
    echo -e "${GREEN}清理完成${NC}"
}

# 注册清理函数
trap cleanup EXIT INT TERM

# 打印带颜色的消息
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_section() {
    echo ""
    echo -e "${CYAN}=========================================="
    echo "$1"
    echo "==========================================${NC}"
}

# 检查进程是否运行
check_process() {
    local pid=$1
    local name=$2
    if kill -0 $pid 2>/dev/null; then
        print_success "$name (PID: $pid) 正在运行"
        return 0
    else
        print_error "$name (PID: $pid) 已停止"
        return 1
    fi
}

# 显示日志文件最后几行
show_log_tail() {
    local log_file=$1
    local name=$2
    if [ -f "$log_file" ]; then
        echo -e "${CYAN}--- $name 最新日志 (最后10行) ---${NC}"
        tail -n 10 "$log_file" 2>/dev/null || echo "日志文件为空"
        echo ""
    fi
}

# 主程序开始
print_section "GWatch 多客户端聚合测试"

# 计算测试时间点
CURRENT_TIME=$(date +"%H:%M")
NEXT_MINUTE=$(date -d "+1 minute" +"%H:%M")
AFTER_MINUTE=$(date -d "+2 minutes" +"%H:%M")

print_info "当前时间: $CURRENT_TIME"
print_info "测试时间点: [$NEXT_MINUTE, $AFTER_MINUTE]"

# 创建第二个客户端配置文件（如果不存在）
if [ ! -f "config/config_client2.yml" ]; then
    print_info "创建 config/config_client2.yml..."
    cp config/config_client.yml config/config_client2.yml
    # 修改title为Client-2
    sed -i 's/title: "Client-.*"/title: "Client-2"/' config/config_client2.yml
    sed -i 's/output: logs\/gwatch_client.log/output: logs\/gwatch_client2.log/' config/config_client2.yml
    print_success "已创建 config_client2.yml"
fi

# 更新配置文件中的测试时间点
print_info "更新配置文件中的测试时间点..."
sed -i "s/push_times: \[.*\]/push_times: [\"$NEXT_MINUTE\", \"$AFTER_MINUTE\"]/" config/config_client.yml
sed -i "s/push_times: \[.*\]/push_times: [\"$NEXT_MINUTE\", \"$AFTER_MINUTE\"]/" config/config_client2.yml
sed -i "s/push_times: \[.*\]/push_times: [\"$NEXT_MINUTE\", \"$AFTER_MINUTE\"]/" config/config_server.yml
print_success "配置文件已更新"

# 检查可执行文件
if [ ! -f "$EXECUTABLE" ]; then
    print_warning "可执行文件不存在，开始编译..."
    make build
    if [ $? -ne 0 ]; then
        print_error "编译失败！"
        exit 1
    fi
    print_success "编译完成"
else
    print_success "可执行文件已存在"
fi

# 清理旧的日志文件
rm -f "$CLIENT1_LOG" "$CLIENT2_LOG" "$SERVER_LOG"

# 启动 Client 1
print_section "启动 Client 1 (Title: Client-1)"
GWATCH_CONFIG=config/config_client.yml ./$EXECUTABLE > "$CLIENT1_LOG" 2>&1 &
CLIENT1_PID=$!
sleep 2
if check_process $CLIENT1_PID "Client 1"; then
    print_info "日志文件: $CLIENT1_LOG"
else
    print_error "Client 1 启动失败！"
    show_log_tail "$CLIENT1_LOG" "Client 1"
    exit 1
fi

# 启动 Client 2
print_section "启动 Client 2 (Title: Client-2)"
GWATCH_CONFIG=config/config_client2.yml ./$EXECUTABLE > "$CLIENT2_LOG" 2>&1 &
CLIENT2_PID=$!
sleep 2
if check_process $CLIENT2_PID "Client 2"; then
    print_info "日志文件: $CLIENT2_LOG"
else
    print_error "Client 2 启动失败！"
    show_log_tail "$CLIENT2_LOG" "Client 2"
    exit 1
fi

# 显示测试信息
print_section "测试信息"
echo -e "${CYAN}测试时间点:${NC} $NEXT_MINUTE, $AFTER_MINUTE"
echo -e "${CYAN}Server聚合延迟:${NC} 30秒"
echo -e "${CYAN}期望结果:${NC} 聚合报告应该包含两个客户端的数据"
echo ""
echo -e "${YELLOW}提示:${NC}"
echo "  - 按 Ctrl+C 停止所有进程"
echo "  - Client 1 日志: $CLIENT1_LOG"
echo "  - Client 2 日志: $CLIENT2_LOG"
echo "  - Server 日志: $SERVER_LOG"
echo ""

# 启动 Server（前台运行，可以看到实时输出）
print_section "启动 Server 模式"
print_info "Server 将在 ${NEXT_MINUTE} 和 ${AFTER_MINUTE} 触发聚合"
print_info "两个 Client 会在相同时间点上传数据"
echo ""
print_success "所有进程已启动！"
echo ""
echo -e "${GREEN}=========================================="
echo "测试运行中..."
echo "==========================================${NC}"
echo ""
echo -e "${CYAN}进程信息:${NC}"
echo "  - Client 1 PID: $CLIENT1_PID (日志: $CLIENT1_LOG)"
echo "  - Client 2 PID: $CLIENT2_PID (日志: $CLIENT2_LOG)"
echo "  - Server 将在前台运行，输出直接显示"
echo ""
echo -e "${YELLOW}提示:${NC}"
echo "  - 按 Ctrl+C 停止所有进程"
echo "  - 查看 Client 日志: tail -f $CLIENT1_LOG 或 tail -f $CLIENT2_LOG"
echo ""

# 前台运行 Server（这样可以看到实时输出）
# 当 Server 退出时，trap 会自动清理其他进程
GWATCH_CONFIG=config/config_server.yml ./$EXECUTABLE

print_section "测试结束"
