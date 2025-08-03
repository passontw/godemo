package rocketmqclient

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
)

// Logger 介面（可用 zap、logrus 等實作）
type Logger interface {
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// MetricsCallback 介面
// event: 事件名稱，如 "send", "consume", "topic_create" 等
// labels: 可選標籤
// value: 數值（如耗時、次數等）
type MetricsCallback func(event string, labels map[string]string, value float64)

type Client struct {
	config      RocketMQConfig
	producer    rocketmq.Producer
	consumer    rocketmq.PushConsumer
	logger      Logger
	metrics     MetricsCallback
	dnsResolver *DNSResolver // DNS 解析器

	// 解析後的 NameServers
	resolvedNameServers []string

	// 狀態追蹤
	consumerStarted bool // 追蹤 consumer 是否已經成功啟動

	// 優雅關機相關
	mu              sync.RWMutex
	processingCount int64         // 正在處理的訊息數量
	shutdownStarted bool          // 是否已開始關機
	processingDone  chan struct{} // 等待處理完成的通道
}

var (
	clients     = make(map[string]*Client)
	clientsMu   sync.RWMutex
	defaultName string
)

// Register 註冊一組 RocketMQ 連線（可多次呼叫，支援多組連線）
//
// 參數：
//
//	ctx - context 控制 timeout/cancel
//	cfg - RocketMQConfig 連線設定，需包含 NameServers、認證等
//
// 回傳：
//
//	error - 若參數不合法、名稱重複、連線失敗等，回傳對應錯誤
func Register(ctx context.Context, cfg RocketMQConfig) error {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	if cfg.Name == "" {
		cfg.Name = "default"
	}
	if _, ok := clients[cfg.Name]; ok {
		return ErrAlreadyExists
	}

	// 參數驗證
	if len(cfg.NameServers) == 0 {
		return ErrInvalidArgument
	}

	// 設定預設值
	if cfg.Retry == nil {
		cfg.Retry = DefaultRetryConfig()
	}
	if cfg.Timeout == nil {
		cfg.Timeout = DefaultTimeoutConfig()
	}
	if cfg.DNS == nil {
		cfg.DNS = DefaultDNSConfig()
	}
	if cfg.ErrorHandling == nil {
		cfg.ErrorHandling = DefaultErrorHandlingConfig()
	}

	// 創建 DNS 解析器
	dnsResolver := NewDNSResolver(cfg.DNS)

	// 解析 NameServers
	resolvedNameServers, err := dnsResolver.ResolveNameServers(cfg.NameServers)
	if err != nil {
		return err
	}

	// 建立 RocketMQ 生產者（使用解析後的 NameServers）
	producerOpts := []producer.Option{
		producer.WithNameServer(resolvedNameServers),
		producer.WithRetry(cfg.Retry.MaxRetries),
		producer.WithSendMsgTimeout(cfg.Timeout.SendTimeout),
	}

	// 認證配置
	if cfg.AccessKey != "" && cfg.SecretKey != "" {
		credentials := &primitive.Credentials{
			AccessKey: cfg.AccessKey,
			SecretKey: cfg.SecretKey,
		}
		if cfg.SecurityToken != "" {
			credentials.SecurityToken = cfg.SecurityToken
		}
		producerOpts = append(producerOpts, producer.WithCredentials(*credentials))
	}

	// 命名空間
	if cfg.Namespace != "" {
		producerOpts = append(producerOpts, producer.WithNamespace(cfg.Namespace))
	}

	// 建立生產者
	prod, err := rocketmq.NewProducer(producerOpts...)
	if err != nil {
		return err
	}

	// 啟動生產者
	if err := prod.Start(); err != nil {
		return err
	}

	// 建立 RocketMQ 消費者（延遲初始化，使用解析後的 NameServers）
	consumerOpts := []consumer.Option{
		consumer.WithNameServer(resolvedNameServers),
	}

	// 認證配置
	if cfg.AccessKey != "" && cfg.SecretKey != "" {
		credentials := &primitive.Credentials{
			AccessKey: cfg.AccessKey,
			SecretKey: cfg.SecretKey,
		}
		if cfg.SecurityToken != "" {
			credentials.SecurityToken = cfg.SecurityToken
		}
		consumerOpts = append(consumerOpts, consumer.WithCredentials(*credentials))
	}

	// 命名空間
	if cfg.Namespace != "" {
		consumerOpts = append(consumerOpts, consumer.WithNamespace(cfg.Namespace))
	}

	// 建立消費者（但不啟動）
	cons, err := rocketmq.NewPushConsumer(consumerOpts...)
	if err != nil {
		prod.Shutdown()
		return err
	}

	clients[cfg.Name] = &Client{
		config:              cfg,
		producer:            prod,
		consumer:            cons,
		dnsResolver:         dnsResolver,
		resolvedNameServers: resolvedNameServers,
		processingDone:      make(chan struct{}),
	}
	if defaultName == "" {
		defaultName = cfg.Name
	}
	return nil
}

// GetClient 取得指定名稱的 client
func GetClient(name string) (*Client, error) {
	clientsMu.RLock()
	defer clientsMu.RUnlock()
	c, ok := clients[name]
	if !ok {
		return nil, errors.New("rocketmq client not found")
	}
	return c, nil
}

// Get 取得指定名稱的 client (簡短版本)
func Get(name string) *Client {
	client, _ := GetClient(name)
	return client
}

// Default 取得預設 client
func Default() (*Client, error) {
	return GetClient(defaultName)
}

// CloseAll 關閉所有連線
func CloseAll() {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	for _, c := range clients {
		if c.producer != nil {
			c.producer.Shutdown()
		}
		// 只有在 consumer 成功啟動後才關閉
		if c.consumer != nil && c.consumerStarted {
			c.consumer.Shutdown()
		}
	}
	clients = make(map[string]*Client)
	defaultName = ""
}

// Close - 關閉連線
func (c *Client) Close() {
	if c.producer != nil {
		c.producer.Shutdown()
	}
	// 只有在 consumer 成功啟動後才關閉
	if c.consumer != nil && c.consumerStarted {
		c.consumer.Shutdown()
	}
}

// StartGracefulShutdown 開始優雅關閉，拒絕新的消息處理
func (c *Client) StartGracefulShutdown() {
	c.startShutdown()
}

// WaitForProcessingComplete 等待所有正在處理的消息完成
func (c *Client) WaitForProcessingComplete(ctx context.Context) error {
	return c.waitForProcessingComplete(ctx)
}

// SetLogger 設定自訂 logger，支援 zap、logrus 等
// logger - 實作 Logger 介面的物件
func (c *Client) SetLogger(logger Logger) {
	c.logger = logger
}

// SetMetrics 設定自訂 metrics callback
// cb - 事件回報 callback，event/labels/value
func (c *Client) SetMetrics(cb MetricsCallback) {
	c.metrics = cb
}

// incrementProcessing 增加正在處理的訊息計數
func (c *Client) incrementProcessing() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 如果已經開始關機，拒絕新的處理
	if c.shutdownStarted {
		return false
	}

	c.processingCount++
	return true
}

