// Package scheduled_push internal/domain/scheduled_push/formatter.go
package scheduled_push

import "GWatch/internal/entity"

// ScheduledPushFormatter 定时推送格式化器接口
type ScheduledPushFormatter interface {
	// FormatClientReport 格式化合并后的客户端报告（按照 dingformat.md 的格式）
	// title: 配置的title，用于每个主机的二级标题（#### {title}）
	FormatClientReport(data []*entity.ClientMonitorData, title string) string
}
