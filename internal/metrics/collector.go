package metrics

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// StandardMetricCollector 标准指标收集器实现
type StandardMetricCollector struct {
	registry *MetricRegistry
	mu       sync.RWMutex
	// 性能优化：使用缓存减少锁竞争
	cache   map[string]*StandardMetric
	cacheMu sync.RWMutex
}

// NewStandardMetricCollector 创建标准指标收集器
func NewStandardMetricCollector() *StandardMetricCollector {
	return &StandardMetricCollector{
		registry: NewMetricRegistry(),
		cache:    make(map[string]*StandardMetric),
	}
}

// RecordCounter 记录计数器指标
func (c *StandardMetricCollector) RecordCounter(name string, value float64, labels map[string]string) {
	metricName := c.buildMetricName(name, labels)

	// 首先尝试从缓存中获取
	c.cacheMu.RLock()
	if cached, exists := c.cache[metricName]; exists {
		c.cacheMu.RUnlock()
		cached.AddValue(value)
		return
	}
	c.cacheMu.RUnlock()

	// 缓存中不存在，需要创建新的指标
	c.mu.Lock()
	defer c.mu.Unlock()

	// 双重检查，防止并发创建
	c.cacheMu.Lock()
	if cached, exists := c.cache[metricName]; exists {
		c.cacheMu.Unlock()
		cached.AddValue(value)
		return
	}

	metric := NewStandardMetric(metricName, Counter, value, labels)
	c.registry.Register(metric)
	c.cache[metricName] = metric
	c.cacheMu.Unlock()
}

// RecordGauge 记录仪表盘指标
func (c *StandardMetricCollector) RecordGauge(name string, value float64, labels map[string]string) {
	metricName := c.buildMetricName(name, labels)

	// 首先尝试从缓存中获取
	c.cacheMu.RLock()
	if cached, exists := c.cache[metricName]; exists {
		c.cacheMu.RUnlock()
		cached.SetValue(value)
		return
	}
	c.cacheMu.RUnlock()

	// 缓存中不存在，需要创建新的指标
	c.mu.Lock()
	defer c.mu.Unlock()

	// 双重检查，防止并发创建
	c.cacheMu.Lock()
	if cached, exists := c.cache[metricName]; exists {
		c.cacheMu.Unlock()
		cached.SetValue(value)
		return
	}

	metric := NewStandardMetric(metricName, Gauge, value, labels)
	c.registry.Register(metric)
	c.cache[metricName] = metric
	c.cacheMu.Unlock()
}

// RecordHistogram 记录直方图指标
func (c *StandardMetricCollector) RecordHistogram(name string, value float64, labels map[string]string) {
	metricName := c.buildMetricName(name, labels)

	// 简化的直方图实现，使用预定义的桶
	buckets := []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
	histogramData := &HistogramData{
		Buckets: make([]HistogramBucket, len(buckets)),
		Count:   1,
		Sum:     value,
	}

	for i, bound := range buckets {
		histogramData.Buckets[i] = HistogramBucket{
			UpperBound: bound,
			Count:      0,
		}
		if value <= bound {
			histogramData.Buckets[i].Count = 1
		}
	}

	// 首先尝试从缓存中获取
	c.cacheMu.RLock()
	if cached, exists := c.cache[metricName]; exists {
		c.cacheMu.RUnlock()
		// 对于直方图，我们需要更新现有数据
		if existingData, ok := cached.GetValue().(*HistogramData); ok {
			existingData.Count++
			existingData.Sum += value
			for i, bound := range buckets {
				if value <= bound {
					existingData.Buckets[i].Count++
				}
			}
			cached.SetValue(existingData)
		}
		return
	}
	c.cacheMu.RUnlock()

	// 缓存中不存在，需要创建新的指标
	c.mu.Lock()
	defer c.mu.Unlock()

	// 双重检查，防止并发创建
	c.cacheMu.Lock()
	if cached, exists := c.cache[metricName]; exists {
		c.cacheMu.Unlock()
		if existingData, ok := cached.GetValue().(*HistogramData); ok {
			existingData.Count++
			existingData.Sum += value
			for i, bound := range buckets {
				if value <= bound {
					existingData.Buckets[i].Count++
				}
			}
			cached.SetValue(existingData)
		}
		return
	}

	metric := NewStandardMetric(metricName, Histogram, histogramData, labels)
	c.registry.Register(metric)
	c.cache[metricName] = metric
	c.cacheMu.Unlock()
}

// RecordLatency 记录延迟指标
func (c *StandardMetricCollector) RecordLatency(operation string, duration time.Duration, labels map[string]string) {
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["operation"] = operation

	// 记录为直方图（毫秒）
	latencyMs := float64(duration.Nanoseconds()) / 1e6
	c.RecordHistogram("latency_ms", latencyMs, labels)

	// 同时记录为仪表盘
	c.RecordGauge("current_latency_ms", latencyMs, labels)
}

// RecordThroughput 记录吞吐量指标
func (c *StandardMetricCollector) RecordThroughput(operation string, count int64, labels map[string]string) {
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["operation"] = operation

	// 记录为计数器
	c.RecordCounter("throughput_total", float64(count), labels)

	// 记录当前速率
	c.RecordGauge("current_throughput_rate", float64(count), labels)
}

// RecordError 记录错误指标
func (c *StandardMetricCollector) RecordError(operation string, errorType string, labels map[string]string) {
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["operation"] = operation
	labels["error_type"] = errorType

	// 记录错误计数
	c.RecordCounter("errors_total", 1, labels)
}

