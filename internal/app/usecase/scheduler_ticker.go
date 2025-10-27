// Package usecase internal/app/usecase/ticker.go
package usecase

import (
	"GWatch/internal/domain/monitoring"
	"GWatch/internal/domain/ticker"
	"GWatch/internal/entity"
	"GWatch/internal/utils"
	"fmt"
	"log"
	"sync"
	"time"
)

// TickerUseCaseImpl 定时器用例实现
type TickerUseCaseImpl struct {
	tickerCollector      ticker.TickerCollector
	tokenProvider        ticker.TokenProvider
	systemMetricsService *SystemMetricsService
	evaluator            monitoring.Evaluator
	formatter            monitoring.Formatter
	tickerFormatter      monitoring.TickerFormatter
	notifier             Notifier
}

// NewTickerUseCase 创建定时器用例
func NewTickerUseCase(
	tickerCollector ticker.TickerCollector,
	tokenProvider ticker.TokenProvider,
	systemMetricsService *SystemMetricsService,
	evaluator monitoring.Evaluator,
	formatter monitoring.Formatter,
	tickerFormatter monitoring.TickerFormatter,
	notifier Notifier,
) ticker.TickerUseCase {
	return &TickerUseCaseImpl{
		tickerCollector:      tickerCollector,
		tokenProvider:        tokenProvider,
		systemMetricsService: systemMetricsService,
		evaluator:            evaluator,
		formatter:            formatter,
		tickerFormatter:      tickerFormatter,
		notifier:             notifier,
	}
}

// CollectTickerMetrics 收集定时器指标
func (tu *TickerUseCaseImpl) CollectTickerMetrics(config *entity.Config) (*entity.TickerMetrics, error) {
	metrics := &entity.TickerMetrics{
		Timestamp: time.Now(),
	}

	// 收集设备状态信息
    if config.AppMonitoring != nil && config.AppMonitoring.Tickers != nil && len(config.AppMonitoring.Tickers.TickerInterfaces) > 0 {
		var wg sync.WaitGroup
        interfaces := make([]entity.TickerInterfaceMetrics, len(config.AppMonitoring.Tickers.TickerInterfaces))

        for i, tickerConfig := range config.AppMonitoring.Tickers.TickerInterfaces {
			wg.Add(1)
			go func(index int, cfg entity.TickerHTTPInterface) {
				defer wg.Done()

				token, tokenErr := tu.tokenProvider.GetToken(cfg.Auth)
				if tokenErr != nil {
					errType := utils.ClassifyError(tokenErr)
					interfaces[index] = entity.TickerInterfaceMetrics{
						Name:                 cfg.Name,
						URL:                  cfg.DeviceURL,
						IsAccessible:         false,
						Error:                tokenErr,
						ErrorType:            errType,
						ChannelOffLineNumber: 0,
						ChannelOnLineNumber:  0,
						TotalDevices:         0,
						OnlineRate:           0,
					}
					return
				}

				// 收集设备状态
				deviceStatus, deviceErr := tu.tickerCollector.CollectDeviceStatusWithToken(cfg, token)

				errType := entity.ErrorTypeNone // 👈 初始化
				if deviceErr != nil {
					log.Printf("[DEBUG] 准备分类 deviceErr: %v", deviceErr) // 👈 加这行
					errType = utils.ClassifyError(deviceErr)
					log.Printf("[DEBUG] 分类结果: %s", errType) // 👈 加这行
				}

				// 接口可用性基于设备状态收集是否成功
				isAccessible := deviceErr == nil && deviceStatus != nil

				interfaces[index] = entity.TickerInterfaceMetrics{
					Name:                 cfg.Name,
					URL:                  cfg.DeviceURL,
					IsAccessible:         isAccessible,
					Error:                deviceErr,
					ErrorType:            errType,
					ChannelOffLineNumber: 0,
					ChannelOnLineNumber:  0,
					TotalDevices:         0,
					OnlineRate:           0,
				}
				log.Printf("[DEBUG] 接口 %s 设置 ErrorType: %s, Error: %v", cfg.Name, errType, deviceErr)
				if deviceErr == nil && deviceStatus != nil {
					interfaces[index].ChannelOffLineNumber = deviceStatus.ChannelOffLineNumber
					interfaces[index].ChannelOnLineNumber = deviceStatus.ChannelOnLineNumber
					interfaces[index].TotalDevices = deviceStatus.TotalDevices
					interfaces[index].OnlineRate = deviceStatus.OnlineRate

					// 设置第一个设备状态作为整体状态
					if index == 0 {
						metrics.DeviceStatus = deviceStatus
					}
				}
			}(i, tickerConfig)
		}

		wg.Wait()
		metrics.Interfaces = interfaces
	}

	return metrics, nil
}

