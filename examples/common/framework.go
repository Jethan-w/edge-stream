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

package common

import (
	"fmt"
	"log"
)

// ExampleConfig 示例配置
type ExampleConfig struct {
	Name        string
	Description string
	SetupFunc   func() (interface{}, error)
	RunFunc     func(interface{}) error
	CleanupFunc func(interface{}) error
}

// RunExample 运行示例的通用框架
func RunExample(config ExampleConfig) {
	fmt.Printf("=== %s ===\n", config.Name)
	if config.Description != "" {
		fmt.Printf("描述: %s\n", config.Description)
	}
	fmt.Println()

	// 设置阶段
	fmt.Println("🔧 初始化组件...")
	component, err := config.SetupFunc()
	if err != nil {
		log.Fatalf("❌ 设置失败: %v", err)
	}
	fmt.Println("✅ 组件初始化完成")
	fmt.Println()

	// 运行阶段
	fmt.Println("🚀 运行示例...")
	if err := config.RunFunc(component); err != nil {
		log.Fatalf("❌ 运行失败: %v", err)
	}
	fmt.Println("✅ 示例运行完成")
	fmt.Println()

	// 清理阶段
	if config.CleanupFunc != nil {
		fmt.Println("🧹 清理资源...")
		if err := config.CleanupFunc(component); err != nil {
			log.Printf("⚠️ 清理警告: %v", err)
		} else {
			fmt.Println("✅ 资源清理完成")
		}
		fmt.Println()
	}

	fmt.Printf("=== %s 完成 ===\n", config.Name)
}

// SimpleExample 简单示例配置，用于不需要清理的场景
type SimpleExample struct {
	Name        string
	Description string
	RunFunc     func() error
}

// Run 运行简单示例
func (s SimpleExample) Run() {
	RunExample(ExampleConfig{
		Name:        s.Name,
		Description: s.Description,
		SetupFunc: func() (interface{}, error) {
			return nil, nil
		},
		RunFunc: func(interface{}) error {
			return s.RunFunc()
		},
	})
}

// ComponentExample 组件示例配置，用于需要组件管理的场景
type ComponentExample[T any] struct {
	Name        string
	Description string
	CreateFunc  func() (T, error)
	ConfigFunc  func(T) error
	RunFunc     func(T) error
	CleanupFunc func(T) error
}

// Run 运行组件示例
func (c ComponentExample[T]) Run() {
	RunExample(ExampleConfig{
		Name:        c.Name,
		Description: c.Description,
		SetupFunc: func() (interface{}, error) {
			component, err := c.CreateFunc()
			if err != nil {
				return nil, err
			}
			if c.ConfigFunc != nil {
				if err := c.ConfigFunc(component); err != nil {
					return nil, err
				}
			}
			return component, nil
		},
		RunFunc: func(component interface{}) error {
			return c.RunFunc(component.(T))
		},
		CleanupFunc: func(component interface{}) error {
			if c.CleanupFunc != nil {
				return c.CleanupFunc(component.(T))
			}
			return nil
		},
	})
}

// PrintSection 打印章节标题
func PrintSection(title string) {
	fmt.Printf("\n--- %s ---\n", title)
}

// PrintResult 打印结果
func PrintResult(label string, value interface{}) {
	fmt.Printf("📊 %s: %v\n", label, value)
}

// PrintError 打印错误
func PrintError(err error) {
	fmt.Printf("❌ 错误: %v\n", err)
}

// PrintSuccess 打印成功信息
func PrintSuccess(message string) {
	fmt.Printf("✅ %s\n", message)
}

// PrintInfo 打印信息
func PrintInfo(message string) {
	fmt.Printf("ℹ️ %s\n", message)
}
