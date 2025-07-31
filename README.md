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

## Kubernetes 部署

### Docker 鏡像構建

#### 服務端 Dockerfile
```dockerfile
# 使用官方 Go 镜像作为构建环境
FROM golang:1.24-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制 go mod 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY server/ ./server/

# 构建服务端
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o /app/myserver ./server/main.go

# 使用轻量级的 alpine 镜像作为运行环境
FROM alpine:latest

# 安装 ca-certificates，用于 HTTPS 请求
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# 从构建阶段复制二进制文件
COPY --from=builder /app/myserver ./server

# 设置执行权限
RUN chmod +x ./server

# 暴露端口（如果需要的话）
EXPOSE 8080

# 运行服务端
CMD ["./server"]
```

#### 客戶端 Dockerfile
```dockerfile
# 使用官方 Go 镜像作为构建环境
FROM golang:1.24-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制 go mod 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY client/ ./client/

# 构建客户端
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o /app/myclient ./client/main.go

# 使用轻量级的 alpine 镜像作为运行环境
FROM alpine:latest

# 安装 ca-certificates，用于 HTTPS 请求
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# 从构建阶段复制二进制文件
COPY --from=builder /app/myclient ./client

# 设置执行权限
RUN chmod +x ./client

# 运行客户端
CMD ["./client"]
```

### Kubernetes 部署文件

#### 服務端部署（server-deployment.yaml）
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: rocketmq-server
  labels:
    app: rocketmq-server
spec:
  replicas: 1
  selector:
    matchLabels:
      app: rocketmq-server
  template:
    metadata:
      labels:
        app: rocketmq-server
    spec:
      containers:
      - name: rocketmq-server
        image: harbor.trevi-dev.cc/cpp_run/rocketmq-server:v4
        env:
        - name: ROCKETMQ_NAMESERVER
          value: "10.43.171.188:9876"
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        ports:
        - containerPort: 8080
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "200m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: rocketmq-server-service
  labels:
    app: rocketmq-server
spec:
  selector:
    app: rocketmq-server
  ports:
  - protocol: TCP
    port: 8080
    targetPort: 8080
  type: ClusterIP
```

#### 客戶端部署（client-deployment.yaml）
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: rocketmq-client
  labels:
    app: rocketmq-client
spec:
  replicas: 1
  selector:
    matchLabels:
      app: rocketmq-client
  template:
    metadata:
      labels:
        app: rocketmq-client
    spec:
      containers:
      - name: rocketmq-client
        image: harbor.trevi-dev.cc/cpp_run/rocketmq-client:v2
        env:
        - name: ROCKETMQ_NAMESERVER
          value: "10.43.171.188:9876"
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "200m"
        command: ["./client"]
        args: []
---
apiVersion: v1
kind: Service
metadata:
  name: rocketmq-client-service
  labels:
    app: rocketmq-client
spec:
  selector:
    app: rocketmq-client
  ports:
  - protocol: TCP
    port: 8080
    targetPort: 8080
  type: ClusterIP
```

### 部署命令
```bash
# 構建鏡像
docker build -f Dockerfile.server -t harbor.trevi-dev.cc/cpp_run/rocketmq-server:v4 .
docker build -f Dockerfile.client -t harbor.trevi-dev.cc/cpp_run/rocketmq-client:v2 .

# 推送鏡像
docker push harbor.trevi-dev.cc/cpp_run/rocketmq-server:v4
docker push harbor.trevi-dev.cc/cpp_run/rocketmq-client:v2

# 部署到 Kubernetes
kubectl apply -f k8s/server-deployment.yaml
kubectl apply -f k8s/client-deployment.yaml
```

## 故障排除與解決方案

### 已解決的關鍵問題

#### 1. **Broker 內部/外部 IP 連接問題** ⭐️
**問題描述**：
- Kubernetes 內部的 pods 無法連接到 RocketMQ Broker
- 錯誤現象：client 和 server 能連接到 NameServer，但無法連接到 Broker
- 根本原因：Broker 配置使用外部 IP `10.1.7.229:10911`，K8s 內部 pods 無法路由到此地址

**解決方案**：
修改 RocketMQ Broker 配置，使用內部服務地址：
```yaml
# 修改前（K8s 內部無法連接）
brokerIP1 = 10.1.7.229

# 修改後（K8s 內部可以連接）
brokerIP1 = rocketmq-broker-ondemand-internal.rocketmq-system.svc.cluster.local
brokerIP2 = rocketmq-broker-ondemand-internal.rocketmq-system.svc.cluster.local
```

**修復步驟**：
```bash
# 1. 修改 broker 配置檔案
vim /Users/tomas/Projects/shared_utils/deployments/self-host/rocketmq/broker-ondemand-deployment.yaml

# 2. 重新應用配置
kubectl apply -f /Users/tomas/Projects/shared_utils/deployments/self-host/rocketmq/broker-ondemand-deployment.yaml

# 3. 重新啟動 broker pods
kubectl rollout restart statefulset/rocketmq-broker-ondemand -n rocketmq-system

# 4. 驗證 broker 使用新的內部地址
kubectl logs rocketmq-broker-ondemand-0 -n rocketmq-system --tail=5
```

#### 2. **Docker 容器架構兼容性問題**
**問題描述**：
- 錯誤信息：`exec format error`
- 原因：Dockerfile 構建配置問題，binary 檔案路徑錯誤

