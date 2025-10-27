// Package usecase internal/app/usecase/scheduled_push.go
package usecase

import (
	"GWatch/internal/domain/monitoring"
	"GWatch/internal/domain/collector"
	"GWatch/internal/domain/scheduled_push"
	"GWatch/internal/domain/ticker"
	"GWatch/internal/entity"
	"fmt"
	"log"
	"time"
	"crypto/rand"
	"encoding/hex"
)

// ScheduledPushUseCaseImpl 全局定时推送用例实现
type ScheduledPushUseCaseImpl struct {
	hostCollector        collector.HostCollector
	redisClient          RedisClient
	httpCollector        collector.HTTPCollector
	tickerCollector      ticker.TickerCollector
	tokenProvider        ticker.TokenProvider
	systemMetricsService *SystemMetricsService
	evaluator            monitoring.Evaluator
	formatter            monitoring.Formatter
	notifier             Notifier
	alertStorage         scheduled_push.ScheduledPushAlertStorage
}

// NewScheduledPushUseCase 创建全局定时推送用例
func NewScheduledPushUseCase(
	hostCollector collector.HostCollector,
	redisClient RedisClient,
	httpCollector collector.HTTPCollector,
	tickerCollector ticker.TickerCollector,
	tokenProvider ticker.TokenProvider,
	systemMetricsService *SystemMetricsService,
	evaluator monitoring.Evaluator,
	formatter monitoring.Formatter,
	notifier Notifier,
	alertStorage scheduled_push.ScheduledPushAlertStorage,
) scheduled_push.ScheduledPushUseCase {
	return &ScheduledPushUseCaseImpl{
		hostCollector:        hostCollector,
		redisClient:          redisClient,
		httpCollector:        httpCollector,
		tickerCollector:      tickerCollector,
		tokenProvider:        tokenProvider,
		systemMetricsService: systemMetricsService,
		evaluator:            evaluator,
		formatter:            formatter,
		notifier:             notifier,
		alertStorage:         alertStorage,
	}
}

// CollectAllMetrics 收集所有监控指标
func (spu *ScheduledPushUseCaseImpl) CollectAllMetrics(config *entity.Config) (*entity.ScheduledPushMetrics, error) {
	metrics := &entity.ScheduledPushMetrics{
		Timestamp: time.Now(),
	}

	// 收集主机监控指标
	if config.ScheduledPush != nil && config.ScheduledPush.IncludeHostMonitoring {
		// 只收集基本的主机指标，避免Redis等外部依赖
		hostMetrics := spu.collectBasicHostMetrics()
		metrics.HostMetrics = hostMetrics
	}

	// 收集应用监控指标
	if config.ScheduledPush != nil && config.ScheduledPush.IncludeAppMonitoring {
		appMetrics := &entity.AppMetrics{}

		// 收集Redis指标
		if config.AppMonitoring != nil && config.AppMonitoring.Redis != nil {
			redisMetrics := spu.collectRedisMetrics(config)
			appMetrics.Redis = redisMetrics
		}

		// 收集MySQL指标
		if config.AppMonitoring != nil && config.AppMonitoring.MySQL != nil {
			mysqlMetrics := spu.collectMySQLMetrics(config)
			appMetrics.MySQL = mysqlMetrics
		}

		// 收集HTTP指标
		if config.AppMonitoring != nil && config.AppMonitoring.HTTP != nil {
			httpMetrics := spu.collectHTTPMetrics(config)
			appMetrics.HTTP = httpMetrics
		}

		metrics.AppMetrics = appMetrics
	}

	// 收集定时器指标
	if config.AppMonitoring != nil && config.AppMonitoring.Tickers != nil {
		tickerMetrics := spu.collectTickerMetrics(config)
		metrics.TickerMetrics = tickerMetrics
	}

	return metrics, nil
}

