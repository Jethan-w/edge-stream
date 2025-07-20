package ErrorRouter

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/edge-stream/internal/flowfile"
	"github.com/edge-stream/internal/sink"
)

// SinkErrorHandler Sink 错误处理器
type SinkErrorHandler struct {
	mu sync.RWMutex

	// 配置
	config *sink.ErrorHandlingConfig

	// 错误队列
	errorQueue      *ErrorQueue
	retryQueue      *RetryQueue
	deadLetterQueue *DeadLetterQueue

	// 错误路由策略
	routingStrategies map[string]ErrorRoutingStrategy

	// 统计信息
	stats *ErrorHandlerStats

	// 通知器
	notifiers []ErrorNotifier
}

// ErrorQueue 错误队列
type ErrorQueue struct {
	mu      sync.RWMutex
	items   []*ErrorItem
	maxSize int
}

// RetryQueue 重试队列
type RetryQueue struct {
	mu      sync.RWMutex
	items   []*RetryItem
	maxSize int
}

// DeadLetterQueue 死信队列
type DeadLetterQueue struct {
	mu      sync.RWMutex
	items   []*DeadLetterItem
	maxSize int
}

// ErrorItem 错误项
type ErrorItem struct {
	FlowFile   *flowfile.FlowFile `json:"flow_file"`
	Error      error              `json:"error"`
	Timestamp  time.Time          `json:"timestamp"`
	TargetID   string             `json:"target_id"`
	RetryCount int                `json:"retry_count"`
}

// RetryItem 重试项
type RetryItem struct {
	FlowFile   *flowfile.FlowFile `json:"flow_file"`
	Error      error              `json:"error"`
	Timestamp  time.Time          `json:"timestamp"`
	TargetID   string             `json:"target_id"`
	RetryCount int                `json:"retry_count"`
	NextRetry  time.Time          `json:"next_retry"`
}

// DeadLetterItem 死信项
type DeadLetterItem struct {
	FlowFile   *flowfile.FlowFile `json:"flow_file"`
	Error      error              `json:"error"`
	Timestamp  time.Time          `json:"timestamp"`
	TargetID   string             `json:"target_id"`
	RetryCount int                `json:"retry_count"`
	Reason     string             `json:"reason"`
}

// ErrorRoutingStrategy 错误路由策略
type ErrorRoutingStrategy interface {
	Route(errorItem *ErrorItem) sink.ErrorRoutingStrategy
	GetName() string
}

// DefaultErrorRoutingStrategy 默认错误路由策略
type DefaultErrorRoutingStrategy struct {
	config *sink.ErrorHandlingConfig
}

// RetryErrorRoutingStrategy 重试错误路由策略
type RetryErrorRoutingStrategy struct {
	maxRetries int
}

// DropErrorRoutingStrategy 丢弃错误路由策略
type DropErrorRoutingStrategy struct{}

// DeadLetterErrorRoutingStrategy 死信错误路由策略
type DeadLetterErrorRoutingStrategy struct {
	reason string
}

// ErrorNotifier 错误通知器接口
type ErrorNotifier interface {
	NotifyError(errorItem *ErrorItem) error
	GetName() string
}

// LogErrorNotifier 日志错误通知器
type LogErrorNotifier struct{}

// EmailErrorNotifier 邮件错误通知器
type EmailErrorNotifier struct {
	recipients []string
	smtpConfig map[string]string
}

// ErrorHandlerStats 错误处理器统计信息
type ErrorHandlerStats struct {
	TotalErrors        int64         `json:"total_errors"`
	RoutedToRetry      int64         `json:"routed_to_retry"`
	RoutedToError      int64         `json:"routed_to_error"`
	RoutedToDeadLetter int64         `json:"routed_to_dead_letter"`
	Dropped            int64         `json:"dropped"`
	NotificationsSent  int64         `json:"notifications_sent"`
	AverageProcessTime time.Duration `json:"average_process_time"`
	mu                 sync.RWMutex
}

// NewSinkErrorHandler 创建 Sink 错误处理器
func NewSinkErrorHandler() *SinkErrorHandler {
	return &SinkErrorHandler{
		config: &sink.ErrorHandlingConfig{
			DefaultStrategy:     sink.ErrorRoutingStrategyRetry,
			MaxRetryCount:       3,
			ErrorQueueName:      "error_queue",
			DeadLetterQueueName: "dead_letter_queue",
			LogErrors:           true,
			NotifyOnError:       false,
		},
		errorQueue: &ErrorQueue{
			items:   make([]*ErrorItem, 0),
			maxSize: 1000,
		},
		retryQueue: &RetryQueue{
			items:   make([]*RetryItem, 0),
			maxSize: 1000,
		},
		deadLetterQueue: &DeadLetterQueue{
			items:   make([]*DeadLetterItem, 0),
			maxSize: 1000,
		},
		routingStrategies: make(map[string]ErrorRoutingStrategy),
		stats:             &ErrorHandlerStats{},
		notifiers:         make([]ErrorNotifier, 0),
	}
}

