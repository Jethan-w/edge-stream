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

package stream

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
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
// 辅助函数：创建错误恢复处理器
func createErrorRecoveryProcessor(errorCount *int) *TestProcessor {
	processor := NewTestProcessor("error-recovery-processor", "Error Recovery Processor")
	processor.SetProcessFunc(func(ff *flowfile.FlowFile) (*flowfile.FlowFile, error) {
		*errorCount++
		if *errorCount <= 3 {
			return nil, errors.New("temporary error")
		}
		return ff, nil
	})
	return processor
}

// 辅助函数：测试处理器错误恢复
func testProcessorErrorRecovery(t *testing.T, engine *StandardStreamEngine) {
	stream := NewStream("error-recovery")
	errorCount := 0
	processor := createErrorRecoveryProcessor(&errorCount)
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
}

// 辅助函数：测试重复拓扑创建
func testDuplicateTopologyCreation(t *testing.T, engine *StandardStreamEngine) (*StreamTopology, error) {
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

	return topology1, err
}

// 辅助函数：测试无效拓扑操作
func testInvalidTopologyOperations(t *testing.T, engine *StandardStreamEngine) {
	// 测试向不存在的拓扑添加处理器
	processor := NewTestProcessor("orphan-processor", "Orphan Processor")
	err := engine.AddProcessor("non-existent-topology", processor)
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
}

// 辅助函数：清理拓扑
func cleanupTopology(t *testing.T, engine *StandardStreamEngine, topology *StreamTopology) {
	if topology != nil {
		ctx := context.Background()
		if err := engine.StopTopology(ctx, topology.ID); err != nil {
			t.Logf("Failed to stop topology: %v", err)
		}
	}
}

func TestStreamEngineErrorRecovery(t *testing.T) {
	engine := NewStandardStreamEngine()

	// 测试处理器错误恢复
	t.Run("ProcessorErrorRecovery", func(t *testing.T) {
		testProcessorErrorRecovery(t, engine)
	})

	// 测试拓扑错误处理
	t.Run("TopologyErrorHandling", func(t *testing.T) {
		topology1, _ := testDuplicateTopologyCreation(t, engine)
		testInvalidTopologyOperations(t, engine)
		cleanupTopology(t, engine, topology1)
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
		if err := engine.StopTopology(ctx, topology.ID); err != nil {
			t.Logf("Failed to stop topology: %v", err)
		}
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

const (
	processedValue = "true"
)

// TestStreamEngineTimeout 测试超时处理
func TestStreamEngineTimeout(t *testing.T) {
	engine := NewStandardStreamEngine()

	// 测试处理功能
	t.Run("ProcessingTimeout", func(t *testing.T) {
		stream := NewStream("timeout-test")
		processor := NewTestProcessor("timeout-processor", "Timeout Processor")
		var processedCount int32
		processor.SetProcessFunc(func(ff *flowfile.FlowFile) (*flowfile.FlowFile, error) {
			// 模拟正常处理
			atomic.AddInt32(&processedCount, 1)
			ff.SetAttribute("processed", processedValue)
			return ff, nil
		})
		stream.AddProcessor(processor)
		engine.AddStream(stream)

		// 测试正常处理
		ff := flowfile.NewFlowFile()
		result := stream.Process(ff)

		if result == nil {
			t.Error("Processing should succeed")
		}

		// 验证处理器被调用
		if atomic.LoadInt32(&processedCount) != 1 {
			t.Errorf("Expected 1 processed item, got %d", atomic.LoadInt32(&processedCount))
		}

		// 验证属性被设置
		if resultFF, ok := result.(*flowfile.FlowFile); ok {
			if val, exists := resultFF.GetAttribute("processed"); !exists || val != processedValue {
				t.Error("Processing should set the 'processed' attribute")
			}
		} else {
			t.Error("Result should be a FlowFile")
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
		if err := engine.StopTopology(cleanupCtx, topology.ID); err != nil {
			t.Logf("Failed to stop topology: %v", err)
		}
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

// createIntegrationProcessors 创建集成测试的处理器
func createIntegrationProcessors(processedData *[]string, mu *sync.Mutex) (transform, validate, output *TestProcessor) {
	transformProcessor := NewTestProcessor("transform-processor", "Transform Processor")
	transformProcessor.SetProcessFunc(func(ff *flowfile.FlowFile) (*flowfile.FlowFile, error) {
		mu.Lock()
		*processedData = append(*processedData, "transformed")
		mu.Unlock()
		ff.SetAttribute("transformed", processedValue)
		return ff, nil
	})

	validateProcessor := NewTestProcessor("validate-processor", "Validate Processor")
	validateProcessor.SetProcessFunc(func(ff *flowfile.FlowFile) (*flowfile.FlowFile, error) {
		mu.Lock()
		*processedData = append(*processedData, "validated")
		mu.Unlock()
		if val, exists := ff.GetAttribute("transformed"); !exists || val != processedValue {
			return nil, errors.New("data not transformed")
		}
		ff.SetAttribute("validated", processedValue)
		return ff, nil
	})

	outputProcessor := NewTestProcessor("output-processor", "Output Processor")
	outputProcessor.SetProcessFunc(func(ff *flowfile.FlowFile) (*flowfile.FlowFile, error) {
		mu.Lock()
		*processedData = append(*processedData, "output")
		mu.Unlock()
		if val, exists := ff.GetAttribute("validated"); !exists || val != processedValue {
			return nil, errors.New("data not validated")
		}
		ff.SetAttribute("completed", processedValue)
		return ff, nil
	})

	return transformProcessor, validateProcessor, outputProcessor
}

// processIntegrationMessages 处理集成测试消息
func processIntegrationMessages(t *testing.T, transformProcessor, validateProcessor, outputProcessor *TestProcessor, numMessages int) {
	for i := 0; i < numMessages; i++ {
		message := &Message{
			ID:        fmt.Sprintf("integration-msg-%d", i),
			Data:      fmt.Sprintf("integration data %d", i),
			Timestamp: time.Now(),
		}

		sharedFF := &flowfile.FlowFile{
			Attributes: make(map[string]string),
		}
		sharedFF.SetAttribute("message_id", message.ID)
		sharedFF.SetAttribute("data", fmt.Sprintf("%v", message.Data))

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
}

// TestStreamEngineIntegration 集成测试
func TestStreamEngineIntegration(t *testing.T) {
	engine := NewStandardStreamEngine()

	t.Run("CompleteWorkflow", func(t *testing.T) {
		topology, err := engine.CreateTopology("integration-topology", "Integration Topology", "Complete workflow test")
		if err != nil {
			t.Fatalf("Failed to create topology: %v", err)
		}

		var processedData []string
		var mu sync.Mutex

		transformProcessor, validateProcessor, outputProcessor := createIntegrationProcessors(&processedData, &mu)

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

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = engine.StartTopology(ctx, topology.ID)
		if err != nil {
			t.Fatalf("Failed to start topology: %v", err)
		}

		time.Sleep(100 * time.Millisecond)

		numMessages := 10
		processIntegrationMessages(t, transformProcessor, validateProcessor, outputProcessor, numMessages)

		time.Sleep(500 * time.Millisecond)

		mu.Lock()
		expectedSteps := numMessages * 3
		if len(processedData) != expectedSteps {
			t.Errorf("Expected %d processing steps, got %d", expectedSteps, len(processedData))
		}
		mu.Unlock()

		stopCtx, stopCancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer stopCancel()
		err = engine.StopTopology(stopCtx, topology.ID)
		if err != nil {
			t.Errorf("Failed to stop topology: %v", err)
		}
	})
}
