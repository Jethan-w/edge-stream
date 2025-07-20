package pipeline

import (
	"context"
	"testing"
	"time"
)

// TestFlowFile 测试FlowFile基本功能
func TestFlowFile(t *testing.T) {
	// 创建FlowFile
	flowFile := NewFlowFile()

	// 测试基本属性
	if flowFile.UUID == "" {
		t.Error("FlowFile UUID should not be empty")
	}

	if flowFile.LineageID == "" {
		t.Error("FlowFile LineageID should not be empty")
	}

	// 测试属性设置
	flowFile.AddAttribute("test_key", "test_value")
	if flowFile.GetAttribute("test_key") != "test_value" {
		t.Error("FlowFile attribute not set correctly")
	}

	// 测试内容设置
	testContent := []byte("test content")
	flowFile.SetContent(testContent)

	if flowFile.GetSize() != int64(len(testContent)) {
		t.Error("FlowFile size not set correctly")
	}

	if string(flowFile.GetContent()) != string(testContent) {
		t.Error("FlowFile content not set correctly")
	}
}

// TestProcessGroup 测试ProcessGroup基本功能
func TestProcessGroup(t *testing.T) {
	// 创建ProcessGroup
	group := NewProcessGroup("test-group", "Test Group")

	if group.ID != "test-group" {
		t.Error("ProcessGroup ID not set correctly")
	}

	if group.Name != "Test Group" {
		t.Error("ProcessGroup Name not set correctly")
	}

	// 测试处理器管理
	processor := NewProcessorNode("test-processor", "Test Processor", "LogProcessor", nil)

	err := group.AddProcessor(processor)
	if err != nil {
		t.Errorf("Failed to add processor: %v", err)
	}

	// 验证处理器已添加
	retrievedProcessor, exists := group.GetProcessor("test-processor")
	if !exists {
		t.Error("Processor not found after adding")
	}

	if retrievedProcessor.ID != "test-processor" {
		t.Error("Retrieved processor ID mismatch")
	}

	// 测试连接管理
	processor2 := NewProcessorNode("test-processor-2", "Test Processor 2", "LogProcessor", nil)
	group.AddProcessor(processor2)

	connection := NewConnection("test-connection", "Test Connection", processor, processor2)

	err = group.AddConnection(connection)
	if err != nil {
		t.Errorf("Failed to add connection: %v", err)
	}

	// 验证连接已添加
	retrievedConnection, exists := group.GetConnection("test-connection")
	if !exists {
		t.Error("Connection not found after adding")
	}

	if retrievedConnection.ID != "test-connection" {
		t.Error("Retrieved connection ID mismatch")
	}
}

// TestConnection 测试Connection基本功能
func TestConnection(t *testing.T) {
	// 创建处理器和连接
	sourceProcessor := NewProcessorNode("source", "Source", "LogProcessor", nil)
	destProcessor := NewProcessorNode("dest", "Destination", "LogProcessor", nil)

	connection := NewConnection("test-conn", "Test Connection", sourceProcessor, destProcessor)

	// 测试队列功能
	flowFile1 := NewFlowFile()
	flowFile1.SetContent([]byte("data1"))
	flowFile1.AddAttribute("order", "1")

	flowFile2 := NewFlowFile()
	flowFile2.SetContent([]byte("data2"))
	flowFile2.AddAttribute("order", "2")

	// 入队
	err := connection.Enqueue(flowFile1)
	if err != nil {
		t.Errorf("Failed to enqueue flowFile1: %v", err)
	}

	err = connection.Enqueue(flowFile2)
	if err != nil {
		t.Errorf("Failed to enqueue flowFile2: %v", err)
	}

	// 检查队列大小
	if connection.GetQueueSize() != 2 {
		t.Errorf("Expected queue size 2, got %d", connection.GetQueueSize())
	}

	// 测试优先级策略
	prioritizer := NewOldestFirstPrioritizer()
	connection.SetPrioritizer(prioritizer)

	// 出队
	dequeuedFlowFile, err := connection.Dequeue()
	if err != nil {
		t.Errorf("Failed to dequeue: %v", err)
	}

	// 验证先进先出
	if dequeuedFlowFile.GetAttribute("order") != "1" {
		t.Error("OldestFirstPrioritizer not working correctly")
	}
}

// TestQueuePrioritizers 测试优先级策略
func TestQueuePrioritizers(t *testing.T) {
	// 创建测试队列
	queue := []*FlowFile{}

	flowFile1 := NewFlowFile()
	flowFile1.SetContent([]byte("data1"))
	flowFile1.Timestamp = time.Now().Add(-time.Hour) // 1小时前

	flowFile2 := NewFlowFile()
	flowFile2.SetContent([]byte("data2"))
	flowFile2.Timestamp = time.Now() // 现在

	queue = append(queue, flowFile1, flowFile2)

	// 测试先进先出
	oldestFirst := NewOldestFirstPrioritizer()
	selected := oldestFirst.SelectNext(queue)

	if selected != flowFile1 {
		t.Error("OldestFirstPrioritizer should select the oldest FlowFile")
	}

	// 测试后进先出
	newestFirst := NewNewestFirstPrioritizer()
	selected = newestFirst.SelectNext(queue)

	if selected != flowFile2 {
		t.Error("NewestFirstPrioritizer should select the newest FlowFile")
	}
}

