// Package external internal/infra/collectors/external/mysql_collector.go
package external

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"GWatch/internal/domain/collector"
	"GWatch/internal/domain/config"
	_ "github.com/go-sql-driver/mysql"
)

// MySQLCollectorImpl MySQL监控数据收集器实现
type MySQLCollectorImpl struct {
	provider config.Provider
	db       *sql.DB
}

// NewMySQLCollector 创建MySQL监控数据收集器
func NewMySQLCollector(provider config.Provider) collector.MySQLCollector {
	return &MySQLCollectorImpl{
		provider: provider,
	}
}

// Init 初始化MySQL连接
func (c *MySQLCollectorImpl) Init() error {
	cfg := c.provider.GetConfig()
	if cfg == nil || cfg.AppMonitoring == nil || cfg.AppMonitoring.MySQL == nil {
		return fmt.Errorf("MySQL 配置未找到或未启用")
	}

	mysqlCfg := cfg.AppMonitoring.MySQL
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?timeout=%s&parseTime=true&loc=Local",
		mysqlCfg.Username,
		mysqlCfg.Password,
		mysqlCfg.Host,
		mysqlCfg.Port,
		mysqlCfg.Database,
		mysqlCfg.Timeout.String(),
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("连接MySQL失败: %v", err)
	}

	// 设置连接池参数
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	// 测试连接
	if err := db.Ping(); err != nil {
		return fmt.Errorf("MySQL连接测试失败: %v", err)
	}

	c.db = db
	return nil
}

// GetConnectionMetrics 获取连接与会话指标
func (c *MySQLCollectorImpl) GetConnectionMetrics(ctx context.Context) (collector.ConnectionMetrics, error) {
	var metrics collector.ConnectionMetrics

	// 获取连接相关指标
	queries := map[string]interface{}{
		"Threads_connected":    &metrics.ThreadsConnected,
		"Threads_running":      &metrics.ThreadsRunning,
		"Connection_errors_max_connections": &metrics.ConnectionErrors,
		"Aborted_connects":     &metrics.AbortedConnects,
	}

	for key, ptr := range queries {
		query := fmt.Sprintf("SHOW GLOBAL STATUS LIKE '%s'", key)
		var name string
		var value string
		
		err := c.db.QueryRowContext(ctx, query).Scan(&name, &value)
		if err != nil {
			// 如果查询失败，设置默认值0，不返回错误
			switch ptr := ptr.(type) {
			case *int:
				*ptr = 0
			}
			continue
		}
		
		val, err := strconv.Atoi(value)
		if err != nil {
			val = 0
		}
		
		switch ptr := ptr.(type) {
		case *int:
			*ptr = val
		}
	}

	// 单独查询max_connections系统变量
	query := "SHOW VARIABLES LIKE 'max_connections'"
	var name string
	var value string
	
	err := c.db.QueryRowContext(ctx, query).Scan(&name, &value)
	if err != nil {
		fmt.Printf("[DEBUG] 查询max_connections失败，使用默认值0: %v\n", err)
		metrics.MaxConnections = 0
	} else {
		val, err := strconv.Atoi(value)
		if err != nil {
			fmt.Printf("[DEBUG] 解析max_connections值失败，使用默认值0: %v\n", err)
			metrics.MaxConnections = 0
		} else {
			metrics.MaxConnections = val
		}
	}

	// 计算连接使用率
	if metrics.MaxConnections > 0 {
		metrics.ConnectionUsage = float64(metrics.ThreadsConnected) / float64(metrics.MaxConnections) * 100
	}


	return metrics, nil
}

