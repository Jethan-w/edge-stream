package stream_engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// Scheduler 调度器接口
type Scheduler interface {
	// Start 启动调度器
	Start(ctx context.Context) error

	// Stop 停止调度器
	Stop(ctx context.Context) error

	// Schedule 调度处理器
	Schedule(processor ProcessorNode, trigger Trigger) error

	// Unschedule 取消调度
	Unschedule(processorID string) error

	// RegisterStrategy 注册调度策略
	RegisterStrategy(name string, strategy SchedulingStrategy) error

	// GetScheduledProcessors 获取已调度的处理器
	GetScheduledProcessors() []ProcessorNode
}

// SchedulingStrategy 调度策略接口
type SchedulingStrategy interface {
	// GetType 获取调度类型
	GetType() SchedulingType

	// GetMaxThreads 获取最大线程数
	GetMaxThreads() int

	// ShouldTrigger 是否应该触发
	ShouldTrigger(context ProcessContext) bool

	// GetNextTriggerTime 获取下次触发时间
	GetNextTriggerTime() time.Time
}

// SchedulingType 调度类型
type SchedulingType string

const (
	SchedulingTypeEventDriven SchedulingType = "EVENT_DRIVEN" // 事件驱动
	SchedulingTypeTimerDriven SchedulingType = "TIMER_DRIVEN" // 定时驱动
	SchedulingTypeCronDriven  SchedulingType = "CRON_DRIVEN"  // CRON驱动
)

// Trigger 触发器接口
type Trigger interface {
	// ShouldFire 是否应该触发
	ShouldFire(context ProcessContext) bool

	// GetNextFireTime 获取下次触发时间
	GetNextFireTime() time.Time

	// GetStrategy 获取调度策略
	GetStrategy() SchedulingStrategy
}

// ProcessContext 处理上下文
type ProcessContext struct {
	QueueSize   int64
	DataSize    int64
	WaitTime    time.Duration
	LastTrigger time.Time
	ProcessorID string
	Attributes  map[string]interface{}
}

// StandardScheduler 标准调度器实现
type StandardScheduler struct {
	mu                   sync.RWMutex
	processors           map[string]ProcessorNode
	triggers             map[string]Trigger
	strategies           map[string]SchedulingStrategy
	cronScheduler        *cron.Cron
	timerScheduler       *TimerScheduler
	eventDrivenScheduler *EventDrivenScheduler
	ctx                  context.Context
	cancel               context.CancelFunc
	running              bool
}

// NewScheduler 创建新的调度器
func NewScheduler() *StandardScheduler {
	return &StandardScheduler{
		processors:           make(map[string]ProcessorNode),
		triggers:             make(map[string]Trigger),
		strategies:           make(map[string]SchedulingStrategy),
		cronScheduler:        cron.New(cron.WithSeconds()),
		timerScheduler:       NewTimerScheduler(),
		eventDrivenScheduler: NewEventDrivenScheduler(),
	}
}

// Start 启动调度器
func (s *StandardScheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("scheduler is already running")
	}

	s.ctx, s.cancel = context.WithCancel(ctx)
	s.running = true

	// 启动CRON调度器
	s.cronScheduler.Start()

	// 启动定时调度器
	if err := s.timerScheduler.Start(s.ctx); err != nil {
		return fmt.Errorf("failed to start timer scheduler: %w", err)
	}

	// 启动事件驱动调度器
	if err := s.eventDrivenScheduler.Start(s.ctx); err != nil {
		return fmt.Errorf("failed to start event driven scheduler: %w", err)
	}

	return nil
}

// Stop 停止调度器
func (s *StandardScheduler) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.running = false

	// 停止CRON调度器
	s.cronScheduler.Stop()

	// 停止定时调度器
	if err := s.timerScheduler.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop timer scheduler: %w", err)
	}

	// 停止事件驱动调度器
	if err := s.eventDrivenScheduler.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop event driven scheduler: %w", err)
	}

	// 取消上下文
	if s.cancel != nil {
		s.cancel()
	}

	return nil
}

