package stream_engine

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ProcessorNode 处理器节点接口
type ProcessorNode interface {
	// GetID 获取处理器ID
	GetID() string

	// GetName 获取处理器名称
	GetName() string

	// GetType 获取处理器类型
	GetType() string

	// OnTrigger 触发处理器
	OnTrigger(context ProcessContext) error

	// Start 启动处理器
	Start(ctx context.Context) error

	// Stop 停止处理器
	Stop(ctx context.Context) error

	// Pause 暂停处理器
	Pause() error

	// Resume 恢复处理器
	Resume() error

	// GetStatus 获取处理器状态
	GetStatus() ProcessorStatus

	// GetInputConnections 获取输入连接
	GetInputConnections() []Connection

	// GetOutputConnections 获取输出连接
	GetOutputConnections() []Connection
}

// ProcessorStatus 处理器状态
type ProcessorStatus string

const (
	ProcessorStatusStopped  ProcessorStatus = "STOPPED"
	ProcessorStatusStarting ProcessorStatus = "STARTING"
	ProcessorStatusRunning  ProcessorStatus = "RUNNING"
	ProcessorStatusPaused   ProcessorStatus = "PAUSED"
	ProcessorStatusStopping ProcessorStatus = "STOPPING"
	ProcessorStatusFailed   ProcessorStatus = "FAILED"
)

// BaseProcessorNode 基础处理器节点
type BaseProcessorNode struct {
	id                string
	name              string
	processorType     string
	status            ProcessorStatus
	inputConnections  []Connection
	outputConnections []Connection
	mu                sync.RWMutex
	ctx               context.Context
	cancel            context.CancelFunc
}

// NewBaseProcessorNode 创建基础处理器节点
func NewBaseProcessorNode(id, name, processorType string) *BaseProcessorNode {
	return &BaseProcessorNode{
		id:            id,
		name:          name,
		processorType: processorType,
		status:        ProcessorStatusStopped,
	}
}

// GetID 获取处理器ID
func (p *BaseProcessorNode) GetID() string {
	return p.id
}

// GetName 获取处理器名称
func (p *BaseProcessorNode) GetName() string {
	return p.name
}

// GetType 获取处理器类型
func (p *BaseProcessorNode) GetType() string {
	return p.processorType
}

// OnTrigger 触发处理器
func (p *BaseProcessorNode) OnTrigger(context ProcessContext) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.status != ProcessorStatusRunning {
		return fmt.Errorf("processor %s is not running, current status: %s", p.id, p.status)
	}

	// 子类需要重写此方法实现具体逻辑
	return fmt.Errorf("OnTrigger method must be implemented by subclass")
}

// Start 启动处理器
func (p *BaseProcessorNode) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.status != ProcessorStatusStopped && p.status != ProcessorStatusPaused {
		return fmt.Errorf("processor cannot start from status: %s", p.status)
	}

	p.status = ProcessorStatusStarting
	p.ctx, p.cancel = context.WithCancel(ctx)

	// 执行启动逻辑
	if err := p.doStart(); err != nil {
		p.status = ProcessorStatusFailed
		return fmt.Errorf("failed to start processor: %w", err)
	}

	p.status = ProcessorStatusRunning
	return nil
}

// Stop 停止处理器
func (p *BaseProcessorNode) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.status == ProcessorStatusStopped {
		return nil
	}

	p.status = ProcessorStatusStopping

	// 取消上下文
	if p.cancel != nil {
		p.cancel()
	}

	// 执行停止逻辑
	if err := p.doStop(); err != nil {
		return fmt.Errorf("failed to stop processor: %w", err)
	}

	p.status = ProcessorStatusStopped
	return nil
}

// Pause 暂停处理器
func (p *BaseProcessorNode) Pause() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.status != ProcessorStatusRunning {
		return fmt.Errorf("processor is not running, current status: %s", p.status)
	}

	p.status = ProcessorStatusPaused

	// 执行暂停逻辑
	if err := p.doPause(); err != nil {
		return fmt.Errorf("failed to pause processor: %w", err)
	}

	return nil
}

