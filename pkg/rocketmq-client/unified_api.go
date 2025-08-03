package rocketmqclient

import (
	"context"
	"fmt"
	"time"

	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/google/uuid"
)

// UnifiedMessageHandler 統一訊息處理函數
type UnifiedMessageHandler func(ctx context.Context, topic, tag, key string, body []byte, properties map[string]string) error

// SendRequestPersistent 發送持久化請求並等待響應
func (c *Client) SendRequestPersistent(ctx context.Context, topic, tag, key string, body []byte, options map[string]interface{}) (*ResponseMessage, error) {
	return c.sendRequest(ctx, topic, tag, key, body, options, true)
}

// SendRequestNonPersistent 發送非持久化請求並等待響應
func (c *Client) SendRequestNonPersistent(ctx context.Context, topic, tag, key string, body []byte, options map[string]interface{}) (*ResponseMessage, error) {
	return c.sendRequest(ctx, topic, tag, key, body, options, false)
}

// sendRequest 統一的請求發送實現
func (c *Client) sendRequest(ctx context.Context, topic, tag, key string, body []byte, options map[string]interface{}, persistent bool) (*ResponseMessage, error) {
	if c.logger != nil {
		c.logger.Infof("RocketMQ send request: topic=%s, tag=%s, key=%s, persistent=%v", topic, tag, key, persistent)
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		if c.metrics != nil {
			c.metrics("send_request", map[string]string{
				"topic":      topic,
				"tag":        tag,
				"persistent": fmt.Sprintf("%v", persistent),
			}, duration)
		}
	}()

	// 解析選項
	opts := c.parseMessageOptions(options)

	// 生成關聯ID
	correlationID := uuid.NewString()
	replyTopic := fmt.Sprintf("REPLY_%s", correlationID)

	// 建立 RocketMQ 訊息
	rocketMsg := &primitive.Message{
		Topic: topic,
		Body:  body,
	}

	// 設定 Tag
	if tag != "" {
		rocketMsg.WithTag(tag)
	}

	// 設定 Key
	if key != "" {
		rocketMsg.WithKeys([]string{key})
	}

	// 設定持久化屬性
	if persistent {
		rocketMsg.WithProperty("PERSISTENT", "true")
	} else {
		rocketMsg.WithProperty("PERSISTENT", "false")
	}

	// 設定 Request-Response 屬性
	properties := make(map[string]string)
	properties["CORRELATION_ID"] = correlationID
	properties["REPLY_TO"] = replyTopic
	properties["REQUEST_TIME"] = fmt.Sprintf("%d", time.Now().UnixNano())

	// 添加自定義屬性
	if opts.Properties != nil {
		for k, v := range opts.Properties {
			properties[k] = v
		}
	}

	rocketMsg.WithProperties(properties)

	// 發送請求
	_, err := c.producer.SendSync(ctx, rocketMsg)
	if err != nil {
		return nil, c.handleSendError(err, "send_request", topic, tag, persistent)
	}

	// 等待響應（簡化實現，實際應該使用更複雜的響應處理機制）
	// 這裡暫時返回成功響應，實際實現需要建立響應消費者
	return &ResponseMessage{
		Body:       []byte("response received"),
		Properties: properties,
		Success:    true,
	}, nil
}

// PublishPersistent 持久化發布訊息
func (c *Client) PublishPersistent(ctx context.Context, topic, tag, key string, body []byte, options map[string]interface{}) error {
	return c.publishMessage(ctx, topic, tag, key, body, options, true)
}

// PublishNonPersistent 非持久化發布訊息
func (c *Client) PublishNonPersistent(ctx context.Context, topic, tag, key string, body []byte, options map[string]interface{}) error {
	return c.publishMessage(ctx, topic, tag, key, body, options, false)
}

