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
	"sync"
	"time"
)

// StreamType 流类型
type StreamType string

const (
	StreamTypeSource    StreamType = "source"
	StreamTypeTransform StreamType = "transform"
	StreamTypeSink      StreamType = "sink"
)

// StreamStatus 流状态
type StreamStatus string

const (
	StreamStatusStopped StreamStatus = "stopped"
	StreamStatusRunning StreamStatus = "running"
	StreamStatusPaused  StreamStatus = "paused"
	StreamStatusError   StreamStatus = "error"
)

// WindowType 窗口类型
type WindowType string

const (
	WindowTypeTumbling WindowType = "tumbling"
	WindowTypeSliding  WindowType = "sliding"
	WindowTypeSession  WindowType = "session"
)

// AggregationType 聚合类型
type AggregationType string

const (
	AggregationTypeSum   AggregationType = "sum"
	AggregationTypeCount AggregationType = "count"
	AggregationTypeAvg   AggregationType = "avg"
	AggregationTypeMin   AggregationType = "min"
	AggregationTypeMax   AggregationType = "max"
)

// Message 流消息
type Message struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Data      interface{}            `json:"data"`
	Headers   map[string]interface{} `json:"headers"`
	Partition string                 `json:"partition"`
	Offset    int64                  `json:"offset"`
}