// Resume 恢复处理器
func (p *BaseProcessorNode) Resume() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.status != ProcessorStatusPaused {
		return fmt.Errorf("processor is not paused, current status: %s", p.status)
	}

	p.status = ProcessorStatusRunning

	// 执行恢复逻辑
	if err := p.doResume(); err != nil {
		return fmt.Errorf("failed to resume processor: %w", err)
	}

	return nil
}

// GetStatus 获取处理器状态
func (p *BaseProcessorNode) GetStatus() ProcessorStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.status
}

// GetInputConnections 获取输入连接
func (p *BaseProcessorNode) GetInputConnections() []Connection {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.inputConnections
}

// GetOutputConnections 获取输出连接
func (p *BaseProcessorNode) GetOutputConnections() []Connection {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.outputConnections
}

// AddInputConnection 添加输入连接
func (p *BaseProcessorNode) AddInputConnection(connection Connection) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.inputConnections = append(p.inputConnections, connection)
}

// AddOutputConnection 添加输出连接
func (p *BaseProcessorNode) AddOutputConnection(connection Connection) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.outputConnections = append(p.outputConnections, connection)
}

// doStart 执行启动逻辑（子类可重写）
func (p *BaseProcessorNode) doStart() error {
	// 默认实现为空
	return nil
}

// doStop 执行停止逻辑（子类可重写）
func (p *BaseProcessorNode) doStop() error {
	// 默认实现为空
	return nil
}

// doPause 执行暂停逻辑（子类可重写）
func (p *BaseProcessorNode) doPause() error {
	// 默认实现为空
	return nil
}

// doResume 执行恢复逻辑（子类可重写）
func (p *BaseProcessorNode) doResume() error {
	// 默认实现为空
	return nil
}

// StandardConnection 标准连接实现
type StandardConnection struct {
	id        string
	source    ProcessorNode
	target    ProcessorNode
	queue     Queue
	rateLimit int
	paused    bool
	mu        sync.RWMutex
}

// NewStandardConnection 创建标准连接
func NewStandardConnection(id string, source, target ProcessorNode, capacity int64) *StandardConnection {
	return &StandardConnection{
		id:        id,
		source:    source,
		target:    target,
		queue:     NewBoundedQueue(capacity),
		rateLimit: 1000, // 默认每秒1000个消息
		paused:    false,
	}
}

// GetID 获取连接ID
func (c *StandardConnection) GetID() string {
	return c.id
}

// GetSource 获取源处理器
func (c *StandardConnection) GetSource() ProcessorNode {
	return c.source
}

// GetTarget 获取目标处理器
func (c *StandardConnection) GetTarget() ProcessorNode {
	return c.target
}

// GetQueue 获取队列
func (c *StandardConnection) GetQueue() Queue {
	return c.queue
}

// Pause 暂停连接
func (c *StandardConnection) Pause() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.paused {
		return fmt.Errorf("connection is already paused")
	}

	c.paused = true
	return nil
}

// Resume 恢复连接
func (c *StandardConnection) Resume() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.paused {
		return fmt.Errorf("connection is not paused")
	}

	c.paused = false
	return nil
}

// SetRateLimit 设置速率限制
func (c *StandardConnection) SetRateLimit(rate int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if rate <= 0 {
		return fmt.Errorf("rate limit must be positive")
	}

	c.rateLimit = rate
	return nil
}

// IsPaused 是否暂停
func (c *StandardConnection) IsPaused() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.paused
}

// GetRateLimit 获取速率限制
func (c *StandardConnection) GetRateLimit() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.rateLimit
}

// EventDrivenScheduler 事件驱动调度器
type EventDrivenScheduler struct {
	processors map[string]ProcessorNode
	triggers   map[string]Trigger
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	running    bool
}

// NewEventDrivenScheduler 创建事件驱动调度器
func NewEventDrivenScheduler() *EventDrivenScheduler {
	return &EventDrivenScheduler{
		processors: make(map[string]ProcessorNode),
		triggers:   make(map[string]Trigger),
	}
}

