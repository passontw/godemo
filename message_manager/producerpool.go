package messagemanager

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/GUAIK-ORG/go-snowflake/snowflake"
	"github.com/apache/rocketmq-client-go/v2/primitive"
)

// 生產者池配置
type ProducerPoolConfig struct {
	Nameservers []string
	PoolSize    int
	GroupName   string
	Prefix      string
	Logger      *log.Logger
}

// 生產者池
type ProducerPool struct {
	producers []*PooledProducer
	config    *ProducerPoolConfig
	mu        sync.RWMutex
	isRunning bool
	snowflake *snowflake.Snowflake
}

type ResponseData struct {
	RequestID string      `json:"request_id"`
	TraceID   string      `json:"trace_id"`
	Data      interface{} `json:"data"`
}

func NewProducerPool(poolConfig *ProducerPoolConfig) *ProducerPool {
	producers := make([]*PooledProducer, poolConfig.PoolSize)
	var err error
	for i := 0; i < poolConfig.PoolSize; i++ {
		producerGroupName := fmt.Sprintf("%s-%s-%d", poolConfig.GroupName, poolConfig.Prefix, i)
		producers[i], err = NewPooledProducer(poolConfig.Nameservers, producerGroupName)
		if err != nil {
			log.Printf("Failed to create producer: %v", err)
		}
	}

	// 初始化雪花演算法
	rand.Seed(time.Now().UnixNano())
	datacenterId := rand.Int63n(32) // 0-31 範圍
	workerId := rand.Int63n(32)     // 0-31 範圍

	sf, err := snowflake.NewSnowflake(datacenterId, workerId)
	if err != nil {
		log.Printf("Failed to create snowflake: %v", err)
		// 如果雪花演算法初始化失敗，使用預設值
		sf, _ = snowflake.NewSnowflake(0, 0)
	}

	return &ProducerPool{
		producers: producers,
		config:    poolConfig,
		snowflake: sf,
	}
}

func (p *ProducerPool) ShutdownAll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, producer := range p.producers {
		if producer != nil {
			producer.Shutdown()
		}
	}
}

func (p *ProducerPool) SendRequest(ctx context.Context, topic string, payload interface{}) (*ResponseData, error) {
	// 檢查 snowflake 是否為 nil
	if p.snowflake == nil {
		return nil, fmt.Errorf("snowflake is not initialized")
	}

	requestId := fmt.Sprintf("%d", p.snowflake.NextVal())
	traceId := fmt.Sprintf("%d", p.snowflake.NextVal())

	requestData := RequestData{
		RequestID: requestId,
		TraceID:   traceId,
		Data:      payload,
	}

	requestBody, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request data: %v", err)
	}

	msg := &primitive.Message{
		Topic: topic,
		Body:  requestBody,
	}
	msg.WithProperty("request_id", requestId)
	msg.WithProperty("trace_id", traceId)

	producer := p.producers[0]

	result, err := producer.SendSync(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}

	log.Printf("send request result: %v", result)

	responseData := &ResponseData{
		RequestID: requestId,
		TraceID:   traceId,
		Data:      result,
	}
	return responseData, nil
}
