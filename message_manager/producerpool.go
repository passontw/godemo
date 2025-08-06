package messagemanager

import (
	"fmt"
	"log"
	"sync"
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
	return &ProducerPool{
		producers: producers,
		config:    poolConfig,
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
