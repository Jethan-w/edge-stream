# EdgeStream 综合示例

这个示例展示了 EdgeStream 框架中所有核心组件如何协同工作，实现从数据源到数据接收器的完整数据处理流程。

## 组件概览

本示例包含以下核心组件：

### 1. 配置管理 (ConfigManager)
- 统一配置管理
- 配置热重载
- 配置验证
- 敏感属性加密

### 2. 连接器注册表 (ConnectorRegistry)
- 动态组件注册
- Bundle 管理
- 版本控制
- 安全验证

### 3. 数据流组件
- **Source**: 数据源，支持文件、Kafka等
- **Pipeline**: 数据处理管道
- **Processor**: 数据处理器
- **Sink**: 数据接收器

### 4. 核心服务
- **FlowFile**: 数据载体
- **MetricCollector**: 指标收集
- **StateManager**: 状态管理
- **StreamEngine**: 流处理引擎
- **WindowManager**: 窗口管理

## 数据流架构

```
数据源 (Source) 
    ↓
FlowFile 创建
    ↓
数据处理管道 (Pipeline)
    ↓
数据处理器 (Processor)
    ↓
窗口聚合 (WindowManager)
    ↓
数据接收器 (Sink)
    ↓
指标收集 (MetricCollector)
```

## 运行示例

### 1. 环境准备

确保您的 Go 环境已正确配置：

```bash
# 检查 Go 版本
go version

# 设置模块
go mod init edge-stream
go mod tidy
```

### 2. 创建必要目录

```bash
# 创建数据目录
mkdir -p data/input data/output

# 创建配置目录
mkdir -p config

# 创建日志目录
mkdir -p logs

# 创建状态目录
mkdir -p state

# 创建Bundle目录
mkdir -p bundles
```

### 3. 配置文件

确保 `config/edge-stream.properties` 文件存在并包含正确的配置。

### 4. 运行示例

```bash
# 运行综合示例
go run examples/comprehensive_example.go
```

## 示例功能

### 1. 数据源处理
- 从文件系统读取日志文件
- 支持 JSON 格式的日志数据
- 自动创建示例数据

### 2. 数据处理管道
- **数据转换**: JSON 到 CSV 格式转换
- **语义提取**: 提取时间戳、日志级别、消息内容
- **窗口聚合**: 基于时间窗口的数据聚合

### 3. 数据接收
- 写入文件系统
- 支持 CSV 格式输出
- 错误处理和重试机制

### 4. 监控和指标
- 实时指标收集
- 性能监控
- 状态报告

## 输出结果

运行示例后，您将看到：

1. **控制台输出**: 详细的处理日志和状态信息
2. **数据文件**: `data/output/` 目录下的处理结果文件
3. **监控报告**: 包含指标快照、状态信息和窗口统计

## 配置说明

### 数据源配置
```properties
source.type=file                    # 数据源类型：file, kafka
source.path=./data/input           # 数据源路径
source.file.pattern=*.log          # 文件匹配模式
```

### 数据接收器配置
```properties
sink.type=file                     # 接收器类型：file, database
sink.path=./data/output           # 输出路径
sink.retry.max=3                  # 最大重试次数
```

### 流引擎配置
```properties
engine.max.threads=10             # 最大线程数
engine.queue.capacity=1000        # 队列容量
engine.backpressure.threshold=0.8 # 背压阈值
```

## 扩展功能

### 1. 添加新的数据源
```go
// 创建自定义协议适配器
customAdapter := source.NewCustomProtocolAdapter()
dataSource.SetProtocolAdapter(customAdapter)
```

### 2. 添加新的处理器
```go
// 注册自定义处理器
designer.ProcessorFactory.RegisterProcessor("CustomProcessor", func() processor.Processor {
    return &CustomProcessor{}
})
```

### 3. 添加新的数据接收器
```go
// 创建自定义目标适配器
customAdapter := sink.NewCustomTargetAdapter()
dataSink.SetTargetAdapter(customAdapter)
```

### 4. 配置集群模式
```properties
cluster.enabled=true
cluster.zookeeper.connect=localhost:2181
cluster.node.id=node-1
```

## 故障排除

### 常见问题

1. **配置文件不存在**
   - 确保 `config/edge-stream.properties` 文件存在
   - 检查文件权限

2. **目录创建失败**
   - 确保有足够的磁盘空间
   - 检查目录权限

3. **组件初始化失败**
   - 检查依赖项是否正确安装
   - 查看错误日志获取详细信息

### 调试模式

启用详细日志：
```properties
logging.level=DEBUG
```

## 性能优化

### 1. 调整线程池
```properties
engine.max.threads=20
engine.core.threads=10
```

### 2. 优化批处理大小
```properties
processor.batch.size=500
source.batch.size=200
```

### 3. 调整窗口参数
```properties
window.size=300
window.slide=60
```

## 监控和告警

### 1. Prometheus 指标
启用 Prometheus 导出：
```properties
metrics.prometheus.enabled=true
metrics.prometheus.port=9090
```

### 2. 健康检查
访问 `http://localhost:9090/health` 检查系统状态

### 3. 指标查询
访问 `http://localhost:9090/metrics` 查看详细指标

## 安全考虑

### 1. 启用安全功能
```properties
security.enabled=true
security.ssl.enabled=true
security.authentication.enabled=true
```

### 2. 敏感属性加密
```properties
config.encryption.enabled=true
```

## 贡献指南

如果您想为这个示例做出贡献：

1. Fork 项目
2. 创建功能分支
3. 提交更改
4. 创建 Pull Request

## 许可证

本项目采用 Apache License 2.0 许可证。 