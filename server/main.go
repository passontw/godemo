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

func (s *ChatServer) Start() error {
	log.Printf("開始啟動服務端...")

	// 创建生产者 - 啟用生產者
	log.Printf("創建生產者...")
	p, err := rocketmq.NewProducer(
		producer.WithNameServer([]string{s.nameserver}),
		producer.WithGroupName(s.groupName+"_producer"),
	)
	if err != nil {
		return fmt.Errorf("创建生产者失败: %v", err)
	}
	s.producer = p

	// 启动生产者
	log.Printf("啟動生產者...")
	if err := s.producer.Start(); err != nil {
		return fmt.Errorf("启动生产者失败: %v", err)
	}
	log.Printf("生產者啟動成功")

	// 创建消费者
	log.Printf("創建消費者...")
	c, err := rocketmq.NewPushConsumer(
		consumer.WithNameServer([]string{s.nameserver}),
		consumer.WithGroupName(s.groupName+"_consumer"),
	)
	if err != nil {
		return fmt.Errorf("创建消费者失败: %v", err)
	}
	s.consumer = c

	// 订阅请求主题
	log.Printf("訂閱主題: TG001-chat-service-requests")
	if err := s.consumer.Subscribe("TG001-chat-service-requests", consumer.MessageSelector{}, s.handleRequest); err != nil {
		return fmt.Errorf("订阅请求主题失败: %v", err)
	}
	log.Printf("主題訂閱成功")

	// 启动消费者
	log.Printf("啟動消費者...")
	if err := s.consumer.Start(); err != nil {
		return fmt.Errorf("启动消费者失败: %v", err)
	}
	log.Printf("消費者啟動成功")

	log.Printf("聊天服务端已启动，监听请求...")
	return nil
}

func (s *ChatServer) handleRequest(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
	for _, msg := range msgs {
		log.Printf("收到请求消息: %s", string(msg.Body))

		var request ChatRequest
		if err := json.Unmarshal(msg.Body, &request); err != nil {
			log.Printf("解析请求失败: %v", err)
			continue
		}

		// 印出訊息內容
		log.Printf("=== 處理訊息 ===")
		log.Printf("RequestID: %s", request.RequestID)
		log.Printf("UserID: %s", request.UserID)
		log.Printf("Action: %s", request.Action)
		log.Printf("Data: %+v", request.Data)
		log.Printf("=================")

		// 等待一秒
		log.Printf("等待一秒...")
		time.Sleep(1 * time.Second)

		response := s.processRequest(request)

		if err := s.sendResponse(request.RequestID, response); err != nil {
			log.Printf("发送响应失败: %v", err)
		}
	}
	return consumer.ConsumeSuccess, nil
}

func (s *ChatServer) processRequest(request ChatRequest) ChatResponse {
	log.Printf("处理请求: %s, 用户: %s, 动作: %s", request.RequestID, request.UserID, request.Action)

	response := ChatResponse{
		RequestID: request.RequestID,
		Success:   true,
	}

	switch request.Action {
	case "send_message":
		// 提取訊息內容
		var messageData map[string]interface{}
		if data, ok := request.Data.(map[string]interface{}); ok {
			messageData = data
		}

		response.Data = map[string]interface{}{
			"message_id": fmt.Sprintf("msg_%d", time.Now().Unix()),
			"status":     "sent",
			"message":    messageData["message"],
			"user_id":    request.UserID,
		}
		log.Printf("消息已发送: %s", response.Data)

	default:
		response.Success = false
		response.Error = "未知的操作类型"
	}

	return response
}

func (s *ChatServer) sendResponse(requestID string, response ChatResponse) error {
	responseData, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("序列化响应失败: %v", err)
	}

	msg := &primitive.Message{
		Topic: "TG001-websocket-service-responses", // 修正為正確的響應主題
		Body:  responseData,
	}

	msg.WithProperty("request_id", requestID)
	msg.WithProperty("response_type", "chat_response")

	result, err := s.producer.SendSync(context.Background(), msg)
	if err != nil {
		return fmt.Errorf("发送响应失败: %v", err)
	}

	log.Printf("响应已发送: %s", result.String())
	return nil
}

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

	environment := os.Getenv("ROCKETMQ_ENVIRONMENT")
	if environment == "" {
		environment = "k8s"
	}

	nameserver := "10.1.7.229:9876"
	log.Printf("使用 nameserver: %s", nameserver)

	// 使用指定的消費者組名稱
	groupName := "chat-req-group"
	log.Printf("使用 group name: %s", groupName)

	server := NewChatServer(nameserver, groupName)

	if err := server.Start(); err != nil {
		log.Fatalf("启动服务端失败: %v", err)
	}

	log.Printf("服务端启动成功，等待信号...")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Printf("收到停止信号，正在关闭服务端...")
	server.Stop()
}
