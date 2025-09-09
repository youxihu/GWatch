// 测试动态认证的独立程序
package main

import (
	"GWatch/internal/domain/ticker"
	"GWatch/internal/entity"
	"GWatch/internal/infra/collectors/ticker/auth"
	"fmt"
	"log"
	"time"
)

func main() {
	// 创建Token提供者
	var tokenProvider ticker.TokenProvider = auth.NewTokenProvider()

	// 测试配置
	config := entity.AuthConfig{
		Mode:           "dynamic",
		LoginURL:       "http://localhost:8080/login",
		Username:       "gwatch",
		Password:       "gwatch@vms2025",
		BackdoorCode:   "123456",
		TokenCacheDuration: "5m",
	}

	fmt.Println("开始测试动态认证...")
	fmt.Println("请确保模拟服务器正在运行 (go run test_mock_login.go)")
	fmt.Println("config:", config)

	// 测试多次获取token，验证缓存机制
	for i := 1; i <= 5; i++ {
		fmt.Printf("\n=== 第 %d 次获取token ===\n", i)
		
		start := time.Now()
		token, err := tokenProvider.GetToken(config)
		duration := time.Since(start)
		
		if err != nil {
			log.Printf("获取token失败: %v", err)
			continue
		}
		
		fmt.Printf("Token: %s\n", token)
		fmt.Printf("耗时: %v\n", duration)
		
		// 第一次是登录，后续应该使用缓存
		if i == 1 {
			fmt.Println("状态: 首次登录")
		} else {
			fmt.Println("状态: 使用缓存")
		}
		
		// 等待1秒再测试下一次
		time.Sleep(1 * time.Second)
	}

	fmt.Println("\n测试完成！")
}
