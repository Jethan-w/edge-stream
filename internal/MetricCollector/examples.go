package MetricCollector

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)

// ExampleBasicUsage 基本使用示例
func ExampleBasicUsage() {
	fmt.Println("=== MetricCollector 基本使用示例 ===")

	// 创建指标收集器
	collector := NewStandardMetricCollector()

	// 注册Gauge指标（瞬时值）
	gauge, err := collector.RegisterGauge("jvm.memory.used", func() float64 {
		// 模拟JVM内存使用量
		return 1024.0 + rand.Float64()*100
	})
	if err != nil {
		log.Fatalf("注册Gauge失败: %v", err)
	}

	// 注册Counter指标（累计值）
	counter, err := collector.RegisterCounter("processed.records")
	if err != nil {
		log.Fatalf("注册Counter失败: %v", err)
	}

	// 注册Histogram指标（数值分布）
	histogram, err := collector.RegisterHistogram("processing.time")
	if err != nil {
		log.Fatalf("注册Histogram失败: %v", err)
	}

	// 注册Timer指标（耗时统计）
	timer, err := collector.RegisterTimer("request.duration")
	if err != nil {
		log.Fatalf("注册Timer失败: %v", err)
	}

	// 注册Meter指标（速率统计）
	meter, err := collector.RegisterMeter("requests.per.second")
	if err != nil {
		log.Fatalf("注册Meter失败: %v", err)
	}

	// 模拟数据更新
	for i := 0; i < 5; i++ {
		// 更新Counter
		counter.Increment(10)

		// 更新Histogram
		processingTime := 50.0 + rand.Float64()*100
		histogram.Update(processingTime)

		// 使用Timer
		ctx := timer.Start()
		time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
		ctx.Stop()

		// 更新Meter
		meter.Mark(5)

		time.Sleep(100 * time.Millisecond)
	}

	// 获取指标快照
	snapshot := collector.GetMetricsSnapshot()
	fmt.Printf("指标快照时间: %s\n", snapshot.Timestamp.Format(time.RFC3339))
	fmt.Printf("Gauge指标数量: %d\n", len(snapshot.Gauges))
	fmt.Printf("Counter指标数量: %d\n", len(snapshot.Counters))
	fmt.Printf("Histogram指标数量: %d\n", len(snapshot.Histograms))
	fmt.Printf("Timer指标数量: %d\n", len(snapshot.Timers))
	fmt.Printf("Meter指标数量: %d\n", len(snapshot.Meters))

	// 获取特定指标
	if gauge, exists := snapshot.Gauges["jvm.memory.used"]; exists {
		fmt.Printf("JVM内存使用量: %.2f\n", gauge.GetValue())
	}

	if counter, exists := snapshot.Counters["processed.records"]; exists {
		fmt.Printf("已处理记录数: %d\n", counter.GetCount())
	}

	fmt.Println()
}

