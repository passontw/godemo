package messagemanager

import (
	"fmt"
	"log"
)

type ConsumerPoolConfig struct {
	ConsumerConfigs []ConsumerConfig
	Nameservers     []string
	Logger          *log.Logger
}

type ConsumerPool struct {
	consumers []*SingleConsumer
	config    *ConsumerPoolConfig
}

func NewConsumerPool(poolConfig *ConsumerPoolConfig) *ConsumerPool {
	consumers := make([]*SingleConsumer, len(poolConfig.ConsumerConfigs))

	for i, config := range poolConfig.ConsumerConfigs {
		consumer, err := NewSingleConsumer(&ConsumerConfig{
			Nameservers:  config.Nameservers,
			GroupName:    config.GroupName,
			Topic:        config.Topic,
			MessageModel: config.MessageModel,
		})
		if err != nil {
			log.Printf("failed to create consumer: %v", err)
			continue
		}

		consumer.Subscribe(config.Topic, config.MessageSelector, config.Handler)
		consumer.Start()

		consumers[i] = consumer
	}

	return &ConsumerPool{
		consumers: consumers,
		config:    poolConfig,
	}
}

func (c *ConsumerPool) StartAll() error {
	for _, consumer := range c.consumers {
		err := consumer.Start()
		if err != nil {
			return fmt.Errorf("failed to start consumer: %v", err)
		}
	}
	return nil
}

func (c *ConsumerPool) ShutdownAll() error {
	for _, consumer := range c.consumers {
		err := consumer.Shutdown()
		if err != nil {
			return fmt.Errorf("failed to shutdown consumer: %v", err)
		}
	}
	return nil
}
