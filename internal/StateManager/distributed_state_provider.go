package statemanager

import (
	"fmt"
	"sync"
	"time"
)

// ZooKeeperStateProvider ZooKeeper 分布式状态提供者实现
type ZooKeeperStateProvider struct {
	zkClient      *MockZooKeeperClient // 在实际实现中，这里应该是真实的 ZooKeeper 客户端
	stateRootPath string
	mu            sync.RWMutex
}

// MockZooKeeperClient 模拟 ZooKeeper 客户端（用于演示）
type MockZooKeeperClient struct {
	nodes map[string][]byte
	mu    sync.RWMutex
}

// NewMockZooKeeperClient 创建模拟 ZooKeeper 客户端
func NewMockZooKeeperClient() *MockZooKeeperClient {
	return &MockZooKeeperClient{
		nodes: make(map[string][]byte),
	}
}

// Create 创建节点
func (zk *MockZooKeeperClient) Create(path string, data []byte, mode CreateMode) error {
	zk.mu.Lock()
	defer zk.mu.Unlock()

	if _, exists := zk.nodes[path]; exists {
		return fmt.Errorf("node already exists: %s", path)
	}

	zk.nodes[path] = data
	return nil
}

// GetData 获取节点数据
func (zk *MockZooKeeperClient) GetData(path string) ([]byte, error) {
	zk.mu.RLock()
	defer zk.mu.RUnlock()

	data, exists := zk.nodes[path]
	if !exists {
		return nil, fmt.Errorf("node not found: %s", path)
	}

	return data, nil
}

// SetData 设置节点数据
func (zk *MockZooKeeperClient) SetData(path string, data []byte) error {
	zk.mu.Lock()
	defer zk.mu.Unlock()

	if _, exists := zk.nodes[path]; !exists {
		return fmt.Errorf("node not found: %s", path)
	}

	zk.nodes[path] = data
	return nil
}

// Delete 删除节点
func (zk *MockZooKeeperClient) Delete(path string) error {
	zk.mu.Lock()
	defer zk.mu.Unlock()

	if _, exists := zk.nodes[path]; !exists {
		return fmt.Errorf("node not found: %s", path)
	}

	delete(zk.nodes, path)
	return nil
}

// GetChildren 获取子节点
func (zk *MockZooKeeperClient) GetChildren(path string) ([]string, error) {
	zk.mu.RLock()
	defer zk.mu.RUnlock()

	var children []string
	prefix := path + "/"

	for nodePath := range zk.nodes {
		if len(nodePath) > len(prefix) && nodePath[:len(prefix)] == prefix {
			childName := nodePath[len(prefix):]
			// 只返回直接子节点
			if !contains(childName, "/") {
				children = append(children, childName)
			}
		}
	}

	return children, nil
}

// CreateMode 创建模式
type CreateMode int

const (
	CreateModePersistent CreateMode = iota
	CreateModeEphemeral
	CreateModePersistentSequential
	CreateModeEphemeralSequential
)

// NewZooKeeperStateProvider 创建 ZooKeeper 状态提供者
func NewZooKeeperStateProvider() *ZooKeeperStateProvider {
	return &ZooKeeperStateProvider{
		zkClient:      NewMockZooKeeperClient(),
		stateRootPath: "/nifi/state",
	}
}

// Initialize 初始化提供者
func (zsp *ZooKeeperStateProvider) Initialize(config ProviderConfiguration) error {
	zsp.mu.Lock()
	defer zsp.mu.Unlock()

	// 获取 ZooKeeper 连接配置
	zkHost, ok := config.Properties["zookeeper.host"]
	if !ok {
		zkHost = "localhost:2181"
	}

	// 获取状态根路径
	if rootPath, ok := config.Properties["state.root.path"]; ok {
		zsp.stateRootPath = rootPath
	}

	// 在实际实现中，这里应该连接到真实的 ZooKeeper 集群
	// 为了演示，我们使用模拟客户端
	fmt.Printf("Initializing ZooKeeper state provider with host: %s, root path: %s\n", zkHost, zsp.stateRootPath)

	return nil
}

// LoadState 加载组件状态
func (zsp *ZooKeeperStateProvider) LoadState(componentID string) (*StateMap, error) {
	zsp.mu.RLock()
	defer zsp.mu.RUnlock()

	path := zsp.stateRootPath + "/" + componentID

	data, err := zsp.zkClient.GetData(path)
	if err != nil {
		// 节点不存在，返回空状态
		return NewStateMap(nil, StateScopeCluster), nil
	}

	stateMap, err := DeserializeStateMap(data)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize state: %w", err)
	}

	return stateMap, nil
}

