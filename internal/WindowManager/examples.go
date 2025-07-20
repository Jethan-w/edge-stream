package windowmanager

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/edge-stream/internal/WindowManager/AggregationCalculator"
	"github.com/edge-stream/internal/WindowManager/TimestampExtractor"
	"github.com/edge-stream/internal/WindowManager/WindowDefiner"
	"github.com/edge-stream/internal/WindowManager/WindowStateManager"
	"github.com/edge-stream/internal/WindowManager/WindowTrigger"
)

// TemperatureWindowAnalyzer 温度传感器数据窗口分析器
type TemperatureWindowAnalyzer struct {
	windowManager       *StandardWindowManager
	timestampExtractor  TimestampExtractor.TimestampExtractor
	aggregationStrategy *AggregationCalculator.MultiAggregationStrategy
	outputProcessor     *WindowDefiner.WindowOutputProcessor
	trigger             *WindowTrigger.StandardWindowTrigger
}

// NewTemperatureWindowAnalyzer 创建温度窗口分析器
func NewTemperatureWindowAnalyzer() *TemperatureWindowAnalyzer {
	// 创建窗口管理器
	windowManager := NewStandardWindowManager()

	// 创建时间戳提取器
	timestampExtractor := TimestampExtractor.NewJsonTimestampExtractor("timestamp", "2006-01-02T15:04:05Z")

	// 创建聚合策略
	aggregationStrategy := AggregationCalculator.NewMultiAggregationStrategy()
	aggregationStrategy.AddAggregator(AggregationCalculator.NewAverageAggregator("temperature"))
	aggregationStrategy.AddAggregator(AggregationCalculator.NewMaxAggregator("temperature"))
	aggregationStrategy.AddAggregator(AggregationCalculator.NewMinAggregator("temperature"))
	aggregationStrategy.AddAggregator(AggregationCalculator.NewCountAggregator())

	// 创建输出处理器
	outputProcessor := WindowDefiner.NewWindowOutputProcessor()

	// 创建触发器
	trigger := WindowTrigger.NewStandardWindowTrigger()
	trigger.SetOutputProcessor(outputProcessor)
	trigger.SetAggregator(aggregationStrategy)

	return &TemperatureWindowAnalyzer{
		windowManager:       windowManager,
		timestampExtractor:  timestampExtractor,
		aggregationStrategy: aggregationStrategy,
		outputProcessor:     outputProcessor,
		trigger:             trigger,
	}
}

// AnalyzeDeviceTemperature 分析设备温度数据
func (twa *TemperatureWindowAnalyzer) AnalyzeDeviceTemperature() error {
	// 创建5分钟滚动窗口
	startTime := time.Now().UnixMilli()
	endTime := startTime + 5*60*1000 // 5分钟后

	window, err := twa.windowManager.CreateTimeWindow(startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to create time window: %w", err)
	}

	// 注册窗口到触发器
	if err := twa.trigger.RegisterWindow(window); err != nil {
		return fmt.Errorf("failed to register window: %w", err)
	}

	// 启动触发器
	if err := twa.trigger.Start(); err != nil {
		return fmt.Errorf("failed to start trigger: %w", err)
	}

	// 模拟处理 FlowFile
	for i := 0; i < 10; i++ {
		// 创建模拟的温度数据
		temperatureData := map[string]interface{}{
			"device_id":   "sensor_001",
			"temperature": 20.0 + float64(i),
			"timestamp":   time.Now().Format("2006-01-02T15:04:05Z"),
		}

		jsonData, _ := json.Marshal(temperatureData)

		flowFile := NewFlowFile(
			fmt.Sprintf("temp_%d", i),
			jsonData,
			map[string]string{
				"device_id": "sensor_001",
				"type":      "temperature",
			},
		)

		// 提取事件时间戳
		eventTime := twa.timestampExtractor.ExtractTimestamp(flowFile)

		// 检查时间是否在窗口内
		if window.IsTimeInWindow(eventTime) {
			if err := window.AddFlowFile(flowFile); err != nil {
				fmt.Printf("Warning: failed to add flow file to window: %v\n", err)
			}
		}

		time.Sleep(100 * time.Millisecond) // 模拟数据间隔
	}

	// 等待窗口处理完成
	time.Sleep(6 * time.Second)

	// 停止触发器
	if err := twa.trigger.Stop(); err != nil {
		return fmt.Errorf("failed to stop trigger: %w", err)
	}

	return nil
}