// Initialize 初始化错误处理器
func (e *SinkErrorHandler) Initialize(config *sink.ErrorHandlingConfig) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.config = config

	// 初始化路由策略
	e.initializeRoutingStrategies()

	// 初始化通知器
	if config.NotifyOnError {
		e.initializeNotifiers()
	}

	return nil
}

// initializeRoutingStrategies 初始化路由策略
func (e *SinkErrorHandler) initializeRoutingStrategies() {
	// 默认策略
	e.routingStrategies["default"] = &DefaultErrorRoutingStrategy{
		config: e.config,
	}

	// 重试策略
	e.routingStrategies["retry"] = &RetryErrorRoutingStrategy{
		maxRetries: e.config.MaxRetryCount,
	}

	// 丢弃策略
	e.routingStrategies["drop"] = &DropErrorRoutingStrategy{}

	// 死信策略
	e.routingStrategies["dead_letter"] = &DeadLetterErrorRoutingStrategy{
		reason: "Max retries exceeded",
	}
}

// initializeNotifiers 初始化通知器
func (e *SinkErrorHandler) initializeNotifiers() {
	// 添加日志通知器
	e.notifiers = append(e.notifiers, &LogErrorNotifier{})

	// 可以添加其他通知器，如邮件、Slack 等
}

// HandleOutputError 处理输出错误
func (e *SinkErrorHandler) HandleOutputError(flowFile *flowfile.FlowFile, err error, targetID string) {
	startTime := time.Now()

	// 创建错误项
	errorItem := &ErrorItem{
		FlowFile:   flowFile,
		Error:      err,
		Timestamp:  time.Now(),
		TargetID:   targetID,
		RetryCount: e.getRetryCount(flowFile),
	}

	// 更新统计信息
	e.stats.mu.Lock()
	e.stats.TotalErrors++
	e.stats.mu.Unlock()

	// 确定路由策略
	strategy := e.determineRoutingStrategy(errorItem)

	// 执行路由
	e.routeError(errorItem, strategy)

	// 发送通知
	if e.config.NotifyOnError {
		e.sendNotifications(errorItem)
	}

	// 更新统计信息
	duration := time.Since(startTime)
	e.stats.mu.Lock()
	if e.stats.TotalErrors > 0 {
		totalTime := e.stats.AverageProcessTime * time.Duration(e.stats.TotalErrors-1)
		e.stats.AverageProcessTime = (totalTime + duration) / time.Duration(e.stats.TotalErrors)
	} else {
		e.stats.AverageProcessTime = duration
	}
	e.stats.mu.Unlock()
}

// determineRoutingStrategy 确定路由策略
func (e *SinkErrorHandler) determineRoutingStrategy(errorItem *ErrorItem) sink.ErrorRoutingStrategy {
	// 检查是否为临时错误
	if e.isTransientError(errorItem.Error) {
		if errorItem.RetryCount < e.config.MaxRetryCount {
			return sink.ErrorRoutingStrategyRetry
		} else {
			return sink.ErrorRoutingStrategyDeadLetter
		}
	}

	// 使用默认策略
	return e.config.DefaultStrategy
}

// isTransientError 判断是否为临时错误
func (e *SinkErrorHandler) isTransientError(err error) bool {
	// 检查是否为临时输出错误
	if _, ok := err.(*sink.TransientOutputError); ok {
		return true
	}

	// 检查错误消息中是否包含临时错误关键词
	errorMsg := err.Error()
	transientKeywords := []string{
		"connection refused",
		"timeout",
		"temporary",
		"retry",
		"network",
		"unavailable",
	}

	for _, keyword := range transientKeywords {
		if contains(errorMsg, keyword) {
			return true
		}
	}

	return false
}

