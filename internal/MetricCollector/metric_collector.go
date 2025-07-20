package MetricCollector

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// MetricCollector 指标收集器接口
type MetricCollector interface {
	// RegisterGauge 注册 Gauge 指标（瞬时值）
	RegisterGauge(name string, supplier func() float64) (Gauge, error)

	// RegisterCounter 注册 Counter 指标（累计值）
	RegisterCounter(name string) (Counter, error)

	// RegisterHistogram 注册 Histogram 指标（数值分布）
	RegisterHistogram(name string) (Histogram, error)

	// RegisterTimer 注册 Timer 指标（耗时统计）
	RegisterTimer(name string) (Timer, error)

	// RegisterMeter 注册 Meter 指标（速率统计）
	RegisterMeter(name string) (Meter, error)

	// GetMetricsSnapshot 获取指标快照
	GetMetricsSnapshot() *MetricsSnapshot

	// GetMetric 获取指定指标
	GetMetric(name string) (Metric, error)

	// RemoveMetric 移除指标
	RemoveMetric(name string) error

	// ListMetrics 列出所有指标
	ListMetrics() []string
	Start()
	Stop()
	RecordMetric(s string, milliseconds int64)
	RecordEngineStop()
	RecordProcessorScheduled(id string)
	RecordEngineStart()
}

// MetricType 指标类型枚举
type MetricType string

const (
	MetricTypeGauge     MetricType = "GAUGE"
	MetricTypeCounter   MetricType = "COUNTER"
	MetricTypeHistogram MetricType = "HISTOGRAM"
	MetricTypeTimer     MetricType = "TIMER"
	MetricTypeMeter     MetricType = "METER"
)

// MetricScope 指标作用域枚举
type MetricScope string

const (
	MetricScopeSystem     MetricScope = "SYSTEM"
	MetricScopeCluster    MetricScope = "CLUSTER"
	MetricScopeFlow       MetricScope = "FLOW"
	MetricScopeProcessor  MetricScope = "PROCESSOR"
	MetricScopeConnection MetricScope = "CONNECTION"
)

// Metric 指标接口
type Metric interface {
	// GetName 获取指标名称
	GetName() string

	// GetType 获取指标类型
	GetType() MetricType

	// GetScope 获取指标作用域
	GetScope() MetricScope

	// GetValue 获取指标值
	GetValue() interface{}

	// GetTimestamp 获取指标时间戳
	GetTimestamp() time.Time
}

// Gauge 瞬时值指标接口
type Gauge interface {
	Metric

	// Update 更新值
	Update(value float64)
}

// Counter 累计值指标接口
type Counter interface {
	Metric

	// Increment 增加计数
	Increment(delta int64)

	// GetCount 获取当前计数
	GetCount() int64

	// Reset 重置计数
	Reset()
}

// Histogram 数值分布指标接口
type Histogram interface {
	Metric

	// Update 更新值
	Update(value float64)

	// GetCount 获取样本数量
	GetCount() int64

	// GetMin 获取最小值
	GetMin() float64

	// GetMax 获取最大值
	GetMax() float64

	// GetMean 获取平均值
	GetMean() float64

	// GetPercentile 获取百分位数
	GetPercentile(percentile float64) float64
}

// Timer 耗时统计指标接口
type Timer interface {
	Metric

	// Start 开始计时
	Start() *TimerContext

	// Record 记录耗时
	Record(duration time.Duration)

	// GetCount 获取样本数量
	GetCount() int64

	// GetMean 获取平均耗时
	GetMean() time.Duration

	// GetMin 获取最小耗时
	GetMin() time.Duration

	// GetMax 获取最大耗时
	GetMax() time.Duration
}

// TimerContext 计时器上下文
type TimerContext struct {
	timer   *StandardTimer
	start   time.Time
	stopped bool
}

// Stop 停止计时
func (tc *TimerContext) Stop() {
	if !tc.stopped {
		tc.timer.Record(time.Since(tc.start))
		tc.stopped = true
	}
}

