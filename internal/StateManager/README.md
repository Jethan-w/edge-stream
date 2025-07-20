# StateManager 模块

## 概述

StateManager 是 Apache NiFi 实现状态持久化与分布式协作的基础组件，负责管理处理器状态的存储、共享与恢复。它是整个数据流处理系统中确保数据处理一致性和可靠性的关键模块，提供了灵活、高效的状态管理能力。

## 核心特性

### 1. 状态持久化
- **多种存储后端**：支持本地文件系统、ZooKeeper、Redis、关系型数据库等
- **原子性操作**：确保状态更新的原子性和一致性
- **版本控制**：支持状态版本管理和冲突解决

### 2. 分布式状态同步
- **集群一致性**：通过分布式共识算法确保集群中各节点状态一致
- **自动同步**：支持定时自动同步和手动同步
- **冲突解决**：智能的状态合并和冲突解决机制

### 3. 故障恢复
- **检查点机制**：定期创建状态检查点，支持快速恢复
- **预写日志（WAL）**：记录所有状态变更，确保故障后数据不丢失
- **自动恢复**：系统重启后自动从最新检查点恢复

### 4. 事务性更新
- **分布式锁**：确保状态更新的互斥性
- **事务保证**：支持 ACID 事务特性
- **回滚机制**：支持事务回滚和错误恢复

### 5. 可扩展架构
- **插件化设计**：支持自定义状态提供者
- **动态注册**：运行时注册和切换状态提供者
- **配置灵活**：支持多种配置方式和参数调整

## 架构设计

### 核心组件

```
StateManager
├── StateProvider (状态提供者接口)
│   ├── LocalStateProvider (本地状态提供者)
│   ├── DistributedStateProvider (分布式状态提供者)
│   └── CustomStateProvider (自定义状态提供者)
├── StateSynchronizer (状态同步器)
├── StateRecoveryManager (状态恢复管理器)
│   ├── StateCheckpointManager (检查点管理器)
│   └── WriteAheadLogManager (预写日志管理器)
└── StateConsistencyManager (一致性管理器)
```

### 状态范围

- **LOCAL**：本地处理器状态，仅在当前节点有效
- **CLUSTER**：集群共享状态，在集群内所有节点可见
- **GLOBAL**：全局系统状态，在整个系统中共享

### 一致性级别

- **EVENTUAL**：最终一致性，性能最高但一致性保证最弱
- **STRONG**：强一致性，一致性保证最强但性能较低
- **READ_COMMITTED**：读已提交，平衡性能和一致性
- **SERIALIZABLE**：串行化，最高隔离级别

## 快速开始

### 基本使用

```go
package main

import (
    "log"
    "time"
    
    "github.com/edge-stream/internal/StateManager"
    "github.com/edge-stream/internal/StateManager/LocalStateProvider"
)

func main() {
    // 创建状态管理器
    stateManager := statemanager.NewStandardStateManager()
    
    // 创建本地状态提供者
    localProvider := localstateprovider.NewFileSystemStateProvider()
    localConfig := statemanager.ProviderConfiguration{
        Properties: map[string]string{
            "state.directory": "./local_state",
        },
        Type: statemanager.StateProviderTypeLocalFileSystem,
    }
    
    if err := localProvider.Initialize(localConfig); err != nil {
        log.Fatalf("Failed to initialize local provider: %v", err)
    }
    
    // 初始化状态管理器
    managerConfig := statemanager.ManagerConfiguration{
        LocalProvider:     localProvider,
        ConsistencyLevel:  statemanager.ConsistencyLevelEventual,
        CheckpointInterval: 5 * time.Minute,
        SyncInterval:       30 * time.Second,
    }
    
    if err := stateManager.Initialize(managerConfig); err != nil {
        log.Fatalf("Failed to initialize state manager: %v", err)
    }
    
    // 设置状态
    componentID := "processor-001"
    stateMap := statemanager.NewStateMap(map[string]string{
        "counter":    "100",
        "status":     "running",
    }, statemanager.StateScopeLocal)
    
    if err := stateManager.SetState(componentID, stateMap); err != nil {
        log.Fatalf("Failed to set state: %v", err)
    }
    
    // 获取状态
    retrievedState, err := stateManager.GetState(componentID)
    if err != nil {
        log.Fatalf("Failed to get state: %v", err)
    }
    
    log.Printf("Retrieved state: counter=%s, status=%s\n", 
        retrievedState.GetValue("counter"), 
        retrievedState.GetValue("status"))
    
    // 关闭状态管理器
    if err := stateManager.Shutdown(); err != nil {
        log.Fatalf("Failed to shutdown state manager: %v", err)
    }
}
```

