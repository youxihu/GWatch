# GWatch - 企业级服务器监控告警系统

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen.svg)](Makefile)

## 📖 项目简介

GWatch 是一个基于 Go 语言开发的企业级服务器监控告警系统，采用 DDD（领域驱动设计）架构，提供全面的系统资源监控、服务健康检查、智能告警通知和定时状态报告功能。

### 🎯 核心特性

- **🔍 全方位监控**: CPU、内存、磁盘、网络、Redis、HTTP接口等
- **⚡ 实时告警**: 智能阈值判断 + 防抖策略，避免误报（基础与 HTTP 各自独立的间隔与计数）
- **📱 多渠道通知**: 支持钉钉机器人、邮件等多种告警方式
- **🔄 自动恢复**: 高负载时自动触发Java堆转储，便于问题定位
- **⏰ 定时报告**: 支持定时设备状态报告，主动推送系统健康状态
- **⚙️ 配置驱动**: 零代码修改，通过配置文件即可扩展监控范围
- **🏗️ 架构清晰**: 分层设计，易于扩展和维护（用例层 Coordinator 负责双周期调度）

## 🚀 快速开始

### 环境要求

- Go 1.21+
- Linux/Unix 系统
- Redis 服务（可选）
- 钉钉机器人配置（可选）

### 安装运行

```bash
# 克隆项目
git clone <repository-url>
cd GWatch

# 配置监控参数
cp config/config_example.yml config/config.yml
vim config/config.yml

# 运行监控
make run

# 构建二进制文件
make build
```

## ⚙️ 配置说明

### 基础配置结构

```yaml
# GWatch 监控工具配置文件
redis:
  addr: "localhost:6379"
  password: ""
  db: 0
  timeout: 5s
  pool_size: 5
  min_idle_conns: 1
  max_idle_conns: 3

dingtalk:
  webhook_url: "https://oapi.dingtalk.com/robot/send?access_token=YOUR_TOKEN"
  secret: "YOUR_SECRET"
  at_mobiles: ["13800138000"]

monitor:
  interval: 30s                    # 基础指标监控间隔（CPU/内存/磁盘/网络/Redis）
  consecutive_threshold: 3         # 连续触发次数阈值 
  cpu_threshold: 80.0              # CPU告警阈值
  memory_threshold: 70.0           # 内存告警阈值
  disk_threshold: 80.0             # 磁盘告警阈值
  redis_min_clients: 0             # Redis最小连接数
  redis_max_clients: 100           # Redis最大连接数
  alert_interval: 2m               # 告警间隔（防抖）
  http_interval: 20s               # HTTP 接口监控专用间隔
  
  # HTTP接口监控配置
  http_interfaces:                 # 并发请求，逐项超时独立
    - name: "VMS系统登录页验证码接口"
      url: "https://vms.example.com/prod-api/captchaImage"
      timeout: 10s
      need_alert: true
      allowed_codes: [200, 201, 204]
    - name: "用户登录接口"
      url: "https://vms.example.com/prod-api/login"
      timeout: 15s
      need_alert: true
      allowed_codes: [200, 201]
    - name: "健康检查接口"
      url: "https://vms.example.com/prod-api/health"
      timeout: 5s
      need_alert: false
      allowed_codes: [200, 201, 204, 401, 403]

# 定时报告配置
tickers:
  alert_title: "视频设备状态定时报告"  # 报告标题 用于钉钉报告
  http_interfaces:
    - name: "number of monitoring devices"
      url: "https://vms.example.com/prod-api/api/device/query/getDDCTree"
      Authorization: "Bearer YOUR_TOKEN"
      Cookie: "Admin-Token=YOUR_TOKEN; sidebarStatus=0"
      alert_time: ["09:00", "14:00", "18:00"]  # 定时报告时间点

log:
  mode: zap
  level: info
  output: logs/gwatch.log

javaAppDumpScript:
  path: "/path/to/java-dump-script.sh"
```

### 监控指标说明（结合当前实现）

| 指标类型 | 监控内容 | 告警条件 | 说明 |
|---------|---------|---------|------|
| **CPU** | 使用率百分比 | > 80% | 连续3次超阈值触发告警 |
| **内存** | 使用率百分比 | > 70% | 连续3次超阈值触发告警 |
| **磁盘** | 使用率、IO速率 | > 80% | 瞬时超阈值触发告警 |
| **网络** | 上传/下载速率 | 监控失败 | 网络异常时触发告警 |
| **Redis** | 连接数、连接详情 | < 0 或 > 100 | 连接数异常时触发告警 |
| **HTTP接口** | 可用性、响应时间、状态码 | 可配置允许状态码 | 连续型计数 + 防抖，独立间隔 |
| **定时报告** | 设备状态、系统指标 | 定时触发 | 主动推送系统健康状态和设备统计 |

