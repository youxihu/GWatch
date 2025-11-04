// Package server internal/domain/scheduled_push/server/usecase.go
package server

import "GWatch/internal/entity"

// ServerUseCase 服务端模式用例接口
type ServerUseCase interface {
	// Run 执行服务端模式：从 Redis 读取数据并聚合成报告发送
	Run(config *entity.Config) error
}
