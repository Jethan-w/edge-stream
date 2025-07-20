# WindowManager 模块

## 概述

WindowManager 是 Apache NiFi 实现流数据实时聚合与分析的核心组件，负责按时间/数量维度组织和管理数据窗口。它是数据流处理系统中实现实时计算和聚合分析的关键模块，提供了灵活、高效的数据窗口处理能力。

## 核心特性

### 1. 多样化窗口类型
- **滚动窗口 (Tumbling Window)**: 固定大小、不重叠的时间窗口
- **滑动窗口 (Sliding Window)**: 固定大小、重叠的时间窗口
- **会话窗口 (Session Window)**: 基于活动间隔的动态窗口
- **计数窗口 (Count-based Window)**: 基于记录数量的窗口

### 2. 灵活的时间戳处理
- **事件时间戳提取**: 支持从 JSON、CSV、属性等多种数据源提取时间戳
- **处理时间戳**: 支持基于系统处理时间的窗口计算
- **延迟数据处理**: 提供乱序数据的处理策略（丢弃、侧边输出、包含）

### 3. 丰富的聚合函数
- **基础聚合**: COUNT、SUM、AVG、MAX、MIN
- **位置聚合**: FIRST、LAST
- **自定义聚合**: 支持扩展自定义聚合逻辑
- **多维度聚合**: 支持同时应用多种聚合函数

### 4. 灵活的触发策略
- **时间触发**: 基于窗口结束时间的触发
- **计数触发**: 基于记录数量的触发
- **混合触发**: 时间和计数的组合触发
- **手动触发**: 支持手动控制窗口触发
- **水印触发**: 基于事件时间水印的触发

### 5. 状态持久化
- **内存状态管理**: 适用于测试和临时场景
- **文件系统状态管理**: 基于 JSON 文件的持久化存储
- **状态恢复**: 支持从持久化存储恢复窗口状态

## 架构设计

### 核心组件

```
WindowManager
├── WindowDefiner          # 窗口定义器
│   ├── TimeWindow         # 时间窗口
│   ├── CountWindow        # 计数窗口
│   ├── SessionWindow      # 会话窗口
│   └── SlidingWindow      # 滑动窗口
├── TimestampExtractor     # 时间戳提取器
│   ├── JsonTimestampExtractor
│   ├── AttributeTimestampExtractor
│   ├── ProcessingTimeExtractor
│   └── CsvTimestampExtractor
├── AggregationCalculator  # 聚合计算器
│   ├── AverageAggregator
│   ├── SumAggregator
│   ├── MaxAggregator
│   ├── MinAggregator
│   ├── CountAggregator
│   ├── FirstAggregator
│   ├── LastAggregator
│   └── MultiAggregationStrategy
├── WindowStateManager     # 窗口状态管理器
│   ├── MemoryStateManager
│   └── ContentRepositoryStateManager
└── WindowTrigger          # 窗口触发器
    ├── TimeTriggerStrategy
    ├── CountTriggerStrategy
    ├── MixedTriggerStrategy
    ├── ManualTriggerStrategy
    └── WatermarkTriggerStrategy
```

### 数据流

```
FlowFile → TimestampExtractor → Window → Aggregator → OutputProcessor
    ↓              ↓              ↓           ↓            ↓
事件时间戳     窗口分配      聚合计算    触发检查    结果输出
```

## 快速开始

### 1. 基本使用

```go
package main

import (
    "fmt"
    "time"
    
    "github.com/edge-stream/internal/WindowManager"
    "github.com/edge-stream/internal/WindowManager/AggregationCalculator"
    "github.com/edge-stream/internal/WindowManager/TimestampExtractor"
)

func main() {
    // 创建窗口管理器
    windowManager := WindowManager.NewStandardWindowManager()
    
    // 创建时间戳提取器
    timestampExtractor := TimestampExtractor.NewJsonTimestampExtractor("timestamp", "2006-01-02T15:04:05Z")
    windowManager.SetTimestampExtractor(timestampExtractor)
    
    // 创建聚合策略
    aggregationStrategy := AggregationCalculator.NewMultiAggregationStrategy()
    aggregationStrategy.AddAggregator(AggregationCalculator.NewAverageAggregator("temperature"))
    aggregationStrategy.AddAggregator(AggregationCalculator.NewMaxAggregator("temperature"))
    
    // 创建5分钟时间窗口
    startTime := time.Now().UnixMilli()
    endTime := startTime + 5*60*1000
    window, err := windowManager.CreateTimeWindow(startTime, endTime)
    if err != nil {
        panic(err)
    }
    
    // 添加数据
    data := map[string]interface{}{
        "temperature": 25.5,
        "timestamp":   time.Now().Format("2006-01-02T15:04:05Z"),
    }
    
    jsonData, _ := json.Marshal(data)
    flowFile := WindowManager.NewFlowFile("temp_001", jsonData, nil)
    
    if err := window.AddFlowFile(flowFile); err != nil {
        panic(err)
    }
    
    // 计算聚合结果
    flowFiles := window.GetFlowFiles()
    results, err := aggregationStrategy.ComputeAggregations(flowFiles)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Aggregation results: %+v\n", results)
}
```

