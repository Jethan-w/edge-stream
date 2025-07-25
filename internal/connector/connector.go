package connector

import (
	"context"
	"sync"
	"time"
)

// ConnectorType 连接器类型
type ConnectorType string

const (
	ConnectorTypeSource    ConnectorType = "source"
	ConnectorTypeSink      ConnectorType = "sink"
	ConnectorTypeTransform ConnectorType = "transform"
)

// ConnectorStatus 连接器状态
type ConnectorStatus string

const (
	ConnectorStatusStopped  ConnectorStatus = "stopped"
	ConnectorStatusStarting ConnectorStatus = "starting"
	ConnectorStatusRunning  ConnectorStatus = "running"
	ConnectorStatusStopping ConnectorStatus = "stopping"
	ConnectorStatusError    ConnectorStatus = "error"
)

// DataFormat 数据格式
type DataFormat string

const (
	DataFormatJSON   DataFormat = "json"
	DataFormatAvro   DataFormat = "avro"
	DataFormatCSV    DataFormat = "csv"
	DataFormatBinary DataFormat = "binary"
	DataFormatText   DataFormat = "text"
)

// Message 消息接口
type Message interface {
	// GetKey 获取消息键
	GetKey() string
	// GetValue 获取消息值
	GetValue() interface{}
	// GetHeaders 获取消息头
	GetHeaders() map[string]string
	// GetTimestamp 获取时间戳
	GetTimestamp() time.Time
	// GetPartition 获取分区
	GetPartition() int
	// GetOffset 获取偏移量
	GetOffset() int64
}

// Connector 连接器接口
type Connector interface {
	// GetInfo 获取连接器信息
	GetInfo() ConnectorInfo
	// Start 启动连接器
	Start(ctx context.Context) error
	// Stop 停止连接器
	Stop(ctx context.Context) error
	// GetStatus 获取状态
	GetStatus() ConnectorStatus
	// GetMetrics 获取指标
	GetMetrics() ConnectorMetrics
	// Validate 验证配置
	Validate(config map[string]interface{}) error
	// Configure 配置连接器
	Configure(config map[string]interface{}) error
}

// SourceConnector 源连接器接口
type SourceConnector interface {
	Connector
	// Read 读取数据
	Read(ctx context.Context) (<-chan Message, <-chan error)
	// Commit 提交偏移量
	Commit(ctx context.Context, offset int64) error
	// GetPartitions 获取分区信息
	GetPartitions() []PartitionInfo
}

// SinkConnector 接收器连接器接口
type SinkConnector interface {
	Connector
	// Write 写入数据
	Write(ctx context.Context, messages <-chan Message) error
	// Flush 刷新缓冲区
	Flush(ctx context.Context) error
	// GetWriteCapacity 获取写入容量
	GetWriteCapacity() int
}

// TransformConnector 转换连接器接口
type TransformConnector interface {
	Connector
	// Transform 转换数据
	Transform(ctx context.Context, input <-chan Message) (<-chan Message, <-chan error)
	// GetTransformRules 获取转换规则
	GetTransformRules() []TransformRule
}

// ConnectorInfo 连接器信息
type ConnectorInfo struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        ConnectorType          `json:"type"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Author      string                 `json:"author"`
	Config      map[string]interface{} `json:"config"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// ConnectorMetrics 连接器指标
type ConnectorMetrics struct {
	MessagesProcessed int64         `json:"messages_processed"`
	BytesProcessed    int64         `json:"bytes_processed"`
	ErrorsCount       int64         `json:"errors_count"`
	AverageLatency    time.Duration `json:"average_latency"`
	MaxLatency        time.Duration `json:"max_latency"`
	Throughput        float64       `json:"throughput"` // messages per second
	Uptime            time.Duration `json:"uptime"`
	LastActivity      time.Time     `json:"last_activity"`
}

// PartitionInfo 分区信息
type PartitionInfo struct {
	ID       int      `json:"id"`
	Offset   int64    `json:"offset"`
	Size     int64    `json:"size"`
	Leader   string   `json:"leader"`
	Replicas []string `json:"replicas"`
}

