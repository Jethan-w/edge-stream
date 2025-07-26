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

// Package connector provides example connector implementations
// Copyright (c) 2024 Edge Stream. All rights reserved.
package connector

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
)

// 示例连接器常量
const (
	DefaultChannelBufferSize = 100
	DefaultErrorChannelSize  = 10
	DefaultConsoleBatchSize  = 1000
	DefaultPriority          = 2
	DefaultSessionGap        = 30 * time.Minute
	DefaultWatermark         = 5 * time.Second
	DefaultFilePermissions   = 0750
)

// FileSourceConnector 文件源连接器
type FileSourceConnector struct {
	info      ConnectorInfo
	config    map[string]interface{}
	status    ConnectorStatus
	metrics   ConnectorMetrics
	file      *os.File
	scanner   *bufio.Scanner
	startTime time.Time
	mutex     sync.RWMutex
	offset    int64
	running   bool
}

// FileSourceFactory 文件源连接器工厂
type FileSourceFactory struct{}

// Create 创建文件源连接器实例
func (f *FileSourceFactory) Create(config map[string]interface{}) (Connector, error) {
	connector := &FileSourceConnector{
		info: ConnectorInfo{
			ID:          "file-source",
			Name:        "File Source",
			Type:        ConnectorTypeSource,
			Version:     "1.0.0",
			Description: "Read data from files",
			Author:      "Edge Stream",
			CreatedAt:   time.Now(),
		},
		config:  config,
		status:  ConnectorStatusStopped,
		metrics: ConnectorMetrics{},
	}

	return connector, nil
}

// GetInfo 获取连接器信息
func (f *FileSourceFactory) GetInfo() ConnectorInfo {
	return ConnectorInfo{
		ID:          "file-source",
		Name:        "File Source",
		Type:        ConnectorTypeSource,
		Version:     "1.0.0",
		Description: "Read data from files",
		Author:      "Edge Stream",
		CreatedAt:   time.Now(),
	}
}

// ValidateConfig 验证配置
func (f *FileSourceFactory) ValidateConfig(config map[string]interface{}) error {
	filePath, ok := config["file_path"]
	if !ok {
		return fmt.Errorf("file_path is required")
	}

	if _, ok := filePath.(string); !ok {
		return fmt.Errorf("file_path must be a string")
	}

	return nil
}

// GetInfo 获取连接器信息
func (c *FileSourceConnector) GetInfo() ConnectorInfo {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.info
}

// Start 启动连接器
func (c *FileSourceConnector) Start(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.running {
		return fmt.Errorf("connector is already running")
	}

	filePath, ok := c.config["file_path"].(string)
	if !ok {
		return fmt.Errorf("file_path not configured")
	}

	// 验证文件路径安全性
	if filepath.IsAbs(filePath) {
		return fmt.Errorf("absolute paths are not allowed for security reasons")
	}
	if strings.Contains(filePath, "..") {
		return fmt.Errorf("path traversal is not allowed")
	}

	file, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	c.file = file
	c.scanner = bufio.NewScanner(file)
	c.status = ConnectorStatusRunning
	c.startTime = time.Now()
	c.running = true
	c.offset = 0

	return nil
}

// Stop 停止连接器
func (c *FileSourceConnector) Stop(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.running {
		return fmt.Errorf("connector is not running")
	}

	if c.file != nil {
		if err := c.file.Close(); err != nil {
			return fmt.Errorf("failed to close file: %w", err)
		}
		c.file = nil
	}

	c.scanner = nil
	c.status = ConnectorStatusStopped
	c.running = false

	return nil
}

// GetStatus 获取状态
func (c *FileSourceConnector) GetStatus() ConnectorStatus {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.status
}

// GetMetrics 获取指标
func (c *FileSourceConnector) GetMetrics() ConnectorMetrics {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	metrics := c.metrics
	if c.running {
		metrics.Uptime = time.Since(c.startTime)
		metrics.LastActivity = time.Now()
	}

	return metrics
}