// Schedule 调度处理器
func (s *StandardScheduler) Schedule(processor ProcessorNode, trigger Trigger) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	processorID := processor.GetID()

	// 检查是否已经调度
	if _, exists := s.processors[processorID]; exists {
		return fmt.Errorf("processor %s is already scheduled", processorID)
	}

	// 根据触发器类型进行调度
	switch trigger.GetStrategy().GetType() {
	case SchedulingTypeEventDriven:
		return s.scheduleEventDriven(processor, trigger)
	case SchedulingTypeTimerDriven:
		return s.scheduleTimerDriven(processor, trigger)
	case SchedulingTypeCronDriven:
		return s.scheduleCronDriven(processor, trigger)
	default:
		return fmt.Errorf("unknown scheduling type: %s", trigger.GetStrategy().GetType())
	}
}

// Unschedule 取消调度
func (s *StandardScheduler) Unschedule(processorID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	processor, exists := s.processors[processorID]
	if !exists {
		return fmt.Errorf("processor %s is not scheduled", processorID)
	}

	// 根据调度类型取消调度
	trigger := s.triggers[processorID]
	switch trigger.GetStrategy().GetType() {
	case SchedulingTypeEventDriven:
		s.eventDrivenScheduler.Unschedule(processorID)
	case SchedulingTypeTimerDriven:
		s.timerScheduler.Unschedule(processorID)
	case SchedulingTypeCronDriven:
		// CRON调度器会自动处理
	}

	// 清理记录
	delete(s.processors, processorID)
	delete(s.triggers, processorID)

	return nil
}

// RegisterStrategy 注册调度策略
func (s *StandardScheduler) RegisterStrategy(name string, strategy SchedulingStrategy) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.strategies[name]; exists {
		return fmt.Errorf("strategy %s is already registered", name)
	}

	s.strategies[name] = strategy
	return nil
}

// GetScheduledProcessors 获取已调度的处理器
func (s *StandardScheduler) GetScheduledProcessors() []ProcessorNode {
	s.mu.RLock()
	defer s.mu.RUnlock()

	processors := make([]ProcessorNode, 0, len(s.processors))
	for _, processor := range s.processors {
		processors = append(processors, processor)
	}

	return processors
}

// scheduleEventDriven 调度事件驱动处理器
func (s *StandardScheduler) scheduleEventDriven(processor ProcessorNode, trigger Trigger) error {
	processorID := processor.GetID()

	// 注册到事件驱动调度器
	if err := s.eventDrivenScheduler.Schedule(processor, trigger); err != nil {
		return fmt.Errorf("failed to schedule event driven processor: %w", err)
	}

	// 记录调度信息
	s.processors[processorID] = processor
	s.triggers[processorID] = trigger

	return nil
}

// scheduleTimerDriven 调度定时驱动处理器
func (s *StandardScheduler) scheduleTimerDriven(processor ProcessorNode, trigger Trigger) error {
	processorID := processor.GetID()

	// 注册到定时调度器
	if err := s.timerScheduler.Schedule(processor, trigger); err != nil {
		return fmt.Errorf("failed to schedule timer driven processor: %w", err)
	}

	// 记录调度信息
	s.processors[processorID] = processor
	s.triggers[processorID] = trigger

	return nil
}

// scheduleCronDriven 调度CRON驱动处理器
func (s *StandardScheduler) scheduleCronDriven(processor ProcessorNode, trigger Trigger) error {
	processorID := processor.GetID()

	// 获取CRON表达式
	cronTrigger, ok := trigger.(*CronDrivenTrigger)
	if !ok {
		return fmt.Errorf("invalid cron trigger type")
	}

	// 添加到CRON调度器
	entryID, err := s.cronScheduler.AddFunc(cronTrigger.GetCronExpression(), func() {
		s.executeProcessor(processor)
	})

	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	// 记录调度信息
	s.processors[processorID] = processor
	s.triggers[processorID] = trigger

	// 保存CRON条目ID（如果需要后续管理）
	cronTrigger.SetEntryID(entryID)

	return nil
}

// executeProcessor 执行处理器
func (s *StandardScheduler) executeProcessor(processor ProcessorNode) {
	// 创建处理上下文
	context := ProcessContext{
		ProcessorID: processor.GetID(),
		LastTrigger: time.Now(),
		Attributes:  make(map[string]interface{}),
	}

	// 执行处理器
	if err := processor.OnTrigger(context); err != nil {
		// 记录错误，但不中断调度
		// 这里可以添加错误处理逻辑
	}
}
