package stream_engine

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// ThreadPoolManager 线程池管理器接口
type ThreadPoolManager interface {
	// Start 启动线程池管理器
	Start(ctx context.Context) error

	// Stop 停止线程池管理器
	Stop(ctx context.Context) error

	// AssignThreads 为处理器分配线程
	AssignThreads(processor ProcessorNode, maxThreads int) error

	// ReleaseThreads 释放处理器线程
	ReleaseThreads(processorID string) error

	// GetThreadPoolStats 获取线程池统计信息
	GetThreadPoolStats() ThreadPoolStats

	// AdjustThreadPool 动态调整线程池
	AdjustThreadPool(processorLoad int) error
}

// ThreadPoolStats 线程池统计信息
type ThreadPoolStats struct {
	TotalThreads    int     `json:"total_threads"`
	ActiveThreads   int     `json:"active_threads"`
	IdleThreads     int     `json:"idle_threads"`
	UtilizationRate float64 `json:"utilization_rate"`
	QueueSize       int     `json:"queue_size"`
	CompletedTasks  int64   `json:"completed_tasks"`
}

// AdaptiveThreadPoolManager 自适应线程池管理器
type AdaptiveThreadPoolManager struct {
	mu             sync.RWMutex
	config         *EngineConfig
	executors      map[string]*ProcessorExecutor
	globalExecutor *GlobalExecutor
	monitor        *ThreadPoolMonitor
	ctx            context.Context
	cancel         context.CancelFunc
	running        bool
}

// ProcessorExecutor 处理器执行器
type ProcessorExecutor struct {
	processorID    string
	executor       *BoundedExecutor
	maxThreads     int
	currentThreads int
	mu             sync.RWMutex
}

// GlobalExecutor 全局执行器
type GlobalExecutor struct {
	executor       *BoundedExecutor
	maxThreads     int
	currentThreads int
	mu             sync.RWMutex
}

// BoundedExecutor 有界执行器
type BoundedExecutor struct {
	workerPool chan struct{}
	taskQueue  chan Task
	stats      *ExecutorStats
	ctx        context.Context
	cancel     context.CancelFunc
	mu         sync.RWMutex
}

// Task 任务接口
type Task interface {
	Execute() error
	GetID() string
	GetPriority() int
}

// ExecutorStats 执行器统计信息
type ExecutorStats struct {
	CompletedTasks int64         `json:"completed_tasks"`
	FailedTasks    int64         `json:"failed_tasks"`
	TotalWaitTime  time.Duration `json:"total_wait_time"`
	AvgWaitTime    time.Duration `json:"avg_wait_time"`
	mu             sync.RWMutex
}

// ThreadPoolMonitor 线程池监控器
type ThreadPoolMonitor struct {
	stats    map[string]*ThreadPoolStats
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	interval time.Duration
}

// NewThreadPoolManager 创建线程池管理器
func NewThreadPoolManager(config *EngineConfig) *AdaptiveThreadPoolManager {
	return &AdaptiveThreadPoolManager{
		config:    config,
		executors: make(map[string]*ProcessorExecutor),
		monitor:   NewThreadPoolMonitor(),
	}
}

// Start 启动线程池管理器
func (m *AdaptiveThreadPoolManager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("thread pool manager is already running")
	}

	m.ctx, m.cancel = context.WithCancel(ctx)
	m.running = true

	// 创建全局执行器
	m.globalExecutor = NewGlobalExecutor(m.config.MaxThreads)

	// 启动全局执行器
	if err := m.globalExecutor.Start(m.ctx); err != nil {
		return fmt.Errorf("failed to start global executor: %w", err)
	}

	// 启动监控器
	if err := m.monitor.Start(m.ctx); err != nil {
		return fmt.Errorf("failed to start thread pool monitor: %w", err)
	}

	return nil
}

// Stop 停止线程池管理器
func (m *AdaptiveThreadPoolManager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	m.running = false

	// 停止全局执行器
	if m.globalExecutor != nil {
		if err := m.globalExecutor.Stop(ctx); err != nil {
			return fmt.Errorf("failed to stop global executor: %w", err)
		}
	}

	// 停止所有处理器执行器
	for _, executor := range m.executors {
		if err := executor.Stop(ctx); err != nil {
			return fmt.Errorf("failed to stop processor executor: %w", err)
		}
	}

	// 停止监控器
	if err := m.monitor.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop thread pool monitor: %w", err)
	}

	// 取消上下文
	if m.cancel != nil {
		m.cancel()
	}

	return nil
}