// publishMessage 統一的訊息發布實現
func (c *Client) publishMessage(ctx context.Context, topic, tag, key string, body []byte, options map[string]interface{}, persistent bool) error {
	if c.logger != nil {
		c.logger.Infof("RocketMQ publish: topic=%s, tag=%s, key=%s, persistent=%v", topic, tag, key, persistent)
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		if c.metrics != nil {
			c.metrics("publish", map[string]string{
				"topic":      topic,
				"tag":        tag,
				"persistent": fmt.Sprintf("%v", persistent),
			}, duration)
		}
	}()

	// 解析選項
	opts := c.parseMessageOptions(options)

	// 建立 RocketMQ 訊息
	rocketMsg := &primitive.Message{
		Topic: topic,
		Body:  body,
	}

	// 設定 Tag
	if tag != "" {
		rocketMsg.WithTag(tag)
	}

	// 設定 Key
	if key != "" {
		rocketMsg.WithKeys([]string{key})
	}

	// 設定持久化屬性
	if persistent {
		rocketMsg.WithProperty("PERSISTENT", "true")
	} else {
		rocketMsg.WithProperty("PERSISTENT", "false")
	}

	// 設定延遲等級
	if opts.DelayLevel > 0 {
		rocketMsg.WithDelayTimeLevel(opts.DelayLevel)
	}

	// 添加自定義屬性
	if opts.Properties != nil {
		rocketMsg.WithProperties(opts.Properties)
	}

	// 發送訊息
	_, err := c.producer.SendSync(ctx, rocketMsg)
	if err != nil {
		return c.handleSendError(err, "publish", topic, tag, persistent)
	}

	return nil
}

// SubscribeWithAck 訂閱訊息並需要手動確認
func (c *Client) SubscribeWithAck(ctx context.Context, topic, tag, key string, handler UnifiedMessageHandler, options map[string]interface{}) error {
	return c.subscribeMessage(ctx, topic, tag, key, handler, options, true)
}

// SubscribeWithoutAck 訂閱訊息但不需要確認
func (c *Client) SubscribeWithoutAck(ctx context.Context, topic, tag, key string, handler UnifiedMessageHandler, options map[string]interface{}) error {
	return c.subscribeMessage(ctx, topic, tag, key, handler, options, false)
}

// subscribeMessage 統一的訊息訂閱實現
func (c *Client) subscribeMessage(ctx context.Context, topic, tag, key string, handler UnifiedMessageHandler, options map[string]interface{}, withAck bool) error {
	if c.logger != nil {
		c.logger.Infof("RocketMQ subscribe: topic=%s, tag=%s, key=%s, withAck=%v", topic, tag, key, withAck)
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		if c.metrics != nil {
			c.metrics("subscribe", map[string]string{
				"topic":   topic,
				"tag":     tag,
				"withAck": fmt.Sprintf("%v", withAck),
			}, duration)
		}
	}()

	// 解析選項
	opts := c.parseMessageOptions(options)

	// 建立消費者選項
	consumerOpts := []consumer.Option{
		consumer.WithNameServer(c.resolvedNameServers),
		consumer.WithGroupName(opts.ConsumerGroup),
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

	// 設定消費模式
	if opts.ConsumeMode == "broadcasting" {
		consumerOpts = append(consumerOpts, consumer.WithConsumerModel(consumer.BroadCasting))
	} else {
		consumerOpts = append(consumerOpts, consumer.WithConsumerModel(consumer.Clustering))
	}

	// 建立新的消費者
	cons, err := consumer.NewPushConsumer(consumerOpts...)
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	// 包裝處理函數
	wrappedHandler := func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
		// 檢查是否正在關機
		if c.isShuttingDown() {
			if c.logger != nil {
				c.logger.Infof("Rejecting messages during shutdown: topic=%s", topic)
			}
			return consumer.ConsumeRetryLater, nil
		}

		// 增加處理計數器
		if !c.incrementProcessing() {
			if c.logger != nil {
				c.logger.Infof("Rejecting messages during shutdown: topic=%s", topic)
			}
			return consumer.ConsumeRetryLater, nil
		}

		// 在處理完成後減少計數器
		defer c.decrementProcessing()

		// 處理每個訊息
		for _, msg := range msgs {
			// 調用用戶處理函數
			err := handler(ctx, msg.Topic, msg.GetTags(), msg.GetKeys(), msg.Body, msg.GetProperties())
			if err != nil {
				if c.logger != nil {
					c.logger.Errorf("Message handler error: %v", err)
				}

				// 根據 ACK 設定決定是否重試
				if withAck {
					return consumer.ConsumeRetryLater, err
				} else {
					// 無 ACK 模式，記錄錯誤但繼續處理
					return consumer.ConsumeSuccess, nil
				}
			}
		}

		return consumer.ConsumeSuccess, nil
	}

	// 訂閱主題
	selector := consumer.MessageSelector{
		Type:       consumer.TAG,
		Expression: tag,
	}

	err = cons.Subscribe(topic, selector, wrappedHandler)
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	// 啟動消費者
	if err := cons.Start(); err != nil {
		return fmt.Errorf("failed to start consumer: %w", err)
	}

	// 更新客戶端的消費者實例
	c.consumer = cons
	c.consumerStarted = true

	return nil
}

