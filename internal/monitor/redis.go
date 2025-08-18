// internal/monitor/redis.go
package monitor

import (
	"GWatch/internal/config"
	"GWatch/internal/entity"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()
var rdb *redis.Client // Global Redis client

// 初始化 Redis 客户端
func init() {
	// 延迟初始化，等待配置加载
}

// InitRedis 初始化Redis连接
func InitRedis() error {
	cfg := config.GetConfig()
	if cfg == nil {
		return fmt.Errorf("配置未加载")
	}

	redisConfig := cfg.Redis

	// 构建Redis连接选项
	options := &redis.Options{
		Addr:         redisConfig.Addr,
		Password:     redisConfig.Password,
		DB:           redisConfig.DB,
		DialTimeout:  2 * time.Second, // 减少连接超时
		ReadTimeout:  1 * time.Second, // 减少读取超时
		WriteTimeout: 1 * time.Second, // 减少写入超时
		PoolSize:     redisConfig.PoolSize,
		MinIdleConns: redisConfig.MinIdleConns,
		MaxIdleConns: redisConfig.MaxIdleConns,
		PoolTimeout:  2 * time.Second, // 减少连接池超时
	}

	rdb = redis.NewClient(options)

	// 测试连接（使用更短的超时）
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("Redis连接测试失败: %v", err)
	}

	fmt.Printf("Redis 连接成功: %s\n", redisConfig.Addr)
	return nil
}

// GetRedisClients 获取 Redis 客户端连接数（排除监控程序自己的连接）
func GetRedisClients() (int, error) {
	val, err := rdb.Info(ctx, "clients").Result()
	if err != nil {
		return 0, fmt.Errorf("执行 INFO clients 失败: %v", err)
	}

	var totalClients int
	for _, line := range strings.Split(val, "\r\n") {
		if strings.HasPrefix(line, "connected_clients:") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				clients, err := strconv.Atoi(parts[1])
				if err != nil {
					return 0, fmt.Errorf("解析 connected_clients 失败: %v", err)
				}
				totalClients = clients
				break
			}
		}
	}

	// 获取客户端列表，排除我们自己的连接
	clientList, err := rdb.ClientList(ctx).Result()
	if err != nil {
		return totalClients, nil // 如果获取失败，返回总数
	}

	// 统计监控程序的连接数
	monitorConnections := 0
	lines := strings.Split(clientList, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 检查是否是我们的监控程序连接（通过命令和地址判断）
		if strings.Contains(line, "cmd=client|list") ||
			strings.Contains(line, "cmd=info") ||
			strings.Contains(line, "cmd=ping") ||
			strings.Contains(line, "cmd=NULL") {
			monitorConnections++
		}
	}

	// 返回排除监控程序后的连接数
	actualClients := totalClients - monitorConnections
	if actualClients < 0 {
		actualClients = 0
	}

	return actualClients, nil
}

// GetRedisClientsDetail 获取所有客户端连接的详细信息（排除监控程序自己的连接）
func GetRedisClientsDetail() ([]entity.ClientInfo, error) {
	val, err := rdb.ClientList(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("执行 CLIENT LIST 失败: %v", err)
	}

	var clients []entity.ClientInfo
	lines := strings.Split(val, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 跳过监控程序自己的连接
		if strings.Contains(line, "cmd=client|list") ||
			strings.Contains(line, "cmd=info") ||
			strings.Contains(line, "cmd=ping") ||
			strings.Contains(line, "cmd=NULL") {
			continue
		}

		pairs := strings.Split(line, " ")
		client := entity.ClientInfo{}

		for _, pair := range pairs {
			kv := strings.SplitN(pair, "=", 2)
			if len(kv) != 2 {
				continue
			}
			key, value := kv[0], kv[1]

			switch key {
			case "id":
				client.ID = value
			case "addr":
				client.Addr = value
			case "age":
				client.Age = value
			case "idle":
				client.Idle = value
			case "flags":
				client.Flags = value
			case "db":
				client.Db = value
			case "cmd":
				client.Cmd = value
			}
		}

		clients = append(clients, client)
	}

	return clients, nil
}

// CloseRedis 关闭Redis连接
func CloseRedis() {
	if rdb != nil {
		rdb.Close()
	}
}
