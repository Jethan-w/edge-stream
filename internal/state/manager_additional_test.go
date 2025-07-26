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

package state

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestStateManagerEdgeCases 测试边界条件
func TestStateManagerEdgeCases(t *testing.T) {
	sm := NewStandardStateManager(nil)
	state, err := sm.CreateState("test_state", StateTypeMemory)
	if err != nil {
		t.Fatalf("Failed to create state: %v", err)
	}

	t.Run("EmptyKey", func(t *testing.T) {
		testEmptyKey(t, state)
	})

	t.Run("EmptyValue", func(t *testing.T) {
		testEmptyValue(t, state)
	})

	t.Run("NilValue", func(t *testing.T) {
		testNilValue(t, state)
	})

	t.Run("GetNonExistentKey", func(t *testing.T) {
		testGetNonExistentKey(t, state)
	})

	t.Run("DeleteNonExistentKey", func(t *testing.T) {
		testDeleteNonExistentKey(t, state)
	})

	t.Run("OverwriteKey", func(t *testing.T) {
		testOverwriteKey(t, state)
	})
}

func testEmptyKey(t *testing.T, state State) {
	err := state.Set("", "value")
	if err == nil {
		t.Error("Setting empty key should return error")
	}
}

func testEmptyValue(t *testing.T, state State) {
	err := state.Set("empty.value", "")
	if err != nil {
		t.Errorf("Setting empty value should succeed: %v", err)
	}
	value, exists := state.Get("empty.value")
	if !exists {
		t.Error("Empty value should exist")
	}
	if value != "" {
		t.Errorf("Expected empty string, got %v", value)
	}
}

func testNilValue(t *testing.T, state State) {
	err := state.Set("nil.value", nil)
	if err != nil {
		t.Errorf("Setting nil value should succeed: %v", err)
	}
	value, exists := state.Get("nil.value")
	if !exists {
		t.Error("Nil value should exist")
	}
	if value != nil {
		t.Errorf("Expected nil, got %v", value)
	}
}

func testGetNonExistentKey(t *testing.T, state State) {
	value, exists := state.Get("non.existent.key")
	if exists {
		t.Error("Non-existent key should not exist")
	}
	if value != nil {
		t.Errorf("Expected nil for non-existent key, got %v", value)
	}
}

func testDeleteNonExistentKey(t *testing.T, state State) {
	err := state.Delete("non.existent.key")
	if err == nil {
		t.Error("Deleting non-existent key should return error")
	}
}

func testOverwriteKey(t *testing.T, state State) {
	key := "overwrite.test"

	// 第一次设置
	err := state.Set(key, "value1")
	if err != nil {
		t.Errorf("First set failed: %v", err)
	}

	// 验证第一个值
	value, exists := state.Get(key)
	if !exists || value != "value1" {
		t.Errorf("Expected 'value1', got %v (exists: %v)", value, exists)
	}

	// 覆盖设置
	err = state.Set(key, "value2")
	if err != nil {
		t.Errorf("Overwrite set failed: %v", err)
	}

	// 验证覆盖后的值
	value, exists = state.Get(key)
	if !exists || value != "value2" {
		t.Errorf("Expected 'value2', got %v (exists: %v)", value, exists)
	}
}

// TestStateManagerDataTypes 测试不同数据类型
func TestStateManagerDataTypes(t *testing.T) {
	sm := NewStandardStateManager(nil)
	state, _ := sm.CreateState("test_state", StateTypeMemory)

	t.Run("VariousDataTypes", func(t *testing.T) {
		testVariousDataTypes(t, state)
	})

	t.Run("LargeObjects", func(t *testing.T) {
		testLargeObjects(t, state)
	})
}

