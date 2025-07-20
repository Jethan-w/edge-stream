package windowmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/crazy/edge-stream/internal/flowfile"
)

// WindowManager 窗口管理器接口
type WindowManager interface {
	// CreateTimeWindow 创建时间窗口
	CreateTimeWindow(startTime, endTime int64) (Window, error)

	// CreateCountWindow 创建计数窗口
	CreateCountWindow(maxCount int) (Window, error)

	// CreateSessionWindow 创建会话窗口
	CreateSessionWindow(sessionTimeout time.Duration) (Window, error)

	// GetActiveWindows 获取活跃窗口
	GetActiveWindows() []Window

	// SetWindowType 设置窗口类型
	SetWindowType(windowType WindowType)

	// SetWindowSize 设置窗口大小
	SetWindowSize(windowSize int64)

	// SetTimestampExtractor 设置时间戳提取器
	SetTimestampExtractor(extractor TimestampExtractor)

	// SetAggregationStrategy 设置聚合策略
	SetAggregationStrategy(strategy AggregationStrategy)

	// Shutdown 关闭窗口管理器
	Shutdown() error
}

// Window 窗口接口
type Window interface {
	// GetID 获取窗口ID
	GetID() string

	// GetStartTime 获取开始时间
	GetStartTime() int64

	// GetEndTime 获取结束时间
	GetEndTime() int64

	// GetWindowType 获取窗口类型
	GetWindowType() WindowType

	// AddFlowFile 添加 FlowFile
	AddFlowFile(flowFile *flowfile.FlowFile) error

	// GetFlowFiles 获取所有 FlowFile
	GetFlowFiles() []*flowfile.FlowFile

	// IsExpired 检查是否过期
	IsExpired(currentTime int64) bool

	// IsTimeInWindow 检查时间是否在窗口内
	IsTimeInWindow(timestamp int64) bool

	// TriggerComputation 触发计算
	TriggerComputation() error

	// Cleanup 清理窗口资源
	Cleanup() error

	// GetMetadata 获取窗口元数据
	GetMetadata() *WindowMetadata
}

// AbstractWindow 窗口抽象类
type AbstractWindow struct {
	id               string
	creationTime     int64
	flowFiles        []*flowfile.FlowFile
	windowType       WindowType
	mu               sync.RWMutex
	triggerThreshold int
	stateManager     WindowStateManager
}

// NewAbstractWindow 创建抽象窗口
func NewAbstractWindow(windowType WindowType, stateManager WindowStateManager) *AbstractWindow {
	return &AbstractWindow{
		id:               generateWindowID(),
		creationTime:     time.Now().UnixMilli(),
		flowFiles:        make([]*flowfile.FlowFile, 0),
		windowType:       windowType,
		triggerThreshold: 100, // 默认触发阈值
		stateManager:     stateManager,
	}
}

// GetID 获取窗口ID
func (aw *AbstractWindow) GetID() string {
	return aw.id
}

// GetStartTime 获取开始时间
func (aw *AbstractWindow) GetStartTime() int64 {
	return aw.creationTime
}

// GetEndTime 获取结束时间
func (aw *AbstractWindow) GetEndTime() int64 {
	// 子类需要重写此方法
	return aw.creationTime
}

// GetWindowType 获取窗口类型
func (aw *AbstractWindow) GetWindowType() WindowType {
	return aw.windowType
}

// AddFlowFile 添加 FlowFile
func (aw *AbstractWindow) AddFlowFile(flowFile *flowfile.FlowFile) error {
	aw.mu.Lock()
	defer aw.mu.Unlock()

	aw.flowFiles = append(aw.flowFiles, flowFile)

	// 检查触发条件
	aw.checkWindowTriggerCondition()

	// 持久化状态
	if aw.stateManager != nil {
		if err := aw.stateManager.PersistState(aw); err != nil {
			return fmt.Errorf("failed to persist window state: %w", err)
		}
	}

	return nil
}

// GetFlowFiles 获取所有 FlowFile
func (aw *AbstractWindow) GetFlowFiles() []*flowfile.FlowFile {
	aw.mu.RLock()
	defer aw.mu.RUnlock()

	result := make([]*flowfile.FlowFile, len(aw.flowFiles))
	copy(result, aw.flowFiles)
	return result
}

// IsExpired 检查是否过期
func (aw *AbstractWindow) IsExpired(currentTime int64) bool {
	// 子类需要重写此方法
	return false
}

// IsTimeInWindow 检查时间是否在窗口内
func (aw *AbstractWindow) IsTimeInWindow(timestamp int64) bool {
	// 子类需要重写此方法
	return true
}

// TriggerComputation 触发计算
func (aw *AbstractWindow) TriggerComputation() error {
	// 子类可以重写此方法
	return nil
}

