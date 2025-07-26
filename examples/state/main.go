package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/crazy/edge-stream/internal/state"
)

func main() {
	fmt.Println("=== Edge Stream 状态管理系统示例 ===")

	// 1. 创建状态管理器
	config := state.DefaultStateConfig()
	config.CheckpointInterval = 10 * time.Second // 10秒创建一次检查点
	config.MaxCheckpoints = 5

	stateManager := state.NewStandardStateManager(config)
	defer stateManager.Close()

	fmt.Printf("状态管理器已创建，检查点间隔: %v\n", config.CheckpointInterval)

	// 2. 创建多个状态
	userState, err := stateManager.CreateState("user_sessions", state.StateTypeMemory)
	if err != nil {
		log.Fatalf("创建用户会话状态失败: %v", err)
	}

	cacheState, err := stateManager.CreateState("cache_data", state.StateTypeMemory)
	if err != nil {
		log.Fatalf("创建缓存状态失败: %v", err)
	}

	metricsState, err := stateManager.CreateState("metrics", state.StateTypeMemory)
	if err != nil {
		log.Fatalf("创建指标状态失败: %v", err)
	}

	fmt.Printf("已创建 %d 个状态: %v\n", len(stateManager.ListStates()), stateManager.ListStates())

	// 3. 设置状态变更监听
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	stateManager.Watch(ctx, func(stateName, key string, oldValue, newValue interface{}) {
		fmt.Printf("[状态变更] %s.%s: %v -> %v\n", stateName, key, oldValue, newValue)
	})

	// 4. 模拟用户会话数据
	fmt.Println("\n=== 模拟用户会话数据 ===")
	for i := 0; i < 5; i++ {
		userID := fmt.Sprintf("user_%d", i+1)
		sessionData := map[string]interface{}{
			"login_time": time.Now(),
			"ip_address": fmt.Sprintf("192.168.1.%d", rand.Intn(255)),
			"user_agent": "EdgeStream-Client/1.0",
		}
		userState.Set(userID, sessionData)
		time.Sleep(200 * time.Millisecond)
	}

	// 5. 模拟缓存数据
	fmt.Println("\n=== 模拟缓存数据 ===")
	cacheKeys := []string{"config", "templates", "user_preferences", "api_tokens"}
	for _, key := range cacheKeys {
		cacheValue := fmt.Sprintf("cached_value_%s_%d", key, time.Now().Unix())
		cacheState.Set(key, cacheValue)
		time.Sleep(100 * time.Millisecond)
	}

	// 6. 模拟指标数据
	fmt.Println("\n=== 模拟指标数据 ===")
	metricsData := map[string]interface{}{
		"requests_total":     rand.Intn(10000),
		"errors_total":       rand.Intn(100),
		"response_time_avg":  rand.Float64() * 1000,
		"active_connections": rand.Intn(500),
		"memory_usage":       rand.Float64() * 100,
	}

	for key, value := range metricsData {
		metricsState.Set(key, value)
		time.Sleep(50 * time.Millisecond)
	}

	// 7. 显示状态信息
	fmt.Println("\n=== 当前状态信息 ===")
	for _, stateName := range stateManager.ListStates() {
		if state, exists := stateManager.GetState(stateName); exists {
			fmt.Printf("%s: %d 个键\n", stateName, state.Size())
			for _, key := range state.Keys() {
				if value, exists := state.Get(key); exists {
					fmt.Printf("  %s: %v\n", key, value)
				}
			}
		}
	}

	// 8. 手动创建检查点
	fmt.Println("\n=== 创建检查点 ===")
	checkpoint, err := stateManager.CreateCheckpoint(ctx)
	if err != nil {
		log.Printf("创建检查点失败: %v", err)
	} else {
		fmt.Printf("检查点已创建: %s (时间: %v)\n", checkpoint.ID, checkpoint.Timestamp)
		fmt.Printf("检查点包含 %d 个状态\n", len(checkpoint.States))
	}

	// 9. 列出所有检查点
	fmt.Println("\n=== 检查点列表 ===")
	checkpointManager := stateManager.GetCheckpointManager()
	checkpoints, err := checkpointManager.List(ctx)
	if err != nil {
		log.Printf("获取检查点列表失败: %v", err)
	} else {
		fmt.Printf("共有 %d 个检查点:\n", len(checkpoints))
		for i, cp := range checkpoints {
			fmt.Printf("  %d. ID: %s, 时间: %v, 大小: %d bytes, 状态数: %d\n",
				i+1, cp.ID, cp.Timestamp.Format("2006-01-02 15:04:05"), cp.Size, cp.States)
		}
	}

	// 10. 模拟状态变更
	fmt.Println("\n=== 模拟状态变更 ===")
	for i := 0; i < 3; i++ {
		// 更新用户会话
		userState.Set("user_1", map[string]interface{}{
			"login_time":    time.Now(),
			"last_activity": time.Now(),
			"page_views":    rand.Intn(50),
		})

		// 更新指标
		metricsState.Set("requests_total", rand.Intn(15000))
		metricsState.Set("active_connections", rand.Intn(800))

		time.Sleep(2 * time.Second)
	}

	// 11. 测试状态删除
	fmt.Println("\n=== 测试状态删除 ===")
	if err := userState.Delete("user_5"); err != nil {
		log.Printf("删除用户状态失败: %v", err)
	} else {
		fmt.Println("已删除 user_5 的会话数据")
	}

	// 12. 显示状态管理器指标
	fmt.Println("\n=== 状态管理器指标 ===")
	managerMetrics := stateManager.GetMetrics()
	fmt.Printf("状态数量: %d\n", managerMetrics.StatesCount)
	fmt.Printf("检查点数量: %d\n", managerMetrics.CheckpointsCount)
	fmt.Printf("总操作数: %d\n", managerMetrics.TotalOperations)
	fmt.Printf("错误数: %d\n", managerMetrics.ErrorsCount)
	fmt.Printf("运行时间: %v\n", time.Since(managerMetrics.StartTime))
	if !managerMetrics.LastCheckpointAt.IsZero() {
		fmt.Printf("最后检查点时间: %v\n", managerMetrics.LastCheckpointAt.Format("2006-01-02 15:04:05"))
	}

	// 13. 显示各状态的指标
	fmt.Println("\n=== 各状态指标 ===")
	for _, stateName := range stateManager.ListStates() {
		if state, exists := stateManager.GetState(stateName); exists {
			stateMetrics := state.GetMetrics()
			fmt.Printf("%s:\n", stateName)
			fmt.Printf("  状态数量: %d\n", stateMetrics.StateCount)
			fmt.Printf("  GET操作: %d\n", stateMetrics.GetOperations)
			fmt.Printf("  SET操作: %d\n", stateMetrics.SetOperations)
			fmt.Printf("  DELETE操作: %d\n", stateMetrics.DeleteOperations)
			fmt.Printf("  平均延迟: %v\n", stateMetrics.AverageLatency)
			fmt.Printf("  最大延迟: %v\n", stateMetrics.MaxLatency)
			fmt.Printf("  错误数: %d\n", stateMetrics.ErrorCount)
		}
	}

	// 14. 导出状态数据
	fmt.Println("\n=== 导出状态数据 ===")
	exportData, err := stateManager.Export()
	if err != nil {
		log.Printf("导出状态数据失败: %v", err)
	} else {
		fmt.Printf("状态数据已导出 (%d 字节)\n", len(exportData))
		fmt.Printf("导出数据预览 (前500字符):\n%s...\n",
			truncateString(string(exportData), 500))
	}

	// 15. 等待自动检查点
	fmt.Println("\n=== 等待自动检查点 ===")
	fmt.Println("等待自动检查点创建...")
	time.Sleep(12 * time.Second)

	// 再次列出检查点
	checkpoints, err = checkpointManager.List(ctx)
	if err != nil {
		log.Printf("获取检查点列表失败: %v", err)
	} else {
		fmt.Printf("现在共有 %d 个检查点\n", len(checkpoints))
	}

	// 16. 测试检查点恢复
	if len(checkpoints) > 0 {
		fmt.Println("\n=== 测试检查点恢复 ===")
		latestCheckpoint := checkpoints[0]
		fmt.Printf("准备从检查点恢复: %s\n", latestCheckpoint.ID)

		// 加载检查点
		checkpointData, err := checkpointManager.Load(ctx, latestCheckpoint.ID)
		if err != nil {
			log.Printf("加载检查点失败: %v", err)
		} else {
			fmt.Printf("检查点加载成功，包含 %d 个状态\n", len(checkpointData.States))

			// 验证检查点
			if err := checkpointManager.Validate(ctx, latestCheckpoint.ID); err != nil {
				log.Printf("检查点验证失败: %v", err)
			} else {
				fmt.Println("检查点验证通过")
			}
		}
	}

	fmt.Println("\n=== 状态管理系统示例完成 ===")
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func init() {
	// Go 1.20+ 自动初始化随机数生成器，无需手动设置种子
}
