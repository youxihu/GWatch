package entity

// ErrorType 错误类型（建议放在同一个文件或 entity/ticker_errors.go）
type ErrorType string

const (
	ErrorTypeNone         ErrorType = "none"
	ErrorTypeToken        ErrorType = "token"        // token 相关错误
	ErrorTypeUnauthorized ErrorType = "unauthorized" // 401/403 认证失败
	ErrorTypeNetwork      ErrorType = "network"      // 超时、连接失败等
	ErrorTypeServer       ErrorType = "server"       // 5xx 服务端错误
	ErrorTypeOther        ErrorType = "other"        // 其他未知错误
)
