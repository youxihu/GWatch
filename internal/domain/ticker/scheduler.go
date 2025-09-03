// Package ticker internal/domain/ticker/scheduler.go
package ticker

import (
	"GWatch/internal/entity"
)

// TickerScheduler 定时器调度器接口
type TickerScheduler interface {
	// Start 启动定时器调度
	Start(config *entity.Config, stopCh <-chan struct{}) error
	
	// Stop 停止定时器调度
	Stop() error
	
	// IsTimeToAlert 检查是否到了告警时间
	IsTimeToAlert(alertTimes []string) bool
}

// TickerUseCase 定时器用例接口
type TickerUseCase interface {
	// RunTickerReport 执行定时器报告
	RunTickerReport(config *entity.Config) error
	
	// CollectTickerMetrics 收集定时器指标
	CollectTickerMetrics(config *entity.Config) (*entity.TickerMetrics, error)
}
