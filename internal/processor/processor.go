package processor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/crazy/edge-stream/internal/flowfile"
)

// Processor 处理器接口定义
type Processor interface {
	// Initialize 初始化处理器
	Initialize(ctx ProcessContext) error

	// Process 处理数据
	Process(ctx ProcessContext, flowFile *flowfile.FlowFile) (*flowfile.FlowFile, error)

	// GetRelationships 获取输出关系
	GetRelationships() []Relationship

	// GetPropertyDescriptors 获取属性描述符
	GetPropertyDescriptors() []PropertyDescriptor

	// GetState 获取处理器状态
	GetState() ProcessorState
	Stop(ctx ProcessContext) error
	Start(ctx ProcessContext) error
}

// ProcessorState 处理器状态
type ProcessorState int

const (
	StateUnconfigured ProcessorState = iota
	StateConfiguring
	StateReady
	StateRunning
	StatePaused
	StateError
	StateStopped
)

// Relationship 输出关系
type Relationship struct {
	Name        string
	Description string
}

// PropertyDescriptor 属性描述符
type PropertyDescriptor struct {
	Name         string
	Description  string
	Required     bool
	DefaultValue string
	Dynamic      bool
}

// ProcessContext 处理上下文
type ProcessContext struct {
	context.Context
	Properties map[string]string
	Session    *ProcessSession
	State      map[string]interface{}
}

// ProcessSession 处理会话
type ProcessSession struct {
	mu sync.RWMutex
	// 会话相关状态
}

// AbstractProcessor 抽象处理器基类
type AbstractProcessor struct {
	relationships       []Relationship
	propertyDescriptors []PropertyDescriptor
	state               ProcessorState
	mu                  sync.RWMutex
}

// Initialize 初始化抽象处理器
func (ap *AbstractProcessor) Initialize(ctx ProcessContext) error {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	ap.state = StateConfiguring

	// 初始化默认关系
	ap.relationships = []Relationship{
		{Name: "success", Description: "成功处理"},
		{Name: "failure", Description: "处理失败"},
	}

	// 初始化默认属性描述符
	ap.propertyDescriptors = []PropertyDescriptor{
		{Name: "name", Description: "处理器名称", Required: true},
	}

	ap.state = StateReady
	return nil
}

// GetRelationships 获取输出关系
func (ap *AbstractProcessor) GetRelationships() []Relationship {
	ap.mu.RLock()
	defer ap.mu.RUnlock()
	return ap.relationships
}

// GetPropertyDescriptors 获取属性描述符
func (ap *AbstractProcessor) GetPropertyDescriptors() []PropertyDescriptor {
	ap.mu.RLock()
	defer ap.mu.RUnlock()
	return ap.propertyDescriptors
}

// GetState 获取处理器状态
func (ap *AbstractProcessor) GetState() ProcessorState {
	ap.mu.RLock()
	defer ap.mu.RUnlock()
	return ap.state
}

// SetState 设置处理器状态
func (ap *AbstractProcessor) SetState(state ProcessorState) {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	ap.state = state
}

// AddRelationship 添加输出关系
func (ap *AbstractProcessor) AddRelationship(name, description string) {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	ap.relationships = append(ap.relationships, Relationship{
		Name:        name,
		Description: description,
	})
}

// AddPropertyDescriptor 添加属性描述符
func (ap *AbstractProcessor) AddPropertyDescriptor(name, description string, required bool) {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	ap.propertyDescriptors = append(ap.propertyDescriptors, PropertyDescriptor{
		Name:        name,
		Description: description,
		Required:    required,
	})
}

// DataTransformer 数据转换接口
type DataTransformer interface {
	Transform(input *flowfile.FlowFile, context *ProcessContext) (*flowfile.FlowFile, error)
	ConvertFormat(input *flowfile.FlowFile, sourceFormat, targetFormat string) (*flowfile.FlowFile, error)
	ModifyContent(input *flowfile.FlowFile, strategy ContentModificationStrategy) (*flowfile.FlowFile, error)
}

// ContentModificationStrategy 内容修改策略
type ContentModificationStrategy int

const (
	StrategyReplace ContentModificationStrategy = iota
	StrategyAppend
	StrategyPrepend
	StrategyRegexReplace
	StrategyTransform
)

// SemanticExtractor 语义提取接口
type SemanticExtractor interface {
	ExtractTextSemantics(text string) map[string]string
	ExtractJsonSemantics(jsonContent string) map[string]interface{}
	ExtractXmlSemantics(xmlContent string) map[string]interface{}
}

// ErrorHandlingStrategy 错误处理策略
type ErrorHandlingStrategy int

const (
	StrategyRouteToFailure ErrorHandlingStrategy = iota
	StrategyRouteToSuccess
	StrategyRouteToRetry
	StrategyDrop
)

// RetryStrategy 重试策略
type RetryStrategy struct {
	MaxRetries int
	BaseDelay  time.Duration
}

// CalculateRetryDelay 计算重试延迟
func (rs *RetryStrategy) CalculateRetryDelay(retryCount int) time.Duration {
	if retryCount >= rs.MaxRetries {
		return 0
	}

	// 指数退避策略
	delay := rs.BaseDelay * time.Duration(1<<retryCount)
	return delay
}

// ShouldRetry 判断是否应该重试
func (rs *RetryStrategy) ShouldRetry(retryCount int, err error) bool {
	return retryCount < rs.MaxRetries && rs.isRetryableError(err)
}

// isRetryableError 判断是否为可重试错误
func (rs *RetryStrategy) isRetryableError(err error) bool {
	// 根据错误类型判断是否可重试
	// 这里可以根据实际需求实现更复杂的错误分类逻辑
	return err != nil
}

// ProcessorManager 处理器管理器
type ProcessorManager struct {
	processors map[string]Processor
	mu         sync.RWMutex
}

// NewProcessorManager 创建处理器管理器
func NewProcessorManager() *ProcessorManager {
	return &ProcessorManager{
		processors: make(map[string]Processor),
	}
}

// RegisterProcessor 注册处理器
func (pm *ProcessorManager) RegisterProcessor(name string, processor Processor) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.processors[name]; exists {
		return fmt.Errorf("processor %s already exists", name)
	}

	pm.processors[name] = processor
	return nil
}

// GetProcessor 获取处理器
func (pm *ProcessorManager) GetProcessor(name string) (Processor, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	processor, exists := pm.processors[name]
	return processor, exists
}

// ListProcessors 列出所有处理器
func (pm *ProcessorManager) ListProcessors() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	names := make([]string, 0, len(pm.processors))
	for name := range pm.processors {
		names = append(names, name)
	}
	return names
}
