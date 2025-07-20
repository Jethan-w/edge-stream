# MetricCollector 模块

## 模块概述

MetricCollector 是 Apache NiFi 系统的"监控中枢"，负责采集、聚合、存储和展示系统运行时的关键指标数据。它是整个数据流处理系统中提供实时性能洞察和系统健康状况监控的关键模块，提供了全面、灵活和可扩展的指标收集能力。

## 核心功能

### 1. 实时指标收集
提供对系统运行时各个层面的实时指标采集机制。

#### 指标类型
- **Gauge（瞬时值）**: 表示某个时刻的瞬时值，如内存使用量、CPU利用率
- **Counter（累计值）**: 表示累计计数值，如处理记录数、错误数
- **Histogram（数值分布）**: 表示数值的分布情况，如处理时间分布
- **Timer（耗时统计）**: 表示操作耗时统计，如请求响应时间
- **Meter（速率统计）**: 表示事件发生速率，如每秒请求数

#### 指标作用域
- **SYSTEM**: 系统级指标
- **CLUSTER**: 集群级指标
- **FLOW**: 流程级指标
- **PROCESSOR**: 处理器级指标
- **CONNECTION**: 连接级指标

### 2. 多维数据聚合
提供对不同维度和层次的指标聚合能力。

#### 聚合策略
- **按组件类型聚合**: 将相同类型的处理器指标进行聚合
- **按流程组聚合**: 将流程组内的所有处理器指标进行聚合
- **按时间窗口聚合**: 按时间窗口对指标进行聚合分析
- **集群环境聚合**: 在集群环境下聚合各节点的指标

### 3. 自定义指标扩展
提供灵活的指标注册和自定义指标开发机制。

#### 扩展机制
- **自定义指标提供者**: 实现自定义指标收集逻辑
- **指标注册管理**: 统一管理所有自定义指标
- **指标描述管理**: 提供指标的详细描述信息

### 4. 指标持久化存储
提供指标数据的持久化和长期存储机制。

#### 存储特性
- **时序数据库支持**: 基于文件系统的时序数据存储
- **指标归档策略**: 按不同精度进行指标归档
- **历史数据查询**: 支持历史指标数据的查询和分析
- **指标摘要统计**: 提供指标数据的摘要统计功能

### 5. 指标导出与可视化
提供多种指标导出和可视化的机制。

#### 导出格式
- **Prometheus**: 标准的Prometheus指标格式
- **OpenTSDB**: OpenTSDB时序数据库格式
- **Graphite**: Graphite监控系统格式
- **JMX**: Java管理扩展格式
- **CSV**: 逗号分隔值格式

#### HTTP端点
- **/metrics**: Prometheus格式指标端点
- **/metrics/json**: JSON格式指标端点
- **/health**: 健康检查端点

### 6. 数据溯源服务
提供完整的数据处理流程追踪能力。

#### 溯源功能
- **FlowFile追踪**: 追踪单个FlowFile的完整处理路径
- **事件记录**: 记录数据处理过程中的关键事件
- **处理路径分析**: 分析数据在处理器间的流转路径
- **溯源摘要统计**: 提供溯源数据的摘要统计

## 架构设计

### 目录结构
```
internal/MetricCollector/
├── metric_collector.go          # 主接口和实现
├── MetricAggregator/
│   └── metric_aggregator.go     # 指标聚合
├── MetricRecorder/
│   └── metric_recorder.go       # 指标存储
├── MetricExporter/
│   └── metric_exporter.go       # 指标导出
├── LineageTracker/
│   └── lineage_tracker.go       # 数据溯源
├── examples.go                  # 使用示例
└── README.md                    # 文档
```

### 核心接口

#### MetricCollector
```go
type MetricCollector interface {
    RegisterGauge(name string, supplier func() float64) (Gauge, error)
    RegisterCounter(name string) (Counter, error)
    RegisterHistogram(name string) (Histogram, error)
    RegisterTimer(name string) (Timer, error)
    RegisterMeter(name string) (Meter, error)
    GetMetricsSnapshot() *MetricsSnapshot
    GetMetric(name string) (Metric, error)
    RemoveMetric(name string) error
    ListMetrics() []string
}
```

