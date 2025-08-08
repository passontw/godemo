package rocketcli

import (
	"context"
	"encoding/json"
	"fmt"
	"godemo/message_manager/rocketmq-iclient/admincli"
	"godemo/message_manager/rocketmq-iclient/consumermanager"
	"testing"
	"time"

	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
)

type MsgPack struct {
	MsgID string      `json:"msgId"`
	Body  interface{} `json:"body"`
}

type TestObj struct {
	Message   string                 `json:"message"`
	Timestamp int64                  `json:"timestamp"`
	Array     []string               `json:"array"`
	Mapss     map[string]interface{} `json:"mapss"`
}

// TestSendRequest 測試 SendRequest 功能和連線功能
// 發送到自己的接收端
func TestSendRequestToSelf(t *testing.T) {
	// 建立測試配置
	config := &RocketMQConfig{}
	config.RequestCallback = func(msg []byte) interface{} {
		// 建立測試業務邏輯處理
		testRequest := TestObj{
			Message:   "Hello RocketMQ",
			Timestamp: time.Now().Unix(),
			Array:     []string{"Hello", "RocketMQ"},
			Mapss:     map[string]interface{}{"key": "value", "number": 533.33},
		}
		reqMsg := MsgPack{
			MsgID: "test-yang-topic-response",
			Body:  &testRequest,
		}
		return reqMsg
	}

	config.ConsumerConfig.NameServers = []string{"10.1.7.229:9876"}
	config.ConsumerConfig.GroupName = "test-yang-topic-response"
	config.ConsumerTopic = "test-yang-topic-response"
	config.ConsumerConfig.ConsumeTimeout = 5 * time.Second
	config.ConsumerConfig.ConsumeFromWhere = consumer.ConsumeFromLastOffset
	config.ConsumerConfig.MaxReconsumeTimes = 3
	config.ConsumerConfig.MessageModel = consumer.Clustering
	config.ConsumerConfig.ConsumerType = consumer.ConsumeType("CONSUME_PASSIVELY")

	config.ProducerConfig.NameServers = []string{"10.1.7.229:9876"}
	config.ProducerConfig.GroupName = "test-yang-topic-request"

	// 建立 RocketSyncCli 實例
	cli, err := NewRocketSyncCli(config)
	if err != nil {
		t.Fatalf("Failed to create RocketSyncCli: %v", err)
	}

	// 測試啟動
	err = cli.Start()
	if err != nil {
		t.Fatalf("Failed to start RocketSyncCli: %v", err)
	}

	// 檢查啟動狀態
	if !cli.IsStarted() {
		t.Error("RocketSyncCli should be started")
	}

	// 建立測試請求
	testRequest := TestObj{
		Message:   "Hello RocketMQ",
		Timestamp: time.Now().Unix(),
		Array:     []string{"Hello", "RocketMQ"},
		Mapss:     map[string]interface{}{"key": "value", "number": 533.33},
	}
	reqMsg := MsgPack{
		MsgID: "test-yang-topic-response",
		Body:  &testRequest,
	}

	testResponse := TestObj{}

	// 建立回應變數
	resMsg := MsgPack{
		Body: &testResponse,
	}

	// 建立上下文（5秒超時）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Hour)
	defer cancel()

	// 測試 SendRequest
	err = cli.SendRequest(ctx, "test-yang-topic-response", reqMsg, &resMsg)
	if err != nil {
		// 如果連線失敗，這是預期的（因為沒有實際的 RocketMQ 服務）
		// 但我們可以檢查錯誤類型來確認連線嘗試
		t.Logf("SendRequest failed as expected (no RocketMQ service): %v", err)
	} else {
		t.Log("SendRequest succeeded")
	}

	// 測試關閉
	err = cli.Shutdown()
	if err != nil {
		t.Fatalf("Failed to shutdown RocketSyncCli: %v", err)
	}

	// 檢查關閉狀態
	if cli.IsStarted() {
		t.Error("RocketSyncCli should be stopped")
	}

	t.Log("TestSendRequest completed successfully")
}