// Validate 验证配置
func (c *FileSourceConnector) Validate(config map[string]interface{}) error {
	filePath, ok := config["file_path"]
	if !ok {
		return fmt.Errorf("file_path is required")
	}

	if _, ok := filePath.(string); !ok {
		return fmt.Errorf("file_path must be a string")
	}

	return nil
}

// Configure 配置连接器
func (c *FileSourceConnector) Configure(config map[string]interface{}) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.config = config
	c.info.Config = config
	c.info.UpdatedAt = time.Now()

	return nil
}

// Read 读取数据
func (c *FileSourceConnector) Read(ctx context.Context) (msgChan <-chan Message, errChan <-chan error) {
	msgCh := make(chan Message, DefaultChannelBufferSize)
	errCh := make(chan error, DefaultErrorChannelSize)

	go func() {
		defer close(msgCh)
		defer close(errCh)

		for {
			select {
			case <-ctx.Done():
				return
			default:
				c.mutex.RLock()
				if !c.running || c.scanner == nil {
					c.mutex.RUnlock()
					return
				}
				c.mutex.RUnlock()

				if c.scanner.Scan() {
					line := c.scanner.Text()
					if line != "" {
						msg := NewStandardMessage(fmt.Sprintf("line_%d", atomic.AddInt64(&c.offset, 1)), line)
						msg.SetOffset(c.offset)
						msg.SetHeader("source", "file")

						select {
						case msgCh <- msg:
							atomic.AddInt64(&c.metrics.MessagesProcessed, 1)
							atomic.AddInt64(&c.metrics.BytesProcessed, int64(len(line)))
						case <-ctx.Done():
							return
						}
					}
				} else {
					if err := c.scanner.Err(); err != nil {
						select {
						case errCh <- err:
							atomic.AddInt64(&c.metrics.ErrorsCount, 1)
						case <-ctx.Done():
						}
					}
					return // EOF
				}
			}
		}
	}()

	return msgCh, errCh
}

// Commit 提交偏移量
func (c *FileSourceConnector) Commit(ctx context.Context, offset int64) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.offset = offset
	return nil
}

// GetPartitions 获取分区信息
func (c *FileSourceConnector) GetPartitions() []PartitionInfo {
	return []PartitionInfo{
		{
			ID:     0,
			Offset: c.offset,
			Leader: "local",
		},
	}
}

// ConsoleSinkConnector 控制台接收器连接器
type ConsoleSinkConnector struct {
	info      ConnectorInfo
	config    map[string]interface{}
	status    ConnectorStatus
	metrics   ConnectorMetrics
	startTime time.Time
	mutex     sync.RWMutex
	running   bool
}

// ConsoleSinkFactory 控制台接收器连接器工厂
type ConsoleSinkFactory struct{}

// Create 创建控制台接收器连接器实例
func (f *ConsoleSinkFactory) Create(config map[string]interface{}) (Connector, error) {
	connector := &ConsoleSinkConnector{
		info: ConnectorInfo{
			ID:          "console-sink",
			Name:        "Console Sink",
			Type:        ConnectorTypeSink,
			Version:     "1.0.0",
			Description: "Write data to console",
			Author:      "Edge Stream",
			CreatedAt:   time.Now(),
		},
		config:  config,
		status:  ConnectorStatusStopped,
		metrics: ConnectorMetrics{},
	}

	return connector, nil
}

// GetInfo 获取连接器信息
func (f *ConsoleSinkFactory) GetInfo() ConnectorInfo {
	return ConnectorInfo{
		ID:          "console-sink",
		Name:        "Console Sink",
		Type:        ConnectorTypeSink,
		Version:     "1.0.0",
		Description: "Write data to console",
		Author:      "Edge Stream",
		CreatedAt:   time.Now(),
	}
}

// ValidateConfig 验证配置
func (f *ConsoleSinkFactory) ValidateConfig(config map[string]interface{}) error {
	// 控制台接收器不需要特殊配置
	return nil
}

// GetInfo 获取连接器信息
func (c *ConsoleSinkConnector) GetInfo() ConnectorInfo {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.info
}

// Start 启动连接器
func (c *ConsoleSinkConnector) Start(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.running {
		return fmt.Errorf("connector is already running")
	}

	c.status = ConnectorStatusRunning
	c.startTime = time.Now()
	c.running = true

	return nil
}

