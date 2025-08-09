package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	messagemanager "godemo/message_manager"
	pb "godemo/message_manager/proto"

	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"google.golang.org/protobuf/proto"
)

// 錯誤代碼常數
const (
	// 系統級錯誤 (1000-1999)
	ErrSystemInternal    = "SYS_001" // 系統內部錯誤
	ErrSystemTimeout     = "SYS_002" // 系統超時
	ErrSystemUnavailable = "SYS_003" // 系統不可用
	ErrSystemConfig      = "SYS_004" // 系統配置錯誤
	ErrSystemResource    = "SYS_005" // 系統資源不足

	// 網路通訊錯誤 (2000-2999)
	ErrNetworkTimeout     = "NET_001" // 網路超時
	ErrNetworkUnreachable = "NET_002" // 網路不可達
	ErrMessageSendFailed  = "NET_003" // 訊息發送失敗
	ErrMessageParseFailed = "NET_004" // 訊息解析失敗
	ErrMessageSerialize   = "NET_005" // 訊息序列化失敗
	ErrMessageDeserialize = "NET_006" // 訊息反序列化失敗
	ErrConnectionLost     = "NET_007" // 連接丟失
	ErrConnectionRefused  = "NET_008" // 連接被拒絕

	// 業務邏輯錯誤 (3000-3999)
	ErrInvalidRequest   = "BIZ_001" // 無效請求
	ErrValidationFailed = "BIZ_002" // 驗證失敗
	ErrResourceNotFound = "BIZ_003" // 資源不存在
	ErrPermissionDenied = "BIZ_004" // 權限不足
	ErrBusinessLogic    = "BIZ_005" // 業務邏輯錯誤
	ErrDataFormat       = "BIZ_006" // 資料格式錯誤
	ErrDataValidation   = "BIZ_007" // 資料驗證失敗

	// 資料庫錯誤 (4000-4999)
	ErrDatabaseConnection = "DB_001" // 資料庫連接失敗
	ErrDatabaseQuery      = "DB_002" // 資料庫查詢失敗
	ErrDatabaseTimeout    = "DB_003" // 資料庫超時
	ErrDatabaseConstraint = "DB_004" // 資料庫約束違反
	ErrDatabaseDeadlock   = "DB_005" // 資料庫死鎖
	ErrDatabaseRollback   = "DB_006" // 資料庫回滾失敗

	// 外部服務錯誤 (5000-5999)
	ErrExternalService   = "EXT_001" // 外部服務錯誤
	ErrExternalTimeout   = "EXT_002" // 外部服務超時
	ErrExternalAuth      = "EXT_003" // 外部服務認證失敗
	ErrExternalRateLimit = "EXT_004" // 外部服務限流
)

// 錯誤類型
const (
	ErrorTypeSystem   = "SYSTEM"   // 系統錯誤
	ErrorTypeNetwork  = "NETWORK"  // 網路錯誤
	ErrorTypeBusiness = "BUSINESS" // 業務錯誤
	ErrorTypeDatabase = "DATABASE" // 資料庫錯誤
	ErrorTypeExternal = "EXTERNAL" // 外部服務錯誤
)

