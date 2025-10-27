# GWatch 企业级服务器监控系统

## 系统概述

GWatch 是一款专为企业内部使用的高性能服务器监控系统，提供全面的系统监控、应用监控、告警通知和定时报告功能。系统采用领域驱动设计（DDD）架构，确保代码的可维护性和扩展性。

## 核心功能

### 1. 主机监控
- **CPU监控**：实时监控CPU使用率，支持阈值告警
- **内存监控**：监控内存使用情况，包括使用率和总量
- **磁盘监控**：监控磁盘使用率和IO性能
- **网络监控**：监控网络上传下载速率
- **进程监控**：支持白名单过滤，避免误报

### 2. 应用监控
- **Redis监控**：监控Redis连接数、连接详情和性能指标
- **MySQL监控**：全面的MySQL性能监控，包括连接数、QPS、慢查询、Buffer Pool等
- **HTTP接口监控**：监控HTTP接口的可用性、响应时间和状态码

### 3. 告警系统
- **智能告警**：基于阈值的智能告警机制
- **防抖机制**：避免频繁告警，支持连续触发阈值配置
- **多渠道通知**：支持钉钉、邮件等多种通知方式
- **告警策略**：可配置的告警策略和过滤规则

### 4. 定时报告
- **定时推送**：支持多时间点定时推送监控报告
- **完整报告**：包含主机信息、应用状态、网络信息等完整监控数据
- **日志存储**：自动保存告警日志，支持按日期分类存储
- **日志轮转**：自动日志轮转，防止磁盘空间不足

### 5. 高级功能
- **主机信息显示**：自动获取并显示监控主机的IP地址和主机名
- **Java堆转储**：高负载时自动触发Java应用堆转储
- **配置化管理**：所有监控参数均可通过配置文件调整
- **优雅退出**：支持信号处理和优雅关闭

## 项目结构

```
GWatch/
├── cmd/                           # 应用入口
│   ├── main.go                   # 主程序
│   ├── wire.go                   # 依赖注入定义
│   └── wire_gen.go               # Wire生成的代码
│
├── config/                        # 配置文件
│   ├── config.yml                # 主配置文件
│   └── config_new_example.yml    # 配置示例
│
└── internal/                      # 内部代码
    ├── app/usecase/               # 应用层 - 用例实现
    │   ├── monitoring_metrics.go      # 监控指标收集
    │   ├── monitoring_system.go       # 系统指标服务
    │   ├── scheduler_coordinator.go    # 调度协调器
    │   ├── scheduler_push.go           # 定时推送调度
    │   ├── scheduler_ticker.go         # Ticker调度
    │   └── service_logger.go           # 日志服务
    │
    ├── domain/                    # 领域层 - 接口定义
    │   ├── collector/             # 数据收集器接口
    │   │   ├── host_redis_http.go
    │   │   └── mysql.go
    │   ├── config/                # 配置接口
    │   │   └── provider.go
    │   ├── logger/                # 日志接口
    │   │   └── logger.go
    │   ├── monitoring/            # 监控核心（Evaluator/Policy/Formatter/Notifier）
    │   │   └── monitoring.go
    │   ├── scheduled_push/        # 定时推送接口
    │   │   ├── scheduler_usecase.go
    │   │   └── storage.go
    │   └── ticker/                # 定时器接口
    │       ├── ticker_collector.go
    │       ├── ticker_scheduler.go
    │       └── ticker_token.go
    │
    ├── entity/                    # 实体层 - 数据模型
    │   ├── config.go              # 配置实体
    │   ├── monitoring_alert_type.go      # 告警类型
    │   ├── monitoring_metrics.go          # 监控指标
    │   ├── monitoring_process.go          # 进程信息
    │   ├── monitoring_java_dump.go        # Java转储
    │   ├── ticker_device.go               # 设备状态
    │   ├── ticker_error.go                # 错误类型
    │   └── scheduled_push_record.go       # 推送记录
    │
    ├── infra/                     # 基础设施层 - 具体实现
    │   ├── collector/             # 数据收集器实现
    │   │   ├── host/
    │   │   │   └── host_collector.go
    │   │   └── external/
    │   │       ├── redis_collector.go
    │   │       ├── mysql_collector.go
    │   │       └── http_collector.go
    │   ├── config/                # 配置实现
    │   │   └── yaml_provider.go
    │   ├── logger/               # 日志实现
    │   │   ├── console_logger.go
    │   │   ├── file_logger.go
    │   │   ├── logger_factory.go
    │   │   └── log_wrapper.go
    │   ├── monitoring/           # 监控实现
    │   │   ├── simple_evaluator.go
    │   │   ├── policy.go
    │   │   ├── formatter_markdown.go
    │   │   ├── formatter_ticker_markdown.go
    │   │   └── dingtalk.go
    │   ├── scheduled_push/        # 定时推送实现
    │   │   └── file_alert_storage.go
    │   └── ticker/                # 定时器实现
    │       ├── ticker_collector.go
    │       └── auth/
    │           └── auth.go
    │
    └── utils/                     # 工具层
        ├── error_classify.go      # 错误分类
        ├── host.go                # 主机信息
        ├── java_dump.go           # Java转储
        └── process_filter.go      # 进程过滤
```

