// internal/monitor/disk.go
package monitor

import (
	"github.com/shirou/gopsutil/v3/disk"
)

// GetDiskUsage 获取根目录磁盘使用率
func GetDiskUsage() (float64, uint64, uint64, error) {
	usage, err := disk.Usage("/")
	if err != nil {
		return 0, 0, 0, err
	}
	usedGB := usage.Used / 1024 / 1024 / 1024
	totalGB := usage.Total / 1024 / 1024 / 1024
	return usage.UsedPercent, usedGB, totalGB, nil
}
