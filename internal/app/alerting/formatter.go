package alerting

import (
	"GWatch/internal/domain/alert"
	"GWatch/internal/entity"
	"fmt"
	"time"
)

// BuildMarkdown composes the markdown body based on metrics and events.
func BuildMarkdown(title string, cfg *entity.Config, metrics *entity.SystemMetrics, events []alert.AlertEvent) string {
	text := "## " + title + "\n\n"
	if len(events) > 0 {
		text += "### 触发告警项\n\n"
		for _, ev := range events {
			text += "> " + ev.Message + "\n\n"
		}
	}
	text += "### 完整监控指标\n\n"
	if metrics.CPU.Error != nil {
		text += fmt.Sprintf("**CPU**: 监控失败 - %v\n\n", metrics.CPU.Error)
	} else {
		text += fmt.Sprintf("**CPU**: %.2f%% %s\n\n", metrics.CPU.Percent, status(metrics.CPU.Percent, cfg.Monitor.CPUThreshold))
	}
	if metrics.Memory.Error != nil {
		text += fmt.Sprintf("**内存**: 监控失败 - %v\n\n", metrics.Memory.Error)
	} else {
		text += fmt.Sprintf("**内存**: %.2f%% (%d/%d MB) %s\n\n", metrics.Memory.Percent, metrics.Memory.UsedMB, metrics.Memory.TotalMB, status(metrics.Memory.Percent, cfg.Monitor.MemoryThreshold))
	}
	if metrics.Disk.Error != nil {
		text += fmt.Sprintf("**磁盘**: 监控失败 - %v\n\n", metrics.Disk.Error)
	} else {
		text += fmt.Sprintf("**磁盘**: %.2f%% (%d/%d GB) %s\n\n", metrics.Disk.Percent, metrics.Disk.UsedGB, metrics.Disk.TotalGB, status(metrics.Disk.Percent, cfg.Monitor.DiskThreshold))
	}
	if metrics.Redis.ConnectionError != nil {
		text += fmt.Sprintf("**Redis**: 连接失败 - %v\n\n", metrics.Redis.ConnectionError)
	} else {
		text += fmt.Sprintf("**Redis**: %d个连接 %s\n\n", metrics.Redis.ClientCount, redisStatus(metrics.Redis.ClientCount, cfg))
	}
	if metrics.Network.Error != nil {
		text += fmt.Sprintf("**网络**: 监控失败 - %v\n\n", metrics.Network.Error)
	} else {
		text += fmt.Sprintf("**网络**: 下载 %.2f KB/s | 上传 %.2f KB/s\n\n", metrics.Network.DownloadKBps, metrics.Network.UploadKBps)
	}
	text += fmt.Sprintf("**监控时间**: %s\n\n", metrics.Timestamp.Format(time.DateTime))
	return text
}

func status(value, threshold float64) string {
	if value > threshold { return "[异常]" }
	return "[正常]"
}

func redisStatus(count int, cfg *entity.Config) string {
	if count < cfg.Monitor.RedisMinClients { return "[连接数过低]" }
	if count > cfg.Monitor.RedisMaxClients { return "[连接数过高]" }
	return "[正常]"
}


