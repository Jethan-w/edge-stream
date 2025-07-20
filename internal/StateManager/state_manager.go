package statemanager

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// StateProvider 定义状态存储和检索的标准接口
type StateProvider interface {
	// Initialize 初始化提供者
	Initialize(config ProviderConfiguration) error

	// LoadState 加载组件状态
	LoadState(componentID string) (*StateMap, error)

	// PersistState 持久化组件状态
	PersistState(componentID string, stateMap *StateMap) error

	// Shutdown 关闭资源
	Shutdown() error
}

// StateMap 标准状态映射实现
type StateMap struct {
	StateValues map[string]string `json:"state_values"`
	Timestamp   int64             `json:"timestamp"`
	Version     int               `json:"version"`
	Scope       StateScope        `json:"scope"`
}

// NewStateMap 创建新的状态映射
func NewStateMap(values map[string]string, scope StateScope) *StateMap {
	if values == nil {
		values = make(map[string]string)
	}

	return &StateMap{
		StateValues: values,
		Timestamp:   time.Now().UnixMilli(),
		Version:     1,
		Scope:       scope,
	}
}

// GetValue 获取状态值
func (sm *StateMap) GetValue(key string) string {
	return sm.StateValues[key]
}

// SetValue 设置状态值
func (sm *StateMap) SetValue(key, value string) {
	sm.StateValues[key] = value
	sm.Timestamp = time.Now().UnixMilli()
	sm.Version++
}

// ToMap 转换为 map
func (sm *StateMap) ToMap() map[string]string {
	result := make(map[string]string)
	for k, v := range sm.StateValues {
		result[k] = v
	}
	return result
}

// StateScope 定义状态的作用域和可见性
type StateScope string

const (
	StateScopeLocal   StateScope = "LOCAL"   // 本地处理器状态
	StateScopeCluster StateScope = "CLUSTER" // 集群共享状态
	StateScopeGlobal  StateScope = "GLOBAL"  // 全局系统状态
)

// ConsistencyLevel 定义状态同步的一致性保证
type ConsistencyLevel string

const (
	ConsistencyLevelEventual      ConsistencyLevel = "EVENTUAL"       // 最终一致性
	ConsistencyLevelStrong        ConsistencyLevel = "STRONG"         // 强一致性
	ConsistencyLevelReadCommitted ConsistencyLevel = "READ_COMMITTED" // 读已提交
	ConsistencyLevelSerializable  ConsistencyLevel = "SERIALIZABLE"   // 串行化
)

// StateUpdateStrategy 定义状态更新的原子性和幂等性
type StateUpdateStrategy string

const (
	StateUpdateStrategyReplace     StateUpdateStrategy = "REPLACE"     // 完全替换
	StateUpdateStrategyMerge       StateUpdateStrategy = "MERGE"       // 合并更新
	StateUpdateStrategyConditional StateUpdateStrategy = "CONDITIONAL" // 条件更新
)

// ProviderConfiguration 提供者配置
type ProviderConfiguration struct {
	Properties map[string]string `json:"properties"`
	Type       StateProviderType `json:"type"`
}

// StateProviderType 状态提供者类型
type StateProviderType string

const (
	StateProviderTypeLocalFileSystem StateProviderType = "LOCAL_FILE_SYSTEM" // 本地文件系统
	StateProviderTypeZooKeeper       StateProviderType = "ZOOKEEPER"         // ZooKeeper 分布式存储
	StateProviderTypeRedis           StateProviderType = "REDIS"             // Redis 缓存
	StateProviderTypeJDBC            StateProviderType = "JDBC"              // 关系型数据库
	StateProviderTypeCustom          StateProviderType = "CUSTOM"            // 自定义提供者
)

// StateUpdate 状态更新函数类型
type StateUpdate func(*StateMap) *StateMap

// StateManager 状态管理器接口
type StateManager interface {
	// Initialize 初始化状态管理器
	Initialize(config ManagerConfiguration) error

	// GetState 获取组件状态
	GetState(componentID string) (*StateMap, error)

	// SetState 设置组件状态
	SetState(componentID string, stateMap *StateMap) error

	// UpdateState 更新组件状态
	UpdateState(componentID string, update StateUpdate) error

	// UpdateStateTransactionally 事务性更新状态
	UpdateStateTransactionally(componentID string, update StateUpdate) error

	// SyncState 同步状态
	SyncState(componentID string) error

	// CreateCheckpoint 创建检查点
	CreateCheckpoint(componentID string) error

	// RestoreCheckpoint 恢复检查点
	RestoreCheckpoint(componentID string) error

	// Shutdown 关闭状态管理器
	Shutdown() error
}

