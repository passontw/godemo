package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

// 服务端结构
type ChatServer struct {
	client     *rocketmqclient.Client
	nameserver string
	groupName  string
	serverID   string
}

// 创建新的服务端
func NewChatServer(nameserver, groupName string) *ChatServer {
	// 為每個 server 實體生成唯一的 group name 和 server ID
	uniqueGroupName := fmt.Sprintf("%s_%d", groupName, time.Now().UnixNano())
	serverID := fmt.Sprintf("server_%d", time.Now().UnixNano())

	return &ChatServer{
		nameserver: nameserver,
		groupName:  uniqueGroupName,
		serverID:   serverID,
	}
}

// 启动服务端
func (s *ChatServer) Start() error {
	// 配置 RocketMQ 客戶端
	config := rocketmqclient.RocketMQConfig{
		Name:          s.serverID,
		NameServers:   []string{s.nameserver},
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
	client, err := rocketmqclient.GetClient(s.serverID)
	if err != nil {
		return fmt.Errorf("獲取 RocketMQ 客戶端失敗: %v", err)
	}
	s.client = client

	// 設置日誌
	client.SetLogger(&ServerLogger{})

	// 設置指標回調
	client.SetMetrics(func(event string, labels map[string]string, value float64) {
		log.Printf("指標: %s, 標籤: %v, 數值: %.2f", event, labels, value)
	})

	// 訂閱請求主题 - 使用延遲訂閱策略
	go func() {
		time.Sleep(5 * time.Second) // 等待 topics 就緒

		// 嘗試訂閱請求主题
		maxRetries := 5
		for i := 0; i < maxRetries; i++ {
			subscribeConfig := &rocketmqclient.SubscribeConfig{
				Topic:               "TG001-chat-service-requests",
				Tag:                 "",
				ConsumerGroup:       s.groupName + "_consumer",
				ConsumeFromWhere:    consumer.ConsumeFromLastOffset,
				ConsumeMode:         consumer.Clustering,
				MaxReconsumeTimes:   3,
				MessageBatchMaxSize: 1,
				PullInterval:        time.Second,
				PullBatchSize:       32,
			}

			if err := s.client.Subscribe(context.Background(), subscribeConfig, s.handleRequest); err != nil {
				log.Printf("订阅请求主题失败 (尝试 %d/%d): %v", i+1, maxRetries, err)
				time.Sleep(time.Duration(i+1) * 2 * time.Second)
				continue
			}
			log.Printf("成功订阅请求主题")
			break
		}

		// 嘗試訂閱事件主题
		for i := 0; i < maxRetries; i++ {
			eventSubscribeConfig := &rocketmqclient.SubscribeConfig{
				Topic:               "TG001-chat-service-events",
				Tag:                 "",
				ConsumerGroup:       s.groupName + "_event_consumer",
				ConsumeFromWhere:    consumer.ConsumeFromLastOffset,
				ConsumeMode:         consumer.Clustering,
				MaxReconsumeTimes:   3,
				MessageBatchMaxSize: 1,
				PullInterval:        time.Second,
				PullBatchSize:       32,
			}

			if err := s.client.Subscribe(context.Background(), eventSubscribeConfig, s.handleEvent); err != nil {
				log.Printf("订阅事件主题失败 (尝试 %d/%d): %v", i+1, maxRetries, err)
				time.Sleep(time.Duration(i+1) * 2 * time.Second)
				continue
			}
			log.Printf("成功订阅事件主题")
			break
		}
	}()

	log.Printf("Chat Server 已啟動 (Server ID: %s, Group: %s)", s.serverID, s.groupName)
	return nil
}

// 處理請求
func (s *ChatServer) handleRequest(ctx context.Context, msg *rocketmqclient.ConsumeMessage) error {
	var request ChatRequest
	if err := json.Unmarshal(msg.Body, &request); err != nil {
		log.Printf("解析請求失敗: %v", err)
		return err
	}

	log.Printf("收到請求: %s, 用戶: %s, 動作: %s", request.RequestID, request.UserID, request.Action)

	// 處理請求
	response := s.processRequest(request)

	// 發送響應
	if err := s.sendResponse(request.RequestID, response); err != nil {
		log.Printf("發送響應失敗: %v", err)
		return err
	}

	return nil
}

// 處理事件
func (s *ChatServer) handleEvent(ctx context.Context, msg *rocketmqclient.ConsumeMessage) error {
	var event ChatMessage
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("解析事件失敗: %v", err)
		return err
	}

	log.Printf("收到事件: 用戶: %s, 類型: %s", event.UserID, event.Type)

	// 處理事件
	s.processEvent(event)

	return nil
}

