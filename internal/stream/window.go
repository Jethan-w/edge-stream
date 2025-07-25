package stream

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// StandardWindowProcessor 标准窗口处理器
type StandardWindowProcessor struct {
	config        *WindowConfig
	windows       map[string]*Window
	activeWindows []string
	mutex         sync.RWMutex
	lastCleanup   time.Time
}

// NewStandardWindowProcessor 创建新的标准窗口处理器
func NewStandardWindowProcessor(config *WindowConfig) *StandardWindowProcessor {
	return &StandardWindowProcessor{
		config:        config,
		windows:       make(map[string]*Window),
		activeWindows: make([]string, 0),
		lastCleanup:   time.Now(),
	}
}

// AddMessage 添加消息到窗口
func (w *StandardWindowProcessor) AddMessage(message *Message) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	// 根据窗口类型处理消息
	switch w.config.Type {
	case WindowTypeTumbling:
		return w.addToTumblingWindow(message)
	case WindowTypeSliding:
		return w.addToSlidingWindow(message)
	case WindowTypeSession:
		return w.addToSessionWindow(message)
	default:
		return fmt.Errorf("unsupported window type: %s", w.config.Type)
	}
}

// addToTumblingWindow 添加消息到滚动窗口
func (w *StandardWindowProcessor) addToTumblingWindow(message *Message) error {
	// 计算窗口开始时间
	windowStart := message.Timestamp.Truncate(w.config.Size)
	windowEnd := windowStart.Add(w.config.Size)
	windowID := fmt.Sprintf("tumbling_%d", windowStart.Unix())

	// 获取或创建窗口
	window, exists := w.windows[windowID]
	if !exists {
		window = &Window{
			ID:        windowID,
			Type:      WindowTypeTumbling,
			StartTime: windowStart,
			EndTime:   windowEnd,
			Size:      w.config.Size,
			Messages:  make([]*Message, 0),
		}
		w.windows[windowID] = window
		w.activeWindows = append(w.activeWindows, windowID)
	}

	// 检查窗口大小限制
	if w.config.MaxSize > 0 && len(window.Messages) >= w.config.MaxSize {
		return fmt.Errorf("window %s has reached maximum size %d", windowID, w.config.MaxSize)
	}

	// 添加消息到窗口
	window.Messages = append(window.Messages, message)

	return nil
}

// addToSlidingWindow 添加消息到滑动窗口
func (w *StandardWindowProcessor) addToSlidingWindow(message *Message) error {
	// 计算所有可能的窗口
	slideInterval := w.config.Slide
	if slideInterval == 0 {
		slideInterval = w.config.Size / 4 // 默认滑动间隔为窗口大小的1/4
	}

	// 计算消息应该属于的窗口
	currentTime := message.Timestamp
	for i := int64(0); i < int64(w.config.Size/slideInterval); i++ {
		windowStart := currentTime.Add(-time.Duration(i) * slideInterval).Truncate(slideInterval)
		windowEnd := windowStart.Add(w.config.Size)

		// 检查消息是否在这个窗口内
		if message.Timestamp.After(windowStart) && message.Timestamp.Before(windowEnd) {
			windowID := fmt.Sprintf("sliding_%d_%d", windowStart.Unix(), w.config.Size.Nanoseconds())

			// 获取或创建窗口
			window, exists := w.windows[windowID]
			if !exists {
				window = &Window{
					ID:        windowID,
					Type:      WindowTypeSliding,
					StartTime: windowStart,
					EndTime:   windowEnd,
					Size:      w.config.Size,
					Slide:     slideInterval,
					Messages:  make([]*Message, 0),
				}
				w.windows[windowID] = window
				w.activeWindows = append(w.activeWindows, windowID)
			}

			// 检查窗口大小限制
			if w.config.MaxSize > 0 && len(window.Messages) >= w.config.MaxSize {
				continue
			}

			// 添加消息到窗口
			window.Messages = append(window.Messages, message)
		}
	}

	return nil
}

