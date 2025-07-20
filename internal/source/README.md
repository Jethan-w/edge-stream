# EdgeStream Source 模块

## 模块概述

Source 是 EdgeStream 数据流转的"入口枢纽"，负责从外部系统接入各类数据并传递至 Pipeline 的底层模块。它是整个数据流处理系统的数据入口，提供了灵活、安全、高性能的数据接入能力。

### 核心特性

- **多模态数据接收**: 支持视频、文本、二进制、音频等多种数据类型
- **安全数据接收**: 提供TLS/SSL、OAuth2认证、IP白名单等安全机制
- **灵活配置管理**: 支持动态配置更新和配置验证
- **协议扩展**: 支持自定义协议适配器扩展
- **高性能**: 支持高并发连接和低延迟数据处理
- **可扩展性**: 模块化设计，易于扩展和维护

## 架构设计

### 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                    Source 模块架构                           │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │  多模态接收器 │  │  安全接收器  │  │  配置管理器  │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
│                                                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │ 协议适配器   │  │ 数据源管理器 │  │ 协议注册表  │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
├─────────────────────────────────────────────────────────────┤
│                    核心接口层                                │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │   Source    │  │ DataPacket  │  │  Protocol   │         │
│  │  Interface  │  │  Interface  │  │  Adapter    │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
└─────────────────────────────────────────────────────────────┘
```

### 组件关系

- **Source**: 数据源接口，定义数据源的基本行为
- **MultiModalReceiver**: 多模态数据接收器，处理不同类型的数据
- **SecureReceiver**: 安全数据接收器，提供安全接入机制
- **ConfigManager**: 配置管理器，处理配置的加载、验证和更新
- **ProtocolAdapter**: 协议适配器，支持多种协议的数据接入
- **SourceManager**: 数据源管理器，统一管理所有数据源

## 核心组件

### 1. Source 接口

Source 接口定义了数据源的基本行为：

```go
type Source interface {
    Initialize(ctx context.Context) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    GetState() SourceState
    GetConfiguration() *SourceConfiguration
    Validate() *ValidationResult
}
```

### 2. 数据包类型

支持多种数据类型的数据包：

- **VideoPacket**: 视频数据包，包含帧数据、编码、分辨率等信息
- **TextPacket**: 文本数据包，支持多种编码格式
- **BinaryPacket**: 二进制数据包，用于原始数据传输
- **AudioPacket**: 音频数据包，支持多种音频格式
- **CustomDataPacket**: 自定义数据包，支持任意类型数据

### 3. 多模态接收器

MultiModalReceiver 提供统一的多模态数据接收能力：

```go
type MultiModalReceiver struct {
    videoReceiver    *VideoReceiver
    textReceiver     *TextReceiver
    binaryReceiver   *BinaryReceiver
    audioReceiver    *AudioReceiver
    customReceiver   *CustomReceiver
    encodingDetector *EncodingDetector
}
```

### 4. 安全接收器

SecureReceiver 提供多层次的安全接入机制：

- **TLS/SSL 支持**: 传输层安全加密
- **OAuth2 认证**: 基于令牌的身份认证
- **IP 白名单**: 基于IP地址的访问控制

### 5. 配置管理器

ConfigManager 提供灵活的配置管理能力：

- **配置解析**: 支持多种格式的配置解析
- **配置验证**: 自动验证配置的有效性
- **动态更新**: 支持运行时配置更新
- **变更通知**: 配置变更事件通知机制

### 6. 协议适配器

ProtocolAdapter 提供协议扩展能力：

- **内置协议**: TCP、UDP、HTTP、Modbus等
- **自定义协议**: 支持自定义协议扩展
- **协议注册**: 动态协议注册机制

## 使用方法

### 基本使用

```go
package main

import (
    "context"
    "log"
    
    "github.com/edge-stream/internal/source"
    "github.com/edge-stream/internal/source/MultiModalReceiver"
)

func main() {
    // 创建多模态数据接收器
    receiver := multimodalreceiver.NewMultiModalReceiver()
    
    // 初始化
    ctx := context.Background()
    if err := receiver.Initialize(ctx); err != nil {
        log.Fatal(err)
    }
    
    // 启动接收器
    if err := receiver.Start(ctx); err != nil {
        log.Fatal(err)
    }
    
    // 接收数据
    videoPacket := &source.VideoPacket{
        FrameData: []byte("video data"),
        Timestamp: time.Now(),
        Codec:     source.CodecH264,
        Resolution: source.Dimension{Width: 1920, Height: 1080},
        FrameType: source.FrameTypeI,
        Metadata:  make(map[string]string),
    }
    
    if err := receiver.ReceiveVideo(videoPacket); err != nil {
        log.Printf("接收视频数据失败: %v", err)
    }
    
    // 停止接收器
    receiver.Stop(ctx)
}
```

### 安全数据接收

```go
// 创建安全数据接收器
secureReceiver := securereceiver.NewSecureReceiver()

