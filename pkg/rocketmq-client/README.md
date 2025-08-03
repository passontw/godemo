# RocketMQ Client

基於 Apache RocketMQ 的 Go 客戶端，提供簡化的 API 介面，支援多種訊息發送和訂閱模式。

## 🚀 功能特色

### ✅ 已實作功能

#### 🔌 連線管理
- 多組連線管理（同時管理多個 RocketMQ 實例）
- 彈性認證機制（ACL、簡單認證、無認證）
- 連線重試和超時控制
- 優雅關機與 graceful 模組整合
- **🆕 智能 DNS 解析**：自動解析 K8s 服務名稱為 IP 地址，支援域名和 IP 混合格式
- **🆕 DNS 快取機制**：內建 DNS 快取，提升解析效能和穩定性
- **🆕 DNS 重試策略**：支援重試和故障恢復，適應動態環境

#### 📨 統一 API 介面
- **Request-Response 模式**：持久化/非持久化請求響應
- **PUB 模式**：持久化/非持久化訊息發布
- **SUB 模式**：ACK/無ACK 訊息訂閱
- **統一參數格式**：所有方法使用相同的參數結構

#### 📨 訊息發送
- **同步發送**：確保訊息發送成功
- **異步發送**：非阻塞發送，提供回調機制
- **延遲發送**：支援延遲訊息（定時發送）
- **佇列選擇發送**：指定訊息發送到特定佇列

#### 📨 訊息訂閱
- **Push 模式訂閱**：自動推送訊息到消費者
- **Pull 模式訂閱**：手動拉取訊息（預留介面）
- **消費者組管理**：支援多個消費者組

#### 📦 批次訊息
- **批次發送**：一次發送多個訊息，提升吞吐量
- **批次生產者**：自動收集訊息並批次發送
- **批次大小控制**：支援按數量和位元組大小控制
- **定時刷新**：定期發送累積的訊息

#### 🛠️ 管理功能
- **主題管理**：建立、刪除、更新主題
- **消費者組管理**：管理消費者組
- **統計資訊查詢**：主題統計、消費進度等
- **叢集資訊查詢**：Broker 資訊、路由資訊

#### 🔍 監控支援
- **健康檢查**：檢查連線狀態和服務可用性
- **Metrics 回調**：提供效能監控數據
- **日誌整合**：支援自訂日誌介面

#### 🔄 Request-Response 模式
- **同步請求-回應**：支援 RPC 調用模式
- **自動超時管理**：預設 10 秒超時，可自訂
- **記憶體自動清理**：防止請求上下文洩漏
- **狀態監控**：提供待處理請求數量查詢

### 📝 待實作功能

#### 📊 順序訊息 (Ordered Messages)
> **使用場景**：電商訂單狀態更新、金融交易記錄等需要嚴格順序的場景

**核心概念：**
- 確保同一分組的訊息按照發送順序被消費
- 使用 MessageQueue 選擇器將相關訊息發送到同一佇列
- 消費者端使用順序消費監聽器

**實作要點：**
```go
// 順序發送
producer.SendOrderly(msg, orderID, selector)

// 順序消費
consumer.RegisterOrderedMessageListener(func(msgs []Message) ConsumeResult {
    // 順序處理訊息
    return ConsumeSuccess
})
```

#### 🔄 事務訊息 (Transaction Messages)
> **使用場景**：分散式事務、本地事務與訊息發送的最終一致性

**核心概念：**
- 半事務訊息：訊息暫存，但不對消費者可見
- 本地事務執行：執行業務邏輯
- 事務狀態確認：根據本地事務結果決定訊息是否投遞

**實作要點：**
```go
// 事務監聽器
type TransactionListener struct {}

func (tl *TransactionListener) ExecuteLocalTransaction(msg *Message) LocalTransactionState {
    // 執行本地事務
    return CommitMessage // 或 RollbackMessage
}

func (tl *TransactionListener) CheckLocalTransaction(msg *Message) LocalTransactionState {
    // 事務狀態回查
    return CommitMessage
}

// 事務生產者
producer := NewTransactionProducer(listener)
producer.SendMessageInTransaction(msg)
```

## 📦 安裝

```bash
# 如果在同一個專案內使用，直接引用即可
# 無需額外安裝，已包含在專案中
```

## 🔧 配置

### 基本配置

