package consumermanager

import (
	"context"

	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
)

type IPushConsumer interface {
	GetTopic() string
	Start() error
	GetConsumerStats() map[string]interface{}
	Shutdown() error
	IsStarted() bool
	SubscribeChan(subCfg SubscriptionInfo, callback chan []*primitive.MessageExt) error
	SubscribeFunc(subCfg *SubscriptionInfo, callback func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error)) error
	Unsubscribe(topic string) error
}
