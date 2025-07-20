# ConnectorRegistry 模块

## 模块概述

ConnectorRegistry 是 Apache NiFi 系统的"组件商店"与"版本控制中枢"，负责管理 Processor、ControllerService 等组件的生命周期。它是整个数据流处理系统中确保组件可扩展、可管理和可追溯的关键模块，提供了高度灵活和安全的组件注册、发现和版本管理能力。

## 核心功能

### 1. 组件注册与发现
- **组件描述符管理**: 提供统一的组件描述符接口，支持NAR包形式的连接器描述符
- **组件注册服务**: 支持组件的注册、注销、查询和管理
- **组件搜索**: 支持基于关键词的组件搜索，包括名称、描述、标签、作者等字段
- **类型过滤**: 支持按组件类型（Processor、ControllerService等）进行过滤

### 2. NAR包管理
- **Bundle仓库**: 提供文件系统形式的NAR包仓库实现
- **依赖解析**: 自动解析和下载NAR包依赖
- **版本管理**: 支持NAR包的版本控制和兼容性检查
- **结构验证**: 验证NAR包的标准目录结构

### 3. 版本控制
- **版本策略**: 支持语义化版本、基于哈希、基于时间戳等多种版本控制策略
- **版本历史**: 维护组件的完整版本历史
- **版本回滚**: 支持版本回滚和恢复
- **兼容性检查**: 自动检查版本兼容性

### 4. 扩展映射
- **映射提供者**: 基于JSON的扩展映射存储和管理
- **动态发现**: 支持组件的动态发现和加载
- **类型映射**: 维护组件类型与NAR包的映射关系

### 5. 安全验证
- **组件验证**: 验证组件的安全性和完整性
- **签名验证**: 支持PGP数字签名验证
- **漏洞扫描**: 扫描组件中的安全漏洞
- **安全策略**: 可配置的安全策略管理

### 6. 社区组件集成
- **仓库管理**: 支持多个社区组件仓库
- **组件搜索**: 从社区仓库搜索组件
- **自动下载**: 自动下载和安装社区组件
- **GitHub集成**: 集成GitHub作为组件仓库

## 架构设计

### 目录结构
```
internal/ConnectorRegistry/
├── connector_registry.go          # 主接口和实现
├── NARPackageManager/
│   └── bundle_repository.go       # NAR包管理
├── VersionController/
│   └── version_controller.go      # 版本控制
├── ComponentRegistrar/
│   └── extension_mapping.go       # 扩展映射
├── SecurityValidator/
│   └── security_validator.go      # 安全验证
├── examples.go                    # 使用示例
└── README.md                      # 文档
```

### 核心接口

#### ConnectorRegistry
```go
type ConnectorRegistry interface {
    RegisterConnector(descriptor ConnectorDescriptor) error
    UnregisterConnector(identifier string) error
    GetConnector(identifier string) (ConnectorDescriptor, error)
    ListConnectors() []ConnectorDescriptor
    ListConnectorsByType(componentType ComponentType) []ConnectorDescriptor
    SearchConnectors(keyword string) []ConnectorDescriptor
    GetConnectorVersions(identifier string) ([]string, error)
    UpdateConnector(descriptor ConnectorDescriptor) error
}
```

#### ConnectorDescriptor
```go
type ConnectorDescriptor interface {
    GetIdentifier() string
    GetName() string
    GetType() ComponentType
    GetVersion() string
    GetDescription() string
    GetAuthor() string
    GetTags() []string
    GetMetadata() map[string]string
}
```

#### BundleRepository
```go
type BundleRepository interface {
    DeployBundle(bundle *Bundle) error
    UndeployBundle(bundleID, version string) error
    GetBundle(bundleID, version string) (*Bundle, error)
    ListBundles() ([]*Bundle, error)
    GetBundlePath(bundleID, version string) (string, error)
}
```

#### VersionController
```go
type VersionController interface {
    CreateVersion(componentID string, snapshot *FlowSnapshot) error
    UpdateVersion(componentID, version string, snapshot *FlowSnapshot) error
    GetVersion(componentID, version string) (*FlowSnapshot, error)
    ListVersions(componentID string) ([]string, error)
    RollbackToVersion(componentID, version string) error
    DeleteVersion(componentID, version string) error
}
```

## 使用方法

### 基本使用

```go
package main

import (
    "fmt"
    "log"
    "github.com/crazy/edge-stream/internal/ConnectorRegistry"
)

func main() {
    // 创建连接器注册表
    registry := ConnectorRegistry.NewStandardConnectorRegistry()
    
    // 创建Bundle
    bundle := &ConnectorRegistry.NARPackageManager.Bundle{
        ID:       "org.apache.nifi:nifi-standard-nar",
        Group:    "org.apache.nifi",
        Artifact: "nifi-standard-nar",
        Version:  "1.20.0",
        File:     "nifi-standard-nar-1.20.0.nar",
    }
    
    // 创建连接器描述符
    descriptor := ConnectorRegistry.NewNarBundleConnectorDescriptor(
        bundle, 
        "NiFi Standard Processors", 
        ConnectorRegistry.ComponentTypeProcessor,
    )
    descriptor.SetDescription("Apache NiFi 标准处理器集合")
    descriptor.SetAuthor("Apache NiFi Team")
    
    // 注册连接器
    err := registry.RegisterConnector(descriptor)
    if err != nil {
        log.Fatalf("注册连接器失败: %v", err)
    }
    
    // 列出所有连接器
    connectors := registry.ListConnectors()
    fmt.Printf("已注册连接器数量: %d\n", len(connectors))
}
```