// TransformRule 转换规则
type TransformRule struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Condition  string                 `json:"condition"`  // 条件表达式
	Action     string                 `json:"action"`     // 转换动作
	Parameters map[string]interface{} `json:"parameters"` // 参数
	Enabled    bool                   `json:"enabled"`
	Priority   int                    `json:"priority"`
}

// ConnectorRegistry 连接器注册中心接口
type ConnectorRegistry interface {
	// Register 注册连接器
	Register(connectorType ConnectorType, name string, factory ConnectorFactory) error
	// Unregister 注销连接器
	Unregister(connectorType ConnectorType, name string) error
	// Create 创建连接器实例
	Create(connectorType ConnectorType, name string, config map[string]interface{}) (Connector, error)
	// List 列出已注册的连接器
	List(connectorType ConnectorType) []string
	// GetInfo 获取连接器信息
	GetInfo(connectorType ConnectorType, name string) (*ConnectorInfo, error)
	// Validate 验证连接器配置
	Validate(connectorType ConnectorType, name string, config map[string]interface{}) error
}

// ConnectorFactory 连接器工厂接口
type ConnectorFactory interface {
	// Create 创建连接器实例
	Create(config map[string]interface{}) (Connector, error)
	// GetInfo 获取连接器信息
	GetInfo() ConnectorInfo
	// ValidateConfig 验证配置
	ValidateConfig(config map[string]interface{}) error
}

// ConnectorManager 连接器管理器接口
type ConnectorManager interface {
	// CreateConnector 创建连接器
	CreateConnector(ctx context.Context, id string, connectorType ConnectorType, name string, config map[string]interface{}) error
	// StartConnector 启动连接器
	StartConnector(ctx context.Context, id string) error
	// StopConnector 停止连接器
	StopConnector(ctx context.Context, id string) error
	// DeleteConnector 删除连接器
	DeleteConnector(ctx context.Context, id string) error
	// GetConnector 获取连接器
	GetConnector(id string) (Connector, bool)
	// ListConnectors 列出所有连接器
	ListConnectors() []string
	// GetConnectorStatus 获取连接器状态
	GetConnectorStatus(id string) (ConnectorStatus, error)
	// GetConnectorMetrics 获取连接器指标
	GetConnectorMetrics(id string) (*ConnectorMetrics, error)
	// RestartConnector 重启连接器
	RestartConnector(ctx context.Context, id string) error
	// UpdateConnectorConfig 更新连接器配置
	UpdateConnectorConfig(ctx context.Context, id string, config map[string]interface{}) error
}

