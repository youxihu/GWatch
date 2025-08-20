// cmd/main.go
package main

import (
	"GWatch/internal/alarm"
	"GWatch/internal/config"
	"GWatch/internal/monitor"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	log.Println("GWatch 服务器监控工具启动")
	log.Println("正在初始化...")

	// 加载配置文件
	cfg, err := config.LoadConfig("config/config.yml")
	if err != nil {
		log.Printf("加载配置文件失败: %v\n", err)
		return
	}

	// 初始化Redis连接
	err = monitor.InitRedis()
	if err != nil {
		log.Printf("Redis初始化失败: %v\n", err)
		return
	}

	// 设置采集间隔
	interval := cfg.Monitor.Interval

	// 优雅退出
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(c)

	// 确保程序退出时关闭Redis连接
	defer monitor.CloseRedis()

	log.Println("开始监控...")
	log.Println("监控间隔:", interval)
	log.Println("报警阈值:")
	log.Printf("   CPU: %.1f%% | 内存: %.1f%% | 磁盘: %.1f%%\n",
		cfg.Monitor.CPUThreshold, cfg.Monitor.MemoryThreshold, cfg.Monitor.DiskThreshold)
	log.Printf("   Redis连接数: %d-%d\n",
		cfg.Monitor.RedisMinClients, cfg.Monitor.RedisMaxClients)
	log.Println("初始化完成，开始监控...")
	log.Println()

	// 立即执行第一次监控
	metrics := monitor.CollectAndPrint()
	alarm.CheckAlarmsWithMetrics(cfg, metrics)

	// 然后开始定时监控
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			metrics := monitor.CollectAndPrint()
			alarm.CheckAlarmsWithMetrics(cfg, metrics)
		case <-c:
			log.Println("\nGWatch 正在退出...")
			log.Println("关闭Redis连接...")
			return
		}
	}
}