## 🏗️ 系统架构

### 分层架构设计

```
┌─────────────────────────────────────────────────────────────┐
│                        cmd/main.go                          │
│                    程序入口，Wire依赖注入                    │
└─────────────────────────────────────────────────────────────┘
                                │
┌─────────────────────────────────────────────────────────────┐
│                    internal/app/usecase                     │
│                应用层：监控流程编排                          │
│  MonitoringUseCase + Coordinator + SystemMetricsService：   │
│  - UseCase：采集 → 判断 → 策略 → 格式化 → 发送              │
│  - Coordinator：双周期调度（基础/HTTP），按需补采与合并通知   │
│  - SystemMetricsService：统一系统指标收集服务               │
└─────────────────────────────────────────────────────────────┘
                                │
┌─────────────────────────────────────────────────────────────┐
│                   internal/domain                           │
│                领域层：接口定义和实体                        │
│        collector, monitor, alert, config, notifier         │
└─────────────────────────────────────────────────────────────┘
                                │
┌─────────────────────────────────────────────────────────────┐
│                   internal/infra                            │
│                基础设施层：具体实现                          │
│    collectors, monitor, alert, config, notifier           │
└─────────────────────────────────────────────────────────────┘
```

### 🔧 依赖注入架构

项目采用 **Google Wire** 进行依赖注入管理，实现了：

- **编译时依赖注入**：零运行时开销，类型安全
- **清晰的依赖关系**：所有依赖在 `cmd/wire.go` 中统一管理
- **易于测试**：可以轻松注入 mock 对象
- **代码简洁**：main.go 从 140+ 行减少到 60+ 行

#### Wire 配置文件结构

```
cmd/
├── main.go          # 程序入口，使用 InitializeApp()
├── wire.go          # Wire 配置文件，定义依赖提供者
└── wire_gen.go      # Wire 自动生成的依赖注入代码
```

#### 依赖注入流程

```go
// 1. 定义提供者函数
func NewSystemMetricsService(...) *SystemMetricsService

// 2. 配置依赖图
var ProviderSet = wire.NewSet(
    NewConfigProvider,
    NewSystemMetricsService,
    // ...
)

// 3. 生成注入器
func InitializeApp() (*App, error) {
    wire.Build(ProviderSet, NewApp)
    return &App{}, nil
}

// 4. 在 main.go 中使用
app, err := InitializeApp()
```

### 核心组件

#### 1. 数据采集器 (Collectors)
- **HostCollector**: 系统资源监控（CPU、内存、磁盘、网络）
- **RedisCollector**: Redis服务监控（连接数、客户端详情）
- **HTTPCollector**: HTTP接口监控（可用性、响应时间、状态码；接口并发检测）
- **TickerCollector**: 定时报告数据采集（设备状态、接口健康检查）

#### 2. 监控评估器 (Evaluator)
- **SimpleEvaluator**: 阈值比较，输出监控决策
- 支持自定义阈值配置
- 区分连续型和瞬时型告警

#### 3. 告警策略 (Policy)
- **StatefulPolicy**: 智能告警策略
- 防抖机制：避免频繁告警（HTTP 使用 `http_interval`，其他使用 `alert_interval`）
- 连续计数：连续型（CPU/内存/HTTP）独立计数；非连续型为瞬时触发

#### 4. 通知器 (Notifier)
- **DingTalkNotifier**: 钉钉机器人通知
- 支持Markdown格式
- 可扩展其他通知方式

#### 5. 定时调度器 (Scheduler)
- **TickerScheduler**: 定时报告调度器
- 支持多时间点配置
- 防重复报告机制
- 高精度时间检查（10秒间隔）

## 📊 监控功能详解

### 系统资源监控

#### CPU监控
- 实时CPU使用率
- 高负载时自动获取Top进程信息
- 连续3次超阈值触发告警

#### 内存监控
- 内存使用率和总量统计
- 高负载时自动获取内存占用Top进程
- 自动触发Java堆转储（如果配置了脚本）

#### 磁盘监控
- 磁盘使用率和IO速率
- 支持读写IO分别监控
- 可配置使用率阈值

#### 网络监控
- 网络上传/下载速率
- 基于差值计算，首轮为基线

### 服务监控

#### Redis监控
- 连接数统计（排除监控自身连接）
- 客户端连接详情
- 支持连接数上下限告警

#### HTTP接口监控
- 接口可用性检查（并发发起，独立超时）
- 响应时间统计、允许状态码过滤（allowed_codes）
- 独立监控间隔 `http_interval` 与独立连续计数
- 触发时按需补采基础指标合并通知（反之亦然）

