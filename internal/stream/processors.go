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
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/crazy/edge-stream/internal/constants"
	"github.com/crazy/edge-stream/internal/flowfile"
)

// FileSourceProcessor 文件源处理器
type FileSourceProcessor struct {
	id        string
	name      string
	filePath  string
	outputs   []chan<- *Message
	status    StreamStatus
	metrics   *StreamMetrics
	cancel    context.CancelFunc
	mutex     sync.RWMutex
	batchSize int
	interval  time.Duration
}

// NewFileSourceProcessor 创建新的文件源处理器
func NewFileSourceProcessor(id, name, filePath string) *FileSourceProcessor {
	return &FileSourceProcessor{
		id:        id,
		name:      name,
		filePath:  filePath,
		status:    StreamStatusStopped,
		metrics:   NewStreamMetrics(id),
		batchSize: constants.DefaultErrorChannelSize,
		interval:  constants.DefaultChannelBufferSize * time.Millisecond,
	}
}

// Process 处理消息（源处理器不需要处理输入消息）
func (p *FileSourceProcessor) Process(ctx context.Context, message *Message) ([]*Message, error) {
	return nil, fmt.Errorf("source processor does not process input messages")
}

// Read 读取消息
func (p *FileSourceProcessor) Read(ctx context.Context) (*Message, error) {
	// 这个方法在实际实现中会被Start方法中的goroutine调用
	return nil, fmt.Errorf("read method should not be called directly")
}

// SetOutput 设置输出通道
func (p *FileSourceProcessor) SetOutput(output chan<- *Message) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.outputs = append(p.outputs, output)
}

// GetType 获取处理器类型
func (p *FileSourceProcessor) GetType() StreamType {
	return StreamTypeSource
}

// GetID 获取处理器ID
func (p *FileSourceProcessor) GetID() string {
	return p.id
}

// GetName 获取处理器名称
func (p *FileSourceProcessor) GetName() string {
	return p.name
}

// Start 启动处理器
func (p *FileSourceProcessor) Start(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.status == StreamStatusRunning {
		return fmt.Errorf("processor '%s' is already running", p.id)
	}

	if len(p.outputs) == 0 {
		return fmt.Errorf("no output channels set for processor '%s'", p.id)
	}

	ctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.status = StreamStatusRunning
	p.metrics.StartTime = time.Now()

	// 启动文件读取goroutine
	go p.readFile(ctx)

	return nil
}

// Stop 停止处理器
func (p *FileSourceProcessor) Stop(ctx context.Context) error {
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
func (p *FileSourceProcessor) GetStatus() StreamStatus {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.status
}

// GetMetrics 获取处理器指标
func (p *FileSourceProcessor) GetMetrics() *StreamMetrics {
	return p.metrics.GetMetrics()
}

// GetMetricsRef 获取处理器指标的直接引用
func (p *FileSourceProcessor) GetMetricsRef() *StreamMetrics {
	return p.metrics
}

// readFile 读取文件内容
func (p *FileSourceProcessor) readFile(ctx context.Context) {
	defer func() {
		// 只有在被取消时才设置状态为Stopped
		// 正常完成文件读取时保持Running状态，由Stop方法来改变状态
		select {
		case <-ctx.Done():
			p.mutex.Lock()
			p.status = StreamStatusStopped
			p.mutex.Unlock()
		default:
			// 文件读取完成，但处理器仍然运行
		}
	}()

	fmt.Printf("[%s] 开始读取文件: %s\n", p.id, p.filePath)
	file, err := os.Open(p.filePath)
	if err != nil {
		fmt.Printf("[%s] 打开文件失败: %v\n", p.id, err)
		p.metrics.UpdateMetrics(0, 0, 0, 0, 0, 1)
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			// Log error but don't fail the operation
		}
	}()
	fmt.Printf("[%s] 文件打开成功\n", p.id)

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	fmt.Printf("[%s] 开始扫描文件内容\n", p.id)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			fmt.Printf("[%s] 收到取消信号，停止读取\n", p.id)
			return
		default:
		}

		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		fmt.Printf("[%s] 读取第%d行: %s\n", p.id, lineNumber, line)
		if line == "" {
			fmt.Printf("[%s] 跳过空行\n", p.id)
			continue
		}

		start := time.Now()

		// 创建消息
		message := &Message{
			ID:        fmt.Sprintf("%s_line_%d", p.id, lineNumber),
			Timestamp: time.Now(),
			Data:      line,
			Headers: map[string]interface{}{
				"source":      p.id,
				"line_number": lineNumber,
				"file_path":   p.filePath,
			},
			Partition: "default",
			Offset:    int64(lineNumber),
		}

		// 发送消息到所有输出通道
		fmt.Printf("[%s] 尝试发送消息: %s\n", p.id, message.ID)
		for i, output := range p.outputs {
			select {
			case output <- message:
				fmt.Printf("[%s] 消息发送成功到输出通道 %d: %s\n", p.id, i, message.ID)
			case <-ctx.Done():
				fmt.Printf("[%s] 收到取消信号，停止发送\n", p.id)
				return
			}
		}
		processingTime := time.Since(start)
		p.metrics.UpdateMetrics(0, 1, 0, int64(len(line)), processingTime, 0)

		// 控制读取速度
		if lineNumber%p.batchSize == 0 {
			time.Sleep(p.interval)
		}
	}

	if err := scanner.Err(); err != nil {
		p.metrics.UpdateMetrics(0, 0, 0, 0, 0, 1)
	}
}