// Stop 停止连接器
func (c *ConsoleSinkConnector) Stop(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.running {
		return fmt.Errorf("connector is not running")
	}

	c.status = ConnectorStatusStopped
	c.running = false

	return nil
}

// GetStatus 获取状态
func (c *ConsoleSinkConnector) GetStatus() ConnectorStatus {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.status
}

// GetMetrics 获取指标
func (c *ConsoleSinkConnector) GetMetrics() ConnectorMetrics {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	metrics := c.metrics
	if c.running {
		metrics.Uptime = time.Since(c.startTime)
		metrics.LastActivity = time.Now()
	}

	return metrics
}

// Validate 验证配置
func (c *ConsoleSinkConnector) Validate(config map[string]interface{}) error {
	return nil
}

// Configure 配置连接器
func (c *ConsoleSinkConnector) Configure(config map[string]interface{}) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.config = config
	c.info.Config = config
	c.info.UpdatedAt = time.Now()

	return nil
}

// Write 写入数据
func (c *ConsoleSinkConnector) Write(ctx context.Context, messages <-chan Message) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-messages:
			if !ok {
				return nil // 通道已关闭
			}

			c.mutex.RLock()
			if !c.running {
				c.mutex.RUnlock()
				return fmt.Errorf("connector is not running")
			}
			c.mutex.RUnlock()

			// 格式化输出
			prefix := fmt.Sprintf("[%s] [%s]", msg.GetTimestamp().Format("15:04:05"), msg.GetKey())
			fmt.Printf("%s %v\n", prefix, msg.GetValue())

			// 更新指标
			atomic.AddInt64(&c.metrics.MessagesProcessed, 1)
			if str, ok := msg.GetValue().(string); ok {
				atomic.AddInt64(&c.metrics.BytesProcessed, int64(len(str)))
			}
		}
	}
}

// Flush 刷新缓冲区
func (c *ConsoleSinkConnector) Flush(ctx context.Context) error {
	// 控制台输出不需要刷新
	return nil
}

// GetWriteCapacity 获取写入容量
func (c *ConsoleSinkConnector) GetWriteCapacity() int {
	return DefaultConsoleBatchSize // 控制台可以处理大量输出
}

// JSONTransformConnector JSON转换连接器
type JSONTransformConnector struct {
	info      ConnectorInfo
	config    map[string]interface{}
	status    ConnectorStatus
	metrics   ConnectorMetrics
	startTime time.Time
	mutex     sync.RWMutex
	running   bool
	rules     []TransformRule
}

// JSONTransformFactory JSON转换连接器工厂
type JSONTransformFactory struct{}

// Create 创建JSON转换连接器实例
func (f *JSONTransformFactory) Create(config map[string]interface{}) (Connector, error) {
	connector := &JSONTransformConnector{
		info: ConnectorInfo{
			ID:          "json-transform",
			Name:        "JSON Transform",
			Type:        ConnectorTypeTransform,
			Version:     "1.0.0",
			Description: "Transform JSON data",
			Author:      "Edge Stream",
			CreatedAt:   time.Now(),
		},
		config:  config,
		status:  ConnectorStatusStopped,
		metrics: ConnectorMetrics{},
		rules:   []TransformRule{},
	}

	return connector, nil
}

// GetInfo 获取连接器信息
func (f *JSONTransformFactory) GetInfo() ConnectorInfo {
	return ConnectorInfo{
		ID:          "json-transform",
		Name:        "JSON Transform",
		Type:        ConnectorTypeTransform,
		Version:     "1.0.0",
		Description: "Transform JSON data",
		Author:      "Edge Stream",
		CreatedAt:   time.Now(),
	}
}

// ValidateConfig 验证配置
func (f *JSONTransformFactory) ValidateConfig(config map[string]interface{}) error {
	return nil
}

// GetInfo 获取连接器信息
func (c *JSONTransformConnector) GetInfo() ConnectorInfo {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.info
}

