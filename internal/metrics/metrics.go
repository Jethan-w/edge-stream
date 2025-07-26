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
		if len(labels) > 0 {
			labelNames := make([]string, 0, len(labels))
			for k := range labels {
				labelNames = append(labelNames, k)
			}
			metric.counterVec = *prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: name,
					Help: "Counter metric for " + name,
				},
				labelNames,
			)
			if err := registerMetricWithRegistry(registry, &metric.counterVec); err != nil {
				if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
					if existingVec, ok := are.ExistingCollector.(*prometheus.CounterVec); ok {
						metric.counterVec = *existingVec
					}
				}
			}
		} else {
			metric.counter = prometheus.NewCounter(
				prometheus.CounterOpts{
					Name: name,
					Help: "Counter metric for " + name,
				},
			)
			if err := registerMetricWithRegistry(registry, metric.counter); err != nil {
				if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
					if existingCounter, ok := are.ExistingCollector.(prometheus.Counter); ok {
						metric.counter = existingCounter
					}
				}
			}
		}
	case Gauge:
		if len(labels) > 0 {
			labelNames := make([]string, 0, len(labels))
			for k := range labels {
				labelNames = append(labelNames, k)
			}
			metric.gaugeVec = *prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: name,
					Help: "Gauge metric for " + name,
				},
				labelNames,
			)
			if err := registerMetricWithRegistry(registry, &metric.gaugeVec); err != nil {
				if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
					if existingVec, ok := are.ExistingCollector.(*prometheus.GaugeVec); ok {
						metric.gaugeVec = *existingVec
					}
				}
			}
		} else {
			metric.gauge = prometheus.NewGauge(
				prometheus.GaugeOpts{
					Name: name,
					Help: "Gauge metric for " + name,
				},
			)
			if err := registerMetricWithRegistry(registry, metric.gauge); err != nil {
				if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
					if existingGauge, ok := are.ExistingCollector.(prometheus.Gauge); ok {
						metric.gauge = existingGauge
					}
				}
			}
		}
	case Histogram:
		metric.createHistogramMetric(name, labels, registry)
	case Summary:
		metric.createSummaryMetric(name, labels, registry)
	}

	return metric
}

// createHistogramMetric 创建Histogram指标
func (m *PrometheusMetric) createHistogramMetric(name string, labels map[string]string, registry *prometheus.Registry) {
	if len(labels) > 0 {
		labelNames := make([]string, 0, len(labels))
		for k := range labels {
			labelNames = append(labelNames, k)
		}
		m.histogramVec = *prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    name,
				Help:    "Histogram metric for " + name,
				Buckets: prometheus.DefBuckets,
			},
			labelNames,
		)
		if err := registerMetricWithRegistry(registry, &m.histogramVec); err != nil {
			if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
				if existingVec, ok := are.ExistingCollector.(*prometheus.HistogramVec); ok {
					m.histogramVec = *existingVec
				}
			}
		}
	} else {
		m.histogram = prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    name,
				Help:    "Histogram metric for " + name,
				Buckets: prometheus.DefBuckets,
			},
		)
		if err := registerMetricWithRegistry(registry, m.histogram); err != nil {
			if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
				if existingHistogram, ok := are.ExistingCollector.(prometheus.Histogram); ok {
					m.histogram = existingHistogram
				}
			}
		}
	}
}

// createSummaryMetric 创建Summary指标
func (m *PrometheusMetric) createSummaryMetric(name string, labels map[string]string, registry *prometheus.Registry) {
	if len(labels) > 0 {
		labelNames := make([]string, 0, len(labels))
		for k := range labels {
			labelNames = append(labelNames, k)
		}
		m.summaryVec = *prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Name: name,
				Help: "Summary metric for " + name,
			},
			labelNames,
		)
		if err := registerMetricWithRegistry(registry, &m.summaryVec); err != nil {
			if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
				if existingVec, ok := are.ExistingCollector.(*prometheus.SummaryVec); ok {
					m.summaryVec = *existingVec
				}
			}
		}
	} else {
		m.summary = prometheus.NewSummary(
			prometheus.SummaryOpts{
				Name: name,
				Help: "Summary metric for " + name,
			},
		)
		if err := registerMetricWithRegistry(registry, m.summary); err != nil {
			if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
				if existingSummary, ok := are.ExistingCollector.(prometheus.Summary); ok {
					m.summary = existingSummary
				}
			}
		}
	}
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
	defer m.mu.RUnlock()

	switch m.metricType {
	case Counter:
		if m.counter != nil {
			return m.counter
		}
		return m.counterVec
	case Gauge:
		if m.gauge != nil {
			return m.gauge
		}
		return m.gaugeVec
	case Histogram:
		if m.histogram != nil {
			return m.histogram
		}
		return m.histogramVec
	case Summary:
		if m.summary != nil {
			return m.summary
		}
		return m.summaryVec
	default:
		return nil
	}
}

// SetValue 设置指标值
func (m *PrometheusMetric) SetValue(value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.timestamp = time.Now()

	switch m.metricType {
	case Gauge:
		if m.gauge != nil {
			m.gauge.Set(value)
		} else {
			labelValues := make([]string, 0, len(m.labels))
			for _, v := range m.labels {
				labelValues = append(labelValues, v)
			}
			m.gaugeVec.WithLabelValues(labelValues...).Set(value)
		}
	}
}

// AddValue 增加指标值
func (m *PrometheusMetric) AddValue(delta float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.timestamp = time.Now()

	switch m.metricType {
	case Counter:
		if m.counter != nil {
			m.counter.Add(delta)
		} else {
			labelValues := make([]string, 0, len(m.labels))
			for _, v := range m.labels {
				labelValues = append(labelValues, v)
			}
			m.counterVec.WithLabelValues(labelValues...).Add(delta)
		}
	case Gauge:
		if m.gauge != nil {
			m.gauge.Add(delta)
		} else {
			labelValues := make([]string, 0, len(m.labels))
			for _, v := range m.labels {
				labelValues = append(labelValues, v)
			}
			m.gaugeVec.WithLabelValues(labelValues...).Add(delta)
		}
	case Histogram, Summary:
		// Histogram和Summary不支持AddValue操作
		// 这些类型应该使用ObserveValue方法
	}
}

// ObserveValue 观察值（用于直方图和摘要）
func (m *PrometheusMetric) ObserveValue(value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.timestamp = time.Now()

	switch m.metricType {
	case Histogram:
		if m.histogram != nil {
			m.histogram.Observe(value)
		} else {
			labelValues := make([]string, 0, len(m.labels))
			for _, v := range m.labels {
				labelValues = append(labelValues, v)
			}
			m.histogramVec.WithLabelValues(labelValues...).Observe(value)
		}
	case Summary:
		if m.summary != nil {
			m.summary.Observe(value)
		} else {
			labelValues := make([]string, 0, len(m.labels))
			for _, v := range m.labels {
				labelValues = append(labelValues, v)
			}
			m.summaryVec.WithLabelValues(labelValues...).Observe(value)
		}
	case Counter, Gauge:
		// Counter和Gauge不支持ObserveValue操作
		// 这些类型应该使用SetValue或AddValue方法
	}
}
