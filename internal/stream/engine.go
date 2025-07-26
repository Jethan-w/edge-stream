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

package stream

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// StandardStreamEngine 标准流处理引擎
type StandardStreamEngine struct {
	topologies   map[string]*StreamTopology
	streams      map[string]*Stream // 添加流存储
	runningTasks map[string]context.CancelFunc
	eventChan    chan *StreamEvent
	metrics      *EngineMetrics
	mutex        sync.RWMutex
	startTime    time.Time
}

// NewStandardStreamEngine 创建新的标准流处理引擎
func NewStandardStreamEngine() *StandardStreamEngine {
	return &StandardStreamEngine{
		topologies:   make(map[string]*StreamTopology),
		streams:      make(map[string]*Stream), // 初始化流存储
		runningTasks: make(map[string]context.CancelFunc),
		eventChan:    make(chan *StreamEvent, 100),
		metrics: &EngineMetrics{
			TopologyMetrics: make(map[string]*TopologyMetrics),
			StartTime:       time.Now(),
			LastActivity:    time.Now(),
		},
		startTime: time.Now(),
	}
}

// AddStream 添加流（用于测试兼容性）
func (e *StandardStreamEngine) AddStream(stream *Stream) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.streams[stream.GetName()] = stream
}

// GetStream 获取流（用于测试兼容性）
func (e *StandardStreamEngine) GetStream(name string) *Stream {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return e.streams[name] // 如果不存在会返回nil
}

// Start 启动引擎（用于测试兼容性）
func (e *StandardStreamEngine) Start(ctx context.Context) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	// 简单的启动逻辑，用于测试
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// 保持运行直到上下文取消
		<-ctx.Done()
		return nil
	}
}

// CreateTopology 创建流拓扑
func (e *StandardStreamEngine) CreateTopology(id, name, description string) (*StreamTopology, error) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if _, exists := e.topologies[id]; exists {
		return nil, fmt.Errorf("topology '%s' already exists", id)
	}

	topology := &StreamTopology{
		ID:          id,
		Name:        name,
		Description: description,
		Processors:  make(map[string]StreamProcessor),
		Connections: make([]StreamConnection, 0),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	e.topologies[id] = topology
	e.metrics.TopologyCount++
	e.metrics.StoppedCount++
	e.metrics.LastActivity = time.Now()

	// 发送事件
	e.sendEvent(StreamEventTopologyCreated, id, "", fmt.Sprintf("Topology '%s' created", name), nil)

	return topology, nil
}

// GetTopology 获取流拓扑
func (e *StandardStreamEngine) GetTopology(id string) (*StreamTopology, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	topology, exists := e.topologies[id]
	if !exists {
		return nil, fmt.Errorf("topology '%s' not found", id)
	}

	return topology, nil
}

// ListTopologies 列出所有流拓扑
func (e *StandardStreamEngine) ListTopologies() ([]*StreamTopology, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	topologies := make([]*StreamTopology, 0, len(e.topologies))
	for _, topology := range e.topologies {
		topologies = append(topologies, topology)
	}

	return topologies, nil
}

// DeleteTopology 删除流拓扑
func (e *StandardStreamEngine) DeleteTopology(id string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	topology, exists := e.topologies[id]
	if !exists {
		return fmt.Errorf("topology '%s' not found", id)
	}

	// 停止拓扑（如果正在运行）
	if cancel, running := e.runningTasks[id]; running {
		cancel()
		delete(e.runningTasks, id)
		e.metrics.RunningCount--
	}

	// 关闭所有连接通道
	for _, conn := range topology.Connections {
		if conn.Channel != nil {
			close(conn.Channel)
		}
	}

	delete(e.topologies, id)
	delete(e.metrics.TopologyMetrics, id)
	e.metrics.TopologyCount--
	e.metrics.StoppedCount--
	e.metrics.LastActivity = time.Now()

	// 发送事件
	e.sendEvent(StreamEventTopologyDeleted, id, "", fmt.Sprintf("Topology '%s' deleted", topology.Name), nil)

	return nil
}

