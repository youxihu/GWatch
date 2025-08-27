package service

import (
	domaincfg "GWatch/internal/domain/config"
	"context"
	"fmt"
	"net/http"
	"time"
)

// HTTPCollector HTTP接口监控收集器
type HTTPCollector struct {
	provider domaincfg.Provider
	client   *http.Client
}

// NewHTTPCollector 创建新的HTTP接口监控收集器
func NewHTTPCollector(p domaincfg.Provider) *HTTPCollector {
	return &HTTPCollector{provider: p}
}

// Init 初始化HTTP客户端
func (c *HTTPCollector) Init() error {
	cfg := c.provider.GetConfig()
	if cfg == nil {
		return fmt.Errorf("配置未加载")
	}

	// 创建HTTP客户端，设置默认超时时间
	c.client = &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     30 * time.Second,
		},
	}

	return nil
}

// CheckInterface 检查指定HTTP接口的可访问性和响应时间
func (c *HTTPCollector) CheckInterface(url string, timeout time.Duration) (bool, time.Duration, int, error) {
	if c.client == nil {
		return false, 0, 0, fmt.Errorf("HTTP客户端未初始化")
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, 0, 0, fmt.Errorf("创建HTTP请求失败: %v", err)
	}

	// 设置User-Agent避免被某些服务拒绝
	req.Header.Set("User-Agent", "GWatch-Monitor/1.0")

	// 记录开始时间
	start := time.Now()

	// 发送请求
	resp, err := c.client.Do(req)
	if err != nil {
		return false, 0, 0, fmt.Errorf("HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 计算响应时间
	responseTime := time.Since(start)

	// 检查HTTP状态码
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true, responseTime, resp.StatusCode, nil
	}

	return false, responseTime, resp.StatusCode, fmt.Errorf("HTTP状态码异常: %d", resp.StatusCode)
}

// Close 释放资源
func (c *HTTPCollector) Close() {
	if c.client != nil {
		c.client.CloseIdleConnections()
	}
}
