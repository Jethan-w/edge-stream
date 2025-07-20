package stream_engine

import (
	"context"
	"fmt"
	"log"
	"time"
)

// ExampleProcessor 示例处理器
type ExampleProcessor struct {
	*BaseProcessorNode
	counter int
}

// NewExampleProcessor 创建示例处理器
func NewExampleProcessor(id, name string) *ExampleProcessor {
	return &ExampleProcessor{
		BaseProcessorNode: NewBaseProcessorNode(id, name, "ExampleProcessor"),
		counter:           0,
	}
}

// OnTrigger 触发处理器
func (p *ExampleProcessor) OnTrigger(context ProcessContext) error {
	p.counter++
	log.Printf("Processor %s triggered, counter: %d", p.GetID(), p.counter)

	// 模拟处理时间
	time.Sleep(time.Millisecond * 100)

	return nil
}

// ExampleDataSizeStrategy 示例数据大小策略
type ExampleDataSizeStrategy struct {
	maxThreads int
	threshold  int64
}

// NewExampleDataSizeStrategy 创建数据大小策略
func NewExampleDataSizeStrategy(maxThreads int, threshold int64) *ExampleDataSizeStrategy {
	return &ExampleDataSizeStrategy{
		maxThreads: maxThreads,
		threshold:  threshold,
	}
}

// GetType 获取调度类型
func (s *ExampleDataSizeStrategy) GetType() SchedulingType {
	return SchedulingTypeEventDriven
}

// GetMaxThreads 获取最大线程数
func (s *ExampleDataSizeStrategy) GetMaxThreads() int {
	return s.maxThreads
}

// ShouldTrigger 是否应该触发
func (s *ExampleDataSizeStrategy) ShouldTrigger(context ProcessContext) bool {
	return context.DataSize >= s.threshold
}

// GetNextTriggerTime 获取下次触发时间
func (s *ExampleDataSizeStrategy) GetNextTriggerTime() time.Time {
	return time.Now().Add(time.Second * 5)
}

// ExampleTimerStrategy 示例定时策略
type ExampleTimerStrategy struct {
	maxThreads int
	interval   time.Duration
}

// NewExampleTimerStrategy 创建定时策略
func NewExampleTimerStrategy(maxThreads int, interval time.Duration) *ExampleTimerStrategy {
	return &ExampleTimerStrategy{
		maxThreads: maxThreads,
		interval:   interval,
	}
}

// GetType 获取调度类型
func (s *ExampleTimerStrategy) GetType() SchedulingType {
	return SchedulingTypeTimerDriven
}

// GetMaxThreads 获取最大线程数
func (s *ExampleTimerStrategy) GetMaxThreads() int {
	return s.maxThreads
}

// ShouldTrigger 是否应该触发
func (s *ExampleTimerStrategy) ShouldTrigger(context ProcessContext) bool {
	// 定时策略由调度器管理
	return false
}

// GetNextTriggerTime 获取下次触发时间
func (s *ExampleTimerStrategy) GetNextTriggerTime() time.Time {
	return time.Now().Add(s.interval)
}

// ExampleCronStrategy 示例CRON策略
type ExampleCronStrategy struct {
	maxThreads int
	expression string
}

// NewExampleCronStrategy 创建CRON策略
func NewExampleCronStrategy(maxThreads int, expression string) *ExampleCronStrategy {
	return &ExampleCronStrategy{
		maxThreads: maxThreads,
		expression: expression,
	}
}

// GetType 获取调度类型
func (s *ExampleCronStrategy) GetType() SchedulingType {
	return SchedulingTypeCronDriven
}

// GetMaxThreads 获取最大线程数
func (s *ExampleCronStrategy) GetMaxThreads() int {
	return s.maxThreads
}

// ShouldTrigger 是否应该触发
func (s *ExampleCronStrategy) ShouldTrigger(context ProcessContext) bool {
	// CRON策略由调度器管理
	return false
}

// GetNextTriggerTime 获取下次触发时间
func (s *ExampleCronStrategy) GetNextTriggerTime() time.Time {
	// 这里应该根据CRON表达式计算下次触发时间
	return time.Now().Add(time.Minute)
}

// RunBasicExample 运行基础示例
func RunBasicExample() {
	fmt.Println("=== StreamEngine 基础示例 ===")

	// 创建引擎配置
	config := &EngineConfig{
		MaxThreads:            8,
		CoreThreads:           4,
		QueueCapacity:         1000,
		BackpressureThreshold: 0.8,
		ClusterEnabled:        false,
		SyncInterval:          time.Second * 5,
	}

	// 创建流引擎
	engine := NewStandardStreamEngine(config)

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动引擎
	if err := engine.Start(ctx); err != nil {
		log.Fatalf("Failed to start engine: %v", err)
	}

	fmt.Println("引擎已启动")

	// 创建处理器
	processor1 := NewExampleProcessor("proc-1", "示例处理器1")
	processor2 := NewExampleProcessor("proc-2", "示例处理器2")

	// 创建调度策略
	strategy1 := NewExampleDataSizeStrategy(2, 100)
	strategy2 := NewExampleTimerStrategy(1, time.Second*3)

	// 调度处理器
	if err := engine.Schedule(processor1, strategy1); err != nil {
		log.Fatalf("Failed to schedule processor1: %v", err)
	}

	if err := engine.Schedule(processor2, strategy2); err != nil {
		log.Fatalf("Failed to schedule processor2: %v", err)
	}

	fmt.Println("处理器已调度")

	// 运行一段时间
	time.Sleep(time.Second * 10)

	// 停止引擎
	if err := engine.Stop(ctx); err != nil {
		log.Fatalf("Failed to stop engine: %v", err)
	}

	fmt.Println("引擎已停止")
}

