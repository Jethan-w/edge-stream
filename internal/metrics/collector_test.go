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
	"strings"
	"testing"
	"time"

	io_prometheus_client "github.com/prometheus/client_model/go"
)

// Test constants
const (
	testService   = "test"
	testOperation = "test_operation"
	testComponent = "test_component"
	testQueue     = "test_queue"
	timeoutError  = "timeout"
)

// TestStandardMetricCollector 测试标准指标收集器
func TestStandardMetricCollector(t *testing.T) {
	collector := NewStandardMetricCollector()

	// 测试计数器
	collector.RecordCounter("test_counter", 1.0, map[string]string{"label1": "value1"})
	collector.RecordCounter("test_counter", 2.0, map[string]string{"label1": "value1"})

	// 测试仪表盘
	collector.RecordGauge("test_gauge", 42.0, map[string]string{"label2": "value2"})

	// 测试直方图
	collector.RecordHistogram("test_histogram", 0.5, map[string]string{"label3": "value3"})

	// 验证指标数量
	metrics := collector.GetMetrics()
	if len(metrics) != 3 {
		t.Errorf("Expected 3 metrics, got %d", len(metrics))
	}

	// 验证指标类型
	for _, metric := range metrics {
		switch metric.GetName() {
		case "test_counter":
			if metric.GetType() != Counter {
				t.Errorf("Expected Counter type for test_counter, got %s", metric.GetType())
			}
		case "test_gauge":
			if metric.GetType() != Gauge {
				t.Errorf("Expected Gauge type for test_gauge, got %s", metric.GetType())
			}
		case "test_histogram":
			if metric.GetType() != Histogram {
				t.Errorf("Expected Histogram type for test_histogram, got %s", metric.GetType())
			}
		}
	}
}

// TestMetricCollectorLatency 测试延迟记录
func TestMetricCollectorLatency(t *testing.T) {
	collector := NewStandardMetricCollector()

	// 记录延迟
	duration := 100 * time.Millisecond
	collector.RecordLatency(testOperation, duration, map[string]string{"service": testService})

	// 验证延迟指标被创建
	metrics := collector.GetMetrics()
	found := false
	for _, metric := range metrics {
		if metric.GetName() != "latency_seconds" || metric.GetType() != Histogram {
			continue
		}
		found = true
		labels := metric.GetLabels()
		if labels["operation"] != testOperation {
			t.Errorf("Expected operation label to be '%s', got '%s'", testOperation, labels["operation"])
		}
		if labels["service"] != testService {
			t.Errorf("Expected service label to be '%s', got '%s'", testService, labels["service"])
		}
		break
	}
	if !found {
		t.Error("Latency metric not found")
	}
}

// TestMetricCollectorThroughput 测试吞吐量记录
func TestMetricCollectorThroughput(t *testing.T) {
	collector := NewStandardMetricCollector()

	// 记录吞吐量
	collector.RecordThroughput(testOperation, 100, map[string]string{"service": testService})

	// 验证吞吐量指标被创建
	metrics := collector.GetMetrics()
	found := false
	for _, metric := range metrics {
		if metric.GetName() == "throughput_total" && metric.GetType() == Counter {
			found = true
			labels := metric.GetLabels()
			if labels["operation"] != testOperation {
				t.Errorf("Expected operation label to be '%s', got '%s'", testOperation, labels["operation"])
			}
			break
		}
	}
	if !found {
		t.Error("Throughput metric not found")
	}
}

// TestMetricCollectorError 测试错误记录
func TestMetricCollectorError(t *testing.T) {
	collector := NewStandardMetricCollector()

	// 记录错误
	collector.RecordError(testOperation, timeoutError, map[string]string{"service": testService})

	// 验证错误指标被创建
	metrics := collector.GetMetrics()
	found := false
	for _, metric := range metrics {
		if metric.GetName() != "errors_total" || metric.GetType() != Counter {
			continue
		}
		found = true
		labels := metric.GetLabels()
		if labels["operation"] != testOperation {
			t.Errorf("Expected operation label to be '%s', got '%s'", testOperation, labels["operation"])
		}
		if labels["error_type"] != timeoutError {
			t.Errorf("Expected error_type label to be '%s', got '%s'", timeoutError, labels["error_type"])
		}
		break
	}
	if !found {
		t.Error("Error metric not found")
	}
}