// Meter 速率统计指标接口
type Meter interface {
	Metric

	// Mark 标记事件
	Mark(count int64)

	// GetCount 获取总事件数
	GetCount() int64

	// GetMeanRate 获取平均速率
	GetMeanRate() float64

	// GetOneMinuteRate 获取1分钟速率
	GetOneMinuteRate() float64

	// GetFiveMinuteRate 获取5分钟速率
	GetFiveMinuteRate() float64

	// GetFifteenMinuteRate 获取15分钟速率
	GetFifteenMinuteRate() float64
}

// StandardMetricCollector 标准指标收集器实现
type StandardMetricCollector struct {
	metrics map[string]Metric
	mutex   sync.RWMutex
}

// NewStandardMetricCollector 创建新的标准指标收集器
func NewStandardMetricCollector() *StandardMetricCollector {
	return &StandardMetricCollector{
		metrics: make(map[string]Metric),
	}
}

// RegisterGauge 注册 Gauge 指标
func (smc *StandardMetricCollector) RegisterGauge(name string, supplier func() float64) (Gauge, error) {
	smc.mutex.Lock()
	defer smc.mutex.Unlock()

	if _, exists := smc.metrics[name]; exists {
		return nil, fmt.Errorf("指标已存在: %s", name)
	}

	gauge := NewStandardGauge(name, supplier)
	smc.metrics[name] = gauge

	return gauge, nil
}

// RegisterCounter 注册 Counter 指标
func (smc *StandardMetricCollector) RegisterCounter(name string) (Counter, error) {
	smc.mutex.Lock()
	defer smc.mutex.Unlock()

	if _, exists := smc.metrics[name]; exists {
		return nil, fmt.Errorf("指标已存在: %s", name)
	}

	counter := NewStandardCounter(name)
	smc.metrics[name] = counter

	return counter, nil
}

// RegisterHistogram 注册 Histogram 指标
func (smc *StandardMetricCollector) RegisterHistogram(name string) (Histogram, error) {
	smc.mutex.Lock()
	defer smc.mutex.Unlock()

	if _, exists := smc.metrics[name]; exists {
		return nil, fmt.Errorf("指标已存在: %s", name)
	}

	histogram := NewStandardHistogram(name)
	smc.metrics[name] = histogram

	return histogram, nil
}

// RegisterTimer 注册 Timer 指标
func (smc *StandardMetricCollector) RegisterTimer(name string) (Timer, error) {
	smc.mutex.Lock()
	defer smc.mutex.Unlock()

	if _, exists := smc.metrics[name]; exists {
		return nil, fmt.Errorf("指标已存在: %s", name)
	}

	timer := NewStandardTimer(name)
	smc.metrics[name] = timer

	return timer, nil
}

// RegisterMeter 注册 Meter 指标
func (smc *StandardMetricCollector) RegisterMeter(name string) (Meter, error) {
	smc.mutex.Lock()
	defer smc.mutex.Unlock()

	if _, exists := smc.metrics[name]; exists {
		return nil, fmt.Errorf("指标已存在: %s", name)
	}

	meter := NewStandardMeter(name)
	smc.metrics[name] = meter

	return meter, nil
}

// GetMetricsSnapshot 获取指标快照
func (smc *StandardMetricCollector) GetMetricsSnapshot() *MetricsSnapshot {
	smc.mutex.RLock()
	defer smc.mutex.RUnlock()

	snapshot := NewMetricsSnapshot()

	for name, metric := range smc.metrics {
		switch m := metric.(type) {
		case Gauge:
			snapshot.Gauges[name] = m.GetValue().(float64)
		case Counter:
			snapshot.Counters[name] = m.GetValue().(int64)
		case Histogram:
			var t HistogramData
			b, _ := json.Marshal(m)
			json.Unmarshal(b, &t)
			snapshot.Histograms[name] = t
		case Timer:
			var t TimerData
			b, _ := json.Marshal(m)
			json.Unmarshal(b, &t)
			snapshot.Timers[name] = t
		case Meter:
			var t MeterData
			b, _ := json.Marshal(m)
			json.Unmarshal(b, &t)
			snapshot.Meters[name] = t
		}
	}

	return snapshot
}

// GetMetric 获取指定指标
func (smc *StandardMetricCollector) GetMetric(name string) (Metric, error) {
	smc.mutex.RLock()
	defer smc.mutex.RUnlock()

	metric, exists := smc.metrics[name]
	if !exists {
		return nil, fmt.Errorf("指标不存在: %s", name)
	}

	return metric, nil
}

