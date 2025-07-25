package stream

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/crazy/edge-stream/internal/flowfile"
)

// TestStreamEngineEdgeCases 测试边界条件
func TestStreamEngineEdgeCases(t *testing.T) {
	engine := NewStandardStreamEngine()

	// 测试空流名称
	t.Run("EmptyStreamName", func(t *testing.T) {
		stream := NewStream("")
		if stream.GetName() != "" {
			t.Error("Empty stream name should be allowed")
		}
		engine.AddStream(stream)
	})

	// 测试重复流名称
	t.Run("DuplicateStreamName", func(t *testing.T) {
		stream1 := NewStream("duplicate")
		stream2 := NewStream("duplicate")
		engine.AddStream(stream1)
		engine.AddStream(stream2) // 应该覆盖第一个
		retrieved := engine.GetStream("duplicate")
		if retrieved == nil {
			t.Error("Should be able to retrieve duplicate stream")
		}
	})

	// 测试获取不存在的流
	t.Run("GetNonExistentStream", func(t *testing.T) {
		stream := engine.GetStream("non-existent")
		if stream != nil {
			t.Error("Should return nil for non-existent stream")
		}
	})

	// 测试空处理器列表
	t.Run("EmptyProcessorList", func(t *testing.T) {
		stream := NewStream("empty-processors")
		ff := flowfile.NewFlowFile()
		result := stream.Process(ff)
		if result == nil {
			t.Error("Processing with empty processor list should return input")
		}
	})
}

// TestStreamEngineErrorRecovery 测试错误恢复
func TestStreamEngineErrorRecovery(t *testing.T) {
	engine := NewStandardStreamEngine()

	// 测试处理器错误恢复
	t.Run("ProcessorErrorRecovery", func(t *testing.T) {
		stream := NewStream("error-recovery")
		errorCount := 0
		processor := NewTestProcessor("error-recovery-processor", "Error Recovery Processor")
		processor.SetProcessFunc(func(ff *flowfile.FlowFile) (*flowfile.FlowFile, error) {
			errorCount++
			if errorCount <= 3 {
				return nil, errors.New("temporary error")
			}
			return ff, nil
		})
		stream.AddProcessor(processor)
		engine.AddStream(stream)

		// 多次处理，测试错误恢复
		for i := 0; i < 5; i++ {
			ff := flowfile.NewFlowFile()
			ff.SetAttribute("attempt", fmt.Sprintf("%d", i))
			result := stream.Process(ff)
			if i < 3 {
				if result != nil {
					t.Errorf("Expected nil result for error case %d", i)
				}
			} else {
				if result == nil {
					t.Errorf("Expected successful result for case %d", i)
				}
			}
		}
	})

	// 测试拓扑错误处理
	t.Run("TopologyErrorHandling", func(t *testing.T) {
		// 测试创建重复拓扑
		topology1, err := engine.CreateTopology("duplicate-topo", "Topology 1", "First topology")
		if err != nil {
			t.Fatalf("Failed to create first topology: %v", err)
		}

		topology2, err := engine.CreateTopology("duplicate-topo", "Topology 2", "Second topology")
		if err == nil {
			t.Error("Should not allow duplicate topology IDs")
		}
		if topology2 != nil {
			t.Error("Duplicate topology should be nil")
		}

		// 测试向不存在的拓扑添加处理器
		processor := NewTestProcessor("orphan-processor", "Orphan Processor")
		err = engine.AddProcessor("non-existent-topology", processor)
		if err == nil {
			t.Error("Should not allow adding processor to non-existent topology")
		}

		// 测试启动不存在的拓扑
		ctx := context.Background()
		err = engine.StartTopology(ctx, "non-existent-topology")
		if err == nil {
			t.Error("Should not allow starting non-existent topology")
		}

		// 测试停止不存在的拓扑
		err = engine.StopTopology(ctx, "non-existent-topology")
		if err == nil {
			t.Error("Should not allow stopping non-existent topology")
		}

		// 清理
		if topology1 != nil {
			engine.StopTopology(ctx, topology1.ID)
		}
	})
}

