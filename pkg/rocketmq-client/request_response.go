package rocketmqclient

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/google/uuid"
)

// RequestStatus 請求狀態
type RequestStatus int

const (
	RequestPending RequestStatus = iota
	RequestProcessing
	RequestSuccess
	RequestTimeout
	RequestFailed
)

// RequestContext 請求上下文
type RequestContext struct {
	CorrelationID string
	RequestTime   time.Time
	Timeout       time.Duration
	ResponseChan  chan *ResponseMessage
	Status        RequestStatus
	mu            sync.RWMutex
}

// GetStatus 安全獲取請求狀態
func (rc *RequestContext) GetStatus() RequestStatus {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.Status
}

// SetStatus 安全設置請求狀態
func (rc *RequestContext) SetStatus(status RequestStatus) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.Status = status
}

// RequestMessage 請求消息
type RequestMessage struct {
	Topic      string
	Tag        string
	Key        string
	Body       []byte
	Properties map[string]string
	Timeout    time.Duration // 可選的自定義超時時間
}

// ResponseMessage 響應消息
type ResponseMessage struct {
	Body       []byte
	Properties map[string]string
	Success    bool
	Error      string
}

// RequestResponseClient Request-Response 客戶端
type RequestResponseClient struct {
	client          *Client
	requestMap      sync.Map
	defaultTimeout  time.Duration
	cleanupInterval time.Duration
	stopChan        chan struct{}
	wg              sync.WaitGroup
	replyTopic      string
	clientID        string
}

// RequestResponseConfig Request-Response 配置
type RequestResponseConfig struct {
	DefaultTimeout  time.Duration // 預設超時時間
	CleanupInterval time.Duration // 清理間隔
	ReplyTopic      string        // 回應主題
	ConsumerGroup   string        // 消費者群組
}

// NewRequestResponseClient 創建新的 Request-Response 客戶端
func NewRequestResponseClient(client *Client, config *RequestResponseConfig) (*RequestResponseClient, error) {
	if config == nil {
		config = &RequestResponseConfig{
			DefaultTimeout:  10 * time.Second,
			CleanupInterval: 30 * time.Second,
			ReplyTopic:      "REPLY_TOPIC",
			ConsumerGroup:   "req_resp_group",
		}
	}

	// 設置預設值
	if config.DefaultTimeout == 0 {
		config.DefaultTimeout = 10 * time.Second
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 30 * time.Second
	}
	if config.ReplyTopic == "" {
		config.ReplyTopic = "REPLY_TOPIC"
	}
	if config.ConsumerGroup == "" {
		config.ConsumerGroup = "req_resp_group"
	}

	clientID := uuid.NewString()
	rrc := &RequestResponseClient{
		client:          client,
		defaultTimeout:  config.DefaultTimeout,
		cleanupInterval: config.CleanupInterval,
		stopChan:        make(chan struct{}),
		replyTopic:      config.ReplyTopic,
		clientID:        clientID,
	}

	// 初始化響應消費者
	if err := rrc.initResponseConsumer(config); err != nil {
		return nil, fmt.Errorf("failed to initialize response consumer: %w", err)
	}

	// 啟動清理協程
	rrc.wg.Add(1)
	go rrc.safeCleanup()

	return rrc, nil
}

// initResponseConsumer 初始化響應消費者
func (rrc *RequestResponseClient) initResponseConsumer(config *RequestResponseConfig) error {
	// 創建響應消費者配置
	subscribeConfig := &SubscribeConfig{
		Topic:         rrc.replyTopic,
		Tag:           "*",
		ConsumerGroup: fmt.Sprintf("%s_%s", config.ConsumerGroup, rrc.clientID),
		ConsumeMode:   consumer.Clustering,
	}

	// 創建並啟動響應消費者
	return rrc.client.Subscribe(context.Background(), subscribeConfig, rrc.handleResponse)
}

