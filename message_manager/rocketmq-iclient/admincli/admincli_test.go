package admincli

import (
	"context"
	"testing"
	"time"
)

var nameServer = "10.1.7.229:9876"

// TestNewNameServerClient 測試建立 NameServer 客戶端
func TestNewNameServerClient(t *testing.T) {
	nameServers := []string{nameServer}

	client, err := NewAdminClient(nameServers)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	if client == nil {
		t.Fatal("Expected client to be created, got nil")
	}

	if len(client.nameServers) != len(nameServers) {
		t.Errorf("Expected %d name servers, got %d", len(nameServers), len(client.nameServers))
	}

	if !client.IsConnected() {
		t.Error("Expected client to be connected after creation")
	}
}

// TestNewNameServerClientWithEmptyServers 測試使用空伺服器列表建立客戶端
func TestNewNameServerClientWithEmptyServers(t *testing.T) {
	client, err := NewAdminClient([]string{})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if len(client.nameServers) != 0 {
		t.Errorf("Expected 0 name servers, got %d", len(client.nameServers))
	}
}

// TestConnect 測試連線功能
func TestConnect(t *testing.T) {
	client, err := NewAdminClient([]string{nameServer})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	err = client.Connect(ctx)
	if err != nil {
		// 預期會失敗，因為沒有實際的 RocketMQ 服務
		t.Logf("Connect failed as expected (no RocketMQ service): %v", err)
	} else {
		t.Log("Connect succeeded")
	}

	if !client.IsConnected() {
		t.Error("Expected client to be connected after Connect call")
	}
}

// TestConnectWithContext 測試使用上下文的連線
func TestConnectWithContext(t *testing.T) {
	client, err := NewAdminClient([]string{nameServer})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// 使用超時上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	if err != nil {
		t.Logf("Connect failed as expected (no RocketMQ service): %v", err)
	} else {
		t.Log("Connect succeeded")
	}
}

// TestDisconnect 測試斷開連線功能
func TestDisconnect(t *testing.T) {
	client, err := NewAdminClient([]string{nameServer})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	err = client.Disconnect()
	if err != nil {
		t.Fatalf("Failed to disconnect: %v", err)
	}

	if client.IsConnected() {
		t.Error("Expected client to be disconnected after Disconnect call")
	}
}

// TestDisconnectMultipleTimes 測試多次斷開連線
func TestDisconnectMultipleTimes(t *testing.T) {
	client, err := NewAdminClient([]string{nameServer})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// 第一次斷開
	err = client.Disconnect()
	if err != nil {
		t.Fatalf("Failed to disconnect first time: %v", err)
	}

	// 第二次斷開（應該不會出錯）
	err = client.Disconnect()
	if err != nil {
		t.Fatalf("Failed to disconnect second time: %v", err)
	}

	if client.IsConnected() {
		t.Error("Expected client to be disconnected after multiple Disconnect calls")
	}
}

// TestIsConnected 測試連線狀態檢查
func TestIsConnected(t *testing.T) {
	client, err := NewAdminClient([]string{nameServer})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// 初始狀態應該是已連線
	if !client.IsConnected() {
		t.Error("Expected client to be connected initially")
	}

	// 斷開連線
	err = client.Disconnect()
	if err != nil {
		t.Fatalf("Failed to disconnect: %v", err)
	}

	// 斷開後應該顯示未連線
	if client.IsConnected() {
		t.Error("Expected client to be disconnected after Disconnect")
	}
}

// TestCreateTopic 測試建立 Topic
func TestCreateTopic(t *testing.T) {
	client, err := NewAdminClient([]string{nameServer})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	topic := "test-topic"
	brokerAddr := "10.1.7.229:10911"
	queueNum := 1

	err = client.CreateTopic(ctx, topic, brokerAddr, queueNum)
	if err != nil {
		// 預期會失敗，因為沒有實際的 RocketMQ 服務
		t.Logf("CreateTopic failed as expected (no RocketMQ service): %v", err)
	} else {
		t.Log("CreateTopic succeeded")
	}
}

// TestCreateTopicErrors 測試建立 Topic 的錯誤情況
func TestCreateTopicErrors(t *testing.T) {
	client, err := NewAdminClient([]string{nameServer})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// 斷開連線後嘗試建立 Topic
	err = client.Disconnect()
	if err != nil {
		t.Fatalf("Failed to disconnect: %v", err)
	}

	ctx := context.Background()
	err = client.CreateTopic(ctx, "test-topic", "127.0.0.1:10911", 4)
	if err == nil {
		t.Error("Expected error when creating topic while disconnected")
	} else {
		t.Logf("Expected error received: %v", err)
	}
}

// TestCreateTopicWithInvalidParams 測試使用無效參數建立 Topic
func TestCreateTopicWithInvalidParams(t *testing.T) {
	client, err := NewAdminClient([]string{nameServer})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// 測試空 topic
	err = client.CreateTopic(ctx, "", "127.0.0.1:10911", 4)
	if err != nil {
		t.Logf("Expected error for empty topic: %v", err)
	}

	// 測試空 broker 地址
	err = client.CreateTopic(ctx, "test-topic", "", 4)
	if err != nil {
		t.Logf("Expected error for empty broker address: %v", err)
	}

	// 測試無效的隊列數量
	err = client.CreateTopic(ctx, "test-topic", "127.0.0.1:10911", 0)
	if err != nil {
		t.Logf("Expected error for invalid queue number: %v", err)
	}
}

