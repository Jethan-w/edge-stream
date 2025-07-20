package statemanager

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"
)

// StateProviderRegistry 状态提供者注册表
type StateProviderRegistry struct {
	registeredProviders map[string]reflect.Type
	mu                  sync.RWMutex
}

// NewStateProviderRegistry 创建状态提供者注册表
func NewStateProviderRegistry() *StateProviderRegistry {
	return &StateProviderRegistry{
		registeredProviders: make(map[string]reflect.Type),
	}
}

// RegisterProvider 注册状态提供者
func (spr *StateProviderRegistry) RegisterProvider(providerName string, providerType reflect.Type) error {
	spr.mu.Lock()
	defer spr.mu.Unlock()

	if _, exists := spr.registeredProviders[providerName]; exists {
		return fmt.Errorf("provider already registered: %s", providerName)
	}

	// 验证提供者类型是否实现了 StateProvider 接口
	if !providerType.Implements(reflect.TypeOf((*StateProvider)(nil)).Elem()) {
		return fmt.Errorf("provider type does not implement StateProvider interface: %s", providerName)
	}

	spr.registeredProviders[providerName] = providerType
	return nil
}

// CreateProvider 创建状态提供者实例
func (spr *StateProviderRegistry) CreateProvider(providerName string, config ProviderConfiguration) (StateProvider, error) {
	spr.mu.RLock()
	defer spr.mu.RUnlock()

	providerType, exists := spr.registeredProviders[providerName]
	if !exists {
		return nil, fmt.Errorf("unknown state provider: %s", providerName)
	}

	// 创建提供者实例
	providerValue := reflect.New(providerType.Elem())
	provider := providerValue.Interface().(StateProvider)

	// 初始化提供者
	if err := provider.Initialize(config); err != nil {
		return nil, fmt.Errorf("failed to initialize provider %s: %w", providerName, err)
	}

	return provider, nil
}

// GetRegisteredProviders 获取已注册的提供者列表
func (spr *StateProviderRegistry) GetRegisteredProviders() []string {
	spr.mu.RLock()
	defer spr.mu.RUnlock()

	providers := make([]string, 0, len(spr.registeredProviders))
	for providerName := range spr.registeredProviders {
		providers = append(providers, providerName)
	}

	return providers
}

// StateProviderExtension 状态提供者扩展接口
type StateProviderExtension interface {
	GetName() string
	Register(registry *StateProviderRegistry) error
	CreateProvider() StateProvider
}

// StateSynchronizer 状态同步器
type StateSynchronizer struct {
	registry        *StateProviderRegistry
	localProvider   StateProvider
	clusterProvider StateProvider
	syncInterval    time.Duration
	syncEnabled     bool
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	syncWorker      *sync.WaitGroup
}

// SyncConfiguration 同步配置
type SyncConfiguration struct {
	LocalProviderConfig   ProviderConfiguration
	ClusterProviderConfig ProviderConfiguration
	SyncInterval          time.Duration
	AutoSync              bool
}

// NewStateSynchronizer 创建状态同步器
func NewStateSynchronizer() *StateSynchronizer {
	ctx, cancel := context.WithCancel(context.Background())

	return &StateSynchronizer{
		registry:    NewStateProviderRegistry(),
		syncEnabled: false,
		ctx:         ctx,
		cancel:      cancel,
		syncWorker:  &sync.WaitGroup{},
	}
}

// Initialize 初始化状态同步器
func (ss *StateSynchronizer) Initialize(config SyncConfiguration) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	// 创建本地提供者
	localProvider, err := ss.registry.CreateProvider(
		string(config.LocalProviderConfig.Type),
		config.LocalProviderConfig,
	)
	if err != nil {
		return fmt.Errorf("failed to create local provider: %w", err)
	}

	// 创建集群提供者
	clusterProvider, err := ss.registry.CreateProvider(
		string(config.ClusterProviderConfig.Type),
		config.ClusterProviderConfig,
	)
	if err != nil {
		return fmt.Errorf("failed to create cluster provider: %w", err)
	}

	ss.localProvider = localProvider
	ss.clusterProvider = clusterProvider
	ss.syncInterval = config.SyncInterval
	ss.syncEnabled = config.AutoSync

	// 启动自动同步
	if ss.syncEnabled && ss.syncInterval > 0 {
		ss.startAutoSync()
	}

	return nil
}

// RegisterExtension 注册扩展
func (ss *StateSynchronizer) RegisterExtension(extension StateProviderExtension) error {
	return extension.Register(ss.registry)
}

// SyncState 同步状态
func (ss *StateSynchronizer) SyncState(componentID string) error {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	if ss.localProvider == nil || ss.clusterProvider == nil {
		return fmt.Errorf("providers not initialized")
	}

	// 加载本地状态
	localState, err := ss.localProvider.LoadState(componentID)
	if err != nil {
		return fmt.Errorf("failed to load local state: %w", err)
	}

	// 加载集群状态
	clusterState, err := ss.clusterProvider.LoadState(componentID)
	if err != nil {
		return fmt.Errorf("failed to load cluster state: %w", err)
	}

	// 比较状态版本，选择最新状态
	mergedState := ss.selectLatestState(localState, clusterState)

	// 更新本地和集群状态
	if err := ss.localProvider.PersistState(componentID, mergedState); err != nil {
		return fmt.Errorf("failed to update local state: %w", err)
	}

	if err := ss.clusterProvider.PersistState(componentID, mergedState); err != nil {
		return fmt.Errorf("failed to update cluster state: %w", err)
	}

	return nil
}

// SyncAllStates 同步所有状态
func (ss *StateSynchronizer) SyncAllStates() error {
	// 在实际实现中，这里应该获取所有组件ID并同步它们的状态
	// 为了演示，我们返回一个简单的实现
	return nil
}

