#!/bin/bash

# 本地測試環境運行腳本
# 用於在本地環境中測試應用程序

set -e

# 顏色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 配置
ROCKETMQ_NAMESERVER=${1:-"localhost:9876"}
ENVIRONMENT="test"

echo -e "${GREEN}啟動本地測試環境...${NC}"
echo -e "${YELLOW}RocketMQ NameServer: ${ROCKETMQ_NAMESERVER}${NC}"

# 檢查 RocketMQ 是否運行
check_rocketmq() {
    echo -e "${YELLOW}檢查 RocketMQ 連接...${NC}"
    
    # 嘗試連接 RocketMQ
    if nc -z $(echo $ROCKETMQ_NAMESERVER | cut -d: -f1) $(echo $ROCKETMQ_NAMESERVER | cut -d: -f2) 2>/dev/null; then
        echo -e "${GREEN}RocketMQ 連接正常${NC}"
    else
        echo -e "${RED}警告: 無法連接到 RocketMQ NameServer${NC}"
        echo -e "${YELLOW}請確保 RocketMQ 正在運行在 ${ROCKETMQ_NAMESERVER}${NC}"
        read -p "是否繼續？(y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi
}

# 構建應用程序
build_app() {
    echo -e "${YELLOW}構建應用程序...${NC}"
    
    # 構建服務端
    echo "構建服務端..."
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server/server server/main.go
    
    # 構建客戶端
    echo "構建客戶端..."
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o client/client client/main.go
    
    echo -e "${GREEN}應用程序構建完成${NC}"
}

# 啟動服務端
start_server() {
    echo -e "${YELLOW}啟動服務端...${NC}"
    
    # 設置環境變數
    export ROCKETMQ_NAMESERVER=$ROCKETMQ_NAMESERVER
    export ROCKETMQ_ENVIRONMENT=$ENVIRONMENT
    export ROCKETMQ_GROUP="chat_server_group"
    
    # 啟動服務端
    ./server/server &
    SERVER_PID=$!
    
    echo -e "${GREEN}服務端已啟動 (PID: $SERVER_PID)${NC}"
}

# 啟動客戶端
start_client() {
    echo -e "${YELLOW}啟動客戶端...${NC}"
    
    # 等待服務端啟動
    sleep 3
    
    # 設置環境變數
    export ROCKETMQ_NAMESERVER=$ROCKETMQ_NAMESERVER
    export ROCKETMQ_ENVIRONMENT=$ENVIRONMENT
    export ROCKETMQ_GROUP="chat_client_group"
    export USER_ID="user_001"
    
    # 啟動客戶端
    ./client/client &
    CLIENT_PID=$!
    
    echo -e "${GREEN}客戶端已啟動 (PID: $CLIENT_PID)${NC}"
}

# 監控進程
monitor_processes() {
    echo -e "${YELLOW}監控進程...${NC}"
    echo "按 Ctrl+C 停止所有進程"
    
    # 等待用戶中斷
    trap 'cleanup' INT TERM
    
    while true; do
        # 檢查進程是否還在運行
        if ! kill -0 $SERVER_PID 2>/dev/null; then
            echo -e "${RED}服務端進程已停止${NC}"
            break
        fi
        
        if ! kill -0 $CLIENT_PID 2>/dev/null; then
            echo -e "${RED}客戶端進程已停止${NC}"
            break
        fi
        
        sleep 5
    done
}

# 清理函數
cleanup() {
    echo -e "${YELLOW}正在停止進程...${NC}"
    
    if [ ! -z "$SERVER_PID" ]; then
        kill $SERVER_PID 2>/dev/null || true
        echo "服務端已停止"
    fi
    
    if [ ! -z "$CLIENT_PID" ]; then
        kill $CLIENT_PID 2>/dev/null || true
        echo "客戶端已停止"
    fi
    
    echo -e "${GREEN}清理完成${NC}"
    exit 0
}

# 顯示幫助信息
show_help() {
    echo "用法: $0 [nameserver]"
    echo ""
    echo "參數:"
    echo "  nameserver  - RocketMQ NameServer 地址 (默認: localhost:9876)"
    echo ""
    echo "示例:"
    echo "  $0 localhost:9876"
    echo "  $0 192.168.1.100:9876"
    echo ""
    echo "環境變數:"
    echo "  ROCKETMQ_NAMESERVER  - RocketMQ NameServer 地址"
    echo "  ROCKETMQ_ENVIRONMENT - 環境類型 (test/k8s)"
    echo "  ROCKETMQ_GROUP      - 消費者組名稱"
    echo "  USER_ID             - 用戶 ID"
}

# 主函數
main() {
    if [ "$1" = "-h" ] || [ "$1" = "--help" ]; then
        show_help
        exit 0
    fi
    
    # 檢查依賴
    if ! command -v go &> /dev/null; then
        echo -e "${RED}錯誤: Go 未安裝${NC}"
        exit 1
    fi
    
    if ! command -v nc &> /dev/null; then
        echo -e "${YELLOW}警告: netcat 未安裝，跳過連接檢查${NC}"
    fi
    
    # 執行步驟
    check_rocketmq
    build_app
    start_server
    start_client
    monitor_processes
}

# 執行主函數
main "$@" 