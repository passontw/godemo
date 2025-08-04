package rocketmqclient

import (
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

// RetryConfig 定義重試參數
// MaxRetries: 最大重試次數
// RetryDelay: 重試間隔
type RetryConfig struct {
	MaxRetries int
	RetryDelay time.Duration
}

// TimeoutConfig 定義超時參數
// SendTimeout: 發送超時
// ReceiveTimeout: 接收超時
// ConnectionTimeout: 連線超時
// HeartbeatTimeout: 心跳超時
type TimeoutConfig struct {
	SendTimeout       time.Duration
	ReceiveTimeout    time.Duration
	ConnectionTimeout time.Duration
	HeartbeatTimeout  time.Duration
}

// ErrorHandlingConfig 錯誤處理配置
type ErrorHandlingConfig struct {
	// 持久化失敗處理
	AllowDegradeToNonPersistent bool          // 允許降級為非持久化
	MaxPersistentRetries        int           // 最大持久化重試次數
	PersistentRetryDelay        time.Duration // 持久化重試延遲

	// ACK 失敗處理
	ContinueOnAckFailure bool          // ACK 失敗時繼續處理
	MaxAckRetries        int           // 最大 ACK 重試次數
	AckRetryDelay        time.Duration // ACK 重試延遲

	// 通用錯誤處理
	EnableErrorMetrics bool        // 啟用錯誤指標
	ErrorCallback      func(error) // 錯誤回調函數
}

// RocketMQConfig 定義 RocketMQ 連線設定
// Name: 連線名稱（多組連線用）
// NameServers: RocketMQ Name Server 位址列表 (格式："192.168.1.100:9876")
// AccessKey/SecretKey: ACL 認證參數
// Username/Password: 簡單認證參數
// SecurityToken: 安全令牌
// Namespace: 命名空間
// Retry: 重試設定
// Timeout: 超時設定
// DNSConfig: DNS 解析配置（K8s 環境使用）
// ErrorHandling: 錯誤處理配置
type RocketMQConfig struct {
	Name        string
	NameServers []string

	// 混合認證配置 (全部可選，支援無認證)
	AccessKey     string // ACL 認證
	SecretKey     string // ACL 認證
	Username      string // 簡單認證
	Password      string // 簡單認證
	SecurityToken string // 安全令牌
	Namespace     string // 命名空間

	// 連線配置
	Retry   *RetryConfig
	Timeout *TimeoutConfig
	DNS     *DNSConfig // DNS 解析配置

	// 錯誤處理配置
	ErrorHandling *ErrorHandlingConfig
}

// DNSConfig DNS 解析配置
type DNSConfig struct {
	// 是否使用 FQDN (完整域名)
	UseFQDN bool
	// DNS 解析超時
	ResolveTimeout time.Duration
	// DNS 刷新間隔
	RefreshInterval time.Duration
	// 是否啟用 DNS 快取
	EnableCache bool
	// DNS 最大重試次數
	MaxRetries int
	// DNS 重試間隔
	RetryInterval time.Duration
}

// DNSResolver DNS 解析器
type DNSResolver struct {
	config      *DNSConfig
	cache       map[string]resolveResult
	cacheMutex  sync.RWMutex
	lastRefresh map[string]time.Time
}

// resolveResult DNS 解析結果
type resolveResult struct {
	resolvedAddress string
	resolveTime     time.Time
	err             error
}

// MessageOptions 訊息選項
type MessageOptions struct {
	// 持久化設定
	Persistent bool

	// 超時設定
	Timeout time.Duration

	// 重試設定
	RetryCount int
	RetryDelay time.Duration

	// 消費者設定
	ConsumerGroup string
	ConsumeMode   string // "clustering" 或 "broadcasting"

	// 批次設定
	BatchSize  int
	BatchBytes int
	FlushTime  time.Duration

	// 延遲設定
	DelayLevel int

	// 自定義屬性
	Properties map[string]string
}

// DefaultRetryConfig 回傳預設重試配置
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries: 3,
		RetryDelay: time.Second,
	}
}

