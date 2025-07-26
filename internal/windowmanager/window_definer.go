package windowmanager

import (
	"fmt"
	"sync"
	"time"
)

// TimeWindow 时间窗口
type TimeWindow struct {
	id        string
	duration  time.Duration
	startTime time.Time
	data      []interface{}
	mu        sync.RWMutex
}

// NewTimeWindow 创建时间窗口
func NewTimeWindow(duration time.Duration) *TimeWindow {
	return &TimeWindow{
		id:        fmt.Sprintf("time-window-%d", time.Now().UnixNano()),
		duration:  duration,
		startTime: time.Now(),
		data:      make([]interface{}, 0),
	}
}

func (tw *TimeWindow) GetID() string {
	return tw.id
}

func (tw *TimeWindow) GetWindowType() WindowType {
	return TIME_WINDOW
}

func (tw *TimeWindow) AddData(data interface{}) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.data = append(tw.data, data)
}

func (tw *TimeWindow) GetData() []interface{} {
	tw.mu.RLock()
	defer tw.mu.RUnlock()
	return append([]interface{}{}, tw.data...)
}

func (tw *TimeWindow) IsReady() bool {
	return time.Since(tw.startTime) >= tw.duration
}

func (tw *TimeWindow) Reset() {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.data = make([]interface{}, 0)
	tw.startTime = time.Now()
}

// CountWindow 计数窗口
type CountWindow struct {
	id       string
	maxCount int
	data     []interface{}
	mu       sync.RWMutex
}

// NewCountWindow 创建计数窗口
func NewCountWindow(maxCount int) *CountWindow {
	return &CountWindow{
		id:       fmt.Sprintf("count-window-%d", time.Now().UnixNano()),
		maxCount: maxCount,
		data:     make([]interface{}, 0),
	}
}

func (cw *CountWindow) GetID() string {
	return cw.id
}

func (cw *CountWindow) GetWindowType() WindowType {
	return COUNT_WINDOW
}

func (cw *CountWindow) AddData(data interface{}) {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	cw.data = append(cw.data, data)
	
	// 如果超过最大数量，移除最旧的数据
	if len(cw.data) > cw.maxCount && cw.maxCount > 0 {
		startIndex := len(cw.data) - cw.maxCount
		if startIndex >= 0 {
			cw.data = cw.data[startIndex:]
		}
	}
}

func (cw *CountWindow) GetData() []interface{} {
	cw.mu.RLock()
	defer cw.mu.RUnlock()
	return append([]interface{}{}, cw.data...)
}

func (cw *CountWindow) IsReady() bool {
	cw.mu.RLock()
	defer cw.mu.RUnlock()
	return len(cw.data) >= cw.maxCount
}

func (cw *CountWindow) Reset() {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	cw.data = make([]interface{}, 0)
}
