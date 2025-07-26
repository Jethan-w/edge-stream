// Copyright 2025 EdgeStream Team
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// MetricType 指标类型
type MetricType string

const (
	Counter   MetricType = "counter"
	Gauge     MetricType = "gauge"
	Histogram MetricType = "histogram"
	Summary   MetricType = "summary"
)

// Metric 指标接口
type Metric interface {
	// GetName 获取指标名称
	GetName() string

	// GetType 获取指标类型
	GetType() MetricType

	// GetValue 获取指标值
	GetValue() float64

	// GetLabels 获取标签
	GetLabels() map[string]string

	// GetTimestamp 获取时间戳
	GetTimestamp() time.Time

	// GetPrometheusMetric 获取Prometheus指标
	GetPrometheusMetric() prometheus.Collector
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

	// GetRegistry 获取Prometheus注册表
	GetRegistry() *prometheus.Registry
}

// MetricSnapshot 指标快照
type MetricSnapshot struct {
	Timestamp time.Time              `json:"timestamp"`
	Metrics   map[string]interface{} `json:"metrics"`
}

// PrometheusMetric Prometheus指标包装器
type PrometheusMetric struct {
	name       string
	metricType MetricType
	labels     map[string]string
	timestamp  time.Time
	mu         sync.RWMutex

	// Prometheus指标
	counter      prometheus.Counter
	counterVec   prometheus.CounterVec
	gauge        prometheus.Gauge
	gaugeVec     prometheus.GaugeVec
	histogram    prometheus.Histogram
	histogramVec prometheus.HistogramVec
	summary      prometheus.Summary
	summaryVec   prometheus.SummaryVec
}

// registerMetricWithRegistry 通用的指标注册函数
func registerMetricWithRegistry(registry *prometheus.Registry, collector prometheus.Collector) error {
	if err := registry.Register(collector); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); ok {
			return nil // 已注册的指标不是错误
		}
		return err
	}
	return nil
}

// getLabelNames 从标签映射中提取标签名称
func (m *PrometheusMetric) getLabelNames(labels map[string]string) []string {
	labelNames := make([]string, 0, len(labels))
	for k := range labels {
		labelNames = append(labelNames, k)
	}
	return labelNames
}

// registerMetricVec 注册向量指标的通用方法
func (m *PrometheusMetric) registerMetricVec(registry *prometheus.Registry, collector prometheus.Collector, existingType string) {
	if err := registerMetricWithRegistry(registry, collector); err != nil {
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			switch existingType {
			case "counter":
				if existingVec, ok := are.ExistingCollector.(*prometheus.CounterVec); ok {
					m.counterVec = *existingVec
				}
			case "gauge":
				if existingVec, ok := are.ExistingCollector.(*prometheus.GaugeVec); ok {
					m.gaugeVec = *existingVec
				}
			case "histogram":
				if existingVec, ok := are.ExistingCollector.(*prometheus.HistogramVec); ok {
					m.histogramVec = *existingVec
				}
			case "summary":
				if existingVec, ok := are.ExistingCollector.(*prometheus.SummaryVec); ok {
					m.summaryVec = *existingVec
				}
			}
		}
	}
}

// registerMetricSingle 注册单一指标的通用方法
func (m *PrometheusMetric) registerMetricSingle(registry *prometheus.Registry, collector prometheus.Collector, existingType string) {
	if err := registerMetricWithRegistry(registry, collector); err != nil {
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			switch existingType {
			case "counter":
				if existing, ok := are.ExistingCollector.(prometheus.Counter); ok {
					m.counter = existing
				}
			case "gauge":
				if existing, ok := are.ExistingCollector.(prometheus.Gauge); ok {
					m.gauge = existing
				}
			case "histogram":
				if existing, ok := are.ExistingCollector.(prometheus.Histogram); ok {
					m.histogram = existing
				}
			case "summary":
				if existing, ok := are.ExistingCollector.(prometheus.Summary); ok {
					m.summary = existing
				}
			}
		}
	}
}