### NAR包管理

```go
// 创建Bundle仓库
repository := ConnectorRegistry.NARPackageManager.NewFileSystemBundleRepository()
repository.SetRepositoryPath("custom-bundles")

// 创建Bundle
bundle := &ConnectorRegistry.NARPackageManager.Bundle{
    ID:       "com.example:custom-processor-nar",
    Group:    "com.example",
    Artifact: "custom-processor-nar",
    Version:  "1.0.0",
    File:     "custom-processor-nar-1.0.0.nar",
}

// 部署Bundle
err := repository.DeployBundle(bundle)
if err != nil {
    log.Fatalf("部署Bundle失败: %v", err)
}

// 创建依赖解析器
resolver := ConnectorRegistry.NARPackageManager.NewBundleDependencyResolver(repository)
err = resolver.ResolveDependencies(bundle)
if err != nil {
    log.Printf("解析依赖失败: %v", err)
}
```

### 版本控制

```go
// 创建版本控制器
controller := ConnectorRegistry.VersionController.NewStandardVersionController()

// 创建版本管理器
manager := ConnectorRegistry.VersionController.NewVersionManager(controller)

// 设置版本控制策略
manager.SetStrategy(ConnectorRegistry.VersionController.VersionControlStrategySemantic)

// 创建流程快照数据
snapshotData := map[string]interface{}{
    "processors": []map[string]interface{}{
        {
            "id":   "processor-1",
            "type": "org.apache.nifi.processors.standard.GenerateFlowFile",
            "properties": map[string]string{
                "File Size": "1KB",
                "Batch Size": "1",
            },
        },
    },
}

// 创建版本
err := manager.CreateVersionWithStrategy(
    "flow-1",
    snapshotData,
    "admin",
    "初始版本",
)
if err != nil {
    log.Fatalf("创建版本失败: %v", err)
}

// 列出版本
versions, err := controller.ListVersions("flow-1")
if err != nil {
    log.Fatalf("列出版本失败: %v", err)
}

fmt.Printf("版本数量: %d\n", len(versions))
```

### 扩展映射

```go
// 创建扩展注册表
extensionRegistry := ConnectorRegistry.ComponentRegistrar.NewExtensionRegistry()

// 注册扩展
err := extensionRegistry.RegisterExtension(
    "PROCESSOR",
    "org.apache.nifi:nifi-standard-nar",
    "org.apache.nifi.processors.standard.GenerateFlowFile",
)
if err != nil {
    log.Fatalf("注册扩展失败: %v", err)
}

// 获取扩展
extensions, err := extensionRegistry.GetExtensions("PROCESSOR")
if err != nil {
    log.Fatalf("获取扩展失败: %v", err)
}

fmt.Printf("处理器扩展数量: %d\n", len(extensions))
```

### 安全验证

```go
// 创建安全管理器
securityManager := ConnectorRegistry.SecurityValidator.NewSecurityManager()

// 创建安全策略
policy := ConnectorRegistry.SecurityValidator.NewSecurityPolicy()
policy.AddAllowedRepository("github.com/apache/nifi")
policy.AddBlockedComponent("malicious-component")

// 设置安全策略
securityManager.SetSecurityPolicy(policy)

// 创建模拟Bundle
bundle := &ConnectorRegistry.NARPackageManager.Bundle{
    ID:       "org.apache.nifi:nifi-standard-nar",
    Group:    "org.apache.nifi",
    Artifact: "nifi-standard-nar",
    Version:  "1.20.0",
    File:     "nifi-standard-nar-1.20.0.nar",
}

// 使用策略验证组件
err := securityManager.ValidateComponentWithPolicy(bundle, "github.com/apache/nifi")
if err != nil {
    log.Fatalf("组件验证失败: %v", err)
}

fmt.Println("组件验证通过")
```

### 社区组件集成

```go
// 创建社区组件注册表
communityRegistry := ConnectorRegistry.SecurityValidator.NewCommunityComponentRegistry()

// 添加GitHub仓库
githubRepo := ConnectorRegistry.SecurityValidator.NewGitHubComponentRepository(
    "https://api.github.com",
    "your-github-token",
)
communityRegistry.AddRepository(githubRepo)

// 搜索组件
results, err := communityRegistry.SearchComponents("kafka")
if err != nil {
    log.Fatalf("搜索组件失败: %v", err)
}

fmt.Printf("找到 %d 个Kafka相关组件\n", len(results))

// 下载并安装组件（如果有结果）
if len(results) > 0 {
    err = communityRegistry.DownloadAndInstallComponent(results[0])
    if err != nil {
        log.Fatalf("下载并安装组件失败: %v", err)
    }
    
    fmt.Println("组件下载并安装成功")
}
```