```go
package main

import (
    rocketmqclient "godemo/pkg/rocketmq-client"
)

func main() {
    config := &rocketmqclient.RocketMQConfig{
        Name:        "main",
        NameServers: []string{"192.168.1.100:9876", "192.168.1.101:9876"},
        
        // 認證配置（可選）
        AccessKey: "your-access-key",
        SecretKey: "your-secret-key",
        
        // 錯誤處理配置
        ErrorHandling: &rocketmqclient.ErrorHandlingConfig{
            AllowDegradeToNonPersistent: true,
            MaxPersistentRetries:        3,
            PersistentRetryDelay:        1 * time.Second,
            ContinueOnAckFailure:        false,
            MaxAckRetries:               2,
            AckRetryDelay:               500 * time.Millisecond,
            EnableErrorMetrics:          true,
        },
    }
    
    client, err := rocketmqclient.NewClient(config)
    if err != nil {
        panic(err)
    }
    defer client.Close()
}
```

### 與 Graceful 模組整合

```go
// 註冊健康檢查
gracefulClient.RegisterHealthCheck("rocketmq", func(ctx context.Context) error {
    return client.HealthCheck(ctx)
})

// 註冊優雅關機
gracefulClient.RegisterShutdownHandler(func(ctx context.Context) error {
    return client.Close()
})
```

## 💡 統一 API 使用範例

### 1. Request-Response 模式

```go
// 持久化請求-響應
options := map[string]interface{}{
    "timeout":    10 * time.Second,
    "retryCount": 3,
    "properties": map[string]string{
        "service": "user-service",
        "version": "1.0",
    },
}

response, err := client.SendRequestPersistent(ctx, "service-topic", "calculate", "req-1", 
    []byte(`{"operation": "add", "a": 10, "b": 20}`), options)
if err != nil {
    log.Printf("Request failed: %v", err)
} else {
    log.Printf("Response received: %s", string(response.Body))
}

// 非持久化請求-響應
response, err = client.SendRequestNonPersistent(ctx, "notification-topic", "send", "notif-1", 
    []byte(`{"user_id": "123", "message": "Hello"}`), nil)
```

### 2. PUB 模式

```go
// 持久化發布
options := map[string]interface{}{
    "delayLevel": 3, // 5秒延遲
    "properties": map[string]string{
        "priority": "high",
    },
}

err := client.PublishPersistent(ctx, "order-topic", "created", "order-123", 
    []byte(`{"order_id": "123", "amount": 100.50}`), options)
if err != nil {
    log.Printf("Publish failed: %v", err)
}

// 非持久化發布
err = client.PublishNonPersistent(ctx, "log-topic", "info", "log-1", 
    []byte(`{"level": "info", "message": "User logged in"}`), nil)
```

### 3. SUB 模式

```go
// 訂閱並需要手動確認 (ACK)
options := map[string]interface{}{
    "consumerGroup": "order-processor-group",
    "consumeMode":   "clustering",
    "retryCount":    3,
}

handler := func(ctx context.Context, topic, tag, key string, body []byte, properties map[string]string) error {
    log.Printf("Processing message: topic=%s, tag=%s, key=%s, body=%s", topic, tag, key, string(body))
    
    // 處理業務邏輯
    if tag == "created" {
        // 處理訂單創建
        log.Println("Processing order creation...")
    }
    
    return nil // 返回 nil 表示處理成功，會發送 ACK
}

err := client.SubscribeWithAck(ctx, "order-topic", "*", "", handler, options)
if err != nil {
    log.Printf("Subscribe failed: %v", err)
}

// 訂閱但不需要確認 (無 ACK)
handler := func(ctx context.Context, topic, tag, key string, body []byte, properties map[string]string) error {
    log.Printf("Processing log: topic=%s, tag=%s, key=%s, body=%s", topic, tag, key, string(body))
    
    // 處理日誌邏輯
    log.Println("Processing log message...")
    
    // 即使處理失敗，也不會重試（無 ACK 模式）
    return fmt.Errorf("simulated error")
}

err = client.SubscribeWithoutAck(ctx, "log-topic", "*", "", handler, nil)
```

### 4. 選項配置說明