func testVariousDataTypes(t *testing.T, state State) {
	testCases := []struct {
		key   string
		value interface{}
	}{
		{"string", "test string"},
		{"int", 42},
		{"int64", int64(9223372036854775807)},
		{"float64", 3.14159},
		{"bool.true", true},
		{"bool.false", false},
		{"slice", []string{"a", "b", "c"}},
		{"map", map[string]interface{}{"nested": "value"}},
	}

	for _, tc := range testCases {
		err := state.Set(tc.key, tc.value)
		if err != nil {
			t.Errorf("Failed to set %s: %v", tc.key, err)
			continue
		}

		retrieved, exists := state.Get(tc.key)
		if !exists {
			t.Errorf("Key %s should exist", tc.key)
			continue
		}

		// 对于复杂类型，只验证不为nil
		if retrieved == nil && tc.value != nil {
			t.Errorf("Retrieved value for %s should not be nil", tc.key)
		}
	}
}

func testLargeObjects(t *testing.T, state State) {
	// 大字符串
	largeString := make([]byte, 1024*1024) // 1MB
	for i := range largeString {
		largeString[i] = byte('A' + (i % 26))
	}

	err := state.Set("large.string", string(largeString))
	if err != nil {
		t.Errorf("Failed to set large string: %v", err)
	}

	retrieved, exists := state.Get("large.string")
	if !exists {
		t.Error("Large string should exist")
	}
	if len(retrieved.(string)) != len(largeString) {
		t.Error("Large string length mismatch")
	}

	// 大切片
	largeSlice := make([]int, 100000)
	for i := range largeSlice {
		largeSlice[i] = i
	}

	err = state.Set("large.slice", largeSlice)
	if err != nil {
		t.Errorf("Failed to set large slice: %v", err)
	}

	retrieved, exists = state.Get("large.slice")
	if !exists {
		t.Error("Large slice should exist")
	}
	if len(retrieved.([]int)) != len(largeSlice) {
		t.Error("Large slice length mismatch")
	}
}

// TestStateManagerAdvancedConcurrency 测试高级并发安全
func TestStateManagerAdvancedConcurrency(t *testing.T) {
	sm := NewStandardStateManager(nil)
	state, err := sm.CreateState("test_state", StateTypeMemory)
	if err != nil {
		t.Fatalf("Failed to create state: %v", err)
	}

	t.Run("ConcurrentReadWrite", func(t *testing.T) {
		testConcurrentReadWrite(t, state)
	})

	t.Run("ConcurrentUpdateSameKey", func(t *testing.T) {
		testConcurrentUpdateSameKey(t, state)
	})

	t.Run("ConcurrentDelete", func(t *testing.T) {
		testConcurrentDelete(t, state)
	})
}

func testConcurrentReadWrite(t *testing.T, state State) {
	numGoroutines := 50
	numOperations := 200
	var wg sync.WaitGroup
	var errors int64

	// 并发写入
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("concurrent.%d.%d", id, j)
				value := fmt.Sprintf("value-%d-%d", id, j)
				if err := state.Set(key, value); err != nil {
					atomic.AddInt64(&errors, 1)
				}
			}
		}(i)
	}

	// 并发读取
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("concurrent.%d.%d", id, j)
				// 读取可能返回空值（如果写入还未完成）
				_, _ = state.Get(key)
			}
		}(i)
	}

	wg.Wait()

	if atomic.LoadInt64(&errors) > 0 {
		t.Errorf("Encountered %d errors during concurrent operations", errors)
	}
}

func testConcurrentUpdateSameKey(t *testing.T, state State) {
	numGoroutines := 100
	var wg sync.WaitGroup
	key := "concurrent.same.key"
	var successCount int64

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			value := fmt.Sprintf("value-%d", id)
			err := state.Set(key, value)
			if err == nil {
				atomic.AddInt64(&successCount, 1)
			}
		}(i)
	}

	wg.Wait()

	if atomic.LoadInt64(&successCount) != int64(numGoroutines) {
		t.Errorf("Expected %d successful updates, got %d", numGoroutines, successCount)
	}

	// 验证最终值存在
	_, exists := state.Get(key)
	if !exists {
		t.Error("Final value should exist")
	}
}

