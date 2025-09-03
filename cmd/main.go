// cmd/main.go
package main

import (
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

	cfg := app.Config
	log.Println("开始监控...")
	log.Println("监控间隔:", cfg.Monitor.Interval)
	log.Println("HTTP监控间隔:", cfg.Monitor.HTTPInterval)

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
	if len(cfg.Tickers.HTTPInterfaces) > 0 {
		log.Println("启动定时器调度器...")
		if err := app.TickerScheduler.Start(cfg, stopCh); err != nil {
			log.Printf("启动定时器调度器失败: %v", err)
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
