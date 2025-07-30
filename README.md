# RocketMQ 聊天服务示例

这个项目演示了如何使用 RocketMQ 实现 Request-Response 和 Publish-Subscribe 两种消息传递模式。

## 项目结构

```
godemo/
├── server/main.go    # 服务端 - 处理请求和发布事件
├── client/main.go    # 客户端 - 发送请求和订阅事件
├── go.mod           # Go 模块文件
└── README.md        # 项目说明
```

## 功能特性

### 1. Request-Response 模式
- **请求主题**: `TG001-chat-service-requests`
- **响应主题**: `TG001-chat-service-responses`
- **支持的操作**:
  - `send_message`: 发送消息
  - `get_history`: 获取历史记录

### 2. Publish-Subscribe 模式
- **事件主题**: `TG001-chat-service-events`
- **支持的事件类型**:
  - `user_join`: 用户加入
  - `user_leave`: 用户离开
  - `message_sent`: 消息发送

## 环境配置

### RocketMQ 连接配置
根据您提供的服务信息，RocketMQ 配置如下：

```bash
# NameServer 地址
ROCKETMQ_NAMESERVER=rocketmq-nameserver.rocketmq-system.svc.cluster.local:9876

# 服务端组名
ROCKETMQ_GROUP=chat_server_group

# 客户端组名
ROCKETMQ_GROUP=chat_client_group

# 用户ID
USER_ID=user_001
```

### 服务信息
- **NameServer**: `10.43.98.224:9876`
- **NodePort**: `30876`
- **Endpoints**: `10.42.0.183:9876`

## 运行示例

### 1. 安装依赖
```bash
go mod tidy
```

### 2. 启动服务端
```bash
# 设置环境变量
export ROCKETMQ_NAMESERVER="rocketmq-nameserver.rocketmq-system.svc.cluster.local:9876"
export ROCKETMQ_GROUP="chat_server_group"

# 运行服务端
go run server/main.go
```

### 3. 启动客户端
```bash
# 设置环境变量
export ROCKETMQ_NAMESERVER="rocketmq-nameserver.rocketmq-system.svc.cluster.local:9876"
export ROCKETMQ_GROUP="chat_client_group"
export USER_ID="user_001"

# 运行客户端
go run client/main.go
```

## 消息传递规范

### Request-Response 模式

#### 请求消息格式
```json
{
  "request_id": "req_1234567890",
  "user_id": "user_001",
  "action": "send_message",
  "data": {
    "message": "Hello, everyone!",
    "room_id": "room_001"
  }
}
```

#### 响应消息格式
```json
{
  "request_id": "req_1234567890",
  "success": true,
  "data": {
    "message_id": "msg_1234567890",
    "status": "sent"
  },
  "error": null
}
```

### Publish-Subscribe 模式

#### 事件消息格式
```json
{
  "user_id": "user_001",
  "message": "Hello, everyone!",
  "timestamp": "2024-01-01T12:00:00Z",
  "type": "message_sent"
}
```

## 消息属性

### 请求消息属性
- `request_id`: 请求ID
- `user_id`: 用户ID
- `action`: 操作类型

### 响应消息属性
- `request_id`: 对应的请求ID
- `response_type`: 响应类型

### 事件消息属性
- `event_type`: 事件类型
- `user_id`: 用户ID

## 错误处理

### 超时处理
- 请求响应超时时间: 10秒
- 自动清理超时的请求通道

### 错误分类
- **业务错误**: 余额不足、用户状态异常等
- **系统错误**: 网络超时、服务不可用等

## 监控和日志

### 日志输出
- 请求/响应消息处理
- 事件消息处理
- 错误信息记录

### 关键指标
- 消息发送成功率
- 响应时间
- 错误率

## 扩展功能

### 1. 添加新的操作类型
在 `ChatRequest.Action` 中添加新的操作类型，并在 `processRequest` 方法中实现相应的处理逻辑。

### 2. 添加新的事件类型
在 `ChatMessage.Type` 中添加新的事件类型，并在 `processEvent` 方法中实现相应的处理逻辑。

### 3. 实现消息持久化
可以添加数据库操作来持久化消息，实现消息的存储和查询功能。

### 4. 添加消息过滤
使用 RocketMQ 的 Tag 功能来实现消息过滤，提高消息处理的效率。

## 注意事项

1. **连接配置**: 确保 RocketMQ NameServer 地址正确
2. **组名唯一性**: 确保生产者和消费者组名在集群中唯一
3. **错误处理**: 实现适当的错误处理和重试机制
4. **资源清理**: 确保在程序退出时正确关闭连接

## 故障排除

### 常见问题

1. **连接失败**
   - 检查 NameServer 地址是否正确
   - 确认网络连接正常
   - 检查防火墙设置

2. **消息发送失败**
   - 检查 Topic 是否存在
   - 确认生产者权限
   - 检查消息格式

3. **消息接收失败**
   - 检查消费者组名
   - 确认订阅的 Topic 正确
   - 检查消费者权限

### 调试方法

1. **启用详细日志**
   ```go
   log.SetLevel(log.DebugLevel)
   ```

2. **检查消息内容**
   ```go
   log.Printf("收到消息: %s", string(msg.Body))
   ```

3. **监控连接状态**
   ```go
   // 检查生产者状态
   if s.producer != nil {
       log.Printf("生产者状态: %v", s.producer)
   }
   ``` 