// addToSessionWindow 添加消息到会话窗口
func (w *StandardWindowProcessor) addToSessionWindow(message *Message) error {
	sessionGap := w.config.SessionGap
	if sessionGap == 0 {
		sessionGap = 30 * time.Minute // 默认会话间隔30分钟
	}

	// 查找可以合并的现有会话窗口
	var targetWindow *Window
	var targetWindowID string

	for windowID, window := range w.windows {
		if window.Type != WindowTypeSession {
			continue
		}

		// 检查消息是否可以加入这个会话窗口
		if len(window.Messages) > 0 {
			lastMessage := window.Messages[len(window.Messages)-1]
			if message.Timestamp.Sub(lastMessage.Timestamp) <= sessionGap {
				targetWindow = window
				targetWindowID = windowID
				break
			}
		}
	}

	// 如果没有找到合适的会话窗口，创建新的
	if targetWindow == nil {
		windowID := fmt.Sprintf("session_%d_%d", message.Timestamp.Unix(), time.Now().UnixNano())
		targetWindow = &Window{
			ID:        windowID,
			Type:      WindowTypeSession,
			StartTime: message.Timestamp,
			EndTime:   message.Timestamp.Add(sessionGap),
			Messages:  make([]*Message, 0),
		}
		w.windows[windowID] = targetWindow
		w.activeWindows = append(w.activeWindows, windowID)
		targetWindowID = windowID
	}

	// 检查窗口大小限制
	if w.config.MaxSize > 0 && len(targetWindow.Messages) >= w.config.MaxSize {
		return fmt.Errorf("session window %s has reached maximum size %d", targetWindowID, w.config.MaxSize)
	}

	// 添加消息到窗口并更新窗口时间
	targetWindow.Messages = append(targetWindow.Messages, message)
	if message.Timestamp.Before(targetWindow.StartTime) {
		targetWindow.StartTime = message.Timestamp
	}
	if message.Timestamp.After(targetWindow.EndTime.Add(-sessionGap)) {
		targetWindow.EndTime = message.Timestamp.Add(sessionGap)
	}

	return nil
}

// GetWindow 获取指定窗口
func (w *StandardWindowProcessor) GetWindow(id string) (*Window, error) {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	window, exists := w.windows[id]
	if !exists {
		return nil, fmt.Errorf("window '%s' not found", id)
	}

	// 返回窗口副本
	windowCopy := *window
	windowCopy.Messages = make([]*Message, len(window.Messages))
	copy(windowCopy.Messages, window.Messages)

	return &windowCopy, nil
}

// GetActiveWindows 获取所有活跃窗口
func (w *StandardWindowProcessor) GetActiveWindows() ([]*Window, error) {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	windows := make([]*Window, 0, len(w.activeWindows))
	for _, windowID := range w.activeWindows {
		if window, exists := w.windows[windowID]; exists {
			// 返回窗口副本
			windowCopy := *window
			windowCopy.Messages = make([]*Message, len(window.Messages))
			copy(windowCopy.Messages, window.Messages)
			windows = append(windows, &windowCopy)
		}
	}

	return windows, nil
}

// CloseExpiredWindows 关闭过期窗口
func (w *StandardWindowProcessor) CloseExpiredWindows(now time.Time) ([]*Window, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	expiredWindows := make([]*Window, 0)
	newActiveWindows := make([]string, 0)

	for _, windowID := range w.activeWindows {
		window, exists := w.windows[windowID]
		if !exists {
			continue
		}

		// 检查窗口是否过期
		expired := false
		switch window.Type {
		case WindowTypeTumbling, WindowTypeSliding:
			// 对于滚动和滑动窗口，检查是否超过了水印时间
			watermark := w.config.Watermark
			if watermark == 0 {
				watermark = 5 * time.Second // 默认水印5秒
			}
			expired = now.Sub(window.EndTime) > watermark
		case WindowTypeSession:
			// 对于会话窗口，检查是否超过了会话间隔
			sessionGap := w.config.SessionGap
			if sessionGap == 0 {
				sessionGap = 30 * time.Minute
			}
			expired = now.Sub(window.EndTime) > sessionGap
		}

		if expired {
			// 窗口过期，添加到过期列表
			windowCopy := *window
			windowCopy.Messages = make([]*Message, len(window.Messages))
			copy(windowCopy.Messages, window.Messages)
			expiredWindows = append(expiredWindows, &windowCopy)

			// 从活跃窗口和窗口映射中删除
			delete(w.windows, windowID)
		} else {
			// 窗口仍然活跃
			newActiveWindows = append(newActiveWindows, windowID)
		}
	}

	w.activeWindows = newActiveWindows
	w.lastCleanup = now

	return expiredWindows, nil
}

// Aggregate 对窗口进行聚合
func (w *StandardWindowProcessor) Aggregate(window *Window, aggType AggregationType, field string) (*AggregationResult, error) {
	if len(window.Messages) == 0 {
		return &AggregationResult{
			Window:    window,
			Type:      aggType,
			Field:     field,
			Value:     nil,
			Count:     0,
			Timestamp: time.Now(),
		}, nil
	}

	switch aggType {
	case AggregationTypeCount:
		return &AggregationResult{
			Window:    window,
			Type:      aggType,
			Field:     field,
			Value:     int64(len(window.Messages)),
			Count:     int64(len(window.Messages)),
			Timestamp: time.Now(),
		}, nil

	case AggregationTypeSum:
		return w.aggregateSum(window, field)

	case AggregationTypeAvg:
		return w.aggregateAvg(window, field)

	case AggregationTypeMin:
		return w.aggregateMin(window, field)

	case AggregationTypeMax:
		return w.aggregateMax(window, field)

	default:
		return nil, fmt.Errorf("unsupported aggregation type: %s", aggType)
	}
}

