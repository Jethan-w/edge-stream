package stream

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/crazy/edge-stream/internal/flowfile"
)

// TestProcessor 测试处理器
type TestProcessor struct {
	id          string
	name        string
	processFunc func(*flowfile.FlowFile) (*flowfile.FlowFile, error)
	status      StreamStatus
	metrics     *StreamMetrics
	mutex       sync.RWMutex
}

// NewTestProcessor 创建新的测试处理器
func NewTestProcessor(id, name string) *TestProcessor {
	return &TestProcessor{
		id:      id,
		name:    name,
		status:  StreamStatusStopped,
		metrics: NewStreamMetrics(id),
	}
}

// SetProcessFunc 设置处理函数
func (p *TestProcessor) SetProcessFunc(processFunc func(*flowfile.FlowFile) (*flowfile.FlowFile, error)) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.processFunc = processFunc
}

// CallProcessFunc 调用处理函数（用于测试）
func (p *TestProcessor) CallProcessFunc(ff *flowfile.FlowFile) (*flowfile.FlowFile, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	if p.processFunc != nil {
		return p.processFunc(ff)
	}
	return ff, nil
}

// ProcessFlowFile 处理FlowFile
func (p *TestProcessor) ProcessFlowFile(ff interface{}) (interface{}, error) {
	if p.processFunc != nil {
		if flowFile, ok := ff.(*flowfile.FlowFile); ok {
			return p.processFunc(flowFile)
		}
	}
	return ff, nil
}

// Process 处理消息
func (p *TestProcessor) Process(ctx context.Context, message *Message) ([]*Message, error) {
	start := time.Now()

	// 如果有自定义处理函数，则调用它
	if p.processFunc != nil {
		// 创建FlowFile用于处理
		ff := &flowfile.FlowFile{
			Attributes: make(map[string]string),
		}
		ff.SetAttribute("message_id", message.ID)
		ff.SetAttribute("data", fmt.Sprintf("%v", message.Data))

		// 调用处理函数
		result, err := p.processFunc(ff)
		if err != nil {
			p.metrics.UpdateMetrics(1, 0, int64(len(fmt.Sprintf("%v", message.Data))), 0, time.Since(start), 1)
			return nil, err
		}

		// 如果处理成功，更新指标
		processingTime := time.Since(start)
		inputSize := int64(len(fmt.Sprintf("%v", message.Data)))
		outputSize := int64(len(fmt.Sprintf("%v", result)))
		p.metrics.UpdateMetrics(1, 1, inputSize, outputSize, processingTime, 0)

		// 返回处理后的消息
		return []*Message{message}, nil
	}

	// 默认直接返回原消息
	p.metrics.UpdateMetrics(1, 1, int64(len(fmt.Sprintf("%v", message.Data))), int64(len(fmt.Sprintf("%v", message.Data))), time.Since(start), 0)
	return []*Message{message}, nil
}

// GetType 获取处理器类型
func (p *TestProcessor) GetType() StreamType {
	return StreamTypeTransform
}

// GetID 获取处理器ID
func (p *TestProcessor) GetID() string {
	return p.id
}

// GetName 获取处理器名称
func (p *TestProcessor) GetName() string {
	return p.name
}

// GetDescription 获取处理器描述
func (p *TestProcessor) GetDescription() string {
	return fmt.Sprintf("Test processor: %s", p.name)
}

// Start 启动处理器
func (p *TestProcessor) Start(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.status = StreamStatusRunning
	p.metrics.StartTime = time.Now()
	return nil
}

// Stop 停止处理器
func (p *TestProcessor) Stop(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.status = StreamStatusStopped
	return nil
}

// GetStatus 获取处理器状态
func (p *TestProcessor) GetStatus() StreamStatus {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.status
}

// GetMetrics 获取处理器指标
func (p *TestProcessor) GetMetrics() *StreamMetrics {
	return p.metrics.GetMetrics()
}

// GetMetricsRef 获取处理器指标的直接引用
func (p *TestProcessor) GetMetricsRef() *StreamMetrics {
	return p.metrics
}

// GenericTransformProcessor 通用转换处理器
type GenericTransformProcessor struct {
	id            string
	name          string
	transformFunc func(context.Context, *Message) ([]*Message, error)
	input         <-chan *Message
	outputs       []chan<- *Message
	status        StreamStatus
	metrics       *StreamMetrics
	cancel        context.CancelFunc
	mutex         sync.RWMutex
}

// NewTransformProcessor 创建新的转换处理器
func NewTransformProcessor(id, name string, transformFunc func(context.Context, *Message) ([]*Message, error)) *GenericTransformProcessor {
	return &GenericTransformProcessor{
		id:            id,
		name:          name,
		transformFunc: transformFunc,
		status:        StreamStatusStopped,
		metrics:       NewStreamMetrics(id),
	}
}

// Process 处理消息
func (p *GenericTransformProcessor) Process(ctx context.Context, message *Message) ([]*Message, error) {
	start := time.Now()

	result, err := p.transformFunc(ctx, message)
	if err != nil {
		p.metrics.UpdateMetrics(1, 0, int64(len(fmt.Sprintf("%v", message.Data))), 0, time.Since(start), 1)
		return nil, err
	}

	processingTime := time.Since(start)
	inputSize := int64(len(fmt.Sprintf("%v", message.Data)))
	outputSize := int64(0)
	for _, msg := range result {
		outputSize += int64(len(fmt.Sprintf("%v", msg.Data)))
	}

	p.metrics.UpdateMetrics(1, int64(len(result)), inputSize, outputSize, processingTime, 0)
	return result, nil
}

