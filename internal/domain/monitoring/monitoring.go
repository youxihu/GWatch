package monitoring

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

// TriggeredAlert 携带具体的告警类型与详细消息
type TriggeredAlert struct {
	Type    entity.AlertType
	Message string
}

// Formatter 负责将告警信息与指标拼成可读文本（例如 Markdown）
type Formatter interface {
	Build(title string, cfg *entity.Config, metrics *entity.SystemMetrics, alerts []TriggeredAlert) string
}

// TickerFormatter 格式化ticker报告内容
type TickerFormatter interface {
	BuildTickerReport(title string, cfg *entity.Config, tickerMetrics *entity.TickerMetrics, systemMetrics *entity.SystemMetrics) string
}

// Policy 将 monitor 层的阈值判断结果，结合防抖/连续计数等策略，输出最终需要通知的告警类型
type Policy interface {
	Apply(cfg *entity.Config, metrics *entity.SystemMetrics, decisions []Decision) []entity.AlertType
}

// Notifier 定义消息通知的能力
type Notifier interface {
	Send(title string, markdown string) error
}