// ExampleMetricAggregation 指标聚合示例
func ExampleMetricAggregation() {
	fmt.Println("=== 指标聚合示例 ===")

	// 创建指标收集器
	collector := NewStandardMetricCollector()

	// 创建模拟的处理器注册表
	processorRegistry := &MockProcessorRegistry{
		processors: []*MockProcessor{
			{
				id:    "processor-1",
				name:  "GenerateFlowFile",
				type_: "org.apache.nifi.processors.standard.GenerateFlowFile",
				metrics: map[string]interface{}{
					"processingTime": 100.0,
					"count":          int64(1000),
					"errors":         int64(5),
				},
			},
			{
				id:    "processor-2",
				name:  "LogAttribute",
				type_: "org.apache.nifi.processors.standard.LogAttribute",
				metrics: map[string]interface{}{
					"processingTime": 50.0,
					"count":          int64(1000),
					"errors":         int64(2),
				},
			},
		},
	}

	// 创建模拟的流程组注册表
	processGroupRegistry := &MockProcessGroupRegistry{
		groups: []*MockProcessGroup{
			{
				name: "Data Ingestion",
				processors: []*MockProcessor{
					processorRegistry.processors[0],
				},
			},
			{
				name: "Data Processing",
				processors: []*MockProcessor{
					processorRegistry.processors[1],
				},
			},
		},
	}

	// 创建模拟的集群管理器
	clusterManager := &MockClusterManager{
		nodes: []*MockNiFiNode{
			{
				id:   "node-1",
				name: "NiFi Node 1",
			},
			{
				id:   "node-2",
				name: "NiFi Node 2",
			},
		},
	}

	// 创建指标聚合器
	aggregator := NewStandardMetricsAggregator(
		collector,
		processorRegistry,
		processGroupRegistry,
		clusterManager,
	)

	// 按处理器类型聚合
	processorTypeAggregations, err := aggregator.AggregateByProcessorType()
	if err != nil {
		log.Fatalf("按处理器类型聚合失败: %v", err)
	}

	fmt.Println("按处理器类型聚合结果:")
	for processorType, metrics := range processorTypeAggregations {
		fmt.Printf("  处理器类型: %s\n", processorType)
		fmt.Printf("    总处理时间: %.2f\n", metrics.TotalProcessingTime)
		fmt.Printf("    总处理记录数: %d\n", metrics.TotalProcessedRecords)
		fmt.Printf("    总错误数: %d\n", metrics.TotalErrorCount)
		fmt.Printf("    平均处理时间: %.2f\n", metrics.AverageProcessingTime)
		fmt.Printf("    吞吐量: %.2f\n", metrics.Throughput)
	}

	// 按流程组聚合
	processGroupAggregations, err := aggregator.AggregateByProcessGroup()
	if err != nil {
		log.Fatalf("按流程组聚合失败: %v", err)
	}

	fmt.Println("\n按流程组聚合结果:")
	for groupName, metrics := range processGroupAggregations {
		fmt.Printf("  流程组: %s\n", groupName)
		fmt.Printf("    总处理时间: %.2f\n", metrics.TotalProcessingTime)
		fmt.Printf("    总处理记录数: %d\n", metrics.TotalProcessedRecords)
		fmt.Printf("    总错误数: %d\n", metrics.TotalErrorCount)
	}

	// 集群聚合
	clusterSnapshot, err := aggregator.AggregateByCluster()
	if err != nil {
		log.Fatalf("集群聚合失败: %v", err)
	}

	fmt.Println("\n集群聚合结果:")
	fmt.Printf("  节点数量: %d\n", len(clusterSnapshot.NodeMetrics))
	for name, value := range clusterSnapshot.ClusterAverages {
		fmt.Printf("  %s: %v\n", name, value)
	}

	fmt.Println()
}

