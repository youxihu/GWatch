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
	"net"
	"os"
	"time"
)

// ScheduledPushUseCaseImpl 全局定时推送用例实现
type ScheduledPushUseCaseImpl struct {
	hostCollector         collector.HostCollector
	redisClient           RedisClient
	httpCollector         collector.HTTPCollector
	tickerCollector       ticker.TickerCollector
	tokenProvider         ticker.TokenProvider
	systemMetricsService  *SystemMetricsService
	evaluator             monitoring.Evaluator
	formatter             monitoring.Formatter
	notifier              Notifier
	alertStorage          scheduled_push.ScheduledPushAlertStorage
	clientDataRepository  scheduled_push.ClientDataRepository
	scheduledPushFormatter scheduled_push.ScheduledPushFormatter
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
	clientDataRepository scheduled_push.ClientDataRepository,
	scheduledPushFormatter scheduled_push.ScheduledPushFormatter,
) scheduled_push.ScheduledPushUseCase {
	return &ScheduledPushUseCaseImpl{
		hostCollector:          hostCollector,
		redisClient:            redisClient,
		httpCollector:          httpCollector,
		tickerCollector:        tickerCollector,
		tokenProvider:          tokenProvider,
		systemMetricsService:   systemMetricsService,
		evaluator:              evaluator,
		formatter:              formatter,
		notifier:               notifier,
		alertStorage:           alertStorage,
		clientDataRepository:   clientDataRepository,
		scheduledPushFormatter: scheduledPushFormatter,
	}
}

// RunScheduledPush 执行全局定时推送（根据 mode 决定是 client 还是 server 模式）
func (spu *ScheduledPushUseCaseImpl) RunScheduledPush(config *entity.Config) error {
	if config.ScheduledPush == nil {
		return fmt.Errorf("scheduled_push 配置不存在")
	}

	mode := config.ScheduledPush.Mode
	if mode == "" {
		mode = "client" // 默认是 client 模式
	}

	switch mode {
	case "client":
		return spu.RunClientMode(config)
	case "server":
		return spu.RunServerMode(config)
	default:
		return fmt.Errorf("不支持的模式: %s", mode)
	}
}

