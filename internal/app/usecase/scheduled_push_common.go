// Package usecase internal/app/usecase/scheduled_push_common.go
package usecase

import (
	"GWatch/internal/domain/collector"
	"GWatch/internal/entity"
	"log"
	"net"
	"os"
	"time"
)

// MetricsCollector 指标收集器（供Client和Server共享使用）
type MetricsCollector struct {
	hostCollector collector.HostCollector
	redisClient   RedisClient
	httpCollector collector.HTTPCollector
}

// NewMetricsCollector 创建指标收集器
func NewMetricsCollector(
	hostCollector collector.HostCollector,
	redisClient RedisClient,
	httpCollector collector.HTTPCollector,
) *MetricsCollector {
	return &MetricsCollector{
		hostCollector: hostCollector,
		redisClient:   redisClient,
		httpCollector: httpCollector,
	}
}

// CollectBasicHostMetrics 收集基本主机指标（不依赖外部服务）
func (mc *MetricsCollector) CollectBasicHostMetrics() *entity.SystemMetrics {
	hostMetrics := &entity.SystemMetrics{
		Timestamp: time.Now(),
	}

	// 收集CPU指标
	cpuPercent, err := mc.hostCollector.GetCPUPercent()
	hostMetrics.CPU = entity.CPUMetrics{
		Percent: cpuPercent,
		Error:   err,
	}

	// 收集内存指标
	memPercent, usedMB, totalMB, err := mc.hostCollector.GetMemoryUsage()
	hostMetrics.Memory = entity.MemoryMetrics{
		Percent: memPercent,
		UsedMB:  usedMB,
		TotalMB: totalMB,
		Error:   err,
	}

	// 收集磁盘指标
	diskPercent, usedGB, totalGB, err := mc.hostCollector.GetDiskUsage()
	hostMetrics.Disk = entity.DiskMetrics{
		Percent: diskPercent,
		UsedGB:  usedGB,
		TotalGB: totalGB,
		Error:   err,
	}

	// 收集磁盘IO指标
	readKBps, writeKBps, err := mc.hostCollector.GetDiskIORate()
	if err == nil {
		hostMetrics.Disk.ReadKBps = readKBps
		hostMetrics.Disk.WriteKBps = writeKBps
	}

	// 收集网络指标
	downloadKBps, uploadKBps, err := mc.hostCollector.GetNetworkRate()
	hostMetrics.Network = entity.NetworkMetrics{
		DownloadKBps: downloadKBps,
		UploadKBps:   uploadKBps,
		Error:        err,
	}

	return hostMetrics
}

// CollectRedisMetrics 收集Redis指标
func (mc *MetricsCollector) CollectRedisMetrics(config *entity.Config) *entity.RedisMetrics {
	redisMetrics := &entity.RedisMetrics{
		ClientCount: 0,
	}

	// 初始化Redis连接
	if err := mc.redisClient.Init(); err != nil {
		redisMetrics.ConnectionError = err
		return redisMetrics
	}

	// 获取Redis连接数
	clientCount, err := mc.redisClient.GetClients()
	if err != nil {
		redisMetrics.ConnectionError = err
	} else {
		redisMetrics.ClientCount = clientCount
	}

	// 获取Redis连接详情
	clientDetails, err := mc.redisClient.GetClientsDetail()
	if err != nil {
		redisMetrics.DetailError = err
	} else {
		redisMetrics.ClientDetails = clientDetails
	}

	return redisMetrics
}

// CollectMySQLMetrics 收集MySQL指标
func (mc *MetricsCollector) CollectMySQLMetrics(config *entity.Config) *entity.MySQLMetrics {
	mysqlMetrics := &entity.MySQLMetrics{
		Error: nil,
	}

	// 这里可以添加MySQL连接和指标收集逻辑
	// 简化处理，只设置基本状态

	return mysqlMetrics
}

// CollectHTTPMetrics 收集HTTP指标
func (mc *MetricsCollector) CollectHTTPMetrics(config *entity.Config) *entity.HTTPMetrics {
	httpMetrics := &entity.HTTPMetrics{
		Interfaces: []entity.HTTPInterfaceMetrics{},
		Error:      nil,
	}

	if config.AppMonitoring != nil && config.AppMonitoring.HTTP != nil {
		// 初始化HTTP收集器
		if err := mc.httpCollector.Init(); err != nil {
			httpMetrics.Error = err
			return httpMetrics
		}

		// 检查每个HTTP接口
		var httpInterfaces []entity.HTTPInterfaceMetrics
		for _, httpConfig := range config.AppMonitoring.HTTP.Interfaces {
			isAccessible, responseTime, statusCode, err := mc.httpCollector.CheckInterface(httpConfig.URL, httpConfig.Timeout)
			
			httpInterfaces = append(httpInterfaces, entity.HTTPInterfaceMetrics{
				Name:         httpConfig.Name,
				URL:          httpConfig.URL,
				IsAccessible: isAccessible,
				ResponseTime: responseTime,
				StatusCode:   statusCode,
				Error:        err,
				NeedAlert:    httpConfig.NeedAlert,
				AllowedCodes: httpConfig.AllowedCodes,
			})
		}
		httpMetrics.Interfaces = httpInterfaces
	}

	return httpMetrics
}

// GetHostIP 获取本机 IP 地址（优先获取非回环的IPv4地址）
func GetHostIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Printf("获取网络接口地址失败: %v", err)
		return "unknown-ip"
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String()
			}
		}
	}

	// 如果没找到，尝试通过连接外部地址获取本机IP
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Printf("获取本机IP失败: %v", err)
		return "unknown-ip"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

// GetHostName 获取本机主机名
func GetHostName() string {
	hostname, err := os.Hostname()
	if err != nil {
		log.Printf("获取主机名失败: %v", err)
		return "unknown-host"
	}
	return hostname
}