// Cleanup 清理窗口资源
func (aw *AbstractWindow) Cleanup() error {
	aw.mu.Lock()
	defer aw.mu.Unlock()

	// 清理 FlowFile 列表
	aw.flowFiles = nil

	// 清理状态
	if aw.stateManager != nil {
		if err := aw.stateManager.DeleteState(aw.GetID()); err != nil {
			return fmt.Errorf("failed to delete window state: %w", err)
		}
	}

	return nil
}

// GetMetadata 获取窗口元数据
func (aw *AbstractWindow) GetMetadata() *WindowMetadata {
	aw.mu.RLock()
	defer aw.mu.RUnlock()

	return &WindowMetadata{
		ID:            aw.id,
		StartTime:     aw.creationTime,
		EndTime:       aw.GetEndTime(),
		Type:          aw.windowType,
		FlowFileCount: len(aw.flowFiles),
		CreationTime:  aw.creationTime,
	}
}

// checkWindowTriggerCondition 检查窗口触发条件
func (aw *AbstractWindow) checkWindowTriggerCondition() {
	if len(aw.flowFiles) >= aw.triggerThreshold {
		if err := aw.TriggerComputation(); err != nil {
			fmt.Printf("Warning: failed to trigger window computation: %v\n", err)
		}
	}
}

// SetTriggerThreshold 设置触发阈值
func (aw *AbstractWindow) SetTriggerThreshold(threshold int) {
	aw.mu.Lock()
	defer aw.mu.Unlock()
	aw.triggerThreshold = threshold
}

// WindowType 窗口类型
type WindowType string

const (
	WindowTypeTumbling   WindowType = "TUMBLING"    // 滚动窗口（固定大小、不重叠）
	WindowTypeSliding    WindowType = "SLIDING"     // 滑动窗口（固定大小、重叠）
	WindowTypeSession    WindowType = "SESSION"     // 会话窗口（动态大小）
	WindowTypeCountBased WindowType = "COUNT_BASED" // 计数窗口（按记录数量）
)

// TimestampType 时间戳类型
type TimestampType string

const (
	TimestampTypeEventTime      TimestampType = "EVENT_TIME"      // 事件发生的实际时间
	TimestampTypeProcessingTime TimestampType = "PROCESSING_TIME" // 系统处理的时间
)

// AggregationFunction 聚合函数
type AggregationFunction string

const (
	AggregationFunctionCount AggregationFunction = "COUNT" // 计数
	AggregationFunctionSum   AggregationFunction = "SUM"   // 求和
	AggregationFunctionAvg   AggregationFunction = "AVG"   // 平均值
	AggregationFunctionMax   AggregationFunction = "MAX"   // 最大值
	AggregationFunctionMin   AggregationFunction = "MIN"   // 最小值
	AggregationFunctionFirst AggregationFunction = "FIRST" // 第一个值
	AggregationFunctionLast  AggregationFunction = "LAST"  // 最后一个值
)

// WindowMetadata 窗口元数据
type WindowMetadata struct {
	ID                  string              `json:"id"`
	StartTime           int64               `json:"start_time"`
	EndTime             int64               `json:"end_time"`
	Type                WindowType          `json:"type"`
	FlowFileCount       int                 `json:"flow_file_count"`
	CreationTime        int64               `json:"creation_time"`
	AggregationFunction AggregationFunction `json:"aggregation_function,omitempty"`
}

// StandardWindowManager 标准窗口管理器实现
type StandardWindowManager struct {
	windowType          WindowType
	windowSize          int64
	timestampExtractor  TimestampExtractor
	aggregationStrategy AggregationStrategy
	stateManager        WindowStateManager
	activeWindows       map[string]Window
	mu                  sync.RWMutex
	ctx                 context.Context
	cancel              context.CancelFunc
	cleanupWorker       *sync.WaitGroup
}

// NewStandardWindowManager 创建标准窗口管理器
func NewStandardWindowManager() *StandardWindowManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &StandardWindowManager{
		activeWindows: make(map[string]Window),
		ctx:           ctx,
		cancel:        cancel,
		cleanupWorker: &sync.WaitGroup{},
	}
}

// CreateTimeWindow 创建时间窗口
func (swm *StandardWindowManager) CreateTimeWindow(startTime, endTime int64) (Window, error) {
	swm.mu.Lock()
	defer swm.mu.Unlock()

	window := NewTimeWindow(startTime, endTime, swm.stateManager)
	swm.activeWindows[window.GetID()] = window

	// 启动清理任务
	swm.startCleanupTask()

	return window, nil
}

