package stream_engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// EventDrivenTrigger 事件驱动触发器
type EventDrivenTrigger struct {
	strategy SchedulingStrategy
	mu       sync.RWMutex
}

// NewEventDrivenTrigger 创建事件驱动触发器
func NewEventDrivenTrigger(strategy SchedulingStrategy) *EventDrivenTrigger {
	return &EventDrivenTrigger{
		strategy: strategy,
	}
}

// ShouldFire 是否应该触发
func (t *EventDrivenTrigger) ShouldFire(context ProcessContext) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// 事件驱动触发器在有数据到达时立即触发
	return context.QueueSize > 0 || context.DataSize > 0
}

// GetNextFireTime 获取下次触发时间
func (t *EventDrivenTrigger) GetNextFireTime() time.Time {
	// 事件驱动触发器没有固定的下次触发时间
	return time.Time{}
}

// GetStrategy 获取调度策略
func (t *EventDrivenTrigger) GetStrategy() SchedulingStrategy {
	return t.strategy
}

// TimerDrivenTrigger 定时驱动触发器
type TimerDrivenTrigger struct {
	strategy SchedulingStrategy
	interval time.Duration
	lastFire time.Time
	mu       sync.RWMutex
}

// NewTimerDrivenTrigger 创建定时驱动触发器
func NewTimerDrivenTrigger(strategy SchedulingStrategy) *TimerDrivenTrigger {
	// 从策略中获取间隔时间
	interval := time.Second * 5 // 默认5秒

	return &TimerDrivenTrigger{
		strategy: strategy,
		interval: interval,
		lastFire: time.Now(),
	}
}

// SetInterval 设置触发间隔
func (t *TimerDrivenTrigger) SetInterval(interval time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.interval = interval
}

// ShouldFire 是否应该触发
func (t *TimerDrivenTrigger) ShouldFire(context ProcessContext) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	now := time.Now()
	return now.Sub(t.lastFire) >= t.interval
}

// GetNextFireTime 获取下次触发时间
func (t *TimerDrivenTrigger) GetNextFireTime() time.Time {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.lastFire.Add(t.interval)
}

// GetStrategy 获取调度策略
func (t *TimerDrivenTrigger) GetStrategy() SchedulingStrategy {
	return t.strategy
}

// UpdateLastFire 更新最后触发时间
func (t *TimerDrivenTrigger) UpdateLastFire() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lastFire = time.Now()
}

// CronDrivenTrigger CRON驱动触发器
type CronDrivenTrigger struct {
	strategy       SchedulingStrategy
	cronExpression string
	entryID        cron.EntryID
	mu             sync.RWMutex
}

// NewCronDrivenTrigger 创建CRON驱动触发器
func NewCronDrivenTrigger(strategy SchedulingStrategy) *CronDrivenTrigger {
	return &CronDrivenTrigger{
		strategy:       strategy,
		cronExpression: "0 * * * *", // 默认每小时触发
	}
}

// SetCronExpression 设置CRON表达式
func (t *CronDrivenTrigger) SetCronExpression(expression string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// 验证CRON表达式
	_, err := cron.ParseStandard(expression)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	t.cronExpression = expression
	return nil
}

// GetCronExpression 获取CRON表达式
func (t *CronDrivenTrigger) GetCronExpression() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.cronExpression
}

// ShouldFire 是否应该触发
func (t *CronDrivenTrigger) ShouldFire(context ProcessContext) bool {
	// CRON触发器由CRON调度器管理，这里主要用于查询
	return false
}

// GetNextFireTime 获取下次触发时间
func (t *CronDrivenTrigger) GetNextFireTime() time.Time {
	// 这里需要根据CRON表达式计算下次触发时间
	// 简化实现，实际应该使用CRON库计算
	return time.Now().Add(time.Hour)
}

// GetStrategy 获取调度策略
func (t *CronDrivenTrigger) GetStrategy() SchedulingStrategy {
	return t.strategy
}

// SetEntryID 设置CRON条目ID
func (t *CronDrivenTrigger) SetEntryID(entryID cron.EntryID) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.entryID = entryID
}

// GetEntryID 获取CRON条目ID
func (t *CronDrivenTrigger) GetEntryID() cron.EntryID {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.entryID
}

// DataSizeTrigger 基于数据大小的触发器
type DataSizeTrigger struct {
	strategy  SchedulingStrategy
	threshold int64
	mu        sync.RWMutex
}

