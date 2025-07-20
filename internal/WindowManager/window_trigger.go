package windowmanager

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// WindowTrigger 窗口触发器接口
type WindowTrigger interface {
	// RegisterWindow 注册窗口
	RegisterWindow(window Window) error

	// UnregisterWindow 注销窗口
	UnregisterWindow(windowID string) error

	// Start 启动触发器
	Start() error

	// Stop 停止触发器
	Stop() error

	// SetTriggerStrategy 设置触发策略
	SetTriggerStrategy(strategy TriggerStrategy)

	// SetOutputProcessor 设置输出处理器
	SetOutputProcessor(processor OutputProcessor)
}

// TriggerStrategy 触发策略接口
type TriggerStrategy interface {
	// ShouldTrigger 判断是否应该触发
	ShouldTrigger(window Window, currentTime int64) bool

	// GetTriggerType 获取触发类型
	GetTriggerType() TriggerType
}

// OutputProcessor 输出处理器接口
type OutputProcessor interface {
	// ProcessWindow 处理窗口
	ProcessWindow(window Window, aggregator WindowAggregator) error
}

// TriggerType 触发类型
type TriggerType string

const (
	TriggerTypeTime   TriggerType = "TIME"   // 时间触发
	TriggerTypeCount  TriggerType = "COUNT"  // 计数触发
	TriggerTypeMixed  TriggerType = "MIXED"  // 混合触发
	TriggerTypeManual TriggerType = "MANUAL" // 手动触发
)

// StandardWindowTrigger 标准窗口触发器
type StandardWindowTrigger struct {
	Windows         map[string]Window
	Strategy        TriggerStrategy
	OutputProcessor OutputProcessor
	Aggregator      WindowAggregator
	ctx             context.Context
	cancel          context.CancelFunc
	mu              sync.RWMutex
	running         bool
	checkInterval   time.Duration
}

// NewStandardWindowTrigger 创建标准窗口触发器
func NewStandardWindowTrigger() *StandardWindowTrigger {
	ctx, cancel := context.WithCancel(context.Background())

	return &StandardWindowTrigger{
		Windows:       make(map[string]Window),
		checkInterval: 1 * time.Second, // 默认检查间隔
		ctx:           ctx,
		cancel:        cancel,
	}
}

// RegisterWindow 注册窗口
func (swt *StandardWindowTrigger) RegisterWindow(window Window) error {
	swt.mu.Lock()
	defer swt.mu.Unlock()

	swt.Windows[window.GetID()] = window
	return nil
}

// UnregisterWindow 注销窗口
func (swt *StandardWindowTrigger) UnregisterWindow(windowID string) error {
	swt.mu.Lock()
	defer swt.mu.Unlock()

	delete(swt.Windows, windowID)
	return nil
}

// Start 启动触发器
func (swt *StandardWindowTrigger) Start() error {
	swt.mu.Lock()
	defer swt.mu.Unlock()

	if swt.running {
		return fmt.Errorf("trigger is already running")
	}

	swt.running = true

	// 启动检查协程
	go swt.checkWindows()

	return nil
}

// Stop 停止触发器
func (swt *StandardWindowTrigger) Stop() error {
	swt.mu.Lock()
	defer swt.mu.Unlock()

	if !swt.running {
		return fmt.Errorf("trigger is not running")
	}

	swt.running = false
	swt.cancel()

	return nil
}

// SetTriggerStrategy 设置触发策略
func (swt *StandardWindowTrigger) SetTriggerStrategy(strategy TriggerStrategy) {
	swt.mu.Lock()
	defer swt.mu.Unlock()
	swt.Strategy = strategy
}

// SetOutputProcessor 设置输出处理器
func (swt *StandardWindowTrigger) SetOutputProcessor(processor OutputProcessor) {
	swt.mu.Lock()
	defer swt.mu.Unlock()
	swt.OutputProcessor = processor
}

// SetAggregator 设置聚合器
func (swt *StandardWindowTrigger) SetAggregator(aggregator WindowAggregator) {
	swt.mu.Lock()
	defer swt.mu.Unlock()
	swt.Aggregator = aggregator
}

// SetCheckInterval 设置检查间隔
func (swt *StandardWindowTrigger) SetCheckInterval(interval time.Duration) {
	swt.mu.Lock()
	defer swt.mu.Unlock()
	swt.checkInterval = interval
}

