package alertimpl

import (
	"GWatch/internal/domain/alert"
	"GWatch/internal/entity"
	"fmt"
	"time"
)

type MarkdownFormatter struct{}

func NewMarkdownFormatter() alert.Formatter { return &MarkdownFormatter{} }

func (f *MarkdownFormatter) Build(title string, cfg *entity.Config, m *entity.SystemMetrics, alerts []alert.TriggeredAlert) string {
	text := "## " + title + "\n\n"
	if len(alerts) > 0 {
		text += "### 触发告警项\n\n"
		for _, a := range alerts {
			line := a.Message
			if line == "" {
				line = a.Type.String()
			}
			text += "> " + line + "\n\n"
		}
	}
	text += "### 完整监控指标\n\n"
	if m.CPU.Error != nil {
		text += fmt.Sprintf("**CPU**: 监控失败 - %v\n\n", m.CPU.Error)
	} else {
		text += fmt.Sprintf("**CPU**: %.2f%% %s\n\n", m.CPU.Percent, status(m.CPU.Percent, cfg.Monitor.CPUThreshold))
	}
	if m.Memory.Error != nil {
		text += fmt.Sprintf("**内存**: 监控失败 - %v\n\n", m.Memory.Error)
	} else {
		text += fmt.Sprintf("**内存**: %.2f%% (%d/%d MB) %s\n\n", m.Memory.Percent, m.Memory.UsedMB, m.Memory.TotalMB, status(m.Memory.Percent, cfg.Monitor.MemoryThreshold))
	}
	if m.Disk.Error != nil {
		text += fmt.Sprintf("**磁盘**: 监控失败 - %v\n\n", m.Disk.Error)
	} else {
		text += fmt.Sprintf("**磁盘**: %.2f%% (%d/%d GB) %s\n\n", m.Disk.Percent, m.Disk.UsedGB, m.Disk.TotalGB, status(m.Disk.Percent, cfg.Monitor.DiskThreshold))
	}
	if m.Redis.ConnectionError != nil {
		text += fmt.Sprintf("**Redis**: 连接失败 - %v\n\n", m.Redis.ConnectionError)
	} else {
		text += fmt.Sprintf("**Redis**: %d个连接 %s\n\n", m.Redis.ClientCount, redisStatus(m.Redis.ClientCount, cfg))
	}
	if m.Network.Error != nil {
		text += fmt.Sprintf("**网络IO**: 监控失败 - %v\n\n", m.Network.Error)
	} else {
		text += fmt.Sprintf("**网络IO**: 下载 %.2f KB/s | 上传 %.2f KB/s\n\n", m.Network.DownloadKBps, m.Network.UploadKBps)
	}
	text += fmt.Sprintf("**磁盘IO**: 读 %.2f KB/s | 写 %.2f KB/s\n\n", m.Disk.ReadKBps, m.Disk.WriteKBps)
	text += fmt.Sprintf("**监控时间**: %s\n\n", m.Timestamp.Format(time.DateTime))
	return text
}

func status(value, threshold float64) string {
	if value > threshold {
		return "[异常]"
	}
	return "[正常]"
}
func redisStatus(count int, cfg *entity.Config) string {
	if count < cfg.Monitor.RedisMinClients {
		return "[连接数过低]"
	}
	if count > cfg.Monitor.RedisMaxClients {
		return "[连接数过高]"
	}
	return "[正常]"
}
