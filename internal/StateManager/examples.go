package statemanager

import (
	"fmt"
	"log"
	"time"

	"github.com/edge-stream/internal/StateManager/DistributedStateProvider"
	"github.com/edge-stream/internal/StateManager/LocalStateProvider"
	
	"github.com/edge-stream/internal/StateManager/StateRecoveryManager"
	"github.com/edge-stream/internal/StateManager/StateSynchronizer"
)

// ExampleBasicUsage 基本使用示例
func ExampleBasicUsage() {

	
	// 创建状态管理器

	
	// 创建本地状态提供者
	localProvider := localstateprovider.NewFileSystemStateProvider()
	localConfig := ProviderConfiguration{
		Properties: map[string]string{
			"state.directory": "./local_state",
		},
		Type: StateProviderTypeLocalFileSystem,

	
	if err := localProvider.Initialize(localConfig); err != nil {
		log.Fatalf("Failed to initialize local provider: %v", err)

	
	// 创建内存状态提供者作为集群提供者（用于演示）
	clusterProvider := localstateprovider.NewMemoryStateProvider()
	clusterConfig := ProviderConfiguration{
		Properties: map[string]string{},
		Type:       StateProviderTypeCustom,

	
	if err := clusterProvider.Initialize(clusterConfig); err != nil {
		log.Fatalf("Failed to initialize cluster provider: %v", err)

	
	// 初始化状态管理器
		LocalProvider:      localProvider,
		ClusterProvider:    clusterProvider,
		ConsistencyLevel:   ConsistencyLevelEventual,
		ConsistencyLevel:  ConsistencyLevelEventual,
		CheckpointInterval: 5 * time.Minute,
		SyncInterval:       30 * time.Second,

	
	if err := stateManager.Initialize(managerConfig); err != nil {
		log.Fatalf("Failed to initialize state manager: %v", err)

	
	// 设置状态
	componentID := "processor-001"
		"counter":    "100",
		"lastUpdate": time.Now().Format(time.RFC3339),
		"status":     "running",
		"status":      "running",

	
	if err := stateManager.SetState(componentID, stateMap); err != nil {
		log.Fatalf("Failed to set state: %v", err)

	

	
	// 获取状态
	retrievedState, err := stateManager.GetState(componentID)
	if err != nil {
		log.Fatalf("Failed to get state: %v", err)

	fmt.Printf("Retrieved state: counter=%s, status=%s\n",
		retrievedState.GetValue("counter"),
		retrievedState.GetValue("counter"), 

	
	// 更新状态
	updateFunc := func(currentState *StateMap) *StateMap {
		currentState.SetValue("counter", "150")
		currentState.SetValue("lastUpdate", time.Now().Format(time.RFC3339))
		return currentState

	
	if err := stateManager.UpdateState(componentID, updateFunc); err != nil {
		log.Fatalf("Failed to update state: %v", err)

	

	
	// 关闭状态管理器
	if err := stateManager.Shutdown(); err != nil {
		log.Fatalf("Failed to shutdown state manager: %v", err)
	}
}

// ExampleDistributedState 分布式状态示例
func ExampleDistributedState() {

	
	// 创建 ZooKeeper 状态提供者
	zkProvider := distributedstateprovider.NewZooKeeperStateProvider()
	zkConfig := ProviderConfiguration{
			"zookeeper.host":  "localhost:2181",
			"state.root.path": "/nifi/state",
			"state.root.path":   "/nifi/state",
		},
		Type: StateProviderTypeZooKeeper,

	
	if err := zkProvider.Initialize(zkConfig); err != nil {
		log.Fatalf("Failed to initialize ZooKeeper provider: %v", err)

	
	// 创建状态管理器

	
	// 使用 ZooKeeper 作为集群提供者
		LocalProvider:      localstateprovider.NewMemoryStateProvider(),
		ClusterProvider:    zkProvider,
		ConsistencyLevel:   ConsistencyLevelStrong,
		ConsistencyLevel:  ConsistencyLevelStrong,
		CheckpointInterval: 2 * time.Minute,
		SyncInterval:       10 * time.Second,

	
	if err := stateManager.Initialize(managerConfig); err != nil {
		log.Fatalf("Failed to initialize state manager: %v", err)

	
	// 设置分布式状态
	componentID := "cluster-processor-001"
		"nodeId":    "node-001",
		"partition": "partition-1",
		"leader":    "true",
		"lastSync":  time.Now().Format(time.RFC3339),
		"lastSync":    time.Now().Format(time.RFC3339),

	
	if err := stateManager.SetState(componentID, clusterState); err != nil {
		log.Fatalf("Failed to set cluster state: %v", err)

	

	
	// 同步状态
	if err := stateManager.SyncState(componentID); err != nil {
		log.Fatalf("Failed to sync state: %v", err)

	

	
	// 关闭状态管理器
	if err := stateManager.Shutdown(); err != nil {
		log.Fatalf("Failed to shutdown state manager: %v", err)
	}
}

// ExampleCheckpointAndRecovery 检查点和恢复示例
func ExampleCheckpointAndRecovery() {

	
	// 创建检查点管理器
	checkpointManager := staterecoverymanager.NewStateCheckpointManager()
	if err := checkpointManager.Initialize("./checkpoints"); err != nil {
		log.Fatalf("Failed to initialize checkpoint manager: %v", err)

	
	// 创建预写日志管理器
	walManager := staterecoverymanager.NewWriteAheadLogManager()
	if err := walManager.Initialize("./wal"); err != nil {
		log.Fatalf("Failed to initialize WAL manager: %v", err)

	
	// 创建状态
	componentID := "recovery-processor-001"
	stateMap := NewStateMap(map[string]string{
		"processedCount": "1000",
		"lastCheckpoint": time.Now().Format(time.RFC3339),
		"status":         "active",

	
	// 创建检查点
	if err := checkpointManager.CreateCheckpoint(componentID, stateMap); err != nil {
		log.Fatalf("Failed to create checkpoint: %v", err)

	

	
	// 记录状态更新
	if err := walManager.LogStateUpdate(componentID, nil, stateMap); err != nil {
		log.Fatalf("Failed to log state update: %v", err)

	

	
	// 模拟故障恢复

	
	// 恢复最新检查点
	recoveredState, err := checkpointManager.RestoreLatestCheckpoint(componentID)
	if err != nil {
		log.Fatalf("Failed to restore checkpoint: %v", err)

	
	fmt.Printf("Recovered state: processedCount=%s, status=%s\n",
		recoveredState.GetValue("processedCount"),

	
	// 恢复 WAL 记录
	recoveredUpdates, err := walManager.RecoverUpdates(componentID)
	if err != nil {
		log.Fatalf("Failed to recover updates: %v", err)

	
	fmt.Printf("Recovered %d updates from WAL\n", len(recoveredUpdates))
}

// ExampleStateSynchronizer 状态同步器示例
func ExampleStateSynchronizer() {

	
	// 创建状态同步器

	
	// 注册 Redis 扩展
	redisExtension := &statesynchronizer.RedisStateProviderExtension{}
	if err := synchronizer.RegisterExtension(redisExtension); err != nil {
		log.Fatalf("Failed to register Redis extension: %v", err)

	
	// 配置同步
	localConfig := ProviderConfiguration{
		Properties: map[string]string{
			"state.directory": "./local_state",
		},
		Type: StateProviderTypeLocalFileSystem,

	
	clusterConfig := ProviderConfiguration{
		Properties: map[string]string{
			"redis.host": "localhost",
			"redis.port": "6379",
		},
		Type: StateProviderTypeRedis,

	
	syncConfig := statesynchronizer.SyncConfiguration{
		LocalProviderConfig:   localConfig,
		ClusterProviderConfig: clusterConfig,
		SyncInterval:          30 * time.Second,
		AutoSync:              true,

	
	if err := synchronizer.Initialize(syncConfig); err != nil {
		log.Fatalf("Failed to initialize synchronizer: %v", err)

	

	
	// 获取已注册的提供者
	providers := synchronizer.GetRegisteredProviders()

	
	// 手动同步状态
	componentID := "sync-processor-001"
	if err := synchronizer.SyncState(componentID); err != nil {
		log.Fatalf("Failed to sync state: %v", err)

	

	
	// 关闭同步器
	if err := synchronizer.Shutdown(); err != nil {
		log.Fatalf("Failed to shutdown synchronizer: %v", err)
	}
}

// ExampleTransactionalUpdates 事务性更新示例
func ExampleTransactionalUpdates() {

	
	// 创建状态管理器

	
	// 使用内存提供者进行演示
	localProvider := localstateprovider.NewMemoryStateProvider()

	
		LocalProvider:      localProvider,
		ClusterProvider:    clusterProvider,
		ConsistencyLevel:   ConsistencyLevelStrong,
		ConsistencyLevel:  ConsistencyLevelStrong,
		CheckpointInterval: 1 * time.Minute,
		SyncInterval:       5 * time.Second,

	
	if err := stateManager.Initialize(managerConfig); err != nil {
		log.Fatalf("Failed to initialize state manager: %v", err)

	
	// 初始状态
	componentID := "transactional-processor-001"
		"balance":      "1000",
		"balance":     "1000",
		"transactions": "0",

	
	if err := stateManager.SetState(componentID, initialState); err != nil {
		log.Fatalf("Failed to set initial state: %v", err)

	
	fmt.Printf("Initial state: balance=%s, transactions=%s\n",
		initialState.GetValue("balance"),

	
	// 事务性更新：模拟银行转账
	transferUpdate := func(currentState *StateMap) *StateMap {
		currentBalance := currentState.GetValue("balance")

		
		// 模拟转账逻辑
		// 在实际应用中，这里会有更复杂的业务逻辑
		newBalance := "950" // 扣除 50

		
		currentState.SetValue("balance", newBalance)
		currentState.SetValue("transactions", newTransactions)

		
		return currentState

	
	// 执行事务性更新
	if err := stateManager.UpdateStateTransactionally(componentID, transferUpdate); err != nil {
		log.Fatalf("Failed to perform transactional update: %v", err)

	

	
	// 验证更新结果
	updatedState, err := stateManager.GetState(componentID)
	if err != nil {
		log.Fatalf("Failed to get updated state: %v", err)

	
	fmt.Printf("Updated state: balance=%s, transactions=%s, lastTransfer=%s\n",
		updatedState.GetValue("balance"),
		updatedState.GetValue("transactions"),

	
	// 关闭状态管理器
	if err := stateManager.Shutdown(); err != nil {
		log.Fatalf("Failed to shutdown state manager: %v", err)
	}
}

// ExampleCustomStateProvider 自定义状态提供者示例
func ExampleCustomStateProvider() {

	
	// 创建自定义状态提供者
	customProvider := &CustomDatabaseStateProvider{
		connectionString: "postgres://localhost:5432/statestore",
		tableName:        "component_states",

	
	config := ProviderConfiguration{
		Properties: map[string]string{
			"db.connection": "postgres://localhost:5432/statestore",
			"db.table":      "component_states",
		},
		Type: StateProviderTypeCustom,

	
	if err := customProvider.Initialize(config); err != nil {
		log.Fatalf("Failed to initialize custom provider: %v", err)

	
	// 使用自定义提供者
	componentID := "custom-processor-001"
	stateMap := NewStateMap(map[string]string{
		"customField1": "value1",
		"customField2": "value2",
		"metadata":     "custom metadata",

	
	if err := customProvider.PersistState(componentID, stateMap); err != nil {
		log.Fatalf("Failed to persist state with custom provider: %v", err)

	

	
	// 加载状态
	loadedState, err := customProvider.LoadState(componentID)
	if err != nil {
		log.Fatalf("Failed to load state with custom provider: %v", err)

	
	fmt.Printf("Loaded state: customField1=%s, customField2=%s\n",
		loadedState.GetValue("customField1"),

	
	// 关闭自定义提供者
	if err := customProvider.Shutdown(); err != nil {
		log.Fatalf("Failed to shutdown custom provider: %v", err)
	}
}

// CustomDatabaseStateProvider 自定义数据库状态提供者示例
type CustomDatabaseStateProvider struct {
	connectionString string
	tableName        string
	mu               sync.RWMutex
}

func (cdsp *CustomDatabaseStateProvider) Initialize(config ProviderConfiguration) error {
	cdsp.mu.Lock()

	
	if connStr, ok := config.Properties["db.connection"]; ok {
		cdsp.connectionString = connStr

	
	if tableName, ok := config.Properties["db.table"]; ok {
		cdsp.tableName = tableName

	fmt.Printf("Custom database provider initialized: %s, table: %s\n",
	fmt.Printf("Custom database provider initialized: %s, table: %s\n", 

	
	return nil
}

func (cdsp *CustomDatabaseStateProvider) LoadState(componentID string) (*StateMap, error) {
	cdsp.mu.RLock()

	
	// 在实际实现中，这里会从数据库加载状态
	// 为了演示，我们返回模拟数据
	stateValues := map[string]string{
		"customField1": "value1",
		"customField2": "value2",
		"metadata":     "custom metadata",

	
	return NewStateMap(stateValues, StateScopeGlobal), nil
}

func (cdsp *CustomDatabaseStateProvider) PersistState(componentID string, stateMap *StateMap) error {
	cdsp.mu.Lock()

	
	fmt.Printf("State persisted to database: component=%s, table=%s\n",
	fmt.Printf("State persisted to database: component=%s, table=%s\n", 

	
	return nil
}

func (cdsp *CustomDatabaseStateProvider) Shutdown() error {
	// 在实际实现中，这里会关闭数据库连接
	fmt.Println("Custom database provider shutdown")
	return nil
}

// RunAllExamples 运行所有示例
func RunAllExamples() {

	
	ExampleBasicUsage()
	ExampleDistributedState()
	ExampleCheckpointAndRecovery()
	ExampleStateSynchronizer()
	ExampleTransactionalUpdates()

	
}

} 