// GetQueryPerformanceMetrics 获取查询性能指标
func (c *MySQLCollectorImpl) GetQueryPerformanceMetrics(ctx context.Context) (collector.QueryPerformanceMetrics, error) {
	var metrics collector.QueryPerformanceMetrics

	// 获取查询性能相关指标
	queries := map[string]interface{}{
		"questions":     &metrics.Questions,
		"com_select":    &metrics.QPS, // 简化处理，实际需要计算差值
		"com_commit":    &metrics.Committed,
		"com_rollback":  &metrics.RolledBack,
		"slow_queries":  &metrics.SlowQueries,
	}

	for key, ptr := range queries {
		query := fmt.Sprintf("SHOW GLOBAL STATUS LIKE '%s'", key)
		var name string
		var value string
		
		err := c.db.QueryRowContext(ctx, query).Scan(&name, &value)
		if err != nil {
			return metrics, fmt.Errorf("查询%s失败: %v", key, err)
		}
		
		val, err := strconv.Atoi(value)
		if err != nil {
			return metrics, fmt.Errorf("解析%s值失败: %v", key, err)
		}
		
		switch ptr := ptr.(type) {
		case *int:
			*ptr = val
		}
	}

	// 计算TPS（简化处理）
	metrics.TPS = metrics.Committed + metrics.RolledBack

	// 响应时间需要从Performance Schema获取，这里简化处理
	metrics.P95ResponseTime = 100 // 示例值
	metrics.P99ResponseTime = 200 // 示例值

	return metrics, nil
}

// GetBufferPoolMetrics 获取InnoDB Buffer Pool指标
func (c *MySQLCollectorImpl) GetBufferPoolMetrics(ctx context.Context) (collector.BufferPoolMetrics, error) {
	var metrics collector.BufferPoolMetrics

	// 获取Buffer Pool相关指标
	queries := map[string]interface{}{
		"Innodb_buffer_pool_pages_total": &metrics.PagesTotal,
		"Innodb_buffer_pool_pages_data":  &metrics.PagesData,
		"Innodb_buffer_pool_pages_free":  &metrics.PagesFree,
		"Innodb_buffer_pool_pages_dirty":  &metrics.PagesDirty,
		"Innodb_buffer_pool_read_requests": &metrics.ReadRequests,
		"Innodb_buffer_pool_reads":        &metrics.Reads,
		"Innodb_buffer_pool_write_requests": &metrics.WriteRequests,
		"Innodb_buffer_pool_writes":         &metrics.Writes,
	}

	for key, ptr := range queries {
		query := fmt.Sprintf("SHOW GLOBAL STATUS LIKE '%s'", key)
		var name string
		var value string
		
		err := c.db.QueryRowContext(ctx, query).Scan(&name, &value)
		if err != nil {
			return metrics, fmt.Errorf("查询%s失败: %v", key, err)
		}
		
		val, err := strconv.Atoi(value)
		if err != nil {
			return metrics, fmt.Errorf("解析%s值失败: %v", key, err)
		}
		
		switch ptr := ptr.(type) {
		case *int:
			*ptr = val
		}
	}

	// 计算命中率
	if metrics.ReadRequests > 0 {
		metrics.HitRate = float64(metrics.ReadRequests-metrics.Reads) / float64(metrics.ReadRequests) * 100
	}

	// 计算使用率
	if metrics.PagesTotal > 0 {
		metrics.Usage = float64(metrics.PagesData) / float64(metrics.PagesTotal) * 100
	}

	return metrics, nil
}

// GetReplicationMetrics 获取复制状态指标
func (c *MySQLCollectorImpl) GetReplicationMetrics(ctx context.Context) (collector.ReplicationMetrics, error) {
	var metrics collector.ReplicationMetrics

	// 检查是否为主从复制环境
	query := "SHOW SLAVE STATUS"
	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return metrics, fmt.Errorf("查询复制状态失败: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		// 不是从库或复制未配置
		return metrics, nil
	}

	// 获取复制状态字段
	columns, err := rows.Columns()
	if err != nil {
		return metrics, fmt.Errorf("获取复制状态字段失败: %v", err)
	}

	// 创建扫描目标
	values := make([]interface{}, len(columns))
	scanArgs := make([]interface{}, len(values))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	err = rows.Scan(scanArgs...)
	if err != nil {
		return metrics, fmt.Errorf("扫描复制状态失败: %v", err)
	}

	// 解析复制状态
	for i, col := range columns {
		if values[i] != nil {
			value := string(values[i].([]byte))
			switch col {
			case "Slave_IO_Running":
				metrics.SlaveIORunning = value
			case "Slave_SQL_Running":
				metrics.SlaveSQLRunning = value
			case "Seconds_Behind_Master":
				if val, err := strconv.Atoi(value); err == nil {
					metrics.SecondsBehindMaster = val
				}
			case "Master_Log_File":
				metrics.MasterLogFile = value
			case "Read_Master_Log_Pos":
				if val, err := strconv.Atoi(value); err == nil {
					metrics.ReadMasterLogPos = val
				}
			case "Relay_Log_File":
				metrics.RelayLogFile = value
			case "Relay_Log_Pos":
				if val, err := strconv.Atoi(value); err == nil {
					metrics.RelayLogPos = val
				}
			}
		}
	}

	return metrics, nil
}

