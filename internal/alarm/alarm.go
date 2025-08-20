// internal/alarm/detect.go
package alarm

import (
	"GWatch/internal/config"
	"GWatch/internal/entity"
	"GWatch/internal/monitor"
	"GWatch/internal/utils"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/youxihu/dingtalk/dingtalk"
)

// 各类报警的状态（线程安全）
var (
	lastAlertTimes = make(map[string]time.Time) // 最后报警时间（防抖）
	alertCounters  = make(map[string]int)       // 连续超标次数
	mu             sync.RWMutex
)

// shouldTriggerAlert 判断是否连续超标 N 次才触发
func shouldTriggerAlert(alertType string, isAlarming bool, consecutive int) bool {
	mu.Lock()
	defer mu.Unlock()

	if !isAlarming {
		alertCounters[alertType] = 0
		return false
	}

	alertCounters[alertType]++
	return alertCounters[alertType] >= consecutive
}

// isAlertAllowed 判断是否允许发送（基于时间间隔防抖）
func isAlertAllowed(alertType string) bool {
	cfg := config.GetConfig()
	if cfg == nil {
		log.Printf("[WARNING] 配置未加载，跳过防抖检查 alertType=%s", alertType)
		return true
	}

	interval := cfg.Monitor.AlertInterval

	mu.RLock()
	lastTime, exists := lastAlertTimes[alertType]
	mu.RUnlock()

	if !exists || time.Since(lastTime) >= interval {
		mu.Lock()
		lastAlertTimes[alertType] = time.Now()
		mu.Unlock()
		debugLog("报警允许触发 alertType=%s", alertType)
		return true
	}
	debugLog("报警被防抖 alertType=%s", alertType)
	return false
}

// getCounter 获取连续超标计数
func getCounter(key string) int {
	mu.RLock()
	defer mu.RUnlock()
	return alertCounters[key]
}

// resetCounter 重置计数
func resetCounter(key string) {
	mu.Lock()
	defer mu.Unlock()
	alertCounters[key] = 0
}

// isDebugMode 返回是否开启调试模式
func isDebugMode() bool {
	cfg := config.GetConfig()
	return cfg != nil && cfg.Log.Debug
}

// debugLog 仅在 debug 模式开启时输出
func debugLog(format string, args ...interface{}) {
	if isDebugMode() {
		log.Printf("[DEBUG] "+format, args...)
	}
}

// infoLog 仅在 debug 模式开启时输出信息日志
func infoLog(format string, args ...interface{}) {
	if isDebugMode() {
		log.Printf("[INFO] "+format, args...)
	}
}

