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
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// StandardStateManager 标准状态管理器
type StandardStateManager struct {
	mu     sync.RWMutex
	states map[string]State
	config *StateConfig

	// 检查点管理
	checkpointManager CheckpointManager
	checkpointTicker  *time.Ticker

	// 事件监听
	callbacks []StateChangeCallback

	// 生命周期管理
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// 指标
	metrics *StateManagerMetrics
}

// StateManagerMetrics 状态管理器指标
type StateManagerMetrics struct {
	mu sync.RWMutex

	StatesCount      int64     `json:"states_count"`
	CheckpointsCount int64     `json:"checkpoints_count"`
	LastCheckpointAt time.Time `json:"last_checkpoint_at"`
	TotalOperations  int64     `json:"total_operations"`
	ErrorsCount      int64     `json:"errors_count"`
	StartTime        time.Time `json:"start_time"`
}

// NewStandardStateManager 创建标准状态管理器
func NewStandardStateManager(config *StateConfig) *StandardStateManager {
	if config == nil {
		config = DefaultStateConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	sm := &StandardStateManager{
		states: make(map[string]State),
		config: config,
		ctx:    ctx,
		cancel: cancel,
		metrics: &StateManagerMetrics{
			StartTime: time.Now(),
		},
	}

	// 创建检查点管理器
	sm.checkpointManager = NewFileCheckpointManager(config.PersistentStorage.Path)

	// 启动检查点定时器
	if config.CheckpointInterval > 0 {
		sm.startCheckpointTimer()
	}

	return sm
}

// CreateState 创建状态
func (sm *StandardStateManager) CreateState(name string, stateType StateType) (State, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.states[name]; exists {
		return nil, fmt.Errorf("state %s already exists", name)
	}

	var state State
	var err error

	switch stateType {
	case StateTypeMemory:
		memoryState := NewMemoryState(name)
		memoryState.SetCallback(sm.notifyCallbacks)
		state = memoryState
	case StateTypePersistent:
		// TODO: 实现持久化状态
		return nil, fmt.Errorf("persistent state not implemented yet")
	case StateTypeDistributed:
		// TODO: 实现分布式状态
		return nil, fmt.Errorf("distributed state not implemented yet")
	default:
		return nil, fmt.Errorf("unsupported state type: %s", stateType)
	}

	if err != nil {
		sm.metrics.mu.Lock()
		sm.metrics.ErrorsCount++
		sm.metrics.mu.Unlock()
		return nil, err
	}

	sm.states[name] = state

	// 更新指标
	sm.metrics.mu.Lock()
	sm.metrics.StatesCount++
	sm.metrics.mu.Unlock()

	return state, nil
}

// GetState 获取状态
func (sm *StandardStateManager) GetState(name string) (State, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	state, exists := sm.states[name]
	return state, exists
}

// DeleteState 删除状态
func (sm *StandardStateManager) DeleteState(name string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.states[name]; !exists {
		return fmt.Errorf("state %s not found", name)
	}

	delete(sm.states, name)

	// 更新指标
	sm.metrics.mu.Lock()
	sm.metrics.StatesCount--
	sm.metrics.mu.Unlock()

	return nil
}

// ListStates 列出所有状态
func (sm *StandardStateManager) ListStates() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	names := make([]string, 0, len(sm.states))
	for name := range sm.states {
		names = append(names, name)
	}
	return names
}

// CreateCheckpoint 创建检查点
func (sm *StandardStateManager) CreateCheckpoint(ctx context.Context) (*Checkpoint, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	checkpoint := &Checkpoint{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		States:    make(map[string]map[string]interface{}),
		Metadata: map[string]interface{}{
			"version":      "1.0",
			"states_count": len(sm.states),
		},
	}

	// 收集所有状态数据
	for name, state := range sm.states {
		if memoryState, ok := state.(*MemoryState); ok {
			checkpoint.States[name] = memoryState.GetData()
		} else {
			// 对于其他类型的状态，收集基本信息
			stateData := make(map[string]interface{})
			for _, key := range state.Keys() {
				if value, exists := state.Get(key); exists {
					stateData[key] = value
				}
			}
			checkpoint.States[name] = stateData
		}
	}

	// 保存检查点
	if err := sm.checkpointManager.Save(ctx, checkpoint); err != nil {
		sm.metrics.mu.Lock()
		sm.metrics.ErrorsCount++
		sm.metrics.mu.Unlock()
		return nil, fmt.Errorf("failed to save checkpoint: %w", err)
	}

	// 更新指标
	sm.metrics.mu.Lock()
	sm.metrics.CheckpointsCount++
	sm.metrics.LastCheckpointAt = time.Now()
	sm.metrics.mu.Unlock()

	return checkpoint, nil
}

