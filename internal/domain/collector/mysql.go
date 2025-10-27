// Package collector internal/domain/collector/mysql_collector.go
package collector

import (
	"context"
)

// MySQLCollector MySQL监控数据收集器接口
type MySQLCollector interface {
	// Init 初始化MySQL连接
	Init() error
	
	// GetConnectionMetrics 获取连接与会话指标
	GetConnectionMetrics(ctx context.Context) (ConnectionMetrics, error)
	
	// GetQueryPerformanceMetrics 获取查询性能指标
	GetQueryPerformanceMetrics(ctx context.Context) (QueryPerformanceMetrics, error)
	
	// GetBufferPoolMetrics 获取InnoDB Buffer Pool指标
	GetBufferPoolMetrics(ctx context.Context) (BufferPoolMetrics, error)
	
	// GetReplicationMetrics 获取复制状态指标
	GetReplicationMetrics(ctx context.Context) (ReplicationMetrics, error)
	
	// GetLockMetrics 获取锁与阻塞指标
	GetLockMetrics(ctx context.Context) (LockMetrics, error)
	
	// GetTransactionMetrics 获取事务与日志指标
	GetTransactionMetrics(ctx context.Context) (TransactionMetrics, error)
	
	// Close 关闭连接
	Close() error
}

// ConnectionMetrics 连接与会话指标
type ConnectionMetrics struct {
	ThreadsConnected    int     `yaml:"threads_connected"`     // 当前连接数
	ThreadsRunning     int     `yaml:"threads_running"`       // 活跃线程数
	MaxConnections     int     `yaml:"max_connections"`       // 最大连接数
	ConnectionErrors   int     `yaml:"connection_errors"`     // 连接错误数
	AbortedConnects    int     `yaml:"aborted_connects"`      // 中断连接数
	ConnectionUsage    float64 `yaml:"connection_usage"`      // 连接使用率（百分比）
}

// QueryPerformanceMetrics 查询性能指标
type QueryPerformanceMetrics struct {
	QPS                int     `yaml:"qps"`                  // 每秒查询数
	TPS                int     `yaml:"tps"`                  // 每秒事务数
	SlowQueries        int     `yaml:"slow_queries"`         // 慢查询数量
	P95ResponseTime    int     `yaml:"p95_response_time"`    // P95响应时间（毫秒）
	P99ResponseTime    int     `yaml:"p99_response_time"`    // P99响应时间（毫秒）
	Questions          int     `yaml:"questions"`            // 总查询数
	Committed          int     `yaml:"committed"`            // 已提交事务数
	RolledBack         int     `yaml:"rolled_back"`          // 回滚事务数
}

// BufferPoolMetrics InnoDB Buffer Pool指标
type BufferPoolMetrics struct {
	HitRate            float64 `yaml:"hit_rate"`             // 命中率（百分比）
	Usage              float64 `yaml:"usage"`                 // 使用率（百分比）
	PagesTotal         int     `yaml:"pages_total"`          // 总页数
	PagesData          int     `yaml:"pages_data"`            // 数据页数
	PagesFree          int     `yaml:"pages_free"`            // 空闲页数
	PagesDirty         int     `yaml:"pages_dirty"`          // 脏页数
	ReadRequests       int     `yaml:"read_requests"`        // 读请求数
	Reads              int     `yaml:"reads"`                // 物理读次数
	WriteRequests      int     `yaml:"write_requests"`       // 写请求数
	Writes             int     `yaml:"writes"`               // 物理写次数
}

// ReplicationMetrics 复制状态指标
type ReplicationMetrics struct {
	SlaveIORunning     string  `yaml:"slave_io_running"`      // IO线程状态
	SlaveSQLRunning    string  `yaml:"slave_sql_running"`    // SQL线程状态
	SecondsBehindMaster int    `yaml:"seconds_behind_master"` // 复制延迟（秒）
	MasterLogFile      string  `yaml:"master_log_file"`       // 主库日志文件
	ReadMasterLogPos   int     `yaml:"read_master_log_pos"`   // 主库日志位置
	RelayLogFile       string  `yaml:"relay_log_file"`        // 中继日志文件
	RelayLogPos        int     `yaml:"relay_log_pos"`         // 中继日志位置
	GTIDMode           string  `yaml:"gtid_mode"`             // GTID模式
	GTIDExecuted       string  `yaml:"gtid_executed"`         // 已执行GTID
}

// LockMetrics 锁与阻塞指标
type LockMetrics struct {
	RowLockWaits       int     `yaml:"row_lock_waits"`        // 行锁等待次数
	RowLockTime        int     `yaml:"row_lock_time"`         // 行锁等待时间（毫秒）
	Deadlocks          int     `yaml:"deadlocks"`             // 死锁次数
	TableLocksWaited   int     `yaml:"table_locks_waited"`    // 表锁等待次数
	TableLocksImmediate int   `yaml:"table_locks_immediate"`  // 立即获得表锁次数
}

// TransactionMetrics 事务与日志指标
type TransactionMetrics struct {
	UncommittedTransactions int     `yaml:"uncommitted_transactions"` // 未提交事务数
	BinlogFiles            int     `yaml:"binlog_files"`             // Binlog文件数
	BinlogSize             int64   `yaml:"binlog_size"`               // Binlog总大小（字节）
	BinlogGrowthRate       float64 `yaml:"binlog_growth_rate"`        // Binlog增长速率（MB/小时）
	InnodbLogWaits         int     `yaml:"innodb_log_waits"`          // InnoDB日志等待次数
	InnodbLogWriteRequests int     `yaml:"innodb_log_write_requests"`  // InnoDB日志写请求数
}
