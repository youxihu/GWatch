package entity

type AlertType string

// 所有告警类型的常量定义
const (
	CPUHigh         AlertType = "cpu_high"           // CPU过高
	CPUErr          AlertType = "cpu_error"          // CPU监控失败
	MemHigh         AlertType = "mem_high"           // 内存过高
	MemErr          AlertType = "mem_error"          // 内存监控失败
	DiskHigh        AlertType = "disk_high"          // 磁盘过高
	DiskErr         AlertType = "disk_error"         // 磁盘监控失败
	DiskIOReadHigh  AlertType = "disk_io_read_high"  // 磁盘读IO过高
	DiskIOWriteHigh AlertType = "disk_io_write_high" // 磁盘写IO过高
	RedisHigh       AlertType = "redis_high"         // Redis连接数过高
	RedisLow        AlertType = "redis_low"          // Redis连接数过低
	RedisErr        AlertType = "redis_error"        // Redis连接异常
	
	// MySQL监控告警类型
	MySQLConnHigh   AlertType = "mysql_conn_high"    // MySQL连接数过高
	MySQLConnErr    AlertType = "mysql_conn_error"   // MySQL连接异常
	MySQLQPSHigh    AlertType = "mysql_qps_high"      // MySQL QPS过高
	MySQLSlowQuery  AlertType = "mysql_slow_query"   // MySQL慢查询过多
	MySQLBufferLow  AlertType = "mysql_buffer_low"    // MySQL Buffer Pool命中率过低
	MySQLReplDelay  AlertType = "mysql_repl_delay"    // MySQL复制延迟
	MySQLLockWait   AlertType = "mysql_lock_wait"    // MySQL锁等待过多
	MySQLDeadlock   AlertType = "mysql_deadlock"     // MySQL死锁
	MySQLTransLong  AlertType = "mysql_trans_long"   // MySQL长时间未提交事务
	
	NetworkErr      AlertType = "network_error"      // 网络监控失败
	HTTPErr         AlertType = "http_error"         // HTTP接口监控失败
	Info            AlertType = "info"
)

// AlertTypeText 告警类型中文描述映射表
var AlertTypeText = map[AlertType]string{
	CPUHigh:         "CPU使用率过高",
	CPUErr:          "CPU监控失败",
	MemHigh:         "内存使用率过高",
	MemErr:          "内存监控失败",
	DiskHigh:        "磁盘使用率过高",
	DiskErr:         "磁盘监控失败",
	DiskIOReadHigh:  "磁盘读IO过高",
	DiskIOWriteHigh: "磁盘写IO过高",
	RedisHigh:       "Redis连接数过高",
	RedisLow:        "Redis连接数过低",
	RedisErr:        "Redis连接异常",
	MySQLConnHigh:   "MySQL连接数过高",
	MySQLConnErr:    "MySQL连接异常",
	MySQLQPSHigh:    "MySQL QPS过高",
	MySQLSlowQuery:  "MySQL慢查询过多",
	MySQLBufferLow:  "MySQL Buffer Pool命中率过低",
	MySQLReplDelay:  "MySQL复制延迟",
	MySQLLockWait:   "MySQL锁等待过多",
	MySQLDeadlock:   "MySQL死锁",
	MySQLTransLong:  "MySQL长时间未提交事务",
	NetworkErr:      "网络监控失败",
	HTTPErr:         "HTTP接口监控失败",
	Info:            "信息",
}

// AlertTypeRequiresConsecutive 是否需要"连续超标"才触发的类型（用于 shouldTriggerAlert）
var AlertTypeRequiresConsecutive = map[AlertType]bool{
	CPUHigh: true,
	MemHigh: true,
	HTTPErr: true,
	// 其他错误类或瞬时类告警不需要连续触发
	CPUErr:          false,
	MemErr:          false,
	DiskHigh:        false,
	DiskErr:         false,
	DiskIOReadHigh:  false,
	DiskIOWriteHigh: false,
	RedisHigh:       false,
	RedisLow:        false,
	RedisErr:        false,
	NetworkErr:      false,
}

// 获取告警中文名
func (a AlertType) String() string {
	if text, exists := AlertTypeText[a]; exists {
		return text
	}
	return "未知告警"
}