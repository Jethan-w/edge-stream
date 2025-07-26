package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/crazy/edge-stream/internal/metrics"
)

func main() {
	fmt.Println("=== Edge Stream 监控系统示例 ===")

	// 1. 创建指标收集器
	collector := metrics.NewStandardMetricCollector()

	// 2. 模拟数据处理指标
	fmt.Println("\n=== 模拟数据处理指标 ===")
	simulateDataProcessing(collector)

	// 3. 模拟系统性能指标
	fmt.Println("\n=== 模拟系统性能指标 ===")
	simulateSystemMetrics(collector)

	// 4. 模拟错误指标
	fmt.Println("\n=== 模拟错误指标 ===")
	simulateErrors(collector)

	// 5. 显示所有指标
	fmt.Println("\n=== 当前所有指标 ===")
	displayMetrics(collector)

	// 6. 导出JSON格式
	fmt.Println("\n=== JSON格式导出 ===")
	exportJSON(collector)

	// 7. 导出Prometheus格式
	fmt.Println("\n=== Prometheus格式导出 ===")
	exportPrometheus(collector)



	fmt.Println("\n=== 监控系统示例完成 ===")
}

// simulateDataProcessing 模拟数据处理指标
func simulateDataProcessing(collector metrics.MetricCollector) {
	// 模拟处理不同类型的数据
	operations := []string{"mysql_read", "redis_write", "file_process", "data_transform"}

	for i := 0; i < 20; i++ {
		operation := operations[rand.Intn(len(operations))]

		// 模拟处理时间
		processingTime := time.Duration(rand.Intn(100)+10) * time.Millisecond
		collector.RecordLatency(operation, processingTime, map[string]string{
			"component": "data_processor",
			"version":   "1.0.0",
		})

		// 模拟吞吐量
		throughput := int64(rand.Intn(1000) + 100)
		collector.RecordThroughput(operation, throughput, map[string]string{
			"component": "data_processor",
		})

		fmt.Printf("处理操作: %s, 耗时: %v, 吞吐量: %d\n", operation, processingTime, throughput)
	}
}

// simulateSystemMetrics 模拟系统性能指标
func simulateSystemMetrics(collector metrics.MetricCollector) {
	components := []string{"flowfile_manager", "processor_engine", "window_manager", "sink_writer"}

	for _, component := range components {
		// 模拟内存使用
		memoryUsage := int64(rand.Intn(100)*1024*1024 + 50*1024*1024) // 50-150MB
		collector.RecordMemoryUsage(component, memoryUsage)

		// 模拟队列深度
		queueDepth := int64(rand.Intn(1000) + 10)
		collector.RecordQueueDepth(component+"_queue", queueDepth)

		// 模拟连接数
		connectionCount := int64(rand.Intn(50) + 5)
		collector.RecordConnectionCount(component, connectionCount)

		fmt.Printf("组件: %s, 内存: %.2fMB, 队列深度: %d, 连接数: %d\n",
			component, float64(memoryUsage)/(1024*1024), queueDepth, connectionCount)
	}
}

// simulateErrors 模拟错误指标
func simulateErrors(collector metrics.MetricCollector) {
	errorTypes := []string{"connection_timeout", "parse_error", "validation_failed", "disk_full"}
	operations := []string{"mysql_read", "redis_write", "file_process"}

	for i := 0; i < 10; i++ {
		operation := operations[rand.Intn(len(operations))]
		errorType := errorTypes[rand.Intn(len(errorTypes))]

		collector.RecordError(operation, errorType, map[string]string{
			"severity":  "error",
			"component": "data_processor",
		})

		fmt.Printf("错误记录: %s 操作发生 %s 错误\n", operation, errorType)
	}
}

// displayMetrics 显示所有指标
func displayMetrics(collector metrics.MetricCollector) {
	allMetrics := collector.GetMetrics()
	fmt.Printf("总共收集了 %d 个指标:\n", len(allMetrics))

	for i, metric := range allMetrics {
		if i >= 10 { // 只显示前10个指标
			fmt.Printf("... 还有 %d 个指标\n", len(allMetrics)-10)
			break
		}

		fmt.Printf("  %d. %s (%s): %v\n",
			i+1, metric.GetName(), metric.GetType(), metric.GetValue())

		labels := metric.GetLabels()
		if len(labels) > 0 {
			fmt.Printf("     标签: %v\n", labels)
		}
	}
}

// exportJSON 导出JSON格式
func exportJSON(collector metrics.MetricCollector) {
	jsonData, err := collector.Export("json")
	if err != nil {
		fmt.Printf("JSON导出失败: %v\n", err)
		return
	}

	fmt.Printf("JSON格式指标 (前500字符):\n%s...\n",
		truncateString(string(jsonData), 500))
}

// exportPrometheus 导出Prometheus格式
func exportPrometheus(collector metrics.MetricCollector) {
	prometheusData, err := collector.Export("prometheus")
	if err != nil {
		fmt.Printf("Prometheus导出失败: %v\n", err)
		return
	}

	fmt.Printf("Prometheus格式指标 (前800字符):\n%s...\n",
		truncateString(string(prometheusData), 800))
}



// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// contains 检查字符串是否包含任一关键词
func contains(s string, keywords []string) bool {
	for _, keyword := range keywords {
		if len(s) >= len(keyword) {
			for i := 0; i <= len(s)-len(keyword); i++ {
				if s[i:i+len(keyword)] == keyword {
					return true
				}
			}
		}
	}
	return false
}

func init() {
	// 设置随机种子
	rand.Seed(time.Now().UnixNano())
}