#### MetricsAggregator
```go
type MetricsAggregator interface {
    AggregateByProcessorType() (map[string]*AggregatedMetrics, error)
    AggregateByProcessGroup() (map[string]*AggregatedMetrics, error)
    AggregateByTimeWindow(window time.Duration) ([]*TimeWindowMetrics, error)
    AggregateByCluster() (*ClusterMetricsSnapshot, error)
    GetAggregatedMetrics(aggregationType string) (interface{}, error)
}
```

#### MetricsRepository
```go
type MetricsRepository interface {
    StoreSnapshot(snapshot *MetricsSnapshot) error
    QueryHistoricalMetrics(metricName string, startTime, endTime time.Time) ([]*MetricsSnapshot, error)
    PurgeExpiredMetrics(retentionPeriod time.Duration) error
    GetMetricsByTimeRange(startTime, endTime time.Time) ([]*MetricsSnapshot, error)
    GetMetricsByType(metricType string) ([]*MetricsSnapshot, error)
}
```

#### MetricsExporter
```go
type MetricsExporter interface {
    ExportToPrometheus(snapshot *MetricsSnapshot) (string, error)
    ExportToOpenTSDB(snapshot *MetricsSnapshot) ([]byte, error)
    ExportToGraphite(snapshot *MetricsSnapshot) ([]byte, error)
    ExportToJMX(snapshot *MetricsSnapshot) ([]byte, error)
    ExportToCSV(snapshot *MetricsSnapshot) ([]byte, error)
    StartHTTPEndpoint(port int) error
    StopHTTPEndpoint() error
}
```

#### LineageService
```go
type LineageService interface {
    TraceFlowFile(flowFileID string) (*LineageResult, error)
    AddLineageEvent(event *LineageEvent) error
    GetLineageEvents(flowFileID string) ([]*LineageEvent, error)
    GetLineageEventsByTimeRange(startTime, endTime time.Time) ([]*LineageEvent, error)
    GetLineageEventsByProcessor(processorID string) ([]*LineageEvent, error)
    GetLineageSummary(flowFileID string) (*LineageSummary, error)
}
```

## 使用方法

### 基本使用

```go
package main

import (
    "fmt"
    "log"
    "time"
    "github.com/crazy/edge-stream/internal/MetricCollector"
)

func main() {
    // 创建指标收集器
    collector := MetricCollector.NewStandardMetricCollector()
    
    // 注册Gauge指标（瞬时值）
    gauge, err := collector.RegisterGauge("jvm.memory.used", func() float64 {
        // 模拟JVM内存使用量
        return 1024.0 + rand.Float64()*100
    })
    if err != nil {
        log.Fatalf("注册Gauge失败: %v", err)
    }
    
    // 注册Counter指标（累计值）
    counter, err := collector.RegisterCounter("processed.records")
    if err != nil {
        log.Fatalf("注册Counter失败: %v", err)
    }
    
    // 注册Timer指标（耗时统计）
    timer, err := collector.RegisterTimer("request.duration")
    if err != nil {
        log.Fatalf("注册Timer失败: %v", err)
    }
    
    // 模拟数据更新
    for i := 0; i < 5; i++ {
        // 更新Counter
        counter.Increment(10)
        
        // 使用Timer
        ctx := timer.Start()
        time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
        ctx.Stop()
        
        time.Sleep(100 * time.Millisecond)
    }
    
    // 获取指标快照
    snapshot := collector.GetMetricsSnapshot()
    fmt.Printf("指标快照时间: %s\n", snapshot.Timestamp.Format(time.RFC3339))
    fmt.Printf("Gauge指标数量: %d\n", len(snapshot.Gauges))
    fmt.Printf("Counter指标数量: %d\n", len(snapshot.Counters))
    fmt.Printf("Timer指标数量: %d\n", len(snapshot.Timers))
}
```

### 指标聚合

