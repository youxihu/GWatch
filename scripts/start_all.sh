#!/bin/bash

# 快速启动三个 GWatch 实例的脚本

# 获取脚本所在目录的父目录（项目根目录）
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT" || exit 1

# 颜色定义
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${CYAN}=========================================="
echo "启动三个 GWatch 实例"
echo "=========================================="
echo "项目根目录: $PROJECT_ROOT"
echo "==========================================${NC}"
echo ""

# 检查可执行文件
if [ ! -f "bin/Gwatch" ]; then
    echo -e "${YELLOW}可执行文件不存在，开始编译...${NC}"
    make build
    if [ $? -ne 0 ]; then
        echo "编译失败！"
        exit 1
    fi
fi

# 清理函数
cleanup() {
    echo ""
    echo -e "${YELLOW}正在停止所有进程...${NC}"
    pkill -f "bin/Gwatch -c config/config.yml"
    pkill -f "bin/Gwatch -c config/config_client.yml"
    pkill -f "bin/Gwatch -c config/config_client2.yml"
    pkill -f "bin/Gwatch -c config/config_server.yml"
    echo -e "${GREEN}已停止所有进程${NC}"
}

# 注册清理函数
trap cleanup EXIT INT TERM

# 启动三个实例
echo -e "${CYAN}启动实例 1: config.yml${NC}"
bin/Gwatch -c config/config.yml > /tmp/gwatch1.log 2>&1 &
PID1=$!
echo -e "${GREEN}  PID: $PID1 (日志: /tmp/gwatch1.log)${NC}"

sleep 1

echo -e "${CYAN}启动实例 2: config_client.yml${NC}"
bin/Gwatch -c config/config_client.yml > /tmp/gwatch2.log 2>&1 &
PID2=$!
echo -e "${GREEN}  PID: $PID2 (日志: /tmp/gwatch2.log)${NC}"

sleep 1

echo -e "${CYAN}启动实例 3: config_client2.yml${NC}"
bin/Gwatch -c config/config_client2.yml > /tmp/gwatch3.log 2>&1 &
PID3=$!
echo -e "${GREEN}  PID: $PID3 (日志: /tmp/gwatch3.log)${NC}"

echo ""
echo -e "${GREEN}=========================================="
echo "所有实例已启动！"
echo "==========================================${NC}"
echo ""
echo -e "${CYAN}进程信息:${NC}"
echo "  - 实例 1 (config.yml): PID $PID1"
echo "  - 实例 2 (config_client.yml): PID $PID2"
echo "  - 实例 3 (config_client2.yml): PID $PID3"
echo ""
echo -e "${YELLOW}提示:${NC}"
echo "  - 按 Ctrl+C 停止所有进程"
echo "  - 查看日志: tail -f /tmp/gwatch1.log"
echo "  - 查看日志: tail -f /tmp/gwatch2.log"
echo "  - 查看日志: tail -f /tmp/gwatch3.log"
echo ""
echo -e "${CYAN}等待进程运行中... (按 Ctrl+C 停止)${NC}"

# 等待所有进程
wait