```go
options := map[string]interface{}{
    // 持久化設定
    "persistent": true,
    
    // 超時設定
    "timeout": 30 * time.Second,
    
    // 重試設定
    "retryCount": 3,
    "retryDelay": 1 * time.Second,
    
    // 消費者設定
    "consumerGroup": "my-group",
    "consumeMode":   "clustering", // "clustering" 或 "broadcasting"
    
    // 批次設定
    "batchSize":  10,
    "batchBytes": 1024 * 1024, // 1MB
    "flushTime":  100 * time.Millisecond,
    
    // 延遲設定
    "delayLevel": 3, // 對應 5 秒延遲
    
    // 自定義屬性
    "properties": map[string]string{
        "service": "user-service",
        "version": "1.0",
        "priority": "high",
    },
}
```

## 🔍 監控與日誌

### 自訂日誌

```go
type MyLogger struct{}

func (l *MyLogger) Infof(format string, args ...interface{}) {
    log.Printf("[INFO] "+format, args...)
}

func (l *MyLogger) Errorf(format string, args ...interface{}) {
    log.Printf("[ERROR] "+format, args...)
}

// 設定自訂日誌
client.SetLogger(&MyLogger{})
```

### Metrics 回調

```go
// 設定 Metrics 回調
client.SetMetrics(func(event string, labels map[string]string, value float64) {
    // 發送到 Prometheus、InfluxDB 等
    fmt.Printf("Metric: %s, Labels: %v, Value: %.2f\n", event, labels, value)
})
```

## 📋 統一 API 參考

### 主要介面

```go
// 客戶端介面
type Client struct {
    // Request-Response 模式
    SendRequestPersistent(ctx context.Context, topic, tag, key string, body []byte, options map[string]interface{}) (*ResponseMessage, error)
    SendRequestNonPersistent(ctx context.Context, topic, tag, key string, body []byte, options map[string]interface{}) (*ResponseMessage, error)
    
    // PUB 模式
    PublishPersistent(ctx context.Context, topic, tag, key string, body []byte, options map[string]interface{}) error
    PublishNonPersistent(ctx context.Context, topic, tag, key string, body []byte, options map[string]interface{}) error
    
    // SUB 模式
    SubscribeWithAck(ctx context.Context, topic, tag, key string, handler UnifiedMessageHandler, options map[string]interface{}) error
    SubscribeWithoutAck(ctx context.Context, topic, tag, key string, handler UnifiedMessageHandler, options map[string]interface{}) error
    
    // 健康檢查
    HealthCheck(ctx context.Context) error
    
    // 優雅關機
    Close() error
}

// 統一訊息處理函數
type UnifiedMessageHandler func(ctx context.Context, topic, tag, key string, body []byte, properties map[string]string) error
```

### 配置結構

```go
// RocketMQ 配置
type RocketMQConfig struct {
    Name        string   // 連線名稱
    NameServers []string // Name Server 位址
    
    // 認證配置
    AccessKey     string
    SecretKey     string
    SecurityToken string
    Namespace     string
    
    // 連線配置
    Retry   *RetryConfig
    Timeout *TimeoutConfig
    DNS     *DNSConfig // DNS 解析配置
    
    // 錯誤處理配置
    ErrorHandling *ErrorHandlingConfig
}

// 錯誤處理配置
type ErrorHandlingConfig struct {
    // 持久化失敗處理
    AllowDegradeToNonPersistent bool          // 允許降級為非持久化
    MaxPersistentRetries        int           // 最大持久化重試次數
    PersistentRetryDelay        time.Duration // 持久化重試延遲
    
    // ACK 失敗處理
    ContinueOnAckFailure        bool          // ACK 失敗時繼續處理
    MaxAckRetries               int           // 最大 ACK 重試次數
    AckRetryDelay               time.Duration // ACK 重試延遲
    
    // 通用錯誤處理
    EnableErrorMetrics          bool          // 啟用錯誤指標
    ErrorCallback               func(error)   // 錯誤回調函數
}

// 訊息選項
type MessageOptions struct {
    // 持久化設定
    Persistent bool
    
    // 超時設定
    Timeout time.Duration
    
    // 重試設定
    RetryCount int
    RetryDelay time.Duration
    
    // 消費者設定
    ConsumerGroup string
    ConsumeMode   string // "clustering" 或 "broadcasting"
    
    // 批次設定
    BatchSize   int
    BatchBytes  int
    FlushTime   time.Duration
    
    // 延遲設定
    DelayLevel int
    
    // 自定義屬性
    Properties map[string]string
}
```

## 🔄 遷移指南

### 從舊 API 遷移到統一 API