// Window 窗口
type Window struct {
	ID        string        `json:"id"`
	Type      WindowType    `json:"type"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Size      time.Duration `json:"size"`
	Slide     time.Duration `json:"slide"`
	Messages  []*Message    `json:"messages"`
}

// AggregationResult 聚合结果
type AggregationResult struct {
	Window    *Window         `json:"window"`
	Type      AggregationType `json:"type"`
	Field     string          `json:"field"`
	Value     interface{}     `json:"value"`
	Count     int64           `json:"count"`
	Timestamp time.Time       `json:"timestamp"`
}

// StreamProcessor 流处理器接口
type StreamProcessor interface {
	Process(ctx context.Context, message *Message) ([]*Message, error)
	GetType() StreamType
	GetID() string
	GetName() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	GetStatus() StreamStatus
	GetMetrics() *StreamMetrics
	GetMetricsRef() *StreamMetrics // 获取指标的直接引用
}

// SourceProcessor 源处理器接口
type SourceProcessor interface {
	StreamProcessor
	Read(ctx context.Context) (*Message, error)
	SetOutput(output chan<- *Message)
}

// TransformProcessor 转换处理器接口
type TransformProcessor interface {
	StreamProcessor
	Transform(ctx context.Context, message *Message) ([]*Message, error)
	SetInput(input <-chan *Message)
	SetOutput(output chan<- *Message)
}

// SinkProcessor 接收器处理器接口
type SinkProcessor interface {
	StreamProcessor
	Write(ctx context.Context, message *Message) error
	SetInput(input <-chan *Message)
}

// WindowProcessor 窗口处理器接口
type WindowProcessor interface {
	AddMessage(message *Message) error
	GetWindow(id string) (*Window, error)
	GetActiveWindows() ([]*Window, error)
	CloseExpiredWindows(now time.Time) ([]*Window, error)
	Aggregate(window *Window, aggType AggregationType, field string) (*AggregationResult, error)
}

// StreamTopology 流拓扑
type StreamTopology struct {
	ID          string                     `json:"id"`
	Name        string                     `json:"name"`
	Description string                     `json:"description"`
	Processors  map[string]StreamProcessor `json:"processors"`
	Connections []StreamConnection         `json:"connections"`
	CreatedAt   time.Time                  `json:"created_at"`
	UpdatedAt   time.Time                  `json:"updated_at"`
}

// StreamConnection 流连接
type StreamConnection struct {
	From       string        `json:"from"`
	To         string        `json:"to"`
	Channel    chan *Message `json:"-"`
	BufferSize int           `json:"buffer_size"`
}

// StreamEngine 流处理引擎接口
type StreamEngine interface {
	CreateTopology(id, name, description string) (*StreamTopology, error)
	GetTopology(id string) (*StreamTopology, error)
	ListTopologies() ([]*StreamTopology, error)
	DeleteTopology(id string) error
	AddProcessor(topologyID string, processor StreamProcessor) error
	RemoveProcessor(topologyID, processorID string) error
	ConnectProcessors(topologyID, fromID, toID string, bufferSize int) error
	DisconnectProcessors(topologyID, fromID, toID string) error
	StartTopology(ctx context.Context, topologyID string) error
	StopTopology(ctx context.Context, topologyID string) error
	GetTopologyStatus(topologyID string) (StreamStatus, error)
	GetTopologyMetrics(topologyID string) (*TopologyMetrics, error)
	GetEngineMetrics() (*EngineMetrics, error)
}

// StreamMetrics 流指标
type StreamMetrics struct {
	ProcessorID    string        `json:"processor_id"`
	MessagesIn     int64         `json:"messages_in"`
	MessagesOut    int64         `json:"messages_out"`
	BytesIn        int64         `json:"bytes_in"`
	BytesOut       int64         `json:"bytes_out"`
	ErrorCount     int64         `json:"error_count"`
	ProcessingTime time.Duration `json:"processing_time"`
	Throughput     float64       `json:"throughput"`
	Latency        time.Duration `json:"latency"`
	LastActivity   time.Time     `json:"last_activity"`
	StartTime      time.Time     `json:"start_time"`
	mutex          sync.RWMutex
}

// TopologyMetrics 拓扑指标
type TopologyMetrics struct {
	TopologyID       string                    `json:"topology_id"`
	Status           StreamStatus              `json:"status"`
	ProcessorCount   int                       `json:"processor_count"`
	ConnectionCount  int                       `json:"connection_count"`
	TotalMessages    int64                     `json:"total_messages"`
	TotalBytes       int64                     `json:"total_bytes"`
	TotalErrors      int64                     `json:"total_errors"`
	Uptime           time.Duration             `json:"uptime"`
	ProcessorMetrics map[string]*StreamMetrics `json:"processor_metrics"`
	StartTime        time.Time                 `json:"start_time"`
	LastActivity     time.Time                 `json:"last_activity"`
}

// EngineMetrics 引擎指标
type EngineMetrics struct {
	TopologyCount   int                         `json:"topology_count"`
	RunningCount    int                         `json:"running_count"`
	StoppedCount    int                         `json:"stopped_count"`
	ErrorCount      int                         `json:"error_count"`
	TotalMessages   int64                       `json:"total_messages"`
	TotalBytes      int64                       `json:"total_bytes"`
	TotalErrors     int64                       `json:"total_errors"`
	Uptime          time.Duration               `json:"uptime"`
	TopologyMetrics map[string]*TopologyMetrics `json:"topology_metrics"`
	StartTime       time.Time                   `json:"start_time"`
	LastActivity    time.Time                   `json:"last_activity"`
}

// WindowConfig 窗口配置
type WindowConfig struct {
	Type       WindowType    `json:"type"`
	Size       time.Duration `json:"size"`
	Slide      time.Duration `json:"slide"`
	SessionGap time.Duration `json:"session_gap"`
	MaxSize    int           `json:"max_size"`
	Watermark  time.Duration `json:"watermark"`
}

// TopologyConfig 拓扑配置
type TopologyConfig struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Processors  []*ProcessorConfig     `json:"processors"`
	Connections []StreamConnection     `json:"connections"`
	Properties  map[string]interface{} `json:"properties"`
}

// StreamEvent 流事件
type StreamEvent struct {
	Type        StreamEventType `json:"type"`
	TopologyID  string          `json:"topology_id"`
	ProcessorID string          `json:"processor_id"`
	Message     string          `json:"message"`
	Timestamp   time.Time       `json:"timestamp"`
	Data        interface{}     `json:"data"`
}

// StreamEventType 流事件类型
type StreamEventType string

const (
	StreamEventTopologyCreated   StreamEventType = "topology_created"
	StreamEventTopologyStarted   StreamEventType = "topology_started"
	StreamEventTopologyStopped   StreamEventType = "topology_stopped"
	StreamEventTopologyDeleted   StreamEventType = "topology_deleted"
	StreamEventProcessorAdded    StreamEventType = "processor_added"
	StreamEventProcessorRemoved  StreamEventType = "processor_removed"
	StreamEventProcessorError    StreamEventType = "processor_error"
	StreamEventConnectionAdded   StreamEventType = "connection_added"
	StreamEventConnectionRemoved StreamEventType = "connection_removed"
)

// UpdateMetrics 更新指标
func (m *StreamMetrics) UpdateMetrics(messagesIn, messagesOut, bytesIn, bytesOut int64, processingTime time.Duration, errorCount int64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.MessagesIn += messagesIn
	m.MessagesOut += messagesOut
	m.BytesIn += bytesIn
	m.BytesOut += bytesOut
	m.ErrorCount += errorCount
	m.ProcessingTime += processingTime
	m.LastActivity = time.Now()

	// 计算吞吐量 (消息/秒)
	if duration := time.Since(m.StartTime).Seconds(); duration > 0 {
		m.Throughput = float64(m.MessagesOut) / duration
	}

	// 计算平均延迟
	if m.MessagesOut > 0 {
		m.Latency = m.ProcessingTime / time.Duration(m.MessagesOut)
	}
}

// GetMetrics 获取指标副本
func (m *StreamMetrics) GetMetrics() *StreamMetrics {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return &StreamMetrics{
		ProcessorID:    m.ProcessorID,
		MessagesIn:     m.MessagesIn,
		MessagesOut:    m.MessagesOut,
		BytesIn:        m.BytesIn,
		BytesOut:       m.BytesOut,
		ErrorCount:     m.ErrorCount,
		ProcessingTime: m.ProcessingTime,
		Throughput:     m.Throughput,
		Latency:        m.Latency,
		LastActivity:   m.LastActivity,
		StartTime:      m.StartTime,
	}
}

// NewStreamMetrics 创建新的流指标
func NewStreamMetrics(processorID string) *StreamMetrics {
	return &StreamMetrics{
		ProcessorID:  processorID,
		StartTime:    time.Now(),
		LastActivity: time.Now(),
		mutex:        sync.RWMutex{},
	}
}

// Stream 简单流结构体（用于测试）
type Stream struct {
	name       string
	processors []StreamProcessor
	mutex      sync.RWMutex
}

// NewStream 创建新的流
func NewStream(name string) *Stream {
	return &Stream{
		name:       name,
		processors: make([]StreamProcessor, 0),
		mutex:      sync.RWMutex{},
	}
}

// GetName 获取流名称
func (s *Stream) GetName() string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.name
}

// AddProcessor 添加处理器
func (s *Stream) AddProcessor(processor StreamProcessor) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.processors = append(s.processors, processor)
}

// FlowFileProcessor 定义了处理FlowFile的接口
type FlowFileProcessor interface {
	ProcessFlowFile(interface{}) (interface{}, error)
}

// Process 处理流文件
func (s *Stream) Process(ff interface{}) interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// 如果没有处理器，直接返回输入
	if len(s.processors) == 0 {
		return ff
	}

	// 简单的处理逻辑，用于测试
	result := ff
	for _, processor := range s.processors {
		if testProcessor, ok := processor.(FlowFileProcessor); ok {
			processed, err := testProcessor.ProcessFlowFile(result)
			if err != nil {
				return nil // 返回nil表示处理失败
			}
			result = processed
		}
	}
	return result
}
