package utils

import (
	"GWatch/internal/entity"
	"errors"
	"log"
	"net"
	"strings"
)

func ClassifyError(err error) entity.ErrorType {
	log.Printf("[DEBUG classifyError] 输入错误: %v", err)
	if err == nil {
		return entity.ErrorTypeNone
	}

	errStr := err.Error() // 👈 获取完整错误字符串
	lower := strings.ToLower(errStr)

	// 👇 重点：检查是否包含 "401" 或 "认证失败"
	if strings.Contains(lower, "401") ||
		strings.Contains(lower, "unauthorized") ||
		strings.Contains(lower, "认证失败") ||
		strings.Contains(lower, "无法访问系统资源") {
		return entity.ErrorTypeUnauthorized
	}

	// Token 相关
	if strings.Contains(lower, "token") &&
		(strings.Contains(lower, "expired") ||
			strings.Contains(lower, "invalid") ||
			strings.Contains(lower, "auth fail")) {
		return entity.ErrorTypeToken
	}

	// 网络错误
	var netErr net.Error
	if errors.As(err, &netErr) {
		return entity.ErrorTypeNetwork
	}
	if strings.Contains(lower, "timeout") ||
		strings.Contains(lower, "dial") ||
		strings.Contains(lower, "connect:") ||
		strings.Contains(lower, "no such host") {
		return entity.ErrorTypeNetwork
	}

	// 服务端 5xx
	if strings.Contains(lower, "500") || strings.Contains(lower, "502") ||
		strings.Contains(lower, "503") || strings.Contains(lower, "504") ||
		strings.Contains(lower, "server error") {
		return entity.ErrorTypeServer
	}

	// 👇 兜底：如果包含 "401" 但上面没命中，也归为 Unauthorized
	if strings.Contains(errStr, "401") {
		return entity.ErrorTypeUnauthorized
	}

	return entity.ErrorTypeOther
}
