package consumermanager

import (
	"context"
	"time"

	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
)

// ConsumerConfig 消費者配置
type ConsumerConfig struct {
	NameServers       []string                  `json:"nameServers" yaml:"nameServers"`             // 名稱伺服器
	GroupName         string                    `json:"groupName" yaml:"groupName"`                 // 消費者群組名稱
	Topic             string                    `json:"topic" yaml:"topic"`                         // 監聽主題
	MessageModel      consumer.MessageModel     `json:"messageModel" yaml:"messageModel"`           // 訊息模型
	ConsumerType      consumer.ConsumeType      `json:"consumerType" yaml:"consumerType"`           // 消費者類型
	ConsumeFromWhere  consumer.ConsumeFromWhere `json:"consumeFromWhere" yaml:"consumeFromWhere"`   // 消費模式
	ConsumeTimeout    time.Duration             `json:"consumeTimeout" yaml:"consumeTimeout"`       // 消費超時時間
	ConsumeOrderly    bool                      `json:"consumeOrderly" yaml:"consumeOrderly"`       // 是否順序消費 *順序消費會讓 callback 不使用並發*
	MaxReconsumeTimes int32                     `json:"maxReconsumeTimes" yaml:"maxReconsumeTimes"` // 最大重試次數

	MessageSelector consumer.MessageSelector // 消息选择器: 默认全部消费
	Handler         func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error)
}

const (
	ConsumerType_PushConsumer = "push"
	ConsumerType_PullConsumer = "pull"
)

// SubscriptionInfo 訂閱資訊
type SubscriptionInfo struct {
	Topic    string                   // Topic 名稱
	Selector consumer.MessageSelector // 訊息選擇器
}

// DefaultConsumerConfig 預設消費者配置
func DefaultConsumerConfig() *ConsumerConfig {
	return &ConsumerConfig{
		NameServers:       []string{"127.0.0.1:9876"},
		GroupName:         "default-consumer-group",
		Topic:             "default-topic",
		MessageModel:      consumer.Clustering,
		ConsumerType:      consumer.ConsumeType("CONSUME_PASSIVELY"),
		ConsumeFromWhere:  consumer.ConsumeFromLastOffset,
		ConsumeTimeout:    5 * time.Second,
		ConsumeOrderly:    false,
		MaxReconsumeTimes: 3,
	}
}
