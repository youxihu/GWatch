package monitoring

import (
	"GWatch/internal/domain/monitoring"
	"GWatch/internal/entity"
	"GWatch/internal/utils"
	"fmt"
	"time"
)

type MarkdownFormatter struct{}

func NewMarkdownFormatter() monitoring.Formatter { return &MarkdownFormatter{} }

func (f *MarkdownFormatter) Build(title string, cfg *entity.Config, m *entity.SystemMetrics, alerts []monitoring.TriggeredAlert) string {
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
	
	// 添加主机信息
	ip, hostname, err := utils.GetHostInfo()
	if err != nil {
		text += fmt.Sprintf("**主机IP**: 获取主机信息失败 - %v\n\n", err)
	} else {
		text += fmt.Sprintf("**主机IP**: %s (%s)\n\n", hostname, ip)
	}
	
	// 主机类监控指标 - 只有当host_monitoring配置存在且启用时才显示
	if cfg.HostMonitoring != nil && cfg.HostMonitoring.Enabled {
		if m.CPU.Error != nil {
			text += fmt.Sprintf("**CPU**: 监控失败 - %v\n\n", m.CPU.Error)
		} else {
			text += fmt.Sprintf("**CPU**: %.2f%% %s\n\n", m.CPU.Percent, status(m.CPU.Percent, cfg.HostMonitoring.CPUThreshold))
		}
		if m.Memory.Error != nil {
			text += fmt.Sprintf("**内存**: 监控失败 - %v\n\n", m.Memory.Error)
		} else {
			text += fmt.Sprintf("**内存**: %.2f%% (%d/%d MB) %s\n\n", m.Memory.Percent, m.Memory.UsedMB, m.Memory.TotalMB, status(m.Memory.Percent, cfg.HostMonitoring.MemoryThreshold))
		}
		if m.Disk.Error != nil {
			text += fmt.Sprintf("**磁盘**: 监控失败 - %v\n\n", m.Disk.Error)
		} else {
			text += fmt.Sprintf("**磁盘**: %.2f%% (%d/%d GB) %s\n\n", m.Disk.Percent, m.Disk.UsedGB, m.Disk.TotalGB, status(m.Disk.Percent, cfg.HostMonitoring.DiskThreshold))
		}
		if m.Network.Error != nil {
			text += fmt.Sprintf("**网络IO**: 监控失败 - %v\n\n", m.Network.Error)
		} else {
			text += fmt.Sprintf("**网络IO**: 下载 %.2f KB/s | 上传 %.2f KB/s\n\n", m.Network.DownloadKBps, m.Network.UploadKBps)
		}
		text += fmt.Sprintf("**磁盘IO**: 读 %.2f KB/s | 写 %.2f KB/s\n\n", m.Disk.ReadKBps, m.Disk.WriteKBps)
	}
	
	// Redis监控指标 - 只有当app_monitoring和redis配置存在且启用时才显示
	if cfg.AppMonitoring != nil && cfg.AppMonitoring.Enabled && cfg.AppMonitoring.Redis != nil && cfg.AppMonitoring.Redis.Enabled {
		if m.Redis.ConnectionError != nil {
			text += fmt.Sprintf("**Redis**: 连接失败 - %v\n\n", m.Redis.ConnectionError)
		} else {
			text += fmt.Sprintf("**Redis**: %d个连接 %s\n\n", m.Redis.ClientCount, redisStatus(m.Redis.ClientCount, cfg))
		}
	}

	// MySQL监控指标 - 只有当app_monitoring和mysql配置存在且启用时才显示
	if cfg.AppMonitoring != nil && cfg.AppMonitoring.Enabled && cfg.AppMonitoring.MySQL != nil && cfg.AppMonitoring.MySQL.Enabled {
		if m.MySQL.Error != nil {
			text += fmt.Sprintf("**MySQL**: 连接失败 - %v\n\n", m.MySQL.Error)
		} else {
			text += fmt.Sprintf("**MySQL**: %d/%d连接 (%.2f%%) %s\n\n", 
				m.MySQL.Connections.ThreadsConnected,
				m.MySQL.Connections.MaxConnections,
				m.MySQL.Connections.ConnectionUsage,
				mysqlConnectionStatus(m.MySQL.Connections.ConnectionUsage, cfg.AppMonitoring.MySQL.Thresholds.MaxConnectionsUsageWarning))
			text += fmt.Sprintf("**MySQL QPS**: %d\n\n", 
				m.MySQL.QueryPerformance.QPS)
			text += fmt.Sprintf("**MySQL Buffer Pool**: %.2f%%命中率 %s\n\n", 
				m.MySQL.BufferPool.HitRate,
				mysqlBufferStatus(m.MySQL.BufferPool.HitRate, cfg.AppMonitoring.MySQL.Thresholds.BufferPoolHitRateWarning))
		}
	}
	// HTTP接口监控信息 - 只有当app_monitoring和http配置存在且启用时才显示
	if cfg.AppMonitoring != nil && cfg.AppMonitoring.Enabled && cfg.AppMonitoring.HTTP != nil && cfg.AppMonitoring.HTTP.Enabled {
		if m.HTTP.Error != nil {
			text += fmt.Sprintf("**HTTP接口**: 监控失败 - %v\n\n", m.HTTP.Error)
		} else if len(m.HTTP.Interfaces) > 0 {
			text += "**HTTP接口**:\n\n"
			for _, httpInterface := range m.HTTP.Interfaces {
		
				// 检查状态码是否在允许的范围内
				isValidCode := false
				if len(httpInterface.AllowedCodes) > 0 {
					for _, allowedCode := range httpInterface.AllowedCodes {
						if httpInterface.StatusCode == allowedCode {
							isValidCode = true
							break
						}
					}
				} else {
					// 如果没有配置allowed_codes，默认只允许200
					isValidCode = (httpInterface.StatusCode == 200)
				}
				
				if isValidCode {
					text += fmt.Sprintf("- %s: 正常 (状态码: %d, 响应时间: %v)\n", 
						httpInterface.Name,  httpInterface.StatusCode, httpInterface.ResponseTime)
				} else {
					text += fmt.Sprintf("- %s: 异常 (状态码: %d) - %v\n", 
						httpInterface.Name,  httpInterface.StatusCode, httpInterface.Error)
				}
			}
			text += "\n"
		}
	}

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
	if cfg.AppMonitoring != nil && cfg.AppMonitoring.Enabled && cfg.AppMonitoring.Redis != nil && cfg.AppMonitoring.Redis.Enabled {
		if count < cfg.AppMonitoring.Redis.MinClients {
			return "[连接数过低]"
		}
		if count > cfg.AppMonitoring.Redis.MaxClients {
			return "[连接数过高]"
		}
	}
	return "[正常]"
}

// MySQL状态判断函数
func mysqlConnectionStatus(usage float64, threshold float64) string {
	if usage > threshold {
		return "[连接数过高]"
	}
	return "[正常]"
}

func mysqlBufferStatus(hitRate float64, threshold float64) string {
	if hitRate < threshold {
		return "[命中率过低]"
	}
	return "[正常]"
}