### 2. 温度传感器数据分析

```go
func analyzeTemperatureData() error {
    // 创建温度窗口分析器
    analyzer := NewTemperatureWindowAnalyzer()
    
    // 分析设备温度数据
    return analyzer.AnalyzeDeviceTemperature()
}
```

### 3. 会话窗口分析

```go
func analyzeUserSessions() error {
    // 创建窗口管理器
    windowManager := WindowManager.NewStandardWindowManager()
    
    // 创建会话窗口（5分钟超时）
    window, err := windowManager.CreateSessionWindow(5 * time.Minute)
    if err != nil {
        return err
    }
    
    // 创建聚合策略
    aggregationStrategy := AggregationCalculator.NewMultiAggregationStrategy()
    aggregationStrategy.AddAggregator(AggregationCalculator.NewCountAggregator())
    aggregationStrategy.AddAggregator(AggregationCalculator.NewFirstAggregator("user_id"))
    
    // 添加用户会话数据
    sessionData := map[string]interface{}{
        "user_id": "user_001",
        "action":  "login",
    }
    
    jsonData, _ := json.Marshal(sessionData)
    flowFile := WindowManager.NewFlowFile("session_001", jsonData, nil)
    
    if err := window.AddFlowFile(flowFile); err != nil {
        return err
    }
    
    // 计算聚合结果
    flowFiles := window.GetFlowFiles()
    results, err := aggregationStrategy.ComputeAggregations(flowFiles)
    if err != nil {
        return err
    }
    
    fmt.Printf("Session analysis results: %+v\n", results)
    return nil
}
```

## 配置说明

### 窗口配置

```go
// 窗口配置结构
type WindowConfig struct {
    WindowSize       int64         // 窗口大小（毫秒）
    SlideInterval    int64         // 滑动间隔（毫秒）
    SessionTimeout   time.Duration // 会话超时时间
    MaxCount         int           // 最大计数
    TriggerThreshold int           // 触发阈值
}

// 创建配置
config := WindowDefiner.WindowConfig{
    WindowSize:       300000, // 5分钟
    SlideInterval:    60000,  // 1分钟
    SessionTimeout:   5 * time.Minute,
    MaxCount:         1000,
    TriggerThreshold: 100,
}
```

### 时间戳提取器配置

```go
// JSON 时间戳提取器
jsonExtractor := TimestampExtractor.NewJsonTimestampExtractor(
    "timestamp",           // 时间戳字段名
    "2006-01-02T15:04:05Z", // 时间格式
)

// 属性时间戳提取器
attrExtractor := TimestampExtractor.NewAttributeTimestampExtractor(
    "event_time",          // 属性名
    "2006-01-02T15:04:05Z", // 时间格式
)

// CSV 时间戳提取器
csvExtractor := TimestampExtractor.NewCsvTimestampExtractor(
    0,                     // 时间戳列索引
    "2006-01-02T15:04:05Z", // 时间格式
    ",",                   // 分隔符
)
```

### 触发策略配置

```go
// 时间触发策略
timeTrigger := WindowTrigger.NewTimeTriggerStrategy(5000) // 5秒延迟

// 计数触发策略
countTrigger := WindowTrigger.NewCountTriggerStrategy(100) // 100条记录

// 混合触发策略
mixedTrigger := WindowTrigger.NewMixedTriggerStrategy(
    5000, // 5秒延迟
    100,  // 100条记录
)

// 水印触发策略
watermarkManager := WindowTrigger.NewSimpleWatermarkManager()
watermarkTrigger := WindowTrigger.NewWatermarkTriggerStrategy(
    watermarkManager,
    300000, // 5分钟最大延迟
)
```

## 性能指标

| 指标 | 目标值 | 说明 |
|------|--------|------|
| 窗口计算延迟 | <50ms | 窗口聚合计算耗时 |
| 状态持久化开销 | <20ms | 窗口状态存储耗时 |
| 乱序数据处理 | <100ms | 延迟数据处理耗时 |
| 并发窗口数 | 100+ | 支持的最大并发窗口数 |
| 内存使用 | <1GB | 单个窗口管理器内存占用 |

## 扩展开发

### 自定义聚合器

