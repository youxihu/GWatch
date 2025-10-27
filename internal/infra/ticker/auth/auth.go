// Package auth internal/infra/collectors/ticker/auth/auth_service.go
package auth

import (
	"GWatch/internal/domain/ticker"
	"GWatch/internal/entity"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// TokenProviderImpl Token提供者实现 - 实现domain层接口
type TokenProviderImpl struct {
	client *http.Client
	cache  map[string]*TokenCache
	mutex  sync.RWMutex
}

// TokenCache token缓存
type TokenCache struct {
	Token     string
	ExpiresAt time.Time
}

// NewTokenProvider 创建Token提供者
func NewTokenProvider() ticker.TokenProvider {
	return &TokenProviderImpl{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache: make(map[string]*TokenCache),
	}
}

// GetToken 获取token
func (tp *TokenProviderImpl) GetToken(config entity.AuthConfig) (string, error) {
	// 静态token模式
	if config.Mode == "static" {
		return config.StaticToken, nil
	}

	// 动态登录模式
	if config.Mode == "dynamic" {
		return tp.getDynamicToken(config)
	}

	return "", fmt.Errorf("不支持的认证模式: %s", config.Mode)
}

// getDynamicToken 获取动态token
func (tp *TokenProviderImpl) getDynamicToken(config entity.AuthConfig) (string, error) {
	// 生成缓存key
	cacheKey := fmt.Sprintf("%s:%s", config.Username, config.LoginURL)
	
	// 检查缓存
	tp.mutex.RLock()
	if cached, exists := tp.cache[cacheKey]; exists && time.Now().Before(cached.ExpiresAt) {
		tp.mutex.RUnlock()
		return cached.Token, nil
	}
	tp.mutex.RUnlock()

	// 执行登录
	token, err := tp.performLogin(config)
	if err != nil {
		return "", err
	}

	// 缓存token
	tp.mutex.Lock()
	defer tp.mutex.Unlock()
	
	// 解析缓存时间
	cacheDuration := 1 * time.Hour // 默认1小时
	if config.TokenCacheDuration != "" {
		if duration, err := time.ParseDuration(config.TokenCacheDuration); err == nil {
			cacheDuration = duration
		}
	}

	tp.cache[cacheKey] = &TokenCache{
		Token:     token,
		ExpiresAt: time.Now().Add(cacheDuration),
	}

	return token, nil
}

// performLogin 执行登录
func (tp *TokenProviderImpl) performLogin(config entity.AuthConfig) (string, error) {
	// 构建登录请求数据
	loginData := map[string]string{
		"username": config.Username,
		"password": config.Password,
		"code":     config.BackdoorCode, // 万能验证码
	}

	jsonData, err := json.Marshal(loginData)
	if err != nil {
		return "", fmt.Errorf("序列化登录数据失败: %v", err)
	}

	// 创建请求
	req, err := http.NewRequest("POST", config.LoginURL, nil)
	if err != nil {
		return "", fmt.Errorf("创建登录请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Body = io.NopCloser(strings.NewReader(string(jsonData)))

	// 发送请求
	resp, err := tp.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("登录请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取登录响应失败: %v", err)
	}

	// 解析响应
	var loginResp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &loginResp); err != nil {
		return "", fmt.Errorf("解析登录响应失败: %v", err)
	}

	if loginResp.Code != 200 {
		return "", fmt.Errorf("登录失败: code=%d, msg=%s", loginResp.Code, loginResp.Msg)
	}

	if loginResp.Data.Token == "" {
		return "", fmt.Errorf("登录响应中未包含token")
	}

	return "Bearer " + loginResp.Data.Token, nil
}
