# Sink 模块

## 概述

Sink 是 Apache NiFi 数据流转的"出口通道"，负责将处理后的数据输出到外部目标系统的底层模块。它是整个数据流处理系统的数据出口，提供了安全、可靠、高性能的数据输出能力。

## 核心功能

### 1. 多目标数据输出
- 支持文件系统、数据库、消息队列、搜索引擎、REST API 等多种输出目标
- 提供统一的数据输出接口
- 支持自定义输出协议扩展

### 2. 安全数据传输
- 支持 TLS/SSL、HTTPS、SFTP 等多种安全协议
- 提供 OAuth2、客户端证书、用户名密码、API 密钥等多种认证方式
- 支持 AES、RSA、ChaCha20 等加密算法

### 3. 可靠交付机制
- 支持最多一次、至少一次、精确一次三种可靠性策略
- 提供指数退避、线性退避、固定延迟等重试策略
- 支持事务性写入

### 4. 错误处理与路由
- 灵活的错误路由策略（重试、错误队列、丢弃、死信队列）
- 支持错误通知（日志、邮件等）
- 提供详细的错误统计信息

### 5. 自定义协议扩展
- 支持动态注册自定义输出协议
- 提供协议适配器工厂模式
- 支持协议元数据管理

## 架构设计

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Sink Core     │    │ Target Adapters │    │  External       │
│                 │    │                 │    │  Systems        │
│ ┌─────────────┐ │    │ ┌─────────────┐ │    │ ┌─────────────┐ │
│ │   Config    │ │    │ │ FileSystem  │ │    │ │ File System │ │
│ └─────────────┘ │    │ └─────────────┘ │    │ └─────────────┘ │
│ ┌─────────────┐ │    │ ┌─────────────┐ │    │ ┌─────────────┐ │
│ │   State     │ │    │ │ Database    │ │    │ │ Database    │ │
│ └─────────────┘ │    │ └─────────────┘ │    │ └─────────────┘ │
│ ┌─────────────┐ │    │ ┌─────────────┐ │    │ ┌─────────────┐ │
│ │   Metrics   │ │    │ │ MessageQueue│ │    │ │ Message Q   │ │
│ └─────────────┘ │    │ └─────────────┘ │    │ └─────────────┘ │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ Error Handler   │    │ Retry Manager   │    │ Security Handler│
│                 │    │                 │    │                 │
│ ┌─────────────┐ │    │ ┌─────────────┐ │    │ ┌─────────────┐ │
│ │ Error Queue │ │    │ │ Exponential │ │    │ │ TLS/SSL     │ │
│ └─────────────┘ │    │ │ Backoff     │ │    │ └─────────────┘ │
│ ┌─────────────┐ │    │ └─────────────┘ │    │ ┌─────────────┐ │
│ │ Retry Queue │ │    │ ┌─────────────┐ │    │ │ OAuth2      │ │
│ └─────────────┘ │    │ │ Linear      │ │    │ └─────────────┘ │
│ ┌─────────────┐ │    │ │ Backoff     │ │    │ ┌─────────────┐ │
│ │ Dead Letter │ │    │ └─────────────┘ │    │ │ Encryption  │ │
│ └─────────────┘ │    └─────────────────┘    │ └─────────────┘ │
└─────────────────┘                           └─────────────────┘
```

## 快速开始

### 1. 基本使用

```go
package main

import (
    "log"
    "time"
    
    "github.com/edge-stream/internal/flowfile"
    "github.com/edge-stream/internal/sink"
)

