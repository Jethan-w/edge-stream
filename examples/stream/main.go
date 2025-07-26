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
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/crazy/edge-stream/internal/stream"
)

func main() {
	fmt.Println("=== Edge Stream 流处理引擎示例 ===")

	// 初始化流处理引擎
	engine := initializeStreamEngine()
	defer engine.Close()

	// 创建示例数据文件
	absDataFile := createStreamSampleData()

	// 创建和配置拓扑
	createTopology(engine, absDataFile)

	// 显示拓扑信息
	displayTopologyInfo(engine)

	// 启动流处理
	ctx := startStreamProcessing(engine)

	// 运行监控和指标显示
	runStreamMonitoring(ctx, engine)

	// 等待中断信号并停止
	stopStreamProcessing(ctx, engine)

	// 清理资源
	cleanupStreamResources(engine)

	fmt.Println("\n=== 流处理引擎示例完成 ===")
}

// initializeStreamEngine 初始化流处理引擎
func initializeStreamEngine() *stream.StandardStreamEngine {
	fmt.Println("正在创建流处理引擎...")
	engine := stream.NewStandardStreamEngine()
	fmt.Println("流处理引擎创建成功")
	return engine
}

// createStreamSampleData 创建示例数据文件
func createStreamSampleData() string {
	fmt.Println("正在创建示例数据文件...")
	dataDir := "data"
	dataFile := filepath.Join(dataDir, "stream_data.txt")
	fmt.Printf("数据文件路径: %s\n", dataFile)
	if err := stream.CreateSampleStreamDataFile(dataFile); err != nil {
		log.Fatalf("创建示例数据文件失败: %v", err)
	}
	fmt.Println("示例数据文件创建成功")

	// 获取绝对路径
	absDataFile, err := filepath.Abs(dataFile)
	if err != nil {
		log.Fatalf("获取绝对路径失败: %v", err)
	}
	fmt.Printf("示例数据文件已创建: %s\n", absDataFile)
	return absDataFile
}

// createTopology 创建和配置拓扑
func createTopology(engine *stream.StandardStreamEngine, absDataFile string) {
	// 创建流拓扑
	topology, err := engine.CreateTopology("demo-topology", "演示拓扑", "展示流处理引擎功能的示例拓扑")
	if err != nil {
		log.Fatalf("创建拓扑失败: %v", err)
	}
	fmt.Printf("流拓扑已创建: %s\n", topology.Name)

	// 创建处理器
	createProcessors(engine, absDataFile)

	// 连接处理器
	connectProcessors(engine)


}

// createProcessors 创建所有处理器
func createProcessors(engine *stream.StandardStreamEngine, absDataFile string) {
	fmt.Println("\n=== 创建处理器 ===")

	// 文件源处理器
	fileSource := stream.NewFileSourceProcessor("file-source", "文件数据源", absDataFile)
	if err := engine.AddProcessor("demo-topology", fileSource); err != nil {
		log.Fatalf("添加文件源处理器失败: %v", err)
	}
	fmt.Println("文件源处理器已添加")

	// JSON转换处理器
	jsonTransform := stream.NewJSONTransformProcessor("json-transform", "JSON转换器")
	if err := engine.AddProcessor("demo-topology", jsonTransform); err != nil {
		log.Fatalf("添加JSON转换处理器失败: %v", err)
	}
	fmt.Println("JSON转换处理器已添加")

	// 聚合处理器（计算平均分数）
	windowConfig := &stream.WindowConfig{
		Type:      stream.WindowTypeTumbling,
		Size:      5 * time.Second,
		MaxSize:   100,
		Watermark: 2 * time.Second,
	}
	aggProcessor := stream.NewAggregationProcessor("score-aggregator", "分数聚合器", windowConfig, stream.AggregationTypeAvg, "score")
	if err := engine.AddProcessor("demo-topology", aggProcessor); err != nil {
		log.Fatalf("添加聚合处理器失败: %v", err)
	}
	fmt.Println("聚合处理器已添加")

	// 控制台输出处理器（用于原始数据）
	consoleSink1 := stream.NewConsoleSinkProcessor("console-sink-1", "控制台输出1")
	if err := engine.AddProcessor("demo-topology", consoleSink1); err != nil {
		log.Fatalf("添加控制台输出处理器1失败: %v", err)
	}
	fmt.Println("控制台输出处理器1已添加")

	// 控制台输出处理器（用于聚合结果）
	consoleSink2 := stream.NewConsoleSinkProcessor("console-sink-2", "控制台输出2")
	if err := engine.AddProcessor("demo-topology", consoleSink2); err != nil {
		log.Fatalf("添加控制台输出处理器2失败: %v", err)
	}
	fmt.Println("控制台输出处理器2已添加")
}