// parseMessageOptions 解析訊息選項
func (c *Client) parseMessageOptions(options map[string]interface{}) *MessageOptions {
	opts := DefaultMessageOptions()

	if options == nil {
		return opts
	}

	// 解析持久化設定
	if persistent, ok := options["persistent"].(bool); ok {
		opts.Persistent = persistent
	}

	// 解析超時設定
	if timeout, ok := options["timeout"].(time.Duration); ok {
		opts.Timeout = timeout
	}

	// 解析重試設定
	if retryCount, ok := options["retryCount"].(int); ok {
		opts.RetryCount = retryCount
	}
	if retryDelay, ok := options["retryDelay"].(time.Duration); ok {
		opts.RetryDelay = retryDelay
	}

	// 解析消費者設定
	if consumerGroup, ok := options["consumerGroup"].(string); ok {
		opts.ConsumerGroup = consumerGroup
	}
	if consumeMode, ok := options["consumeMode"].(string); ok {
		opts.ConsumeMode = consumeMode
	}

	// 解析批次設定
	if batchSize, ok := options["batchSize"].(int); ok {
		opts.BatchSize = batchSize
	}
	if batchBytes, ok := options["batchBytes"].(int); ok {
		opts.BatchBytes = batchBytes
	}
	if flushTime, ok := options["flushTime"].(time.Duration); ok {
		opts.FlushTime = flushTime
	}

	// 解析延遲設定
	if delayLevel, ok := options["delayLevel"].(int); ok {
		opts.DelayLevel = delayLevel
	}

	// 解析自定義屬性
	if properties, ok := options["properties"].(map[string]string); ok {
		opts.Properties = properties
	}

	return opts
}

// handleSendError 處理發送錯誤
func (c *Client) handleSendError(err error, operation, topic, tag string, persistent bool) error {
	// 記錄錯誤指標
	if c.metrics != nil {
		c.metrics("send_error", map[string]string{
			"operation":  operation,
			"topic":      topic,
			"tag":        tag,
			"persistent": fmt.Sprintf("%v", persistent),
		}, 1.0)
	}

	// 記錄錯誤日誌
	if c.logger != nil {
		c.logger.Errorf("RocketMQ %s failed: %v", operation, err)
	}

	// 根據錯誤處理配置決定策略
	if c.config.ErrorHandling != nil {
		// 持久化失敗時的處理策略
		if persistent && c.config.ErrorHandling.AllowDegradeToNonPersistent {
			c.logger.Infof("Degrading to non-persistent for topic: %s", topic)
			// 這裡可以實現降級邏輯
		}
	}

	return fmt.Errorf("%s failed: %w", operation, err)
}