// createMetricWithLabels 通用的指标创建方法
func (m *PrometheusMetric) createMetricWithLabels(
	name string,
	labels map[string]string,
	registry *prometheus.Registry,
	metricType string,
	withLabels func([]string),
	withoutLabels func(),
) {
	if len(labels) > 0 {
		labelNames := m.getLabelNames(labels)
		withLabels(labelNames)
	} else {
		withoutLabels()
	}
}

// createCounterMetric 创建Counter指标
func (m *PrometheusMetric) createCounterMetric(name string, labels map[string]string, registry *prometheus.Registry) {
	m.createMetricWithLabels(name, labels, registry, "counter", func(labelNames []string) {
		m.counterVec = *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: name,
				Help: "Counter metric for " + name,
			},
			labelNames,
		)
		m.registerMetricVec(registry, &m.counterVec, "counter")
	}, func() {
		m.counter = prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: name,
				Help: "Counter metric for " + name,
			},
		)
		m.registerMetricSingle(registry, m.counter, "counter")
	})
}

// createGaugeMetric 创建Gauge指标
func (m *PrometheusMetric) createGaugeMetric(name string, labels map[string]string, registry *prometheus.Registry) {
	m.createMetricWithLabels(name, labels, registry, "gauge", func(labelNames []string) {
		m.gaugeVec = *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: name,
				Help: "Gauge metric for " + name,
			},
			labelNames,
		)
		m.registerMetricVec(registry, &m.gaugeVec, "gauge")
	}, func() {
		m.gauge = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: name,
				Help: "Gauge metric for " + name,
			},
		)
		m.registerMetricSingle(registry, m.gauge, "gauge")
	})
}

// NewPrometheusMetric 创建Prometheus指标
func NewPrometheusMetric(name string, metricType MetricType, labels map[string]string, registry *prometheus.Registry) *PrometheusMetric {
	if labels == nil {
		labels = make(map[string]string)
	}

	metric := &PrometheusMetric{
		name:       name,
		metricType: metricType,
		labels:     labels,
		timestamp:  time.Now(),
	}

	// 根据类型创建相应的Prometheus指标
	switch metricType {
	case Counter:
		metric.createCounterMetric(name, labels, registry)
	case Gauge:
		metric.createGaugeMetric(name, labels, registry)
	case Histogram:
		metric.createHistogramMetric(name, labels, registry)
	case Summary:
		metric.createSummaryMetric(name, labels, registry)
	}

	return metric
}

// createHistogramMetric 创建Histogram指标
func (m *PrometheusMetric) createHistogramMetric(name string, labels map[string]string, registry *prometheus.Registry) {
	m.createMetricWithLabels(name, labels, registry, "histogram", func(labelNames []string) {
		m.histogramVec = *prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    name,
				Help:    "Histogram metric for " + name,
				Buckets: prometheus.DefBuckets,
			},
			labelNames,
		)
		m.registerMetricVec(registry, &m.histogramVec, "histogram")
	}, func() {
		m.histogram = prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    name,
				Help:    "Histogram metric for " + name,
				Buckets: prometheus.DefBuckets,
			},
		)
		m.registerMetricSingle(registry, m.histogram, "histogram")
	})
}

// createSummaryMetric 创建Summary指标
func (m *PrometheusMetric) createSummaryMetric(name string, labels map[string]string, registry *prometheus.Registry) {
	m.createMetricWithLabels(name, labels, registry, "summary", func(labelNames []string) {
		m.summaryVec = *prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Name: name,
				Help: "Summary metric for " + name,
			},
			labelNames,
		)
		m.registerMetricVec(registry, &m.summaryVec, "summary")
	}, func() {
		m.summary = prometheus.NewSummary(
			prometheus.SummaryOpts{
				Name: name,
				Help: "Summary metric for " + name,
			},
		)
		m.registerMetricSingle(registry, m.summary, "summary")
	})
}