### 定时报告功能

#### 设备状态报告
- 定时调用设备状态API，获取在线/离线设备统计
- 结合系统监控指标，生成综合健康报告
- 支持多时间点配置（如：09:00, 14:00, 18:00）
- 防重复报告机制，确保每个时间点只报告一次

#### 报告内容
- 系统监控指标（CPU、内存、磁盘、Redis、网络）
- HTTP接口状态详情
- 设备状态概览（在线/离线数量、在线率）
- 监控时间戳

### 智能告警

#### 告警策略
- **防抖机制**: 同类告警间隔控制（HTTP 使用 `http_interval`）
- **连续计数**: 避免瞬时尖峰误报（基础/HTTP 独立推进）
- **分级告警**: 支持不同告警类型

#### 告警内容
- 详细的监控指标展示
- 异常进程信息（CPU/内存告警时）
- Java堆转储状态提示
- HTTP接口详细状态

## ⏰ 定时报告功能详解

### 功能概述

定时报告功能允许系统在指定时间点主动推送设备状态和系统健康报告，无需等待异常告警。这对于日常运维监控和定期状态汇报非常有用。

### 配置说明

```yaml
tickers:
  alert_title: "视频设备状态定时报告"  # 报告标题
  http_interfaces:
    - name: "设备状态接口"
      url: "https://api.example.com/device/status"
      Authorization: "Bearer YOUR_TOKEN"
      Cookie: "Admin-Token=YOUR_TOKEN"
      alert_time: ["09:00", "14:00", "18:00"]  # 支持多个时间点
```

### 核心特性

#### 1. 多时间点支持
- 支持配置多个报告时间点
- 时间格式：`HH:MM`（24小时制）
- 示例：`["09:00", "14:00", "18:00"]`

#### 2. 防重复报告
- 每个时间点只报告一次
- 10秒精度检查，避免重复触发
- 智能跳过机制

#### 3. 综合数据收集
- 设备状态数据（在线/离线统计）
- 系统监控指标（CPU、内存、磁盘等）
- HTTP接口健康状态
- 网络和Redis状态

#### 4. 高精度调度
- 每10秒检查一次时间匹配
- 启动时立即检查，避免错过时间点
- 支持应用重启后继续调度

### 使用场景

1. **日常运维报告**：每日定时推送系统健康状态
2. **设备状态监控**：定期检查设备在线率
3. **业务状态汇报**：向管理层定期汇报系统运行情况
4. **故障预防**：主动发现潜在问题

### 报告内容结构

```
## 视频设备状态定时报告

### 完整监控指标
- CPU、内存、磁盘使用率
- Redis连接数
- 网络IO和磁盘IO

### HTTP接口状态
- 各接口响应状态
- 响应时间和状态码

### 设备状态概览
- 在线/离线设备数量
- 总设备数和在线率

### 监控时间
- 报告生成时间戳
```

## 🔧 扩展开发

### 添加新的监控指标

1. **定义接口** (domain层)
```go
type NewCollector interface {
    Collect() (interface{}, error)
}
```

2. **实现接口** (infra层)
```go
type NewCollectorImpl struct {
    // 实现逻辑
}
```

3. **注入使用** (app层)
```go
// 在MonitoringUseCase中添加
```

### 添加新的通知方式

1. **实现Notifier接口**
```go
type EmailNotifier struct {
    // 邮件发送逻辑
}
```

2. **配置注入**
```go
// 在main.go中替换或添加
```

## 📝 使用示例

### 基本监控

```bash
# 启动监控
make run

# 查看实时输出
2025/08/29 16:16:03 GWatch 服务器监控工具启动
2025/08/29 16:16:03 正在初始化...
2025/08/29 16:16:03 开始监控...
2025/08/29 16:16:03 监控间隔: 3s
2025/08/29 16:16:03 HTTP监控间隔: 20s
===========采集数据============
CPU 使用率: 2.07%
内存使用: 17.16% (4158/24237 MB)
磁盘使用: 23.04% (64/294 GB)
Redis 连接数: 3
网络: 下载 5.43 KB/s | 上传 1.35 KB/s
磁盘IO: 读 0.00 KB/s | 写 97.48 KB/s
HTTP接口 vms-public-captcha [需告警]: 正常 (状态码: 200, 响应时间: 1.006623712s)
HTTP接口 vms-private-captcha [仅监控]: 正常 (状态码: 200, 响应时间: 132.714494ms)
监控时间: 2025-08-29 16:16:08
```

### 告警通知示例

当系统出现异常时，钉钉会收到如下格式的通知：