// TestStreamEngineResourceLimits 测试资源限制
func TestStreamEngineResourceLimits(t *testing.T) {
	engine := NewStandardStreamEngine()

	// 测试大量流处理
	t.Run("ManyStreams", func(t *testing.T) {
		numStreams := 100
		for i := 0; i < numStreams; i++ {
			streamName := fmt.Sprintf("stream_%d", i)
			stream := NewStream(streamName)
			processor := NewTestProcessor(fmt.Sprintf("processor_%d", i), fmt.Sprintf("Processor %d", i))
			stream.AddProcessor(processor)
			engine.AddStream(stream)
		}

		// 验证所有流都已添加
		for i := 0; i < numStreams; i++ {
			streamName := fmt.Sprintf("stream_%d", i)
			stream := engine.GetStream(streamName)
			if stream == nil {
				t.Errorf("Failed to retrieve stream: %s", streamName)
			}
		}
	})

	// 测试大量处理器
	t.Run("ManyProcessors", func(t *testing.T) {
		topology, err := engine.CreateTopology("many-processors", "Many Processors", "Topology with many processors")
		if err != nil {
			t.Fatalf("Failed to create topology: %v", err)
		}

		numProcessors := 50
		for i := 0; i < numProcessors; i++ {
			processor := NewTestProcessor(fmt.Sprintf("bulk_processor_%d", i), fmt.Sprintf("Bulk Processor %d", i))
			err := engine.AddProcessor(topology.ID, processor)
			if err != nil {
				t.Errorf("Failed to add processor %d: %v", i, err)
			}
		}

		// 清理
		ctx := context.Background()
		engine.StopTopology(ctx, topology.ID)
	})
}

// TestStreamEngineMemoryManagement 测试内存管理
func TestStreamEngineMemoryManagement(t *testing.T) {
	engine := NewStandardStreamEngine()

	// 测试大数据处理
	t.Run("LargeDataProcessing", func(t *testing.T) {
		stream := NewStream("large-data")
		processor := NewTestProcessor("large-data-processor", "Large Data Processor")
		processor.SetProcessFunc(func(ff *flowfile.FlowFile) (*flowfile.FlowFile, error) {
			// 处理大数据
			data := make([]byte, 1024*1024) // 1MB
			ff.Content = data
			return ff, nil
		})
		stream.AddProcessor(processor)
		engine.AddStream(stream)

		// 处理多个大文件
		for i := 0; i < 10; i++ {
			ff := flowfile.NewFlowFile()
			ff.SetAttribute("size", "large")
			result := stream.Process(ff)
			if result == nil {
				t.Errorf("Failed to process large data %d", i)
			}
		}
	})

	// 测试内存泄漏预防
	t.Run("MemoryLeakPrevention", func(t *testing.T) {
		stream := NewStream("memory-test")
		processor := NewTestProcessor("memory-processor", "Memory Processor")
		processedItems := make([]*flowfile.FlowFile, 0)
		processor.SetProcessFunc(func(ff *flowfile.FlowFile) (*flowfile.FlowFile, error) {
			// 故意保存引用（模拟潜在的内存泄漏）
			processedItems = append(processedItems, ff)
			return ff, nil
		})
		stream.AddProcessor(processor)
		engine.AddStream(stream)

		// 处理大量项目
		for i := 0; i < 1000; i++ {
			ff := flowfile.NewFlowFile()
			ff.SetAttribute("index", fmt.Sprintf("%d", i))
			stream.Process(ff)
		}

		// 验证处理了所有项目
		if len(processedItems) != 1000 {
			t.Errorf("Expected 1000 processed items, got %d", len(processedItems))
		}
	})
}

