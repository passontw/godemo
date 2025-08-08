package producermanager

import (
	"fmt"
	"sync"
	"time"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"github.com/apache/rocketmq-client-go/v2/rlog"
)

// Producer 生產者管理器
type Producer struct {
	config   *ProducerConfig
	producer rocketmq.Producer
	mu       sync.RWMutex
	started  bool
}

// NewProducerManager 建立新的生產者管理器
func NewProducerManager(cfg *ProducerConfig) (IProducer, error) {
	if cfg.NameServers == nil {
		return nil, fmt.Errorf("name servers is required")
	}

	if cfg.GroupName == "" {
		return nil, fmt.Errorf("group name is required")
	}

	if cfg.SendMsgTimeout == 0 {
		cfg.SendMsgTimeout = 3 * time.Second
		rlog.Info("send msg timeout is required, use default value", map[string]interface{}{
			"sendMsgTimeout": cfg.SendMsgTimeout,
		})
	}

	if cfg.RetryTimesWhenSendFailed == 0 {
		cfg.RetryTimesWhenSendFailed = 2
		rlog.Info("retry times when send failed is required, use default value", map[string]interface{}{
			"retryTimesWhenSendFailed": cfg.RetryTimesWhenSendFailed,
		})
	}

	if cfg.CompressMsgBodyOverHowmuch == 0 {
		cfg.CompressMsgBodyOverHowmuch = 4096
		rlog.Info("compress msg body over howmuch is required, use default value", map[string]interface{}{
			"compressMsgBodyOverHowmuch": cfg.CompressMsgBodyOverHowmuch,
		})
	}

	// 建立生產者選項
	opts := []producer.Option{
		producer.WithNameServer(cfg.NameServers),
		producer.WithGroupName(cfg.GroupName),
		producer.WithSendMsgTimeout(cfg.SendMsgTimeout),
		producer.WithRetry(cfg.RetryTimesWhenSendFailed),
		producer.WithCompressMsgBodyOverHowmuch(cfg.CompressMsgBodyOverHowmuch),
	}

	// 建立生產者
	prod, err := rocketmq.NewProducer(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	return &Producer{
		config:   cfg,
		producer: prod,
	}, nil
}

// Start 啟動生產者
func (p *Producer) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started {
		return nil
	}

	// 啟動生產者
	err := p.producer.Start()
	if err != nil {
		return fmt.Errorf("failed to start producer: %w", err)
	}

	p.started = true
	rlog.Info("Producer started successfully", map[string]interface{}{
		"groupName":   p.config.GroupName,
		"nameServers": p.config.NameServers,
	})

	return nil
}

// Shutdown 關閉生產者
func (p *Producer) Shutdown() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.started || p.producer == nil {
		return nil
	}

	err := p.producer.Shutdown()
	if err != nil {
		return fmt.Errorf("failed to shutdown producer: %w", err)
	}

	p.started = false
	rlog.Info("Producer shutdown successfully", map[string]interface{}{})

	return nil
}

// IsStarted 檢查是否已啟動
func (p *Producer) IsStarted() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.started
}

// GetProducerStats 取得生產者統計資訊
func (p *Producer) GetProducerStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := map[string]interface{}{
		"started": p.started,
		"config":  p.config,
	}

	return stats
}