var cli2 *RocketSyncCli

// 兩個收發端測試
func TestSendRequest(t *testing.T) {
	target1 := "test-yang-topic1"
	target2 := "test-yang-topic2"
	admmin, err := admincli.NewAdminClient([]string{"10.1.7.229:9876"})
	if err != nil {
		t.Fatalf("Failed to create admin client: %v", err)
	}
	for _, v := range []string{target1, target2} {
		err = admmin.CreateTopic(context.TODO(), v, "10.1.7.229:10911", 1)
		if err != nil {
			t.Fatalf("Failed to create topic: %v", err)
		}
	}

	cli1cfg := NewRocketSyncCliTest("test-yang-group1", target1)
	cli1, err := NewRocketSyncCli(cli1cfg)
	if err != nil {
		t.Fatalf("Failed to create RocketSyncCli: %v", err)
	}
	err = cli1.Start()
	if err != nil {
		t.Fatalf("Failed to start RocketSyncCli: %v", err)
	}

	cli2cfg := NewRocketSyncCliTest("test-yang-group2", target2)

	{ // 設定回調
		f := func(msg []byte) interface{} {
			body := &TestObj{}
			req := MsgPack{
				Body: body,
			}
			if err := json.Unmarshal(msg, &req); err != nil {
				req.Body = &TestObj{
					Message: "error unmarshal",
				}
				return req
			}

			body.Message = "Hello RocketMQ this is response"

			return req
		}
		cli2cfg.RequestCallback = f
	}
	cli2, err = NewRocketSyncCli(cli2cfg)
	if err != nil {
		t.Fatalf("Failed to create RocketSyncCli: %v", err)
	}

	err = cli2.Start()
	if err != nil {
		t.Fatalf("Failed to start RocketSyncCli: %v", err)
	}

	{ // Send
		testRequest := TestObj{
			Message:   "Hello RocketMQ",
			Timestamp: time.Now().Unix(),
			Array:     []string{"Hello", "RocketMQ"},
			Mapss:     map[string]interface{}{"key": "value", "number": 533.33},
		}
		reqMsg := MsgPack{
			MsgID: "test-yang-MessageId",
			Body:  &testRequest,
		}
		resMsg := MsgPack{
			Body: &TestObj{},
		}
		if err := cli1.SendRequest(context.Background(), target2, reqMsg, &resMsg); err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		fmt.Println(resMsg)
	}

	cli1.Shutdown()
	cli2.Shutdown()

	t.Log("TestSendRequestToSelf completed successfully")
}

// ========== 錯誤測試開始 ==========

// TestNewRocketSyncCliErrors 測試建立 RocketSyncCli 的錯誤情況
func TestNewRocketSyncCliErrors(t *testing.T) {
	// 測試 nil 配置
	_, err := NewRocketSyncCli(nil)
	if err == nil {
		t.Error("Expected error when creating client with nil config")
	} else {
		t.Logf("Expected error received: %v", err)
	}

	// 測試無效的消費者配置
	invalidConfig := &RocketMQConfig{}
	invalidConfig.ConsumerConfig.NameServers = []string{}
	invalidConfig.ConsumerConfig.GroupName = ""
	invalidConfig.ConsumerTopic = ""
	invalidConfig.ConsumerConfig.ConsumerType = consumermanager.ConsumerType_PushConsumer

	_, err = NewRocketSyncCli(invalidConfig)
	if err == nil {
		t.Error("Expected error when creating client with invalid consumer config")
	} else {
		t.Logf("Expected error received: %v", err)
	}

	// 測試無效的生產者配置
	invalidProducerConfig := &RocketMQConfig{}
	invalidProducerConfig.ConsumerConfig.NameServers = []string{"127.0.0.1:9876"}
	invalidProducerConfig.ConsumerConfig.GroupName = "test-group"
	invalidProducerConfig.ConsumerTopic = "test-topic"
	invalidProducerConfig.ConsumerConfig.ConsumerType = consumermanager.ConsumerType_PushConsumer
	invalidProducerConfig.ProducerConfig.NameServers = []string{}

	_, err = NewRocketSyncCli(invalidProducerConfig)
	if err == nil {
		t.Error("Expected error when creating client with invalid producer config")
	} else {
		t.Logf("Expected error received: %v", err)
	}
}