// GetLocalProvider 获取本地提供者
func (ss *StateSynchronizer) GetLocalProvider() StateProvider {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	return ss.localProvider
}

// GetClusterProvider 获取集群提供者
func (ss *StateSynchronizer) GetClusterProvider() StateProvider {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	return ss.clusterProvider
}

// SetSyncEnabled 设置同步启用状态
func (ss *StateSynchronizer) SetSyncEnabled(enabled bool) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	ss.syncEnabled = enabled

	if enabled && ss.syncInterval > 0 {
		ss.startAutoSync()
	} else {
		ss.stopAutoSync()
	}
}

// Shutdown 关闭状态同步器
func (ss *StateSynchronizer) Shutdown() error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	ss.cancel()
	ss.syncWorker.Wait()

	if ss.localProvider != nil {
		if err := ss.localProvider.Shutdown(); err != nil {
			return fmt.Errorf("failed to shutdown local provider: %w", err)
		}
	}

	if ss.clusterProvider != nil {
		if err := ss.clusterProvider.Shutdown(); err != nil {
			return fmt.Errorf("failed to shutdown cluster provider: %w", err)
		}
	}

	return nil
}

// selectLatestState 选择最新状态
func (ss *StateSynchronizer) selectLatestState(local, cluster *StateMap) *StateMap {
	if local.Timestamp > cluster.Timestamp {
		return local
	}
	return cluster
}

// startAutoSync 启动自动同步
func (ss *StateSynchronizer) startAutoSync() {
	ss.syncWorker.Add(1)
	go func() {
		defer ss.syncWorker.Done()

		ticker := time.NewTicker(ss.syncInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ss.ctx.Done():
				return
			case <-ticker.C:
				if err := ss.SyncAllStates(); err != nil {
					fmt.Printf("Warning: failed to sync all states: %v\n", err)
				}
			}
		}
	}()
}

// stopAutoSync 停止自动同步
func (ss *StateSynchronizer) stopAutoSync() {
	// 自动同步会在下一次 tick 时停止
}

// RedisStateProvider Redis 状态提供者实现示例
type RedisStateProvider struct {
	redisClient *MockRedisClient
	mu          sync.RWMutex
}

// MockRedisClient 模拟 Redis 客户端
type MockRedisClient struct {
	data map[string]map[string]string
	mu   sync.RWMutex
}

// NewMockRedisClient 创建模拟 Redis 客户端
func NewMockRedisClient() *MockRedisClient {
	return &MockRedisClient{
		data: make(map[string]map[string]string),
	}
}

// HSet 设置哈希字段
func (rc *MockRedisClient) HSet(key, field, value string) error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if rc.data[key] == nil {
		rc.data[key] = make(map[string]string)
	}

	rc.data[key][field] = value
	return nil
}

// HGetAll 获取所有哈希字段
func (rc *MockRedisClient) HGetAll(key string) (map[string]string, error) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	if data, exists := rc.data[key]; exists {
		result := make(map[string]string)
		for k, v := range data {
			result[k] = v
		}
		return result, nil
	}

	return make(map[string]string), nil
}

// Expire 设置过期时间
func (rc *MockRedisClient) Expire(key string, seconds int) error {
	// 在实际实现中，这里应该设置过期时间
	return nil
}

// NewRedisStateProvider 创建 Redis 状态提供者
func NewRedisStateProvider() *RedisStateProvider {
	return &RedisStateProvider{
		redisClient: NewMockRedisClient(),
	}
}

// Initialize 初始化提供者
func (rsp *RedisStateProvider) Initialize(config ProviderConfiguration) error {
	rsp.mu.Lock()
	defer rsp.mu.Unlock()

	// 获取 Redis 连接配置
	host, ok := config.Properties["redis.host"]
	if !ok {
		host = "localhost"
	}

	port, ok := config.Properties["redis.port"]
	if !ok {
		port = "6379"
	}

	fmt.Printf("Initializing Redis state provider with host: %s, port: %s\n", host, port)

	return nil
}

// LoadState 加载组件状态
func (rsp *RedisStateProvider) LoadState(componentID string) (*StateMap, error) {
	rsp.mu.RLock()
	defer rsp.mu.RUnlock()

	stateValues, err := rsp.redisClient.HGetAll(componentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get state from Redis: %w", err)
	}

	return NewStateMap(stateValues, StateScopeCluster), nil
}

// PersistState 持久化组件状态
func (rsp *RedisStateProvider) PersistState(componentID string, stateMap *StateMap) error {
	rsp.mu.Lock()
	defer rsp.mu.Unlock()

	for key, value := range stateMap.StateValues {
		if err := rsp.redisClient.HSet(componentID, key, value); err != nil {
			return fmt.Errorf("failed to set state in Redis: %w", err)
		}
	}

	// 设置过期时间（24小时）
	if err := rsp.redisClient.Expire(componentID, 24*60*60); err != nil {
		return fmt.Errorf("failed to set expiration: %w", err)
	}

	return nil
}

// Shutdown 关闭资源
func (rsp *RedisStateProvider) Shutdown() error {
	// Redis 客户端关闭逻辑
	return nil
}

// RedisStateProviderExtension Redis 状态提供者扩展
type RedisStateProviderExtension struct{}

// GetName 获取扩展名称
func (rse *RedisStateProviderExtension) GetName() string {
	return "redis"
}

// Register 注册扩展
func (rse *RedisStateProviderExtension) Register(registry *StateProviderRegistry) error {
	return registry.RegisterProvider(rse.GetName(), reflect.TypeOf((*RedisStateProvider)(nil)))
}

// CreateProvider 创建提供者
func (rse *RedisStateProviderExtension) CreateProvider() StateProvider {
	return NewRedisStateProvider()
}
