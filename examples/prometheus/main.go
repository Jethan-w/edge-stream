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

package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/crazy/edge-stream/internal/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	fmt.Println("🚀 Edge Stream Prometheus Integration Demo")
	fmt.Println("===========================================")

	// 创建指标收集器
	collector := metrics.NewStandardMetricCollector()

	// 获取注册表
	registry := collector.GetRegistry()

	// 启动HTTP服务器提供Prometheus指标
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

		fmt.Println("📊 Prometheus metrics server started at http://localhost:8080/metrics")
		if err := http.ListenAndServe(":8080", mux); err != nil {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// 等待服务器启动
	time.Sleep(1 * time.Second)

	// 演示指标记录
	fmt.Println("\n📈 Recording sample metrics...")

	// 记录计数器指标
	collector.RecordCounter("requests_total", 1.0, map[string]string{
		"method": "GET",
		"status": "200",
	})
	collector.RecordCounter("requests_total", 1.0, map[string]string{
		"method": "POST",
		"status": "201",
	})

	// 记录仪表盘指标
	collector.RecordGauge("active_connections", 42.0, map[string]string{
		"server": "web-01",
	})
	collector.RecordGauge("memory_usage_bytes", 1024*1024*256, map[string]string{
		"component": "processor",
	})

	// 记录直方图指标
	collector.RecordHistogram("request_duration_seconds", 0.123, map[string]string{
		"endpoint": "/api/v1/data",
	})
	collector.RecordHistogram("request_duration_seconds", 0.456, map[string]string{
		"endpoint": "/api/v1/status",
	})

	// 记录摘要指标（使用直方图代替）
	collector.RecordHistogram("response_size_bytes", 2048, map[string]string{
		"content_type": "application/json",
	})

	fmt.Println("✅ Sample metrics recorded successfully")

	// 导出指标
	fmt.Println("\n📤 Exporting metrics...")
	exportData, err := collector.Export("prometheus")
	if err != nil {
		log.Fatalf("Failed to export metrics: %v", err)
	}

	fmt.Printf("📊 Exported %d bytes of metrics data\n", len(exportData))

	// 显示指标统计
	fmt.Println("\n📋 Metrics Summary:")
	metricNames := collector.GetMetricNames()
	fmt.Printf("   Total metrics: %d\n", len(metricNames))
	for _, name := range metricNames {
		fmt.Printf("   - %s\n", name)
	}

	// 按类型分组显示指标
	fmt.Println("\n🏷️  Metrics by Type:")
	counters := collector.GetMetricsByType(metrics.Counter)
	gauges := collector.GetMetricsByType(metrics.Gauge)
	histograms := collector.GetMetricsByType(metrics.Histogram)
	summaries := collector.GetMetricsByType(metrics.Summary)

	fmt.Printf("   Counters: %d\n", len(counters))
	fmt.Printf("   Gauges: %d\n", len(gauges))
	fmt.Printf("   Histograms: %d\n", len(histograms))
	fmt.Printf("   Summaries: %d\n", len(summaries))

	fmt.Println("\n🌐 Access Prometheus metrics at: http://localhost:8080/metrics")
	fmt.Println("\n⏰ Demo will run for 30 seconds...")

	// 模拟持续的指标更新
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	counter := 0
	for {
		select {
		case <-ticker.C:
			counter++

			// 更新指标
			collector.RecordCounter("demo_operations_total", 1.0, map[string]string{
				"operation": "update",
			})
			collector.RecordGauge("demo_active_tasks", float64(counter*3), nil)
			collector.RecordHistogram("demo_task_duration_seconds", float64(counter)*0.1, nil)

			fmt.Printf("📊 Updated metrics (iteration %d)\n", counter)

			if counter >= 15 {
				fmt.Println("\n🎉 Demo completed successfully!")
				fmt.Println("\n📊 Final metrics export:")

				// 最终导出
				finalExport, err := collector.Export("prometheus")
				if err != nil {
					log.Printf("Final export error: %v", err)
				} else {
					fmt.Printf("   Exported %d bytes of metrics data\n", len(finalExport))
				}

				return
			}
		case <-time.After(35 * time.Second):
			fmt.Println("\n⏰ Demo timeout reached")
			return
		}
	}
}
