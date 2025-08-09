package messagemanager

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/GUAIK-ORG/go-snowflake/snowflake"
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
)

type RequestCallback func(ctx context.Context, msg *primitive.Message, err error)

// 池化生產者
type PooledProducer struct {
	producer  rocketmq.Producer
	groupName string
	mu        sync.RWMutex
	snowflake *snowflake.Snowflake
}

// 消息结构 - 這個結構已被 protobuf RequestData 取代，保留用於兼容性
type LegacyRequestData struct {
	RequestID string      `json:"request_id"`
	TraceID   string      `json:"trace_id"`
	Data      interface{} `json:"data"`
	Group     string      `json:"group"`
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

	// 使用亂數初始化雪花演算法的 datacenterId 和 workerId
	rand.Seed(time.Now().UnixNano())
	datacenterId := rand.Int63n(32) // 0-31 範圍
	workerId := rand.Int63n(32)     // 0-31 範圍

	sf, err := snowflake.NewSnowflake(datacenterId, workerId)
	if err != nil {
		return nil, fmt.Errorf("failed to create snowflake: %v", err)
	}
	return &PooledProducer{
		producer:  producer,
		groupName: groupName,
		snowflake: sf,
	}, nil
}

func (p *PooledProducer) SendRequest(ctx context.Context, ttl time.Duration, msg *primitive.Message) (*primitive.Message, error) {
	return p.producer.Request(ctx, ttl, msg)
}

func (p *PooledProducer) SendRequestAsync(ctx context.Context, ttl time.Duration, callback RequestCallback, msg *primitive.Message) error {
	// 使用類型轉換將我們的 RequestCallback 轉換為 internal.RequestCallback
	internalCallback := func(ctx context.Context, msg *primitive.Message, err error) {
		callback(ctx, msg, err)
	}
	return p.producer.RequestAsync(ctx, ttl, internalCallback, msg)
}

func (p *PooledProducer) SendSync(ctx context.Context, msg *primitive.Message) (*primitive.SendResult, error) {
	result, err := p.producer.SendSync(ctx, msg)

	log.Printf("send request result: %v", result)
	log.Printf("send request error: %v", err)
	return result, err
}

func (p *PooledProducer) Shutdown() error {
	return p.producer.Shutdown()
}
