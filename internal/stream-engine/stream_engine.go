package stream_engine

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/edge-stream/internal/ConfigManager"
	"github.com/edge-stream/internal/ConnectorRegistry"
	"github.com/edge-stream/internal/MetricCollector"
	"github.com/edge-stream/internal/StateManager"
)

// StreamEngine 是数据流转的"动力核心"，驱动数据在Pipeline中高效传输、处理与调度的底层引擎
type StreamEngine interface {
	// Start 启动引擎
	Start(ctx context.Context) error

	// Stop 停止引擎
	Stop(ctx context.Context) error

	// Schedule 调度处理器
	Schedule(processor ProcessorNode, strategy SchedulingStrategy) error

	// ApplyBackpressure 对连接应用背压
	ApplyBackpressure(connection Connection) error

	// GetStatus 获取引擎状态
	GetStatus() EngineStatus

	// GetScheduler 获取调度器实例
	GetScheduler() Scheduler

	// GetThreadPoolManager 获取线程池管理器
	GetThreadPoolManager() ThreadPoolManager
}

// EngineStatus 引擎状态
type EngineStatus string

const (
	StatusInitialized EngineStatus = "INITIALIZED"
	StatusStarting    EngineStatus = "STARTING"
	StatusRunning     EngineStatus = "RUNNING"
	StatusPausing     EngineStatus = "PAUSING"
	StatusPaused      EngineStatus = "PAUSED"
	StatusResuming    EngineStatus = "RESUMING"
	StatusStopping    EngineStatus = "STOPPING"
	StatusStopped     EngineStatus = "STOPPED"
	StatusFailed      EngineStatus = "FAILED"
)

// StandardStreamEngine 标准流引擎实现
type StandardStreamEngine struct {
	mu                 sync.RWMutex
	status             EngineStatus
	configManager      *ConfigManager.ConfigManager
	connectorRegistry  *ConnectorRegistry.ConnectorRegistry
	metricCollector    *MetricCollector.MetricCollector
	stateManager       *StateManager.StateManager
	threadPoolManager  ThreadPoolManager
	clusterCoordinator ClusterCoordinator
	queueManager       QueueManager
	flowController     FlowController
	scheduler          Scheduler
	backpressureMgr    *BackpressureManager

	// 配置参数
	config *EngineConfig

	// 上下文控制
	ctx    context.Context
	cancel context.CancelFunc
}

// EngineConfig 引擎配置
type EngineConfig struct {
	MaxThreads            int           `json:"max_threads"`
	CoreThreads           int           `json:"core_threads"`
	QueueCapacity         int           `json:"queue_capacity"`
	BackpressureThreshold float64       `json:"backpressure_threshold"`
	ClusterEnabled        bool          `json:"cluster_enabled"`
	ZooKeeperConnect      string        `json:"zookeeper_connect"`
	SyncInterval          time.Duration `json:"sync_interval"`
}

// NewStandardStreamEngine 创建标准流引擎实例
func NewStandardStreamEngine(config *EngineConfig) *StandardStreamEngine {
	if config == nil {
		config = &EngineConfig{
			MaxThreads:            runtime.NumCPU() * 4,
			CoreThreads:           runtime.NumCPU() * 2,
			QueueCapacity:         10000,
			BackpressureThreshold: 0.8,
			ClusterEnabled:        false,
			SyncInterval:          time.Second * 5,
		}
	}

	engine := &StandardStreamEngine{
		status: StatusInitialized,
		config: config,
	}

	// 初始化组件
	engine.initializeComponents()

	return engine
}

// initializeComponents 初始化引擎组件
func (e *StandardStreamEngine) initializeComponents() {
	// 初始化线程池管理器
	e.threadPoolManager = NewThreadPoolManager(e.config)

	// 初始化调度器
	e.scheduler = NewScheduler()

	// 初始化队列管理器
	e.queueManager = NewQueueManager(e.config.QueueCapacity)

	// 初始化背压管理器
	e.backpressureMgr = NewBackpressureManager(e.config.BackpressureThreshold)

	// 初始化集群协调器（如果启用）
	if e.config.ClusterEnabled {
		e.clusterCoordinator = NewClusterCoordinator(e.config.ZooKeeperConnect)
	} else {
		e.clusterCoordinator = NewLocalClusterCoordinator()
	}

	// 初始化流程控制器
	e.flowController = NewFlowController(e.scheduler, e.threadPoolManager)
}