// GetName 获取指标名称
func (m *PrometheusMetric) GetName() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.name
}

// GetType 获取指标类型
func (m *PrometheusMetric) GetType() MetricType {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.metricType
}

// GetValue 获取指标值
func (m *PrometheusMetric) GetValue() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 对于Prometheus指标，我们返回0作为占位符
	// 实际值需要通过Prometheus的Gather方法获取
	return 0.0
}

// GetLabels 获取标签
func (m *PrometheusMetric) GetLabels() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]string)
	for k, v := range m.labels {
		result[k] = v
	}
	return result
}

// GetTimestamp 获取时间戳
func (m *PrometheusMetric) GetTimestamp() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.timestamp
}

// GetPrometheusMetric 获取Prometheus指标
func (m *PrometheusMetric) GetPrometheusMetric() prometheus.Collector {
	m.mu.RLock()

	switch m.metricType {
	case Counter:
		if m.counter != nil {
			m.mu.RUnlock()
			return m.counter
		}
		m.mu.RUnlock()
		return m.counterVec
	case Gauge:
		if m.gauge != nil {
			m.mu.RUnlock()
			return m.gauge
		}
		m.mu.RUnlock()
		return m.gaugeVec
	case Histogram:
		if m.histogram != nil {
			m.mu.RUnlock()
			return m.histogram
		}
		m.mu.RUnlock()
		return m.histogramVec
	case Summary:
		if m.summary != nil {
			m.mu.RUnlock()
			return m.summary
		}
		m.mu.RUnlock()
		return m.summaryVec
	default:
		m.mu.RUnlock()
		return nil
	}
}

// getLabelValues 获取标签值切片
func (m *PrometheusMetric) getLabelValues() []string {
	labelValues := make([]string, 0, len(m.labels))
	for _, v := range m.labels {
		labelValues = append(labelValues, v)
	}
	return labelValues
}

// SetValue 设置指标值
func (m *PrometheusMetric) SetValue(value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateTimestamp()

	switch m.metricType {
	case Gauge:
		if m.gauge != nil {
			m.gauge.Set(value)
		} else {
			m.gaugeVec.WithLabelValues(m.getLabelValues()...).Set(value)
		}
	case Counter, Histogram, Summary:
		// 这些类型不支持SetValue操作
	}
}

// updateTimestamp 更新时间戳的通用方法
func (m *PrometheusMetric) updateTimestamp() {
	m.timestamp = time.Now()
}

// performMetricOperation 执行指标操作的通用方法
func (m *PrometheusMetric) performMetricOperation(operationType string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateTimestamp()

	labelValues := m.getLabelValues()

	switch operationType {
	case "add":
		switch m.metricType {
		case Counter:
			if m.counter != nil {
				m.counter.Add(value)
			} else {
				m.counterVec.WithLabelValues(labelValues...).Add(value)
			}
		case Gauge:
			if m.gauge != nil {
				m.gauge.Add(value)
			} else {
				m.gaugeVec.WithLabelValues(labelValues...).Add(value)
			}
		}
	case "observe":
		switch m.metricType {
		case Histogram:
			if m.histogram != nil {
				m.histogram.Observe(value)
			} else {
				m.histogramVec.WithLabelValues(labelValues...).Observe(value)
			}
		case Summary:
			if m.summary != nil {
				m.summary.Observe(value)
			} else {
				m.summaryVec.WithLabelValues(labelValues...).Observe(value)
			}
		}
	}
}

// AddValue 增加指标值
func (m *PrometheusMetric) AddValue(delta float64) {
	m.performMetricOperation("add", delta)
}

// ObserveValue 观察值（用于直方图和摘要）
func (m *PrometheusMetric) ObserveValue(value float64) {
	m.performMetricOperation("observe", value)
}
