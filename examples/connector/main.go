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

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/crazy/edge-stream/internal/connector"
)

func main() {
	fmt.Println("=== Edge Stream 连接器注册中心示例 ===")

	// 初始化注册中心和管理器
	registry, manager := initializeConnectorSystem()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		manager.Close(ctx)
	}()

	// 创建示例数据文件
	dataFile := createSampleData()

	// 创建和配置连接器
	ctx := context.Background()
	connectorIDs := createConnectors(ctx, manager, dataFile)

	// 启动连接器
	startConnectors(ctx, manager, connectorIDs)

	// 运行数据处理管道
	runDataPipeline(ctx, manager)

	// 运行监控和管理操作
	runMonitoringAndManagement(ctx, manager, registry)

	// 清理资源
	cleanupResources(ctx, manager)

	fmt.Println("\n=== 连接器注册中心示例完成 ===")
}

// initializeConnectorSystem 初始化连接器注册中心和管理器
func initializeConnectorSystem() (*connector.StandardConnectorRegistry, *connector.StandardConnectorManager) {
	// 创建连接器注册中心
	registry := connector.NewStandardConnectorRegistry()
	fmt.Println("连接器注册中心已创建")

	// 注册内置连接器
	if err := connector.RegisterBuiltinConnectors(registry); err != nil {
		log.Fatalf("注册内置连接器失败: %v", err)
	}
	fmt.Println("内置连接器注册完成")

	// 显示已注册的连接器
	displayRegisteredConnectors(registry)

	// 创建连接器管理器
	config := connector.DefaultConnectorConfig()
	manager := connector.NewStandardConnectorManager(registry, config)
	fmt.Printf("\n连接器管理器已创建，配置: 最大重试=%d, 重试间隔=%v\n",
		config.MaxRetries, config.RetryInterval)

	return registry, manager
}

// displayRegisteredConnectors 显示已注册的连接器
func displayRegisteredConnectors(registry *connector.StandardConnectorRegistry) {
	fmt.Println("\n=== 已注册的连接器 ===")
	for _, connectorType := range []connector.ConnectorType{
		connector.ConnectorTypeSource,
		connector.ConnectorTypeSink,
		connector.ConnectorTypeTransform,
	} {
		connectors := registry.List(connectorType)
		fmt.Printf("%s 连接器 (%d 个): %v\n", connectorType, len(connectors), connectors)

		// 显示每个连接器的详细信息
		for _, name := range connectors {
			if info, err := registry.GetInfo(connectorType, name); err == nil {
				fmt.Printf("  - %s: %s (v%s) - %s\n", info.Name, info.ID, info.Version, info.Description)
			}
		}
	}
}

// createSampleData 创建示例数据文件
func createSampleData() string {
	dataDir := "data"
	dataFile := filepath.Join(dataDir, "sample.jsonl")
	if err := connector.CreateSampleDataFile(dataFile); err != nil {
		log.Fatalf("创建示例数据文件失败: %v", err)
	}
	fmt.Printf("示例数据文件已创建: %s\n", dataFile)
	return dataFile
}

// createConnectors 创建所有连接器
func createConnectors(ctx context.Context, manager *connector.StandardConnectorManager, dataFile string) []string {
	// 创建文件源连接器
	sourceConfig := map[string]interface{}{
		"file_path": dataFile,
	}
	if err := manager.CreateConnector(ctx, "file-source-1", connector.ConnectorTypeSource, "file", sourceConfig); err != nil {
		log.Fatalf("创建文件源连接器失败: %v", err)
	}
	fmt.Println("文件源连接器已创建: file-source-1")

	// 创建JSON转换连接器
	transformConfig := map[string]interface{}{
		"enabled": true,
	}
	if err := manager.CreateConnector(ctx, "json-transform-1", connector.ConnectorTypeTransform, "json", transformConfig); err != nil {
		log.Fatalf("创建JSON转换连接器失败: %v", err)
	}
	fmt.Println("JSON转换连接器已创建: json-transform-1")

	// 创建控制台接收器连接器
	sinkConfig := map[string]interface{}{
		"format": "json",
	}
	if err := manager.CreateConnector(ctx, "console-sink-1", connector.ConnectorTypeSink, "console", sinkConfig); err != nil {
		log.Fatalf("创建控制台接收器连接器失败: %v", err)
	}
	fmt.Println("控制台接收器连接器已创建: console-sink-1")

	return manager.ListConnectors()
}

