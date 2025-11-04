// Package client internal/domain/scheduled_push/client/usecase.go
package client

import "GWatch/internal/entity"

// ClientUseCase 客户端模式用例接口
type ClientUseCase interface {
	// Run 执行客户端模式：收集数据并上传到 Redis
	Run(config *entity.Config) error
}
