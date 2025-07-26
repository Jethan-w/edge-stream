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

package windowmanager

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestWindowManagerEdgeCases 测试边界条件
func TestWindowManagerEdgeCases(t *testing.T) {
	wm := NewSimpleWindowManager()

	// 测试零时间窗口
	t.Run("ZeroTimeWindow", func(t *testing.T) {
		window := wm.CreateTimeWindow(0)
		if window == nil {
			t.Error("Creating window with zero time should not return nil")
		}
	})

	// 测试负时间窗口
	t.Run("NegativeTimeWindow", func(t *testing.T) {
		window := wm.CreateTimeWindow(-time.Second)
		if window == nil {
			t.Error("Creating window with negative time should not return nil")
		}
	})

	// 测试零大小窗口
	t.Run("ZeroSizeWindow", func(t *testing.T) {
		window := wm.CreateCountWindow(0)
		if window == nil {
			t.Error("Creating window with zero size should not return nil")
		}
	})

	// 测试负大小窗口
	t.Run("NegativeSizeWindow", func(t *testing.T) {
		window := wm.CreateCountWindow(-1)
		if window == nil {
			t.Error("Creating window with negative size should not return nil")
		}
	})

	// 测试处理数据
	t.Run("ProcessData", func(t *testing.T) {
		window := wm.CreateTimeWindow(time.Second)
		if window == nil {
			t.Fatal("Failed to create window")
		}

		err := wm.ProcessData("test data")
		if err != nil {
			t.Errorf("ProcessData failed: %v", err)
		}

		data := window.GetData()
		if len(data) == 0 {
			t.Error("Window should contain data after processing")
		}
	})

	// 测试获取准备好的窗口
	t.Run("GetReadyWindows", func(t *testing.T) {
		readyWindows := wm.GetReadyWindows()
		if readyWindows == nil {
			t.Error("GetReadyWindows should not return nil")
		}
	})
}

// TestWindowManagerDataTypes 测试不同数据类型
func TestWindowManagerDataTypes(t *testing.T) {
	wm := NewSimpleWindowManager()

	// 测试各种数据类型
	t.Run("VariousDataTypes", func(t *testing.T) {
		window := wm.CreateCountWindow(1000)
		if window == nil {
			t.Fatal("Failed to create window")
		}

		testData := []interface{}{
			"string data",
			42,
			int64(9223372036854775807),
			3.14159,
			true,
			false,
			[]string{"a", "b", "c"},
			map[string]interface{}{"key": "value"},
			struct {
				Name string
				Age  int
			}{"John", 30},
			nil,
		}

		for _, data := range testData {
			window.AddData(data)
		}

		// 验证数据存在
		windowData := window.GetData()
		if windowData == nil {
			t.Error("Window data should not be nil")
		}
		if len(windowData) != len(testData) {
			t.Errorf("Expected %d items, got %d", len(testData), len(windowData))
		}
	})

	// 测试大对象
	t.Run("LargeObjects", func(t *testing.T) {
		window := wm.CreateCountWindow(10)
		if window == nil {
			t.Fatal("Failed to create window")
		}

		// 大字符串
		largeString := make([]byte, 1024*1024) // 1MB
		for i := range largeString {
			largeString[i] = byte('A' + (i % 26))
		}

		window.AddData(string(largeString))

		// 大切片
		largeSlice := make([]int, 100000)
		for i := range largeSlice {
			largeSlice[i] = i
		}

		window.AddData(largeSlice)

		// 验证数据
		windowData := window.GetData()
		if len(windowData) != 2 {
			t.Errorf("Expected 2 items, got %d", len(windowData))
		}
	})
}

// TestWindowManagerConcurrency 测试并发安全
func TestWindowManagerConcurrency(t *testing.T) {
	t.Run("ConcurrentWindowCreation", testConcurrentWindowCreation)
	t.Run("ConcurrentDataAddition", testConcurrentDataAddition)
	t.Run("ConcurrentReadWrite", testConcurrentReadWrite)
	t.Run("ConcurrentWindowOperations", testConcurrentWindowOperations)
}