// collectBasicHostMetrics 收集基本主机指标（不依赖外部服务）
func (spu *ScheduledPushUseCaseImpl) collectBasicHostMetrics() *entity.SystemMetrics {
	// 只收集CPU、内存、磁盘、网络等基本指标
	hostMetrics := &entity.SystemMetrics{
		Timestamp: time.Now(),
	}

	// 收集CPU指标
	cpuPercent, err := spu.hostCollector.GetCPUPercent()
	hostMetrics.CPU = entity.CPUMetrics{
		Percent: cpuPercent,
		Error:   err,
	}

	// 收集内存指标
	memPercent, usedMB, totalMB, err := spu.hostCollector.GetMemoryUsage()
	hostMetrics.Memory = entity.MemoryMetrics{
		Percent: memPercent,
		UsedMB:  usedMB,
		TotalMB: totalMB,
		Error:   err,
	}

	// 收集磁盘指标
	diskPercent, usedGB, totalGB, err := spu.hostCollector.GetDiskUsage()
	hostMetrics.Disk = entity.DiskMetrics{
		Percent: diskPercent,
		UsedGB:  usedGB,
		TotalGB: totalGB,
		Error:   err,
	}

	// 收集磁盘IO指标
	readKBps, writeKBps, err := spu.hostCollector.GetDiskIORate()
	if err == nil {
		hostMetrics.Disk.ReadKBps = readKBps
		hostMetrics.Disk.WriteKBps = writeKBps
	}

	// 收集网络指标
	downloadKBps, uploadKBps, err := spu.hostCollector.GetNetworkRate()
	hostMetrics.Network = entity.NetworkMetrics{
		DownloadKBps: downloadKBps,
		UploadKBps:   uploadKBps,
		Error:        err,
	}

	return hostMetrics
}

// collectRedisMetrics 收集Redis指标
func (spu *ScheduledPushUseCaseImpl) collectRedisMetrics(config *entity.Config) *entity.RedisMetrics {
	redisMetrics := &entity.RedisMetrics{
		ClientCount: 0,
	}

	// 初始化Redis连接
	if err := spu.redisClient.Init(); err != nil {
		redisMetrics.ConnectionError = err
		return redisMetrics
	}

	// 获取Redis连接数
	clientCount, err := spu.redisClient.GetClients()
	if err != nil {
		redisMetrics.ConnectionError = err
	} else {
		redisMetrics.ClientCount = clientCount
	}

	// 获取Redis连接详情
	clientDetails, err := spu.redisClient.GetClientsDetail()
	if err != nil {
		redisMetrics.DetailError = err
	} else {
		redisMetrics.ClientDetails = clientDetails
	}

	return redisMetrics
}

// collectMySQLMetrics 收集MySQL指标
func (spu *ScheduledPushUseCaseImpl) collectMySQLMetrics(config *entity.Config) *entity.MySQLMetrics {
	mysqlMetrics := &entity.MySQLMetrics{
		Error: nil,
	}

	// 这里可以添加MySQL连接和指标收集逻辑
	// 简化处理，只设置基本状态

	return mysqlMetrics
}