// 錯誤響應結構
type ErrorResponse struct {
	RequestID    string                 `json:"request_id"`
	TraceID      string                 `json:"trace_id"`
	Success      bool                   `json:"success"`
	ErrorCode    string                 `json:"error_code"`
	ErrorMessage string                 `json:"error_message"`
	ErrorType    string                 `json:"error_type"`
	Details      map[string]interface{} `json:"details,omitempty"`
	Retryable    bool                   `json:"retryable"`
	HTTPStatus   int                    `json:"http_status,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
	ServiceName  string                 `json:"service_name"`
}

// 錯誤代碼資訊結構
type ErrorCodeInfo struct {
	Message    string `json:"message"`
	Type       string `json:"type"`
	Retryable  bool   `json:"retryable"`
	HTTPStatus int    `json:"http_status"`
}

// 錯誤代碼映射表
var ErrorCodeMap = map[string]ErrorCodeInfo{
	// 系統錯誤
	ErrSystemInternal: {
		Message:    "系統內部錯誤",
		Type:       ErrorTypeSystem,
		Retryable:  false,
		HTTPStatus: 500,
	},
	ErrSystemTimeout: {
		Message:    "系統處理超時",
		Type:       ErrorTypeSystem,
		Retryable:  true,
		HTTPStatus: 504,
	},
	ErrSystemUnavailable: {
		Message:    "系統暫時不可用",
		Type:       ErrorTypeSystem,
		Retryable:  true,
		HTTPStatus: 503,
	},
	ErrSystemConfig: {
		Message:    "系統配置錯誤",
		Type:       ErrorTypeSystem,
		Retryable:  false,
		HTTPStatus: 500,
	},
	ErrSystemResource: {
		Message:    "系統資源不足",
		Type:       ErrorTypeSystem,
		Retryable:  true,
		HTTPStatus: 503,
	},

	// 網路錯誤
	ErrNetworkTimeout: {
		Message:    "網路連接超時",
		Type:       ErrorTypeNetwork,
		Retryable:  true,
		HTTPStatus: 503,
	},
	ErrNetworkUnreachable: {
		Message:    "網路不可達",
		Type:       ErrorTypeNetwork,
		Retryable:  true,
		HTTPStatus: 503,
	},
	ErrMessageSendFailed: {
		Message:    "訊息發送失敗",
		Type:       ErrorTypeNetwork,
		Retryable:  true,
		HTTPStatus: 503,
	},
	ErrMessageParseFailed: {
		Message:    "訊息解析失敗",
		Type:       ErrorTypeNetwork,
		Retryable:  false,
		HTTPStatus: 400,
	},
	ErrMessageSerialize: {
		Message:    "訊息序列化失敗",
		Type:       ErrorTypeNetwork,
		Retryable:  false,
		HTTPStatus: 500,
	},
	ErrMessageDeserialize: {
		Message:    "訊息反序列化失敗",
		Type:       ErrorTypeNetwork,
		Retryable:  false,
		HTTPStatus: 400,
	},
	ErrConnectionLost: {
		Message:    "連接丟失",
		Type:       ErrorTypeNetwork,
		Retryable:  true,
		HTTPStatus: 503,
	},
	ErrConnectionRefused: {
		Message:    "連接被拒絕",
		Type:       ErrorTypeNetwork,
		Retryable:  true,
		HTTPStatus: 503,
	},

	// 業務錯誤
	ErrInvalidRequest: {
		Message:    "無效的請求",
		Type:       ErrorTypeBusiness,
		Retryable:  false,
		HTTPStatus: 400,
	},
	ErrValidationFailed: {
		Message:    "資料驗證失敗",
		Type:       ErrorTypeBusiness,
		Retryable:  false,
		HTTPStatus: 400,
	},
	ErrResourceNotFound: {
		Message:    "資源不存在",
		Type:       ErrorTypeBusiness,
		Retryable:  false,
		HTTPStatus: 404,
	},
	ErrPermissionDenied: {
		Message:    "權限不足",
		Type:       ErrorTypeBusiness,
		Retryable:  false,
		HTTPStatus: 403,
	},
	ErrBusinessLogic: {
		Message:    "業務邏輯錯誤",
		Type:       ErrorTypeBusiness,
		Retryable:  false,
		HTTPStatus: 400,
	},
	ErrDataFormat: {
		Message:    "資料格式錯誤",
		Type:       ErrorTypeBusiness,
		Retryable:  false,
		HTTPStatus: 400,
	},
	ErrDataValidation: {
		Message:    "資料驗證失敗",
		Type:       ErrorTypeBusiness,
		Retryable:  false,
		HTTPStatus: 400,
	},

	// 資料庫錯誤
	ErrDatabaseConnection: {
		Message:    "資料庫連接失敗",
		Type:       ErrorTypeDatabase,
		Retryable:  true,
		HTTPStatus: 503,
	},
	ErrDatabaseQuery: {
		Message:    "資料庫查詢失敗",
		Type:       ErrorTypeDatabase,
		Retryable:  true,
		HTTPStatus: 500,
	},
	ErrDatabaseTimeout: {
		Message:    "資料庫操作超時",
		Type:       ErrorTypeDatabase,
		Retryable:  true,
		HTTPStatus: 504,
	},
	ErrDatabaseConstraint: {
		Message:    "資料庫約束違反",
		Type:       ErrorTypeDatabase,
		Retryable:  false,
		HTTPStatus: 400,
	},
	ErrDatabaseDeadlock: {
		Message:    "資料庫死鎖",
		Type:       ErrorTypeDatabase,
		Retryable:  true,
		HTTPStatus: 500,
	},
	ErrDatabaseRollback: {
		Message:    "資料庫回滾失敗",
		Type:       ErrorTypeDatabase,
		Retryable:  false,
		HTTPStatus: 500,
	},

	// 外部服務錯誤
	ErrExternalService: {
		Message:    "外部服務錯誤",
		Type:       ErrorTypeExternal,
		Retryable:  true,
		HTTPStatus: 502,
	},
	ErrExternalTimeout: {
		Message:    "外部服務超時",
		Type:       ErrorTypeExternal,
		Retryable:  true,
		HTTPStatus: 504,
	},
	ErrExternalAuth: {
		Message:    "外部服務認證失敗",
		Type:       ErrorTypeExternal,
		Retryable:  false,
		HTTPStatus: 401,
	},
	ErrExternalRateLimit: {
		Message:    "外部服務限流",
		Type:       ErrorTypeExternal,
		Retryable:  true,
		HTTPStatus: 429,
	},
}

// 錯誤處理工具函數
func GetErrorType(errorCode string) string {
	if errorInfo, exists := ErrorCodeMap[errorCode]; exists {
		return errorInfo.Type
	}
	return ErrorTypeSystem
}

func IsRetryableError(errorCode string) bool {
	if errorInfo, exists := ErrorCodeMap[errorCode]; exists {
		return errorInfo.Retryable
	}
	return false
}

func GetHTTPStatus(errorCode string) int {
	if errorInfo, exists := ErrorCodeMap[errorCode]; exists {
		return errorInfo.HTTPStatus
	}
	return 500
}

func GetErrorMessage(errorCode string) string {
	if errorInfo, exists := ErrorCodeMap[errorCode]; exists {
		return errorInfo.Message
	}
	return "未知錯誤"
}

// 建立錯誤響應
func NewErrorResponse(requestID, traceID string, errorCode string, customMessage string) ErrorResponse {
	errorInfo, exists := ErrorCodeMap[errorCode]
	if !exists {
		errorInfo = ErrorCodeMap[ErrSystemInternal]
		errorCode = ErrSystemInternal
	}

	message := errorInfo.Message
	if customMessage != "" {
		message = customMessage
	}

	return ErrorResponse{
		RequestID:    requestID,
		TraceID:      traceID,
		Success:      false,
		ErrorCode:    errorCode,
		ErrorMessage: message,
		ErrorType:    errorInfo.Type,
		Retryable:    errorInfo.Retryable,
		HTTPStatus:   errorInfo.HTTPStatus,
		Timestamp:    time.Now(),
		ServiceName:  "chat-service",
		Details:      make(map[string]interface{}),
	}
}

// 消息结构
type ChatMessage struct {
	UserID    string    `json:"user_id"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"` // "request", "response", "event"
}

