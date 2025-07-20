package stream_engine

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// FlowController 流程控制器接口
type FlowController interface {
	// Start 启动流程
	Start(ctx context.Context) error

	// Stop 停止流程
	Stop(ctx context.Context) error

	// Pause 暂停流程
	Pause() error

	// Resume 恢复流程
	Resume() error

	// GetStatus 获取流程状态
	GetStatus() FlowStatus

	// RegisterProcessor 注册处理器
	RegisterProcessor(processor ProcessorNode) error

	// UnregisterProcessor 注销处理器
	UnregisterProcessor(processorID string) error

	// GetScheduler 获取调度器
	GetScheduler() Scheduler

	// GetThreadPoolManager 获取线程池管理器
	GetThreadPoolManager() ThreadPoolManager
}

// FlowStatus 流程状态
type FlowStatus string

const (
	FlowStatusStopped  FlowStatus = "STOPPED"
	FlowStatusStarting FlowStatus = "STARTING"
	FlowStatusRunning  FlowStatus = "RUNNING"
	FlowStatusPaused   FlowStatus = "PAUSED"
	FlowStatusStopping FlowStatus = "STOPPING"
)

// StandardFlowController 标准流程控制器
type StandardFlowController struct {
	scheduler         Scheduler
	threadPoolManager ThreadPoolManager
	processors        map[string]ProcessorNode
	connections       map[string]Connection
	status            FlowStatus
	mu                sync.RWMutex
	ctx               context.Context
	cancel            context.CancelFunc
}

// NewFlowController 创建流程控制器
func NewFlowController(scheduler Scheduler, threadPoolManager ThreadPoolManager) *StandardFlowController {
	return &StandardFlowController{
		scheduler:         scheduler,
		threadPoolManager: threadPoolManager,
		processors:        make(map[string]ProcessorNode),
		connections:       make(map[string]Connection),
		status:            FlowStatusStopped,
	}
}

// Start 启动流程
func (f *StandardFlowController) Start(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.status != FlowStatusStopped && f.status != FlowStatusPaused {
		return fmt.Errorf("flow cannot start from status: %s", f.status)
	}

	f.status = FlowStatusStarting
	f.ctx, f.cancel = context.WithCancel(ctx)

	// 启动所有处理器
	for _, processor := range f.processors {
		if err := f.startProcessor(processor); err != nil {
			f.status = FlowStatusStopped
			return fmt.Errorf("failed to start processor %s: %w", processor.GetID(), err)
		}
	}

	f.status = FlowStatusRunning
	return nil
}

// Stop 停止流程
func (f *StandardFlowController) Stop(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.status == FlowStatusStopped {
		return nil
	}

	f.status = FlowStatusStopping

	// 取消上下文
	if f.cancel != nil {
		f.cancel()
	}

	// 停止所有处理器
	for _, processor := range f.processors {
		if err := f.stopProcessor(processor); err != nil {
			// 记录错误但不中断停止过程
		}
	}

	f.status = FlowStatusStopped
	return nil
}

// Pause 暂停流程
func (f *StandardFlowController) Pause() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.status != FlowStatusRunning {
		return fmt.Errorf("flow is not running, current status: %s", f.status)
	}

	f.status = FlowStatusPaused

	// 暂停所有处理器
	for _, processor := range f.processors {
		if err := f.pauseProcessor(processor); err != nil {
			// 记录错误但不中断暂停过程
		}
	}

	return nil
}

// Resume 恢复流程
func (f *StandardFlowController) Resume() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.status != FlowStatusPaused {
		return fmt.Errorf("flow is not paused, current status: %s", f.status)
	}

	f.status = FlowStatusRunning

	// 恢复所有处理器
	for _, processor := range f.processors {
		if err := f.resumeProcessor(processor); err != nil {
			// 记录错误但不中断恢复过程
		}
	}

	return nil
}

// GetStatus 获取流程状态
func (f *StandardFlowController) GetStatus() FlowStatus {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.status
}

// RegisterProcessor 注册处理器
func (f *StandardFlowController) RegisterProcessor(processor ProcessorNode) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	processorID := processor.GetID()
	if _, exists := f.processors[processorID]; exists {
		return fmt.Errorf("processor %s is already registered", processorID)
	}

	f.processors[processorID] = processor

	// 如果流程正在运行，启动处理器
	if f.status == FlowStatusRunning {
		return f.startProcessor(processor)
	}

	return nil
}

