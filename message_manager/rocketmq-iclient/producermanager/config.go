package producermanager

import (
	"context"
	"time"

	"github.com/apache/rocketmq-client-go/v2/primitive"
)

// ProducerConfig 生產者配置
type ProducerConfig struct {
	NameServers                []string      `json:"nameServers" yaml:"nameServers"`                               // 名稱伺服器
	GroupName                  string        `json:"groupName" yaml:"groupName"`                                   // 生產者群組名稱
	SendMsgTimeout             time.Duration `json:"sendMsgTimeout" yaml:"sendMsgTimeout"`                         // 發送訊息超時時間
	RetryTimesWhenSendFailed   int           `json:"retryTimesWhenSendFailed" yaml:"retryTimesWhenSendFailed"`     // 重試次數
	CompressMsgBodyOverHowmuch int           `json:"compressMsgBodyOverHowmuch" yaml:"compressMsgBodyOverHowmuch"` // 壓縮訊息體閾值

	SendStatusCallback func(ctx context.Context, result *primitive.SendResult, err error)
}

func DefaultProducerConfig() *ProducerConfig {
	return &ProducerConfig{
		NameServers:                []string{"127.0.0.1:9876"},
		GroupName:                  "default-producer-group",
		SendMsgTimeout:             3 * time.Second,
		RetryTimesWhenSendFailed:   2,
		CompressMsgBodyOverHowmuch: 4096,
	}
}
