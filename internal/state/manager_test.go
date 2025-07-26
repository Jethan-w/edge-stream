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
	"testing"
	"time"
)

// MemoryCheckpointManager 内存检查点管理器（仅用于测试）
type MemoryCheckpointManager struct {
	mu          sync.RWMutex
	checkpoints map[string]map[string]interface{}
}

// CreateCheckpoint 创建检查点
func (m *MemoryCheckpointManager) CreateCheckpoint(processorID string, data map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checkpoints[processorID] = data
	return nil
}

// RestoreCheckpoint 恢复检查点
func (m *MemoryCheckpointManager) RestoreCheckpoint(processorID string) (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if data, exists := m.checkpoints[processorID]; exists {
		return data, nil
	}
	return nil, fmt.Errorf("checkpoint not found for processor %s", processorID)
}

// ListCheckpoints 列出检查点
func (m *MemoryCheckpointManager) ListCheckpoints() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var keys []string
	for key := range m.checkpoints {
		keys = append(keys, key)
	}
	return keys
}

// DeleteCheckpoint 删除检查点
func (m *MemoryCheckpointManager) DeleteCheckpoint(processorID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.checkpoints[processorID]; !exists {
		return fmt.Errorf("checkpoint not found for processor %s", processorID)
	}
	delete(m.checkpoints, processorID)
	return nil
}

func TestMemoryStateManager(t *testing.T) {
	sm := NewStandardStateManager(nil)

	t.Run("BasicOperations", func(t *testing.T) {
		testBasicOperations(t, sm)
	})

	t.Run("NonExistentKey", func(t *testing.T) {
		testNonExistentKey(t, sm)
	})

	t.Run("ClearAll", func(t *testing.T) {
		testClearAll(t, sm)
	})
}

func testBasicOperations(t *testing.T, sm StateManager) {
	key := "test_key"
	value := "test_value"

	// 测试设置状态
	state, err := sm.CreateState("test_state", StateTypeMemory)
	if err != nil {
		t.Errorf("CreateState failed: %v", err)
	}
	err = state.Set(key, value)
	if err != nil {
		t.Errorf("Set failed: %v", err)
	}

	// 测试获取状态
	retrievedValue, exists := state.Get(key)
	if !exists {
		t.Errorf("Get failed: key not found")
	}
	if retrievedValue != value {
		t.Errorf("Expected %s, got %s", value, retrievedValue)
	}

	// 测试状态存在性检查
	exists = state.Exists(key)
	if !exists {
		t.Error("Expected state to exist")
	}

	// 测试删除状态
	err = state.Delete(key)
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}

	// 验证状态已删除
	exists = state.Exists(key)
	if exists {
		t.Error("Expected state to be deleted")
	}
}

func testNonExistentKey(t *testing.T, sm StateManager) {
	state, _ := sm.CreateState("test_state2", StateTypeMemory)
	_, exists := state.Get("nonexistent")
	if exists {
		t.Error("Expected false for nonexistent key")
	}

	exists = state.Exists("nonexistent")
	if exists {
		t.Error("Expected nonexistent key to return false")
	}
}

func testClearAll(t *testing.T, sm StateManager) {
	state, _ := sm.CreateState("test_state3", StateTypeMemory)
	// 添加一些状态
	if err := state.Set("key1", "value1"); err != nil {
		t.Errorf("Failed to set key1: %v", err)
	}
	if err := state.Set("key2", "value2"); err != nil {
		t.Errorf("Failed to set key2: %v", err)
	}
	if err := state.Set("key3", "value3"); err != nil {
		t.Errorf("Failed to set key3: %v", err)
	}

	// 清空所有状态
	err := state.Clear()
	if err != nil {
		t.Errorf("Clear failed: %v", err)
	}

	// 验证所有状态都被清空
	if state.Exists("key1") || state.Exists("key2") || state.Exists("key3") {
		t.Error("Expected all states to be cleared")
	}
}

// 辅助函数：并发写入操作
func runConcurrentWrites(t *testing.T, sm StateManager, numGoroutines, numOperations int) {
	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key_%d_%d", id, j)
				value := fmt.Sprintf("value_%d_%d", id, j)
				state, _ := sm.CreateState(fmt.Sprintf("state_%d_%d", id, j), StateTypeMemory)
				if err := state.Set(key, value); err != nil {
					t.Errorf("Failed to set key %s: %v", key, err)
				}
			}
		}(i)
	}
	wg.Wait()
}