// AddProcessor 添加处理器到拓扑
func (e *StandardStreamEngine) AddProcessor(topologyID string, processor StreamProcessor) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	topology, exists := e.topologies[topologyID]
	if !exists {
		return fmt.Errorf("topology '%s' not found", topologyID)
	}

	processorID := processor.GetID()
	if _, exists := topology.Processors[processorID]; exists {
		return fmt.Errorf("processor '%s' already exists in topology '%s'", processorID, topologyID)
	}

	topology.Processors[processorID] = processor
	topology.UpdatedAt = time.Now()
	e.metrics.LastActivity = time.Now()

	// 发送事件
	message := fmt.Sprintf("Processor '%s' added to topology '%s'", processor.GetName(), topology.Name)
	e.sendEvent(StreamEventProcessorAdded, topologyID, processorID, message, nil)

	return nil
}

// RemoveProcessor 从拓扑中移除处理器
func (e *StandardStreamEngine) RemoveProcessor(topologyID, processorID string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	topology, exists := e.topologies[topologyID]
	if !exists {
		return fmt.Errorf("topology '%s' not found", topologyID)
	}

	processor, exists := topology.Processors[processorID]
	if !exists {
		return fmt.Errorf("processor '%s' not found in topology '%s'", processorID, topologyID)
	}

	// 停止处理器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := processor.Stop(ctx); err != nil {
		// 记录停止错误但继续删除
	}

	// 移除相关连接
	newConnections := make([]StreamConnection, 0)
	for _, conn := range topology.Connections {
		if conn.From != processorID && conn.To != processorID {
			newConnections = append(newConnections, conn)
		} else if conn.Channel != nil {
			close(conn.Channel)
		}
	}
	topology.Connections = newConnections

	delete(topology.Processors, processorID)
	topology.UpdatedAt = time.Now()
	e.metrics.LastActivity = time.Now()

	// 发送事件
	message := fmt.Sprintf("Processor '%s' removed from topology '%s'", processor.GetName(), topology.Name)
	e.sendEvent(StreamEventProcessorRemoved, topologyID, processorID, message, nil)

	return nil
}

// ConnectProcessors 连接处理器
func (e *StandardStreamEngine) ConnectProcessors(topologyID, fromID, toID string, bufferSize int) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	topology, exists := e.topologies[topologyID]
	if !exists {
		return fmt.Errorf("topology '%s' not found", topologyID)
	}

	fromProcessor, exists := topology.Processors[fromID]
	if !exists {
		return fmt.Errorf("source processor '%s' not found in topology '%s'", fromID, topologyID)
	}

	toProcessor, exists := topology.Processors[toID]
	if !exists {
		return fmt.Errorf("target processor '%s' not found in topology '%s'", toID, topologyID)
	}

	// 检查连接是否已存在
	for _, conn := range topology.Connections {
		if conn.From == fromID && conn.To == toID {
			return fmt.Errorf("connection from '%s' to '%s' already exists", fromID, toID)
		}
	}

	// 创建通道
	channel := make(chan *Message, bufferSize)

	// 设置处理器的输入输出
	switch from := fromProcessor.(type) {
	case SourceProcessor:
		from.SetOutput(channel)
	case TransformProcessor:
		from.SetOutput(channel)
	default:
		return fmt.Errorf("processor '%s' cannot be a source in connection", fromID)
	}

	switch to := toProcessor.(type) {
	case TransformProcessor:
		to.SetInput(channel)
	case SinkProcessor:
		to.SetInput(channel)
	default:
		return fmt.Errorf("processor '%s' cannot be a target in connection", toID)
	}

	// 添加连接
	connection := StreamConnection{
		From:       fromID,
		To:         toID,
		Channel:    channel,
		BufferSize: bufferSize,
	}

	topology.Connections = append(topology.Connections, connection)
	topology.UpdatedAt = time.Now()
	e.metrics.LastActivity = time.Now()

	// 发送事件
	e.sendEvent(StreamEventConnectionAdded, topologyID, "", fmt.Sprintf("Connection added from '%s' to '%s'", fromID, toID), nil)

	return nil
}

