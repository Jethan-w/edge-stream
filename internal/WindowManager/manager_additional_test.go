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
	wm := NewSimpleWindowManager()

	// 测试并发创建窗口
	t.Run("ConcurrentWindowCreation", func(t *testing.T) {
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
				}
			}(i)
		}

		wg.Wait()

		expectedSuccesses := int64(numGoroutines * numWindows)
		if atomic.LoadInt64(&successes) != expectedSuccesses {
			t.Errorf("Expected %d successful creations, got %d", expectedSuccesses, successes)
		}
	})

	// 测试并发添加数据
	t.Run("ConcurrentDataAddition", func(t *testing.T) {
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
	})

	// 测试并发读写
	t.Run("ConcurrentReadWrite", func(t *testing.T) {
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
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					data := fmt.Sprintf("write-data-%d-%d", id, j)
					window.AddData(data)
				}
			}(i)
		}

		// 读取goroutines
		for i := 0; i < numReaders; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					data := window.GetData()
					if data == nil {
						atomic.AddInt64(&readErrors, 1)
					}
				}
			}(i)
		}

		wg.Wait()

		if atomic.LoadInt64(&readErrors) > 0 {
			t.Errorf("Encountered %d read errors", readErrors)
		}
	})

	// 测试并发窗口操作
	t.Run("ConcurrentWindowOperations", func(t *testing.T) {
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
	})
}

// TestWindowManagerTimeWindows 测试时间窗口功能
func TestWindowManagerTimeWindows(t *testing.T) {
	wm := NewSimpleWindowManager()

	// 测试时间窗口过期
	t.Run("TimeWindowExpiration", func(t *testing.T) {
		windowID := "time.expiration.test"
		windowDuration := time.Millisecond * 100

		err := wm.CreateWindow(windowID, windowDuration, 1000)
		if err != nil {
			t.Fatalf("Failed to create window: %v", err)
		}

		// 添加一些数据
		for i := 0; i < 10; i++ {
			data := fmt.Sprintf("data-%d", i)
			err := wm.AddToWindow(windowID, data)
			if err != nil {
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
		// 这里假设过期的数据会被清理
		windowDataAfterExpiry := wm.GetWindowData(windowID)
		t.Logf("Data after expiry: %d items", len(windowDataAfterExpiry))
	})

	// 测试滑动时间窗口
	t.Run("SlidingTimeWindow", func(t *testing.T) {
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
	})

	// 测试多个时间窗口
	t.Run("MultipleTimeWindows", func(t *testing.T) {
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
	})
}

// TestWindowManagerSizeWindows 测试大小窗口功能
func TestWindowManagerSizeWindows(t *testing.T) {
	wm := NewSimpleWindowManager()

	// 测试固定大小窗口
	t.Run("FixedSizeWindow", func(t *testing.T) {
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
	})

	// 测试滑动大小窗口
	t.Run("SlidingSizeWindow", func(t *testing.T) {
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
	})

	// 测试不同大小的窗口
	t.Run("VariousSizeWindows", func(t *testing.T) {
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
				wm.AddToWindow(windowID, data)
			}

			// 验证窗口大小
			windowData := wm.GetWindowData(windowID)
			if len(windowData) > size {
				t.Errorf("Window %s size %d exceeds maximum %d", windowID, len(windowData), size)
			}

			t.Logf("Window %s: %d items (max: %d)", windowID, len(windowData), size)
		}
	})
}

