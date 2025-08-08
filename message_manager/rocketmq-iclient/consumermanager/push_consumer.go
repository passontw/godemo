package consumermanager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/rlog"
)

// PushConsumer Push 消費者管理器
type PushConsumer struct {
	topic    string
	config   *ConsumerConfig
	consumer rocketmq.PushConsumer
	mu       sync.RWMutex
	started  bool
}

// NewPushConsumerManager 建立 Push 消費者
func NewPushConsumerManager(cfg ConsumerConfig) (IPushConsumer, error) {

	if cfg.NameServers == nil {
		return nil, fmt.Errorf("name servers is required")
	}

	if cfg.GroupName == "" {
		return nil, fmt.Errorf("consumer group name is required")
	}

	// if cfg.Topic == "" {
	// 	return nil, fmt.Errorf("topic is required")
	// }

	if cfg.MessageModel < consumer.BroadCasting || cfg.MessageModel > consumer.Clustering {
		return nil, fmt.Errorf("message model is required")
	}

	switch cfg.ConsumerType {
	case ConsumerType_PushConsumer:
		cfg.ConsumerType = consumer.ConsumeType("CONSUME_PASSIVELY")
	case ConsumerType_PullConsumer:
		cfg.ConsumerType = consumer.ConsumeType("CONSUME_ACTIVELY")
	default:
		return nil, fmt.Errorf("consumer type is required")
	}

	if cfg.ConsumeFromWhere < consumer.ConsumeFromLastOffset || cfg.ConsumeFromWhere > consumer.ConsumeFromTimestamp {
		return nil, fmt.Errorf("consume from where is required")
	}

	if cfg.ConsumeTimeout == 0 {
		cfg.ConsumeTimeout = 5 * time.Second
		rlog.Info("consume timeout is required, use default value", map[string]interface{}{
			"consumeTimeout": cfg.ConsumeTimeout,
		})
	}

	if cfg.MaxReconsumeTimes == 0 {
		cfg.MaxReconsumeTimes = 3
		rlog.Info("max reconsume times is required, use default value", map[string]interface{}{
			"maxReconsumeTimes": cfg.MaxReconsumeTimes,
		})
	}

	consumerManager := &PushConsumer{
		config: &cfg,
	}

	// 建立 Push Consumer
	opts := []consumer.Option{
		consumer.WithNameServer(cfg.NameServers),
		consumer.WithGroupName(cfg.GroupName),
		consumer.WithConsumerModel(cfg.MessageModel),
		consumer.WithConsumeFromWhere(cfg.ConsumeFromWhere),
		consumer.WithConsumeTimeout(cfg.ConsumeTimeout),
		consumer.WithConsumerOrder(cfg.ConsumeOrderly),
		consumer.WithMaxReconsumeTimes(cfg.MaxReconsumeTimes),
	}

	newConsumer, err := rocketmq.NewPushConsumer(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create push consumer: %w", err)
	}

	consumerManager.consumer = newConsumer
	return consumerManager, nil
}

func (c *PushConsumer) GetTopic() string {
	return c.topic
}

// Start 啟動 Push 消費者
func (c *PushConsumer) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return nil
	}

	// 啟動 Push Consumer
	err := c.consumer.Start()
	if err != nil {
		return fmt.Errorf("failed to start push consumer: %w", err)
	}

	c.started = true

	rlog.Info("Push consumer started successfully", map[string]interface{}{
		"groupName":   c.config.GroupName,
		"nameServers": c.config.NameServers,
	})

	return nil
}

// GetConsumerStats 取得消費者資訊
func (c *PushConsumer) GetConsumerStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"started": c.started,
		"config":  c.config,
	}
}

// Shutdown 關閉 Push 消費者
func (c *PushConsumer) Shutdown() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started || c.consumer == nil {
		return nil
	}

	err := c.consumer.Shutdown()
	if err != nil {
		return fmt.Errorf("failed to shutdown push consumer: %w", err)
	}

	c.started = false
	rlog.Info("Push consumer shutdown successfully", map[string]interface{}{})

	return nil
}

// IsStarted 檢查是否已啟動
func (c *PushConsumer) IsStarted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.started
}

// SubscribeChan 註冊訂閱到 Push Consumer（使用通道）
func (c *PushConsumer) SubscribeChan(subCfg SubscriptionInfo, callback chan []*primitive.MessageExt) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return fmt.Errorf("cannot subscribe after consumer is started")
	}

	if c.topic != "" {
		rlog.Info("topic is updated", map[string]interface{}{})
		if err := c.consumer.Unsubscribe(c.topic); err != nil {
			return fmt.Errorf("failed to unsubscribe topic %s: %w", c.topic, err)
		}
		c.topic = subCfg.Topic
	}

	return c.consumer.Subscribe(subCfg.Topic, subCfg.Selector, func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
		callback <- msgs
		return consumer.ConsumeSuccess, nil
	})
}

// SubscribeFunc 註冊訂閱到 Push Consumer（使用函數）
func (c *PushConsumer) SubscribeFunc(subCfg *SubscriptionInfo, callback func(context.Context, ...*primitive.MessageExt) (consumer.ConsumeResult, error)) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return fmt.Errorf("cannot subscribe after consumer is started")
	}

	if c.topic != subCfg.Topic {
		rlog.Info("topic is updated", map[string]interface{}{})
		if err := c.consumer.Unsubscribe(c.topic); err != nil {
			return fmt.Errorf("failed to unsubscribe topic %s: %w", c.topic, err)
		}
		c.topic = subCfg.Topic
	}

	return c.consumer.Subscribe(subCfg.Topic, subCfg.Selector, callback)
}

// Unsubscribe 取消訂閱主題
func (c *PushConsumer) Unsubscribe(topic string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return fmt.Errorf("cannot unsubscribe after consumer is started")
	}

	if err := c.consumer.Unsubscribe(topic); err != nil {
		return fmt.Errorf("failed to unsubscribe topic %s: %w", topic, err)
	}

	return nil
}