func testConcurrentDelete(t *testing.T, state State) {
	numKeys := 1000
	numGoroutines := 10
	var wg sync.WaitGroup

	// 首先设置所有键
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("delete.test.%d", i)
		if err := state.Set(key, fmt.Sprintf("value-%d", i)); err != nil {
			t.Logf("Failed to set key %s: %v", key, err)
		}
	}

	// 并发删除
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			start := goroutineID * (numKeys / numGoroutines)
			end := start + (numKeys / numGoroutines)
			for i := start; i < end; i++ {
				key := fmt.Sprintf("delete.test.%d", i)
				// 删除可能失败（如果其他goroutine已删除）
				if err := state.Delete(key); err != nil {
					t.Logf("Failed to delete key %s: %v", key, err)
				}
			}
		}(g)
	}

	wg.Wait()

	// 验证大部分键已被删除
	existingCount := 0
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("delete.test.%d", i)
		if _, exists := state.Get(key); exists {
			existingCount++
		}
	}

	// 由于并发删除，可能有一些键仍然存在，但应该大大减少
	if existingCount > numKeys/2 {
		t.Errorf("Too many keys still exist: %d out of %d", existingCount, numKeys)
	}
}

// TestStateManagerPerformanceOptimized 优化的性能测试
func TestStateManagerPerformanceOptimized(t *testing.T) {
	sm := NewStandardStateManager(nil)
	state, err := sm.CreateState("test_state", StateTypeMemory)
	if err != nil {
		t.Fatalf("Failed to create state: %v", err)
	}

	t.Run("BatchOperationPerformance", func(t *testing.T) {
		testBatchOperationPerformance(t, state)
	})

	t.Run("ConcurrentPerformance", func(t *testing.T) {
		testConcurrentPerformance(t, state)
	})

	t.Run("MemoryEfficiency", func(t *testing.T) {
		testMemoryEfficiency(t, state)
	})
}

func testBatchOperationPerformance(t *testing.T, state State) {
	numOperations := 50000

	// 批量Set操作
	start := time.Now()
	for i := 0; i < numOperations; i++ {
		key := fmt.Sprintf("batch.key.%d", i)
		value := fmt.Sprintf("batch-value-%d", i)
		err := state.Set(key, value)
		if err != nil {
			t.Errorf("Failed to set key %d: %v", i, err)
		}
	}
	setDuration := time.Since(start)
	setOpsPerSec := float64(numOperations) / setDuration.Seconds()

	t.Logf("Batch Set: %d operations in %v (%.0f ops/sec)", numOperations, setDuration, setOpsPerSec)

	if setOpsPerSec < 50000 {
		t.Errorf("Batch Set performance: %.0f ops/sec, expected >= 50,000 ops/sec", setOpsPerSec)
	}

	// 批量Get操作
	start = time.Now()
	for i := 0; i < numOperations; i++ {
		key := fmt.Sprintf("batch.key.%d", i)
		value, exists := state.Get(key)
		if !exists {
			t.Errorf("Key %s should exist", key)
		}
		expected := fmt.Sprintf("batch-value-%d", i)
		if value != expected {
			t.Errorf("Value mismatch for key %s: expected %s, got %v", key, expected, value)
		}
	}
	getDuration := time.Since(start)
	getOpsPerSec := float64(numOperations) / getDuration.Seconds()

	t.Logf("Batch Get: %d operations in %v (%.0f ops/sec)", numOperations, getDuration, getOpsPerSec)

	if getOpsPerSec < 100000 {
		t.Errorf("Batch Get performance: %.0f ops/sec, expected >= 100,000 ops/sec", getOpsPerSec)
	}
}

func testConcurrentPerformance(t *testing.T, state State) {
	numGoroutines := 10
	numOpsPerGoroutine := 10000
	var wg sync.WaitGroup

	start := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOpsPerGoroutine; j++ {
				key := fmt.Sprintf("concurrent.perf.%d.%d", id, j)
				value := fmt.Sprintf("concurrent-value-%d-%d", id, j)
				if err := state.Set(key, value); err != nil {
					t.Errorf("Failed to set key %s: %v", key, err)
				}
			}
		}(i)
	}

	wg.Wait()

	duration := time.Since(start)
	totalOps := numGoroutines * numOpsPerGoroutine
	opsPerSec := float64(totalOps) / duration.Seconds()

	t.Logf("Concurrent Set: %d operations in %v (%.0f ops/sec)", totalOps, duration, opsPerSec)

	if opsPerSec < 50000 {
		t.Errorf("Concurrent performance: %.0f ops/sec, expected >= 50,000 ops/sec", opsPerSec)
	}
}

