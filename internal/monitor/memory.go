// internal/monitor/memory_detect.go
package monitor

import (
	"github.com/shirou/gopsutil/v3/mem"
)

// GetMemoryUsage 获取内存使用率
func GetMemoryUsage() (float64, uint64, uint64, error) {
	vm, err := mem.VirtualMemory()
	if err != nil {
		return 0, 0, 0, err
	}
	usedMB := vm.Used / 1024 / 1024
	totalMB := vm.Total / 1024 / 1024
	percent := vm.UsedPercent
	return percent, usedMB, totalMB, nil
}
