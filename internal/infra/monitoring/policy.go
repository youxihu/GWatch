package monitoring

import (
	"GWatch/internal/domain/monitoring"
	domainMonitor "GWatch/internal/domain/monitoring"
	"GWatch/internal/entity"
	"log"
	"sync"
	"time"
)

// StatefulPolicy 实现防抖与连续计数策略
type StatefulPolicy struct {
	mu        sync.RWMutex
	counters  map[entity.AlertType]int
	lastTimes map[entity.AlertType]time.Time
}

func NewStatefulPolicy() monitoring.Policy {
	return &StatefulPolicy{counters: map[entity.AlertType]int{}, lastTimes: map[entity.AlertType]time.Time{}}
}

func (p *StatefulPolicy) Apply(cfg *entity.Config, _ *entity.SystemMetrics, decisions []domainMonitor.Decision) []entity.AlertType {
	now := time.Now()
	var result []entity.AlertType
	// 累加连续计数并应用防抖
	p.mu.Lock()
	defer p.mu.Unlock()

	// 首先将未命中的连续型类型计数清零
	consecutiveSet := map[entity.AlertType]bool{entity.CPUHigh: true, entity.MemHigh: true, entity.HTTPErr: true}
	hit := map[entity.AlertType]bool{}
	for _, d := range decisions {
		hit[d.Type] = true
	}
	for t := range consecutiveSet {
		if !hit[t] {
			p.counters[t] = 0
		}
	}

	for _, d := range decisions {
		t := d.Type
		// 针对不同类型选择防抖间隔：HTTP使用专有 http_interval，其他使用通用 alert_interval
		var usedInterval time.Duration
		var consecutiveThreshold int
		
		if cfg.HostMonitoring != nil {
			usedInterval = cfg.HostMonitoring.AlertInterval
			consecutiveThreshold = cfg.HostMonitoring.ConsecutiveThreshold
		} else {
			usedInterval = 2 * time.Minute // 默认间隔
			consecutiveThreshold = 3       // 默认阈值
		}
		
		// HTTP使用专用间隔
		if t == entity.HTTPErr && cfg.AppMonitoring != nil && cfg.AppMonitoring.HTTP != nil {
			usedInterval = cfg.AppMonitoring.HTTP.Interval
		}

		// 连续计数
		if consecutiveSet[t] {
			p.counters[t] = p.counters[t] + 1
		} else {
			p.counters[t] = 1
		}
		// 防抖：间隔未到不触发
		last, ok := p.lastTimes[t]
		if ok && now.Sub(last) < usedInterval {
			continue
		}
		// 连续型达到连续触发次数阈值才触发
		if consecutiveSet[t] && p.counters[t] < consecutiveThreshold {
			log.Printf("[INFO] %s 连续第 %d/%d 次超阈值，暂不告警", t.String(), p.counters[t], consecutiveThreshold)
			continue
		}
		// 触发
		p.lastTimes[t] = now
		if consecutiveSet[t] {
			log.Printf("[WARN] %s 连续第 %d 次超阈值，告警已触发", t.String(), p.counters[t])
		} else {
			log.Printf("[WARN] %s 告警触发", t.String())
		}
		result = append(result, t)
	}
	return result
}

// PeekApply 在不修改内部状态的前提下，根据当前 decisions 预判会触发的告警类型
func (p *StatefulPolicy) PeekApply(cfg *entity.Config, _ *entity.SystemMetrics, decisions []domainMonitor.Decision) []entity.AlertType {
    now := time.Now()
    var result []entity.AlertType

    consecutiveSet := map[entity.AlertType]bool{entity.CPUHigh: true, entity.MemHigh: true, entity.HTTPErr: true}

    p.mu.RLock()
    // 拷贝当前状态
    countersCopy := make(map[entity.AlertType]int, len(p.counters))
    for k, v := range p.counters {
        countersCopy[k] = v
    }
    lastTimesCopy := make(map[entity.AlertType]time.Time, len(p.lastTimes))
    for k, v := range p.lastTimes {
        lastTimesCopy[k] = v
    }
    p.mu.RUnlock()

    hit := map[entity.AlertType]bool{}
    for _, d := range decisions {
        hit[d.Type] = true
    }
    // 先清零未命中的连续型计数（模拟）
    for t := range consecutiveSet {
        if !hit[t] {
            countersCopy[t] = 0
        }
    }

    for _, d := range decisions {
        t := d.Type
        // 连续计数（模拟）
        if consecutiveSet[t] {
            countersCopy[t] = countersCopy[t] + 1
        } else {
            countersCopy[t] = 1
        }
        // 防抖：间隔未到则不会触发
        last, ok := lastTimesCopy[t]
        var usedInterval time.Duration
        var consecutiveThreshold int
		
		if cfg.HostMonitoring != nil {
			usedInterval = cfg.HostMonitoring.AlertInterval
			consecutiveThreshold = cfg.HostMonitoring.ConsecutiveThreshold
		} else {
			usedInterval = 2 * time.Minute // 默认间隔
			consecutiveThreshold = 3       // 默认阈值
		}
		
		// HTTP使用专用间隔
		if t == entity.HTTPErr && cfg.AppMonitoring != nil && cfg.AppMonitoring.HTTP != nil {
			usedInterval = cfg.AppMonitoring.HTTP.Interval
		}
		
        if ok && now.Sub(last) < usedInterval {
            continue
        }
        // 连续触发阈值判断
        if consecutiveSet[t] && countersCopy[t] < consecutiveThreshold {
            continue
        }
        result = append(result, t)
    }
    return result
}
