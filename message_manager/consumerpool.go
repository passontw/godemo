package messagemanager

import (
	"fmt"
	"godemo/message_manager/rocketmq-iclient/consumermanager"
	"log"
)

type ConsumerPoolConfig struct {
	ConsumerConfigs []consumermanager.ConsumerConfig
	Nameservers     []string
	Logger          *log.Logger
}

type ConsumerPool struct {
	consumers []consumermanager.IPushConsumer
	config    *ConsumerPoolConfig
}

func NewConsumerPool(poolConfig *ConsumerPoolConfig) *ConsumerPool {
	consumers := make([]consumermanager.IPushConsumer, len(poolConfig.ConsumerConfigs))

	for i, config := range poolConfig.ConsumerConfigs {
		consumer, err := consumermanager.NewPushConsumerManager(config)
		if err != nil {
			log.Printf("failed to create consumer: %v", err)
			continue
		}

		consumer.SubscribeFunc(&consumermanager.SubscriptionInfo{
			Topic:    config.Topic,
			Selector: config.MessageSelector,
		}, config.Handler)
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
