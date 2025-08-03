package rocketmqclient

import (
	"context"
	"fmt"
	"time"

	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/rlog"
)

// ConsumeMessage 表示接收到的訊息
type ConsumeMessage struct {
	MessageID      string
	Topic          string
	Tag            string
	Key            string
	Body           []byte
	Properties     map[string]string // 添加 Properties 字段
	QueueID        int32
	QueueOffset    int64
	ReconsumeTimes int32
	BornHost       string
	BornTime       time.Time
	StoreHost      string
	StoreTime      time.Time
}

// MessageHandler 消息處理函數
type MessageHandler func(ctx context.Context, msg *ConsumeMessage) error

// SubscribeConfig 訂閱配置
type SubscribeConfig struct {
	Topic               string
	Tag                 string
	ConsumerGroup       string
	ConsumeFromWhere    consumer.ConsumeFromWhere
	ConsumeMode         consumer.MessageModel
	MaxReconsumeTimes   int32
	MessageBatchMaxSize int
	PullInterval        time.Duration
	PullBatchSize       int32
}

// Subscribe 訂閱主題（Push 模式）
func (c *Client) Subscribe(ctx context.Context, config *SubscribeConfig, handler MessageHandler) error {
	if c.logger != nil {
		c.logger.Infof("RocketMQ subscribe: topic=%s, tag=%s, group=%s", config.Topic, config.Tag, config.ConsumerGroup)
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		if c.metrics != nil {
			c.metrics("subscribe", map[string]string{
				"topic": config.Topic,
				"tag":   config.Tag,
				"group": config.ConsumerGroup,
			}, duration)
		}
	}()

	// 建立消費者選項
	consumerOpts := []consumer.Option{
		consumer.WithNameServer(c.resolvedNameServers),
		consumer.WithConsumerModel(config.ConsumeMode),
		consumer.WithGroupName(config.ConsumerGroup),
	}

	// 認證配置
	if c.config.AccessKey != "" && c.config.SecretKey != "" {
		credentials := primitive.Credentials{
			AccessKey: c.config.AccessKey,
			SecretKey: c.config.SecretKey,
		}
		if c.config.SecurityToken != "" {
			credentials.SecurityToken = c.config.SecurityToken
		}
		consumerOpts = append(consumerOpts, consumer.WithCredentials(credentials))
	}

	// 命名空間
	if c.config.Namespace != "" {
		consumerOpts = append(consumerOpts, consumer.WithNamespace(c.config.Namespace))
	}

	// 消費位置
	if config.ConsumeFromWhere != 0 {
		consumerOpts = append(consumerOpts, consumer.WithConsumeFromWhere(config.ConsumeFromWhere))
	}

	// 最大重試次數
	if config.MaxReconsumeTimes > 0 {
		consumerOpts = append(consumerOpts, consumer.WithMaxReconsumeTimes(config.MaxReconsumeTimes))
	}

	// 批次大小
	if config.MessageBatchMaxSize > 0 {
		consumerOpts = append(consumerOpts, consumer.WithConsumeMessageBatchMaxSize(config.MessageBatchMaxSize))
	}

	// 建立新的消費者
	cons, err := consumer.NewPushConsumer(consumerOpts...)
	if err != nil {
		return err
	}

	// 包裝 handler 來追蹤訊息處理
	wrappedHandler := func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
		// 檢查是否正在關機
		if c.isShuttingDown() {
			if c.logger != nil {
				c.logger.Infof("Rejecting messages during shutdown: topic=%s", config.Topic)
			}
			// 在關機過程中，拒絕新訊息
			return consumer.ConsumeRetryLater, nil
		}

		// 增加處理計數器
		if !c.incrementProcessing() {
			if c.logger != nil {
				c.logger.Infof("Rejecting messages during shutdown: topic=%s", config.Topic)
			}
			return consumer.ConsumeRetryLater, nil
		}

		// 在處理完成後減少計數器
		defer c.decrementProcessing()

		// 處理每個訊息
		for _, msg := range msgs {
			consumeMsg := &ConsumeMessage{
				MessageID:      msg.MsgId,
				Topic:          msg.Topic,
				Tag:            msg.GetTags(),
				Key:            msg.GetKeys(),
				Body:           msg.Body,
				Properties:     msg.GetProperties(), // 使用 GetProperties() 方法
				QueueID:        int32(msg.Queue.QueueId),
				QueueOffset:    msg.QueueOffset,
				ReconsumeTimes: msg.ReconsumeTimes,
				BornHost:       msg.BornHost,
				BornTime:       time.Unix(msg.BornTimestamp/1000, 0),
				StoreHost:      msg.StoreHost,
				StoreTime:      time.Unix(msg.StoreTimestamp/1000, 0),
			}

			// 調用用戶處理函數
			if err := handler(ctx, consumeMsg); err != nil {
				if c.logger != nil {
					c.logger.Errorf("Message handler error: %v", err)
				}
				return consumer.ConsumeRetryLater, err
			}
		}

		return consumer.ConsumeSuccess, nil
	}

	// 訂閱主題
	selector := consumer.MessageSelector{
		Type:       consumer.TAG,
		Expression: config.Tag,
	}

	err = cons.Subscribe(config.Topic, selector, wrappedHandler)
	if err != nil {
		return err
	}

	// 啟動消費者
	if err := cons.Start(); err != nil {
		return err
	}

	// 更新客戶端的消費者實例
	c.consumer = cons
	c.consumerStarted = true // 標記 consumer 已成功啟動

	return nil
}

