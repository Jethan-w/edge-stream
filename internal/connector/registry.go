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

	"github.com/crazy/edge-stream/internal/constants"
)

// StandardConnectorRegistry 标准连接器注册中心
type StandardConnectorRegistry struct {
	sourceFactories    map[string]ConnectorFactory
	sinkFactories      map[string]ConnectorFactory
	transformFactories map[string]ConnectorFactory
	mutex              sync.RWMutex
}

// NewStandardConnectorRegistry 创建标准连接器注册中心
func NewStandardConnectorRegistry() *StandardConnectorRegistry {
	return &StandardConnectorRegistry{
		sourceFactories:    make(map[string]ConnectorFactory),
		sinkFactories:      make(map[string]ConnectorFactory),
		transformFactories: make(map[string]ConnectorFactory),
	}
}

// Register 注册连接器
func (r *StandardConnectorRegistry) Register(connectorType ConnectorType, name string, factory ConnectorFactory) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if name == "" {
		return fmt.Errorf("connector name cannot be empty")
	}

	if factory == nil {
		return fmt.Errorf("connector factory cannot be nil")
	}

	switch connectorType {
	case ConnectorTypeSource:
		if _, exists := r.sourceFactories[name]; exists {
			return fmt.Errorf("source connector '%s' already registered", name)
		}
		r.sourceFactories[name] = factory
	case ConnectorTypeSink:
		if _, exists := r.sinkFactories[name]; exists {
			return fmt.Errorf("sink connector '%s' already registered", name)
		}
		r.sinkFactories[name] = factory
	case ConnectorTypeTransform:
		if _, exists := r.transformFactories[name]; exists {
			return fmt.Errorf("transform connector '%s' already registered", name)
		}
		r.transformFactories[name] = factory
	default:
		return fmt.Errorf("unknown connector type: %s", connectorType)
	}

	return nil
}

// Unregister 注销连接器
func (r *StandardConnectorRegistry) Unregister(connectorType ConnectorType, name string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	switch connectorType {
	case ConnectorTypeSource:
		if _, exists := r.sourceFactories[name]; !exists {
			return fmt.Errorf("source connector '%s' not found", name)
		}
		delete(r.sourceFactories, name)
	case ConnectorTypeSink:
		if _, exists := r.sinkFactories[name]; !exists {
			return fmt.Errorf("sink connector '%s' not found", name)
		}
		delete(r.sinkFactories, name)
	case ConnectorTypeTransform:
		if _, exists := r.transformFactories[name]; !exists {
			return fmt.Errorf("transform connector '%s' not found", name)
		}
		delete(r.transformFactories, name)
	default:
		return fmt.Errorf("unknown connector type: %s", connectorType)
	}

	return nil
}

// Create 创建连接器实例
func (r *StandardConnectorRegistry) Create(connectorType ConnectorType, name string, config map[string]interface{}) (Connector, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var factory ConnectorFactory
	var exists bool

	switch connectorType {
	case ConnectorTypeSource:
		factory, exists = r.sourceFactories[name]
	case ConnectorTypeSink:
		factory, exists = r.sinkFactories[name]
	case ConnectorTypeTransform:
		factory, exists = r.transformFactories[name]
	default:
		return nil, fmt.Errorf("unknown connector type: %s", connectorType)
	}

	if !exists {
		return nil, fmt.Errorf("%s connector '%s' not found", connectorType, name)
	}

	return factory.Create(config)
}

// List 列出已注册的连接器
func (r *StandardConnectorRegistry) List(connectorType ConnectorType) []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var names []string

	switch connectorType {
	case ConnectorTypeSource:
		for name := range r.sourceFactories {
			names = append(names, name)
		}
	case ConnectorTypeSink:
		for name := range r.sinkFactories {
			names = append(names, name)
		}
	case ConnectorTypeTransform:
		for name := range r.transformFactories {
			names = append(names, name)
		}
	}

	return names
}

// GetInfo 获取连接器信息
func (r *StandardConnectorRegistry) GetInfo(connectorType ConnectorType, name string) (*ConnectorInfo, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var factory ConnectorFactory
	var exists bool

	switch connectorType {
	case ConnectorTypeSource:
		factory, exists = r.sourceFactories[name]
	case ConnectorTypeSink:
		factory, exists = r.sinkFactories[name]
	case ConnectorTypeTransform:
		factory, exists = r.transformFactories[name]
	default:
		return nil, fmt.Errorf("unknown connector type: %s", connectorType)
	}

	if !exists {
		return nil, fmt.Errorf("%s connector '%s' not found", connectorType, name)
	}

	info := factory.GetInfo()
	return &info, nil
}

// Validate 验证连接器配置
func (r *StandardConnectorRegistry) Validate(connectorType ConnectorType, name string, config map[string]interface{}) error {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var factory ConnectorFactory
	var exists bool

	switch connectorType {
	case ConnectorTypeSource:
		factory, exists = r.sourceFactories[name]
	case ConnectorTypeSink:
		factory, exists = r.sinkFactories[name]
	case ConnectorTypeTransform:
		factory, exists = r.transformFactories[name]
	default:
		return fmt.Errorf("unknown connector type: %s", connectorType)
	}

	if !exists {
		return fmt.Errorf("%s connector '%s' not found", connectorType, name)
	}

	return factory.ValidateConfig(config)
}