// GetLockMetrics 获取锁与阻塞指标
func (c *MySQLCollectorImpl) GetLockMetrics(ctx context.Context) (collector.LockMetrics, error) {
	var metrics collector.LockMetrics

	// 获取锁相关指标
	queries := map[string]interface{}{
		"Innodb_row_lock_waits":      &metrics.RowLockWaits,
		"Innodb_row_lock_time":       &metrics.RowLockTime,
		"Innodb_deadlocks":           &metrics.Deadlocks,
		"Table_locks_waited":         &metrics.TableLocksWaited,
		"Table_locks_immediate":      &metrics.TableLocksImmediate,
	}

	for key, ptr := range queries {
		query := fmt.Sprintf("SHOW GLOBAL STATUS LIKE '%s'", key)
		var name string
		var value string
		
		err := c.db.QueryRowContext(ctx, query).Scan(&name, &value)
		if err != nil {
			// 如果查询失败，设置默认值0，不返回错误
			switch ptr := ptr.(type) {
			case *int:
				*ptr = 0
			}
			continue
		}
		
		val, err := strconv.Atoi(value)
		if err != nil {
			val = 0
		}
		
		switch ptr := ptr.(type) {
		case *int:
			*ptr = val
		}
	}

	return metrics, nil
}

// GetTransactionMetrics 获取事务与日志指标
func (c *MySQLCollectorImpl) GetTransactionMetrics(ctx context.Context) (collector.TransactionMetrics, error) {
	var metrics collector.TransactionMetrics

	// 获取事务相关指标
	queries := map[string]interface{}{
		"innodb_log_waits":          &metrics.InnodbLogWaits,
		"innodb_log_write_requests": &metrics.InnodbLogWriteRequests,
	}

	for key, ptr := range queries {
		query := fmt.Sprintf("SHOW GLOBAL STATUS LIKE '%s'", key)
		var name string
		var value string
		
		err := c.db.QueryRowContext(ctx, query).Scan(&name, &value)
		if err != nil {
			return metrics, fmt.Errorf("查询%s失败: %v", key, err)
		}
		
		val, err := strconv.Atoi(value)
		if err != nil {
			return metrics, fmt.Errorf("解析%s值失败: %v", key, err)
		}
		
		switch ptr := ptr.(type) {
		case *int:
			*ptr = val
		}
	}

	// 获取未提交事务数（需要查询INFORMATION_SCHEMA）
	query := "SELECT COUNT(*) FROM INFORMATION_SCHEMA.INNODB_TRX"
	err := c.db.QueryRowContext(ctx, query).Scan(&metrics.UncommittedTransactions)
	if err != nil {
		return metrics, fmt.Errorf("查询未提交事务数失败: %v", err)
	}

	// 获取Binlog信息
	query = "SHOW BINARY LOGS"
	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return metrics, fmt.Errorf("查询Binlog信息失败: %v", err)
	}
	defer rows.Close()

	metrics.BinlogFiles = 0
	metrics.BinlogSize = 0
	for rows.Next() {
		var filename string
		var size int64
		err := rows.Scan(&filename, &size)
		if err != nil {
			continue
		}
		metrics.BinlogFiles++
		metrics.BinlogSize += size
	}

	// 计算Binlog增长速率（简化处理，实际需要历史数据）
	metrics.BinlogGrowthRate = 0.0

	return metrics, nil
}

// Close 关闭连接
func (c *MySQLCollectorImpl) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}
