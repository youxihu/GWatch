// Package entity internal/entity/device_status.go
package entity

import "time"

// DeviceStatus 设备状态信息
type DeviceStatus struct {
	Timestamp            time.Time `json:"timestamp"`
	ChannelOffLineNumber int       `json:"channelOffLineNumber"`
	ChannelOnLineNumber  int       `json:"channelOnLineNumber"`
	TotalDevices         int       `json:"totalDevices"`
	OnlineRate           float64   `json:"onlineRate"`
}

// TickerMetrics 定时器指标
type TickerMetrics struct {
	Timestamp    time.Time                `json:"timestamp"`
	DeviceStatus *DeviceStatus            `json:"deviceStatus"`
	Interfaces   []TickerInterfaceMetrics `json:"interfaces"`
}

// TickerInterfaceMetrics 定时器接口指标
type TickerInterfaceMetrics struct {
	Name                 string        `json:"name"`
	URL                  string        `json:"url"`
	IsAccessible         bool          `json:"isAccessible"`
	ResponseTime         time.Duration `json:"responseTime"`
	StatusCode           int           `json:"statusCode"`
	Error                error         `json:"error,omitempty"`
	ErrorType            ErrorType     `json:"errorType"`
	ChannelOffLineNumber int           `json:"channelOffLineNumber"`
	ChannelOnLineNumber  int           `json:"channelOnLineNumber"`
	TotalDevices         int           `json:"totalDevices"`
	OnlineRate           float64       `json:"onlineRate"`
}

// TickerResponse 定时器接口响应结构
type TickerResponse struct {
	Msg  string `json:"msg"`
	Code int    `json:"code"`
	Data []struct {
		ChannelOffLineNumber int `json:"channelOffLineNumber"`
		ChannelOnLineNumber  int `json:"channelOnLineNumber"`
	} `json:"data"`
}
