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

// 客户端结构
type ChatClient struct {
	producer   rocketmq.Producer
	consumer   rocketmq.PushConsumer
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
	// 创建生产者
	p, err := rocketmq.NewProducer(
		producer.WithNameServer([]string{c.nameserver}),
		producer.WithGroupName(c.groupName+"_producer"),
	)
	if err != nil {
		return fmt.Errorf("创建生产者失败: %v", err)
	}
	c.producer = p

	// 启动生产者
	if err := c.producer.Start(); err != nil {
		return fmt.Errorf("启动生产者失败: %v", err)
	}

	// 创建消费者
	consumerInstance, err := rocketmq.NewPushConsumer(
		consumer.WithNameServer([]string{c.nameserver}),
		consumer.WithGroupName(c.groupName+"_consumer"),
	)
	if err != nil {
		return fmt.Errorf("创建消费者失败: %v", err)
	}
	c.consumer = consumerInstance

	// 订阅响应主题 - 使用空的 MessageSelector
	emptySelector := consumer.MessageSelector{}
	if err := c.consumer.Subscribe("TG001-chat-service-responses", emptySelector, c.handleResponse); err != nil {
		return fmt.Errorf("订阅响应主题失败: %v", err)
	}

	// 订阅事件主题
	if err := c.consumer.Subscribe("TG001-chat-service-events", emptySelector, c.handleEvent); err != nil {
		return fmt.Errorf("订阅事件主题失败: %v", err)
	}

	// 启动消费者
	if err := c.consumer.Start(); err != nil {
		return fmt.Errorf("启动消费者失败: %v", err)
	}

	log.Printf("聊天客户端已启动，用户: %s", c.userID)
	return nil
}

// 处理响应消息 (Request-Response 模式)
func (c *ChatClient) handleResponse(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
	for _, msg := range msgs {
		log.Printf("收到响应消息: %s", string(msg.Body))

		// 解析响应
		var response ChatResponse
		if err := json.Unmarshal(msg.Body, &response); err != nil {
			log.Printf("解析响应失败: %v", err)
			continue
		}

		// 检查是否有对应的请求等待响应
		if ch, exists := c.responses[response.RequestID]; exists {
			ch <- response
			delete(c.responses, response.RequestID)
		} else {
			log.Printf("未找到对应的请求: %s", response.RequestID)
		}
	}
	return consumer.ConsumeSuccess, nil
}

// 处理事件消息 (Publish-Subscribe 模式)
func (c *ChatClient) handleEvent(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
	for _, msg := range msgs {
		log.Printf("收到事件消息: %s", string(msg.Body))

		// 解析事件
		var event ChatMessage
		if err := json.Unmarshal(msg.Body, &event); err != nil {
			log.Printf("解析事件失败: %v", err)
			continue
		}

		// 处理事件
		c.processEvent(event)
	}
	return consumer.ConsumeSuccess, nil
}

// 处理事件
func (c *ChatClient) processEvent(event ChatMessage) {
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

// 发送请求 (Request-Response 模式)
func (c *ChatClient) SendRequest(action string, data interface{}) (*ChatResponse, error) {
	requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())

	request := ChatRequest{
		RequestID: requestID,
		UserID:    c.userID,
		Action:    action,
		Data:      data,
	}

	requestData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %v", err)
	}

	msg := &primitive.Message{
		Topic: "TG001-chat-service-requests",
		Body:  requestData,
	}

	// 设置消息属性
	msg.WithProperty("request_id", requestID)
	msg.WithProperty("user_id", c.userID)
	msg.WithProperty("action", action)

	// 创建响应通道
	responseCh := make(chan ChatResponse, 1)
	c.responses[requestID] = responseCh

	// 发送消息
	result, err := c.producer.SendSync(context.Background(), msg)
	if err != nil {
		delete(c.responses, requestID)
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}

	log.Printf("请求已发送: %s", result.String())

	// 等待响应
	select {
	case response := <-responseCh:
		return &response, nil
	case <-time.After(10 * time.Second):
		delete(c.responses, requestID)
		return nil, fmt.Errorf("等待响应超时")
	}
}

// 发布事件 (Publish-Subscribe 模式)
func (c *ChatClient) PublishEvent(eventType, message string) error {
	event := ChatMessage{
		UserID:    c.userID,
		Message:   message,
		Timestamp: time.Now(),
		Type:      eventType,
	}

	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("序列化事件失败: %v", err)
	}

	msg := &primitive.Message{
		Topic: "TG001-chat-service-events",
		Body:  eventData,
	}

	// 设置消息属性
	msg.WithProperty("event_type", eventType)
	msg.WithProperty("user_id", c.userID)

	// 发送消息
	result, err := c.producer.SendSync(context.Background(), msg)
	if err != nil {
		return fmt.Errorf("发布事件失败: %v", err)
	}

	log.Printf("事件已发布: %s", result.String())
	return nil
}

// 停止客户端
func (c *ChatClient) Stop() {
	if c.producer != nil {
		c.producer.Shutdown()
	}
	if c.consumer != nil {
		c.consumer.Shutdown()
	}
	log.Printf("聊天客户端已停止")
}

func main() {
	// 从环境变量获取配置
	nameserver := os.Getenv("ROCKETMQ_NAMESERVER")
	if nameserver == "" {
		// 使用 RocketMQ nameserver 的对外访问地址
		// 支持多种访问方式：
		// 1. NodePort: <任何節點IP>:30876
		// 2. LoadBalancer: <LoadBalancer IP>:9876
		// 3. 本地端口转发: 127.0.0.1:9876
		nameserver = "127.0.0.1:9876" // 默认使用本地端口转发
	}

	groupName := os.Getenv("ROCKETMQ_GROUP")
	if groupName == "" {
		groupName = "chat_client_group"
	}

	userID := os.Getenv("USER_ID")
	if userID == "" {
		userID = "user_001"
	}

	// 创建客户端
	client := NewChatClient(nameserver, groupName, userID)

	// 启动客户端
	if err := client.Start(); err != nil {
		log.Fatalf("启动客户端失败: %v", err)
	}

	// 模拟客户端操作
	go func() {
		time.Sleep(2 * time.Second)

		// 发布用户加入事件
		log.Printf("发布用户加入事件...")
		if err := client.PublishEvent("user_join", "用户加入聊天"); err != nil {
			log.Printf("发布事件失败: %v", err)
		}

		time.Sleep(2 * time.Second)

		// 发送消息请求
		log.Printf("发送消息请求...")
		response, err := client.SendRequest("send_message", map[string]interface{}{
			"message": "Hello, everyone!",
			"room_id": "room_001",
		})
		if err != nil {
			log.Printf("发送请求失败: %v", err)
		} else {
			log.Printf("收到响应: %+v", response)
		}

		time.Sleep(2 * time.Second)

		// 获取历史记录请求
		log.Printf("获取历史记录请求...")
		response, err = client.SendRequest("get_history", map[string]interface{}{
			"room_id": "room_001",
			"limit":   10,
		})
		if err != nil {
			log.Printf("发送请求失败: %v", err)
		} else {
			log.Printf("收到响应: %+v", response)
		}

		time.Sleep(2 * time.Second)

		// 发布消息事件
		log.Printf("发布消息事件...")
		if err := client.PublishEvent("message_sent", "Hello, everyone!"); err != nil {
			log.Printf("发布事件失败: %v", err)
		}
	}()

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// 优雅关闭
	client.Stop()
}