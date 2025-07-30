#!/bin/bash

# RocketMQ 聊天服务启动脚本

# 设置默认环境变量
export ROCKETMQ_NAMESERVER=${ROCKETMQ_NAMESERVER:-"rocketmq-nameserver.rocketmq-system.svc.cluster.local:9876"}
export ROCKETMQ_GROUP=${ROCKETMQ_GROUP:-"chat_server_group"}
export USER_ID=${USER_ID:-"user_001"}

echo "=== RocketMQ 聊天服务启动脚本 ==="
echo "NameServer: $ROCKETMQ_NAMESERVER"
echo "Group: $ROCKETMQ_GROUP"
echo "UserID: $USER_ID"
echo ""

# 检查参数
if [ "$1" = "server" ]; then
    echo "启动服务端..."
    export ROCKETMQ_GROUP="chat_server_group"
    go run server/main.go
elif [ "$1" = "client" ]; then
    echo "启动客户端..."
    export ROCKETMQ_GROUP="chat_client_group"
    go run client/main.go
elif [ "$1" = "both" ]; then
    echo "同时启动服务端和客户端..."
    
    # 启动服务端
    export ROCKETMQ_GROUP="chat_server_group"
    go run server/main.go &
    SERVER_PID=$!
    
    # 等待服务端启动
    sleep 3
    
    # 启动客户端
    export ROCKETMQ_GROUP="chat_client_group"
    go run client/main.go &
    CLIENT_PID=$!
    
    # 等待用户中断
    echo "按 Ctrl+C 停止所有服务"
    wait
else
    echo "用法: $0 {server|client|both}"
    echo ""
    echo "参数说明:"
    echo "  server  - 启动服务端"
    echo "  client  - 启动客户端"
    echo "  both    - 同时启动服务端和客户端"
    echo ""
    echo "环境变量:"
    echo "  ROCKETMQ_NAMESERVER - RocketMQ NameServer 地址"
    echo "  ROCKETMQ_GROUP      - 消费者组名"
    echo "  USER_ID             - 用户ID"
fi 