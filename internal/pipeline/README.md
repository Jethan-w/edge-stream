# Pipeline 模块

## 概述

Pipeline 是 EdgeStream 数据处理流程的"全链路载体"，是连接数据源、处理器和接收器的核心模块。它提供了数据流的可视化设计、版本管理、监控和溯源能力，是系统的神经中枢。

## 核心特性

### 1. 可视化流程设计
- 支持拖拽式的流程设计界面
- 灵活的组件组合和配置
- 直观的流程可视化

### 2. 数据流转管控
- FlowFile 抽象，携带数据内容和元数据
- Connection 队列管理，支持多种优先级策略
- ProcessGroup 支持嵌套和模块化管理

### 3. 版本管理
- 流程配置的版本控制和追溯
- 支持版本比较和回滚
- NiFi Registry 集成

### 4. 全链路监控与溯源
- 数据处理全流程的监控和追踪
- 数据血缘关系追踪
- 事件记录和查询

## 模块架构

```
internal/pipeline/
├── pipeline.go              # 核心接口和数据结构
├── Designer/                # 流程设计器
│   └── pipeline_designer.go
├── VersionManager/          # 版本管理
│   └── version_manager.go
├── Provenance/             # 数据溯源
│   └── provenance_tracker.go
├── example/                # 使用示例
│   └── example.go
└── README.md              # 本文档
```

## 核心组件

### 1. Pipeline 接口

```go
type Pipeline interface {
    Initialize(ctx context.Context) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Pause(ctx context.Context) error
    Resume(ctx context.Context) error
    GetState() PipelineState
    GetConfiguration() *PipelineConfiguration
    Validate() *ValidationResult
    GetStatistics() *PipelineStatistics
}
```

### 2. FlowFile

数据流转的基本单元，携带数据内容和元数据：

```go
type FlowFile struct {
    UUID       string
    Attributes map[string]string
    Content    []byte
    Size       int64
    Timestamp  time.Time
    LineageID  string
}
```

### 3. ProcessGroup

支持流程组的嵌套和模块化管理：

```go
type ProcessGroup struct {
    ID           string
    Name         string
    Processors   map[string]*ProcessorNode
    Connections  map[string]*Connection
    ChildGroups  map[string]*ProcessGroup
    ParentGroup  *ProcessGroup
    Configuration *ProcessGroupConfiguration
}
```

### 4. Connection

数据传输通道，支持队列管理和优先级策略：

```go
type Connection struct {
    ID              string
    Name            string
    Source          *ProcessorNode
    Destination     *ProcessorNode
    Queue           *ConnectionQueue
    Prioritizer     QueuePrioritizer
    ParentGroup     *ProcessGroup
    Configuration   *ConnectionConfiguration
}
```

## 使用示例

### 1. 基本流程设计

```go
// 创建流程设计器
designer := designer.NewPipelineDesigner()
designer.Initialize(ctx)

// 创建流程组
processGroup := designer.CreateProcessGroup("日志处理流程")

// 添加处理器
processor1 := designer.NewProcessorDTO("processor-1", "GetFile", "GetFile")
processor1.SetProperty("Input Directory", "/var/log")
designer.AddProcessor(processGroup, processor1)

// 创建连接
connection1 := designer.NewConnectionDTO("connection-1", "GetFile to SplitText", "processor-1", "processor-2")
designer.CreateConnection(processGroup, connection1)
```

### 2. 版本管理

```go
// 创建版本管理器
versionManager := versionmanager.NewVersionManager()
versionManager.Initialize(ctx)

// 保存版本
versionManager.SaveVersion(processGroup, "初始版本")

// 列出版本
versions, _ := versionManager.ListVersions()
for _, version := range versions {
    fmt.Printf("版本: %s, 注释: %s\n", version.Version, version.Comment)
}

// 比较版本
comparison, _ := versionManager.CompareVersions(versionID1, versionID2)
```