// Start 启动连接器
func (c *JSONTransformConnector) Start(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.running {
		return fmt.Errorf("connector is already running")
	}

	c.status = ConnectorStatusRunning
	c.startTime = time.Now()
	c.running = true

	// 初始化转换规则
	c.rules = []TransformRule{
		{
			ID:     "add_timestamp",
			Name:   "Add Timestamp",
			Action: "add_field",
			Parameters: map[string]interface{}{
				"field": "processed_at",
				"value": "{{now}}",
			},
			Enabled:  true,
			Priority: 1,
		},
		{
			ID:     "uppercase_text",
			Name:   "Uppercase Text Fields",
			Action: "transform_field",
			Parameters: map[string]interface{}{
				"field":    "text",
				"function": "uppercase",
			},
			Enabled:  true,
			Priority: DefaultPriority,
		},
	}

	return nil
}

// Stop 停止连接器
func (c *JSONTransformConnector) Stop(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.running {
		return fmt.Errorf("connector is not running")
	}

	c.status = ConnectorStatusStopped
	c.running = false

	return nil
}

// GetStatus 获取状态
func (c *JSONTransformConnector) GetStatus() ConnectorStatus {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.status
}

// GetMetrics 获取指标
func (c *JSONTransformConnector) GetMetrics() ConnectorMetrics {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	metrics := c.metrics
	if c.running {
		metrics.Uptime = time.Since(c.startTime)
		metrics.LastActivity = time.Now()
	}

	return metrics
}

// Validate 验证配置
func (c *JSONTransformConnector) Validate(config map[string]interface{}) error {
	return nil
}

// Configure 配置连接器
func (c *JSONTransformConnector) Configure(config map[string]interface{}) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.config = config
	c.info.Config = config
	c.info.UpdatedAt = time.Now()

	return nil
}