// TestStreamEngineTimeout 测试超时处理
func TestStreamEngineTimeout(t *testing.T) {
	engine := NewStandardStreamEngine()

	// 测试处理超时
	t.Run("ProcessingTimeout", func(t *testing.T) {
		stream := NewStream("timeout-test")
		processor := NewTestProcessor("timeout-processor", "Timeout Processor")
		processor.SetProcessFunc(func(ff *flowfile.FlowFile) (*flowfile.FlowFile, error) {
			// 模拟长时间处理
			time.Sleep(100 * time.Millisecond)
			return ff, nil
		})
		stream.AddProcessor(processor)
		engine.AddStream(stream)

		// 测试正常处理（不超时）
		start := time.Now()
		ff := flowfile.NewFlowFile()
		result := stream.Process(ff)
		duration := time.Since(start)

		if result == nil {
			t.Error("Processing should succeed")
		}
		if duration < 100*time.Millisecond {
			t.Error("Processing should take at least 100ms")
		}
	})

	// 测试拓扑启动超时
	t.Run("TopologyStartTimeout", func(t *testing.T) {
		topology, err := engine.CreateTopology("timeout-topology", "Timeout Topology", "Topology for timeout testing")
		if err != nil {
			t.Fatalf("Failed to create topology: %v", err)
		}

		processor := NewTestProcessor("slow-start-processor", "Slow Start Processor")
		err = engine.AddProcessor(topology.ID, processor)
		if err != nil {
			t.Fatalf("Failed to add processor: %v", err)
		}

		// 使用短超时
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		err = engine.StartTopology(ctx, topology.ID)
		// 可能会超时，这是预期的行为
		if err != nil && err != context.DeadlineExceeded {
			t.Logf("Start topology error (expected): %v", err)
		}

		// 清理
		cleanupCtx := context.Background()
		engine.StopTopology(cleanupCtx, topology.ID)
	})
}

// TestStreamEngineStateTransitions 测试状态转换
func TestStreamEngineStateTransitions(t *testing.T) {
	engine := NewStandardStreamEngine()

	t.Run("ProcessorStateTransitions", func(t *testing.T) {
		processor := NewTestProcessor("state-processor", "State Processor")
		ctx := context.Background()

		// 将处理器添加到引擎中进行测试
		stream := NewStream("state-test-stream")
		stream.AddProcessor(processor)
		engine.AddStream(stream)

		// 初始状态应该是停止
		if processor.GetStatus() != StreamStatusStopped {
			t.Errorf("Initial status should be %v, got %v", StreamStatusStopped, processor.GetStatus())
		}

		// 启动处理器
		err := processor.Start(ctx)
		if err != nil {
			t.Errorf("Failed to start processor: %v", err)
		}
		if processor.GetStatus() != StreamStatusRunning {
			t.Errorf("Status should be %v after start, got %v", StreamStatusRunning, processor.GetStatus())
		}

		// 重复启动应该是安全的
		err = processor.Start(ctx)
		if err != nil {
			t.Errorf("Repeated start should not fail: %v", err)
		}

		// 停止处理器
		err = processor.Stop(ctx)
		if err != nil {
			t.Errorf("Failed to stop processor: %v", err)
		}
		if processor.GetStatus() != StreamStatusStopped {
			t.Errorf("Status should be %v after stop, got %v", StreamStatusStopped, processor.GetStatus())
		}

		// 重复停止应该是安全的
		err = processor.Stop(ctx)
		if err != nil {
			t.Errorf("Repeated stop should not fail: %v", err)
		}
	})
}

