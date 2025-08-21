package metrics

import (
	"GWatch/internal/domain/collector"
	"GWatch/internal/entity"
	"time"
)

// CollectAllMetrics 聚合主机与服务采集结果
func CollectAllMetrics(host collector.HostCollector, redis collector.RedisCollector) *entity.SystemMetrics {
	metrics := &entity.SystemMetrics{ Timestamp: time.Now() }
	if host != nil {
		metrics.CPU.Percent, metrics.CPU.Error = host.GetCPUPercent()
		metrics.Memory.Percent, metrics.Memory.UsedMB, metrics.Memory.TotalMB, metrics.Memory.Error = host.GetMemoryUsage()
		metrics.Disk.Percent, metrics.Disk.UsedGB, metrics.Disk.TotalGB, metrics.Disk.Error = host.GetDiskUsage()
		metrics.Network.DownloadKBps, metrics.Network.UploadKBps, metrics.Network.Error = host.GetNetworkRate()
	}
	if redis != nil {
		count, err1 := redis.GetClients()
		detail, err2 := redis.GetClientsDetail()
		metrics.Redis.ClientCount = count
		metrics.Redis.ClientDetails = detail
		metrics.Redis.ConnectionError = err1
		metrics.Redis.DetailError = err2
	}
	return metrics
}


