// cmd/main.go
package main

import (
	"GWatch/internal/app/usecase"

	formatterImpl "GWatch/internal/infra/alert"
	policyImpl "GWatch/internal/infra/alert"
	// infra 实现
	hostCollector "GWatch/internal/infra/collectors/host"
	redisCollector "GWatch/internal/infra/collectors/service"
	evaluatorImpl "GWatch/internal/infra/monitor"
	notifierImpl "GWatch/internal/infra/notifier"

	configimpl "GWatch/internal/infra/config"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log.Println("GWatch 服务器监控工具启动")
	log.Println("正在初始化...")

	provider, err := configimpl.NewYAMLProvider("config/config.yml")
	if err != nil {
		log.Printf("加载配置文件失败: %v\n", err)
		return
	}
	cfg := provider.GetConfig()

	log.Println("开始监控...")
	log.Println("监控间隔:", cfg.Monitor.Interval)
	log.Println("HTTP监控间隔:", cfg.Monitor.HTTPInterval)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(c)

	// 创建 infra 实现
	hostInfo := hostCollector.New()
	redisInfo := redisCollector.NewRedisCollector(provider)
	httpInfo := redisCollector.NewHTTPCollector(provider)
	evaluator := evaluatorImpl.NewSimpleEvaluator()
	formatter := formatterImpl.NewMarkdownFormatter()
	notifier := notifierImpl.NewDingTalkNotifier(provider)

	// 策略实例分离：基础与HTTP分别维护自己的计数与防抖状态
	policyBase := policyImpl.NewStatefulPolicy()
	policyHTTP := policyImpl.NewStatefulPolicy()

	// 注入实现（两个用例实例，共享采集器/评估器/通知器，但策略独立）
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

	coord := usecase.NewCoordinator(runnerBase, runnerHTTP, policyBase.(*policyImpl.StatefulPolicy), policyHTTP.(*policyImpl.StatefulPolicy))

	stopCh := make(chan struct{})
	go func() {
		<-c
		close(stopCh)
	}()

	coord.RunWithIntervals(cfg, stopCh)

	log.Println("GWatch 正在退出...")
}