// JSONTransformProcessor JSON转换处理器
type JSONTransformProcessor struct {
	id      string
	name    string
	input   <-chan *Message
	outputs []chan<- *Message
	status  StreamStatus
	metrics *StreamMetrics
	cancel  context.CancelFunc
	mutex   sync.RWMutex
}

// NewJSONTransformProcessor 创建新的JSON转换处理器
func NewJSONTransformProcessor(id, name string) *JSONTransformProcessor {
	return &JSONTransformProcessor{
		id:      id,
		name:    name,
		status:  StreamStatusStopped,
		metrics: NewStreamMetrics(id),
	}
}

// Process 处理消息
func (p *JSONTransformProcessor) Process(ctx context.Context, message *Message) ([]*Message, error) {
	start := time.Now()

	// 转换消息
	transformedMessage, err := p.Transform(ctx, message)
	if err != nil {
		p.metrics.UpdateMetrics(1, 0, int64(len(fmt.Sprintf("%v", message.Data))), 0, time.Since(start), 1)
		return nil, err
	}

	processingTime := time.Since(start)
	inputSize := int64(len(fmt.Sprintf("%v", message.Data)))
	outputSize := int64(0)
	for _, msg := range transformedMessage {
		outputSize += int64(len(fmt.Sprintf("%v", msg.Data)))
	}

	p.metrics.UpdateMetrics(1, int64(len(transformedMessage)), inputSize, outputSize, processingTime, 0)

	return transformedMessage, nil
}

// Transform 转换消息
func (p *JSONTransformProcessor) Transform(ctx context.Context, message *Message) ([]*Message, error) {
	// 尝试解析JSON数据
	var jsonData map[string]interface{}
	var err error

	switch data := message.Data.(type) {
	case string:
		err = json.Unmarshal([]byte(data), &jsonData)
		if err != nil {
			// 如果不是JSON，创建一个包装对象
			jsonData = map[string]interface{}{
				"text": data,
			}
		}
	case map[string]interface{}:
		jsonData = data
	default:
		// 对于其他类型，转换为字符串并包装
		jsonData = map[string]interface{}{
			"value": data,
		}
	}

	// 添加处理时间戳
	jsonData["processed_at"] = time.Now().Format(time.RFC3339)

	// 添加处理器信息
	jsonData["processor_id"] = p.id

	// 创建转换后的消息
	transformedMessage := &Message{
		ID:        fmt.Sprintf("%s_transformed", message.ID),
		Timestamp: time.Now(),
		Data:      jsonData,
		Headers: map[string]interface{}{
			"original_id":    message.ID,
			"transformer":    p.id,
			"transform_time": time.Now().Format(time.RFC3339),
		},
		Partition: message.Partition,
		Offset:    message.Offset,
	}

	// 复制原始头部信息
	for k, v := range message.Headers {
		if _, exists := transformedMessage.Headers[k]; !exists {
			transformedMessage.Headers[k] = v
		}
	}

	return []*Message{transformedMessage}, nil
}

