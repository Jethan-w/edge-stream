package stream_engine

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// BackpressureManager 背压管理器
type BackpressureManager struct {
	threshold   float64
	strategies  map[string]BackpressureStrategy
	connections map[string]*ConnectionState
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	running     bool
}

// BackpressureStrategy 背压策略
type BackpressureStrategy interface {
	// ShouldApply 是否应该应用背压
	ShouldApply(queue Queue) bool

	// Apply 应用背压
	Apply(connection Connection) error

	// Release 释放背压
	Release(connection Connection) error

	// GetType 获取策略类型
	GetType() BackpressureType
}

// BackpressureType 背压类型
type BackpressureType string

const (
	BackpressureTypePause    BackpressureType = "PAUSE"     // 暂停
	BackpressureTypeSlowDown BackpressureType = "SLOW_DOWN" // 减速
	BackpressureTypeDrop     BackpressureType = "DROP"      // 丢弃
	BackpressureTypeRedirect BackpressureType = "REDIRECT"  // 重定向
)

// ConnectionState 连接状态
type ConnectionState struct {
	ConnectionID        string
	QueueSize           int64
	DataSize            int64
	LastUpdate          time.Time
	BackpressureApplied bool
	Strategy            BackpressureStrategy
}

// Queue 队列接口
type Queue interface {
	// Size 获取队列大小
	Size() int64

	// DataSize 获取数据大小
	DataSize() int64

	// Capacity 获取队列容量
	Capacity() int64

	// UtilizationRate 获取利用率
	UtilizationRate() float64
}

// Connection 连接接口
type Connection interface {
	// GetID 获取连接ID
	GetID() string

	// GetSource 获取源处理器
	GetSource() ProcessorNode

	// GetTarget 获取目标处理器
	GetTarget() ProcessorNode

	// GetQueue 获取队列
	GetQueue() Queue

	// Pause 暂停连接
	Pause() error

	// Resume 恢复连接
	Resume() error

	// SetRateLimit 设置速率限制
	SetRateLimit(rate int) error
}

// NewBackpressureManager 创建背压管理器
func NewBackpressureManager(threshold float64) *BackpressureManager {
	return &BackpressureManager{
		threshold:   threshold,
		strategies:  make(map[string]BackpressureStrategy),
		connections: make(map[string]*ConnectionState),
	}
}

// Start 启动背压管理器
func (b *BackpressureManager) Start(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.running {
		return fmt.Errorf("backpressure manager is already running")
	}

	b.ctx, b.cancel = context.WithCancel(ctx)
	b.running = true

	// 启动监控协程
	go b.monitor()

	return nil
}

// Stop 停止背压管理器
func (b *BackpressureManager) Stop(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.running {
		return nil
	}

	b.running = false

	// 取消上下文
	if b.cancel != nil {
		b.cancel()
	}

	// 释放所有背压
	for _, state := range b.connections {
		if state.BackpressureApplied {
			b.releaseBackpressure(state.ConnectionID)
		}
	}

	return nil
}

// ApplyBackpressure 对连接应用背压
func (b *BackpressureManager) ApplyBackpressure(connection Connection) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	connectionID := connection.GetID()
	queue := connection.GetQueue()

	// 检查是否应该应用背压
	if !b.shouldApplyBackpressure(queue) {
		return nil
	}

	// 获取或创建连接状态
	state, exists := b.connections[connectionID]
	if !exists {
		state = &ConnectionState{
			ConnectionID: connectionID,
			LastUpdate:   time.Now(),
		}
		b.connections[connectionID] = state
	}

	// 更新状态
	state.QueueSize = queue.Size()
	state.DataSize = queue.DataSize()
	state.LastUpdate = time.Now()

	// 如果已经应用了背压，不需要重复应用
	if state.BackpressureApplied {
		return nil
	}

	// 选择背压策略
	strategy := b.selectStrategy(queue)
	state.Strategy = strategy

	// 应用背压
	if err := strategy.Apply(connection); err != nil {
		return fmt.Errorf("failed to apply backpressure: %w", err)
	}

	state.BackpressureApplied = true

	return nil
}

// ReleaseBackpressure 释放连接背压
func (b *BackpressureManager) ReleaseBackpressure(connection Connection) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	connectionID := connection.GetID()
	return b.releaseBackpressure(connectionID)
}

// releaseBackpressure 释放背压（内部方法）
func (b *BackpressureManager) releaseBackpressure(connectionID string) error {
	state, exists := b.connections[connectionID]
	if !exists || !state.BackpressureApplied {
		return nil
	}

	// 释放背压
	if state.Strategy != nil {
		// 这里需要获取连接对象，简化处理
		// 实际实现中需要维护连接映射
	}

	state.BackpressureApplied = false
	return nil
}

// RegisterStrategy 注册背压策略
func (b *BackpressureManager) RegisterStrategy(name string, strategy BackpressureStrategy) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.strategies[name]; exists {
		return fmt.Errorf("strategy %s is already registered", name)
	}

	b.strategies[name] = strategy
	return nil
}

// shouldApplyBackpressure 检查是否应该应用背压
func (b *BackpressureManager) shouldApplyBackpressure(queue Queue) bool {
	utilizationRate := queue.UtilizationRate()
	return utilizationRate >= b.threshold
}