// RunTickerReport 执行定时器报告
func (tu *TickerUseCaseImpl) RunTickerReport(config *entity.Config) error {
	tickerMetrics, err := tu.CollectTickerMetrics(config)
	if err != nil {
		return fmt.Errorf("收集定时器指标失败: %v", err)
	}

	systemMetrics := tu.systemMetricsService.CollectBasicMetrics()

	title := tu.getTickerTitle(config)

	// 👇 检查是否有 Token/认证错误，标题加【紧急】前缀
	for _, iface := range tickerMetrics.Interfaces {
		if iface.ErrorType == entity.ErrorTypeToken || iface.ErrorType == entity.ErrorTypeUnauthorized {
			title = "【紧急】" + title
			break
		}
	}

	report := tu.buildTickerReport(config, tickerMetrics, systemMetrics, title)

	return tu.notifier.Send(title, report) // 钉钉会显示红色标题！
}

// getTickerTitle 获取定时器报告标题
func (tu *TickerUseCaseImpl) getTickerTitle(config *entity.Config) string {
    if config.AppMonitoring != nil && config.AppMonitoring.Tickers != nil && config.AppMonitoring.Tickers.AlertTitle != "" {
        return config.AppMonitoring.Tickers.AlertTitle
	}
	return "设备状态定时报告"
}

// buildTickerReport 构建定时器报告
func (tu *TickerUseCaseImpl) buildTickerReport(config *entity.Config, tickerMetrics *entity.TickerMetrics, systemMetrics *entity.SystemMetrics, title string) string {
	// 使用ticker格式化器构建报告
	if tu.tickerFormatter != nil {
		return tu.tickerFormatter.BuildTickerReport(title, config, tickerMetrics, systemMetrics)
	}

	// 如果没有ticker格式化器，使用默认的formatter
	return tu.formatter.Build(title, config, systemMetrics, []monitoring.TriggeredAlert{})
}

// TickerSchedulerImpl 定时器调度器实现
type TickerSchedulerImpl struct {
	tickerUseCase ticker.TickerUseCase
	config        *entity.Config
	ticker        *time.Ticker
	stopCh        chan struct{}
	lastReported  map[string]time.Time // 记录每个时间点最后报告的时间
}

// NewTickerScheduler 创建定时器调度器
func NewTickerScheduler(tickerUseCase ticker.TickerUseCase) ticker.TickerScheduler {
	return &TickerSchedulerImpl{
		tickerUseCase: tickerUseCase,
		stopCh:        make(chan struct{}),
		lastReported:  make(map[string]time.Time),
	}
}

// Start 启动定时器调度
func (ts *TickerSchedulerImpl) Start(config *entity.Config, stopCh <-chan struct{}) error {
	ts.config = config

	// 每10秒检查一次是否到了告警时间，提高响应速度
	ts.ticker = time.NewTicker(10 * time.Second)

	go func() {
		defer ts.ticker.Stop()

		// 启动时立即检查一次，避免错过启动时间
		log.Println("启动时检查告警时间...")
		ts.executeTickerReportIfNeeded(config, "启动时匹配到告警时间，立即执行设备状态报告")

		for {
			select {
			case <-ts.ticker.C:
				// 检查所有ticker接口的告警时间
				ts.executeTickerReportIfNeeded(config, "定时器触发：开始执行设备状态报告")
			case <-stopCh:
				log.Println("定时器调度器收到停止信号")
				return
			case <-ts.stopCh:
				log.Println("定时器调度器停止")
				return
			}
		}
	}()

	return nil
}

// executeTickerReportIfNeeded 如果需要则执行定时器报告
func (ts *TickerSchedulerImpl) executeTickerReportIfNeeded(config *entity.Config, logPrefix string) {
    if config.AppMonitoring == nil || config.AppMonitoring.Tickers == nil {
        return
    }
    for _, tickerConfig := range config.AppMonitoring.Tickers.TickerInterfaces {
		if ts.IsTimeToAlert(tickerConfig.AlertTime) {
			log.Printf("%s (接口: %s)", logPrefix, tickerConfig.Name)
			if err := ts.tickerUseCase.RunTickerReport(config); err != nil {
				log.Printf("执行定时器报告失败: %v", err)
			} else {
				log.Println("定时器报告发送成功")
			}
			break // 只执行一次报告
		}
	}
}

// Stop 停止定时器调度
func (ts *TickerSchedulerImpl) Stop() error {
	close(ts.stopCh)
	return nil
}

// IsTimeToAlert 检查是否到了告警时间
func (ts *TickerSchedulerImpl) IsTimeToAlert(alertTimes []string) bool {
	now := time.Now()
	currentTime := fmt.Sprintf("%d:%02d", now.Hour(), now.Minute())

	log.Printf("检查告警时间: 当前时间=%s, 配置时间=%v", currentTime, alertTimes)

	for _, alertTime := range alertTimes {
		if currentTime == alertTime {
			// 检查是否已经在这个时间点报告过
			if lastReported, exists := ts.lastReported[alertTime]; exists {
				// 如果上次报告时间与当前时间在同一分钟内，则不重复报告
				if now.Truncate(time.Minute).Equal(lastReported.Truncate(time.Minute)) {
					log.Printf("时间点 %s 已在本分钟内报告过，跳过", alertTime)
					continue
				}
			}

			log.Printf("匹配到告警时间: %s", alertTime)
			// 记录报告时间
			ts.lastReported[alertTime] = now
			return true
		}
	}

	return false
}