// 辅助函数：并发读取操作
func runConcurrentReads(t *testing.T, sm StateManager, numGoroutines, numOperations int) {
	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key_%d_%d", id, j)
				if state, exists := sm.GetState(fmt.Sprintf("state_%d_%d", id, j)); exists {
					_, _ = state.Get(key)
					_ = state.Exists(key)
				}
			}
		}(i)
	}
	wg.Wait()
}

// 辅助函数：设置删除测试的初始状态
func setupDeleteTestStates(t *testing.T, sm StateManager, count int) []State {
	states := make([]State, count)
	for i := 0; i < count; i++ {
		key := fmt.Sprintf("delete_key_%d", i)
		state, _ := sm.CreateState(fmt.Sprintf("delete_state_%d", i), StateTypeMemory)
		if err := state.Set(key, "value"); err != nil {
			t.Errorf("Failed to set key %s: %v", key, err)
		}
		states[i] = state
	}
	return states
}

// 辅助函数：并发删除操作
func runConcurrentDeletes(t *testing.T, sm StateManager, count int) {
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("delete_key_%d", id)
			if state, exists := sm.GetState(fmt.Sprintf("delete_state_%d", id)); exists {
				if err := state.Delete(key); err != nil {
					t.Logf("Failed to delete key %s: %v", key, err)
				}
			}
		}(i)
	}
	wg.Wait()
}

// 辅助函数：验证删除结果
func verifyDeletions(t *testing.T, sm StateManager, count int) {
	for i := 0; i < count; i++ {
		key := fmt.Sprintf("delete_key_%d", i)
		if state, exists := sm.GetState(fmt.Sprintf("delete_state_%d", i)); exists {
			if state.Exists(key) {
				t.Errorf("Expected key %s to be deleted", key)
			}
		}
	}
}

func TestStateManagerConcurrency(t *testing.T) {
	sm := NewStandardStateManager(nil)

	// 并发读写测试
	t.Run("ConcurrentReadWrite", func(t *testing.T) {
		numGoroutines := 10
		numOperations := 100
		runConcurrentWrites(t, sm, numGoroutines, numOperations)
		runConcurrentReads(t, sm, numGoroutines, numOperations)
	})

	// 并发删除测试
	t.Run("ConcurrentDelete", func(t *testing.T) {
		count := 100
		_ = setupDeleteTestStates(t, sm, count)
		runConcurrentDeletes(t, sm, count)
		verifyDeletions(t, sm, count)
	})
}

func TestCheckpointManager(t *testing.T) {
	// 创建一个简单的内存检查点管理器用于测试
	manager := &MemoryCheckpointManager{
		checkpoints: make(map[string]map[string]interface{}),
	}

	// 测试检查点创建和恢复
	t.Run("CreateAndRestore", func(t *testing.T) {
		processorID := "test_processor"
		checkpointData := map[string]interface{}{
			"processed_count": 100,
			"last_timestamp":  time.Now().Unix(),
			"status":          "running",
		}

		// 创建检查点
		err := manager.CreateCheckpoint(processorID, checkpointData)
		if err != nil {
			t.Errorf("CreateCheckpoint failed: %v", err)
		}

		// 恢复检查点
		restoredData, err := manager.RestoreCheckpoint(processorID)
		if err != nil {
			t.Errorf("RestoreCheckpoint failed: %v", err)
		}

		// 验证数据
		if restoredData["processed_count"] != checkpointData["processed_count"] {
			t.Error("Checkpoint data mismatch")
		}
		if restoredData["status"] != checkpointData["status"] {
			t.Error("Checkpoint status mismatch")
		}
	})

	// 测试检查点列表
	t.Run("ListCheckpoints", func(t *testing.T) {
		// 创建多个检查点
		for i := 0; i < 5; i++ {
			processorID := fmt.Sprintf("processor_%d", i)
			checkpointData := map[string]interface{}{"id": i}
			err := manager.CreateCheckpoint(processorID, checkpointData)
			if err != nil {
				t.Errorf("CreateCheckpoint failed: %v", err)
			}
		}

		checkpoints := manager.ListCheckpoints()
		if len(checkpoints) < 5 {
			t.Errorf("Expected at least 5 checkpoints, got %d", len(checkpoints))
		}
	})

	// 测试检查点删除
	t.Run("DeleteCheckpoint", func(t *testing.T) {
		processorID := "delete_test_processor"
		checkpointData := map[string]interface{}{"test": "data"}

		// 创建检查点
		err := manager.CreateCheckpoint(processorID, checkpointData)
		if err != nil {
			t.Errorf("CreateCheckpoint failed: %v", err)
		}

		// 验证检查点存在
		_, err = manager.RestoreCheckpoint(processorID)
		if err != nil {
			t.Error("Checkpoint should exist before deletion")
		}

		// 删除检查点
		err = manager.DeleteCheckpoint(processorID)
		if err != nil {
			t.Errorf("DeleteCheckpoint failed: %v", err)
		}

		// 验证检查点已删除
		_, err = manager.RestoreCheckpoint(processorID)
		if err == nil {
			t.Error("Expected error when restoring deleted checkpoint")
		}
	})
}

