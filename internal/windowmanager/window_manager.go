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

// Package windowmanager 提供窗口管理功能，用于处理时间窗口和计数窗口。
package windowmanager

import (
	"fmt"
	"sync"
	"time"
)

// SimpleWindowManager 简单窗口管理器
type SimpleWindowManager struct {
	windows map[string]Window
	mu      sync.RWMutex
}

// NewSimpleWindowManager 创建简单窗口管理器
func NewSimpleWindowManager() *SimpleWindowManager {
	return &SimpleWindowManager{
		windows: make(map[string]Window),
	}
}

// CreateTimeWindow 创建时间窗口
func (sm *SimpleWindowManager) CreateTimeWindow(duration time.Duration) Window {
	window := NewTimeWindow(duration)
	sm.mu.Lock()
	sm.windows[window.GetID()] = window
	sm.mu.Unlock()
	return window
}

// CreateCountWindow 创建计数窗口
func (sm *SimpleWindowManager) CreateCountWindow(count int) Window {
	window := NewCountWindow(count)
	sm.mu.Lock()
	sm.windows[window.GetID()] = window
	sm.mu.Unlock()
	return window
}

// ProcessData 处理数据
func (sm *SimpleWindowManager) ProcessData(data interface{}) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for _, window := range sm.windows {
		window.AddData(data)
	}
	return nil
}

// GetReadyWindows 获取准备好的窗口
func (sm *SimpleWindowManager) GetReadyWindows() []Window {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var readyWindows []Window
	for _, window := range sm.windows {
		if window.IsReady() {
			readyWindows = append(readyWindows, window)
		}
	}
	return readyWindows
}

// PrintWindowStatus 打印窗口状态
func (sm *SimpleWindowManager) PrintWindowStatus() {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	fmt.Println("=== Window Status ===")
	i := 1
	for id, window := range sm.windows {
		data := window.GetData()
		fmt.Printf("Window %d [%s]: Type=%v, DataCount=%d, Ready=%v\n",
			i, id, window.GetWindowType(), len(data), window.IsReady())
		i++
	}
	fmt.Println("=====================")
}

// CreateWindow 创建窗口（兼容测试）
func (sm *SimpleWindowManager) CreateWindow(id string, duration time.Duration, maxSize int) error {
	if id == "" {
		return fmt.Errorf("window ID cannot be empty")
	}

	// 根据参数创建合适的窗口类型
	var window Window
	if maxSize > 0 {
		// 如果指定了maxSize，创建CountWindow以支持大小限制
		window = NewCountWindow(maxSize)
	} else if duration > 0 {
		// 否则创建TimeWindow
		window = NewTimeWindow(duration)
	} else {
		return fmt.Errorf("either duration or maxSize must be specified")
	}

	sm.mu.Lock()
	sm.windows[id] = window
	sm.mu.Unlock()
	return nil
}

// AddToWindow 向指定窗口添加数据（兼容测试）
func (sm *SimpleWindowManager) AddToWindow(id string, data interface{}) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// 找到对应的窗口并添加数据
	if window, exists := sm.windows[id]; exists {
		window.AddData(data)
		return nil
	}
	return fmt.Errorf("window %s not found", id)
}

// GetWindowData 获取指定窗口的数据（兼容测试）
func (sm *SimpleWindowManager) GetWindowData(id string) []interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// 返回指定窗口的数据
	if window, exists := sm.windows[id]; exists {
		return window.GetData()
	}
	return nil
}

// DeleteWindow 删除指定窗口（兼容测试）
func (sm *SimpleWindowManager) DeleteWindow(id string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 删除指定窗口
	delete(sm.windows, id)
}

// GetWindow 获取指定窗口（兼容测试）
func (sm *SimpleWindowManager) GetWindow(id string) Window {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// 返回指定窗口
	if window, exists := sm.windows[id]; exists {
		return window
	}
	return nil
}
