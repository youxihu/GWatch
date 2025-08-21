package notifier

// Notifier 定义消息通知的能力
type Notifier interface {
	Send(title string, markdown string) error
}