// selectStrategy 选择背压策略
func (b *BackpressureManager) selectStrategy(queue Queue) BackpressureStrategy {
	utilizationRate := queue.UtilizationRate()

	// 根据利用率选择策略
	if utilizationRate >= 0.95 {
		// 严重过载，暂停
		return &PauseStrategy{}
	} else if utilizationRate >= 0.8 {
		// 过载，减速
		return &SlowDownStrategy{}
	} else if utilizationRate >= 0.7 {
		// 接近过载，重定向
		return &RedirectStrategy{}
	} else {
		// 轻微过载，丢弃
		return &DropStrategy{}
	}
}

// monitor 监控协程
func (b *BackpressureManager) monitor() {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for {
		select {
		case <-b.ctx.Done():
			return
		case <-ticker.C:
			b.checkAndAdjustBackpressure()
		}
	}
}

// checkAndAdjustBackpressure 检查并调整背压
func (b *BackpressureManager) checkAndAdjustBackpressure() {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for connectionID, state := range b.connections {
		// 检查是否需要释放背压
		if state.BackpressureApplied && time.Since(state.LastUpdate) > time.Minute*5 {
			// 5分钟没有更新，可能连接已经恢复
			go b.releaseBackpressure(connectionID)
		}
	}
}

// PauseStrategy 暂停策略
type PauseStrategy struct{}

// ShouldApply 是否应该应用背压
func (p *PauseStrategy) ShouldApply(queue Queue) bool {
	return queue.UtilizationRate() >= 0.95
}

// Apply 应用背压
func (p *PauseStrategy) Apply(connection Connection) error {
	return connection.Pause()
}

// Release 释放背压
func (p *PauseStrategy) Release(connection Connection) error {
	return connection.Resume()
}

// GetType 获取策略类型
func (p *PauseStrategy) GetType() BackpressureType {
	return BackpressureTypePause
}

// SlowDownStrategy 减速策略
type SlowDownStrategy struct{}

// ShouldApply 是否应该应用背压
func (s *SlowDownStrategy) ShouldApply(queue Queue) bool {
	return queue.UtilizationRate() >= 0.8
}

// Apply 应用背压
func (s *SlowDownStrategy) Apply(connection Connection) error {
	// 设置较低的速率限制
	return connection.SetRateLimit(100) // 每秒100个消息
}

// Release 释放背压
func (s *SlowDownStrategy) Release(connection Connection) error {
	// 恢复正常速率
	return connection.SetRateLimit(1000) // 每秒1000个消息
}

// GetType 获取策略类型
func (s *SlowDownStrategy) GetType() BackpressureType {
	return BackpressureTypeSlowDown
}

// DropStrategy 丢弃策略
type DropStrategy struct{}

// ShouldApply 是否应该应用背压
func (d *DropStrategy) ShouldApply(queue Queue) bool {
	return queue.UtilizationRate() >= 0.7
}

// Apply 应用背压
func (d *DropStrategy) Apply(connection Connection) error {
	// 丢弃策略通常不需要特殊处理
	// 在数据到达时进行丢弃
	return nil
}

// Release 释放背压
func (d *DropStrategy) Release(connection Connection) error {
	// 停止丢弃
	return nil
}

// GetType 获取策略类型
func (d *DropStrategy) GetType() BackpressureType {
	return BackpressureTypeDrop
}

// RedirectStrategy 重定向策略
type RedirectStrategy struct{}

// ShouldApply 是否应该应用背压
func (r *RedirectStrategy) ShouldApply(queue Queue) bool {
	return queue.UtilizationRate() >= 0.7
}

// Apply 应用背压
func (r *RedirectStrategy) Apply(connection Connection) error {
	// 重定向到备用处理器
	// 这里需要实现重定向逻辑
	return nil
}

// Release 释放背压
func (r *RedirectStrategy) Release(connection Connection) error {
	// 恢复正常路由
	return nil
}

// GetType 获取策略类型
func (r *RedirectStrategy) GetType() BackpressureType {
	return BackpressureTypeRedirect
}

// AdvancedBackpressureController 高级背压控制器
type AdvancedBackpressureController struct {
	manager *BackpressureManager
	mu      sync.RWMutex
}

// NewAdvancedBackpressureController 创建高级背压控制器
func NewAdvancedBackpressureController(threshold float64) *AdvancedBackpressureController {
	return &AdvancedBackpressureController{
		manager: NewBackpressureManager(threshold),
	}
}

// DetermineStrategy 确定背压策略
func (c *AdvancedBackpressureController) DetermineStrategy(queue Queue) BackpressureStrategy {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 多维度分析
	if c.isQueueOverloaded(queue) {
		return &PauseStrategy{}
	}
	if c.isHighLatency(queue) {
		return &SlowDownStrategy{}
	}
	if c.isMemoryPressure(queue) {
		return &DropStrategy{}
	}

	return &RedirectStrategy{}
}

// isQueueOverloaded 检查队列是否过载
func (c *AdvancedBackpressureController) isQueueOverloaded(queue Queue) bool {
	return queue.UtilizationRate() >= 0.9
}

// isHighLatency 检查是否有高延迟
func (c *AdvancedBackpressureController) isHighLatency(queue Queue) bool {
	// 这里需要实现延迟检测逻辑
	return false
}

// isMemoryPressure 检查是否有内存压力
func (c *AdvancedBackpressureController) isMemoryPressure(queue Queue) bool {
	// 这里需要实现内存压力检测逻辑
	return false
}
