package alert

import (
    domainMonitor "GWatch/internal/domain/monitor"
    "GWatch/internal/entity"
)

// Policy 将 monitor 层的阈值判断结果，结合防抖/连续计数等策略，输出最终需要通知的告警类型
type Policy interface {
    Apply(cfg *entity.Config, metrics *entity.SystemMetrics, decisions []domainMonitor.Decision) []entity.AlertType
}


