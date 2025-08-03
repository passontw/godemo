package rocketmqclient

import (
	"context"
	"fmt"
	"time"

	"github.com/apache/rocketmq-client-go/v2/primitive"
)

// BatchSendConfig 批次發送配置
type BatchSendConfig struct {
	BatchSize  int           // 批次大小
	BatchBytes int           // 批次最大字節數
	FlushTime  time.Duration // 刷新時間間隔
	MaxRetries int           // 最大重試次數
	RetryDelay time.Duration // 重試延遲
	Timeout    time.Duration // 發送超時
}

// BatchMessage 批次訊息
type BatchMessage struct {
	Messages []*SendMessage
	Config   *BatchSendConfig
}

// NewBatchMessage 建立批次訊息
func NewBatchMessage(config *BatchSendConfig) *BatchMessage {
	if config == nil {
		config = &BatchSendConfig{
			BatchSize:  10,
			BatchBytes: 1024 * 1024, // 1MB
			FlushTime:  100 * time.Millisecond,
			MaxRetries: 3,
			RetryDelay: 100 * time.Millisecond,
			Timeout:    5 * time.Second,
		}
	}

	return &BatchMessage{
		Messages: make([]*SendMessage, 0, config.BatchSize),
		Config:   config,
	}
}

// AddMessage 添加訊息到批次
func (bm *BatchMessage) AddMessage(msg *SendMessage) error {
	if len(bm.Messages) >= bm.Config.BatchSize {
		return fmt.Errorf("batch size limit exceeded: %d", bm.Config.BatchSize)
	}

	// 檢查批次大小限制
	currentSize := bm.calculateBatchSize()
	msgSize := len(msg.Body)
	if currentSize+msgSize > bm.Config.BatchBytes {
		return fmt.Errorf("batch bytes limit exceeded: %d + %d > %d", currentSize, msgSize, bm.Config.BatchBytes)
	}

	bm.Messages = append(bm.Messages, msg)
	return nil
}

// calculateBatchSize 計算批次大小
func (bm *BatchMessage) calculateBatchSize() int {
	size := 0
	for _, msg := range bm.Messages {
		size += len(msg.Body)
		size += len(msg.Topic)
		size += len(msg.Tag)
		size += len(msg.Key)
	}
	return size
}

// Clear 清空批次訊息
func (bm *BatchMessage) Clear() {
	bm.Messages = bm.Messages[:0]
}

// Count 取得批次訊息數量
func (bm *BatchMessage) Count() int {
	return len(bm.Messages)
}

// Size 取得批次大小（字節）
func (bm *BatchMessage) Size() int {
	return bm.calculateBatchSize()
}

// IsFull 檢查批次是否已滿
func (bm *BatchMessage) IsFull() bool {
	return len(bm.Messages) >= bm.Config.BatchSize || bm.calculateBatchSize() >= bm.Config.BatchBytes
}

// IsEmpty 檢查批次是否為空
func (bm *BatchMessage) IsEmpty() bool {
	return len(bm.Messages) == 0
}

// SendBatch 批次發送訊息
func (c *Client) SendBatch(ctx context.Context, batchMsg *BatchMessage) ([]*SendResult, error) {
	if c.logger != nil {
		c.logger.Infof("RocketMQ send batch: count=%d, size=%d bytes", batchMsg.Count(), batchMsg.Size())
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		if c.metrics != nil {
			c.metrics("send_batch", map[string]string{
				"count": fmt.Sprintf("%d", batchMsg.Count()),
				"size":  fmt.Sprintf("%d", batchMsg.Size()),
			}, duration)
		}
	}()

	if batchMsg.IsEmpty() {
		return []*SendResult{}, nil
	}

	// 檢查所有訊息是否屬於同一個主題
	topic := batchMsg.Messages[0].Topic
	for _, msg := range batchMsg.Messages {
		if msg.Topic != topic {
			return nil, fmt.Errorf("batch messages must have the same topic")
		}
	}

	// 轉換為 RocketMQ 訊息格式
	var rocketMsgs []*primitive.Message
	for _, msg := range batchMsg.Messages {
		rocketMsg := &primitive.Message{
			Topic: msg.Topic,
			Body:  msg.Body,
		}

		// 設定標籤
		if msg.Tag != "" {
			rocketMsg.WithTag(msg.Tag)
		}

		// 設定鍵值
		if msg.Key != "" {
			rocketMsg.WithKeys([]string{msg.Key})
		}

		rocketMsgs = append(rocketMsgs, rocketMsg)
	}

	// 批次發送
	result, err := c.producer.SendSync(ctx, rocketMsgs...)
	if err != nil {
		if c.logger != nil {
			c.logger.Errorf("RocketMQ batch send failed: %v", err)
		}
		return nil, err
	}

	// 轉換發送結果
	var results []*SendResult
	results = append(results, &SendResult{
		MessageID:   result.MsgID,
		QueueID:     int32(result.MessageQueue.QueueId),
		QueueOffset: result.QueueOffset,
		Status:      result.Status,
	})

	return results, nil
}