// Request 發送請求並等待響應
func (rrc *RequestResponseClient) Request(ctx context.Context, req *RequestMessage) (*ResponseMessage, error) {
	// 確定超時時間
	timeout := rrc.defaultTimeout
	if req.Timeout > 0 {
		timeout = req.Timeout
	}

	// 創建帶超時的上下文
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 生成唯一的關聯ID
	correlationID := uuid.NewString()

	// 創建請求上下文
	reqContext := &RequestContext{
		CorrelationID: correlationID,
		RequestTime:   time.Now(),
		Timeout:       timeout,
		ResponseChan:  make(chan *ResponseMessage, 1),
		Status:        RequestPending,
	}

	// 註冊請求上下文
	rrc.requestMap.Store(correlationID, reqContext)

	// 確保清理
	defer func() {
		rrc.requestMap.Delete(correlationID)
		reqContext.SetStatus(RequestFailed)
		// 安全關閉 channel
		select {
		case <-reqContext.ResponseChan:
		default:
			close(reqContext.ResponseChan)
		}
	}()

	// 創建 RocketMQ 消息
	rocketMsg := &primitive.Message{
		Topic: req.Topic,
		Body:  req.Body,
	}

	// 設定 Tag
	if req.Tag != "" {
		rocketMsg.WithTag(req.Tag)
	}

	// 設定 Key
	if req.Key != "" {
		rocketMsg.WithKeys([]string{req.Key})
	}

	// 設定 Properties
	properties := make(map[string]string)
	properties["CORRELATION_ID"] = correlationID
	properties["REPLY_TO"] = rrc.replyTopic
	properties["CLIENT_ID"] = rrc.clientID

	// 添加自定義屬性
	if req.Properties != nil {
		for k, v := range req.Properties {
			properties[k] = v
		}
	}

	// 設定消息屬性
	rocketMsg.WithProperties(properties)

	// 發送請求
	reqContext.SetStatus(RequestProcessing)
	if _, err := rrc.client.producer.SendSync(reqCtx, rocketMsg); err != nil {
		reqContext.SetStatus(RequestFailed)
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// 等待響應
	select {
	case resp := <-reqContext.ResponseChan:
		if resp == nil {
			reqContext.SetStatus(RequestTimeout)
			return nil, ErrRequestTimeout
		}
		reqContext.SetStatus(RequestSuccess)
		return resp, nil
	case <-reqCtx.Done():
		reqContext.SetStatus(RequestTimeout)
		return nil, reqCtx.Err()
	}
}

// handleResponse 處理響應消息
func (rrc *RequestResponseClient) handleResponse(ctx context.Context, msg *ConsumeMessage) error {
	// 從消息的 Properties 中獲取關聯ID
	correlationID := msg.Properties["CORRELATION_ID"]

	// 如果沒有關聯ID，忽略消息
	if correlationID == "" {
		return nil
	}

	// 查找對應的請求上下文
	if value, exists := rrc.requestMap.Load(correlationID); exists {
		reqContext := value.(*RequestContext)

		// 檢查請求狀態
		if reqContext.GetStatus() == RequestTimeout {
			// 請求已超時，忽略響應
			return nil
		}

		// 創建響應消息
		response := &ResponseMessage{
			Body:       msg.Body,
			Properties: msg.Properties,
			Success:    true,
		}

		// 檢查是否有錯誤信息
		if errorMsg, exists := msg.Properties["ERROR"]; exists {
			response.Success = false
			response.Error = errorMsg
		}

		// 發送響應到請求協程
		select {
		case reqContext.ResponseChan <- response:
		default:
			// Channel 已滿或已關閉，忽略
		}
	}

	return nil
}

// safeCleanup 安全清理過期請求
func (rrc *RequestResponseClient) safeCleanup() {
	defer rrc.wg.Done()
	ticker := time.NewTicker(rrc.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rrc.performSafeCleanup()
		case <-rrc.stopChan:
			return
		}
	}
}

// performSafeCleanup 執行安全清理
func (rrc *RequestResponseClient) performSafeCleanup() {
	now := time.Now()
	expiredKeys := make([]interface{}, 0)

	rrc.requestMap.Range(func(key, value interface{}) bool {
		reqContext := value.(*RequestContext)
		status := reqContext.GetStatus()

		// 只清理明確超時的請求或超過超時時間 + 緩衝時間的請求
		if status == RequestTimeout ||
			(status == RequestPending && now.Sub(reqContext.RequestTime) > reqContext.Timeout+5*time.Second) {
			expiredKeys = append(expiredKeys, key)
		}
		return true
	})

	// 安全清理過期請求
	for _, key := range expiredKeys {
		if value, exists := rrc.requestMap.LoadAndDelete(key); exists {
			reqContext := value.(*RequestContext)
			reqContext.SetStatus(RequestTimeout)

			// 安全關閉 channel
			select {
			case <-reqContext.ResponseChan:
			default:
				close(reqContext.ResponseChan)
			}
		}
	}
}

// GetPendingRequestCount 獲取待處理請求數量
func (rrc *RequestResponseClient) GetPendingRequestCount() int {
	count := 0
	rrc.requestMap.Range(func(key, value interface{}) bool {
		reqContext := value.(*RequestContext)
		if reqContext.GetStatus() == RequestPending || reqContext.GetStatus() == RequestProcessing {
			count++
		}
		return true
	})
	return count
}

// Close 關閉 Request-Response 客戶端
func (rrc *RequestResponseClient) Close() error {
	// 停止清理協程
	close(rrc.stopChan)
	rrc.wg.Wait()

	// 清理所有待處理的請求
	rrc.requestMap.Range(func(key, value interface{}) bool {
		reqContext := value.(*RequestContext)
		reqContext.SetStatus(RequestFailed)

		// 安全關閉 channel
		select {
		case <-reqContext.ResponseChan:
		default:
			close(reqContext.ResponseChan)
		}

		rrc.requestMap.Delete(key)
		return true
	})

	return nil
}

// WithTimeout 設置自定義超時時間的請求
func (rrc *RequestResponseClient) WithTimeout(timeout time.Duration) *RequestResponseClient {
	// 創建一個新的客戶端副本，但共享底層資源
	return &RequestResponseClient{
		client:          rrc.client,
		requestMap:      rrc.requestMap,
		defaultTimeout:  timeout,
		cleanupInterval: rrc.cleanupInterval,
		stopChan:        rrc.stopChan,
		replyTopic:      rrc.replyTopic,
		clientID:        rrc.clientID,
	}
}
