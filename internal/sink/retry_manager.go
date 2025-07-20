package sink

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"
)

// RetryManager 重试管理器
type RetryManager struct {
	mu sync.RWMutex

	// 配置
	config *ReliabilityConfig

	// 重试策略
	strategy RetryStrategy

	// 统计信息
	stats *RetryStats

	// 上下文控制
	ctx    context.Context
	cancel context.CancelFunc
}

// RetryStrategy 重试策略接口
type RetryStrategy interface {
	CalculateDelay(retryCount int) time.Duration
	ShouldRetry(err error, retryCount int) bool
	GetMaxRetries() int
}

// ExponentialBackoffStrategy 指数退避策略
type ExponentialBackoffStrategy struct {
	baseDelay  time.Duration
	maxDelay   time.Duration
	multiplier float64
	jitter     bool
	maxRetries int
}

// LinearBackoffStrategy 线性退避策略
type LinearBackoffStrategy struct {
	baseDelay  time.Duration
	maxDelay   time.Duration
	step       time.Duration
	maxRetries int
}

// FixedDelayStrategy 固定延迟策略
type FixedDelayStrategy struct {
	delay      time.Duration
	maxRetries int
}

// RetryStats 重试统计信息
type RetryStats struct {
	TotalOperations      int64         `json:"total_operations"`
	SuccessfulOperations int64         `json:"successful_operations"`
	FailedOperations     int64         `json:"failed_operations"`
	TotalRetries         int64         `json:"total_retries"`
	AverageRetries       float64       `json:"average_retries"`
	TotalRetryTime       time.Duration `json:"total_retry_time"`
	AverageRetryTime     time.Duration `json:"average_retry_time"`
	mu                   sync.RWMutex
}

// RetryableOperation 可重试操作接口
type RetryableOperation interface {
	Execute() error
	GetOperationID() string
}

// NewRetryManager 创建重试管理器
func NewRetryManager() *RetryManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &RetryManager{
		config: &ReliabilityConfig{
			Strategy:           ReliabilityStrategyAtLeastOnce,
			MaxRetries:         3,
			BaseRetryDelay:     time.Second,
			MaxRetryDelay:      30 * time.Second,
			RetryMultiplier:    2.0,
			EnableTransaction:  false,
			TransactionTimeout: 30 * time.Second,
		},
		stats:  &RetryStats{},
		ctx:    ctx,
		cancel: cancel,
	}
}

// Initialize 初始化重试管理器
func (r *RetryManager) Initialize(config *ReliabilityConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.config = config

	// 根据策略创建重试策略
	switch config.Strategy {
	case ReliabilityStrategyAtMostOnce:
		r.strategy = &FixedDelayStrategy{
			delay:      config.BaseRetryDelay,
			maxRetries: 0, // 不重试
		}
	case ReliabilityStrategyAtLeastOnce:
		r.strategy = &ExponentialBackoffStrategy{
			baseDelay:  config.BaseRetryDelay,
			maxDelay:   config.MaxRetryDelay,
			multiplier: config.RetryMultiplier,
			jitter:     true,
			maxRetries: config.MaxRetries,
		}
	case ReliabilityStrategyExactlyOnce:
		r.strategy = &ExponentialBackoffStrategy{
			baseDelay:  config.BaseRetryDelay,
			maxDelay:   config.MaxRetryDelay,
			multiplier: config.RetryMultiplier,
			jitter:     false, // 精确一次不需要抖动
			maxRetries: config.MaxRetries,
		}
	default:
		return fmt.Errorf("unsupported reliability strategy: %s", config.Strategy)
	}

	return nil
}

// ExecuteWithRetry 执行带重试的操作
func (r *RetryManager) ExecuteWithRetry(operation RetryableOperation) error {
	startTime := time.Now()
	retryCount := 0

	// 更新统计信息
	r.stats.mu.Lock()
	r.stats.TotalOperations++
	r.stats.mu.Unlock()

	for {
		// 检查上下文是否已取消
		select {
		case <-r.ctx.Done():
			return fmt.Errorf("retry manager context cancelled")
		default:
		}

		// 执行操作
		err := operation.Execute()

		if err == nil {
			// 操作成功
			r.stats.mu.Lock()
			r.stats.SuccessfulOperations++
			r.stats.mu.Unlock()
			return nil
		}

		// 检查是否应该重试
		if !r.strategy.ShouldRetry(err, retryCount) {
			r.stats.mu.Lock()
			r.stats.FailedOperations++
			r.stats.mu.Unlock()
			return fmt.Errorf("operation failed after %d retries: %w", retryCount, err)
		}

		// 计算重试延迟
		delay := r.strategy.CalculateDelay(retryCount)

		// 记录重试
		r.stats.mu.Lock()
		r.stats.TotalRetries++
		r.stats.TotalRetryTime += delay
		r.stats.mu.Unlock()

		// 等待延迟时间
		select {
		case <-time.After(delay):
			// 继续重试
		case <-r.ctx.Done():
			return fmt.Errorf("retry manager context cancelled during retry")
		}

		retryCount++
	}
	log.Fatal("retry manager context cancelled", time.Since(startTime))
	return nil
}

