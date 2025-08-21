package monitor

import "GWatch/internal/entity"

// Decision 表示一次阈值判断的结果（不包含消息体与发送）
type Decision struct {
	Type entity.AlertType
}

// Evaluator 负责根据配置与指标进行阈值判断，返回触发的决策
// 注意：不做防抖、连续计数、消息拼接与发送
type Evaluator interface {
	Evaluate(cfg *entity.Config, metrics *entity.SystemMetrics) ([]Decision, error)
}


