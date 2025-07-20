package example

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/crazy/edge-stream/internal/flowfile"
	"github.com/crazy/edge-stream/internal/pipeline"
	"github.com/crazy/edge-stream/internal/processor"
)

// ExamplePipelineUsage 展示Pipeline模块的基本使用
func ExamplePipelineUsage() {
	fmt.Println("=== Pipeline模块使用示例 ===")

	// 创建上下文
	// ctx := context.Background()

	// 1. 流程设计与编排示例
	fmt.Println("\n1. 流程设计与编排示例")
	examplePipelineDesign()

	// 2. 版本管理示例
	fmt.Println("\n2. 版本管理示例")
	exampleVersionManagement()

	// 3. 数据溯源示例
	fmt.Println("\n3. 数据溯源示例")
	exampleDataProvenance()

	// 4. FlowFile处理示例
	fmt.Println("\n4. FlowFile处理示例")
	exampleFlowFileProcessing()

	// 5. 流程组管理示例
	fmt.Println("\n5. 流程组管理示例")
	exampleProcessGroupManagement()
}

// examplePipelineDesign 流程设计与编排示例
func examplePipelineDesign() {
	// 创建流程设计器
	designer := pipeline.NewPipelineDesigner()

	// 初始化设计器
	ctx := context.Background()
	if err := designer.Initialize(ctx); err != nil {
		log.Fatalf("初始化流程设计器失败: %v", err)
	}

	// 创建流程组
	processGroup := designer.CreateProcessGroup("日志处理流程")
	fmt.Printf("创建流程组: %s (ID: %s)\n", processGroup.Name, processGroup.ID)

	// 添加处理器
	processor1 := pipeline.NewProcessorDTO("processor-1", "GetFile", "GetFile")
	processor1.SetProperty("Input Directory", "/var/log")
	processor1.SetProperty("File Filter", "*.log")
	processor1.SetScheduling("timer", "primary", 5*time.Second, 1)

	if err := designer.AddProcessor(processGroup, processor1); err != nil {
		log.Printf("添加处理器失败: %v", err)
	}

	processor2 := pipeline.NewProcessorDTO("processor-2", "SplitText", "SplitText")
	processor2.SetProperty("Line Split Count", "100")
	processor2.SetScheduling("event", "primary", 0, 2)

	if err := designer.AddProcessor(processGroup, processor2); err != nil {
		log.Printf("添加处理器失败: %v", err)
	}

	processor3 := pipeline.NewProcessorDTO("processor-3", "PutKafka", "PutKafka")
	processor3.SetProperty("Kafka Topic", "log-events")
	processor3.SetProperty("Bootstrap Servers", "localhost:9092")
	processor3.SetScheduling("event", "primary", 0, 1)

	if err := designer.AddProcessor(processGroup, processor3); err != nil {
		log.Printf("添加处理器失败: %v", err)
	}

	// 创建连接
	connection1 := pipeline.NewConnectionDTO("connection-1", "GetFile to SplitText", "processor-1", "processor-2")
	connection1.SetPrioritizer("oldest_first")

	if err := designer.CreateConnection(processGroup, connection1); err != nil {
		log.Printf("创建连接失败: %v", err)
	}

	connection2 := pipeline.NewConnectionDTO("connection-2", "SplitText to PutKafka", "processor-2", "processor-3")
	connection2.SetPrioritizer("newest_first")

	if err := designer.CreateConnection(processGroup, connection2); err != nil {
		log.Printf("创建连接失败: %v", err)
	}

	// 列出流程组信息
	fmt.Printf("流程组处理器: %v\n", processGroup.ListProcessors())
	fmt.Printf("流程组连接: %v\n", processGroup.ListConnections())
}

