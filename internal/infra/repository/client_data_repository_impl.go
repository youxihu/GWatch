// Package repository internal/infra/repository/client_data_repository_impl.go
package repository

import (
	"GWatch/internal/domain/scheduled_push"
	"GWatch/internal/entity"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ClientDataRepositoryImpl Redis 客户端数据仓库实现
type ClientDataRepositoryImpl struct {
	client *redis.Client
}

// NewClientDataRepository 创建客户端数据仓库
func NewClientDataRepository() scheduled_push.ClientDataRepository {
	return &ClientDataRepositoryImpl{}
}

// Init 初始化 Redis 连接
func (r *ClientDataRepositoryImpl) Init(config *entity.Config) error {
	if config.ScheduledPush == nil {
		return fmt.Errorf("scheduled_push 配置未找到")
	}

	sp := config.ScheduledPush
	options := &redis.Options{
		Addr:         sp.RdsURL,
		Password:     sp.RdsPassword,
		DB:           sp.RdsDB,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
		PoolSize:     10,
		MinIdleConns: 2,
		MaxIdleConns: 5,
		PoolTimeout:  2 * time.Second,
	}

	r.client = redis.NewClient(options)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if _, err := r.client.Ping(ctx).Result(); err != nil {
		return fmt.Errorf("Redis 连接失败: %v", err)
	}

	return nil
}

// SaveClientData 保存客户端监控数据到 Redis
func (r *ClientDataRepositoryImpl) SaveClientData(data *entity.ClientMonitorData, ttl time.Duration) error {
	if r.client == nil {
		return fmt.Errorf("Redis 客户端未初始化")
	}

	key := entity.ClientDataKey(data.HostIP, data.Timestamp)
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("序列化数据失败: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := r.client.Set(ctx, key, jsonData, ttl).Err(); err != nil {
		return fmt.Errorf("保存数据到 Redis 失败: %v", err)
	}

	return nil
}

// GetClientDataByKey 根据 key 获取客户端数据
func (r *ClientDataRepositoryImpl) GetClientDataByKey(key string) (*entity.ClientMonitorData, error) {
	if r.client == nil {
		return nil, fmt.Errorf("Redis 客户端未初始化")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("从 Redis 读取数据失败: %v", err)
	}

	var data entity.ClientMonitorData
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return nil, fmt.Errorf("反序列化数据失败: %v", err)
	}

	return &data, nil
}

// GetAllClientDataKeys 获取所有客户端数据 key
func (r *ClientDataRepositoryImpl) GetAllClientDataKeys() ([]string, error) {
	if r.client == nil {
		return nil, fmt.Errorf("Redis 客户端未初始化")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pattern := "gwatch:client:*"
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("获取 keys 失败: %v", err)
	}

	return keys, nil
}

// DeleteClientData 删除客户端数据
func (r *ClientDataRepositoryImpl) DeleteClientData(key string) error {
	if r.client == nil {
		return fmt.Errorf("Redis 客户端未初始化")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("删除数据失败: %v", err)
	}

	return nil
}
