// Package common internal/domain/scheduled_push/common/repository.go
package common

import (
	"GWatch/internal/entity"
	"time"
)

// ClientDataRepository 客户端数据仓库接口（领域层）
type ClientDataRepository interface {
	// SaveClientData 保存客户端监控数据到 Redis
	SaveClientData(data *entity.ClientMonitorData, ttl time.Duration) error
	
	// GetClientDataByKey 根据 key 获取客户端数据
	GetClientDataByKey(key string) (*entity.ClientMonitorData, error)
	
	// GetAllClientDataKeys 获取所有客户端数据 key
	GetAllClientDataKeys() ([]string, error)
	
	// DeleteClientData 删除客户端数据
	DeleteClientData(key string) error
	
	// Init 初始化 Redis 连接
	Init(config *entity.Config) error
}
