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
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/crazy/edge-stream/internal/config"
	"github.com/crazy/edge-stream/internal/metrics"
)

func main() {
	// 创建上下文，用于优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. 配置管理示例
	fmt.Println("=== 配置管理示例 ===")
	configManager := setupConfigManager()
	// 配置管理器不需要显式关闭

	// 2. 监控系统示例
	fmt.Println("\n=== 监控系统示例 ===")
	metricsCollector := setupMonitoring(ctx)

	// 3. 模拟业务处理
	fmt.Println("\n=== 业务处理模拟 ===")
	simulateBusinessLogic(configManager, metricsCollector)

	// 4. 导出指标
	fmt.Println("\n=== 指标导出 ===")
	exportMetrics(metricsCollector)

	// 5. 等待信号退出
	waitForShutdown(ctx, cancel)
}

// setupConfigManager 设置配置管理器
func setupConfigManager() config.ConfigManager {
	// 创建配置管理器
	cm := config.NewStandardConfigManager("")

	// 从文件加载配置（如果存在）
	if err := cm.LoadConfig("config.yaml"); err != nil {
		log.Printf("Warning: Failed to load config file: %v", err)
	}

	// 设置默认配置
	setupDefaultConfig(cm)

	// 监听配置变更
	go func() {
		cm.Watch(context.Background(), func(key string, oldValue, newValue interface{}) {
			fmt.Printf("配置变更: %s = %v -> %v\n", key, oldValue, newValue)
		})
	}()

	// 显示当前配置
	displayConfig(cm)

	return cm
}

// setupDefaultConfig 设置默认配置
func setupDefaultConfig(cm config.ConfigManager) {
	// 数据库配置
	cm.Set("database.mysql.host", "localhost")
	cm.Set("database.mysql.port", 3306)
	cm.Set("database.mysql.username", "root")
	cm.Set("database.mysql.password", "secret123")
	cm.Set("database.mysql.database", "edge_stream")
	cm.Set("database.mysql.max_open_conns", 10)
	cm.Set("database.mysql.max_idle_conns", 5)
	cm.Set("database.mysql.timeout", "30s")

	// Redis配置
	cm.Set("redis.host", "localhost")
	cm.Set("redis.port", 6379)
	cm.Set("redis.database", 0)
	cm.Set("redis.pool_size", 10)

	// 服务器配置
	cm.Set("server.host", "0.0.0.0")
	cm.Set("server.port", 8080)
	cm.Set("server.read_timeout", "30s")
	cm.Set("server.write_timeout", "30s")

	// 处理配置
	cm.Set("processing.batch_size", 100)
	cm.Set("processing.worker_count", 4)
	cm.Set("processing.output_path", "./data/output")
	cm.Set("processing.file_format", "csv")

	// 监控配置
	cm.Set("monitoring.enabled", true)
	cm.Set("monitoring.collection_interval", "30s")
	cm.Set("monitoring.export_format", "json")
	cm.Set("monitoring.http_server.enabled", true)
	cm.Set("monitoring.http_server.port", 9090)
}

// displayConfig 显示当前配置
func displayConfig(cm config.ConfigManager) {
	fmt.Println("当前配置:")
	fmt.Printf("  数据库主机: %s\n", cm.GetString("database.mysql.host"))
	fmt.Printf("  数据库端口: %d\n", cm.GetInt("database.mysql.port"))
	fmt.Printf("  批处理大小: %d\n", cm.GetInt("processing.batch_size"))
	fmt.Printf("  工作线程数: %d\n", cm.GetInt("processing.worker_count"))
	fmt.Printf("  监控启用: %t\n", cm.GetBool("monitoring.enabled"))
	fmt.Printf("  收集间隔: %v\n", cm.GetDuration("monitoring.collection_interval"))

	// 获取密码配置
	password := cm.GetString("database.mysql.password")
	fmt.Printf("  数据库密码: %s\n", password)
}

// setupMonitoring 设置监控系统
func setupMonitoring(ctx context.Context) metrics.MetricCollector {
	// 创建指标收集器
	collector := metrics.NewStandardMetricCollector()

	fmt.Println("指标收集器已创建")

	return collector
}

