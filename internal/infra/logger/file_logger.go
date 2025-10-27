// Package logger internal/infra/logger/file_logger.go
package logger

import (
	"GWatch/internal/domain/logger"
	"GWatch/internal/entity"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

// FileLogger 文件日志实现
type FileLogger struct {
	config      *entity.LogConfig
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
	debugLogger *log.Logger
	file        *os.File
}

// NewFileLogger 创建文件日志器
func NewFileLogger(config *entity.LogConfig) (logger.Logger, error) {
	// 确保日志目录存在
	dir := filepath.Dir(config.Output)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %v", err)
	}

	// 打开日志文件
	file, err := os.OpenFile(config.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("打开日志文件失败: %v", err)
	}

	// 创建多输出器（同时输出到文件和控制台）
	var multiWriter io.Writer
	if config.Mode == "both" {
		multiWriter = io.MultiWriter(file, os.Stdout)
	} else {
		multiWriter = file
	}

	// 创建不同级别的日志器
	infoLogger := log.New(multiWriter, "[INFO] ", log.LstdFlags|log.Lshortfile)
	warnLogger := log.New(multiWriter, "[WARN] ", log.LstdFlags|log.Lshortfile)
	errorLogger := log.New(multiWriter, "[ERROR] ", log.LstdFlags|log.Lshortfile)
	debugLogger := log.New(multiWriter, "[DEBUG] ", log.LstdFlags|log.Lshortfile)

	return &FileLogger{
		config:      config,
		infoLogger:  infoLogger,
		warnLogger:  warnLogger,
		errorLogger: errorLogger,
		debugLogger: debugLogger,
		file:        file,
	}, nil
}

// Info 输出信息日志
func (l *FileLogger) Info(v ...interface{}) {
	l.infoLogger.Println(v...)
}

// Infof 输出格式化信息日志
func (l *FileLogger) Infof(format string, v ...interface{}) {
	l.infoLogger.Printf(format, v...)
}

// Warn 输出警告日志
func (l *FileLogger) Warn(v ...interface{}) {
	l.warnLogger.Println(v...)
}

// Warnf 输出格式化警告日志
func (l *FileLogger) Warnf(format string, v ...interface{}) {
	l.warnLogger.Printf(format, v...)
}

// Error 输出错误日志
func (l *FileLogger) Error(v ...interface{}) {
	l.errorLogger.Println(v...)
}

// Errorf 输出格式化错误日志
func (l *FileLogger) Errorf(format string, v ...interface{}) {
	l.errorLogger.Printf(format, v...)
}

// Debug 输出调试日志
func (l *FileLogger) Debug(v ...interface{}) {
	l.debugLogger.Println(v...)
}

// Debugf 输出格式化调试日志
func (l *FileLogger) Debugf(format string, v ...interface{}) {
	l.debugLogger.Printf(format, v...)
}

// Close 关闭日志器
func (l *FileLogger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}
