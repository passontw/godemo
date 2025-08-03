package rocketmqclient

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/apache/rocketmq-client-go/v2/primitive"
)

// HealthCheck 檢查 RocketMQ 連線與生產者/消費者狀態
// ctx - context 控制 timeout/cancel
// 回傳 error - 若斷線、無法訪問等，回傳對應錯誤
func (c *Client) HealthCheck(ctx context.Context) error {
	start := time.Now()

	// 檢查生產者狀態
	if c.producer == nil {
		if c.logger != nil {
			c.logger.Errorf("RocketMQ health check failed: producer not available")
		}
		if c.metrics != nil {
			c.metrics("rocketmq_health_check", map[string]string{
				"status": "failed",
				"reason": "producer_unavailable",
			}, time.Since(start).Seconds())
		}
		return fmt.Errorf("RocketMQ producer not available")
	}

	// 檢查消費者狀態（如果消費者已啟用）
	if c.consumer == nil {
		if c.logger != nil {
			c.logger.Errorf("RocketMQ health check failed: consumer not available")
		}
		if c.metrics != nil {
			c.metrics("rocketmq_health_check", map[string]string{
				"status": "failed",
				"reason": "consumer_unavailable",
			}, time.Since(start).Seconds())
		}
		return fmt.Errorf("RocketMQ consumer not available")
	}

	// 檢查 Name Server 連線狀態
	ch := make(chan error, 1)
	go func() {
		// 嘗試發送一個測試訊息來檢查連線
		testMsg := &primitive.Message{
			Topic: "__health_check_topic__",
			Body:  []byte("health_check"),
		}
		_, err := c.producer.SendSync(context.Background(), testMsg)
		// 即使 topic 不存在，只要能連接到 Name Server 就算成功
		if err != nil && err.Error() != "topic not exist" {
			ch <- err
		} else {
			ch <- nil
		}
	}()

	select {
	case <-ctx.Done():
		if c.logger != nil {
			c.logger.Errorf("RocketMQ health check timeout")
		}
		if c.metrics != nil {
			c.metrics("rocketmq_health_check", map[string]string{
				"status": "failed",
				"reason": "timeout",
			}, time.Since(start).Seconds())
		}
		return ctx.Err()
	case err := <-ch:
		if err != nil {
			if c.logger != nil {
				c.logger.Errorf("RocketMQ health check failed: %v", err)
			}
			if c.metrics != nil {
				c.metrics("rocketmq_health_check", map[string]string{
					"status": "failed",
					"reason": "connection_error",
				}, time.Since(start).Seconds())
			}
			return fmt.Errorf("RocketMQ connection not available: %w", err)
		}
	}

	// 健康檢查成功
	if c.logger != nil {
		c.logger.Infof("RocketMQ health check passed")
	}
	if c.metrics != nil {
		c.metrics("rocketmq_health_check", map[string]string{
			"status": "success",
		}, time.Since(start).Seconds())
	}

	return nil
}

// GetHealthCheckFunc 返回用於 graceful 模組的健康檢查函數
func (c *Client) GetHealthCheckFunc() func(context.Context) error {
	return func(ctx context.Context) error {
		return c.HealthCheck(ctx)
	}
}

// GetAllHealthCheckFuncs 返回所有 RocketMQ 客戶端的健康檢查函數
// 可用於 graceful.Config.ReadinessChecks
func GetAllHealthCheckFuncs() []func(context.Context) error {
	clientsMu.RLock()
	defer clientsMu.RUnlock()

	var checks []func(context.Context) error
	for _, client := range clients {
		checks = append(checks, client.GetHealthCheckFunc())
	}

	return checks
}

// GetHealthCheckFunc 根據客戶端名稱取得健康檢查函數
func GetHealthCheckFunc(name string) func(context.Context) error {
	client, err := GetClient(name)
	if err != nil {
		return func(ctx context.Context) error {
			return fmt.Errorf("RocketMQ client %s not found", name)
		}
	}
	return client.GetHealthCheckFunc()
}

// GracefulShutdown 執行優雅關機，關閉所有 RocketMQ 連線
func GracefulShutdown(ctx context.Context) error {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	log.Printf("[rocketmq-client] starting graceful shutdown for %d clients", len(clients))

	var wg sync.WaitGroup
	errCh := make(chan error, len(clients))

	// 並行關閉所有客戶端
	for name, client := range clients {
		wg.Add(1)
		go func(name string, client *Client) {
			defer wg.Done()

			log.Printf("[rocketmq-client] shutting down client: %s", name)

			// 設定超時關閉 context
			shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			// 關閉連線
			if err := client.closeWithContext(shutdownCtx); err != nil {
				errCh <- fmt.Errorf("failed to close client %s: %w", name, err)
				return
			}

			log.Printf("[rocketmq-client] client %s shutdown completed", name)
		}(name, client)
	}

	// 等待所有客戶端關閉完成
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// 清空客戶端列表
		clients = make(map[string]*Client)
		log.Printf("[rocketmq-client] graceful shutdown completed successfully")
		return nil
	case <-ctx.Done():
		log.Printf("[rocketmq-client] graceful shutdown timeout")
		return ctx.Err()
	case err := <-errCh:
		log.Printf("[rocketmq-client] graceful shutdown error: %v", err)
		return err
	}
}

// closeWithContext 帶 context 的優雅關閉方法
func (c *Client) closeWithContext(ctx context.Context) error {
	if c.producer == nil && c.consumer == nil {
		return nil
	}

	if c.logger != nil {
		c.logger.Infof("Starting graceful shutdown, stopping new message processing")
	}

	// 1. 開始關機流程，停止接收新訊息
	c.startShutdown()

	// 2. 等待所有正在處理的訊息完成
	if err := c.waitForProcessingComplete(ctx); err != nil {
		if c.logger != nil {
			c.logger.Errorf("Timeout waiting for message processing, forcing shutdown: %v", err)
		}
		// 超時後強制關閉
		if c.producer != nil {
			c.producer.Shutdown()
		}
		if c.consumer != nil {
			c.consumer.Shutdown()
		}
		return err
	}

	// 3. 所有訊息處理完成，安全關閉 RocketMQ 連線
	if c.logger != nil {
		c.logger.Infof("All messages processed, closing RocketMQ connection")
	}

	// 優雅關閉 RocketMQ 連線
	if c.producer != nil {
		if err := c.producer.Shutdown(); err != nil {
			if c.logger != nil {
				c.logger.Errorf("Error shutting down producer: %v", err)
			}
			return err
		}
	}

	if c.consumer != nil {
		if err := c.consumer.Shutdown(); err != nil {
			if c.logger != nil {
				c.logger.Errorf("Error shutting down consumer: %v", err)
			}
			return err
		}
	}

	if c.logger != nil {
		c.logger.Infof("RocketMQ connection closed successfully")
	}

	return nil
}
