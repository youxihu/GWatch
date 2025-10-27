// Package utils internal/utils/java_dump_script.go
package utils

import (
	"context"
	"log"
	"os/exec"
	"strings"
	"time"
)

// ExecuteJavaDumpScriptAsync 异步执行 jmap 脚本（带超时），不阻塞主流程
func ExecuteJavaDumpScriptAsync(scriptPath string) {
	if scriptPath == "" {
		log.Printf("[WARN] 未配置脚本路径，跳过执行 jmap")
		return
	}

	log.Printf("[INFO] 准备异步执行 jmap 脚本: %s", scriptPath)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute) // 最长运行 5 分钟
	defer cancel()

	cmd := exec.CommandContext(ctx, "/bin/bash", scriptPath)
	output, err := cmd.CombinedOutput() // 捕获 stdout + stderr

	if ctx.Err() == context.DeadlineExceeded {
		log.Printf("[ERROR] jmap 脚本执行超时（>5分钟），进程已被终止")
		return
	}

	if err != nil {
		log.Printf("[ERROR] jmap 脚本执行失败: %v", err)
		log.Printf("[OUTPUT] 脚本输出: %s", string(output))
		return
	}

	result := strings.TrimSpace(string(output))
	log.Printf("[INFO] jmap 脚本执行成功，返回: %s", result)
}

// ExecuteJavaDumpScriptResult 同步执行 jmap 脚本（带超时），返回原始输出
func ExecuteJavaDumpScriptResult(scriptPath string, timeout time.Duration) (string, error) {
	if scriptPath == "" {
		return "", nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/bin/bash", scriptPath)
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "", context.DeadlineExceeded
	}
	return strings.TrimSpace(string(output)), err
}