// CountWindowExample 计数窗口示例
func CountWindowExample() error {
	// 创建窗口管理器
	windowManager := NewStandardWindowManager()

	// 创建计数窗口（最多100条记录）
	window, err := windowManager.CreateCountWindow(100)
	if err != nil {
		return fmt.Errorf("failed to create count window: %w", err)
	}

	// 创建聚合策略
	aggregationStrategy := AggregationCalculator.NewMultiAggregationStrategy()
	aggregationStrategy.AddAggregator(AggregationCalculator.NewSumAggregator("value"))
	aggregationStrategy.AddAggregator(AggregationCalculator.NewAverageAggregator("value"))

	// 模拟添加数据
	for i := 0; i < 50; i++ {
		data := map[string]interface{}{
			"value":     float64(i),
			"timestamp": time.Now().UnixMilli(),
		}

		jsonData, _ := json.Marshal(data)
		flowFile := NewFlowFile(fmt.Sprintf("count_%d", i), jsonData, nil)

		if err := window.AddFlowFile(flowFile); err != nil {
			fmt.Printf("Warning: failed to add flow file: %v\n", err)
		}
	}

	// 计算聚合结果
	flowFiles := window.GetFlowFiles()
	results, err := aggregationStrategy.ComputeAggregations(flowFiles)
	if err != nil {
		return fmt.Errorf("failed to compute aggregations: %w", err)
	}

	fmt.Printf("Count window aggregation results: %+v\n", results)

	return nil
}

// SessionWindowExample 会话窗口示例
func SessionWindowExample() error {
	// 创建窗口管理器
	windowManager := NewStandardWindowManager()

	// 创建会话窗口（5分钟超时）
	window, err := windowManager.CreateSessionWindow(5 * time.Minute)
	if err != nil {
		return fmt.Errorf("failed to create session window: %w", err)
	}

	// 创建聚合策略
	aggregationStrategy := AggregationCalculator.NewMultiAggregationStrategy()
	aggregationStrategy.AddAggregator(AggregationCalculator.NewCountAggregator())
	aggregationStrategy.AddAggregator(AggregationCalculator.NewFirstAggregator("user_id"))
	aggregationStrategy.AddAggregator(AggregationCalculator.NewLastAggregator("user_id"))

	// 模拟用户会话数据
	sessionData := []map[string]interface{}{
		{"user_id": "user_001", "action": "login", "timestamp": time.Now().UnixMilli()},
		{"user_id": "user_001", "action": "browse", "timestamp": time.Now().UnixMilli()},
		{"user_id": "user_001", "action": "purchase", "timestamp": time.Now().UnixMilli()},
		{"user_id": "user_001", "action": "logout", "timestamp": time.Now().UnixMilli()},
	}

	// 添加会话数据
	for i, data := range sessionData {
		jsonData, _ := json.Marshal(data)
		flowFile := NewFlowFile(fmt.Sprintf("session_%d", i), jsonData, nil)

		if err := window.AddFlowFile(flowFile); err != nil {
			fmt.Printf("Warning: failed to add flow file: %v\n", err)
		}

		time.Sleep(1 * time.Second) // 模拟用户操作间隔
	}

	// 计算聚合结果
	flowFiles := window.GetFlowFiles()
	results, err := aggregationStrategy.ComputeAggregations(flowFiles)
	if err != nil {
		return fmt.Errorf("failed to compute aggregations: %w", err)
	}

	fmt.Printf("Session window aggregation results: %+v\n", results)

	return nil
}

