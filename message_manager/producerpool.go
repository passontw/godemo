package messagemanager

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	pb "godemo/message_manager/proto"

	"github.com/GUAIK-ORG/go-snowflake/snowflake"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"google.golang.org/protobuf/proto"
)

type AsyncRequestCallback func(ctx context.Context, msg *primitive.Message, err error)
type SendRequestOptions struct {
	SourceService string
	TargetService string
	Topic         string
	GroupName     string
}

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

func (p *ProducerPool) SendRequestAsync(ctx context.Context, options SendRequestOptions, payload interface{}, callback AsyncRequestCallback) {
	// 檢查 snowflake 是否為 nil
	if p.snowflake == nil {
		log.Printf("snowflake is not initialized")
		return
	}

	requestId := fmt.Sprintf("%d", p.snowflake.NextVal())
	traceId := fmt.Sprintf("%d", p.snowflake.NextVal())

	// 將 payload 轉為字串
	var payloadStr string
	if payload != nil {
		payloadStr = fmt.Sprintf("%v", payload)
	}

	// 建立 protobuf 訊息
	requestData := &pb.RequestData{
		RequestId:     requestId,
		TraceId:       traceId,
		Data:          payloadStr,
		SourceService: options.SourceService,
		TargetService: options.TargetService,
	}

	// 序列化為 protobuf
	requestBody, err := proto.Marshal(requestData)
	if err != nil {
		log.Printf("failed to marshal protobuf request data: %v", err)
		return
	}

	msg := &primitive.Message{
		Topic: options.Topic,
		Body:  requestBody,
	}
	msg.WithProperty("request_id", requestId)
	msg.WithProperty("trace_id", traceId)
	msg.WithProperty("source_service", options.SourceService)
	msg.WithProperty("target_service", options.TargetService)

	producer := p.producers[0]

	producer.SendRequestAsync(ctx, 10*time.Second, func(ctx context.Context, msg *primitive.Message, err error) {
		log.Printf("SendRequestAsync callback send request result: %v", msg)
		callback(ctx, msg, err)
	}, msg)
}

func (p *ProducerPool) SendRequest(ctx context.Context, options SendRequestOptions, payload interface{}) (*ResponseData, error) {
	// 檢查 snowflake 是否為 nil
	if p.snowflake == nil {
		return nil, fmt.Errorf("snowflake is not initialized")
	}

	requestId := fmt.Sprintf("%d", p.snowflake.NextVal())
	traceId := fmt.Sprintf("%d", p.snowflake.NextVal())

	// 將 payload 轉為字串
	var payloadStr string
	if payload != nil {
		payloadStr = fmt.Sprintf("%v", payload)
	}

	// 建立 protobuf 訊息
	requestData := &pb.RequestData{
		RequestId:     requestId,
		TraceId:       traceId,
		Data:          payloadStr,
		SourceService: options.SourceService,
		TargetService: options.TargetService,
	}

	// 序列化為 protobuf
	requestBody, err := proto.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf request data: %v", err)
	}

	msg := &primitive.Message{
		Topic: options.Topic,
		Body:  requestBody,
	}
	msg.WithProperty("request_id", requestId)
	msg.WithProperty("trace_id", traceId)
	msg.WithProperty("source_service", options.SourceService)
	msg.WithProperty("target_service", options.TargetService)

	producer := p.producers[0]

	result, err := producer.SendSync(ctx, msg)
	log.Printf("send request result: %v", result)
	log.Printf("send request error: %v", err)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}

	responseData := &ResponseData{
		RequestID: requestId,
		TraceID:   traceId,
		Data:      result,
	}
	return responseData, nil
}
