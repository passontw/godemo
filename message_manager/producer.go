package messagemanager

import (
	"context"
	"fmt"
	"sync"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
)

// 池化生產者
type PooledProducer struct {
	producer  rocketmq.Producer
	groupName string
	mu        sync.RWMutex
}

func NewPooledProducer(nameservers []string, groupName string) (*PooledProducer, error) {
	producer, err := rocketmq.NewProducer(
		producer.WithNameServer(nameservers),
		producer.WithGroupName(groupName),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %v", err)
	}

	err = producer.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start producer: %v", err)
	}

	return &PooledProducer{
		producer:  producer,
		groupName: groupName,
	}, nil
}

func (p *PooledProducer) SendSync(ctx context.Context, msg *primitive.Message) (*primitive.SendResult, error) {
	return p.producer.SendSync(ctx, msg)
}