// RemoveMetric 移除指标
func (smc *StandardMetricCollector) RemoveMetric(name string) error {
	smc.mutex.Lock()
	defer smc.mutex.Unlock()

	if _, exists := smc.metrics[name]; !exists {
		return fmt.Errorf("指标不存在: %s", name)
	}

	delete(smc.metrics, name)
	return nil
}
func (smc *StandardMetricCollector) Start() {
	panic("1")
}
func (smc *StandardMetricCollector) Stop() {
	panic("1")

}
func (smc *StandardMetricCollector) RecordMetric(s string, milliseconds int64) {
	panic("1")

}
func (smc *StandardMetricCollector) RecordEngineStop() {
	panic("1")

}
func (smc *StandardMetricCollector) RecordProcessorScheduled(id string) {
	panic("1")

}
func (smc *StandardMetricCollector) RecordEngineStart() {
	panic("1")

}

// ListMetrics 列出所有指标
func (smc *StandardMetricCollector) ListMetrics() []string {
	smc.mutex.RLock()
	defer smc.mutex.RUnlock()

	metrics := make([]string, 0, len(smc.metrics))
	for name := range smc.metrics {
		metrics = append(metrics, name)
	}

	return metrics
}

// StandardGauge 标准Gauge实现
type StandardGauge struct {
	name      string
	supplier  func() float64
	value     float64
	timestamp time.Time
	mutex     sync.RWMutex
}

// NewStandardGauge 创建新的标准Gauge
func NewStandardGauge(name string, supplier func() float64) *StandardGauge {
	return &StandardGauge{
		name:      name,
		supplier:  supplier,
		timestamp: time.Now(),
	}
}

// GetName 获取指标名称
func (sg *StandardGauge) GetName() string {
	return sg.name
}

// GetType 获取指标类型
func (sg *StandardGauge) GetType() MetricType {
	return MetricTypeGauge
}

// GetScope 获取指标作用域
func (sg *StandardGauge) GetScope() MetricScope {
	return MetricScopeSystem
}

// GetValue 获取当前值
func (sg *StandardGauge) GetValue() interface{} {
	sg.mutex.RLock()
	defer sg.mutex.RUnlock()

	if sg.supplier != nil {
		return sg.supplier()
	}
	return sg.value
}

// Update 更新值
func (sg *StandardGauge) Update(value float64) {
	sg.mutex.Lock()
	defer sg.mutex.Unlock()

	sg.value = value
	sg.timestamp = time.Now()
}

// GetTimestamp 获取指标时间戳
func (sg *StandardGauge) GetTimestamp() time.Time {
	sg.mutex.RLock()
	defer sg.mutex.RUnlock()

	return sg.timestamp
}

// StandardCounter 标准Counter实现
type StandardCounter struct {
	name      string
	count     int64
	timestamp time.Time
	mutex     sync.RWMutex
}

// NewStandardCounter 创建新的标准Counter
func NewStandardCounter(name string) *StandardCounter {
	return &StandardCounter{
		name:      name,
		timestamp: time.Now(),
	}
}

// GetName 获取指标名称
func (sc *StandardCounter) GetName() string {
	return sc.name
}

// GetType 获取指标类型
func (sc *StandardCounter) GetType() MetricType {
	return MetricTypeCounter
}

// GetScope 获取指标作用域
func (sc *StandardCounter) GetScope() MetricScope {
	return MetricScopeSystem
}

// GetValue 获取指标值
func (sc *StandardCounter) GetValue() interface{} {
	return sc.GetCount()
}

// Increment 增加计数
func (sc *StandardCounter) Increment(delta int64) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	sc.count += delta
	sc.timestamp = time.Now()
}

// GetCount 获取当前计数
func (sc *StandardCounter) GetCount() int64 {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()

	return sc.count
}

// Reset 重置计数
func (sc *StandardCounter) Reset() {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	sc.count = 0
	sc.timestamp = time.Now()
}

// GetTimestamp 获取指标时间戳
func (sc *StandardCounter) GetTimestamp() time.Time {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()

	return sc.timestamp
}

