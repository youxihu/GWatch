package monitorimpl

import (
    domainMonitor "GWatch/internal/domain/monitor"
    "GWatch/internal/entity"
)

// SimpleEvaluator 仅做阈值比较，不做防抖与连续计数
type SimpleEvaluator struct{}

func NewSimpleEvaluator() *SimpleEvaluator { return &SimpleEvaluator{} }

func (s *SimpleEvaluator) Evaluate(cfg *entity.Config, metrics *entity.SystemMetrics) ([]domainMonitor.Decision, error) {
    var decisions []domainMonitor.Decision

    if metrics.CPU.Error != nil {
        decisions = append(decisions, domainMonitor.Decision{Type: entity.CPUErr})
    } else if metrics.CPU.Percent > cfg.Monitor.CPUThreshold {
        decisions = append(decisions, domainMonitor.Decision{Type: entity.CPUHigh})
    }

    if metrics.Memory.Error != nil {
        decisions = append(decisions, domainMonitor.Decision{Type: entity.MemErr})
    } else if metrics.Memory.Percent > cfg.Monitor.MemoryThreshold {
        decisions = append(decisions, domainMonitor.Decision{Type: entity.MemHigh})
    }

    if metrics.Disk.Error != nil {
        decisions = append(decisions, domainMonitor.Decision{Type: entity.DiskErr})
    } else if metrics.Disk.Percent > cfg.Monitor.DiskThreshold {
        decisions = append(decisions, domainMonitor.Decision{Type: entity.DiskHigh})
    }

    if metrics.Redis.ConnectionError != nil {
        decisions = append(decisions, domainMonitor.Decision{Type: entity.RedisErr})
    } else {
        if metrics.Redis.ClientCount < cfg.Monitor.RedisMinClients {
            decisions = append(decisions, domainMonitor.Decision{Type: entity.RedisLow})
        } else if metrics.Redis.ClientCount > cfg.Monitor.RedisMaxClients {
            decisions = append(decisions, domainMonitor.Decision{Type: entity.RedisHigh})
        }
    }

    if metrics.Network.Error != nil {
        decisions = append(decisions, domainMonitor.Decision{Type: entity.NetworkErr})
    }

    return decisions, nil
}


