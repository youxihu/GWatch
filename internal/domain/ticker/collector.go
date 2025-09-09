// Package ticker internal/domain/ticker/collector.go
package ticker

import "GWatch/internal/entity"

// TickerCollector 定时器数据收集器接口
type TickerCollector interface {
	// Init 初始化收集器
	Init() error
	
	// CollectDeviceStatus 收集设备状态信息
	CollectDeviceStatus(config entity.TickerHTTPInterface) (*entity.DeviceStatus, error)
	
	// CollectDeviceStatusWithToken 使用指定token收集设备状态信息
	CollectDeviceStatusWithToken(config entity.TickerHTTPInterface, token string) (*entity.DeviceStatus, error)
	
	// CheckInterface 检查接口可用性
	CheckInterface(config entity.TickerHTTPInterface) (bool, error)
}
