package monitoring

import (
    domainMonitor "GWatch/internal/domain/monitoring"
    "GWatch/internal/entity"
)

// SimpleEvaluator 仅做阈值比较，不做防抖与连续计数
type SimpleEvaluator struct{}

func NewSimpleEvaluator() *SimpleEvaluator { return &SimpleEvaluator{} }

func (s *SimpleEvaluator) Evaluate(cfg *entity.Config, metrics *entity.SystemMetrics) ([]domainMonitor.Decision, error) {
    var decisions []domainMonitor.Decision

    // 主机类监控评估：只有当host_monitoring配置存在且启用时才评估
    if cfg != nil && cfg.HostMonitoring != nil && cfg.HostMonitoring.Enabled {
        if metrics.CPU.Error != nil {
            decisions = append(decisions, domainMonitor.Decision{Type: entity.CPUErr})
        } else if metrics.CPU.Percent > cfg.HostMonitoring.CPUThreshold {
            decisions = append(decisions, domainMonitor.Decision{Type: entity.CPUHigh})
        }

        if metrics.Memory.Error != nil {
            decisions = append(decisions, domainMonitor.Decision{Type: entity.MemErr})
        } else if metrics.Memory.Percent > cfg.HostMonitoring.MemoryThreshold {
            decisions = append(decisions, domainMonitor.Decision{Type: entity.MemHigh})
        }

        if metrics.Disk.Error != nil {
            decisions = append(decisions, domainMonitor.Decision{Type: entity.DiskErr})
        } else if metrics.Disk.Percent > cfg.HostMonitoring.DiskThreshold {
            decisions = append(decisions, domainMonitor.Decision{Type: entity.DiskHigh})
        }

        if metrics.Network.Error != nil {
            decisions = append(decisions, domainMonitor.Decision{Type: entity.NetworkErr})
        }
    }

    // 应用层类监控评估：只有当app_monitoring配置存在且启用时才评估
    if cfg != nil && cfg.AppMonitoring != nil && cfg.AppMonitoring.Enabled {
        // Redis监控评估：只有当Redis配置存在且启用时才评估
        if cfg.AppMonitoring.Redis != nil && cfg.AppMonitoring.Redis.Enabled {
            if metrics.Redis.ConnectionError != nil {
                decisions = append(decisions, domainMonitor.Decision{Type: entity.RedisErr})
            } else {
                if metrics.Redis.ClientCount < cfg.AppMonitoring.Redis.MinClients {
                    decisions = append(decisions, domainMonitor.Decision{Type: entity.RedisLow})
                } else if metrics.Redis.ClientCount > cfg.AppMonitoring.Redis.MaxClients {
                    decisions = append(decisions, domainMonitor.Decision{Type: entity.RedisHigh})
                }
            }
        }

		// MySQL监控评估：只有当MySQL配置存在且启用时才评估
		if cfg.AppMonitoring != nil && cfg.AppMonitoring.MySQL != nil && cfg.AppMonitoring.MySQL.Enabled {
			if metrics.MySQL.Error != nil {
				decisions = append(decisions, domainMonitor.Decision{Type: entity.MySQLConnErr})
			} else {
				// 连接数评估 - 只有当连接数大于0时才评估
				if metrics.MySQL.Connections.ThreadsConnected > 0 && 
					metrics.MySQL.Connections.ConnectionUsage > cfg.AppMonitoring.MySQL.Thresholds.MaxConnectionsUsageWarning {
					decisions = append(decisions, domainMonitor.Decision{Type: entity.MySQLConnHigh})
				}

				// 活跃线程数评估 - 只有当活跃线程数大于0时才评估
				if metrics.MySQL.Connections.ThreadsRunning > cfg.AppMonitoring.MySQL.Thresholds.ThreadsRunningWarning {
					decisions = append(decisions, domainMonitor.Decision{Type: entity.MySQLThreadsHigh})
				}

				// 慢查询评估 - 只有当慢查询数大于0时才评估
				if metrics.MySQL.QueryPerformance.SlowQueries > 0 && 
					metrics.MySQL.QueryPerformance.SlowQueries > cfg.AppMonitoring.MySQL.Thresholds.SlowQueriesRateWarning {
					decisions = append(decisions, domainMonitor.Decision{Type: entity.MySQLSlowQuery})
				}

				// Buffer Pool命中率评估 - 只有当命中率数据有效时才评估
				if metrics.MySQL.BufferPool.HitRate > 0 && 
					metrics.MySQL.BufferPool.HitRate < cfg.AppMonitoring.MySQL.Thresholds.BufferPoolHitRateWarning {
					decisions = append(decisions, domainMonitor.Decision{Type: entity.MySQLBufferLow})
				}

				// 复制延迟评估 - 只有当复制配置启用且延迟大于0时才评估
				if cfg.AppMonitoring.MySQL.Replication != nil && 
					cfg.AppMonitoring.MySQL.Replication.Enabled &&
					metrics.MySQL.Replication != nil && 
					metrics.MySQL.Replication.SecondsBehindMaster > 0 &&
					metrics.MySQL.Replication.SecondsBehindMaster > cfg.AppMonitoring.MySQL.Thresholds.ReplicationDelayWarningSeconds {
					decisions = append(decisions, domainMonitor.Decision{Type: entity.MySQLReplDelay})
				}

				// 死锁评估 - 只有当死锁数大于0时才评估
				if metrics.MySQL.Locks.Deadlocks > 0 && 
					metrics.MySQL.Locks.Deadlocks > cfg.AppMonitoring.MySQL.Thresholds.DeadlocksPerHourWarning {
					decisions = append(decisions, domainMonitor.Decision{Type: entity.MySQLDeadlock})
				}
			}
		}

		// HTTP接口监控评估：只有当HTTP配置存在且启用时才评估
		if cfg.AppMonitoring.HTTP != nil && cfg.AppMonitoring.HTTP.Enabled {
			if metrics.HTTP.Error != nil {
				decisions = append(decisions, domainMonitor.Decision{Type: entity.HTTPErr})
			} else {
				// 统计需要告警的异常接口数量
				errorCount := 0
				for _, httpInterface := range metrics.HTTP.Interfaces {
					// 检查状态码是否在允许的范围内
					isValidCode := false
					if len(httpInterface.AllowedCodes) > 0 {
						for _, allowedCode := range httpInterface.AllowedCodes {
							if httpInterface.StatusCode == allowedCode {
								isValidCode = true
								break
							}
						}
					} else {
						// 如果没有配置allowed_codes，默认只允许200
						isValidCode = (httpInterface.StatusCode == 200)
					}

					// 如果状态码不在允许范围内且需要告警，则计数
					if httpInterface.NeedAlert && !isValidCode {
						errorCount++
					}
				}

				// 如果异常数量超过配置阈值，触发告警
				if errorCount > cfg.AppMonitoring.HTTP.ErrorThreshold {
					decisions = append(decisions, domainMonitor.Decision{Type: entity.HTTPErr})
				}
			}
		}
    }

    return decisions, nil
}