// SetInput 设置输入通道
func (p *JSONTransformProcessor) SetInput(input <-chan *Message) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.input = input
}

// SetOutput 设置输出通道
func (p *JSONTransformProcessor) SetOutput(output chan<- *Message) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.outputs = append(p.outputs, output)
}

// GetType 获取处理器类型
func (p *JSONTransformProcessor) GetType() StreamType {
	return StreamTypeTransform
}

// GetID 获取处理器ID
func (p *JSONTransformProcessor) GetID() string {
	return p.id
}

// GetName 获取处理器名称
func (p *JSONTransformProcessor) GetName() string {
	return p.name
}

// Start 启动处理器
func (p *JSONTransformProcessor) Start(ctx context.Context) error {
	config := &ProcessorConfig{
		ID:          p.id,
		Mutex:       &p.mutex,
		Status:      &p.status,
		Input:       p.input,
		Outputs:     p.outputs,
		Cancel:      &p.cancel,
		Metrics:     p.metrics,
		ProcessFunc: func(ctx context.Context) { p.processMessages(ctx) },
	}
	return startProcessor(ctx, config)
}

// Stop 停止处理器
func (p *JSONTransformProcessor) Stop(ctx context.Context) error {
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
func (p *JSONTransformProcessor) GetStatus() StreamStatus {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.status
}

// GetMetrics 获取处理器指标
func (p *JSONTransformProcessor) GetMetrics() *StreamMetrics {
	return p.metrics.GetMetrics()
}

// GetMetricsRef 获取处理器指标的直接引用
func (p *JSONTransformProcessor) GetMetricsRef() *StreamMetrics {
	return p.metrics
}

// processMessages 处理消息
func (p *JSONTransformProcessor) processMessages(ctx context.Context) {
	defer func() {
		p.mutex.Lock()
		p.status = StreamStatusStopped
		p.mutex.Unlock()
	}()

	fmt.Printf("[%s] JSON转换处理器开始监听消息\n", p.id)
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("[%s] JSON转换处理器收到取消信号\n", p.id)
			return
		case message, ok := <-p.input:
			if !ok {
				fmt.Printf("[%s] 输入通道已关闭\n", p.id)
				return
			}

			fmt.Printf("[%s] 接收到消息: %s\n", p.id, message.ID)
			// 处理消息
			transformedMessages, err := p.Process(ctx, message)
			if err != nil {
				fmt.Printf("[%s] 处理消息失败: %v\n", p.id, err)
				continue
			}

			// 发送转换后的消息到所有输出通道
			for _, transformedMessage := range transformedMessages {
				fmt.Printf("[%s] 发送转换后的消息: %s 到 %d 个输出通道\n", p.id, transformedMessage.ID, len(p.outputs))
				for i, output := range p.outputs {
					select {
					case output <- transformedMessage:
						fmt.Printf("[%s] 消息发送成功到输出通道 %d: %s\n", p.id, i, transformedMessage.ID)
					case <-ctx.Done():
						fmt.Printf("[%s] 发送时收到取消信号\n", p.id)
						return
					}
				}
			}
		}
	}
}

// ConsoleSinkProcessor 控制台接收器处理器
type ConsoleSinkProcessor struct {
	id      string
	name    string
	input   <-chan *Message
	status  StreamStatus
	metrics *StreamMetrics
	cancel  context.CancelFunc
	mutex   sync.RWMutex
	prefix  string
}

// NewConsoleSinkProcessor 创建新的控制台接收器处理器
func NewConsoleSinkProcessor(id, name string) *ConsoleSinkProcessor {
	return &ConsoleSinkProcessor{
		id:      id,
		name:    name,
		status:  StreamStatusStopped,
		metrics: NewStreamMetrics(id),
		prefix:  "[STREAM]",
	}
}

