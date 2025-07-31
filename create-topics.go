package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/apache/rocketmq-client-go/v2/admin"
	"github.com/apache/rocketmq-client-go/v2/primitive"
)

func main() {
	// 從環境變數獲取 nameserver 地址
	nameserver := os.Getenv("ROCKETMQ_NAMESERVER")
	if nameserver == "" {
		nameserver = "10.1.7.229:9876" // 默認值
	}

	log.Printf("使用 nameserver: %s", nameserver)

	// 創建 admin 客戶端
	adminClient, err := admin.NewAdmin(
		admin.WithResolver(primitive.NewPassthroughResolver([]string{nameserver})),
	)
	if err != nil {
		log.Fatalf("創建 admin 客戶端失敗: %v", err)
	}
	defer adminClient.Close()

	// 定義需要創建的 topics
	topics := []string{
		"TG001-chat-service-requests",  // 請求 topic
		"TG001-chat-service-responses", // 響應 topic
		"TG001-chat-service-events",    // 事件 topic
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 創建 topics
	for _, topic := range topics {
		log.Printf("正在創建 topic: %s", topic)

		err := adminClient.CreateTopic(
			ctx,
			admin.WithTopicCreate(topic),
			admin.WithReadQueueNums(4),
			admin.WithWriteQueueNums(4),
		)

		if err != nil {
			log.Printf("創建 topic %s 失敗: %v", topic, err)
		} else {
			log.Printf("成功創建 topic: %s", topic)
		}
	}

	log.Println("Topics 創建完成")
}