// TestMetricCollectorMemoryUsage 测试内存使用记录
func TestMetricCollectorMemoryUsage(t *testing.T) {
	collector := NewStandardMetricCollector()

	// 记录内存使用
	collector.RecordMemoryUsage(testComponent, 1024*1024) // 1MB

	// 验证内存使用指标被创建
	metrics := collector.GetMetrics()
	found := false
	for _, metric := range metrics {
		if metric.GetName() == "memory_usage_bytes" && metric.GetType() == Gauge {
			found = true
			labels := metric.GetLabels()
			if labels["component"] != testComponent {
				t.Errorf("Expected component label to be '%s', got '%s'", testComponent, labels["component"])
			}
			break
		}
	}
	if !found {
		t.Error("Memory usage metric not found")
	}
}

// TestMetricCollectorQueueDepth 测试队列深度记录
func TestMetricCollectorQueueDepth(t *testing.T) {
	collector := NewStandardMetricCollector()

	// 记录队列深度
	collector.RecordQueueDepth(testQueue, 50)

	// 验证队列深度指标被创建
	metrics := collector.GetMetrics()
	found := false
	for _, metric := range metrics {
		if metric.GetName() == "queue_depth" && metric.GetType() == Gauge {
			found = true
			labels := metric.GetLabels()
			if labels["queue"] != testQueue {
				t.Errorf("Expected queue label to be '%s', got '%s'", testQueue, labels["queue"])
			}
			break
		}
	}
	if !found {
		t.Error("Queue depth metric not found")
	}
}

// TestMetricCollectorConnectionCount 测试连接数记录
func TestMetricCollectorConnectionCount(t *testing.T) {
	collector := NewStandardMetricCollector()

	// 记录连接数
	collector.RecordConnectionCount(testService, 25)

	// 验证连接数指标被创建
	metrics := collector.GetMetrics()
	found := false
	for _, metric := range metrics {
		if metric.GetName() == "connection_count" && metric.GetType() == Gauge {
			found = true
			labels := metric.GetLabels()
			if labels["service"] != testService {
				t.Errorf("Expected service label to be '%s', got '%s'", testService, labels["service"])
			}
			break
		}
	}
	if !found {
		t.Error("Connection count metric not found")
	}
}

// TestMetricCollectorReset 测试重置功能
func TestMetricCollectorReset(t *testing.T) {
	collector := NewStandardMetricCollector()

	// 添加一些指标
	collector.RecordCounter("test_counter", 1.0, nil)
	collector.RecordGauge("test_gauge", 42.0, nil)

	// 验证指标存在
	if len(collector.GetMetrics()) != 2 {
		t.Errorf("Expected 2 metrics before reset, got %d", len(collector.GetMetrics()))
	}

	// 重置
	collector.Reset()

	// 验证指标被清空
	if len(collector.GetMetrics()) != 0 {
		t.Errorf("Expected 0 metrics after reset, got %d", len(collector.GetMetrics()))
	}
}

// TestMetricCollectorExport 测试导出功能
func TestMetricCollectorExport(t *testing.T) {
	collector := NewStandardMetricCollector()

	// 添加一些指标
	collector.RecordCounter("test_counter", 1.0, map[string]string{"label1": "value1"})
	collector.RecordGauge("test_gauge", 42.0, map[string]string{"label2": "value2"})
	collector.RecordHistogram("test_histogram", 0.5, map[string]string{"label3": "value3"})

	// 测试JSON导出
	jsonData, err := collector.Export("json")
	if err != nil {
		t.Errorf("Failed to export JSON: %v", err)
		return
	}
	if len(jsonData) == 0 {
		t.Error("JSON export returned empty data")
	}

	// 测试Prometheus导出
	prometheusData, err := collector.Export("prometheus")
	if err != nil {
		t.Errorf("Failed to export Prometheus: %v", err)
		return
	}
	if len(prometheusData) == 0 {
		t.Error("Prometheus export returned empty data")
	}

	// 测试不支持的格式
	_, err = collector.Export("unsupported")
	if err == nil {
		t.Error("Expected error for unsupported format")
	}
}

