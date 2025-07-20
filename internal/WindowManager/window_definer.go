package windowmanager

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/crazy/edge-stream/internal/flowfile"
)

// TimeWindow 时间窗口实现
type TimeWindow struct {
	*AbstractWindow
	startTime int64
	endTime   int64
}

// NewTimeWindow 创建时间窗口
func NewTimeWindow(startTime, endTime int64, stateManager WindowStateManager) *TimeWindow {
	abstractWindow := NewAbstractWindow(WindowTypeTumbling, stateManager)

	return &TimeWindow{
		AbstractWindow: abstractWindow,
		startTime:      startTime,
		endTime:        endTime,
	}
}

// GetStartTime 获取开始时间
func (tw *TimeWindow) GetStartTime() int64 {
	return tw.startTime
}

// GetEndTime 获取结束时间
func (tw *TimeWindow) GetEndTime() int64 {
	return tw.endTime
}

// IsExpired 检查是否过期
func (tw *TimeWindow) IsExpired(currentTime int64) bool {
	return currentTime > tw.endTime
}

// IsTimeInWindow 检查时间是否在窗口内
func (tw *TimeWindow) IsTimeInWindow(timestamp int64) bool {
	return timestamp >= tw.startTime && timestamp <= tw.endTime
}

// TriggerComputation 触发计算
func (tw *TimeWindow) TriggerComputation() error {
	// 时间窗口的计算逻辑
	fmt.Printf("Time window %s triggered computation at %d\n", tw.GetID(), time.Now().UnixMilli())
	return nil
}

// CountWindow 计数窗口实现
type CountWindow struct {
	*AbstractWindow
	maxCount int
}

// NewCountWindow 创建计数窗口
func NewCountWindow(maxCount int, stateManager WindowStateManager) *CountWindow {
	abstractWindow := NewAbstractWindow(WindowTypeCountBased, stateManager)

	// 设置触发阈值为最大计数
	abstractWindow.SetTriggerThreshold(maxCount)

	return &CountWindow{
		AbstractWindow: abstractWindow,
		maxCount:       maxCount,
	}
}

// GetEndTime 获取结束时间
func (tw *CountWindow) GetEndTime() int64 {
	// 计数窗口没有固定的结束时间，返回创建时间
	return tw.GetStartTime()
}

// IsExpired 检查是否过期
func (tw *CountWindow) IsExpired(currentTime int64) bool {
	// 计数窗口在达到最大计数后过期
	flowFiles := tw.GetFlowFiles()
	return len(flowFiles) >= tw.maxCount
}

// IsTimeInWindow 检查时间是否在窗口内
func (tw *CountWindow) IsTimeInWindow(timestamp int64) bool {
	// 计数窗口接受所有时间的数据，直到达到最大计数
	flowFiles := tw.GetFlowFiles()
	return len(flowFiles) < tw.maxCount
}

// TriggerComputation 触发计算
func (tw *CountWindow) TriggerComputation() error {
	// 计数窗口的计算逻辑
	fmt.Printf("Count window %s triggered computation with %d flow files\n",
		tw.GetID(), len(tw.GetFlowFiles()))
	return nil
}

// SessionWindow 会话窗口实现
type SessionWindow struct {
	*AbstractWindow
	sessionTimeout time.Duration
	lastActivity   int64
	mu             sync.RWMutex
}

// NewSessionWindow 创建会话窗口
func NewSessionWindow(sessionTimeout time.Duration, stateManager WindowStateManager) *SessionWindow {
	abstractWindow := NewAbstractWindow(WindowTypeSession, stateManager)

	return &SessionWindow{
		AbstractWindow: abstractWindow,
		sessionTimeout: sessionTimeout,
		lastActivity:   time.Now().UnixMilli(),
	}
}

// GetEndTime 获取结束时间
func (sw *SessionWindow) GetEndTime() int64 {
	sw.mu.RLock()
	defer sw.mu.RUnlock()

	// 会话窗口的结束时间是最后活动时间加上会话超时时间
	return sw.lastActivity + int64(sw.sessionTimeout.Milliseconds())
}

// IsExpired 检查是否过期
func (sw *SessionWindow) IsExpired(currentTime int64) bool {
	sw.mu.RLock()
	defer sw.mu.RUnlock()

	// 如果当前时间超过最后活动时间加上会话超时时间，则过期
	return currentTime > sw.lastActivity+int64(sw.sessionTimeout.Milliseconds())
}