// ExampleMetricStorage 指标存储示例
func ExampleMetricStorage() {
	fmt.Println("=== 指标存储示例 ===")

	// 创建指标存储库
	repository := NewTimeSeriesMetricsRepository("./metrics-storage")

	// 创建指标存储管理器
	storageManager := NewMetricsStorageManager(repository)

	// 创建指标快照
	snapshot := &MetricRecorder.MetricsSnapshot{
		Timestamp: time.Now(),
		Gauges: map[string]float64{
			"jvm.memory.used": 1024.0,
			"jvm.cpu.usage":   0.5,
		},
		Counters: map[string]int64{
			"processed.records": 1000,
			"error.count":       5,
		},
		Histograms: map[string]MetricRecorder.HistogramData{
			"processing.time": {
				Count: 100,
				Min:   10.0,
				Max:   200.0,
				Mean:  50.0,
				Sum:   5000.0,
			},
		},
		Timers: map[string]MetricRecorder.TimerData{
			"request.duration": {
				Count: 50,
				Min:   10 * time.Millisecond,
				Max:   100 * time.Millisecond,
				Mean:  50 * time.Millisecond,
				Sum:   2500 * time.Millisecond,
			},
		},
		Meters: map[string]MetricRecorder.MeterData{
			"requests.per.second": {
				Count:             1000,
				MeanRate:          10.5,
				OneMinuteRate:     10.0,
				FiveMinuteRate:    9.5,
				FifteenMinuteRate: 9.0,
			},
		},
	}

	// 存储指标快照
	err := storageManager.StoreMetrics(snapshot)
	if err != nil {
		log.Fatalf("存储指标失败: %v", err)
	}
	fmt.Println("指标快照已存储")

	// 查询历史指标
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now()

	historicalMetrics, err := storageManager.QueryMetrics("jvm.memory.used", startTime, endTime)
	if err != nil {
		log.Fatalf("查询历史指标失败: %v", err)
	}

	fmt.Printf("查询到 %d 个历史指标快照\n", len(historicalMetrics))

	// 获取指标摘要
	summary, err := storageManager.GetMetricsSummary(startTime, endTime)
	if err != nil {
		log.Fatalf("获取指标摘要失败: %v", err)
	}

	fmt.Printf("指标摘要:\n")
	fmt.Printf("  总快照数: %d\n", summary.TotalSnapshots)
	fmt.Printf("  时间范围: %s 到 %s\n",
		summary.TimeRange.StartTime.Format(time.RFC3339),
		summary.TimeRange.EndTime.Format(time.RFC3339))
	fmt.Printf("  持续时间: %v\n", summary.TimeRange.EndTime.Sub(summary.TimeRange.StartTime))

	// 归档指标
	err = storageManager.ArchiveMetrics()
	if err != nil {
		log.Fatalf("归档指标失败: %v", err)
	}
	fmt.Println("指标已归档")

	fmt.Println()
}

// ExampleMetricExport 指标导出示例
func ExampleMetricExport() {
	fmt.Println("=== 指标导出示例 ===")

	// 创建指标导出器
	exporter := NewStandardMetricsExporter()

	// 创建指标快照
	snapshot := &MetricExporter.MetricsSnapshot{
		Timestamp: time.Now(),
		Gauges: map[string]float64{
			"jvm.memory.used": 1024.0,
			"jvm.cpu.usage":   0.5,
		},
		Counters: map[string]int64{
			"processed.records": 1000,
			"error.count":       5,
		},
		Histograms: map[string]MetricExporter.HistogramData{
			"processing.time": {
				Count: 100,
				Min:   10.0,
				Max:   200.0,
				Mean:  50.0,
				Sum:   5000.0,
			},
		},
		Timers: map[string]MetricExporter.TimerData{
			"request.duration": {
				Count: 50,
				Min:   10 * time.Millisecond,
				Max:   100 * time.Millisecond,
				Mean:  50 * time.Millisecond,
				Sum:   2500 * time.Millisecond,
			},
		},
		Meters: map[string]MetricExporter.MeterData{
			"requests.per.second": {
				Count:             1000,
				MeanRate:          10.5,
				OneMinuteRate:     10.0,
				FiveMinuteRate:    9.5,
				FifteenMinuteRate: 9.0,
			},
		},
	}

	// 导出到Prometheus格式
	prometheusData, err := exporter.ExportToPrometheus(snapshot)
	if err != nil {
		log.Fatalf("导出到Prometheus失败: %v", err)
	}
	fmt.Println("Prometheus格式导出:")
	fmt.Println(prometheusData[:200] + "...")

	// 导出到OpenTSDB格式
	opentsdbData, err := exporter.ExportToOpenTSDB(snapshot)
	if err != nil {
		log.Fatalf("导出到OpenTSDB失败: %v", err)
	}
	fmt.Printf("\nOpenTSDB格式导出 (前200字符):\n%s...\n", string(opentsdbData[:200]))

	// 导出到Graphite格式
	graphiteData, err := exporter.ExportToGraphite(snapshot)
	if err != nil {
		log.Fatalf("导出到Graphite失败: %v", err)
	}
	fmt.Printf("\nGraphite格式导出 (前200字符):\n%s...\n", string(graphiteData[:200]))

	// 导出到JMX格式
	jmxData, err := exporter.ExportToJMX(snapshot)
	if err != nil {
		log.Fatalf("导出到JMX失败: %v", err)
	}
	fmt.Printf("\nJMX格式导出 (前200字符):\n%s...\n", string(jmxData[:200]))

	// 导出到CSV格式
	csvData, err := exporter.ExportToCSV(snapshot)
	if err != nil {
		log.Fatalf("导出到CSV失败: %v", err)
	}
	fmt.Printf("\nCSV格式导出 (前200字符):\n%s...\n", string(csvData[:200]))

	// 启动HTTP端点
	err = exporter.StartHTTPEndpoint(8080)
	if err != nil {
		log.Fatalf("启动HTTP端点失败: %v", err)
	}
	fmt.Println("\nHTTP指标端点已启动在端口 8080")
	fmt.Println("访问 http://localhost:8080/metrics 获取Prometheus格式指标")
	fmt.Println("访问 http://localhost:8080/metrics/json 获取JSON格式指标")
	fmt.Println("访问 http://localhost:8080/health 获取健康状态")

	// 等待一段时间后停止
	time.Sleep(2 * time.Second)

	err = exporter.StopHTTPEndpoint()
	if err != nil {
		log.Printf("停止HTTP端点失败: %v", err)
	}

	fmt.Println()
}

