//go:build wireinject
// +build wireinject

package main

import (
	"GWatch/internal/app/usecase"
	"GWatch/internal/domain/collector"
	"GWatch/internal/domain/config"
	"GWatch/internal/domain/logger"
	"GWatch/internal/domain/monitoring"
	"GWatch/internal/domain/scheduled_push"
	"GWatch/internal/domain/ticker"
	"GWatch/internal/entity"

	// infra 实现
	monitoringImpl "GWatch/internal/infra/monitoring"
	redisCollector "GWatch/internal/infra/collector/external"
	mysqlCollector "GWatch/internal/infra/collector/external"
	hostCollector "GWatch/internal/infra/collector/host"
	tickerCollector "GWatch/internal/infra/ticker"
	tickerAuth "GWatch/internal/infra/ticker/auth"
	configimpl "GWatch/internal/infra/config"
	loggerImpl "GWatch/internal/infra/logger"
	scheduledPushStorage "GWatch/internal/infra/scheduled_push"

	"github.com/google/wire"
)

// BasePolicy 基础告警策略类型别名
type BasePolicy *monitoringImpl.StatefulPolicy

// HTTPPolicy HTTP告警策略类型别名
type HTTPPolicy *monitoringImpl.StatefulPolicy

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
	NewMySQLCollector,
	NewHTTPCollector,
	NewTickerCollector,

	// Token提供者
	NewTokenProvider,

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


// NewMySQLCollector 创建MySQL收集器
func NewMySQLCollector(provider config.Provider) collector.MySQLCollector {
	return mysqlCollector.NewMySQLCollector(provider)
}

// NewHTTPCollector 创建 HTTP 收集器
func NewHTTPCollector(provider config.Provider) collector.HTTPCollector {
	return redisCollector.NewHTTPCollector(provider)
}

// NewTickerCollector 创建 Ticker 收集器
func NewTickerCollector(tokenProvider ticker.TokenProvider) ticker.TickerCollector {
	return tickerCollector.NewTickerCollector(tokenProvider)
}

// NewTokenProvider 创建 Token 提供者
func NewTokenProvider() ticker.TokenProvider {
	return tickerAuth.NewTokenProvider()
}

// NewEvaluator 创建评估器
func NewEvaluator() monitoring.Evaluator {
	return monitoringImpl.NewSimpleEvaluator()
}

// NewMarkdownFormatter 创建 Markdown 格式化器
func NewMarkdownFormatter() monitoring.Formatter {
	return monitoringImpl.NewMarkdownFormatter()
}

// NewTickerMarkdownFormatter 创建 Ticker Markdown 格式化器
func NewTickerMarkdownFormatter() monitoring.TickerFormatter {
	return monitoringImpl.NewTickerMarkdownFormatter().(monitoring.TickerFormatter)
}

// NewDingTalkNotifier 创建钉钉通知器
func NewDingTalkNotifier(provider config.Provider) monitoring.Notifier {
	return monitoringImpl.NewDingTalkNotifier(provider)
}

// NewBasePolicy 创建基础告警策略
func NewBasePolicy() BasePolicy {
	return monitoringImpl.NewStatefulPolicy().(*monitoringImpl.StatefulPolicy)
}

// NewHTTPPolicy 创建 HTTP 告警策略
func NewHTTPPolicy() HTTPPolicy {
	return monitoringImpl.NewStatefulPolicy().(*monitoringImpl.StatefulPolicy)
}

// InitializeApp 初始化应用程序的所有依赖
func InitializeApp() (*App, error) {
	wire.Build(
		ProviderSet,
		NewTickerUseCase,
		NewCoordinator,
		NewTickerScheduler,
		NewScheduledPushAlertStorage,
		NewScheduledPushUseCase,
		NewScheduledPushScheduler,
		NewLoggerFactory,
		NewLogger,
		NewLoggerService,
		NewApp,
	)
	return &App{}, nil
}

// NewBaseMonitoringUseCase 创建基础监控用例
func NewBaseMonitoringUseCase(
	hostInfo collector.HostCollector,
	redisInfo usecase.RedisClient,
	mysqlInfo collector.MySQLCollector,
	httpInfo collector.HTTPCollector,
	evaluator monitoring.Evaluator,
	policy BasePolicy,
	formatter monitoring.Formatter,
	notifier monitoring.Notifier,
) BaseMonitoringUseCase {
	return usecase.NewMonitoringUseCase(
		hostInfo,
		redisInfo,
		mysqlInfo,
		httpInfo,
		evaluator,
		(*monitoringImpl.StatefulPolicy)(policy),
		formatter,
		notifier,
	)
}