### 分布式状态使用

```go
// 创建 ZooKeeper 状态提供者
zkProvider := distributedstateprovider.NewZooKeeperStateProvider()
zkConfig := statemanager.ProviderConfiguration{
    Properties: map[string]string{
        "zookeeper.host":  "localhost:2181",
        "state.root.path": "/nifi/state",
    },
    Type: statemanager.StateProviderTypeZooKeeper,
}

if err := zkProvider.Initialize(zkConfig); err != nil {
    log.Fatalf("Failed to initialize ZooKeeper provider: %v", err)
}

// 使用强一致性
managerConfig := statemanager.ManagerConfiguration{
    LocalProvider:     localProvider,
    ClusterProvider:   zkProvider,
    ConsistencyLevel:  statemanager.ConsistencyLevelStrong,
    CheckpointInterval: 2 * time.Minute,
    SyncInterval:       10 * time.Second,
}
```

### 事务性更新

```go
// 事务性更新：模拟银行转账
transferUpdate := func(currentState *statemanager.StateMap) *statemanager.StateMap {
    currentBalance := currentState.GetValue("balance")
    currentTransactions := currentState.GetValue("transactions")
    
    // 业务逻辑
    newBalance := "950" // 扣除 50
    newTransactions := "1"
    
    currentState.SetValue("balance", newBalance)
    currentState.SetValue("transactions", newTransactions)
    currentState.SetValue("lastTransfer", time.Now().Format(time.RFC3339))
    
    return currentState
}

// 执行事务性更新
if err := stateManager.UpdateStateTransactionally(componentID, transferUpdate); err != nil {
    log.Fatalf("Failed to perform transactional update: %v", err)
}
```

## 配置说明

### 状态提供者配置

#### 本地文件系统提供者
```go
config := statemanager.ProviderConfiguration{
    Properties: map[string]string{
        "state.directory": "./local_state",  // 状态文件存储目录
    },
    Type: statemanager.StateProviderTypeLocalFileSystem,
}
```

#### ZooKeeper 提供者
```go
config := statemanager.ProviderConfiguration{
    Properties: map[string]string{
        "zookeeper.host":    "localhost:2181",  // ZooKeeper 连接地址
        "state.root.path":   "/nifi/state",     // 状态根路径
    },
    Type: statemanager.StateProviderTypeZooKeeper,
}
```

#### Redis 提供者
```go
config := statemanager.ProviderConfiguration{
    Properties: map[string]string{
        "redis.host": "localhost",  // Redis 主机
        "redis.port": "6379",       // Redis 端口
    },
    Type: statemanager.StateProviderTypeRedis,
}
```

### 管理器配置

```go
config := statemanager.ManagerConfiguration{
    LocalProvider:     localProvider,      // 本地状态提供者
    ClusterProvider:   clusterProvider,    // 集群状态提供者
    ConsistencyLevel:  ConsistencyLevelStrong,  // 一致性级别
    CheckpointInterval: 5 * time.Minute,   // 检查点间隔
    SyncInterval:       30 * time.Second,  // 同步间隔
}
```

## 性能指标