```go
// 创建指标聚合器
aggregator := MetricCollector.NewStandardMetricsAggregator(
    collector,
    processorRegistry,
    processGroupRegistry,
    clusterManager,
)

// 按处理器类型聚合
processorTypeAggregations, err := aggregator.AggregateByProcessorType()
if err != nil {
    log.Fatalf("按处理器类型聚合失败: %v", err)
}

fmt.Println("按处理器类型聚合结果:")
for processorType, metrics := range processorTypeAggregations {
    fmt.Printf("  处理器类型: %s\n", processorType)
    fmt.Printf("    总处理时间: %.2f\n", metrics.TotalProcessingTime)
    fmt.Printf("    总处理记录数: %d\n", metrics.TotalProcessedRecords)
    fmt.Printf("    总错误数: %d\n", metrics.TotalErrorCount)
    fmt.Printf("    平均处理时间: %.2f\n", metrics.AverageProcessingTime)
    fmt.Printf("    吞吐量: %.2f\n", metrics.Throughput)
}

// 按流程组聚合
processGroupAggregations, err := aggregator.AggregateByProcessGroup()
if err != nil {
    log.Fatalf("按流程组聚合失败: %v", err)
}

fmt.Println("\n按流程组聚合结果:")
for groupName, metrics := range processGroupAggregations {
    fmt.Printf("  流程组: %s\n", groupName)
    fmt.Printf("    总处理时间: %.2f\n", metrics.TotalProcessingTime)
    fmt.Printf("    总处理记录数: %d\n", metrics.TotalProcessedRecords)
    fmt.Printf("    总错误数: %d\n", metrics.TotalErrorCount)
}
```

### 指标存储

```go
// 创建指标存储库
repository := MetricCollector.NewTimeSeriesMetricsRepository("./metrics-storage")

// 创建指标存储管理器
storageManager := MetricCollector.NewMetricsStorageManager(repository)

// 创建指标快照
snapshot := &MetricCollector.MetricRecorder.MetricsSnapshot{
    Timestamp: time.Now(),
    Gauges: map[string]float64{
        "jvm.memory.used": 1024.0,
        "jvm.cpu.usage":   0.5,
    },
    Counters: map[string]int64{
        "processed.records": 1000,
        "error.count":       5,
    },
}

// 存储指标快照
err := storageManager.StoreMetrics(snapshot)
if err != nil {
    log.Fatalf("存储指标失败: %v", err)
}
fmt.Println("指标快照已存储")

// 查询历史指标
startTime := time.Now().Add(-1 * time.Hour)
endTime := time.Now()

historicalMetrics, err := storageManager.QueryMetrics("jvm.memory.used", startTime, endTime)
if err != nil {
    log.Fatalf("查询历史指标失败: %v", err)
}

fmt.Printf("查询到 %d 个历史指标快照\n", len(historicalMetrics))

// 获取指标摘要
summary, err := storageManager.GetMetricsSummary(startTime, endTime)
if err != nil {
    log.Fatalf("获取指标摘要失败: %v", err)
}

fmt.Printf("指标摘要:\n")
fmt.Printf("  总快照数: %d\n", summary.TotalSnapshots)
fmt.Printf("  时间范围: %s 到 %s\n", 
    summary.TimeRange.StartTime.Format(time.RFC3339),
    summary.TimeRange.EndTime.Format(time.RFC3339))
fmt.Printf("  持续时间: %v\n", summary.TimeRange.EndTime.Sub(summary.TimeRange.StartTime))
```

### 指标导出

