// Package usecase internal/app/usecase/scheduled_push_client.go
package usecase

import (
	"GWatch/internal/domain/scheduled_push/client"
	"GWatch/internal/domain/scheduled_push/common"
	"GWatch/internal/entity"
	"fmt"
	"log"
	"time"
)

// ClientUseCaseImpl 客户端模式用例实现
type ClientUseCaseImpl struct {
	metricsCollector     *MetricsCollector
	clientDataRepository common.ClientDataRepository
	dataLogStorage       common.ScheduledPushDataLogStorage
}

// NewClientUseCase 创建客户端模式用例
func NewClientUseCase(
	metricsCollector *MetricsCollector,
	clientDataRepository common.ClientDataRepository,
	dataLogStorage common.ScheduledPushDataLogStorage,
) client.ClientUseCase {
	return &ClientUseCaseImpl{
		metricsCollector:     metricsCollector,
		clientDataRepository: clientDataRepository,
		dataLogStorage:       dataLogStorage,
	}
}

// Run 执行客户端模式：收集数据并上传到 Redis
func (cu *ClientUseCaseImpl) Run(config *entity.Config) error {
	// 明确记录：Client模式绝对不应该发送通知
	log.Printf("[Client模式] 开始执行：只上传数据到Redis，不会发送任何通知")
	
	// 初始化 Repository（如果未初始化）
	if cu.clientDataRepository == nil {
		return fmt.Errorf("clientDataRepository 未初始化")
	}

	// 初始化 Redis 连接
	if err := cu.clientDataRepository.Init(config); err != nil {
		return fmt.Errorf("初始化 Redis 连接失败: %v", err)
	}

	// 收集主机监控指标（Client模式目前只收集主机监控数据）
	hostMetrics := cu.metricsCollector.CollectBasicHostMetrics()

	// 构建客户端数据（目前只包含主机监控）
	clientMetrics := &entity.ClientMetrics{
		CPU:     &hostMetrics.CPU,
		Memory:  &hostMetrics.Memory,
		Disk:    &hostMetrics.Disk,
		Network: &hostMetrics.Network,
		// 注意：目前Client模式只收集主机监控数据，不收集应用监控数据（Redis、MySQL、HTTP）
		// 这些字段为nil，格式化器会根据条件渲染来决定是否显示
		// 未来如果需要，可以在这里根据配置来决定是否收集应用监控数据
	}

	// 获取client配置的title
	clientTitle := ""
	if config.ScheduledPush != nil && config.ScheduledPush.Title != "" {
		clientTitle = config.ScheduledPush.Title
	}

	clientData := &entity.ClientMonitorData{
		HostIP:    GetHostIP(),
		HostName:  GetHostName(),
		Title:     clientTitle,
		Timestamp: time.Now(),
		Metrics:   clientMetrics,
	}

	// 保存到 Redis，设置 5 分钟过期时间
	if err := cu.clientDataRepository.SaveClientData(clientData, 5*time.Minute); err != nil {
		return fmt.Errorf("保存客户端数据到 Redis 失败: %v", err)
	}

	// 保存到数据日志文件（如果启用）
	if cu.dataLogStorage != nil {
		if err := cu.dataLogStorage.Init(config); err != nil {
			log.Printf("[Client模式] 初始化数据日志存储失败: %v", err)
		} else {
			if err := cu.dataLogStorage.SaveClientData(clientData, clientData.Timestamp); err != nil {
				log.Printf("[Client模式] 保存数据日志失败: %v", err)
			} else {
				log.Printf("[Client模式] 已保存数据日志")
			}
		}
	}

	log.Printf("[Client模式] 成功上报监控数据到 Redis: %s (%s, Title: %s)，注意：Client模式不会发送任何通知", 
		clientData.HostIP, clientData.HostName, clientData.Title)
	return nil
}
