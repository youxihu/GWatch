# GWatch

基于 DDD（领域驱动设计）的轻量监控告警工具。按间隔采集主机与服务（Redis）指标，阈值判断 + 防抖/连续计数策略，通过钉钉发送告警；CPU/内存告警时异步触发 Java 堆转储脚本用于定位。

### 运行
```
make run
```
构建：`make build`

### 配置
编辑 `config/config.yml`：
- monitor: interval、cpu_threshold、memory_threshold、disk_threshold、redis_min_clients、redis_max_clients、alert_interval
- redis: 连接参数
- dingtalk: webhook、secret、at_mobiles
- javaAppDumpScript.path: dump 脚本路径

---

## 架构与代码逻辑

### 分层
- domain：领域接口与实体，稳定抽象
  - collector：主机与服务采集接口（`HostCollector`、`RedisCollector`）
  - monitor：阈值判断接口（`Evaluator`，只做比较，不含策略/消息）
  - alert：策略接口（`Policy`）、格式化接口（`Formatter`）、`TriggeredAlert`
  - config：配置提供者接口（`Provider`）
  - notifier：通知接口（`Notifier`）
- infra：基础设施实现
  - collectors/host：gopsutil 实现 CPU/内存/磁盘/网络速率、TopN 进程
  - collectors/service：go-redis 实现连接数/详情（排除自连）
  - monitor：`SimpleEvaluator` 纯阈值比较
  - alert：`StatefulPolicy`（防抖+CPU/内存连续3次）、`MarkdownFormatter`
  - notifier：`DingTalkNotifier`
  - config：`YAMLProvider` 从 `config.yml` 加载配置
- app：应用编排
  - runtime.Runner：采集 → 阈值判断 → 策略 → 格式化 → 发送；同时在 CPU/内存告警时触发 dump 脚本并在通知中提示
- cmd：程序入口，加载配置、组装依赖、按 `interval` 定时执行

### 指标与采集
- 主机（HostCollector）
  - CPU 平均占用率
  - 内存使用率、已用/总量（MB）
  - 磁盘使用率（根分区），已用/总量（GB）
  - 网络速率（KB/s）：首轮为基线 0，之后按差值计算
  - TopN 进程（CPU/内存），用于“元凶进程”定位
- Redis（RedisCollector）
  - 连接数（排除监控自有连接）
  - 连接详情（排除监控自有连接）

### 阈值判断与策略
- monitor.Evaluator（领域）：仅输出“决策”
- alert.StatefulPolicy（基础设施）：
  - 防抖：同类告警间隔 `alert_interval`
  - 连续计数：CPUHigh、MemHigh 连续 3 次超阈值才触发
  - 日志：输出第几次不告警/第几次触发，方便观测

### 消息格式化与通知
- MarkdownFormatter：
  - “触发告警项”中包含每条的详细消息
  - CPU/内存告警会带“元凶进程”（进程名、PID、占用）
  - 完整监控指标：CPU/内存/磁盘/网络/Redis + 监控时间
  - 若触发 CPU/内存告警：追加“已自动触发 Java 堆转储生成（异步执行中）...”
- DingTalkNotifier：发送 Markdown 消息

### Dump 脚本
- 触发时机：当告警类型包含 CPUHigh 或 MemHigh
- 执行方式：`utils.ExecuteJavaDumpScriptAsync` 异步执行，5 分钟超时，日志输出结果（success/file_exist/failed/timeout）
- 报告呈现：当前通知中包含“已触发 dump”的提示；脚本结果写日志
- 可选扩展：如需“将最近一次脚本结果呈现在后续通知里”，可在应用层缓存最近结果（内存或文件），在 Formatter 中追加展示（保持现有接口不变）

---

## 入口与编排
入口：`cmd/main.go`
1. `configimpl.NewYAMLProvider("config/config.yml")` 加载配置
2. 组装：HostCollector + RedisCollector(provider) + SimpleEvaluator + StatefulPolicy + MarkdownFormatter + DingTalkNotifier(provider)
3. `runtime.Runner` 按 `interval` 定时：
   - `CollectOnce()` 采集
   - `EvaluateAndNotify()` 执行判断/策略/格式化/发送
   - `PrintMetrics()` 简要打印一行 CPU/内存/磁盘/网络/Redis（用于观察节奏与网络）

---

## 运行日志与观测
- 启动信息、阈值、间隔
- 每轮采集的关键指标打印
- 策略日志：第 N/3 次暂不告警、达到 N 次触发等
- dump 脚本执行结果在日志中输出

---

## 常见问题
- 首次网络速率为 0？用于基线初始化，后续才有波动值
- CPU/内存告警为何要 3 次？避免瞬时尖峰误报（可修改策略实现）
- Redis 为什么只初始化一次？减少连接与性能开销

---

## 扩展建议
- 在通知中追加“最近一次 dump 结果”：应用层缓存最近结果并在 Formatter 展示
- 新服务采集：在 domain/collector 增口，infra/collectors 实现，Runner 注入
- 新通知渠道：在 domain/notifier 保持接口，infra/notifier 新实现

