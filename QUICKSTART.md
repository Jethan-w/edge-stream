# EdgeStreamPro 快速开始指南

本指南将帮助您快速上手EdgeStreamPro系统，运行完整的示例程序。

## 🚀 快速开始

### 1. 环境准备

确保您的系统满足以下要求：

- **Go 1.19+** 
- **内存**: 至少 2GB
- **磁盘空间**: 至少 1GB

检查Go环境：
```bash
go version
```

### 2. 克隆项目

```bash
git clone <repository-url>
cd edge-stream
```

### 3. 安装依赖

```bash
go mod download
go mod tidy
```

### 4. 创建必要目录

```bash
make setup
```

或者手动创建：
```bash
mkdir -p config bundles state/local metrics logs output temp input window-state
```

### 5. 运行示例

#### 运行完整示例
```bash
make run
```

#### 运行窗口管理示例
```bash
make run-window
```

#### 运行所有示例
```bash
make run-all
```

## 📋 示例说明

### 完整示例 (`complete_example.go`)

这个示例展示了EdgeStreamPro系统的所有主要功能：

1. **配置管理** - 加载配置、敏感属性加密、配置变更监听
2. **连接器注册** - NAR包管理、组件注册、版本控制
3. **状态管理** - 本地和分布式状态存储
4. **指标收集** - 实时监控、指标聚合、多种格式导出
5. **流引擎** - 多线程处理、背压管理
6. **数据源** - 多协议支持、安全接收
7. **处理器** - 数据转换、错误处理、路由、脚本处理
8. **数据接收器** - 数据库和文件系统适配器
9. **管道** - 组件连接、溯源追踪

### 窗口管理示例 (`window_example.go`)

专门展示窗口管理功能：

1. **时间窗口** - 基于时间的滑动窗口
2. **计数窗口** - 基于记录数量的窗口
3. **会话窗口** - 基于活动时间的窗口
4. **自定义窗口** - 用户定义的窗口条件
5. **聚合计算** - 求和、平均值、计数等
6. **窗口触发器** - 时间、计数、自定义触发器

## 🔧 配置说明

### 主要配置文件

- `config/application.properties` - 主配置文件

### 关键配置项

```properties
# 流引擎配置
stream.engine.max.threads=8
stream.engine.queue.size=1000

# 处理器配置
processor.threads=4
processor.batch.size=100

# 数据库配置
database.url=jdbc:postgresql://localhost:5432/edgestream

# Kafka配置
kafka.brokers=localhost:9092
```

## 📊 监控和指标

### HTTP端点

运行示例后，可以通过以下端点访问指标：

- `http://localhost:8080/metrics` - Prometheus格式
- `http://localhost:8080/metrics/json` - JSON格式
- `http://localhost:8080/health` - 健康检查

### 示例输出

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

## 🛠️ 开发工具

### Makefile 命令

```bash
# 构建
make build          # 构建完整示例
make build-window   # 构建窗口示例
make build-all      # 构建所有示例

# 运行
make run            # 运行完整示例
make run-window     # 运行窗口示例
make run-all        # 运行所有示例

# 开发
make dev            # 开发模式（需要安装air）
make fmt            # 格式化代码
make lint           # 代码检查
make test           # 运行测试

# 其他
make help           # 查看所有命令
make check-env      # 检查环境
make profile        # 性能分析
```

### 推荐的开发工具

1. **Air** - 热重载工具
   ```bash
   go install github.com/cosmtrek/air@latest
   ```

2. **golangci-lint** - 代码检查
   ```bash
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   ```

3. **godoc** - 文档生成
   ```bash
   go install golang.org/x/tools/cmd/godoc@latest
   ```

## 🔍 故障排除

### 常见问题

1. **配置文件找不到**
   ```bash
   # 确保配置文件存在
   ls config/application.properties
   ```

2. **端口被占用**
   ```bash
   # 检查端口使用情况
   lsof -i :8080
   # 或修改配置文件中的端口
   ```

3. **权限问题**
   ```bash
   # 确保有写入权限
   chmod -R 755 config bundles state metrics logs output temp input window-state
   ```

4. **内存不足**
   ```bash
   # 减少配置中的线程数
   # 修改 config/application.properties
   stream.engine.max.threads=4
   processor.threads=2
   ```

### 日志查看

```bash
# 查看应用日志
tail -f logs/edgestream.log

# 查看错误日志
grep ERROR logs/edgestream.log
```

## 📈 性能调优

### 关键参数

- `stream.engine.max.threads` - 最大线程数
- `stream.engine.queue.size` - 队列大小
- `processor.batch.size` - 批处理大小
- `database.pool.size` - 数据库连接池大小

### 监控指标

- **处理延迟** - 数据处理的平均时间
- **吞吐量** - 每秒处理的记录数
- **错误率** - 处理失败的比例
- **资源使用率** - CPU、内存、磁盘使用情况

## 🔐 安全考虑

1. **敏感配置加密**
   - 数据库密码等敏感信息使用加密存储
   - 使用AES加密算法

2. **网络安全**
   - 启用HTTPS进行数据传输
   - 配置防火墙规则

3. **访问控制**
   - 启用认证和授权
   - 定期更新密钥和证书

## 📚 扩展开发

### 添加自定义处理器

```go
type CustomProcessor struct {
    // 实现processor.Processor接口
}

func (cp *CustomProcessor) Process(flowFile *flowfile.FlowFile) error {
    // 自定义处理逻辑
    return nil
}
```

### 添加自定义数据源

```go
type CustomSource struct {
    // 实现source.Source接口
}

func (cs *CustomSource) Start(ctx context.Context) error {
    // 启动数据源
    return nil
}
```

### 添加自定义聚合函数

```go
calculator.RegisterAggregation("custom", func(values []interface{}) interface{} {
    // 自定义聚合逻辑
    return result
})
```

## 🤝 贡献

欢迎提交Issue和Pull Request来改进这个项目。

### 贡献流程

1. Fork 项目
2. 创建功能分支
3. 提交更改
4. 推送到分支
5. 创建 Pull Request

## 📄 许可证

本项目采用MIT许可证。

## 📞 支持

如果您遇到问题或有疑问，请：

1. 查看 [README.md](examples/README.md) 获取详细文档
2. 查看 [故障排除](#故障排除) 部分
3. 提交 [Issue](https://github.com/your-repo/issues)

---

**开始您的EdgeStreamPro之旅吧！** 🎉 