// NewDataSizeTrigger 创建基于数据大小的触发器
func NewDataSizeTrigger(strategy SchedulingStrategy, threshold int64) *DataSizeTrigger {
	return &DataSizeTrigger{
		strategy:  strategy,
		threshold: threshold,
	}
}

// ShouldFire 是否应该触发
func (t *DataSizeTrigger) ShouldFire(context ProcessContext) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return context.DataSize >= t.threshold
}

// GetNextFireTime 获取下次触发时间
func (t *DataSizeTrigger) GetNextFireTime() time.Time {
	// 基于数据大小的触发器没有固定的下次触发时间
	return time.Time{}
}

// GetStrategy 获取调度策略
func (t *DataSizeTrigger) GetStrategy() SchedulingStrategy {
	return t.strategy
}

// SetThreshold 设置阈值
func (t *DataSizeTrigger) SetThreshold(threshold int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.threshold = threshold
}

// QueueSizeTrigger 基于队列大小的触发器
type QueueSizeTrigger struct {
	strategy  SchedulingStrategy
	threshold int64
	mu        sync.RWMutex
}

// NewQueueSizeTrigger 创建基于队列大小的触发器
func NewQueueSizeTrigger(strategy SchedulingStrategy, threshold int64) *QueueSizeTrigger {
	return &QueueSizeTrigger{
		strategy:  strategy,
		threshold: threshold,
	}
}

// ShouldFire 是否应该触发
func (t *QueueSizeTrigger) ShouldFire(context ProcessContext) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return context.QueueSize >= t.threshold
}

// GetNextFireTime 获取下次触发时间
func (t *QueueSizeTrigger) GetNextFireTime() time.Time {
	// 基于队列大小的触发器没有固定的下次触发时间
	return time.Time{}
}

// GetStrategy 获取调度策略
func (t *QueueSizeTrigger) GetStrategy() SchedulingStrategy {
	return t.strategy
}

// SetThreshold 设置阈值
func (t *QueueSizeTrigger) SetThreshold(threshold int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.threshold = threshold
}

// CompositeTrigger 组合触发器
type CompositeTrigger struct {
	strategy SchedulingStrategy
	triggers []Trigger
	operator TriggerOperator
	mu       sync.RWMutex
}

// TriggerOperator 触发器操作符
type TriggerOperator string

const (
	TriggerOperatorAND TriggerOperator = "AND" // 所有触发器都满足
	TriggerOperatorOR  TriggerOperator = "OR"  // 任一触发器满足
)

// NewCompositeTrigger 创建组合触发器
func NewCompositeTrigger(strategy SchedulingStrategy, operator TriggerOperator) *CompositeTrigger {
	return &CompositeTrigger{
		strategy: strategy,
		triggers: make([]Trigger, 0),
		operator: operator,
	}
}

// AddTrigger 添加触发器
func (t *CompositeTrigger) AddTrigger(trigger Trigger) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.triggers = append(t.triggers, trigger)
}

// ShouldFire 是否应该触发
func (t *CompositeTrigger) ShouldFire(context ProcessContext) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.triggers) == 0 {
		return false
	}

	switch t.operator {
	case TriggerOperatorAND:
		// 所有触发器都满足
		for _, trigger := range t.triggers {
			if !trigger.ShouldFire(context) {
				return false
			}
		}
		return true

	case TriggerOperatorOR:
		// 任一触发器满足
		for _, trigger := range t.triggers {
			if trigger.ShouldFire(context) {
				return true
			}
		}
		return false

	default:
		return false
	}
}

// GetNextFireTime 获取下次触发时间
func (t *CompositeTrigger) GetNextFireTime() time.Time {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.triggers) == 0 {
		return time.Time{}
	}

	switch t.operator {
	case TriggerOperatorAND:
		// 取最晚的触发时间
		var latestTime time.Time
		for _, trigger := range t.triggers {
			nextTime := trigger.GetNextFireTime()
			if nextTime.After(latestTime) {
				latestTime = nextTime
			}
		}
		return latestTime

	case TriggerOperatorOR:
		// 取最早的触发时间
		var earliestTime time.Time
		for i, trigger := range t.triggers {
			nextTime := trigger.GetNextFireTime()
			if i == 0 || nextTime.Before(earliestTime) {
				earliestTime = nextTime
			}
		}
		return earliestTime

	default:
		return time.Time{}
	}
}

// GetStrategy 获取调度策略
func (t *CompositeTrigger) GetStrategy() SchedulingStrategy {
	return t.strategy
}
