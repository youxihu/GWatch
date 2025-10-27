// Package scheduled_push internal/domain/scheduled_push/alert_storage.go
package scheduled_push

import (
	"GWatch/internal/entity"
	"time"
)

// ScheduledPushAlertStorage 全局定时推送告警存储接口（领域服务接口）
type ScheduledPushAlertStorage interface {
	// SaveScheduledPushAlert 保存全局定时推送告警信息
	SaveScheduledPushAlert(alert *entity.ScheduledPushAlertRecord) error
	
	// GetScheduledPushAlerts 获取全局定时推送告警信息
	GetScheduledPushAlerts(startTime, endTime time.Time) ([]*entity.ScheduledPushAlertRecord, error)
	
	// CleanupOldAlerts 清理过期告警信息
	CleanupOldAlerts() error
}