// PersistState 持久化组件状态
func (zsp *ZooKeeperStateProvider) PersistState(componentID string, stateMap *StateMap) error {
	zsp.mu.Lock()
	defer zsp.mu.Unlock()

	path := zsp.stateRootPath + "/" + componentID

	// 序列化状态
	data, err := SerializeStateMap(stateMap)
	if err != nil {
		return fmt.Errorf("failed to serialize state: %w", err)
	}

	// 尝试设置数据（节点已存在）
	err = zsp.zkClient.SetData(path, data)
	if err != nil {
		// 节点不存在，创建新节点
		err = zsp.zkClient.Create(path, data, CreateModeEphemeral)
		if err != nil {
			return fmt.Errorf("failed to create state node: %w", err)
		}
	}

	return nil
}

// Shutdown 关闭资源
func (zsp *ZooKeeperStateProvider) Shutdown() error {
	// 在实际实现中，这里应该关闭 ZooKeeper 连接
	return nil
}

// DistributedStateLock 分布式锁实现
type DistributedStateLock struct {
	zkClient     *MockZooKeeperClient
	lockRootPath string
	mu           sync.Mutex
}

// NewDistributedStateLock 创建分布式锁
func NewDistributedStateLock() *DistributedStateLock {
	return &DistributedStateLock{
		zkClient:     NewMockZooKeeperClient(),
		lockRootPath: "/nifi/locks",
	}
}

// AcquireLock 获取锁
func (dsl *DistributedStateLock) AcquireLock(componentID string) error {
	dsl.mu.Lock()
	defer dsl.mu.Unlock()

	lockPath := dsl.lockRootPath + "/" + componentID

	// 创建临时顺序节点
	sequentialPath := lockPath + "/lock-"
	data := []byte(fmt.Sprintf("%d", time.Now().UnixNano()))

	err := dsl.zkClient.Create(sequentialPath, data, CreateModeEphemeralSequential)
	if err != nil {
		return fmt.Errorf("failed to create lock node: %w", err)
	}

	// 检查是否获得锁
	children, err := dsl.zkClient.GetChildren(lockPath)
	if err != nil {
		return fmt.Errorf("failed to get lock children: %w", err)
	}

	// 在实际实现中，这里应该等待前一个锁释放
	// 为了演示，我们简化处理
	if len(children) > 1 {
		// 有多个锁节点，等待前一个释放
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// ReleaseLock 释放锁
func (dsl *DistributedStateLock) ReleaseLock(componentID string) error {
	dsl.mu.Lock()
	defer dsl.mu.Unlock()

	lockPath := dsl.lockRootPath + "/" + componentID

	// 删除锁节点
	err := dsl.zkClient.Delete(lockPath)
	if err != nil {
		return fmt.Errorf("failed to delete lock node: %w", err)
	}

	return nil
}

// StateConsistencyManager 状态一致性管理器
type StateConsistencyManager struct {
	localProvider   StateProvider
	clusterProvider StateProvider
	mu              sync.RWMutex
}

// NewStateConsistencyManager 创建状态一致性管理器
func NewStateConsistencyManager(localProvider, clusterProvider StateProvider) *StateConsistencyManager {
	return &StateConsistencyManager{
		localProvider:   localProvider,
		clusterProvider: clusterProvider,
	}
}

// SyncState 同步状态
func (scm *StateConsistencyManager) SyncState(componentID string) error {
	scm.mu.Lock()
	defer scm.mu.Unlock()

	// 加载本地状态
	localState, err := scm.localProvider.LoadState(componentID)
	if err != nil {
		return fmt.Errorf("failed to load local state: %w", err)
	}

	// 加载集群状态
	clusterState, err := scm.clusterProvider.LoadState(componentID)
	if err != nil {
		return fmt.Errorf("failed to load cluster state: %w", err)
	}

	// 比较状态版本，选择最新状态
	mergedState := scm.selectLatestState(localState, clusterState)

	// 更新本地和集群状态
	if err := scm.localProvider.PersistState(componentID, mergedState); err != nil {
		return fmt.Errorf("failed to update local state: %w", err)
	}

	if err := scm.clusterProvider.PersistState(componentID, mergedState); err != nil {
		return fmt.Errorf("failed to update cluster state: %w", err)
	}

	return nil
}

// selectLatestState 选择最新状态
func (scm *StateConsistencyManager) selectLatestState(local, cluster *StateMap) *StateMap {
	if local.Timestamp > cluster.Timestamp {
		return local
	}
	return cluster
}

// 辅助函数
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}
