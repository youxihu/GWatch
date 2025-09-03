package alertimpl

import (
	"GWatch/internal/domain/alert"
	"GWatch/internal/entity"
	"fmt"
)

type TickerMarkdownFormatter struct {
	*MarkdownFormatter // 嵌入原有的MarkdownFormatter
}

func NewTickerMarkdownFormatter() alert.Formatter { 
	return &TickerMarkdownFormatter{
		MarkdownFormatter: &MarkdownFormatter{},
	}
}

// 确保TickerMarkdownFormatter同时实现TickerFormatter接口
var _ alert.TickerFormatter = (*TickerMarkdownFormatter)(nil)

func (f *TickerMarkdownFormatter) Build(title string, cfg *entity.Config, m *entity.SystemMetrics, alerts []alert.TriggeredAlert) string {
	// 直接使用父类的Build方法
	return f.MarkdownFormatter.Build(title, cfg, m, alerts)
}

// 添加一个专门用于ticker报告的方法
func (f *TickerMarkdownFormatter) BuildTickerReport(title string, cfg *entity.Config, tickerMetrics *entity.TickerMetrics, systemMetrics *entity.SystemMetrics) string {
	text := "## " + title + "\n\n"
	text += "### 完整监控指标\n\n"
	
	// 生成系统监控指标
	text += f.buildSystemMetrics(systemMetrics, cfg)
	
	// 添加HTTP接口状态
	text += f.buildHTTPInterfaces(systemMetrics)
	
	// 添加设备状态概览
	if tickerMetrics.DeviceStatus != nil {
		deviceStatus := tickerMetrics.DeviceStatus
		text += "### 设备状态概览\n\n"
		text += fmt.Sprintf("- **在线设备**: %d 台\n\n", deviceStatus.ChannelOnLineNumber)
		text += fmt.Sprintf("- **离线设备**: %d 台\n\n", deviceStatus.ChannelOffLineNumber)
		text += fmt.Sprintf("- **总设备数**: %d 台\n\n", deviceStatus.TotalDevices)
		text += fmt.Sprintf("- **在线率**: %.2f%%\n\n", deviceStatus.OnlineRate)
	}
	
	// 添加监控时间
	text += fmt.Sprintf("**监控时间**: %s", tickerMetrics.Timestamp.Format("2006-01-02 15:04:05"))
	
	return text
}

// buildSystemMetrics 构建系统监控指标（复用父类逻辑）
func (f *TickerMarkdownFormatter) buildSystemMetrics(m *entity.SystemMetrics, cfg *entity.Config) string {
	text := ""
	
	// CPU状态
	text += fmt.Sprintf("**CPU**: %.2f%% %s\n\n", m.CPU.Percent, status(m.CPU.Percent, cfg.Monitor.CPUThreshold))
	
	// 内存状态
	text += fmt.Sprintf("**内存**: %.2f%% (%d/%d MB) %s\n\n", m.Memory.Percent, m.Memory.UsedMB, m.Memory.TotalMB, status(m.Memory.Percent, cfg.Monitor.MemoryThreshold))
	
	// 磁盘状态
	text += fmt.Sprintf("**磁盘**: %.2f%% (%d/%d GB) %s\n\n", m.Disk.Percent, m.Disk.UsedGB, m.Disk.TotalGB, status(m.Disk.Percent, cfg.Monitor.DiskThreshold))
	
	// Redis状态
	text += fmt.Sprintf("**Redis**: %d个连接 %s\n\n", m.Redis.ClientCount, redisStatus(m.Redis.ClientCount, cfg))
	
	// 网络IO
	text += fmt.Sprintf("**网络IO**: 下载 %.2f KB/s | 上传 %.2f KB/s\n\n", m.Network.DownloadKBps, m.Network.UploadKBps)
	
	// 磁盘IO
	text += fmt.Sprintf("**磁盘IO**: 读 %.2f KB/s | 写 %.2f KB/s\n\n", m.Disk.ReadKBps, m.Disk.WriteKBps)
	
	return text
}

// buildHTTPInterfaces 构建HTTP接口状态
func (f *TickerMarkdownFormatter) buildHTTPInterfaces(m *entity.SystemMetrics) string {
	text := ""
	
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