// Process 处理消息
func (p *ConsoleSinkProcessor) Process(ctx context.Context, message *Message) ([]*Message, error) {
	return nil, p.Write(ctx, message)
}

// Write 写入消息
func (p *ConsoleSinkProcessor) Write(ctx context.Context, message *Message) error {
	start := time.Now()

	// 格式化输出
	var output string
	switch data := message.Data.(type) {
	case string:
		output = data
	case map[string]interface{}:
		jsonBytes, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			output = fmt.Sprintf("%v", data)
		} else {
			output = string(jsonBytes)
		}
	default:
		output = fmt.Sprintf("%v", data)
	}

	// 输出到控制台
	timestamp := message.Timestamp.Format("15:04:05")
	messageID := message.ID
	if len(messageID) > constants.DefaultBatchSize {
		messageID = messageID[:constants.DefaultBatchSize] + "..."
	}

	fmt.Printf("%s [%s] [%s] %s\n", p.prefix, timestamp, messageID, output)

	processingTime := time.Since(start)
	dataSize := int64(len(output))
	p.metrics.UpdateMetrics(1, 0, dataSize, 0, processingTime, 0)

	return nil
}

// SetInput 设置输入通道
func (p *ConsoleSinkProcessor) SetInput(input <-chan *Message) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.input = input
}

// GetType 获取处理器类型
func (p *ConsoleSinkProcessor) GetType() StreamType {
	return StreamTypeSink
}

// GetID 获取处理器ID
func (p *ConsoleSinkProcessor) GetID() string {
	return p.id
}

// GetName 获取处理器名称
func (p *ConsoleSinkProcessor) GetName() string {
	return p.name
}

// Start 启动处理器
func (p *ConsoleSinkProcessor) Start(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.status == StreamStatusRunning {
		return fmt.Errorf("processor '%s' is already running", p.id)
	}

	if p.input == nil {
		return fmt.Errorf("input channel not set for processor '%s'", p.id)
	}

	ctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.status = StreamStatusRunning
	p.metrics.StartTime = time.Now()

	// 启动消息处理goroutine
	go p.processMessages(ctx)

	return nil
}

// Stop 停止处理器
func (p *ConsoleSinkProcessor) Stop(ctx context.Context) error {
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
func (p *ConsoleSinkProcessor) GetStatus() StreamStatus {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.status
}

// GetMetrics 获取处理器指标
func (p *ConsoleSinkProcessor) GetMetrics() *StreamMetrics {
	return p.metrics.GetMetrics()
}

// GetMetricsRef 获取处理器指标的直接引用
func (p *ConsoleSinkProcessor) GetMetricsRef() *StreamMetrics {
	return p.metrics
}

// processMessages 处理消息
func (p *ConsoleSinkProcessor) processMessages(ctx context.Context) {
	defer func() {
		p.mutex.Lock()
		p.status = StreamStatusStopped
		p.mutex.Unlock()
	}()

	fmt.Printf("[%s] 控制台输出处理器开始监听消息\n", p.id)
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("[%s] 控制台输出处理器收到取消信号\n", p.id)
			return
		case message, ok := <-p.input:
			if !ok {
				fmt.Printf("[%s] 输入通道已关闭\n", p.id)
				return
			}

			fmt.Printf("[%s] 接收到消息: %s\n", p.id, message.ID)
			// 写入消息
			if err := p.Write(ctx, message); err != nil {
				fmt.Printf("[%s] 写入消息失败: %v\n", p.id, err)
				p.metrics.UpdateMetrics(1, 0, 0, 0, 0, 1)
			} else {
				fmt.Printf("[%s] 消息写入成功: %s\n", p.id, message.ID)
			}
		}
	}
}

// AggregationProcessor 聚合处理器
type AggregationProcessor struct {
	id          string
	name        string
	input       <-chan *Message
	outputs     []chan<- *Message
	status      StreamStatus
	metrics     *StreamMetrics
	cancel      context.CancelFunc
	mutex       sync.RWMutex
	window      WindowProcessor
	aggType     AggregationType
	aggField    string
	windowCount int64
}

