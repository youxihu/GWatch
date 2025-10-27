package monitoring

import (
	"GWatch/internal/domain/monitoring"
	"GWatch/internal/entity"
	"GWatch/internal/utils"
	"fmt"
)

type TickerMarkdownFormatter struct {
	*MarkdownFormatter // 嵌入原有的MarkdownFormatter
}

func NewTickerMarkdownFormatter() monitoring.Formatter {
	return &TickerMarkdownFormatter{
		MarkdownFormatter: &MarkdownFormatter{},
	}
}

// 确保TickerMarkdownFormatter同时实现TickerFormatter接口
var _ monitoring.TickerFormatter = (*TickerMarkdownFormatter)(nil)

func (f *TickerMarkdownFormatter) Build(title string, cfg *entity.Config, m *entity.SystemMetrics, alerts []monitoring.TriggeredAlert) string {
	// 直接使用父类的Build方法
	return f.MarkdownFormatter.Build(title, cfg, m, alerts)
}

// 添加一个专门用于ticker报告的方法
func (f *TickerMarkdownFormatter) BuildTickerReport(title string, cfg *entity.Config, tickerMetrics *entity.TickerMetrics, systemMetrics *entity.SystemMetrics) string {
	text := "## " + title + "\n\n"
	text += "### 完整监控指标\n\n"

	// 添加主机信息
	ip, hostname, err := utils.GetHostInfo()
	if err != nil {
		text += fmt.Sprintf("**监控主机**: 获取主机信息失败 - %v\n\n", err)
	} else {
		text += fmt.Sprintf("**监控主机**: %s (%s)\n\n", hostname, ip)
	}

	// 生成系统监控指标
	text += f.buildSystemMetrics(systemMetrics, cfg)

	// 添加HTTP接口状态
	text += f.buildHTTPInterfaces(systemMetrics)

	// 👇👇👇 专业简洁错误提示 👇👇👇
	text += "### 设备状态概览\n\n"

	if len(tickerMetrics.Interfaces) == 0 {
		text += "未配置任何设备状态接口\n\n"
	} else {
		var hasError bool
		var firstError error
		var errType entity.ErrorType

		for _, iface := range tickerMetrics.Interfaces {
			if !iface.IsAccessible {
				hasError = true
				firstError = iface.Error
				errType = iface.ErrorType
				break
			}
		}

		if hasError {
			switch errType {
			case entity.ErrorTypeToken:
				text += "**错误类型**：Token 已过期\n\n"
				if firstError != nil {
					text += fmt.Sprintf("**错误详情**：%v\n\n", firstError.Error())
				}

			case entity.ErrorTypeUnauthorized:
				text += "**错误类型**：认证失败（401/403）\n\n"
				if firstError != nil {
					text += fmt.Sprintf("**错误详情**：%v\n\n", firstError.Error())
				}

			case entity.ErrorTypeNetwork:
				text += "**错误类型**：网络连接失败\n\n"
				if firstError != nil {
					text += fmt.Sprintf("**错误详情**：%v\n\n", firstError.Error())
				}

			case entity.ErrorTypeServer:
				text += "**错误类型**：服务端异常（5xx）\n\n"
				if firstError != nil {
					text += fmt.Sprintf("**错误详情**：%v\n\n", firstError.Error())
				}

			default:
				text += "**错误类型**：未知错误\n\n"
				if firstError != nil {
					text += fmt.Sprintf("**错误详情**：%v\n\n", firstError.Error())
				}
			}
			text += "\n" // 保证段落间距
		} else {
			// 找第一个可访问的接口显示数据
			var validStatus *entity.TickerInterfaceMetrics
			for i := range tickerMetrics.Interfaces {
				if tickerMetrics.Interfaces[i].IsAccessible {
					validStatus = &tickerMetrics.Interfaces[i]
					break
				}
			}

			if validStatus != nil {
				text += fmt.Sprintf("- **在线设备**: %d 台\n\n", validStatus.ChannelOnLineNumber)
				text += fmt.Sprintf("- **离线设备**: %d 台\n\n", validStatus.ChannelOffLineNumber)
				text += fmt.Sprintf("- **总设备数**: %d 台\n\n", validStatus.TotalDevices)
				text += fmt.Sprintf("- **在线率**: %.2f%%\n\n", validStatus.OnlineRate)
			} else {
				text += "设备状态获取失败（无可用接口）\n"
			}
			text += "\n"
		}
	}

	// 添加监控时间
	text += fmt.Sprintf("**监控时间**：%s", tickerMetrics.Timestamp.Format("2006-01-02 15:04:05"))

	return text
}

// buildSystemMetrics 构建系统监控指标（复用父类逻辑）
func (f *TickerMarkdownFormatter) buildSystemMetrics(m *entity.SystemMetrics, cfg *entity.Config) string {
	text := ""

	// 主机类监控指标 - 只有当host_monitoring配置存在时才显示
	if cfg.HostMonitoring != nil {
		// CPU状态
		text += fmt.Sprintf("**CPU**: %.2f%% %s\n\n", m.CPU.Percent, status(m.CPU.Percent, cfg.HostMonitoring.CPUThreshold))

		// 内存状态
		text += fmt.Sprintf("**内存**: %.2f%% (%d/%d MB) %s\n\n", m.Memory.Percent, m.Memory.UsedMB, m.Memory.TotalMB, status(m.Memory.Percent, cfg.HostMonitoring.MemoryThreshold))

		// 磁盘状态
		text += fmt.Sprintf("**磁盘**: %.2f%% (%d/%d GB) %s\n\n", m.Disk.Percent, m.Disk.UsedGB, m.Disk.TotalGB, status(m.Disk.Percent, cfg.HostMonitoring.DiskThreshold))

		// 网络IO
		text += fmt.Sprintf("**网络IO**: 下载 %.2f KB/s | 上传 %.2f KB/s\n\n", m.Network.DownloadKBps, m.Network.UploadKBps)

		// 磁盘IO
		text += fmt.Sprintf("**磁盘IO**: 读 %.2f KB/s | 写 %.2f KB/s\n\n", m.Disk.ReadKBps, m.Disk.WriteKBps)
	}

	// Redis状态 - 只有当app_monitoring和redis配置存在时才显示
	if cfg.AppMonitoring != nil && cfg.AppMonitoring.Redis != nil {
		text += fmt.Sprintf("**Redis**: %d个连接 %s\n\n", m.Redis.ClientCount, redisStatus(m.Redis.ClientCount, cfg))
	}

	return text
}

// buildHTTPInterfaces 构建HTTP接口状态
func (f *TickerMarkdownFormatter) buildHTTPInterfaces(m *entity.SystemMetrics) string {
	text := ""

	// HTTP接口监控信息 - 只有当app_monitoring和http配置存在时才显示
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
				text += fmt.Sprintf("- **%s**: 正常 (状态码: %d, 响应时间: %v)\n\n",
					httpInterface.Name, httpInterface.StatusCode, httpInterface.ResponseTime)
			} else {
				text += fmt.Sprintf("- **%s**: 异常 (状态码: %d) - %v\n\n",
					httpInterface.Name, httpInterface.StatusCode, httpInterface.Error)
			}
		}
	}

	return text
}
