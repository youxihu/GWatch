// Package entity internal/entity/config.go
package entity

import (
	"time"
)

// Config 总配置结构
type Config struct {
	// 主机类监控配置
	HostMonitoring    *HostMonitoringConfig `yaml:"host_monitoring,omitempty"`    // 主机类监控，nil表示不监控
	
	// 应用层类监控配置  
	AppMonitoring     *AppMonitoringConfig  `yaml:"app_monitoring,omitempty"`    // 应用层类监控，nil表示不监控
	
	// 全局定时推送配置
	ScheduledPush     *ScheduledPushConfig  `yaml:"scheduled_push,omitempty"`    // 全局定时推送，nil表示不启用
	
	// 通用配置
	DingTalk          DingTalkConfig         `yaml:"dingtalk"`
	Log               LogConfig              `yaml:"log"`
	WhiteProcessList  []string               `yaml:"whiteProcessList"`
	JavaAppDumpScript *JavaAppDumpScript     `yaml:"javaAppDumpScript,omitempty"`
}

// HostMonitoringConfig 主机类监控配置
type HostMonitoringConfig struct {
	// 是否启用主机监控
	Enabled bool `yaml:"enabled"` // 是否启用监控
	
	// 监控间隔和阈值
	Interval             time.Duration `yaml:"interval"`
	ConsecutiveThreshold int           `yaml:"consecutive_threshold"`
	AlertInterval        time.Duration `yaml:"alert_interval"`
	AlertTitle           string        `yaml:"alert_title"`
	
	// CPU监控
	CPUThreshold float64 `yaml:"cpu_threshold"`
	
	// 内存监控  
	MemoryThreshold float64 `yaml:"memory_threshold"`
	
	// 磁盘监控
	DiskThreshold float64 `yaml:"disk_threshold"`
	
	// 网络监控（默认启用，无需额外配置）
}

// AppMonitoringConfig 应用层类监控配置
type AppMonitoringConfig struct {
	// 是否启用应用监控
	Enabled bool `yaml:"enabled"` // 是否启用应用监控
	
	// Redis监控
	Redis *RedisConfig `yaml:"redis,omitempty"`
	
	// MySQL监控
	MySQL *MySQLMonitoringConfig `yaml:"mysql,omitempty"`
	
	// HTTP接口监控
	HTTP *HTTPMonitoringConfig `yaml:"http,omitempty"`
	
	// 定时器监控
	Tickers *TickersConfig `yaml:"tickers,omitempty"`
}

// MySQLMonitoringConfig MySQL监控配置
type MySQLMonitoringConfig struct {
	// 是否启用MySQL监控
	Enabled bool `yaml:"enabled"` // 是否启用MySQL监控
	
	// 连接配置
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	Timeout  time.Duration `yaml:"timeout"`
	
	// 监控间隔
	Interval time.Duration `yaml:"interval"`
	
	// 核心监控阈值（精简版）
	Thresholds MySQLThresholds `yaml:"thresholds"`
	
	// 复制状态监控（主从架构）
	Replication *ReplicationConfig `yaml:"replication,omitempty"`
}

// MySQLThresholds MySQL监控阈值（精简版）
type MySQLThresholds struct {
	// 连接数使用率（%）
	MaxConnectionsUsageWarning float64 `yaml:"max_connections_usage_warning"`
	// 活跃线程数
	ThreadsRunningWarning int `yaml:"threads_running_warning"`
	// 每分钟慢查询增量告警阈值
	SlowQueriesRateWarning int `yaml:"slow_queries_rate_warning"`
	// Buffer Pool 命中率低于此值告警（%）
	BufferPoolHitRateWarning float64 `yaml:"buffer_pool_hit_rate_warning"`
	// 复制延迟告警阈值（秒），仅当 replication.enabled=true 时生效
	ReplicationDelayWarningSeconds int `yaml:"replication_delay_warning_seconds"`
	// 每小时死锁次数
	DeadlocksPerHourWarning int `yaml:"deadlocks_per_hour_warning"`
}

// ConnectionThresholds 连接与会话监控阈值
type ConnectionThresholds struct {
	MaxConnectionsWarning    int `yaml:"max_connections_warning"`    // 连接数告警阈值（百分比）
	ActiveThreadsWarning     int `yaml:"active_threads_warning"`      // 活跃线程数告警阈值
	ConnectionErrorsWarning  int `yaml:"connection_errors_warning"`   // 连接错误数告警阈值
	AbortedConnectsWarning  int `yaml:"aborted_connects_warning"`   // 中断连接数告警阈值
}

// QueryThresholds 查询性能监控阈值
type QueryThresholds struct {
	QPSWarning        int `yaml:"qps_warning"`         // QPS告警阈值
	TPSWarning        int `yaml:"tps_warning"`         // TPS告警阈值
	SlowQueriesWarning int `yaml:"slow_queries_warning"` // 慢查询数告警阈值
	P95ResponseTime   int `yaml:"p95_response_time"`    // P95响应时间告警阈值（毫秒）
	P99ResponseTime   int `yaml:"p99_response_time"`   // P99响应时间告警阈值（毫秒）
}

// BufferPoolThresholds InnoDB Buffer Pool监控阈值
type BufferPoolThresholds struct {
	HitRateWarning    float64 `yaml:"hit_rate_warning"`     // 命中率告警阈值（百分比）
	UsageWarning      float64 `yaml:"usage_warning"`        // 使用率告警阈值（百分比）
}

