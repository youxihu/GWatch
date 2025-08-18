// internal/entity/config.go
package entity

import "time"

// Config 总配置结构
type Config struct {
	Redis    RedisConfig    `yaml:"redis"`
	DingTalk DingTalkConfig `yaml:"dingtalk"`
	Monitor  MonitorConfig  `yaml:"monitor"`
	Log      LogConfig      `yaml:"log" json:"log"` // 新增
}

type LogConfig struct {
	Debug bool `yaml:"debug" json:"debug"`
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
	Interval        time.Duration `yaml:"interval"`
	CPUThreshold    float64       `yaml:"cpu_threshold"`
	MemoryThreshold float64       `yaml:"memory_threshold"`
	DiskThreshold   float64       `yaml:"disk_threshold"`
	RedisMinClients int           `yaml:"redis_min_clients"`
	RedisMaxClients int           `yaml:"redis_max_clients"`
	AlertInterval   time.Duration `yaml:"alert_interval"`
}
