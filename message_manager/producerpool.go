package messagemanager

import (
	"fmt"
	"log"
	"sync"
)

// 生產者池配置
type ProducerPoolConfig struct {
	Nameserver []string
	PoolSize   int
	GroupName  string
	Logger     *log.Logger
}

// 生產者池
type ProducerPool struct {
	producers []*PooledProducer
	config    *ProducerPoolConfig
	mu        sync.RWMutex
	isRunning bool
}

func NewProducerPool(nameservers []string, groupName string, prefix string, poolSize int) *ProducerPool {
	producers := make([]*PooledProducer, poolSize)
	var err error
	for i := 0; i < poolSize; i++ {
		producerGroupName := fmt.Sprintf("%s-%s-%d", groupName, prefix, i)
		producers[i], err = NewPooledProducer(nameservers, producerGroupName)
		if err != nil {
			log.Printf("Failed to create producer: %v", err)
		}
	}
	return &ProducerPool{
		producers: producers,
		config:    &ProducerPoolConfig{Nameserver: nameservers, GroupName: groupName, PoolSize: poolSize},
	}
}
