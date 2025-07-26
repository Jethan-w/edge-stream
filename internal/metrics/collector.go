package metrics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

// StandardMetricCollector 标准指标收集器实现
type StandardMetricCollector struct {
	registry *prometheus.Registry
	metrics  map[string]*PrometheusMetric
	mu       sync.RWMutex
}

// NewStandardMetricCollector 创建标准指标收集器
func NewStandardMetricCollector() *StandardMetricCollector {
	return &StandardMetricCollector{
		registry: prometheus.NewRegistry(),
		metrics:  make(map[string]*PrometheusMetric),
	}
}

// RecordCounter 记录计数器指标
func (c *StandardMetricCollector) RecordCounter(name string, value float64, labels map[string]string) {
	metricKey := c.buildMetricKey(name, labels)
	
	c.mu.Lock()
	metric, exists := c.metrics[metricKey]
	if !exists {
		metric = NewPrometheusMetric(name, Counter, labels, c.registry)
		c.metrics[metricKey] = metric
	}
	c.mu.Unlock()
	
	metric.AddValue(value)
}

// RecordGauge 记录仪表盘指标
func (c *StandardMetricCollector) RecordGauge(name string, value float64, labels map[string]string) {
	metricKey := c.buildMetricKey(name, labels)
	
	c.mu.Lock()
	metric, exists := c.metrics[metricKey]
	if !exists {
		metric = NewPrometheusMetric(name, Gauge, labels, c.registry)
		c.metrics[metricKey] = metric
	}
	c.mu.Unlock()
	
	metric.SetValue(value)
}

// RecordHistogram 记录直方图指标
func (c *StandardMetricCollector) RecordHistogram(name string, value float64, labels map[string]string) {
	metricKey := c.buildMetricKey(name, labels)
	
	c.mu.Lock()
	metric, exists := c.metrics[metricKey]
	if !exists {
		metric = NewPrometheusMetric(name, Histogram, labels, c.registry)
		c.metrics[metricKey] = metric
	}
	c.mu.Unlock()
	
	metric.ObserveValue(value)
}

// RecordLatency 记录延迟指标
func (c *StandardMetricCollector) RecordLatency(operation string, duration time.Duration, labels map[string]string) {
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["operation"] = operation
	
	latencySeconds := duration.Seconds()
	c.RecordHistogram("latency_seconds", latencySeconds, labels)
}

// RecordThroughput 记录吞吐量指标
func (c *StandardMetricCollector) RecordThroughput(operation string, count int64, labels map[string]string) {
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["operation"] = operation
	
	c.RecordCounter("throughput_total", float64(count), labels)
}

// RecordError 记录错误指标
func (c *StandardMetricCollector) RecordError(operation string, errorType string, labels map[string]string) {
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["operation"] = operation
	labels["error_type"] = errorType
	
	c.RecordCounter("errors_total", 1, labels)
}

// RecordMemoryUsage 记录内存使用情况
func (c *StandardMetricCollector) RecordMemoryUsage(component string, bytes int64) {
	labels := map[string]string{
		"component": component,
	}
	
	c.RecordGauge("memory_usage_bytes", float64(bytes), labels)
}

// RecordQueueDepth 记录队列深度
func (c *StandardMetricCollector) RecordQueueDepth(queueName string, depth int64) {
	labels := map[string]string{
		"queue": queueName,
	}
	
	c.RecordGauge("queue_depth", float64(depth), labels)
}

// RecordConnectionCount 记录连接数
func (c *StandardMetricCollector) RecordConnectionCount(service string, count int64) {
	labels := map[string]string{
		"service": service,
	}
	
	c.RecordGauge("connection_count", float64(count), labels)
}

// GetMetrics 获取所有指标
func (c *StandardMetricCollector) GetMetrics() []Metric {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	result := make([]Metric, 0, len(c.metrics))
	for _, metric := range c.metrics {
		result = append(result, metric)
	}
	return result
}

// GetMetric 获取指定指标
func (c *StandardMetricCollector) GetMetric(name string) Metric {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	for _, metric := range c.metrics {
		if metric.GetName() == name {
			return metric
		}
	}
	return nil
}

// Reset 重置指标
func (c *StandardMetricCollector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// 创建新的注册表和指标映射
	c.registry = prometheus.NewRegistry()
	c.metrics = make(map[string]*PrometheusMetric)
}

// Export 导出指标
func (c *StandardMetricCollector) Export(format string) ([]byte, error) {
	switch format {
	case "prometheus":
		return c.exportPrometheus()
	case "json":
		return c.exportJSON()
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// GetRegistry 获取Prometheus注册表
func (c *StandardMetricCollector) GetRegistry() *prometheus.Registry {
	return c.registry
}

// exportPrometheus 导出Prometheus格式
func (c *StandardMetricCollector) exportPrometheus() ([]byte, error) {
	metricFamilies, err := c.registry.Gather()
	if err != nil {
		return nil, fmt.Errorf("failed to gather metrics: %w", err)
	}
	
	var buf bytes.Buffer
	for _, mf := range metricFamilies {
		if _, err := expfmt.MetricFamilyToText(&buf, mf); err != nil {
			return nil, fmt.Errorf("failed to write metric family: %w", err)
		}
	}
	
	return buf.Bytes(), nil
}

// exportJSON 导出JSON格式
func (c *StandardMetricCollector) exportJSON() ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	snapshot := MetricSnapshot{
		Timestamp: time.Now(),
		Metrics:   make(map[string]interface{}),
	}
	
	for key, metric := range c.metrics {
		snapshot.Metrics[key] = map[string]interface{}{
			"name":      metric.GetName(),
			"type":      string(metric.GetType()),
			"labels":    metric.GetLabels(),
			"timestamp": metric.GetTimestamp(),
		}
	}
	
	return json.Marshal(snapshot)
}

// buildMetricKey 构建指标键
func (c *StandardMetricCollector) buildMetricKey(name string, labels map[string]string) string {
	key := name
	if len(labels) > 0 {
		key += "{"
		first := true
		for k, v := range labels {
			if !first {
				key += ","
			}
			key += fmt.Sprintf("%s=%s", k, v)
			first = false
		}
		key += "}"
	}
	return key
}

// GetMetricCount 获取指标数量
func (c *StandardMetricCollector) GetMetricCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.metrics)
}

// GetMetricNames 获取所有指标名称
func (c *StandardMetricCollector) GetMetricNames() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	names := make(map[string]bool)
	for _, metric := range c.metrics {
		names[metric.GetName()] = true
	}
	
	result := make([]string, 0, len(names))
	for name := range names {
		result = append(result, name)
	}
	return result
}

// GetMetricsByType 根据类型获取指标
func (c *StandardMetricCollector) GetMetricsByType(metricType MetricType) []Metric {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	var result []Metric
	for _, metric := range c.metrics {
		if metric.GetType() == metricType {
			result = append(result, metric)
		}
	}
	return result
}
