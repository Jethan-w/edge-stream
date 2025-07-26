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
	"fmt"
	"log"
	"strings"

	"github.com/crazy/edge-stream/internal/flowfile"
	"github.com/crazy/edge-stream/internal/processor"
	"github.com/crazy/edge-stream/internal/sink"
	"github.com/crazy/edge-stream/internal/source"
)

func main() {
	fmt.Println("=== Edge Stream 基础框架测试 ===")

	// 1. 创建数据源
	testData := []string{
		"Hello World",
		"Edge Stream Framework",
		"Simple Data Processing",
		"Go Programming",
	}

	dataSource := source.NewStringSource("test-source", testData)
	fmt.Printf("创建数据源: %s\n", dataSource.GetName())

	// 2. 创建处理器 - 转换为大写
	upperCaseProcessor := processor.NewTransformProcessor("uppercase-processor", func(ff *flowfile.FlowFile) (*flowfile.FlowFile, error) {
		// 转换内容为大写
		content := string(ff.Content)
		upperContent := strings.ToUpper(content)

		// 创建新的 FlowFile
		newFlowFile := flowfile.NewFlowFile()
		newFlowFile.Content = []byte(upperContent)
		newFlowFile.Size = int64(len(newFlowFile.Content))
		newFlowFile.Attributes["original_content"] = content
		newFlowFile.Attributes["processor"] = "uppercase-processor"

		return newFlowFile, nil
	})
	fmt.Printf("创建处理器: %s\n", upperCaseProcessor.GetName())

	// 3. 创建输出器
	consoleSink := sink.NewConsoleSink("console-sink")
	memorySink := sink.NewSimpleSink("memory-sink")
	fmt.Printf("创建输出器: %s, %s\n", consoleSink.GetName(), memorySink.GetName())

	// 4. 处理数据流
	fmt.Println("\n开始处理数据流...")
	processedCount := 0

	for dataSource.HasNext() {
		// 从数据源读取数据
		flowFile, err := dataSource.Read()
		if err != nil {
			log.Printf("读取数据失败: %v", err)
			continue
		}
		if flowFile == nil {
			break
		}

		fmt.Printf("\n--- 处理第 %d 个数据 ---\n", processedCount+1)
		fmt.Printf("原始数据: %s\n", string(flowFile.Content))

		// 通过处理器处理数据
		processedFlowFile, err := upperCaseProcessor.Process(flowFile)
		if err != nil {
			log.Printf("处理数据失败: %v", err)
			continue
		}

		fmt.Printf("处理后数据: %s\n", string(processedFlowFile.Content))

		// 输出到控制台
		if err := consoleSink.Write(processedFlowFile); err != nil {
			log.Printf("控制台输出失败: %v", err)
		}

		// 输出到内存
		if err := memorySink.Write(processedFlowFile); err != nil {
			log.Printf("内存输出失败: %v", err)
		}

		processedCount++
	}

	// 5. 显示处理结果
	fmt.Printf("\n=== 处理完成 ===\n")
	fmt.Printf("总共处理了 %d 个数据\n", processedCount)
	fmt.Printf("内存中存储了 %d 个数据\n", len(memorySink.GetData()))

	// 6. 显示内存中的数据
	fmt.Println("\n内存中的数据:")
	for i, ff := range memorySink.GetData() {
		fmt.Printf("  %d. %s (原始: %s)\n", i+1, string(ff.Content), ff.Attributes["original_content"])
	}

	fmt.Println("\n=== 测试完成 ===")
}