### 3. 数据溯源

```go
// 创建溯源跟踪器
provenanceTracker := provenance.NewProvenanceTracker()
provenanceTracker.Initialize(ctx)

// 记录事件
flowFile := pipeline.NewFlowFile()
details := map[string]string{"source": "file_system"}
provenanceTracker.RecordEvent(flowFile, provenance.EventTypeCreate, "processor-id", details)

// 追踪血缘关系
lineage, _ := provenanceTracker.TraceFlowFile(flowFile.UUID)
```

### 4. FlowFile 处理

```go
// 创建 FlowFile
flowFile := pipeline.NewFlowFile()
flowFile.SetContent([]byte("test data"))
flowFile.AddAttribute("filename", "test.log")

// 处理 FlowFile
processor := pipeline.NewProcessorNode("proc-1", "TestProcessor", "LogProcessor", nil)
processor.Process(ctx, flowFile)
```

## 优先级策略

支持多种队列优先级策略：

### 1. 先进先出 (OldestFirst)
```go
prioritizer := pipeline.NewOldestFirstPrioritizer()
```

### 2. 后进先出 (NewestFirst)
```go
prioritizer := pipeline.NewNewestFirstPrioritizer()
```

## 状态管理

Pipeline 支持完整的状态机：

- `StateInitialized`: 已初始化
- `StateDeploying`: 部署中
- `StateRunning`: 运行中
- `StatePausing`: 暂停中
- `StatePaused`: 已暂停
- `StateResuming`: 恢复中
- `StateStopping`: 停止中
- `StateStopped`: 已停止

## 性能指标

| 指标           | 目标值        | 说明                   |
|---------------|---------------|------------------------|
| 流程设计响应时间 | <100ms        | UI操作响应速度         |
| 数据路由延迟    | <10ms         | FlowFile 传输耗时      |
| 版本控制开销    | <50ms         | 版本保存和回滚时间     |
| 溯源查询性能    | <200ms        | 追踪数据处理路径       |

## 扩展性

### 1. 自定义处理器

```go
type CustomProcessor struct{}

func (cp *CustomProcessor) Process(ctx context.Context, data interface{}) (interface{}, error) {
    // 实现自定义处理逻辑
    return data, nil
}

// 注册处理器
processorFactory.RegisterProcessor("CustomProcessor", func() processor.Processor {
    return &CustomProcessor{}
})
```

### 2. 自定义优先级策略

```go
type CustomPrioritizer struct{}

func (cp *CustomPrioritizer) SelectNext(queue []*pipeline.FlowFile) *pipeline.FlowFile {
    // 实现自定义优先级逻辑
    return queue[0]
}
```

## 最佳实践

### 1. 流程设计
- 使用有意义的处理器和连接名称
- 合理配置处理器属性
- 选择合适的优先级策略

### 2. 版本管理
- 定期保存重要版本
- 使用描述性的版本注释
- 在修改前备份当前版本

### 3. 数据溯源
- 及时记录重要事件
- 合理设置事件详情
- 定期清理过期事件

### 4. 性能优化
- 合理设置队列大小
- 避免过深的流程嵌套
- 监控关键性能指标

## 故障排除

### 常见问题

1. **处理器启动失败**
   - 检查处理器配置
   - 验证依赖资源
   - 查看错误日志

2. **连接队列满**
   - 增加队列大小
   - 检查下游处理器状态
   - 优化处理性能

3. **版本保存失败**
   - 检查存储权限
   - 验证序列化格式
   - 确认网络连接

### 调试技巧

1. 启用详细日志
2. 使用性能监控
3. 检查状态信息
4. 验证配置参数

## 相关文档

- [Apache NiFi Pipeline 深度技术分析](../doc/apache_nifi/Pipeline_深度技术分析.md)
- [Processor 模块](../processor/README.md)
- [Source 模块](../source/README.md)
- [Sink 模块](../sink/README.md) 