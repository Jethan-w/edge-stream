package sink

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/edge-stream/internal/MetricCollector"
	"github.com/edge-stream/internal/StateManager"
	"github.com/edge-stream/internal/flowfile"
	"github.com/edge-stream/internal/sink/ErrorRouter"
	"github.com/edge-stream/internal/sink/RetryManager"
)

// Sink 是数据流转的出口通道，负责将处理后的数据输出到外部目标系统
type Sink struct {
	mu sync.RWMutex

	// 配置相关
	config   *SinkConfig
	adapters map[string]TargetAdapter
	registry *CustomOutputProtocolRegistry

	// 状态管理
	state    SinkState
	stateMgr StateManager.StateManager

	// 错误处理
	errorHandler *SinkErrorHandler
	retryMgr     *RetryManager

	// 安全传输
	secureHandler *SecureTransferHandler

	// 监控
	metricCollector MetricCollector.MetricCollector

	// 上下文控制
	ctx    context.Context
	cancel context.CancelFunc
}

// SinkConfig Sink 配置
type SinkConfig struct {
	ID                   string                 `json:"id"`
	Name                 string                 `json:"name"`
	Description          string                 `json:"description"`
	OutputTargets        []OutputTargetConfig   `json:"output_targets"`
	SecurityConfig       *SecurityConfiguration `json:"security_config"`
	ReliabilityConfig    *ReliabilityConfig     `json:"reliability_config"`
	ErrorHandlingConfig  *ErrorHandlingConfig   `json:"error_handling_config"`
	MaxConcurrentOutputs int                    `json:"max_concurrent_outputs"`
	OutputTimeout        time.Duration          `json:"output_timeout"`
	EnableMetrics        bool                   `json:"enable_metrics"`
}

// OutputTargetConfig 输出目标配置
type OutputTargetConfig struct {
	ID               string                 `json:"id"`
	Type             OutputTargetType       `json:"type"`
	ConnectionString string                 `json:"connection_string"`
	Properties       map[string]string      `json:"properties"`
	SecurityConfig   *SecurityConfiguration `json:"security_config"`
	RetryConfig      *RetryConfig           `json:"retry_config"`
}

// SinkState Sink 状态
type SinkState int

const (
	SinkStateUnconfigured SinkState = iota
	SinkStateConfiguring
	SinkStateReady
	SinkStateRunning
	SinkStatePaused
	SinkStateError
	SinkStateStopped
)

func (s SinkState) String() string {
	switch s {
	case SinkStateUnconfigured:
		return "Unconfigured"
	case SinkStateConfiguring:
		return "Configuring"
	case SinkStateReady:
		return "Ready"
	case SinkStateRunning:
		return "Running"
	case SinkStatePaused:
		return "Paused"
	case SinkStateError:
		return "Error"
	case SinkStateStopped:
		return "Stopped"
	default:
		return "Unknown"
	}
}

// NewSink 创建新的 Sink 实例
func NewSink(config *SinkConfig) *Sink {
	ctx, cancel := context.WithCancel(context.Background())

	sink := &Sink{
		config:        config,
		adapters:      make(map[string]TargetAdapter),
		registry:      NewCustomOutputProtocolRegistry(),
		state:         SinkStateUnconfigured,
		errorHandler:  NewSinkErrorHandler(),
		retryMgr:      NewRetryManager(),
		secureHandler: NewSecureTransferHandler(),
		ctx:           ctx,
		cancel:        cancel,
	}

	return sink
}

// Initialize 初始化 Sink
func (s *Sink) Initialize() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.state = SinkStateConfiguring

	// 初始化安全传输处理器
	if err := s.secureHandler.Initialize(s.config.SecurityConfig); err != nil {
		s.state = SinkStateError
		return fmt.Errorf("failed to initialize secure handler: %w", err)
	}

	// 初始化错误处理器
	if err := s.errorHandler.Initialize(s.config.ErrorHandlingConfig); err != nil {
		s.state = SinkStateError
		return fmt.Errorf("failed to initialize error handler: %w", err)
	}

	// 初始化重试管理器
	if err := s.retryMgr.Initialize(s.config.ReliabilityConfig); err != nil {
		s.state = SinkStateError
		return fmt.Errorf("failed to initialize retry manager: %w", err)
	}

	// 初始化输出目标适配器
	for _, targetConfig := range s.config.OutputTargets {
		adapter, err := s.createTargetAdapter(targetConfig)
		if err != nil {
			s.state = SinkStateError
			return fmt.Errorf("failed to create adapter for target %s: %w", targetConfig.ID, err)
		}
		s.adapters[targetConfig.ID] = adapter
	}

	s.state = SinkStateReady
	return nil
}