// startConnectors 启动所有连接器
func startConnectors(ctx context.Context, manager *connector.StandardConnectorManager, connectorIDs []string) {
	// 显示所有连接器状态
	fmt.Println("\n=== 连接器状态 ===")
	for _, id := range connectorIDs {
		status, _ := manager.GetConnectorStatus(id)
		fmt.Printf("%s: %s\n", id, status)
	}

	// 启动所有连接器
	fmt.Println("\n=== 启动连接器 ===")
	for _, id := range connectorIDs {
		if err := manager.StartConnector(ctx, id); err != nil {
			log.Printf("启动连接器 %s 失败: %v", id, err)
		} else {
			fmt.Printf("连接器 %s 已启动\n", id)
		}
	}

}

// runDataPipeline 运行数据处理管道
func runDataPipeline(ctx context.Context, manager *connector.StandardConnectorManager) {
	// 启动事件监听
	startEventListening(manager)

	// 创建和启动数据处理管道
	startDataProcessingPipeline(manager)

	// 启动指标监控
	startMetricsMonitoring(manager)

	// 等待处理完成
	time.Sleep(15 * time.Second)
}

// startEventListening 启动事件监听
func startEventListening(manager *connector.StandardConnectorManager) {
	fmt.Println("\n=== 连接器事件监听 ===")
	eventCtx, eventCancel := context.WithTimeout(context.Background(), 30*time.Second)

	go func() {
		defer eventCancel()
		eventChan := manager.GetEventChannel()
		for {
			select {
			case <-eventCtx.Done():
				return
			case event, ok := <-eventChan:
				if !ok {
					return
				}
				fmt.Printf("[事件] %s - %s: %s\n",
					event.Timestamp.Format("15:04:05"),
					event.Type,
					event.Message)
			}
		}
	}()
}

// startDataProcessingPipeline 创建和启动数据处理管道
func startDataProcessingPipeline(manager *connector.StandardConnectorManager) {

	// 创建数据处理管道
	fmt.Println("\n=== 数据处理管道 ===")

	// 获取连接器实例
	source, transform, sink := getConnectorInstances(manager)

	// 启动数据处理
	startDataProcessing(source, transform, sink)

	fmt.Println("数据处理管道已启动，正在处理数据...")
}

// getConnectorInstances 获取连接器实例
func getConnectorInstances(manager *connector.StandardConnectorManager) (connector.SourceConnector, connector.TransformConnector, connector.SinkConnector) {
	// 获取连接器实例
	sourceConnector, _ := manager.GetConnector("file-source-1")
	transformConnector, _ := manager.GetConnector("json-transform-1")
	sinkConnector, _ := manager.GetConnector("console-sink-1")

	// 类型断言
	source, ok := sourceConnector.(connector.SourceConnector)
	if !ok {
		log.Fatal("源连接器类型断言失败")
	}

	transform, ok := transformConnector.(connector.TransformConnector)
	if !ok {
		log.Fatal("转换连接器类型断言失败")
	}

	sink, ok := sinkConnector.(connector.SinkConnector)
	if !ok {
		log.Fatal("接收器连接器类型断言失败")
	}

	return source, transform, sink
}

// startDataProcessing 启动数据处理
func startDataProcessing(source connector.SourceConnector, transform connector.TransformConnector, sink connector.SinkConnector) {
	// 创建处理上下文
	processCtx, processCancel := context.WithTimeout(context.Background(), 20*time.Second)

	// 启动数据处理管道
	go func() {
		defer processCancel()
		// 从源读取数据
		msgChan, errChan := source.Read(processCtx)

		// 转换数据
		transformedChan, transformErrChan := transform.Transform(processCtx, msgChan)

		// 写入接收器
		go func() {
			if err := sink.Write(processCtx, transformedChan); err != nil {
				log.Printf("写入接收器失败: %v", err)
			}
		}()

		// 处理错误
		handleProcessingErrors(processCtx, errChan, transformErrChan)
	}()
}

// handleProcessingErrors 处理处理过程中的错误
func handleProcessingErrors(ctx context.Context, errChan, transformErrChan <-chan error) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case err, ok := <-errChan:
				if !ok {
					return
				}
				log.Printf("源连接器错误: %v", err)
			case err, ok := <-transformErrChan:
				if !ok {
					return
				}
				log.Printf("转换连接器错误: %v", err)
			}
		}
	}()
}

