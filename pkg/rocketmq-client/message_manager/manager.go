package message_manager

import (
	"context"
	"fmt"
	"log"

	rocketmqclient "godemo/pkg/rocketmq-client"
	"github.com/teris-io/shortid"
)

// MessageManager 管理 RocketMQ 連線和實例 ID
type MessageManager struct {
	client     *rocketmqclient.Client
	instanceID string
	nameserver string
	clientName string
}

// Config 配置結構
type Config struct {
	NameServer string
	ClientName string
}

// NewMessageManager 建立新的 MessageManager
func NewMessageManager(config Config) (*MessageManager, error) {
	// 生成唯一實例 ID
	sid, err := shortid.New(1, shortid.DefaultABC, 2342)
	if err != nil {
		return nil, fmt.Errorf("建立 shortid 生成器失敗: %v", err)
	}
	
	instanceID, err := sid.Generate()
	if err != nil {
		return nil, fmt.Errorf("生成實例 ID 失敗: %v", err)
	}

	return &MessageManager{
		instanceID: instanceID,
		nameserver: config.NameServer,
		clientName: fmt.Sprintf("%s_%s", config.ClientName, instanceID),
	}, nil
}

// Initialize 初始化並連線到 RocketMQ
func (m *MessageManager) Initialize() error {
	// 配置 RocketMQ 客戶端
	config := rocketmqclient.RocketMQConfig{
		Name:          m.clientName,
		NameServers:   []string{m.nameserver},
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
	client, err := rocketmqclient.GetClient(m.clientName)
	if err != nil {
		return fmt.Errorf("獲取 RocketMQ 客戶端失敗: %v", err)
	}
	
	m.client = client

	// 連線成功 log
	log.Printf("🎉 MessageManager 連線 RocketMQ 成功！實例 ID: %s, 客戶端名稱: %s, NameServer: %s", 
		m.instanceID, m.clientName, m.nameserver)

	return nil
}

// GetInstanceID 取得實例 ID
func (m *MessageManager) GetInstanceID() string {
	return m.instanceID
}

// GetClient 取得 RocketMQ 客戶端
func (m *MessageManager) GetClient() *rocketmqclient.Client {
	return m.client
}

// GetUniqueGroupName 產生唯一的 Consumer Group 名稱
func (m *MessageManager) GetUniqueGroupName(baseGroupName string) string {
	return fmt.Sprintf("%s_%s", baseGroupName, m.instanceID)
}

// GenerateRequestID 產生唯一的請求 ID
func (m *MessageManager) GenerateRequestID(userID string) string {
	return fmt.Sprintf("req_%s_%s", userID, m.instanceID)
}

// Close 關閉連線
func (m *MessageManager) Close() {
	if m.client != nil {
		// 開始優雅關機
		m.client.StartGracefulShutdown()

		// 關閉客戶端
		m.client.Close()
		log.Printf("MessageManager 已關閉 (實例 ID: %s)", m.instanceID)
	}
}