// Transform 转换数据
func (c *JSONTransformConnector) Transform(ctx context.Context, input <-chan Message) (outputChan <-chan Message, errorChan <-chan error) {
	outputCh := make(chan Message, DefaultChannelBufferSize)
	errorCh := make(chan error, DefaultErrorChannelSize)

	go func() {
		defer close(outputCh)
		defer close(errorCh)

		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-input:
				if !ok {
					return
				}

				c.mutex.RLock()
				if !c.running {
					c.mutex.RUnlock()
					return
				}
				c.mutex.RUnlock()

				// 转换消息
				transformedMsg, err := c.transformMessage(msg)
				if err != nil {
					select {
					case errorCh <- err:
						atomic.AddInt64(&c.metrics.ErrorsCount, 1)
					case <-ctx.Done():
						return
					}
					continue
				}

				select {
				case outputCh <- transformedMsg:
					atomic.AddInt64(&c.metrics.MessagesProcessed, 1)
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return outputCh, errorCh
}

// parseMessageToData 解析消息为数据映射
func parseMessageToData(msg Message) map[string]interface{} {
	var data map[string]interface{}
	if str, ok := msg.GetValue().(string); ok {
		if err := json.Unmarshal([]byte(str), &data); err != nil {
			// 如果不是JSON，创建一个包装对象
			data = map[string]interface{}{
				"text": str,
			}
		}
	} else {
		// 如果不是字符串，尝试直接转换
		if jsonData, ok := msg.GetValue().(map[string]interface{}); ok {
			data = jsonData
		} else {
			data = map[string]interface{}{
				"value": msg.GetValue(),
			}
		}
	}
	return data
}

// applyAddFieldRule 应用添加字段规则
func applyAddFieldRule(data map[string]interface{}, rule *TransformRule) {
	field, ok := rule.Parameters["field"].(string)
	if !ok {
		return
	}
	value := rule.Parameters["value"]
	if valueStr, ok := value.(string); ok && valueStr == "{{now}}" {
		data[field] = time.Now().Format(time.RFC3339)
	} else {
		data[field] = value
	}
}

// applyTransformFieldRule 应用字段转换规则
func applyTransformFieldRule(data map[string]interface{}, rule *TransformRule) {
	field, ok := rule.Parameters["field"].(string)
	if !ok {
		return
	}
	function, ok := rule.Parameters["function"].(string)
	if !ok {
		return
	}

	if fieldValue, exists := data[field]; exists {
		if str, ok := fieldValue.(string); ok {
			switch function {
			case "uppercase":
				data[field] = strings.ToUpper(str)
			case "lowercase":
				data[field] = strings.ToLower(str)
			case "trim":
				data[field] = strings.TrimSpace(str)
			}
		}
	}
}

// applyTransformRules 应用转换规则
func (c *JSONTransformConnector) applyTransformRules(data map[string]interface{}) {
	for _, rule := range c.rules {
		if !rule.Enabled {
			continue
		}

		switch rule.Action {
		case "add_field":
			applyAddFieldRule(data, &rule)
		case "transform_field":
			applyTransformFieldRule(data, &rule)
		}
	}
}

// createTransformedMessage 创建转换后的消息
func createTransformedMessage(msg Message, data map[string]interface{}) (Message, error) {
	// 序列化回JSON
	transformedData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transformed data: %w", err)
	}

	// 创建新消息
	transformedMsg := NewStandardMessage(msg.GetKey(), string(transformedData))
	transformedMsg.SetPartition(msg.GetPartition())
	transformedMsg.SetOffset(msg.GetOffset())

	// 复制头部信息
	for k, v := range msg.GetHeaders() {
		transformedMsg.SetHeader(k, v)
	}
	transformedMsg.SetHeader("transformed", "true")
	transformedMsg.SetHeader("transformer", "json-transform")

	return transformedMsg, nil
}

// transformMessage 转换单个消息
func (c *JSONTransformConnector) transformMessage(msg Message) (Message, error) {
	// 解析消息为数据映射
	data := parseMessageToData(msg)

	// 应用转换规则
	c.applyTransformRules(data)

	// 创建转换后的消息
	return createTransformedMessage(msg, data)
}

// GetTransformRules 获取转换规则
func (c *JSONTransformConnector) GetTransformRules() []TransformRule {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.rules
}

// RegisterBuiltinConnectors 注册内置连接器
func RegisterBuiltinConnectors(registry ConnectorRegistry) error {
	// 注册文件源连接器
	if err := registry.Register(ConnectorTypeSource, "file", &FileSourceFactory{}); err != nil {
		return fmt.Errorf("failed to register file source connector: %w", err)
	}

	// 注册控制台接收器连接器
	if err := registry.Register(ConnectorTypeSink, "console", &ConsoleSinkFactory{}); err != nil {
		return fmt.Errorf("failed to register console sink connector: %w", err)
	}

	// 注册JSON转换连接器
	if err := registry.Register(ConnectorTypeTransform, "json", &JSONTransformFactory{}); err != nil {
		return fmt.Errorf("failed to register json transform connector: %w", err)
	}

	return nil
}

// CreateSampleDataFile 创建示例数据文件
func CreateSampleDataFile(filePath string) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, DefaultFilePermissions); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			// Log error but don't return it as it's in defer
		}
	}()

	// 写入示例数据
	sampleData := []string{
		`{"id": 1, "name": "Alice", "age": 30, "city": "New York"}`,
		`{"id": 2, "name": "Bob", "age": 25, "city": "Los Angeles"}`,
		`{"id": 3, "name": "Charlie", "age": 35, "city": "Chicago"}`,
		`{"id": 4, "name": "Diana", "age": 28, "city": "Houston"}`,
		`{"id": 5, "name": "Eve", "age": 32, "city": "Phoenix"}`,
		"Simple text message without JSON format",
		`{"id": 6, "name": "Frank", "age": 29, "city": "Philadelphia", "text": "hello world"}`,
		`{"id": 7, "name": "Grace", "age": 27, "city": "San Antonio", "text": "edge stream processing"}`,
		"Another plain text message",
		`{"id": 8, "name": "Henry", "age": 31, "city": "San Diego", "text": "data transformation"}`,
	}

	for i, line := range sampleData {
		if i > 0 {
			if _, err := file.WriteString("\n"); err != nil {
				return fmt.Errorf("failed to write newline: %w", err)
			}
		}
		if _, err := file.WriteString(line); err != nil {
			return fmt.Errorf("failed to write line: %w", err)
		}
	}

	return nil
}
