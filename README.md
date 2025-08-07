# RocketMQ Go 示例項目

## 📋 概述

本項目演示了如何使用 RocketMQ 實現 Request-Response 和 Publish-Subscribe 模式，並提供了完整的 Docker 鏡像構建和 Kubernetes 部署方案，支持多環境部署和固定 IP 配置。

## 🏗️ 項目結構

```
godemo/
├── server/                 # 服務端代碼
│   └── main.go
├── client/                 # 客戶端代碼
│   └── main.go
├── k8s/                   # Kubernetes 部署文件
│   ├── server-deployment.yaml
│   ├── client-deployment.yaml
│   └── create-topics-job.yaml
├── Dockerfile.server       # 服務端 Dockerfile
├── Dockerfile.client       # 客戶端 Dockerfile
├── Dockerfile.topics       # Topics 創建工具 Dockerfile
├── build-and-push.sh      # 構建和推送腳本
├── deploy.sh              # 多環境部署腳本

├── run-local.sh           # 本地測試腳本
└── go.mod                 # Go 模塊文件
```

## 🚀 快速開始

### 1. 安裝依賴
```bash
go mod tidy
```

### 2. 生成 Protobuf 檔案

#### 安裝 protoc 編譯器
```bash
# macOS
brew install protobuf

# Ubuntu/Debian
sudo apt-get install protobuf-compiler

# CentOS/RHEL
sudo yum install protobuf-compiler

# 或者從官方下載
# https://github.com/protocolbuffers/protobuf/releases
```

#### 安裝 Go protobuf 插件
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

#### 生成 pb.go 檔案
```bash
# 進入 proto 目錄
cd message_manager/proto

# 生成 pb.go 檔案
protoc --go_out=. --go_opt=paths=source_relative request.proto

# 或者使用完整路徑
protoc --go_out=. --go_opt=paths=source_relative \
  --proto_path=. \
  request.proto
```

#### 自動化腳本（可選）
```bash
# 創建生成腳本
cat > generate-proto.sh << 'EOF'
#!/bin/bash
cd message_manager/proto
protoc --go_out=. --go_opt=paths=source_relative request.proto
echo "Protobuf 檔案生成完成"
EOF

# 設置執行權限
chmod +x generate-proto.sh

# 執行生成
./generate-proto.sh
```

### 3. 運行示例

#### 方法一：使用啟動腳本
```bash
# 啟動服務端
./run.sh server

# 啟動客戶端（在另一個終端）
./run.sh client

# 或者同時啟動服務端和客戶端
./run.sh both
```

#### 方法二：本地測試環境
```bash
# 本地運行（需要本地 RocketMQ）
./run-local.sh localhost:9876

# 連接到遠程 RocketMQ
./run-local.sh 192.168.1.100:9876
```

#### 方法三：直接運行
```bash
# 啟動服務端
export ROCKETMQ_NAMESERVER="rocketmq-nameserver.rocketmq-system.svc.cluster.local:9876"
export ROCKETMQ_GROUP="chat_server_group"
go run server/main.go

# 啟動客戶端（在另一個終端）
export ROCKETMQ_NAMESERVER="rocketmq-nameserver.rocketmq-system.svc.cluster.local:9876"
export ROCKETMQ_GROUP="chat_client_group"
export USER_ID="user_001"
go run client/main.go
```

### 3. 預期輸出

#### 服務端輸出
```
2024/01/01 12:00:00 聊天服務端已啟動，監聽請求和事件...
2024/01/01 12:00:05 事件已發布: SendResult [MessageId: xxx, QueueId: 0, QueueOffset: 0]
2024/01/01 12:00:05 事件已發布: SendResult [MessageId: xxx, QueueId: 0, QueueOffset: 1]
2024/01/01 12:00:10 收到請求消息: {"request_id":"req_xxx","user_id":"user_001","action":"send_message",...}
2024/01/01 12:00:10 處理請求: req_xxx, 用戶: user_001, 動作: send_message
2024/01/01 12:00:10 消息已發送: map[message_id:msg_xxx status:sent]
2024/01/01 12:00:10 響應已發送: SendResult [MessageId: xxx, QueueId: 0, QueueOffset: 2]
```

#### 客戶端輸出
```
2024/01/01 12:00:00 聊天客戶端已啟動，用戶: user_001
2024/01/01 12:00:02 發布用戶加入事件...
2024/01/01 12:00:02 事件已發布: SendResult [MessageId: xxx, QueueId: 0, QueueOffset: 0]
2024/01/01 12:00:04 發送消息請求...
2024/01/01 12:00:04 請求已發送: SendResult [MessageId: xxx, QueueId: 0, QueueOffset: 1]
2024/01/01 12:00:04 收到響應消息: {"request_id":"req_xxx","success":true,"data":{...}}
2024/01/01 12:00:04 收到響應: &{RequestID:req_xxx Success:true Data:map[...] Error:}
```

