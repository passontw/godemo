package admincli

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/apache/rocketmq-client-go/v2/admin"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/rlog"
)

// Client 使用 RocketMQ 官方套件的 NameServer 客戶端
type Client struct {
	adminClient admin.Admin
	nameServers []string
	mu          sync.RWMutex
	connected   bool
}

// NewAdminClient 建立新的 NameServer 客戶端
func NewAdminClient(nameServers []string) (*Client, error) {
	if len(nameServers) == 0 {
		return nil, fmt.Errorf("nameServers is empty")
	}

	adminOpts := []admin.AdminOption{
		admin.WithResolver(primitive.NewPassthroughResolver(nameServers)),
	}

	adminClient, err := admin.NewAdmin(adminOpts...)
	if err != nil {
		rlog.Error("Failed to create admin client", map[string]interface{}{
			"error": err,
		})
		return nil, err
	}

	return &Client{
		adminClient: adminClient,
		nameServers: nameServers,
		connected:   true,
	}, nil
}

// Connect 連線到 NameServer
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	// 重新建立 admin client
	adminOpts := []admin.AdminOption{
		admin.WithResolver(primitive.NewPassthroughResolver(c.nameServers)),
	}

	adminClient, err := admin.NewAdmin(adminOpts...)
	if err != nil {
		return fmt.Errorf("failed to create admin client: %w", err)
	}

	c.adminClient = adminClient
	c.connected = true

	rlog.Info("Successfully connected to RocketMQ Client", map[string]interface{}{
		"nameServers": c.nameServers,
	})
	return nil
}

// Disconnect 斷開 NameServer 連線
func (c *Client) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.adminClient == nil {
		return nil
	}

	err := c.adminClient.Close()
	c.connected = false
	rlog.Info("Disconnected from RocketMQ Client", map[string]interface{}{})

	return err
}

// IsConnected 檢查是否已連線
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// CreateTopic 建立 Topic
func (c *Client) CreateTopic(ctx context.Context, topic string, brokerAddr string, queueNum int) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected to nameserver")
	}

	// 使用 admin 套件的 CreateTopic 方法
	err := c.adminClient.CreateTopic(ctx,
		admin.WithTopicCreate(topic),
		admin.WithBrokerAddrCreate(brokerAddr),
		admin.WithReadQueueNums(queueNum),
		admin.WithWriteQueueNums(queueNum),
	)

	if err != nil {
		rlog.Error("create topic error", map[string]interface{}{
			"topic":         topic,
			"brokerAddr":    brokerAddr,
			"underlayError": err,
		})
		return fmt.Errorf("failed to create topic %s: %w", topic, err)
	}

	rlog.Info("Topic created successfully", map[string]interface{}{
		"topic":      topic,
		"brokerAddr": brokerAddr,
		"queueNum":   queueNum,
	})

	return nil
}

// DeleteTopic 刪除 Topic
func (c *Client) DeleteTopic(ctx context.Context, topic string, brokerAddr string) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected to nameserver")
	}

	// 使用 admin 套件的 DeleteTopic 方法
	err := c.adminClient.DeleteTopic(ctx,
		admin.WithTopicDelete(topic),
		admin.WithBrokerAddrDelete(brokerAddr),
	)

	if err != nil {
		rlog.Error("delete topic error", map[string]interface{}{
			"topic":         topic,
			"brokerAddr":    brokerAddr,
			"underlayError": err,
		})
		return fmt.Errorf("failed to delete topic %s: %w", topic, err)
	}

	rlog.Info("Topic deleted successfully", map[string]interface{}{
		"topic":      topic,
		"brokerAddr": brokerAddr,
	})

	return nil
}

// GetTopicList 取得 Topic 列表
func (c *Client) GetTopicList(ctx context.Context) ([]string, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("not connected to nameserver")
	}

	// 使用 admin 套件的 FetchAllTopicList 方法
	result, err := c.adminClient.FetchAllTopicList(ctx)
	if err != nil {
		rlog.Error("get topic list error", map[string]interface{}{
			"underlayError": err,
		})
		return nil, fmt.Errorf("failed to get topic list: %w", err)
	}

	rlog.Info("Get topic list successfully", map[string]interface{}{
		"topicCount": len(result.TopicList),
	})

	return result.TopicList, nil
}

// GetTopicRoute 取得 Topic 路由資訊
func (c *Client) GetTopicRoute(ctx context.Context, topic string) ([]*primitive.MessageQueue, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("not connected to nameserver")
	}

	// 使用 admin 套件的 FetchPublishMessageQueues 方法
	messageQueues, err := c.adminClient.FetchPublishMessageQueues(ctx, topic)
	if err != nil {
		rlog.Error("get topic route error", map[string]interface{}{
			"topic":         topic,
			"underlayError": err,
		})
		return nil, fmt.Errorf("failed to get topic route for %s: %w", topic, err)
	}

	rlog.Info("Get topic route successfully", map[string]interface{}{
		"topic":      topic,
		"queueCount": len(messageQueues),
	})

	return messageQueues, nil
}

// GetConsumerGroupList 取得消費者組列表
func (c *Client) GetConsumerGroupList(ctx context.Context, brokerAddr string) ([]string, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("not connected to nameserver")
	}

	// 使用 admin 套件的 GetAllSubscriptionGroup 方法
	result, err := c.adminClient.GetAllSubscriptionGroup(ctx, brokerAddr, 5*time.Second)
	if err != nil {
		rlog.Error("get consumer group list error", map[string]interface{}{
			"brokerAddr":    brokerAddr,
			"underlayError": err,
		})
		return nil, fmt.Errorf("failed to get consumer group list from broker %s: %w", brokerAddr, err)
	}

	// 提取消費者組名稱
	var groupNames []string
	for groupName := range result.SubscriptionGroupTable {
		groupNames = append(groupNames, groupName)
	}

	rlog.Info("Get consumer group list successfully", map[string]interface{}{
		"brokerAddr": brokerAddr,
		"groupCount": len(groupNames),
	})

	return groupNames, nil
}