## 配置说明

### Bundle仓库配置
```go
// 设置仓库路径
repository.SetRepositoryPath("/path/to/bundles")

// 配置依赖解析策略
resolver := NewBundleDependencyResolver(repository)
```

### 版本控制配置
```go
// 设置版本控制策略
manager.SetStrategy(VersionControlStrategySemantic) // 语义化版本
manager.SetStrategy(VersionControlStrategyHash)     // 基于哈希
manager.SetStrategy(VersionControlStrategyTimestamp) // 基于时间戳
```

### 安全策略配置
```go
policy := NewSecurityPolicy()
policy.AllowUnsignedComponents = false
policy.RequireSignature = true
policy.VulnerabilityThreshold = "HIGH"
policy.AddAllowedRepository("github.com/apache/nifi")
policy.AddBlockedComponent("malicious-component")
```

## 性能指标

| 指标           | 目标值        | 说明                   |
|---------------|---------------|------------------------|
| 组件注册延迟    | <50ms         | 组件注册耗时           |
| 版本控制开销    | <100ms        | 版本管理耗时           |
| 依赖解析性能    | <200ms        | 依赖包解析耗时         |
| 组件发现速度    | <300ms        | 组件动态发现耗时       |
| 安全验证延迟    | <500ms        | 安全验证耗时           |

## 扩展点

### 自定义Bundle仓库
```go
type CustomBundleRepository struct {
    // 实现BundleRepository接口
}

func (cbr *CustomBundleRepository) DeployBundle(bundle *Bundle) error {
    // 自定义部署逻辑
}

func (cbr *CustomBundleRepository) GetBundle(bundleID, version string) (*Bundle, error) {
    // 自定义获取逻辑
}
```

### 自定义安全验证器
```go
type CustomSecurityValidator struct {
    // 实现SecurityValidator接口
}

func (csv *CustomSecurityValidator) ValidateComponent(bundle interface{}) error {
    // 自定义验证逻辑
}

func (csv *CustomSecurityValidator) ScanVulnerabilities(bundle interface{}) ([]*SecurityVulnerability, error) {
    // 自定义漏洞扫描逻辑
}
```

### 自定义版本控制器
```go
type CustomVersionController struct {
    // 实现VersionController接口
}

func (cvc *CustomVersionController) CreateVersion(componentID string, snapshot *FlowSnapshot) error {
    // 自定义版本创建逻辑
}

func (cvc *CustomVersionController) ListVersions(componentID string) ([]string, error) {
    // 自定义版本列表逻辑
}
```

## 最佳实践

### 1. 组件注册
- 使用有意义的组件标识符
- 提供详细的组件描述和标签
- 设置合适的组件作者信息
- 添加相关的元数据

### 2. 版本管理
- 使用语义化版本号
- 为每个版本提供清晰的描述
- 定期清理过期的版本
- 验证版本兼容性

### 3. 安全验证
- 启用数字签名验证
- 配置合适的安全策略
- 定期更新漏洞数据库
- 监控安全事件

### 4. 依赖管理
- 明确指定依赖版本
- 避免循环依赖
- 定期更新依赖
- 验证依赖兼容性

### 5. 性能优化
- 使用连接池管理资源
- 实现缓存机制
- 异步处理耗时操作
- 监控性能指标

## 故障排除

### 常见问题

1. **组件注册失败**
   - 检查组件标识符是否重复
   - 验证Bundle文件是否存在
   - 确认依赖是否已安装

2. **版本控制错误**
   - 检查版本号格式
   - 验证版本兼容性
   - 确认版本历史完整性

3. **安全验证失败**
   - 检查数字签名
   - 更新漏洞数据库
   - 验证安全策略配置

4. **依赖解析失败**
   - 检查网络连接
   - 验证仓库配置
   - 确认依赖版本兼容性

### 调试技巧

1. **启用详细日志**
   ```go
   // 设置日志级别
   log.SetLevel(log.DebugLevel)
   ```

2. **检查组件状态**
   ```go
   // 获取已注册组件数量
   count := registry.GetRegisteredConnectorsCount()
   fmt.Printf("已注册组件数量: %d\n", count)
   ```

3. **验证Bundle结构**
   ```go
   structure := NewNarBundleStructure()
   err := structure.ValidateStructure(bundlePath)
   if err != nil {
       log.Printf("Bundle结构验证失败: %v", err)
   }
   ```

## 版本历史

### v1.0.0 (2024-01-01)
- 初始版本发布
- 支持基本的组件注册和发现
- 实现NAR包管理功能
- 提供版本控制能力
- 集成安全验证机制
- 支持社区组件集成

## 贡献指南

1. Fork 项目
2. 创建功能分支
3. 提交更改
4. 推送到分支
5. 创建 Pull Request

## 许可证

本项目采用 Apache License 2.0 许可证。 