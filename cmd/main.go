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
	"time"
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

	interval := cfg.Monitor.Interval

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(c)

	// 创建 infra 实现
	hostInfo := hostCollector.New()
	redisInfo := redisCollector.NewRedisCollector(provider)
	httpInfo := redisCollector.NewHTTPCollector(provider)
	evaluator := evaluatorImpl.NewSimpleEvaluator()
	policy := policyImpl.NewStatefulPolicy()
	formatter := formatterImpl.NewMarkdownFormatter()
	notifier := notifierImpl.NewDingTalkNotifier(provider)

	// 注入实现
	runner := usecase.NewMonitoringUseCase(
		hostInfo,
		redisInfo,
		httpInfo,
		evaluator,
		policy,
		formatter,
		notifier,
	)

	log.Println("开始监控...")
	log.Println("监控间隔:", interval)
		
	log.Printf("报警阈值: CPU: %.1f%% | 内存: %.1f%% | 磁盘: %.1f%% | Redis连接数: %d-%d | HTTP接口异常阈值: %d\n",
		cfg.Monitor.CPUThreshold, cfg.Monitor.MemoryThreshold, cfg.Monitor.DiskThreshold,
		cfg.Monitor.RedisMinClients, cfg.Monitor.RedisMaxClients, cfg.Monitor.HTTPErrorThreshold)

	// 立即执行一次
	metrics := runner.CollectOnce(cfg)
	runner.PrintMetrics(metrics)
	

	
	_ = runner.EvaluateAndNotify(cfg, metrics)

	// 定时执行
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			metrics := runner.CollectOnce(cfg)
			runner.PrintMetrics(metrics)
			_ = runner.EvaluateAndNotify(cfg, metrics)
		case <-c:
			log.Println("\nGWatch 正在退出...")
			return
		}
	}
}