// exampleVersionManagement 版本管理示例
func exampleVersionManagement() {
	// 创建版本管理器
	versionManager := pipeline.NewVersionManager()

	// 初始化版本管理器
	ctx := context.Background()
	if err := versionManager.Initialize(ctx); err != nil {
		log.Fatalf("初始化版本管理器失败: %v", err)
	}

	// 创建流程组
	processGroup := pipeline.NewProcessGroup("version-test", "版本测试流程")

	// 添加一些处理器和连接
	processor1 := pipeline.NewProcessorNode("proc-1", "TestProcessor1", "LogProcessor", nil)
	processor2 := pipeline.NewProcessorNode("proc-2", "TestProcessor2", "LogProcessor", nil)

	processGroup.AddProcessor(processor1)
	processGroup.AddProcessor(processor2)

	connection := pipeline.NewConnection("conn-1", "TestConnection", processor1, processor2)
	processGroup.AddConnection(connection)

	// 保存第一个版本
	if err := versionManager.SaveVersion(processGroup, "初始版本"); err != nil {
		log.Printf("保存版本失败: %v", err)
	}

	// 修改流程组
	processor3 := pipeline.NewProcessorNode("proc-3", "TestProcessor3", "LogProcessor", nil)
	processGroup.AddProcessor(processor3)

	// 保存第二个版本
	if err := versionManager.SaveVersion(processGroup, "添加新处理器"); err != nil {
		log.Printf("保存版本失败: %v", err)
	}

	// 列出所有版本
	versions, err := versionManager.ListVersions()
	if err != nil {
		log.Printf("列出版本失败: %v", err)
	} else {
		fmt.Printf("版本列表:\n")
		for _, version := range versions {
			fmt.Printf("  - %s: %s (%s)\n", version.Version, version.Name, version.Comment)
		}
	}

	// 比较版本
	if len(versions) >= 2 {
		comparison, err := versionManager.CompareVersions(versions[0].ID+"-"+versions[0].Version, versions[1].ID+"-"+versions[1].Version)
		if err != nil {
			log.Printf("比较版本失败: %v", err)
		} else {
			fmt.Printf("版本比较结果:\n")
			for _, diff := range comparison.Differences {
				fmt.Printf("  - %s: %s\n", diff.Type.String(), diff.Description)
			}
		}
	}
}

// exampleDataProvenance 数据溯源示例
func exampleDataProvenance() {
	// 创建溯源跟踪器
	provenanceTracker := pipeline.NewProvenanceTracker()

	// 初始化溯源跟踪器
	ctx := context.Background()
	if err := provenanceTracker.Initialize(ctx); err != nil {
		log.Fatalf("初始化溯源跟踪器失败: %v", err)
	}

	// 创建FlowFile
	flowFile := flowfile.NewFlowFile()
	flowFile.Content = ([]byte("test data"))
	flowFile.Attributes["filename"] = "test.log"
	flowFile.Attributes["size"] = "9"

	fmt.Printf("创建FlowFile: %s (血缘ID: %s)\n", flowFile.UUID, flowFile.LineageID)

	// 记录溯源事件
	details1 := map[string]string{
		"source": "file_system",
		"path":   "/var/log/test.log",
	}

	if err := provenanceTracker.RecordEvent(flowFile, pipeline.EventTypeCreate, "source-processor", details1); err != nil {
		log.Printf("记录创建事件失败: %v", err)
	}

	// 模拟处理过程
	time.Sleep(100 * time.Millisecond)

	details2 := map[string]string{
		"operation": "split_text",
		"lines":     "1",
	}

	if err := provenanceTracker.RecordEvent(flowFile, pipeline.EventTypeProcess, "split-processor", details2); err != nil {
		log.Printf("记录处理事件失败: %v", err)
	}

	// 模拟发送过程
	time.Sleep(50 * time.Millisecond)

	details3 := map[string]string{
		"destination": "kafka",
		"topic":       "log-events",
	}

	if err := provenanceTracker.RecordEvent(flowFile, pipeline.EventTypeSend, "kafka-processor", details3); err != nil {
		log.Printf("记录发送事件失败: %v", err)
	}

	// 追踪FlowFile的血缘关系
	lineage, err := provenanceTracker.TraceFlowFile(flowFile.UUID)
	if err != nil {
		log.Printf("追踪FlowFile失败: %v", err)
	} else {
		fmt.Printf("FlowFile血缘追踪结果:\n")
		fmt.Printf("  血缘ID: %s\n", lineage.LineageID)
		fmt.Printf("  路径长度: %d\n", lineage.GetPathLength())
		fmt.Printf("  处理时长: %v\n", lineage.GetDuration())
		fmt.Printf("  处理器数量: %d\n", lineage.GetProcessorCount())

		fmt.Printf("  处理路径:\n")
		for i, node := range lineage.Path {
			fmt.Printf("    %d. %s -> %s (%s)\n", i+1, node.ProcessorID, node.EventType.String(), node.Timestamp.Format(time.RFC3339))
		}
	}

	// 创建血缘跟踪器
	lineageTracker := pipeline.NewDataLineageTracker(provenanceTracker)

	// 获取血缘统计信息
	stats, err := lineageTracker.GetLineageStatistics(flowFile.LineageID)
	if err != nil {
		log.Printf("获取血缘统计失败: %v", err)
	} else {
		fmt.Printf("血缘统计信息:\n")
		fmt.Printf("  血缘ID: %s\n", stats.LineageID)
		fmt.Printf("  路径长度: %d\n", stats.PathLength)
		fmt.Printf("  处理时长: %v\n", stats.Duration)
		fmt.Printf("  处理器数量: %d\n", stats.ProcessorCount)
		fmt.Printf("  事件类型统计: %+v\n", stats.EventTypeCounts)
	}

	// 创建溯源监控器
	provenanceMonitor := pipeline.NewProvenanceMonitor(provenanceTracker)

	// 获取事件统计
	eventCount, err := provenanceMonitor.GetEventCount()
	if err != nil {
		log.Printf("获取事件数量失败: %v", err)
	} else {
		fmt.Printf("总事件数量: %d\n", eventCount)
	}

	// 获取最近事件
	recentEvents, err := provenanceMonitor.GetRecentEvents(5)
	if err != nil {
		log.Printf("获取最近事件失败: %v", err)
	} else {
		fmt.Printf("最近5个事件:\n")
		for i, event := range recentEvents {
			fmt.Printf("  %d. %s -> %s (%s)\n", i+1, event.ProcessorID, event.EventType.String(), event.Timestamp.Format(time.RFC3339))
		}
	}
}