func main() {
    // 创建 Sink 配置
    sinkConfig := &sink.SinkConfig{
        ID:   "my-sink",
        Name: "My Sink",
        OutputTargets: []sink.OutputTargetConfig{
            {
                ID:   "file-output",
                Type: sink.OutputTargetTypeFileSystem,
                Properties: map[string]string{
                    "output.directory": "/tmp/output",
                },
            },
        },
    }

    // 创建 Sink 实例
    sinkInstance := sink.NewSink(sinkConfig)

    // 初始化
    if err := sinkInstance.Initialize(); err != nil {
        log.Fatalf("Failed to initialize sink: %v", err)
    }

    // 启动
    if err := sinkInstance.Start(); err != nil {
        log.Fatalf("Failed to start sink: %v", err)
    }

    // 创建 FlowFile
    flowFile := flowfile.NewFlowFile()
    flowFile.SetContent([]byte("Hello, World!"))
    flowFile.SetAttribute("filename", "test.txt")

    // 写入数据
    if err := sinkInstance.Write(flowFile, "file-output"); err != nil {
        log.Printf("Failed to write: %v", err)
    }

    // 停止
    sinkInstance.Stop()
}
```

### 2. 数据库输出示例

```go
// 数据库输出配置
dbConfig := sink.OutputTargetConfig{
    ID:   "mysql-output",
    Type: sink.OutputTargetTypeDatabase,
    Properties: map[string]string{
        "driver_name":        "mysql",
        "dsn":               "user:password@tcp(localhost:3306)/testdb",
        "table_name":        "flowfiles",
        "enable_transaction": "true",
    },
}
```

### 3. 自定义协议示例

```go
// 注册自定义 Redis 协议
sinkInstance.RegisterCustomProtocol("redis", &RedisOutputAdapter{
    AbstractTargetAdapter: TargetAdapter.NewAbstractTargetAdapter(
        "redis-adapter",
        "Redis Output Adapter",
        "Redis",
    ),
    host:     "localhost",
    port:     6379,
    database: 0,
})

// 使用自定义协议
redisConfig := sink.OutputTargetConfig{
    ID:   "redis-output",
    Type: sink.OutputTargetTypeCustom,
    Properties: map[string]string{
        "protocol": "redis",
    },
}
```

## 配置说明

### SinkConfig 配置项

| 配置项 | 类型 | 说明 | 默认值 |
|--------|------|------|--------|
| ID | string | Sink 唯一标识 | - |
| Name | string | Sink 名称 | - |
| Description | string | Sink 描述 | - |
| OutputTargets | []OutputTargetConfig | 输出目标配置列表 | - |
| SecurityConfig | *SecurityConfiguration | 安全配置 | nil |
| ReliabilityConfig | *ReliabilityConfig | 可靠性配置 | 默认重试策略 |
| ErrorHandlingConfig | *ErrorHandlingConfig | 错误处理配置 | 默认错误处理 |
| MaxConcurrentOutputs | int | 最大并发输出数 | 10 |
| OutputTimeout | time.Duration | 输出超时时间 | 30s |
| EnableMetrics | bool | 是否启用指标收集 | false |

### 输出目标类型

| 类型 | 说明 | 必需配置 |
|------|------|----------|
| FileSystem | 文件系统输出 | output.directory |
| Database | 数据库输出 | driver_name, dsn, table_name |
| MessageQueue | 消息队列输出 | broker_url, topic_name |
| SearchEngine | 搜索引擎输出 | server_url, index_name |
| RESTAPI | REST API 输出 | base_url, endpoint |
| Custom | 自定义协议输出 | protocol |

### 安全配置

```go
securityConfig := &sink.SecurityConfiguration{
    Protocol:             sink.SecurityProtocolTLS,
    AuthenticationMethod: sink.AuthenticationMethodOAuth2,
    EncryptionAlgorithm:  sink.EncryptionAlgorithmAES256,
    OAuth2Config: &sink.OAuth2Config{
        ClientID:     "your-client-id",
        ClientSecret: "your-client-secret",
        TokenURL:     "https://oauth.example.com/token",
        Scope:        "read write",
    },
}
```

### 可靠性配置

```go
reliabilityConfig := &sink.ReliabilityConfig{
    Strategy:           sink.ReliabilityStrategyAtLeastOnce,
    MaxRetries:         3,
    BaseRetryDelay:     time.Second,
    MaxRetryDelay:      30 * time.Second,
    RetryMultiplier:    2.0,
    EnableTransaction:  true,
    TransactionTimeout: 30 * time.Second,
}
```

## 错误处理

### 错误路由策略

- **Retry**: 重试策略，将错误路由到重试队列
- **RouteToError**: 错误队列策略，将错误路由到错误队列
- **Drop**: 丢弃策略，直接丢弃错误数据
- **DeadLetter**: 死信队列策略，将错误路由到死信队列

### 错误通知

支持多种错误通知方式：
- 日志通知（默认）
- 邮件通知
- 可扩展自定义通知器

## 监控和指标

### 统计信息

Sink 提供丰富的统计信息：

- **输出统计**: 总输出数、成功数、失败数、平均延迟
- **重试统计**: 重试次数、平均重试时间、重试成功率
- **错误统计**: 错误总数、错误类型分布、错误处理时间
- **安全统计**: 加密数据量、认证成功率、传输统计

### 健康检查

每个适配器都提供健康检查功能：

```go
// 检查适配器健康状态
if adapter.IsHealthy() {
    log.Println("Adapter is healthy")
} else {
    log.Println("Adapter is unhealthy")
}
```

## 扩展开发

### 自定义适配器

要实现自定义输出适配器，需要：

1. 实现 `TargetAdapter` 接口
2. 继承 `AbstractTargetAdapter` 基类
3. 实现 `doWrite` 方法
4. 注册到协议注册表

```go
type CustomAdapter struct {
    *TargetAdapter.AbstractTargetAdapter
    // 自定义字段
}

