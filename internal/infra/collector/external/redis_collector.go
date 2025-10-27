package external

import (
	domaincfg "GWatch/internal/domain/config"
	"GWatch/internal/entity"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var rdb *redis.Client

type RedisCollector struct{ provider domaincfg.Provider }

func NewRedisCollector(p domaincfg.Provider) *RedisCollector { return &RedisCollector{provider: p} }

func (c *RedisCollector) Init() error {
	cfg := c.provider.GetConfig()
	if cfg == nil {
		return fmt.Errorf("配置未加载")
	}
    if cfg.AppMonitoring == nil || cfg.AppMonitoring.Redis == nil {
        return fmt.Errorf("未启用Redis监控")
    }
    r := cfg.AppMonitoring.Redis
	options := &redis.Options{
        Addr:         r.Addr,
        Password:     r.Password,
        DB:           r.DB,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
        PoolSize:     r.PoolSize,
        MinIdleConns: r.MinIdleConns,
        MaxIdleConns: r.MaxIdleConns,
		PoolTimeout:  2 * time.Second,
	}
	rdb = redis.NewClient(options)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		return fmt.Errorf("Redis连接测试失败: %v", err)
	}
	return nil
}

func (c *RedisCollector) GetClients() (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	val, err := rdb.Info(ctx, "clients").Result()
	if err != nil {
		return 0, fmt.Errorf("执行 INFO clients 失败: %v", err)
	}
	var total int
	for _, line := range strings.Split(val, "\r\n") {
		if strings.HasPrefix(line, "connected_clients:") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				clients, err := strconv.Atoi(parts[1])
				if err != nil {
					return 0, fmt.Errorf("解析 connected_clients 失败: %v", err)
				}
				total = clients
				break
			}
		}
	}
	list, err := rdb.ClientList(ctx).Result()
	if err != nil {
		return total, nil
	}
	monitorConnections := 0
	for _, line := range strings.Split(list, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Contains(line, "cmd=client|list") || strings.Contains(line, "cmd=info") || strings.Contains(line, "cmd=ping") || strings.Contains(line, "cmd=NULL") {
			monitorConnections++
		}
	}
	actual := total - monitorConnections
	if actual < 0 {
		actual = 0
	}
	return actual, nil
}

func (c *RedisCollector) GetClientsDetail() ([]entity.ClientInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	val, err := rdb.ClientList(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("执行 CLIENT LIST 失败: %v", err)
	}
	var clients []entity.ClientInfo
	for _, line := range strings.Split(val, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Contains(line, "cmd=client|list") || strings.Contains(line, "cmd=info") || strings.Contains(line, "cmd=ping") || strings.Contains(line, "cmd=NULL") {
			continue
		}
		pairs := strings.Split(line, " ")
		client := entity.ClientInfo{}
		for _, pair := range pairs {
			kv := strings.SplitN(pair, "=", 2)
			if len(kv) != 2 {
				continue
			}
			switch kv[0] {
			case "id":
				client.ID = kv[1]
			case "addr":
				client.Addr = kv[1]
			case "age":
				client.Age = kv[1]
			case "idle":
				client.Idle = kv[1]
			case "flags":
				client.Flags = kv[1]
			case "db":
				client.Db = kv[1]
			case "cmd":
				client.Cmd = kv[1]
			}
		}
		clients = append(clients, client)
	}
	return clients, nil
}

func (c *RedisCollector) Close() {
	if rdb != nil {
		rdb.Close()
	}
}