// StandardHistogram 标准Histogram实现
type StandardHistogram struct {
	name      string
	count     int64
	min       float64
	max       float64
	sum       float64
	samples   []float64
	timestamp time.Time
	mutex     sync.RWMutex
}

// NewStandardHistogram 创建新的标准Histogram
func NewStandardHistogram(name string) *StandardHistogram {
	return &StandardHistogram{
		name:      name,
		samples:   make([]float64, 0),
		timestamp: time.Now(),
	}
}

// GetName 获取指标名称
func (sh *StandardHistogram) GetName() string {
	return sh.name
}

// GetType 获取指标类型
func (sh *StandardHistogram) GetType() MetricType {
	return MetricTypeHistogram
}

// GetScope 获取指标作用域
func (sh *StandardHistogram) GetScope() MetricScope {
	return MetricScopeSystem
}

// GetValue 获取指标值
func (sh *StandardHistogram) GetValue() interface{} {
	return map[string]interface{}{
		"count": sh.GetCount(),
		"min":   sh.GetMin(),
		"max":   sh.GetMax(),
		"mean":  sh.GetMean(),
	}
}

// Update 更新值
func (sh *StandardHistogram) Update(value float64) {
	sh.mutex.Lock()
	defer sh.mutex.Unlock()

	sh.count++
	sh.sum += value
	sh.samples = append(sh.samples, value)

	if sh.count == 1 || value < sh.min {
		sh.min = value
	}
	if sh.count == 1 || value > sh.max {
		sh.max = value
	}

	sh.timestamp = time.Now()
}

// GetCount 获取样本数量
func (sh *StandardHistogram) GetCount() int64 {
	sh.mutex.RLock()
	defer sh.mutex.RUnlock()

	return sh.count
}

// GetMin 获取最小值
func (sh *StandardHistogram) GetMin() float64 {
	sh.mutex.RLock()
	defer sh.mutex.RUnlock()

	return sh.min
}

// GetMax 获取最大值
func (sh *StandardHistogram) GetMax() float64 {
	sh.mutex.RLock()
	defer sh.mutex.RUnlock()

	return sh.max
}

// GetMean 获取平均值
func (sh *StandardHistogram) GetMean() float64 {
	sh.mutex.RLock()
	defer sh.mutex.RUnlock()

	if sh.count == 0 {
		return 0.0
	}
	return sh.sum / float64(sh.count)
}

// GetPercentile 获取百分位数
func (sh *StandardHistogram) GetPercentile(percentile float64) float64 {
	sh.mutex.RLock()
	defer sh.mutex.RUnlock()

	if len(sh.samples) == 0 {
		return 0.0
	}

	// 简化的百分位数计算
	// 实际应该使用更高效的算法
	index := int(float64(len(sh.samples)-1) * percentile / 100.0)
	if index >= len(sh.samples) {
		index = len(sh.samples) - 1
	}

	return sh.samples[index]
}

// GetTimestamp 获取指标时间戳
func (sh *StandardHistogram) GetTimestamp() time.Time {
	sh.mutex.RLock()
	defer sh.mutex.RUnlock()

	return sh.timestamp
}

// StandardTimer 标准Timer实现
type StandardTimer struct {
	name      string
	count     int64
	min       time.Duration
	max       time.Duration
	sum       time.Duration
	samples   []time.Duration
	timestamp time.Time
	mutex     sync.RWMutex
}

// NewStandardTimer 创建新的标准Timer
func NewStandardTimer(name string) *StandardTimer {
	return &StandardTimer{
		name:      name,
		samples:   make([]time.Duration, 0),
		timestamp: time.Now(),
	}
}

// GetName 获取指标名称
func (st *StandardTimer) GetName() string {
	return st.name
}

// GetType 获取指标类型
func (st *StandardTimer) GetType() MetricType {
	return MetricTypeTimer
}

// GetScope 获取指标作用域
func (st *StandardTimer) GetScope() MetricScope {
	return MetricScopeSystem
}

// GetValue 获取指标值
func (st *StandardTimer) GetValue() interface{} {
	return map[string]interface{}{
		"count": st.GetCount(),
		"min":   st.GetMin().String(),
		"max":   st.GetMax().String(),
		"mean":  st.GetMean().String(),
	}
}

