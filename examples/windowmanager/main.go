package main

import (
	"fmt"
	"time"

	"github.com/crazy/edge-stream/internal/windowmanager"
)

func main() {
	fmt.Println("=== 窗口管理器基础框架测试 ===")

	// 创建窗口管理器
	manager := windowmanager.NewSimpleWindowManager()

	// 创建时间窗口（5秒）
	timeWindow := manager.CreateTimeWindow(5 * time.Second)
	fmt.Printf("创建时间窗口: %s\n", timeWindow.GetID())

	// 创建计数窗口（3个数据）
	countWindow := manager.CreateCountWindow(3)
	fmt.Printf("创建计数窗口: %s\n", countWindow.GetID())

	// 添加一些测试数据
	fmt.Println("\n开始添加数据...")
	for i := 1; i <= 5; i++ {
		data := fmt.Sprintf("数据-%d", i)
		fmt.Printf("添加数据: %s\n", data)
		
		// 处理数据
		if err := manager.ProcessData(data); err != nil {
			fmt.Printf("处理数据失败: %v\n", err)
			continue
		}
		
		// 打印窗口状态
		manager.PrintWindowStatus()
		
		// 检查准备好的窗口
		readyWindows := manager.GetReadyWindows()
		if len(readyWindows) > 0 {
			fmt.Printf("发现 %d 个准备好的窗口:\n", len(readyWindows))
			for _, window := range readyWindows {
				data := window.GetData()
				fmt.Printf("  窗口 %s: %v\n", window.GetID(), data)
				// 重置窗口
				window.Reset()
				fmt.Printf("  窗口 %s 已重置\n", window.GetID())
			}
		}
		
		fmt.Println("---")
		time.Sleep(1 * time.Second)
	}

	// 等待时间窗口准备好
	fmt.Println("\n等待时间窗口准备好...")
	for {
		readyWindows := manager.GetReadyWindows()
		if len(readyWindows) > 0 {
			fmt.Printf("时间窗口准备好了！\n")
			for _, window := range readyWindows {
				if window.GetWindowType() == windowmanager.TIME_WINDOW {
					data := window.GetData()
					fmt.Printf("时间窗口 %s 的数据: %v\n", window.GetID(), data)
				}
			}
			break
		}
		time.Sleep(1 * time.Second)
	}

	fmt.Println("\n=== 测试完成 ===")
}