package usecase

import (
	domainMonitor "GWatch/internal/domain/monitoring"
	"GWatch/internal/entity"
	policyImpl "GWatch/internal/infra/monitoring"
	"sync"
	"time"
)

// Coordinator 负责在用例层承接调度、补采与合并通知的逻辑
type Coordinator struct {
	runnerBase *MonitoringUseCase
	runnerHTTP *MonitoringUseCase
	policyBase *policyImpl.StatefulPolicy
	policyHTTP *policyImpl.StatefulPolicy

	mu         sync.RWMutex
	latestBase *entity.SystemMetrics
	latestHTTP *entity.SystemMetrics
}

func NewCoordinator(runnerBase, runnerHTTP *MonitoringUseCase, policyBase, policyHTTP *policyImpl.StatefulPolicy) *Coordinator {
	return &Coordinator{runnerBase: runnerBase, runnerHTTP: runnerHTTP, policyBase: policyBase, policyHTTP: policyHTTP}
}

// RunWithIntervals 启动双周期调度，stopCh 关闭时退出
func (c *Coordinator) RunWithIntervals(cfg *entity.Config, stopCh <-chan struct{}) {
	// 获取监控间隔配置
	var baseInterval, httpInterval time.Duration
	if cfg.HostMonitoring != nil {
		baseInterval = cfg.HostMonitoring.Interval
	} else {
		baseInterval = 5 * time.Second // 默认间隔
	}
	
	if cfg.AppMonitoring != nil && cfg.AppMonitoring.HTTP != nil {
		httpInterval = cfg.AppMonitoring.HTTP.Interval
	} else {
		httpInterval = 10 * time.Second // 默认间隔
	}

	// 首次采集
	c.latestBase = c.runnerBase.CollectBaseOnce(cfg)
	c.latestHTTP = c.runnerHTTP.CollectHTTPOnce(cfg)

	var wg sync.WaitGroup
	wg.Add(2)

	// 基础周期 goroutine
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(baseInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				base := c.runnerBase.CollectBaseOnce(cfg)
				c.mu.Lock()
				c.latestBase = base
				merged := CombineMetrics(c.latestBase, c.latestHTTP)
				c.mu.Unlock()

				if c.wouldTrigger(cfg, merged, true) {
					httpSnap := c.runnerHTTP.CollectHTTPOnce(cfg)
					c.mu.Lock()
					c.latestHTTP = httpSnap
					merged = CombineMetrics(c.latestBase, c.latestHTTP)
					c.mu.Unlock()

					decisions := c.evaluate(cfg, merged)
					baseTypes := c.policyBase.Apply(cfg, merged, filterNonHTTP(decisions))
					httpTypes := c.policyHTTP.PeekApply(cfg, merged, filterOnlyHTTP(decisions))
					_ = c.runnerBase.NotifyWithAlertTypes(cfg, merged, unionTypes(baseTypes, httpTypes))
					continue
				}
				c.runnerBase.PrintMetrics(cfg, merged)
				_ = c.runnerBase.EvaluateAndNotifyBaseOnly(cfg, merged)
			case <-stopCh:
				return
			}
		}
	}()

	// HTTP 周期 goroutine
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(httpInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				httpSnap := c.runnerHTTP.CollectHTTPOnce(cfg)
				c.mu.Lock()
				c.latestHTTP = httpSnap
				merged := CombineMetrics(c.latestBase, c.latestHTTP)
				c.mu.Unlock()

				if c.wouldTrigger(cfg, merged, false) {
					base := c.runnerBase.CollectBaseOnce(cfg)
					c.mu.Lock()
					c.latestBase = base
					merged = CombineMetrics(c.latestBase, c.latestHTTP)
					c.mu.Unlock()

					decisions := c.evaluate(cfg, merged)
					httpTypes := c.policyHTTP.Apply(cfg, merged, filterOnlyHTTP(decisions))
					baseTypes := c.policyBase.PeekApply(cfg, merged, filterNonHTTP(decisions))
					_ = c.runnerHTTP.NotifyWithAlertTypes(cfg, merged, unionTypes(baseTypes, httpTypes))
					continue
				}
				c.runnerHTTP.PrintMetrics(cfg, merged)
				_ = c.runnerHTTP.EvaluateAndNotifyHTTPOnly(cfg, merged)
			case <-stopCh:
				return
			}
		}
	}()

	// 等待退出信号
	<-stopCh
	// 等待两个循环优雅退出
	wg.Wait()
}

func (c *Coordinator) evaluate(cfg *entity.Config, m *entity.SystemMetrics) []domainMonitor.Decision {
	dec, _ := c.runnerBase.evaluator.Evaluate(cfg, m)
	return dec
}

func (c *Coordinator) wouldTrigger(cfg *entity.Config, m *entity.SystemMetrics, base bool) bool {
	decisions := c.evaluate(cfg, m)
	if base {
		peek := policyImpl.NewStatefulPolicy()
		alerts := peek.Apply(cfg, m, filterNonHTTP(decisions))
		return len(alerts) > 0
	}
	peek := policyImpl.NewStatefulPolicy()
	alerts := peek.Apply(cfg, m, filterOnlyHTTP(decisions))
	return len(alerts) > 0
}

func filterOnlyHTTP(decisions []domainMonitor.Decision) []domainMonitor.Decision {
	res := make([]domainMonitor.Decision, 0, len(decisions))
	for _, d := range decisions {
		if d.Type == entity.HTTPErr {
			res = append(res, d)
		}
	}
	return res
}

func filterNonHTTP(decisions []domainMonitor.Decision) []domainMonitor.Decision {
	res := make([]domainMonitor.Decision, 0, len(decisions))
	for _, d := range decisions {
		if d.Type != entity.HTTPErr {
			res = append(res, d)
		}
	}
	return res
}

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
