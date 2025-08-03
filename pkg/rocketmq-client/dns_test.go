package rocketmqclient

import (
	"testing"
	"time"
)

// 測試 DNS 解析器基本功能
func TestDNSResolver_ResolveName(t *testing.T) {
	// 創建 DNS 解析器
	config := DefaultDNSConfig()
	resolver := NewDNSResolver(config)

	// 測試 IP 地址格式（應該直接通過）
	testCases := []struct {
		name     string
		input    string
		expected bool // true 表示期望成功
	}{
		{
			name:     "IP地址格式",
			input:    "192.168.1.100:9876",
			expected: true,
		},
		{
			name:     "無效格式_缺少端口",
			input:    "192.168.1.100",
			expected: false,
		},
		{
			name:     "空字符串",
			input:    "",
			expected: false,
		},
		{
			name:     "localhost域名",
			input:    "localhost:9876",
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := resolver.ResolveName(tc.input)

			if tc.expected {
				if err != nil {
					t.Errorf("期望成功，但出現錯誤: %v", err)
				}
				if result == "" {
					t.Errorf("期望返回非空結果")
				}
				t.Logf("解析結果: %s -> %s", tc.input, result)
			} else {
				if err == nil {
					t.Errorf("期望失敗，但成功了: %s", result)
				}
				t.Logf("預期錯誤: %v", err)
			}
		})
	}
}

// 測試 DNS 解析器重試功能
func TestDNSResolver_ResolveWithRetry(t *testing.T) {
	config := &DNSConfig{
		MaxRetries:    2,
		RetryInterval: 100 * time.Millisecond,
		EnableCache:   false, // 關閉快取以測試重試
	}
	resolver := NewDNSResolver(config)

	// 測試正常域名
	result, err := resolver.ResolveWithRetry("localhost:9876")
	if err != nil {
		t.Errorf("localhost 解析失敗: %v", err)
	} else {
		t.Logf("localhost 解析成功: %s", result)
	}
}

// 測試 DNS 快取功能
func TestDNSResolver_Cache(t *testing.T) {
	config := &DNSConfig{
		EnableCache:     true,
		RefreshInterval: 1 * time.Second,
	}
	resolver := NewDNSResolver(config)

	// 第一次解析
	result1, err1 := resolver.ResolveName("localhost:9876")
	if err1 != nil {
		t.Fatalf("第一次解析失敗: %v", err1)
	}

	// 第二次解析（應該使用快取）
	result2, err2 := resolver.ResolveName("localhost:9876")
	if err2 != nil {
		t.Fatalf("第二次解析失敗: %v", err2)
	}

	if result1 != result2 {
		t.Errorf("快取結果不一致: %s != %s", result1, result2)
	}

	// 檢查快取統計
	stats := resolver.GetCacheStats()
	if !stats["cache_enabled"].(bool) {
		t.Errorf("快取應該是啟用的")
	}

	t.Logf("DNS 快取統計: %+v", stats)
}

// 測試多個 NameServer 解析
func TestDNSResolver_ResolveNameServers(t *testing.T) {
	resolver := NewDNSResolver(DefaultDNSConfig())

	nameservers := []string{
		"localhost:9876",
		"127.0.0.1:9876",
	}

	resolved, err := resolver.ResolveNameServers(nameservers)
	if err != nil {
		t.Fatalf("解析多個 nameserver 失敗: %v", err)
	}

	if len(resolved) == 0 {
		t.Errorf("應該至少解析出一個 nameserver")
	}

	t.Logf("原始: %v", nameservers)
	t.Logf("解析: %v", resolved)
}

// 測試配置預設值
func TestDefaultConfigs(t *testing.T) {
	dnsConfig := DefaultDNSConfig()
	if dnsConfig == nil {
		t.Error("DefaultDNSConfig 不應該返回 nil")
	}

	if dnsConfig.MaxRetries == 0 {
		t.Error("MaxRetries 應該有預設值")
	}

	if dnsConfig.ResolveTimeout == 0 {
		t.Error("ResolveTimeout 應該有預設值")
	}

	t.Logf("預設 DNS 配置: %+v", dnsConfig)
}
