package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/admin"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
)

func createTopicUsingProducer(nameserver, topicName string) error {
	// 使用生產者來創建 topic
	p, err := rocketmq.NewProducer(
		producer.WithNameServer([]string{nameserver}),
		producer.WithGroupName("topic_creator_group"),
		producer.WithCreateTopicKey("TBW102"), // 默認的 topic key
	)
	if err != nil {
		return fmt.Errorf("創建生產者失败: %v", err)
	}

	if err := p.Start(); err != nil {
		return fmt.Errorf("啟動生產者失败: %v", err)
	}
	defer p.Shutdown()

	// 嘗試發送一個測試消息來觸發 topic 創建
	msg := &primitive.Message{
		Topic: topicName,
		Body:  []byte("test message for topic creation"),
	}

	result, err := p.SendSync(context.Background(), msg)
	if err != nil {
		log.Printf("發送測試消息失敗，但可能 topic 已創建: %v", err)
	} else {
		log.Printf("成功發送測試消息: %s", result.String())
	}

	return nil
}

func main() {
	nameserver := os.Getenv("ROCKETMQ_NAMESERVER")
	if nameserver == "" {
		nameserver = "10.1.7.229:9876"
	}

	log.Printf("連接到 nameserver: %s", nameserver)

	topics := []string{
		"TG001-chat-service-requests",
		"TG001-chat-service-responses", 
		"TG001-chat-service-events",
	}

	// 方法1: 嘗試使用 admin 工具
	log.Printf("嘗試使用 admin 工具創建 topics...")
	adminTool, err := admin.NewAdmin(admin.WithResolver(primitive.NewPassthroughResolver([]string{nameserver})))
	if err != nil {
		log.Printf("創建 admin 工具失敗: %v", err)
	} else {
		defer adminTool.Close()
		
		for _, topicName := range topics {
			err := adminTool.CreateTopic(context.Background(), admin.WithTopicCreate(topicName))
			if err != nil {
				log.Printf("使用 admin 創建 topic %s 失敗: %v", topicName, err)
			} else {
				log.Printf("成功創建 topic: %s", topicName)
			}
		}
	}

	// 方法2: 使用生產者方式創建 topics
	log.Printf("嘗試使用生產者創建 topics...")
	for _, topicName := range topics {
		log.Printf("創建 topic: %s", topicName)
		if err := createTopicUsingProducer(nameserver, topicName); err != nil {
			log.Printf("創建 topic %s 失敗: %v", topicName, err)
		} else {
			log.Printf("成功處理 topic: %s", topicName)
		}
	}

	log.Printf("Topic 創建完成")
}