// internal/monitor/cpu.go
package monitor

import (
	"github.com/shirou/gopsutil/v3/cpu"
)

// GetCPUPercent 获取 CPU 使用率（平均）
func GetCPUPercent() (float64, error) {
	percent, err := cpu.Percent(0, true)
	if err != nil {
		return 0, err
	}

	// 计算所有核心平均值
	var total float64
	for _, p := range percent {
		total += p
	}
	return total / float64(len(percent)), nil
}