// 配置TLS
tlsContext := &source.TLSContext{
    Protocol: "TLS",
    CertFile: "/path/to/cert.pem",
    KeyFile:  "/path/to/key.pem",
    CAFile:   "/path/to/ca.pem",
}
secureReceiver.ConfigureTLS(tlsContext)

// 配置IP白名单
allowedIPs := []string{"192.168.1.100", "192.168.1.101"}
secureReceiver.ConfigureIPWhitelist(allowedIPs)

// 初始化和启动
secureReceiver.Initialize(ctx)
secureReceiver.Start(ctx)
```

### 配置管理

```go
// 创建配置管理器
configManager := configmanager.NewConfigManager()
configManager.Initialize(ctx)

// 加载配置
configData := map[string]string{
    "protocol":        "tcp",
    "host":            "0.0.0.0",
    "port":            "8080",
    "security.mode":   "tls",
    "max.connections": "100",
}

configManager.LoadConfiguration("my_source", configData)

// 获取配置
config, exists := configManager.GetConfiguration("my_source")
if exists {
    fmt.Printf("配置: %+v\n", config)
}
```

### 协议适配器

```go
// 创建协议管理器
protocolManager := protocoladapter.NewProtocolManager()
protocolManager.Initialize(ctx)

// 创建TCP适配器
tcpConfig := map[string]string{
    "host": "0.0.0.0",
    "port": "8080",
}

tcpAdapter, err := protocolManager.CreateAdapter("tcp", tcpConfig)
if err != nil {
    log.Fatal(err)
}

// 接收数据
packet, err := tcpAdapter.Receive()
if err != nil {
    log.Printf("接收数据失败: %v", err)
} else {
    fmt.Printf("接收到数据: %+v\n", packet)
}

// 关闭适配器
tcpAdapter.Close()
```

### 自定义协议扩展

```go
// 定义自定义协议扩展
type CustomProtocolExtension struct{}

func (cpe *CustomProtocolExtension) GetName() string {
    return "custom-protocol"
}

func (cpe *CustomProtocolExtension) Register(registry *protocoladapter.CustomProtocolRegistry) {
    registry.RegisterProtocol(cpe.GetName(), cpe.CreateAdapter)
}

func (cpe *CustomProtocolExtension) CreateAdapter() protocoladapter.ProtocolAdapter {
    return &CustomProtocolAdapter{
        AbstractProtocolAdapter: protocoladapter.NewAbstractProtocolAdapter("custom-protocol"),
    }
}

// 注册自定义协议
protocolManager := protocoladapter.NewProtocolManager()
customExtension := &CustomProtocolExtension{}
protocolManager.RegisterExtension(customExtension)

