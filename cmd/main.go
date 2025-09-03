// cmd/main.go
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"GWatch/internal/app/usecase"
	"GWatch/internal/domain/alert"

	// infra 实现
	formatterImpl "GWatch/internal/infra/alert"
	policyImpl "GWatch/internal/infra/alert"
	hostCollector "GWatch/internal/infra/collectors/host"
	redisCollector "GWatch/internal/infra/collectors/service"
	tickerCollector "GWatch/internal/infra/collectors/ticker"
	evaluatorImpl "GWatch/internal/infra/monitor"
	notifierImpl "GWatch/internal/infra/notifier"

	configimpl "GWatch/internal/infra/config"
)

func main() {
	log.Println("GWatch 服务器监控工具启动")
	log.Println("正在初始化...")

	// ========================================
	// 1. 配置加载
	// ========================================
	provider, err := configimpl.NewYAMLProvider("config/config.yml")
	if err != nil {
		log.Printf("加载配置文件失败: %v\n", err)
		return
	}
	cfg := provider.GetConfig()

	log.Println("开始监控...")
	log.Println("监控间隔:", cfg.Monitor.Interval)
	log.Println("HTTP监控间隔:", cfg.Monitor.HTTPInterval)

	// ========================================
	// 2. 核心依赖注入（Infra 层实现）
	// ========================================
	hostInfo := hostCollector.New()
	redisInfo := redisCollector.NewRedisCollector(provider)
	httpInfo := redisCollector.NewHTTPCollector(provider)
	tickerInfo := tickerCollector.NewTickerCollector()
	evaluator := evaluatorImpl.NewSimpleEvaluator()
	formatter := formatterImpl.NewMarkdownFormatter()
	tickerFormatter := formatterImpl.NewTickerMarkdownFormatter()
	notifier := notifierImpl.NewDingTalkNotifier(provider)

	// ========================================
	// 3. 告警策略（独立状态，用于基础指标和 HTTP）
	// ========================================
	policyBase := policyImpl.NewStatefulPolicy()
	policyHTTP := policyImpl.NewStatefulPolicy()

	// ========================================
	// 4. UseCase 实例化（基础监控 + HTTP 监控 + Ticker）
	// ========================================
	runnerBase := usecase.NewMonitoringUseCase(
		hostInfo,
		redisInfo,
		httpInfo,
		evaluator,
		policyBase,
		formatter,
		notifier,
	)
	runnerHTTP := usecase.NewMonitoringUseCase(
		hostInfo,
		redisInfo,
		httpInfo,
		evaluator,
		policyHTTP,
		formatter,
		notifier,
	)
	
	// Ticker用例
	tickerRunner := usecase.NewTickerUseCase(
		tickerInfo,
		hostInfo,
		redisInfo,
		httpInfo,
		evaluator,
		formatter,
		tickerFormatter.(alert.TickerFormatter),
		notifier,
	)

	// ========================================
	// 5. 协调器：管理双频监控 + Ticker调度器
	// ========================================
	coord := usecase.NewCoordinator(
		runnerBase,
		runnerHTTP,
		policyBase.(*policyImpl.StatefulPolicy),
		policyHTTP.(*policyImpl.StatefulPolicy),
	)
	
	// Ticker调度器
	tickerScheduler := usecase.NewTickerScheduler(tickerRunner)

	// ========================================
	// 6. 信号监听与优雅退出
	// ========================================
	stopCh := make(chan struct{})
	go func() {
		sig := <-signalChan()
		log.Printf("接收到信号 %v，正在优雅退出...\n", sig)
		close(stopCh)
	}()

	// ========================================
	// 7. 启动监控和定时器
	// ========================================
	// 启动Ticker调度器
	if len(cfg.Tickers.HTTPInterfaces) > 0 {
		log.Println("启动定时器调度器...")
		if err := tickerScheduler.Start(cfg, stopCh); err != nil {
			log.Printf("启动定时器调度器失败: %v", err)
		}
	}
	
	// 启动监控协调器（阻塞运行）
	coord.RunWithIntervals(cfg, stopCh)

	log.Println("GWatch 正在退出...")
}

// signalChan 返回一个监听系统中断信号的 channel
func signalChan() chan os.Signal {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	return c
}
