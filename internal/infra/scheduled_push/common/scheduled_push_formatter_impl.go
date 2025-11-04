// Package common internal/infra/scheduled_push/common/scheduled_push_formatter_impl.go
package common

import (
	"GWatch/internal/domain/scheduled_push/common"
	"GWatch/internal/entity"
	"fmt"
	"strings"
	"time"
)

// ScheduledPushFormatterImpl 定时推送格式化器实现
type ScheduledPushFormatterImpl struct{}

// NewScheduledPushFormatter 创建定时推送格式化器
func NewScheduledPushFormatter() common.ScheduledPushFormatter {
	return &ScheduledPushFormatterImpl{}
}

// FormatClientReport 格式化合并后的客户端报告（按照 dingformat.md 的格式）
// 只显示已开启的监控项，如果没有开启就取消该栏
// title: 配置的title，用于每个主机的二级标题（#### {title}）
func (f *ScheduledPushFormatterImpl) FormatClientReport(data []*entity.ClientMonitorData, title string) string {
	if len(data) == 0 {
		return "暂无监控数据"
	}

	var sb strings.Builder

	// 报告内容的大标题（固定为"定时性能监控报告"，不使用配置的title）
	// 注意：通知标题（钉钉消息标题）使用的是config.ScheduledPush.Title，而报告内容的大标题固定为这个
	sb.WriteString("### 定时性能监控报告\n")

	// 遍历所有客户端数据
	for i, clientData := range data {
		metrics := clientData.Metrics
		if metrics == nil {
			continue
		}

		// 每个主机的二级标题（优先使用真实主机名，如果没有则使用IP，最后使用配置的title作为后备）
		hostTitle := clientData.HostName
		if hostTitle == "" || hostTitle == "unknown-host" {
			hostTitle = clientData.HostIP
		}
		// 如果主机名和IP都没有，使用配置的title作为后备
		if hostTitle == "" {
			hostTitle = clientData.Title
			if hostTitle == "" {
				hostTitle = title
			}
		}
		sb.WriteString(fmt.Sprintf("#### %s\n", hostTitle))
		
		// 主机信息（显示IP和主机名，如果主机名可用）
		if clientData.HostName != "" && clientData.HostName != "unknown-host" {
			sb.WriteString(fmt.Sprintf("主机IP: %s (%s)\n", clientData.HostIP, clientData.HostName))
		} else {
			sb.WriteString(fmt.Sprintf("主机IP: %s\n", clientData.HostIP))
		}

		// CPU 信息（必须有）
		if metrics.CPU != nil && metrics.CPU.Error == nil {
			sb.WriteString(fmt.Sprintf("- CPU: %.2f%% [正常]\n", metrics.CPU.Percent))
		}

		// 内存信息（必须有）
		if metrics.Memory != nil && metrics.Memory.Error == nil {
			sb.WriteString(fmt.Sprintf("- 内存: %.2f%% (%d/%d MB) [正常]\n",
				metrics.Memory.Percent, metrics.Memory.UsedMB, metrics.Memory.TotalMB))
		}

		// 磁盘信息（必须有）
		if metrics.Disk != nil && metrics.Disk.Error == nil {
			sb.WriteString(fmt.Sprintf("- 磁盘: %.2f%% (%d/%d GB) [正常]\n",
				metrics.Disk.Percent, metrics.Disk.UsedGB, metrics.Disk.TotalGB))
		}

		// 网络IO（必须有）
		if metrics.Network != nil && metrics.Network.Error == nil {
			sb.WriteString(fmt.Sprintf("- 网络IO: 下载 %.2f KB/s | 上传 %.2f KB/s\n",
				metrics.Network.DownloadKBps, metrics.Network.UploadKBps))
		}

		// 磁盘IO（必须有）
		if metrics.Disk != nil && metrics.Disk.Error == nil {
			sb.WriteString(fmt.Sprintf("- 磁盘IO: 读 %.2f KB/s | 写 %.2f KB/s\n",
				metrics.Disk.ReadKBps, metrics.Disk.WriteKBps))
		}

		// Redis（如果开启才显示）
		if metrics.Redis != nil && metrics.Redis.ConnectionError == nil {
			sb.WriteString(fmt.Sprintf("- Redis: %d个连接 [正常]\n", metrics.Redis.ClientCount))
		}

		// MySQL（如果开启才显示）
		if metrics.MySQL != nil && metrics.MySQL.Error == nil {
			// MySQL 暂时没有显示具体指标，可以后续添加
			sb.WriteString("- MySQL: [正常]\n")
		}

		// HTTP接口（如果开启才显示，只显示正常的接口）
		if metrics.HTTP != nil && metrics.HTTP.Error == nil && len(metrics.HTTP.Interfaces) > 0 {
			// 过滤出正常的接口
			validInterfaces := []entity.HTTPInterfaceMetrics{}
			for _, iface := range metrics.HTTP.Interfaces {
				if iface.Error == nil && iface.IsAccessible {
					validInterfaces = append(validInterfaces, iface)
				}
			}
			// 只有当有正常接口时才显示HTTP接口这一栏
			if len(validInterfaces) > 0 {
				sb.WriteString("- HTTP接口:\n")
				for _, iface := range validInterfaces {
					// 响应时间转换为毫秒，保留小数
					responseTimeMs := float64(iface.ResponseTime.Nanoseconds()) / 1e6
					sb.WriteString(fmt.Sprintf("    - %s: 正常 (状态码: %d, 响应时间: %.6fms)\n",
						iface.Name, iface.StatusCode, responseTimeMs))
				}
			}
		}

		// 如果不是最后一个，添加分隔符
		if i < len(data)-1 {
			sb.WriteString("---\n")
		}
	}

	// 监控时间（使用最后一个数据的时间戳，或当前时间）
	var timestamp time.Time
	if len(data) > 0 {
		timestamp = data[len(data)-1].Timestamp
	} else {
		timestamp = time.Now()
	}
	reportTime := timestamp.Format("2006-01-02 15:04:05")
	sb.WriteString(fmt.Sprintf("---\n监控时间: %s\n", reportTime))

	return sb.String()
}