// 辅助函数：基准测试Set操作
func benchmarkSetState(b *testing.B, state State) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bench_key_%d", i)
		if err := state.Set(key, "benchmark_value"); err != nil {
			b.Errorf("Failed to set key %s: %v", key, err)
		}
	}
}

// 辅助函数：为Get基准测试准备数据
func prepareGetBenchmarkData(b *testing.B, state State) {
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("get_bench_key_%d", i)
		if err := state.Set(key, "value"); err != nil {
			b.Errorf("Failed to set key %s: %v", key, err)
		}
	}
}

// 辅助函数：基准测试Get操作
func benchmarkGetState(b *testing.B, state State) {
	prepareGetBenchmarkData(b, state)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("get_bench_key_%d", i%1000)
		_, _ = state.Get(key)
	}
}

// 辅助函数：为Exists基准测试准备数据
func prepareExistsBenchmarkData(b *testing.B, state State) {
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("has_bench_key_%d", i)
		if err := state.Set(key, "value"); err != nil {
			b.Errorf("Failed to set key %s: %v", key, err)
		}
	}
}

// 辅助函数：基准测试Exists操作
func benchmarkExistsState(b *testing.B, state State) {
	prepareExistsBenchmarkData(b, state)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("has_bench_key_%d", i%1000)
		state.Exists(key)
	}
}

func BenchmarkStateManager(b *testing.B) {
	sm := NewStandardStateManager(nil)
	state, _ := sm.CreateState("bench_state", StateTypeMemory)

	b.Run("SetState", func(b *testing.B) {
		benchmarkSetState(b, state)
	})

	b.Run("GetState", func(b *testing.B) {
		benchmarkGetState(b, state)
	})

	b.Run("HasState", func(b *testing.B) {
		benchmarkExistsState(b, state)
	})
}

// 辅助函数：测试单次Set操作性能
func testSingleSetPerformance(t *testing.T, state State) {
	start := time.Now()
	if err := state.Set("perf_test", "value"); err != nil {
		t.Errorf("Failed to set key: %v", err)
	}
	duration := time.Since(start)

	// 单次操作应该在100μs内完成
	if duration > 100*time.Microsecond {
		t.Errorf("Single Set took %v, expected < 100μs", duration)
	}
}

// 辅助函数：测试单次Get操作性能
func testSingleGetPerformance(t *testing.T, state State) {
	start := time.Now()
	_, _ = state.Get("perf_test")
	duration := time.Since(start)

	// 单次获取应该在50μs内完成
	if duration > 50*time.Microsecond {
		t.Errorf("Single Get took %v, expected < 50μs", duration)
	}
}

// 性能标准测试
func TestStateManagerPerformanceStandards(t *testing.T) {
	sm := NewStandardStateManager(nil)
	state, _ := sm.CreateState("perf_state", StateTypeMemory)

	// 测试单次操作性能
	t.Run("SingleOperationPerformance", func(t *testing.T) {
		testSingleSetPerformance(t, state)
		testSingleGetPerformance(t, state)
	})

	// 测试批量操作性能
	t.Run("BatchOperationPerformance", func(t *testing.T) {
		testBatchOperationPerformance(t, state)
	})

	// 测试内存使用效率
	t.Run("MemoryEfficiency", func(t *testing.T) {
		testMemoryEfficiency(t, state)
	})
}
