// Package usecase internal/app/usecase/scheduled_push_unified.go
package usecase

import (
	"GWatch/internal/domain/scheduled_push"
	"GWatch/internal/domain/scheduled_push/client"
	"GWatch/internal/domain/scheduled_push/server"
	"GWatch/internal/entity"
	"fmt"
)

// ScheduledPushUseCaseImpl 统一的定时推送用例实现（根据mode调用client或server）
type ScheduledPushUseCaseImpl struct {
	clientUseCase client.ClientUseCase
	serverUseCase server.ServerUseCase
}

// NewScheduledPushUseCase 创建统一的定时推送用例
func NewScheduledPushUseCase(
	clientUseCase client.ClientUseCase,
	serverUseCase server.ServerUseCase,
) scheduled_push.ScheduledPushUseCase {
	return &ScheduledPushUseCaseImpl{
		clientUseCase: clientUseCase,
		serverUseCase: serverUseCase,
	}
}

// RunScheduledPush 执行全局定时推送（根据 mode 决定是 client 还是 server 模式）
func (spu *ScheduledPushUseCaseImpl) RunScheduledPush(config *entity.Config) error {
	if config.ScheduledPush == nil {
		return fmt.Errorf("scheduled_push 配置不存在")
	}

	mode := config.ScheduledPush.Mode
	if mode == "" {
		mode = "client" // 默认是 client 模式
	}

	switch mode {
	case "client":
		return spu.clientUseCase.Run(config)
	case "server":
		return spu.serverUseCase.Run(config)
	default:
		return fmt.Errorf("不支持的模式: %s", mode)
	}
}