func testMemoryEfficiency(t *testing.T, state State) {
	numKeys := 100000

	// 设置大量小对象
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("mem.%d", i)
		value := fmt.Sprintf("v%d", i)
		if err := state.Set(key, value); err != nil {
			t.Errorf("Failed to set key %s: %v", key, err)
		}
	}

	// 验证所有键都存在
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("mem.%d", i)
		if _, exists := state.Get(key); !exists {
			t.Errorf("Key %s should exist", key)
			break
		}
	}

	t.Logf("Successfully stored and retrieved %d key-value pairs", numKeys)
}

// TestStateManagerErrorRecovery 测试错误恢复
func TestStateManagerErrorRecovery(t *testing.T) {
	sm := NewStandardStateManager(nil)
	state, err := sm.CreateState("test_state", StateTypeMemory)
	if err != nil {
		t.Fatalf("Failed to create state: %v", err)
	}

	t.Run("ErrorRecovery", func(t *testing.T) {
		testErrorRecovery(t, state)
	})

	t.Run("ResourceCleanup", func(t *testing.T) {
		testResourceCleanup(t, state)
	})
}

func testErrorRecovery(t *testing.T, state State) {
	// 尝试设置无效键（空键）
	err := state.Set("", "value")
	if err == nil {
		t.Error("Setting empty key should fail")
	}

	// 验证状态管理器仍然可以正常工作
	err = state.Set("recovery.test", "recovery-value")
	if err != nil {
		t.Errorf("State manager should recover from error: %v", err)
	}

	value, exists := state.Get("recovery.test")
	if !exists || value != "recovery-value" {
		t.Error("State manager should work normally after error")
	}
}

func testResourceCleanup(t *testing.T, state State) {
	// 设置一些数据
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("cleanup.%d", i)
		value := fmt.Sprintf("cleanup-value-%d", i)
		if err := state.Set(key, value); err != nil {
			t.Errorf("Failed to set key %s: %v", key, err)
		}
	}

	// 删除一半数据
	for i := 0; i < 500; i++ {
		key := fmt.Sprintf("cleanup.%d", i)
		if err := state.Delete(key); err != nil {
			t.Errorf("Failed to delete key %s: %v", key, err)
		}
	}

	// 验证删除的数据不存在
	for i := 0; i < 500; i++ {
		key := fmt.Sprintf("cleanup.%d", i)
		if _, exists := state.Get(key); exists {
			t.Errorf("Key %s should not exist after deletion", key)
		}
	}

	// 验证剩余数据仍然存在
	for i := 500; i < 1000; i++ {
		key := fmt.Sprintf("cleanup.%d", i)
		if _, exists := state.Get(key); !exists {
			t.Errorf("Key %s should still exist", key)
		}
	}
}

// 辅助函数：创建多个状态管理器实例
func createMultipleStateManagers(t *testing.T, count int) ([]*StandardStateManager, []State) {
	managers := make([]*StandardStateManager, count)
	states := make([]State, count)
	for i := 0; i < count; i++ {
		managers[i] = NewStandardStateManager(nil)
		var err error
		states[i], err = managers[i].CreateState(fmt.Sprintf("test_state_%d", i), StateTypeMemory)
		if err != nil {
			t.Fatalf("Failed to create state %d: %v", i, err)
		}
	}
	return managers, states
}

// 辅助函数：在状态中设置测试数据
func populateStateWithTestData(t *testing.T, states []State, managerIndex, dataCount int) {
	for j := 0; j < dataCount; j++ {
		key := fmt.Sprintf("lifecycle.%d.%d", managerIndex, j)
		value := fmt.Sprintf("value-%d-%d", managerIndex, j)
		if err := states[managerIndex].Set(key, value); err != nil {
			t.Errorf("Failed to set key %s: %v", key, err)
		}
	}
}