// AssignThreads 为处理器分配线程
func (m *AdaptiveThreadPoolManager) AssignThreads(processor ProcessorNode, maxThreads int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	processorID := processor.GetID()

	// 检查是否已经分配
	if _, exists := m.executors[processorID]; exists {
		return fmt.Errorf("threads already assigned for processor %s", processorID)
	}

	// 创建处理器执行器
	executor := NewProcessorExecutor(processorID, maxThreads)

	// 启动执行器
	if err := executor.Start(m.ctx); err != nil {
		return fmt.Errorf("failed to start processor executor: %w", err)
	}

	// 记录执行器
	m.executors[processorID] = executor

	return nil
}

// ReleaseThreads 释放处理器线程
func (m *AdaptiveThreadPoolManager) ReleaseThreads(processorID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	executor, exists := m.executors[processorID]
	if !exists {
		return fmt.Errorf("no threads assigned for processor %s", processorID)
	}

	// 停止执行器
	if err := executor.Stop(m.ctx); err != nil {
		return fmt.Errorf("failed to stop processor executor: %w", err)
	}

	// 移除执行器
	delete(m.executors, processorID)

	return nil
}

// GetThreadPoolStats 获取线程池统计信息
func (m *AdaptiveThreadPoolManager) GetThreadPoolStats() ThreadPoolStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := ThreadPoolStats{}

	// 汇总所有执行器的统计信息
	for _, executor := range m.executors {
		executorStats := executor.GetStats()
		stats.TotalThreads += executorStats.TotalThreads
		stats.ActiveThreads += executorStats.ActiveThreads
		stats.IdleThreads += executorStats.IdleThreads
		stats.CompletedTasks += executorStats.CompletedTasks
	}

	// 计算利用率
	if stats.TotalThreads > 0 {
		stats.UtilizationRate = float64(stats.ActiveThreads) / float64(stats.TotalThreads)
	}

	return stats
}

// AdjustThreadPool 动态调整线程池
func (m *AdaptiveThreadPoolManager) AdjustThreadPool(processorLoad int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 根据负载计算最优线程数
	optimalThreads := m.calculateOptimalThreads(processorLoad)

	// 调整全局执行器
	if m.globalExecutor != nil {
		m.globalExecutor.SetMaxThreads(optimalThreads)
	}

	return nil
}

// calculateOptimalThreads 计算最优线程数
func (m *AdaptiveThreadPoolManager) calculateOptimalThreads(processorLoad int) int {
	// 基于CPU核心数和负载计算
	cpuCount := runtime.NumCPU()
	baseThreads := cpuCount * 2

	// 根据负载调整
	if processorLoad > 80 {
		return baseThreads * 2
	} else if processorLoad > 50 {
		return int(float64(baseThreads) * 1.5)
	} else {
		return baseThreads
	}
}

// NewProcessorExecutor 创建处理器执行器
func NewProcessorExecutor(processorID string, maxThreads int) *ProcessorExecutor {
	return &ProcessorExecutor{
		processorID:    processorID,
		maxThreads:     maxThreads,
		currentThreads: 0,
	}
}

// Start 启动处理器执行器
func (p *ProcessorExecutor) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 创建有界执行器
	p.executor = NewBoundedExecutor(p.maxThreads)

	// 启动执行器
	return p.executor.Start(ctx)
}

// Stop 停止处理器执行器
func (p *ProcessorExecutor) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.executor != nil {
		return p.executor.Stop(ctx)
	}

	return nil
}

// GetStats 获取统计信息
func (p *ProcessorExecutor) GetStats() ThreadPoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.executor == nil {
		return ThreadPoolStats{}
	}

	return p.executor.GetStats()
}

// NewGlobalExecutor 创建全局执行器
func NewGlobalExecutor(maxThreads int) *GlobalExecutor {
	return &GlobalExecutor{
		maxThreads:     maxThreads,
		currentThreads: 0,
	}
}

// Start 启动全局执行器
func (g *GlobalExecutor) Start(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.executor = NewBoundedExecutor(g.maxThreads)
	return g.executor.Start(ctx)
}