// TestProcessorNode 测试ProcessorNode基本功能
func TestProcessorNode(t *testing.T) {
	// 创建处理器节点
	processor := NewProcessorNode("test-proc", "Test Processor", "LogProcessor", nil)

	// 测试状态管理
	if processor.GetState() != ProcessorNodeStateStopped {
		t.Error("New processor should be in stopped state")
	}

	processor.SetState(ProcessorNodeStateRunning)
	if processor.GetState() != ProcessorNodeStateRunning {
		t.Error("Processor state not set correctly")
	}

	// 测试配置管理
	processor.Configuration.SetProperty("test_prop", "test_value")
	if processor.Configuration.GetProperty("test_prop") != "test_value" {
		t.Error("Processor property not set correctly")
	}
}

// TestPipelineConfiguration 测试PipelineConfiguration
func TestPipelineConfiguration(t *testing.T) {
	config := NewPipelineConfiguration()

	config.SetProperty("test_key", "test_value")
	if config.GetProperty("test_key") != "test_value" {
		t.Error("Configuration property not set correctly")
	}

	if config.GetProperty("non_existent") != "" {
		t.Error("Non-existent property should return empty string")
	}
}

// TestValidationResult 测试ValidationResult
func TestValidationResult(t *testing.T) {
	result := &ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	result.AddError("test error")
	if len(result.Errors) != 1 {
		t.Error("Error not added correctly")
	}

	if result.Errors[0] != "test error" {
		t.Error("Error message not set correctly")
	}

	result.AddWarning("test warning")
	if len(result.Warnings) != 1 {
		t.Error("Warning not added correctly")
	}

	if result.Warnings[0] != "test warning" {
		t.Error("Warning message not set correctly")
	}
}

// TestPipelineStatistics 测试PipelineStatistics
func TestPipelineStatistics(t *testing.T) {
	stats := NewPipelineStatistics()

	// 测试计数器
	stats.IncrementFlowFilesProcessed()
	if stats.TotalFlowFilesProcessed != 1 {
		t.Error("FlowFiles processed counter not incremented")
	}

	// 测试字节计数
	stats.AddBytesProcessed(1024)
	if stats.TotalBytesProcessed != 1024 {
		t.Error("Bytes processed not added correctly")
	}

	// 测试时间设置
	testTime := time.Now()
	stats.SetLastProcessedTime(testTime)
	if stats.LastProcessedTime != testTime {
		t.Error("Last processed time not set correctly")
	}

	// 测试活跃连接数
	stats.SetActiveConnections(5)
	if stats.ActiveConnections != 5 {
		t.Error("Active connections not set correctly")
	}

	// 测试活跃处理器数
	stats.SetActiveProcessors(3)
	if stats.ActiveProcessors != 3 {
		t.Error("Active processors not set correctly")
	}
}

// TestConnectionQueue 测试ConnectionQueue
func TestConnectionQueue(t *testing.T) {
	queue := NewConnectionQueue()

	// 测试空队列
	if queue.Size() != 0 {
		t.Error("New queue should be empty")
	}

	// 测试入队
	flowFile := NewFlowFile()
	err := queue.Enqueue(flowFile)
	if err != nil {
		t.Errorf("Failed to enqueue: %v", err)
	}

	if queue.Size() != 1 {
		t.Error("Queue size should be 1 after enqueue")
	}

	// 测试出队
	prioritizer := NewOldestFirstPrioritizer()
	dequeued, err := queue.Dequeue(prioritizer)
	if err != nil {
		t.Errorf("Failed to dequeue: %v", err)
	}

	if dequeued != flowFile {
		t.Error("Dequeued FlowFile should match enqueued FlowFile")
	}

	if queue.Size() != 0 {
		t.Error("Queue should be empty after dequeue")
	}
}

// TestUUIDGeneration 测试UUID生成
func TestUUIDGeneration(t *testing.T) {
	uuid1 := generateUUID()
	uuid2 := generateUUID()

	if uuid1 == "" {
		t.Error("Generated UUID should not be empty")
	}

	if uuid1 == uuid2 {
		t.Error("Generated UUIDs should be unique")
	}
}

// TestLineageIDGeneration 测试LineageID生成
func TestLineageIDGeneration(t *testing.T) {
	lineageID1 := generateLineageID()
	lineageID2 := generateLineageID()

	if lineageID1 == "" {
		t.Error("Generated LineageID should not be empty")
	}

	if lineageID1 == lineageID2 {
		t.Error("Generated LineageIDs should be unique")
	}
}

// BenchmarkFlowFileCreation 基准测试FlowFile创建
func BenchmarkFlowFileCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewFlowFile()
	}
}

// BenchmarkFlowFileAttributeAccess 基准测试FlowFile属性访问
func BenchmarkFlowFileAttributeAccess(b *testing.B) {
	flowFile := NewFlowFile()
	flowFile.AddAttribute("test_key", "test_value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		flowFile.GetAttribute("test_key")
	}
}

// BenchmarkConnectionEnqueue 基准测试连接入队
func BenchmarkConnectionEnqueue(b *testing.B) {
	sourceProcessor := NewProcessorNode("source", "Source", "LogProcessor", nil)
	destProcessor := NewProcessorNode("dest", "Destination", "LogProcessor", nil)
	connection := NewConnection("test-conn", "Test Connection", sourceProcessor, destProcessor)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		flowFile := NewFlowFile()
		connection.Enqueue(flowFile)
	}
}