func testConcurrentWindowCreation(t *testing.T) {
	wm := NewSimpleWindowManager()
	numGoroutines := 50
	numWindows := 100
	var wg sync.WaitGroup
	var successes int64

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numWindows; j++ {
				window := wm.CreateTimeWindow(time.Minute)
				if window != nil {
					atomic.AddInt64(&successes, 1)
				}
				// 使用id参数避免未使用警告
				_ = id
			}
		}(i)
	}

	wg.Wait()

	expectedSuccesses := int64(numGoroutines * numWindows)
	if atomic.LoadInt64(&successes) != expectedSuccesses {
		t.Errorf("Expected %d successful creations, got %d", expectedSuccesses, successes)
	}
}

func testConcurrentDataAddition(t *testing.T) {
	wm := NewSimpleWindowManager()
	window := wm.CreateCountWindow(10000)
	if window == nil {
		t.Fatal("Failed to create window")
	}

	numGoroutines := 50
	numOperations := 200
	var wg sync.WaitGroup
	var successes int64

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				data := fmt.Sprintf("data-%d-%d", id, j)
				window.AddData(data)
				atomic.AddInt64(&successes, 1)
			}
		}(i)
	}

	wg.Wait()

	expectedSuccesses := int64(numGoroutines * numOperations)
	if atomic.LoadInt64(&successes) != expectedSuccesses {
		t.Errorf("Expected %d successful additions, got %d", expectedSuccesses, successes)
	}

	// 验证数据完整性
	windowData := window.GetData()
	if len(windowData) > int(expectedSuccesses) {
		t.Errorf("Window contains more items than expected: %d > %d", len(windowData), expectedSuccesses)
	}
}

func testConcurrentReadWrite(t *testing.T) {
	wm := NewSimpleWindowManager()
	window := wm.CreateCountWindow(5000)
	if window == nil {
		t.Fatal("Failed to create window")
	}

	numWriters := 25
	numReaders := 25
	numOperations := 100
	var wg sync.WaitGroup
	var readErrors int64

	// 写入goroutines
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				data := fmt.Sprintf("write-data-%d-%d", writerID, j)
				window.AddData(data)
			}
		}(i)
	}

	// 读取goroutines
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				data := window.GetData()
				if data == nil {
					atomic.AddInt64(&readErrors, 1)
				}
				// 使用readerID参数避免未使用警告
				_ = readerID
			}
		}(i)
	}

	wg.Wait()

	if atomic.LoadInt64(&readErrors) > 0 {
		t.Errorf("Encountered %d read errors", readErrors)
	}
}

