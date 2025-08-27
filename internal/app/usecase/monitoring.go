package usecase

import (
	domainAlert "GWatch/internal/domain/alert"
	"GWatch/internal/domain/collector"
	domainMonitor "GWatch/internal/domain/monitor"
	"GWatch/internal/entity"
	"GWatch/internal/utils"
	"fmt"
	"strings"
	"time"
)

// MonitoringUseCase 负责完整的监控流程：采集 → 判断 → 策略 → 格式化 → 发送
type MonitoringUseCase struct {
	host        collector.HostCollector
	redis       RedisClient
	http        collector.HTTPCollector
	evaluator   domainMonitor.Evaluator
	policy      domainAlert.Policy
	formatter   domainAlert.Formatter
	notifier    Notifier
	redisInited bool
	httpInited  bool
}

// RedisClient 是 redis 操作接口
type RedisClient interface {
	Init() error
	GetClients() (int, error)
	GetClientsDetail() ([]entity.ClientInfo, error)
}

// Notifier 发送告警通知
type Notifier interface {
	Send(title, markdown string) error
}

// NewMonitoringUseCase 创建监控用例
func NewMonitoringUseCase(
	host collector.HostCollector,
	redis RedisClient,
	http collector.HTTPCollector,
	evaluator domainMonitor.Evaluator,
	policy domainAlert.Policy,
	formatter domainAlert.Formatter,
	notifier Notifier,
) *MonitoringUseCase {
	return &MonitoringUseCase{
		host:      host,
		redis:     redis,
		http:      http,
		evaluator: evaluator,
		policy:    policy,
		formatter: formatter,
		notifier:  notifier,
	}
}

// Run 执行一次完整的监控流程
func (uc *MonitoringUseCase) Run(cfg *entity.Config) error {
	// 1. 采集指标
	metrics := uc.CollectOnce(cfg)

	// 2. 打印采集结果（可选，用于本地观察）
	uc.PrintMetrics(metrics)

	// 3. 阈值判断与告警处理
	return uc.EvaluateAndNotify(cfg, metrics)
}

func (uc *MonitoringUseCase) CollectOnce(cfg *entity.Config) *entity.SystemMetrics {
	m := &entity.SystemMetrics{Timestamp: time.Now()}

	m.CPU.Percent, m.CPU.Error = uc.host.GetCPUPercent()
	m.Memory.Percent, m.Memory.UsedMB, m.Memory.TotalMB, m.Memory.Error = uc.host.GetMemoryUsage()
	m.Disk.Percent, m.Disk.UsedGB, m.Disk.TotalGB, m.Disk.Error = uc.host.GetDiskUsage()
	m.Disk.ReadKBps, m.Disk.WriteKBps, _ = uc.host.GetDiskIORate()
	m.Network.DownloadKBps, m.Network.UploadKBps, m.Network.Error = uc.host.GetNetworkRate()

	if !uc.redisInited {
		if err := uc.redis.Init(); err != nil {
			m.Redis.ConnectionError = err
		} else {
			uc.redisInited = true
		}
	}

	if uc.redisInited {
		m.Redis.ClientCount, m.Redis.ConnectionError = uc.redis.GetClients()
		m.Redis.ClientDetails, m.Redis.DetailError = uc.redis.GetClientsDetail()
	}

	// 收集HTTP接口监控指标
	if !uc.httpInited {
		if err := uc.http.Init(); err != nil {
			m.HTTP.Error = err
		} else {
			uc.httpInited = true
		}
	}

	if uc.httpInited {
		// 从配置中获取HTTP接口列表进行监控
		var httpInterfaces []entity.HTTPInterfaceMetrics
		if cfg != nil && cfg.Monitor.HTTPInterfaces != nil {
			for _, httpConfig := range cfg.Monitor.HTTPInterfaces {
				isAccessible, responseTime, statusCode, err := uc.http.CheckInterface(httpConfig.URL, httpConfig.Timeout)

				httpInterfaces = append(httpInterfaces, entity.HTTPInterfaceMetrics{
					Name:         httpConfig.Name,
					URL:          httpConfig.URL,
					IsAccessible: isAccessible,
					ResponseTime: responseTime,
					StatusCode:   statusCode,
					Error:        err,
				})
			}
		}
		m.HTTP.Interfaces = httpInterfaces
	}

	return m
}