## 🐳 Docker 鏡像構建

### 1. 構建和推送鏡像

```bash
# 執行構建和推送腳本
./build-and-push.sh
```

這將：
- 構建服務端鏡像：`harbor.trevi-dev.cc/cpp_run/rocketmq-server:latest`
- 構建客戶端鏡像：`harbor.trevi-dev.cc/cpp_run/rocketmq-client:latest`
- 推送鏡像到 Harbor 倉庫

### 2. 手動構建（可選）

```bash
# 構建服務端鏡像
docker build -f Dockerfile.server -t harbor.trevi-dev.cc/cpp_run/rocketmq-server:latest .

# 構建客戶端鏡像
docker build -f Dockerfile.client -t harbor.trevi-dev.cc/cpp_run/rocketmq-client:latest .

# 推送鏡像
docker push harbor.trevi-dev.cc/cpp_run/rocketmq-server:latest
docker push harbor.trevi-dev.cc/cpp_run/rocketmq-client:latest
```

## ☸️ Kubernetes 部署

### 1. 多環境部署

#### Kubernetes 環境部署
```bash
# 部署到 Kubernetes 環境
./deploy.sh k8s default v1.0.0

# 或者使用默認參數
./deploy.sh
```

#### 測試環境部署
```bash
# 部署到測試環境
./deploy.sh test default latest
```



### 3. 手動部署

```bash
# 構建並推送 topics 創建工具
docker build -f Dockerfile.topics -t harbor.trevi-dev.cc/cpp_run/rocketmq-topics:latest .
docker push harbor.trevi-dev.cc/cpp_run/rocketmq-topics:latest

# 創建 RocketMQ topics
kubectl apply -f k8s/create-topics-job.yaml

# 部署服務端
kubectl apply -f k8s/server-deployment.yaml

# 部署客戶端
kubectl apply -f k8s/client-deployment.yaml

# 查看部署狀態
kubectl get pods -l app=rocketmq-server
kubectl get pods -l app=rocketmq-client

# 查看日誌
kubectl logs -l app=rocketmq-server
kubectl logs -l app=rocketmq-client
```

## 🔧 配置說明

### 環境變數

| 變數名 | 描述 | 默認值 | 示例 |
|--------|------|--------|------|
| `ROCKETMQ_NAMESERVER` | RocketMQ NameServer 地址 | 根據環境自動設置 | `localhost:9876` |
| `ROCKETMQ_ENVIRONMENT` | 環境類型 | `k8s` | `test` |
| `ROCKETMQ_GROUP` | 消費者組名稱 | `chat_server_group` | `my_group` |
| `USER_ID` | 用戶 ID | `user_001` | `test_user` |

### Kubernetes 配置

#### 服務端部署
- **鏡像**: `harbor.trevi-dev.cc/cpp_run/rocketmq-server:latest`
- **端口**: 8080 (健康檢查)
- **資源限制**: 128Mi 內存，100m CPU
- **健康檢查**: `/health` 和 `/ready` 端點

#### 客戶端部署
- **鏡像**: `harbor.trevi-dev.cc/cpp_run/rocketmq-client:latest`
- **資源限制**: 128Mi 內存，100m CPU
- **一次性任務**: 發送請求後退出

## 📊 監控和日誌

### 查看 Pod 狀態

```bash
# 查看所有相關 Pod
kubectl get pods -l app=rocketmq-server
kubectl get pods -l app=rocketmq-client

# 查看服務
kubectl get services -l app=rocketmq-server
kubectl get services -l app=rocketmq-client
```

### 查看日誌

```bash
# 查看服務端日誌
kubectl logs -l app=rocketmq-server --tail=100 -f

# 查看客戶端日誌
kubectl logs -l app=rocketmq-client --tail=100 -f
```

### 健康檢查

```bash
# 檢查服務端健康狀態
kubectl exec -it $(kubectl get pods -l app=rocketmq-server -o jsonpath='{.items[0].metadata.name}') -- curl http://localhost:8080/health

# 檢查服務端就緒狀態
kubectl exec -it $(kubectl get pods -l app=rocketmq-server -o jsonpath='{.items[0].metadata.name}') -- curl http://localhost:8080/ready
```

### 自動 IP 監控

```bash
# 檢查服務狀態
kubectl get pods,services -l app=rocketmq-server
kubectl get pods,services -l app=rocketmq-client
```



## 🔍 故障排除

### 常見問題