// ManagerConfiguration 管理器配置
type ManagerConfiguration struct {
	LocalProvider      StateProvider
	ClusterProvider    StateProvider
	ConsistencyLevel   ConsistencyLevel
	CheckpointInterval time.Duration
	SyncInterval       time.Duration
}

// StandardStateManager 标准状态管理器实现
type StandardStateManager struct {
	Config          ManagerConfiguration
	LocalProvider   StateProvider
	ClusterProvider StateProvider
	ConsistencyMgr  *StateConsistencyManager
	CheckpointMgr   *StateCheckpointManager
	WalManager      *WriteAheadLogManager
	LockManager     *DistributedStateLock
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
}

// NewStandardStateManager 创建标准状态管理器
func NewStandardStateManager() *StandardStateManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &StandardStateManager{
		ctx:    ctx,
		cancel: cancel,
	}
}

// Initialize 初始化状态管理器
func (sm *StandardStateManager) Initialize(config ManagerConfiguration) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.Config = config
	sm.LocalProvider = config.LocalProvider
	sm.ClusterProvider = config.ClusterProvider

	// 初始化一致性管理器
	sm.ConsistencyMgr = NewStateConsistencyManager(config.LocalProvider, config.ClusterProvider)

	// 初始化检查点管理器
	sm.CheckpointMgr = NewStateCheckpointManager()

	// 初始化预写日志管理器
	sm.WalManager = NewWriteAheadLogManager()

	// 初始化分布式锁管理器
	sm.LockManager = NewDistributedStateLock()

	// 启动后台同步任务
	if config.SyncInterval > 0 {
		go sm.startSyncTask()
	}

	// 启动检查点任务
	if config.CheckpointInterval > 0 {
		go sm.startCheckpointTask()
	}

	return nil
}

// GetState 获取组件状态
func (sm *StandardStateManager) GetState(componentID string) (*StateMap, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// 根据一致性级别选择状态源
	switch sm.Config.ConsistencyLevel {
	case ConsistencyLevelStrong:
		// 强一致性：从集群获取最新状态
		return sm.ClusterProvider.LoadState(componentID)
	case ConsistencyLevelReadCommitted:
		// 读已提交：从本地获取，但确保已同步
		state, err := sm.LocalProvider.LoadState(componentID)
		if err != nil {
			return nil, err
		}
		// 异步同步到集群
		go sm.SyncState(componentID)
		return state, nil
	default:
		// 最终一致性：从本地获取
		return sm.LocalProvider.LoadState(componentID)
	}
}

// SetState 设置组件状态
func (sm *StandardStateManager) SetState(componentID string, stateMap *StateMap) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 记录预写日志
	if err := sm.WalManager.LogStateUpdate(componentID, nil, stateMap); err != nil {
		return fmt.Errorf("failed to log state update: %w", err)
	}

	// 持久化到本地
	if err := sm.LocalProvider.PersistState(componentID, stateMap); err != nil {
		return fmt.Errorf("failed to persist state locally: %w", err)
	}

	// 根据一致性级别决定是否同步到集群
	if sm.Config.ConsistencyLevel == ConsistencyLevelStrong {
		if err := sm.ClusterProvider.PersistState(componentID, stateMap); err != nil {
			return fmt.Errorf("failed to persist state to cluster: %w", err)
		}
	}

	return nil
}

// UpdateState 更新组件状态
func (sm *StandardStateManager) UpdateState(componentID string, update StateUpdate) error {
	// 加载当前状态
	currentState, err := sm.GetState(componentID)
	if err != nil {
		return fmt.Errorf("failed to load current state: %w", err)
	}

	// 应用更新
	newState := update(currentState)

	// 设置新状态
	return sm.SetState(componentID, newState)
}

