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

package performance

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/crazy/edge-stream/internal/config"
	"github.com/crazy/edge-stream/internal/metrics"
	"github.com/crazy/edge-stream/internal/state"
	"github.com/crazy/edge-stream/internal/stream"
)

// 全局变量防止编译器优化
var (
	globalResult interface{}
	globalError  error
)

// BenchmarkConfigManager 配置管理器性能测试
func BenchmarkConfigManager(b *testing.B) {
	b.Run("Set", func(b *testing.B) {
		cm := config.NewStandardConfigManager("")
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			err := cm.Set(fmt.Sprintf("benchmark_key_%d", i), fmt.Sprintf("value_%d", i))
			globalError = err // 防止编译器优化
		}
	})

	b.Run("Get", func(b *testing.B) {
		cm := config.NewStandardConfigManager("")
		// 预设数据
		for i := 0; i < 1000; i++ {
			cm.Set(fmt.Sprintf("get_key_%d", i), fmt.Sprintf("value_%d", i))
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			result := cm.GetString(fmt.Sprintf("get_key_%d", i%1000))
			globalResult = result // 防止编译器优化
		}
	})

	b.Run("SetParallel", func(b *testing.B) {
		cm := config.NewStandardConfigManager("")
		b.ResetTimer()
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				err := cm.Set(fmt.Sprintf("parallel_key_%d", i), fmt.Sprintf("value_%d", i))
				globalError = err
				i++
			}
		})
	})
}

// BenchmarkMetricsCollector 指标收集器性能测试
func BenchmarkMetricsCollector(b *testing.B) {
	b.Run("RecordCounter", func(b *testing.B) {
		mc := metrics.NewStandardMetricCollector()
		labels := map[string]string{"service": "test", "method": "benchmark"}
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			mc.RecordCounter("benchmark_counter", 1.0, labels)
		}
	})

	b.Run("RecordGauge", func(b *testing.B) {
		mc := metrics.NewStandardMetricCollector()
		labels := map[string]string{"service": "test", "type": "gauge"}
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			mc.RecordGauge("benchmark_gauge", float64(i), labels)
		}
	})

	b.Run("RecordLatency", func(b *testing.B) {
		mc := metrics.NewStandardMetricCollector()
		labels := map[string]string{"operation": "test"}
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			mc.RecordLatency("benchmark_latency", time.Duration(i)*time.Microsecond, labels)
		}
	})

	b.Run("RecordParallel", func(b *testing.B) {
		mc := metrics.NewStandardMetricCollector()
		b.ResetTimer()
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				labels := map[string]string{"worker": fmt.Sprintf("%d", i%4)}
				mc.RecordCounter("parallel_counter", 1.0, labels)
				i++
			}
		})
	})
}

// BenchmarkStateManager 状态管理器性能测试
func BenchmarkStateManager(b *testing.B) {
	b.Run("CreateState", func(b *testing.B) {
		sm := state.NewStandardStateManager(state.DefaultStateConfig())
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			stateName := fmt.Sprintf("benchmark_state_%d", i)
			testState, err := sm.CreateState(stateName, state.StateTypeMemory)
			globalResult = testState
			globalError = err
		}
	})

	b.Run("StateSet", func(b *testing.B) {
		sm := state.NewStandardStateManager(state.DefaultStateConfig())
		testState, err := sm.CreateState("benchmark_state", state.StateTypeMemory)
		if err != nil {
			b.Fatalf("Failed to create state: %v", err)
		}
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			err := testState.Set(fmt.Sprintf("key_%d", i), fmt.Sprintf("value_%d", i))
			globalError = err
		}
	})

	b.Run("StateGet", func(b *testing.B) {
		sm := state.NewStandardStateManager(state.DefaultStateConfig())
		testState, _ := sm.CreateState("benchmark_state", state.StateTypeMemory)
		// 预设数据
		for i := 0; i < 1000; i++ {
			err := testState.Set(fmt.Sprintf("get_key_%d", i), fmt.Sprintf("value_%d", i))
			if err != nil {
				b.Errorf("Failed to set state: %v", err)
			}
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			result, exists := testState.Get(fmt.Sprintf("get_key_%d", i%1000))
			globalResult = result
			globalError = nil
			if !exists {
				globalError = fmt.Errorf("key not found")
			}
		}
	})

	b.Run("StateParallel", func(b *testing.B) {
		sm := state.NewStandardStateManager(state.DefaultStateConfig())
		testState, err := sm.CreateState("parallel_state", state.StateTypeMemory)
		if err != nil {
			b.Fatalf("Failed to create state: %v", err)
		}
		b.ResetTimer()
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				err := testState.Set(fmt.Sprintf("parallel_key_%d", i), fmt.Sprintf("value_%d", i))
				globalError = err
				i++
			}
		})
	})
}

