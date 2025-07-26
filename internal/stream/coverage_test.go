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
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestStreamMetrics 测试流指标功能
func TestStreamMetrics(t *testing.T) {
	metrics := NewStreamMetrics("test-processor")

	// 测试初始状态
	if metrics.ProcessorID != "test-processor" {
		t.Errorf("Expected processor ID 'test-processor', got '%s'", metrics.ProcessorID)
	}

	// 测试指标更新
	metrics.UpdateMetrics(10, 8, 1000, 800, 50*time.Millisecond, 1)

	if metrics.MessagesIn != 10 {
		t.Errorf("Expected MessagesIn 10, got %d", metrics.MessagesIn)
	}
	if metrics.MessagesOut != 8 {
		t.Errorf("Expected MessagesOut 8, got %d", metrics.MessagesOut)
	}
	if metrics.BytesIn != 1000 {
		t.Errorf("Expected BytesIn 1000, got %d", metrics.BytesIn)
	}
	if metrics.BytesOut != 800 {
		t.Errorf("Expected BytesOut 800, got %d", metrics.BytesOut)
	}
	if metrics.ErrorCount != 1 {
		t.Errorf("Expected ErrorCount 1, got %d", metrics.ErrorCount)
	}

	// 测试指标获取
	metricsSnapshot := metrics.GetMetrics()
	if metricsSnapshot.ProcessorID != "test-processor" {
		t.Errorf("Expected snapshot processor ID 'test-processor', got '%s'", metricsSnapshot.ProcessorID)
	}
}

// TestFileSourceProcessor 测试文件源处理器
func TestFileSourceProcessor(t *testing.T) {
	processor, _ := setupFileSourceProcessor(t)
	testFileSourceBasicProperties(t, processor)
	testFileSourceErrorCases(t, processor)
	testFileSourceStartStop(t, processor)
}

func setupFileSourceProcessor(t *testing.T) (*FileSourceProcessor, string) {
	// 创建临时测试文件
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "line1\nline2\nline3\n"
	err := os.WriteFile(testFile, []byte(testContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// 创建文件源处理器
	processor := NewFileSourceProcessor("file-source", "Test File Source", testFile)
	return processor, testFile
}

func testFileSourceBasicProperties(t *testing.T, processor *FileSourceProcessor) {
	// 测试基本属性
	if processor.GetID() != "file-source" {
		t.Errorf("Expected ID 'file-source', got '%s'", processor.GetID())
	}
	if processor.GetName() != "Test File Source" {
		t.Errorf("Expected name 'Test File Source', got '%s'", processor.GetName())
	}
	if processor.GetType() != StreamTypeSource {
		t.Errorf("Expected type %s, got %s", StreamTypeSource, processor.GetType())
	}
	if processor.GetStatus() != StreamStatusStopped {
		t.Errorf("Expected status %s, got %s", StreamStatusStopped, processor.GetStatus())
	}
}

func testFileSourceErrorCases(t *testing.T, processor *FileSourceProcessor) {
	// 测试Process方法（源处理器不应该处理输入消息）
	_, err := processor.Process(context.Background(), &Message{})
	if err == nil {
		t.Error("Expected error when calling Process on source processor")
	}

	// 测试Read方法（不应该直接调用）
	_, err = processor.Read(context.Background())
	if err == nil {
		t.Error("Expected error when calling Read directly")
	}

	// 测试启动前没有输出通道的情况
	err = processor.Start(context.Background())
	if err == nil {
		t.Error("Expected error when starting without output channels")
	}
}

func testFileSourceStartStop(t *testing.T, processor *FileSourceProcessor) {
	// 设置输出通道
	outputChan := make(chan *Message, 10)
	processor.SetOutput(outputChan)

	// 测试启动
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := processor.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start processor: %v", err)
	}

	if processor.GetStatus() != StreamStatusRunning {
		t.Errorf("Expected status %s after start, got %s", StreamStatusRunning, processor.GetStatus())
	}

	// 测试重复启动
	err = processor.Start(ctx)
	if err == nil {
		t.Error("Expected error when starting already running processor")
	}

	// 收集和验证消息
	testFileSourceMessages(t, outputChan)

	// 测试停止
	err = processor.Stop(ctx)
	if err != nil {
		t.Fatalf("Failed to stop processor: %v", err)
	}

	// 等待状态更新
	time.Sleep(100 * time.Millisecond)
	if processor.GetStatus() != StreamStatusStopped {
		t.Errorf("Expected status %s after stop, got %s", StreamStatusStopped, processor.GetStatus())
	}

	// 测试停止非运行状态的处理器
	err = processor.Stop(ctx)
	if err == nil {
		t.Error("Expected error when stopping non-running processor")
	}
}

func testFileSourceMessages(t *testing.T, outputChan chan *Message) {
	// 收集消息
	var messages []*Message
	timeout := time.After(1 * time.Second)
	for len(messages) < 3 {
		select {
		case msg := <-outputChan:
			messages = append(messages, msg)
		case <-timeout:
			t.Fatal("Timeout waiting for messages")
		}
	}

	// 验证消息
	if len(messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(messages))
	}

	expectedLines := []string{"line1", "line2", "line3"}
	for i, msg := range messages {
		if msg.Data != expectedLines[i] {
			t.Errorf("Expected message data '%s', got '%s'", expectedLines[i], msg.Data)
		}
		if msg.Headers["source"] != "file-source" {
			t.Errorf("Expected source header 'file-source', got '%v'", msg.Headers["source"])
		}
	}
}