| 指标           | 目标值        | 说明                   |
|---------------|---------------|------------------------|
| 状态读取延迟    | <10ms         | 状态读取耗时           |
| 状态写入开销    | <20ms         | 状态持久化耗时         |
| 集群同步延迟    | <50ms         | 状态在集群间同步耗时   |
| 并发状态操作    | 100+          | 支持的最大并发状态操作 |

## 扩展开发

### 自定义状态提供者

```go
type CustomStateProvider struct {
    // 自定义字段
    mu sync.RWMutex
}

func (csp *CustomStateProvider) Initialize(config statemanager.ProviderConfiguration) error {
    // 初始化逻辑
    return nil
}

func (csp *CustomStateProvider) LoadState(componentID string) (*statemanager.StateMap, error) {
    csp.mu.RLock()
    defer csp.mu.RUnlock()
    
    // 从自定义存储加载状态
    stateValues := map[string]string{
        "key1": "value1",
        "key2": "value2",
    }
    
    return statemanager.NewStateMap(stateValues, statemanager.StateScopeLocal), nil
}

func (csp *CustomStateProvider) PersistState(componentID string, stateMap *statemanager.StateMap) error {
    csp.mu.Lock()
    defer csp.mu.Unlock()
    
    // 保存状态到自定义存储
    return nil
}

func (csp *CustomStateProvider) Shutdown() error {
    // 清理资源
    return nil
}
```

### 注册自定义提供者

```go
// 创建状态同步器
synchronizer := statesynchronizer.NewStateSynchronizer()

// 注册自定义扩展
customExtension := &CustomStateProviderExtension{}
if err := synchronizer.RegisterExtension(customExtension); err != nil {
    log.Fatalf("Failed to register custom extension: %v", err)
}
```

## 最佳实践

### 1. 状态设计
- **合理分区**：根据业务逻辑合理划分状态范围
- **版本管理**：为状态添加版本信息，便于升级和迁移
- **序列化优化**：选择合适的序列化格式，平衡性能和兼容性

### 2. 性能优化
- **批量操作**：尽量使用批量状态更新
- **缓存策略**：对频繁访问的状态实施缓存
- **异步处理**：非关键状态更新使用异步处理

### 3. 故障处理
- **定期检查点**：根据业务重要性设置合适的检查点间隔
- **监控告警**：监控状态同步和恢复过程
- **备份策略**：实施多级备份和恢复策略

### 4. 安全考虑
- **访问控制**：实施细粒度的状态访问控制
- **加密存储**：对敏感状态进行加密存储
- **审计日志**：记录所有状态变更操作

## 故障排除

### 常见问题

1. **状态同步失败**
   - 检查网络连接
   - 验证集群配置
   - 查看同步日志

2. **性能问题**
   - 调整一致性级别
   - 优化状态大小
   - 增加并发处理能力

3. **恢复失败**
   - 检查检查点文件完整性
   - 验证 WAL 日志
   - 确认存储空间充足

### 调试工具

```go
// 启用调试日志
log.SetLevel(log.DebugLevel)

// 检查状态提供者状态
if provider, ok := stateManager.GetLocalProvider().(*localstateprovider.FileSystemStateProvider); ok {
    // 检查文件系统状态
}

// 监控同步状态
go func() {
    ticker := time.NewTicker(1 * time.Minute)
    for range ticker.C {
        // 检查同步状态
    }
}()
```

## 版本历史

### v1.0.0
- 初始版本发布
- 支持基本的状态管理功能
- 实现本地文件系统和内存状态提供者

### v1.1.0
- 添加分布式状态支持
- 实现 ZooKeeper 和 Redis 状态提供者
- 增加事务性更新功能

### v1.2.0
- 完善故障恢复机制
- 优化性能指标
- 增加更多配置选项

## 贡献指南

1. Fork 项目
2. 创建功能分支
3. 提交更改
4. 推送到分支
5. 创建 Pull Request

## 许可证

本项目采用 Apache License 2.0 许可证。 