// exampleFlowFileProcessing FlowFile处理示例
func exampleFlowFileProcessing() {
	// 创建FlowFile
	flowFile := flowfile.NewFlowFile()
	flowFile.Content = ([]byte("Hello, EdgeStream Pipeline!"))
	flowFile.Attributes["filename"] = "hello.txt"
	flowFile.Attributes["encoding"] = "UTF-8"
	flowFile.Attributes["size"] = "26"

	fmt.Printf("创建FlowFile:\n")
	fmt.Printf("  UUID: %s\n", flowFile.UUID)
	fmt.Printf("  血缘ID: %s\n", flowFile.LineageID)
	fmt.Printf("  大小: %d bytes\n", flowFile.Size)
	fmt.Printf("  时间戳: %s\n", flowFile.Timestamp.Format(time.RFC3339))
	fmt.Printf("  内容: %s\n", string(flowFile.Content))

	// 修改FlowFile属性
	flowFile.Attributes["processed"] = "true"
	flowFile.Attributes["processor"] = "example-processor"

	fmt.Printf("修改后的属性:\n")
	for key, value := range flowFile.Attributes {
		fmt.Printf("  %s: %s\n", key, value)
	}

	// 创建连接队列
	queue := pipeline.NewConnectionQueue()

	// 入队FlowFile
	if err := queue.Enqueue(flowFile); err != nil {
		log.Printf("入队失败: %v", err)
	}

	fmt.Printf("队列大小: %d\n", queue.Size())

	// 使用不同优先级策略出队
	oldestPrioritizer := pipeline.NewOldestFirstPrioritizer()
	newestPrioritizer := pipeline.NewNewestFirstPrioritizer()

	// 使用先进先出策略
	dequeuedFlowFile, err := queue.Dequeue(oldestPrioritizer)
	if err != nil {
		log.Printf("出队失败: %v", err)
	} else {
		fmt.Printf("出队FlowFile: %s\n", dequeuedFlowFile.UUID)
	}

	// 重新入队
	queue.Enqueue(flowFile)

	// 使用后进先出策略
	dequeuedFlowFile, err = queue.Dequeue(newestPrioritizer)
	if err != nil {
		log.Printf("出队失败: %v", err)
	} else {
		fmt.Printf("出队FlowFile: %s\n", dequeuedFlowFile.UUID)
	}
}

// exampleProcessGroupManagement 流程组管理示例
func exampleProcessGroupManagement() {
	// 创建根流程组
	rootGroup := pipeline.NewProcessGroup("root", "根流程组")

	// 创建子流程组
	childGroup1 := pipeline.NewProcessGroup("child1", "子流程组1")
	childGroup2 := pipeline.NewProcessGroup("child2", "子流程组2")

	// 添加子流程组
	rootGroup.AddChildGroup(childGroup1)
	rootGroup.AddChildGroup(childGroup2)

	// 在子流程组中添加处理器
	processor1 := pipeline.NewProcessorNode("proc-1", "处理器1", "LogProcessor", nil)
	processor2 := pipeline.NewProcessorNode("proc-2", "处理器2", "LogProcessor", nil)

	childGroup1.AddProcessor(processor1)
	childGroup2.AddProcessor(processor2)

	// 创建连接
	connection := pipeline.NewConnection("conn-1", "连接1", processor1, processor2)
	childGroup1.AddConnection(connection)

	// 列出流程组信息
	fmt.Printf("根流程组信息:\n")
	fmt.Printf("  ID: %s\n", rootGroup.ID)
	fmt.Printf("  名称: %s\n", rootGroup.Name)
	fmt.Printf("  子流程组: %v\n", rootGroup.ListChildGroups())

	fmt.Printf("子流程组1信息:\n")
	fmt.Printf("  ID: %s\n", childGroup1.ID)
	fmt.Printf("  名称: %s\n", childGroup1.Name)
	fmt.Printf("  处理器: %v\n", childGroup1.ListProcessors())
	fmt.Printf("  连接: %v\n", childGroup1.ListConnections())
	fmt.Printf("  父流程组: %s\n", childGroup1.GetParentGroup().Name)

	// 验证流程组
	result := validateProcessGroup(rootGroup)
	if result.Valid {
		fmt.Printf("流程组验证通过\n")
	} else {
		fmt.Printf("流程组验证失败:\n")
		for _, err := range result.Errors {
			fmt.Printf("  - %s\n", err)
		}
	}

	// 移除子流程组
	rootGroup.RemoveChildGroup("child1")
	fmt.Printf("移除子流程组后，根流程组的子流程组: %v\n", rootGroup.ListChildGroups())
}

