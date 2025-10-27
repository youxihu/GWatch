// cmd/main.go
package main

import (
	"GWatch/internal/infra/logger"
	"log"
)

func main() {
	log.Println("GWatch 服务器监控工具启动")
	log.Println("正在初始化...")

	// 1. 使用 Wire 进行依赖注入
	app, err := InitializeApp()
	if err != nil {
		log.Printf("初始化应用程序失败: %v\n", err)
		return
	}

	// 2. 初始化日志系统
	logger.InitLogWrapper(app.LoggerService.GetLogger())

	// 3. 启动应用程序
	if err := app.Start(); err != nil {
		log.Printf("应用程序运行失败: %v\n", err)
		return
	}
}