// UnregisterProcessor 注销处理器
func (f *StandardFlowController) UnregisterProcessor(processorID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	processor, exists := f.processors[processorID]
	if !exists {
		return fmt.Errorf("processor %s is not registered", processorID)
	}

	// 停止处理器
	if f.status == FlowStatusRunning {
		if err := f.stopProcessor(processor); err != nil {
			return fmt.Errorf("failed to stop processor: %w", err)
		}
	}

	delete(f.processors, processorID)
	return nil
}

// GetScheduler 获取调度器
func (f *StandardFlowController) GetScheduler() Scheduler {
	return f.scheduler
}

// GetThreadPoolManager 获取线程池管理器
func (f *StandardFlowController) GetThreadPoolManager() ThreadPoolManager {
	return f.threadPoolManager
}

// startProcessor 启动处理器
func (f *StandardFlowController) startProcessor(processor ProcessorNode) error {
	// 这里可以添加处理器启动逻辑
	return nil
}

// stopProcessor 停止处理器
func (f *StandardFlowController) stopProcessor(processor ProcessorNode) error {
	// 这里可以添加处理器停止逻辑
	return nil
}

// pauseProcessor 暂停处理器
func (f *StandardFlowController) pauseProcessor(processor ProcessorNode) error {
	// 这里可以添加处理器暂停逻辑
	return nil
}

// resumeProcessor 恢复处理器
func (f *StandardFlowController) resumeProcessor(processor ProcessorNode) error {
	// 这里可以添加处理器恢复逻辑
	return nil
}

// QueueManager 队列管理器接口
type QueueManager interface {
	// CreateQueue 创建队列
	CreateQueue(name string, capacity int64) (Queue, error)

	// GetQueue 获取队列
	GetQueue(name string) (Queue, error)

	// RemoveQueue 移除队列
	RemoveQueue(name string) error

	// GetQueueStats 获取队列统计信息
	GetQueueStats() map[string]QueueStats
}

// QueueStats 队列统计信息
type QueueStats struct {
	Name            string  `json:"name"`
	Size            int64   `json:"size"`
	Capacity        int64   `json:"capacity"`
	DataSize        int64   `json:"data_size"`
	UtilizationRate float64 `json:"utilization_rate"`
	EnqueueRate     float64 `json:"enqueue_rate"`
	DequeueRate     float64 `json:"dequeue_rate"`
}

// StandardQueueManager 标准队列管理器
type StandardQueueManager struct {
	queues map[string]Queue
	mu     sync.RWMutex
}

// NewQueueManager 创建队列管理器
func NewQueueManager(defaultCapacity int64) *StandardQueueManager {
	return &StandardQueueManager{
		queues: make(map[string]Queue),
	}
}

// CreateQueue 创建队列
func (q *StandardQueueManager) CreateQueue(name string, capacity int64) (Queue, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, exists := q.queues[name]; exists {
		return nil, fmt.Errorf("queue %s already exists", name)
	}

	queue := NewBoundedQueue(capacity)
	q.queues[name] = queue

	return queue, nil
}

// GetQueue 获取队列
func (q *StandardQueueManager) GetQueue(name string) (Queue, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	queue, exists := q.queues[name]
	if !exists {
		return nil, fmt.Errorf("queue %s does not exist", name)
	}

	return queue, nil
}

// RemoveQueue 移除队列
func (q *StandardQueueManager) RemoveQueue(name string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, exists := q.queues[name]; !exists {
		return fmt.Errorf("queue %s does not exist", name)
	}

	delete(q.queues, name)
	return nil
}

// GetQueueStats 获取队列统计信息
func (q *StandardQueueManager) GetQueueStats() map[string]QueueStats {
	q.mu.RLock()
	defer q.mu.RUnlock()

	stats := make(map[string]QueueStats)
	for name, queue := range q.queues {
		stats[name] = QueueStats{
			Name:            name,
			Size:            queue.Size(),
			Capacity:        queue.Capacity(),
			DataSize:        queue.DataSize(),
			UtilizationRate: queue.UtilizationRate(),
		}
	}

	return stats
}