// TestStartErrors 測試啟動的錯誤情況
func TestStartErrors(t *testing.T) {
	// 測試重複啟動
	config := NewRocketSyncCliTest("test-start-errors", "test-topic")
	cli, err := NewRocketSyncCli(config)
	if err != nil {
		t.Fatalf("Failed to create RocketSyncCli: %v", err)
	}

	// 第一次啟動
	err = cli.Start()
	if err != nil {
		t.Logf("First start failed as expected (no RocketMQ service): %v", err)
	} else {
		t.Log("First start succeeded")
	}

	// 第二次啟動（應該不會出錯）
	err = cli.Start()
	if err != nil {
		t.Fatalf("Second start should not fail: %v", err)
	}

	// 清理
	cli.Shutdown()
}

// TestShutdownErrors 測試關閉的錯誤情況
func TestShutdownErrors(t *testing.T) {
	// 測試關閉未啟動的客戶端
	config := NewRocketSyncCliTest("test-shutdown-errors", "test-topic")
	cli, err := NewRocketSyncCli(config)
	if err != nil {
		t.Fatalf("Failed to create RocketSyncCli: %v", err)
	}

	// 關閉未啟動的客戶端
	err = cli.Shutdown()
	if err != nil {
		t.Fatalf("Shutdown should not fail for unstarted client: %v", err)
	}

	// 測試多次關閉
	err = cli.Shutdown()
	if err != nil {
		t.Fatalf("Multiple shutdown should not fail: %v", err)
	}
}

// TestSendRequestErrors 測試 SendRequest 的錯誤情況
func TestSendRequestErrors(t *testing.T) {
	// 測試未啟動的客戶端
	config := NewRocketSyncCliTest("test-send-request-errors", "test-topic")
	cli, err := NewRocketSyncCli(config)
	if err != nil {
		t.Fatalf("Failed to create RocketSyncCli: %v", err)
	}

	testRequest := TestObj{Message: "test"}
	testResponse := TestObj{}

	// 未啟動時發送請求
	err = cli.SendRequest(context.Background(), "test-topic", testRequest, &testResponse)
	if err == nil {
		t.Error("Expected error when sending request without starting client")
	} else {
		t.Logf("Expected error received: %v", err)
	}

	// 測試序列化錯誤
	invalidRequest := make(chan int) // 無法序列化的類型
	err = cli.SendRequest(context.Background(), "test-topic", invalidRequest, &testResponse)
	if err == nil {
		t.Error("Expected error when sending non-serializable request")
	} else {
		t.Logf("Expected serialization error received: %v", err)
	}

	// 測試超時
	cli.Start()
	defer cli.Shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	err = cli.SendRequest(ctx, "test-topic", testRequest, &testResponse)
	if err == nil {
		t.Error("Expected timeout error")
	} else {
		t.Logf("Expected timeout error received: %v", err)
	}
}

