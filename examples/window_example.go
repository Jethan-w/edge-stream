package main

//
// import (
// 	"context"
// 	"fmt"
// 	"log"
// 	"time"
//
// 	"github.com/crazy/edge-stream/internal/WindowManager"
// )
//
// // 窗口管理示例
// func main() {
// 	fmt.Println("=== EdgeStreamPro 窗口管理示例 ===")
//
// 	// 1. 创建窗口管理器
// 	windowManager := createWindowManager()
//
// 	// 2. 定义窗口
// 	defineWindows(windowManager)
//
// 	// 3. 创建时间戳提取器
// 	timestampExtractor := createTimestampExtractor()
//
// 	// 4. 创建聚合计算器
// 	aggregationCalculator := createAggregationCalculator()
//
// 	// 5. 创建窗口触发器
// 	windowTrigger := createWindowTrigger()
//
// 	// 6. 创建窗口状态管理器
// 	windowStateManager := createWindowStateManager()
//
// 	// 7. 模拟数据流和处理
// 	simulateDataProcessing(windowManager, timestampExtractor, aggregationCalculator, windowTrigger, windowStateManager)
//
// 	fmt.Println("=== 窗口管理示例完成 ===")
// }
//
// // 创建窗口管理器
// func createWindowManager() *windowmanager.WindowManager {
// 	fmt.Println("1. 创建窗口管理器...")
//
// 	// 创建窗口管理器
// 	manager := WindowManager.NewWindowManager()
//
// 	// 设置配置
// 	manager.SetMaxWindowSize(10000)             // 最大窗口大小
// 	manager.SetWindowTimeout(5 * time.Minute)   // 窗口超时时间
// 	manager.SetCleanupInterval(1 * time.Minute) // 清理间隔
//
// 	return manager
// }
//
// // 定义窗口
// func defineWindows(manager *WindowManager.WindowManager) {
// 	fmt.Println("2. 定义窗口...")
//
// 	// 定义时间窗口 - 5分钟
// 	timeWindow := WindowManager.NewTimeWindow("time-window-5min", 5*time.Minute)
// 	timeWindow.SetSlidingInterval(1 * time.Minute) // 滑动间隔1分钟
// 	manager.RegisterWindow(timeWindow)
//
// 	// 定义计数窗口 - 1000条记录
// 	countWindow := WindowManager.NewCountWindow("count-window-1000", 1000)
// 	countWindow.SetSlidingInterval(100) // 滑动间隔100条记录
// 	manager.RegisterWindow(countWindow)
//
// 	// 定义会话窗口 - 30分钟无活动超时
// 	sessionWindow := WindowManager.NewSessionWindow("session-window-30min", 30*time.Minute)
// 	manager.RegisterWindow(sessionWindow)
//
// 	// 定义自定义窗口
// 	customWindow := WindowManager.NewCustomWindow("custom-window", func(data interface{}) bool {
// 		// 自定义窗口条件：数据大小超过1KB
// 		if record, ok := data.(map[string]interface{}); ok {
// 			if size, exists := record["size"]; exists {
// 				if sizeInt, ok := size.(int); ok {
// 					return sizeInt > 1024
// 				}
// 			}
// 		}
// 		return false
// 	})
// 	manager.RegisterWindow(customWindow)
//
// 	fmt.Printf("已注册 %d 个窗口\n", len(manager.GetRegisteredWindows()))
// }
//
// // 创建时间戳提取器
// func createTimestampExtractor() *WindowManager.TimestampExtractor {
// 	fmt.Println("3. 创建时间戳提取器...")
//
// 	extractor := WindowManager.NewTimestampExtractor()
//
// 	// 注册时间戳提取策略
// 	extractor.RegisterStrategy("iso8601", func(data interface{}) (time.Time, error) {
// 		if record, ok := data.(map[string]interface{}); ok {
// 			if timestamp, exists := record["timestamp"]; exists {
// 				if tsStr, ok := timestamp.(string); ok {
// 					return time.Parse(time.RFC3339, tsStr)
// 				}
// 			}
// 		}
// 		return time.Time{}, fmt.Errorf("无法提取时间戳")
// 	})
//
// 	// 注册Unix时间戳策略
// 	extractor.RegisterStrategy("unix", func(data interface{}) (time.Time, error) {
// 		if record, ok := data.(map[string]interface{}); ok {
// 			if timestamp, exists := record["unix_timestamp"]; exists {
// 				if tsFloat, ok := timestamp.(float64); ok {
// 					return time.Unix(int64(tsFloat), 0), nil
// 				}
// 			}
// 		}
// 		return time.Time{}, fmt.Errorf("无法提取Unix时间戳")
// 	})
//
// 	return extractor
// }
//
// // 创建聚合计算器
// func createAggregationCalculator() *WindowManager.AggregationCalculator {
// 	fmt.Println("4. 创建聚合计算器...")
//
// 	calculator := WindowManager.NewAggregationCalculator()
//
// 	// 注册聚合函数
// 	calculator.RegisterAggregation("sum", func(values []interface{}) interface{} {
// 		var sum float64
// 		for _, v := range values {
// 			if num, ok := v.(float64); ok {
// 				sum += num
// 			}
// 		}
// 		return sum
// 	})
//
// 	calculator.RegisterAggregation("avg", func(values []interface{}) interface{} {
// 		if len(values) == 0 {
// 			return 0.0
// 		}
// 		var sum float64
// 		for _, v := range values {
// 			if num, ok := v.(float64); ok {
// 				sum += num
// 			}
// 		}
// 		return sum / float64(len(values))
// 	})
//
// 	calculator.RegisterAggregation("count", func(values []interface{}) interface{} {
// 		return len(values)
// 	})
//
// 	calculator.RegisterAggregation("max", func(values []interface{}) interface{} {
// 		if len(values) == 0 {
// 			return nil
// 		}
// 		max := values[0]
// 		for _, v := range values {
// 			if num, ok := v.(float64); ok {
// 				if maxNum, ok := max.(float64); ok {
// 					if num > maxNum {
// 						max = num
// 					}
// 				}
// 			}
// 		}
// 		return max
// 	})
//
// 	calculator.RegisterAggregation("min", func(values []interface{}) interface{} {
// 		if len(values) == 0 {
// 			return nil
// 		}
// 		min := values[0]
// 		for _, v := range values {
// 			if num, ok := v.(float64); ok {
// 				if minNum, ok := min.(float64); ok {
// 					if num < minNum {
// 						min = num
// 					}
// 				}
// 			}
// 		}
// 		return min
// 	})
//
// 	return calculator
// }
//
// // 创建窗口触发器
// func createWindowTrigger() *WindowManager.WindowTrigger {
// 	fmt.Println("5. 创建窗口触发器...")
//
// 	trigger := WindowManager.NewWindowTrigger()
//
// 	// 注册时间触发器
// 	trigger.RegisterTimeTrigger("every-minute", func() time.Duration {
// 		return 1 * time.Minute
// 	})
//
// 	// 注册计数触发器
// 	trigger.RegisterCountTrigger("every-100-records", func() int {
// 		return 100
// 	})
//
// 	// 注册自定义触发器
// 	trigger.RegisterCustomTrigger("high-value-trigger", func(window *WindowManager.Window) bool {
// 		// 当窗口中的高价值记录超过10条时触发
// 		highValueCount := 0
// 		for _, record := range window.GetRecords() {
// 			if recordMap, ok := record.(map[string]interface{}); ok {
// 				if value, exists := recordMap["value"]; exists {
// 					if val, ok := value.(float64); ok && val > 1000 {
// 						highValueCount++
// 					}
// 				}
// 			}
// 		}
// 		return highValueCount >= 10
// 	})
//
// 	return trigger
// }
//
// // 创建窗口状态管理器
// func createWindowStateManager() *WindowManager.WindowStateManager {
// 	fmt.Println("6. 创建窗口状态管理器...")
//
// 	stateManager := WindowManager.NewWindowStateManager()
//
// 	// 设置状态存储路径
// 	stateManager.SetStoragePath("./window-state")
//
// 	// 设置状态序列化器
// 	stateManager.SetSerializer(func(state interface{}) ([]byte, error) {
// 		// 简单的JSON序列化
// 		return []byte(fmt.Sprintf("%v", state)), nil
// 	})
//
// 	// 设置状态反序列化器
// 	stateManager.SetDeserializer(func(data []byte) (interface{}, error) {
// 		// 简单的反序列化
// 		return string(data), nil
// 	})
//
// 	return stateManager
// }
//
// // 模拟数据流和处理
// func simulateDataProcessing(
// 	manager *WindowManager.WindowManager,
// 	extractor *WindowManager.TimestampExtractor,
// 	calculator *WindowManager.AggregationCalculator,
// 	trigger *WindowManager.WindowTrigger,
// 	stateManager *WindowManager.WindowStateManager,
// ) {
// 	fmt.Println("7. 模拟数据流和处理...")
//
// 	// 启动窗口管理器
// 	ctx := context.Background()
// 	err := manager.Start(ctx)
// 	if err != nil {
// 		log.Printf("启动窗口管理器失败: %v", err)
// 		return
// 	}
//
// 	// 设置窗口处理器
// 	manager.SetWindowProcessor(func(window *WindowManager.Window) error {
// 		fmt.Printf("处理窗口: %s, 记录数: %d\n", window.GetID(), len(window.GetRecords()))
//
// 		// 执行聚合计算
// 		results := executeAggregations(window, calculator)
//
// 		// 保存窗口状态
// 		saveWindowState(window, stateManager)
//
// 		// 输出聚合结果
// 		printAggregationResults(window.GetID(), results)
//
// 		return nil
// 	})
//
// 	// 模拟数据流入
// 	go generateTestData(manager, extractor)
//
// 	// 运行一段时间
// 	time.Sleep(2 * time.Minute)
//
// 	// 停止窗口管理器
// 	err = manager.Stop(ctx)
// 	if err != nil {
// 		log.Printf("停止窗口管理器失败: %v", err)
// 	}
// }
//
// // 生成测试数据
// func generateTestData(manager *WindowManager.WindowManager, extractor *WindowManager.TimestampExtractor) {
// 	for i := 0; i < 1000; i++ {
// 		// 创建测试记录
// 		record := map[string]interface{}{
// 			"id":        i,
// 			"value":     float64(i * 10),
// 			"size":      i * 100,
// 			"timestamp": time.Now().Format(time.RFC3339),
// 			"category":  fmt.Sprintf("category-%d", i%5),
// 		}
//
// 		// 提取时间戳
// 		timestamp, err := extractor.ExtractTimestamp(record, "iso8601")
// 		if err != nil {
// 			log.Printf("提取时间戳失败: %v", err)
// 			continue
// 		}
//
// 		// 添加到窗口
// 		err = manager.AddRecord("time-window-5min", record, timestamp)
// 		if err != nil {
// 			log.Printf("添加记录到时间窗口失败: %v", err)
// 		}
//
// 		err = manager.AddRecord("count-window-1000", record, timestamp)
// 		if err != nil {
// 			log.Printf("添加记录到计数窗口失败: %v", err)
// 		}
//
// 		err = manager.AddRecord("session-window-30min", record, timestamp)
// 		if err != nil {
// 			log.Printf("添加记录到会话窗口失败: %v", err)
// 		}
//
// 		err = manager.AddRecord("custom-window", record, timestamp)
// 		if err != nil {
// 			log.Printf("添加记录到自定义窗口失败: %v", err)
// 		}
//
// 		// 模拟数据流入间隔
// 		time.Sleep(100 * time.Millisecond)
// 	}
// }
//
// // 执行聚合计算
// func executeAggregations(window *WindowManager.Window, calculator *WindowManager.AggregationCalculator) map[string]interface{} {
// 	results := make(map[string]interface{})
//
// 	// 提取数值字段
// 	var values []interface{}
// 	for _, record := range window.GetRecords() {
// 		if recordMap, ok := record.(map[string]interface{}); ok {
// 			if value, exists := recordMap["value"]; exists {
// 				values = append(values, value)
// 			}
// 		}
// 	}
//
// 	// 执行各种聚合
// 	if len(values) > 0 {
// 		results["sum"] = calculator.Calculate("sum", values)
// 		results["avg"] = calculator.Calculate("avg", values)
// 		results["count"] = calculator.Calculate("count", values)
// 		results["max"] = calculator.Calculate("max", values)
// 		results["min"] = calculator.Calculate("min", values)
// 	}
//
// 	return results
// }
//
// // 保存窗口状态
// func saveWindowState(window *WindowManager.Window, stateManager *WindowManager.WindowStateManager) {
// 	state := map[string]interface{}{
// 		"window_id":    window.GetID(),
// 		"record_count": len(window.GetRecords()),
// 		"start_time":   window.GetStartTime(),
// 		"end_time":     window.GetEndTime(),
// 		"last_updated": time.Now(),
// 	}
//
// 	err := stateManager.SaveState(window.GetID(), state)
// 	if err != nil {
// 		log.Printf("保存窗口状态失败: %v", err)
// 	}
// }
//
// // 打印聚合结果
// func printAggregationResults(windowID string, results map[string]interface{}) {
// 	fmt.Printf("窗口 %s 聚合结果:\n", windowID)
// 	for key, value := range results {
// 		fmt.Printf("  %s: %v\n", key, value)
// 	}
// 	fmt.Println()
// }
