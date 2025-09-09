// Package entity internal/entity/config.go
package entity

import "time"

// Config 总配置结构
type Config struct {
	Redis             RedisConfig       `yaml:"redis"`
	DingTalk          DingTalkConfig    `yaml:"dingtalk"`
	Monitor           MonitorConfig     `yaml:"monitor"`
	Tickers           TickersConfig     `yaml:"tickers"`
	Log               LogConfig         `yaml:"log"`
	JavaAppDumpScript JavaAppDumpScript `yaml:"javaAppDumpScript"`
}

type LogConfig struct {
	Mode   string `yaml:"mode"`
	Level  string `yaml:"level"`
	Output string `yaml:"output"`
}
type JavaAppDumpScript struct {
	Path string `yaml:"path"`
}

// RedisConfig Redis连接配置
type RedisConfig struct {
	Addr         string        `yaml:"addr"`
	Password     string        `yaml:"password"`
	DB           int           `yaml:"db"`
	Timeout      time.Duration `yaml:"timeout"`
	PoolSize     int           `yaml:"pool_size"`
	MinIdleConns int           `yaml:"min_idle_conns"`
	MaxIdleConns int           `yaml:"max_idle_conns"`
}

// DingTalkConfig 钉钉配置
type DingTalkConfig struct {
	WebhookURL string   `yaml:"webhook_url"`
	Secret     string   `yaml:"secret"`
	AtMobiles  []string `yaml:"at_mobiles"`
}

// MonitorConfig 监控配置
type MonitorConfig struct {
	Interval             time.Duration   `yaml:"interval"`
	ConsecutiveThreshold int             `yaml:"consecutive_threshold"`
	CPUThreshold         float64         `yaml:"cpu_threshold"`
	MemoryThreshold      float64         `yaml:"memory_threshold"`
	DiskThreshold        float64         `yaml:"disk_threshold"`
	RedisMinClients      int             `yaml:"redis_min_clients"`
	RedisMaxClients      int             `yaml:"redis_max_clients"`
	AlertInterval        time.Duration   `yaml:"alert_interval"`
	HTTPErrorThreshold   int             `yaml:"http_error_threshold"`
	HTTPInterval         time.Duration   `yaml:"http_interval"`
	HTTPInterfaces       []HTTPInterface `yaml:"http_interfaces"`
	AlertTitle           string          `yaml:"alert_title"`
}

// HTTPInterface HTTP接口监控配置
type HTTPInterface struct {
	Name         string        `yaml:"name"`
	URL          string        `yaml:"url"`
	Timeout      time.Duration `yaml:"timeout"`
	NeedAlert    bool          `yaml:"need_alert"`
	AllowedCodes []int         `yaml:"allowed_codes"`
}

// TickersConfig 定时器配置
type TickersConfig struct {
	AlertTitle       string                `yaml:"alert_title"`
	TickerInterfaces []TickerHTTPInterface `yaml:"ticker_interfaces"`
}

// TickerHTTPInterface 定时器HTTP接口配置
type TickerHTTPInterface struct {
	Name      string   `yaml:"name"`
	DeviceURL string   `yaml:"device_url"` // 设备数据接口
	AlertTime []string `yaml:"alert_time"` // 格式: ["0:00", "7:00", "12:00", "18:00", "21:00"]

	// 认证配置 - 支持两种模式
	Auth AuthConfig `yaml:"auth"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	// 模式: "static" 使用静态token, "dynamic" 使用动态登录
	Mode string `yaml:"mode"`

	// 静态token模式 (当前使用)
	StaticToken string `yaml:"static_token,omitempty"`

	// 动态登录模式 (未来使用)
	LoginURL     string `yaml:"login_url,omitempty"`
	Username     string `yaml:"username,omitempty"`
	Password     string `yaml:"password,omitempty"`
	BackdoorCode string `yaml:"backdoor_code,omitempty"` // 万能验证码

	// Token缓存配置
	TokenCacheDuration string `yaml:"token_cache_duration,omitempty"` // 如: "1h", "30m"
}