func (a *CustomAdapter) doWrite(flowFile interface{}) error {
    // 实现具体的写入逻辑
    return nil
}

// 注册自定义适配器
sinkInstance.RegisterCustomProtocol("custom", func(config map[string]string) (TargetAdapter.TargetAdapter, error) {
    return &CustomAdapter{}, nil
})
```

### 自定义错误通知器

```go
type CustomNotifier struct{}

func (n *CustomNotifier) NotifyError(errorItem *ErrorItem) error {
    // 实现自定义通知逻辑
    return nil
}

func (n *CustomNotifier) GetName() string {
    return "Custom"
}
```

## 性能优化

### 批量处理

对于支持批量处理的目标系统，可以启用批量模式：

```go
config.Properties["enable_batch"] = "true"
config.Properties["batch_size"] = "100"
```

### 连接池

数据库适配器支持连接池配置：

```go
config.Properties["max_open_conns"] = "10"
config.Properties["max_idle_conns"] = "5"
config.Properties["conn_max_lifetime"] = "1h"
```

### 并发控制

通过 `MaxConcurrentOutputs` 配置控制并发输出数量，避免资源耗尽。

## 最佳实践

1. **配置验证**: 在初始化前验证所有必需配置
2. **错误处理**: 合理配置错误路由策略和重试机制
3. **监控告警**: 启用指标收集和错误通知
4. **资源管理**: 合理配置连接池和并发数
5. **安全配置**: 使用适当的安全协议和认证方式
6. **性能调优**: 根据实际负载调整批量大小和重试策略

## 故障排除

### 常见问题

1. **连接失败**: 检查网络连接和认证配置
2. **权限错误**: 验证文件系统权限或数据库权限
3. **配置错误**: 检查必需配置项是否完整
4. **性能问题**: 调整批量大小和并发数
5. **内存泄漏**: 确保正确关闭适配器和连接

### 调试模式

启用详细日志记录：

```go
log.SetLevel(log.DebugLevel)
```

## 版本历史

- v1.0.0: 初始版本，支持基本的文件系统和数据库输出
- v1.1.0: 添加消息队列和搜索引擎支持
- v1.2.0: 增强安全功能和错误处理
- v1.3.0: 添加自定义协议扩展支持

## 贡献指南

欢迎贡献代码！请遵循以下步骤：

1. Fork 项目
2. 创建功能分支
3. 提交更改
4. 创建 Pull Request

## 许可证

本项目采用 MIT 许可证。详见 [LICENSE](LICENSE) 文件。 