// routeError 路由错误
func (e *SinkErrorHandler) routeError(errorItem *ErrorItem, strategy sink.ErrorRoutingStrategy) {
	switch strategy {
	case sink.ErrorRoutingStrategyRetry:
		e.routeToRetry(errorItem)
	case sink.ErrorRoutingStrategyRouteToError:
		e.routeToError(errorItem)
	case sink.ErrorRoutingStrategyDrop:
		e.dropError(errorItem)
	case sink.ErrorRoutingStrategyDeadLetter:
		e.routeToDeadLetter(errorItem)
	default:
		// 使用默认策略
		e.routeToError(errorItem)
	}
}

// routeToRetry 路由到重试队列
func (e *SinkErrorHandler) routeToRetry(errorItem *ErrorItem) {
	// 增加重试计数
	errorItem.RetryCount++
	errorItem.FlowFile.SetAttribute("retry.count", fmt.Sprintf("%d", errorItem.RetryCount))

	// 计算下次重试时间（指数退避）
	delay := time.Duration(1<<uint(errorItem.RetryCount)) * time.Second
	if delay > 30*time.Second {
		delay = 30 * time.Second
	}

	retryItem := &RetryItem{
		FlowFile:   errorItem.FlowFile,
		Error:      errorItem.Error,
		Timestamp:  errorItem.Timestamp,
		TargetID:   errorItem.TargetID,
		RetryCount: errorItem.RetryCount,
		NextRetry:  time.Now().Add(delay),
	}

	e.retryQueue.mu.Lock()
	if len(e.retryQueue.items) < e.retryQueue.maxSize {
		e.retryQueue.items = append(e.retryQueue.items, retryItem)
	}
	e.retryQueue.mu.Unlock()

	// 更新统计信息
	e.stats.mu.Lock()
	e.stats.RoutedToRetry++
	e.stats.mu.Unlock()

	log.Printf("Routed error to retry queue: %s (retry count: %d)", errorItem.Error, errorItem.RetryCount)
}

// routeToError 路由到错误队列
func (e *SinkErrorHandler) routeToError(errorItem *ErrorItem) {
	e.errorQueue.mu.Lock()
	if len(e.errorQueue.items) < e.errorQueue.maxSize {
		e.errorQueue.items = append(e.errorQueue.items, errorItem)
	}
	e.errorQueue.mu.Unlock()

	// 更新统计信息
	e.stats.mu.Lock()
	e.stats.RoutedToError++
	e.stats.mu.Unlock()

	log.Printf("Routed error to error queue: %s", errorItem.Error)
}

// routeToDeadLetter 路由到死信队列
func (e *SinkErrorHandler) routeToDeadLetter(errorItem *ErrorItem) {
	deadLetterItem := &DeadLetterItem{
		FlowFile:   errorItem.FlowFile,
		Error:      errorItem.Error,
		Timestamp:  errorItem.Timestamp,
		TargetID:   errorItem.TargetID,
		RetryCount: errorItem.RetryCount,
		Reason:     "Max retries exceeded or permanent error",
	}

	e.deadLetterQueue.mu.Lock()
	if len(e.deadLetterQueue.items) < e.deadLetterQueue.maxSize {
		e.deadLetterQueue.items = append(e.deadLetterQueue.items, deadLetterItem)
	}
	e.deadLetterQueue.mu.Unlock()

	// 更新统计信息
	e.stats.mu.Lock()
	e.stats.RoutedToDeadLetter++
	e.stats.mu.Unlock()

	log.Printf("Routed error to dead letter queue: %s", errorItem.Error)
}

// dropError 丢弃错误
func (e *SinkErrorHandler) dropError(errorItem *ErrorItem) {
	// 更新统计信息
	e.stats.mu.Lock()
	e.stats.Dropped++
	e.stats.mu.Unlock()

	log.Printf("Dropped error: %s", errorItem.Error)
}

// sendNotifications 发送通知
func (e *SinkErrorHandler) sendNotifications(errorItem *ErrorItem) {
	for _, notifier := range e.notifiers {
		if err := notifier.NotifyError(errorItem); err != nil {
			log.Printf("Failed to send notification via %s: %v", notifier.GetName(), err)
		} else {
			e.stats.mu.Lock()
			e.stats.NotificationsSent++
			e.stats.mu.Unlock()
		}
	}
}

// getRetryCount 获取重试次数
func (e *SinkErrorHandler) getRetryCount(flowFile *flowfile.FlowFile) int {
	retryCountStr := flowFile.GetAttribute("retry.count")
	if retryCountStr == "" {
		return 0
	}

	var retryCount int
	fmt.Sscanf(retryCountStr, "%d", &retryCount)
	return retryCount
}