1. **連接問題**
   ```bash
   # 檢查網絡連接
   telnet rocketmq-nameserver.rocketmq-system.svc.cluster.local 9876

   # 檢查環境變數
   echo $ROCKETMQ_NAMESERVER
   echo $ROCKETMQ_GROUP
   ```

2. **編譯問題**
   ```bash
   # 清理並重新安裝依賴
   go clean -modcache
   go mod tidy
   ```

3. **運行時問題**
   ```bash
   # 啟用詳細日誌
   export ROCKETMQ_LOG_LEVEL=debug

   # 檢查端口佔用
   netstat -tlnp | grep 9876
   ```

4. **鏡像拉取失敗**
   ```bash
   # 檢查 Harbor 登錄狀態
   docker login harbor.trevi-dev.cc
   ```

5. **RocketMQ 連接失敗**
   ```bash
   # 檢查 RocketMQ 服務狀態
   kubectl get pods -n rocketmq-system
   
   # 檢查 NameServer 服務
   kubectl get services -n rocketmq-system
   ```

6. **Pod 啟動失敗**
   ```bash
   # 查看 Pod 詳細狀態
   kubectl describe pod <pod-name>
   
   # 查看 Pod 事件
   kubectl get events --sort-by='.lastTimestamp'
   ```

7. **健康檢查失敗**
   ```bash
   # 檢查服務端是否正常監聽
   kubectl exec -it <server-pod-name> -- netstat -tlnp
   ```

### 調試命令

```bash
# 進入 Pod 調試
kubectl exec -it <pod-name> -- /bin/sh

# 查看環境變數
kubectl exec -it <pod-name> -- env

# 測試網絡連接
kubectl exec -it <pod-name> -- nslookup rocketmq-nameserver.rocketmq-system.svc.cluster.local
```

## 🧹 清理

### 刪除部署

```bash
# 刪除所有相關資源
kubectl delete -f k8s/server-deployment.yaml
kubectl delete -f k8s/client-deployment.yaml

# 或者刪除所有相關 Pod 和服務
kubectl delete pods,services -l app=rocketmq-server
kubectl delete pods,services -l app=rocketmq-client
```

### 刪除鏡像（可選）

```bash
# 從 Harbor 刪除鏡像（需要管理員權限）
docker rmi harbor.trevi-dev.cc/cpp_run/rocketmq-server:latest
docker rmi harbor.trevi-dev.cc/cpp_run/rocketmq-client:latest
```

## 📝 注意事項

1. **RocketMQ 依賴**: 確保 Kubernetes 集群中已部署 RocketMQ
2. **網絡策略**: 確保 Pod 可以訪問 RocketMQ 服務
3. **資源限制**: 根據實際需求調整 CPU 和內存限制
4. **鏡像版本**: 建議使用特定版本標籤而不是 `latest`
5. **日誌輪轉**: 生產環境建議配置日誌輪轉和持久化

## 🚀 擴展建議

1. **水平擴展**: 可以部署多個服務端實例
2. **配置管理**: 使用 ConfigMap 管理配置
3. **密鑰管理**: 使用 Secret 管理敏感信息
4. **監控集成**: 集成 Prometheus 和 Grafana
5. **日誌聚合**: 使用 ELK 或 Fluentd 進行日誌聚合

## 🎯 最佳實踐

### 1. **配置管理**
- 使用 ConfigMap 集中管理配置
- 為不同環境創建不同的配置文件
- 使用環境變數覆蓋默認配置

### 2. **監控和告警**
- 設置日誌監控和告警
- 定期檢查服務健康狀態

### 3. **部署策略**
- 使用藍綠部署或滾動更新
- 設置資源限制和健康檢查
- 配置自動擴縮容

### 4. **安全考慮**
- 使用 RBAC 控制訪問權限
- 配置網絡策略
- 定期更新鏡像和依賴



## 🛠️ 實用工具

### 監控腳本
```bash
# 檢查服務狀態
kubectl get pods,services -l app=rocketmq-server
kubectl get pods,services -l app=rocketmq-client
```

### 部署腳本
```bash
# 部署到 Kubernetes
./deploy.sh

# 手動部署
kubectl apply -f k8s/server-deployment.yaml
kubectl apply -f k8s/client-deployment.yaml
```

### 本地測試
```bash
# 本地運行
./run-local.sh <nameserver-address>
```



## 🎉 總結

這個多環境解決方案提供了：

✅ **靈活性** - 支持 Kubernetes 和測試環境  
✅ **簡潔性** - 簡化的部署配置  
✅ **穩定性** - 直接使用固定 IP 地址  
✅ **可維護性** - 簡化的配置管理  
✅ **兼容性** - 與 RocketMQ Go 客戶端完全兼容  

這個簡化的解決方案確保服務的穩定運行和易於維護。 