// ExampleLineageTracking 数据溯源示例
func ExampleLineageTracking() {
	fmt.Println("=== 数据溯源示例 ===")

	// 创建溯源存储库
	repository := NewInMemoryProvenanceRepository()

	// 创建溯源管理器
	lineageManager := NewLineageManager(repository)

	// 创建溯源事件
	flowFileID := "flowfile-12345"

	// 创建事件
	createEvent := NewLineageEvent(
		flowFileID,
		LineageEventTypeCreate,
		"processor-1",
		"GenerateFlowFile",
	)
	createEvent.Size = 1024
	createEvent.LineageStart = true
	createEvent.Attributes["filename"] = "data.txt"
	createEvent.Attributes["size"] = "1024"

	// 接收事件
	receiveEvent := NewLineageEvent(
		flowFileID,
		LineageEventTypeReceive,
		"processor-2",
		"LogAttribute",
	)
	receiveEvent.Size = 1024
	receiveEvent.Details["source"] = "processor-1"

	// 修改事件
	modifyEvent := NewLineageEvent(
		flowFileID,
		LineageEventTypeModify,
		"processor-3",
		"UpdateAttribute",
	)
	modifyEvent.Size = 1024
	modifyEvent.Attributes["processed"] = "true"
	modifyEvent.Attributes["timestamp"] = time.Now().Format(time.RFC3339)

	// 发送事件
	sendEvent := NewLineageEvent(
		flowFileID,
		LineageEventTypeSend,
		"processor-4",
		"PutFile",
	)
	sendEvent.Size = 1024
	sendEvent.LineageEnd = true
	sendEvent.Details["destination"] = "/output/data.txt"

	// 添加事件到溯源系统
	events := []*LineageEvent{createEvent, receiveEvent, modifyEvent, sendEvent}
	for _, event := range events {
		err := lineageManager.AddEvent(event)
		if err != nil {
			log.Fatalf("添加溯源事件失败: %v", err)
		}
	}

	fmt.Printf("已添加 %d 个溯源事件\n", len(events))

	// 追踪FlowFile
	result, err := lineageManager.TraceFlowFile(flowFileID)
	if err != nil {
		log.Fatalf("追踪FlowFile失败: %v", err)
	}

	fmt.Printf("\nFlowFile追踪结果:\n")
	fmt.Printf("  FlowFile ID: %s\n", result.FlowFileID)
	fmt.Printf("  总事件数: %d\n", result.TotalEvents)
	fmt.Printf("  开始时间: %s\n", result.StartTime.Format(time.RFC3339))
	fmt.Printf("  结束时间: %s\n", result.EndTime.Format(time.RFC3339))
	fmt.Printf("  持续时间: %v\n", result.Duration)
	fmt.Printf("  总大小: %d\n", result.TotalSize)
	fmt.Printf("  最终大小: %d\n", result.FinalSize)

	fmt.Printf("\n处理路径:\n")
	for i, processor := range result.ProcessingPath {
		fmt.Printf("  %d. %s (%s)\n", i+1, processor.Name, processor.ID)
		fmt.Printf("     输入: %d, 输出: %d, 错误: %d\n",
			processor.InputCount, processor.OutputCount, processor.ErrorCount)
	}

	// 获取溯源摘要
	summary, err := lineageManager.GetSummary(flowFileID)
	if err != nil {
		log.Fatalf("获取溯源摘要失败: %v", err)
	}

	fmt.Printf("\n溯源摘要:\n")
	fmt.Printf("  总事件数: %d\n", summary.TotalEvents)
	fmt.Printf("  唯一处理器数: %d\n", summary.UniqueProcessors)
	fmt.Printf("  持续时间: %v\n", summary.Duration)

	fmt.Printf("\n事件类型统计:\n")
	for eventType, count := range summary.EventTypes {
		fmt.Printf("  %s: %d\n", eventType, count)
	}

	fmt.Printf("\n处理器统计:\n")
	for processorID, processor := range summary.Processors {
		fmt.Printf("  %s (%s):\n", processor.Name, processorID)
		fmt.Printf("    事件数: %d\n", processor.EventCount)
		fmt.Printf("    输入: %d, 输出: %d, 错误: %d\n",
			processor.InputCount, processor.OutputCount, processor.ErrorCount)
		fmt.Printf("    首次出现: %s\n", processor.FirstSeen.Format(time.RFC3339))
		fmt.Printf("    最后出现: %s\n", processor.LastSeen.Format(time.RFC3339))
	}

	// 清理过期事件
	err = lineageManager.PurgeExpiredEvents(24 * time.Hour)
	if err != nil {
		log.Fatalf("清理过期事件失败: %v", err)
	}

	fmt.Println()
}