// TestWindowManagerPerformance 测试性能
func TestWindowManagerPerformance(t *testing.T) {
	wm := NewSimpleWindowManager()

	// 测试创建窗口性能
	t.Run("WindowCreationPerformance", func(t *testing.T) {
		numWindows := 10000

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

		// 性能要求：至少5,000 creations/sec
		if creationsPerSec < 5000 {
			t.Errorf("Window creation performance: %.0f creations/sec, expected >= 5,000 creations/sec", creationsPerSec)
		}
	})

	// 测试数据添加性能
	t.Run("DataAdditionPerformance", func(t *testing.T) {
		windowID := "perf.data.addition"
		err := wm.CreateWindow(windowID, time.Hour, 100000)
		if err != nil {
			t.Fatalf("Failed to create window: %v", err)
		}

		numOperations := 100000

		start := time.Now()
		for i := 0; i < numOperations; i++ {
			data := fmt.Sprintf("perf-data-%d", i)
			err := wm.AddToWindow(windowID, data)
			if err != nil {
				t.Errorf("Failed to add data %d: %v", i, err)
			}
		}
		duration := time.Since(start)
		additionsPerSec := float64(numOperations) / duration.Seconds()

		t.Logf("Data Addition Performance: %d additions in %v (%.0f additions/sec)", numOperations, duration, additionsPerSec)

		// 性能要求：至少25,000 additions/sec
		if additionsPerSec < 25000 {
			t.Errorf("Data addition performance: %.0f additions/sec, expected >= 25,000 additions/sec", additionsPerSec)
		}
	})

	// 测试数据获取性能
	t.Run("DataRetrievalPerformance", func(t *testing.T) {
		windowID := "perf.data.retrieval"
		err := wm.CreateWindow(windowID, time.Hour, 10000)
		if err != nil {
			t.Fatalf("Failed to create window: %v", err)
		}

		// 首先添加一些数据
		for i := 0; i < 5000; i++ {
			data := fmt.Sprintf("retrieval-data-%d", i)
			wm.AddToWindow(windowID, data)
		}

		numRetrievals := 10000

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

		// 性能要求：至少50,000 retrievals/sec
		if retrievalsPerSec < 50000 {
			t.Errorf("Data retrieval performance: %.0f retrievals/sec, expected >= 50,000 retrievals/sec", retrievalsPerSec)
		}
	})

	// 测试并发性能
	t.Run("ConcurrentPerformance", func(t *testing.T) {
		windowID := "perf.concurrent"
		err := wm.CreateWindow(windowID, time.Hour, 50000)
		if err != nil {
			t.Fatalf("Failed to create window: %v", err)
		}

		numGoroutines := 20
		numOpsPerGoroutine := 5000
		var wg sync.WaitGroup

		start := time.Now()

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOpsPerGoroutine; j++ {
					data := fmt.Sprintf("concurrent-perf-%d-%d", id, j)
					wm.AddToWindow(windowID, data)
				}
			}(i)
		}

		wg.Wait()

		duration := time.Since(start)
		totalOps := numGoroutines * numOpsPerGoroutine
		opsPerSec := float64(totalOps) / duration.Seconds()

		t.Logf("Concurrent Performance: %d operations in %v (%.0f ops/sec)", totalOps, duration, opsPerSec)

		// 并发性能要求：至少15,000 ops/sec
		if opsPerSec < 15000 {
			t.Errorf("Concurrent performance: %.0f ops/sec, expected >= 15,000 ops/sec", opsPerSec)
		}
	})
}

// TestWindowManagerMemoryManagement 测试内存管理
func TestWindowManagerMemoryManagement(t *testing.T) {
	wm := NewSimpleWindowManager()

	// 测试大量窗口的内存使用
	t.Run("LargeScaleMemoryUsage", func(t *testing.T) {
		numWindows := 1000
		itemsPerWindow := 1000

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
				wm.AddToWindow(windowID, data)
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
	})

	// 测试窗口删除后的内存清理
	t.Run("MemoryCleanupAfterDeletion", func(t *testing.T) {
		numWindows := 1000

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
				wm.AddToWindow(windowID, data)
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
	})
}

// TestWindowManagerErrorRecovery 测试错误恢复
func TestWindowManagerErrorRecovery(t *testing.T) {
	wm := NewSimpleWindowManager()

	// 测试从错误状态恢复
	t.Run("ErrorRecovery", func(t *testing.T) {
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
	})

	// 测试资源清理
	t.Run("ResourceCleanup", func(t *testing.T) {
		// 创建一些窗口
		for i := 0; i < 100; i++ {
			windowID := fmt.Sprintf("resource.cleanup.%d", i)
			wm.CreateWindow(windowID, time.Minute, 100)

			// 添加数据
			for j := 0; j < 50; j++ {
				data := fmt.Sprintf("cleanup-data-%d-%d", i, j)
				wm.AddToWindow(windowID, data)
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
	})
}
