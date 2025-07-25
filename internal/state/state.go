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
	"sync"
	"time"

	"github.com/crazy/edge-stream/internal/constants"
)

// StateType 状态类型
type StateType string

const (
	StateTypeMemory      StateType = "memory"
	StateTypePersistent  StateType = "persistent"
	StateTypeDistributed StateType = "distributed"
)

// State 状态接口
type State interface {
	// Get 获取状态值
	Get(key string) (interface{}, bool)
	// Set 设置状态值
	Set(key string, value interface{}) error
	// Delete 删除状态
	Delete(key string) error
	// Exists 检查状态是否存在
	Exists(key string) bool
	// Keys 获取所有键
	Keys() []string
	// Size 获取状态数量
	Size() int
	// Clear 清空所有状态
	Clear() error
	// GetMetrics 获取状态指标
	GetMetrics() StateMetrics
}

// StateManager 状态管理器接口
type StateManager interface {
	// CreateState 创建状态
	CreateState(name string, stateType StateType) (State, error)
	// GetState 获取状态
	GetState(name string) (State, bool)
	// DeleteState 删除状态
	DeleteState(name string) error
	// ListStates 列出所有状态
	ListStates() []string
	// CreateCheckpoint 创建检查点
	CreateCheckpoint(ctx context.Context) (*Checkpoint, error)
	// RestoreFromCheckpoint 从检查点恢复
	RestoreFromCheckpoint(ctx context.Context, checkpoint *Checkpoint) error
	// Watch 监听状态变更
	Watch(ctx context.Context, callback StateChangeCallback)
	// GetCheckpointManager 获取检查点管理器
	GetCheckpointManager() CheckpointManager
	// GetMetrics 获取状态管理器指标
	GetMetrics() StateManagerMetrics
	// Export 导出状态数据
	Export() ([]byte, error)
	// Close 关闭状态管理器
	Close() error
}

// StateChangeCallback 状态变更回调
type StateChangeCallback func(stateName, key string, oldValue, newValue interface{})

// Checkpoint 检查点
type Checkpoint struct {
	ID        string                            `json:"id"`
	Timestamp time.Time                         `json:"timestamp"`
	States    map[string]map[string]interface{} `json:"states"`
	Metadata  map[string]interface{}            `json:"metadata"`
}

// CheckpointManager 检查点管理器接口
type CheckpointManager interface {
	// Save 保存检查点
	Save(ctx context.Context, checkpoint *Checkpoint) error
	// Load 加载检查点
	Load(ctx context.Context, checkpointID string) (*Checkpoint, error)
	// List 列出检查点
	List(ctx context.Context) ([]*CheckpointInfo, error)
	// Delete 删除检查点
	Delete(ctx context.Context, checkpointID string) error
	// Cleanup 清理过期检查点
	Cleanup(ctx context.Context, retentionPeriod time.Duration) error
	// Validate 验证检查点
	Validate(ctx context.Context, checkpointID string) error
}

// CheckpointInfo 检查点信息
type CheckpointInfo struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Size      int64     `json:"size"`
	States    int       `json:"states"`
}

// StateConfig 状态配置
type StateConfig struct {
	// CheckpointInterval 检查点间隔
	CheckpointInterval time.Duration `yaml:"checkpoint_interval" json:"checkpoint_interval"`
	// MaxCheckpoints 最大检查点数量
	MaxCheckpoints int `yaml:"max_checkpoints" json:"max_checkpoints"`
	// PersistentStorage 持久化存储配置
	PersistentStorage persistentStorageConfig `yaml:"persistent_storage" json:"persistent_storage"`
	// DistributedConfig 分布式配置
	DistributedConfig distributedConfig `yaml:"distributed" json:"distributed"`
}

// persistentStorageConfig 持久化存储配置
type persistentStorageConfig struct {
	// Type 存储类型 (file, redis, etcd)
	Type string `yaml:"type" json:"type"`
	// Path 文件路径 (for file type)
	Path string `yaml:"path" json:"path"`
	// Redis Redis配置 (for redis type)
	Redis redisConfig `yaml:"redis" json:"redis"`
	// Etcd Etcd配置 (for etcd type)
	Etcd etcdConfig `yaml:"etcd" json:"etcd"`
}

// redisConfig Redis配置
type redisConfig struct {
	Host     string `yaml:"host" json:"host"`
	Port     int    `yaml:"port" json:"port"`
	Password string `yaml:"password" json:"password"`
	DB       int    `yaml:"db" json:"db"`
	Prefix   string `yaml:"prefix" json:"prefix"`
}

// etcdConfig Etcd配置
type etcdConfig struct {
	Endpoints []string `yaml:"endpoints" json:"endpoints"`
	Username  string   `yaml:"username" json:"username"`
	Password  string   `yaml:"password" json:"password"`
	Prefix    string   `yaml:"prefix" json:"prefix"`
}