// 使用自定义协议
adapter, err := protocolManager.CreateAdapter("custom-protocol", config)
```

## API 参考

### Source 接口

#### Initialize
```go
func (s Source) Initialize(ctx context.Context) error
```
初始化数据源，验证配置并准备资源。

#### Start
```go
func (s Source) Start(ctx context.Context) error
```
启动数据源，开始接收数据。

#### Stop
```go
func (s Source) Stop(ctx context.Context) error
```
停止数据源，释放资源。

#### GetState
```go
func (s Source) GetState() SourceState
```
获取数据源当前状态。

#### GetConfiguration
```go
func (s Source) GetConfiguration() *SourceConfiguration
```
获取数据源配置信息。

#### Validate
```go
func (s Source) Validate() *ValidationResult
```
验证数据源配置的有效性。

### MultiModalReceiver

#### ReceiveVideo
```go
func (mmr *MultiModalReceiver) ReceiveVideo(packet *VideoPacket) error
```
接收视频数据包。

#### ReceiveText
```go
func (mmr *MultiModalReceiver) ReceiveText(packet *TextPacket) error
```
接收文本数据包。

#### ReceiveBinary
```go
func (mmr *MultiModalReceiver) ReceiveBinary(packet *BinaryPacket) error
```
接收二进制数据包。

#### ReceiveCustom
```go
func (mmr *MultiModalReceiver) ReceiveCustom(packet *CustomDataPacket) error
```
接收自定义数据包。

#### GetStatistics
```go
func (mmr *MultiModalReceiver) GetStatistics() *ReceiverStatistics
```
获取接收统计信息。

### SecureReceiver

#### ConfigureTLS
```go
func (sr *SecureReceiver) ConfigureTLS(context *TLSContext) error
```
配置TLS安全连接。

#### Authenticate
```go
func (sr *SecureReceiver) Authenticate(token *OAuth2Token) error
```
进行OAuth2认证。

#### ConfigureIPWhitelist
```go
func (sr *SecureReceiver) ConfigureIPWhitelist(allowedIPs []string) error
```
配置IP白名单。

#### CheckIPWhitelist
```go
func (sr *SecureReceiver) CheckIPWhitelist(clientIP string) bool
```
检查IP是否在白名单中。

### ConfigManager

#### LoadConfiguration
```go
func (cm *ConfigManager) LoadConfiguration(sourceID string, configData map[string]string) error
```
加载数据源配置。

#### GetConfiguration
```go
func (cm *ConfigManager) GetConfiguration(sourceID string) (*SourceConfiguration, bool)
```
获取数据源配置。

#### UpdateConfiguration
```go
func (cm *ConfigManager) UpdateConfiguration(sourceID string, newConfig map[string]string) error
```
更新数据源配置。

#### ValidateConfiguration
```go
func (cm *ConfigManager) ValidateConfiguration(sourceID string) *ValidationResult
```
验证数据源配置。

#### ListConfigurations
```go
func (cm *ConfigManager) ListConfigurations() []string
```
列出所有配置。

### ProtocolAdapter

#### Initialize
```go
func (pa ProtocolAdapter) Initialize(config map[string]string) error
```
初始化协议适配器。

#### Receive
```go
func (pa ProtocolAdapter) Receive() (DataPacket, error)
```
接收数据包。

#### Close
```go
func (pa ProtocolAdapter) Close() error
```
关闭协议适配器。

#### GetProtocolName
```go
func (pa ProtocolAdapter) GetProtocolName() string
```
获取协议名称。

#### GetStatus
```go
func (pa ProtocolAdapter) GetStatus() *AdapterStatus
```
获取适配器状态。

## 配置说明

### 数据源配置

```json
{
    "id": "my_source",
    "type": "multimodal",
    "protocol": "tcp",
    "host": "0.0.0.0",
    "port": 8080,
    "security_mode": "tls",
    "max_connections": 100,
    "connect_timeout": "5000ms",
    "properties": {
        "custom.property": "custom_value"
    }
}
```

### 安全配置

```json
{
    "tls": {
        "protocol": "TLS",
        "cert_file": "/path/to/cert.pem",
        "key_file": "/path/to/key.pem",
        "ca_file": "/path/to/ca.pem"
    },
    "oauth2": {
        "client_id": "your_client_id",
        "client_secret": "your_client_secret",
        "token_url": "https://oauth2.example.com/token",
        "scopes": ["read", "write"]
    },
    "ip_whitelist": ["192.168.1.100", "192.168.1.101"]
}
```

## 性能指标

| 指标 | 目标值 | 说明 |
|------|--------|------|
| 数据接入延迟 | <50ms | 数据包处理耗时 |
| 并发连接数 | 100+ | 支持的最大并发连接 |
| 安全认证开销 | <20ms | 认证耗时 |
| 协议转换性能 | <10ms | 数据包转换耗时 |
| 配置更新延迟 | <100ms | 配置变更生效时间 |

## 最佳实践

### 1. 数据源设计

- 使用适当的数据包类型
- 合理设置缓冲区大小
- 实现错误重试机制
- 添加监控和日志

### 2. 安全配置

- 使用强加密算法
- 定期更新证书
- 限制访问权限
- 监控异常访问

### 3. 性能优化

- 使用连接池
- 实现数据压缩
- 优化内存使用
- 监控性能指标

### 4. 错误处理

- 实现优雅降级
- 记录详细错误信息
- 提供错误恢复机制
- 设置合理的超时时间

## 故障排除

### 常见问题

1. **连接失败**
   - 检查网络连接
   - 验证端口配置
   - 确认防火墙设置

2. **认证失败**
   - 检查证书有效性
   - 验证OAuth2配置
   - 确认IP白名单设置

3. **性能问题**
   - 检查并发连接数
   - 优化缓冲区大小
   - 监控系统资源

4. **配置问题**
   - 验证配置格式
   - 检查必需参数
   - 确认配置权限

### 调试方法

1. **启用详细日志**
   ```go
   // 设置日志级别
   log.SetLevel(log.DebugLevel)
   ```

2. **监控状态**
   ```go
   // 获取数据源状态
   state := source.GetState()
   fmt.Printf("数据源状态: %v\n", state)
   ```

3. **检查统计信息**
   ```go
   // 获取统计信息
   stats := receiver.GetStatistics()
   fmt.Printf("统计信息: %+v\n", stats)
   ```

## 版本历史

### v1.0.0
- 初始版本发布
- 支持多模态数据接收
- 实现安全接入机制
- 提供配置管理功能
- 支持协议扩展

## 贡献指南

1. Fork 项目
2. 创建功能分支
3. 提交更改
4. 推送到分支
5. 创建 Pull Request

## 许可证

本项目采用 MIT 许可证。详见 [LICENSE](LICENSE) 文件。 