// ExampleCustomMetricProvider 自定义指标提供者示例
func ExampleCustomMetricProvider() {
	fmt.Println("=== 自定义指标提供者示例 ===")

	// 创建指标收集器
	collector := NewStandardMetricCollector()

	// 创建自定义指标提供者
	provider := &DatabaseConnectionPoolMetricProvider{
		activeConnections:  10,
		waitingConnections: 2,
		totalConnections:   20,
		idleConnections:    8,
	}

	// 注册自定义指标
	provider.RegisterMetrics(collector)

	// 模拟数据库连接池状态变化
	for i := 0; i < 3; i++ {
		// 模拟连接使用
		provider.activeConnections = 10 + i*2
		provider.waitingConnections = 2 + i
		provider.idleConnections = 8 - i

		time.Sleep(100 * time.Millisecond)

		// 获取指标快照
		snapshot := collector.GetMetricsSnapshot()

		fmt.Printf("数据库连接池状态 %d:\n", i+1)
		if gauge, exists := snapshot.Gauges["database.connection.active"]; exists {
			fmt.Printf("  活跃连接数: %.0f\n", gauge.GetValue())
		}
		if gauge, exists := snapshot.Gauges["database.connection.waiting"]; exists {
			fmt.Printf("  等待连接数: %.0f\n", gauge.GetValue())
		}
		if gauge, exists := snapshot.Gauges["database.connection.idle"]; exists {
			fmt.Printf("  空闲连接数: %.0f\n", gauge.GetValue())
		}
		if gauge, exists := snapshot.Gauges["database.connection.total"]; exists {
			fmt.Printf("  总连接数: %.0f\n", gauge.GetValue())
		}
	}

	fmt.Println()
}

// DatabaseConnectionPoolMetricProvider 数据库连接池指标提供者
type DatabaseConnectionPoolMetricProvider struct {
	activeConnections  int
	waitingConnections int
	totalConnections   int
	idleConnections    int
}