func (uc *MonitoringUseCase) EvaluateAndNotify(cfg *entity.Config, m *entity.SystemMetrics) error {
	decisions, _ := uc.evaluator.Evaluate(cfg, m)
	alertTypes := uc.policy.Apply(cfg, m, decisions)
	if len(alertTypes) == 0 {
		return nil
	}

	var alerts []domainAlert.TriggeredAlert
	dumpTriggeredAsync := false // 变量名改得更清晰

	for _, t := range alertTypes {
		msg := t.String()

		if t == entity.CPUHigh || t == entity.MemHigh {
			if topCPU, topMem, err := uc.host.GetTopProcesses(5); err == nil {
				if t == entity.CPUHigh && len(topCPU) > 0 {
					culprit := topCPU[0]
					msg = fmt.Sprintf(
						"CPU 使用率过高: %.2f%%（元凶: %s PID=%d %.2f%% CPU）",
						m.CPU.Percent, culprit.Name, culprit.PID, culprit.CPUPercent,
					)
				}
				if t == entity.MemHigh && len(topMem) > 0 {
					culprit := topMem[0]
					msg = fmt.Sprintf(
						"内存使用率过高: %.2f%%（元凶: %s PID=%d %.1f%% MEM, %dMB）",
						m.Memory.Percent, culprit.Name, culprit.PID, culprit.MemPercent, culprit.MemRSS,
					)
				}
			}

			// 执行脚本并等待最多 3 秒
			done := make(chan struct{}, 1)
			var result string
			go func() {
				r, err := utils.ExecuteJavaDumpScriptResult(cfg.JavaAppDumpScript.Path, 3*time.Second)
				if err == nil {
					result = r
				}
				done <- struct{}{}
			}()

			select {
			case <-done:
				// 同步执行完成，根据结果附加信息
				if strings.Contains(result, "file_exist") {
					msg += "\n\n> 提示：堆转储文件已存在，跳过生成"
				} else if strings.Contains(result, "failed") {
					msg += "\n\n> 提示：Java堆转储生成失败"
				} else if strings.Contains(result, "success") {
					msg += "\n\n> 提示：已生成 Java 堆转储"
				} else if result != "" {
					msg += "\n\n> 提示：" + result + ""
				}
			case <-time.After(3 * time.Second):
				// 超时，转为异步执行
				go utils.ExecuteJavaDumpScriptAsync(cfg.JavaAppDumpScript.Path)
				dumpTriggeredAsync = true // <<-- **只在这里设置标志位**
			}
		}

		alerts = append(alerts, domainAlert.TriggeredAlert{Type: t, Message: msg})
	}

	// 只有在真正超时后，才添加异步执行的提示
	if dumpTriggeredAsync {
		alerts = append(alerts, domainAlert.TriggeredAlert{
			Type:    entity.Info,
			Message: "检测到高负载，已自动触发 Java 堆转储生成（异步执行中）...", // <<-- 提示信息更准确
		})
	}

	body := uc.formatter.Build("香港视频化服务器告警", cfg, m, alerts)
	return uc.notifier.Send("香港视频化服务器告警", body)
}

// PrintMetrics 仅用于本地观察，不属于核心业务
func (uc *MonitoringUseCase) PrintMetrics(m *entity.SystemMetrics) {
	now := time.Now() // 获取当前时间
	fmt.Println("===========采集数据============")
	if m.CPU.Error != nil {
		fmt.Println("CPU 监控失败:", m.CPU.Error.Error())
	} else {
		fmt.Printf("CPU 使用率: %.2f%%\n", m.CPU.Percent)
	}
	if m.Memory.Error != nil {
		fmt.Println("内存监控失败:", m.Memory.Error.Error())
	} else {
		fmt.Printf("内存使用: %.2f%% (%d/%d MB)\n", m.Memory.Percent, m.Memory.UsedMB, m.Memory.TotalMB)
	}
	if m.Disk.Error != nil {
		fmt.Println("磁盘监控失败:", m.Disk.Error.Error())
	} else {
		fmt.Printf("磁盘使用: %.2f%% (%d/%d GB)\n",
			m.Disk.Percent, m.Disk.UsedGB, m.Disk.TotalGB)
	}
	if m.Redis.ConnectionError != nil {
		fmt.Println("Redis 连接失败:", m.Redis.ConnectionError.Error())
	} else {
		fmt.Printf("Redis 连接数: %d\n", m.Redis.ClientCount)
	}
	if m.Network.Error != nil {
		fmt.Println("网络监控失败:", m.Network.Error.Error())
	} else {
		fmt.Printf("网络: 下载 %.2f KB/s | 上传 %.2f KB/s\n", m.Network.DownloadKBps, m.Network.UploadKBps)
	}
	fmt.Printf("磁盘IO: 读 %.2f KB/s | 写 %.2f KB/s\n", m.Disk.ReadKBps, m.Disk.WriteKBps)

	// 打印HTTP接口监控信息
	if m.HTTP.Error != nil {
		fmt.Println("HTTP接口监控失败:", m.HTTP.Error.Error())
	} else {
		for _, httpInterface := range m.HTTP.Interfaces {
			if httpInterface.IsAccessible {
				fmt.Printf("HTTP接口 %s: 正常 (状态码: %d, 响应时间: %v)\n",
					httpInterface.Name, httpInterface.StatusCode, httpInterface.ResponseTime)
			} else {
				fmt.Printf("HTTP接口 %s: 异常 (状态码: %d) - %v\n",
					httpInterface.Name, httpInterface.StatusCode, httpInterface.Error)
			}
		}
	}

	fmt.Printf("监控时间: %s\n", now.Format(time.DateTime))
}
