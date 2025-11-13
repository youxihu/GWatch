#!/bin/bash

# 清理脚本：清理测试产生的临时文件和日志

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

# 获取脚本所在目录的父目录（项目根目录）
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT" || exit 1

echo -e "${CYAN}=========================================="
echo "GWatch 清理脚本"
echo "=========================================="
echo "项目根目录: $PROJECT_ROOT"
echo "==========================================${NC}"
echo ""

# 停止所有 GWatch 进程
echo -e "${YELLOW}停止所有 GWatch 进程...${NC}"
KILLED_COUNT=0

# 尝试多种匹配模式
for pattern in "bin/Gwatch" "Gwatch" "GWatch"; do
    PIDS=$(pgrep -f "$pattern" 2>/dev/null)
    if [ ! -z "$PIDS" ]; then
        for pid in $PIDS; do
            kill $pid 2>/dev/null && KILLED_COUNT=$((KILLED_COUNT + 1))
        done
    fi
done

if [ $KILLED_COUNT -gt 0 ]; then
    echo -e "${GREEN}已停止 $KILLED_COUNT 个 GWatch 进程${NC}"
    sleep 2
    # 强制杀死残留进程
    for pattern in "bin/Gwatch" "Gwatch" "GWatch"; do
        pkill -9 -f "$pattern" 2>/dev/null
    done
else
    echo -e "${CYAN}没有运行中的 GWatch 进程${NC}"
fi

# 清理临时日志文件
echo ""
echo -e "${YELLOW}清理临时日志文件...${NC}"
TEMP_FILES=$(ls /tmp/gwatch*.log /tmp/gwatch*.out 2>/dev/null)
if [ ! -z "$TEMP_FILES" ]; then
    rm -f /tmp/gwatch*.log /tmp/gwatch*.out 2>/dev/null
    echo -e "${GREEN}已清理临时日志文件${NC}"
else
    echo -e "${CYAN}没有临时日志文件需要清理${NC}"
fi

# 清理测试目录
echo ""
echo -e "${YELLOW}清理测试目录...${NC}"
if [ -d "/tmp/gwatch_io_test" ]; then
    rm -rf /tmp/gwatch_io_test 2>/dev/null
    echo -e "${GREEN}已清理测试目录${NC}"
else
    echo -e "${CYAN}没有测试目录需要清理${NC}"
fi

# 询问是否清理 logs 目录
echo ""
LOGS_DIR="$PROJECT_ROOT/logs"
if [ -d "$LOGS_DIR" ]; then
    read -p "是否清理 logs 目录下的所有日志文件? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${YELLOW}清理 logs 目录 ($LOGS_DIR)...${NC}"
        rm -rf "$LOGS_DIR"/* 2>/dev/null
        mkdir -p "$LOGS_DIR/scheduled_push/client" "$LOGS_DIR/scheduled_push/server"
        echo -e "${GREEN}已清理 logs 目录${NC}"
    else
        echo -e "${CYAN}保留 logs 目录${NC}"
    fi
else
    echo -e "${CYAN}logs 目录不存在，跳过${NC}"
fi

echo ""
echo -e "${GREEN}=========================================="
echo "清理完成！"
echo "==========================================${NC}"