// TestMetricCollectorConcurrency 测试并发安全性
func TestMetricCollectorConcurrency(t *testing.T) {
	collector := NewStandardMetricCollector()

	// 并发写入测试
	t.Run("ConcurrentWrites", func(t *testing.T) {
		const numGoroutines = 10
		const numOperations = 100

		done := make(chan bool, numGoroutines)

		// 启动多个goroutine并发写入
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				for j := 0; j < numOperations; j++ {
					collector.RecordCounter("concurrent_counter", 1.0, map[string]string{"worker": string(rune('A' + id))})
					collector.RecordGauge("concurrent_gauge", float64(j), map[string]string{"worker": string(rune('A' + id))})
					collector.RecordHistogram("concurrent_histogram", float64(j)*0.01, map[string]string{"worker": string(rune('A' + id))})
				}
				done <- true
			}(i)
		}

		// 等待所有goroutine完成
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// 验证指标数量
		metrics := collector.GetMetrics()
		if len(metrics) == 0 {
			t.Error("No metrics found after concurrent operations")
		}
	})

	// 并发读写测试
	t.Run("ConcurrentReadsAndWrites", func(t *testing.T) {
		const numReaders = 5
		const numWriters = 5
		const numOperations = 50

		done := make(chan bool, numReaders+numWriters)

		// 启动写入goroutine
		for i := 0; i < numWriters; i++ {
			go func(id int) {
				for j := 0; j < numOperations; j++ {
					collector.RecordCounter("rw_counter", 1.0, map[string]string{"writer": string(rune('A' + id))})
				}
				done <- true
			}(i)
		}

		// 启动读取goroutine
		for i := 0; i < numReaders; i++ {
			go func() {
				for j := 0; j < numOperations; j++ {
					_ = collector.GetMetrics()
					_ = collector.GetMetric("rw_counter")
				}
				done <- true
			}()
		}

		// 等待所有goroutine完成
		for i := 0; i < numReaders+numWriters; i++ {
			<-done
		}
	})
}

// TestMetricCollectorDataTypes 测试不同数据类型
func TestMetricCollectorDataTypes(t *testing.T) {
	collector := NewStandardMetricCollector()

	// 测试不同的数值类型
	collector.RecordCounter("int_counter", 42, nil)
	collector.RecordGauge("float_gauge", 3.14159, nil)
	collector.RecordHistogram("small_histogram", 0.001, nil)
	collector.RecordHistogram("large_histogram", 1000.0, nil)

	// 验证指标被正确创建
	metrics := collector.GetMetrics()
	if len(metrics) != 4 {
		t.Errorf("Expected 4 metrics, got %d", len(metrics))
	}
}

// TestMetricCollectorEdgeCases 测试边界情况
func TestMetricCollectorEdgeCases(t *testing.T) {
	collector := NewStandardMetricCollector()

	// 测试空标签
	collector.RecordCounter("empty_labels_counter", 1.0, map[string]string{})
	collector.RecordCounter("nil_labels_counter", 1.0, nil)

	// 测试零值
	collector.RecordGauge("zero_gauge", 0.0, nil)
	collector.RecordHistogram("zero_histogram", 0.0, nil)

	// 测试负值
	collector.RecordGauge("negative_gauge", -42.0, nil)

	// 验证指标被正确创建
	metrics := collector.GetMetrics()
	if len(metrics) != 5 {
		t.Errorf("Expected 5 metrics, got %d", len(metrics))
	}
}

// TestMetricCollectorGetMetricsByType 测试按类型获取指标
func TestMetricCollectorGetMetricsByType(t *testing.T) {
	collector := NewStandardMetricCollector()

	// 添加不同类型的指标
	collector.RecordCounter("counter1", 1.0, nil)
	collector.RecordCounter("counter2", 2.0, nil)
	collector.RecordGauge("gauge1", 10.0, nil)
	collector.RecordHistogram("histogram1", 0.5, nil)

	// 测试获取计数器指标
	counters := collector.GetMetricsByType(Counter)
	if len(counters) != 2 {
		t.Errorf("Expected 2 counter metrics, got %d", len(counters))
	}

	// 测试获取仪表盘指标
	gauges := collector.GetMetricsByType(Gauge)
	if len(gauges) != 1 {
		t.Errorf("Expected 1 gauge metric, got %d", len(gauges))
	}

	// 测试获取直方图指标
	histograms := collector.GetMetricsByType(Histogram)
	if len(histograms) != 1 {
		t.Errorf("Expected 1 histogram metric, got %d", len(histograms))
	}
}