// DisconnectProcessors 断开处理器连接
func (e *StandardStreamEngine) DisconnectProcessors(topologyID, fromID, toID string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	topology, exists := e.topologies[topologyID]
	if !exists {
		return fmt.Errorf("topology '%s' not found", topologyID)
	}

	// 查找并移除连接
	newConnections := make([]StreamConnection, 0)
	connectionFound := false

	for _, conn := range topology.Connections {
		if conn.From == fromID && conn.To == toID {
			connectionFound = true
			if conn.Channel != nil {
				close(conn.Channel)
			}
		} else {
			newConnections = append(newConnections, conn)
		}
	}

	if !connectionFound {
		return fmt.Errorf("connection from '%s' to '%s' not found", fromID, toID)
	}

	topology.Connections = newConnections
	topology.UpdatedAt = time.Now()
	e.metrics.LastActivity = time.Now()

	// 发送事件
	e.sendEvent(StreamEventConnectionRemoved, topologyID, "", fmt.Sprintf("Connection removed from '%s' to '%s'", fromID, toID), nil)

	return nil
}

// StartTopology 启动流拓扑
func (e *StandardStreamEngine) StartTopology(ctx context.Context, topologyID string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	topology, exists := e.topologies[topologyID]
	if !exists {
		return fmt.Errorf("topology '%s' not found", topologyID)
	}

	if _, running := e.runningTasks[topologyID]; running {
		return fmt.Errorf("topology '%s' is already running", topologyID)
	}

	// 创建拓扑上下文
	topologyCtx, cancel := context.WithCancel(ctx)
	e.runningTasks[topologyID] = cancel

	// 初始化拓扑指标
	e.metrics.TopologyMetrics[topologyID] = &TopologyMetrics{
		TopologyID:       topologyID,
		Status:           StreamStatusRunning,
		ProcessorCount:   len(topology.Processors),
		ConnectionCount:  len(topology.Connections),
		ProcessorMetrics: make(map[string]*StreamMetrics),
		StartTime:        time.Now(),
		LastActivity:     time.Now(),
	}

	// 启动所有处理器
	for processorID, processor := range topology.Processors {
		if err := processor.Start(topologyCtx); err != nil {
			// 如果启动失败，停止已启动的处理器
			cancel()
			delete(e.runningTasks, topologyID)
			delete(e.metrics.TopologyMetrics, topologyID)
			return fmt.Errorf("failed to start processor '%s': %v", processorID, err)
		}

		// 使用处理器自身的指标实例
		e.metrics.TopologyMetrics[topologyID].ProcessorMetrics[processorID] = processor.GetMetricsRef()
	}

	e.metrics.RunningCount++
	e.metrics.StoppedCount--
	e.metrics.LastActivity = time.Now()

	// 发送事件
	e.sendEvent(StreamEventTopologyStarted, topologyID, "", fmt.Sprintf("Topology '%s' started", topology.Name), nil)

	return nil
}

// StopTopology 停止流拓扑
func (e *StandardStreamEngine) StopTopology(ctx context.Context, topologyID string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	topology, exists := e.topologies[topologyID]
	if !exists {
		return fmt.Errorf("topology '%s' not found", topologyID)
	}

	cancel, running := e.runningTasks[topologyID]
	if !running {
		return fmt.Errorf("topology '%s' is not running", topologyID)
	}

	// 停止所有处理器
	for processorID, processor := range topology.Processors {
		if err := processor.Stop(ctx); err != nil {
			// 记录错误但继续停止其他处理器
			message := fmt.Sprintf("Error stopping processor '%s': %v", processorID, err)
			e.sendEvent(StreamEventProcessorError, topologyID, processorID, message, err)
		}
	}

	// 取消拓扑上下文
	cancel()
	delete(e.runningTasks, topologyID)

	// 更新指标
	if topologyMetrics, exists := e.metrics.TopologyMetrics[topologyID]; exists {
		topologyMetrics.Status = StreamStatusStopped
		topologyMetrics.Uptime = time.Since(topologyMetrics.StartTime)
	}

	e.metrics.RunningCount--
	e.metrics.StoppedCount++
	e.metrics.LastActivity = time.Now()

	// 发送事件
	e.sendEvent(StreamEventTopologyStopped, topologyID, "", fmt.Sprintf("Topology '%s' stopped", topology.Name), nil)

	return nil
}

