// Package usecase internal/app/usecase/ticker.go
package usecase

import (
	"GWatch/internal/domain/alert"
	"GWatch/internal/domain/monitor"
	"GWatch/internal/domain/ticker"
	"GWatch/internal/entity"
	"fmt"
	"log"
	"sync"
	"time"
)

// TickerUseCaseImpl 定时器用例实现
type TickerUseCaseImpl struct {
	tickerCollector     ticker.TickerCollector
	systemMetricsService *SystemMetricsService
	evaluator           monitor.Evaluator
	formatter           alert.Formatter
	tickerFormatter     alert.TickerFormatter
	notifier            Notifier
}

// NewTickerUseCase 创建定时器用例
func NewTickerUseCase(
	tickerCollector ticker.TickerCollector,
	systemMetricsService *SystemMetricsService,
	evaluator monitor.Evaluator,
	formatter alert.Formatter,
	tickerFormatter alert.TickerFormatter,
	notifier Notifier,
) ticker.TickerUseCase {
	return &TickerUseCaseImpl{
		tickerCollector:     tickerCollector,
		systemMetricsService: systemMetricsService,
		evaluator:           evaluator,
		formatter:           formatter,
		tickerFormatter:     tickerFormatter,
		notifier:            notifier,
	}
}

// CollectTickerMetrics 收集定时器指标
func (tu *TickerUseCaseImpl) CollectTickerMetrics(config *entity.Config) (*entity.TickerMetrics, error) {
	metrics := &entity.TickerMetrics{
		Timestamp: time.Now(),
	}

	// 收集设备状态信息
	if len(config.Tickers.HTTPInterfaces) > 0 {
		var wg sync.WaitGroup
		interfaces := make([]entity.TickerInterfaceMetrics, len(config.Tickers.HTTPInterfaces))
		
		for i, tickerConfig := range config.Tickers.HTTPInterfaces {
			wg.Add(1)
			go func(index int, cfg entity.TickerHTTPInterface) {
				defer wg.Done()
				
				// 收集设备状态
				deviceStatus, deviceErr := tu.tickerCollector.CollectDeviceStatus(cfg)
				
				// 接口可用性基于设备状态收集是否成功
				isAccessible := deviceErr == nil && deviceStatus != nil
				
				interfaces[index] = entity.TickerInterfaceMetrics{
					Name:                 cfg.Name,
					URL:                  cfg.URL,
					IsAccessible:         isAccessible,
					Error:                deviceErr,
					ChannelOffLineNumber: 0,
					ChannelOnLineNumber:  0,
					TotalDevices:         0,
					OnlineRate:           0,
				}
				
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
	// 收集定时器指标
	tickerMetrics, err := tu.CollectTickerMetrics(config)
	if err != nil {
		return fmt.Errorf("收集定时器指标失败: %v", err)
	}

	// 收集当前系统监控指标
	systemMetrics := tu.systemMetricsService.CollectBasicMetrics()

	// 构建报告内容
	title := tu.getTickerTitle(config)
	report := tu.buildTickerReport(config, tickerMetrics, systemMetrics, title)

	// 发送到钉钉
	return tu.notifier.Send(title, report)
}



// getTickerTitle 获取定时器报告标题
func (tu *TickerUseCaseImpl) getTickerTitle(config *entity.Config) string {
	if config.Tickers.AlertTitle != "" {
		return config.Tickers.AlertTitle
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
	return tu.formatter.Build(title, config, systemMetrics, []alert.TriggeredAlert{})
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
	for _, tickerConfig := range config.Tickers.HTTPInterfaces {
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
