package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/crazy/edge-stream/internal/config"
	"github.com/crazy/edge-stream/internal/flowfile"
	"github.com/crazy/edge-stream/internal/metrics"
	"github.com/crazy/edge-stream/internal/processor"
)

// EdgeStreamEngine 边缘流处理引擎
type EdgeStreamEngine struct {
	configManager   config.ConfigManager
	metricCollector metrics.MetricCollector
	processors      []processor.Processor
	running         bool
}

// NewEdgeStreamEngine 创建边缘流处理引擎
func NewEdgeStreamEngine(configFile string, encryptionKey string) (*EdgeStreamEngine, error) {
	// 创建配置管理器
	configManager := config.NewStandardConfigManager(encryptionKey)
	if err := configManager.LoadConfig(configFile); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// 验证配置
	if err := configManager.Validate(); err != nil {
		log.Printf("Config validation warning: %v", err)
	}

	// 创建指标收集器
	metricCollector := metrics.NewStandardMetricCollector()

	// 创建处理器
	processors := []processor.Processor{
		processor.NewSimpleProcessor("data_validator"),
		processor.NewTransformProcessor("data_transformer", transformData),
		processor.NewSimpleProcessor("data_enricher"),
	}

	return &EdgeStreamEngine{
		configManager:   configManager,
		metricCollector: metricCollector,
		processors:      processors,
		running:         false,
	}, nil
}

// Start 启动引擎
func (e *EdgeStreamEngine) Start(ctx context.Context) error {
	e.running = true
	fmt.Println("Edge Stream 引擎启动中...")

	// 记录启动指标
	e.metricCollector.RecordCounter("engine_starts_total", 1, map[string]string{
		"version": "1.0.0",
	})

	// 启动配置监听
	go e.watchConfig(ctx)

	// 启动指标收集
	go e.collectSystemMetrics(ctx)

	// 模拟数据处理
	go e.processData(ctx)

	fmt.Println("Edge Stream 引擎已启动")
	return nil
}

// Stop 停止引擎
func (e *EdgeStreamEngine) Stop() error {
	e.running = false
	fmt.Println("Edge Stream 引擎已停止")
	return nil
}

// watchConfig 监听配置变更
func (e *EdgeStreamEngine) watchConfig(ctx context.Context) {
	e.configManager.Watch(ctx, func(key string, oldValue, newValue interface{}) {
		fmt.Printf("配置变更: %s = %v -> %v\n", key, oldValue, newValue)
		e.metricCollector.RecordCounter("config_changes_total", 1, map[string]string{
			"key": key,
		})
	})
}

// collectSystemMetrics 收集系统指标
func (e *EdgeStreamEngine) collectSystemMetrics(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !e.running {
				return
			}

			// 模拟系统指标
			e.metricCollector.RecordMemoryUsage("engine", int64(rand.Intn(100)*1024*1024+50*1024*1024))
			e.metricCollector.RecordQueueDepth("processing_queue", int64(rand.Intn(1000)+100))
			e.metricCollector.RecordConnectionCount("database", int64(rand.Intn(20)+5))

			// 记录引擎状态
			e.metricCollector.RecordGauge("engine_running", 1, map[string]string{
				"status": "active",
			})
		}
	}
}

// processData 处理数据
func (e *EdgeStreamEngine) processData(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !e.running {
				return
			}

			// 创建FlowFile
			flowFile := flowfile.NewFlowFile()
			flowFile.Content = []byte(fmt.Sprintf("test data %d", time.Now().Unix()))
			flowFile.Size = int64(len(flowFile.Content))

			// 处理数据
			start := time.Now()
			processedFlowFile, err := e.processFlowFile(flowFile)
			processingTime := time.Since(start)

			if err != nil {
				e.metricCollector.RecordError("data_processing", "processing_error", map[string]string{
					"component": "engine",
				})
				fmt.Printf("数据处理失败: %v\n", err)
			} else {
				// 记录成功指标
				e.metricCollector.RecordLatency("data_processing", processingTime, map[string]string{
					"component": "engine",
					"status":    "success",
				})
				e.metricCollector.RecordThroughput("data_processing", 1, map[string]string{
					"component": "engine",
				})

				fmt.Printf("数据处理成功: %s -> %s (耗时: %v)\n",
					flowFile.UUID, processedFlowFile.UUID, processingTime)
			}
		}
	}
}