```go
// 创建指标导出器
exporter := MetricCollector.NewStandardMetricsExporter()

// 创建指标快照
snapshot := &MetricCollector.MetricExporter.MetricsSnapshot{
    Timestamp: time.Now(),
    Gauges: map[string]float64{
        "jvm.memory.used": 1024.0,
        "jvm.cpu.usage":   0.5,
    },
    Counters: map[string]int64{
        "processed.records": 1000,
        "error.count":       5,
    },
}

// 导出到Prometheus格式
prometheusData, err := exporter.ExportToPrometheus(snapshot)
if err != nil {
    log.Fatalf("导出到Prometheus失败: %v", err)
}
fmt.Println("Prometheus格式导出:")
fmt.Println(prometheusData[:200] + "...")

// 导出到OpenTSDB格式
opentsdbData, err := exporter.ExportToOpenTSDB(snapshot)
if err != nil {
    log.Fatalf("导出到OpenTSDB失败: %v", err)
}
fmt.Printf("\nOpenTSDB格式导出 (前200字符):\n%s...\n", string(opentsdbData[:200]))

// 导出到Graphite格式
graphiteData, err := exporter.ExportToGraphite(snapshot)
if err != nil {
    log.Fatalf("导出到Graphite失败: %v", err)
}
fmt.Printf("\nGraphite格式导出 (前200字符):\n%s...\n", string(graphiteData[:200]))

// 启动HTTP端点
err = exporter.StartHTTPEndpoint(8080)
if err != nil {
    log.Fatalf("启动HTTP端点失败: %v", err)
}
fmt.Println("\nHTTP指标端点已启动在端口 8080")
fmt.Println("访问 http://localhost:8080/metrics 获取Prometheus格式指标")
fmt.Println("访问 http://localhost:8080/metrics/json 获取JSON格式指标")
fmt.Println("访问 http://localhost:8080/health 获取健康状态")

// 等待一段时间后停止
time.Sleep(2 * time.Second)

err = exporter.StopHTTPEndpoint()
if err != nil {
    log.Printf("停止HTTP端点失败: %v", err)
}
```

### 数据溯源

```go
// 创建溯源存储库
repository := MetricCollector.NewInMemoryProvenanceRepository()

// 创建溯源管理器
lineageManager := MetricCollector.NewLineageManager(repository)

// 创建溯源事件
flowFileID := "flowfile-12345"

// 创建事件
createEvent := MetricCollector.NewLineageEvent(
    flowFileID,
    MetricCollector.LineageEventTypeCreate,
    "processor-1",
    "GenerateFlowFile",
)
createEvent.Size = 1024
createEvent.LineageStart = true
createEvent.Attributes["filename"] = "data.txt"
createEvent.Attributes["size"] = "1024"

// 接收事件
receiveEvent := MetricCollector.NewLineageEvent(
    flowFileID,
    MetricCollector.LineageEventTypeReceive,
    "processor-2",
    "LogAttribute",
)
receiveEvent.Size = 1024
receiveEvent.Details["source"] = "processor-1"

// 修改事件
modifyEvent := MetricCollector.NewLineageEvent(
    flowFileID,
    MetricCollector.LineageEventTypeModify,
    "processor-3",
    "UpdateAttribute",
)
modifyEvent.Size = 1024
modifyEvent.Attributes["processed"] = "true"
modifyEvent.Attributes["timestamp"] = time.Now().Format(time.RFC3339)

// 发送事件
sendEvent := MetricCollector.NewLineageEvent(
    flowFileID,
    MetricCollector.LineageEventTypeSend,
    "processor-4",
    "PutFile",
)
sendEvent.Size = 1024
sendEvent.LineageEnd = true
sendEvent.Details["destination"] = "/output/data.txt"

// 添加事件到溯源系统
events := []*MetricCollector.LineageEvent{createEvent, receiveEvent, modifyEvent, sendEvent}
for _, event := range events {
    err := lineageManager.AddEvent(event)
    if err != nil {
        log.Fatalf("添加溯源事件失败: %v", err)
    }
}

fmt.Printf("已添加 %d 个溯源事件\n", len(events))

// 追踪FlowFile
result, err := lineageManager.TraceFlowFile(flowFileID)
if err != nil {
    log.Fatalf("追踪FlowFile失败: %v", err)
}

fmt.Printf("\nFlowFile追踪结果:\n")
fmt.Printf("  FlowFile ID: %s\n", result.FlowFileID)
fmt.Printf("  总事件数: %d\n", result.TotalEvents)
fmt.Printf("  开始时间: %s\n", result.StartTime.Format(time.RFC3339))
fmt.Printf("  结束时间: %s\n", result.EndTime.Format(time.RFC3339))
fmt.Printf("  持续时间: %v\n", result.Duration)
fmt.Printf("  总大小: %d\n", result.TotalSize)
fmt.Printf("  最终大小: %d\n", result.FinalSize)

fmt.Printf("\n处理路径:\n")
for i, processor := range result.ProcessingPath {
    fmt.Printf("  %d. %s (%s)\n", i+1, processor.Name, processor.ID)
    fmt.Printf("     输入: %d, 输出: %d, 错误: %d\n", 
        processor.InputCount, processor.OutputCount, processor.ErrorCount)
}

// 获取溯源摘要
summary, err := lineageManager.GetSummary(flowFileID)
if err != nil {
    log.Fatalf("获取溯源摘要失败: %v", err)
}

fmt.Printf("\n溯源摘要:\n")
fmt.Printf("  总事件数: %d\n", summary.TotalEvents)
fmt.Printf("  唯一处理器数: %d\n", summary.UniqueProcessors)
fmt.Printf("  持续时间: %v\n", summary.Duration)

fmt.Printf("\n事件类型统计:\n")
for eventType, count := range summary.EventTypes {
    fmt.Printf("  %s: %d\n", eventType, count)
}

fmt.Printf("\n处理器统计:\n")
for processorID, processor := range summary.Processors {
    fmt.Printf("  %s (%s):\n", processor.Name, processorID)
    fmt.Printf("    事件数: %d\n", processor.EventCount)
    fmt.Printf("    输入: %d, 输出: %d, 错误: %d\n", 
        processor.InputCount, processor.OutputCount, processor.ErrorCount)
    fmt.Printf("    首次出现: %s\n", processor.FirstSeen.Format(time.RFC3339))
    fmt.Printf("    最后出现: %s\n", processor.LastSeen.Format(time.RFC3339))
}
```

