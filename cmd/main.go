// cmd/main.go
package main

import (
	"GWatch/internal/infra/logger"
	"GWatch/internal/utils"
	"flag"
	"log"
	"os"
)

func main() {
	// 解析命令行参数
	var configPath string
	var showVersion bool
	flag.StringVar(&configPath, "config", "", "配置文件路径（优先级高于环境变量 GWATCH_CONFIG）")
	flag.StringVar(&configPath, "c", "", "配置文件路径（-config 的简写）")
	flag.BoolVar(&showVersion, "version", false, "显示版本信息")
	flag.BoolVar(&showVersion, "v", false, "显示版本信息（简写）")
	flag.Parse()

	// 如果请求显示版本信息，直接打印并退出
	if showVersion {
		utils.PrintVersion()
		os.Exit(0)
	}

	// 如果通过命令行参数指定了配置文件，设置到环境变量中（这样 NewConfigProvider 可以读取）
	if configPath != "" {
		os.Setenv("GWATCH_CONFIG", configPath)
		log.Printf("使用配置文件: %s", configPath)
	}

	log.Printf("GWatch 服务器监控工具启动 (%s)", utils.FormatVersion())
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
