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

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
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
	producer   rocketmq.Producer
	consumer   rocketmq.PushConsumer
	nameserver string
	groupName  string
}

// 创建新的服务端
func NewChatServer(nameserver, groupName string) *ChatServer {
	return &ChatServer{
		nameserver: nameserver,
		groupName:  groupName,
	}
}

// 启动服务端
func (s *ChatServer) Start() error {
	// 创建生产者
	p, err := rocketmq.NewProducer(
		producer.WithNameServer([]string{s.nameserver}),
		producer.WithGroupName(s.groupName+"_producer"),
	)
	if err != nil {
		return fmt.Errorf("创建生产者失败: %v", err)
	}
	s.producer = p

	// 启动生产者
	if err := s.producer.Start(); err != nil {
		return fmt.Errorf("启动生产者失败: %v", err)
	}

	// 创建消费者
	c, err := rocketmq.NewPushConsumer(
		consumer.WithNameServer([]string{s.nameserver}),
		consumer.WithGroupName(s.groupName+"_consumer"),
	)
	if err != nil {
		return fmt.Errorf("创建消费者失败: %v", err)
	}
	s.consumer = c

	// 订阅请求主题 - 使用延遲訂閱策略
	go func() {
		time.Sleep(5 * time.Second) // 等待 topics 就緒

		// 嘗試訂閱請求主题
		maxRetries := 5
		for i := 0; i < maxRetries; i++ {
			if err := s.consumer.Subscribe("TG001-chat-service-requests", consumer.MessageSelector{}, s.handleRequest); err != nil {
				log.Printf("订阅请求主题失败 (尝试 %d/%d): %v", i+1, maxRetries, err)
				time.Sleep(time.Duration(i+1) * 2 * time.Second)
				continue
			}
			log.Printf("成功订阅请求主题")
			break
		}

		// 嘗試訂閱事件主题
		for i := 0; i < maxRetries; i++ {
			if err := s.consumer.Subscribe("TG001-chat-service-events", consumer.MessageSelector{}, s.handleEvent); err != nil {
				log.Printf("订阅事件主题失败 (尝试 %d/%d): %v", i+1, maxRetries, err)
				time.Sleep(time.Duration(i+1) * 2 * time.Second)
				continue
			}
			log.Printf("成功订阅事件主题")
			break
		}

		// 启动消费者
		for i := 0; i < maxRetries; i++ {
			if err := s.consumer.Start(); err != nil {
				log.Printf("启动消费者失败 (尝试 %d/%d): %v", i+1, maxRetries, err)
				time.Sleep(time.Duration(i+1) * 2 * time.Second)
				continue
			}
			log.Printf("成功启动消费者")
			break
		}
	}()

	log.Printf("聊天服务端已启动，监听请求和事件...")
	return nil
}

// 处理请求消息 (Request-Response 模式)
func (s *ChatServer) handleRequest(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
	for _, msg := range msgs {
		log.Printf("收到请求消息: %s", string(msg.Body))

		// 解析请求
		var request ChatRequest
		if err := json.Unmarshal(msg.Body, &request); err != nil {
			log.Printf("解析请求失败: %v", err)
			continue
		}

		// 处理请求
		response := s.processRequest(request)

		// 发送响应
		if err := s.sendResponse(request.RequestID, response); err != nil {
			log.Printf("发送响应失败: %v", err)
		}
	}
	return consumer.ConsumeSuccess, nil
}

// 处理事件消息 (Publish-Subscribe 模式)
func (s *ChatServer) handleEvent(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
	for _, msg := range msgs {
		log.Printf("收到事件消息: %s", string(msg.Body))

		// 解析事件
		var event ChatMessage
		if err := json.Unmarshal(msg.Body, &event); err != nil {
			log.Printf("解析事件失败: %v", err)
			continue
		}

		// 处理事件
		s.processEvent(event)
	}
	return consumer.ConsumeSuccess, nil
}

// 处理请求
func (s *ChatServer) processRequest(request ChatRequest) ChatResponse {
	log.Printf("处理请求: %s, 用户: %s, 动作: %s", request.RequestID, request.UserID, request.Action)

	response := ChatResponse{
		RequestID: request.RequestID,
		Success:   true,
	}

	switch request.Action {
	case "send_message":
		// 模拟发送消息
		response.Data = map[string]interface{}{
			"message_id": fmt.Sprintf("msg_%d", time.Now().Unix()),
			"status":     "sent",
		}
		log.Printf("消息已发送: %s", response.Data)

	case "get_history":
		// 模拟获取历史记录
		response.Data = []map[string]interface{}{
			{"message": "Hello", "timestamp": time.Now().Add(-time.Hour)},
			{"message": "How are you?", "timestamp": time.Now().Add(-30 * time.Minute)},
		}
		log.Printf("历史记录已返回: %d 条消息", len(response.Data.([]map[string]interface{})))

	default:
		response.Success = false
		response.Error = "未知的操作类型"
	}

	return response
}