**解決方案**：
```dockerfile
# 修改前（錯誤的構建路徑）
RUN go build -o server ./server/main.go
COPY --from=builder /app/server ./server

# 修改後（正確的構建路徑）
RUN go build -o /app/myserver ./server/main.go
COPY --from=builder /app/myserver ./server
```

#### 3. **RocketMQ Topics 創建**
**問題描述**：
- 應用啟動時 topics 不存在，訂閱失敗

**解決方案**：
```bash
# 進入 broker pod 創建 topics
kubectl exec -it rocketmq-broker-ondemand-0 -n rocketmq-system -- sh

# 創建必要的 topics
sh mqadmin updateTopic -t TG001-chat-service-requests -c DefaultCluster
sh mqadmin updateTopic -t TG001-chat-service-responses -c DefaultCluster  
sh mqadmin updateTopic -t TG001-chat-service-events -c DefaultCluster

# 驗證 topics 創建成功
sh mqadmin topicList -c DefaultCluster
```

### 常見問題與排除方法

#### 1. **連接失敗**
**診斷步驟**：
```bash
# 檢查 NameServer 狀態
kubectl get pods -n rocketmq-system -l app=rocketmq-nameserver

# 檢查 Broker 狀態
kubectl get pods -n rocketmq-system -l app=rocketmq-broker-ondemand

# 檢查服務端點
kubectl get svc -n rocketmq-system

# 測試網絡連通性
kubectl exec -it <your-pod> -- telnet 10.43.171.188 9876
```

**解決方案**：
- 檢查 NameServer 地址是否正確
- 確認網絡連接正常  
- 檢查防火墻設置
- 驗證 Service 和 Endpoints 配置

#### 2. **消息發送失敗**
**診斷步驟**：
```bash
# 檢查 producer 日誌
kubectl logs -l app=rocketmq-server --tail=50

# 檢查 topics 是否存在
kubectl exec -it rocketmq-broker-ondemand-0 -n rocketmq-system -- sh mqadmin topicList -c DefaultCluster

# 檢查 broker 註冊狀態
kubectl logs rocketmq-broker-ondemand-0 -n rocketmq-system | grep "boot success"
```

**解決方案**：
- 檢查 Topic 是否存在
- 確認生產者權限
- 檢查消息格式
- 驗證 broker 連接狀態

#### 3. **消息接收失敗**  
**診斷步驟**：
```bash
# 檢查 consumer 日誌
kubectl logs -l app=rocketmq-client --tail=50

# 檢查消費者組狀態
kubectl exec -it rocketmq-broker-ondemand-0 -n rocketmq-system -- sh mqadmin consumerProgress -g chat_client_group_consumer

# 檢查訂閱關係
kubectl exec -it rocketmq-broker-ondemand-0 -n rocketmq-system -- sh mqadmin updateSubGroup -g chat_client_group_consumer -c DefaultCluster
```

**解決方案**：
- 檢查消費者組名
- 確認訂閱的 Topic 正確
- 檢查消費者權限
- 驗證消息偏移量

### 監控與調試

#### 1. **啟用詳細日誌**
```go
// 在應用中加入詳細日誌
log.Printf("連接 NameServer: %s", nameserver)
log.Printf("創建生產者，組名: %s", groupName+"_producer")
log.Printf("訂閱主題: %s", "TG001-chat-service-requests")
```

#### 2. **檢查連接狀態**
```bash
# 檢查 pod 狀態
kubectl get pods -l app=rocketmq-server -o wide
kubectl get pods -l app=rocketmq-client -o wide

# 檢查服務狀態  
kubectl get svc rocketmq-server-service
kubectl get svc rocketmq-client-service

# 檢查資源使用情況
kubectl top pods -l app=rocketmq-server
kubectl top pods -l app=rocketmq-client
```

#### 3. **網絡連通性測試**
```bash
# 從應用 pod 測試連接 NameServer
kubectl exec -it <pod-name> -- telnet 10.43.171.188 9876

# 測試 broker 連接（內部地址）
kubectl exec -it <pod-name> -- nslookup rocketmq-broker-ondemand-internal.rocketmq-system.svc.cluster.local

# 檢查 DNS 解析
kubectl exec -it <pod-name> -- cat /etc/resolv.conf
```

### 成功驗證指標

#### ✅ 系統正常運行的標誌：
1. **Broker 日誌顯示內部地址**：
   ```
   The broker[broker-ondemand, rocketmq-broker-ondemand-internal.rocketmq-system.svc.cluster.local:10911] boot success.
   ```

2. **Consumer 成功更新偏移量**：
   ```
   time="2025-07-31T01:21:45Z" level=info msg="update offset to broker success" MessageQueue="MessageQueue [topic=TG001-chat-service-events, brokerName=broker-ondemand, queueId=1]"
   ```

3. **Server 和 Client 都正常運行**：
   ```bash
   kubectl get pods -l app=rocketmq-server  # STATUS: Running
   kubectl get pods -l app=rocketmq-client  # STATUS: Running
   ```

4. **消息正常傳遞**：
   ```
   2025/07/31 01:22:07 收到事件消息: {"user_id":"user_001","message":"Hello, everyone!"}
   2025/07/31 01:22:07 处理事件: 用户 user_001 发送消息: Hello, everyone!
   ```

### 部署最佳實踐

1. **健康檢查配置**：應用包含 `/health` 和 `/ready` 端點
2. **資源限制**：設置適當的 CPU 和記憶體限制
3. **環境變數管理**：使用 ConfigMap 和 Secret 管理配置
4. **日誌收集**：結構化日誌輸出，便於監控和調試
5. **優雅關閉**：實現信號處理，確保連接正確關閉 