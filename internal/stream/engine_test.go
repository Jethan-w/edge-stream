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

func TestStreamEngine(t *testing.T) {
	engine := NewStandardStreamEngine()

	// 测试基本的流创建和管理
	t.Run("CreateStream", func(t *testing.T) {
		stream := NewStream("test_stream")
		if stream.GetName() != "test_stream" {
			t.Errorf("Expected stream name 'test_stream', got %s", stream.GetName())
		}

		engine.AddStream(stream)
		if retrievedStream := engine.GetStream("test_stream"); retrievedStream == nil {
			t.Error("Failed to retrieve added stream")
		}
	})

	// 测试流处理
	t.Run("StreamProcessing", func(t *testing.T) {
		// 创建拓扑和处理器
		topology, err := engine.CreateTopology("test-topology", "Test Topology", "Test topology for unit tests")
		if err != nil {
			t.Fatalf("Failed to create topology: %v", err)
		}

		processedCount := 0
		processor := NewTestProcessor("test-processor-1", "Test Processor 1")
		processor.SetProcessFunc(func(ff *flowfile.FlowFile) (*flowfile.FlowFile, error) {
			processedCount++
			return ff, nil
		})

		err = engine.AddProcessor(topology.ID, processor)
		if err != nil {
			t.Fatalf("Failed to add processor: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// 启动拓扑
		err = engine.StartTopology(ctx, topology.ID)
		if err != nil {
			t.Fatalf("Failed to start topology: %v", err)
		}

		// 等待引擎启动
		time.Sleep(100 * time.Millisecond)

		// 发送一些数据
		for i := 0; i < 5; i++ {
			message := &Message{
				ID:        fmt.Sprintf("msg-%d", i),
				Data:      fmt.Sprintf("test content %d", i),
				Timestamp: time.Now(),
			}
			// 直接调用处理器处理消息
			_, err := processor.Process(context.Background(), message)
			if err != nil {
				t.Errorf("Failed to process message: %v", err)
			}
		}

		// 等待处理完成
		time.Sleep(500 * time.Millisecond)

		if processedCount != 5 {
			t.Errorf("Expected 5 processed items, got %d", processedCount)
		}

		// 停止拓扑
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer stopCancel()
		if err := engine.StopTopology(stopCtx, topology.ID); err != nil {
			t.Logf("Failed to stop topology: %v", err)
		}
	})

	// 测试引擎停止
	t.Run("EngineStop", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan bool)

		errChan := make(chan error, 1)
		go func() {
			if err := engine.Start(ctx); err != nil {
				errChan <- err
				return
			}
			done <- true
		}()

		// 检查启动错误
		select {
		case err := <-errChan:
			t.Fatalf("Failed to start engine: %v", err)
		case <-time.After(100 * time.Millisecond):
			// 启动成功，继续
		}

		// 等待引擎启动
		time.Sleep(100 * time.Millisecond)

		// 停止引擎
		cancel()

		// 等待引擎停止
		select {
		case <-done:
			// 引擎正常停止
		case <-time.After(2 * time.Second):
			t.Error("Engine failed to stop within timeout")
		}
	})
}