### 自定义指标提供者

```go
// 创建自定义指标提供者
provider := &DatabaseConnectionPoolMetricProvider{
    activeConnections:   10,
    waitingConnections:  2,
    totalConnections:    20,
    idleConnections:     8,
}

// 注册自定义指标
provider.RegisterMetrics(collector)

// 模拟数据库连接池状态变化
for i := 0; i < 3; i++ {
    // 模拟连接使用
    provider.activeConnections = 10 + i*2
    provider.waitingConnections = 2 + i
    provider.idleConnections = 8 - i
    
    time.Sleep(100 * time.Millisecond)
    
    // 获取指标快照
    snapshot := collector.GetMetricsSnapshot()
    
    fmt.Printf("数据库连接池状态 %d:\n", i+1)
    if gauge, exists := snapshot.Gauges["database.connection.active"]; exists {
        fmt.Printf("  活跃连接数: %.0f\n", gauge.GetValue())
    }
    if gauge, exists := snapshot.Gauges["database.connection.waiting"]; exists {
        fmt.Printf("  等待连接数: %.0f\n", gauge.GetValue())
    }
    if gauge, exists := snapshot.Gauges["database.connection.idle"]; exists {
        fmt.Printf("  空闲连接数: %.0f\n", gauge.GetValue())
    }
    if gauge, exists := snapshot.Gauges["database.connection.total"]; exists {
        fmt.Printf("  总连接数: %.0f\n", gauge.GetValue())
    }
}

// DatabaseConnectionPoolMetricProvider 数据库连接池指标提供者
type DatabaseConnectionPoolMetricProvider struct {
    activeConnections   int
    waitingConnections  int
    totalConnections    int
    idleConnections     int
}

// RegisterMetrics 注册指标
func (dcpmp *DatabaseConnectionPoolMetricProvider) RegisterMetrics(collector MetricCollector) {
    // 注册活跃连接数指标
    collector.RegisterGauge("database.connection.active", func() float64 {
        return float64(dcpmp.activeConnections)
    })
    
    // 注册等待连接数指标
    collector.RegisterGauge("database.connection.waiting", func() float64 {
        return float64(dcpmp.waitingConnections)
    })
    
    // 注册空闲连接数指标
    collector.RegisterGauge("database.connection.idle", func() float64 {
        return float64(dcpmp.idleConnections)
    })
    
    // 注册总连接数指标
    collector.RegisterGauge("database.connection.total", func() float64 {
        return float64(dcpmp.totalConnections)
    })
}

// GetMetricDescriptions 获取指标描述
func (dcpmp *DatabaseConnectionPoolMetricProvider) GetMetricDescriptions() map[string]string {
    return map[string]string{
        "database.connection.active": "当前活跃的数据库连接数",
        "database.connection.waiting": "等待获取连接的线程数",
        "database.connection.idle": "当前空闲的数据库连接数",
        "database.connection.total": "数据库连接池总连接数",
    }
}
```

