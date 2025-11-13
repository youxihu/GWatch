#!/bin/bash

# 测试脚本：模拟高网络IO和高磁盘IO，然后运行 start_all.sh 测试

# 获取脚本所在目录的父目录（项目根目录）
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT" || exit 1

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
RED='\033[0;31m'
NC='\033[0m'

# 全局变量
DISK_IO_PID=""
NETWORK_IO_PID=""
TEST_DIR="/tmp/gwatch_io_test"
TEST_FILE="$TEST_DIR/test_io.dat"

# 清理函数
cleanup() {
    echo ""
    echo -e "${YELLOW}=========================================="
    echo "正在清理测试进程..."
    echo "==========================================${NC}"
    
    if [ ! -z "$DISK_IO_PID" ] && kill -0 $DISK_IO_PID 2>/dev/null; then
        echo -e "${CYAN}停止磁盘IO测试进程 (PID: $DISK_IO_PID)...${NC}"
        kill $DISK_IO_PID 2>/dev/null || true
        wait $DISK_IO_PID 2>/dev/null || true
    fi
    
    if [ ! -z "$NETWORK_IO_PID" ] && kill -0 $NETWORK_IO_PID 2>/dev/null; then
        echo -e "${CYAN}停止网络IO测试进程 (PID: $NETWORK_IO_PID)...${NC}"
        kill $NETWORK_IO_PID 2>/dev/null || true
        wait $NETWORK_IO_PID 2>/dev/null || true
    fi
    
    # 清理测试文件
    if [ -d "$TEST_DIR" ]; then
        echo -e "${CYAN}清理测试文件...${NC}"
        rm -rf "$TEST_DIR"
    fi
    
    echo -e "${GREEN}清理完成${NC}"
}

# 注册清理函数
trap cleanup EXIT INT TERM

print_info() {
    echo -e "${CYAN}[INFO]${NC} $1"
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

# 主程序开始
print_section "GWatch 高IO测试脚本"

# 创建测试目录
print_info "创建测试目录: $TEST_DIR"
mkdir -p "$TEST_DIR"

# 启动磁盘IO测试（后台运行）
print_section "启动磁盘IO测试"
print_info "使用 dd 命令持续写入文件模拟高磁盘IO..."
print_info "写入速度: 约 100MB/s"
print_info "文件大小: 循环写入，每次 1GB"

# 磁盘IO测试：持续写入和删除文件
(
    while true; do
        # 写入 500MB 数据（避免一次性写入太大）
        dd if=/dev/zero of="$TEST_FILE" bs=1M count=500 2>/dev/null
        # 同步到磁盘
        sync
        # 删除文件
        rm -f "$TEST_FILE"
        # 同步删除操作
        sync
        # 短暂休眠避免过度占用
        sleep 0.3
    done
) &
DISK_IO_PID=$!

if kill -0 $DISK_IO_PID 2>/dev/null; then
    print_success "磁盘IO测试进程已启动 (PID: $DISK_IO_PID)"
else
    print_error "磁盘IO测试进程启动失败！"
    exit 1
fi

# 启动网络IO测试（后台运行）
print_section "启动网络IO测试"
print_info "使用 curl 下载大文件模拟高网络IO..."

# 网络IO测试：持续下载大文件
(
    # 使用一些公开的大文件下载URL进行测试
    # 如果这些URL不可用，可以替换为其他可用的URL
    TEST_URLS=(
        "https://speed.hetzner.de/100MB.bin"
        "https://speed.hetzner.de/1GB.bin"
        "http://ipv4.download.thinkbroadband.com/100MB.zip"
        "http://ipv4.download.thinkbroadband.com/200MB.zip"
    )
    
    while true; do
        for url in "${TEST_URLS[@]}"; do
            # 下载文件到 /dev/null（不保存，只消耗带宽）
            # 使用限速避免过度占用带宽，设置为 10MB/s
            curl -s --max-time 60 --limit-rate 10M "$url" -o /dev/null 2>/dev/null || true
            sleep 0.5
        done
    done
) &
NETWORK_IO_PID=$!

if kill -0 $NETWORK_IO_PID 2>/dev/null; then
    print_success "网络IO测试进程已启动 (PID: $NETWORK_IO_PID)"
else
    print_error "网络IO测试进程启动失败！"
    exit 1
fi

# 等待一下让IO测试开始
print_info "等待5秒让IO测试开始..."
sleep 5

# 显示IO测试状态
print_section "IO测试状态"
echo -e "${CYAN}磁盘IO测试:${NC} PID $DISK_IO_PID (持续写入/删除文件)"
echo -e "${CYAN}网络IO测试:${NC} PID $NETWORK_IO_PID (持续下载文件)"
echo ""
echo -e "${YELLOW}提示:${NC}"
echo "  - 磁盘IO测试会持续写入和删除文件"
echo "  - 网络IO测试会持续下载文件（消耗带宽）"
echo "  - 按 Ctrl+C 停止所有测试"
echo ""

# 检查 start_all.sh 是否存在
if [ ! -f "start_all.sh" ]; then
    print_error "start_all.sh 不存在！"
    exit 1
fi

# 运行 start_all.sh
print_section "启动 GWatch 监控"
print_info "现在将启动 GWatch 监控来观察高IO情况..."
echo ""
echo -e "${GREEN}=========================================="
echo "IO测试运行中，GWatch监控已启动"
echo "==========================================${NC}"
echo ""
echo -e "${CYAN}监控信息:${NC}"
echo "  - 磁盘IO测试进程: PID $DISK_IO_PID"
echo "  - 网络IO测试进程: PID $NETWORK_IO_PID"
echo "  - 测试文件目录: $TEST_DIR"
echo ""
echo -e "${YELLOW}按 Ctrl+C 停止所有测试和监控${NC}"
echo ""

# 运行 start_all.sh（使用绝对路径）
"$SCRIPT_DIR/start_all.sh"