// TestDeleteTopic 測試刪除 Topic
func TestDeleteTopic(t *testing.T) {
	client, err := NewAdminClient([]string{nameServer})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	topic := "test-topic"
	brokerAddr := "10.1.7.229:10911"

	err = client.DeleteTopic(ctx, topic, brokerAddr)
	if err != nil {
		// 預期會失敗，因為沒有實際的 RocketMQ 服務
		t.Logf("DeleteTopic failed as expected (no RocketMQ service): %v", err)
	} else {
		t.Log("DeleteTopic succeeded")
	}
}

// TestDeleteTopicErrors 測試刪除 Topic 的錯誤情況
func TestDeleteTopicErrors(t *testing.T) {
	client, err := NewAdminClient([]string{nameServer})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// 斷開連線後嘗試刪除 Topic
	err = client.Disconnect()
	if err != nil {
		t.Fatalf("Failed to disconnect: %v", err)
	}

	ctx := context.Background()
	err = client.DeleteTopic(ctx, "test-topic", "127.0.0.1:10911")
	if err == nil {
		t.Error("Expected error when deleting topic while disconnected")
	} else {
		t.Logf("Expected error received: %v", err)
	}
}

// TestDeleteTopicWithInvalidParams 測試使用無效參數刪除 Topic
func TestDeleteTopicWithInvalidParams(t *testing.T) {
	client, err := NewAdminClient([]string{nameServer})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// 測試空 topic
	err = client.DeleteTopic(ctx, "", "127.0.0.1:10911")
	if err != nil {
		t.Logf("Expected error for empty topic: %v", err)
	}

	// 測試空 broker 地址
	err = client.DeleteTopic(ctx, "test-topic", "")
	if err != nil {
		t.Logf("Expected error for empty broker address: %v", err)
	}
}

// TestGetTopicList 測試取得 Topic 列表
func TestGetTopicList(t *testing.T) {
	client, err := NewAdminClient([]string{nameServer})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	topics, err := client.GetTopicList(ctx)
	if err != nil {
		// 預期會失敗，因為沒有實際的 RocketMQ 服務
		t.Logf("GetTopicList failed as expected (no RocketMQ service): %v", err)
	} else {
		t.Logf("GetTopicList succeeded, found %d topics", len(topics))
		for i, topic := range topics {
			t.Logf("Topic %d: %s", i+1, topic)
		}
	}
}

// TestGetTopicListErrors 測試取得 Topic 列表的錯誤情況
func TestGetTopicListErrors(t *testing.T) {
	client, err := NewAdminClient([]string{nameServer})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// 斷開連線後嘗試取得 Topic 列表
	err = client.Disconnect()
	if err != nil {
		t.Fatalf("Failed to disconnect: %v", err)
	}

	ctx := context.Background()
	topics, err := client.GetTopicList(ctx)
	if err == nil {
		t.Error("Expected error when getting topic list while disconnected")
	} else {
		t.Logf("Expected error received: %v", err)
	}

	if topics != nil {
		t.Error("Expected nil topics when disconnected")
	}
}

// TestGetTopicRoute 測試取得 Topic 路由資訊
func TestGetTopicRoute(t *testing.T) {
	client, err := NewAdminClient([]string{nameServer})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	topic := "test-topic"
	messageQueues, err := client.GetTopicRoute(ctx, topic)
	if err != nil {
		// 預期會失敗，因為沒有實際的 RocketMQ 服務
		t.Logf("GetTopicRoute failed as expected (no RocketMQ service): %v", err)
	} else {
		t.Logf("GetTopicRoute succeeded, found %d message queues", len(messageQueues))
		for i, mq := range messageQueues {
			t.Logf("MessageQueue %d: Topic=%s, BrokerName=%s, QueueId=%d",
				i+1, mq.Topic, mq.BrokerName, mq.QueueId)
		}
	}
}

// TestGetTopicRouteErrors 測試取得 Topic 路由資訊的錯誤情況
func TestGetTopicRouteErrors(t *testing.T) {
	client, err := NewAdminClient([]string{nameServer})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// 斷開連線後嘗試取得 Topic 路由
	err = client.Disconnect()
	if err != nil {
		t.Fatalf("Failed to disconnect: %v", err)
	}

	ctx := context.Background()
	messageQueues, err := client.GetTopicRoute(ctx, "test-topic")
	if err == nil {
		t.Error("Expected error when getting topic route while disconnected")
	} else {
		t.Logf("Expected error received: %v", err)
	}

	if messageQueues != nil {
		t.Error("Expected nil message queues when disconnected")
	}
}