// checkWindows 检查窗口
func (swt *StandardWindowTrigger) checkWindows() {
	ticker := time.NewTicker(swt.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-swt.ctx.Done():
			return
		case <-ticker.C:
			swt.processWindows()
		}
	}
}

// processWindows 处理窗口
func (swt *StandardWindowTrigger) processWindows() {
	swt.mu.RLock()
	defer swt.mu.RUnlock()

	if swt.Strategy == nil || swt.OutputProcessor == nil || swt.Aggregator == nil {
		return
	}

	currentTime := time.Now().UnixMilli()
	var triggeredWindows []string

	// 检查所有窗口
	for windowID, window := range swt.Windows {
		if swt.Strategy.ShouldTrigger(window, currentTime) {
			triggeredWindows = append(triggeredWindows, windowID)
		}
	}

	// 处理触发的窗口
	for _, windowID := range triggeredWindows {
		window := swt.Windows[windowID]

		// 重置聚合器
		swt.Aggregator.Reset()

		// 处理窗口
		if err := swt.OutputProcessor.ProcessWindow(window, swt.Aggregator); err != nil {
			fmt.Printf("Error processing window %s: %v\n", windowID, err)
		}

		// 注销窗口
		delete(swt.Windows, windowID)
	}
}

// TimeTriggerStrategy 时间触发策略
type TimeTriggerStrategy struct {
	triggerDelay int64 // 触发延迟（毫秒）
}

// NewTimeTriggerStrategy 创建时间触发策略
func NewTimeTriggerStrategy(triggerDelay int64) *TimeTriggerStrategy {
	return &TimeTriggerStrategy{
		triggerDelay: triggerDelay,
	}
}

// ShouldTrigger 判断是否应该触发
func (tts *TimeTriggerStrategy) ShouldTrigger(window Window, currentTime int64) bool {
	return currentTime >= window.GetEndTime()+tts.triggerDelay
}

// GetTriggerType 获取触发类型
func (tts *TimeTriggerStrategy) GetTriggerType() TriggerType {
	return TriggerTypeTime
}

// CountTriggerStrategy 计数触发策略
type CountTriggerStrategy struct {
	threshold int
}

// NewCountTriggerStrategy 创建计数触发策略
func NewCountTriggerStrategy(threshold int) *CountTriggerStrategy {
	return &CountTriggerStrategy{
		threshold: threshold,
	}
}

// ShouldTrigger 判断是否应该触发
func (cts *CountTriggerStrategy) ShouldTrigger(window Window, currentTime int64) bool {
	return len(window.GetFlowFiles()) >= cts.threshold
}

// GetTriggerType 获取触发类型
func (cts *CountTriggerStrategy) GetTriggerType() TriggerType {
	return TriggerTypeCount
}

// MixedTriggerStrategy 混合触发策略
type MixedTriggerStrategy struct {
	timeTrigger  *TimeTriggerStrategy
	countTrigger *CountTriggerStrategy
}

// NewMixedTriggerStrategy 创建混合触发策略
func NewMixedTriggerStrategy(timeDelay int64, countThreshold int) *MixedTriggerStrategy {
	return &MixedTriggerStrategy{
		timeTrigger:  NewTimeTriggerStrategy(timeDelay),
		countTrigger: NewCountTriggerStrategy(countThreshold),
	}
}

// ShouldTrigger 判断是否应该触发
func (mts *MixedTriggerStrategy) ShouldTrigger(window Window, currentTime int64) bool {
	return mts.timeTrigger.ShouldTrigger(window, currentTime) ||
		mts.countTrigger.ShouldTrigger(window, currentTime)
}

// GetTriggerType 获取触发类型
func (mts *MixedTriggerStrategy) GetTriggerType() TriggerType {
	return TriggerTypeMixed
}

// ManualTriggerStrategy 手动触发策略
type ManualTriggerStrategy struct {
	triggeredWindows map[string]bool
	mu               sync.RWMutex
}

// NewManualTriggerStrategy 创建手动触发策略
func NewManualTriggerStrategy() *ManualTriggerStrategy {
	return &ManualTriggerStrategy{
		triggeredWindows: make(map[string]bool),
	}
}

