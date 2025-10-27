// Package scheduled_push internal/domain/scheduled_push/scheduler.go
package scheduled_push

import (
	"GWatch/internal/entity"
)

// ScheduledPushScheduler 全局定时推送调度器接口
type ScheduledPushScheduler interface {
	// Start 启动全局定时推送调度
	Start(config *entity.Config, stopCh <-chan struct{}) error
	
	// Stop 停止全局定时推送调度
	Stop() error
	
	// IsTimeToPush 检查是否到了推送时间
	IsTimeToPush(pushTimes []string) bool
}

// ScheduledPushUseCase 全局定时推送用例接口
type ScheduledPushUseCase interface {
	// RunScheduledPush 执行全局定时推送
	RunScheduledPush(config *entity.Config) error
	
	// CollectAllMetrics 收集所有监控指标
	CollectAllMetrics(config *entity.Config) (*entity.ScheduledPushMetrics, error)
}