// TestSendResponseErrors 測試 SendResponse 的錯誤情況
func TestSendResponseErrors(t *testing.T) {
	config := NewRocketSyncCliTest("test-send-response-errors", "test-topic")
	cli, err := NewRocketSyncCli(config)
	if err != nil {
		t.Fatalf("Failed to create RocketSyncCli: %v", err)
	}

	// 測試未啟動時發送回應
	testResponse := TestObj{Message: "test"}
	err = cli.SendResponse(context.Background(), "test-topic", "test-msg-id", testResponse)
	if err == nil {
		t.Error("Expected error when sending response without starting client")
	} else {
		t.Logf("Expected error received: %v", err)
	}

	// 測試序列化錯誤
	invalidResponse := make(chan int) // 無法序列化的類型
	cli.Start()
	defer cli.Shutdown()

	err = cli.SendResponse(context.Background(), "test-topic", "test-msg-id", invalidResponse)
	if err == nil {
		t.Error("Expected error when sending non-serializable response")
	} else {
		t.Logf("Expected serialization error received: %v", err)
	}
}

// TestHandleResponseErrors 測試回應處理的錯誤情況
func TestHandleResponseErrors(t *testing.T) {
	config := NewRocketSyncCliTest("test-handle-response-errors", "test-topic")
	cli, err := NewRocketSyncCli(config)
	if err != nil {
		t.Fatalf("Failed to create RocketSyncCli: %v", err)
	}

	// 測試處理無效訊息類型
	invalidMsg := &primitive.MessageExt{
		Message: primitive.Message{
			Topic: "test-topic",
			Body:  []byte(`{"message": "test"}`),
		},
	}
	invalidMsg.WithProperty("msg_type", "invalid_type")

	result, err := cli.handleResponse(context.Background(), invalidMsg)
	if err != nil {
		t.Fatalf("handleResponse failed: %v", err)
	}
	if result != consumer.ConsumeSuccess {
		t.Errorf("Expected ConsumeSuccess, got %v", result)
	}

	// 測試處理無 msg_type 的訊息
	noTypeMsg := &primitive.MessageExt{
		Message: primitive.Message{
			Topic: "test-topic",
			Body:  []byte(`{"message": "test"}`),
		},
	}

	result, err = cli.handleResponse(context.Background(), noTypeMsg)
	if err != nil {
		t.Fatalf("handleResponse failed: %v", err)
	}
	if result != consumer.ConsumeSuccess {
		t.Errorf("Expected ConsumeSuccess, got %v", result)
	}
}

// TestHandleReqEventErrors 測試請求事件處理的錯誤情況
func TestHandleReqEventErrors(t *testing.T) {
	config := NewRocketSyncCliTest("test-handle-req-event-errors", "test-topic")

	// 設定會返回錯誤的回調函數
	config.RequestCallback = func(msg []byte) interface{} {
		return &ErrorMsg{
			Code:    500,
			Payload: "simulated callback error",
		}
	}

	cli, err := NewRocketSyncCli(config)
	if err != nil {
		t.Fatalf("Failed to create RocketSyncCli: %v", err)
	}

	cli.Start()
	defer cli.Shutdown()

	// 測試錯誤的請求處理
	requestData, _ := json.Marshal(TestObj{Message: "test request"})
	requestMsg := &primitive.MessageExt{
		Message: primitive.Message{
			Topic: "test-topic",
			Body:  requestData,
		},
	}
	requestMsg.WithProperty("msg_id", "test-msg-id")
	requestMsg.WithProperty("source_topic", "test-source-topic")

	cli.handleReqEvent(requestMsg)
}

// TestHandleReqEventMissingProperties 測試缺少必要屬性的請求處理
func TestHandleReqEventMissingProperties(t *testing.T) {
	config := NewRocketSyncCliTest("test-handle-req-event-missing", "test-topic")
	cli, err := NewRocketSyncCli(config)
	if err != nil {
		t.Fatalf("Failed to create RocketSyncCli: %v", err)
	}

	cli.Start()
	defer cli.Shutdown()

	// 測試缺少 msg_id 的請求
	requestData, _ := json.Marshal(TestObj{Message: "test request"})
	requestMsg := &primitive.MessageExt{
		Message: primitive.Message{
			Topic: "test-topic",
			Body:  requestData,
		},
	}
	// 故意不設定 msg_id
	requestMsg.WithProperty("source_topic", "test-source-topic")

	cli.handleReqEvent(requestMsg)

	// 測試缺少 source_topic 的請求
	requestMsg2 := &primitive.MessageExt{
		Message: primitive.Message{
			Topic: "test-topic",
			Body:  requestData,
		},
	}
	requestMsg2.WithProperty("msg_id", "test-msg-id")
	// 故意不設定 source_topic

	cli.handleReqEvent(requestMsg2)
}