// DefaultTimeoutConfig 回傳預設超時配置
func DefaultTimeoutConfig() *TimeoutConfig {
	return &TimeoutConfig{
		SendTimeout:       30 * time.Second,
		ReceiveTimeout:    30 * time.Second,
		ConnectionTimeout: 10 * time.Second,
		HeartbeatTimeout:  60 * time.Second,
	}
}

// DefaultDNSConfig 回傳預設 DNS 配置
func DefaultDNSConfig() *DNSConfig {
	return &DNSConfig{

		UseFQDN:         true, // K8s 環境建議使用 FQDN
		ResolveTimeout:  5 * time.Second,
		RefreshInterval: 30 * time.Second,
		EnableCache:     true,
		MaxRetries:      3,
		RetryInterval:   1 * time.Second,
	}
}

// DefaultErrorHandlingConfig 回傳預設錯誤處理配置
func DefaultErrorHandlingConfig() *ErrorHandlingConfig {
	return &ErrorHandlingConfig{
		AllowDegradeToNonPersistent: true,
		MaxPersistentRetries:        3,
		PersistentRetryDelay:        1 * time.Second,
		ContinueOnAckFailure:        false,
		MaxAckRetries:               2,
		AckRetryDelay:               500 * time.Millisecond,
		EnableErrorMetrics:          true,
		ErrorCallback:               nil,
	}
}

// DefaultMessageOptions 回傳預設訊息選項
func DefaultMessageOptions() *MessageOptions {
	return &MessageOptions{
		Persistent:    true,
		Timeout:       30 * time.Second,
		RetryCount:    3,
		RetryDelay:    1 * time.Second,
		ConsumerGroup: "default_group",
		ConsumeMode:   "clustering",
		BatchSize:     10,
		BatchBytes:    1024 * 1024, // 1MB
		FlushTime:     100 * time.Millisecond,
		DelayLevel:    0,
		Properties:    make(map[string]string),
	}
}

// NewDNSResolver 創建新的 DNS 解析器
func NewDNSResolver(config *DNSConfig) *DNSResolver {
	if config == nil {
		config = DefaultDNSConfig()
	}

	resolver := &DNSResolver{
		config:      config,
		lastRefresh: make(map[string]time.Time),
	}

	if config.EnableCache {
		resolver.cache = make(map[string]resolveResult)
	}

	return resolver
}

// ResolveName 解析域名為 IP 地址
// 輸入格式: "service-name.namespace.svc.cluster.local:port" 或 "ip:port"
// 輸出格式: "ip:port"
func (r *DNSResolver) ResolveName(nameserver string) (string, error) {
	// 驗證輸入格式
	if err := r.validateNameserver(nameserver); err != nil {
		return "", err
	}

	// 如果已經是 IP 地址格式，直接返回
	if r.isIPAddress(nameserver) {
		return nameserver, nil
	}

	// 檢查快取
	if r.config.EnableCache {
		if cached := r.getCachedResult(nameserver); cached != nil {
			if cached.err != nil {
				return "", cached.err
			}
			return cached.resolvedAddress, nil
		}
	}

	// 執行 DNS 解析
	resolved, err := r.doResolve(nameserver)

	// 更新快取
	if r.config.EnableCache {
		r.setCachedResult(nameserver, resolved, err)
	}

	return resolved, err
}

// ResolveWithRetry 帶重試的域名解析
func (r *DNSResolver) ResolveWithRetry(nameserver string) (string, error) {
	var lastErr error

	for i := 0; i <= r.config.MaxRetries; i++ {
		result, err := r.ResolveName(nameserver)
		if err == nil {
			return result, nil
		}

		lastErr = err
		if i < r.config.MaxRetries {
			time.Sleep(time.Duration(i+1) * r.config.RetryInterval)
		}
	}

	return "", fmt.Errorf("DNS 解析失敗，已重試 %d 次，最後錯誤: %v", r.config.MaxRetries, lastErr)
}

