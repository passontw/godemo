package messagemanager

import (
	"context"
	"fmt"
	"sync"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
)

type ConsumerConfig struct {
	Nameservers       []string
	GroupName         string
	Topic             string
	MessageModel      consumer.MessageModel    // 消息模式: Clustering, Broadcasting
	MessageSelector   consumer.MessageSelector // 消息选择器: 默认全部消费
	ConsumerOrder     bool                     // 是否有序消费
	MaxReconsumeTimes int32                    // 最大重试次数
	Handler           func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error)
}

type SingleConsumer struct {
	topic     string
	groupName string
	mu        sync.RWMutex
	consumer  rocketmq.PushConsumer
}

func NewSingleConsumer(config *ConsumerConfig) (*SingleConsumer, error) {
	consumer, err := rocketmq.NewPushConsumer(
		consumer.WithNameServer(config.Nameservers),
		consumer.WithGroupName(config.GroupName),
		consumer.WithConsumerModel(config.MessageModel),
		consumer.WithConsumerOrder(config.ConsumerOrder),
		consumer.WithMaxReconsumeTimes(config.MaxReconsumeTimes),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %v", err)
	}

	// 立即訂閱 topic
	if err := consumer.Subscribe(config.Topic, config.MessageSelector, config.Handler); err != nil {
		return nil, fmt.Errorf("failed to subscribe topic %s: %v", config.Topic, err)
	}

	return &SingleConsumer{
		topic:     config.Topic,
		groupName: config.GroupName,
		consumer:  consumer,
	}, nil
}

func (c *SingleConsumer) Start() error {
	err := c.consumer.Start()
	if err != nil {
		return fmt.Errorf("failed to start consumer: %v", err)
	}
	return nil
}

func (c *SingleConsumer) Subscribe(topic string, selector consumer.MessageSelector, handler func(context.Context, ...*primitive.MessageExt) (consumer.ConsumeResult, error)) error {
	return c.consumer.Subscribe(topic, selector, handler)
}

func (c *SingleConsumer) Shutdown() error {
	return c.consumer.Shutdown()
}
