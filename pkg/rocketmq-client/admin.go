package rocketmqclient

import (
	"context"
	"fmt"
	"time"

	"github.com/apache/rocketmq-client-go/v2/admin"
	"github.com/apache/rocketmq-client-go/v2/primitive"
)

// TopicConfig 主題配置
type TopicConfig struct {
	TopicName      string
	ReadQueueNums  int
	WriteQueueNums int
	Permission     int
	TopicSysFlag   int
	Order          bool
}

// TopicInfo 主題資訊
type TopicInfo struct {
	TopicName      string
	ReadQueueNums  int
	WriteQueueNums int
	Permission     int
	TopicSysFlag   int
	Order          bool
	CreateTime     time.Time
	UpdateTime     time.Time
}

// AdminClient 管理客戶端
type AdminClient struct {
	client *Client
	admin  admin.Admin
}

// NewAdminClient 建立管理客戶端
func (c *Client) NewAdminClient() (*AdminClient, error) {
	// 解析 NameServers
	var resolvedNameServers []string
	var err error

	if c.dnsResolver != nil {
		resolvedNameServers, err = c.dnsResolver.ResolveNameServers(c.config.NameServers)
		if err != nil {
			return nil, fmt.Errorf("DNS 解析失敗: %v", err)
		}
	} else {
		resolvedNameServers = c.config.NameServers
	}

	// 建立管理客戶端
	adminOpts := []admin.AdminOption{
		admin.WithResolver(primitive.NewPassthroughResolver(resolvedNameServers)),
	}

	// 認證配置
	if c.config.AccessKey != "" && c.config.SecretKey != "" {
		credentials := &primitive.Credentials{
			AccessKey: c.config.AccessKey,
			SecretKey: c.config.SecretKey,
		}
		if c.config.SecurityToken != "" {
			credentials.SecurityToken = c.config.SecurityToken
		}
		adminOpts = append(adminOpts, admin.WithCredentials(*credentials))
	}

	adminClient, err := admin.NewAdmin(adminOpts...)
	if err != nil {
		return nil, err
	}

	return &AdminClient{
		client: c,
		admin:  adminClient,
	}, nil
}

// CreateTopic 建立主題（暫時不實作）
func (ac *AdminClient) CreateTopic(ctx context.Context, config *TopicConfig) error {
	if ac.client.logger != nil {
		ac.client.logger.Infof("RocketMQ create topic: %s", config.TopicName)
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		if ac.client.metrics != nil {
			ac.client.metrics("create_topic", map[string]string{
				"topic": config.TopicName,
			}, duration)
		}
	}()

	// 暫時不實作，RocketMQ Topic 通常在 Broker 端預先建立
	return fmt.Errorf("create topic not implemented yet, topics should be created on broker side")
}

// DeleteTopic 刪除主題
func (ac *AdminClient) DeleteTopic(ctx context.Context, topicName string) error {
	if ac.client.logger != nil {
		ac.client.logger.Infof("RocketMQ delete topic: %s", topicName)
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		if ac.client.metrics != nil {
			ac.client.metrics("delete_topic", map[string]string{
				"topic": topicName,
			}, duration)
		}
	}()

	return ac.admin.DeleteTopic(ctx, admin.WithTopicDelete(topicName))
}

// GetTopicList 取得主題列表
func (ac *AdminClient) GetTopicList(ctx context.Context) ([]string, error) {
	if ac.client.logger != nil {
		ac.client.logger.Infof("RocketMQ get topic list")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		if ac.client.metrics != nil {
			ac.client.metrics("get_topic_list", nil, duration)
		}
	}()

	result, err := ac.admin.FetchAllTopicList(ctx)
	if err != nil {
		return nil, err
	}

	return result.TopicList, nil
}

// CreateConsumerGroup 建立消費者組（RocketMQ 動態建立）
func (ac *AdminClient) CreateConsumerGroup(ctx context.Context, groupName string, config map[string]interface{}) error {
	if ac.client.logger != nil {
		ac.client.logger.Infof("RocketMQ create consumer group: %s", groupName)
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		if ac.client.metrics != nil {
			ac.client.metrics("create_consumer_group", map[string]string{
				"group": groupName,
			}, duration)
		}
	}()

	// RocketMQ 的消費者組是動態建立的，不需要事先建立
	// 這裡僅做紀錄，實際功能由客戶端自動處理
	return nil
}

// Close 關閉管理客戶端
func (ac *AdminClient) Close() error {
	if ac.client.logger != nil {
		ac.client.logger.Infof("RocketMQ close admin client")
	}

	return ac.admin.Close()
}

// UpdateTopic 更新主題配置
func (ac *AdminClient) UpdateTopic(ctx context.Context, config *TopicConfig) error {
	if ac.client.logger != nil {
		ac.client.logger.Infof("RocketMQ update topic: %s", config.TopicName)
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		if ac.client.metrics != nil {
			ac.client.metrics("update_topic", map[string]string{
				"topic": config.TopicName,
			}, duration)
		}
	}()

	// 暫時不實作，RocketMQ Topic 通常在 Broker 端預先建立
	return fmt.Errorf("update topic not implemented yet, topics should be updated on broker side")
}

// GetTopicStats 取得主題統計資訊（暫時不實作）
func (ac *AdminClient) GetTopicStats(ctx context.Context, topicName string) (map[string]interface{}, error) {
	if ac.client.logger != nil {
		ac.client.logger.Infof("RocketMQ get topic stats: %s", topicName)
	}

	// 暫時不實作，返回空統計
	return map[string]interface{}{
		"topic":   topicName,
		"message": "topic stats not implemented yet",
	}, nil
}

// GetConsumerGroupList 取得消費者組列表（暫時不實作）
func (ac *AdminClient) GetConsumerGroupList(ctx context.Context) ([]string, error) {
	if ac.client.logger != nil {
		ac.client.logger.Infof("RocketMQ get consumer group list")
	}

	// 暫時不實作，返回空列表
	return []string{}, nil
}

// GetConsumerProgress 取得消費者進度（暫時不實作）
func (ac *AdminClient) GetConsumerProgress(ctx context.Context, groupName string) (map[string]interface{}, error) {
	if ac.client.logger != nil {
		ac.client.logger.Infof("RocketMQ get consumer progress: %s", groupName)
	}

	// 暫時不實作，返回空進度
	return map[string]interface{}{
		"group":   groupName,
		"message": "consumer progress not implemented yet",
	}, nil
}

// ResetConsumerOffset 重設消費者偏移量（暫時不實作）
func (ac *AdminClient) ResetConsumerOffset(ctx context.Context, groupName, topicName string, timestamp int64) error {
	if ac.client.logger != nil {
		ac.client.logger.Infof("RocketMQ reset consumer offset: group=%s, topic=%s, timestamp=%d", groupName, topicName, timestamp)
	}

	// 暫時不實作
	return fmt.Errorf("reset consumer offset not implemented yet")
}

// GetClusterInfo 取得叢集資訊（暫時不實作）
func (ac *AdminClient) GetClusterInfo(ctx context.Context) (map[string]interface{}, error) {
	if ac.client.logger != nil {
		ac.client.logger.Infof("RocketMQ get cluster info")
	}

	// 暫時不實作，返回空資訊
	return map[string]interface{}{
		"message": "cluster info not implemented yet",
	}, nil
}