// CheckAlarmsWithMetrics 使用指定的监控指标检查告警
func CheckAlarmsWithMetrics(cfg *entity.Config, metrics *entity.SystemMetrics) {
	var alerts []string
	var alertTypes []string
	var cpuDebug, memDebug, diskDebug, redisDebug, netDebug string

	var cpuHighTriggered, memHighTriggered bool

	// === CPU 检查（连续3次超标才报警）===
	if metrics.CPU.Error != nil {
		if isAlertAllowed(string(entity.CPUErr)) {
			msg := fmt.Sprintf("CPU 监控失败: %v", metrics.CPU.Error)
			alerts = append(alerts, msg)
			alertTypes = append(alertTypes, string(entity.CPUErr))
			cpuDebug = "CPU错误告警已触发"
			log.Printf("[ERROR] CPU监控失败 error=%v", metrics.CPU.Error)
		} else {
			cpuDebug = "CPU错误告警被防抖"
			debugLog("CPU错误告警被防抖")
		}
	} else if metrics.CPU.Percent > cfg.Monitor.CPUThreshold {
		if shouldTriggerAlert(string(entity.CPUHigh), true, 3) {
			if isAlertAllowed(string(entity.CPUHigh)) {
				topCPU, _, err := monitor.GetTopProcesses(5)
				var culprit string
				if err == nil && len(topCPU) > 0 {
					culprit = fmt.Sprintf("（元凶: %s PID=%d %.2f%% CPU）",
						truncate(topCPU[0].Name, 16), topCPU[0].PID, topCPU[0].CPUPercent)
					log.Printf("[WARNING] CPU过高，疑似元凶: %s (PID=%d) %.2f%% CPU",
						topCPU[0].Name, topCPU[0].PID, topCPU[0].CPUPercent)
				} else {
					culprit = "（无法获取进程信息）"
				}

				msg := fmt.Sprintf("CPU 使用率过高: %.2f%%%s", metrics.CPU.Percent, culprit)
				alerts = append(alerts, msg)
				alertTypes = append(alertTypes, string(entity.CPUHigh))
				cpuDebug = fmt.Sprintf("CPU告警已触发: %.2f%%", metrics.CPU.Percent)
			} else {
				cpuDebug = fmt.Sprintf("CPU告警被防抖: %.2f%%", metrics.CPU.Percent)
			}
		} else {
			count := getCounter(string(entity.CPUHigh))
			cpuDebug = fmt.Sprintf("CPU过高（%d/3）暂不报警", count)
			infoLog("CPU连续超标计数: %d/3", count)
		}
		cpuHighTriggered = true
	} else {
		cpuDebug = fmt.Sprintf("CPU正常: %.2f%%", metrics.CPU.Percent)
		debugLog("CPU使用率正常 percent=%.2f", metrics.CPU.Percent)
	}

	// 清零 CPU 计数器
	if !cpuHighTriggered {
		resetCounter(string(entity.CPUHigh))
	}

	// === 内存检查（连续3次超标才报警）===
	if metrics.Memory.Error != nil {
		if isAlertAllowed(string(entity.MemErr)) {
			msg := fmt.Sprintf("内存监控失败: %v", metrics.Memory.Error)
			alerts = append(alerts, msg)
			alertTypes = append(alertTypes, string(entity.MemErr))
			memDebug = "内存错误告警已触发"
			log.Printf("[ERROR] 内存监控失败 error=%v", metrics.Memory.Error)
		} else {
			memDebug = "内存错误告警被防抖"
			debugLog("内存错误告警被防抖")
		}
	} else if metrics.Memory.Percent > cfg.Monitor.MemoryThreshold {
		if shouldTriggerAlert(string(entity.MemHigh), true, 3) {
			if isAlertAllowed(string(entity.MemHigh)) {
				topMem, _, err := monitor.GetTopProcesses(5)
				var culprit string
				if err == nil && len(topMem) > 0 {
					culprit = fmt.Sprintf("（元凶: %s PID=%d %.1f%% MEM, %dMB）",
						truncate(topMem[0].Name, 16), topMem[0].PID, topMem[0].MemPercent, topMem[0].MemRSS)
					log.Printf("[WARNING] 内存过高，疑似元凶: %s (PID=%d) %.1f%% MEM, %dMB",
						topMem[0].Name, topMem[0].PID, topMem[0].MemPercent, topMem[0].MemRSS)
				} else {
					culprit = "（无法获取进程信息）"
				}

				msg := fmt.Sprintf("内存使用率过高: %.2f%%%s", metrics.Memory.Percent, culprit)
				alerts = append(alerts, msg)
				alertTypes = append(alertTypes, string(entity.MemHigh))
				memDebug = fmt.Sprintf("内存告警已触发: %.2f%%", metrics.Memory.Percent)
			} else {
				memDebug = fmt.Sprintf("内存告警被防抖: %.2f%%", metrics.Memory.Percent)
			}
		} else {
			count := getCounter(string(entity.MemHigh))
			memDebug = fmt.Sprintf("内存过高（%d/3）暂不报警", count)
			infoLog("内存连续超标计数: %d/3", count)
		}
		memHighTriggered = true
	} else {
		memDebug = fmt.Sprintf("内存正常: %.2f%%", metrics.Memory.Percent)
		debugLog("内存使用率正常 percent=%.2f", metrics.Memory.Percent)
	}

	// 清零 内存 计数器
	if !memHighTriggered {
		resetCounter(string(entity.MemHigh))
	}

	// === 磁盘检查 ===
	if metrics.Disk.Error != nil {
		if isAlertAllowed(string(entity.DiskErr)) {
			msg := fmt.Sprintf("磁盘监控失败: %v", metrics.Disk.Error)
			alerts = append(alerts, msg)
			alertTypes = append(alertTypes, string(entity.DiskErr))
			diskDebug = "磁盘错误告警已触发"
			log.Printf("[ERROR] 磁盘监控失败 error=%v", metrics.Disk.Error)
		} else {
			diskDebug = "磁盘错误告警被防抖"
			debugLog("磁盘错误告警被防抖")
		}
	} else if metrics.Disk.Percent > cfg.Monitor.DiskThreshold {
		if isAlertAllowed(string(entity.DiskHigh)) {
			msg := fmt.Sprintf("磁盘使用率过高: %.2f%%", metrics.Disk.Percent)
			alerts = append(alerts, msg)
			alertTypes = append(alertTypes, string(entity.DiskHigh))
			diskDebug = fmt.Sprintf("磁盘告警已触发: %.2f%%", metrics.Disk.Percent)
			log.Printf("[WARNING] 磁盘使用率过高 percent=%.2f threshold=%.2f",
				metrics.Disk.Percent, cfg.Monitor.DiskThreshold)
		} else {
			diskDebug = fmt.Sprintf("磁盘告警被防抖: %.2f%%", metrics.Disk.Percent)
			debugLog("磁盘告警被防抖")
		}
	} else {
		diskDebug = fmt.Sprintf("磁盘正常: %.2f%%", metrics.Disk.Percent)
		debugLog("磁盘使用率正常 percent=%.2f", metrics.Disk.Percent)
	}

	// === Redis 检查 ===
	if metrics.Redis.ConnectionError != nil {
		if isAlertAllowed(string(entity.RedisErr)) {
			msg := fmt.Sprintf("Redis 连接异常: %v", metrics.Redis.ConnectionError)
			alerts = append(alerts, msg)
			alertTypes = append(alertTypes, string(entity.RedisErr))
			redisDebug = "Redis错误告警已触发"
			log.Printf("[ERROR] Redis连接异常 error=%v", metrics.Redis.ConnectionError)
		} else {
			redisDebug = "Redis错误告警被防抖"
			debugLog("Redis错误告警被防抖")
		}
	} else if metrics.Redis.ClientCount < cfg.Monitor.RedisMinClients {
		if isAlertAllowed(string(entity.RedisLow)) {
			msg := fmt.Sprintf("Redis 连接数过低: %d (阈值: %d)", metrics.Redis.ClientCount, cfg.Monitor.RedisMinClients)
			alerts = append(alerts, msg)
			alertTypes = append(alertTypes, string(entity.RedisLow))
			redisDebug = fmt.Sprintf("Redis连接数过低告警已触发: %d", metrics.Redis.ClientCount)
			log.Printf("[WARNING] Redis连接数过低 count=%d min=%d", metrics.Redis.ClientCount, cfg.Monitor.RedisMinClients)
		} else {
			redisDebug = fmt.Sprintf("Redis连接数过低告警被防抖: %d", metrics.Redis.ClientCount)
			debugLog("Redis连接数过低告警被防抖")
		}
	} else if metrics.Redis.ClientCount > cfg.Monitor.RedisMaxClients {
		if isAlertAllowed(string(entity.RedisHigh)) {
			msg := fmt.Sprintf("Redis 连接数过高: %d (阈值: %d)", metrics.Redis.ClientCount, cfg.Monitor.RedisMaxClients)
			alerts = append(alerts, msg)
			alertTypes = append(alertTypes, string(entity.RedisHigh))
			redisDebug = fmt.Sprintf("Redis连接数过高告警已触发: %d", metrics.Redis.ClientCount)
			log.Printf("[WARNING] Redis连接数过高 count=%d max=%d", metrics.Redis.ClientCount, cfg.Monitor.RedisMaxClients)
		} else {
			redisDebug = fmt.Sprintf("Redis连接数过高告警被防抖: %d", metrics.Redis.ClientCount)
			debugLog("Redis连接数过高告警被防抖")
		}
	} else {
		redisDebug = fmt.Sprintf("Redis正常: %d个连接", metrics.Redis.ClientCount)
		debugLog("Redis连接数正常 count=%d", metrics.Redis.ClientCount)
	}

	// === 网络检查 ===
	if metrics.Network.Error != nil {
		if isAlertAllowed(string(entity.NetworkErr)) {
			msg := fmt.Sprintf("网络监控失败: %v", metrics.Network.Error)
			alerts = append(alerts, msg)
			alertTypes = append(alertTypes, string(entity.NetworkErr))
			netDebug = "网络错误告警已触发"
			log.Printf("[ERROR] 网络监控失败 error=%v", metrics.Network.Error)
		} else {
			netDebug = "网络错误告警被防抖"
			debugLog("网络错误告警被防抖")
		}
	} else {
		dl := metrics.Network.DownloadKBps
		ul := metrics.Network.UploadKBps
		netDebug = fmt.Sprintf("网络正常: ↓%.2f KB/s ↑%.2f KB/s", dl, ul)
		debugLog("%s", netDebug)
	}

	log.Printf("[STATUS] 告警状态: CPU=%s | Memory=%s | Disk=%s | Redis=%s | Network=%s",
		cpuDebug, memDebug, diskDebug, redisDebug, netDebug)

	// === 发送钉钉通知 ===
	if len(alerts) > 0 {
		title := "香港视频化服务器告警"
		text := "## " + title + "\n\n"

		// 触发的告警
		text += "### 触发告警项\n\n"
		for _, msg := range alerts {
			text += "> " + msg + "\n\n"
		}

		// 完整监控指标
		text += "### 完整监控指标\n\n"

		if metrics.CPU.Error != nil {
			text += fmt.Sprintf("**CPU**: 监控失败 - %v\n\n", metrics.CPU.Error)
		} else {
			text += fmt.Sprintf("**CPU**: %.2f%% %s\n\n",
				metrics.CPU.Percent, getStatusText(metrics.CPU.Percent, cfg.Monitor.CPUThreshold))
		}

		if metrics.Memory.Error != nil {
			text += fmt.Sprintf("**内存**: 监控失败 - %v\n\n", metrics.Memory.Error)
		} else {
			text += fmt.Sprintf("**内存**: %.2f%% (%d/%d MB) %s\n\n",
				metrics.Memory.Percent, metrics.Memory.UsedMB, metrics.Memory.TotalMB,
				getStatusText(metrics.Memory.Percent, cfg.Monitor.MemoryThreshold))
		}

		if metrics.Disk.Error != nil {
			text += fmt.Sprintf("**磁盘**: 监控失败 - %v\n\n", metrics.Disk.Error)
		} else {
			text += fmt.Sprintf("**磁盘**: %.2f%% (%d/%d GB) %s\n\n",
				metrics.Disk.Percent, metrics.Disk.UsedGB, metrics.Disk.TotalGB,
				getStatusText(metrics.Disk.Percent, cfg.Monitor.DiskThreshold))
		}

		if metrics.Redis.ConnectionError != nil {
			text += fmt.Sprintf("**Redis**: 连接失败 - %v\n\n", metrics.Redis.ConnectionError)
		} else {
			text += fmt.Sprintf("**Redis**: %d个连接 %s\n\n",
				metrics.Redis.ClientCount, getRedisStatusText(metrics.Redis.ClientCount, cfg))
		}

		if metrics.Network.Error != nil {
			text += fmt.Sprintf("**网络**: 监控失败 - %v\n\n", metrics.Network.Error)
		} else {
			text += fmt.Sprintf("**网络**: 下载 %.2f KB/s | 上传 %.2f KB/s\n\n",
				metrics.Network.DownloadKBps, metrics.Network.UploadKBps)
		}

		text += fmt.Sprintf("**监控时间**: %s\n\n", metrics.Timestamp.Format(time.DateTime))

		// === 是否需要触发 jmap？仅在 CPU 或内存高时 ===
		shouldTriggerDump := false
		for _, typ := range alertTypes {
			if typ == string(entity.CPUHigh) || typ == string(entity.MemHigh) {
				shouldTriggerDump = true
				break
			}
		}

		if shouldTriggerDump {
			// 异步执行脚本，不阻塞通知
			go utils.ExecuteJavaDumpScriptAsync(cfg.JavaAppDumpScript.Path)

			// 在钉钉中提示“已触发”
			text += "> 检测到高负载，已自动触发 Java 堆转储生成（异步执行中）...\n\n"
		}

		// 发送钉钉通知
		err := dingtalk.SendDingDingNotification(
			cfg.DingTalk.WebhookURL,
			cfg.DingTalk.Secret,
			title,
			text,
			cfg.DingTalk.AtMobiles,
			false,
		)
		if err != nil {
			log.Printf("[ERROR] 钉钉消息发送失败: %v", err)
		} else {
			log.Printf("[INFO] 钉钉告警已发送，共 %d 个告警", len(alerts))
		}
	}
}

// 工具函数
func getStatusText(value, threshold float64) string {
	if value > threshold {
		return "[异常]"
	}
	return "[正常]"
}

func getRedisStatusText(count int, cfg *entity.Config) string {
	if count < cfg.Monitor.RedisMinClients {
		return "[连接数过低]"
	} else if count > cfg.Monitor.RedisMaxClients {
		return "[连接数过高]"
	}
	return "[正常]"
}

func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}
