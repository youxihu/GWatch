package runtime

import (
	domainAlert "GWatch/internal/domain/alert"
	"GWatch/internal/domain/collector"
	domainMonitor "GWatch/internal/domain/monitor"
	"GWatch/internal/entity"
	"GWatch/internal/utils"
	"time"
)

// Runner 编排一次完整流程：采集 -> 阈值判断 -> 策略（防抖/连续）-> 格式化 -> 发送
type Runner struct {
	host  collector.HostCollector
	redis interface {
		Init() error
		GetClients() (int, error)
		GetClientsDetail() ([]entity.ClientInfo, error)
	}
	evaluator domainMonitor.Evaluator
	policy    domainAlert.Policy
	formatter domainAlert.Formatter
	notifier  interface {
		Send(title string, markdown string) error
	}
	redisInited bool
}

func NewRunner(host collector.HostCollector,
	redis interface {
		Init() error
		GetClients() (int, error)
		GetClientsDetail() ([]entity.ClientInfo, error)
	},
	evaluator domainMonitor.Evaluator,
	policy domainAlert.Policy,
	formatter domainAlert.Formatter,
	notifier interface{ Send(string, string) error },
) *Runner {
	return &Runner{host: host, redis: redis, evaluator: evaluator, policy: policy, formatter: formatter, notifier: notifier}
}

func (r *Runner) CollectOnce() *entity.SystemMetrics {
	m := &entity.SystemMetrics{Timestamp: time.Now()}
	// CPU
	m.CPU.Percent, m.CPU.Error = r.host.GetCPUPercent()
	// Memory
	m.Memory.Percent, m.Memory.UsedMB, m.Memory.TotalMB, m.Memory.Error = r.host.GetMemoryUsage()
	// Disk
	m.Disk.Percent, m.Disk.UsedGB, m.Disk.TotalGB, m.Disk.Error = r.host.GetDiskUsage()
	// Network
	m.Network.DownloadKBps, m.Network.UploadKBps, m.Network.Error = r.host.GetNetworkRate()
	// Redis（首次初始化，后续复用）
	if !r.redisInited {
		if err := r.redis.Init(); err != nil {
			m.Redis.ConnectionError = err
		} else {
			r.redisInited = true
		}
	}
	if r.redisInited {
		m.Redis.ClientCount, m.Redis.ConnectionError = r.redis.GetClients()
		m.Redis.ClientDetails, m.Redis.DetailError = r.redis.GetClientsDetail()
	}
	return m
}

func (r *Runner) EvaluateAndNotify(cfg *entity.Config, m *entity.SystemMetrics) error {
	decisions, _ := r.evaluator.Evaluate(cfg, m)
	alertTypes := r.policy.Apply(cfg, m, decisions)
	if len(alertTypes) == 0 {
		return nil
	}

	// 组装详细告警消息（含元凶进程）
	var alerts []domainAlert.TriggeredAlert
	dumpTriggered := false
	for _, t := range alertTypes {
		msg := t.String()
		if t == entity.CPUHigh || t == entity.MemHigh {
			if topCPU, topMem, err := r.host.GetTopProcesses(5); err == nil {
				if t == entity.CPUHigh && len(topCPU) > 0 {
					culprit := topCPU[0]
					msg = "CPU 使用率过高: " + formatFloat(m.CPU.Percent, 2) + "%（元凶: " + culprit.Name + " PID=" + string(itoa(int64(culprit.PID))) + " " + formatFloat(culprit.CPUPercent, 2) + "% CPU）"
				}
				if t == entity.MemHigh && len(topMem) > 0 {
					culprit := topMem[0]
					msg = "内存使用率过高: " + formatFloat(m.Memory.Percent, 2) + "%（元凶: " + culprit.Name + " PID=" + string(itoa(int64(culprit.PID))) + " " + formatFloat(float64(culprit.MemPercent), 1) + "% MEM, " + string(itoa(int64(culprit.MemRSS))) + "MB）"
				}
			}
			// 触发异步脚本
			go utils.ExecuteJavaDumpScriptAsync(cfg.JavaAppDumpScript.Path)
			dumpTriggered = true
		}
		alerts = append(alerts, domainAlert.TriggeredAlert{Type: t, Message: msg})
	}

	if dumpTriggered {
		alerts = append(alerts, domainAlert.TriggeredAlert{Type: entity.AlertType(""), Message: "检测到高负载，已自动触发 Java 堆转储生成（异步执行中）..."})
	}

	body := r.formatter.Build("香港视频化服务器告警", cfg, m, alerts)
	return r.notifier.Send("香港视频化服务器告警", body)
}

// PrintMetrics 简单输出一行状态，便于观察网络与间隔
func (r *Runner) PrintMetrics(m *entity.SystemMetrics) {
	// 简要打印各类指标与错误
	// CPU
	if m.CPU.Error != nil {
		println("CPU 监控失败:", m.CPU.Error.Error())
	} else {
		println("CPU 使用率:", formatFloat(m.CPU.Percent, 2), "%")
	}
	// Memory
	if m.Memory.Error != nil {
		println("内存监控失败:", m.Memory.Error.Error())
	} else {
		println("内存使用:", formatFloat(m.Memory.Percent, 2), "% (", m.Memory.UsedMB, "/", m.Memory.TotalMB, "MB)")
	}
	// Disk
	if m.Disk.Error != nil {
		println("磁盘监控失败:", m.Disk.Error.Error())
	} else {
		println("磁盘使用:", formatFloat(m.Disk.Percent, 2), "% (", m.Disk.UsedGB, "/", m.Disk.TotalGB, "GB)")
	}
	// Network
	if m.Network.Error != nil {
		println("网络监控失败:", m.Network.Error.Error())
	} else {
		println("网络: 下载", formatFloat(m.Network.DownloadKBps, 2), "KB/s | 上传", formatFloat(m.Network.UploadKBps, 2), "KB/s")
	}
	// Redis
	if m.Redis.ConnectionError != nil {
		println("Redis 连接失败:", m.Redis.ConnectionError.Error())
	} else {
		println("Redis 连接数:", m.Redis.ClientCount)
	}
}

// 简单的浮点格式化，避免引入额外依赖
func formatFloat(v float64, prec int) string {
	// 手写格式化以避免额外 import；粗略格式足够观察
	// 注意：这不是高性能实现，仅用于日志
	s := "" // 复用 Sprintf 需要 fmt 包，这里保持轻量
	// 回退方案：直接转换为字符串
	// 在 Go 中不引入 fmt 难以指定精度，这里退化使用默认转换
	return s + (func(x float64) string { return fmtFloat(x) })(v)
}

// 用到 fmt 需引入；为了保持依赖简单，这里实现一个极简转换
func fmtFloat(v float64) string {
	// 近似输出，避免过长小数
	// 将其乘以 100 四舍五入，再除以 100
	mul := float64(100)
	vv := float64(int64(v*mul+0.5)) / mul
	// 转字符串
	b := []byte{}
	// 简化：转为整数与小数部分
	iv := int64(vv)
	fv := int64((vv - float64(iv)) * mul)
	b = append(b, itoa(iv)...)
	b = append(b, '.')
	if fv < 10 {
		b = append(b, '0')
	}
	b = append(b, itoa(fv)...)
	return string(b)
}

func itoa(x int64) []byte {
	if x == 0 {
		return []byte{'0'}
	}
	neg := false
	if x < 0 {
		neg = true
		x = -x
	}
	var buf [20]byte
	i := len(buf)
	for x > 0 {
		i--
		buf[i] = byte('0' + x%10)
		x /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return buf[i:]
}
