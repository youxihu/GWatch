// internal/monitor/process.go
package monitor

import (
	"fmt"
	"sort"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

type ProcessInfo struct {
	PID        int32   `json:"pid"`
	Name       string  `json:"name"`
	CPUPercent float64 `json:"cpu_percent"`
	MemPercent float32 `json:"mem_percent"`
	MemRSS     uint64  `json:"mem_rss_mb"`
}

// GetTopProcesses 获取 CPU 和内存占用最高的前 N 个进程
func GetTopProcesses(n int) ([]ProcessInfo, []ProcessInfo, error) {
	pids, err := process.Pids()
	if err != nil {
		return nil, nil, fmt.Errorf("无法获取 PID 列表: %w", err)
	}

	var processes []*process.Process
	for _, pid := range pids {
		p, err := process.NewProcess(pid)
		if err != nil {
			// 常见：权限不足、进程已退出
			continue
		}
		if p == nil {
			continue // 安全起见
		}
		processes = append(processes, p)
	}

	var cpuList, memList []ProcessInfo

	// 采集 CPU 使用率前等待一小段时间
	time.Sleep(300 * time.Millisecond)

	for _, p := range processes {
		if p == nil {
			continue
		}

		// 获取 CPU 使用率
		cpuPercent, err := p.CPUPercent()
		if err != nil {
			// 进程可能已退出
			continue
		}

		// 获取内存信息
		memInfo, err := p.MemoryInfo()
		if err != nil {
			continue
		}

		memPercent, err := p.MemoryPercent()
		if err != nil {
			continue
		}

		name, err := p.Name()
		if err != nil {
			// 可以设为 unknown
			name = "unknown"
		}

		info := ProcessInfo{
			PID:        p.Pid,
			Name:       name,
			CPUPercent: cpuPercent,
			MemPercent: memPercent,
			MemRSS:     memInfo.RSS / 1024 / 1024, // 转 MB
		}

		if cpuPercent > 0.1 {
			cpuList = append(cpuList, info)
		}
		if memPercent > 0.1 {
			memList = append(memList, info)
		}
	}

	// 排序
	sort.Slice(cpuList, func(i, j int) bool {
		return cpuList[i].CPUPercent > cpuList[j].CPUPercent
	})
	sort.Slice(memList, func(i, j int) bool {
		return memList[i].MemPercent > memList[j].MemPercent
	})

	// 截取前 N 个
	if len(cpuList) > n {
		cpuList = cpuList[:n]
	}
	if len(memList) > n {
		memList = memList[:n]
	}

	return cpuList, memList, nil
}