// BenchmarkStreamEngine 流处理引擎性能测试
func BenchmarkStreamEngine(b *testing.B) {
	b.Run("CreateEngine", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			engine := stream.NewStandardStreamEngine()
			globalResult = engine
		}
	})
}

// BenchmarkMemoryIntensive 内存密集型操作测试
func BenchmarkMemoryIntensive(b *testing.B) {
	b.Run("LargeDataSet", func(b *testing.B) {
		cm := config.NewStandardConfigManager("")
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			b.StopTimer()
			// 生成大数据，不计入测试时间
			largeData := make([]byte, 1024) // 1KB数据
			for j := range largeData {
				largeData[j] = byte(i + j)
			}
			b.StartTimer()

			// 核心操作计时
			err := cm.Set(fmt.Sprintf("large_key_%d", i), string(largeData))
			globalError = err
		}
	})

	b.Run("BatchOperations", func(b *testing.B) {
		mc := metrics.NewStandardMetricCollector()
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// 批量操作
			for j := 0; j < 10; j++ {
				labels := map[string]string{
					"batch": fmt.Sprintf("%d", i),
					"item":  fmt.Sprintf("%d", j),
				}
				mc.RecordCounter("batch_counter", 1.0, labels)
			}
		}
	})
}

// BenchmarkConcurrentAccess 并发访问测试
func BenchmarkConcurrentAccess(b *testing.B) {
	b.Run("ConfigConcurrent", func(b *testing.B) {
		cm := config.NewStandardConfigManager("")
		var wg sync.WaitGroup
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			wg.Add(2)
			// 并发读写
			go func(id int) {
				defer wg.Done()
				cm.Set(fmt.Sprintf("concurrent_key_%d", id), fmt.Sprintf("value_%d", id))
			}(i)
			go func(id int) {
				defer wg.Done()
				result := cm.GetString(fmt.Sprintf("concurrent_key_%d", id))
				globalResult = result
			}(i)
		}
		wg.Wait()
	})

	b.Run("MetricsConcurrent", func(b *testing.B) {
		mc := metrics.NewStandardMetricCollector()
		var wg sync.WaitGroup
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			wg.Add(3)
			// 并发记录不同类型指标
			go func(id int) {
				defer wg.Done()
				mc.RecordCounter("concurrent_counter", 1.0, map[string]string{"id": fmt.Sprintf("%d", id)})
			}(i)
			go func(id int) {
				defer wg.Done()
				mc.RecordGauge("concurrent_gauge", float64(id), map[string]string{"id": fmt.Sprintf("%d", id)})
			}(i)
			go func(id int) {
				defer wg.Done()
				mc.RecordLatency("concurrent_latency", time.Duration(id)*time.Microsecond, map[string]string{"id": fmt.Sprintf("%d", id)})
			}(i)
		}
		wg.Wait()
	})
}

// BenchmarkSystemIntegration 系统集成性能测试
func BenchmarkSystemIntegration(b *testing.B) {
	b.Run("FullWorkflow", func(b *testing.B) {
		// 初始化所有组件
		cm := config.NewStandardConfigManager("")
		mc := metrics.NewStandardMetricCollector()
		sm := state.NewStandardStateManager(state.DefaultStateConfig())
		testState, _ := sm.CreateState("integration_state", state.StateTypeMemory)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			// 模拟完整工作流程
			key := fmt.Sprintf("workflow_key_%d", i)
			value := fmt.Sprintf("workflow_value_%d", i)

			// 1. 配置操作
			cm.Set(key, value)

			// 2. 状态操作
			err := testState.Set(key, value)
			if err != nil {
				b.Errorf("Failed to set state: %v", err)
			}

			// 3. 指标记录
			mc.RecordCounter("workflow_operations", 1.0, map[string]string{
				"operation": "integration",
				"step":      "complete",
			})
		}
	})
}