// aggregateSum 求和聚合
func (w *StandardWindowProcessor) aggregateSum(window *Window, field string) (*AggregationResult, error) {
	var sum float64
	count := int64(0)

	for _, message := range window.Messages {
		value, err := w.extractNumericValue(message.Data, field)
		if err != nil {
			continue // 跳过无法提取数值的消息
		}
		sum += value
		count++
	}

	return &AggregationResult{
		Window:    window,
		Type:      AggregationTypeSum,
		Field:     field,
		Value:     sum,
		Count:     count,
		Timestamp: time.Now(),
	}, nil
}

// aggregateAvg 平均值聚合
func (w *StandardWindowProcessor) aggregateAvg(window *Window, field string) (*AggregationResult, error) {
	var sum float64
	count := int64(0)

	for _, message := range window.Messages {
		value, err := w.extractNumericValue(message.Data, field)
		if err != nil {
			continue
		}
		sum += value
		count++
	}

	var avg float64
	if count > 0 {
		avg = sum / float64(count)
	}

	return &AggregationResult{
		Window:    window,
		Type:      AggregationTypeAvg,
		Field:     field,
		Value:     avg,
		Count:     count,
		Timestamp: time.Now(),
	}, nil
}

// aggregateMin 最小值聚合
func (w *StandardWindowProcessor) aggregateMin(window *Window, field string) (*AggregationResult, error) {
	min := math.Inf(1)
	count := int64(0)

	for _, message := range window.Messages {
		value, err := w.extractNumericValue(message.Data, field)
		if err != nil {
			continue
		}
		if value < min {
			min = value
		}
		count++
	}

	var result interface{}
	if count > 0 {
		result = min
	}

	return &AggregationResult{
		Window:    window,
		Type:      AggregationTypeMin,
		Field:     field,
		Value:     result,
		Count:     count,
		Timestamp: time.Now(),
	}, nil
}

// aggregateMax 最大值聚合
func (w *StandardWindowProcessor) aggregateMax(window *Window, field string) (*AggregationResult, error) {
	max := math.Inf(-1)
	count := int64(0)

	for _, message := range window.Messages {
		value, err := w.extractNumericValue(message.Data, field)
		if err != nil {
			continue
		}
		if value > max {
			max = value
		}
		count++
	}

	var result interface{}
	if count > 0 {
		result = max
	}

	return &AggregationResult{
		Window:    window,
		Type:      AggregationTypeMax,
		Field:     field,
		Value:     result,
		Count:     count,
		Timestamp: time.Now(),
	}, nil
}

// extractNumericValue 从数据中提取数值
func (w *StandardWindowProcessor) extractNumericValue(data interface{}, field string) (float64, error) {
	switch d := data.(type) {
	case map[string]interface{}:
		if value, exists := d[field]; exists {
			return w.convertToFloat64(value)
		}
		return 0, fmt.Errorf("field '%s' not found", field)

	case string:
		// 尝试解析JSON字符串
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(d), &jsonData); err == nil {
			if value, exists := jsonData[field]; exists {
				return w.convertToFloat64(value)
			}
		}
		return 0, fmt.Errorf("field '%s' not found in JSON string", field)

	default:
		// 如果field为空，尝试直接转换数据
		if field == "" {
			return w.convertToFloat64(data)
		}
		return 0, fmt.Errorf("unsupported data type for field extraction")
	}
}

// convertToFloat64 转换值为float64
func (w *StandardWindowProcessor) convertToFloat64(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", value)
	}
}

// GetWindowCount 获取窗口数量
func (w *StandardWindowProcessor) GetWindowCount() int {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	return len(w.windows)
}

// GetActiveWindowCount 获取活跃窗口数量
func (w *StandardWindowProcessor) GetActiveWindowCount() int {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	return len(w.activeWindows)
}

// GetWindowIDs 获取所有窗口ID
func (w *StandardWindowProcessor) GetWindowIDs() []string {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	ids := make([]string, 0, len(w.windows))
	for id := range w.windows {
		ids = append(ids, id)
	}

	sort.Strings(ids)
	return ids
}

// Clear 清空所有窗口
func (w *StandardWindowProcessor) Clear() {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	w.windows = make(map[string]*Window)
	w.activeWindows = make([]string, 0)
	w.lastCleanup = time.Now()
}