// ExecuteWithTransaction 执行带事务的操作
func (r *RetryManager) ExecuteWithTransaction(operations []RetryableOperation) error {
	if !r.config.EnableTransaction {
		// 如果不启用事务，逐个执行
		for _, op := range operations {
			if err := r.ExecuteWithRetry(op); err != nil {
				return err
			}
		}
		return nil
	}

	// 创建事务上下文
	ctx, cancel := context.WithTimeout(r.ctx, r.config.TransactionTimeout)
	defer cancel()

	// 开始事务
	tx := &SimpleTransaction{
		operations: operations,
		ctx:        ctx,
	}

	return r.executeTransaction(tx)
}

// executeTransaction 执行事务
func (r *RetryManager) executeTransaction(tx *SimpleTransaction) error {
	// 开始事务
	if err := tx.Begin(); err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// 执行所有操作
	for _, op := range tx.operations {
		if err := op.Execute(); err != nil {
			// 回滚事务
			tx.Rollback()
			return fmt.Errorf("transaction failed, rolled back: %w", err)
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetStats 获取统计信息
func (r *RetryManager) GetStats() *RetryStats {
	r.stats.mu.RLock()
	defer r.stats.mu.RUnlock()

	// 计算平均值
	stats := *r.stats
	if stats.TotalOperations > 0 {
		stats.AverageRetries = float64(stats.TotalRetries) / float64(stats.TotalOperations)
	}
	if stats.TotalRetries > 0 {
		stats.AverageRetryTime = stats.TotalRetryTime / time.Duration(stats.TotalRetries)
	}

	return &stats
}

// ResetStats 重置统计信息
func (r *RetryManager) ResetStats() {
	r.stats.mu.Lock()
	defer r.stats.mu.Unlock()

	r.stats = &RetryStats{}
}

// Stop 停止重试管理器
func (r *RetryManager) Stop() {
	r.cancel()
}

// ExponentialBackoffStrategy 实现

// CalculateDelay 计算指数退避延迟
func (s *ExponentialBackoffStrategy) CalculateDelay(retryCount int) time.Duration {
	delay := float64(s.baseDelay) * math.Pow(s.multiplier, float64(retryCount))

	// 限制最大延迟
	if delay > float64(s.maxDelay) {
		delay = float64(s.maxDelay)
	}

	// 添加抖动
	if s.jitter {
		jitter := rand.Float64() * 0.1 * delay // 10% 抖动
		delay += jitter
	}

	return time.Duration(delay)
}

// ShouldRetry 判断是否应该重试
func (s *ExponentialBackoffStrategy) ShouldRetry(err error, retryCount int) bool {
	return retryCount < s.maxRetries
}

// GetMaxRetries 获取最大重试次数
func (s *ExponentialBackoffStrategy) GetMaxRetries() int {
	return s.maxRetries
}

// LinearBackoffStrategy 实现

// CalculateDelay 计算线性退避延迟
func (s *LinearBackoffStrategy) CalculateDelay(retryCount int) time.Duration {
	delay := s.baseDelay + time.Duration(retryCount)*s.step

	if delay > s.maxDelay {
		delay = s.maxDelay
	}

	return delay
}

// ShouldRetry 判断是否应该重试
func (s *LinearBackoffStrategy) ShouldRetry(err error, retryCount int) bool {
	return retryCount < s.maxRetries
}

// GetMaxRetries 获取最大重试次数
func (s *LinearBackoffStrategy) GetMaxRetries() int {
	return s.maxRetries
}

// FixedDelayStrategy 实现

// CalculateDelay 计算固定延迟
func (s *FixedDelayStrategy) CalculateDelay(retryCount int) time.Duration {
	return s.delay
}

// ShouldRetry 判断是否应该重试
func (s *FixedDelayStrategy) ShouldRetry(err error, retryCount int) bool {
	return retryCount < s.maxRetries
}

// GetMaxRetries 获取最大重试次数
func (s *FixedDelayStrategy) GetMaxRetries() int {
	return s.maxRetries
}

// Transaction 事务接口
type Transaction interface {
	Begin() error
	Commit() error
	Rollback() error
}

// SimpleTransaction 简单事务实现
type SimpleTransaction struct {
	operations []RetryableOperation
	ctx        context.Context
	committed  bool
	rolledBack bool
}

// Begin 开始事务
func (t *SimpleTransaction) Begin() error {
	// 简单实现，实际应该根据具体的数据源实现
	return nil
}

// Commit 提交事务
func (t *SimpleTransaction) Commit() error {
	if t.committed || t.rolledBack {
		return fmt.Errorf("transaction already completed")
	}
	t.committed = true
	return nil
}

// Rollback 回滚事务
func (t *SimpleTransaction) Rollback() error {
	if t.committed || t.rolledBack {
		return fmt.Errorf("transaction already completed")
	}
	t.rolledBack = true
	return nil
}