// simulateBusinessLogic 模拟业务逻辑
func simulateBusinessLogic(cm config.ConfigManager, mc metrics.MetricCollector) {
	batchSize := cm.GetInt("processing.batch_size")
	workerCount := cm.GetInt("processing.worker_count")

	fmt.Printf("开始处理，批大小: %d, 工作线程: %d\n", batchSize, workerCount)

	// 模拟处理多个批次
	for i := 0; i < 5; i++ {
		processBatch(mc, i+1, batchSize)
		time.Sleep(1 * time.Second)
	}

	// 模拟一些错误
	mc.RecordError("processing_error", "validation_failed", map[string]string{
		"component": "data_processor",
	})

	// 记录系统指标
	mc.RecordMemoryUsage("data_processor", 1024*1024*50) // 50MB
	mc.RecordQueueDepth("input_queue", 25)
	mc.RecordConnectionCount("mysql", 8)
}

// processBatch 处理单个批次
func processBatch(mc metrics.MetricCollector, batchNum, batchSize int) {
	fmt.Printf("处理批次 %d...\n", batchNum)

	// 记录处理开始时间
	start := time.Now()

	// 模拟处理时间
	processingTime := time.Duration(50+batchNum*10) * time.Millisecond
	time.Sleep(processingTime)

	// 记录指标
	labels := map[string]string{
		"component": "data_processor",
		"batch_num": fmt.Sprintf("%d", batchNum),
	}

	// 记录处理时间
	mc.RecordLatency("batch_processor", processingTime, map[string]string{
		"batch_num": fmt.Sprintf("%d", batchNum),
	})

	// 记录吞吐量
	mc.RecordThroughput("records_processed", int64(batchSize), labels)

	// 记录延迟
	mc.RecordLatency("processing_latency", processingTime, labels)

	// 记录计数器
	mc.RecordCounter("batches_processed_total", 1, labels)

	// 记录仪表盘（当前活跃批次）
	mc.RecordGauge("active_batches", float64(batchNum%3+1), nil)

	fmt.Printf("批次 %d 处理完成，耗时: %v\n", batchNum, time.Since(start))
}

// exportMetrics 导出指标
func exportMetrics(mc metrics.MetricCollector) {
	// 导出JSON格式
	jsonData, err := mc.Export("json")
	if err != nil {
		log.Printf("导出JSON指标失败: %v", err)
	} else {
		fmt.Println("JSON格式指标:")
		fmt.Println(string(jsonData))
	}

	fmt.Println("\n" + strings.Repeat("=", 50))

	// 导出Prometheus格式
	prometheusData, err := mc.Export("prometheus")
	if err != nil {
		log.Printf("导出Prometheus指标失败: %v", err)
	} else {
		fmt.Println("Prometheus格式指标:")
		fmt.Println(string(prometheusData))
	}

	// 显示指标摘要
	displayMetricsSummary(mc)
}

// displayMetricsSummary 显示指标摘要
func displayMetricsSummary(mc metrics.MetricCollector) {
	fmt.Println("\n=== 指标摘要 ===")
	metricsList := mc.GetMetrics()
	fmt.Printf("总指标数: %d\n", len(metricsList))

	// 按类型分组统计
	typeCounts := make(map[metrics.MetricType]int)
	for _, metric := range metricsList {
		typeCounts[metric.GetType()]++
	}

	fmt.Println("指标类型分布:")
	for metricType, count := range typeCounts {
		fmt.Printf("  %s: %d\n", metricType, count)
	}

	// 显示一些关键指标
	fmt.Println("\n关键指标:")
	if metric := mc.GetMetric("batches_processed_total"); metric != nil {
		fmt.Printf("  处理批次总数: %v\n", metric.GetValue())
	}
	if metric := mc.GetMetric("records_processed_total"); metric != nil {
		fmt.Printf("  处理记录总数: %v\n", metric.GetValue())
	}
	if metric := mc.GetMetric("active_batches"); metric != nil {
		fmt.Printf("  当前活跃批次: %v\n", metric.GetValue())
	}
}

// waitForShutdown 等待关闭信号
func waitForShutdown(ctx context.Context, cancel context.CancelFunc) {
	fmt.Println("\n=== 系统运行中 ===")
	fmt.Println("按 Ctrl+C 退出...")
	fmt.Println("或访问 http://localhost:9090/metrics 查看实时指标")

	// 创建信号通道
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 等待信号
	select {
	case sig := <-sigChan:
		fmt.Printf("\n收到信号: %v，开始优雅关闭...\n", sig)
		cancel()
	case <-ctx.Done():
		fmt.Println("\n上下文已取消，开始关闭...")
	}

	// 给一些时间让goroutine清理
	time.Sleep(1 * time.Second)
	fmt.Println("系统已关闭")
}
