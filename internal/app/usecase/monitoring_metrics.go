package usecase

import (
	domainAlert "GWatch/internal/domain/monitoring"
	"GWatch/internal/domain/collector"
	domainMonitor "GWatch/internal/domain/monitoring"
	"GWatch/internal/entity"
	"GWatch/internal/utils"
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// MonitoringUseCase 负责完整的监控流程：采集 → 判断 → 策略 → 格式化 → 发送
type MonitoringUseCase struct {
	hostCollector   collector.HostCollector
	redisClient     RedisClient
	mysqlCollector  collector.MySQLCollector
	httpCollector   collector.HTTPCollector
	evaluator       domainMonitor.Evaluator
	alertPolicy     domainAlert.Policy
	alertFormatter  domainAlert.Formatter
	alertNotifier   Notifier
	isRedisInited   bool
	isMySQLInited   bool
	isHTTPInited    bool
}

// RedisClient 是 redis 操作接口
type RedisClient interface {
	Init() error
	GetClients() (int, error)
	GetClientsDetail() ([]entity.ClientInfo, error)
}

// Notifier 发送告警通知
type Notifier interface {
	Send(title, markdown string) error
}

// NewMonitoringUseCase 创建监控用例
func NewMonitoringUseCase(
	hostCollector collector.HostCollector,
	redisClient RedisClient,
	mysqlCollector collector.MySQLCollector,
	httpCollector collector.HTTPCollector,
	evaluator domainMonitor.Evaluator,
	alertPolicy domainAlert.Policy,
	alertFormatter domainAlert.Formatter,
	alertNotifier Notifier,
) *MonitoringUseCase {
	return &MonitoringUseCase{
		hostCollector:  hostCollector,
		redisClient:    redisClient,
		mysqlCollector: mysqlCollector,
		httpCollector:  httpCollector,
		evaluator:      evaluator,
		alertPolicy:    alertPolicy,
		alertFormatter: alertFormatter,
		alertNotifier:  alertNotifier,
	}
}

// Run 执行一次完整的监控流程
func (useCase *MonitoringUseCase) Run(config *entity.Config) error {
	// 1. 采集指标
	metrics := useCase.CollectOnce(config)

	// 2. 打印采集结果（可选，用于本地观察）
	useCase.PrintMetrics(config, metrics)

	// 3. 阈值判断与告警处理
	return useCase.EvaluateAndNotify(config, metrics)
}

func (useCase *MonitoringUseCase) CollectOnce(config *entity.Config) *entity.SystemMetrics {
	metrics := &entity.SystemMetrics{Timestamp: time.Now()}

	// 主机类监控：只有当host_monitoring配置存在且启用时才执行
	if config != nil && config.HostMonitoring != nil && config.HostMonitoring.Enabled {
		// CPU监控
		metrics.CPU.Percent, metrics.CPU.Error = useCase.hostCollector.GetCPUPercent()
		
		// 内存监控
		metrics.Memory.Percent, metrics.Memory.UsedMB, metrics.Memory.TotalMB, metrics.Memory.Error = useCase.hostCollector.GetMemoryUsage()
		
		// 磁盘监控
		metrics.Disk.Percent, metrics.Disk.UsedGB, metrics.Disk.TotalGB, metrics.Disk.Error = useCase.hostCollector.GetDiskUsage()
		metrics.Disk.ReadKBps, metrics.Disk.WriteKBps, _ = useCase.hostCollector.GetDiskIORate()
		
		// 网络监控
		metrics.Network.DownloadKBps, metrics.Network.UploadKBps, metrics.Network.Error = useCase.hostCollector.GetNetworkRate()
	}

	// 应用层类监控：只有当app_monitoring配置存在且启用时才执行
	if config != nil && config.AppMonitoring != nil && config.AppMonitoring.Enabled {
		// Redis监控：只有当Redis配置存在且启用时才监控
		if config.AppMonitoring.Redis != nil && config.AppMonitoring.Redis.Enabled {
		if !useCase.isRedisInited {
			if err := useCase.redisClient.Init(); err != nil {
				metrics.Redis.ConnectionError = err
			} else {
				useCase.isRedisInited = true
			}
		}

		if useCase.isRedisInited {
			clientCount, err := useCase.redisClient.GetClients()
			if err != nil {
				metrics.Redis.ConnectionError = err
			} else {
				metrics.Redis.ClientCount = clientCount
			}
			metrics.Redis.ClientDetails, metrics.Redis.DetailError = useCase.redisClient.GetClientsDetail()
		}
		}

		// MySQL监控：只有当MySQL配置存在且启用时才监控
		if config.AppMonitoring.MySQL != nil && config.AppMonitoring.MySQL.Enabled {
			if !useCase.isMySQLInited {
				if err := useCase.mysqlCollector.Init(); err != nil {
					metrics.MySQL.Error = err
				} else {
					useCase.isMySQLInited = true
				}
			}

			if useCase.isMySQLInited {
				// 收集MySQL连接指标
				mysqlMetrics, err := useCase.mysqlCollector.GetConnectionMetrics(context.Background())
				if err != nil {
					metrics.MySQL.Error = err
				} else {
					metrics.MySQL.Connections = entity.ConnectionMetrics{
						ThreadsConnected:    mysqlMetrics.ThreadsConnected,
						ThreadsRunning:      mysqlMetrics.ThreadsRunning,
						MaxConnections:      mysqlMetrics.MaxConnections,
						ConnectionErrors:    mysqlMetrics.ConnectionErrors,
						AbortedConnects:     mysqlMetrics.AbortedConnects,
						ConnectionUsage:     mysqlMetrics.ConnectionUsage,
					}
				}

				// 收集MySQL查询性能指标
				queryMetrics, err := useCase.mysqlCollector.GetQueryPerformanceMetrics(context.Background())
				if err != nil {
					metrics.MySQL.Error = err
				} else {
					metrics.MySQL.QueryPerformance = entity.QueryPerformanceMetrics{
						QPS:             queryMetrics.QPS,
						TPS:             queryMetrics.TPS,
						SlowQueries:     queryMetrics.SlowQueries,
						P95ResponseTime: queryMetrics.P95ResponseTime,
						P99ResponseTime: queryMetrics.P99ResponseTime,
						Questions:       queryMetrics.Questions,
						Committed:       queryMetrics.Committed,
						RolledBack:      queryMetrics.RolledBack,
					}
				}

				// 收集MySQL Buffer Pool指标
				bufferPoolMetrics, err := useCase.mysqlCollector.GetBufferPoolMetrics(context.Background())
				if err != nil {
					metrics.MySQL.Error = err
				} else {
					metrics.MySQL.BufferPool = entity.BufferPoolMetrics{
						HitRate:       bufferPoolMetrics.HitRate,
						Usage:         bufferPoolMetrics.Usage,
						PagesTotal:    bufferPoolMetrics.PagesTotal,
						PagesData:     bufferPoolMetrics.PagesData,
						PagesFree:     bufferPoolMetrics.PagesFree,
						PagesDirty:    bufferPoolMetrics.PagesDirty,
						ReadRequests:  bufferPoolMetrics.ReadRequests,
						Reads:         bufferPoolMetrics.Reads,
						WriteRequests: bufferPoolMetrics.WriteRequests,
						Writes:        bufferPoolMetrics.Writes,
					}
				}

				// 收集MySQL锁指标
				lockMetrics, err := useCase.mysqlCollector.GetLockMetrics(context.Background())
				if err != nil {
					metrics.MySQL.Error = err
				} else {
					metrics.MySQL.Locks = entity.LockMetrics{
						RowLockWaits: lockMetrics.RowLockWaits,
						RowLockTime:  lockMetrics.RowLockTime,
						Deadlocks:    lockMetrics.Deadlocks,
					}
				}

				// 收集MySQL事务指标
				transactionMetrics, err := useCase.mysqlCollector.GetTransactionMetrics(context.Background())
				if err != nil {
					metrics.MySQL.Error = err
				} else {
					metrics.MySQL.Transactions = entity.TransactionMetrics{
						UncommittedTransactions: transactionMetrics.UncommittedTransactions,
						BinlogGrowthRate:        transactionMetrics.BinlogGrowthRate,
					}
				}

				// 收集MySQL复制指标（如果启用）
				if config.AppMonitoring.MySQL.Replication != nil && config.AppMonitoring.MySQL.Replication.Enabled {
					replicationMetrics, err := useCase.mysqlCollector.GetReplicationMetrics(context.Background())
					if err != nil {
						metrics.MySQL.Error = err
					} else {
						metrics.MySQL.Replication = &entity.ReplicationMetrics{
							SlaveIORunning:        replicationMetrics.SlaveIORunning,
							SlaveSQLRunning:       replicationMetrics.SlaveSQLRunning,
							SecondsBehindMaster:   replicationMetrics.SecondsBehindMaster,
							MasterLogFile:         replicationMetrics.MasterLogFile,
							ReadMasterLogPos:      replicationMetrics.ReadMasterLogPos,
							RelayLogFile:          replicationMetrics.RelayLogFile,
							RelayLogPos:           replicationMetrics.RelayLogPos,
							GTIDMode:              replicationMetrics.GTIDMode,
							GTIDExecuted:          replicationMetrics.GTIDExecuted,
						}
					}
				}
			}
		}

		// HTTP接口监控：只有当HTTP配置存在且启用时才监控
		if config.AppMonitoring.HTTP != nil && config.AppMonitoring.HTTP.Enabled {
			if !useCase.isHTTPInited {
				if err := useCase.httpCollector.Init(); err != nil {
					metrics.HTTP.Error = err
				} else {
					useCase.isHTTPInited = true
				}
			}

			if useCase.isHTTPInited {
				var httpInterfaces []entity.HTTPInterfaceMetrics
				if config.AppMonitoring.HTTP.Interfaces != nil {
					for _, httpConfig := range config.AppMonitoring.HTTP.Interfaces {
						isAccessible, responseTime, statusCode, err := useCase.httpCollector.CheckInterface(httpConfig.URL, httpConfig.Timeout)

						httpInterfaces = append(httpInterfaces, entity.HTTPInterfaceMetrics{
							Name:         httpConfig.Name,
							URL:          httpConfig.URL,
							IsAccessible: isAccessible,
							ResponseTime: responseTime,
							StatusCode:   statusCode,
							Error:        err,
							NeedAlert:    httpConfig.NeedAlert,
							AllowedCodes: httpConfig.AllowedCodes,
						})
					}
				}
				metrics.HTTP.Interfaces = httpInterfaces
			}
		}
	}

	return metrics
}

func (useCase *MonitoringUseCase) EvaluateAndNotify(config *entity.Config, metrics *entity.SystemMetrics) error {
	decisions, _ := useCase.evaluator.Evaluate(config, metrics)
	alertTypes := useCase.alertPolicy.Apply(config, metrics, decisions)
	if len(alertTypes) == 0 {
		return nil
	}

	// 👇 直接委托给 NotifyWithAlertTypes —— 不再自己处理告警构造！
	return useCase.NotifyWithAlertTypes(config, metrics, alertTypes)
}

func (useCase *MonitoringUseCase) EvaluateAndNotifyBaseOnly(config *entity.Config, metrics *entity.SystemMetrics) error {
	decisions, _ := useCase.evaluator.Evaluate(config, metrics)
	var filteredDecisions []domainMonitor.Decision
	for _, decision := range decisions {
		if decision.Type != entity.HTTPErr {
			filteredDecisions = append(filteredDecisions, decision)
		}
	}
	alertTypes := useCase.alertPolicy.Apply(config, metrics, filteredDecisions)
	return useCase.NotifyWithAlertTypes(config, metrics, alertTypes) // ← 统一出口
}

func (useCase *MonitoringUseCase) EvaluateAndNotifyHTTPOnly(config *entity.Config, metrics *entity.SystemMetrics) error {
	decisions, _ := useCase.evaluator.Evaluate(config, metrics)
	var filteredDecisions []domainMonitor.Decision
	for _, decision := range decisions {
		if decision.Type == entity.HTTPErr {
			filteredDecisions = append(filteredDecisions, decision)
		}
	}
	alertTypes := useCase.alertPolicy.Apply(config, metrics, filteredDecisions)
	return useCase.NotifyWithAlertTypes(config, metrics, alertTypes) // ← 统一出口
}

// PrintMetrics 仅用于本地观察，不属于核心业务
func (useCase *MonitoringUseCase) PrintMetrics(config *entity.Config, metrics *entity.SystemMetrics) {
	now := time.Now() // 获取当前时间
	log.Println("===========采集数据============")
	
	// 主机类监控信息 - 只有当host_monitoring配置存在且启用时才显示
	if config != nil && config.HostMonitoring != nil && config.HostMonitoring.Enabled {
		if metrics.CPU.Error != nil {
			log.Println("CPU 监控失败:", metrics.CPU.Error.Error())
		} else {
			log.Printf("CPU 使用率: %.2f%%\n", metrics.CPU.Percent)
		}
		if metrics.Memory.Error != nil {
			log.Println("内存监控失败:", metrics.Memory.Error.Error())
		} else {
			log.Printf("内存使用: %.2f%% (%d/%d MB)\n", metrics.Memory.Percent, metrics.Memory.UsedMB, metrics.Memory.TotalMB)
		}
		if metrics.Disk.Error != nil {
			log.Println("磁盘监控失败:", metrics.Disk.Error.Error())
		} else {
			log.Printf("磁盘使用: %.2f%% (%d/%d GB)\n",
				metrics.Disk.Percent, metrics.Disk.UsedGB, metrics.Disk.TotalGB)
		}
		if metrics.Network.Error != nil {
			log.Println("网络监控失败:", metrics.Network.Error.Error())
		} else {
			log.Printf("网络: 下载 %.2f KB/s | 上传 %.2f KB/s\n", metrics.Network.DownloadKBps, metrics.Network.UploadKBps)
		}
		log.Printf("磁盘IO: 读 %.2f KB/s | 写 %.2f KB/s\n", metrics.Disk.ReadKBps, metrics.Disk.WriteKBps)
	}

	// Redis监控信息 - 只有当app_monitoring和redis配置存在且启用时才显示
	if config != nil && config.AppMonitoring != nil && config.AppMonitoring.Enabled && config.AppMonitoring.Redis != nil && config.AppMonitoring.Redis.Enabled {
		if metrics.Redis.ConnectionError != nil {
			log.Println("Redis 连接失败:", metrics.Redis.ConnectionError.Error())
		} else {
			log.Printf("Redis 连接数: %d\n", metrics.Redis.ClientCount)
		}
	}

	// MySQL监控信息 - 只有当app_monitoring和mysql配置存在且启用时才显示
	if config != nil && config.AppMonitoring != nil && config.AppMonitoring.Enabled && config.AppMonitoring.MySQL != nil && config.AppMonitoring.MySQL.Enabled {
		if metrics.MySQL.Error != nil {
			log.Println("MySQL 连接失败:", metrics.MySQL.Error.Error())
		} else {
			log.Printf("MySQL 连接数: %d/%d (%.2f%%)\n", 
				metrics.MySQL.Connections.ThreadsConnected, 
				metrics.MySQL.Connections.MaxConnections,
				metrics.MySQL.Connections.ConnectionUsage)
			log.Printf("MySQL QPS: %d, 慢查询: %d\n", 
				metrics.MySQL.QueryPerformance.QPS,
				metrics.MySQL.QueryPerformance.SlowQueries)
			log.Printf("MySQL Buffer Pool 命中率: %.2f%%\n", 
				metrics.MySQL.BufferPool.HitRate)
		}
	}

	// HTTP接口监控信息 - 只有当app_monitoring和http配置存在且启用时才显示
	if config != nil && config.AppMonitoring != nil && config.AppMonitoring.Enabled && config.AppMonitoring.HTTP != nil && config.AppMonitoring.HTTP.Enabled {
		if metrics.HTTP.Error != nil {
			log.Println("HTTP接口监控失败:", metrics.HTTP.Error.Error())
		} else {
			for _, httpInterface := range metrics.HTTP.Interfaces {
				alertMark := ""
				if httpInterface.NeedAlert {
					alertMark = " [需告警]"
				} else {
					alertMark = " [仅监控]"
				}

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

				if isValidCode {
					log.Printf("HTTP接口 %s%s: 正常 (状态码: %d, 响应时间: %v)\n",
						httpInterface.Name, alertMark, httpInterface.StatusCode, httpInterface.ResponseTime)
				} else {
					log.Printf("HTTP接口 %s%s: 异常 (状态码: %d) - %v\n",
						httpInterface.Name, alertMark, httpInterface.StatusCode, httpInterface.Error)
				}
			}
		}
	}

	log.Printf("监控时间: %s\n", now.Format(time.DateTime))
}

// CombineMetrics 将基础指标与 HTTP 指标合并为一个整体快照
func CombineMetrics(baseMetrics, httpMetrics *entity.SystemMetrics) *entity.SystemMetrics {
	mergedMetrics := &entity.SystemMetrics{Timestamp: time.Now()}
	if baseMetrics != nil {
		mergedMetrics.CPU = baseMetrics.CPU
		mergedMetrics.Memory = baseMetrics.Memory
		mergedMetrics.Disk = baseMetrics.Disk
		mergedMetrics.Network = baseMetrics.Network
		mergedMetrics.Redis = baseMetrics.Redis
		mergedMetrics.MySQL = baseMetrics.MySQL
	}
	if httpMetrics != nil {
		mergedMetrics.HTTP = httpMetrics.HTTP
	}
	return mergedMetrics
}

// CollectBaseOnce 仅采集基础主机/Redis/网络等指标（不采集 HTTP）
func (useCase *MonitoringUseCase) CollectBaseOnce(config *entity.Config) *entity.SystemMetrics {
	metrics := &entity.SystemMetrics{Timestamp: time.Now()}

	// 主机类监控：只有当host_monitoring配置存在且启用时才执行
	if config != nil && config.HostMonitoring != nil && config.HostMonitoring.Enabled {
		metrics.CPU.Percent, metrics.CPU.Error = useCase.hostCollector.GetCPUPercent()
		metrics.Memory.Percent, metrics.Memory.UsedMB, metrics.Memory.TotalMB, metrics.Memory.Error = useCase.hostCollector.GetMemoryUsage()
		metrics.Disk.Percent, metrics.Disk.UsedGB, metrics.Disk.TotalGB, metrics.Disk.Error = useCase.hostCollector.GetDiskUsage()
		metrics.Disk.ReadKBps, metrics.Disk.WriteKBps, _ = useCase.hostCollector.GetDiskIORate()
		metrics.Network.DownloadKBps, metrics.Network.UploadKBps, metrics.Network.Error = useCase.hostCollector.GetNetworkRate()
	}

	// Redis监控：只有当app_monitoring存在且启用，Redis配置存在且启用时才执行
	if config != nil && config.AppMonitoring != nil && config.AppMonitoring.Enabled && config.AppMonitoring.Redis != nil && config.AppMonitoring.Redis.Enabled {
		if !useCase.isRedisInited {
			if err := useCase.redisClient.Init(); err != nil {
				metrics.Redis.ConnectionError = err
			} else {
				useCase.isRedisInited = true
			}
		}

		if useCase.isRedisInited {
			clientCount, err := useCase.redisClient.GetClients()
			if err != nil {
				metrics.Redis.ConnectionError = err
			} else {
				metrics.Redis.ClientCount = clientCount
			}
			metrics.Redis.ClientDetails, metrics.Redis.DetailError = useCase.redisClient.GetClientsDetail()
		}
	}

	// MySQL监控：只有当app_monitoring存在且启用，MySQL配置存在且启用时才执行
	if config != nil && config.AppMonitoring != nil && config.AppMonitoring.Enabled && config.AppMonitoring.MySQL != nil && config.AppMonitoring.MySQL.Enabled {
		if !useCase.isMySQLInited {
			if err := useCase.mysqlCollector.Init(); err != nil {
				metrics.MySQL.Error = err
			} else {
				useCase.isMySQLInited = true
			}
		}

		if useCase.isMySQLInited {
			// 收集MySQL连接指标
			mysqlMetrics, err := useCase.mysqlCollector.GetConnectionMetrics(context.Background())
			if err != nil {
				metrics.MySQL.Error = err
			} else {
				metrics.MySQL.Connections = entity.ConnectionMetrics{
					ThreadsConnected:    mysqlMetrics.ThreadsConnected,
					ThreadsRunning:      mysqlMetrics.ThreadsRunning,
					MaxConnections:      mysqlMetrics.MaxConnections,
					ConnectionErrors:    mysqlMetrics.ConnectionErrors,
					AbortedConnects:     mysqlMetrics.AbortedConnects,
					ConnectionUsage:     mysqlMetrics.ConnectionUsage,
				}
			}

			// 收集MySQL查询性能指标
			queryMetrics, err := useCase.mysqlCollector.GetQueryPerformanceMetrics(context.Background())
			if err != nil {
				metrics.MySQL.Error = err
			} else {
				metrics.MySQL.QueryPerformance = entity.QueryPerformanceMetrics{
					QPS:             queryMetrics.QPS,
					TPS:             queryMetrics.TPS,
					SlowQueries:     queryMetrics.SlowQueries,
					P95ResponseTime: queryMetrics.P95ResponseTime,
					P99ResponseTime: queryMetrics.P99ResponseTime,
					Questions:       queryMetrics.Questions,
					Committed:       queryMetrics.Committed,
					RolledBack:      queryMetrics.RolledBack,
				}
			}

			// 收集MySQL Buffer Pool指标
			bufferPoolMetrics, err := useCase.mysqlCollector.GetBufferPoolMetrics(context.Background())
			if err != nil {
				metrics.MySQL.Error = err
			} else {
				metrics.MySQL.BufferPool = entity.BufferPoolMetrics{
					HitRate:       bufferPoolMetrics.HitRate,
					Usage:         bufferPoolMetrics.Usage,
					PagesTotal:    bufferPoolMetrics.PagesTotal,
					PagesData:     bufferPoolMetrics.PagesData,
					PagesFree:     bufferPoolMetrics.PagesFree,
					PagesDirty:    bufferPoolMetrics.PagesDirty,
					ReadRequests:  bufferPoolMetrics.ReadRequests,
					Reads:         bufferPoolMetrics.Reads,
					WriteRequests: bufferPoolMetrics.WriteRequests,
					Writes:        bufferPoolMetrics.Writes,
				}
			}

			// 收集MySQL锁指标
			lockMetrics, err := useCase.mysqlCollector.GetLockMetrics(context.Background())
			if err != nil {
				metrics.MySQL.Error = err
			} else {
				metrics.MySQL.Locks = entity.LockMetrics{
					RowLockWaits: lockMetrics.RowLockWaits,
					RowLockTime:  lockMetrics.RowLockTime,
					Deadlocks:    lockMetrics.Deadlocks,
				}
			}

			// 收集MySQL事务指标
			transactionMetrics, err := useCase.mysqlCollector.GetTransactionMetrics(context.Background())
			if err != nil {
				metrics.MySQL.Error = err
			} else {
				metrics.MySQL.Transactions = entity.TransactionMetrics{
					UncommittedTransactions: transactionMetrics.UncommittedTransactions,
					BinlogGrowthRate:        transactionMetrics.BinlogGrowthRate,
				}
			}

			// 收集MySQL复制指标（如果启用）
			if config.AppMonitoring.MySQL.Replication != nil && config.AppMonitoring.MySQL.Replication.Enabled {
				replicationMetrics, err := useCase.mysqlCollector.GetReplicationMetrics(context.Background())
				if err != nil {
					metrics.MySQL.Error = err
				} else {
					metrics.MySQL.Replication = &entity.ReplicationMetrics{
						SlaveIORunning:        replicationMetrics.SlaveIORunning,
						SlaveSQLRunning:       replicationMetrics.SlaveSQLRunning,
						SecondsBehindMaster:   replicationMetrics.SecondsBehindMaster,
						MasterLogFile:         replicationMetrics.MasterLogFile,
						ReadMasterLogPos:      replicationMetrics.ReadMasterLogPos,
						RelayLogFile:          replicationMetrics.RelayLogFile,
						RelayLogPos:           replicationMetrics.RelayLogPos,
						GTIDMode:              replicationMetrics.GTIDMode,
						GTIDExecuted:          replicationMetrics.GTIDExecuted,
					}
				}
			}
		}
	}

	return metrics
}

// CollectHTTPOnce 仅采集 HTTP 接口指标
func (useCase *MonitoringUseCase) CollectHTTPOnce(config *entity.Config) *entity.SystemMetrics {
	metrics := &entity.SystemMetrics{Timestamp: time.Now()}

	// HTTP接口监控：只有当app_monitoring存在且启用，HTTP配置存在且启用时才执行
	if config != nil && config.AppMonitoring != nil && config.AppMonitoring.Enabled && config.AppMonitoring.HTTP != nil && config.AppMonitoring.HTTP.Enabled {
		if !useCase.isHTTPInited {
			if err := useCase.httpCollector.Init(); err != nil {
				metrics.HTTP.Error = err
				return metrics
			}
			useCase.isHTTPInited = true
		}

		if useCase.isHTTPInited {
			var httpInterfaces []entity.HTTPInterfaceMetrics
			if config.AppMonitoring.HTTP.Interfaces != nil {
				count := len(config.AppMonitoring.HTTP.Interfaces)
				results := make([]entity.HTTPInterfaceMetrics, count)
				var wg sync.WaitGroup
				wg.Add(count)
				for i := 0; i < count; i++ {
					i := i
					httpConfig := config.AppMonitoring.HTTP.Interfaces[i]
					go func() {
						defer wg.Done()
						isAccessible, responseTime, statusCode, err := useCase.httpCollector.CheckInterface(httpConfig.URL, httpConfig.Timeout)
						results[i] = entity.HTTPInterfaceMetrics{
							Name:         httpConfig.Name,
							URL:          httpConfig.URL,
							IsAccessible: isAccessible,
							ResponseTime: responseTime,
							StatusCode:   statusCode,
							Error:        err,
							NeedAlert:    httpConfig.NeedAlert,
							AllowedCodes: httpConfig.AllowedCodes,
						}
					}()
				}
				wg.Wait()
				httpInterfaces = results
			}
			metrics.HTTP.Interfaces = httpInterfaces
		}
	}

	return metrics
}

// NotifyWithAlertTypes 按给定的告警类型集合直接构建并发送通知（用于“同时告警”合并场景）
func (useCase *MonitoringUseCase) NotifyWithAlertTypes(config *entity.Config, metrics *entity.SystemMetrics, alertTypes []entity.AlertType) error {
	if len(alertTypes) == 0 {
		return nil
	}

	var triggeredAlerts []domainAlert.TriggeredAlert
	isDumpTriggeredAsync := false

	for _, alertType := range alertTypes {
		message, isAsyncDump, isSkipped := useCase.buildAlertMessageAndMaybeDump(config, metrics, alertType)

		if isSkipped || strings.TrimSpace(message) == "" {
			log.Printf("[完全跳过告警] 类型: %v, 原因: %s", alertType, map[bool]string{true: "白名单", false: "消息为空"}[isSkipped])
			continue
		}

		if isAsyncDump {
			isDumpTriggeredAsync = true
		}

		triggeredAlerts = append(triggeredAlerts, domainAlert.TriggeredAlert{
			Type:    alertType,
			Message: message,
		})
	}

	// ✅✅✅ 核心新增：如果所有告警项都被过滤 → 不发送通知
	if len(triggeredAlerts) == 0 {
		log.Println("[INFO] 所有告警项均被过滤，本次通知已取消")
		return nil // 👈 彻底静默，不发任何消息
	}

	if isDumpTriggeredAsync {
		triggeredAlerts = append(triggeredAlerts, domainAlert.TriggeredAlert{
			Type:    entity.Info,
			Message: "检测到高负载，已自动触发 Java 堆转储生成（异步执行中）...",
		})
	}

	alertTitle := "GWatch 服务器告警" // 默认标题
	if config.HostMonitoring != nil && config.HostMonitoring.AlertTitle != "" {
		alertTitle = config.HostMonitoring.AlertTitle
	}
	alertBody := useCase.alertFormatter.Build(alertTitle, config, metrics, triggeredAlerts)
	return useCase.alertNotifier.Send(alertTitle, alertBody)
}

// buildAlertMessageAndMaybeDump 根据告警类型构建消息，并根据需要执行dump脚本
// 返回: 最终消息, 是否触发了异步dump, 是否应跳过该告警（如白名单）
func (useCase *MonitoringUseCase) buildAlertMessageAndMaybeDump(
	config *entity.Config,
	metrics *entity.SystemMetrics,
	alertType entity.AlertType,
) (string, bool, bool) {
	message := alertType.String() // 默认使用 AlertType 的 String()
	isTriggerAsyncDump := false

	if alertType == entity.CPUHigh || alertType == entity.MemHigh {
		if topCPUProcesses, topMemProcesses, err := useCase.hostCollector.GetTopProcesses(5); err == nil {
			var culpritProcess *entity.ProcessInfo

			if alertType == entity.CPUHigh && len(topCPUProcesses) > 0 {
				culpritProcess = &topCPUProcesses[0]
			}
			if alertType == entity.MemHigh && len(topMemProcesses) > 0 {
				culpritProcess = &topMemProcesses[0]
			}

			// ✅✅✅ 白名单核心逻辑 —— 只在这里写一次！
			if culpritProcess != nil && config != nil && utils.IsProcessInWhiteList(culpritProcess.Name, config.WhiteProcessList) {
				log.Printf("[白名单忽略] 进程 '%s' (PID=%d) 触发 %v，已在白名单中，跳过告警", culpritProcess.Name, culpritProcess.PID, alertType)
				return "", false, true // 跳过告警：空消息、不dump、标记跳过
			}

			// 构造详细告警消息
			if alertType == entity.CPUHigh && culpritProcess != nil {
				message = fmt.Sprintf(
					"CPU 使用率过高: %.2f%%（元凶: %s PID=%d %.2f%% CPU）",
					metrics.CPU.Percent, culpritProcess.Name, culpritProcess.PID, culpritProcess.CPUPercent,
				)
			}
			if alertType == entity.MemHigh && culpritProcess != nil {
				message = fmt.Sprintf(
					"内存使用率过高: %.2f%%（元凶: %s PID=%d %.1f%% MEM, %dMB）",
					metrics.Memory.Percent, culpritProcess.Name, culpritProcess.PID, culpritProcess.MemPercent, culpritProcess.MemRSS,
				)
			}

				// 执行脚本并等待最多 3 秒
				if config.JavaAppDumpScript != nil {
					done := make(chan struct{}, 1)
					var result string
					go func() {
						r, err := utils.ExecuteJavaDumpScriptResult(config.JavaAppDumpScript.Path, 3*time.Second)
						if err == nil {
							result = r
						}
						done <- struct{}{}
					}()

					select {
					case <-done:
						if strings.Contains(result, "file_exist") {
							message += "\n\n> 提示：堆转储文件已存在，跳过生成"
						} else if strings.Contains(result, "failed") {
							message += "\n\n> 提示：Java堆转储生成失败"
						} else if strings.Contains(result, "success") {
							message += "\n\n> 提示：已生成 Java 堆转储"
						} else if result != "" {
							message += "\n\n> 提示：" + result
						}
					case <-time.After(3 * time.Second):
						go utils.ExecuteJavaDumpScriptAsync(config.JavaAppDumpScript.Path)
						isTriggerAsyncDump = true
					}
				}
		}
	}

	return message, isTriggerAsyncDump, false
}