// UpdateStateTransactionally 事务性更新状态
func (sm *StandardStateManager) UpdateStateTransactionally(componentID string, update StateUpdate) error {
	// 获取分布式锁
	if err := sm.LockManager.AcquireLock(componentID); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer sm.LockManager.ReleaseLock(componentID)

	// 加载当前状态
	currentState, err := sm.LocalProvider.LoadState(componentID)
	if err != nil {
		return fmt.Errorf("failed to load current state: %w", err)
	}

	// 记录预写日志
	if err := sm.WalManager.LogStateUpdate(componentID, currentState, nil); err != nil {
		return fmt.Errorf("failed to log state update: %w", err)
	}

	// 应用更新
	newState := update(currentState)

	// 持久化新状态
	if err := sm.LocalProvider.PersistState(componentID, newState); err != nil {
		return fmt.Errorf("failed to persist new state: %w", err)
	}

	// 同步到集群
	if sm.ClusterProvider != nil {
		if err := sm.ClusterProvider.PersistState(componentID, newState); err != nil {
			return fmt.Errorf("failed to persist state to cluster: %w", err)
		}
	}

	// 完成预写日志
	if err := sm.WalManager.LogStateUpdate(componentID, currentState, newState); err != nil {
		return fmt.Errorf("failed to complete state update log: %w", err)
	}

	return nil
}

// SyncState 同步状态
func (sm *StandardStateManager) SyncState(componentID string) error {
	return sm.ConsistencyMgr.SyncState(componentID)
}

// CreateCheckpoint 创建检查点
func (sm *StandardStateManager) CreateCheckpoint(componentID string) error {
	state, err := sm.GetState(componentID)
	if err != nil {
		return fmt.Errorf("failed to get state for checkpoint: %w", err)
	}

	return sm.CheckpointMgr.CreateCheckpoint(componentID, state)
}

// RestoreCheckpoint 恢复检查点
func (sm *StandardStateManager) RestoreCheckpoint(componentID string) error {
	state, err := sm.CheckpointMgr.RestoreLatestCheckpoint(componentID)
	if err != nil {
		return fmt.Errorf("failed to restore checkpoint: %w", err)
	}

	return sm.SetState(componentID, state)
}

// Shutdown 关闭状态管理器
func (sm *StandardStateManager) Shutdown() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.cancel()

	if sm.LocalProvider != nil {
		if err := sm.LocalProvider.Shutdown(); err != nil {
			return fmt.Errorf("failed to shutdown local provider: %w", err)
		}
	}

	if sm.ClusterProvider != nil {
		if err := sm.ClusterProvider.Shutdown(); err != nil {
			return fmt.Errorf("failed to shutdown cluster provider: %w", err)
		}
	}

	return nil
}

// startSyncTask 启动同步任务
func (sm *StandardStateManager) startSyncTask() {
	ticker := time.NewTicker(sm.Config.SyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-sm.ctx.Done():
			return
		case <-ticker.C:
			// 执行同步逻辑
			// 这里可以实现定期同步所有组件状态的逻辑
		}
	}
}

// startCheckpointTask 启动检查点任务
func (sm *StandardStateManager) startCheckpointTask() {
	ticker := time.NewTicker(sm.Config.CheckpointInterval)
	defer ticker.Stop()

	for {
		select {
		case <-sm.ctx.Done():
			return
		case <-ticker.C:
			// 执行检查点创建逻辑
			// 这里可以实现定期创建检查点的逻辑
		}
	}
}

// StateMetadata 记录状态的详细信息和生命周期
type StateMetadata struct {
	ComponentID           string            `json:"component_id"`
	CreationTimestamp     int64             `json:"creation_timestamp"`
	LastModifiedTimestamp int64             `json:"last_modified_timestamp"`
	Scope                 StateScope        `json:"scope"`
	Version               int               `json:"version"`
	AdditionalMetadata    map[string]string `json:"additional_metadata"`
}

// NewStateMetadata 创建新的状态元数据
func NewStateMetadata(componentID string, scope StateScope) *StateMetadata {
	now := time.Now().UnixMilli()
	return &StateMetadata{
		ComponentID:           componentID,
		CreationTimestamp:     now,
		LastModifiedTimestamp: now,
		Scope:                 scope,
		Version:               1,
		AdditionalMetadata:    make(map[string]string),
	}
}

// UpdateMetadata 更新元数据
func (sm *StateMetadata) UpdateMetadata() {
	sm.LastModifiedTimestamp = time.Now().UnixMilli()
	sm.Version++
}

// SerializeStateMap 序列化状态映射
func SerializeStateMap(stateMap *StateMap) ([]byte, error) {
	return json.Marshal(stateMap)
}

// DeserializeStateMap 反序列化状态映射
func DeserializeStateMap(data []byte) (*StateMap, error) {
	var stateMap StateMap
	if err := json.Unmarshal(data, &stateMap); err != nil {
		return nil, err
	}
	return &stateMap, nil
}
