package rocketcli

import (
	"fmt"
	"sync"
	"time"

	"godemo/message_manager/rocketmq-iclient/consumermanager"
	"godemo/message_manager/rocketmq-iclient/producermanager"

	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/rlog"
)

// RocketSyncCli RocketMQ 同步客戶端
type RocketSyncCli struct {
	producerCli producermanager.IProducer
	consumerCli consumermanager.IPushConsumer
	config      *RocketMQConfig

	requestCallback RequestCallback
	requestMap      map[string]chan *primitive.MessageExt
	mu              sync.RWMutex
	started         bool

	errorConsumerCallback func(msg *primitive.Message) bool                          // 通知上層使用者收到錯誤消息 並確認是否回傳錯誤訊息
	errorProducerCallback func(result *primitive.SendResult, msg *primitive.Message) // 通知上層使用者發送失敗
}

// NewRocketSyncCli 建立新的 RocketMQ 同步客戶端
func NewRocketSyncCli(cfg *RocketMQConfig) (*RocketSyncCli, error) {
	cli := &RocketSyncCli{
		config:          cfg,
		requestMap:      make(map[string]chan *primitive.MessageExt),
		started:         false,
		requestCallback: cfg.RequestCallback,
	}

	if cfg.ConsumerConfig.ConsumerType == consumermanager.ConsumerType_PushConsumer {
		consumerCli, err := consumermanager.NewPushConsumerManager(cfg.ConsumerConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create consumer manager: %w", err)
		}
		consumerCli.SubscribeFunc(&consumermanager.SubscriptionInfo{
			Topic: cfg.ConsumerTopic,
			Selector: consumer.MessageSelector{
				Type:       consumer.TAG,
				Expression: "*",
			},
		}, cli.handleResponse)
		cli.consumerCli = consumerCli
	}

	producerCli, err := producermanager.NewProducerManager(&cfg.ProducerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer manager: %w", err)
	}

	cli.producerCli = producerCli
	return cli, nil
}

// Start 啟動同步客戶端
func (r *RocketSyncCli) Start() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.started {
		return nil
	}

	// 啟動消費者
	if err := r.consumerCli.Start(); err != nil {
		return fmt.Errorf("failed to start consumer: %w", err)
	}

	if err := r.producerCli.Start(); err != nil {
		return fmt.Errorf("failed to start producer: %w", err)
	}

	r.started = true
	rlog.Info("RocketSyncCli started successfully", map[string]interface{}{
		"producerGroup": r.config.ProducerConfig.GroupName,
		"consumerGroup": r.config.ConsumerConfig.GroupName,
		"nameServers":   r.config.ConsumerConfig.NameServers,
	})

	return nil
}

// Shutdown 關閉同步客戶端
func (r *RocketSyncCli) Shutdown() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.started {
		return nil
	}

	var errs []error

	// 關閉生產者
	if r.producerCli != nil {
		if err := r.producerCli.Shutdown(); err != nil {
			errs = append(errs, fmt.Errorf("failed to shutdown producer: %w", err))
		}
	}

	// 關閉消費者
	if r.consumerCli != nil {
		if err := r.consumerCli.Shutdown(); err != nil {
			errs = append(errs, fmt.Errorf("failed to shutdown consumer: %w", err))
		}
	}

	// 清理請求映射
	r.requestMap = make(map[string]chan *primitive.MessageExt)
	r.started = false

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}

	rlog.Info("RocketSyncCli shutdown successfully", map[string]interface{}{})
	return nil
}

// IsStarted 檢查是否已啟動
func (r *RocketSyncCli) IsStarted() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.started
}

// generateRequestID 生成請求ID
func generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}