// distributedConfig 分布式配置
type distributedConfig struct {
	// Enabled 是否启用分布式
	Enabled bool `yaml:"enabled" json:"enabled"`
	// NodeID 节点ID
	NodeID string `yaml:"node_id" json:"node_id"`
	// ClusterNodes 集群节点
	ClusterNodes []string `yaml:"cluster_nodes" json:"cluster_nodes"`
	// ReplicationFactor 复制因子
	ReplicationFactor int `yaml:"replication_factor" json:"replication_factor"`
	// ConsistencyLevel 一致性级别
	ConsistencyLevel string `yaml:"consistency_level" json:"consistency_level"`
}

// DefaultStateConfig 默认状态配置
func DefaultStateConfig() *StateConfig {
	return &StateConfig{
		CheckpointInterval: constants.DefaultCheckpointIntervalMinutes * time.Minute,
		MaxCheckpoints:     constants.DefaultMaxCheckpoints,
		PersistentStorage: persistentStorageConfig{
			Type: "file",
			Path: "./data/state",
		},
		DistributedConfig: distributedConfig{
			Enabled:           false,
			ReplicationFactor: constants.DefaultReplicationFactor,
			ConsistencyLevel:  "strong",
		},
	}
}

// StateEvent 状态事件
type StateEvent struct {
	Type      StateEventType `json:"type"`
	StateName string         `json:"state_name"`
	Key       string         `json:"key"`
	OldValue  interface{}    `json:"old_value"`
	NewValue  interface{}    `json:"new_value"`
	Timestamp time.Time      `json:"timestamp"`
}

// StateEventType 状态事件类型
type StateEventType string

const (
	StateEventTypeSet    StateEventType = "set"
	StateEventTypeDelete StateEventType = "delete"
	StateEventTypeClear  StateEventType = "clear"
)

// StateMetrics 状态指标
type StateMetrics struct {
	mu sync.RWMutex

	// 基础指标
	StateCount       int64 `json:"state_count"`
	TotalOperations  int64 `json:"total_operations"`
	GetOperations    int64 `json:"get_operations"`
	SetOperations    int64 `json:"set_operations"`
	DeleteOperations int64 `json:"delete_operations"`

	// 性能指标
	AverageLatency time.Duration `json:"average_latency"`
	MaxLatency     time.Duration `json:"max_latency"`
	MinLatency     time.Duration `json:"min_latency"`

	// 检查点指标
	CheckpointCount    int64     `json:"checkpoint_count"`
	LastCheckpointTime time.Time `json:"last_checkpoint_time"`

	// 错误指标
	ErrorCount int64 `json:"error_count"`
}

// StateManagerMetrics 状态管理器指标
type StateManagerMetrics struct {
	StatesCount      int64     `json:"states_count"`
	CheckpointsCount int64     `json:"checkpoints_count"`
	LastCheckpointAt time.Time `json:"last_checkpoint_at"`
	TotalOperations  int64     `json:"total_operations"`
	ErrorsCount      int64     `json:"errors_count"`
	StartTime        time.Time `json:"start_time"`
}

// IncrementOperation 增加操作计数
func (sm *StateMetrics) IncrementOperation(operationType string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.TotalOperations++
	switch operationType {
	case "get":
		sm.GetOperations++
	case "set":
		sm.SetOperations++
	case "delete":
		sm.DeleteOperations++
	}
}

// UpdateLatency 更新延迟指标
func (sm *StateMetrics) UpdateLatency(latency time.Duration) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.MaxLatency == 0 || latency > sm.MaxLatency {
		sm.MaxLatency = latency
	}
	if sm.MinLatency == 0 || latency < sm.MinLatency {
		sm.MinLatency = latency
	}

	// 简单的移动平均
	if sm.AverageLatency == 0 {
		sm.AverageLatency = latency
	} else {
		sm.AverageLatency = (sm.AverageLatency + latency) / 2
	}
}

// IncrementError 增加错误计数
func (sm *StateMetrics) IncrementError() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.ErrorCount++
}

// GetMetrics 获取指标快照
func (sm *StateMetrics) GetMetrics() StateMetrics {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// 返回不包含锁的副本
	return StateMetrics{
		StateCount:         sm.StateCount,
		TotalOperations:    sm.TotalOperations,
		GetOperations:      sm.GetOperations,
		SetOperations:      sm.SetOperations,
		DeleteOperations:   sm.DeleteOperations,
		AverageLatency:     sm.AverageLatency,
		MaxLatency:         sm.MaxLatency,
		MinLatency:         sm.MinLatency,
		CheckpointCount:    sm.CheckpointCount,
		LastCheckpointTime: sm.LastCheckpointTime,
		ErrorCount:         sm.ErrorCount,
	}
}
