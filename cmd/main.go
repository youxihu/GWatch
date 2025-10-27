// cmd/main.go
package main

import (
	"GWatch/internal/infra/logger"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log.Println("GWatch 服务器监控工具启动")
	log.Println("正在初始化...")

	// ========================================
	// 1. 使用 Wire 进行依赖注入
	// ========================================
	app, err := InitializeApp()
	if err != nil {
		log.Printf("初始化应用程序失败: %v\n", err)
		return
	}

	// ========================================
	// 2. 初始化日志系统
	// ========================================
	// 初始化日志包装器，重定向标准 log 输出
	logger.InitLogWrapper(app.LoggerService.GetLogger())

    cfg := app.Config
    log.Println("开始监控...")
    if cfg.HostMonitoring != nil {
        log.Println("监控间隔:", cfg.HostMonitoring.Interval)
    }
    if cfg.AppMonitoring != nil && cfg.AppMonitoring.HTTP != nil {
        log.Println("HTTP监控间隔:", cfg.AppMonitoring.HTTP.Interval)
    }

	// ========================================
	// 2. 信号监听与优雅退出
	// ========================================
	stopCh := make(chan struct{})
	go func() {
		sig := <-signalChan()
		log.Printf("接收到信号 %v，正在优雅退出...\n", sig)
		close(stopCh)
	}()

	// ========================================
	// 3. 启动监控和定时器
	// ========================================
    // 启动Ticker调度器
    if cfg.AppMonitoring != nil && cfg.AppMonitoring.Tickers != nil && len(cfg.AppMonitoring.Tickers.TickerInterfaces) > 0 {
        log.Println("启动定时器调度器...")
        if err := app.TickerScheduler.Start(cfg, stopCh); err != nil {
            log.Printf("启动定时器调度器失败: %v", err)
        }
    }

    // 启动全局定时推送调度器
    if cfg.ScheduledPush != nil && cfg.ScheduledPush.Enabled {
        log.Println("启动全局定时推送调度器...")
        if err := app.ScheduledPushScheduler.Start(cfg, stopCh); err != nil {
            log.Printf("启动全局定时推送调度器失败: %v", err)
        }
    }
	
	// 启动监控协调器（阻塞运行）
	app.Coordinator.RunWithIntervals(cfg, stopCh)

	log.Println("GWatch 正在退出...")
}

// signalChan 返回一个监听系统中断信号的 channel
func signalChan() chan os.Signal {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	return c
}