// getScheduledPushTitle 获取全局定时推送标题
// 直接从配置的 scheduled_push.title 字段获取，不使用主机名
func (spu *ScheduledPushUseCaseImpl) getScheduledPushTitle(config *entity.Config) string {
	if config.ScheduledPush != nil && config.ScheduledPush.Title != "" {
		return config.ScheduledPush.Title
	}
	return "系统监控定时报告"
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

	if config.AppMonitoring != nil && config.AppMonitoring.HTTP != nil {
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
		mode := config.ScheduledPush.Mode
		if mode == "" {
			mode = "client" // 默认是 client 模式
		}

		// Server模式：延迟执行聚合，等待所有Client上传完数据
		if mode == "server" {
			delaySeconds := config.ScheduledPush.ServerAggregationDelaySeconds
			if delaySeconds <= 0 {
				delaySeconds = 60 // 默认延迟60秒
			}
			
			log.Printf("%s (Server模式，将延迟%d秒后聚合)", logPrefix, delaySeconds)
			
			// 异步延迟执行，避免阻塞调度器
			go func() {
				time.Sleep(time.Duration(delaySeconds) * time.Second)
				log.Printf("[Server模式] 延迟等待完成，开始聚合数据")
				if err := sps.scheduledPushUseCase.RunScheduledPush(config); err != nil {
					log.Printf("执行全局定时推送失败: %v", err)
				} else {
					log.Println("全局定时推送发送成功")
				}
			}()
		} else {
			// Client模式：立即执行
			log.Printf("%s", logPrefix)
			if err := sps.scheduledPushUseCase.RunScheduledPush(config); err != nil {
				log.Printf("执行全局定时推送失败: %v", err)
			} else {
				log.Println("全局定时推送发送成功")
			}
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

// RunClientMode 运行客户端模式：收集数据并上传到 Redis
func (spu *ScheduledPushUseCaseImpl) RunClientMode(config *entity.Config) error {
	// 明确记录：Client模式绝对不应该发送通知
	log.Printf("[Client模式] 开始执行：只上传数据到Redis，不会发送任何通知")
	
	// 初始化 Repository（如果未初始化）
	if spu.clientDataRepository == nil {
		return fmt.Errorf("clientDataRepository 未初始化")
	}

	// 初始化 Redis 连接
	if err := spu.clientDataRepository.Init(config); err != nil {
		return fmt.Errorf("初始化 Redis 连接失败: %v", err)
	}

	// 收集主机监控指标（Client模式目前只收集主机监控数据）
	hostMetrics := spu.collectBasicHostMetrics()

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
		HostIP:    getHostIP(),
		HostName:  getHostName(),
		Title:     clientTitle,
		Timestamp: time.Now(),
		Metrics:   clientMetrics,
	}

	// 保存到 Redis，设置 5 分钟过期时间
	if err := spu.clientDataRepository.SaveClientData(clientData, 5*time.Minute); err != nil {
		return fmt.Errorf("保存客户端数据到 Redis 失败: %v", err)
	}

	log.Printf("[Client模式] 成功上报监控数据到 Redis: %s (%s, Title: %s)，注意：Client模式不会发送任何通知", 
		clientData.HostIP, clientData.HostName, clientData.Title)
	return nil
}

// RunServerMode 运行服务端模式：从 Redis 读取数据并聚合成报告发送
func (spu *ScheduledPushUseCaseImpl) RunServerMode(config *entity.Config) error {
	// 明确记录：Server模式会发送通知
	log.Printf("[Server模式] 开始执行：将从Redis读取客户端数据并聚合成报告发送通知")
	
	// 初始化 Repository（如果未初始化）
	if spu.clientDataRepository == nil {
		return fmt.Errorf("clientDataRepository 未初始化")
	}

	// 初始化 Redis 连接
	if err := spu.clientDataRepository.Init(config); err != nil {
		return fmt.Errorf("初始化 Redis 连接失败: %v", err)
	}

	// 获取所有客户端数据的 keys
	keys, err := spu.clientDataRepository.GetAllClientDataKeys()
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
			clientData, err := spu.clientDataRepository.GetClientDataByKey(key)
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
	serverIP := getHostIP()
	serverHostName := getHostName()
	
	log.Printf("[Server模式] Server自身IP: %s, HostName: %s", serverIP, serverHostName)

	// 收集Server的主机监控数据
	serverHostMetrics := spu.collectBasicHostMetrics()

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
			redisMetrics := spu.collectRedisMetrics(config)
			if redisMetrics != nil {
				serverMetrics.Redis = redisMetrics
				log.Printf("[Server模式] 已收集Redis指标")
			}
		}

		// 收集MySQL指标
		if config.AppMonitoring != nil && config.AppMonitoring.MySQL != nil && config.AppMonitoring.MySQL.Enabled {
			mysqlMetrics := spu.collectMySQLMetrics(config)
			if mysqlMetrics != nil {
				serverMetrics.MySQL = mysqlMetrics
				log.Printf("[Server模式] 已收集MySQL指标")
			}
		}

		// 收集HTTP指标
		if config.AppMonitoring != nil && config.AppMonitoring.HTTP != nil && config.AppMonitoring.HTTP.Enabled {
			httpMetrics := spu.collectHTTPMetrics(config)
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

	// 检查clientDataList中是否已经有Server自己的数据（通过IP判断）
	var serverDataIndex int = -1
	for i, data := range clientDataList {
		if data.HostIP == serverIP {
			// 如果已经存在，更新它
			clientDataList[i] = serverData
			serverDataIndex = i
			log.Printf("[Server模式] 发现Server自己的数据已在Redis中，将更新: %s (%s, Title: %s)", 
				serverIP, serverHostName, serverTitle)
			break
		}
	}

	// 如果没有找到Server自己的数据，添加到列表
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
	title := spu.getScheduledPushTitle(config)
	
	log.Printf("[Server模式] 准备发送聚合报告，标题: %s，包含 %d 台主机的数据", title, len(clientDataList))
	for i, data := range clientDataList {
		log.Printf("[Server模式] 聚合报告主机 %d: IP=%s, HostName=%s, Title=%s", 
			i+1, data.HostIP, data.HostName, data.Title)
	}
	
	// 格式化报告（传入title用于每个主机的二级标题）
	report := spu.scheduledPushFormatter.FormatClientReport(clientDataList, title)

	// 发送通知（使用配置的title作为通知标题）
	log.Printf("[Server模式] 正在发送通知到钉钉，标题: %s", title)
	if err := spu.notifier.Send(title, report); err != nil {
		return fmt.Errorf("发送合并报告失败: %v", err)
	}

	log.Printf("[Server模式] ✅ 成功发送合并报告到钉钉，标题: %s，包含 %d 台主机的数据", title, len(clientDataList))

	// 清理已发送的数据（可选）
	for _, key := range validKeys {
		if err := spu.clientDataRepository.DeleteClientData(key); err != nil {
			log.Printf("[Server模式] 删除客户端数据失败 (key=%s): %v", key, err)
		}
	}

	return nil
}

// getHostIP 获取本机 IP 地址（优先获取非回环的IPv4地址）
func getHostIP() string {
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

// getHostName 获取本机主机名
func getHostName() string {
	hostname, err := os.Hostname()
	if err != nil {
		log.Printf("获取主机名失败: %v", err)
		return "unknown-host"
	}
	return hostname
}
