package rocketmqclient

import (
	"context"
	"fmt"
	"time"

	"github.com/apache/rocketmq-client-go/v2/primitive"
)

// SendMessage 表示要發送的訊息
type SendMessage struct {
	Topic string
	Tag   string
	Key   string
	Body  []byte
}

// SendResult 表示發送結果
type SendResult struct {
	MessageID   string
	QueueID     int32
	QueueOffset int64
	Status      primitive.SendStatus
}

// Send 發送訊息 (同步)
func (c *Client) Send(ctx context.Context, msg *SendMessage) (*SendResult, error) {
	if c.logger != nil {
		c.logger.Infof("RocketMQ send: topic=%s, tag=%s, key=%s", msg.Topic, msg.Tag, msg.Key)
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		if c.metrics != nil {
			c.metrics("send", map[string]string{
				"topic": msg.Topic,
				"tag":   msg.Tag,
			}, duration)
		}
	}()

	// 建立 RocketMQ 訊息
	rocketMsg := &primitive.Message{
		Topic: msg.Topic,
		Body:  msg.Body,
	}

	// 設定 Tag
	if msg.Tag != "" {
		rocketMsg.WithTag(msg.Tag)
	}

	// 設定 Key
	if msg.Key != "" {
		rocketMsg.WithKeys([]string{msg.Key})
	}

	// 發送訊息
	result, err := c.producer.SendSync(ctx, rocketMsg)
	if err != nil {
		if c.logger != nil {
			c.logger.Errorf("RocketMQ send failed: %v", err)
		}
		return nil, err
	}

	return &SendResult{
		MessageID:   result.MsgID,
		QueueID:     int32(result.MessageQueue.QueueId),
		QueueOffset: result.QueueOffset,
		Status:      result.Status,
	}, nil
}

// SendSync 同步發送訊息（別名）
func (c *Client) SendSync(ctx context.Context, msg *SendMessage) (*SendResult, error) {
	return c.Send(ctx, msg)
}

// SendAsync 異步發送訊息
func (c *Client) SendAsync(ctx context.Context, msg *SendMessage, callback func(*SendResult, error)) error {
	if c.logger != nil {
		c.logger.Infof("RocketMQ send async: topic=%s, tag=%s, key=%s", msg.Topic, msg.Tag, msg.Key)
	}

	start := time.Now()

	// 建立 RocketMQ 訊息
	rocketMsg := &primitive.Message{
		Topic: msg.Topic,
		Body:  msg.Body,
	}

	// 設定 Tag
	if msg.Tag != "" {
		rocketMsg.WithTag(msg.Tag)
	}

	// 設定 Key
	if msg.Key != "" {
		rocketMsg.WithKeys([]string{msg.Key})
	}

	// 包裝 callback 來記錄 metrics
	wrappedCallback := func(ctx context.Context, result *primitive.SendResult, err error) {
		duration := time.Since(start).Seconds()
		if c.metrics != nil {
			status := "success"
			if err != nil {
				status = "failed"
			}
			c.metrics("send_async", map[string]string{
				"topic":  msg.Topic,
				"tag":    msg.Tag,
				"status": status,
			}, duration)
		}

		if callback != nil {
			if err != nil {
				callback(nil, err)
			} else {
				callback(&SendResult{
					MessageID:   result.MsgID,
					QueueID:     int32(result.MessageQueue.QueueId),
					QueueOffset: result.QueueOffset,
					Status:      result.Status,
				}, nil)
			}
		}
	}

	// 異步發送訊息
	return c.producer.SendAsync(ctx, wrappedCallback, rocketMsg)
}