// ShouldTrigger 判断是否应该触发
func (mts *ManualTriggerStrategy) ShouldTrigger(window Window, currentTime int64) bool {
	mts.mu.RLock()
	defer mts.mu.RUnlock()

	return mts.triggeredWindows[window.GetID()]
}

// GetTriggerType 获取触发类型
func (mts *ManualTriggerStrategy) GetTriggerType() TriggerType {
	return TriggerTypeManual
}

// TriggerWindow 手动触发窗口
func (mts *ManualTriggerStrategy) TriggerWindow(windowID string) {
	mts.mu.Lock()
	defer mts.mu.Unlock()

	mts.triggeredWindows[windowID] = true
}

// ClearTrigger 清除触发状态
func (mts *ManualTriggerStrategy) ClearTrigger(windowID string) {
	mts.mu.Lock()
	defer mts.mu.Unlock()

	delete(mts.triggeredWindows, windowID)
}

// WatermarkTriggerStrategy 水印触发策略
type WatermarkTriggerStrategy struct {
	watermarkManager WatermarkManager
	maxLateness      int64
}

// NewWatermarkTriggerStrategy 创建水印触发策略
func NewWatermarkTriggerStrategy(watermarkManager WatermarkManager, maxLateness int64) *WatermarkTriggerStrategy {
	return &WatermarkTriggerStrategy{
		watermarkManager: watermarkManager,
		maxLateness:      maxLateness,
	}
}

// ShouldTrigger 判断是否应该触发
func (wts *WatermarkTriggerStrategy) ShouldTrigger(window Window, currentTime int64) bool {
	watermark := wts.watermarkManager.GetCurrentWatermark()
	return watermark >= window.GetEndTime()+wts.maxLateness
}

// GetTriggerType 获取触发类型
func (wts *WatermarkTriggerStrategy) GetTriggerType() TriggerType {
	return TriggerTypeTime
}

// WatermarkManager 水印管理器接口
type WatermarkManager interface {
	// GetCurrentWatermark 获取当前水印
	GetCurrentWatermark() int64

	// UpdateWatermark 更新水印
	UpdateWatermark(timestamp int64)
}

// SimpleWatermarkManager 简单水印管理器
type SimpleWatermarkManager struct {
	currentWatermark int64
	mu               sync.RWMutex
}

// NewSimpleWatermarkManager 创建简单水印管理器
func NewSimpleWatermarkManager() *SimpleWatermarkManager {
	return &SimpleWatermarkManager{}
}

// GetCurrentWatermark 获取当前水印
func (swm *SimpleWatermarkManager) GetCurrentWatermark() int64 {
	swm.mu.RLock()
	defer swm.mu.RUnlock()
	return swm.currentWatermark
}

// UpdateWatermark 更新水印
func (swm *SimpleWatermarkManager) UpdateWatermark(timestamp int64) {
	swm.mu.Lock()
	defer swm.mu.Unlock()

	if timestamp > swm.currentWatermark {
		swm.currentWatermark = timestamp
	}
}

// TriggerFactory 触发器工厂
type TriggerFactory struct{}

// NewTriggerFactory 创建触发器工厂
func NewTriggerFactory() *TriggerFactory {
	return &TriggerFactory{}
}

// CreateTriggerStrategy 创建触发策略
func (tf *TriggerFactory) CreateTriggerStrategy(triggerType TriggerType, config map[string]interface{}) (TriggerStrategy, error) {
	switch triggerType {
	case TriggerTypeTime:
		delay, _ := config["delay"].(int64)
		return NewTimeTriggerStrategy(delay), nil
	case TriggerTypeCount:
		threshold, _ := config["threshold"].(int)
		return NewCountTriggerStrategy(threshold), nil
	case TriggerTypeMixed:
		delay, _ := config["delay"].(int64)
		threshold, _ := config["threshold"].(int)
		return NewMixedTriggerStrategy(delay, threshold), nil
	case TriggerTypeManual:
		return NewManualTriggerStrategy(), nil
	default:
		return nil, fmt.Errorf("unsupported trigger type: %s", triggerType)
	}
}

// CreateWatermarkTriggerStrategy 创建水印触发策略
func (tf *TriggerFactory) CreateWatermarkTriggerStrategy(watermarkManager WatermarkManager, maxLateness int64) *WatermarkTriggerStrategy {
	return NewWatermarkTriggerStrategy(watermarkManager, maxLateness)
}