// IsTimeInWindow 检查时间是否在窗口内
func (sw *SessionWindow) IsTimeInWindow(timestamp int64) bool {
	sw.mu.RLock()
	defer sw.mu.RUnlock()

	// 时间在最后活动时间加上会话超时时间内
	return timestamp <= sw.lastActivity+int64(sw.sessionTimeout.Milliseconds())
}

// AddFlowFile 添加 FlowFile（重写以更新最后活动时间）
func (sw *SessionWindow) AddFlowFile(flowFile *flowfile.FlowFile) error {
	sw.mu.Lock()
	sw.lastActivity = time.Now().UnixMilli()
	sw.mu.Unlock()

	return sw.AbstractWindow.AddFlowFile(flowFile)
}

// TriggerComputation 触发计算
func (sw *SessionWindow) TriggerComputation() error {
	// 会话窗口的计算逻辑
	fmt.Printf("Session window %s triggered computation with %d flow files\n",
		sw.GetID(), len(sw.GetFlowFiles()))
	return nil
}

// SlidingWindow 滑动窗口实现
type SlidingWindow struct {
	*AbstractWindow
	windowSize    int64
	slideInterval int64
	startTime     int64
	endTime       int64
}

// NewSlidingWindow 创建滑动窗口
func NewSlidingWindow(windowSize, slideInterval int64, stateManager WindowStateManager) *SlidingWindow {
	now := time.Now().UnixMilli()
	abstractWindow := NewAbstractWindow(WindowTypeSliding, stateManager)

	return &SlidingWindow{
		AbstractWindow: abstractWindow,
		windowSize:     windowSize,
		slideInterval:  slideInterval,
		startTime:      now,
		endTime:        now + windowSize,
	}
}

// GetStartTime 获取开始时间
func (sw *SlidingWindow) GetStartTime() int64 {
	return sw.startTime
}

// GetEndTime 获取结束时间
func (sw *SlidingWindow) GetEndTime() int64 {
	return sw.endTime
}

// IsExpired 检查是否过期
func (sw *SlidingWindow) IsExpired(currentTime int64) bool {
	return currentTime > sw.endTime
}

// IsTimeInWindow 检查时间是否在窗口内
func (sw *SlidingWindow) IsTimeInWindow(timestamp int64) bool {
	return timestamp >= sw.startTime && timestamp <= sw.endTime
}

// Slide 滑动窗口
func (sw *SlidingWindow) Slide() {
	sw.startTime += sw.slideInterval
	sw.endTime += sw.slideInterval
}

// TriggerComputation 触发计算
func (sw *SlidingWindow) TriggerComputation() error {
	// 滑动窗口的计算逻辑
	fmt.Printf("Sliding window %s triggered computation with %d flow files\n",
		sw.GetID(), len(sw.GetFlowFiles()))
	return nil
}

// WindowFactory 窗口工厂
type WindowFactory struct{}

// NewWindowFactory 创建窗口工厂
func NewWindowFactory() *WindowFactory {
	return &WindowFactory{}
}

// CreateWindow 创建窗口
func (wf *WindowFactory) CreateWindow(windowType WindowType, config WindowConfig, stateManager WindowStateManager) (Window, error) {
	switch windowType {
	case WindowTypeTumbling:
		return wf.createTumblingWindow(config, stateManager)
	case WindowTypeSliding:
		return wf.createSlidingWindow(config, stateManager)
	case WindowTypeSession:
		return wf.createSessionWindow(config, stateManager)
	case WindowTypeCountBased:
		return wf.createCountWindow(config, stateManager)
	default:
		return nil, fmt.Errorf("unsupported window type: %s", windowType)
	}
}

// createTumblingWindow 创建滚动窗口
func (wf *WindowFactory) createTumblingWindow(config WindowConfig, stateManager WindowStateManager) (Window, error) {
	if config.WindowSize <= 0 {
		return nil, fmt.Errorf("invalid window size for tumbling window: %d", config.WindowSize)
	}

	now := time.Now().UnixMilli()
	startTime := now - (now % config.WindowSize)
	endTime := startTime + config.WindowSize

	return NewTimeWindow(startTime, endTime, stateManager), nil
}

// createSlidingWindow 创建滑动窗口
func (wf *WindowFactory) createSlidingWindow(config WindowConfig, stateManager WindowStateManager) (Window, error) {
	if config.WindowSize <= 0 || config.SlideInterval <= 0 {
		return nil, fmt.Errorf("invalid window size or slide interval for sliding window: %d, %d",
			config.WindowSize, config.SlideInterval)
	}

	return NewSlidingWindow(config.WindowSize, config.SlideInterval, stateManager), nil
}