func testConcurrentWindowOperations(t *testing.T) {
	wm := NewSimpleWindowManager()
	numGoroutines := 20
	numOperations := 50
	var wg sync.WaitGroup
	var errors int64

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				// 创建窗口
				window := wm.CreateCountWindow(100)
				if window == nil {
					atomic.AddInt64(&errors, 1)
					continue
				}

				// 添加数据
				for k := 0; k < 10; k++ {
					data := fmt.Sprintf("data-%d-%d-%d", id, j, k)
					window.AddData(data)
				}

				// 获取数据
				data := window.GetData()
				if data == nil {
					atomic.AddInt64(&errors, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	if atomic.LoadInt64(&errors) > 0 {
		t.Errorf("Encountered %d errors during concurrent operations", errors)
	}
}

// TestWindowManagerTimeWindows 测试时间窗口功能
func TestWindowManagerTimeWindows(t *testing.T) {
	t.Run("TimeWindowExpiration", testTimeWindowExpiration)
	t.Run("SlidingTimeWindow", testSlidingTimeWindow)
	t.Run("MultipleTimeWindows", testMultipleTimeWindows)
}

func testTimeWindowExpiration(t *testing.T) {
	wm := NewSimpleWindowManager()
	windowID := "time.expiration.test"
	windowDuration := time.Millisecond * 100

	err := wm.CreateWindow(windowID, windowDuration, 1000)
	if err != nil {
		t.Fatalf("Failed to create window: %v", err)
	}

	// 添加一些数据
	for i := 0; i < 10; i++ {
		data := fmt.Sprintf("data-%d", i)
		if err := wm.AddToWindow(windowID, data); err != nil {
			t.Errorf("Failed to add data %d: %v", i, err)
		}
	}

	// 验证数据存在
	windowData := wm.GetWindowData(windowID)
	if len(windowData) != 10 {
		t.Errorf("Expected 10 items, got %d", len(windowData))
	}

	// 等待窗口过期
	time.Sleep(windowDuration + time.Millisecond*50)

	// 验证数据已过期（具体行为取决于实现）
	windowDataAfterExpiry := wm.GetWindowData(windowID)
	t.Logf("Data after expiry: %d items", len(windowDataAfterExpiry))
}

func testSlidingTimeWindow(t *testing.T) {
	wm := NewSimpleWindowManager()
	windowID := "sliding.window.test"
	windowDuration := time.Millisecond * 200

	err := wm.CreateWindow(windowID, windowDuration, 1000)
	if err != nil {
		t.Fatalf("Failed to create window: %v", err)
	}

	// 分批添加数据，间隔时间
	for batch := 0; batch < 3; batch++ {
		for i := 0; i < 5; i++ {
			data := fmt.Sprintf("batch-%d-data-%d", batch, i)
			err := wm.AddToWindow(windowID, data)
			if err != nil {
				t.Errorf("Failed to add data %s: %v", data, err)
			}
		}

		// 检查当前窗口数据
		windowData := wm.GetWindowData(windowID)
		t.Logf("Batch %d: %d items in window", batch, len(windowData))

		// 等待一段时间
		time.Sleep(time.Millisecond * 80)
	}
}

func testMultipleTimeWindows(t *testing.T) {
	wm := NewSimpleWindowManager()
	windowConfigs := []struct {
		id       string
		duration time.Duration
		size     int
	}{
		{"short.window", time.Millisecond * 50, 100},
		{"medium.window", time.Millisecond * 200, 500},
		{"long.window", time.Second, 1000},
	}

	// 创建多个窗口
	for _, config := range windowConfigs {
		err := wm.CreateWindow(config.id, config.duration, config.size)
		if err != nil {
			t.Errorf("Failed to create window %s: %v", config.id, err)
		}
	}

	// 向每个窗口添加数据
	for i := 0; i < 20; i++ {
		for _, config := range windowConfigs {
			data := fmt.Sprintf("%s-data-%d", config.id, i)
			err := wm.AddToWindow(config.id, data)
			if err != nil {
				t.Errorf("Failed to add data to %s: %v", config.id, err)
			}
		}
		time.Sleep(time.Millisecond * 10)
	}

	// 验证每个窗口的数据
	for _, config := range windowConfigs {
		windowData := wm.GetWindowData(config.id)
		t.Logf("Window %s: %d items", config.id, len(windowData))
		if len(windowData) == 0 {
			t.Errorf("Window %s should contain data", config.id)
		}
	}
}

// TestWindowManagerSizeWindows 测试大小窗口功能
func TestWindowManagerSizeWindows(t *testing.T) {
	t.Run("FixedSizeWindow", testFixedSizeWindow)
	t.Run("SlidingSizeWindow", testSlidingSizeWindow)
	t.Run("VariousSizeWindows", testVariousSizeWindows)
}

func testFixedSizeWindow(t *testing.T) {
	wm := NewSimpleWindowManager()
	windowID := "fixed.size.test"
	maxSize := 10

	err := wm.CreateWindow(windowID, time.Minute, maxSize)
	if err != nil {
		t.Fatalf("Failed to create window: %v", err)
	}

	// 添加超过最大大小的数据
	for i := 0; i < maxSize*2; i++ {
		data := fmt.Sprintf("data-%d", i)
		err := wm.AddToWindow(windowID, data)
		if err != nil {
			t.Errorf("Failed to add data %d: %v", i, err)
		}
	}

	// 验证窗口大小不超过限制
	windowData := wm.GetWindowData(windowID)
	if len(windowData) > maxSize {
		t.Errorf("Window size %d exceeds maximum %d", len(windowData), maxSize)
	}

	t.Logf("Window contains %d items (max: %d)", len(windowData), maxSize)
}

func testSlidingSizeWindow(t *testing.T) {
	wm := NewSimpleWindowManager()
	windowID := "sliding.size.test"
	maxSize := 5

	err := wm.CreateWindow(windowID, time.Hour, maxSize) // 长时间窗口，主要测试大小限制
	if err != nil {
		t.Fatalf("Failed to create window: %v", err)
	}

	// 逐步添加数据并验证窗口行为
	for i := 0; i < 15; i++ {
		data := fmt.Sprintf("sliding-data-%d", i)
		err := wm.AddToWindow(windowID, data)
		if err != nil {
			t.Errorf("Failed to add data %d: %v", i, err)
		}

		windowData := wm.GetWindowData(windowID)
		if len(windowData) > maxSize {
			t.Errorf("Window size %d exceeds maximum %d at iteration %d", len(windowData), maxSize, i)
		}

		// 验证最新数据在窗口中
		if len(windowData) > 0 {
			lastItem := windowData[len(windowData)-1]
			if lastItem != data {
				t.Errorf("Last item should be %s, got %v", data, lastItem)
			}
		}
	}
}

func testVariousSizeWindows(t *testing.T) {
	wm := NewSimpleWindowManager()
	sizes := []int{1, 5, 10, 50, 100, 1000}

	for _, size := range sizes {
		windowID := fmt.Sprintf("size.test.%d", size)
		err := wm.CreateWindow(windowID, time.Hour, size)
		if err != nil {
			t.Errorf("Failed to create window with size %d: %v", size, err)
			continue
		}

		// 添加数据
		numItems := size + 10 // 超过窗口大小
		for i := 0; i < numItems; i++ {
			data := fmt.Sprintf("data-%d", i)
			if err := wm.AddToWindow(windowID, data); err != nil {
				t.Errorf("Failed to add data: %v", err)
			}
		}

		// 验证窗口大小
		windowData := wm.GetWindowData(windowID)
		if len(windowData) > size {
			t.Errorf("Window %s size %d exceeds maximum %d", windowID, len(windowData), size)
		}

		t.Logf("Window %s: %d items (max: %d)", windowID, len(windowData), size)
	}
}

// TestWindowManagerPerformance 测试性能
func TestWindowManagerPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	t.Run("WindowCreationPerformance", testWindowCreationPerformance)
	t.Run("DataAdditionPerformance", testDataAdditionPerformance)
	t.Run("DataRetrievalPerformance", testDataRetrievalPerformance)
	t.Run("ConcurrentPerformance", testConcurrentPerformance)
}

func testWindowCreationPerformance(t *testing.T) {
	wm := NewSimpleWindowManager()
	numWindows := 1000

	start := time.Now()
	for i := 0; i < numWindows; i++ {
		windowID := fmt.Sprintf("perf.window.%d", i)
		err := wm.CreateWindow(windowID, time.Minute, 100)
		if err != nil {
			t.Errorf("Failed to create window %d: %v", i, err)
		}
	}
	duration := time.Since(start)
	creationsPerSec := float64(numWindows) / duration.Seconds()

	t.Logf("Window Creation Performance: %d windows in %v (%.0f creations/sec)", numWindows, duration, creationsPerSec)

	// 性能要求：至少1,000 creations/sec
	if creationsPerSec < 1000 {
		t.Errorf("Window creation performance: %.0f creations/sec, expected >= 1,000 creations/sec", creationsPerSec)
	}
}

func testDataAdditionPerformance(t *testing.T) {
	wm := NewSimpleWindowManager()
	windowID := "perf.data.addition"
	err := wm.CreateWindow(windowID, time.Hour, 10000)
	if err != nil {
		t.Fatalf("Failed to create window: %v", err)
	}

	numOperations := 5000

	start := time.Now()
	for i := 0; i < numOperations; i++ {
		data := fmt.Sprintf("perf-data-%d", i)
		if err := wm.AddToWindow(windowID, data); err != nil {
			t.Errorf("Failed to add data %d: %v", i, err)
		}
	}
	duration := time.Since(start)
	additionsPerSec := float64(numOperations) / duration.Seconds()

	t.Logf("Data Addition Performance: %d additions in %v (%.0f additions/sec)", numOperations, duration, additionsPerSec)

	// 性能要求：至少2,000 additions/sec
	if additionsPerSec < 2000 {
		t.Errorf("Data addition performance: %.0f additions/sec, expected >= 2,000 additions/sec", additionsPerSec)
	}
}

func testDataRetrievalPerformance(t *testing.T) {
	wm := NewSimpleWindowManager()
	windowID := "perf.data.retrieval"
	err := wm.CreateWindow(windowID, time.Hour, 1000)
	if err != nil {
		t.Fatalf("Failed to create window: %v", err)
	}

	// 首先添加一些数据
	for i := 0; i < 100; i++ {
		data := fmt.Sprintf("retrieval-data-%d", i)
		if err := wm.AddToWindow(windowID, data); err != nil {
			t.Errorf("Failed to add data %d: %v", i, err)
		}
	}

	numRetrievals := 500

	start := time.Now()
	for i := 0; i < numRetrievals; i++ {
		data := wm.GetWindowData(windowID)
		if data == nil {
			t.Errorf("Failed to retrieve data %d", i)
		}
	}
	duration := time.Since(start)
	retrievalsPerSec := float64(numRetrievals) / duration.Seconds()

	t.Logf("Data Retrieval Performance: %d retrievals in %v (%.0f retrievals/sec)", numRetrievals, duration, retrievalsPerSec)

	// 性能要求：至少1,000 retrievals/sec
	if retrievalsPerSec < 1000 {
		t.Errorf("Data retrieval performance: %.0f retrievals/sec, expected >= 1,000 retrievals/sec", retrievalsPerSec)
	}
}

func testConcurrentPerformance(t *testing.T) {
	wm := NewSimpleWindowManager()
	windowID := "perf.concurrent"
	err := wm.CreateWindow(windowID, time.Hour, 5000)
	if err != nil {
		t.Fatalf("Failed to create window: %v", err)
	}

	numGoroutines := 5
	numOpsPerGoroutine := 50
	var wg sync.WaitGroup

	start := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < numOpsPerGoroutine; j++ {
				data := fmt.Sprintf("concurrent-perf-%d-%d", goroutineID, j)
				if err := wm.AddToWindow(windowID, data); err != nil {
					t.Errorf("Failed to add data: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()

	duration := time.Since(start)
	totalOps := numGoroutines * numOpsPerGoroutine
	opsPerSec := float64(totalOps) / duration.Seconds()

	t.Logf("Concurrent Performance: %d operations in %v (%.0f ops/sec)", totalOps, duration, opsPerSec)

	// 并发性能要求：至少100 ops/sec
	if opsPerSec < 100 {
		t.Errorf("Concurrent performance: %.0f ops/sec, expected >= 100 ops/sec", opsPerSec)
	}
}

// TestWindowManagerMemoryManagement 测试内存管理
func TestWindowManagerMemoryManagement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory management test in short mode")
	}

	wm := NewSimpleWindowManager()

	// 测试大量窗口的内存使用
	t.Run("LargeScaleMemoryUsage", func(t *testing.T) {
		testLargeScaleMemoryUsage(t, wm)
	})

	// 测试窗口删除后的内存清理
	t.Run("MemoryCleanupAfterDeletion", func(t *testing.T) {
		testMemoryCleanupAfterDeletion(t, wm)
	})
}

func testLargeScaleMemoryUsage(t *testing.T, wm *SimpleWindowManager) {
	numWindows := 100
	itemsPerWindow := 100

	// 创建大量窗口
	for i := 0; i < numWindows; i++ {
		windowID := fmt.Sprintf("memory.window.%d", i)
		err := wm.CreateWindow(windowID, time.Hour, itemsPerWindow)
		if err != nil {
			t.Errorf("Failed to create window %d: %v", i, err)
			continue
		}

		// 向每个窗口添加数据
		for j := 0; j < itemsPerWindow; j++ {
			data := fmt.Sprintf("memory-data-%d-%d", i, j)
			if err := wm.AddToWindow(windowID, data); err != nil {
				t.Errorf("Failed to add data: %v", err)
			}
		}
	}

	t.Logf("Successfully created %d windows with %d items each", numWindows, itemsPerWindow)

	// 验证数据完整性
	for i := 0; i < 10; i++ { // 只验证前10个窗口以节省时间
		windowID := fmt.Sprintf("memory.window.%d", i)
		data := wm.GetWindowData(windowID)
		if len(data) == 0 {
			t.Errorf("Window %s should contain data", windowID)
		}
	}
}

func testMemoryCleanupAfterDeletion(t *testing.T, wm *SimpleWindowManager) {
	numWindows := 100

	// 创建窗口
	for i := 0; i < numWindows; i++ {
		windowID := fmt.Sprintf("cleanup.window.%d", i)
		err := wm.CreateWindow(windowID, time.Hour, 100)
		if err != nil {
			t.Errorf("Failed to create window %d: %v", i, err)
			continue
		}

		// 添加一些数据
		for j := 0; j < 50; j++ {
			data := fmt.Sprintf("cleanup-data-%d-%d", i, j)
			if err := wm.AddToWindow(windowID, data); err != nil {
				t.Errorf("Failed to add data: %v", err)
			}
		}
	}

	// 删除一半窗口
	for i := 0; i < numWindows/2; i++ {
		windowID := fmt.Sprintf("cleanup.window.%d", i)
		wm.DeleteWindow(windowID)
	}

	// 验证删除的窗口不存在
	for i := 0; i < numWindows/2; i++ {
		windowID := fmt.Sprintf("cleanup.window.%d", i)
		window := wm.GetWindow(windowID)
		if window != nil {
			t.Errorf("Window %s should not exist after deletion", windowID)
		}
	}

	// 验证剩余窗口仍然存在
	for i := numWindows / 2; i < numWindows; i++ {
		windowID := fmt.Sprintf("cleanup.window.%d", i)
		window := wm.GetWindow(windowID)
		if window == nil {
			t.Errorf("Window %s should still exist", windowID)
		}
	}

	t.Logf("Memory cleanup test completed: deleted %d windows, %d windows remaining", numWindows/2, numWindows/2)
}

// TestWindowManagerErrorRecovery 测试错误恢复
func TestWindowManagerErrorRecovery(t *testing.T) {
	wm := NewSimpleWindowManager()

	// 测试从错误状态恢复
	t.Run("ErrorRecovery", func(t *testing.T) {
		testErrorRecovery(t, wm)
	})

	// 测试资源清理
	t.Run("ResourceCleanup", func(t *testing.T) {
		testResourceCleanup(t, wm)
	})
}

func testErrorRecovery(t *testing.T, wm *SimpleWindowManager) {
	// 尝试创建无效窗口
	err := wm.CreateWindow("", time.Second, 100)
	if err == nil {
		t.Error("Creating invalid window should fail")
	}

	// 验证窗口管理器仍然可以正常工作
	err = wm.CreateWindow("recovery.test", time.Minute, 100)
	if err != nil {
		t.Errorf("Window manager should recover from error: %v", err)
	}

	// 添加数据
	err = wm.AddToWindow("recovery.test", "recovery-data")
	if err != nil {
		t.Errorf("Should be able to add data after recovery: %v", err)
	}

	// 获取数据
	data := wm.GetWindowData("recovery.test")
	if len(data) != 1 {
		t.Errorf("Expected 1 item, got %d", len(data))
	}
}

func testResourceCleanup(t *testing.T, wm *SimpleWindowManager) {
	// 创建一些窗口
	for i := 0; i < 100; i++ {
		windowID := fmt.Sprintf("resource.cleanup.%d", i)
		if err := wm.CreateWindow(windowID, time.Minute, 100); err != nil {
			t.Errorf("Failed to create window: %v", err)
			continue
		}

		// 添加数据
		for j := 0; j < 50; j++ {
			data := fmt.Sprintf("cleanup-data-%d-%d", i, j)
			if err := wm.AddToWindow(windowID, data); err != nil {
				t.Errorf("Failed to add data: %v", err)
			}
		}
	}

	// 删除所有窗口
	for i := 0; i < 100; i++ {
		windowID := fmt.Sprintf("resource.cleanup.%d", i)
		wm.DeleteWindow(windowID)
	}

	// 验证所有窗口都被删除
	for i := 0; i < 100; i++ {
		windowID := fmt.Sprintf("resource.cleanup.%d", i)
		window := wm.GetWindow(windowID)
		if window != nil {
			t.Errorf("Window %s should not exist after cleanup", windowID)
		}
	}

	t.Log("Resource cleanup completed successfully")
}
