package collector

import (
	"GWatch/internal/entity"
	"time"
)

// HostCollector defines capabilities for collecting host-level metrics.
type HostCollector interface {
	GetCPUPercent() (float64, error)
	GetMemoryUsage() (float64, uint64, uint64, error)
	GetDiskUsage() (float64, uint64, uint64, error)
	GetDiskIORate() (float64, float64, error)
	GetNetworkRate() (float64, float64, error)
	// GetTopProcesses returns top N processes by CPU and Memory
	GetTopProcesses(n int) ([]entity.ProcessInfo, []entity.ProcessInfo, error)
}

// RedisCollector defines capabilities for collecting Redis service metrics.
type RedisCollector interface {
	// Init prepares underlying connections according to global config.
	Init() error
	// GetClients returns number of client connections (excluding self when possible).
	GetClients() (int, error)
	// GetClientsDetail returns detailed client list (excluding self when possible).
	GetClientsDetail() ([]entity.ClientInfo, error)
	// Close releases resources.
	Close()
}

// HTTPCollector defines capabilities for collecting HTTP interface monitoring metrics.
type HTTPCollector interface {
	// Init prepares underlying HTTP client according to global config.
	Init() error
	// CheckInterface checks if a specific HTTP interface is accessible and returns response time and status code.
	CheckInterface(url string, timeout time.Duration) (bool, time.Duration, int, error)
	// Close releases resources.
	Close()
}


