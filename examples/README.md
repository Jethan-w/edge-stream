# EdgeStreamPro 完整示例

这个示例展示了如何使用EdgeStreamPro系统的所有主要组件，包括配置管理、连接器注册、流引擎、处理器、数据源、数据接收器等。

## 功能特性

本示例演示了以下功能：

### 1. 配置管理 (ConfigManager)
- 加载和解析配置文件
- 敏感属性加密
- 配置变更监听
- 配置验证
- 环境变量注入

### 2. 连接器注册表 (ConnectorRegistry)
- NAR包管理
- 组件注册和发现
- 版本控制
- 安全验证
- 扩展映射

### 3. 状态管理 (StateManager)
- 本地状态存储
- 分布式状态同步
- 状态恢复
- 状态备份

### 4. 指标收集 (MetricCollector)
- 实时指标收集
- 指标聚合
- 指标导出 (Prometheus, JMX, CSV)
- 溯源追踪
- 历史指标存储

### 5. 流引擎 (StreamEngine)
- 多线程处理
- 背压管理
- 负载均衡
- 性能监控

### 6. 数据源 (Source)
- 多协议支持 (HTTP, TCP, UDP)
- 安全接收
- 配置管理

### 7. 处理器 (Processor)
- 数据转换
- 错误处理
- 路由处理
- 脚本处理
- 语义提取

### 8. 数据接收器 (Sink)
- 数据库适配器
- 文件系统适配器
- 重试机制
- 安全传输

### 9. 管道 (Pipeline)
- 管道设计
- 组件连接
- 溯源追踪
- 版本管理

## 目录结构

```
examples/
├── complete_example.go    # 完整示例代码
├── README.md             # 本文件
└── config/
    └── application.properties  # 配置文件
```

## 运行要求

### 系统要求
- Go 1.19+
- 至少 2GB 内存
- 至少 1GB 磁盘空间

### 依赖服务 (可选)
- PostgreSQL 数据库
- Kafka 消息队列
- Prometheus 监控系统

## 运行步骤

### 1. 准备环境

```bash
# 创建必要的目录
mkdir -p config bundles state/local metrics logs output temp input

# 复制配置文件
cp config/application.properties ./config/
```

### 2. 编译和运行

```bash
# 编译示例
go build -o edgestream-example examples/complete_example.go

# 运行示例
./edgestream-example
```

### 3. 查看输出

示例运行时会输出以下信息：

```
=== EdgeStreamPro 完整示例 ===
1. 初始化配置管理器...
2. 初始化连接器注册表...
3. 初始化状态管理器...
4. 初始化指标收集器...
5. 初始化流引擎...
6. 创建数据源...
7. 创建处理器...
8. 创建数据接收器...
9. 创建管道...
10. 启动流处理...
11. 监控和导出指标...
=== 示例完成 ===
```

## 配置说明

### 主要配置项

- **流引擎配置**: 控制线程数、队列大小、背压限制等
- **处理器配置**: 设置批处理大小、超时时间、重试次数等
- **数据库配置**: 数据库连接信息
- **Kafka配置**: 消息队列配置
- **安全配置**: 加密、认证设置
- **指标配置**: 监控和导出设置

### 环境变量

可以通过环境变量覆盖配置：

```bash
export EDGESTREAM_PROCESSOR_THREADS=8
export EDGESTREAM_DATABASE_URL=jdbc:postgresql://localhost:5432/mydb
```

## 监控和指标

### HTTP端点

示例启动后，可以通过以下端点访问指标：

- `http://localhost:8080/metrics` - Prometheus格式指标
- `http://localhost:8080/metrics/json` - JSON格式指标
- `http://localhost:8080/health` - 健康检查

### 指标类型

- **Gauge**: 当前值指标 (如活跃线程数)
- **Counter**: 累计计数指标 (如处理记录数)
- **Timer**: 时间指标 (如处理时间)
- **Histogram**: 分布指标 (如延迟分布)

## 扩展和定制

### 添加新的处理器

```go
type CustomProcessor struct {
    // 实现processor.Processor接口
}

func (cp *CustomProcessor) Process(flowFile *flowfile.FlowFile) error {
    // 自定义处理逻辑
    return nil
}
```

### 添加新的数据源

```go
type CustomSource struct {
    // 实现source.Source接口
}

func (cs *CustomSource) Start(ctx context.Context) error {
    // 启动数据源
    return nil
}
```

### 添加新的数据接收器

```go
type CustomSink struct {
    // 实现sink.Sink接口
}

func (cs *CustomSink) Send(data []byte) error {
    // 发送数据到目标
    return nil
}
```

## 故障排除

### 常见问题

1. **配置文件找不到**
   - 确保 `config/application.properties` 文件存在
   - 检查文件路径是否正确

2. **端口被占用**
   - 修改配置文件中的端口设置
   - 或者停止占用端口的其他服务

3. **权限问题**
   - 确保有写入日志、状态、指标目录的权限
   - 在Linux/Mac上可能需要使用sudo

4. **内存不足**
   - 减少配置中的线程数和队列大小
   - 增加系统内存

### 日志查看

```bash
# 查看应用日志
tail -f logs/edgestream.log

# 查看错误日志
grep ERROR logs/edgestream.log
```

## 性能调优

### 关键参数

- `stream.engine.max.threads`: 最大线程数
- `stream.engine.queue.size`: 队列大小
- `processor.batch.size`: 批处理大小
- `database.pool.size`: 数据库连接池大小

### 监控指标

- 处理延迟 (Processing Latency)
- 吞吐量 (Throughput)
- 错误率 (Error Rate)
- 资源使用率 (CPU, Memory, Disk)

## 安全考虑

- 敏感配置使用加密存储
- 启用认证和授权
- 使用HTTPS进行网络传输
- 定期更新密钥和证书

## 贡献

欢迎提交Issue和Pull Request来改进这个示例。

## 许可证

本项目采用MIT许可证。 