// TestTransformProcessor 测试转换处理器
func TestTransformProcessor(t *testing.T) {
	processor := NewTransformProcessor("transform", "Test Transform", func(ctx context.Context, msg *Message) ([]*Message, error) {
		// 简单的转换：将字符串转为大写
		if str, ok := msg.Data.(string); ok {
			newMsg := &Message{
				ID:        msg.ID + "_transformed",
				Timestamp: time.Now(),
				Data:      strings.ToUpper(str),
				Headers:   msg.Headers,
				Partition: msg.Partition,
				Offset:    msg.Offset,
			}
			return []*Message{newMsg}, nil
		}
		return []*Message{msg}, nil
	})

	// 测试基本属性
	if processor.GetID() != "transform" {
		t.Errorf("Expected ID 'transform', got '%s'", processor.GetID())
	}
	if processor.GetName() != "Test Transform" {
		t.Errorf("Expected name 'Test Transform', got '%s'", processor.GetName())
	}
	if processor.GetType() != StreamTypeTransform {
		t.Errorf("Expected type %s, got %s", StreamTypeTransform, processor.GetType())
	}

	// 测试转换功能
	inputMsg := &Message{
		ID:        "test-msg",
		Timestamp: time.Now(),
		Data:      "hello world",
		Headers:   make(map[string]interface{}),
		Partition: "default",
		Offset:    1,
	}

	result, err := processor.Transform(context.Background(), inputMsg)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("Expected 1 result message, got %d", len(result))
	}

	if result[0].Data != "HELLO WORLD" {
		t.Errorf("Expected transformed data 'HELLO WORLD', got '%s'", result[0].Data)
	}

	// 测试Process方法
	result2, err := processor.Process(context.Background(), inputMsg)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if len(result2) != 1 {
		t.Errorf("Expected 1 result message from Process, got %d", len(result2))
	}

	// 测试启动和停止
	ctx := context.Background()
	err = processor.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start processor: %v", err)
	}

	if processor.GetStatus() != StreamStatusRunning {
		t.Errorf("Expected status %s after start, got %s", StreamStatusRunning, processor.GetStatus())
	}

	err = processor.Stop(ctx)
	if err != nil {
		t.Fatalf("Failed to stop processor: %v", err)
	}

	if processor.GetStatus() != StreamStatusStopped {
		t.Errorf("Expected status %s after stop, got %s", StreamStatusStopped, processor.GetStatus())
	}
}