// Start 启动 Sink
func (s *Sink) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != SinkStateReady {
		return fmt.Errorf("sink is not ready, current state: %s", s.state)
	}

	s.state = SinkStateRunning

	// 启动监控
	if s.config.EnableMetrics && s.metricCollector != nil {
		s.metricCollector.Start()
	}

	return nil
}

// Stop 停止 Sink
func (s *Sink) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == SinkStateStopped {
		return nil
	}

	s.cancel()
	s.state = SinkStateStopped

	// 停止监控
	if s.metricCollector != nil {
		s.metricCollector.Stop()
	}

	// 关闭所有适配器
	for _, adapter := range s.adapters {
		if closer, ok := adapter.(interface{ Close() error }); ok {
			closer.Close()
		}
	}

	return nil
}

// Pause 暂停 Sink
func (s *Sink) Pause() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != SinkStateRunning {
		return fmt.Errorf("sink is not running, current state: %s", s.state)
	}

	s.state = SinkStatePaused
	return nil
}

// Resume 恢复 Sink
func (s *Sink) Resume() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != SinkStatePaused {
		return fmt.Errorf("sink is not paused, current state: %s", s.state)
	}

	s.state = SinkStateRunning
	return nil
}

// Write 写入数据到目标系统
func (s *Sink) Write(flowFile *flowfile.FlowFile, targetID string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.state != SinkStateRunning {
		return fmt.Errorf("sink is not running, current state: %s", s.state)
	}

	adapter, exists := s.adapters[targetID]
	if !exists {
		return fmt.Errorf("target adapter not found: %s", targetID)
	}

	// 记录开始时间
	startTime := time.Now()

	// 执行输出操作
	err := s.executeOutput(flowFile, adapter)

	// 记录指标
	if s.metricCollector != nil {
		duration := time.Since(startTime)
		s.metricCollector.RecordMetric("sink_output_duration", duration.Milliseconds())

		if err != nil {
			s.metricCollector.RecordMetric("sink_output_errors", 1)
		} else {
			s.metricCollector.RecordMetric("sink_output_success", 1)
		}
	}

	return err
}

// executeOutput 执行输出操作
func (s *Sink) executeOutput(flowFile *flowfile.FlowFile, adapter TargetAdapter) error {
	// 创建输出操作
	operation := &OutputOperation{
		flowFile: flowFile,
		adapter:  adapter,
		sink:     s,
	}

	// 使用重试管理器执行
	return s.retryMgr.ExecuteWithRetry(operation)
}

// createTargetAdapter 创建目标适配器
func (s *Sink) createTargetAdapter(config OutputTargetConfig) (TargetAdapter, error) {
	switch config.Type {
	case OutputTargetTypeFileSystem:
		return NewFileSystemOutputAdapter(config), nil
	case OutputTargetTypeDatabase:
		return NewDatabaseOutputAdapter(config), nil
	case OutputTargetTypeMessageQueue:
		return NewMessageQueueOutputAdapter(config), nil
	case OutputTargetTypeSearchEngine:
		return NewSearchEngineOutputAdapter(config), nil
	case OutputTargetTypeRESTAPI:
		return NewRESTAPIOutputAdapter(config), nil
	case OutputTargetTypeCustom:
		return s.registry.CreateAdapter(config.Properties["protocol"])
	default:
		return nil, fmt.Errorf("unsupported output target type: %s", config.Type)
	}
}

// GetState 获取当前状态
func (s *Sink) GetState() SinkState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// GetConfig 获取配置
func (s *Sink) GetConfig() *SinkConfig {
	return s.config
}

// RegisterCustomProtocol 注册自定义输出协议
func (s *Sink) RegisterCustomProtocol(name string, adapter TargetAdapter) {
	s.registry.RegisterProtocol(name, func(config map[string]string) (TargetAdapter, error) {
		return adapter, nil
	})
}

// GetErrorHandler 获取错误处理器
func (s *Sink) GetErrorHandler() *ErrorRouter.SinkErrorHandler {
	return s.errorHandler
}

// GetRetryManager 获取重试管理器
func (s *Sink) GetRetryManager() *RetryManager.RetryManager {
	return s.retryMgr
}