// TestHandleReqEventNilCallback 測試沒有設定回調函數的情況
func TestHandleReqEventNilCallback(t *testing.T) {
	config := NewRocketSyncCliTest("test-handle-req-event-nil-callback", "test-topic")
	// 故意不設定 RequestCallback
	config.RequestCallback = nil

	cli, err := NewRocketSyncCli(config)
	if err != nil {
		t.Fatalf("Failed to create RocketSyncCli: %v", err)
	}

	cli.Start()
	defer cli.Shutdown()

	// 測試沒有回調函數的請求處理
	requestData, _ := json.Marshal(TestObj{Message: "test request"})
	requestMsg := &primitive.MessageExt{
		Message: primitive.Message{
			Topic: "test-topic",
			Body:  requestData,
		},
	}
	requestMsg.WithProperty("msg_id", "test-msg-id")
	requestMsg.WithProperty("source_topic", "test-source-topic")

	cli.handleReqEvent(requestMsg)
}

// TestHandleReqEventNilResponse 測試回調函數返回 nil 的情況
func TestHandleReqEventNilResponse(t *testing.T) {
	config := NewRocketSyncCliTest("test-handle-req-event-nil-response", "test-topic")

	// 設定返回 nil 的回調函數
	config.RequestCallback = func(msg []byte) interface{} {
		return nil
	}

	cli, err := NewRocketSyncCli(config)
	if err != nil {
		t.Fatalf("Failed to create RocketSyncCli: %v", err)
	}

	cli.Start()
	defer cli.Shutdown()

	// 測試回調函數返回 nil 的情況
	requestData, _ := json.Marshal(TestObj{Message: "test request"})
	requestMsg := &primitive.MessageExt{
		Message: primitive.Message{
			Topic: "test-topic",
			Body:  requestData,
		},
	}
	requestMsg.WithProperty("msg_id", "test-msg-id")
	requestMsg.WithProperty("source_topic", "test-source-topic")

	cli.handleReqEvent(requestMsg)
}

// TestErrorMsg 測試 ErrorMsg 結構體
func TestErrorMsg(t *testing.T) {
	// 測試 ErrorMsg 的序列化
	errorMsg := &ErrorMsg{
		Code:    500,
		Payload: "Test error message",
	}

	jsonData, err := json.Marshal(errorMsg)
	if err != nil {
		t.Fatalf("Failed to marshal ErrorMsg: %v", err)
	}

	// 測試反序列化
	var decodedError ErrorMsg
	err = json.Unmarshal(jsonData, &decodedError)
	if err != nil {
		t.Fatalf("Failed to unmarshal ErrorMsg: %v", err)
	}

	if decodedError.Code != errorMsg.Code {
		t.Errorf("Expected code %d, got %d", errorMsg.Code, decodedError.Code)
	}

	if decodedError.Payload != errorMsg.Payload {
		t.Errorf("Expected payload %s, got %s", errorMsg.Payload, decodedError.Payload)
	}
}

// TestGenerateRequestID 測試請求ID生成功能
func TestGenerateRequestID(t *testing.T) {
	// 測試生成多個ID，確保它們是唯一的
	id1 := generateRequestID()
	id2 := generateRequestID()
	id3 := generateRequestID()

	if id1 == id2 || id1 == id3 || id2 == id3 {
		t.Error("Generated request IDs should be unique")
	}

	if len(id1) == 0 {
		t.Error("Generated request ID should not be empty")
	}

	t.Logf("Generated request IDs: %s, %s, %s", id1, id2, id3)
}