// GetTopologyStatus 获取拓扑状态
func (e *StandardStreamEngine) GetTopologyStatus(topologyID string) (StreamStatus, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	if _, exists := e.topologies[topologyID]; !exists {
		return "", fmt.Errorf("topology '%s' not found", topologyID)
	}

	if _, running := e.runningTasks[topologyID]; running {
		return StreamStatusRunning, nil
	}

	return StreamStatusStopped, nil
}

// GetTopologyMetrics 获取拓扑指标
func (e *StandardStreamEngine) GetTopologyMetrics(topologyID string) (*TopologyMetrics, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	metrics, exists := e.metrics.TopologyMetrics[topologyID]
	if !exists {
		return nil, fmt.Errorf("topology '%s' metrics not found", topologyID)
	}

	// 返回指标副本
	metricsCopy := *metrics
	metricsCopy.ProcessorMetrics = make(map[string]*StreamMetrics)
	for id, pm := range metrics.ProcessorMetrics {
		metricsCopy.ProcessorMetrics[id] = pm.GetMetrics()
	}

	// 计算拓扑总计指标
	var totalMessages, totalBytes, totalErrors int64
	for _, pm := range metricsCopy.ProcessorMetrics {
		totalMessages += pm.MessagesIn + pm.MessagesOut
		totalBytes += pm.BytesIn + pm.BytesOut
		totalErrors += pm.ErrorCount
	}
	metricsCopy.TotalMessages = totalMessages
	metricsCopy.TotalBytes = totalBytes
	metricsCopy.TotalErrors = totalErrors
	metricsCopy.Uptime = time.Since(metricsCopy.StartTime)

	return &metricsCopy, nil
}

// GetEngineMetrics 获取引擎指标
func (e *StandardStreamEngine) GetEngineMetrics() (*EngineMetrics, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	// 计算总计指标
	var totalMessages, totalBytes, totalErrors int64
	for _, topologyMetrics := range e.metrics.TopologyMetrics {
		totalMessages += topologyMetrics.TotalMessages
		totalBytes += topologyMetrics.TotalBytes
		totalErrors += topologyMetrics.TotalErrors
	}

	// 返回指标副本
	metricsCopy := *e.metrics
	metricsCopy.TotalMessages = totalMessages
	metricsCopy.TotalBytes = totalBytes
	metricsCopy.TotalErrors = totalErrors
	metricsCopy.Uptime = time.Since(e.metrics.StartTime)
	metricsCopy.TopologyMetrics = make(map[string]*TopologyMetrics)

	for id, tm := range e.metrics.TopologyMetrics {
		tmCopy := *tm
		tmCopy.ProcessorMetrics = make(map[string]*StreamMetrics)
		for pid, pm := range tm.ProcessorMetrics {
			tmCopy.ProcessorMetrics[pid] = pm.GetMetrics()
		}
		metricsCopy.TopologyMetrics[id] = &tmCopy
	}

	return &metricsCopy, nil
}

// GetEventChannel 获取事件通道
func (e *StandardStreamEngine) GetEventChannel() <-chan *StreamEvent {
	return e.eventChan
}

// sendEvent 发送事件
func (e *StandardStreamEngine) sendEvent(eventType StreamEventType, topologyID, processorID, message string, data interface{}) {
	event := &StreamEvent{
		Type:        eventType,
		TopologyID:  topologyID,
		ProcessorID: processorID,
		Message:     message,
		Timestamp:   time.Now(),
		Data:        data,
	}

	select {
	case e.eventChan <- event:
	default:
		// 如果事件通道满了，丢弃事件
	}
}

// Close 关闭引擎
func (e *StandardStreamEngine) Close() error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	// 停止所有运行中的拓扑
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for topologyID := range e.runningTasks {
		if err := e.StopTopology(ctx, topologyID); err != nil {
			// 记录停止错误但不中断关闭过程
		}
	}

	// 关闭事件通道
	close(e.eventChan)

	return nil
}