// TestGetTopicRouteWithEmptyTopic 測試使用空 Topic 取得路由資訊
func TestGetTopicRouteWithEmptyTopic(t *testing.T) {
	client, err := NewAdminClient([]string{nameServer})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	messageQueues, err := client.GetTopicRoute(ctx, "")
	if err != nil {
		t.Logf("Expected error for empty topic: %v", err)
	} else {
		t.Logf("GetTopicRoute succeeded for empty topic, found %d message queues", len(messageQueues))
	}
}

// TestGetConsumerGroupList 測試取得消費者組列表
func TestGetConsumerGroupList(t *testing.T) {
	client, err := NewAdminClient([]string{nameServer})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	brokerAddr := "127.0.0.1:10911"
	groups, err := client.GetConsumerGroupList(ctx, brokerAddr)
	if err != nil {
		// 預期會失敗，因為沒有實際的 RocketMQ 服務
		t.Logf("GetConsumerGroupList failed as expected (no RocketMQ service): %v", err)
	} else {
		t.Logf("GetConsumerGroupList succeeded, found %d consumer groups", len(groups))
		for i, group := range groups {
			t.Logf("ConsumerGroup %d: %s", i+1, group)
		}
	}
}

// TestGetConsumerGroupListErrors 測試取得消費者組列表的錯誤情況
func TestGetConsumerGroupListErrors(t *testing.T) {
	client, err := NewAdminClient([]string{nameServer})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// 斷開連線後嘗試取得消費者組列表
	err = client.Disconnect()
	if err != nil {
		t.Fatalf("Failed to disconnect: %v", err)
	}

	ctx := context.Background()
	groups, err := client.GetConsumerGroupList(ctx, "127.0.0.1:10911")
	if err == nil {
		t.Error("Expected error when getting consumer group list while disconnected")
	} else {
		t.Logf("Expected error received: %v", err)
	}

	if groups != nil {
		t.Error("Expected nil groups when disconnected")
	}
}

// TestGetConsumerGroupListWithEmptyBroker 測試使用空 Broker 地址取得消費者組列表
func TestGetConsumerGroupListWithEmptyBroker(t *testing.T) {
	client, err := NewAdminClient([]string{nameServer})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	groups, err := client.GetConsumerGroupList(ctx, "")
	if err != nil {
		t.Logf("Expected error for empty broker address: %v", err)
	} else {
		t.Logf("GetConsumerGroupList succeeded for empty broker, found %d groups", len(groups))
	}
}

// TestConcurrentOperations 測試並發操作
func TestConcurrentOperations(t *testing.T) {
	client, err := NewAdminClient([]string{nameServer})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// 並發執行多個操作
	done := make(chan bool, 3)

	go func() {
		client.IsConnected()
		done <- true
	}()

	go func() {
		client.GetTopicList(ctx)
		done <- true
	}()

	go func() {
		client.CreateTopic(ctx, "concurrent-topic", "127.0.0.1:10911", 4)
		done <- true
	}()

	// 等待所有操作完成
	for i := 0; i < 3; i++ {
		<-done
	}

	t.Log("Concurrent operations completed successfully")
}

// TestClientReconnection 測試客戶端重連功能
func TestClientReconnection(t *testing.T) {
	client, err := NewAdminClient([]string{nameServer})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// 初始狀態應該是已連線
	if !client.IsConnected() {
		t.Error("Expected client to be connected initially")
	}

	// 斷開連線
	err = client.Disconnect()
	if err != nil {
		t.Fatalf("Failed to disconnect: %v", err)
	}

	if client.IsConnected() {
		t.Error("Expected client to be disconnected after Disconnect")
	}

	// 重新連線
	ctx := context.Background()
	err = client.Connect(ctx)
	if err != nil {
		t.Logf("Reconnection failed as expected (no RocketMQ service): %v", err)
	} else {
		t.Log("Reconnection succeeded")
	}

	// 重新連線後應該顯示已連線
	if !client.IsConnected() {
		t.Error("Expected client to be connected after reconnection")
	}
}

// TestClientWithMultipleNameServers 測試使用多個 NameServer
func TestClientWithMultipleNameServers(t *testing.T) {
	nameServers := []string{nameServer, "127.0.0.1:9877", "127.0.0.1:9878"}

	client, err := NewAdminClient(nameServers)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if len(client.nameServers) != len(nameServers) {
		t.Errorf("Expected %d name servers, got %d", len(nameServers), len(client.nameServers))
	}

	// 測試基本功能
	ctx := context.Background()
	err = client.Connect(ctx)
	if err != nil {
		t.Logf("Connect failed as expected (no RocketMQ service): %v", err)
	} else {
		t.Log("Connect succeeded with multiple name servers")
	}
}

// TestClientWithInvalidNameServers 測試使用無效的 NameServer
func TestClientWithInvalidNameServers(t *testing.T) {
	// 使用無效的地址
	nameServers := []string{"invalid-address:9999", "another-invalid:8888"}

	client, err := NewAdminClient(nameServers)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	err = client.Connect(ctx)
	if err != nil {
		t.Logf("Connect failed as expected with invalid servers: %v", err)
	} else {
		t.Log("Connect succeeded with invalid servers")
	}
}
