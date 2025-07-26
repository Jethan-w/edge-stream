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

package connector

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// BaseConnector 基础连接器实现
// 提供所有连接器的通用功能，减少重复代码
type BaseConnector struct {
	info      ConnectorInfo
	config    map[string]interface{}
	status    ConnectorStatus
	metrics   ConnectorMetrics
	startTime time.Time
	mutex     sync.RWMutex
	running   bool

	// 可选的钩子函数，允许子类自定义行为
	onStart  func(ctx context.Context) error
	onStop   func(ctx context.Context) error
	onConfig func(config map[string]interface{}) error
}

// BaseConnectorConfig 基础连接器配置
type BaseConnectorConfig struct {
	Info          ConnectorInfo
	OnStart       func(ctx context.Context) error
	OnStop        func(ctx context.Context) error
	OnConfig      func(config map[string]interface{}) error
	InitialConfig map[string]interface{}
}

// NewBaseConnector 创建基础连接器
func NewBaseConnector(config BaseConnectorConfig) *BaseConnector {
	connector := &BaseConnector{
		info:     config.Info,
		status:   ConnectorStatusStopped,
		metrics:  ConnectorMetrics{},
		onStart:  config.OnStart,
		onStop:   config.OnStop,
		onConfig: config.OnConfig,
	}

	if config.InitialConfig != nil {
		connector.config = config.InitialConfig
		connector.info.Config = config.InitialConfig
	}

	return connector
}

// GetInfo 获取连接器信息
func (c *BaseConnector) GetInfo() ConnectorInfo {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.info
}

// Start 启动连接器
func (c *BaseConnector) Start(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.running {
		return fmt.Errorf("connector %s is already running", c.info.ID)
	}

	// 调用自定义启动逻辑
	if c.onStart != nil {
		if err := c.onStart(ctx); err != nil {
			return fmt.Errorf("failed to start connector %s: %w", c.info.ID, err)
		}
	}

	c.status = ConnectorStatusRunning
	c.startTime = time.Now()
	c.running = true
	c.metrics.LastActivity = c.startTime
	c.info.UpdatedAt = time.Now()

	return nil
}

// Stop 停止连接器
func (c *BaseConnector) Stop(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.running {
		return fmt.Errorf("connector %s is not running", c.info.ID)
	}

	// 调用自定义停止逻辑
	if c.onStop != nil {
		if err := c.onStop(ctx); err != nil {
			return fmt.Errorf("failed to stop connector %s: %w", c.info.ID, err)
		}
	}

	c.status = ConnectorStatusStopped
	c.running = false
	c.info.UpdatedAt = time.Now()

	return nil
}

// Configure 配置连接器
func (c *BaseConnector) Configure(config map[string]interface{}) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// 调用自定义配置逻辑
	if c.onConfig != nil {
		if err := c.onConfig(config); err != nil {
			return fmt.Errorf("failed to configure connector %s: %w", c.info.ID, err)
		}
	}

	c.config = config
	c.info.Config = config
	c.info.UpdatedAt = time.Now()

	return nil
}

// Validate 验证配置（默认实现，子类可以重写）
func (c *BaseConnector) Validate(config map[string]interface{}) error {
	// 默认验证逻辑：检查必需字段
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	return nil
}

// GetStatus 获取状态
func (c *BaseConnector) GetStatus() ConnectorStatus {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.status
}

// GetMetrics 获取指标
func (c *BaseConnector) GetMetrics() ConnectorMetrics {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.metrics
}

// IsRunning 检查是否运行中
func (c *BaseConnector) IsRunning() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.running
}

// GetConfig 获取配置
func (c *BaseConnector) GetConfig() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.config
}

// UpdateMetrics 更新指标（线程安全）
func (c *BaseConnector) UpdateMetrics(updateFunc func(*ConnectorMetrics)) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	updateFunc(&c.metrics)
}

// SetStatus 设置状态（线程安全）
func (c *BaseConnector) SetStatus(status ConnectorStatus) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.status = status
	c.info.UpdatedAt = time.Now()
}

// GetUptime 获取运行时间
func (c *BaseConnector) GetUptime() time.Duration {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if !c.running {
		return 0
	}
	return time.Since(c.startTime)
}

// IncrementProcessedCount 增加处理计数
func (c *BaseConnector) IncrementProcessedCount(count int64) {
	c.UpdateMetrics(func(m *ConnectorMetrics) {
		m.MessagesProcessed += count
		m.LastActivity = time.Now()
	})
}

// IncrementErrorCount 增加错误计数
func (c *BaseConnector) IncrementErrorCount(count int64) {
	c.UpdateMetrics(func(m *ConnectorMetrics) {
		m.ErrorsCount += count
		m.LastActivity = time.Now()
	})
}

// UpdateThroughput 更新吞吐量
func (c *BaseConnector) UpdateThroughput(bytesPerSecond float64) {
	c.UpdateMetrics(func(m *ConnectorMetrics) {
		m.Throughput = bytesPerSecond
	})
}

// UpdateLatency 更新延迟
func (c *BaseConnector) UpdateLatency(latency time.Duration) {
	c.UpdateMetrics(func(m *ConnectorMetrics) {
		m.AverageLatency = latency
	})
}

// BaseConnectorFactory 基础连接器工厂
type BaseConnectorFactory struct {
	info         ConnectorInfo
	createFunc   func(config map[string]interface{}) (Connector, error)
	validateFunc func(config map[string]interface{}) error
}

// NewBaseConnectorFactory 创建基础连接器工厂
func NewBaseConnectorFactory(
	info ConnectorInfo,
	createFunc func(config map[string]interface{}) (Connector, error),
	validateFunc func(config map[string]interface{}) error,
) *BaseConnectorFactory {
	return &BaseConnectorFactory{
		info:         info,
		createFunc:   createFunc,
		validateFunc: validateFunc,
	}
}

// Create 创建连接器实例
func (f *BaseConnectorFactory) Create(config map[string]interface{}) (Connector, error) {
	if f.createFunc == nil {
		return nil, fmt.Errorf("create function not implemented for %s", f.info.ID)
	}
	return f.createFunc(config)
}

// GetInfo 获取连接器信息
func (f *BaseConnectorFactory) GetInfo() ConnectorInfo {
	return f.info
}

// ValidateConfig 验证配置
func (f *BaseConnectorFactory) ValidateConfig(config map[string]interface{}) error {
	if f.validateFunc == nil {
		// 默认验证
		if config == nil {
			return fmt.Errorf("config cannot be nil")
		}
		return nil
	}
	return f.validateFunc(config)
}