// Start 开始计时
func (st *StandardTimer) Start() *TimerContext {
	return &TimerContext{
		timer: st,
		start: time.Now(),
	}
}

// Record 记录耗时
func (st *StandardTimer) Record(duration time.Duration) {
	st.mutex.Lock()
	defer st.mutex.Unlock()

	st.count++
	st.sum += duration
	st.samples = append(st.samples, duration)

	if st.count == 1 || duration < st.min {
		st.min = duration
	}
	if st.count == 1 || duration > st.max {
		st.max = duration
	}

	st.timestamp = time.Now()
}

// GetCount 获取样本数量
func (st *StandardTimer) GetCount() int64 {
	st.mutex.RLock()
	defer st.mutex.RUnlock()

	return st.count
}

// GetMean 获取平均耗时
func (st *StandardTimer) GetMean() time.Duration {
	st.mutex.RLock()
	defer st.mutex.RUnlock()

	if st.count == 0 {
		return 0
	}
	return st.sum / time.Duration(st.count)
}

// GetMin 获取最小耗时
func (st *StandardTimer) GetMin() time.Duration {
	st.mutex.RLock()
	defer st.mutex.RUnlock()

	return st.min
}

// GetMax 获取最大耗时
func (st *StandardTimer) GetMax() time.Duration {
	st.mutex.RLock()
	defer st.mutex.RUnlock()

	return st.max
}

// GetTimestamp 获取指标时间戳
func (st *StandardTimer) GetTimestamp() time.Time {
	st.mutex.RLock()
	defer st.mutex.RUnlock()

	return st.timestamp
}

// StandardMeter 标准Meter实现
type StandardMeter struct {
	name      string
	count     int64
	startTime time.Time
	lastMark  time.Time
	timestamp time.Time
	mutex     sync.RWMutex
}

// NewStandardMeter 创建新的标准Meter
func NewStandardMeter(name string) *StandardMeter {
	return &StandardMeter{
		name:      name,
		startTime: time.Now(),
		lastMark:  time.Now(),
		timestamp: time.Now(),
	}
}

// GetName 获取指标名称
func (sm *StandardMeter) GetName() string {
	return sm.name
}

// GetType 获取指标类型
func (sm *StandardMeter) GetType() MetricType {
	return MetricTypeMeter
}

// GetScope 获取指标作用域
func (sm *StandardMeter) GetScope() MetricScope {
	return MetricScopeSystem
}

// GetValue 获取指标值
func (sm *StandardMeter) GetValue() interface{} {
	return map[string]interface{}{
		"count":             sm.GetCount(),
		"meanRate":          sm.GetMeanRate(),
		"oneMinuteRate":     sm.GetOneMinuteRate(),
		"fiveMinuteRate":    sm.GetFiveMinuteRate(),
		"fifteenMinuteRate": sm.GetFifteenMinuteRate(),
	}
}

// Mark 标记事件
func (sm *StandardMeter) Mark(count int64) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.count += count
	sm.lastMark = time.Now()
	sm.timestamp = time.Now()
}

// GetCount 获取总事件数
func (sm *StandardMeter) GetCount() int64 {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	return sm.count
}

// GetMeanRate 获取平均速率
func (sm *StandardMeter) GetMeanRate() float64 {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	elapsed := time.Since(sm.startTime).Seconds()
	if elapsed == 0 {
		return 0.0
	}
	return float64(sm.count) / elapsed
}

// GetOneMinuteRate 获取1分钟速率
func (sm *StandardMeter) GetOneMinuteRate() float64 {
	// 简化实现，实际应该使用滑动窗口
	return sm.GetMeanRate()
}

// GetFiveMinuteRate 获取5分钟速率
func (sm *StandardMeter) GetFiveMinuteRate() float64 {
	// 简化实现，实际应该使用滑动窗口
	return sm.GetMeanRate()
}

// GetFifteenMinuteRate 获取15分钟速率
func (sm *StandardMeter) GetFifteenMinuteRate() float64 {
	// 简化实现，实际应该使用滑动窗口
	return sm.GetMeanRate()
}

// GetTimestamp 获取指标时间戳
func (sm *StandardMeter) GetTimestamp() time.Time {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	return sm.timestamp
}