// SlidingWindowExample 滑动窗口示例
func SlidingWindowExample() error {
	// 创建窗口工厂
	factory := WindowDefiner.NewWindowFactory()

	// 创建滑动窗口配置
	config := WindowDefiner.WindowConfig{
		WindowSize:    300000, // 5分钟窗口
		SlideInterval: 60000,  // 1分钟滑动间隔
	}

	// 创建状态管理器
	stateManager, err := WindowStateManager.NewMemoryStateManager()
	if err != nil {
		return fmt.Errorf("failed to create state manager: %w", err)
	}

	// 创建滑动窗口
	window, err := factory.CreateWindow(WindowTypeSliding, config, stateManager)
	if err != nil {
		return fmt.Errorf("failed to create sliding window: %w", err)
	}

	// 创建聚合策略
	aggregationStrategy := AggregationCalculator.NewMultiAggregationStrategy()
	aggregationStrategy.AddAggregator(AggregationCalculator.NewAverageAggregator("value"))
	aggregationStrategy.AddAggregator(AggregationCalculator.NewMaxAggregator("value"))

	// 模拟添加数据
	for i := 0; i < 20; i++ {
		data := map[string]interface{}{
			"value":     float64(i * 10),
			"timestamp": time.Now().UnixMilli(),
		}

		jsonData, _ := json.Marshal(data)
		flowFile := NewFlowFile(fmt.Sprintf("slide_%d", i), jsonData, nil)

		if err := window.AddFlowFile(flowFile); err != nil {
			fmt.Printf("Warning: failed to add flow file: %w\n", err)
		}

		time.Sleep(30 * time.Second) // 30秒间隔
	}

	// 计算聚合结果
	flowFiles := window.GetFlowFiles()
	results, err := aggregationStrategy.ComputeAggregations(flowFiles)
	if err != nil {
		return fmt.Errorf("failed to compute aggregations: %w", err)
	}

	fmt.Printf("Sliding window aggregation results: %+v\n", results)

	return nil
}

// LateDataHandlingExample 延迟数据处理示例
func LateDataHandlingExample() error {
	// 创建时间戳提取器
	timestampExtractor := TimestampExtractor.NewJsonTimestampExtractor("timestamp", "2006-01-02T15:04:05Z")

	// 创建延迟数据处理策略
	lateDataStrategy := TimestampExtractor.NewLateDataStrategy(
		300000, // 5分钟最大延迟
		TimestampExtractor.LateDataHandlingSideOutput,
	)

	// 创建水印管理器
	watermarkManager := WindowTrigger.NewSimpleWatermarkManager()

	// 创建水印触发策略
	triggerStrategy := WindowTrigger.NewWatermarkTriggerStrategy(watermarkManager, 300000)

	// 创建窗口管理器
	windowManager := NewStandardWindowManager()

	// 创建时间窗口
	startTime := time.Now().UnixMilli()
	endTime := startTime + 10*60*1000 // 10分钟窗口
	window, err := windowManager.CreateTimeWindow(startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to create time window: %w", err)
	}

	// 模拟处理乱序数据
	dataPoints := []struct {
		value     float64
		timestamp int64
	}{
		{20.0, time.Now().UnixMilli()},                       // 正常数据
		{25.0, time.Now().Add(-2 * time.Minute).UnixMilli()}, // 延迟数据
		{30.0, time.Now().Add(-8 * time.Minute).UnixMilli()}, // 严重延迟数据
	}

	for i, dataPoint := range dataPoints {
		// 创建数据
		data := map[string]interface{}{
			"value":     dataPoint.value,
			"timestamp": time.Unix(dataPoint.timestamp/1000, 0).Format("2006-01-02T15:04:05Z"),
		}

		jsonData, _ := json.Marshal(data)
		flowFile := NewFlowFile(fmt.Sprintf("late_%d", i), jsonData, nil)

		// 提取事件时间戳
		eventTime := timestampExtractor.ExtractTimestamp(flowFile)

		// 更新水印
		watermarkManager.UpdateWatermark(eventTime)

		// 处理延迟数据
		currentWatermark := watermarkManager.GetCurrentWatermark()
		if lateDataStrategy.HandleLateData(flowFile, eventTime, currentWatermark) {
			// 正常处理
			if err := window.AddFlowFile(flowFile); err != nil {
				fmt.Printf("Warning: failed to add flow file: %v\n", err)
			}
		} else {
			fmt.Printf("Late data handled: value=%.1f, eventTime=%d, watermark=%d\n",
				dataPoint.value, eventTime, currentWatermark)
		}
	}

	// 检查侧边输出
	sideOutputQueue := lateDataStrategy.GetSideOutputQueue()
	select {
	case lateFlowFile := <-sideOutputQueue:
		fmt.Printf("Late data in side output: %s\n", lateFlowFile.ID)
	default:
		fmt.Println("No late data in side output")
	}

	return nil
}