// GetMetrics 获取所有指标
func (c *StandardMetricCollector) GetMetrics() []Metric {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.registry.GetAll()
}

// GetMetric 获取指定指标
func (c *StandardMetricCollector) GetMetric(name string) Metric {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.registry.Get(name)
}

// Reset 重置指标
func (c *StandardMetricCollector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	c.registry.Clear()
	c.cache = make(map[string]*StandardMetric)
}

// Export 导出指标
func (c *StandardMetricCollector) Export(format string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	switch strings.ToLower(format) {
	case "json":
		return c.exportJSON()
	case "prometheus":
		return c.exportPrometheus()
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// exportJSON 导出为JSON格式
func (c *StandardMetricCollector) exportJSON() ([]byte, error) {
	metrics := c.registry.GetAll()
	snapshot := MetricSnapshot{
		Timestamp: time.Now(),
		Metrics:   make(map[string]interface{}),
	}

	for _, metric := range metrics {
		snapshot.Metrics[metric.GetName()] = map[string]interface{}{
			"type":      metric.GetType(),
			"value":     metric.GetValue(),
			"labels":    metric.GetLabels(),
			"timestamp": metric.GetTimestamp(),
		}
	}

	return json.MarshalIndent(snapshot, "", "  ")
}

// exportPrometheus 导出为Prometheus格式
func (c *StandardMetricCollector) exportPrometheus() ([]byte, error) {
	metrics := c.registry.GetAll()
	var lines []string

	for _, metric := range metrics {
		name := metric.GetName()
		value := metric.GetValue()
		labels := metric.GetLabels()

		// 构建标签字符串
		labelStr := c.buildPrometheusLabels(labels)

		switch metric.GetType() {
		case Counter:
			lines = append(lines, fmt.Sprintf("# TYPE %s counter", name))
			lines = append(lines, fmt.Sprintf("%s%s %v", name, labelStr, value))
		case Gauge:
			lines = append(lines, fmt.Sprintf("# TYPE %s gauge", name))
			lines = append(lines, fmt.Sprintf("%s%s %v", name, labelStr, value))
		case Histogram:
			lines = append(lines, fmt.Sprintf("# TYPE %s histogram", name))
			if histData, ok := value.(*HistogramData); ok {
				for _, bucket := range histData.Buckets {
					bucketLabels := c.buildPrometheusLabels(mergeMaps(labels, map[string]string{"le": fmt.Sprintf("%g", bucket.UpperBound)}))
					lines = append(lines, fmt.Sprintf("%s_bucket%s %d", name, bucketLabels, bucket.Count))
				}
				lines = append(lines, fmt.Sprintf("%s_count%s %d", name, labelStr, histData.Count))
				lines = append(lines, fmt.Sprintf("%s_sum%s %g", name, labelStr, histData.Sum))
			}
		}
	}

	return []byte(strings.Join(lines, "\n")), nil
}

// buildMetricName 构建指标名称
func (c *StandardMetricCollector) buildMetricName(name string, labels map[string]string) string {
	if len(labels) == 0 {
		return name
	}

	// 为了简化，这里直接使用原名称
	// 在实际实现中，可能需要更复杂的命名策略
	return name
}

// buildPrometheusLabels 构建Prometheus标签字符串
func (c *StandardMetricCollector) buildPrometheusLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}

	var pairs []string
	for key, value := range labels {
		pairs = append(pairs, fmt.Sprintf(`%s="%s"`, key, value))
	}

	sort.Strings(pairs)
	return "{" + strings.Join(pairs, ",") + "}"
}

// mergeMaps 合并两个map
func mergeMaps(map1, map2 map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range map1 {
		result[k] = v
	}
	for k, v := range map2 {
		result[k] = v
	}
	return result
}

// GetSnapshot 获取指标快照
func (c *StandardMetricCollector) GetSnapshot() MetricSnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()

	metrics := c.registry.GetAll()
	snapshot := MetricSnapshot{
		Timestamp: time.Now(),
		Metrics:   make(map[string]interface{}),
	}

	for _, metric := range metrics {
		snapshot.Metrics[metric.GetName()] = map[string]interface{}{
			"type":      metric.GetType(),
			"value":     metric.GetValue(),
			"labels":    metric.GetLabels(),
			"timestamp": metric.GetTimestamp(),
		}
	}

	return snapshot
}

// RecordProcessingTime 记录处理时间
func (c *StandardMetricCollector) RecordProcessingTime(component string, duration time.Duration) {
	labels := map[string]string{"component": component}
	c.RecordLatency("processing_time", duration, labels)
}

// RecordMemoryUsage 记录内存使用情况
func (c *StandardMetricCollector) RecordMemoryUsage(component string, bytes int64) {
	labels := map[string]string{"component": component}
	c.RecordGauge("memory_usage_bytes", float64(bytes), labels)
}

// RecordQueueDepth 记录队列深度
func (c *StandardMetricCollector) RecordQueueDepth(queueName string, depth int64) {
	labels := map[string]string{"queue": queueName}
	c.RecordGauge("queue_depth", float64(depth), labels)
}

// RecordConnectionCount 记录连接数
func (c *StandardMetricCollector) RecordConnectionCount(service string, count int64) {
	labels := map[string]string{"service": service}
	c.RecordGauge("connection_count", float64(count), labels)
}
