// Package common internal/domain/scheduled_push/common/formatter.go
package common

import "GWatch/internal/entity"

// ScheduledPushFormatter 定时推送格式化器接口
type ScheduledPushFormatter interface {
	// FormatClientReport 格式化合并后的客户端报告（按照 dingformat.md 的格式）
	// title: 配置的title，用于每个主机的二级标题（#### {title}）
	// config: 配置信息，用于判断阈值和状态
	FormatClientReport(data []*entity.ClientMonitorData, title string, config *entity.Config) string
}