// NewAggregationProcessor 创建新的聚合处理器
func NewAggregationProcessor(id, name string, windowConfig *WindowConfig, aggType AggregationType, aggField string) *AggregationProcessor {
	return &AggregationProcessor{
		id:       id,
		name:     name,
		status:   StreamStatusStopped,
		metrics:  NewStreamMetrics(id),
		window:   NewStandardWindowProcessor(windowConfig),
		aggType:  aggType,
		aggField: aggField,
	}
}

// Process 处理消息
func (p *AggregationProcessor) Process(ctx context.Context, message *Message) ([]*Message, error) {
	start := time.Now()

	// 添加消息到窗口
	if err := p.window.AddMessage(message); err != nil {
		p.metrics.UpdateMetrics(1, 0, int64(len(fmt.Sprintf("%v", message.Data))), 0, time.Since(start), 1)
		return nil, err
	}

	// 检查并关闭过期窗口
	expiredWindows, err := p.window.CloseExpiredWindows(time.Now())
	if err != nil {
		p.metrics.UpdateMetrics(1, 0, int64(len(fmt.Sprintf("%v", message.Data))), 0, time.Since(start), 1)
		return nil, err
	}

	// 对过期窗口进行聚合
	results := make([]*Message, 0)
	for _, window := range expiredWindows {
		aggResult, err := p.window.Aggregate(window, p.aggType, p.aggField)
		if err != nil {
			continue
		}

		// 创建聚合结果消息
		resultMessage := &Message{
			ID:        fmt.Sprintf("%s_agg_%d", p.id, atomic.AddInt64(&p.windowCount, 1)),
			Timestamp: time.Now(),
			Data:      aggResult,
			Headers: map[string]interface{}{
				"processor_id":  p.id,
				"window_id":     window.ID,
				"window_type":   string(window.Type),
				"aggregation":   string(p.aggType),
				"field":         p.aggField,
				"message_count": len(window.Messages),
				"window_start":  window.StartTime.Format(time.RFC3339),
				"window_end":    window.EndTime.Format(time.RFC3339),
			},
			Partition: "aggregation",
			Offset:    atomic.LoadInt64(&p.windowCount),
		}

		results = append(results, resultMessage)
	}

	processingTime := time.Since(start)
	inputSize := int64(len(fmt.Sprintf("%v", message.Data)))
	outputSize := int64(0)
	for _, msg := range results {
		outputSize += int64(len(fmt.Sprintf("%v", msg.Data)))
	}

	p.metrics.UpdateMetrics(1, int64(len(results)), inputSize, outputSize, processingTime, 0)

	return results, nil
}

// Transform 转换消息（聚合处理器实现TransformProcessor接口）
func (p *AggregationProcessor) Transform(ctx context.Context, message *Message) ([]*Message, error) {
	return p.Process(ctx, message)
}

// SetInput 设置输入通道
func (p *AggregationProcessor) SetInput(input <-chan *Message) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.input = input
}

// SetOutput 设置输出通道
func (p *AggregationProcessor) SetOutput(output chan<- *Message) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.outputs = append(p.outputs, output)
}

// GetType 获取处理器类型
func (p *AggregationProcessor) GetType() StreamType {
	return StreamTypeTransform
}

// GetID 获取处理器ID
func (p *AggregationProcessor) GetID() string {
	return p.id
}

// GetName 获取处理器名称
func (p *AggregationProcessor) GetName() string {
	return p.name
}

// Start 启动处理器
func (p *AggregationProcessor) Start(ctx context.Context) error {
	config := &ProcessorConfig{
		ID:          p.id,
		Mutex:       &p.mutex,
		Status:      &p.status,
		Input:       p.input,
		Outputs:     p.outputs,
		Cancel:      &p.cancel,
		Metrics:     p.metrics,
		ProcessFunc: func(ctx context.Context) { p.processMessages(ctx) },
	}
	return startProcessor(ctx, config)
}