// connectProcessors 连接处理器
func connectProcessors(engine *stream.StandardStreamEngine) {
	fmt.Println("\n=== 连接处理器 ===")

	// 文件源 -> JSON转换器
	if err := engine.ConnectProcessors("demo-topology", "file-source", "json-transform", 10); err != nil {
		log.Fatalf("连接文件源和JSON转换器失败: %v", err)
	}
	fmt.Println("文件源 -> JSON转换器 连接已建立")

	// JSON转换器 -> 控制台输出1（显示转换后的数据）
	if err := engine.ConnectProcessors("demo-topology", "json-transform", "console-sink-1", 10); err != nil {
		log.Fatalf("连接JSON转换器和控制台输出1失败: %v", err)
	}
	fmt.Println("JSON转换器 -> 控制台输出1 连接已建立")

	// JSON转换器 -> 聚合处理器
	if err := engine.ConnectProcessors("demo-topology", "json-transform", "score-aggregator", 10); err != nil {
		log.Fatalf("连接JSON转换器和聚合处理器失败: %v", err)
	}
	fmt.Println("JSON转换器 -> 聚合处理器 连接已建立")

	// 聚合处理器 -> 控制台输出2（显示聚合结果）
	if err := engine.ConnectProcessors("demo-topology", "score-aggregator", "console-sink-2", 10); err != nil {
		log.Fatalf("连接聚合处理器和控制台输出2失败: %v", err)
	}
	fmt.Println("聚合处理器 -> 控制台输出2 连接已建立")

}

// displayTopologyInfo 显示拓扑信息
func displayTopologyInfo(engine *stream.StandardStreamEngine) {
	fmt.Println("\n=== 拓扑信息 ===")
	topologies, _ := engine.ListTopologies()
	for _, topo := range topologies {
		fmt.Printf("拓扑: %s (%s)\n", topo.Name, topo.ID)
		fmt.Printf("  描述: %s\n", topo.Description)
		fmt.Printf("  处理器数量: %d\n", len(topo.Processors))
		fmt.Printf("  连接数量: %d\n", len(topo.Connections))
		fmt.Printf("  创建时间: %s\n", topo.CreatedAt.Format("2006-01-02 15:04:05"))

		fmt.Println("  处理器列表:")
		for id, processor := range topo.Processors {
			fmt.Printf("    - %s: %s (%s)\n", id, processor.GetName(), processor.GetType())
		}

		fmt.Println("  连接列表:")
		for _, conn := range topo.Connections {
			fmt.Printf("    - %s -> %s (缓冲区大小: %d)\n", conn.From, conn.To, conn.BufferSize)
		}
	}
}

// startStreamProcessing 启动流处理
func startStreamProcessing(engine *stream.StandardStreamEngine) context.Context {
	// 启动事件监听
	fmt.Println("\n=== 启动事件监听 ===")
	ctx, _ := context.WithCancel(context.Background())

	go func() {
		eventChan := engine.GetEventChannel()
		for event := range eventChan {
			timestamp := event.Timestamp.Format("15:04:05")
			fmt.Printf("[事件] %s - %s: %s\n", timestamp, event.Type, event.Message)
		}
	}()

	// 启动拓扑
	fmt.Println("\n=== 启动流处理拓扑 ===")
	if err := engine.StartTopology(ctx, "demo-topology"); err != nil {
		log.Fatalf("启动拓扑失败: %v", err)
	}
	fmt.Println("流处理拓扑已启动")

	// 显示拓扑状态
	status, _ := engine.GetTopologyStatus("demo-topology")
	fmt.Printf("拓扑状态: %s\n", status)

	return ctx
}

