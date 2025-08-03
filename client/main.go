package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	rocketmqclient "godemo/pkg/rocketmq-client"

	"github.com/apache/rocketmq-client-go/v2/consumer"
)

// 消息结构
type ChatMessage struct {
	UserID    string    `json:"user_id"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"` // "request", "response", "event"
}

type ChatRequest struct {
	RequestID string      `json:"request_id"`
	UserID    string      `json:"user_id"`
	Action    string      `json:"action"` // "send_message", "get_history"
	Data      interface{} `json:"data"`
}

type ChatResponse struct {
	RequestID string      `json:"request_id"`
	Success   bool        `json:"success"`
	Data      interface{} `json:"data"`
	Error     string      `json:"error,omitempty"`
}

// 客户端结构
type ChatClient struct {
	client     *rocketmqclient.Client
	nameserver string
	groupName  string
	userID     string
	responses  map[string]chan ChatResponse
}

// 创建新的客户端
func NewChatClient(nameserver, groupName, userID string) *ChatClient {
	return &ChatClient{
		nameserver: nameserver,
		groupName:  groupName,
		userID:     userID,
		responses:  make(map[string]chan ChatResponse),
	}
}

// 启动客户端
func (c *ChatClient) Start() error {
	// 配置 RocketMQ 客戶端
	config := rocketmqclient.RocketMQConfig{
		Name:          fmt.Sprintf("client_%s", c.userID),
		NameServers:   []string{c.nameserver},
		Retry:         rocketmqclient.DefaultRetryConfig(),
		Timeout:       rocketmqclient.DefaultTimeoutConfig(),
		DNS:           rocketmqclient.DefaultDNSConfig(),
		ErrorHandling: rocketmqclient.DefaultErrorHandlingConfig(),
	}

	// 註冊客戶端
	if err := rocketmqclient.Register(context.Background(), config); err != nil {
		return fmt.Errorf("註冊 RocketMQ 客戶端失敗: %v", err)
	}

	// 獲取客戶端實例
	client, err := rocketmqclient.GetClient(fmt.Sprintf("client_%s", c.userID))
	if err != nil {
		return fmt.Errorf("獲取 RocketMQ 客戶端失敗: %v", err)
	}
	c.client = client

	// 設置日誌
	client.SetLogger(&ClientLogger{})

	// 設置指標回調
	client.SetMetrics(func(event string, labels map[string]string, value float64) {
		log.Printf("指標: %s, 標籤: %v, 數值: %.2f", event, labels, value)
	})

	// 訂閱響應主题
	responseSubscribeConfig := &rocketmqclient.SubscribeConfig{
		Topic:               "TG001-chat-service-responses",
		Tag:                 "",
		ConsumerGroup:       c.groupName + "_response_consumer",
		ConsumeFromWhere:    consumer.ConsumeFromLastOffset,
		ConsumeMode:         consumer.Clustering,
		MaxReconsumeTimes:   3,
		MessageBatchMaxSize: 1,
		PullInterval:        time.Second,
		PullBatchSize:       32,
	}

	if err := c.client.Subscribe(context.Background(), responseSubscribeConfig, c.handleResponse); err != nil {
		return fmt.Errorf("訂閱響應主题失敗: %v", err)
	}

	// 訂閱事件主题
	eventSubscribeConfig := &rocketmqclient.SubscribeConfig{
		Topic:               "TG001-chat-service-events",
		Tag:                 "",
		ConsumerGroup:       c.groupName + "_event_consumer",
		ConsumeFromWhere:    consumer.ConsumeFromLastOffset,
		ConsumeMode:         consumer.Clustering,
		MaxReconsumeTimes:   3,
		MessageBatchMaxSize: 1,
		PullInterval:        time.Second,
		PullBatchSize:       32,
	}

	if err := c.client.Subscribe(context.Background(), eventSubscribeConfig, c.handleEvent); err != nil {
		return fmt.Errorf("訂閱事件主题失敗: %v", err)
	}

	log.Printf("Chat Client 已啟動 (User ID: %s, Group: %s)", c.userID, c.groupName)
	return nil
}

// 處理響應
func (c *ChatClient) handleResponse(ctx context.Context, msg *rocketmqclient.ConsumeMessage) error {
	var response ChatResponse
	if err := json.Unmarshal(msg.Body, &response); err != nil {
		log.Printf("解析響應失敗: %v", err)
		return err
	}

	log.Printf("收到響應: %s, 成功: %v", response.RequestID, response.Success)

	// 檢查是否有對應的請求等待響應
	if responseChan, exists := c.responses[response.RequestID]; exists {
		select {
		case responseChan <- response:
			log.Printf("響應已發送到等待通道: %s", response.RequestID)
		default:
			log.Printf("響應通道已滿，丟棄響應: %s", response.RequestID)
		}
		delete(c.responses, response.RequestID)
	} else {
		log.Printf("未找到對應的請求等待響應: %s", response.RequestID)
	}

	return nil
}

// 處理事件
func (c *ChatClient) handleEvent(ctx context.Context, msg *rocketmqclient.ConsumeMessage) error {
	var event ChatMessage
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("解析事件失敗: %v", err)
		return err
	}

	log.Printf("收到事件: 用戶: %s, 類型: %s, 消息: %s", event.UserID, event.Type, event.Message)

	// 處理事件
	c.processEvent(event)

	return nil
}