// startMetricsMonitoring 启动指标监控
func startMetricsMonitoring(manager *connector.StandardConnectorManager) {

	// 定期显示连接器指标
	metricsCtx, metricsCancel := context.WithTimeout(context.Background(), 25*time.Second)

	go func() {
		defer metricsCancel()
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-metricsCtx.Done():
				return
			case <-ticker.C:
				displayConnectorMetrics(manager)
			}
		}
	}()
}

// displayConnectorMetrics 显示连接器指标
func displayConnectorMetrics(manager *connector.StandardConnectorManager) {
	fmt.Println("\n=== 连接器指标 ===")
	for _, id := range manager.ListConnectors() {
		if metrics, err := manager.GetConnectorMetrics(id); err == nil {
			fmt.Printf("%s:\n", id)
			fmt.Printf("  处理消息数: %d\n", metrics.MessagesProcessed)
			fmt.Printf("  处理字节数: %d\n", metrics.BytesProcessed)
			fmt.Printf("  错误数: %d\n", metrics.ErrorsCount)
			fmt.Printf("  运行时间: %v\n", metrics.Uptime)
			if !metrics.LastActivity.IsZero() {
				fmt.Printf("  最后活动: %v\n", metrics.LastActivity.Format("15:04:05"))
			}
		}
	}
}

// runMonitoringAndManagement 运行监控和管理操作
func runMonitoringAndManagement(ctx context.Context, manager *connector.StandardConnectorManager, registry *connector.StandardConnectorRegistry) {
	// 测试连接器管理操作
	fmt.Println("\n=== 连接器管理操作测试 ===")

	// 停止一个连接器
	if err := manager.StopConnector(ctx, "json-transform-1"); err != nil {
		log.Printf("停止连接器失败: %v", err)
	} else {
		fmt.Println("JSON转换连接器已停止")
	}

	// 重启连接器
	time.Sleep(2 * time.Second)
	if err := manager.RestartConnector(ctx, "json-transform-1"); err != nil {
		log.Printf("重启连接器失败: %v", err)
	} else {
		fmt.Println("JSON转换连接器已重启")
	}

	// 更新连接器配置
	newConfig := map[string]interface{}{
		"enabled":    true,
		"debug_mode": true,
	}
	if err := manager.UpdateConnectorConfig(ctx, "json-transform-1", newConfig); err != nil {
		log.Printf("更新连接器配置失败: %v", err)
	} else {
		fmt.Println("JSON转换连接器配置已更新")
	}

	// 显示最终状态
	fmt.Println("\n=== 最终状态 ===")
	for _, id := range manager.ListConnectors() {
		status, _ := manager.GetConnectorStatus(id)
		metrics, _ := manager.GetConnectorMetrics(id)
		fmt.Printf("%s: %s (消息: %d, 错误: %d)\n",
			id, status, metrics.MessagesProcessed, metrics.ErrorsCount)
	}

	// 显示注册中心统计
	fmt.Println("\n=== 注册中心统计 ===")
	counts := registry.GetConnectorCount()
	for connectorType, count := range counts {
		fmt.Printf("%s: %d 个\n", connectorType, count)
	}

	// 显示所有连接器详细信息
	fmt.Println("\n=== 所有连接器详细信息 ===")
	allConnectors := registry.GetAllConnectors()
	for connectorType, infos := range allConnectors {
		fmt.Printf("\n%s 连接器:\n", connectorType)
		for _, info := range infos {
			fmt.Printf("  ID: %s\n", info.ID)
			fmt.Printf("  名称: %s\n", info.Name)
			fmt.Printf("  版本: %s\n", info.Version)
			fmt.Printf("  描述: %s\n", info.Description)
			fmt.Printf("  作者: %s\n", info.Author)
			fmt.Printf("  创建时间: %v\n", info.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Println()
		}
	}
}

// cleanupResources 清理资源
func cleanupResources(ctx context.Context, manager *connector.StandardConnectorManager) {
	// 停止所有连接器
	fmt.Println("=== 停止所有连接器 ===")
	for _, id := range manager.ListConnectors() {
		if err := manager.StopConnector(ctx, id); err != nil {
			log.Printf("停止连接器 %s 失败: %v", id, err)
		} else {
			fmt.Printf("连接器 %s 已停止\n", id)
		}
	}

	// 清理数据文件
	dataDir := "data"
	if err := os.RemoveAll(dataDir); err != nil {
		log.Printf("清理数据目录失败: %v", err)
	} else {
		fmt.Printf("数据目录 %s 已清理\n", dataDir)
	}
}