// decrementProcessing 減少正在處理的訊息計數
func (c *Client) decrementProcessing() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.processingCount--
	if c.processingCount <= 0 && c.shutdownStarted {
		close(c.processingDone)
	}
}

// isShuttingDown 檢查是否正在關機
func (c *Client) isShuttingDown() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.shutdownStarted
}

// startShutdown 開始關機流程
func (c *Client) startShutdown() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.shutdownStarted {
		return
	}

	c.shutdownStarted = true

	// 如果沒有正在處理的訊息，立即關閉通道
	if c.processingCount <= 0 {
		close(c.processingDone)
	}
}

// waitForProcessingComplete 等待所有訊息處理完成
func (c *Client) waitForProcessingComplete(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.processingDone:
		return nil
	}
}

// GetProducer 取得生產者實例
func (c *Client) GetProducer() rocketmq.Producer {
	return c.producer
}

// GetConsumer 取得消費者實例
func (c *Client) GetConsumer() rocketmq.PushConsumer {
	return c.consumer
}

// GetConfig 取得配置
func (c *Client) GetConfig() RocketMQConfig {
	return c.config
}

// GetDNSResolver 取得 DNS 解析器
func (c *Client) GetDNSResolver() *DNSResolver {
	return c.dnsResolver
}

// RefreshDNS 手動刷新 DNS 快取並重新解析 NameServers
func (c *Client) RefreshDNS() error {
	if c.dnsResolver == nil {
		return fmt.Errorf("DNS 解析器未初始化")
	}

	// 清除快取
	c.dnsResolver.ClearCache()

	// 重新解析 NameServers
	resolvedNameServers, err := c.dnsResolver.ResolveNameServers(c.config.NameServers)
	if err != nil {
		return fmt.Errorf("重新解析 NameServers 失敗: %v", err)
	}

	// 記錄日誌
	if c.logger != nil {
		c.logger.Infof("DNS 快取已刷新，解析結果: %v -> %v", c.config.NameServers, resolvedNameServers)
	}

	return nil
}

// GetDNSStats 取得 DNS 統計信息
func (c *Client) GetDNSStats() map[string]interface{} {
	if c.dnsResolver == nil {
		return map[string]interface{}{
			"dns_enabled": false,
		}
	}

	stats := c.dnsResolver.GetCacheStats()
	stats["dns_enabled"] = true
	stats["original_nameservers"] = c.config.NameServers

	return stats
}