// Start 启动引擎
func (e *StandardStreamEngine) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.status != StatusInitialized && e.status != StatusStopped {
		return fmt.Errorf("engine cannot start from status: %s", e.status)
	}

	e.status = StatusStarting
	e.ctx, e.cancel = context.WithCancel(ctx)

	// 启动线程池管理器
	if err := e.threadPoolManager.Start(e.ctx); err != nil {
		e.status = StatusFailed
		return fmt.Errorf("failed to start thread pool manager: %w", err)
	}

	// 启动调度器
	if err := e.scheduler.Start(e.ctx); err != nil {
		e.status = StatusFailed
		return fmt.Errorf("failed to start scheduler: %w", err)
	}

	// 启动集群协调器
	if err := e.clusterCoordinator.Start(e.ctx); err != nil {
		e.status = StatusFailed
		return fmt.Errorf("failed to start cluster coordinator: %w", err)
	}

	// 启动背压管理器
	if err := e.backpressureMgr.Start(e.ctx); err != nil {
		e.status = StatusFailed
		return fmt.Errorf("failed to start backpressure manager: %w", err)
	}

	e.status = StatusRunning

	// 记录启动指标
	if e.metricCollector != nil {
		e.metricCollector.RecordEngineStart()
	}

	return nil
}

// Stop 停止引擎
func (e *StandardStreamEngine) Stop(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.status == StatusStopped {
		return nil
	}

	e.status = StatusStopping

	// 取消上下文
	if e.cancel != nil {
		e.cancel()
	}

	// 停止各个组件
	var errs []error

	if err := e.scheduler.Stop(ctx); err != nil {
		errs = append(errs, fmt.Errorf("failed to stop scheduler: %w", err))
	}

	if err := e.threadPoolManager.Stop(ctx); err != nil {
		errs = append(errs, fmt.Errorf("failed to stop thread pool manager: %w", err))
	}

	if err := e.clusterCoordinator.Stop(ctx); err != nil {
		errs = append(errs, fmt.Errorf("failed to stop cluster coordinator: %w", err))
	}

	if err := e.backpressureMgr.Stop(ctx); err != nil {
		errs = append(errs, fmt.Errorf("failed to stop backpressure manager: %w", err))
	}

	e.status = StatusStopped

	// 记录停止指标
	if e.metricCollector != nil {
		e.metricCollector.RecordEngineStop()
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during engine stop: %v", errs)
	}

	return nil
}

// Schedule 调度处理器
func (e *StandardStreamEngine) Schedule(processor ProcessorNode, strategy SchedulingStrategy) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.status != StatusRunning {
		return fmt.Errorf("engine is not running, current status: %s", e.status)
	}

	// 创建触发器
	trigger := e.createTrigger(strategy)

	// 调度处理器
	if err := e.scheduler.Schedule(processor, trigger); err != nil {
		return fmt.Errorf("failed to schedule processor: %w", err)
	}

	// 分配线程资源
	if err := e.threadPoolManager.AssignThreads(processor, strategy.GetMaxThreads()); err != nil {
		return fmt.Errorf("failed to assign threads: %w", err)
	}

	// 记录调度指标
	if e.metricCollector != nil {
		e.metricCollector.RecordProcessorScheduled(processor.GetID())
	}

	return nil
}

// ApplyBackpressure 对连接应用背压
func (e *StandardStreamEngine) ApplyBackpressure(connection Connection) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.status != StatusRunning {
		return fmt.Errorf("engine is not running, current status: %s", e.status)
	}

	return e.backpressureMgr.ApplyBackpressure(connection)
}

// GetStatus 获取引擎状态
func (e *StandardStreamEngine) GetStatus() EngineStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.status
}

// GetScheduler 获取调度器实例
func (e *StandardStreamEngine) GetScheduler() Scheduler {
	return e.scheduler
}

// GetThreadPoolManager 获取线程池管理器
func (e *StandardStreamEngine) GetThreadPoolManager() ThreadPoolManager {
	return e.threadPoolManager
}

// createTrigger 根据调度策略创建触发器
func (e *StandardStreamEngine) createTrigger(strategy SchedulingStrategy) Trigger {
	switch strategy.GetType() {
	case SchedulingTypeEventDriven:
		return NewEventDrivenTrigger(strategy)
	case SchedulingTypeTimerDriven:
		return NewTimerDrivenTrigger(strategy)
	case SchedulingTypeCronDriven:
		return NewCronDrivenTrigger(strategy)
	default:
		return NewEventDrivenTrigger(strategy)
	}
}

// SetConfigManager 设置配置管理器
func (e *StandardStreamEngine) SetConfigManager(cm *ConfigManager.ConfigManager) {
	e.configManager = cm
}

// SetConnectorRegistry 设置连接器注册表
func (e *StandardStreamEngine) SetConnectorRegistry(cr *ConnectorRegistry.ConnectorRegistry) {
	e.connectorRegistry = cr
}

// SetMetricCollector 设置指标收集器
func (e *StandardStreamEngine) SetMetricCollector(mc *MetricCollector.MetricCollector) {
	e.metricCollector = mc
}

// SetStateManager 设置状态管理器
func (e *StandardStreamEngine) SetStateManager(sm *StateManager.StateManager) {
	e.stateManager = sm
}
