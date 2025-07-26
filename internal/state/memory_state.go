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
	"time"
)

// MemoryState 内存状态实现
type MemoryState struct {
	mu   sync.RWMutex
	data map[string]interface{}
	name string

	// 指标
	metrics *StateMetrics

	// 事件回调
	callback StateChangeCallback
}

// NewMemoryState 创建内存状态
func NewMemoryState(name string) *MemoryState {
	return &MemoryState{
		data:    make(map[string]interface{}),
		name:    name,
		metrics: &StateMetrics{},
	}
}

// Get 获取状态值
func (ms *MemoryState) Get(key string) (interface{}, bool) {
	start := time.Now()
	defer func() {
		ms.metrics.UpdateLatency(time.Since(start))
		ms.metrics.IncrementOperation("get")
	}()

	ms.mu.RLock()
	defer ms.mu.RUnlock()

	value, exists := ms.data[key]
	return value, exists
}

// Set 设置状态值
func (ms *MemoryState) Set(key string, value interface{}) error {
	start := time.Now()
	defer func() {
		ms.metrics.UpdateLatency(time.Since(start))
		ms.metrics.IncrementOperation("set")
	}()

	// 验证空键
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	ms.mu.Lock()
	defer ms.mu.Unlock()

	oldValue, exists := ms.data[key]
	ms.data[key] = value

	// 触发回调
	if ms.callback != nil {
		go ms.callback(ms.name, key, oldValue, value)
	}

	// 更新指标
	if !exists {
		ms.metrics.mu.Lock()
		ms.metrics.StateCount++
		ms.metrics.mu.Unlock()
	}

	return nil
}

// Delete 删除状态
func (ms *MemoryState) Delete(key string) error {
	start := time.Now()
	defer func() {
		ms.metrics.UpdateLatency(time.Since(start))
		ms.metrics.IncrementOperation("delete")
	}()

	ms.mu.Lock()
	defer ms.mu.Unlock()

	oldValue, exists := ms.data[key]
	if !exists {
		return fmt.Errorf("key %s not found", key)
	}

	delete(ms.data, key)

	// 触发回调
	if ms.callback != nil {
		go ms.callback(ms.name, key, oldValue, nil)
	}

	// 更新指标
	ms.metrics.mu.Lock()
	ms.metrics.StateCount--
	ms.metrics.mu.Unlock()

	return nil
}

// Exists 检查状态是否存在
func (ms *MemoryState) Exists(key string) bool {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	_, exists := ms.data[key]
	return exists
}

// Keys 获取所有键
func (ms *MemoryState) Keys() []string {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	keys := make([]string, 0, len(ms.data))
	for key := range ms.data {
		keys = append(keys, key)
	}
	return keys
}

// Size 获取状态数量
func (ms *MemoryState) Size() int {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return len(ms.data)
}

// Clear 清空所有状态
func (ms *MemoryState) Clear() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.data = make(map[string]interface{})

	// 触发回调
	if ms.callback != nil {
		go ms.callback(ms.name, "", nil, nil)
	}

	// 重置指标
	ms.metrics.mu.Lock()
	ms.metrics.StateCount = 0
	ms.metrics.mu.Unlock()

	return nil
}

// SetCallback 设置状态变更回调
func (ms *MemoryState) SetCallback(callback StateChangeCallback) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.callback = callback
}

// GetMetrics 获取状态指标
func (ms *MemoryState) GetMetrics() StateMetrics {
	return ms.metrics.GetMetrics()
}

// GetData 获取所有数据（用于检查点）
func (ms *MemoryState) GetData() map[string]interface{} {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	// 深拷贝数据
	data := make(map[string]interface{})
	for key, value := range ms.data {
		data[key] = value
	}
	return data
}

// SetData 设置所有数据（用于检查点恢复）
func (ms *MemoryState) SetData(data map[string]interface{}) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.data = make(map[string]interface{})
	for key, value := range data {
		ms.data[key] = value
	}

	// 更新指标
	ms.metrics.mu.Lock()
	ms.metrics.StateCount = int64(len(data))
	ms.metrics.mu.Unlock()

	return nil
}

// GetName 获取状态名称
func (ms *MemoryState) GetName() string {
	return ms.name
}