```markdown
## 香港视频化服务器告警

### 触发告警项

> CPU 使用率过高: 85.20%（元凶: java PID=1234 45.20% CPU）

### 完整监控指标

**CPU**: 85.20% [异常]

**内存**: 75.30% (18245/24237 MB) [异常]

**磁盘**: 45.20% (133/294 GB) [正常]

**Redis**: 15个连接 [正常]

**HTTP接口**:

- VMS系统登录页验证码接口: 正常 (状态码: 200, 响应时间: 286ms)

**监控时间**: 2025-08-27 11:39:10
```

### 定时报告示例

定时报告会主动推送系统健康状态，格式如下：

```markdown
## 视频设备状态定时报告

### 完整监控指标

**CPU**: 4.91% [正常]

**内存**: 18.38% (4453/24237 MB) [正常]

**磁盘**: 23.07% (64/294 GB) [正常]

**Redis**: 3个连接 [正常]

**网络IO**: 下载 2.91 KB/s | 上传 0.54 KB/s

**磁盘IO**: 读 0.00 KB/s | 写 0.00 KB/s

**HTTP接口**:

- vms-public-captcha: 正常 (状态码: 200, 响应时间: 276.10377ms)
- vms-private-captcha: 正常 (状态码: 200, 响应时间: 76.078856ms)
- example-captcha: 异常 (状态码: 0) - HTTP请求失败: Get "http://172.25.216.169:7080/captchaImage": context deadline exceeded

### 设备状态概览

- **在线设备**: 0 台
- **离线设备**: 201 台
- **总设备数**: 201 台
- **在线率**: 0.00%

**监控时间**: 2025-08-29 17:40:49
```

## 🚨 故障排查

### 常见问题

#### 1. 监控数据异常
- 检查配置文件格式
- 确认监控间隔设置
- 查看系统日志

#### 2. 告警不触发
- 检查阈值配置
- 确认告警间隔设置
- 查看防抖策略日志

#### 3. 通知发送失败
- 检查钉钉机器人配置
- 确认网络连接
- 查看错误日志

#### 4. 定时报告不触发
- 检查tickers配置格式
- 确认alert_time时间格式（HH:MM）
- 查看调度器日志
- 确认防重复报告机制

### 日志分析

```bash
# 查看详细日志
tail -f /var/log/gwatch.log

# 常见日志模式
[INFO] CPU 连续第 1/3 次超阈值，暂不告警
[WARN] CPU 连续第 3 次超阈值，告警已触发
[INFO] 已自动触发 Java 堆转储生成（异步执行中）...
[INFO] 启动时匹配到告警时间，立即执行设备状态报告
[INFO] 定时器触发：开始执行设备状态报告
[INFO] 时间点 14:16 已在本分钟内报告过，跳过
```

## 📈 性能优化

### 监控间隔调优
- 生产环境建议：30s-60s
- 测试环境建议：10s-30s
- 根据系统负载调整

### 资源使用
- 内存占用：约 50-100MB
- CPU占用：监控间隔内 < 1%
- 网络IO：最小化，仅告警时发送

## 📝 更新日志

### v2.0.0 - Wire 依赖注入重构 (2025-09-03)

#### 🎉 重大更新

**依赖注入架构升级**
- ✅ 引入 Google Wire 进行依赖注入管理
- ✅ 实现编译时依赖注入，零运行时开销
- ✅ main.go 代码从 140+ 行精简到 60+ 行
- ✅ 所有依赖关系在 `cmd/wire.go` 中统一管理

**代码重构优化**
- ✅ 创建 `SystemMetricsService` 统一系统指标收集
- ✅ 消除重复代码，提高代码复用性
- ✅ 优化命名规范，提升代码可读性
- ✅ 重构目录结构，提升扩展性

**架构改进**
- ✅ 重命名 `service/` 目录为 `external/`，避免与 DDD Service 混淆
- ✅ 重命名 `dumpscript_result.go` 为 `java_dump_result.go`
- ✅ 重命名 `dump_script.go` 为 `java_dump_script.go`
- ✅ 提取公共方法，消除代码重复

#### 🔧 技术改进

**依赖注入**
- 使用 Wire 自动生成依赖注入代码
- 类型安全的依赖关系管理
- 易于单元测试和集成测试

**代码质量**
- 消除约 40+ 行重复代码
- 提升代码可维护性
- 改善分层架构清晰度

**扩展性**
- `external/` 目录可轻松扩展 MySQL、PostgreSQL 等收集器
- 统一的系统指标收集服务
- 清晰的职责分离

#### 📊 性能提升

- 编译时依赖注入，零运行时开销
- 减少内存占用和初始化时间
- 提升代码执行效率

---

**GWatch** - 让服务器监控变得简单而强大！ 🚀