## 配置说明

### 指标收集配置
```go
// 创建指标收集器
collector := NewStandardMetricCollector()

// 注册系统指标
collector.RegisterGauge("jvm.memory.used", func() float64 {
    return getJVMMemoryUsed()
})

collector.RegisterGauge("jvm.cpu.usage", func() float64 {
    return getCPUsage()
})

// 注册业务指标
collector.RegisterCounter("processed.records")
collector.RegisterCounter("error.count")
collector.RegisterTimer("processing.time")
collector.RegisterMeter("requests.per.second")
```

### 指标聚合配置
```go
// 创建聚合器
aggregator := NewStandardMetricsAggregator(
    collector,
    processorRegistry,
    processGroupRegistry,
    clusterManager,
)

// 配置聚合策略
// 按处理器类型聚合
processorTypeAggregations, err := aggregator.AggregateByProcessorType()

// 按流程组聚合
processGroupAggregations, err := aggregator.AggregateByProcessGroup()

// 按时间窗口聚合
timeWindowAggregations, err := aggregator.AggregateByTimeWindow(5 * time.Minute)

// 集群聚合
clusterAggregations, err := aggregator.AggregateByCluster()
```

### 指标存储配置
```go
// 配置存储路径
repository := NewTimeSeriesMetricsRepository("./metrics-storage")

// 配置存储管理器
storageManager := NewMetricsStorageManager(repository)

// 配置归档策略
err := storageManager.ArchiveMetrics()
if err != nil {
    log.Printf("归档指标失败: %v", err)
}
```

### 指标导出配置
```go
// 配置导出器
exporter := NewStandardMetricsExporter()

// 启动HTTP端点
err := exporter.StartHTTPEndpoint(8080)
if err != nil {
    log.Fatalf("启动HTTP端点失败: %v", err)
}

// 配置Prometheus导出
prometheusData, err := exporter.ExportToPrometheus(snapshot)

// 配置OpenTSDB导出
opentsdbData, err := exporter.ExportToOpenTSDB(snapshot)

// 配置Graphite导出
graphiteData, err := exporter.ExportToGraphite(snapshot)
```

### 数据溯源配置
```go
// 配置溯源存储库
repository := NewInMemoryProvenanceRepository()

// 配置溯源管理器
lineageManager := NewLineageManager(repository)

// 配置事件清理
err := lineageManager.PurgeExpiredEvents(24 * time.Hour)
if err != nil {
    log.Printf("清理过期事件失败: %v", err)
}
```

## 性能指标

| 指标           | 目标值        | 说明                   |
|---------------|---------------|------------------------|
| 指标收集延迟    | <10ms         | 指标采集耗时           |
| 导出性能        | <50ms         | 指标导出耗时           |
| 存储开销        | <100ms        | 指标持久化耗时         |
| 查询性能        | <200ms        | 历史指标查询耗时       |
| 聚合性能        | <300ms        | 指标聚合耗时           |
| 溯源查询性能    | <500ms        | 数据溯源查询耗时       |

## 扩展点

### 自定义指标提供者
```go
type CustomMetricProvider struct {
    // 自定义字段
}

func (cmp *CustomMetricProvider) RegisterMetrics(collector MetricCollector) {
    // 注册自定义指标
    collector.RegisterGauge("custom.metric", func() float64 {
        return cmp.getCustomValue()
    })
}

func (cmp *CustomMetricProvider) GetMetricDescriptions() map[string]string {
    return map[string]string{
        "custom.metric": "自定义指标描述",
    }
}
```

### 自定义指标聚合器
```go
type CustomMetricsAggregator struct {
    // 实现MetricsAggregator接口
}

func (cma *CustomMetricsAggregator) AggregateByProcessorType() (map[string]*AggregatedMetrics, error) {
    // 自定义聚合逻辑
}

func (cma *CustomMetricsAggregator) AggregateByProcessGroup() (map[string]*AggregatedMetrics, error) {
    // 自定义聚合逻辑
}
```

