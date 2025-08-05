package messagemanager

type MessageManager struct {
	// consumer        rocketmq.PushConsumer
	reqresProducers *ProducerPool
	pubsubProducers *ProducerPool
}

func NewMessageManager(nameservers []string, groupName string) *MessageManager {
	reqresProducers := NewProducerPool(nameservers, groupName, "reqres", 10)
	pubsubProducers := NewProducerPool(nameservers, groupName, "pubsub", 10)
	return &MessageManager{
		reqresProducers: reqresProducers,
		pubsubProducers: pubsubProducers,
	}
}

func (m *MessageManager) GetReqResProducer() *PooledProducer {
	return m.reqresProducers.producers[0]
}

func (m *MessageManager) GetPubSubProducer() *PooledProducer {
	return m.pubsubProducers.producers[0]
}
