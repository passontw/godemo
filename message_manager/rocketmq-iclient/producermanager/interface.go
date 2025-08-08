package producermanager

import (
	"context"

	"github.com/apache/rocketmq-client-go/v2/primitive"
)

type IProducer interface {
	Start() error
	Shutdown() error
	IsStarted() bool
	GetProducerStats() map[string]interface{}

	SendSyncMessage(ctx context.Context, msg *primitive.Message) (*primitive.SendResult, error)
	SendAsyncMessage(ctx context.Context, msg *primitive.Message, callback func(ctx context.Context, result *primitive.SendResult, err error)) error
	SendOneWayMessage(ctx context.Context, msg *primitive.Message) error
	SendDelayMessage(ctx context.Context, msg *primitive.Message, delayLevel int) (*primitive.SendResult, error)
	SendBatchMessages(ctx context.Context, msgs []*primitive.Message) ([]*primitive.SendResult, error)
}