// TestMetricCollectorGetMetricNames 测试获取指标名称
func TestMetricCollectorGetMetricNames(t *testing.T) {
	collector := NewStandardMetricCollector()

	// 添加指标
	collector.RecordCounter("test_counter", 1.0, nil)
	collector.RecordGauge("test_gauge", 10.0, nil)
	collector.RecordHistogram("test_histogram", 0.5, nil)

	// 获取指标名称
	names := collector.GetMetricNames()
	if len(names) != 3 {
		t.Errorf("Expected 3 metric names, got %d", len(names))
	}

	// 验证名称包含预期的指标
	expectedNames := map[string]bool{
		"test_counter":   true,
		"test_gauge":     true,
		"test_histogram": true,
	}

	for _, name := range names {
		if !expectedNames[name] {
			t.Errorf("Unexpected metric name: %s", name)
		}
	}
}

// TestPrometheusIntegration 测试Prometheus集成
func TestPrometheusIntegration(t *testing.T) {
	collector := NewStandardMetricCollector()

	// 添加指标
	collector.RecordCounter("prometheus_counter", 5.0, map[string]string{"service": "test"})
	collector.RecordGauge("prometheus_gauge", 100.0, map[string]string{"service": "test"})
	collector.RecordHistogram("prometheus_histogram", 0.25, map[string]string{"service": "test"})

	// 获取Prometheus注册表
	registry := collector.GetRegistry()
	if registry == nil {
		t.Error("Registry should not be nil")
	}

	// 收集指标
	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Errorf("Failed to gather metrics: %v", err)
		return
	}

	if len(metricFamilies) == 0 {
		t.Error("No metric families found")
	}

	// 验证指标类型
	for _, mf := range metricFamilies {
		name := mf.GetName()
		switch name {
		case "prometheus_counter":
			if mf.GetType() != io_prometheus_client.MetricType_COUNTER {
				t.Errorf("Expected COUNTER type for %s", name)
			}
		case "prometheus_gauge":
			if mf.GetType() != io_prometheus_client.MetricType_GAUGE {
				t.Errorf("Expected GAUGE type for %s", name)
			}
		case "prometheus_histogram":
			if mf.GetType() != io_prometheus_client.MetricType_HISTOGRAM {
				t.Errorf("Expected HISTOGRAM type for %s", name)
			}
		}
	}
}

// TestPerformanceStandards 测试性能标准
func TestPerformanceStandards(t *testing.T) {
	collector := NewStandardMetricCollector()

	// 测试大量指标的性能
	t.Run("BulkOperations", func(t *testing.T) {
		start := time.Now()
		const numMetrics = 1000

		for i := 0; i < numMetrics; i++ {
			collector.RecordCounter("bulk_counter", 1.0, map[string]string{"id": string(rune(i))})
		}

		duration := time.Since(start)
		if duration > 100*time.Millisecond {
			t.Errorf("Bulk operations took too long: %v (expected <= 100ms)", duration)
		}
	})

	// 测试导出性能
	t.Run("ExportPerformance", func(t *testing.T) {
		// 添加一些指标
		for i := 0; i < 100; i++ {
			collector.RecordCounter("export_counter", 1.0, map[string]string{"id": string(rune(i))})
			collector.RecordGauge("export_gauge", float64(i), map[string]string{"id": string(rune(i))})
		}

		start := time.Now()
		_, err := collector.Export("prometheus")
		if err != nil {
			t.Errorf("Export failed: %v", err)
			return
		}
		duration := time.Since(start)

		// 放宽性能要求，因为Prometheus导出可能比较慢
		if duration > 500*time.Millisecond {
			t.Errorf("Export took too long: %v (expected <= 500ms)", duration)
		}
	})
}

