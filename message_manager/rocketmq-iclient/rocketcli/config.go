package rocketcli

import (
	"godemo/message_manager/rocketmq-iclient/consumermanager"
	"godemo/message_manager/rocketmq-iclient/producermanager"
)

const (
	MsgTypeRequest  = "sync_request"
	MsgTypeResponse = "sync_response"
)

type RequestCallback func(msg []byte) interface{}

// RocketMQConfig RocketMQ 客戶端配置
type RocketMQConfig struct {
	// 生產者配置
	ProducerConfig producermanager.ProducerConfig `json:"producer" yaml:"producer"`

	// 消費者配置
	ConsumerTopic  string                         `json:"consumerTopic" yaml:"consumerTopic"`
	ConsumerConfig consumermanager.ConsumerConfig `json:"consumer" yaml:"consumer"`

	// 請求處理邏輯
	RequestCallback RequestCallback `json:"requestCallback" yaml:"requestCallback"`
}

type ErrorMsg struct {
	Code    int         `json:"code"`
	Payload interface{} `json:"payload"`
}
