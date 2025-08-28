# GWatch - 企业级服务器监控告警系统

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen.svg)](Makefile)

## 📖 项目简介

GWatch 是一个基于 Go 语言开发的企业级服务器监控告警系统，采用 DDD（领域驱动设计）架构，提供全面的系统资源监控、服务健康检查和智能告警通知功能。

### 🎯 核心特性

- **🔍 全方位监控**: CPU、内存、磁盘、网络、Redis、HTTP接口等
- **⚡ 实时告警**: 智能阈值判断 + 防抖策略，避免误报
- **📱 多渠道通知**: 支持钉钉机器人、邮件等多种告警方式
- **🔄 自动恢复**: 高负载时自动触发Java堆转储，便于问题定位
- **⚙️ 配置驱动**: 零代码修改，通过配置文件即可扩展监控范围
- **🏗️ 架构清晰**: 分层设计，易于扩展和维护

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
cp config/config.yml.example config/config.yml
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
  interval: 30s                    # 监控间隔
  cpu_threshold: 80.0             # CPU告警阈值
  memory_threshold: 70.0           # 内存告警阈值
  disk_threshold: 80.0             # 磁盘告警阈值
  redis_min_clients: 0             # Redis最小连接数
  redis_max_clients: 100           # Redis最大连接数
  alert_interval: 2m               # 告警间隔（防抖）
  
  # HTTP接口监控配置
  http_interfaces:
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

log:
  debug: true

javaAppDumpScript:
  path: "/path/to/java-dump-script.sh"
```

### 监控指标说明

| 指标类型 | 监控内容 | 告警条件 | 说明 |
|---------|---------|---------|------|
| **CPU** | 使用率百分比 | > 80% | 连续3次超阈值触发告警 |
| **内存** | 使用率百分比 | > 70% | 连续3次超阈值触发告警 |
| **磁盘** | 使用率、IO速率 | > 80% | 瞬时超阈值触发告警 |
| **网络** | 上传/下载速率 | 监控失败 | 网络异常时触发告警 |
| **Redis** | 连接数、连接详情 | < 0 或 > 100 | 连接数异常时触发告警 |
| **HTTP接口** | 可用性、响应时间、状态码 | 非2xx状态码 | 接口异常时触发告警 |

## 🏗️ 系统架构

### 分层架构设计

```
┌─────────────────────────────────────────────────────────────┐
│                        cmd/main.go                          │
│                    程序入口，依赖注入                        │
└─────────────────────────────────────────────────────────────┘
                                │
┌─────────────────────────────────────────────────────────────┐
│                    internal/app/usecase                     │
│                应用层：监控流程编排                          │
│           采集 → 判断 → 策略 → 格式化 → 发送                │
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

### 核心组件

#### 1. 数据采集器 (Collectors)
- **HostCollector**: 系统资源监控（CPU、内存、磁盘、网络）
- **RedisCollector**: Redis服务监控（连接数、客户端详情）
- **HTTPCollector**: HTTP接口监控（可用性、响应时间、状态码）

#### 2. 监控评估器 (Evaluator)
- **SimpleEvaluator**: 阈值比较，输出监控决策
- 支持自定义阈值配置
- 区分连续型和瞬时型告警

#### 3. 告警策略 (Policy)
- **StatefulPolicy**: 智能告警策略
- 防抖机制：避免频繁告警
- 连续计数：CPU/内存需连续3次超阈值才触发

#### 4. 通知器 (Notifier)
- **DingTalkNotifier**: 钉钉机器人通知
- 支持Markdown格式
- 可扩展其他通知方式

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
- 接口可用性检查
- 响应时间统计
- HTTP状态码监控
- 支持批量配置，零代码扩展

### 智能告警

#### 告警策略
- **防抖机制**: 同类告警间隔控制
- **连续计数**: 避免瞬时尖峰误报
- **分级告警**: 支持不同告警类型

#### 告警内容
- 详细的监控指标展示
- 异常进程信息（CPU/内存告警时）
- Java堆转储状态提示
- HTTP接口详细状态

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
===========采集数据============
CPU 使用率: 25.00%
内存使用: 17.80% (4314/24237 MB)
磁盘使用: 19.28% (53/294 GB)
Redis 连接数: 3
网络: 下载 5.73 KB/s | 上传 1.37 KB/s
HTTP接口 VMS系统登录页验证码接口: 正常 (状态码: 200, 响应时间: 448ms)
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

### 日志分析

```bash
# 查看详细日志
tail -f /var/log/gwatch.log

# 常见日志模式
[INFO] CPU 连续第 1/3 次超阈值，暂不告警
[WARN] CPU 连续第 3 次超阈值，告警已触发
[INFO] 已自动触发 Java 堆转储生成（异步执行中）...
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

---

**GWatch** - 让服务器监控变得简单而强大！ 🚀