// NewHTTPMonitoringUseCase 创建HTTP监控用例
func NewHTTPMonitoringUseCase(
	hostInfo collector.HostCollector,
	redisInfo usecase.RedisClient,
	mysqlInfo collector.MySQLCollector,
	httpInfo collector.HTTPCollector,
	evaluator monitoring.Evaluator,
	policy HTTPPolicy,
	formatter monitoring.Formatter,
	notifier monitoring.Notifier,
) HTTPMonitoringUseCase {
	return usecase.NewMonitoringUseCase(
		hostInfo,
		redisInfo,
		mysqlInfo,
		httpInfo,
		evaluator,
		(*monitoringImpl.StatefulPolicy)(policy),
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
	tokenProvider ticker.TokenProvider,
	systemMetricsService *usecase.SystemMetricsService,
	evaluator monitoring.Evaluator,
	formatter monitoring.Formatter,
	tickerFormatter monitoring.TickerFormatter,
	notifier monitoring.Notifier,
) ticker.TickerUseCase {
	return usecase.NewTickerUseCase(
		tickerInfo,
		tokenProvider,
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
		(*monitoringImpl.StatefulPolicy)(policyBase),
		(*monitoringImpl.StatefulPolicy)(policyHTTP),
	)
}

// NewTickerScheduler 创建 Ticker 调度器
func NewTickerScheduler(tickerRunner ticker.TickerUseCase) ticker.TickerScheduler {
	return usecase.NewTickerScheduler(tickerRunner)
}

// NewScheduledPushUseCase 创建全局定时推送用例
func NewScheduledPushUseCase(
	hostInfo collector.HostCollector,
	redisInfo usecase.RedisClient,
	httpInfo collector.HTTPCollector,
	tickerInfo ticker.TickerCollector,
	tokenProvider ticker.TokenProvider,
	systemMetricsService *usecase.SystemMetricsService,
	evaluator monitoring.Evaluator,
	formatter monitoring.Formatter,
	notifier monitoring.Notifier,
	alertStorage scheduled_push.ScheduledPushAlertStorage,
) scheduled_push.ScheduledPushUseCase {
	return usecase.NewScheduledPushUseCase(
		hostInfo,
		redisInfo,
		httpInfo,
		tickerInfo,
		tokenProvider,
		systemMetricsService,
		evaluator,
		formatter,
		notifier,
		alertStorage,
	)
}

// NewScheduledPushAlertStorage 创建全局定时推送告警存储
func NewScheduledPushAlertStorage(config *entity.Config) scheduled_push.ScheduledPushAlertStorage {
	if config.ScheduledPush == nil || config.ScheduledPush.AlertStorage == nil {
		return nil
	}
	return scheduledPushStorage.NewFileAlertStorage(config.ScheduledPush.AlertStorage)
}

// NewScheduledPushScheduler 创建全局定时推送调度器
func NewScheduledPushScheduler(scheduledPushUseCase scheduled_push.ScheduledPushUseCase) scheduled_push.ScheduledPushScheduler {
	return usecase.NewScheduledPushScheduler(scheduledPushUseCase)
}

// NewLoggerFactory 创建日志工厂
func NewLoggerFactory(config *entity.Config) logger.LoggerFactory {
	return loggerImpl.NewLoggerFactory(&config.Log)
}

// NewLogger 创建日志器
func NewLogger(factory logger.LoggerFactory) logger.Logger {
	logger, err := factory.CreateLogger()
	if err != nil {
		// 如果创建失败，返回控制台日志器
		return loggerImpl.NewConsoleLogger()
	}
	return logger
}

// NewLoggerService 创建日志服务
func NewLoggerService(logger logger.Logger) *usecase.LoggerService {
	return usecase.NewLoggerService(logger)
}

// App 应用程序结构体，包含所有需要的组件
type App struct {
	Config                *entity.Config
	Coordinator           *usecase.Coordinator
	TickerScheduler       ticker.TickerScheduler
	ScheduledPushScheduler scheduled_push.ScheduledPushScheduler
	LoggerService         *usecase.LoggerService
}

// NewApp 创建应用程序实例
func NewApp(
	config *entity.Config,
	coordinator *usecase.Coordinator,
	tickerScheduler ticker.TickerScheduler,
	scheduledPushScheduler scheduled_push.ScheduledPushScheduler,
	loggerService *usecase.LoggerService,
) *App {
	return &App{
		Config:                config,
		Coordinator:           coordinator,
		TickerScheduler:       tickerScheduler,
		ScheduledPushScheduler: scheduledPushScheduler,
		LoggerService:         loggerService,
	}
}

// NewConfig 从配置提供者获取配置
func NewConfig(provider config.Provider) *entity.Config {
	return provider.GetConfig()
}
