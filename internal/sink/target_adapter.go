package sink

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/crazy/edge-stream/internal/flowfile"
)

// TargetAdapter 目标适配器接口，定义数据输出的标准接口
type TargetAdapter interface {
	// Initialize 初始化适配器
	Initialize(config map[string]string) error

	// Write 写入数据到目标系统
	Write(flowFile *flowfile.FlowFile) error

	// Close 关闭适配器
	Close() error

	// GetMetadata 获取适配器元数据
	GetMetadata() *TargetAdapterMetadata

	// IsHealthy 检查适配器健康状态
	IsHealthy() bool
}

// AbstractTargetAdapter 抽象目标适配器基类
type AbstractTargetAdapter struct {
	mu sync.RWMutex

	// 配置
	config map[string]string

	// 元数据
	metadata *TargetAdapterMetadata

	// 健康状态
	healthy         bool
	lastHealthCheck time.Time

	// 统计信息
	stats *TargetAdapterStats
}

// TargetAdapterMetadata 目标适配器元数据
type TargetAdapterMetadata struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Version     string    `json:"version"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TargetAdapterStats 目标适配器统计信息
type TargetAdapterStats struct {
	TotalWrites      int64         `json:"total_writes"`
	SuccessfulWrites int64         `json:"successful_writes"`
	FailedWrites     int64         `json:"failed_writes"`
	TotalBytes       int64         `json:"total_bytes"`
	AverageLatency   time.Duration `json:"average_latency"`
	LastWriteTime    time.Time     `json:"last_write_time"`
	mu               sync.RWMutex
}

// NewAbstractTargetAdapter 创建抽象目标适配器
func NewAbstractTargetAdapter(id, name, adapterType string) *AbstractTargetAdapter {
	now := time.Now()

	return &AbstractTargetAdapter{
		config: make(map[string]string),
		metadata: &TargetAdapterMetadata{
			ID:          id,
			Name:        name,
			Type:        adapterType,
			Version:     "1.0.0",
			Description: fmt.Sprintf("%s target adapter", adapterType),
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		healthy:         true,
		lastHealthCheck: now,
		stats: &TargetAdapterStats{
			LastWriteTime: now,
		},
	}
}

// Initialize 初始化适配器
func (a *AbstractTargetAdapter) Initialize(config map[string]string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.config = config
	a.metadata.UpdatedAt = time.Now()

	log.Printf("Initialized target adapter: %s", a.metadata.Name)
	return nil
}

// Write 写入数据（需要子类实现）
func (a *AbstractTargetAdapter) Write(flowFile *flowfile.FlowFile) error {
	// 记录开始时间
	startTime := time.Now()

	// 更新统计信息
	a.stats.mu.Lock()
	a.stats.TotalWrites++
	a.stats.LastWriteTime = time.Now()
	a.stats.mu.Unlock()

	// 子类实现具体的写入逻辑
	err := a.doWrite(flowFile)

	// 记录统计信息
	duration := time.Since(startTime)
	a.stats.mu.Lock()
	if err != nil {
		a.stats.FailedWrites++
	} else {
		a.stats.SuccessfulWrites++
		// 计算平均延迟
		if a.stats.SuccessfulWrites > 0 {
			totalLatency := a.stats.AverageLatency * time.Duration(a.stats.SuccessfulWrites-1)
			a.stats.AverageLatency = (totalLatency + duration) / time.Duration(a.stats.SuccessfulWrites)
		} else {
			a.stats.AverageLatency = duration
		}
	}
	a.stats.mu.Unlock()

	return err
}

// doWrite 具体的写入实现（子类需要重写）
func (a *AbstractTargetAdapter) doWrite(flowFile interface{}) error {
	return fmt.Errorf("doWrite method must be implemented by subclass")
}

// Close 关闭适配器
func (a *AbstractTargetAdapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.healthy = false
	a.metadata.UpdatedAt = time.Now()

	log.Printf("Closed target adapter: %s", a.metadata.Name)
	return nil
}

// GetMetadata 获取适配器元数据
func (a *AbstractTargetAdapter) GetMetadata() *TargetAdapterMetadata {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.metadata
}

// IsHealthy 检查适配器健康状态
func (a *AbstractTargetAdapter) IsHealthy() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// 如果超过5分钟没有健康检查，执行一次
	if time.Since(a.lastHealthCheck) > 5*time.Minute {
		a.mu.RUnlock()
		a.mu.Lock()
		a.healthy = a.performHealthCheck()
		a.lastHealthCheck = time.Now()
		a.mu.Unlock()
		a.mu.RLock()
	}

	return a.healthy
}

// performHealthCheck 执行健康检查（子类可以重写）
func (a *AbstractTargetAdapter) performHealthCheck() bool {
	// 默认实现：检查配置是否有效
	return len(a.config) > 0
}

// GetStats 获取统计信息
func (a *AbstractTargetAdapter) GetStats() *TargetAdapterStats {
	a.stats.mu.RLock()
	defer a.stats.mu.RUnlock()

	// 返回副本以避免并发修改
	stats := *a.stats
	return &stats
}

// GetConfig 获取配置
func (a *AbstractTargetAdapter) GetConfig() map[string]string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// 返回副本以避免并发修改
	config := make(map[string]string)
	for k, v := range a.config {
		config[k] = v
	}
	return config
}

// SetConfig 设置配置
func (a *AbstractTargetAdapter) SetConfig(key, value string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.config[key] = value
	a.metadata.UpdatedAt = time.Now()
}

// LogOutputActivity 记录输出活动日志
func (a *AbstractTargetAdapter) LogOutputActivity(message string) {
	log.Printf("[%s] %s", a.metadata.Name, message)
}

// ValidateFlowFile 验证 FlowFile
func (a *AbstractTargetAdapter) ValidateFlowFile(flowFile interface{}) error {
	if flowFile == nil {
		return fmt.Errorf("flowFile cannot be nil")
	}

	// 检查是否为 FlowFile 类型
	if _, ok := flowFile.(*flowfile.FlowFile); !ok {
		return fmt.Errorf("flowFile must be of type *flowfile.FlowFile")
	}

	return nil
}

// GetRequiredConfig 获取必需的配置项
func (a *AbstractTargetAdapter) GetRequiredConfig() []string {
	return []string{}
}

// ValidateConfig 验证配置
func (a *AbstractTargetAdapter) ValidateConfig() error {
	required := a.GetRequiredConfig()

	for _, key := range required {
		if _, exists := a.config[key]; !exists {
			return fmt.Errorf("required config missing: %s", key)
		}
	}

	return nil
}