// 处理事件
func (s *ChatServer) processEvent(event ChatMessage) {
	log.Printf("处理事件: 用户 %s 发送消息: %s", event.UserID, event.Message)

	// 模拟事件处理逻辑
	switch event.Type {
	case "user_join":
		log.Printf("用户 %s 加入聊天", event.UserID)
	case "user_leave":
		log.Printf("用户 %s 离开聊天", event.UserID)
	case "message_sent":
		log.Printf("用户 %s 发送消息: %s", event.UserID, event.Message)
	}
}

// 发送响应
func (s *ChatServer) sendResponse(requestID string, response ChatResponse) error {
	responseData, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("序列化响应失败: %v", err)
	}

	msg := &primitive.Message{
		Topic: "TG001-chat-service-responses",
		Body:  responseData,
	}

	// 设置消息属性
	msg.WithProperty("request_id", requestID)
	msg.WithProperty("response_type", "chat_response")

	// 发送消息
	result, err := s.producer.SendSync(context.Background(), msg)
	if err != nil {
		return fmt.Errorf("发送响应失败: %v", err)
	}

	log.Printf("响应已发送: %s", result.String())
	return nil
}

// 发布事件
func (s *ChatServer) PublishEvent(event ChatMessage) error {
	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("序列化事件失败: %v", err)
	}

	msg := &primitive.Message{
		Topic: "TG001-chat-service-events",
		Body:  eventData,
	}

	// 设置消息属性
	msg.WithProperty("event_type", event.Type)
	msg.WithProperty("user_id", event.UserID)

	// 发送消息
	result, err := s.producer.SendSync(context.Background(), msg)
	if err != nil {
		return fmt.Errorf("发布事件失败: %v", err)
	}

	log.Printf("事件已发布: %s", result.String())
	return nil
}

// 停止服务端
func (s *ChatServer) Stop() {
	if s.producer != nil {
		s.producer.Shutdown()
	}
	if s.consumer != nil {
		s.consumer.Shutdown()
	}
	log.Printf("聊天服务端已停止")
}

func main() {
	log.Printf("服务端启动中...")

	// 从环境变量获取配置
	environment := os.Getenv("ROCKETMQ_ENVIRONMENT")
	if environment == "" {
		environment = "k8s" // 默認使用 Kubernetes 環境
	}

	nameserver := os.Getenv("ROCKETMQ_NAMESERVER")
	if nameserver == "" {
		// 根據環境設置默認值
		if environment == "test" {
			nameserver = "localhost:9876" // 測試環境默認值
		} else {
			nameserver = "127.0.0.1:9876" // Kubernetes 環境默認值
		}
	}

	log.Printf("使用環境: %s", environment)
	log.Printf("使用 nameserver: %s", nameserver)

	groupName := os.Getenv("ROCKETMQ_GROUP")
	if groupName == "" {
		groupName = "chat_server_group"
	}

	log.Printf("使用 group name: %s", groupName)

	// 启动健康检查服务
	go func() {
		log.Printf("启动健康检查服务...")
		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			log.Printf("收到健康检查请求")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
		http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
			log.Printf("收到就绪检查请求")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
		log.Printf("健康检查服务启动在端口 8080")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Printf("健康检查服务启动失败: %v", err)
		}
	}()

	// 创建服务端
	log.Printf("创建聊天服务端...")
	server := NewChatServer(nameserver, groupName)

	// 启动服务端
	log.Printf("启动聊天服务端...")
	if err := server.Start(); err != nil {
		log.Fatalf("启动服务端失败: %v", err)
	}

	log.Printf("服务端启动成功，等待信号...")

	// 模拟发布一些事件
	go func() {
		time.Sleep(5 * time.Second)
		log.Printf("发布测试事件...")

		// 发布用户加入事件
		server.PublishEvent(ChatMessage{
			UserID:    "user_001",
			Message:   "用户加入聊天",
			Timestamp: time.Now(),
			Type:      "user_join",
		})

		// 发布消息事件
		server.PublishEvent(ChatMessage{
			UserID:    "user_001",
			Message:   "Hello, everyone!",
			Timestamp: time.Now(),
			Type:      "message_sent",
		})
	}()

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Printf("收到停止信号，正在关闭服务端...")
	// 优雅关闭
	server.Stop()
}
