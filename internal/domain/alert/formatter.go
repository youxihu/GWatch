package alert

import "GWatch/internal/entity"

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