// Stop 停止处理器
func (p *AggregationProcessor) Stop(ctx context.Context) error {
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
func (p *AggregationProcessor) GetStatus() StreamStatus {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.status
}

// GetMetrics 获取处理器指标
func (p *AggregationProcessor) GetMetrics() *StreamMetrics {
	return p.metrics.GetMetrics()
}

// GetMetricsRef 获取处理器指标的直接引用
func (p *AggregationProcessor) GetMetricsRef() *StreamMetrics {
	return p.metrics
}

// ProcessorConfig 处理器配置结构体
type ProcessorConfig struct {
	ID          string
	Mutex       *sync.RWMutex
	Status      *StreamStatus
	Input       <-chan *Message
	Outputs     []chan<- *Message
	Cancel      *context.CancelFunc
	Metrics     *StreamMetrics
	ProcessFunc func(context.Context)
}

// startProcessor 通用的处理器启动函数
func startProcessor(ctx context.Context, config *ProcessorConfig) error {
	config.Mutex.Lock()
	defer config.Mutex.Unlock()

	if *config.Status == StreamStatusRunning {
		return fmt.Errorf("processor '%s' is already running", config.ID)
	}

	if config.Input == nil {
		return fmt.Errorf("input channel not set for processor '%s'", config.ID)
	}

	if len(config.Outputs) == 0 {
		return fmt.Errorf("output channel not set for processor '%s'", config.ID)
	}

	ctx, cancelFunc := context.WithCancel(ctx)
	*config.Cancel = cancelFunc
	*config.Status = StreamStatusRunning
	config.Metrics.StartTime = time.Now()

	// 启动消息处理goroutine
	go config.ProcessFunc(ctx)

	return nil
}

// MessageProcessor 消息处理器接口
type MessageProcessor interface {
	Process(context.Context, *Message) ([]*Message, error)
}

// processMessages 通用消息处理函数
func processMessages(
	ctx context.Context,
	processor MessageProcessor,
	mutex *sync.RWMutex,
	status *StreamStatus,
	input <-chan *Message,
	outputs []chan<- *Message,
) {
	defer func() {
		mutex.Lock()
		*status = StreamStatusStopped
		mutex.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case message, ok := <-input:
			if !ok {
				return
			}

			// 处理消息
			results, err := processor.Process(ctx, message)
			if err != nil {
				continue
			}

			// 发送结果到所有输出通道
			for _, result := range results {
				for _, output := range outputs {
					select {
					case output <- result:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}
}

// processMessages 处理消息
func (p *AggregationProcessor) processMessages(ctx context.Context) {
	processMessages(ctx, p, &p.mutex, &p.status, p.input, p.outputs)
}

// CreateSampleStreamDataFile 创建示例流数据文件
func CreateSampleStreamDataFile(filePath string) error {
	// 确保目录存在
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, constants.DefaultDirectoryPermission); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// 验证文件路径安全性
	if !filepath.IsAbs(filePath) || strings.Contains(filePath, "..") {
		return fmt.Errorf("invalid file path: %s", filePath)
	}

	file, err := os.Create(filepath.Clean(filePath))
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			// Log error but don't fail the operation
		}
	}()

	// 生成示例数据
	sampleData := []string{
		`{"id":1,"name":"Alice","age":30,"score":85.5,"city":"New York"}`,
		`{"id":2,"name":"Bob","age":25,"score":92.0,"city":"Los Angeles"}`,
		`{"id":3,"name":"Charlie","age":35,"score":78.5,"city":"Chicago"}`,
		`{"id":4,"name":"Diana","age":28,"score":88.0,"city":"Houston"}`,
		`{"id":5,"name":"Eve","age":32,"score":95.5,"city":"Phoenix"}`,
		`SIMPLE TEXT MESSAGE`,
		`{"id":6,"name":"Frank","age":29,"score":82.0,"city":"Philadelphia"}`,
		`{"id":7,"name":"Grace","age":27,"score":90.5,"city":"San Antonio"}`,
		`ANOTHER TEXT MESSAGE`,
		`{"id":8,"name":"Henry","age":31,"score":87.0,"city":"San Diego"}`,
		`{"id":9,"name":"Ivy","age":26,"score":93.5,"city":"Dallas"}`,
		`{"id":10,"name":"Jack","age":33,"score":79.0,"city":"San Jose"}`,
	}

	// 写入数据，每行添加时间戳
	for i, data := range sampleData {
		// 为每条数据添加序号和时间戳信息
		line := fmt.Sprintf("%s\n", data)
		if _, err := file.WriteString(line); err != nil {
			return fmt.Errorf("failed to write line %d: %v", i+1, err)
		}

		// 每隔几行添加一些延迟，模拟实时数据
		if i%3 == 0 {
			time.Sleep(constants.DefaultErrorChannelSize * time.Millisecond)
		}
	}

	return nil
}

// TestProcessor 测试处理器，用于单元测试
type TestProcessor struct {
	id          string
	name        string
	input       <-chan *Message
	outputs     []chan<- *Message
	status      StreamStatus
	metrics     *StreamMetrics
	cancel      context.CancelFunc
	mutex       sync.RWMutex
	processFunc func(ff *flowfile.FlowFile) (*flowfile.FlowFile, error)
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

// Process 处理消息
func (p *TestProcessor) Process(ctx context.Context, message *Message) ([]*Message, error) {
	start := time.Now()

	// 如果设置了处理函数，先调用处理函数
	if p.processFunc != nil {
		// 将Message转换为FlowFile进行处理
		ff := flowfile.NewFlowFile()
		ff.SetAttribute("message_id", message.ID)
		ff.SetAttribute("timestamp", message.Timestamp.String())
		if data, ok := message.Data.(string); ok {
			ff.Content = []byte(data)
			ff.Size = int64(len(ff.Content))
		}

		// 调用处理函数
		_, err := p.processFunc(ff)
		if err != nil {
			return nil, err
		}
	}

	// 简单的消息处理：添加处理器标识
	processedMessage := &Message{
		ID:        fmt.Sprintf("%s_processed_%s", p.id, message.ID),
		Timestamp: time.Now(),
		Data:      fmt.Sprintf("[%s] %v", p.name, message.Data),
		Headers: map[string]interface{}{
			"processor_id":   p.id,
			"processor_name": p.name,
			"original_id":    message.ID,
		},
		Partition: message.Partition,
		Offset:    message.Offset,
	}

	processingTime := time.Since(start)
	inputSize := int64(len(fmt.Sprintf("%v", message.Data)))
	outputSize := int64(len(fmt.Sprintf("%v", processedMessage.Data)))

	p.metrics.UpdateMetrics(1, 1, inputSize, outputSize, processingTime, 0)

	return []*Message{processedMessage}, nil
}

// Transform 转换消息（实现TransformProcessor接口）
func (p *TestProcessor) Transform(ctx context.Context, message *Message) ([]*Message, error) {
	return p.Process(ctx, message)
}

// SetInput 设置输入通道
func (p *TestProcessor) SetInput(input <-chan *Message) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.input = input
}

// SetOutput 设置输出通道
func (p *TestProcessor) SetOutput(output chan<- *Message) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.outputs = append(p.outputs, output)
}