// Stop 停止全局执行器
func (g *GlobalExecutor) Stop(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.executor != nil {
		return g.executor.Stop(ctx)
	}

	return nil
}

// SetMaxThreads 设置最大线程数
func (g *GlobalExecutor) SetMaxThreads(maxThreads int) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.maxThreads = maxThreads
	if g.executor != nil {
		g.executor.SetMaxWorkers(maxThreads)
	}
}

// NewBoundedExecutor 创建有界执行器
func NewBoundedExecutor(maxWorkers int) *BoundedExecutor {
	return &BoundedExecutor{
		workerPool: make(chan struct{}, maxWorkers),
		taskQueue:  make(chan Task, 1000),
		stats:      &ExecutorStats{},
	}
}

// Start 启动有界执行器
func (b *BoundedExecutor) Start(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.ctx, b.cancel = context.WithCancel(ctx)

	// 启动工作协程
	for i := 0; i < cap(b.workerPool); i++ {
		go b.worker()
	}

	return nil
}

// Stop 停止有界执行器
func (b *BoundedExecutor) Stop(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.cancel != nil {
		b.cancel()
	}

	return nil
}

// SetMaxWorkers 设置最大工作协程数
func (b *BoundedExecutor) SetMaxWorkers(maxWorkers int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// 创建新的工作协程池
	newWorkerPool := make(chan struct{}, maxWorkers)

	// 迁移现有任务
	for {
		select {
		case task := <-b.taskQueue:
			select {
			case newWorkerPool <- struct{}{}:
				go b.executeTask(task)
			default:
				// 工作协程池已满，重新放回队列
				b.taskQueue <- task
			}
		default:
			// 队列为空，退出
			goto done
		}
	}

done:
	b.workerPool = newWorkerPool
}

// worker 工作协程
func (b *BoundedExecutor) worker() {
	for {
		select {
		case <-b.ctx.Done():
			return
		case task := <-b.taskQueue:
			b.executeTask(task)
		}
	}
}

// executeTask 执行任务
func (b *BoundedExecutor) executeTask(task Task) {
	startTime := time.Now()

	// 获取工作协程
	<-b.workerPool
	defer func() {
		b.workerPool <- struct{}{}
	}()

	// 执行任务
	err := task.Execute()

	// 更新统计信息
	b.stats.mu.Lock()
	b.stats.CompletedTasks++
	if err != nil {
		b.stats.FailedTasks++
	}
	waitTime := time.Since(startTime)
	b.stats.TotalWaitTime += waitTime
	b.stats.AvgWaitTime = b.stats.TotalWaitTime / time.Duration(b.stats.CompletedTasks)
	b.stats.mu.Unlock()
}

// GetStats 获取统计信息
func (b *BoundedExecutor) GetStats() ThreadPoolStats {
	b.stats.mu.RLock()
	defer b.stats.mu.RUnlock()

	return ThreadPoolStats{
		TotalThreads:    cap(b.workerPool),
		ActiveThreads:   len(b.workerPool),
		IdleThreads:     cap(b.workerPool) - len(b.workerPool),
		UtilizationRate: float64(len(b.workerPool)) / float64(cap(b.workerPool)),
		QueueSize:       len(b.taskQueue),
		CompletedTasks:  b.stats.CompletedTasks,
	}
}

// NewThreadPoolMonitor 创建线程池监控器
func NewThreadPoolMonitor() *ThreadPoolMonitor {
	return &ThreadPoolMonitor{
		stats:    make(map[string]*ThreadPoolStats),
		interval: time.Second * 5,
	}
}

// Start 启动监控器
func (m *ThreadPoolMonitor) Start(ctx context.Context) error {
	m.ctx, m.cancel = context.WithCancel(ctx)

	// 启动监控协程
	go m.monitor()

	return nil
}

// Stop 停止监控器
func (m *ThreadPoolMonitor) Stop(ctx context.Context) error {
	if m.cancel != nil {
		m.cancel()
	}

	return nil
}

// monitor 监控协程
func (m *ThreadPoolMonitor) monitor() {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.collectStats()
		}
	}
}

// collectStats 收集统计信息
func (m *ThreadPoolMonitor) collectStats() {
	// 这里可以添加具体的统计信息收集逻辑
	// 例如：监控线程池利用率、队列大小等
}
