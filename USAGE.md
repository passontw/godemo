# 快速开始

## 1. 安装依赖
```bash
go mod tidy
```

## 2. 运行示例

### 方法一：使用启动脚本
```bash
# 启动服务端
./run.sh server

# 启动客户端（在另一个终端）
./run.sh client

# 或者同时启动服务端和客户端
./run.sh both
```

### 方法二：直接运行
```bash
# 启动服务端
export ROCKETMQ_NAMESERVER="rocketmq-nameserver.rocketmq-system.svc.cluster.local:9876"
export ROCKETMQ_GROUP="chat_server_group"
go run server/main.go

# 启动客户端（在另一个终端）
export ROCKETMQ_NAMESERVER="rocketmq-nameserver.rocketmq-system.svc.cluster.local:9876"
export ROCKETMQ_GROUP="chat_client_group"
export USER_ID="user_001"
go run client/main.go
```

## 3. 预期输出

### 服务端输出
```
2024/01/01 12:00:00 聊天服务端已启动，监听请求和事件...
2024/01/01 12:00:05 事件已发布: SendResult [MessageId: xxx, QueueId: 0, QueueOffset: 0]
2024/01/01 12:00:05 事件已发布: SendResult [MessageId: xxx, QueueId: 0, QueueOffset: 1]
2024/01/01 12:00:10 收到请求消息: {"request_id":"req_xxx","user_id":"user_001","action":"send_message",...}
2024/01/01 12:00:10 处理请求: req_xxx, 用户: user_001, 动作: send_message
2024/01/01 12:00:10 消息已发送: map[message_id:msg_xxx status:sent]
2024/01/01 12:00:10 响应已发送: SendResult [MessageId: xxx, QueueId: 0, QueueOffset: 2]
```

### 客户端输出
```
2024/01/01 12:00:00 聊天客户端已启动，用户: user_001
2024/01/01 12:00:02 发布用户加入事件...
2024/01/01 12:00:02 事件已发布: SendResult [MessageId: xxx, QueueId: 0, QueueOffset: 0]
2024/01/01 12:00:04 发送消息请求...
2024/01/01 12:00:04 请求已发送: SendResult [MessageId: xxx, QueueId: 0, QueueOffset: 1]
2024/01/01 12:00:04 收到响应消息: {"request_id":"req_xxx","success":true,"data":{...}}
2024/01/01 12:00:04 收到响应: &{RequestID:req_xxx Success:true Data:map[...] Error:}
```

## 4. 消息流程

### Request-Response 模式
1. 客户端发送请求到 `TG001-chat-service-requests`
2. 服务端接收请求并处理
3. 服务端发送响应到 `TG001-chat-service-responses`
4. 客户端接收响应

### Publish-Subscribe 模式
1. 客户端发布事件到 `TG001-chat-service-events`
2. 服务端订阅并处理事件
3. 服务端也可以发布事件
4. 客户端订阅并处理事件

## 5. 自定义配置

### 环境变量
```bash
# RocketMQ NameServer 地址
export ROCKETMQ_NAMESERVER="your-nameserver:9876"

# 消费者组名
export ROCKETMQ_GROUP="your-group-name"

# 用户ID
export USER_ID="your-user-id"
```

### 修改主题名称
在 `server/main.go` 和 `client/main.go` 中修改以下常量：
```go
// 请求主题
"TG001-chat-service-requests"

// 响应主题
"TG001-chat-service-responses"

// 事件主题
"TG001-chat-service-events"
```

## 6. 故障排除

### 连接问题
```bash
# 检查网络连接
telnet rocketmq-nameserver.rocketmq-system.svc.cluster.local 9876

# 检查环境变量
echo $ROCKETMQ_NAMESERVER
echo $ROCKETMQ_GROUP
```

### 编译问题
```bash
# 清理并重新安装依赖
go clean -modcache
go mod tidy
```

### 运行时问题
```bash
# 启用详细日志
export ROCKETMQ_LOG_LEVEL=debug

# 检查端口占用
netstat -tlnp | grep 9876
``` 