type ChatRequest struct {
	RequestID string      `json:"request_id"`
	TraceID   string      `json:"trace_id"`
	Data      interface{} `json:"data"`
}

type ChatResponse struct {
	RequestID string      `json:"request_id"`
	TraceID   string      `json:"trace_id"`
	Success   bool        `json:"success"`
	Data      interface{} `json:"data"`
	Error     string      `json:"error,omitempty"`
}

type ChatServer struct {
	manager    *messagemanager.MessageManager
	nameserver string
	groupName  string
}

func NewChatServer(nameserver, groupName string) *ChatServer {
	return &ChatServer{
		nameserver: nameserver,
		groupName:  groupName,
	}
}

func (s *ChatServer) Start() error {
	nameservers := []string{s.nameserver}
	consumerConfig := messagemanager.ConsumerConfig{
		Nameservers:  nameservers,
		GroupName:    s.groupName + "_consumer",
		Topic:        "TG001-chat-service-requests",
		MessageModel: consumer.Clustering,
		MessageSelector: consumer.MessageSelector{
			Type:       consumer.TAG,
			Expression: "*",
		},
		ConsumerOrder:     true,
		MaxReconsumeTimes: 3,
		Handler:           s.handleRequest,
	}

	messageManager := messagemanager.NewMessageManager(
		&messagemanager.ConsumerPoolConfig{
			ConsumerConfigs: []messagemanager.ConsumerConfig{consumerConfig},
			Nameservers:     nameservers,
			Logger:          log.Default(),
		},
		&messagemanager.ProducerPoolConfig{
			Nameservers: nameservers,
			GroupName:   s.groupName + "_producer",
			Prefix:      "reqres",
			PoolSize:    1,
			Logger:      log.Default(),
		},
		nil,
	)

	s.manager = messageManager

	log.Printf("聊天服务端已启动，监听请求...")
	return nil
}

