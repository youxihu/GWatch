package usecase

import (
	domainAlert "GWatch/internal/domain/alert"
	"GWatch/internal/domain/collector"
	domainMonitor "GWatch/internal/domain/monitor"
	"GWatch/internal/entity"
	"GWatch/internal/utils"
	"fmt"
	"time"
)

// MonitoringUseCase 负责完整的监控流程：采集 → 判断 → 策略 → 格式化 → 发送
type MonitoringUseCase struct {
	host        collector.HostCollector
	redis       RedisClient
	evaluator   domainMonitor.Evaluator
	policy      domainAlert.Policy
	formatter   domainAlert.Formatter
	notifier    Notifier
	redisInited bool
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
	evaluator domainMonitor.Evaluator,
	policy domainAlert.Policy,
	formatter domainAlert.Formatter,
	notifier Notifier,
) *MonitoringUseCase {
	return &MonitoringUseCase{
		host:      host,
		redis:     redis,
		evaluator: evaluator,
		policy:    policy,
		formatter: formatter,
		notifier:  notifier,
	}
}

// Run 执行一次完整的监控流程
func (uc *MonitoringUseCase) Run(cfg *entity.Config) error {
	// 1. 采集指标
	metrics := uc.CollectOnce()

	// 2. 打印采集结果（可选，用于本地观察）
	uc.PrintMetrics(metrics)

	// 3. 阈值判断与告警处理
	return uc.EvaluateAndNotify(cfg, metrics)
}

func (uc *MonitoringUseCase) CollectOnce() *entity.SystemMetrics {
	m := &entity.SystemMetrics{Timestamp: time.Now()}

	m.CPU.Percent, m.CPU.Error = uc.host.GetCPUPercent()
	m.Memory.Percent, m.Memory.UsedMB, m.Memory.TotalMB, m.Memory.Error = uc.host.GetMemoryUsage()
	m.Disk.Percent, m.Disk.UsedGB, m.Disk.TotalGB, m.Disk.Error = uc.host.GetDiskUsage()
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

	return m
}

func (uc *MonitoringUseCase) EvaluateAndNotify(cfg *entity.Config, m *entity.SystemMetrics) error {
	decisions, _ := uc.evaluator.Evaluate(cfg, m)
	alertTypes := uc.policy.Apply(cfg, m, decisions)
	if len(alertTypes) == 0 {
		return nil
	}

	var alerts []domainAlert.TriggeredAlert
	dumpTriggered := false

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
			go utils.ExecuteJavaDumpScriptAsync(cfg.JavaAppDumpScript.Path)
			dumpTriggered = true
		}

		alerts = append(alerts, domainAlert.TriggeredAlert{Type: t, Message: msg})
	}

	if dumpTriggered {
		alerts = append(alerts, domainAlert.TriggeredAlert{
			Type:    entity.Info,
			Message: "检测到高负载，已自动触发 Java 堆转储生成（异步执行中）...",
		})
	}

	body := uc.formatter.Build("香港视频化服务器告警", cfg, m, alerts)
	return uc.notifier.Send("香港视频化服务器告警", body)
}

// PrintMetrics 仅用于本地观察，不属于核心业务
func (uc *MonitoringUseCase) PrintMetrics(m *entity.SystemMetrics) {
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
		fmt.Printf("磁盘使用: %.2f%% (%d/%d GB)\n", m.Disk.Percent, m.Disk.UsedGB, m.Disk.TotalGB)
	}
	if m.Network.Error != nil {
		fmt.Println("网络监控失败:", m.Network.Error.Error())
	} else {
		fmt.Printf("网络: 下载 %.2f KB/s | 上传 %.2f KB/s\n", m.Network.DownloadKBps, m.Network.UploadKBps)
	}
	if m.Redis.ConnectionError != nil {
		fmt.Println("Redis 连接失败:", m.Redis.ConnectionError.Error())
	} else {
		fmt.Printf("Redis 连接数: %d\n", m.Redis.ClientCount)
	}
}