// BoundedQueue 有界队列实现
type BoundedQueue struct {
	capacity    int64
	size        int64
	dataSize    int64
	items       chan interface{}
	mu          sync.RWMutex
	enqueueRate float64
	dequeueRate float64
	lastUpdate  time.Time
}

// NewBoundedQueue 创建有界队列
func NewBoundedQueue(capacity int64) *BoundedQueue {
	return &BoundedQueue{
		capacity:   capacity,
		items:      make(chan interface{}, capacity),
		lastUpdate: time.Now(),
	}
}

// Size 获取队列大小
func (q *BoundedQueue) Size() int64 {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.size
}

// DataSize 获取数据大小
func (q *BoundedQueue) DataSize() int64 {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.dataSize
}

// Capacity 获取队列容量
func (q *BoundedQueue) Capacity() int64 {
	return q.capacity
}

// UtilizationRate 获取利用率
func (q *BoundedQueue) UtilizationRate() float64 {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.capacity == 0 {
		return 0
	}

	return float64(q.size) / float64(q.capacity)
}

// Enqueue 入队
func (q *BoundedQueue) Enqueue(item interface{}) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.size >= q.capacity {
		return fmt.Errorf("queue is full")
	}

	select {
	case q.items <- item:
		q.size++
		q.updateRates()
		return nil
	default:
		return fmt.Errorf("queue is full")
	}
}

// Dequeue 出队
func (q *BoundedQueue) Dequeue() (interface{}, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.size == 0 {
		return nil, fmt.Errorf("queue is empty")
	}

	select {
	case item := <-q.items:
		q.size--
		q.updateRates()
		return item, nil
	default:
		return nil, fmt.Errorf("queue is empty")
	}
}

// updateRates 更新速率
func (q *BoundedQueue) updateRates() {
	now := time.Now()
	elapsed := now.Sub(q.lastUpdate).Seconds()

	if elapsed > 0 {
		// 这里可以添加更复杂的速率计算逻辑
		q.lastUpdate = now
	}
}

// PrioritizedQueue 优先级队列
type PrioritizedQueue struct {
	capacity int64
	size     int64
	dataSize int64
	items    []*PriorityItem
	mu       sync.RWMutex
}

// PriorityItem 优先级项
type PriorityItem struct {
	Item     interface{}
	Priority int
	Time     time.Time
}

// NewPrioritizedQueue 创建优先级队列
func NewPrioritizedQueue(capacity int64) *PrioritizedQueue {
	return &PrioritizedQueue{
		capacity: capacity,
		items:    make([]*PriorityItem, 0),
	}
}

// Size 获取队列大小
func (p *PrioritizedQueue) Size() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.size
}

// DataSize 获取数据大小
func (p *PrioritizedQueue) DataSize() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.dataSize
}

// Capacity 获取队列容量
func (p *PrioritizedQueue) Capacity() int64 {
	return p.capacity
}

// UtilizationRate 获取利用率
func (p *PrioritizedQueue) UtilizationRate() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.capacity == 0 {
		return 0
	}

	return float64(p.size) / float64(p.capacity)
}

// Enqueue 入队
func (p *PrioritizedQueue) Enqueue(item interface{}, priority int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.size >= p.capacity {
		return fmt.Errorf("queue is full")
	}

	priorityItem := &PriorityItem{
		Item:     item,
		Priority: priority,
		Time:     time.Now(),
	}

	// 按优先级插入
	p.insertByPriority(priorityItem)
	p.size++

	return nil
}

// Dequeue 出队
func (p *PrioritizedQueue) Dequeue() (interface{}, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.size == 0 {
		return nil, fmt.Errorf("queue is empty")
	}

	// 取出最高优先级的项
	item := p.items[0]
	p.items = p.items[1:]
	p.size--

	return item.Item, nil
}

// insertByPriority 按优先级插入
func (p *PrioritizedQueue) insertByPriority(item *PriorityItem) {
	// 简单的插入排序，实际可以使用更高效的算法
	for i, existing := range p.items {
		if item.Priority > existing.Priority {
			p.items = append(p.items[:i], append([]*PriorityItem{item}, p.items[i:]...)...)
			return
		}
	}

	p.items = append(p.items, item)
}