func (s *ChatServer) handleRequest(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
	for _, msg := range msgs {
		log.Printf("收到请求消息: %s", string(msg.Body))

		// 印出訊息的 properties
		log.Printf("=== 訊息 Properties ===")
		log.Printf("request_id: %s", msg.GetProperty("request_id"))
		log.Printf("trace_id: %s", msg.GetProperty("trace_id"))
		log.Printf("source_service: %s", msg.GetProperty("source_service"))
		log.Printf("target_service: %s", msg.GetProperty("target_service"))
		log.Printf("======================")

		// 使用 protobuf 反序列化
		var requestData pb.RequestData
		if err := proto.Unmarshal(msg.Body, &requestData); err != nil {
			log.Printf("解析 protobuf 请求失败: %v", err)
			// 從訊息 properties 中獲取 request_id
			requestID := msg.GetProperty("request_id")
			if requestID == "" {
				requestID = "unknown"
			}
			// 發送錯誤回應
			s.sendErrorResponse(requestID, ErrMessageParseFailed, "解析請求失敗")
			continue
		}

		// 印出 protobuf 訊息內容
		log.Printf("=== Protobuf 訊息內容 ===")
		log.Printf("RequestID: %s", requestData.RequestId)
		log.Printf("TraceID: %s", requestData.TraceId)
		log.Printf("Data: %s", requestData.Data)
		log.Printf("SourceService: %s", requestData.SourceService)
		log.Printf("TargetService: %s", requestData.TargetService)
		log.Printf("=========================")

		// 轉換為 ChatRequest 結構
		request := ChatRequest{
			RequestID: requestData.RequestId,
			TraceID:   requestData.TraceId,
			Data:      requestData.Data, // 直接使用字串，不需要轉換
		}

		// 印出處理後的訊息內容
		log.Printf("=== 處理訊息 ===")
		log.Printf("RequestID: %s", request.RequestID)
		log.Printf("TraceID: %s", request.TraceID)
		log.Printf("Data: %s", request.Data)
		log.Printf("=================")

		// 等待一秒模擬處理時間
		log.Printf("等待一秒...")
		time.Sleep(1 * time.Second)
		log.Printf("工作完成...")
		// 處理請求並生成回應
		response := s.processRequest(request)

		// 發送回應
		if err := s.sendResponse(request.RequestID, response); err != nil {
			log.Printf("发送响应失败: %v", err)
			// 使用新的錯誤響應機制
			s.sendErrorResponse(request.RequestID, ErrMessageSendFailed, "发送响应失败")
		}
	}
	return consumer.ConsumeSuccess, nil
}