// TestSinkProcessor 测试接收器处理器
func TestSinkProcessor(t *testing.T) {
	var receivedMessages []*Message
	var mutex sync.Mutex

	processor := NewSinkProcessor("sink", "Test Sink", func(ctx context.Context, msg *Message) error {
		mutex.Lock()
		defer mutex.Unlock()
		receivedMessages = append(receivedMessages, msg)
		return nil
	})

	// 测试基本属性
	if processor.GetID() != "sink" {
		t.Errorf("Expected ID 'sink', got '%s'", processor.GetID())
	}
	if processor.GetName() != "Test Sink" {
		t.Errorf("Expected name 'Test Sink', got '%s'", processor.GetName())
	}
	if processor.GetType() != StreamTypeSink {
		t.Errorf("Expected type %s, got %s", StreamTypeSink, processor.GetType())
	}

	// 测试写入功能
	testMsg := &Message{
		ID:        "test-sink-msg",
		Timestamp: time.Now(),
		Data:      "test data",
		Headers:   make(map[string]interface{}),
		Partition: "default",
		Offset:    1,
	}

	err := processor.Write(context.Background(), testMsg)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	mutex.Lock()
	if len(receivedMessages) != 1 {
		t.Errorf("Expected 1 received message, got %d", len(receivedMessages))
	}
	if receivedMessages[0].ID != "test-sink-msg" {
		t.Errorf("Expected message ID 'test-sink-msg', got '%s'", receivedMessages[0].ID)
	}
	mutex.Unlock()

	// 测试Process方法（接收器不应该返回消息）
	result, err := processor.Process(context.Background(), testMsg)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("Expected 0 result messages from sink processor, got %d", len(result))
	}

	// 测试启动和停止
	ctx := context.Background()
	err = processor.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start processor: %v", err)
	}

	err = processor.Stop(ctx)
	if err != nil {
		t.Fatalf("Failed to stop processor: %v", err)
	}
}

// TestStreamEngineTopologyManagement 测试拓扑管理功能
func TestStreamEngineTopologyManagement(t *testing.T) {
	engine := NewStandardStreamEngine()

	// 测试创建拓扑
	topology, err := engine.CreateTopology("test-topo", "Test Topology", "A test topology")
	if err != nil {
		t.Fatalf("Failed to create topology: %v", err)
	}

	if topology.ID != "test-topo" {
		t.Errorf("Expected topology ID 'test-topo', got '%s'", topology.ID)
	}
	if topology.Name != "Test Topology" {
		t.Errorf("Expected topology name 'Test Topology', got '%s'", topology.Name)
	}

	// 测试重复创建拓扑
	_, err = engine.CreateTopology("test-topo", "Duplicate", "Duplicate topology")
	if err == nil {
		t.Error("Expected error when creating duplicate topology")
	}

	// 测试获取拓扑
	retrieved, err := engine.GetTopology("test-topo")
	if err != nil {
		t.Fatalf("Failed to get topology: %v", err)
	}
	if retrieved.ID != "test-topo" {
		t.Errorf("Expected retrieved topology ID 'test-topo', got '%s'", retrieved.ID)
	}

	// 测试获取不存在的拓扑
	_, err = engine.GetTopology("non-existent")
	if err == nil {
		t.Error("Expected error when getting non-existent topology")
	}

	// 测试列出拓扑
	topologies, err := engine.ListTopologies()
	if err != nil {
		t.Fatalf("Failed to list topologies: %v", err)
	}
	if len(topologies) != 1 {
		t.Errorf("Expected 1 topology, got %d", len(topologies))
	}

	// 测试删除拓扑
	err = engine.DeleteTopology("test-topo")
	if err != nil {
		t.Fatalf("Failed to delete topology: %v", err)
	}

	// 验证拓扑已删除
	_, err = engine.GetTopology("test-topo")
	if err == nil {
		t.Error("Expected error when getting deleted topology")
	}

	// 测试删除不存在的拓扑
	err = engine.DeleteTopology("non-existent")
	if err == nil {
		t.Error("Expected error when deleting non-existent topology")
	}
}

// TestStreamEngineProcessorManagement 测试处理器管理功能
func TestStreamEngineProcessorManagement(t *testing.T) {
	engine := NewStandardStreamEngine()

	// 创建拓扑
	topology, err := engine.CreateTopology("processor-test", "Processor Test", "Test processor management")
	if err != nil {
		t.Fatalf("Failed to create topology: %v", err)
	}

	// 创建处理器
	processor := NewTransformProcessor("test-processor", "Test Processor", func(ctx context.Context, msg *Message) ([]*Message, error) {
		return []*Message{msg}, nil
	})

	// 测试添加处理器
	err = engine.AddProcessor(topology.ID, processor)
	if err != nil {
		t.Fatalf("Failed to add processor: %v", err)
	}

	// 测试重复添加处理器
	err = engine.AddProcessor(topology.ID, processor)
	if err == nil {
		t.Error("Expected error when adding duplicate processor")
	}

	// 测试向不存在的拓扑添加处理器
	err = engine.AddProcessor("non-existent", processor)
	if err == nil {
		t.Error("Expected error when adding processor to non-existent topology")
	}

	// 测试移除处理器
	err = engine.RemoveProcessor(topology.ID, processor.GetID())
	if err != nil {
		t.Fatalf("Failed to remove processor: %v", err)
	}

	// 测试移除不存在的处理器
	err = engine.RemoveProcessor(topology.ID, "non-existent")
	if err == nil {
		t.Error("Expected error when removing non-existent processor")
	}

	// 测试从不存在的拓扑移除处理器
	err = engine.RemoveProcessor("non-existent", processor.GetID())
	if err == nil {
		t.Error("Expected error when removing processor from non-existent topology")
	}

	// 清理
	if err := engine.DeleteTopology(topology.ID); err != nil {
		t.Logf("Failed to delete topology: %v", err)
	}
}