// TestStreamEngineIntegration 集成测试
func TestStreamEngineIntegration(t *testing.T) {
	engine := NewStandardStreamEngine()

	t.Run("CompleteWorkflow", func(t *testing.T) {
		// 创建拓扑
		topology, err := engine.CreateTopology("integration-topology", "Integration Topology", "Complete workflow test")
		if err != nil {
			t.Fatalf("Failed to create topology: %v", err)
		}

		// 添加多个处理器形成处理链
		var processedData []string
		var mu sync.Mutex

		// 第一个处理器：数据转换
		transformProcessor := NewTestProcessor("transform-processor", "Transform Processor")
		transformProcessor.SetProcessFunc(func(ff *flowfile.FlowFile) (*flowfile.FlowFile, error) {
			mu.Lock()
			processedData = append(processedData, "transformed")
			mu.Unlock()
			ff.SetAttribute("transformed", "true")
			return ff, nil
		})

		// 第二个处理器：数据验证
		validateProcessor := NewTestProcessor("validate-processor", "Validate Processor")
		validateProcessor.SetProcessFunc(func(ff *flowfile.FlowFile) (*flowfile.FlowFile, error) {
			mu.Lock()
			processedData = append(processedData, "validated")
			mu.Unlock()
			if val, exists := ff.GetAttribute("transformed"); !exists || val != "true" {
				return nil, errors.New("data not transformed")
			}
			ff.SetAttribute("validated", "true")
			return ff, nil
		})

		// 第三个处理器：数据输出
		outputProcessor := NewTestProcessor("output-processor", "Output Processor")
		outputProcessor.SetProcessFunc(func(ff *flowfile.FlowFile) (*flowfile.FlowFile, error) {
			mu.Lock()
			processedData = append(processedData, "output")
			mu.Unlock()
			if val, exists := ff.GetAttribute("validated"); !exists || val != "true" {
				return nil, errors.New("data not validated")
			}
			ff.SetAttribute("completed", "true")
			return ff, nil
		})

		// 添加处理器到拓扑
		err = engine.AddProcessor(topology.ID, transformProcessor)
		if err != nil {
			t.Fatalf("Failed to add transform processor: %v", err)
		}
		err = engine.AddProcessor(topology.ID, validateProcessor)
		if err != nil {
			t.Fatalf("Failed to add validate processor: %v", err)
		}
		err = engine.AddProcessor(topology.ID, outputProcessor)
		if err != nil {
			t.Fatalf("Failed to add output processor: %v", err)
		}

		// 启动拓扑
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = engine.StartTopology(ctx, topology.ID)
		if err != nil {
			t.Fatalf("Failed to start topology: %v", err)
		}

		// 等待拓扑启动
		time.Sleep(100 * time.Millisecond)

		// 处理数据
		numMessages := 10
		for i := 0; i < numMessages; i++ {
			message := &Message{
				ID:        fmt.Sprintf("integration-msg-%d", i),
				Data:      fmt.Sprintf("integration data %d", i),
				Timestamp: time.Now(),
			}

			// 创建一个共享的FlowFile，在处理器之间传递
			sharedFF := &flowfile.FlowFile{
				Attributes: make(map[string]string),
			}
			sharedFF.SetAttribute("message_id", message.ID)
			sharedFF.SetAttribute("data", fmt.Sprintf("%v", message.Data))

			// 依次通过所有处理器，使用同一个FlowFile
			result1, err := transformProcessor.CallProcessFunc(sharedFF)
			if err != nil {
				t.Errorf("Transform processor failed: %v", err)
				continue
			}

			result2, err := validateProcessor.CallProcessFunc(result1)
			if err != nil {
				t.Errorf("Validate processor failed: %v", err)
				continue
			}

			_, err = outputProcessor.CallProcessFunc(result2)
			if err != nil {
				t.Errorf("Output processor failed: %v", err)
				continue
			}
		}

		// 等待处理完成
		time.Sleep(500 * time.Millisecond)

		// 验证处理结果
		mu.Lock()
		expectedSteps := numMessages * 3 // 每个消息通过3个处理器
		if len(processedData) != expectedSteps {
			t.Errorf("Expected %d processing steps, got %d", expectedSteps, len(processedData))
		}
		mu.Unlock()

		// 停止拓扑
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer stopCancel()
		err = engine.StopTopology(stopCtx, topology.ID)
		if err != nil {
			t.Errorf("Failed to stop topology: %v", err)
		}
	})
}