// Start 启动事件驱动调度器
func (e *EventDrivenScheduler) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return fmt.Errorf("event driven scheduler is already running")
	}

	e.ctx, e.cancel = context.WithCancel(ctx)
	e.running = true

	// 启动事件监听协程
	go e.monitorEvents()

	return nil
}

// Stop 停止事件驱动调度器
func (e *EventDrivenScheduler) Stop(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return nil
	}

	e.running = false

	if e.cancel != nil {
		e.cancel()
	}

	return nil
}

// Schedule 调度处理器
func (e *EventDrivenScheduler) Schedule(processor ProcessorNode, trigger Trigger) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	processorID := processor.GetID()
	e.processors[processorID] = processor
	e.triggers[processorID] = trigger

	return nil
}

// Unschedule 取消调度
func (e *EventDrivenScheduler) Unschedule(processorID string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.processors, processorID)
	delete(e.triggers, processorID)
}

// monitorEvents 监控事件
func (e *EventDrivenScheduler) monitorEvents() {
	ticker := time.NewTicker(time.Millisecond * 100)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			e.checkTriggers()
		}
	}
}

// checkTriggers 检查触发器
func (e *EventDrivenScheduler) checkTriggers() {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for processorID, processor := range e.processors {
		trigger := e.triggers[processorID]

		// 创建处理上下文
		context := ProcessContext{
			ProcessorID: processorID,
			LastTrigger: time.Now(),
			Attributes:  make(map[string]interface{}),
		}

		// 检查是否应该触发
		if trigger.ShouldFire(context) {
			// 异步执行处理器
			go func(p ProcessorNode, ctx ProcessContext) {
				if err := p.OnTrigger(ctx); err != nil {
					// 记录错误
				}
			}(processor, context)
		}
	}
}

// TimerScheduler 定时调度器
type TimerScheduler struct {
	processors map[string]ProcessorNode
	triggers   map[string]Trigger
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	running    bool
}

// NewTimerScheduler 创建定时调度器
func NewTimerScheduler() *TimerScheduler {
	return &TimerScheduler{
		processors: make(map[string]ProcessorNode),
		triggers:   make(map[string]Trigger),
	}
}

// Start 启动定时调度器
func (t *TimerScheduler) Start(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.running {
		return fmt.Errorf("timer scheduler is already running")
	}

	t.ctx, t.cancel = context.WithCancel(ctx)
	t.running = true

	// 启动定时检查协程
	go t.checkTimers()

	return nil
}

// Stop 停止定时调度器
func (t *TimerScheduler) Stop(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.running {
		return nil
	}

	t.running = false

	if t.cancel != nil {
		t.cancel()
	}

	return nil
}

// Schedule 调度处理器
func (t *TimerScheduler) Schedule(processor ProcessorNode, trigger Trigger) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	processorID := processor.GetID()
	t.processors[processorID] = processor
	t.triggers[processorID] = trigger

	return nil
}

// Unschedule 取消调度
func (t *TimerScheduler) Unschedule(processorID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.processors, processorID)
	delete(t.triggers, processorID)
}

// checkTimers 检查定时器
func (t *TimerScheduler) checkTimers() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-t.ctx.Done():
			return
		case <-ticker.C:
			t.checkTriggers()
		}
	}
}

// checkTriggers 检查触发器
func (t *TimerScheduler) checkTriggers() {
	t.mu.RLock()
	defer t.mu.RUnlock()

	for processorID, processor := range t.processors {
		trigger := t.triggers[processorID]

		// 创建处理上下文
		context := ProcessContext{
			ProcessorID: processorID,
			LastTrigger: time.Now(),
			Attributes:  make(map[string]interface{}),
		}

		// 检查是否应该触发
		if trigger.ShouldFire(context) {
			// 异步执行处理器
			go func(p ProcessorNode, ctx ProcessContext) {
				if err := p.OnTrigger(ctx); err != nil {
					// 记录错误
				}
			}(processor, context)

			// 更新触发器最后触发时间
			if timerTrigger, ok := trigger.(*TimerDrivenTrigger); ok {
				timerTrigger.UpdateLastFire()
			}
		}
	}
}