// createSessionWindow 创建会话窗口
func (wf *WindowFactory) createSessionWindow(config WindowConfig, stateManager WindowStateManager) (Window, error) {
	if config.SessionTimeout <= 0 {
		return nil, fmt.Errorf("invalid session timeout: %v", config.SessionTimeout)
	}

	return NewSessionWindow(config.SessionTimeout, stateManager), nil
}

// createCountWindow 创建计数窗口
func (wf *WindowFactory) createCountWindow(config WindowConfig, stateManager WindowStateManager) (Window, error) {
	if config.MaxCount <= 0 {
		return nil, fmt.Errorf("invalid max count for count window: %d", config.MaxCount)
	}

	return NewCountWindow(config.MaxCount, stateManager), nil
}

// WindowConfig 窗口配置
type WindowConfig struct {
	WindowSize       int64         `json:"window_size"`       // 窗口大小（毫秒）
	SlideInterval    int64         `json:"slide_interval"`    // 滑动间隔（毫秒）
	SessionTimeout   time.Duration `json:"session_timeout"`   // 会话超时时间
	MaxCount         int           `json:"max_count"`         // 最大计数
	TriggerThreshold int           `json:"trigger_threshold"` // 触发阈值
}

// NewWindowConfig 创建窗口配置
func NewWindowConfig() *WindowConfig {
	return &WindowConfig{
		WindowSize:       300000, // 默认5分钟
		SlideInterval:    60000,  // 默认1分钟
		SessionTimeout:   5 * time.Minute,
		MaxCount:         1000,
		TriggerThreshold: 100,
	}
}

// WindowTriggerStrategy 窗口触发策略
type WindowTriggerStrategy struct{}

// NewWindowTriggerStrategy 创建窗口触发策略
func NewWindowTriggerStrategy() *WindowTriggerStrategy {
	return &WindowTriggerStrategy{}
}

// ShouldTriggerByTime 基于时间的触发策略
func (wts *WindowTriggerStrategy) ShouldTriggerByTime(window Window, currentTime int64) bool {
	return currentTime >= window.GetEndTime()
}

// ShouldTriggerByCount 基于数量的触发策略
func (wts *WindowTriggerStrategy) ShouldTriggerByCount(window Window, threshold int) bool {
	return len(window.GetFlowFiles()) >= threshold
}

// ShouldTrigger 混合触发策略
func (wts *WindowTriggerStrategy) ShouldTrigger(window Window, currentTime int64, countThreshold int) bool {
	return wts.ShouldTriggerByTime(window, currentTime) ||
		wts.ShouldTriggerByCount(window, countThreshold)
}

// WindowOutputProcessor 窗口输出处理器
type WindowOutputProcessor struct {
	mu sync.Mutex
}

// NewWindowOutputProcessor 创建窗口输出处理器
func NewWindowOutputProcessor() *WindowOutputProcessor {
	return &WindowOutputProcessor{}
}

// ProcessWindow 处理窗口
func (wop *WindowOutputProcessor) ProcessWindow(window Window, aggregator WindowAggregator) error {
	wop.mu.Lock()
	defer wop.mu.Unlock()
	// 获取聚合结果
	aggregationResult := aggregator.GetResult()

	// 创建输出 FlowFile
	// outputFlowFile := wop.createOutputFlowFile(aggregationResult, window)

	// 输出结果（在实际实现中，这里会传递给下游处理器）
	fmt.Printf("Window %s processed with result: %v\n", window.GetID(), aggregationResult)

	// 清理窗口资源
	if err := window.Cleanup(); err != nil {
		return fmt.Errorf("failed to cleanup window: %w", err)
	}

	return nil
}

// createOutputFlowFile 创建输出 FlowFile
func (wop *WindowOutputProcessor) createOutputFlowFile(result interface{}, window Window) *flowfile.FlowFile {
	// 序列化结果
	resultData, err := json.Marshal(result)
	if err != nil {
		fmt.Printf("Warning: failed to marshal aggregation result: %v\n", err)
		resultData = []byte("{}")
	}

	// 创建属性
	attributes := map[string]string{
		"window.id":        window.GetID(),
		"window.type":      string(window.GetWindowType()),
		"window.startTime": fmt.Sprintf("%d", window.GetStartTime()),
		"window.endTime":   fmt.Sprintf("%d", window.GetEndTime()),
		"flowFile.count":   fmt.Sprintf("%d", len(window.GetFlowFiles())),
	}
	f := flowfile.NewFlowFile()
	f.Content = resultData
	f.Attributes = attributes
	return f
}