// ResolveNameServers 解析多個 nameserver 地址
func (r *DNSResolver) ResolveNameServers(nameservers []string) ([]string, error) {
	log.Println("ResolveNameServers", nameservers)
	if len(nameservers) == 0 {
		return nil, fmt.Errorf("nameserver 列表不能為空")
	}

	resolved := make([]string, 0, len(nameservers))
	var lastErr error

	for _, ns := range nameservers {
		addr, err := r.ResolveWithRetry(ns)
		if err != nil {
			lastErr = err
			continue // 繼續嘗試其他 nameserver
		}
		resolved = append(resolved, addr)
	}

	if len(resolved) == 0 {
		return nil, fmt.Errorf("無法解析任何 nameserver，最後錯誤: %v", lastErr)
	}

	return resolved, nil
}

// validateNameserver 驗證 nameserver 格式
func (r *DNSResolver) validateNameserver(nameserver string) error {
	if nameserver == "" {
		return fmt.Errorf("nameserver 不能為空")
	}

	if !strings.Contains(nameserver, ":") {
		return fmt.Errorf("nameserver 必須包含端口，格式: host:port")
	}

	_, _, err := net.SplitHostPort(nameserver)
	if err != nil {
		return fmt.Errorf("無效的 nameserver 格式: %v", err)
	}

	return nil
}

// isIPAddress 檢查字符串是否為 IP 地址格式
func (r *DNSResolver) isIPAddress(nameserver string) bool {
	host, _, err := net.SplitHostPort(nameserver)
	if err != nil {
		return false
	}
	return net.ParseIP(host) != nil
}

// doResolve 執行實際的 DNS 解析
func (r *DNSResolver) doResolve(nameserver string) (string, error) {
	host, port, err := net.SplitHostPort(nameserver)
	if err != nil {
		return "", fmt.Errorf("無效的 nameserver 格式: %v", err)
	}

	// 執行 DNS 查詢
	ips, err := net.LookupIP(host)
	if err != nil {
		return "", fmt.Errorf("域名解析失敗 %s: %v", host, err)
	}

	if len(ips) == 0 {
		return "", fmt.Errorf("域名 %s 沒有對應的 IP 地址", host)
	}

	// 優先返回 IPv4 地址
	for _, ip := range ips {
		if ipv4 := ip.To4(); ipv4 != nil {
			return net.JoinHostPort(ipv4.String(), port), nil
		}
	}

	// 如果沒有 IPv6，使用第一個 IPv6
	return net.JoinHostPort(ips[0].String(), port), nil
}

// getCachedResult 獲取快取結果
func (r *DNSResolver) getCachedResult(nameserver string) *resolveResult {
	if !r.config.EnableCache {
		return nil
	}

	r.cacheMutex.RLock()
	defer r.cacheMutex.RUnlock()

	result, exists := r.cache[nameserver]
	if !exists {
		return nil
	}

	// 檢查快取是否過期
	if time.Since(result.resolveTime) > r.config.RefreshInterval {
		return nil
	}

	return &result
}

// setCachedResult 設置快取結果
func (r *DNSResolver) setCachedResult(nameserver, resolved string, err error) {
	if !r.config.EnableCache {
		return
	}

	r.cacheMutex.Lock()
	defer r.cacheMutex.Unlock()

	r.cache[nameserver] = resolveResult{
		resolvedAddress: resolved,
		resolveTime:     time.Now(),
		err:             err,
	}
	r.lastRefresh[nameserver] = time.Now()
}

// ClearCache 清除 DNS 快取
func (r *DNSResolver) ClearCache() {
	if !r.config.EnableCache {
		return
	}

	r.cacheMutex.Lock()
	defer r.cacheMutex.Unlock()

	r.cache = make(map[string]resolveResult)
	r.lastRefresh = make(map[string]time.Time)
}

// GetCacheStats 獲取快取統計信息
func (r *DNSResolver) GetCacheStats() map[string]interface{} {
	if !r.config.EnableCache {
		return map[string]interface{}{
			"cache_enabled": false,
		}
	}

	r.cacheMutex.RLock()
	defer r.cacheMutex.RUnlock()

	return map[string]interface{}{
		"cache_enabled": true,
		"cache_size":    len(r.cache),
		"last_refresh":  r.lastRefresh,
	}
}
