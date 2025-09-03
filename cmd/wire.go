//go:build wireinject
// +build wireinject

package main

import (
	"GWatch/internal/app/usecase"
	"GWatch/internal/domain/alert"
	"GWatch/internal/domain/collector"
	"GWatch/internal/domain/config"
	"GWatch/internal/domain/monitor"
	"GWatch/internal/domain/notifier"
	"GWatch/internal/domain/ticker"
	"GWatch/internal/entity"

	// infra 实现
	formatterImpl "GWatch/internal/infra/alert"
	policyImpl "GWatch/internal/infra/alert"
	hostCollector "GWatch/internal/infra/collectors/host"
	redisCollector "GWatch/internal/infra/collectors/external"
	tickerCollector "GWatch/internal/infra/collectors/ticker"
	evaluatorImpl "GWatch/internal/infra/monitor"
	notifierImpl "GWatch/internal/infra/notifier"
	configimpl "GWatch/internal/infra/config"

	"github.com/google/wire"
)

// BasePolicy 基础告警策略类型别名
type BasePolicy *policyImpl.StatefulPolicy

// HTTPPolicy HTTP告警策略类型别名
type HTTPPolicy *policyImpl.StatefulPolicy

// BaseMonitoringUseCase 基础监控用例类型别名
type BaseMonitoringUseCase *usecase.MonitoringUseCase

// HTTPMonitoringUseCase HTTP监控用例类型别名
type HTTPMonitoringUseCase *usecase.MonitoringUseCase

// ProviderSet 定义所有基础设施提供者
var ProviderSet = wire.NewSet(
	// 配置提供者
	NewConfigProvider,
	NewConfig,
	
	// 收集器提供者
	NewHostCollector,
	NewRedisCollector,
	NewHTTPCollector,
	NewTickerCollector,
	
	// 评估器和格式化器提供者
	NewEvaluator,
	NewMarkdownFormatter,
	NewTickerMarkdownFormatter,
	
	// 通知器提供者
	NewDingTalkNotifier,
	
	// 告警策略提供者
	NewBasePolicy,
	NewHTTPPolicy,
	
	// 系统指标服务提供者
	NewSystemMetricsService,
	
	// 监控用例提供者
	NewBaseMonitoringUseCase,
	NewHTTPMonitoringUseCase,
)

// NewConfigProvider 创建配置提供者
func NewConfigProvider() (config.Provider, error) {
	return configimpl.NewYAMLProvider("config/config.yml")
}

// NewHostCollector 创建主机信息收集器
func NewHostCollector() collector.HostCollector {
	return hostCollector.New()
}

// NewRedisCollector 创建 Redis 收集器
func NewRedisCollector(provider config.Provider) usecase.RedisClient {
	return redisCollector.NewRedisCollector(provider)
}

// NewHTTPCollector 创建 HTTP 收集器
func NewHTTPCollector(provider config.Provider) collector.HTTPCollector {
	return redisCollector.NewHTTPCollector(provider)
}

// NewTickerCollector 创建 Ticker 收集器
func NewTickerCollector() ticker.TickerCollector {
	return tickerCollector.NewTickerCollector()
}

// NewEvaluator 创建评估器
func NewEvaluator() monitor.Evaluator {
	return evaluatorImpl.NewSimpleEvaluator()
}

// NewMarkdownFormatter 创建 Markdown 格式化器
func NewMarkdownFormatter() alert.Formatter {
	return formatterImpl.NewMarkdownFormatter()
}

// NewTickerMarkdownFormatter 创建 Ticker Markdown 格式化器
func NewTickerMarkdownFormatter() alert.TickerFormatter {
	return formatterImpl.NewTickerMarkdownFormatter().(alert.TickerFormatter)
}

// NewDingTalkNotifier 创建钉钉通知器
func NewDingTalkNotifier(provider config.Provider) notifier.Notifier {
	return notifierImpl.NewDingTalkNotifier(provider)
}

// NewBasePolicy 创建基础告警策略
func NewBasePolicy() BasePolicy {
	return policyImpl.NewStatefulPolicy().(*policyImpl.StatefulPolicy)
}

