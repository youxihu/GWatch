// cmd/main.go
package main

import (
	"GWatch/internal/app/usecase"
	"GWatch/internal/entity"

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
	httpInterval := cfg.Monitor.HTTPInterval

	log.Println("开始监控...")
	log.Println("监控间隔:", interval)
	log.Println("HTTP监控间隔:", httpInterval)

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

	// 立即执行一次（基础与HTTP各一次）并缓存
	latestBase := runnerBase.CollectBaseOnce(cfg)
	latestHTTP := runnerHTTP.CollectHTTPOnce(cfg)

	// 定时执行（基础 与 HTTP 分离，必要时补采另一侧并合并发送）
	ticker := time.NewTicker(interval)
	httpTicker := time.NewTicker(httpInterval)
	defer ticker.Stop()
	defer httpTicker.Stop()

	for {
		select {
		case <-ticker.C:
			// 基础侧采集
			latestBase = runnerBase.CollectBaseOnce(cfg)
			merged := usecase.CombineMetrics(latestBase, latestHTTP)
			// 如果基础侧会告警，则立即刷新HTTP侧，合并后一次发送
			if wouldTrigger(runnerBase, cfg, merged) {
				latestHTTP = runnerHTTP.CollectHTTPOnce(cfg)
				merged = usecase.CombineMetrics(latestBase, latestHTTP)
				decisions, _ := evaluatorImpl.NewSimpleEvaluator().Evaluate(cfg, merged)
				baseTypes := policyBase.Apply(cfg, merged, decisions)
				httpTypes := policyHTTP.Apply(cfg, merged, decisions)
				union := unionTypes(baseTypes, httpTypes)
				_ = runnerBase.NotifyWithAlertTypes(cfg, merged, union)
				continue
			}
			runnerBase.PrintMetrics(merged)
			_ = runnerBase.EvaluateAndNotify(cfg, merged)
		case <-httpTicker.C:
			// HTTP 侧采集
			latestHTTP = runnerHTTP.CollectHTTPOnce(cfg)
			merged := usecase.CombineMetrics(latestBase, latestHTTP)
			// 如果HTTP侧会告警，则立即刷新基础侧，合并后一次发送
			if wouldTrigger(runnerHTTP, cfg, merged) {
				latestBase = runnerBase.CollectBaseOnce(cfg)
				merged = usecase.CombineMetrics(latestBase, latestHTTP)
				decisions, _ := evaluatorImpl.NewSimpleEvaluator().Evaluate(cfg, merged)
				baseTypes := policyBase.Apply(cfg, merged, decisions)
				httpTypes := policyHTTP.Apply(cfg, merged, decisions)
				union := unionTypes(baseTypes, httpTypes)
				_ = runnerHTTP.NotifyWithAlertTypes(cfg, merged, union)
				continue
			}
			runnerHTTP.PrintMetrics(merged)
			_ = runnerHTTP.EvaluateAndNotify(cfg, merged)
		case <-c:
			log.Println("\nGWatch 正在退出...")
			return
		}
	}
}

// wouldTrigger 使用新的策略实例进行一次干跑，判断是否会触发任意告警
func wouldTrigger(r *usecase.MonitoringUseCase, cfg *entity.Config, m *entity.SystemMetrics) bool {
	decisions, _ := evaluatorImpl.NewSimpleEvaluator().Evaluate(cfg, m)
	peekPolicy := policyImpl.NewStatefulPolicy()
	alerts := peekPolicy.Apply(cfg, m, decisions)
	return len(alerts) > 0
}

// 合并两组告警类型去重
func unionTypes(a, b []entity.AlertType) []entity.AlertType {
	set := map[entity.AlertType]struct{}{}
	for _, t := range a {
		set[t] = struct{}{}
	}
	for _, t := range b {
		set[t] = struct{}{}
	}
	res := make([]entity.AlertType, 0, len(set))
	for t := range set {
		res = append(res, t)
	}
	return res
}
