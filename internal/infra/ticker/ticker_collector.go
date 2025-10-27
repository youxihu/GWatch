// Package ticker internal/infra/collectors/ticker/ticker_collector.go
package ticker

import (
	"GWatch/internal/domain/ticker"
	"GWatch/internal/entity"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// TickerCollectorImpl 定时器收集器实现
type TickerCollectorImpl struct {
	client        *http.Client
	tokenProvider ticker.TokenProvider
}

// NewTickerCollector 创建定时器收集器
func NewTickerCollector(tokenProvider ticker.TokenProvider) ticker.TickerCollector {
	return &TickerCollectorImpl{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		tokenProvider: tokenProvider,
	}
}

// Init 初始化收集器
func (tc *TickerCollectorImpl) Init() error {
	// 这里可以添加初始化逻辑，比如验证网络连接等
	return nil
}


// CollectDeviceStatus 收集设备状态信息
func (tc *TickerCollectorImpl) CollectDeviceStatus(config entity.TickerHTTPInterface) (*entity.DeviceStatus, error) {
	// 获取认证token
	token, err := tc.tokenProvider.GetToken(config.Auth)
	if err != nil {
		return nil, fmt.Errorf("获取认证token失败: %v", err)
	}

	return tc.CollectDeviceStatusWithToken(config, token)
}

// CollectDeviceStatusWithToken 使用指定token收集设备状态信息
func (tc *TickerCollectorImpl) CollectDeviceStatusWithToken(config entity.TickerHTTPInterface, token string) (*entity.DeviceStatus, error) {
	req, err := http.NewRequest("GET", config.DeviceURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	if token != "" {
		req.Header.Set("Authorization", token)
	}

	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := tc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 解析响应
	var tickerResp entity.TickerResponse
	if err := json.Unmarshal(body, &tickerResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	// 检查响应状态
	if tickerResp.Code != 200 {
		return nil, fmt.Errorf("接口返回错误: code=%d, msg=%s", tickerResp.Code, tickerResp.Msg)
	}

	// 计算设备状态
	if len(tickerResp.Data) == 0 {
		return nil, fmt.Errorf("响应数据为空")
	}

	data := tickerResp.Data[0]
	totalDevices := data.ChannelOnLineNumber + data.ChannelOffLineNumber
	var onlineRate float64
	if totalDevices > 0 {
		onlineRate = float64(data.ChannelOnLineNumber) / float64(totalDevices) * 100
	}

	return &entity.DeviceStatus{
		Timestamp:            time.Now(),
		ChannelOffLineNumber: data.ChannelOffLineNumber,
		ChannelOnLineNumber:  data.ChannelOnLineNumber,
		TotalDevices:         totalDevices,
		OnlineRate:           onlineRate,
	}, nil
}

// CheckInterface 检查接口可用性
func (tc *TickerCollectorImpl) CheckInterface(config entity.TickerHTTPInterface) (bool, error) {
	// 获取认证token
	token, err := tc.tokenProvider.GetToken(config.Auth)
	if err != nil {
		return false, fmt.Errorf("获取认证token失败: %v", err)
	}

	req, err := http.NewRequest("GET", config.DeviceURL, nil)
	if err != nil {
		return false, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	if token != "" {
		req.Header.Set("Authorization", token)
	}

	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := tc.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查状态码
	return resp.StatusCode == 200, nil
}