// ReplicationConfig 复制状态监控配置
type ReplicationConfig struct {
	Enabled              bool `yaml:"enabled"`                // 是否启用复制监控
	DelayWarningSeconds  int  `yaml:"delay_warning_seconds"`  // 复制延迟告警阈值（秒）
	CheckGTID           bool `yaml:"check_gtid"`             // 是否检查GTID一致性
}

// LockThresholds 锁与阻塞监控阈值
type LockThresholds struct {
	RowLockWaitsWarning    int `yaml:"row_lock_waits_warning"`     // 行锁等待次数告警阈值
	RowLockTimeWarning     int `yaml:"row_lock_time_warning"`      // 行锁等待时间告警阈值（毫秒）
	DeadlocksWarning       int `yaml:"deadlocks_warning"`          // 死锁次数告警阈值
}

// TransactionThresholds 事务与日志监控阈值
type TransactionThresholds struct {
	UncommittedTransactionsWarning int `yaml:"uncommitted_transactions_warning"` // 未提交事务数告警阈值
	BinlogGrowthRateWarning        int `yaml:"binlog_growth_rate_warning"`       // Binlog增长速率告警阈值（MB/小时）
}

// HTTPMonitoringConfig HTTP接口监控配置
type HTTPMonitoringConfig struct {
	// 是否启用HTTP监控
	Enabled bool `yaml:"enabled"` // 是否启用HTTP监控
	
	ErrorThreshold int             `yaml:"error_threshold"`
	Interval        time.Duration   `yaml:"interval"`
	Interfaces      []HTTPInterface `yaml:"interfaces"`
}

type LogConfig struct {
	// 日志模式: file, console, both
	Mode string `yaml:"mode"`
	
	// 日志级别: debug, info, warn, error
	Level string `yaml:"level"`
	
	// 日志输出路径（文件模式时使用）
	Output string `yaml:"output"`
	
	// 是否启用日志轮转
	EnableRotation bool `yaml:"enable_rotation"`
	
	// 日志文件最大大小（MB）
	MaxSize int `yaml:"max_size"`
	
	// 日志文件保留天数
	MaxAge int `yaml:"max_age"`
	
	// 日志文件最大备份数量
	MaxBackups int `yaml:"max_backups"`
}

type JavaAppDumpScript struct {
	Path string `yaml:"path"`
}

// RedisConfig Redis连接配置
type RedisConfig struct {
	// 是否启用Redis监控
	Enabled bool `yaml:"enabled"` // 是否启用Redis监控
	
	Addr         string        `yaml:"addr"`
	Password     string        `yaml:"password"`
	DB           int           `yaml:"db"`
	Timeout      time.Duration `yaml:"timeout"`
	PoolSize     int           `yaml:"pool_size"`
	MinIdleConns int           `yaml:"min_idle_conns"`
	MaxIdleConns int           `yaml:"max_idle_conns"`
	
	// Redis监控阈值
	MinClients   int `yaml:"min_clients"`
	MaxClients   int `yaml:"max_clients"`
}

// DingTalkConfig 钉钉配置
type DingTalkConfig struct {
	WebhookURL string   `yaml:"webhook_url"`
	Secret     string   `yaml:"secret"`
	AtMobiles  []string `yaml:"at_mobiles"`
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
	// 是否启用定时器监控
	Enabled bool `yaml:"enabled"` // 是否启用定时器监控
	
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

// ScheduledPushConfig 全局定时推送配置
type ScheduledPushConfig struct {
	// 是否启用全局定时推送
	Enabled bool `yaml:"enabled"`
	
	// 运行模式: "client" 或 "server"
	Mode string `yaml:"mode"` // client: 上传数据到Redis, server: 从Redis聚合并发送
	
	// Redis 连接配置（用于 client/server 模式数据交换）
	RdsURL      string `yaml:"rds_url"`
	RdsPassword string `yaml:"rds_password"`
	RdsDB       int    `yaml:"rds_db"`
	
	// 推送时间点列表，格式: ["7:00", "9:00", "11:00", "13:00", "15:00", "17:00", "19:00"]
	PushTimes []string `yaml:"push_times"`
	
	// 推送标题
	Title string `yaml:"title"`
	
	// 是否包含主机监控信息
	IncludeHostMonitoring bool `yaml:"include_host_monitoring"`
	
	// 是否包含应用监控信息
	IncludeAppMonitoring bool `yaml:"include_app_monitoring"`
	
	// Server模式聚合延迟时间（秒），用于等待所有Client上传完数据
	// 默认60秒，表示在推送时间点后延迟60秒再聚合
	ServerAggregationDelaySeconds int `yaml:"server_aggregation_delay_seconds"`
	
	// 告警信息保存配置
	AlertStorage *ScheduledPushAlertStorageConfig `yaml:"alert_storage,omitempty"`
}

// ScheduledPushAlertStorageConfig 全局定时推送告警存储配置
type ScheduledPushAlertStorageConfig struct {
	// 是否启用告警信息保存
	Enabled bool `yaml:"enabled"`
	
	// 告警信息保存路径模板（按天存储）
	AlertLogPathTemplate string `yaml:"alert_log_path_template"`
	
	// 告警信息保存格式: json, text
	Format string `yaml:"format"`
	
	// 告警信息保留天数（0表示永久保留）
	RetentionDays int `yaml:"retention_days"`
}