// SendOneway 單向發送訊息（不需要響應）
func (c *Client) SendOneway(ctx context.Context, msg *SendMessage) error {
	if c.logger != nil {
		c.logger.Infof("RocketMQ send oneway: topic=%s, tag=%s, key=%s", msg.Topic, msg.Tag, msg.Key)
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		if c.metrics != nil {
			c.metrics("send_oneway", map[string]string{
				"topic": msg.Topic,
				"tag":   msg.Tag,
			}, duration)
		}
	}()

	// 建立 RocketMQ 訊息
	rocketMsg := &primitive.Message{
		Topic: msg.Topic,
		Body:  msg.Body,
	}

	// 設定 Tag
	if msg.Tag != "" {
		rocketMsg.WithTag(msg.Tag)
	}

	// 設定 Key
	if msg.Key != "" {
		rocketMsg.WithKeys([]string{msg.Key})
	}

	// 單向發送訊息
	err := c.producer.SendOneWay(ctx, rocketMsg)
	if err != nil {
		if c.logger != nil {
			c.logger.Errorf("RocketMQ send oneway failed: %v", err)
		}
		return err
	}

	return nil
}

// SendWithQueue 指定佇列發送訊息
func (c *Client) SendWithQueue(ctx context.Context, msg *SendMessage, queueSelector func(*SendMessage) int) (*SendResult, error) {
	if c.logger != nil {
		c.logger.Infof("RocketMQ send with queue: topic=%s, tag=%s, key=%s", msg.Topic, msg.Tag, msg.Key)
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		if c.metrics != nil {
			c.metrics("send_with_queue", map[string]string{
				"topic": msg.Topic,
				"tag":   msg.Tag,
			}, duration)
		}
	}()

	// 建立 RocketMQ 訊息
	rocketMsg := &primitive.Message{
		Topic: msg.Topic,
		Body:  msg.Body,
	}

	// 設定 Tag
	if msg.Tag != "" {
		rocketMsg.WithTag(msg.Tag)
	}

	// 設定 Key
	if msg.Key != "" {
		rocketMsg.WithKeys([]string{msg.Key})
	}

	// 發送訊息 (佇列選擇器功能在 RocketMQ 中需要自定義實作)
	// 暫時使用標準發送，後續可以根據需要實作佇列選擇邏輯
	result, err := c.producer.SendSync(ctx, rocketMsg)
	if err != nil {
		if c.logger != nil {
			c.logger.Errorf("RocketMQ send with queue failed: %v", err)
		}
		return nil, err
	}

	return &SendResult{
		MessageID:   result.MsgID,
		QueueID:     int32(result.MessageQueue.QueueId),
		QueueOffset: result.QueueOffset,
		Status:      result.Status,
	}, nil
}

// SendDelayed 發送延遲訊息
func (c *Client) SendDelayed(ctx context.Context, msg *SendMessage, delayLevel int) (*SendResult, error) {
	if c.logger != nil {
		c.logger.Infof("RocketMQ send delayed: topic=%s, tag=%s, key=%s, delay=%d", msg.Topic, msg.Tag, msg.Key, delayLevel)
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		if c.metrics != nil {
			c.metrics("send_delayed", map[string]string{
				"topic":       msg.Topic,
				"tag":         msg.Tag,
				"delay_level": fmt.Sprintf("%d", delayLevel),
			}, duration)
		}
	}()

	// 建立 RocketMQ 訊息
	rocketMsg := &primitive.Message{
		Topic: msg.Topic,
		Body:  msg.Body,
	}

	// 設定 Tag
	if msg.Tag != "" {
		rocketMsg.WithTag(msg.Tag)
	}

	// 設定 Key
	if msg.Key != "" {
		rocketMsg.WithKeys([]string{msg.Key})
	}

	// 設定延遲等級
	rocketMsg.WithDelayTimeLevel(delayLevel)

	// 發送訊息
	result, err := c.producer.SendSync(ctx, rocketMsg)
	if err != nil {
		if c.logger != nil {
			c.logger.Errorf("RocketMQ send delayed failed: %v", err)
		}
		return nil, err
	}

	return &SendResult{
		MessageID:   result.MsgID,
		QueueID:     int32(result.MessageQueue.QueueId),
		QueueOffset: result.QueueOffset,
		Status:      result.Status,
	}, nil
}
