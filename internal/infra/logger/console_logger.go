// Package logger internal/infra/logger/console_logger.go
package logger

import (
	"GWatch/internal/domain/logger"
	"fmt"
	"log"
)

// ConsoleLogger 控制台日志实现
type ConsoleLogger struct{}

// NewConsoleLogger 创建控制台日志器
func NewConsoleLogger() logger.Logger {
	return &ConsoleLogger{}
}

// Info 输出信息日志
func (l *ConsoleLogger) Info(v ...interface{}) {
	log.Println("[INFO]", fmt.Sprint(v...))
}

// Infof 输出格式化信息日志
func (l *ConsoleLogger) Infof(format string, v ...interface{}) {
	log.Printf("[INFO] "+format, v...)
}

// Warn 输出警告日志
func (l *ConsoleLogger) Warn(v ...interface{}) {
	log.Println("[WARN]", fmt.Sprint(v...))
}

// Warnf 输出格式化警告日志
func (l *ConsoleLogger) Warnf(format string, v ...interface{}) {
	log.Printf("[WARN] "+format, v...)
}

// Error 输出错误日志
func (l *ConsoleLogger) Error(v ...interface{}) {
	log.Println("[ERROR]", fmt.Sprint(v...))
}

// Errorf 输出格式化错误日志
func (l *ConsoleLogger) Errorf(format string, v ...interface{}) {
	log.Printf("[ERROR] "+format, v...)
}

// Debug 输出调试日志
func (l *ConsoleLogger) Debug(v ...interface{}) {
	log.Println("[DEBUG]", fmt.Sprint(v...))
}

// Debugf 输出格式化调试日志
func (l *ConsoleLogger) Debugf(format string, v ...interface{}) {
	log.Printf("[DEBUG] "+format, v...)
}