## 技术架构

### 架构设计
- **领域驱动设计（DDD）**：清晰的领域边界和职责分离
  - **Domain层**：定义业务接口，不依赖基础设施
  - **Infra层**：实现Domain接口，提供具体技术方案
  - **Entity层**：定义业务实体和数据模型
  - **App层**：编排业务逻辑，协调各领域服务
- **依赖注入**：使用Wire进行依赖管理，提高代码可测试性
- **接口抽象**：完善的接口设计，支持多种实现方式
- **配置驱动**：所有功能通过配置文件控制

### 技术栈
- **语言**：Go 1.21+
- **依赖注入**：Google Wire
- **配置管理**：YAML配置文件
- **日志管理**：自定义日志系统，支持文件和控制台输出
- **通知服务**：钉钉机器人、HTTP接口

## 快速开始

### 环境要求
- Go 1.21 或更高版本
- Linux/Unix 系统
- 网络访问权限（用于外部服务监控）

### 安装部署

1. **克隆项目**
```bash
git clone <repository-url>
cd GWatch
```

2. **安装依赖**
```bash
go mod tidy
```

3. **配置系统**
```bash
cp config/config_new_example.yml config/config.yml
# 编辑配置文件，设置监控参数和通知方式
```

4. **生成依赖注入代码**
```bash
make wire
```

5. **启动监控**
```bash
make run
```

### 编译部署
```bash
# 编译
make build

# 运行
./bin/gwatch
```

## 配置说明

### 主机监控配置
```yaml
host_monitoring:
  interval: 5s                    # 监控间隔
  consecutive_threshold: 3        # 连续触发次数阈值
  alert_interval: 2m              # 告警间隔
  alert_title: "服务器告警"        # 告警标题
  cpu_threshold: 80.0             # CPU使用率阈值
  memory_threshold: 70.0          # 内存使用率阈值
  disk_threshold: 80.0            # 磁盘使用率阈值
```

### 应用监控配置
```yaml
app_monitoring:
  # Redis监控
  redis:
    addr: "localhost:6379"
    password: ""
    db: 0
    timeout: 5s
    min_clients: 0
    max_clients: 10
  
  # MySQL监控
  mysql:
    host: "localhost"
    port: 3306
    username: "user"
    password: "password"
    database: "information_schema"
    timeout: 10s
    interval: 60s
    # 各种阈值配置...
  
  # HTTP接口监控
  http:
    error_threshold: 0
    interval: 10s
    interfaces:
      - name: "API接口"
        url: "https://api.example.com/health"
        need_alert: true
        timeout: 10s
        allowed_codes: [200, 201, 204]
```

