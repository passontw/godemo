package messagemanager

type MessageManager struct {
	ConsumerPool    *ConsumerPool
	ReqresProducers *ProducerPool
	PubsubProducers *ProducerPool
}

func NewMessageManager(consumerPoolConfig *ConsumerPoolConfig, reqresProducerConfig *ProducerPoolConfig, pubsubProducerConfig *ProducerPoolConfig) *MessageManager {
	consumerPool := NewConsumerPool(consumerPoolConfig)
	reqresProducers := NewProducerPool(reqresProducerConfig)
	var pubsubProducers *ProducerPool
	if pubsubProducerConfig != nil {
		pubsubProducers = NewProducerPool(pubsubProducerConfig)
	}

	return &MessageManager{
		ConsumerPool:    consumerPool,
		ReqresProducers: reqresProducers,
		PubsubProducers: pubsubProducers,
	}
}

func (m *MessageManager) GetReqResProducer() *PooledProducer {
	return m.ReqresProducers.producers[0]
}

func (m *MessageManager) GetPubSubProducer() *PooledProducer {
	if m.PubsubProducers == nil {
		return nil
	}
	return m.PubsubProducers.producers[0]
}

func (m *MessageManager) ShutdownAll() {
	m.ConsumerPool.ShutdownAll()
	m.ReqresProducers.ShutdownAll()
	if m.PubsubProducers != nil {
		m.PubsubProducers.ShutdownAll()
	}
}