| 舊 API | 新統一 API | 說明 |
|--------|------------|------|
| `client.Send(msg)` | `client.PublishPersistent(ctx, topic, tag, key, body, options)` | 持久化發布 |
| `client.SendAsync(msg, callback)` | `client.PublishNonPersistent(ctx, topic, tag, key, body, options)` | 非持久化發布 |
| `client.Subscribe(config, handler)` | `client.SubscribeWithAck(ctx, topic, tag, key, handler, options)` | 訂閱並確認 |
| `client.Request(req)` | `client.SendRequestPersistent(ctx, topic, tag, key, body, options)` | 持久化請求 |

### 配置對應

| 舊配置 | 新配置 | 說明 |
|--------|--------|------|
| `SendMessage` | `topic, tag, key, body, options` | 統一參數格式 |
| `SubscribeConfig` | `options` | 選項配置 |
| `RequestMessage` | `topic, tag, key, body, options` | 統一參數格式 |

## 🏗️ 架構設計

```
┌─────────────────────────────────────────────────────────────┐
│                    RocketMQ Client                          │
├─────────────────────────────────────────────────────────────┤
│  統一 API 層                                                │
│  - SendRequestPersistent/NonPersistent                     │
│  - PublishPersistent/NonPersistent                         │
│  - SubscribeWithAck/WithoutAck                             │
├─────────────────────────────────────────────────────────────┤
│  內部實現層                                                 │
│  - sendRequest()                                           │
│  - publishMessage()                                        │
│  - subscribeMessage()                                       │
├─────────────────────────────────────────────────────────────┤
│                 Apache RocketMQ Go SDK                     │
├─────────────────────────────────────────────────────────────┤
│                    RocketMQ Cluster                        │
│  NameServer    │    Broker     │    Broker     │    Broker  │
└─────────────────────────────────────────────────────────────┘
```

## 🛠️ 開發與測試

### 開發環境設定

```bash
# 下載依賴
go mod download

# 程式碼檢查
go vet ./...

# 執行測試
go test ./... -v
```

### 效能測試

```bash
# 基準測試
go test -bench=. -benchmem

# 壓力測試
go test -run=TestStress -timeout=30s
```

## 📄 許可證

MIT License

## 🤝 貢獻

歡迎提交 Issue 和 Pull Request！

## 📞 支援

如有問題，請建立 GitHub Issue 或聯繫維護團隊。

---

## 🔗 相關連結