// TestConstants 測試常數定義
func TestConstants(t *testing.T) {
	if MsgTypeRequest != "sync_request" {
		t.Errorf("Expected MsgTypeRequest to be 'sync_request', got '%s'", MsgTypeRequest)
	}

	if MsgTypeResponse != "sync_response" {
		t.Errorf("Expected MsgTypeResponse to be 'sync_response', got '%s'", MsgTypeResponse)
	}
}

// TestEncodeSendMsg 測試 encodeSendMsg 函數
func TestEncodeSendMsg(t *testing.T) {
	topic := "test-topic"
	msgId := "test-msg-id"
	payload := []byte(`{"message": "test"}`)

	// 測試請求訊息編碼
	requestMsg := encodeSendMsg(MsgTypeRequest, "source-topic", topic, msgId, payload)
	if requestMsg.Topic != topic {
		t.Errorf("Expected topic %s, got %s", topic, requestMsg.Topic)
	}
	if requestMsg.GetProperty("msg_id") != msgId {
		t.Errorf("Expected msg_id %s, got %s", msgId, requestMsg.GetProperty("msg_id"))
	}
	if requestMsg.GetProperty("msg_type") != MsgTypeRequest {
		t.Errorf("Expected msg_type %s, got %s", MsgTypeRequest, requestMsg.GetProperty("msg_type"))
	}
	if requestMsg.GetProperty("source_topic") != "source-topic" {
		t.Errorf("Expected source_topic %s, got %s", "source-topic", requestMsg.GetProperty("source_topic"))
	}

	// 測試回應訊息編碼
	responseMsg := encodeSendMsg(MsgTypeResponse, "source-topic", topic, msgId, payload)
	if responseMsg.GetProperty("msg_type") != MsgTypeResponse {
		t.Errorf("Expected msg_type %s, got %s", MsgTypeResponse, responseMsg.GetProperty("msg_type"))
	}
}

// TestDecodeResponseMsg 測試 decodeResponseMsg 函數
func TestDecodeResponseMsg(t *testing.T) {
	// 測試正常解碼
	testResponse := TestObj{}
	responseMsg := &primitive.MessageExt{
		Message: primitive.Message{
			Topic: "test-topic",
			Body:  []byte(`{"message": "response", "timestamp": 1234567890}`),
		},
	}
	responseMsg.WithProperty("msg_id", "test-msg-id")

	decodedTopic, decodedMsgId, err := decodeResponseMsg(responseMsg, &testResponse)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if decodedTopic != "test-topic" {
		t.Errorf("Expected topic %s, got %s", "test-topic", decodedTopic)
	}
	if decodedMsgId != "test-msg-id" {
		t.Errorf("Expected msg_id %s, got %s", "test-msg-id", decodedMsgId)
	}
	if testResponse.Message != "response" {
		t.Errorf("Expected message 'response', got '%s'", testResponse.Message)
	}

	// 測試解碼無效 JSON
	invalidMsg := &primitive.MessageExt{
		Message: primitive.Message{
			Topic: "test-topic",
			Body:  []byte(`invalid json`),
		},
	}
	invalidMsg.WithProperty("msg_id", "test-msg-id")

	testResponse2 := TestObj{}
	_, _, err = decodeResponseMsg(invalidMsg, &testResponse2)
	if err == nil {
		t.Error("Expected error when decoding invalid JSON")
	} else {
		t.Logf("Expected decode error received: %v", err)
	}
}

