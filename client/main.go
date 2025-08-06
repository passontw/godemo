package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	messagemanager "godemo/message_manager"

	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/gorilla/websocket"
)

// 消息结构
// type ChatRequest struct {
// 	RequestID string      `json:"request_id"`
// 	UserID    string      `json:"user_id"`
// 	Action    string      `json:"action"`
// 	Data      interface{} `json:"data"`
// }

type ChatResponse struct {
	RequestID string      `json:"request_id"`
	Success   bool        `json:"success"`
	Data      interface{} `json:"data"`
	Error     string      `json:"error,omitempty"`
}

// WebSocket 消息结构
type WSMessage struct {
	Type    string      `json:"type"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// 客户端结构
type ChatClient struct {
	manager    *messagemanager.MessageManager
	nameserver string
	groupName  string
	userID     string
	responses  map[string]chan ChatResponse
	wsConn     *websocket.Conn
	upgrader   websocket.Upgrader
}

// 创建新的客户端
func NewChatClient(nameserver, groupName, userID string) *ChatClient {
	return &ChatClient{
		nameserver: nameserver,
		groupName:  groupName,
		manager:    nil,
		userID:     userID,
		responses:  make(map[string]chan ChatResponse),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // 允許所有來源
			},
		},
	}
}

// 启动客户端
func (c *ChatClient) Start() error {
	nameservers := []string{c.nameserver}
	consumerConfig := messagemanager.ConsumerConfig{
		Nameservers:  nameservers,
		GroupName:    c.groupName + "_consumer",
		Topic:        "TG001-websocket-service-responses",
		MessageModel: consumer.Clustering,
		MessageSelector: consumer.MessageSelector{
			Type:       consumer.TAG,
			Expression: "*",
		},
		ConsumerOrder:     true,
		MaxReconsumeTimes: 3,
		Handler:           c.handleResponse,
	}

	messageManager := messagemanager.NewMessageManager(
		&messagemanager.ConsumerPoolConfig{
			ConsumerConfigs: []messagemanager.ConsumerConfig{consumerConfig},
			Nameservers:     nameservers,
			Logger:          log.Default(),
		},
		&messagemanager.ProducerPoolConfig{
			Nameservers: nameservers,
			GroupName:   c.groupName + "_producer",
			Prefix:      "reqres",
			PoolSize:    1,
			Logger:      log.Default(),
		},
		nil,
	)

	c.manager = messageManager

	// 设置 WebSocket 路由
	http.HandleFunc("/ws", c.handleWebSocket)

	// 启动 HTTP 服务器
	go func() {
		log.Printf("启动 WebSocket 服务器在 :8081")
		if err := http.ListenAndServe(":8081", nil); err != nil {
			log.Fatalf("HTTP 服务器启动失败: %v", err)
		}
	}()

	log.Printf("聊天客户端已启动，用户: %s", c.userID)
	return nil
}

// 处理响应消息
func (c *ChatClient) handleResponse(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
	for _, msg := range msgs {
		log.Printf("收到响应消息: %s", string(msg.Body))

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

// WebSocket 处理函数
func (c *ChatClient) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := c.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket 升级失败: %v", err)
		return
	}
	defer conn.Close()

	c.wsConn = conn
	log.Printf("WebSocket 连接已建立")

	// 处理 WebSocket 消息
	for {
		var wsMsg WSMessage
		err := conn.ReadJSON(&wsMsg)
		if err != nil {
			log.Printf("读取 WebSocket 消息失败: %v", err)
			break
		}

		log.Printf("收到 WebSocket 消息: %+v", wsMsg)

		// 发送到 RocketMQ
		options := messagemanager.SendRequestOptions{
			SourceService: "websocket-service",
			TargetService: "chat-service",
			Topic:         "TG001-chat-service-requests",
			GroupName:     "chat-req-group_consumer",
		}
		payload := map[string]interface{}{
			"message": wsMsg.Message,
		}
		response, err := c.manager.ReqresProducers.SendRequest(context.Background(), options, payload)
		if err != nil {
			log.Printf("发送请求失败: %v", err)
			// 发送错误响应到前端
			errorMsg := WSMessage{
				Type:    "error",
				Message: "发送失败: " + err.Error(),
			}
			conn.WriteJSON(errorMsg)
			continue
		}

		// 发送成功响应到前端
		successMsg := WSMessage{
			Type:    "response",
			Message: "消息已发送",
			Data:    response,
		}
		conn.WriteJSON(successMsg)
	}
}

// 停止客户端
func (c *ChatClient) Stop() {
	// if c.producer != nil {
	// 	c.producer.Shutdown()
	// }

	c.manager.ShutdownAll()
	if c.wsConn != nil {
		c.wsConn.Close()
	}
	log.Printf("聊天客户端已停止")
}

func main() {
	log.Printf("聊天客户端启动中...")

	environment := os.Getenv("ROCKETMQ_ENVIRONMENT")
	if environment == "" {
		environment = "k8s"
	}

	nameserver := "10.1.7.229:9876"
	log.Printf("使用 nameserver: %s", nameserver)

	// 使用指定的消費者組名稱
	groupName := "websocket-resp-uniqueid"
	log.Printf("使用 group name: %s", groupName)

	userID := "user_001"
	client := NewChatClient(nameserver, groupName, userID)

	if err := client.Start(); err != nil {
		log.Fatalf("启动客户端失败: %v", err)
	}

	log.Printf("聊天客户端已启动，用户: %s", client.userID)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	client.manager.ShutdownAll()
	log.Printf("收到停止信号，正在关闭客户端...")
	time.Sleep(30 * time.Second)
	client.Stop()
}
