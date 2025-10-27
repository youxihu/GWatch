package usecase

import (
	"GWatch/internal/domain/collector"
	"GWatch/internal/entity"
	"time"
)

// SystemMetricsService 系统指标服务 - 负责聚合和协调各种指标收集
type SystemMetricsService struct {
	hostCollector collector.HostCollector
	redisClient   RedisClient
	httpCollector collector.HTTPCollector
}

// NewSystemMetricsService 创建系统指标服务
func NewSystemMetricsService(
	hostCollector collector.HostCollector,
	redisClient RedisClient,
	httpCollector collector.HTTPCollector,
) *SystemMetricsService {
	return &SystemMetricsService{
		hostCollector: hostCollector,
		redisClient:   redisClient,
		httpCollector: httpCollector,
	}
}

// CollectBasicMetrics 收集基础系统指标
func (sms *SystemMetricsService) CollectBasicMetrics() *entity.SystemMetrics {
	m := &entity.SystemMetrics{Timestamp: time.Now()}

	// 收集基础指标
	m.CPU.Percent, m.CPU.Error = sms.hostCollector.GetCPUPercent()
	m.Memory.Percent, m.Memory.UsedMB, m.Memory.TotalMB, m.Memory.Error = sms.hostCollector.GetMemoryUsage()
	m.Disk.Percent, m.Disk.UsedGB, m.Disk.TotalGB, m.Disk.Error = sms.hostCollector.GetDiskUsage()
	m.Disk.ReadKBps, m.Disk.WriteKBps, _ = sms.hostCollector.GetDiskIORate()
	m.Network.DownloadKBps, m.Network.UploadKBps, m.Network.Error = sms.hostCollector.GetNetworkRate()

	// 收集Redis信息
	if clientCount, err := sms.redisClient.GetClients(); err == nil {
		m.Redis.ClientCount = clientCount
	}

	return m
}

// CollectFullMetrics 收集完整的系统指标（包括HTTP接口）
func (sms *SystemMetricsService) CollectFullMetrics(config *entity.Config) *entity.SystemMetrics {
    m := sms.CollectBasicMetrics()

    // 收集HTTP接口信息（仅当启用并配置了HTTP）
    if config.AppMonitoring != nil && config.AppMonitoring.HTTP != nil && len(config.AppMonitoring.HTTP.Interfaces) > 0 {
        var httpInterfaces []entity.HTTPInterfaceMetrics
        for _, httpConfig := range config.AppMonitoring.HTTP.Interfaces {
            accessible, responseTime, statusCode, err := sms.httpCollector.CheckInterface(
                httpConfig.URL,
                httpConfig.Timeout,
            )

            httpInterfaces = append(httpInterfaces, entity.HTTPInterfaceMetrics{
                Name:         httpConfig.Name,
                URL:          httpConfig.URL,
                IsAccessible: accessible,
                ResponseTime: responseTime,
                StatusCode:   statusCode,
                Error:        err,
                AllowedCodes: httpConfig.AllowedCodes,
            })
        }
        m.HTTP.Interfaces = httpInterfaces
    }

    return m
}