// NewHTTPPolicy 创建 HTTP 告警策略
func NewHTTPPolicy() HTTPPolicy {
	return policyImpl.NewStatefulPolicy().(*policyImpl.StatefulPolicy)
}

// InitializeApp 初始化应用程序的所有依赖
func InitializeApp() (*App, error) {
	wire.Build(
		ProviderSet,
		NewTickerUseCase,
		NewCoordinator,
		NewTickerScheduler,
		NewApp,
	)
	return &App{}, nil
}

// NewBaseMonitoringUseCase 创建基础监控用例
func NewBaseMonitoringUseCase(
	hostInfo collector.HostCollector,
	redisInfo usecase.RedisClient,
	httpInfo collector.HTTPCollector,
	evaluator monitor.Evaluator,
	policy BasePolicy,
	formatter alert.Formatter,
	notifier notifier.Notifier,
) BaseMonitoringUseCase {
	return usecase.NewMonitoringUseCase(
		hostInfo,
		redisInfo,
		httpInfo,
		evaluator,
		(*policyImpl.StatefulPolicy)(policy),
		formatter,
		notifier,
	)
}

// NewHTTPMonitoringUseCase 创建HTTP监控用例
func NewHTTPMonitoringUseCase(
	hostInfo collector.HostCollector,
	redisInfo usecase.RedisClient,
	httpInfo collector.HTTPCollector,
	evaluator monitor.Evaluator,
	policy HTTPPolicy,
	formatter alert.Formatter,
	notifier notifier.Notifier,
) HTTPMonitoringUseCase {
	return usecase.NewMonitoringUseCase(
		hostInfo,
		redisInfo,
		httpInfo,
		evaluator,
		(*policyImpl.StatefulPolicy)(policy),
		formatter,
		notifier,
	)
}

// NewSystemMetricsService 创建系统指标服务
func NewSystemMetricsService(
	hostInfo collector.HostCollector,
	redisInfo usecase.RedisClient,
	httpInfo collector.HTTPCollector,
) *usecase.SystemMetricsService {
	return usecase.NewSystemMetricsService(hostInfo, redisInfo, httpInfo)
}

// NewTickerUseCase 创建 Ticker 用例
func NewTickerUseCase(
	tickerInfo ticker.TickerCollector,
	systemMetricsService *usecase.SystemMetricsService,
	evaluator monitor.Evaluator,
	formatter alert.Formatter,
	tickerFormatter alert.TickerFormatter,
	notifier notifier.Notifier,
) ticker.TickerUseCase {
	return usecase.NewTickerUseCase(
		tickerInfo,
		systemMetricsService,
		evaluator,
		formatter,
		tickerFormatter,
		notifier,
	)
}

// NewCoordinator 创建协调器
func NewCoordinator(
	runnerBase BaseMonitoringUseCase,
	runnerHTTP HTTPMonitoringUseCase,
	policyBase BasePolicy,
	policyHTTP HTTPPolicy,
) *usecase.Coordinator {
	return usecase.NewCoordinator(
		(*usecase.MonitoringUseCase)(runnerBase),
		(*usecase.MonitoringUseCase)(runnerHTTP),
		(*policyImpl.StatefulPolicy)(policyBase),
		(*policyImpl.StatefulPolicy)(policyHTTP),
	)
}

// NewTickerScheduler 创建 Ticker 调度器
func NewTickerScheduler(tickerRunner ticker.TickerUseCase) ticker.TickerScheduler {
	return usecase.NewTickerScheduler(tickerRunner)
}

// App 应用程序结构体，包含所有需要的组件
type App struct {
	Config          *entity.Config
	Coordinator     *usecase.Coordinator
	TickerScheduler ticker.TickerScheduler
}

// NewApp 创建应用程序实例
func NewApp(
	config *entity.Config,
	coordinator *usecase.Coordinator,
	tickerScheduler ticker.TickerScheduler,
) *App {
	return &App{
		Config:          config,
		Coordinator:     coordinator,
		TickerScheduler: tickerScheduler,
	}
}

// NewConfig 从配置提供者获取配置
func NewConfig(provider config.Provider) *entity.Config {
	return provider.GetConfig()
}