// BatchProducer 批次生產者
type BatchProducer struct {
	client   *Client
	config   *BatchSendConfig
	batch    *BatchMessage
	ticker   *time.Ticker
	sendCh   chan *SendMessage
	resultCh chan *SendResult
	errCh    chan error
	closeCh  chan struct{}
	closedCh chan struct{}
}

// NewBatchProducer 建立批次生產者
func (c *Client) NewBatchProducer(config *BatchSendConfig) *BatchProducer {
	if config == nil {
		config = &BatchSendConfig{
			BatchSize:  10,
			BatchBytes: 1024 * 1024, // 1MB
			FlushTime:  100 * time.Millisecond,
			MaxRetries: 3,
			RetryDelay: 100 * time.Millisecond,
			Timeout:    5 * time.Second,
		}
	}

	bp := &BatchProducer{
		client:   c,
		config:   config,
		batch:    NewBatchMessage(config),
		ticker:   time.NewTicker(config.FlushTime),
		sendCh:   make(chan *SendMessage, config.BatchSize*2),
		resultCh: make(chan *SendResult, config.BatchSize*2),
		errCh:    make(chan error, 1),
		closeCh:  make(chan struct{}),
		closedCh: make(chan struct{}),
	}

	go bp.run()
	return bp
}

// run 批次生產者運行邏輯
func (bp *BatchProducer) run() {
	defer close(bp.closedCh)
	defer bp.ticker.Stop()

	for {
		select {
		case msg := <-bp.sendCh:
			if err := bp.batch.AddMessage(msg); err != nil {
				// 批次已滿，先發送現有批次
				if !bp.batch.IsEmpty() {
					bp.flush()
				}
				// 再次嘗試添加訊息
				if err := bp.batch.AddMessage(msg); err != nil {
					bp.errCh <- err
					continue
				}
			}

			// 檢查是否需要發送
			if bp.batch.IsFull() {
				bp.flush()
			}

		case <-bp.ticker.C:
			// 定時刷新
			if !bp.batch.IsEmpty() {
				bp.flush()
			}

		case <-bp.closeCh:
			// 關閉前發送剩餘訊息
			if !bp.batch.IsEmpty() {
				bp.flush()
			}
			return
		}
	}
}

// flush 刷新批次訊息
func (bp *BatchProducer) flush() {
	if bp.batch.IsEmpty() {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), bp.config.Timeout)
	defer cancel()

	results, err := bp.client.SendBatch(ctx, bp.batch)
	if err != nil {
		bp.errCh <- err
		return
	}

	// 發送結果
	for _, result := range results {
		select {
		case bp.resultCh <- result:
		default:
			// 結果通道已滿，丟棄結果
		}
	}

	bp.batch.Clear()
}

// Send 發送訊息到批次
func (bp *BatchProducer) Send(msg *SendMessage) error {
	select {
	case bp.sendCh <- msg:
		return nil
	default:
		return fmt.Errorf("batch producer send channel is full")
	}
}

// Results 取得發送結果通道
func (bp *BatchProducer) Results() <-chan *SendResult {
	return bp.resultCh
}

// Errors 取得錯誤通道
func (bp *BatchProducer) Errors() <-chan error {
	return bp.errCh
}

// Close 關閉批次生產者
func (bp *BatchProducer) Close() error {
	close(bp.closeCh)
	<-bp.closedCh
	return nil
}

// Flush 強制刷新批次訊息
func (bp *BatchProducer) Flush() {
	bp.flush()
}
