package rocketcli

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/rlog"
)

// SendRequest 發送同步請求
func (r *RocketSyncCli) SendRequest(ctx context.Context, topic string, request interface{}, response interface{}) error {
	if !r.IsStarted() {
		return fmt.Errorf("RocketSyncCli is not started")
	}

	// 生成請求ID
	msgId := generateRequestID()

	// 序列化請求
	requestData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// 建立回應通道
	responseChan := make(chan *primitive.MessageExt, 1) // 使用帶緩衝的 channel
	r.mu.Lock()
	r.requestMap[msgId] = responseChan
	r.mu.Unlock()

	// 清理請求映射和關閉 channel
	defer func() {
		r.mu.Lock()
		delete(r.requestMap, msgId)
		r.mu.Unlock()
		close(responseChan)
	}()

	// 發送請求訊息
	msg := encodeSendMsg(MsgTypeRequest, r.consumerCli.GetTopic(), topic, msgId, requestData)

	if !r.producerCli.IsStarted() {
		return fmt.Errorf("producer is not started")
	}

	if result, err := r.producerCli.SendSyncMessage(ctx, msg); err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	} else {
		switch result.Status {
		case primitive.SendOK:
		default:
			return fmt.Errorf("error callback failed")
		}
	}

	rlog.Info("Request sent successfully", map[string]interface{}{
		"requestID": msgId,
	})

	// 等待回應
	select {
	case responseMsg := <-responseChan:
		_, _, err := decodeResponseMsg(responseMsg, response)
		if err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
		rlog.Info("Response received successfully", map[string]interface{}{"msgId": msgId})
		return nil

	case <-ctx.Done():
		return fmt.Errorf("request timeout: %w", ctx.Err())

	case <-time.After(30 * time.Second): // 預設超時時間
		return fmt.Errorf("request timeout after 30 seconds")
	}
}

func (r *RocketSyncCli) SendResponse(ctx context.Context, topic, msgId string, response interface{}) error {
	responseData, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	msg := encodeSendMsg(MsgTypeResponse, r.consumerCli.GetTopic(), topic, msgId, responseData)
	if result, err := r.producerCli.SendSyncMessage(ctx, msg); err != nil {
		return fmt.Errorf("failed to send response: %w", err)
	} else {
		switch result.Status {
		case primitive.SendOK:
		default:
			if r.errorProducerCallback != nil {
				r.errorProducerCallback(result, msg)
				return fmt.Errorf("error callback failed")
			}
		}
	}

	rlog.Info("Response sent successfully", map[string]interface{}{
		"msg_id": msgId,
	})
	return nil
}

// HandleResponse 處理回應訊息
func (r *RocketSyncCli) handleResponse(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
	for _, msg := range msgs {
		msgType := msg.GetProperty("msg_type")
		if msgType == "" {
			continue
		}

		switch msgType {
		case "sync_request":
			r.handleReqEvent(msg)
		case "sync_response":

			// 檢查是否是回應訊息
			msgId := msg.GetProperty("msg_id")
			if msgId == "" {
				continue
			}

			r.mu.RLock()
			responseChan, exists := r.requestMap[msgId]
			r.mu.RUnlock()

			// 如果沒有找到對應的通道，則執行回傳的錯誤處理
			if !exists && r.errorConsumerCallback != nil {
				// 創建一個新的 Message 對象用於回調
				callbackMsg := &primitive.Message{
					Topic: msg.Topic,
					Body:  msg.Body,
				}
				// 複製屬性
				for k, v := range msg.GetProperties() {
					callbackMsg.WithProperty(k, v)
				}
				r.errorConsumerCallback(callbackMsg)

				continue // 請求已超時或被清理
			}

			// 發送回應到對應的通道
			select {
			case responseChan <- msg:
				rlog.Info("Response handled successfully", map[string]interface{}{"msgId": msgId})
			default:
				rlog.Error("response channel is full for request %s", map[string]interface{}{"msgId": msgId})
			}
		}

	}
	return consumer.ConsumeSuccess, nil
}

func (r *RocketSyncCli) handleReqEvent(requsetMsg *primitive.MessageExt) {
	msgId := requsetMsg.GetProperty("msg_id")
	sourceTopic := requsetMsg.GetProperty("source_topic")

	var errMsg *ErrorMsg
	if msgId == "" {
		errMsg = &ErrorMsg{
			Code:    500,
			Payload: "msg_id is empty",
		}

	} else if sourceTopic == "" {
		errMsg = &ErrorMsg{
			Code:    500,
			Payload: "source_topic is empty",
		}
	}
	if errMsg != nil {
		jsonData, _ := json.Marshal(errMsg)
		msg := encodeSendMsg(MsgTypeResponse, r.consumerCli.GetTopic(), sourceTopic, msgId, jsonData)
		r.producerCli.SendOneWayMessage(context.Background(), msg)
		rlog.Error("handleReqEvent error", map[string]interface{}{"msgId": msgId, "error": errMsg})
		return
	}

	if r.requestCallback != nil {
		response := r.requestCallback(requsetMsg.Body)
		if response == nil {
			rlog.Error("handleReqEvent response is nil", map[string]interface{}{"msgId": msgId})
			return
		}
		r.SendResponse(context.Background(), sourceTopic, msgId, response)
	} else {
		rlog.Error("request callback is not set", map[string]interface{}{
			"msg_id": msgId,
		})
	}
}

func encodeSendMsg(msgType, sourceTopic, targetTopic, msgId string, payload []byte) *primitive.Message {
	msg := primitive.NewMessage(targetTopic, payload)
	msg.WithProperty("msg_id", msgId)
	msg.WithProperty("msg_type", msgType)         // 訊息類型
	msg.WithProperty("source_topic", sourceTopic) // 來源主題
	return msg
}

func decodeResponseMsg(msg *primitive.MessageExt, response interface{}) (topic, msgId string, err error) {
	if err = json.Unmarshal(msg.Body, response); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return msg.Topic, msg.GetProperty("msg_id"), nil
}