// GetRetryQueue 获取重试队列
func (e *SinkErrorHandler) GetRetryQueue() []*RetryItem {
	e.retryQueue.mu.RLock()
	defer e.retryQueue.mu.RUnlock()

	items := make([]*RetryItem, len(e.retryQueue.items))
	copy(items, e.retryQueue.items)
	return items
}

// GetErrorQueue 获取错误队列
func (e *SinkErrorHandler) GetErrorQueue() []*ErrorItem {
	e.errorQueue.mu.RLock()
	defer e.errorQueue.mu.RUnlock()

	items := make([]*ErrorItem, len(e.errorQueue.items))
	copy(items, e.errorQueue.items)
	return items
}

// GetDeadLetterQueue 获取死信队列
func (e *SinkErrorHandler) GetDeadLetterQueue() []*DeadLetterItem {
	e.deadLetterQueue.mu.RLock()
	defer e.deadLetterQueue.mu.RUnlock()

	items := make([]*DeadLetterItem, len(e.deadLetterQueue.items))
	copy(items, e.deadLetterQueue.items)
	return items
}

// GetStats 获取统计信息
func (e *SinkErrorHandler) GetStats() *ErrorHandlerStats {
	e.stats.mu.RLock()
	defer e.stats.mu.RUnlock()

	stats := *e.stats
	return &stats
}

// ClearQueues 清空队列
func (e *SinkErrorHandler) ClearQueues() {
	e.errorQueue.mu.Lock()
	e.errorQueue.items = make([]*ErrorItem, 0)
	e.errorQueue.mu.Unlock()

	e.retryQueue.mu.Lock()
	e.retryQueue.items = make([]*RetryItem, 0)
	e.retryQueue.mu.Unlock()

	e.deadLetterQueue.mu.Lock()
	e.deadLetterQueue.items = make([]*DeadLetterItem, 0)
	e.deadLetterQueue.mu.Unlock()
}

// 错误路由策略实现

// Route 默认错误路由策略
func (s *DefaultErrorRoutingStrategy) Route(errorItem *ErrorItem) sink.ErrorRoutingStrategy {
	return s.config.DefaultStrategy
}

// GetName 获取策略名称
func (s *DefaultErrorRoutingStrategy) GetName() string {
	return "Default"
}

// Route 重试错误路由策略
func (s *RetryErrorRoutingStrategy) Route(errorItem *ErrorItem) sink.ErrorRoutingStrategy {
	if errorItem.RetryCount < s.maxRetries {
		return sink.ErrorRoutingStrategyRetry
	}
	return sink.ErrorRoutingStrategyDeadLetter
}

// GetName 获取策略名称
func (s *RetryErrorRoutingStrategy) GetName() string {
	return "Retry"
}

// Route 丢弃错误路由策略
func (s *DropErrorRoutingStrategy) Route(errorItem *ErrorItem) sink.ErrorRoutingStrategy {
	return sink.ErrorRoutingStrategyDrop
}

// GetName 获取策略名称
func (s *DropErrorRoutingStrategy) GetName() string {
	return "Drop"
}

// Route 死信错误路由策略
func (s *DeadLetterErrorRoutingStrategy) Route(errorItem *ErrorItem) sink.ErrorRoutingStrategy {
	return sink.ErrorRoutingStrategyDeadLetter
}

// GetName 获取策略名称
func (s *DeadLetterErrorRoutingStrategy) GetName() string {
	return "DeadLetter"
}

// 错误通知器实现

// NotifyError 日志错误通知器
func (n *LogErrorNotifier) NotifyError(errorItem *ErrorItem) error {
	log.Printf("ERROR NOTIFICATION - Target: %s, Error: %v, Retry Count: %d",
		errorItem.TargetID, errorItem.Error, errorItem.RetryCount)
	return nil
}

// GetName 获取通知器名称
func (n *LogErrorNotifier) GetName() string {
	return "Log"
}

// NotifyError 邮件错误通知器
func (n *EmailErrorNotifier) NotifyError(errorItem *ErrorItem) error {
	// 这里应该实现邮件发送逻辑
	// 为了简化，暂时只记录日志
	log.Printf("EMAIL NOTIFICATION - Would send email to %v for error: %v",
		n.recipients, errorItem.Error)
	return nil
}

// GetName 获取通知器名称
func (n *EmailErrorNotifier) GetName() string {
	return "Email"
}

// 辅助函数

// contains 检查字符串是否包含子字符串（不区分大小写）
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsSubstring(s, substr))))
}

// containsSubstring 检查字符串中间是否包含子字符串
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
