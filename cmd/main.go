// cmd/main.go
package main

import (
	"GWatch/internal/app/runtime"
	alertimpl "GWatch/internal/infra/alert"
	"GWatch/internal/infra/collectors/host"
	service "GWatch/internal/infra/collectors/service"
	configimpl "GWatch/internal/infra/config"
	monitorimpl "GWatch/internal/infra/monitor"
	notifierimpl "GWatch/internal/infra/notifier"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	log.Println("GWatch 服务器监控工具启动")
	log.Println("正在初始化...")

	// 加载配置（DDD Provider）
	provider, err := configimpl.NewYAMLProvider("config/config.yml")
	if err != nil {
		log.Printf("加载配置文件失败: %v\n", err)
		return
	}
	cfg := provider.GetConfig()

	// 设置采集间隔
	interval := cfg.Monitor.Interval

	// 优雅退出
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(c)

	// 组装DDD链路
	hostCollector := host.New()
	redisCollector := service.NewRedisCollector(provider)
	evaluator := monitorimpl.NewSimpleEvaluator()
	policy := alertimpl.NewStatefulPolicy()
	formatter := alertimpl.NewMarkdownFormatter()
	notifier := notifierimpl.NewDingTalkNotifier(provider)
	runner := runtime.NewRunner(hostCollector, redisCollector, evaluator, policy, formatter, notifier)

	log.Println("开始监控...")
	log.Println("监控间隔:", interval)
	log.Println("报警阈值:")
	log.Printf("   CPU: %.1f%% | 内存: %.1f%% | 磁盘: %.1f%% | Redis连接数: %d-%d\n",
		cfg.Monitor.CPUThreshold, cfg.Monitor.MemoryThreshold, cfg.Monitor.DiskThreshold,
		cfg.Monitor.RedisMinClients, cfg.Monitor.RedisMaxClients)
	log.Printf("   Redis连接数: %d-%d\n",
		cfg.Monitor.RedisMinClients, cfg.Monitor.RedisMaxClients)
	log.Println("初始化完成，开始监控...")
	log.Println()

	// 立即执行一次监控
	metrics := runner.CollectOnce()
	runner.PrintMetrics(metrics)
	_ = runner.EvaluateAndNotify(cfg, metrics)

	// 然后开始定时监控
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			metrics := runner.CollectOnce()
			runner.PrintMetrics(metrics)
			_ = runner.EvaluateAndNotify(cfg, metrics)
		case <-c:
			log.Println("\nGWatch 正在退出...")
			return
		}
	}
}
