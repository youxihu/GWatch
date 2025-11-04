// Package usecase internal/app/usecase/scheduled_push_server.go
package usecase

import (
	"GWatch/internal/domain/scheduled_push/common"
	"GWatch/internal/domain/scheduled_push/server"
	"GWatch/internal/entity"
	"fmt"
	"log"
	"time"
)

// ServerUseCaseImpl 服务端模式用例实现
type ServerUseCaseImpl struct {
	metricsCollector        *MetricsCollector
	clientDataRepository    common.ClientDataRepository
	scheduledPushFormatter  common.ScheduledPushFormatter
	notifier                Notifier
	dataLogStorage          common.ScheduledPushDataLogStorage
}

// NewServerUseCase 创建服务端模式用例
func NewServerUseCase(
	metricsCollector *MetricsCollector,
	clientDataRepository common.ClientDataRepository,
	scheduledPushFormatter common.ScheduledPushFormatter,
	notifier Notifier,
	dataLogStorage common.ScheduledPushDataLogStorage,
) server.ServerUseCase {
	return &ServerUseCaseImpl{
		metricsCollector:       metricsCollector,
		clientDataRepository:   clientDataRepository,
		scheduledPushFormatter: scheduledPushFormatter,
		notifier:               notifier,
		dataLogStorage:         dataLogStorage,
	}
}