### 定时推送配置
```yaml
scheduled_push:
  enabled: true
  push_times: ["8:00", "12:00", "18:00"]  # 推送时间点
  title: "服务器性能监控定时报告"
  include_host_monitoring: true
  include_app_monitoring: true
  
  # 告警存储配置
  alert_storage:
    enabled: true
    alert_log_path_template: "logs/%y/%m-%d/scheduled_push-%H%M.log"
    format: "json"
    retention_days: 30
```

### 日志配置
```yaml
log:
  mode: both                      # 输出模式：console/file/both
  level: info                     # 日志级别
  output: logs/gwatch.log         # 日志文件路径
  enable_rotation: true           # 启用日志轮转
  max_size: 100                   # 单个文件最大大小(MB)
  max_age: 30                     # 文件保留天数
  max_backups: 10                 # 最大备份文件数
```

## 监控指标说明

### 主机指标
- **CPU使用率**：系统CPU使用百分比
- **内存使用率**：系统内存使用百分比和绝对值
- **磁盘使用率**：磁盘空间使用百分比和绝对值
- **磁盘IO**：磁盘读写速率
- **网络IO**：网络上传下载速率

### Redis指标
- **连接数**：当前Redis客户端连接数
- **连接详情**：详细的客户端连接信息
- **性能指标**：连接错误数、中断连接数等

### MySQL指标
- **连接指标**：当前连接数、最大连接数、连接使用率
- **查询性能**：QPS、TPS、慢查询数、响应时间
- **Buffer Pool**：命中率、使用率、页面统计
- **锁信息**：行锁等待、死锁统计
- **事务信息**：未提交事务、Binlog增长速率
- **复制状态**：主从复制延迟、GTID状态

### HTTP指标
- **接口状态**：接口可用性、响应时间
- **状态码**：HTTP响应状态码
- **错误统计**：接口错误数量和类型

## 告警规则

### 告警类型
- **CPU过高**：CPU使用率超过阈值
- **内存过高**：内存使用率超过阈值
- **磁盘过高**：磁盘使用率超过阈值
- **Redis异常**：Redis连接异常或连接数异常
- **MySQL异常**：MySQL连接异常或性能指标异常
- **HTTP异常**：HTTP接口不可用或响应异常

### 告警策略
- **防抖机制**：避免短时间内重复告警
- **白名单过滤**：支持进程白名单，避免误报
- **阈值配置**：所有告警阈值均可配置
- **告警间隔**：可配置告警发送间隔

## 日志管理

### 日志类型
1. **运行日志**：程序运行状态和错误信息
2. **告警日志**：定时推送的监控报告
3. **调试日志**：详细的调试信息

### 日志轮转
- **大小轮转**：单个文件达到指定大小时自动轮转
- **时间轮转**：按时间自动轮转日志文件
- **备份管理**：自动管理备份文件数量和保留时间
- **清理机制**：自动清理过期日志文件

## 部署建议

### 生产环境
- 建议部署在独立的监控服务器上
- 确保网络连通性，能够访问被监控的服务
- 配置合适的日志轮转和清理策略
- 设置监控告警，确保监控系统本身正常运行

### 配置优化
- 根据服务器性能调整监控间隔
- 根据业务需求设置合适的告警阈值
- 配置合适的日志保留策略
- 定期检查和优化配置参数

## 故障排查

### 常见问题
1. **监控数据不准确**：检查网络连接和权限配置
2. **告警不发送**：检查通知配置和网络连通性
3. **日志文件过大**：调整日志轮转配置
4. **性能问题**：调整监控间隔和并发数

### 日志分析
- 查看运行日志了解程序状态
- 分析告警日志了解监控数据
- 使用日志轮转功能管理磁盘空间

## 维护指南

### 日常维护
- 定期检查监控数据准确性
- 清理过期日志文件
- 更新配置参数
- 检查系统性能

### 升级更新
- 备份配置文件
- 停止监控服务
- 更新程序文件
- 重新启动服务
- 验证功能正常

## 技术支持

本系统为企业内部使用，如有问题请联系系统管理员。

---

**版本信息**：v1.0.0  
**最后更新**：2025年10月24日  
**维护团队**：系统运维团队