// PullSubscribe Pull 模式訂閱 (暫時不實作，RocketMQ Pull 模式 API 較複雜)
func (c *Client) PullSubscribe(ctx context.Context, config *SubscribeConfig) error {
	if c.logger != nil {
		c.logger.Infof("RocketMQ pull subscribe: topic=%s, tag=%s, group=%s", config.Topic, config.Tag, config.ConsumerGroup)
	}

	// Pull 模式實作較複雜，暫時保留介面
	return fmt.Errorf("pull subscribe not implemented yet")
}

// PullMessage 手動拉取訊息 (暫時不實作)
func (c *Client) PullMessage(ctx context.Context, topic, tag string, queueID int, offset int64, maxNums int32) ([]*ConsumeMessage, error) {
	if c.logger != nil {
		c.logger.Infof("RocketMQ pull message: topic=%s, tag=%s, queue=%d, offset=%d", topic, tag, queueID, offset)
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		if c.metrics != nil {
			c.metrics("pull_message", map[string]string{
				"topic": topic,
				"tag":   tag,
			}, duration)
		}
	}()

	// Pull 模式暫時不實作，回傳錯誤
	return nil, fmt.Errorf("pull message not implemented yet")
}

// Unsubscribe 取消訂閱
func (c *Client) Unsubscribe(topic string) error {
	if c.consumer == nil {
		return fmt.Errorf("consumer not initialized")
	}

	if c.logger != nil {
		c.logger.Infof("RocketMQ unsubscribe: topic=%s", topic)
	}

	return c.consumer.Unsubscribe(topic)
}

// ShutdownConsumer 關閉消費者
func (c *Client) ShutdownConsumer() error {
	if c.consumer == nil {
		return nil
	}

	if c.logger != nil {
		c.logger.Infof("RocketMQ shutdown consumer")
	}

	return c.consumer.Shutdown()
}

// SetRocketMQLogLevel 設定 RocketMQ 日誌級別
func SetRocketMQLogLevel(level string) {
	switch level {
	case "debug":
		rlog.SetLogLevel("debug")
	case "info":
		rlog.SetLogLevel("info")
	case "warn":
		rlog.SetLogLevel("warn")
	case "error":
		rlog.SetLogLevel("error")
	default:
		rlog.SetLogLevel("info")
	}
}