// collectHTTPMetrics 收集HTTP指标
func (spu *ScheduledPushUseCaseImpl) collectHTTPMetrics(config *entity.Config) *entity.HTTPMetrics {
	httpMetrics := &entity.HTTPMetrics{
		Interfaces: []entity.HTTPInterfaceMetrics{},
		Error:      nil,
	}

	if config.AppMonitoring.HTTP != nil {
		// 初始化HTTP收集器
		if err := spu.httpCollector.Init(); err != nil {
			httpMetrics.Error = err
			return httpMetrics
		}

		// 检查每个HTTP接口
		var httpInterfaces []entity.HTTPInterfaceMetrics
		for _, httpConfig := range config.AppMonitoring.HTTP.Interfaces {
			isAccessible, responseTime, statusCode, err := spu.httpCollector.CheckInterface(httpConfig.URL, httpConfig.Timeout)
			
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

// collectTickerMetrics 收集定时器指标
func (spu *ScheduledPushUseCaseImpl) collectTickerMetrics(config *entity.Config) *entity.TickerMetrics {
	tickerMetrics := &entity.TickerMetrics{
		Timestamp: time.Now(),
	}

	if config.AppMonitoring.Tickers != nil && len(config.AppMonitoring.Tickers.TickerInterfaces) > 0 {
		interfaces := make([]entity.TickerInterfaceMetrics, len(config.AppMonitoring.Tickers.TickerInterfaces))

		for i, tickerConfig := range config.AppMonitoring.Tickers.TickerInterfaces {
			// 简化处理，只设置基本信息
			interfaces[i] = entity.TickerInterfaceMetrics{
				Name:         tickerConfig.Name,
				URL:          tickerConfig.DeviceURL,
				IsAccessible: true, // 简化处理
				Error:        nil,
			}
		}

		tickerMetrics.Interfaces = interfaces
	}

	return tickerMetrics
}

// RunScheduledPush 执行全局定时推送
func (spu *ScheduledPushUseCaseImpl) RunScheduledPush(config *entity.Config) error {
	metrics, err := spu.CollectAllMetrics(config)
	if err != nil {
		return fmt.Errorf("收集监控指标失败: %v", err)
	}

	title := spu.getScheduledPushTitle(config)
	report := spu.buildScheduledPushReport(config, metrics, title)

	// 发送通知
	err = spu.notifier.Send(title, report)
	if err != nil {
		log.Printf("发送全局定时推送失败: %v", err)
	}

	// 保存告警信息
	if config.ScheduledPush != nil && config.ScheduledPush.AlertStorage != nil && config.ScheduledPush.AlertStorage.Enabled {
		spu.saveScheduledPushAlert(config, title, report, metrics)
	}

	return err
}

// getScheduledPushTitle 获取全局定时推送标题
func (spu *ScheduledPushUseCaseImpl) getScheduledPushTitle(config *entity.Config) string {
	if config.ScheduledPush != nil && config.ScheduledPush.Title != "" {
		return config.ScheduledPush.Title
	}
	return "系统监控定时报告"
}

// buildScheduledPushReport 构建全局定时推送报告
func (spu *ScheduledPushUseCaseImpl) buildScheduledPushReport(config *entity.Config, metrics *entity.ScheduledPushMetrics, title string) string {
	// 将应用监控数据合并到主机指标中
	combinedMetrics := metrics.HostMetrics
	if metrics.AppMetrics != nil {
		// 合并Redis数据
		if metrics.AppMetrics.Redis != nil {
			combinedMetrics.Redis = *metrics.AppMetrics.Redis
		}
		// 合并MySQL数据
		if metrics.AppMetrics.MySQL != nil {
			combinedMetrics.MySQL = *metrics.AppMetrics.MySQL
		}
		// 合并HTTP数据
		if metrics.AppMetrics.HTTP != nil {
			combinedMetrics.HTTP = *metrics.AppMetrics.HTTP
		}
	}
	
	// 使用默认的formatter构建报告
	return spu.formatter.Build(title, config, combinedMetrics, []monitoring.TriggeredAlert{})
}

// saveScheduledPushAlert 保存全局定时推送告警信息
func (spu *ScheduledPushUseCaseImpl) saveScheduledPushAlert(config *entity.Config, title, report string, metrics *entity.ScheduledPushMetrics) {
	if spu.alertStorage == nil {
		return
	}

	// 生成唯一ID
	id := spu.generateAlertID()
	
	// 获取当前推送时间
	pushTime := spu.getCurrentPushTime(config)
	
	// 创建告警记录
	alertRecord := entity.NewScheduledPushAlertRecord(id, title, report, pushTime)

	// 保存告警记录
	if err := spu.alertStorage.SaveScheduledPushAlert(alertRecord); err != nil {
		log.Printf("保存全局定时推送告警信息失败: %v", err)
	} else {
		log.Printf("全局定时推送告警信息已保存: %s (推送时间: %s)", title, pushTime)
	}
}

// generateAlertID 生成告警ID
func (spu *ScheduledPushUseCaseImpl) generateAlertID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// getCurrentPushTime 获取当前推送时间
func (spu *ScheduledPushUseCaseImpl) getCurrentPushTime(config *entity.Config) string {
	now := time.Now()
	currentTime := fmt.Sprintf("%d:%02d", now.Hour(), now.Minute())
	
	// 在配置的推送时间中查找匹配的时间
	if config.ScheduledPush != nil {
		for _, pushTime := range config.ScheduledPush.PushTimes {
			if currentTime == pushTime {
				return pushTime
			}
		}
	}
	
	return currentTime
}

// ScheduledPushSchedulerImpl 全局定时推送调度器实现
type ScheduledPushSchedulerImpl struct {
	scheduledPushUseCase scheduled_push.ScheduledPushUseCase
	config               *entity.Config
	ticker               *time.Ticker
	stopCh               chan struct{}
	lastReported         map[string]time.Time // 记录每个时间点最后报告的时间
}

// NewScheduledPushScheduler 创建全局定时推送调度器
func NewScheduledPushScheduler(scheduledPushUseCase scheduled_push.ScheduledPushUseCase) scheduled_push.ScheduledPushScheduler {
	return &ScheduledPushSchedulerImpl{
		scheduledPushUseCase: scheduledPushUseCase,
		stopCh:               make(chan struct{}),
		lastReported:         make(map[string]time.Time),
	}
}

// Start 启动全局定时推送调度
func (sps *ScheduledPushSchedulerImpl) Start(config *entity.Config, stopCh <-chan struct{}) error {
	sps.config = config

	// 每10秒检查一次是否到了推送时间，提高响应速度
	sps.ticker = time.NewTicker(10 * time.Second)

	go func() {
		defer sps.ticker.Stop()

		// 启动时立即检查一次，避免错过推送时间
		log.Println("启动时检查全局定时推送时间...")
		sps.executeScheduledPushIfNeeded(config, "启动时匹配到推送时间，立即执行全局监控报告")

		for {
			select {
			case <-sps.ticker.C:
				// 检查是否到了推送时间
				sps.executeScheduledPushIfNeeded(config, "定时器触发：开始执行全局监控报告")
			case <-stopCh:
				log.Println("全局定时推送调度器收到停止信号")
				return
			case <-sps.stopCh:
				log.Println("全局定时推送调度器停止")
				return
			}
		}
	}()

	return nil
}

// executeScheduledPushIfNeeded 如果需要则执行全局定时推送
func (sps *ScheduledPushSchedulerImpl) executeScheduledPushIfNeeded(config *entity.Config, logPrefix string) {
	if config.ScheduledPush == nil || !config.ScheduledPush.Enabled {
		return
	}

	if sps.IsTimeToPush(config.ScheduledPush.PushTimes) {
		log.Printf("%s", logPrefix)
		if err := sps.scheduledPushUseCase.RunScheduledPush(config); err != nil {
			log.Printf("执行全局定时推送失败: %v", err)
		} else {
			log.Println("全局定时推送发送成功")
		}
	}
}

// Stop 停止全局定时推送调度
func (sps *ScheduledPushSchedulerImpl) Stop() error {
	close(sps.stopCh)
	return nil
}

// IsTimeToPush 检查是否到了推送时间
func (sps *ScheduledPushSchedulerImpl) IsTimeToPush(pushTimes []string) bool {
	now := time.Now()
	currentTime := fmt.Sprintf("%d:%02d", now.Hour(), now.Minute())

	log.Printf("检查全局推送时间: 当前时间=%s, 配置时间=%v", currentTime, pushTimes)

	for _, pushTime := range pushTimes {
		if currentTime == pushTime {
			// 检查是否已经在这个时间点推送过
			if lastReported, exists := sps.lastReported[pushTime]; exists {
				// 如果上次推送时间与当前时间在同一分钟内，则不重复推送
				if now.Truncate(time.Minute).Equal(lastReported.Truncate(time.Minute)) {
					log.Printf("时间点 %s 已在本分钟内推送过，跳过", pushTime)
					continue
				}
			}

			log.Printf("匹配到推送时间: %s", pushTime)
			// 记录推送时间
			sps.lastReported[pushTime] = now
			return true
		}
	}

	return false
}