// Run 执行服务端模式：从 Redis 读取数据并聚合成报告发送
func (su *ServerUseCaseImpl) Run(config *entity.Config) error {
	// 明确记录：Server模式会发送通知
	log.Printf("[Server模式] 开始执行：将从Redis读取客户端数据并聚合成报告发送通知")
	
	// 初始化 Repository（如果未初始化）
	if su.clientDataRepository == nil {
		return fmt.Errorf("clientDataRepository 未初始化")
	}

	// 初始化 Redis 连接
	if err := su.clientDataRepository.Init(config); err != nil {
		return fmt.Errorf("初始化 Redis 连接失败: %v", err)
	}

	// 获取所有客户端数据的 keys
	keys, err := su.clientDataRepository.GetAllClientDataKeys()
	if err != nil {
		log.Printf("[Server模式] 获取客户端数据 keys 失败: %v，将继续收集Server自己的数据", err)
		keys = []string{} // 设置为空，继续执行
	}

	log.Printf("[Server模式] 从Redis获取到 %d 个客户端数据key", len(keys))

	// 读取所有客户端数据
	var clientDataList []*entity.ClientMonitorData
	validKeys := []string{}
	if len(keys) > 0 {
		for _, key := range keys {
			clientData, err := su.clientDataRepository.GetClientDataByKey(key)
			if err != nil {
				log.Printf("[Server模式] 读取客户端数据失败 (key=%s): %v", key, err)
				continue
			}
			if clientData != nil {
				clientDataList = append(clientDataList, clientData)
				validKeys = append(validKeys, key)
				log.Printf("[Server模式] 成功读取客户端数据: %s (%s, Title: %s)", 
					clientData.HostIP, clientData.HostName, clientData.Title)
			}
		}
	}

	if len(clientDataList) == 0 {
		log.Println("[Server模式] 暂无客户端数据，将只发送Server自己的监控数据")
	}

	// 收集Server自己的监控数据并合并到聚合报告中
	serverIP := GetHostIP()
	serverHostName := GetHostName()
	
	log.Printf("[Server模式] Server自身IP: %s, HostName: %s", serverIP, serverHostName)

	// 收集Server的主机监控数据
	serverHostMetrics := su.metricsCollector.CollectBasicHostMetrics()

	// 构建Server的监控数据
	serverMetrics := &entity.ClientMetrics{
		CPU:     &serverHostMetrics.CPU,
		Memory:  &serverHostMetrics.Memory,
		Disk:    &serverHostMetrics.Disk,
		Network: &serverHostMetrics.Network,
	}

	// 如果配置了包含应用监控，收集Server的应用监控数据
	if config.ScheduledPush != nil && config.ScheduledPush.IncludeAppMonitoring {
		log.Printf("[Server模式] 配置了包含应用监控，开始收集Server的应用监控数据")
		
		// 收集Redis指标
		if config.AppMonitoring != nil && config.AppMonitoring.Redis != nil && config.AppMonitoring.Redis.Enabled {
			redisMetrics := su.metricsCollector.CollectRedisMetrics(config)
			if redisMetrics != nil {
				serverMetrics.Redis = redisMetrics
				log.Printf("[Server模式] 已收集Redis指标")
			}
		}

		// 收集MySQL指标
		if config.AppMonitoring != nil && config.AppMonitoring.MySQL != nil && config.AppMonitoring.MySQL.Enabled {
			mysqlMetrics := su.metricsCollector.CollectMySQLMetrics(config)
			if mysqlMetrics != nil {
				serverMetrics.MySQL = mysqlMetrics
				log.Printf("[Server模式] 已收集MySQL指标")
			}
		}

		// 收集HTTP指标
		if config.AppMonitoring != nil && config.AppMonitoring.HTTP != nil && config.AppMonitoring.HTTP.Enabled {
			httpMetrics := su.metricsCollector.CollectHTTPMetrics(config)
			if httpMetrics != nil {
				serverMetrics.HTTP = httpMetrics
				log.Printf("[Server模式] 已收集HTTP指标")
			}
		}
	}

	// 获取server配置的title
	serverTitle := ""
	if config.ScheduledPush != nil && config.ScheduledPush.Title != "" {
		serverTitle = config.ScheduledPush.Title
	}

	// 构建Server的监控数据对象
	serverData := &entity.ClientMonitorData{
		HostIP:    serverIP,
		HostName:  serverHostName,
		Title:     serverTitle,
		Timestamp: time.Now(),
		Metrics:   serverMetrics,
	}

	// 检查clientDataList中是否已经有Server自己的数据（通过IP+Title组合判断，因为同一机器可能有多个实例）
	var serverDataIndex int = -1
	for i, data := range clientDataList {
		// 通过 IP + Title 组合判断是否是同一个实例
		if data.HostIP == serverIP && data.Title == serverTitle {
			// 如果已经存在相同的实例，更新它
			clientDataList[i] = serverData
			serverDataIndex = i
			log.Printf("[Server模式] 发现Server自己的数据已在Redis中，将更新: %s (%s, Title: %s)", 
				serverIP, serverHostName, serverTitle)
			break
		}
	}

	// 如果没有找到Server自己的数据，添加到列表（允许同一机器上有多个不同title的实例）
	if serverDataIndex == -1 {
		clientDataList = append(clientDataList, serverData)
		log.Printf("[Server模式] Server自己的数据未在Redis中找到，将添加到聚合列表: %s (%s, Title: %s)", 
			serverIP, serverHostName, serverTitle)
	}

	// 如果没有任何数据（包括Server自己的数据），返回
	if len(clientDataList) == 0 {
		log.Println("[Server模式] 未找到有效的监控数据，不发送通知")
		return nil
	}

	// 格式化并发送报告
	// 通知标题直接从配置的 title 字段获取，不使用主机名
	title := su.getScheduledPushTitle(config)
	
	log.Printf("[Server模式] 准备发送聚合报告，标题: %s，包含 %d 台主机的数据", title, len(clientDataList))
	for i, data := range clientDataList {
		log.Printf("[Server模式] 聚合报告主机 %d: IP=%s, HostName=%s, Title=%s", 
			i+1, data.HostIP, data.HostName, data.Title)
	}
	
	// 格式化报告（传入title用于每个主机的二级标题）
	report := su.scheduledPushFormatter.FormatClientReport(clientDataList, title)

	// 发送通知（使用配置的title作为通知标题）
	log.Printf("[Server模式] 正在发送通知到钉钉，标题: %s", title)
	if err := su.notifier.Send(title, report); err != nil {
		return fmt.Errorf("发送合并报告失败: %v", err)
	}

	log.Printf("[Server模式] ✅ 成功发送合并报告到钉钉，标题: %s，包含 %d 台主机的数据", title, len(clientDataList))

	// 保存报告到数据日志文件（如果启用）
	if su.dataLogStorage != nil {
		if err := su.dataLogStorage.Init(config); err != nil {
			log.Printf("[Server模式] 初始化数据日志存储失败: %v", err)
		} else {
			reportTimestamp := time.Now()
			if err := su.dataLogStorage.SaveServerReport(report, title, reportTimestamp); err != nil {
				log.Printf("[Server模式] 保存报告日志失败: %v", err)
			} else {
				log.Printf("[Server模式] 已保存报告日志")
				// 清理过期日志（后台执行，不阻塞）
				go func() {
					if err := su.dataLogStorage.CleanupOldLogs(); err != nil {
						log.Printf("[Server模式] 清理过期日志失败: %v", err)
					} else {
						log.Printf("[Server模式] 已清理过期日志")
					}
				}()
			}
		}
	}

	// 清理已发送的数据（可选）
	for _, key := range validKeys {
		if err := su.clientDataRepository.DeleteClientData(key); err != nil {
			log.Printf("[Server模式] 删除客户端数据失败 (key=%s): %v", key, err)
		}
	}

	return nil
}

// getScheduledPushTitle 获取全局定时推送标题
// 直接从配置的 scheduled_push.title 字段获取，不使用主机名
func (su *ServerUseCaseImpl) getScheduledPushTitle(config *entity.Config) string {
	if config.ScheduledPush != nil && config.ScheduledPush.Title != "" {
		return config.ScheduledPush.Title
	}
	return "系统监控定时报告"
}
