package alertimpl

import (
    domainMonitor "GWatch/internal/domain/monitor"
    "GWatch/internal/domain/alert"
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

func NewStatefulPolicy() alert.Policy {
    return &StatefulPolicy{ counters: map[entity.AlertType]int{}, lastTimes: map[entity.AlertType]time.Time{} }
}

func (p *StatefulPolicy) Apply(cfg *entity.Config, _ *entity.SystemMetrics, decisions []domainMonitor.Decision) []entity.AlertType {
    now := time.Now()
    var result []entity.AlertType
    // 累加连续计数并应用防抖
    p.mu.Lock()
    defer p.mu.Unlock()

    // 首先将未命中的连续型类型计数清零
    consecutiveSet := map[entity.AlertType]bool{ entity.CPUHigh: true, entity.MemHigh: true }
    hit := map[entity.AlertType]bool{}
    for _, d := range decisions { hit[d.Type] = true }
    for t := range consecutiveSet { if !hit[t] { p.counters[t] = 0 } }

    for _, d := range decisions {
        t := d.Type
        // 连续计数
        if consecutiveSet[t] { p.counters[t] = p.counters[t] + 1 } else { p.counters[t] = 1 }
        // 防抖：间隔未到不触发
        last, ok := p.lastTimes[t]
        if ok && now.Sub(last) < cfg.Monitor.AlertInterval { continue }
        // 连续型达到3次才触发
        if consecutiveSet[t] && p.counters[t] < 3 {
            log.Printf("[INFO] %s 连续第 %d/3 次超阈值，暂不告警", t.String(), p.counters[t])
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