// Transform 转换消息
func (p *GenericTransformProcessor) Transform(ctx context.Context, message *Message) ([]*Message, error) {
	return p.transformFunc(ctx, message)
}

// SetInput 设置输入通道
func (p *GenericTransformProcessor) SetInput(input <-chan *Message) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.input = input
}

// SetOutput 设置输出通道
func (p *GenericTransformProcessor) SetOutput(output chan<- *Message) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.outputs = append(p.outputs, output)
}

// GetType 获取处理器类型
func (p *GenericTransformProcessor) GetType() StreamType {
	return StreamTypeTransform
}

// GetID 获取处理器ID
func (p *GenericTransformProcessor) GetID() string {
	return p.id
}

// GetName 获取处理器名称
func (p *GenericTransformProcessor) GetName() string {
	return p.name
}

// Start 启动处理器
func (p *GenericTransformProcessor) Start(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.status == StreamStatusRunning {
		return fmt.Errorf("processor '%s' is already running", p.id)
	}

	ctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.status = StreamStatusRunning
	p.metrics.StartTime = time.Now()

	return nil
}

// Stop 停止处理器
func (p *GenericTransformProcessor) Stop(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.status != StreamStatusRunning {
		return fmt.Errorf("processor '%s' is not running", p.id)
	}

	if p.cancel != nil {
		p.cancel()
	}

	p.status = StreamStatusStopped
	return nil
}

// GetStatus 获取处理器状态
func (p *GenericTransformProcessor) GetStatus() StreamStatus {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.status
}

// GetMetrics 获取处理器指标
func (p *GenericTransformProcessor) GetMetrics() *StreamMetrics {
	return p.metrics.GetMetrics()
}

// GetMetricsRef 获取处理器指标的直接引用
func (p *GenericTransformProcessor) GetMetricsRef() *StreamMetrics {
	return p.metrics
}

// GenericSinkProcessor 通用接收器处理器
type GenericSinkProcessor struct {
	id        string
	name      string
	writeFunc func(context.Context, *Message) error
	input     <-chan *Message
	status    StreamStatus
	metrics   *StreamMetrics
	cancel    context.CancelFunc
	mutex     sync.RWMutex
}

// NewSinkProcessor 创建新的接收器处理器
func NewSinkProcessor(id, name string, writeFunc func(context.Context, *Message) error) *GenericSinkProcessor {
	return &GenericSinkProcessor{
		id:        id,
		name:      name,
		writeFunc: writeFunc,
		status:    StreamStatusStopped,
		metrics:   NewStreamMetrics(id),
	}
}

// Process 处理消息
func (p *GenericSinkProcessor) Process(ctx context.Context, message *Message) ([]*Message, error) {
	err := p.Write(ctx, message)
	return []*Message{}, err
}

// Write 写入消息
func (p *GenericSinkProcessor) Write(ctx context.Context, message *Message) error {
	start := time.Now()

	err := p.writeFunc(ctx, message)
	if err != nil {
		p.metrics.UpdateMetrics(1, 0, int64(len(fmt.Sprintf("%v", message.Data))), 0, time.Since(start), 1)
		return err
	}

	processingTime := time.Since(start)
	dataSize := int64(len(fmt.Sprintf("%v", message.Data)))
	p.metrics.UpdateMetrics(1, 0, dataSize, 0, processingTime, 0)

	return nil
}

// SetInput 设置输入通道
func (p *GenericSinkProcessor) SetInput(input <-chan *Message) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.input = input
}

// GetType 获取处理器类型
func (p *GenericSinkProcessor) GetType() StreamType {
	return StreamTypeSink
}

// GetID 获取处理器ID
func (p *GenericSinkProcessor) GetID() string {
	return p.id
}

// GetName 获取处理器名称
func (p *GenericSinkProcessor) GetName() string {
	return p.name
}

// Start 启动处理器
func (p *GenericSinkProcessor) Start(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.status == StreamStatusRunning {
		return fmt.Errorf("processor '%s' is already running", p.id)
	}

	ctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.status = StreamStatusRunning
	p.metrics.StartTime = time.Now()

	return nil
}

// Stop 停止处理器
func (p *GenericSinkProcessor) Stop(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.status != StreamStatusRunning {
		return fmt.Errorf("processor '%s' is not running", p.id)
	}

	if p.cancel != nil {
		p.cancel()
	}

	p.status = StreamStatusStopped
	return nil
}

// GetStatus 获取处理器状态
func (p *GenericSinkProcessor) GetStatus() StreamStatus {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.status
}

// GetMetrics 获取处理器指标
func (p *GenericSinkProcessor) GetMetrics() *StreamMetrics {
	return p.metrics.GetMetrics()
}

// GetMetricsRef 获取处理器指标的直接引用
func (p *GenericSinkProcessor) GetMetricsRef() *StreamMetrics {
	return p.metrics
}