// TestStreamEngineConnectionManagement 测试连接管理功能
func TestStreamEngineConnectionManagement(t *testing.T) {
	engine := NewStandardStreamEngine()

	// 创建拓扑
	topology, err := engine.CreateTopology("connection-test", "Connection Test", "Test connection management")
	if err != nil {
		t.Fatalf("Failed to create topology: %v", err)
	}

	// 创建处理器
	sourceProcessor := NewTransformProcessor("source", "Source", func(ctx context.Context, msg *Message) ([]*Message, error) {
		return []*Message{msg}, nil
	})
	targetProcessor := NewTransformProcessor("target", "Target", func(ctx context.Context, msg *Message) ([]*Message, error) {
		return []*Message{msg}, nil
	})

	// 添加处理器
	if err := engine.AddProcessor(topology.ID, sourceProcessor); err != nil {
		t.Fatalf("Failed to add source processor: %v", err)
	}
	if err := engine.AddProcessor(topology.ID, targetProcessor); err != nil {
		t.Fatalf("Failed to add target processor: %v", err)
	}

	// 测试连接处理器
	err = engine.ConnectProcessors(topology.ID, "source", "target", 10)
	if err != nil {
		t.Fatalf("Failed to connect processors: %v", err)
	}

	// 测试重复连接
	err = engine.ConnectProcessors(topology.ID, "source", "target", 10)
	if err == nil {
		t.Error("Expected error when creating duplicate connection")
	}

	// 测试连接不存在的处理器
	err = engine.ConnectProcessors(topology.ID, "non-existent", "target", 10)
	if err == nil {
		t.Error("Expected error when connecting non-existent source processor")
	}

	err = engine.ConnectProcessors(topology.ID, "source", "non-existent", 10)
	if err == nil {
		t.Error("Expected error when connecting to non-existent target processor")
	}

	// 测试在不存在的拓扑中连接
	err = engine.ConnectProcessors("non-existent", "source", "target", 10)
	if err == nil {
		t.Error("Expected error when connecting in non-existent topology")
	}

	// 测试断开连接
	err = engine.DisconnectProcessors(topology.ID, "source", "target")
	if err != nil {
		t.Fatalf("Failed to disconnect processors: %v", err)
	}

	// 测试断开不存在的连接
	err = engine.DisconnectProcessors(topology.ID, "source", "target")
	if err == nil {
		t.Error("Expected error when disconnecting non-existent connection")
	}

	// 测试在不存在的拓扑中断开连接
	err = engine.DisconnectProcessors("non-existent", "source", "target")
	if err == nil {
		t.Error("Expected error when disconnecting in non-existent topology")
	}

	// 清理
	if err := engine.DeleteTopology(topology.ID); err != nil {
		t.Logf("Failed to delete topology: %v", err)
	}
}

// TestStreamEngineStatus 测试拓扑状态
func TestStreamEngineStatus(t *testing.T) {
	engine := NewStandardStreamEngine()

	// 创建拓扑
	topology, err := engine.CreateTopology("status-test", "Status Test", "Test status")
	if err != nil {
		t.Fatalf("Failed to create topology: %v", err)
	}

	// 测试初始状态
	status, err := engine.GetTopologyStatus(topology.ID)
	if err != nil {
		t.Fatalf("Failed to get topology status: %v", err)
	}

	if status != StreamStatusStopped {
		t.Errorf("Expected initial status %s, got %s", StreamStatusStopped, status)
	}

	// 测试不存在拓扑的状态
	_, err = engine.GetTopologyStatus("non-existent")
	if err == nil {
		t.Error("Expected error when getting status for non-existent topology")
	}

	// 清理
	if err := engine.DeleteTopology(topology.ID); err != nil {
		t.Logf("Failed to delete topology: %v", err)
	}
}
