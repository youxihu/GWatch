// internal/monitor/monitor.go
package monitor

import (
	"GWatch/internal/entity"
	"fmt"
	"time"
)

// CollectAllMetrics 收集所有系统指标
func CollectAllMetrics() *entity.SystemMetrics {
	metrics := &entity.SystemMetrics{
		Timestamp: time.Now(),
	}

	// 收集CPU指标
	metrics.CPU.Percent, metrics.CPU.Error = GetCPUPercent()

	// 收集内存指标
	metrics.Memory.Percent, metrics.Memory.UsedMB, metrics.Memory.TotalMB, metrics.Memory.Error = GetMemoryUsage()

	// 收集磁盘指标
	metrics.Disk.Percent, metrics.Disk.UsedGB, metrics.Disk.TotalGB, metrics.Disk.Error = GetDiskUsage()

	// 收集网络指标
	metrics.Network.DownloadKBps, metrics.Network.UploadKBps, metrics.Network.Error = GetNetworkRate()

	// 收集Redis指标
	metrics.Redis.ClientCount, metrics.Redis.ConnectionError = GetRedisClients()
	metrics.Redis.ClientDetails, metrics.Redis.DetailError = GetRedisClientsDetail()

	return metrics
}

// PrintMetrics 打印所有监控指标
func PrintMetrics(metrics *entity.SystemMetrics) {
	fmt.Println("--- 系统状态 ---")

	// CPU
	if metrics.CPU.Error != nil {
		fmt.Printf("CPU 监控失败: %v\n", metrics.CPU.Error)
	} else {
		fmt.Printf("CPU 使用率: %.2f%%\n", metrics.CPU.Percent)
	}

	// 内存
	if metrics.Memory.Error != nil {
		fmt.Printf("内存监控失败: %v\n", metrics.Memory.Error)
	} else {
		fmt.Printf("内存使用: %.2f%% (%d/%d MB)\n",
			metrics.Memory.Percent, metrics.Memory.UsedMB, metrics.Memory.TotalMB)
	}

	// 磁盘
	if metrics.Disk.Error != nil {
		fmt.Printf("磁盘监控失败: %v\n", metrics.Disk.Error)
	} else {
		fmt.Printf("磁盘使用: %.2f%% (%d/%d GB)\n",
			metrics.Disk.Percent, metrics.Disk.UsedGB, metrics.Disk.TotalGB)
	}

	// 网络
	if metrics.Network.Error != nil {
		fmt.Printf("网络监控失败: %v\n", metrics.Network.Error)
	} else {
		fmt.Printf("网络波动: 下载 %.2f KB/s | 上传 %.2f KB/s\n",
			metrics.Network.DownloadKBps, metrics.Network.UploadKBps)
	}

	fmt.Println("\n--- Redis 状态 ---")

	// Redis连接数
	if metrics.Redis.ConnectionError != nil {
		fmt.Printf("Redis 连接失败: %v\n", metrics.Redis.ConnectionError)
	} else {
		fmt.Printf("当前连接数: %d\n", metrics.Redis.ClientCount)
	}

	// Redis连接详情
	if metrics.Redis.DetailError != nil {
		fmt.Printf("获取连接详情失败: %v\n", metrics.Redis.DetailError)
	} else if len(metrics.Redis.ClientDetails) == 0 {
		fmt.Println("当前无任何客户端连接")
	} else {
		for _, c := range metrics.Redis.ClientDetails {
			fmt.Printf("ID=%s Addr=%s Age=%s Idle=%s Flags=%s DB=%s CMD=%s\n",
				c.ID, c.Addr, c.Age, c.Idle, c.Flags, c.Db, c.Cmd)
		}
	}

	fmt.Println()
}

// CollectAndPrint 收集并打印监控指标，返回收集的指标
func CollectAndPrint() *entity.SystemMetrics {
	metrics := CollectAllMetrics()
	PrintMetrics(metrics)
	return metrics
}