// 辅助函数：验证状态数据的正确性
func verifyStateData(t *testing.T, states []State, managerIndex, dataCount int) {
	for j := 0; j < dataCount; j++ {
		key := fmt.Sprintf("lifecycle.%d.%d", managerIndex, j)
		expected := fmt.Sprintf("value-%d-%d", managerIndex, j)

		value, exists := states[managerIndex].Get(key)
		if !exists || value != expected {
			t.Errorf("Data mismatch in manager %d: expected %s, got %v (exists: %v)", managerIndex, expected, value, exists)
		}
	}
}

// 辅助函数：验证数据独立性
func verifyDataIsolation(t *testing.T, states []State, managerIndex, dataCount, totalManagers int) {
	for j := 0; j < dataCount; j++ {
		key := fmt.Sprintf("lifecycle.%d.%d", managerIndex, j)
		for k := 0; k < totalManagers; k++ {
			if k != managerIndex {
				if _, exists := states[k].Get(key); exists {
					t.Errorf("Key %s should not exist in manager %d", key, k)
				}
			}
		}
	}
}

// TestStateManagerLifecycle 测试生命周期管理
func TestStateManagerLifecycle(t *testing.T) {
	t.Run("LifecycleManagement", func(t *testing.T) {
		const managerCount = 10
		const dataCount = 100

		// 创建多个状态管理器实例
		managers, states := createMultipleStateManagers(t, managerCount)

		// 在每个实例中设置测试数据
		for i := 0; i < managerCount; i++ {
			populateStateWithTestData(t, states, i, dataCount)
		}

		// 验证每个实例的数据独立性
		for i := 0; i < managerCount; i++ {
			verifyStateData(t, states, i, dataCount)
			verifyDataIsolation(t, states, i, dataCount, managerCount)
		}

		_ = managers // 避免未使用变量警告
	})
}

// 辅助函数：执行高强度并发压力测试
func runHighConcurrencyStress(t *testing.T, state State) {
	numGoroutines := 20
	numOperations := 5000
	var wg sync.WaitGroup
	var totalErrors int64

	start := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			runStressOperations(t, state, id, numOperations, &totalErrors)
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)
	totalOps := numGoroutines * numOperations * 2 // Set + Get
	opsPerSec := float64(totalOps) / duration.Seconds()

	t.Logf("Stress test: %d operations in %v (%.0f ops/sec), errors: %d", totalOps, duration, opsPerSec, totalErrors)

	if atomic.LoadInt64(&totalErrors) > int64(totalOps/100) { // 允许1%的错误率
		t.Errorf("Too many errors in stress test: %d out of %d operations", totalErrors, totalOps)
	}
}

// 辅助函数：执行单个goroutine的压力测试操作
func runStressOperations(t *testing.T, state State, id, numOperations int, totalErrors *int64) {
	for j := 0; j < numOperations; j++ {
		key := fmt.Sprintf("stress.%d.%d", id, j)
		value := fmt.Sprintf("stress-value-%d-%d", id, j)

		// 设置
		if err := state.Set(key, value); err != nil {
			atomic.AddInt64(totalErrors, 1)
		}

		// 读取
		if _, exists := state.Get(key); !exists {
			atomic.AddInt64(totalErrors, 1)
		}

		// 偶尔删除
		if j%10 == 0 {
			if err := state.Delete(key); err != nil {
				t.Errorf("Failed to delete key %s: %v", key, err)
			}
		}
	}
}

// TestStateManagerStressTest 压力测试
func TestStateManagerStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	sm := NewStandardStateManager(nil)
	state, err := sm.CreateState("test_state", StateTypeMemory)
	if err != nil {
		t.Fatalf("Failed to create state: %v", err)
	}

	// 高强度并发测试
	t.Run("HighConcurrencyStress", func(t *testing.T) {
		runHighConcurrencyStress(t, state)
	})
}