// CustomAggregatorExample 自定义聚合器示例
func CustomAggregatorExample() error {
	// 创建自定义聚合器
	customAggregator := &CustomTemperatureAggregator{
		field: "temperature",
	}

	// 创建聚合策略
	aggregationStrategy := AggregationCalculator.NewMultiAggregationStrategy()
	aggregationStrategy.AddAggregator(customAggregator)

	// 创建窗口
	windowManager := NewStandardWindowManager()
	window, err := windowManager.CreateTimeWindow(
		time.Now().UnixMilli(),
		time.Now().Add(5*time.Minute).UnixMilli(),
	)
	if err != nil {
		return fmt.Errorf("failed to create window: %w", err)
	}

	// 添加测试数据
	for i := 0; i < 10; i++ {
		data := map[string]interface{}{
			"temperature": 20.0 + float64(i),
			"humidity":    50.0 + float64(i),
		}

		jsonData, _ := json.Marshal(data)
		flowFile := NewFlowFile(fmt.Sprintf("custom_%d", i), jsonData, nil)

		if err := window.AddFlowFile(flowFile); err != nil {
			fmt.Printf("Warning: failed to add flow file: %v\n", err)
		}
	}

	// 计算聚合结果
	flowFiles := window.GetFlowFiles()
	results, err := aggregationStrategy.ComputeAggregations(flowFiles)
	if err != nil {
		return fmt.Errorf("failed to compute aggregations: %w", err)
	}

	fmt.Printf("Custom aggregator results: %+v\n", results)

	return nil
}

// CustomTemperatureAggregator 自定义温度聚合器
type CustomTemperatureAggregator struct {
	field string
	sum   float64
	count int
	mu    sync.RWMutex
}

// Aggregate 聚合单个 FlowFile
func (cta *CustomTemperatureAggregator) Aggregate(flowFile *FlowFile) error {
	cta.mu.Lock()
	defer cta.mu.Unlock()

	content := flowFile.GetContent()

	var jsonData map[string]interface{}
	if err := json.Unmarshal(content, &jsonData); err != nil {
		return fmt.Errorf("failed to parse JSON content")
	}

	value, exists := jsonData[cta.field]
	if !exists {
		return fmt.Errorf("field %s not found", cta.field)
	}

	switch v := value.(type) {
	case float64:
		cta.sum += v
		cta.count++
	case int:
		cta.sum += float64(v)
		cta.count++
	default:
		return fmt.Errorf("unsupported value type: %T", value)
	}

	return nil
}

// GetResult 获取聚合结果
func (cta *CustomTemperatureAggregator) GetResult() interface{} {
	cta.mu.RLock()
	defer cta.mu.RUnlock()

	if cta.count > 0 {
		return map[string]interface{}{
			"average": cta.sum / float64(cta.count),
			"total":   cta.sum,
			"count":   cta.count,
		}
	}

	return map[string]interface{}{
		"average": 0.0,
		"total":   0.0,
		"count":   0,
	}
}

// Reset 重置聚合状态
func (cta *CustomTemperatureAggregator) Reset() {
	cta.mu.Lock()
	defer cta.mu.Unlock()

	cta.sum = 0
	cta.count = 0
}

// GetAggregationFunction 获取聚合函数类型
func (cta *CustomTemperatureAggregator) GetAggregationFunction() AggregationFunction {
	return AggregationFunctionAvg
}

// RunAllExamples 运行所有示例
func RunAllExamples() {
	fmt.Println("=== WindowManager Examples ===")

	// 温度窗口分析示例
	fmt.Println("\n1. Temperature Window Analysis:")
	if err := func() error {
		analyzer := NewTemperatureWindowAnalyzer()
		return analyzer.AnalyzeDeviceTemperature()
	}(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// 计数窗口示例
	fmt.Println("\n2. Count Window Example:")
	if err := CountWindowExample(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// 会话窗口示例
	fmt.Println("\n3. Session Window Example:")
	if err := SessionWindowExample(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// 滑动窗口示例
	fmt.Println("\n4. Sliding Window Example:")
	if err := SlidingWindowExample(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// 延迟数据处理示例
	fmt.Println("\n5. Late Data Handling Example:")
	if err := LateDataHandlingExample(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// 自定义聚合器示例
	fmt.Println("\n6. Custom Aggregator Example:")
	if err := CustomAggregatorExample(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	fmt.Println("\n=== All Examples Completed ===")
}
