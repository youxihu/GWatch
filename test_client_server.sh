#!/bin/bash

# 测试脚本：启动两个Client和Server模式（测试多客户端聚合）

CURRENT_TIME=$(date +"%H:%M")
NEXT_MINUTE=$(date -d "+1 minute" +"%H:%M")
AFTER_MINUTE=$(date -d "+2 minutes" +"%H:%M")

echo "当前时间: $CURRENT_TIME"
echo "测试时间点: [$NEXT_MINUTE, $AFTER_MINUTE]"
echo ""

# 创建第二个客户端配置文件（如果不存在）
if [ ! -f "config/config_client2.yml" ]; then
    echo "创建 config/config_client2.yml..."
    cp config/config_client.yml config/config_client2.yml
    # 修改title为Client-2
    sed -i 's/title: "Client-.*"/title: "Client-2"/' config/config_client2.yml
    sed -i 's/output: logs\/gwatch_client.log/output: logs\/gwatch_client2.log/' config/config_client2.yml
fi

# 更新配置文件中的测试时间点
sed -i "s/push_times: \[.*\]/push_times: [\"$NEXT_MINUTE\", \"$AFTER_MINUTE\"]/" config/config_client.yml
sed -i "s/push_times: \[.*\]/push_times: [\"$NEXT_MINUTE\", \"$AFTER_MINUTE\"]/" config/config_client2.yml
sed -i "s/push_times: \[.*\]/push_times: [\"$NEXT_MINUTE\", \"$AFTER_MINUTE\"]/" config/config_server.yml

echo "配置文件已更新，测试时间点: [$NEXT_MINUTE, $AFTER_MINUTE]"
echo ""

# 检查可执行文件
EXECUTABLE="bin/Gwatch"
if [ ! -f "$EXECUTABLE" ]; then
    echo "编译程序..."
    make build
    if [ $? -ne 0 ]; then
        echo "编译失败！"
        exit 1
    fi
fi

echo "=========================================="
echo "启动 Client 1 模式（后台运行，Title: Client-1）..."
echo "=========================================="
GWATCH_CONFIG=config/config_client.yml ./$EXECUTABLE > /tmp/gwatch_client1.out 2>&1 &
CLIENT1_PID=$!
echo "Client 1 PID: $CLIENT1_PID"

sleep 2

echo ""
echo "=========================================="
echo "启动 Client 2 模式（后台运行，Title: Client-2）..."
echo "=========================================="
GWATCH_CONFIG=config/config_client2.yml ./$EXECUTABLE > /tmp/gwatch_client2.out 2>&1 &
CLIENT2_PID=$!
echo "Client 2 PID: $CLIENT2_PID"

sleep 2

echo ""
echo "=========================================="
echo "启动 Server 模式（前台运行，按Ctrl+C停止）..."
echo "=========================================="
echo "提示：Server会在 ${NEXT_MINUTE} 和 ${AFTER_MINUTE} 触发聚合"
echo "两个Client会在相同时间点上传数据"
echo "Server聚合延迟: 30秒"
echo "期望：聚合报告应该包含两个客户端的数据"
echo ""

# 前台运行Server
GWATCH_CONFIG=config/config_server.yml ./$EXECUTABLE

# 当Server退出时，也停止Clients
echo ""
echo "停止 Clients..."
kill $CLIENT1_PID 2>/dev/null
wait $CLIENT1_PID 2>/dev/null
kill $CLIENT2_PID 2>/dev/null
wait $CLIENT2_PID 2>/dev/null

echo "测试结束"