// GetAllConnectors 获取所有连接器信息
func (r *StandardConnectorRegistry) GetAllConnectors() map[ConnectorType][]ConnectorInfo {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result := make(map[ConnectorType][]ConnectorInfo)

	// 获取源连接器
	var sourceInfos []ConnectorInfo
	for _, factory := range r.sourceFactories {
		info := factory.GetInfo()
		sourceInfos = append(sourceInfos, info)
	}
	result[ConnectorTypeSource] = sourceInfos

	// 获取接收器连接器
	var sinkInfos []ConnectorInfo
	for _, factory := range r.sinkFactories {
		info := factory.GetInfo()
		sinkInfos = append(sinkInfos, info)
	}
	result[ConnectorTypeSink] = sinkInfos

	// 获取转换连接器
	var transformInfos []ConnectorInfo
	for _, factory := range r.transformFactories {
		info := factory.GetInfo()
		transformInfos = append(transformInfos, info)
	}
	result[ConnectorTypeTransform] = transformInfos

	return result
}

// GetConnectorCount 获取连接器数量
func (r *StandardConnectorRegistry) GetConnectorCount() map[ConnectorType]int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return map[ConnectorType]int{
		ConnectorTypeSource:    len(r.sourceFactories),
		ConnectorTypeSink:      len(r.sinkFactories),
		ConnectorTypeTransform: len(r.transformFactories),
	}
}

// Clear 清空所有注册的连接器
func (r *StandardConnectorRegistry) Clear() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.sourceFactories = make(map[string]ConnectorFactory)
	r.sinkFactories = make(map[string]ConnectorFactory)
	r.transformFactories = make(map[string]ConnectorFactory)
}

// StandardConnectorManager 标准连接器管理器
type StandardConnectorManager struct {
	registry   ConnectorRegistry
	connectors map[string]*ConnectorInstance
	config     *ConnectorConfig
	mutex      sync.RWMutex
	eventChan  chan ConnectorEvent
}

// NewStandardConnectorManager 创建标准连接器管理器
func NewStandardConnectorManager(registry ConnectorRegistry, config *ConnectorConfig) *StandardConnectorManager {
	if config == nil {
		config = DefaultConnectorConfig()
	}

	return &StandardConnectorManager{
		registry:   registry,
		connectors: make(map[string]*ConnectorInstance),
		config:     config,
		eventChan:  make(chan ConnectorEvent, constants.DefaultConnectorEventChannelSize),
	}
}

// CreateConnector 创建连接器
func (m *StandardConnectorManager) CreateConnector(
	ctx context.Context,
	id string,
	connectorType ConnectorType,
	name string,
	config map[string]interface{},
) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.connectors[id]; exists {
		return fmt.Errorf("connector with id '%s' already exists", id)
	}

	// 验证配置
	if err := m.registry.Validate(connectorType, name, config); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// 创建连接器实例
	connector, err := m.registry.Create(connectorType, name, config)
	if err != nil {
		return fmt.Errorf("failed to create connector: %w", err)
	}

	// 配置连接器
	if err := connector.Configure(config); err != nil {
		return fmt.Errorf("failed to configure connector: %w", err)
	}

	// 创建连接器实例
	instance := NewConnectorInstance(id, connector, config)
	m.connectors[id] = instance

	// 发送事件
	m.sendEvent(ConnectorEventCreated, id, fmt.Sprintf("Connector '%s' created", id))

	return nil
}

// StartConnector 启动连接器
func (m *StandardConnectorManager) StartConnector(ctx context.Context, id string) error {
	m.mutex.RLock()
	instance, exists := m.connectors[id]
	m.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("connector '%s' not found", id)
	}

	if instance.GetStatus() == ConnectorStatusRunning {
		return fmt.Errorf("connector '%s' is already running", id)
	}

	instance.SetStatus(ConnectorStatusStarting)

	if err := instance.Connector.Start(ctx); err != nil {
		instance.SetError(err)
		m.sendEvent(ConnectorEventError, id, fmt.Sprintf("Failed to start connector: %v", err))
		return fmt.Errorf("failed to start connector: %w", err)
	}

	instance.SetStatus(ConnectorStatusRunning)
	m.sendEvent(ConnectorEventStarted, id, fmt.Sprintf("Connector '%s' started", id))

	return nil
}

