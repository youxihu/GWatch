// Package logger internal/infra/logger/log_wrapper.go
package logger

import (
	"GWatch/internal/domain/logger"
	"fmt"
	"log"
	"os"
)

// LogWrapper 日志包装器，用于替换标准 log 包
type LogWrapper struct {
	logger logger.Logger
}

// NewLogWrapper 创建日志包装器
func NewLogWrapper(logger logger.Logger) *LogWrapper {
	return &LogWrapper{
		logger: logger,
	}
}

// InitLogWrapper 初始化全局日志包装器
func InitLogWrapper(logger logger.Logger) {
	wrapper := NewLogWrapper(logger)
	
	// 重定向标准 log 输出
	log.SetFlags(0) // 移除默认的时间戳和文件信息
	log.SetOutput(&logWriter{wrapper: wrapper})
}

// logWriter 实现 io.Writer 接口
type logWriter struct {
	wrapper *LogWrapper
}

func (w *logWriter) Write(p []byte) (n int, err error) {
	// 使用我们的日志器输出
	w.wrapper.logger.Info(string(p))
	return len(p), nil
}

// 提供兼容标准 log 包的函数
func (w *LogWrapper) Print(v ...interface{}) {
	w.logger.Info(v...)
}

func (w *LogWrapper) Printf(format string, v ...interface{}) {
	w.logger.Infof(format, v...)
}

func (w *LogWrapper) Println(v ...interface{}) {
	w.logger.Info(v...)
}

func (w *LogWrapper) Fatal(v ...interface{}) {
	w.logger.Error(v...)
	os.Exit(1)
}

func (w *LogWrapper) Fatalf(format string, v ...interface{}) {
	w.logger.Errorf(format, v...)
	os.Exit(1)
}

func (w *LogWrapper) Fatalln(v ...interface{}) {
	w.logger.Error(v...)
	os.Exit(1)
}

func (w *LogWrapper) Panic(v ...interface{}) {
	w.logger.Error(v...)
	panic(fmt.Sprint(v...))
}

func (w *LogWrapper) Panicf(format string, v ...interface{}) {
	w.logger.Errorf(format, v...)
	panic(fmt.Sprintf(format, v...))
}

func (w *LogWrapper) Panicln(v ...interface{}) {
	w.logger.Error(v...)
	panic(fmt.Sprint(v...))
}