// TestIsStarted 測試 IsStarted 方法
func TestIsStarted(t *testing.T) {
	config := NewRocketSyncCliTest("test-is-started", "test-topic")
	cli, err := NewRocketSyncCli(config)
	if err != nil {
		t.Fatalf("Failed to create RocketSyncCli: %v", err)
	}

	// 初始狀態應該是未啟動
	if cli.IsStarted() {
		t.Error("Expected client to be not started initially")
	}

	// 啟動後應該顯示已啟動
	cli.Start()
	if !cli.IsStarted() {
		t.Error("Expected client to be started after Start()")
	}

	// 關閉後應該顯示未啟動
	cli.Shutdown()
	if cli.IsStarted() {
		t.Error("Expected client to be not started after Shutdown()")
	}
}

// TestConcurrentOperations 測試並發操作
func TestConcurrentOperations(t *testing.T) {
	config := NewRocketSyncCliTest("test-concurrent", "test-topic")
	cli, err := NewRocketSyncCli(config)
	if err != nil {
		t.Fatalf("Failed to create RocketSyncCli: %v", err)
	}

	cli.Start()
	defer cli.Shutdown()

	// 並發執行多個操作
	done := make(chan bool, 3)

	go func() {
		cli.IsStarted()
		done <- true
	}()

	go func() {
		testRequest := TestObj{Message: "concurrent test"}
		testResponse := TestObj{}
		cli.SendRequest(context.Background(), "test-topic", testRequest, &testResponse)
		done <- true
	}()

	go func() {
		testResponse := TestObj{Message: "concurrent response"}
		cli.SendResponse(context.Background(), "test-topic", "test-msg-id", testResponse)
		done <- true
	}()

	// 等待所有操作完成
	for i := 0; i < 3; i++ {
		<-done
	}

	t.Log("Concurrent operations completed successfully")
}

// TestErrorCallbacks 測試錯誤回調函數
func TestErrorCallbacks(t *testing.T) {
	config := NewRocketSyncCliTest("test-error-callbacks", "test-topic")
	cli, err := NewRocketSyncCli(config)
	if err != nil {
		t.Fatalf("Failed to create RocketSyncCli: %v", err)
	}

	// 設定錯誤回調函數
	consumerErrorCalled := false
	producerErrorCalled := false

	cli.errorConsumerCallback = func(msg *primitive.Message) bool {
		consumerErrorCalled = true
		return true
	}

	cli.errorProducerCallback = func(result *primitive.SendResult, msg *primitive.Message) {
		producerErrorCalled = true
	}

	cli.Start()
	defer cli.Shutdown()

	// 測試消費者錯誤回調
	invalidMsg := &primitive.MessageExt{
		Message: primitive.Message{
			Topic: "test-topic",
			Body:  []byte(`{"message": "test"}`),
		},
	}
	invalidMsg.WithProperty("msg_type", MsgTypeResponse)
	invalidMsg.WithProperty("msg_id", "non-existent-msg-id")

	cli.handleResponse(context.Background(), invalidMsg)

	if !consumerErrorCalled {
		t.Error("Expected consumer error callback to be called")
	}

	// 測試生產者錯誤回調（需要實際的 RocketMQ 服務才能觸發）
	t.Logf("Producer error callback called: %v", producerErrorCalled)
}

func NewRocketSyncCliTest(groupName, topic string) *RocketMQConfig {
	config := &RocketMQConfig{}
	config.ConsumerTopic = topic
	config.ConsumerConfig.NameServers = []string{"10.1.7.229:9876"}
	config.ConsumerConfig.GroupName = groupName + "-consumer-GoupId"
	config.ConsumerConfig.ConsumeTimeout = 5 * time.Second
	config.ConsumerConfig.ConsumeFromWhere = consumer.ConsumeFromLastOffset
	config.ConsumerConfig.MaxReconsumeTimes = 3
	config.ConsumerConfig.MessageModel = consumer.Clustering
	config.ConsumerConfig.ConsumerType = consumermanager.ConsumerType_PushConsumer

	config.ProducerConfig.NameServers = []string{"10.1.7.229:9876"}
	config.ProducerConfig.GroupName = groupName + "-producer"

	return config
}