// RestoreFromCheckpoint 从检查点恢复
func (sm *StandardStateManager) RestoreFromCheckpoint(ctx context.Context, checkpoint *Checkpoint) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 清空现有状态
	sm.states = make(map[string]State)

	// 恢复状态
	for stateName, stateData := range checkpoint.States {
		// 创建内存状态（目前只支持内存状态）
		memoryState := NewMemoryState(stateName)
		memoryState.SetCallback(sm.notifyCallbacks)

		// 恢复数据
		if err := memoryState.SetData(stateData); err != nil {
			return fmt.Errorf("failed to restore state %s: %w", stateName, err)
		}

		sm.states[stateName] = memoryState
	}

	// 更新指标
	sm.metrics.mu.Lock()
	sm.metrics.StatesCount = int64(len(sm.states))
	sm.metrics.mu.Unlock()

	return nil
}

// Watch 监听状态变更
func (sm *StandardStateManager) Watch(ctx context.Context, callback StateChangeCallback) {
	sm.mu.Lock()
	sm.callbacks = append(sm.callbacks, callback)
	sm.mu.Unlock()

	// 启动监听goroutine
	sm.wg.Add(1)
	go func() {
		defer sm.wg.Done()
		<-ctx.Done()
		// 清理回调（可选）
	}()
}

// notifyCallbacks 通知所有回调
func (sm *StandardStateManager) notifyCallbacks(stateName, key string, oldValue, newValue interface{}) {
	sm.mu.RLock()
	callbacks := make([]StateChangeCallback, len(sm.callbacks))
	copy(callbacks, sm.callbacks)
	sm.mu.RUnlock()

	for _, callback := range callbacks {
		if callback != nil {
			callback(stateName, key, oldValue, newValue)
		}
	}
}

// startCheckpointTimer 启动检查点定时器
func (sm *StandardStateManager) startCheckpointTimer() {
	sm.checkpointTicker = time.NewTicker(sm.config.CheckpointInterval)

	sm.wg.Add(1)
	go func() {
		defer sm.wg.Done()
		defer sm.checkpointTicker.Stop()

		for {
			select {
			case <-sm.ctx.Done():
				return
			case <-sm.checkpointTicker.C:
				if _, err := sm.CreateCheckpoint(sm.ctx); err != nil {
					// 记录错误但不中断服务
					fmt.Printf("Failed to create checkpoint: %v\n", err)
				}
			}
		}
	}()
}

// GetMetrics 获取状态管理器指标
func (sm *StandardStateManager) GetMetrics() StateManagerMetrics {
	sm.metrics.mu.RLock()
	defer sm.metrics.mu.RUnlock()

	// 返回不包含锁的副本
	return StateManagerMetrics{
		StatesCount:      sm.metrics.StatesCount,
		CheckpointsCount: sm.metrics.CheckpointsCount,
		LastCheckpointAt: sm.metrics.LastCheckpointAt,
		TotalOperations:  sm.metrics.TotalOperations,
		ErrorsCount:      sm.metrics.ErrorsCount,
		StartTime:        sm.metrics.StartTime,
	}
}

// GetCheckpointManager 获取检查点管理器
func (sm *StandardStateManager) GetCheckpointManager() CheckpointManager {
	return sm.checkpointManager
}

// Close 关闭状态管理器
func (sm *StandardStateManager) Close() error {
	// 取消上下文
	sm.cancel()

	// 等待所有goroutine结束
	sm.wg.Wait()

	// 创建最终检查点
	if _, err := sm.CreateCheckpoint(context.Background()); err != nil {
		fmt.Printf("Failed to create final checkpoint: %v\n", err)
	}

	return nil
}

// Export 导出状态数据
func (sm *StandardStateManager) Export() ([]byte, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	exportData := map[string]interface{}{
		"timestamp": time.Now(),
		"states":    make(map[string]map[string]interface{}),
		"metrics":   sm.GetMetrics(),
	}

	// 导出所有状态数据
	for name, state := range sm.states {
		if memoryState, ok := state.(*MemoryState); ok {
			exportData["states"].(map[string]map[string]interface{})[name] = memoryState.GetData()
		}
	}

	return json.MarshalIndent(exportData, "", "  ")
}