// SetProcessFunc 设置处理函数
func (p *TestProcessor) SetProcessFunc(processFunc func(ff *flowfile.FlowFile) (*flowfile.FlowFile, error)) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.processFunc = processFunc
}

// CallProcessFunc 调用处理函数
func (p *TestProcessor) CallProcessFunc(ff *flowfile.FlowFile) (*flowfile.FlowFile, error) {
	p.mutex.RLock()
	processFunc := p.processFunc
	p.mutex.RUnlock()

	if processFunc == nil {
		return ff, nil
	}
	return processFunc(ff)
}

// GetDescription 获取处理器描述
func (p *TestProcessor) GetDescription() string {
	return fmt.Sprintf("Test processor: %s", p.name)
}

// Write 写入消息（用于测试）
func (p *TestProcessor) Write(ctx context.Context, message *Message) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// 计算数据大小
	dataSize := int64(0)
	if message.Data != nil {
		switch data := message.Data.(type) {
		case string:
			dataSize = int64(len(data))
		case []byte:
			dataSize = int64(len(data))
		default:
			// 对于其他类型，使用估算值
			dataSize = 100
		}
	}

	p.metrics.UpdateMetrics(1, 0, dataSize, 0, 0, 0)

	// 这里可以添加实际的写入逻辑
	return nil
}