// CreateCountWindow 创建计数窗口
func (swm *StandardWindowManager) CreateCountWindow(maxCount int) (Window, error) {
	swm.mu.Lock()
	defer swm.mu.Unlock()

	window := NewCountWindow(maxCount, swm.stateManager)
	swm.activeWindows[window.GetID()] = window

	return window, nil
}

// CreateSessionWindow 创建会话窗口
func (swm *StandardWindowManager) CreateSessionWindow(sessionTimeout time.Duration) (Window, error) {
	swm.mu.Lock()
	defer swm.mu.Unlock()

	window := NewSessionWindow(sessionTimeout, swm.stateManager)
	swm.activeWindows[window.GetID()] = window

	return window, nil
}

// GetActiveWindows 获取活跃窗口
func (swm *StandardWindowManager) GetActiveWindows() []Window {
	swm.mu.RLock()
	defer swm.mu.RUnlock()

	windows := make([]Window, 0, len(swm.activeWindows))
	for _, window := range swm.activeWindows {
		windows = append(windows, window)
	}

	return windows
}

// SetWindowType 设置窗口类型
func (swm *StandardWindowManager) SetWindowType(windowType WindowType) {
	swm.mu.Lock()
	defer swm.mu.Unlock()
	swm.windowType = windowType
}

// SetWindowSize 设置窗口大小
func (swm *StandardWindowManager) SetWindowSize(windowSize int64) {
	swm.mu.Lock()
	defer swm.mu.Unlock()
	swm.windowSize = windowSize
}

// SetTimestampExtractor 设置时间戳提取器
func (swm *StandardWindowManager) SetTimestampExtractor(extractor TimestampExtractor) {
	swm.mu.Lock()
	defer swm.mu.Unlock()
	swm.timestampExtractor = extractor
}

// SetAggregationStrategy 设置聚合策略
func (swm *StandardWindowManager) SetAggregationStrategy(strategy AggregationStrategy) {
	swm.mu.Lock()
	defer swm.mu.Unlock()
	swm.aggregationStrategy = strategy
}

// SetStateManager 设置状态管理器
func (swm *StandardWindowManager) SetStateManager(stateManager WindowStateManager) {
	swm.mu.Lock()
	defer swm.mu.Unlock()
	swm.stateManager = stateManager
}

// Shutdown 关闭窗口管理器
func (swm *StandardWindowManager) Shutdown() error {
	swm.mu.Lock()
	defer swm.mu.Unlock()

	swm.cancel()
	swm.cleanupWorker.Wait()

	// 清理所有活跃窗口
	for _, window := range swm.activeWindows {
		if err := window.Cleanup(); err != nil {
			fmt.Printf("Warning: failed to cleanup window %s: %v\n", window.GetID(), err)
		}
	}

	swm.activeWindows = make(map[string]Window)

	return nil
}

// startCleanupTask 启动清理任务
func (swm *StandardWindowManager) startCleanupTask() {
	swm.cleanupWorker.Add(1)
	go func() {
		defer swm.cleanupWorker.Done()

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-swm.ctx.Done():
				return
			case <-ticker.C:
				swm.cleanupExpiredWindows()
			}
		}
	}()
}

// cleanupExpiredWindows 清理过期窗口
func (swm *StandardWindowManager) cleanupExpiredWindows() {
	swm.mu.Lock()
	defer swm.mu.Unlock()

	currentTime := time.Now().UnixMilli()
	var expiredWindows []string

	for id, window := range swm.activeWindows {
		if window.IsExpired(currentTime) {
			expiredWindows = append(expiredWindows, id)
		}
	}

	for _, id := range expiredWindows {
		window := swm.activeWindows[id]
		if err := window.Cleanup(); err != nil {
			fmt.Printf("Warning: failed to cleanup expired window %s: %v\n", id, err)
		}
		delete(swm.activeWindows, id)
	}
}

// WindowStateManager 窗口状态管理器接口
type WindowStateManager interface {
	// PersistState 持久化窗口状态
	PersistState(window Window) error

	// RestoreState 恢复窗口状态
	RestoreState(windowID string) (Window, error)

	// DeleteState 删除窗口状态
	DeleteState(windowID string) error

	// CleanupExpiredStates 清理过期状态
	CleanupExpiredStates(expirationTime int64) error
}

// 辅助函数
func generateWindowID() string {
	return fmt.Sprintf("window_%d", time.Now().UnixNano())
}

// SerializeWindow 序列化窗口
func SerializeWindow(window Window) ([]byte, error) {
	return json.Marshal(window)
}

// DeserializeWindow 反序列化窗口
func DeserializeWindow(data []byte) (Window, error) {
	// 在实际实现中，这里需要根据窗口类型进行反序列化
	// 为了演示，我们返回一个简单的实现
	return nil, fmt.Errorf("deserialization not implemented")
}
