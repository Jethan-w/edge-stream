# Edge Stream 基础框架

## 概述

Edge Stream 是一个简化的数据流处理框架，专注于提供最基础的数据处理能力。该框架已经过大幅简化，移除了复杂的企业级功能，保留了核心的数据流处理组件。

## 架构简化

### 保留的核心组件

#### 1. FlowFile (`internal/flowfile`)
- **功能**: 数据流转的基本单元
- **特性**: 
  - UUID 唯一标识
  - 属性映射
  - 内容存储
  - 时间戳和血缘追踪

#### 2. Source (`internal/source`)
- **功能**: 数据源接口
- **简化接口**:
  - `Read()` - 读取数据
  - `GetName()` - 获取名称
  - `HasNext()` - 检查是否有更多数据
- **实现**: `SimpleSource`, `StringSource`

#### 3. Processor (`internal/processor`)
- **功能**: 数据处理器接口
- **简化接口**:
  - `Process()` - 处理数据
  - `GetName()` - 获取名称
- **实现**: `SimpleProcessor`, `TransformProcessor`

#### 4. Sink (`internal/sink`)
- **功能**: 数据输出接口
- **简化接口**:
  - `Write()` - 写入数据
  - `GetName()` - 获取名称
- **实现**: `SimpleSink`, `ConsoleSink`

#### 5. WindowManager (`internal/windowmanager`)
- **功能**: 窗口管理器
- **特性**:
  - 时间窗口
  - 计数窗口
  - 线程安全
  - 简单易用的 API

### 移除的复杂功能

以下复杂的企业级组件已被完全移除：
- ConfigManager - 配置管理
- ConnectorRegistry - 连接器注册
- MetricCollector - 指标收集
- StateManager - 状态管理
- Pipeline - 流水线管理
- StreamEngine - 流引擎

## 快速开始

### 1. 基础框架示例

```bash
go run ./examples/basic_framework
```

这个示例展示了：
- 创建数据源
- 数据处理（转换为大写）
- 输出到控制台和内存

### 2. 窗口管理器示例

```bash
go run ./examples/windowmanager
```

这个示例展示了：
- 时间窗口和计数窗口
- 数据聚合
- 窗口触发和重置

## 构建和测试

### 构建项目

```bash
# 构建核心模块
go build ./internal/... ./pkg/...

# 构建示例程序
go build ./examples/basic_framework
go build ./examples/windowmanager
```

### 运行测试

```bash
# 测试所有组件
go test ./...

# 测试特定组件
go test ./internal/windowmanager
```

## 项目结构

```
edge-stream/
├── internal/           # 所有核心组件
│   ├── flowfile/      # 数据流文件
│   ├── source/        # 数据源
│   ├── processor/     # 数据处理器
│   ├── sink/          # 数据输出
│   └── windowmanager/ # 窗口管理器
├── examples/          # 示例程序
│   ├── basic_framework/
│   └── windowmanager/
└── README.md
```

## 设计原则

1. **简单性**: 移除了复杂的企业级功能，专注于核心数据处理
2. **可扩展性**: 保留了基础接口，便于后续扩展
3. **易用性**: 提供简单直观的 API
4. **模块化**: 组件之间松耦合，便于独立使用

## 使用场景

- 简单的数据转换和处理
- 数据流的基础操作
- 学习和原型开发
- 轻量级数据处理任务

## 扩展指南

如需要更复杂的功能，可以：
1. 从 `archived/` 目录恢复相应组件
2. 基于现有接口实现自定义组件
3. 参考示例程序了解使用模式
4. 查阅 Apache NiFi 借鉴分析文档获取企业级功能实现建议

## 文档资源

### Apache NiFi 研究与借鉴
- [Apache NiFi 借鉴分析与功能增强建议](doc/Apache_NiFi_借鉴分析与功能增强建议.md) - 全面分析 Apache NiFi 架构，提出 Edge Stream 功能增强建议
- [Apache NiFi 核心设计模式与架构总结](doc/Apache_NiFi_核心设计模式与架构总结.md) - 深入解析 Apache NiFi 的设计模式和架构原则
- [技术实现指南：配置管理与监控](doc/技术实现指南_配置管理与监控.md) - 详细的配置管理和监控系统实现指南

### Apache NiFi 详细技术分析
- [Apache NiFi 架构全景解析](doc/apache_nifi/Apache_NiFi_架构全景解析.md)
- [Apache NiFi 信息流与数据流全景解析](doc/apache_nifi/Apache_NiFi_信息流与数据流全景解析.md)
- [ConfigManager 深度技术分析](doc/apache_nifi/ConfigManager_深度技术分析.md)
- [Pipeline 深度技术分析](doc/apache_nifi/Pipeline_深度技术分析.md)
- [Source 深度技术分析](doc/apache_nifi/Source_深度技术分析.md)
- [Processor 深度技术分析](doc/apache_nifi/Processor_深度技术分析.md)

### 数据处理组件
- [数据处理组件文档](pkg/dataprocessor/README.md) - MySQL到文件和Redis到数据库的流式处理组件

## 验证结果

✅ 编译成功  
✅ 基础框架示例运行正常  
✅ 窗口管理器示例运行正常  
✅ 核心模块单元测试通过  

框架已成功简化为最基础的数据流处理能力，适合轻量级使用场景。