// runStreamMonitoring 运行监控和指标显示
func runStreamMonitoring(ctx context.Context, engine *stream.StandardStreamEngine) {
	// 定期显示指标
	fmt.Println("\n=== 流处理进行中 ===")
	fmt.Println("正在处理数据流，按 Ctrl+C 停止...")

	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				fmt.Println("\n--- 流处理指标 ---")

				// 显示拓扑指标
				topologyMetrics, err := engine.GetTopologyMetrics("demo-topology")
				if err == nil {
					fmt.Printf("拓扑 %s:\n", topologyMetrics.TopologyID)
					fmt.Printf("  状态: %s\n", topologyMetrics.Status)
					fmt.Printf("  运行时间: %v\n", topologyMetrics.Uptime)
					fmt.Printf("  总消息数: %d\n", topologyMetrics.TotalMessages)
					fmt.Printf("  总字节数: %d\n", topologyMetrics.TotalBytes)
					fmt.Printf("  总错误数: %d\n", topologyMetrics.TotalErrors)

					// 显示处理器指标
					for processorID, metrics := range topologyMetrics.ProcessorMetrics {
						fmt.Printf("  处理器 %s:\n", processorID)
						fmt.Printf("    输入消息: %d, 输出消息: %d\n", metrics.MessagesIn, metrics.MessagesOut)
						fmt.Printf("    输入字节: %d, 输出字节: %d\n", metrics.BytesIn, metrics.BytesOut)
						fmt.Printf("    错误数: %d\n", metrics.ErrorCount)
						fmt.Printf("    吞吐量: %.2f 消息/秒\n", metrics.Throughput)
						fmt.Printf("    平均延迟: %v\n", metrics.Latency)
						fmt.Printf("    最后活动: %s\n", metrics.LastActivity.Format("15:04:05"))
					}
				}

				// 显示引擎指标
				engineMetrics, err := engine.GetEngineMetrics()
				if err == nil {
					fmt.Printf("\n引擎总体指标:\n")
					fmt.Printf("  拓扑总数: %d (运行中: %d, 已停止: %d)\n",
						engineMetrics.TopologyCount, engineMetrics.RunningCount, engineMetrics.StoppedCount)
					fmt.Printf("  引擎运行时间: %v\n", engineMetrics.Uptime)
					fmt.Printf("  总处理消息: %d\n", engineMetrics.TotalMessages)
					fmt.Printf("  总处理字节: %d\n", engineMetrics.TotalBytes)
					fmt.Printf("  总错误数: %d\n", engineMetrics.TotalErrors)
				}
			}
		}
	}()
}

// stopStreamProcessing 等待中断信号并停止流处理
func stopStreamProcessing(ctx context.Context, engine *stream.StandardStreamEngine) {
	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n\n=== 停止流处理引擎 ===")

	// 停止拓扑
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer stopCancel()

	if err := engine.StopTopology(stopCtx, "demo-topology"); err != nil {
		log.Printf("停止拓扑失败: %v", err)
	} else {
		fmt.Println("流处理拓扑已停止")
	}

}

// cleanupStreamResources 清理资源
func cleanupStreamResources(engine *stream.StandardStreamEngine) {
	// 显示最终指标
	fmt.Println("\n=== 最终指标 ===")
	topologyMetrics, err := engine.GetTopologyMetrics("demo-topology")
	if err == nil {
		fmt.Printf("拓扑 %s 最终统计:\n", topologyMetrics.TopologyID)
		fmt.Printf("  总运行时间: %v\n", topologyMetrics.Uptime)
		fmt.Printf("  总处理消息: %d\n", topologyMetrics.TotalMessages)
		fmt.Printf("  总处理字节: %d\n", topologyMetrics.TotalBytes)
		fmt.Printf("  总错误数: %d\n", topologyMetrics.TotalErrors)

		fmt.Println("\n各处理器最终指标:")
		for processorID, metrics := range topologyMetrics.ProcessorMetrics {
			fmt.Printf("  %s:\n", processorID)
			fmt.Printf("    处理消息: 输入 %d, 输出 %d\n", metrics.MessagesIn, metrics.MessagesOut)
			fmt.Printf("    处理字节: 输入 %d, 输出 %d\n", metrics.BytesIn, metrics.BytesOut)
			fmt.Printf("    错误数: %d\n", metrics.ErrorCount)
			fmt.Printf("    平均吞吐量: %.2f 消息/秒\n", metrics.Throughput)
			fmt.Printf("    平均延迟: %v\n", metrics.Latency)
			fmt.Printf("    总处理时间: %v\n", metrics.ProcessingTime)
		}
	}

	// 删除拓扑
	if err := engine.DeleteTopology("demo-topology"); err != nil {
		log.Printf("删除拓扑失败: %v", err)
	} else {
		fmt.Println("\n拓扑已删除")
	}

	// 清理数据文件
	dataDir := "data"
	if err := os.RemoveAll(dataDir); err != nil {
		log.Printf("清理数据目录失败: %v", err)
	} else {
		fmt.Printf("数据目录 %s 已清理\n", dataDir)
	}
}