// RegisterMetrics 注册指标
func (dcpmp *DatabaseConnectionPoolMetricProvider) RegisterMetrics(collector MetricCollector) {
	// 注册活跃连接数指标
	collector.RegisterGauge("database.connection.active", func() float64 {
		return float64(dcpmp.activeConnections)
	})

	// 注册等待连接数指标
	collector.RegisterGauge("database.connection.waiting", func() float64 {
		return float64(dcpmp.waitingConnections)
	})

	// 注册空闲连接数指标
	collector.RegisterGauge("database.connection.idle", func() float64 {
		return float64(dcpmp.idleConnections)
	})

	// 注册总连接数指标
	collector.RegisterGauge("database.connection.total", func() float64 {
		return float64(dcpmp.totalConnections)
	})
}

// GetMetricDescriptions 获取指标描述
func (dcpmp *DatabaseConnectionPoolMetricProvider) GetMetricDescriptions() map[string]string {
	return map[string]string{
		"database.connection.active":  "当前活跃的数据库连接数",
		"database.connection.waiting": "等待获取连接的线程数",
		"database.connection.idle":    "当前空闲的数据库连接数",
		"database.connection.total":   "数据库连接池总连接数",
	}
}

// MockProcessorRegistry 模拟处理器注册表
type MockProcessorRegistry struct {
	processors []*MockProcessor
}

// GetAllProcessors 获取所有处理器
func (mpr *MockProcessorRegistry) GetAllProcessors() ([]Processor, error) {
	var processors []Processor
	for _, p := range mpr.processors {
		processors = append(processors, p)
	}
	return processors, nil
}

// MockProcessor 模拟处理器
type MockProcessor struct {
	id      string
	name    string
	type_   string
	metrics map[string]interface{}
}

// GetType 获取处理器类型
func (mp *MockProcessor) GetType() string {
	return mp.type_
}

// GetMetrics 获取处理器指标
func (mp *MockProcessor) GetMetrics() map[string]interface{} {
	return mp.metrics
}

// MockProcessGroupRegistry 模拟流程组注册表
type MockProcessGroupRegistry struct {
	groups []*MockProcessGroup
}

// GetAllGroups 获取所有流程组
func (mpgr *MockProcessGroupRegistry) GetAllGroups() ([]ProcessGroup, error) {
	var groups []ProcessGroup
	for _, g := range mpgr.groups {
		groups = append(groups, g)
	}
	return groups, nil
}

// MockProcessGroup 模拟流程组
type MockProcessGroup struct {
	name       string
	processors []*MockProcessor
}

// GetName 获取流程组名称
func (mpg *MockProcessGroup) GetName() string {
	return mpg.name
}

// GetProcessors 获取流程组中的处理器
func (mpg *MockProcessGroup) GetProcessors() ([]Processor, error) {
	var processors []Processor
	for _, p := range mpg.processors {
		processors = append(processors, p)
	}
	return processors, nil
}

// MockClusterManager 模拟集群管理器
type MockClusterManager struct {
	nodes []*MockNiFiNode
}

// GetAllNodes 获取所有节点
func (mcm *MockClusterManager) GetAllNodes() ([]NiFiNode, error) {
	var nodes []NiFiNode
	for _, n := range mcm.nodes {
		nodes = append(nodes, n)
	}
	return nodes, nil
}

// MockNiFiNode 模拟NiFi节点
type MockNiFiNode struct {
	id   string
	name string
}

// GetID 获取节点ID
func (mnfn *MockNiFiNode) GetID() string {
	return mnfn.id
}

// GetName 获取节点名称
func (mnfn *MockNiFiNode) GetName() string {
	return mnfn.name
}

// RunAllExamples 运行所有示例
func RunAllExamples() {
	fmt.Println("开始运行 MetricCollector 模块示例...\n")

	ExampleBasicUsage()
	ExampleMetricAggregation()
	ExampleMetricStorage()
	ExampleMetricExport()
	ExampleLineageTracking()
	ExampleCustomMetricProvider()

	fmt.Println("所有示例运行完成！")
}