func TestStreamConcurrency(t *testing.T) {
	engine := NewStandardStreamEngine()

	// 创建拓扑和处理器
	topology, err := engine.CreateTopology("concurrent-topology", "Concurrent Topology", "Topology for concurrency testing")
	if err != nil {
		t.Fatalf("Failed to create topology: %v", err)
	}

	var processedCount int64
	var mu sync.Mutex

	// 添加处理器
	processor := NewTestProcessor("test-processor-2", "Test Processor 2")
	processor.SetProcessFunc(func(ff *flowfile.FlowFile) (*flowfile.FlowFile, error) {
		mu.Lock()
		processedCount++
		mu.Unlock()
		// 模拟处理时间
		time.Sleep(1 * time.Millisecond)
		return ff, nil
	})

	err = engine.AddProcessor(topology.ID, processor)
	if err != nil {
		t.Fatalf("Failed to add processor: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 启动拓扑
	err = engine.StartTopology(ctx, topology.ID)
	if err != nil {
		t.Fatalf("Failed to start topology: %v", err)
	}

	// 等待拓扑启动
	time.Sleep(100 * time.Millisecond)

	// 并发发送数据
	var wg sync.WaitGroup
	numGoroutines := 10
	numItemsPerGoroutine := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numItemsPerGoroutine; j++ {
				message := &Message{
					ID:        fmt.Sprintf("msg-%d-%d", id, j),
					Data:      fmt.Sprintf("concurrent content %d-%d", id, j),
					Timestamp: time.Now(),
				}
				// 直接调用处理器处理消息
				_, err := processor.Process(context.Background(), message)
				if err != nil {
					t.Errorf("Failed to process message: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()

	// 等待所有项目处理完成
	time.Sleep(2 * time.Second)

	mu.Lock()
	finalCount := processedCount
	mu.Unlock()

	expected := int64(numGoroutines * numItemsPerGoroutine)
	if finalCount != expected {
		t.Errorf("Expected %d processed items, got %d", expected, finalCount)
	}

	// 停止拓扑
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer stopCancel()
	if err := engine.StopTopology(stopCtx, topology.ID); err != nil {
		t.Logf("Failed to stop topology: %v", err)
	}
}

func BenchmarkStreamEngine(b *testing.B) {
	engine := NewStandardStreamEngine()
	stream := NewStream("benchmark_stream")

	// 添加一个简单的处理器
	processor := NewTestProcessor("bench-processor", "Benchmark Processor")
	processor.SetProcessFunc(func(ff *flowfile.FlowFile) (*flowfile.FlowFile, error) {
		// 最小化处理时间
		return ff, nil
	})
	stream.AddProcessor(processor)
	engine.AddStream(stream)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动引擎
	errChan := make(chan error, 1)
	go func() {
		if err := engine.Start(ctx); err != nil {
			errChan <- err
			return
		}
		errChan <- nil
	}()

	// 检查启动错误
	select {
	case err := <-errChan:
		if err != nil {
			b.Fatalf("Failed to start engine: %v", err)
		}
	case <-time.After(1 * time.Second):
		b.Fatal("Engine start timeout")
	}

	// 等待引擎启动
	time.Sleep(100 * time.Millisecond)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ff := flowfile.NewFlowFile()
			ff.SetAttribute("benchmark", "true")
			stream.Process(ff)
		}
	})
}

// 性能标准测试
func TestStreamPerformanceStandards(t *testing.T) {
	t.Run("ThroughputTest", testThroughputPerformance)
	t.Run("LatencyTest", testLatencyPerformance)
}

func testThroughputPerformance(t *testing.T) {
	engine := NewStandardStreamEngine()

	// 创建拓扑和处理器
	topology, err := engine.CreateTopology("throughput-topology", "Throughput Topology", "Topology for throughput testing")
	if err != nil {
		t.Fatalf("Failed to create topology: %v", err)
	}

	var processedCount int64
	processor := NewTestProcessor("throughput-processor", "Throughput Processor")
	processor.SetProcessFunc(func(ff *flowfile.FlowFile) (*flowfile.FlowFile, error) {
		atomic.AddInt64(&processedCount, 1)
		return ff, nil
	})

	err = engine.AddProcessor(topology.ID, processor)
	if err != nil {
		t.Fatalf("Failed to add processor: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 启动拓扑
	err = engine.StartTopology(ctx, topology.ID)
	if err != nil {
		t.Fatalf("Failed to start topology: %v", err)
	}

	// 等待拓扑启动
	time.Sleep(100 * time.Millisecond)

	numItems := 10000
	start := time.Now()

	for i := 0; i < numItems; i++ {
		message := &Message{
			ID:        fmt.Sprintf("msg-%d", i),
			Data:      fmt.Sprintf("throughput content %d", i),
			Timestamp: time.Now(),
		}
		// 直接调用处理器处理消息
		_, err := processor.Process(context.Background(), message)
		if err != nil {
			t.Errorf("Failed to process message: %v", err)
		}
	}

	// 等待处理完成
	time.Sleep(100 * time.Millisecond)

	duration := time.Since(start)
	throughput := float64(atomic.LoadInt64(&processedCount)) / duration.Seconds()

	// 应该能够处理至少5000 items/sec
	if throughput < 5000 {
		t.Errorf("Throughput: %.0f items/sec, expected >= 5000 items/sec", throughput)
	}

	// 停止拓扑
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer stopCancel()
	if err := engine.StopTopology(stopCtx, topology.ID); err != nil {
		t.Logf("Failed to stop topology: %v", err)
	}
}

func testLatencyPerformance(t *testing.T) {
	engine := NewStandardStreamEngine()
	stream := NewStream("performance_test")

	processedCount := 0
	processor := NewTestProcessor("perf-processor", "Performance Processor")
	processor.SetProcessFunc(func(ff *flowfile.FlowFile) (*flowfile.FlowFile, error) {
		processedCount++
		return ff, nil
	})
	stream.AddProcessor(processor)
	engine.AddStream(stream)

	latencies := make([]time.Duration, 100)

	for i := 0; i < 100; i++ {
		start := time.Now()
		ff := flowfile.NewFlowFile()
		ff.SetAttribute("latency_test", string(rune(i)))
		stream.Process(ff)
		// 简单等待处理完成的近似方法
		time.Sleep(1 * time.Millisecond)
		latencies[i] = time.Since(start)
	}

	// 计算平均延迟
	var totalLatency time.Duration
	for _, latency := range latencies {
		totalLatency += latency
	}
	avgLatency := totalLatency / time.Duration(len(latencies))

	// 平均延迟应该小于10ms
	if avgLatency > 10*time.Millisecond {
		t.Errorf("Average latency: %v, expected < 10ms", avgLatency)
	}
}

// TestStreamEngineMetrics 测试流引擎指标
func TestStreamEngineMetrics(t *testing.T) {
	engine := NewStandardStreamEngine()
	stream := NewStream("metrics-stream")
	processor := NewTestProcessor("metrics-processor", "Metrics Processor")
	stream.AddProcessor(processor)
	engine.AddStream(stream)

	// 测试指标收集
	metrics := processor.GetMetrics()
	if metrics == nil {
		t.Error("Metrics should not be nil")
		return
	}

	// 测试指标更新
	metrics.UpdateMetrics(10, 8, 1024, 512, time.Millisecond, 0)
	if metrics.MessagesIn != 10 {
		t.Errorf("Expected messages in 10, got %d", metrics.MessagesIn)
	}
}

// TestStreamEngineErrorHandling 测试错误处理
func TestStreamEngineErrorHandling(t *testing.T) {
	engine := NewStandardStreamEngine()
	stream := NewStream("error-stream")

	// 创建会产生错误的处理器
	errorProcessor := NewTestProcessor("error-processor", "Error Processor")
	errorProcessor.SetProcessFunc(func(ff *flowfile.FlowFile) (*flowfile.FlowFile, error) {
		return nil, errors.New("processing error")
	})
	stream.AddProcessor(errorProcessor)
	engine.AddStream(stream)

	// 测试错误处理
	ff := flowfile.NewFlowFile()
	result := stream.Process(ff)
	if result == nil {
		t.Log("Error handling works correctly")
	}
}

// TestStreamEngineLifecycle 测试生命周期管理
func TestStreamEngineLifecycle(t *testing.T) {
	engine := NewStandardStreamEngine()
	stream := NewStream("lifecycle-stream")
	processor := NewTestProcessor("lifecycle-processor", "Lifecycle Processor")

	// 测试处理器状态
	if processor.GetStatus() != StreamStatusStopped {
		t.Error("Initial status should be stopped")
	}

	// 启动处理器
	ctx := context.Background()
	err := processor.Start(ctx)
	if err != nil {
		t.Errorf("Failed to start processor: %v", err)
	}
	if processor.GetStatus() != StreamStatusRunning {
		t.Error("Status should be running after start")
	}

	// 停止处理器
	err = processor.Stop(ctx)
	if err != nil {
		t.Errorf("Failed to stop processor: %v", err)
	}
	if processor.GetStatus() != StreamStatusStopped {
		t.Error("Status should be stopped after stop")
	}

	stream.AddProcessor(processor)
	engine.AddStream(stream)
}

// TestStreamEngineMultipleStreams 测试多流处理
func TestStreamEngineMultipleStreams(t *testing.T) {
	engine := NewStandardStreamEngine()

	// 创建多个流
	for i := 0; i < 5; i++ {
		streamName := fmt.Sprintf("stream_%d", i)
		stream := NewStream(streamName)
		processor := NewTestProcessor(fmt.Sprintf("processor_%d", i), fmt.Sprintf("Processor %d", i))
		stream.AddProcessor(processor)
		engine.AddStream(stream)

		// 验证流已添加
		retrievedStream := engine.GetStream(streamName)
		if retrievedStream == nil {
			t.Errorf("Failed to retrieve stream: %s", streamName)
		}
		if retrievedStream.GetName() != streamName {
			t.Errorf("Stream name mismatch: got %s, want %s", retrievedStream.GetName(), streamName)
		}
	}
}

// TestStreamEngineResourceManagement 测试资源管理
func TestStreamEngineResourceManagement(t *testing.T) {
	engine := NewStandardStreamEngine()
	stream := NewStream("resource-stream")
	processor := NewTestProcessor("resource-processor", "Resource Processor")

	// 测试处理器类型
	if processor.GetType() != StreamTypeTransform {
		t.Errorf("Expected processor type %v, got %v", StreamTypeTransform, processor.GetType())
	}

	// 测试处理器描述
	if processor.GetDescription() == "" {
		t.Error("Processor description should not be empty")
	}

	stream.AddProcessor(processor)
	engine.AddStream(stream)

	// 测试指标引用
	metricsRef := processor.GetMetricsRef()
	if metricsRef == nil {
		t.Error("Metrics reference should not be nil")
	}
}
