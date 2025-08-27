// Package entity internal/entity/metrics.go
package entity

import "time"

// SystemMetrics 系统监控指标结构
type SystemMetrics struct {
	Timestamp time.Time
	CPU       CPUMetrics
	Memory    MemoryMetrics
	Disk      DiskMetrics
	Network   NetworkMetrics
	Redis     RedisMetrics
}

// CPUMetrics CPU指标
type CPUMetrics struct {
	Percent float64
	Error   error
}

// MemoryMetrics 内存指标
type MemoryMetrics struct {
	Percent float64
	UsedMB  uint64
	TotalMB uint64
	Error   error
}

// DiskMetrics 磁盘指标
type DiskMetrics struct {
	Percent     float64
	UsedGB      uint64
	TotalGB     uint64
	ReadKBps    float64
	WriteKBps   float64
	Error       error
}

// NetworkMetrics 网络指标
type NetworkMetrics struct {
	DownloadKBps float64
	UploadKBps   float64
	Error        error
}

// RedisMetrics Redis指标
type RedisMetrics struct {
	ClientCount     int
	ClientDetails   []ClientInfo
	ConnectionError error
	DetailError     error
}

// ClientInfo Redis客户端信息
type ClientInfo struct {
	ID    string
	Addr  string
	Age   string
	Idle  string
	Flags string
	Db    string
	Cmd   string
}
