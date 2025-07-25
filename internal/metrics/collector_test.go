package metrics

import (
	"sync"
	"testing"
	"time"
)

func TestStandardMetricCollector(t *testing.T) {
	collector := NewStandardMetricCollector()

	// 测试计数器指标
	t.Run("Counter", func(t *testing.T) {
		labels := map[string]string{"component": "test"}
		collector.RecordCounter("test_counter", 1.0, labels)
		collector.RecordCounter("test_counter", 2.0, labels)

		metrics := collector.GetMetrics()
		if len(metrics) == 0 {
			t.Error("Expected metrics to be recorded")
		}

		found := false
		for _, metric := range metrics {
			if metric.GetName() == "test_counter" && metric.GetType() == Counter {
				found = true
				break
			}
		}
		if !found {
			t.Error("Counter metric not found")
		}
	})

	// 测试仪表盘指标
	t.Run("Gauge", func(t *testing.T) {
		labels := map[string]string{"service": "test"}
		collector.RecordGauge("test_gauge", 42.5, labels)

		metric := collector.GetMetric("test_gauge")
		if metric == nil {
			t.Error("Gauge metric not found")
		}
		if metric.GetType() != Gauge {
			t.Errorf("Expected Gauge type, got %v", metric.GetType())
		}
	})

	// 测试延迟指标
	t.Run("Latency", func(t *testing.T) {
		labels := map[string]string{"operation": "test_op"}
		collector.RecordLatency("test_operation", 100*time.Millisecond, labels)

		metrics := collector.GetMetrics()
		found := false
		for _, metric := range metrics {
			if metric.GetName() == "latency_ms" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Latency metric not found")
		}
	})

	// 测试错误指标
	t.Run("Error", func(t *testing.T) {
		labels := map[string]string{"component": "test"}
		collector.RecordError("test_operation", "validation_error", labels)

		metrics := collector.GetMetrics()
		found := false
		for _, metric := range metrics {
			if metric.GetName() == "errors_total" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Error metric not found")
		}
	})

	// 测试导出功能
	t.Run("Export", func(t *testing.T) {
		jsonData, err := collector.Export("json")
		if err != nil {
			t.Errorf("JSON export failed: %v", err)
		}
		if len(jsonData) == 0 {
			t.Error("JSON export returned empty data")
		}

		prometheusData, err := collector.Export("prometheus")
		if err != nil {
			t.Errorf("Prometheus export failed: %v", err)
		}
		if len(prometheusData) == 0 {
			t.Error("Prometheus export returned empty data")
		}
	})

	// 测试重置功能
	t.Run("Reset", func(t *testing.T) {
		collector.RecordCounter("reset_test", 1.0, nil)
		if len(collector.GetMetrics()) == 0 {
			t.Error("Expected metrics before reset")
		}

		collector.Reset()
		if len(collector.GetMetrics()) != 0 {
			t.Error("Expected no metrics after reset")
		}
	})
}

func TestMetricCollectorConcurrency(t *testing.T) {
	collector := NewStandardMetricCollector()

	// 并发写入测试
	t.Run("ConcurrentWrites", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 10
		numOperations := 100

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					labels := map[string]string{"worker": string(rune(id))}
					collector.RecordCounter("concurrent_test", 1.0, labels)
					collector.RecordGauge("concurrent_gauge", float64(j), labels)
					collector.RecordLatency("concurrent_latency", time.Millisecond, labels)
				}
			}(i)
		}

		wg.Wait()

		// 验证指标被正确记录
		metrics := collector.GetMetrics()
		if len(metrics) == 0 {
			t.Error("Expected metrics to be recorded in concurrent test")
		}
	})

	// 并发读写测试
	t.Run("ConcurrentReadWrite", func(t *testing.T) {
		var wg sync.WaitGroup

		// 写入goroutine
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				collector.RecordCounter("rw_test", 1.0, nil)
			}
		}()

		// 读取goroutine
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				collector.GetMetrics()
				collector.GetMetric("rw_test")
			}
		}()

		wg.Wait()
	})
}

func BenchmarkMetricCollector(b *testing.B) {
	collector := NewStandardMetricCollector()
	labels := map[string]string{"component": "benchmark"}

	b.Run("RecordCounter", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			collector.RecordCounter("bench_counter", 1.0, labels)
		}
	})

	b.Run("RecordGauge", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			collector.RecordGauge("bench_gauge", float64(i), labels)
		}
	})

	b.Run("RecordLatency", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			collector.RecordLatency("bench_latency", time.Millisecond, labels)
		}
	})

	b.Run("GetMetrics", func(b *testing.B) {
		// 预先添加一些指标
		for i := 0; i < 100; i++ {
			collector.RecordCounter("setup_counter", 1.0, labels)
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			collector.GetMetrics()
		}
	})

	b.Run("Export", func(b *testing.B) {
		// 预先添加一些指标
		for i := 0; i < 50; i++ {
			collector.RecordCounter("export_counter", 1.0, labels)
			collector.RecordGauge("export_gauge", float64(i), labels)
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			collector.Export("json")
		}
	})
}

// 性能标准测试
func TestPerformanceStandards(t *testing.T) {
	collector := NewStandardMetricCollector()
	labels := map[string]string{"component": "perf_test"}

	// 测试单次操作性能
	t.Run("SingleOperationPerformance", func(t *testing.T) {
		start := time.Now()
		collector.RecordCounter("perf_counter", 1.0, labels)
		duration := time.Since(start)

		// 单次操作应该在1ms内完成
		if duration > time.Millisecond {
			t.Errorf("Single operation took %v, expected < 1ms", duration)
		}
	})

	// 测试批量操作性能
	t.Run("BatchOperationPerformance", func(t *testing.T) {
		numOps := 10000
		start := time.Now()

		for i := 0; i < numOps; i++ {
			collector.RecordCounter("batch_counter", 1.0, labels)
		}

		duration := time.Since(start)
		opsPerSecond := float64(numOps) / duration.Seconds()

		// 应该能够处理至少100,000 ops/sec
		if opsPerSecond < 100000 {
			t.Errorf("Batch operations: %.0f ops/sec, expected >= 100,000 ops/sec", opsPerSecond)
		}
	})

	// 测试内存使用
	t.Run("MemoryUsage", func(t *testing.T) {
		// 记录大量指标
		for i := 0; i < 1000; i++ {
			collector.RecordCounter("memory_test", 1.0, labels)
			collector.RecordGauge("memory_gauge", float64(i), labels)
		}

		metrics := collector.GetMetrics()
		if len(metrics) == 0 {
			t.Error("Expected metrics to be recorded")
		}

		// 验证指标数量合理
		if len(metrics) > 10000 {
			t.Errorf("Too many metrics stored: %d, may indicate memory leak", len(metrics))
		}
	})
}