// 處理請求邏輯
func (s *ChatServer) processRequest(request ChatRequest) ChatResponse {
	response := ChatResponse{
		RequestID: request.RequestID,
		Success:   true,
		Data:      nil,
	}

	switch request.Action {
	case "send_message":
		if data, ok := request.Data.(map[string]interface{}); ok {
			if message, exists := data["message"].(string); exists {
				// 模擬處理消息
				response.Data = map[string]interface{}{
					"status":    "sent",
					"message":   message,
					"timestamp": time.Now(),
				}
				log.Printf("處理發送消息請求: %s", message)
			}
		}
	case "get_history":
		// 模擬獲取歷史記錄
		response.Data = map[string]interface{}{
			"messages": []map[string]interface{}{
				{"id": "1", "message": "Hello", "timestamp": time.Now().Add(-time.Hour)},
				{"id": "2", "message": "World", "timestamp": time.Now().Add(-30 * time.Minute)},
			},
		}
		log.Printf("處理獲取歷史記錄請求")
	default:
		response.Success = false
		response.Error = fmt.Sprintf("未知動作: %s", request.Action)
	}

	return response
}

// 處理事件邏輯
func (s *ChatServer) processEvent(event ChatMessage) {
	log.Printf("處理事件: 用戶 %s 的 %s 事件", event.UserID, event.Type)
	// 這裡可以添加事件處理邏輯
}

// 發送響應
func (s *ChatServer) sendResponse(requestID string, response ChatResponse) error {
	responseBody, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("序列化響應失敗: %v", err)
	}

	// 使用 pkg/rocketmq-client 發送響應
	options := map[string]interface{}{
		"properties": map[string]string{
			"request_id": requestID,
			"server_id":  s.serverID,
		},
	}

	key := fmt.Sprintf("response:%s", requestID)
	if err := s.client.PublishPersistent(context.Background(), "TG001-chat-service-responses", "", key, responseBody, options); err != nil {
		return fmt.Errorf("發送響應失敗: %v", err)
	}

	log.Printf("響應已發送: %s", requestID)
	return nil
}

// 發布事件
func (s *ChatServer) PublishEvent(event ChatMessage) error {
	eventBody, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("序列化事件失敗: %v", err)
	}

	// 使用 pkg/rocketmq-client 發布事件
	options := map[string]interface{}{
		"properties": map[string]string{
			"server_id":  s.serverID,
			"event_type": event.Type,
		},
	}

	key := fmt.Sprintf("event:%s:%s", event.UserID, event.Type)
	if err := s.client.PublishPersistent(context.Background(), "TG001-chat-service-events", "", key, eventBody, options); err != nil {
		return fmt.Errorf("發布事件失敗: %v", err)
	}

	log.Printf("事件已發布: 用戶 %s, 類型 %s", event.UserID, event.Type)
	return nil
}

// 停止服務端
func (s *ChatServer) Stop() {
	if s.client != nil {
		// 開始優雅關機
		s.client.StartGracefulShutdown()

		// 等待處理完成
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := s.client.WaitForProcessingComplete(ctx); err != nil {
			log.Printf("等待處理完成時發生錯誤: %v", err)
		}

		// 關閉客戶端
		s.client.Close()
	}
	log.Printf("Chat Server 已停止")
}

// ServerLogger 實作 Logger 介面
type ServerLogger struct{}

func (l *ServerLogger) Infof(format string, args ...interface{}) {
	log.Printf("[SERVER] "+format, args...)
}

func (l *ServerLogger) Errorf(format string, args ...interface{}) {
	log.Printf("[SERVER ERROR] "+format, args...)
}

func main() {
	log.Printf("🚀 啟動 Chat Server...")

	// 配置
	nameserver := "localhost:9876"
	if envNS := os.Getenv("ROCKETMQ_NAMESERVER"); envNS != "" {
		nameserver = envNS
	}

	// 創建服務端
	server := NewChatServer(nameserver, "chat_server_group")

	// 啟動服務端
	if err := server.Start(); err != nil {
		log.Fatalf("啟動服務端失敗: %v", err)
	}

	// 設置 HTTP 服務器（可選）
	go func() {
		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Chat Server is running"))
		})

		port := "3200"
		if envPort := os.Getenv("HTTP_PORT"); envPort != "" {
			port = envPort
		}

		log.Printf("HTTP 服務器啟動在端口 %s", port)
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Printf("HTTP 服務器啟動失敗: %v", err)
		}
	}()

	// 等待中斷信號
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Printf("收到中斷信號，正在關閉...")

	// 優雅關閉
	server.Stop()
	log.Printf("Chat Server 已關閉")
}
