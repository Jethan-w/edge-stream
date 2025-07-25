package metrics

import (
	"sync"
	"time"
)

// MetricType 指标类型
type MetricType string

const (
	// Counter 计数器类型
	Counter MetricType = "counter"
	// Gauge 仪表盘类型
	Gauge MetricType = "gauge"
	// Histogram 直方图类型
	Histogram MetricType = "histogram"
	// Summary 摘要类型
	Summary MetricType = "summary"
)

// Metric 指标接口
type Metric interface {
	// GetName 获取指标名称
	GetName() string

	// GetType 获取指标类型
	GetType() MetricType

	// GetValue 获取指标值
	GetValue() interface{}

	// GetLabels 获取标签
	GetLabels() map[string]string

	// GetTimestamp 获取时间戳
	GetTimestamp() time.Time
}

// MetricCollector 指标收集器接口
type MetricCollector interface {
	// RecordCounter 记录计数器指标
	RecordCounter(name string, value float64, labels map[string]string)

	// RecordGauge 记录仪表盘指标
	RecordGauge(name string, value float64, labels map[string]string)

	// RecordHistogram 记录直方图指标
	RecordHistogram(name string, value float64, labels map[string]string)

	// RecordLatency 记录延迟指标
	RecordLatency(operation string, duration time.Duration, labels map[string]string)

	// RecordThroughput 记录吞吐量指标
	RecordThroughput(operation string, count int64, labels map[string]string)

	// RecordError 记录错误指标
	RecordError(operation string, errorType string, labels map[string]string)

	// RecordMemoryUsage 记录内存使用情况
	RecordMemoryUsage(component string, bytes int64)

	// RecordQueueDepth 记录队列深度
	RecordQueueDepth(queueName string, depth int64)

	// RecordConnectionCount 记录连接数
	RecordConnectionCount(service string, count int64)

	// GetMetrics 获取所有指标
	GetMetrics() []Metric

	// GetMetric 获取指定指标
	GetMetric(name string) Metric

	// Reset 重置指标
	Reset()

	// Export 导出指标
	Export(format string) ([]byte, error)
}

// MetricSnapshot 指标快照
type MetricSnapshot struct {
	Timestamp time.Time              `json:"timestamp"`
	Metrics   map[string]interface{} `json:"metrics"`
}

// StandardMetric 标准指标实现
type StandardMetric struct {
	name       string
	metricType MetricType
	value      interface{}
	labels     map[string]string
	timestamp  time.Time
	mu         sync.RWMutex
}

// NewStandardMetric 创建标准指标
func NewStandardMetric(name string, metricType MetricType, value interface{}, labels map[string]string) *StandardMetric {
	if labels == nil {
		labels = make(map[string]string)
	}

	return &StandardMetric{
		name:       name,
		metricType: metricType,
		value:      value,
		labels:     labels,
		timestamp:  time.Now(),
	}
}

// GetName 获取指标名称
func (m *StandardMetric) GetName() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.name
}

// GetType 获取指标类型
func (m *StandardMetric) GetType() MetricType {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.metricType
}

// GetValue 获取指标值
func (m *StandardMetric) GetValue() interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.value
}

// GetLabels 获取标签
func (m *StandardMetric) GetLabels() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]string)
	for k, v := range m.labels {
		result[k] = v
	}
	return result
}

// GetTimestamp 获取时间戳
func (m *StandardMetric) GetTimestamp() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.timestamp
}

// SetValue 设置指标值
func (m *StandardMetric) SetValue(value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.value = value
	m.timestamp = time.Now()
}

// AddValue 增加指标值（仅适用于数值类型）
func (m *StandardMetric) AddValue(delta float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch v := m.value.(type) {
	case float64:
		m.value = v + delta
	case int64:
		m.value = v + int64(delta)
	case int:
		m.value = v + int(delta)
	default:
		m.value = delta
	}

	m.timestamp = time.Now()
}

// HistogramData 直方图数据
type HistogramData struct {
	Buckets []HistogramBucket `json:"buckets"`
	Count   int64             `json:"count"`
	Sum     float64           `json:"sum"`
}

// HistogramBucket 直方图桶
type HistogramBucket struct {
	UpperBound float64 `json:"upper_bound"`
	Count      int64   `json:"count"`
}

// SummaryData 摘要数据
type SummaryData struct {
	Quantiles []SummaryQuantile `json:"quantiles"`
	Count     int64             `json:"count"`
	Sum       float64           `json:"sum"`
}

// SummaryQuantile 摘要分位数
type SummaryQuantile struct {
	Quantile float64 `json:"quantile"`
	Value    float64 `json:"value"`
}

// MetricRegistry 指标注册表
type MetricRegistry struct {
	metrics map[string]Metric
	mu      sync.RWMutex
}

// NewMetricRegistry 创建指标注册表
func NewMetricRegistry() *MetricRegistry {
	return &MetricRegistry{
		metrics: make(map[string]Metric),
	}
}

// Register 注册指标
func (r *MetricRegistry) Register(metric Metric) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.metrics[metric.GetName()] = metric
}

// Unregister 注销指标
func (r *MetricRegistry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.metrics, name)
}

// Get 获取指标
func (r *MetricRegistry) Get(name string) Metric {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.metrics[name]
}

// GetAll 获取所有指标
func (r *MetricRegistry) GetAll() []Metric {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Metric, 0, len(r.metrics))
	for _, metric := range r.metrics {
		result = append(result, metric)
	}
	return result
}

// Clear 清空所有指标
func (r *MetricRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.metrics = make(map[string]Metric)
}