// ConnectorEvent 连接器事件
type ConnectorEvent struct {
	Type        ConnectorEventType     `json:"type"`
	ConnectorID string                 `json:"connector_id"`
	Timestamp   time.Time              `json:"timestamp"`
	Message     string                 `json:"message"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ConnectorEventType 连接器事件类型
type ConnectorEventType string

const (
	ConnectorEventCreated ConnectorEventType = "created"
	ConnectorEventStarted ConnectorEventType = "started"
	ConnectorEventStopped ConnectorEventType = "stopped"
	ConnectorEventError   ConnectorEventType = "error"
	ConnectorEventDeleted ConnectorEventType = "deleted"
	ConnectorEventUpdated ConnectorEventType = "updated"
)

// ConnectorConfig 连接器配置
type ConnectorConfig struct {
	// MaxRetries 最大重试次数
	MaxRetries int `yaml:"max_retries" json:"max_retries"`
	// RetryInterval 重试间隔
	RetryInterval time.Duration `yaml:"retry_interval" json:"retry_interval"`
	// HealthCheckInterval 健康检查间隔
	HealthCheckInterval time.Duration `yaml:"health_check_interval" json:"health_check_interval"`
	// MetricsInterval 指标收集间隔
	MetricsInterval time.Duration `yaml:"metrics_interval" json:"metrics_interval"`
	// BufferSize 缓冲区大小
	BufferSize int `yaml:"buffer_size" json:"buffer_size"`
	// BatchSize 批处理大小
	BatchSize int `yaml:"batch_size" json:"batch_size"`
	// Timeout 超时时间
	Timeout time.Duration `yaml:"timeout" json:"timeout"`
}

// DefaultConnectorConfig 默认连接器配置
func DefaultConnectorConfig() *ConnectorConfig {
	return &ConnectorConfig{
		MaxRetries:          3,
		RetryInterval:       5 * time.Second,
		HealthCheckInterval: 30 * time.Second,
		MetricsInterval:     10 * time.Second,
		BufferSize:          1000,
		BatchSize:           100,
		Timeout:             30 * time.Second,
	}
}

// StandardMessage 标准消息实现
type StandardMessage struct {
	key       string
	value     interface{}
	headers   map[string]string
	timestamp time.Time
	partition int
	offset    int64
}

// NewStandardMessage 创建标准消息
func NewStandardMessage(key string, value interface{}) *StandardMessage {
	return &StandardMessage{
		key:       key,
		value:     value,
		headers:   make(map[string]string),
		timestamp: time.Now(),
		partition: 0,
		offset:    0,
	}
}

// GetKey 获取消息键
func (m *StandardMessage) GetKey() string {
	return m.key
}

// GetValue 获取消息值
func (m *StandardMessage) GetValue() interface{} {
	return m.value
}

// GetHeaders 获取消息头
func (m *StandardMessage) GetHeaders() map[string]string {
	return m.headers
}

// GetTimestamp 获取时间戳
func (m *StandardMessage) GetTimestamp() time.Time {
	return m.timestamp
}

// GetPartition 获取分区
func (m *StandardMessage) GetPartition() int {
	return m.partition
}

// GetOffset 获取偏移量
func (m *StandardMessage) GetOffset() int64 {
	return m.offset
}

// SetHeader 设置消息头
func (m *StandardMessage) SetHeader(key, value string) {
	m.headers[key] = value
}

// SetPartition 设置分区
func (m *StandardMessage) SetPartition(partition int) {
	m.partition = partition
}

// SetOffset 设置偏移量
func (m *StandardMessage) SetOffset(offset int64) {
	m.offset = offset
}

// ConnectorInstance 连接器实例
type ConnectorInstance struct {
	ID        string
	Connector Connector
	Config    map[string]interface{}
	Status    ConnectorStatus
	CreatedAt time.Time
	StartedAt time.Time
	StoppedAt time.Time
	ErrorMsg  string
	mutex     sync.RWMutex
}

// NewConnectorInstance 创建连接器实例
func NewConnectorInstance(id string, connector Connector, config map[string]interface{}) *ConnectorInstance {
	return &ConnectorInstance{
		ID:        id,
		Connector: connector,
		Config:    config,
		Status:    ConnectorStatusStopped,
		CreatedAt: time.Now(),
	}
}

// GetStatus 获取状态
func (ci *ConnectorInstance) GetStatus() ConnectorStatus {
	ci.mutex.RLock()
	defer ci.mutex.RUnlock()
	return ci.Status
}

// SetStatus 设置状态
func (ci *ConnectorInstance) SetStatus(status ConnectorStatus) {
	ci.mutex.Lock()
	defer ci.mutex.Unlock()
	ci.Status = status
	if status == ConnectorStatusRunning {
		ci.StartedAt = time.Now()
	} else if status == ConnectorStatusStopped {
		ci.StoppedAt = time.Now()
	}
}

// SetError 设置错误
func (ci *ConnectorInstance) SetError(err error) {
	ci.mutex.Lock()
	defer ci.mutex.Unlock()
	ci.Status = ConnectorStatusError
	if err != nil {
		ci.ErrorMsg = err.Error()
	} else {
		ci.ErrorMsg = ""
	}
}

// GetError 获取错误信息
func (ci *ConnectorInstance) GetError() string {
	ci.mutex.RLock()
	defer ci.mutex.RUnlock()
	return ci.ErrorMsg
}

// GetUptime 获取运行时间
func (ci *ConnectorInstance) GetUptime() time.Duration {
	ci.mutex.RLock()
	defer ci.mutex.RUnlock()
	if ci.Status == ConnectorStatusRunning && !ci.StartedAt.IsZero() {
		return time.Since(ci.StartedAt)
	}
	return 0
}