### 自定义指标存储库
```go
type CustomMetricsRepository struct {
    // 实现MetricsRepository接口
}

func (cmr *CustomMetricsRepository) StoreSnapshot(snapshot *MetricsSnapshot) error {
    // 自定义存储逻辑
}

func (cmr *CustomMetricsRepository) QueryHistoricalMetrics(metricName string, startTime, endTime time.Time) ([]*MetricsSnapshot, error) {
    // 自定义查询逻辑
}
```

### 自定义指标导出器
```go
type CustomMetricsExporter struct {
    // 实现MetricsExporter接口
}

func (cme *CustomMetricsExporter) ExportToPrometheus(snapshot *MetricsSnapshot) (string, error) {
    // 自定义导出逻辑
}

func (cme *CustomMetricsExporter) ExportToCustomFormat(snapshot *MetricsSnapshot) ([]byte, error) {
    // 自定义格式导出
}
```

### 自定义溯源服务
```go
type CustomLineageService struct {
    // 实现LineageService接口
}

func (cls *CustomLineageService) TraceFlowFile(flowFileID string) (*LineageResult, error) {
    // 自定义追踪逻辑
}

func (cls *CustomLineageService) AddLineageEvent(event *LineageEvent) error {
    // 自定义事件添加逻辑
}
```

## 最佳实践

### 1. 指标命名规范
- 使用小写字母和点号分隔
- 使用有意义的名称
- 遵循层次化命名结构
- 避免使用特殊字符

### 2. 指标收集策略
- 合理设置收集频率
- 避免过度收集
- 使用适当的指标类型
- 及时清理无用指标

### 3. 指标聚合优化
- 选择合适的聚合维度
- 避免重复聚合
- 使用缓存机制
- 定期清理聚合结果

### 4. 存储管理
- 合理设置保留期限
- 定期归档数据
- 监控存储空间
- 优化查询性能

### 5. 导出配置
- 选择合适的导出格式
- 配置适当的导出频率
- 监控导出性能
- 处理导出失败

### 6. 溯源管理
- 合理设置事件保留期限
- 定期清理过期事件
- 优化溯源查询
- 监控溯源性能

## 故障排除

### 常见问题

1. **指标收集失败**
   - 检查指标名称是否重复
   - 验证指标提供者是否正确
   - 确认系统资源是否充足

2. **聚合性能问题**
   - 检查聚合策略是否合理
   - 验证数据量是否过大
   - 确认缓存是否有效

3. **存储空间不足**
   - 检查保留期限设置
   - 验证归档策略
   - 确认清理任务是否正常

4. **导出失败**
   - 检查网络连接
   - 验证目标系统状态
   - 确认导出格式是否正确

5. **溯源查询慢**
   - 检查索引是否有效
   - 验证查询条件是否合理
   - 确认数据量是否过大

### 调试技巧

1. **启用详细日志**
   ```go
   // 设置日志级别
   log.SetLevel(log.DebugLevel)
   ```

2. **监控指标状态**
   ```go
   // 获取指标数量
   metrics := collector.ListMetrics()
   fmt.Printf("指标数量: %d\n", len(metrics))
   ```

3. **检查存储状态**
   ```go
   // 获取存储摘要
   summary, err := storageManager.GetMetricsSummary(startTime, endTime)
   if err != nil {
       log.Printf("获取存储摘要失败: %v", err)
   }
   ```

4. **验证导出格式**
   ```go
   // 测试导出
   data, err := exporter.ExportToPrometheus(snapshot)
   if err != nil {
       log.Printf("导出失败: %v", err)
   }
   ```

## 版本历史

### v1.0.0 (2024-01-01)
- 初始版本发布
- 支持基本的指标收集功能
- 实现指标聚合能力
- 提供指标存储功能
- 集成指标导出机制
- 支持数据溯源服务

## 贡献指南

1. Fork 项目
2. 创建功能分支
3. 提交更改
4. 推送到分支
5. 创建 Pull Request

## 许可证

本项目采用 Apache License 2.0 许可证。 