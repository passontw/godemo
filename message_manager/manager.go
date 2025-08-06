package messagemanager

type MessageManager struct {
	consumerPool    *ConsumerPool
	reqresProducers *ProducerPool
	pubsubProducers *ProducerPool
}

func NewMessageManager(consumerPoolConfig *ConsumerPoolConfig, reqresProducerConfig *ProducerPoolConfig, pubsubProducerConfig *ProducerPoolConfig) *MessageManager {
	consumerPool := NewConsumerPool(consumerPoolConfig)
	reqresProducers := NewProducerPool(reqresProducerConfig)
	var pubsubProducers *ProducerPool
	if pubsubProducerConfig != nil {
		pubsubProducers = NewProducerPool(pubsubProducerConfig)
	}

	return &MessageManager{
		consumerPool:    consumerPool,
		reqresProducers: reqresProducers,
		pubsubProducers: pubsubProducers,
	}
}

func (m *MessageManager) GetReqResProducer() *PooledProducer {
	return m.reqresProducers.producers[0]
}

func (m *MessageManager) GetPubSubProducer() *PooledProducer {
	if m.pubsubProducers == nil {
		return nil
	}
	return m.pubsubProducers.producers[0]
}

func (m *MessageManager) ShutdownAll() {
	m.consumerPool.ShutdownAll()
	m.reqresProducers.ShutdownAll()
	if m.pubsubProducers != nil {
		m.pubsubProducers.ShutdownAll()
	}
}
