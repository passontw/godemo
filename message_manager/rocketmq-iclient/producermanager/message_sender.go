package producermanager

import (
	"context"
	"fmt"

	"github.com/apache/rocketmq-client-go/v2/primitive"
)

// SendSyncMessage 同步發送訊息
func (s *Producer) SendSyncMessage(ctx context.Context, msg *primitive.Message) (*primitive.SendResult, error) {
	if s.producer == nil {
		return nil, fmt.Errorf("producer not initialized")
	}

	if !s.IsStarted() {
		return nil, fmt.Errorf("producer not started")
	}

	return s.producer.SendSync(ctx, msg)
}

// SendAsyncMessage 非同步發送訊息
func (s *Producer) SendAsyncMessage(ctx context.Context, msg *primitive.Message, callback func(ctx context.Context, result *primitive.SendResult, err error)) error {
	if s.producer == nil {
		return fmt.Errorf("producer not initialized")
	}

	if !s.IsStarted() {
		return fmt.Errorf("producer not started")
	}

	return s.producer.SendAsync(ctx, callback, msg)
}

// SendOneWayMessage 單向發送訊息
func (s *Producer) SendOneWayMessage(ctx context.Context, msg *primitive.Message) error {
	if s.producer == nil {
		return fmt.Errorf("producer not initialized")
	}

	if !s.IsStarted() {
		return fmt.Errorf("producer not started")
	}

	return s.producer.SendOneWay(ctx, msg)
}

// SendDelayMessage 發送延遲訊息
func (s *Producer) SendDelayMessage(ctx context.Context, msg *primitive.Message, delayLevel int) (*primitive.SendResult, error) {
	if s.producer == nil {
		return nil, fmt.Errorf("producer not initialized")
	}

	if !s.IsStarted() {
		return nil, fmt.Errorf("producer not started")
	}

	msg.WithDelayTimeLevel(delayLevel)

	return s.producer.SendSync(ctx, msg)
}

// SendBatchMessages 批次發送訊息
func (s *Producer) SendBatchMessages(ctx context.Context, msgs []*primitive.Message) ([]*primitive.SendResult, error) {
	if len(msgs) == 0 {
		return nil, fmt.Errorf("no messages to send")
	}

	if s.producer == nil {
		return nil, fmt.Errorf("producer not initialized")
	}

	if !s.IsStarted() {
		return nil, fmt.Errorf("producer not started")
	}

	var finelErr error
	results := make([]*primitive.SendResult, 0, len(msgs))
	for _, msg := range msgs {
		result, err := s.producer.SendSync(ctx, msg)
		if finelErr != nil {
			finelErr = err
		}
		results = append(results, result)
	}

	return results, finelErr
}