- [Apache RocketMQ 官方文件](https://rocketmq.apache.org/)
- [RocketMQ Go SDK](https://github.com/apache/rocketmq-client-go)
- [RocketMQ 最佳實踐](https://rocketmq.apache.org/docs/bestPractice/)

## 📈 版本歷史

### v2.1.0 (當前版本)
- ✅ 統一 API 設計
- ✅ Request-Response 持久化/非持久化
- ✅ PUB 持久化/非持久化
- ✅ SUB ACK/無ACK
- ✅ 錯誤處理配置整合
- ✅ 優雅關機支援
- ✅ 健康檢查和監控
- **🆕 智能 DNS 解析**：支援 K8s 服務名稱自動解析
- **🆕 DNS 快取機制**：提升連接效能和穩定性
- **🆕 混合格式支援**：同時支援 IP 地址和域名格式

### 未來版本計劃
- v2.2.0: 順序訊息支援
- v2.3.0: 事務訊息支援
- v2.4.0: 進階監控和指標
- v2.5.0: 效能優化和快取機制

---

## 🌐 DNS 解析功能

### 功能概述

新增的 DNS 解析功能讓 RocketMQ 客戶端能夠：

- **自動解析 K8s 服務名稱**：將 `rocketmq-nameserver.rocketmq-system.svc.cluster.local:9876` 解析為實際 IP 地址
- **支援混合格式**：同時支援域名和 IP 地址格式的 NameServer 配置
- **智能快取機制**：內建 DNS 快取，避免重複解析，提升效能
- **自動重試**：DNS 解析失敗時自動重試，增強穩定性
- **動態刷新**：支援手動刷新 DNS 快取，適應動態環境

### 使用方式

#### 1. 基本使用（自動 DNS 解析）

```go
import rocketmqclient "godemo/pkg/rocketmq-client"

// K8s 環境配置
config := rocketmqclient.RocketMQConfig{
    Name: "k8s_client",
    NameServers: []string{
        "rocketmq-nameserver.rocketmq-system.svc.cluster.local:9876",
    },
    // DNS 配置會使用默認值，自動解析域名
}

// 註冊客戶端（會自動解析域名）
ctx := context.Background()
if err := rocketmqclient.Register(ctx, config); err != nil {
    log.Fatalf("註冊失敗: %v", err)
}
```

#### 2. 混合格式支援

```go
// 支援 IP 地址和域名混合使用
config := rocketmqclient.RocketMQConfig{
    Name: "mixed_client",
    NameServers: []string{
        "192.168.1.100:9876",  // IP 地址格式
        "rocketmq-nameserver.rocketmq-system.svc.cluster.local:9876", // 域名格式
        "10.0.0.50:9876",      // IP 地址格式
    },
}
```

#### 3. 自定義 DNS 配置

```go
config := rocketmqclient.RocketMQConfig{
    Name: "custom_dns_client",
    NameServers: []string{
        "rocketmq-nameserver.rocketmq-system.svc.cluster.local:9876",
    },
    DNS: &rocketmqclient.DNSConfig{
        UseFQDN:         true,                    // 使用完整域名
        ResolveTimeout:  3 * time.Second,         // DNS 解析超時
        RefreshInterval: 30 * time.Second,        // 快取刷新間隔
        EnableCache:     true,                    // 啟用快取
        MaxRetries:      5,                       // 最大重試次數
        RetryInterval:   500 * time.Millisecond,  // 重試間隔
    },
}
```

#### 4. DNS 管理功能

```go
// 獲取客戶端
client := rocketmqclient.Get("k8s_client")

// 手動刷新 DNS 快取
if err := client.RefreshDNS(); err != nil {
    log.Printf("刷新 DNS 失敗: %v", err)
}

// 獲取 DNS 統計信息
stats := client.GetDNSStats()
log.Printf("DNS 統計: %+v", stats)

// 獲取 DNS 解析器進行進階操作
resolver := client.GetDNSResolver()
if resolver != nil {
    // 手動解析特定域名
    resolved, err := resolver.ResolveWithRetry("example.com:9876")
    if err != nil {
        log.Printf("解析失敗: %v", err)
    } else {
        log.Printf("解析結果: %s", resolved)
    }
}
```

### 配置參數說明

#### DNSConfig 參數

| 參數 | 類型 | 預設值 | 說明 |
|------|------|--------|------|
| `UseFQDN` | bool | true | 是否使用完整域名（K8s 建議） |
| `ResolveTimeout` | Duration | 5s | DNS 解析超時時間 |
| `RefreshInterval` | Duration | 30s | 快取刷新間隔 |
| `EnableCache` | bool | true | 是否啟用 DNS 快取 |
| `MaxRetries` | int | 3 | DNS 解析最大重試次數 |
| `RetryInterval` | Duration | 1s | 重試間隔時間 |

### 環境變數支援

可以通過環境變數配置 NameServer：

```bash
# 設定 K8s 服務名稱
export ROCKETMQ_NAMESERVER="rocketmq-nameserver.rocketmq-system.svc.cluster.local:9876"

# 設定多個 NameServer（逗號分隔）
export ROCKETMQ_NAMESERVER="server1.example.com:9876,server2.example.com:9876"
```

### 日誌範例

啟用 DNS 解析後，會看到類似以下的日誌：

```
[INFO] 解析域名 rocketmq-nameserver.rocketmq-system.svc.cluster.local:9876 -> 10.96.1.100:9876
[INFO] DNS 快取已刷新，解析結果: [rocketmq-nameserver.rocketmq-system.svc.cluster.local:9876] -> [10.96.1.100:9876]
```

### 故障排除

#### 常見問題

1. **DNS 解析失敗**
   ```
   DNS 解析失敗，已重試 3 次，最後錯誤: lookup rocketmq-nameserver.rocketmq-system.svc.cluster.local: no such host
   ```
   - 檢查 K8s 服務是否存在
   - 確認命名空間和服務名稱正確
   - 檢查 DNS 服務是否正常運作

2. **混用格式錯誤**
   ```
   無效的 nameserver 格式: missing port in address
   ```
   - 確保所有 NameServer 都包含端口號
   - 格式必須是 `host:port`

#### 除錯建議

```go
// 啟用詳細日誌
client.SetLogger(&SimpleLogger{})

// 檢查 DNS 統計
stats := client.GetDNSStats()
log.Printf("DNS 統計: %+v", stats)

// 測試單個域名解析
resolver := client.GetDNSResolver()
resolved, err := resolver.ResolveWithRetry("your-service:9876")
``` 