// validateProcessGroup 验证流程组
func validateProcessGroup(group *pipeline.ProcessGroup) *pipeline.ValidationResult {
	result := &pipeline.ValidationResult{Valid: true}

	// 检查处理器
	processors := group.ListProcessors()
	if len(processors) == 0 {
		result.AddWarning("流程组没有处理器")
	}

	// 检查连接
	connections := group.ListConnections()
	if len(connections) == 0 {
		result.AddWarning("流程组没有连接")
	}

	// 检查子流程组
	childGroups := group.ListChildGroups()
	for _, childGroupID := range childGroups {
		childGroup, exists := group.GetChildGroup(childGroupID)
		if exists {
			childResult := validateProcessGroup(childGroup)
			if !childResult.Valid {
				for _, err := range childResult.Errors {
					result.AddError(fmt.Sprintf("子流程组 %s: %s", childGroupID, err))
				}
			}
		}
	}

	return result
}

// ExampleCustomProcessor 自定义处理器示例
func ExampleCustomProcessor() {
	fmt.Println("\n=== 自定义处理器示例 ===")

	// 创建流程设计器
	designer := pipeline.NewPipelineDesigner()

	// 注册自定义处理器
	designer.ProcessorFactory.RegisterProcessor("CustomProcessor", func() interface{} {
		return processor.FormatConverter{}
	})

	// 创建流程组
	processGroup := designer.CreateProcessGroup("custom-processor-test")

	// 添加自定义处理器
	processorDTO := pipeline.NewProcessorDTO("custom-1", "CustomProcessor", "CustomProcessor")
	processorDTO.SetProperty("custom.property", "custom.value")

	if err := designer.AddProcessor(processGroup, processorDTO); err != nil {
		log.Printf("添加自定义处理器失败: %v", err)
	} else {
		fmt.Printf("成功添加自定义处理器\n")
	}
}

// CustomProcessor 自定义处理器
type CustomProcessor struct{}

// Process 处理数据
func (cp *CustomProcessor) Process(ctx context.Context, data interface{}) (interface{}, error) {
	fmt.Printf("[CustomProcessor] 处理数据: %+v\n", data)
	return data, nil
}

// ExamplePipelineOrchestration 流程编排示例
func ExamplePipelineOrchestration() {
	fmt.Println("\n=== 流程编排示例 ===")

	// 创建流程设计器
	designer := pipeline.NewPipelineDesigner()

	// 创建流程编排器
	orchestrator := pipeline.NewPipelineOrchestrator(designer)

	// 初始化编排器
	ctx := context.Background()
	if err := orchestrator.Initialize(ctx); err != nil {
		log.Fatalf("初始化流程编排器失败: %v", err)
	}

	// 创建流程组
	processGroup := designer.CreateProcessGroup("orchestration-test")

	// 添加处理器
	processor1 := pipeline.NewProcessorDTO("proc-1", "LogProcessor", "LogProcessor")
	processor2 := pipeline.NewProcessorDTO("proc-2", "LogProcessor", "LogProcessor")

	designer.AddProcessor(processGroup, processor1)
	designer.AddProcessor(processGroup, processor2)

	// 创建连接
	connection := pipeline.NewConnectionDTO("conn-1", "连接", "proc-1", "proc-2")
	designer.CreateConnection(processGroup, connection)

	// 部署流程
	if err := orchestrator.DeployPipeline(processGroup); err != nil {
		log.Printf("部署流程失败: %v", err)
	} else {
		fmt.Printf("流程部署成功\n")
	}

	// 启动流程
	if err := orchestrator.StartPipeline(processGroup); err != nil {
		log.Printf("启动流程失败: %v", err)
	} else {
		fmt.Printf("流程启动成功\n")
	}

	// 等待一段时间
	time.Sleep(2 * time.Second)

	// 停止流程
	if err := orchestrator.StopPipeline(processGroup); err != nil {
		log.Printf("停止流程失败: %v", err)
	} else {
		fmt.Printf("流程停止成功\n")
	}
}