// TransformProcessorWrapper 转换处理器包装器
type TransformProcessorWrapper struct {
	*TestProcessor
	transformFunc func(ctx context.Context, msg *Message) ([]*Message, error)
}

// Transform 实现转换接口
func (p *TransformProcessorWrapper) Transform(ctx context.Context, message *Message) ([]*Message, error) {
	if p.transformFunc != nil {
		return p.transformFunc(ctx, message)
	}
	return p.Process(ctx, message)
}

// GetType 获取处理器类型
func (p *TransformProcessorWrapper) GetType() StreamType {
	return StreamTypeTransform
}

// NewTransformProcessor 创建新的转换处理器
func NewTransformProcessor(
	id, name string,
	transformFunc ...func(ctx context.Context, msg *Message) ([]*Message, error),
) *TransformProcessorWrapper {
	processor := &TestProcessor{
		id:      id,
		name:    name,
		status:  StreamStatusStopped,
		metrics: NewStreamMetrics(id),
	}

	wrapper := &TransformProcessorWrapper{
		TestProcessor: processor,
	}

	// 如果提供了转换函数，设置转换函数
	if len(transformFunc) > 0 {
		wrapper.transformFunc = transformFunc[0]
	}

	return wrapper
}

// SinkProcessorWrapper 接收器处理器包装器
type SinkProcessorWrapper struct {
	*TestProcessor
	sinkFunc func(ctx context.Context, msg *Message) error
}

// Write 实现写入接口
func (p *SinkProcessorWrapper) Write(ctx context.Context, message *Message) error {
	if p.sinkFunc != nil {
		return p.sinkFunc(ctx, message)
	}
	return p.TestProcessor.Write(ctx, message)
}

// Process 处理消息（接收器不返回消息）
func (p *SinkProcessorWrapper) Process(ctx context.Context, message *Message) ([]*Message, error) {
	err := p.Write(ctx, message)
	return nil, err
}

// GetType 获取处理器类型
func (p *SinkProcessorWrapper) GetType() StreamType {
	return StreamTypeSink
}

// NewSinkProcessor 创建新的输出处理器
func NewSinkProcessor(
	id, name string,
	sinkFunc ...func(ctx context.Context, msg *Message) error,
) *SinkProcessorWrapper {
	processor := &TestProcessor{
		id:      id,
		name:    name,
		status:  StreamStatusStopped,
		metrics: NewStreamMetrics(id),
	}

	wrapper := &SinkProcessorWrapper{
		TestProcessor: processor,
	}

	// 如果提供了sink函数，设置sink函数
	if len(sinkFunc) > 0 {
		wrapper.sinkFunc = sinkFunc[0]
	}

	return wrapper
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

// Start 启动处理器
func (p *TestProcessor) Start(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.status == StreamStatusRunning {
		// 重复启动是安全的，直接返回成功
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.status = StreamStatusRunning
	p.metrics.StartTime = time.Now()

	// 启动消息处理goroutine
	go p.processMessages(ctx)

	return nil
}

// Stop 停止处理器
func (p *TestProcessor) Stop(ctx context.Context) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.status != StreamStatusRunning {
		// 重复停止是安全的，直接返回成功
		return nil
	}

	if p.cancel != nil {
		p.cancel()
	}

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

// processMessages 处理消息
func (p *TestProcessor) processMessages(ctx context.Context) {
	processMessages(ctx, p, &p.mutex, &p.status, p.input, p.outputs)
}

// ProcessFlowFile 实现FlowFileProcessor接口
func (p *TestProcessor) ProcessFlowFile(ff interface{}) (interface{}, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// 如果有自定义处理函数，使用它
	if p.processFunc != nil {
		if flowFile, ok := ff.(*flowfile.FlowFile); ok {
			return p.processFunc(flowFile)
		}
	}

	// 默认处理：直接返回输入
	return ff, nil
}
