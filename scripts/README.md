# 测试脚本目录

本目录包含用于测试和开发的各种脚本。

## 脚本说明

### `start_all.sh`
快速启动三个 GWatch 实例（config.yml, config_client.yml, config_client2.yml）的脚本。

**使用方法：**
```bash
./scripts/start_all.sh
```

### `test_client_server.sh`
测试多客户端聚合功能的脚本。启动一个 Server 和两个 Client，测试数据聚合和通知发送。

**使用方法：**
```bash
./scripts/test_client_server.sh
```

**功能：**
- 自动计算测试时间点（当前时间 +1 分钟和 +2 分钟）
- 创建/更新配置文件
- 启动 Client 1 和 Client 2（后台运行）
- 启动 Server（前台运行，可看到实时输出）
- 按 Ctrl+C 自动清理所有进程

### `test_high_io.sh`
模拟高网络IO和高磁盘IO的测试脚本，用于测试监控系统在高负载情况下的表现。

**使用方法：**
```bash
./scripts/test_high_io.sh
```

**功能：**
- 启动磁盘IO测试（持续写入/删除文件）
- 启动网络IO测试（持续下载文件）
- 自动运行 `start_all.sh` 启动监控
- 按 Ctrl+C 自动清理所有测试进程和监控进程

**注意：**
- 磁盘IO测试会持续写入文件，可能占用磁盘空间
- 网络IO测试会消耗带宽，请确保网络连接正常
- 测试完成后会自动清理临时文件

### `cleanup.sh`
清理测试产生的临时文件和日志的脚本。

**使用方法：**
```bash
./scripts/cleanup.sh
```

**功能：**
- 停止所有运行中的 GWatch 进程
- 清理临时日志文件（/tmp/gwatch*.log）
- 清理测试目录（/tmp/gwatch_io_test）
- 可选清理 logs 目录下的所有日志文件

## 注意事项

- 所有脚本都需要在项目根目录运行
- 确保已编译 `bin/Gwatch` 可执行文件
- 确保配置文件已正确配置（特别是 Redis 连接信息）
- 测试脚本会修改配置文件中的 `push_times`，测试完成后建议恢复