// StopConnector 停止连接器
func (m *StandardConnectorManager) StopConnector(ctx context.Context, id string) error {
	m.mutex.RLock()
	instance, exists := m.connectors[id]
	m.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("connector '%s' not found", id)
	}

	if instance.GetStatus() == ConnectorStatusStopped {
		return fmt.Errorf("connector '%s' is already stopped", id)
	}

	instance.SetStatus(ConnectorStatusStopping)

	if err := instance.Connector.Stop(ctx); err != nil {
		instance.SetError(err)
		m.sendEvent(ConnectorEventError, id, fmt.Sprintf("Failed to stop connector: %v", err))
		return fmt.Errorf("failed to stop connector: %w", err)
	}

	instance.SetStatus(ConnectorStatusStopped)
	m.sendEvent(ConnectorEventStopped, id, fmt.Sprintf("Connector '%s' stopped", id))

	return nil
}

// DeleteConnector 删除连接器
func (m *StandardConnectorManager) DeleteConnector(ctx context.Context, id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	instance, exists := m.connectors[id]
	if !exists {
		return fmt.Errorf("connector '%s' not found", id)
	}

	// 如果连接器正在运行，先停止它
	if instance.GetStatus() == ConnectorStatusRunning {
		if err := instance.Connector.Stop(ctx); err != nil {
			return fmt.Errorf("failed to stop connector before deletion: %w", err)
		}
	}

	delete(m.connectors, id)
	m.sendEvent(ConnectorEventDeleted, id, fmt.Sprintf("Connector '%s' deleted", id))

	return nil
}

// GetConnector 获取连接器
func (m *StandardConnectorManager) GetConnector(id string) (Connector, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if instance, exists := m.connectors[id]; exists {
		return instance.Connector, true
	}
	return nil, false
}

// ListConnectors 列出所有连接器
func (m *StandardConnectorManager) ListConnectors() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var ids []string
	for id := range m.connectors {
		ids = append(ids, id)
	}
	return ids
}

// GetConnectorStatus 获取连接器状态
func (m *StandardConnectorManager) GetConnectorStatus(id string) (ConnectorStatus, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if instance, exists := m.connectors[id]; exists {
		return instance.GetStatus(), nil
	}
	return "", fmt.Errorf("connector '%s' not found", id)
}

// GetConnectorMetrics 获取连接器指标
func (m *StandardConnectorManager) GetConnectorMetrics(id string) (*ConnectorMetrics, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if instance, exists := m.connectors[id]; exists {
		metrics := instance.Connector.GetMetrics()
		metrics.Uptime = instance.GetUptime()
		return &metrics, nil
	}
	return nil, fmt.Errorf("connector '%s' not found", id)
}

// RestartConnector 重启连接器
func (m *StandardConnectorManager) RestartConnector(ctx context.Context, id string) error {
	if err := m.StopConnector(ctx, id); err != nil {
		return fmt.Errorf("failed to stop connector: %w", err)
	}

	// 等待一小段时间确保连接器完全停止
	time.Sleep(constants.DefaultConnectorRestartDelayMilliseconds * time.Millisecond)

	if err := m.StartConnector(ctx, id); err != nil {
		return fmt.Errorf("failed to start connector: %w", err)
	}

	return nil
}

// UpdateConnectorConfig 更新连接器配置
func (m *StandardConnectorManager) UpdateConnectorConfig(ctx context.Context, id string, config map[string]interface{}) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	instance, exists := m.connectors[id]
	if !exists {
		return fmt.Errorf("connector '%s' not found", id)
	}

	// 验证新配置
	info := instance.Connector.GetInfo()
	if err := m.registry.Validate(info.Type, info.Name, config); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// 配置连接器
	if err := instance.Connector.Configure(config); err != nil {
		return fmt.Errorf("failed to configure connector: %w", err)
	}

	// 更新实例配置
	instance.Config = config
	m.sendEvent(ConnectorEventUpdated, id, fmt.Sprintf("Connector '%s' configuration updated", id))

	return nil
}

// GetEventChannel 获取事件通道
func (m *StandardConnectorManager) GetEventChannel() <-chan ConnectorEvent {
	return m.eventChan
}

// sendEvent 发送事件
func (m *StandardConnectorManager) sendEvent(eventType ConnectorEventType, connectorID, message string) {
	event := ConnectorEvent{
		Type:        eventType,
		ConnectorID: connectorID,
		Timestamp:   time.Now(),
		Message:     message,
		Metadata:    nil,
	}

	select {
	case m.eventChan <- event:
	default:
		// 如果事件通道满了，丢弃事件
	}
}

// GetConnectorInstances 获取所有连接器实例信息
func (m *StandardConnectorManager) GetConnectorInstances() map[string]*ConnectorInstance {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make(map[string]*ConnectorInstance)
	for id, instance := range m.connectors {
		result[id] = instance
	}
	return result
}

// Close 关闭连接器管理器
func (m *StandardConnectorManager) Close(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 停止所有运行中的连接器
	for id, instance := range m.connectors {
		if instance.GetStatus() == ConnectorStatusRunning {
			if err := instance.Connector.Stop(ctx); err != nil {
				// 记录错误但继续关闭其他连接器
				fmt.Printf("Error stopping connector %s: %v\n", id, err)
			}
		}
	}

	// 关闭事件通道
	close(m.eventChan)

	return nil
}