// RunAdvancedExample 运行高级示例
func RunAdvancedExample() {
	fmt.Println("=== StreamEngine 高级示例 ===")

	// 创建引擎配置（启用集群）
	config := &EngineConfig{
		MaxThreads:            16,
		CoreThreads:           8,
		QueueCapacity:         5000,
		BackpressureThreshold: 0.7,
		ClusterEnabled:        true,
		ZooKeeperConnect:      "localhost:2181",
		SyncInterval:          time.Second * 3,
	}

	// 创建流引擎
	engine := NewStandardStreamEngine(config)

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动引擎
	if err := engine.Start(ctx); err != nil {
		log.Fatalf("Failed to start engine: %v", err)
	}

	fmt.Println("高级引擎已启动")

	// 创建多个处理器
	processors := make([]*ExampleProcessor, 5)
	strategies := make([]SchedulingStrategy, 5)

	for i := 0; i < 5; i++ {
		processors[i] = NewExampleProcessor(fmt.Sprintf("proc-%d", i+1), fmt.Sprintf("处理器%d", i+1))

		// 使用不同的调度策略
		switch i % 3 {
		case 0:
			strategies[i] = NewExampleDataSizeStrategy(2, int64(100*(i+1)))
		case 1:
			strategies[i] = NewExampleTimerStrategy(1, time.Duration(i+1)*time.Second)
		case 2:
			strategies[i] = NewExampleCronStrategy(1, "0 */1 * * *") // 每小时执行
		}

		// 调度处理器
		if err := engine.Schedule(processors[i], strategies[i]); err != nil {
			log.Fatalf("Failed to schedule processor %d: %v", i+1, err)
		}
	}

	fmt.Println("所有处理器已调度")

	// 监控引擎状态
	go func() {
		ticker := time.NewTicker(time.Second * 2)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				status := engine.GetStatus()
				threadPoolStats := engine.GetThreadPoolManager().GetThreadPoolStats()

				fmt.Printf("引擎状态: %s, 线程池利用率: %.2f%%\n",
					status, threadPoolStats.UtilizationRate*100)
			}
		}
	}()

	// 运行一段时间
	time.Sleep(time.Second * 30)

	// 停止引擎
	if err := engine.Stop(ctx); err != nil {
		log.Fatalf("Failed to stop engine: %v", err)
	}

	fmt.Println("高级引擎已停止")
}

// RunBackpressureExample 运行背压示例
func RunBackpressureExample() {
	fmt.Println("=== StreamEngine 背压示例 ===")

	// 创建引擎配置
	config := &EngineConfig{
		MaxThreads:            4,
		CoreThreads:           2,
		QueueCapacity:         100, // 小容量队列，容易触发背压
		BackpressureThreshold: 0.5, // 低阈值，容易触发背压
		ClusterEnabled:        false,
		SyncInterval:          time.Second * 2,
	}

	// 创建流引擎
	engine := NewStandardStreamEngine(config)

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动引擎
	if err := engine.Start(ctx); err != nil {
		log.Fatalf("Failed to start engine: %v", err)
	}

	fmt.Println("背压测试引擎已启动")

	// 创建处理器
	processor := NewExampleProcessor("backpressure-proc", "背压测试处理器")

	// 创建快速触发策略
	strategy := NewExampleTimerStrategy(1, time.Millisecond*100) // 每100ms触发一次

	// 调度处理器
	if err := engine.Schedule(processor, strategy); err != nil {
		log.Fatalf("Failed to schedule processor: %v", err)
	}

	fmt.Println("处理器已调度，将快速触发以测试背压")

	// 运行一段时间
	time.Sleep(time.Second * 15)

	// 停止引擎
	if err := engine.Stop(ctx); err != nil {
		log.Fatalf("Failed to stop engine: %v", err)
	}

	fmt.Println("背压测试引擎已停止")
}

// RunClusterExample 运行集群示例
func RunClusterExample() {
	fmt.Println("=== StreamEngine 集群示例 ===")

	// 创建引擎配置（启用集群）
	config := &EngineConfig{
		MaxThreads:            8,
		CoreThreads:           4,
		QueueCapacity:         2000,
		BackpressureThreshold: 0.8,
		ClusterEnabled:        true,
		ZooKeeperConnect:      "localhost:2181",
		SyncInterval:          time.Second * 5,
	}

	// 创建流引擎
	engine := NewStandardStreamEngine(config)

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动引擎
	if err := engine.Start(ctx); err != nil {
		log.Fatalf("Failed to start engine: %v", err)
	}

	fmt.Println("集群引擎已启动")

	// 监控集群状态
	go func() {
		ticker := time.NewTicker(time.Second * 5)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				clusterState := engine.(*StandardStreamEngine).clusterCoordinator.GetClusterState()
				isLeader := engine.(*StandardStreamEngine).clusterCoordinator.IsLeader()

				fmt.Printf("集群状态: 总节点数=%d, 活跃节点数=%d, 是否为领导者=%v\n",
					clusterState.TotalNodes, clusterState.ActiveNodes, isLeader)
			}
		}
	}()

	// 创建处理器
	processor := NewExampleProcessor("cluster-proc", "集群处理器")
	strategy := NewExampleTimerStrategy(2, time.Second*2)

	// 调度处理器
	if err := engine.Schedule(processor, strategy); err != nil {
		log.Fatalf("Failed to schedule processor: %v", err)
	}

	fmt.Println("集群处理器已调度")

	// 运行一段时间
	time.Sleep(time.Second * 20)

	// 停止引擎
	if err := engine.Stop(ctx); err != nil {
		log.Fatalf("Failed to stop engine: %v", err)
	}

	fmt.Println("集群引擎已停止")
}
