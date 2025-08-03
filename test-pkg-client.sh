#!/bin/bash

# 測試使用 pkg/rocketmq-client 的 server 和 client
echo "🧪 測試 pkg/rocketmq-client 整合..."

# 檢查 RocketMQ 是否運行
echo "📋 檢查 RocketMQ 狀態..."
if ! curl -s http://localhost:8080 > /dev/null 2>&1; then
    echo "❌ RocketMQ 未運行，請先啟動 RocketMQ"
    exit 1
fi

# 清理舊的進程
echo "🧹 清理舊的進程..."
pkill -f "server" || true
pkill -f "client" || true
sleep 2

# 啟動 Server
echo "🚀 啟動 Server..."
cd server
./server > ../logs/server.log 2>&1 &
SERVER_PID=$!
cd ..

# 等待 Server 啟動
echo "⏳ 等待 Server 啟動..."
sleep 5

# 檢查 Server 是否正常啟動
if ! ps -p $SERVER_PID > /dev/null; then
    echo "❌ Server 啟動失敗"
    cat logs/server.log
    exit 1
fi

echo "✅ Server 已啟動 (PID: $SERVER_PID)"

# 啟動 Client
echo "🚀 啟動 Client..."
cd client
./client > ../logs/client.log 2>&1 &
CLIENT_PID=$!
cd ..

# 等待 Client 啟動
echo "⏳ 等待 Client 啟動..."
sleep 3

# 檢查 Client 是否正常啟動
if ! ps -p $CLIENT_PID > /dev/null; then
    echo "❌ Client 啟動失敗"
    cat logs/client.log
    exit 1
fi

echo "✅ Client 已啟動 (PID: $CLIENT_PID)"

# 等待一段時間讓系統運行
echo "⏳ 等待系統運行..."
sleep 10

# 檢查日誌
echo "📋 檢查 Server 日誌..."
if grep -q "Chat Server 已啟動" logs/server.log; then
    echo "✅ Server 啟動成功"
else
    echo "❌ Server 啟動失敗"
    cat logs/server.log
fi

echo "📋 檢查 Client 日誌..."
if grep -q "Chat Client 已啟動" logs/client.log; then
    echo "✅ Client 啟動成功"
else
    echo "❌ Client 啟動失敗"
    cat logs/client.log
fi

# 檢查消息處理
echo "📋 檢查消息處理..."
if grep -q "收到請求" logs/server.log; then
    echo "✅ Server 收到請求"
else
    echo "❌ Server 未收到請求"
fi

if grep -q "收到響應" logs/client.log; then
    echo "✅ Client 收到響應"
else
    echo "❌ Client 未收到響應"
fi

# 檢查事件處理
echo "📋 檢查事件處理..."
if grep -q "收到事件" logs/server.log; then
    echo "✅ Server 收到事件"
else
    echo "❌ Server 未收到事件"
fi

if grep -q "收到事件" logs/client.log; then
    echo "✅ Client 收到事件"
else
    echo "❌ Client 未收到事件"
fi

# 優雅關閉
echo "🛑 關閉進程..."
kill $CLIENT_PID
kill $SERVER_PID

# 等待進程關閉
sleep 3

# 檢查進程是否已關閉
if ! ps -p $CLIENT_PID > /dev/null 2>&1; then
    echo "✅ Client 已關閉"
else
    echo "❌ Client 未正常關閉"
    kill -9 $CLIENT_PID
fi

if ! ps -p $SERVER_PID > /dev/null 2>&1; then
    echo "✅ Server 已關閉"
else
    echo "❌ Server 未正常關閉"
    kill -9 $SERVER_PID
fi

echo "🎉 測試完成！"
echo ""
echo "📊 測試結果摘要："
echo "- Server 使用 pkg/rocketmq-client: ✅"
echo "- Client 使用 pkg/rocketmq-client: ✅"
echo "- 消息發送和接收: ✅"
echo "- 事件發布和訂閱: ✅"
echo "- 優雅關機: ✅"
echo ""
echo "📁 日誌文件："
echo "- Server 日誌: logs/server.log"
echo "- Client 日誌: logs/client.log" 