func (s *ChatServer) processRequest(request ChatRequest) ChatResponse {
	log.Printf("处理请求: %s, TraceID: %s, 数据: %s", request.RequestID, request.TraceID, request.Data)

	response := ChatResponse{
		RequestID: request.RequestID,
		TraceID:   request.TraceID,
		Success:   true,
	}

	response.Data = map[string]interface{}{
		"request_id": request.RequestID,
		"trace_id":   request.TraceID,
		"message":    request.Data,
	}
	log.Printf("消息已发送: %s", response.Data)

	return response
}

func (s *ChatServer) sendResponse(requestID string, response interface{}) error {
	// 將響應轉換為 JSON 字符串
	responseJSON, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("序列化響應失败: %v", err)
	}

	// 使用 protobuf SuccessResponse 結構
	successResponse := &pb.SuccessResponse{
		BaseResponse: &pb.BaseResponse{
			RequestId:     requestID,
			SourceService: "chat-service",
			TargetService: "websocket-service",
		},
		Data: string(responseJSON), // 將 JSON 作為字符串存儲
	}

	// 序列化 protobuf
	responseData, err := proto.Marshal(successResponse)
	if err != nil {
		return fmt.Errorf("序列化 protobuf 響應失敗: %v", err)
	}

	msg := &primitive.Message{
		Topic: "TG001-websocket-service-responses", // 發送到 client 監聽的 topic
		Body:  responseData,
	}

	msg.WithProperty("request_id", requestID)
	msg.WithProperty("response_type", "success_response")

	result, err := s.manager.GetReqResProducer().SendSync(context.Background(), msg)
	if err != nil {
		return fmt.Errorf("发送响应失败: %v", err)
	}

	log.Printf("响应已发送: %s", result.String())
	return nil
}

// 發送錯誤響應
func (s *ChatServer) sendErrorResponse(requestID, errorCode, message string) error {
	// 使用 protobuf ErrorResponse 結構
	errorResponse := &pb.ErrorResponse{
		BaseResponse: &pb.BaseResponse{
			RequestId:     requestID,
			SourceService: "chat-service",
			TargetService: "websocket-service",
		},
		Errorcode: fmt.Sprintf("%s: %s", errorCode, message),
	}

	errorData, err := proto.Marshal(errorResponse)
	if err != nil {
		log.Printf("序列化 protobuf 錯誤響應失敗: %v", err)
		return err
	}

	msg := &primitive.Message{
		Topic: "TG001-websocket-service-responses",
		Body:  errorData,
	}

	msg.WithProperty("request_id", requestID)
	msg.WithProperty("response_type", "error_response")
	msg.WithProperty("error_code", errorCode)

	result, err := s.manager.GetReqResProducer().SendSync(context.Background(), msg)
	if err != nil {
		log.Printf("发送错误响应失败: %v", err)
		return err
	}

	log.Printf("错误响应已发送: %s", result.String())
	return nil
}

func (s *ChatServer) Stop() {
	s.manager.ShutdownAll()
	log.Printf("聊天服务端已停止")
}

func main() {
	log.Printf("聊天服务端启动中...")

	environment := os.Getenv("ROCKETMQ_ENVIRONMENT")
	if environment == "" {
		environment = "k8s"
	}

	nameserver := "10.1.7.229:9876"
	log.Printf("使用 nameserver: %s", nameserver)

	// 使用指定的消費者組名稱
	groupName := "chat-service-uniqueid"
	log.Printf("使用 group name: %s", groupName)

	server := NewChatServer(nameserver, groupName)

	if err := server.Start(); err != nil {
		log.Fatalf("启动服务端失败: %v", err)
	}

	log.Printf("聊天服务端已启动")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	server.Stop()
	log.Printf("收到停止信号，正在关闭服务端...")
}