// processFlowFile 处理FlowFile
func (e *EdgeStreamEngine) processFlowFile(flowFile *flowfile.FlowFile) (*flowfile.FlowFile, error) {
	currentFlowFile := flowFile

	for i, proc := range e.processors {
		start := time.Now()
		processedFlowFile, err := proc.Process(currentFlowFile)
		processingTime := time.Since(start)

		if err != nil {
			e.metricCollector.RecordError("processor", "processing_error", map[string]string{
				"processor": proc.GetName(),
				"step":      fmt.Sprintf("%d", i),
			})
			return nil, fmt.Errorf("processor %s failed: %w", proc.GetName(), err)
		}

		// 记录处理器指标
		e.metricCollector.RecordLatency("processor", processingTime, map[string]string{
			"processor": proc.GetName(),
			"step":      fmt.Sprintf("%d", i),
		})

		currentFlowFile = processedFlowFile
	}

	return currentFlowFile, nil
}

// GetMetrics 获取指标
func (e *EdgeStreamEngine) GetMetrics() []metrics.Metric {
	return e.metricCollector.GetMetrics()
}

// GetConfig 获取配置
func (e *EdgeStreamEngine) GetConfig(key string) string {
	return e.configManager.GetString(key)
}

// transformData 数据转换函数
func transformData(flowFile *flowfile.FlowFile) (*flowfile.FlowFile, error) {
	// 模拟数据转换
	time.Sleep(time.Duration(rand.Intn(50)+10) * time.Millisecond)

	// 添加转换标记
	flowFile.Attributes["transformed"] = "true"
	flowFile.Attributes["transform_time"] = time.Now().Format(time.RFC3339)

	return flowFile, nil
}

func main() {
	fmt.Println("=== Edge Stream 集成示例 ===")

	// 1. 创建引擎
	encryptionKey := "my-secret-encryption-key-32-bytes!"
	engine, err := NewEdgeStreamEngine("config.yaml", encryptionKey)
	if err != nil {
		log.Fatalf("创建引擎失败: %v", err)
	}

	// 2. 启动引擎
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := engine.Start(ctx); err != nil {
		log.Fatalf("启动引擎失败: %v", err)
	}

	// 3. 运行一段时间
	fmt.Println("\n引擎运行中，观察指标变化...")
	time.Sleep(15 * time.Second)

	// 4. 显示配置信息
	fmt.Println("\n=== 当前配置信息 ===")
	fmt.Printf("MySQL主机: %s\n", engine.GetConfig("database.mysql.host"))
	fmt.Printf("MySQL端口: %s\n", engine.GetConfig("database.mysql.port"))
	fmt.Printf("Redis主机: %s\n", engine.GetConfig("redis.host"))
	fmt.Printf("服务器端口: %s\n", engine.GetConfig("server.port"))
	fmt.Printf("日志级别: %s\n", engine.GetConfig("logging.level"))

	// 5. 显示指标统计
	fmt.Println("\n=== 指标统计 ===")
	allMetrics := engine.GetMetrics()
	fmt.Printf("总指标数: %d\n", len(allMetrics))

	// 按类型统计指标
	metricTypes := make(map[metrics.MetricType]int)
	for _, metric := range allMetrics {
		metricTypes[metric.GetType()]++
	}

	for metricType, count := range metricTypes {
		fmt.Printf("%s 类型指标: %d 个\n", metricType, count)
	}

	// 6. 显示关键指标
	fmt.Println("\n=== 关键指标 ===")
	for _, metric := range allMetrics {
		name := metric.GetName()
		if contains(name, []string{"engine_starts", "data_processing", "memory_usage", "errors_total"}) {
			fmt.Printf("%s: %v\n", name, metric.GetValue())
		}
	}

	// 7. 导出指标
	fmt.Println("\n=== 导出Prometheus指标 ===")
	if prometheusData, err := engine.metricCollector.Export("prometheus"); err == nil {
		fmt.Printf("Prometheus格式指标 (前500字符):\n%s...\n",
			truncateString(string(prometheusData), 500))
	}

	// 8. 停止引擎
	if err := engine.Stop(); err != nil {
		log.Printf("停止引擎失败: %v", err)
	}

	fmt.Println("\n=== 集成示例完成 ===")
}

// contains 检查字符串是否包含任一关键词
func contains(s string, keywords []string) bool {
	for _, keyword := range keywords {
		for i := 0; i <= len(s)-len(keyword); i++ {
			if i+len(keyword) <= len(s) && s[i:i+len(keyword)] == keyword {
				return true
			}
		}
	}
	return false
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func init() {
	// Go 1.20+ 会自动初始化随机数生成器，无需手动设置种子
}
