// internal/entity/alert_type.go
package entity

type AlertType string

// 所有告警类型的常量定义
const (
	CPUHigh    AlertType = "cpu_high"      // CPU过高
	CPUErr     AlertType = "cpu_error"     // CPU监控失败
	MemHigh    AlertType = "mem_high"      // 内存过高
	MemErr     AlertType = "mem_error"     // 内存监控失败
	DiskHigh   AlertType = "disk_high"     // 磁盘过高
	DiskErr    AlertType = "disk_error"    // 磁盘监控失败
	RedisHigh  AlertType = "redis_high"    // Redis连接数过高
	RedisLow   AlertType = "redis_low"     // Redis连接数过低
	RedisErr   AlertType = "redis_error"   // Redis连接异常
	NetworkErr AlertType = "network_error" // 网络监控失败
)

// 告警类型中文描述映射表
var AlertTypeText = map[AlertType]string{
	CPUHigh:    "CPU 使用率过高",
	CPUErr:     "CPU 监控失败",
	MemHigh:    "内存使用率过高",
	MemErr:     "内存监控失败",
	DiskHigh:   "磁盘使用率过高",
	DiskErr:    "磁盘监控失败",
	RedisHigh:  "Redis 连接数过高",
	RedisLow:   "Redis 连接数过低",
	RedisErr:   "Redis 连接异常",
	NetworkErr: "网络监控失败",
}

// 是否需要“连续超标”才触发的类型（用于 shouldTriggerAlert）
var AlertTypeRequiresConsecutive = map[AlertType]bool{
	CPUHigh: true,
	MemHigh: true,
	// 其他错误类或瞬时类告警不需要连续触发
	CPUErr:     false,
	MemErr:     false,
	DiskHigh:   false,
	DiskErr:    false,
	RedisHigh:  false,
	RedisLow:   false,
	RedisErr:   false,
	NetworkErr: false,
}

// 获取告警中文名
func (a AlertType) String() string {
	if text, exists := AlertTypeText[a]; exists {
		return text
	}
	return "未知告警"
}
