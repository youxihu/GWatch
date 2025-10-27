// Package ticker internal/domain/ticker/token_provider.go
package ticker

import "GWatch/internal/entity"

// TokenProvider Token提供者接口 - 领域层定义
type TokenProvider interface {
	// GetToken 获取认证token
	// 支持静态token和动态登录两种模式
	GetToken(config entity.AuthConfig) (string, error)
}
