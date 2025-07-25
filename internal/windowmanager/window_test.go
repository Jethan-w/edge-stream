package windowmanager

import (
	"testing"
	"time"
)

func TestTimeWindow(t *testing.T) {
	// 创建时间窗口
	window := NewTimeWindow(2 * time.Second)

	// 检查初始状态
	if window.GetWindowType() != TIME_WINDOW {
		t.Errorf("Expected window type %v, got %v", TIME_WINDOW, window.GetWindowType())
	}

	if window.IsReady() {
		t.Error("Window should not be ready initially")
	}

	// 添加数据
	window.AddData("test1")
	window.AddData("test2")

	// 检查数据
	data := window.GetData()
	if len(data) != 2 {
		t.Errorf("Expected 2 data items, got %d", len(data))
	}

	// 等待窗口准备好
	time.Sleep(2100 * time.Millisecond)

	if !window.IsReady() {
		t.Error("Window should be ready after timeout")
	}

	// 重置窗口
	window.Reset()
	data = window.GetData()
	if len(data) != 0 {
		t.Errorf("Expected 0 data items after reset, got %d", len(data))
	}

	if window.IsReady() {
		t.Error("Window should not be ready after reset")
	}
}

func TestCountWindow(t *testing.T) {
	// 创建计数窗口
	window := NewCountWindow(3)

	// 检查初始状态
	if window.GetWindowType() != COUNT_WINDOW {
		t.Errorf("Expected window type %v, got %v", COUNT_WINDOW, window.GetWindowType())
	}

	if window.IsReady() {
		t.Error("Window should not be ready initially")
	}

	// 添加数据
	window.AddData("test1")
	if window.IsReady() {
		t.Error("Window should not be ready with 1 item")
	}

	window.AddData("test2")
	if window.IsReady() {
		t.Error("Window should not be ready with 2 items")
	}

	window.AddData("test3")
	if !window.IsReady() {
		t.Error("Window should be ready with 3 items")
	}

	// 检查数据
	data := window.GetData()
	if len(data) != 3 {
		t.Errorf("Expected 3 data items, got %d", len(data))
	}

	// 重置窗口
	window.Reset()
	data = window.GetData()
	if len(data) != 0 {
		t.Errorf("Expected 0 data items after reset, got %d", len(data))
	}

	if window.IsReady() {
		t.Error("Window should not be ready after reset")
	}
}

func TestSimpleWindowManager(t *testing.T) {
	// 创建窗口管理器
	manager := NewSimpleWindowManager()

	// 创建窗口
	_ = manager.CreateTimeWindow(1 * time.Second)
	_ = manager.CreateCountWindow(2)

	// 处理数据
	err := manager.ProcessData("test1")
	if err != nil {
		t.Errorf("ProcessData failed: %v", err)
	}

	err = manager.ProcessData("test2")
	if err != nil {
		t.Errorf("ProcessData failed: %v", err)
	}

	// 检查计数窗口是否准备好
	readyWindows := manager.GetReadyWindows()
	if len(readyWindows) != 1 {
		t.Errorf("Expected 1 ready window, got %d", len(readyWindows))
	}

	if readyWindows[0].GetWindowType() != COUNT_WINDOW {
		t.Errorf("Expected count window to be ready, got %v", readyWindows[0].GetWindowType())
	}

	// 等待时间窗口准备好
	time.Sleep(1100 * time.Millisecond)

	readyWindows = manager.GetReadyWindows()
	found := false
	for _, window := range readyWindows {
		if window.GetWindowType() == TIME_WINDOW {
			found = true
			break
		}
	}

	if !found {
		t.Error("Time window should be ready after timeout")
	}
}