// TestMetricCollectorConcurrencyAdditional 额外的并发测试
func TestMetricCollectorConcurrencyAdditional(t *testing.T) {
	collector := NewStandardMetricCollector()

	// 测试重置期间的并发操作
	t.Run("ConcurrentReset", func(t *testing.T) {
		const numGoroutines = 5
		done := make(chan bool, numGoroutines)

		// 启动写入goroutine
		for i := 0; i < numGoroutines-1; i++ {
			go func(id int) {
				for j := 0; j < 50; j++ {
					collector.RecordCounter("reset_counter", 1.0, map[string]string{"worker": string(rune('A' + id))})
					time.Sleep(time.Microsecond)
				}
				done <- true
			}(i)
		}

		// 启动重置goroutine
		go func() {
			for i := 0; i < 10; i++ {
				collector.Reset()
				time.Sleep(5 * time.Millisecond)
			}
			done <- true
		}()

		// 等待所有goroutine完成
		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})

	// 测试导出期间的并发操作
	t.Run("ConcurrentExport", func(t *testing.T) {
		const numGoroutines = 3
		done := make(chan bool, numGoroutines)

		// 添加一些初始指标
		for i := 0; i < 10; i++ {
			collector.RecordCounter("export_test_counter", 1.0, map[string]string{"id": string(rune('A' + i))})
		}

		// 启动写入goroutine
		go func() {
			for i := 0; i < 20; i++ {
				collector.RecordGauge("export_test_gauge", float64(i), nil)
				time.Sleep(time.Millisecond)
			}
			done <- true
		}()

		// 启动导出goroutine
		for i := 0; i < 2; i++ {
			go func() {
				for j := 0; j < 5; j++ {
					_, err := collector.Export("json")
					if err != nil {
						t.Errorf("Export failed: %v", err)
						return
					}
					time.Sleep(2 * time.Millisecond)
				}
				done <- true
			}()
		}

		// 等待所有goroutine完成
		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})
}

// BenchmarkMetricCollector 性能基准测试
func BenchmarkMetricCollector(b *testing.B) {
	collector := NewStandardMetricCollector()

	b.Run("RecordCounter", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			collector.RecordCounter("bench_counter", 1.0, map[string]string{"id": "test"})
		}
	})

	b.Run("RecordGauge", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			collector.RecordGauge("bench_gauge", float64(i), map[string]string{"id": "test"})
		}
	})

	b.Run("RecordHistogram", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			collector.RecordHistogram("bench_histogram", float64(i)*0.01, map[string]string{"id": "test"})
		}
	})

	b.Run("GetMetrics", func(b *testing.B) {
		// 预先添加一些指标
		for i := 0; i < 100; i++ {
			collector.RecordCounter("pre_counter", 1.0, map[string]string{"id": string(rune(i))})
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = collector.GetMetrics()
		}
	})

	b.Run("Export", func(b *testing.B) {
		// 预先添加一些指标
		for i := 0; i < 50; i++ {
			collector.RecordCounter("export_counter", 1.0, map[string]string{"id": string(rune(i))})
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			data, err := collector.Export("json")
			if err != nil {
				b.Errorf("Export failed: %v", err)
			}
			_ = data
		}
	})
}

// TestPrometheusExportFormat 测试Prometheus导出格式
func TestPrometheusExportFormat(t *testing.T) {
	collector := NewStandardMetricCollector()

	// 添加指标
	collector.RecordCounter("test_requests_total", 100, map[string]string{"method": "GET", "status": "200"})
	collector.RecordGauge("test_memory_usage_bytes", 1024*1024, map[string]string{"component": "cache"})
	collector.RecordHistogram("test_request_duration_seconds", 0.25, map[string]string{"endpoint": "/api/v1"})

	// 导出Prometheus格式
	data, err := collector.Export("prometheus")
	if err != nil {
		t.Fatalf("Failed to export Prometheus format: %v", err)
	}

	output := string(data)

	// 验证输出包含预期的指标
	if !strings.Contains(output, "test_requests_total") {
		t.Error("Prometheus output should contain test_requests_total")
	}
	if !strings.Contains(output, "test_memory_usage_bytes") {
		t.Error("Prometheus output should contain test_memory_usage_bytes")
	}
	if !strings.Contains(output, "test_request_duration_seconds") {
		t.Error("Prometheus output should contain test_request_duration_seconds")
	}

	// 验证直方图包含桶信息
	if !strings.Contains(output, "_bucket") {
		t.Error("Histogram should contain bucket information")
	}
	if !strings.Contains(output, "_count") {
		t.Error("Histogram should contain count information")
	}
	if !strings.Contains(output, "_sum") {
		t.Error("Histogram should contain sum information")
	}
}