```go
// 实现 WindowAggregator 接口
type CustomAggregator struct {
    field string
    sum   float64
    count int
    mu    sync.RWMutex
}

func (ca *CustomAggregator) Aggregate(flowFile *WindowManager.FlowFile) error {
    // 实现聚合逻辑
    return nil
}

func (ca *CustomAggregator) GetResult() interface{} {
    // 返回聚合结果
    return nil
}

func (ca *CustomAggregator) Reset() {
    // 重置聚合状态
}

func (ca *CustomAggregator) GetAggregationFunction() WindowManager.AggregationFunction {
    return WindowManager.AggregationFunctionAvg
}
```

### 自定义时间戳提取器

```go
// 实现 TimestampExtractor 接口
type CustomTimestampExtractor struct {
    // 自定义字段
}

func (cte *CustomTimestampExtractor) ExtractTimestamp(flowFile *WindowManager.FlowFile) int64 {
    // 实现时间戳提取逻辑
    return time.Now().UnixMilli()
}

func (cte *CustomTimestampExtractor) GetTimestampFormat() string {
    return "custom_format"
}

func (cte *CustomTimestampExtractor) AllowLateData(maxLateness int64) bool {
    return true
}
```

### 自定义触发策略

```go
// 实现 TriggerStrategy 接口
type CustomTriggerStrategy struct {
    // 自定义字段
}

func (cts *CustomTriggerStrategy) ShouldTrigger(window WindowManager.Window, currentTime int64) bool {
    // 实现触发判断逻辑
    return false
}

func (cts *CustomTriggerStrategy) GetTriggerType() WindowTrigger.TriggerType {
    return WindowTrigger.TriggerTypeTime
}
```

## 最佳实践

### 1. 窗口大小选择
- **时间窗口**: 根据业务需求选择合适的时间间隔（秒、分钟、小时）
- **计数窗口**: 根据数据量和处理能力选择记录数量
- **会话窗口**: 根据用户行为模式选择超时时间

### 2. 聚合函数选择
- **实时监控**: 使用 AVG、MAX、MIN 等统计函数
- **数据统计**: 使用 COUNT、SUM 等累计函数
- **状态跟踪**: 使用 FIRST、LAST 等位置函数

### 3. 触发策略配置
- **实时性要求高**: 使用较小的触发延迟或计数阈值
- **资源优化**: 使用较大的触发延迟或计数阈值
- **混合场景**: 使用混合触发策略平衡实时性和资源使用

### 4. 状态管理
- **测试环境**: 使用内存状态管理器
- **生产环境**: 使用文件系统状态管理器
- **高可用**: 考虑使用分布式状态存储

### 5. 延迟数据处理
- **数据完整性**: 使用 INCLUDE 策略包含延迟数据
- **实时性**: 使用 DROP 策略丢弃延迟数据
- **监控分析**: 使用 SIDE_OUTPUT 策略输出到侧边流

## 故障排除

### 常见问题

1. **窗口不触发**
   - 检查触发策略配置
   - 验证时间戳提取是否正确
   - 确认窗口是否已注册到触发器

2. **聚合结果异常**
   - 检查数据格式是否正确
   - 验证聚合字段是否存在
   - 确认聚合器配置是否正确

3. **内存使用过高**
   - 减少并发窗口数量
   - 增加触发频率
   - 使用状态持久化

4. **延迟数据处理问题**
   - 检查水印更新逻辑
   - 验证延迟策略配置
   - 确认时间戳格式

### 调试技巧

1. **启用详细日志**
   ```go
   // 设置日志级别
   log.SetLevel(log.DebugLevel)
   ```

2. **监控窗口状态**
   ```go
   // 获取活跃窗口
   activeWindows := windowManager.GetActiveWindows()
   for _, window := range activeWindows {
       fmt.Printf("Window: %s, FlowFiles: %d\n", 
           window.GetID(), len(window.GetFlowFiles()))
   }
   ```

3. **检查聚合结果**
   ```go
   // 手动计算聚合
   results, err := aggregationStrategy.ComputeAggregations(flowFiles)
   if err != nil {
       fmt.Printf("Aggregation error: %v\n", err)
   }
   ```

## 版本历史

### v1.0.0 (2024-01-01)
- 初始版本发布
- 支持基本窗口类型（时间、计数、会话、滑动）
- 支持基础聚合函数
- 支持时间戳提取和延迟数据处理
- 支持状态持久化和恢复

### 计划功能
- 分布式窗口处理
- 更多聚合函数（百分位数、标准差等）
- 窗口状态压缩
- 实时监控和指标收集
- 云原生部署支持

## 贡献指南

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打开 Pull Request

## 许可证

本项目采用 Apache License 2.0 许可证。详见 [LICENSE](LICENSE) 文件。 