// 處理事件邏輯
func (c *ChatClient) processEvent(event ChatMessage) {
	log.Printf("處理事件: 用戶 %s 的 %s 事件", event.UserID, event.Type)
	// 這裡可以添加事件處理邏輯
}

// 發送請求
func (c *ChatClient) SendRequest(action string, data interface{}) (*ChatResponse, error) {
	requestID := fmt.Sprintf("req_%s_%d", c.userID, time.Now().UnixNano())

	request := ChatRequest{
		RequestID: requestID,
		UserID:    c.userID,
		Action:    action,
		Data:      data,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("序列化請求失敗: %v", err)
	}

	// 創建響應通道
	responseChan := make(chan ChatResponse, 1)
	c.responses[requestID] = responseChan

	// 清理函數
	defer func() {
		delete(c.responses, requestID)
		close(responseChan)
	}()

	// 使用 pkg/rocketmq-client 發送請求
	options := map[string]interface{}{
		"properties": map[string]string{
			"user_id": c.userID,
			"action":  action,
		},
	}

	key := fmt.Sprintf("request:%s:%s", c.userID, action)
	if err := c.client.PublishPersistent(context.Background(), "TG001-chat-service-requests", "", key, requestBody, options); err != nil {
		return nil, fmt.Errorf("發送請求失敗: %v", err)
	}

	log.Printf("請求已發送: %s, 動作: %s", requestID, action)

	// 等待響應
	select {
	case response := <-responseChan:
		return &response, nil
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("請求超時: %s", requestID)
	}
}

// 發布事件
func (c *ChatClient) PublishEvent(eventType, message string) error {
	event := ChatMessage{
		UserID:    c.userID,
		Message:   message,
		Timestamp: time.Now(),
		Type:      eventType,
	}

	eventBody, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("序列化事件失敗: %v", err)
	}

	// 使用 pkg/rocketmq-client 發布事件
	options := map[string]interface{}{
		"properties": map[string]string{
			"user_id":    c.userID,
			"event_type": eventType,
		},
	}

	key := fmt.Sprintf("event:%s:%s", c.userID, eventType)
	if err := c.client.PublishPersistent(context.Background(), "TG001-chat-service-events", "", key, eventBody, options); err != nil {
		return fmt.Errorf("發布事件失敗: %v", err)
	}

	log.Printf("事件已發布: 用戶 %s, 類型 %s, 消息 %s", c.userID, eventType, message)
	return nil
}

// 停止客戶端
func (c *ChatClient) Stop() {
	if c.client != nil {
		// 開始優雅關機
		c.client.StartGracefulShutdown()

		// 等待處理完成
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := c.client.WaitForProcessingComplete(ctx); err != nil {
			log.Printf("等待處理完成時發生錯誤: %v", err)
		}

		// 關閉客戶端
		c.client.Close()
	}
	log.Printf("Chat Client 已停止")
}

// ClientLogger 實作 Logger 介面
type ClientLogger struct{}

func (l *ClientLogger) Infof(format string, args ...interface{}) {
	log.Printf("[CLIENT] "+format, args...)
}

func (l *ClientLogger) Errorf(format string, args ...interface{}) {
	log.Printf("[CLIENT ERROR] "+format, args...)
}

func main() {
	log.Printf("🚀 啟動 Chat Client...")

	// 配置
	nameserver := "localhost:9876"
	if envNS := os.Getenv("ROCKETMQ_NAMESERVER"); envNS != "" {
		nameserver = envNS
	}

	userID := "user_001"
	if envUserID := os.Getenv("USER_ID"); envUserID != "" {
		userID = envUserID
	}

	groupName := "chat_client_group"
	if envGroup := os.Getenv("ROCKETMQ_GROUP"); envGroup != "" {
		groupName = envGroup
	}

	// 創建客戶端
	client := NewChatClient(nameserver, groupName, userID)

	// 啟動客戶端
	if err := client.Start(); err != nil {
		log.Fatalf("啟動客戶端失敗: %v", err)
	}

	// 模擬用戶操作
	go func() {
		time.Sleep(3 * time.Second) // 等待客戶端完全啟動

		log.Printf("開始模擬用戶操作...")

		// 發送消息請求
		log.Printf("發送消息請求...")
		response, err := client.SendRequest("send_message", map[string]interface{}{
			"message": "Hello, everyone!",
		})
		if err != nil {
			log.Printf("發送消息請求失敗: %v", err)
		} else {
			log.Printf("消息發送響應: %+v", response)
		}

		// 獲取歷史記錄請求
		log.Printf("獲取歷史記錄請求...")
		historyResponse, err := client.SendRequest("get_history", nil)
		if err != nil {
			log.Printf("獲取歷史記錄請求失敗: %v", err)
		} else {
			log.Printf("歷史記錄響應: %+v", historyResponse)
		}

		// 發布用戶加入事件
		log.Printf("發布用戶加入事件...")
		if err := client.PublishEvent("user_join", "用戶加入聊天"); err != nil {
			log.Printf("發布用戶加入事件失敗: %v", err)
		}

		// 發布消息事件
		log.Printf("發布消息事件...")
		if err := client.PublishEvent("message_sent", "Hello, everyone!"); err != nil {
			log.Printf("發布消息事件失敗: %v", err)
		}

		log.Printf("用戶操作完成")
	}()

	// 等待中斷信號
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Printf("收到中斷信號，正在關閉...")

	// 優雅關閉
	client.Stop()
	log.Printf("Chat Client 已關閉")
}
