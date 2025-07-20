package windowmanager

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"sync"

	"github.com/crazy/edge-stream/internal/flowfile"
)

// WindowAggregator 窗口聚合器接口
type WindowAggregator interface {
	// Aggregate 聚合单个 FlowFile
	Aggregate(flowFile *flowfile.FlowFile) error

	// GetResult 获取聚合结果
	GetResult() interface{}

	// Reset 重置聚合状态
	Reset()

	// GetAggregationFunction 获取聚合函数类型
	GetAggregationFunction() AggregationFunction
}

// AggregationStrategy 聚合策略接口
type AggregationStrategy interface {
	// AddAggregator 添加聚合器
	AddAggregator(aggregator WindowAggregator)

	// ComputeAggregations 计算聚合结果
	ComputeAggregations(flowFiles []*flowfile.FlowFile) (map[string]interface{}, error)

	// Reset 重置所有聚合器
	Reset()
}

// AverageAggregator 平均值聚合器
type AverageAggregator struct {
	sum              float64
	count            int
	aggregationField string
	mu               sync.RWMutex
}

// NewAverageAggregator 创建平均值聚合器
func NewAverageAggregator(aggregationField string) *AverageAggregator {
	return &AverageAggregator{
		aggregationField: aggregationField,
	}
}

// Aggregate 聚合单个 FlowFile
func (aa *AverageAggregator) Aggregate(flowFile *flowfile.FlowFile) error {
	aa.mu.Lock()
	defer aa.mu.Unlock()

	value, err := aa.extractValue(flowFile)
	if err != nil {
		return fmt.Errorf("failed to extract value for aggregation: %w", err)
	}

	aa.sum += value
	aa.count++

	return nil
}

// GetResult 获取聚合结果
func (aa *AverageAggregator) GetResult() interface{} {
	aa.mu.RLock()
	defer aa.mu.RUnlock()

	if aa.count > 0 {
		return aa.sum / float64(aa.count)
	}
	return 0.0
}

// Reset 重置聚合状态
func (aa *AverageAggregator) Reset() {
	aa.mu.Lock()
	defer aa.mu.Unlock()

	aa.sum = 0
	aa.count = 0
}

// GetAggregationFunction 获取聚合函数类型
func (aa *AverageAggregator) GetAggregationFunction() AggregationFunction {
	return AggregationFunctionAvg
}

// extractValue 提取数值
func (aa *AverageAggregator) extractValue(flowFile *flowfile.FlowFile) (float64, error) {
	content := flowFile.Content

	var jsonData map[string]interface{}
	if err := json.Unmarshal(content, &jsonData); err != nil {
		// 尝试从属性中获取
		if valueStr := flowFile.Attributes[aa.aggregationField]; valueStr != "" {
			if value, err := strconv.ParseFloat(valueStr, 64); err == nil {
				return value, nil
			}
		}
		return 0, fmt.Errorf("failed to parse JSON content")
	}

	value, exists := jsonData[aa.aggregationField]
	if !exists {
		return 0, fmt.Errorf("field %s not found", aa.aggregationField)
	}

	switch v := value.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		if floatValue, err := strconv.ParseFloat(v, 64); err == nil {
			return floatValue, nil
		}
		return 0, fmt.Errorf("failed to parse string value: %s", v)
	default:
		return 0, fmt.Errorf("unsupported value type: %T", value)
	}
}

// SumAggregator 求和聚合器
type SumAggregator struct {
	sum              float64
	aggregationField string
	mu               sync.RWMutex
}

// NewSumAggregator 创建求和聚合器
func NewSumAggregator(aggregationField string) *SumAggregator {
	return &SumAggregator{
		aggregationField: aggregationField,
	}
}

// Aggregate 聚合单个 FlowFile
func (sa *SumAggregator) Aggregate(flowFile *flowfile.FlowFile) error {
	sa.mu.Lock()
	defer sa.mu.Unlock()

	value, err := sa.extractValue(flowFile)
	if err != nil {
		return fmt.Errorf("failed to extract value for aggregation: %w", err)
	}

	sa.sum += value
	return nil
}

// GetResult 获取聚合结果
func (sa *SumAggregator) GetResult() interface{} {
	sa.mu.RLock()
	defer sa.mu.RUnlock()
	return sa.sum
}

// Reset 重置聚合状态
func (sa *SumAggregator) Reset() {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	sa.sum = 0
}

// GetAggregationFunction 获取聚合函数类型
func (sa *SumAggregator) GetAggregationFunction() AggregationFunction {
	return AggregationFunctionSum
}

// extractValue 提取数值
func (sa *SumAggregator) extractValue(flowFile *flowfile.FlowFile) (float64, error) {
	content := flowFile.Content
	var jsonData map[string]interface{}
	if err := json.Unmarshal(content, &jsonData); err != nil {
		// 尝试从属性中获取
		if valueStr := flowFile.Attributes[sa.aggregationField]; valueStr != "" {
			if value, err := strconv.ParseFloat(valueStr, 64); err == nil {
				return value, nil
			}
		}
		return 0, fmt.Errorf("failed to parse JSON content")
	}

	value, exists := jsonData[sa.aggregationField]
	if !exists {
		return 0, fmt.Errorf("field %s not found", sa.aggregationField)
	}

	switch v := value.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		if floatValue, err := strconv.ParseFloat(v, 64); err == nil {
			return floatValue, nil
		}
		return 0, fmt.Errorf("failed to parse string value: %s", v)
	default:
		return 0, fmt.Errorf("unsupported value type: %T", value)
	}
}

// MaxAggregator 最大值聚合器
type MaxAggregator struct {
	max              float64
	aggregationField string
	initialized      bool
	mu               sync.RWMutex
}

// NewMaxAggregator 创建最大值聚合器
func NewMaxAggregator(aggregationField string) *MaxAggregator {
	return &MaxAggregator{
		aggregationField: aggregationField,
		max:              math.Inf(-1),
	}
}

// Aggregate 聚合单个 FlowFile
func (ma *MaxAggregator) Aggregate(flowFile *flowfile.FlowFile) error {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	value, err := ma.extractValue(flowFile)
	if err != nil {
		return fmt.Errorf("failed to extract value for aggregation: %w", err)
	}

	if !ma.initialized || value > ma.max {
		ma.max = value
		ma.initialized = true
	}

	return nil
}

// GetResult 获取聚合结果
func (ma *MaxAggregator) GetResult() interface{} {
	ma.mu.RLock()
	defer ma.mu.RUnlock()

	if !ma.initialized {
		return nil
	}
	return ma.max
}

// Reset 重置聚合状态
func (ma *MaxAggregator) Reset() {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	ma.max = math.Inf(-1)
	ma.initialized = false
}

// GetAggregationFunction 获取聚合函数类型
func (ma *MaxAggregator) GetAggregationFunction() AggregationFunction {
	return AggregationFunctionMax
}

// extractValue 提取数值
func (ma *MaxAggregator) extractValue(flowFile *flowfile.FlowFile) (float64, error) {
	content := flowFile.Content

	var jsonData map[string]interface{}
	if err := json.Unmarshal(content, &jsonData); err != nil {
		// 尝试从属性中获取
		if valueStr := flowFile.Attributes[ma.aggregationField]; valueStr != "" {
			if value, err := strconv.ParseFloat(valueStr, 64); err == nil {
				return value, nil
			}
		}
		return 0, fmt.Errorf("failed to parse JSON content")
	}

	value, exists := jsonData[ma.aggregationField]
	if !exists {
		return 0, fmt.Errorf("field %s not found", ma.aggregationField)
	}

	switch v := value.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		if floatValue, err := strconv.ParseFloat(v, 64); err == nil {
			return floatValue, nil
		}
		return 0, fmt.Errorf("failed to parse string value: %s", v)
	default:
		return 0, fmt.Errorf("unsupported value type: %T", value)
	}
}

// MinAggregator 最小值聚合器
type MinAggregator struct {
	min              float64
	aggregationField string
	initialized      bool
	mu               sync.RWMutex
}

// NewMinAggregator 创建最小值聚合器
func NewMinAggregator(aggregationField string) *MinAggregator {
	return &MinAggregator{
		aggregationField: aggregationField,
		min:              math.Inf(1),
	}
}

// Aggregate 聚合单个 FlowFile
func (ma *MinAggregator) Aggregate(flowFile *flowfile.FlowFile) error {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	value, err := ma.extractValue(flowFile)
	if err != nil {
		return fmt.Errorf("failed to extract value for aggregation: %w", err)
	}

	if !ma.initialized || value < ma.min {
		ma.min = value
		ma.initialized = true
	}

	return nil
}

// GetResult 获取聚合结果
func (ma *MinAggregator) GetResult() interface{} {
	ma.mu.RLock()
	defer ma.mu.RUnlock()

	if !ma.initialized {
		return nil
	}
	return ma.min
}

// Reset 重置聚合状态
func (ma *MinAggregator) Reset() {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	ma.min = math.Inf(1)
	ma.initialized = false
}

// GetAggregationFunction 获取聚合函数类型
func (ma *MinAggregator) GetAggregationFunction() AggregationFunction {
	return AggregationFunctionMin
}

// extractValue 提取数值
func (ma *MinAggregator) extractValue(flowFile *flowfile.FlowFile) (float64, error) {
	content := flowFile.Content

	var jsonData map[string]interface{}
	if err := json.Unmarshal(content, &jsonData); err != nil {
		// 尝试从属性中获取
		if valueStr := flowFile.Attributes[ma.aggregationField]; valueStr != "" {
			if value, err := strconv.ParseFloat(valueStr, 64); err == nil {
				return value, nil
			}
		}
		return 0, fmt.Errorf("failed to parse JSON content")
	}

	value, exists := jsonData[ma.aggregationField]
	if !exists {
		return 0, fmt.Errorf("field %s not found", ma.aggregationField)
	}

	switch v := value.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		if floatValue, err := strconv.ParseFloat(v, 64); err == nil {
			return floatValue, nil
		}
		return 0, fmt.Errorf("failed to parse string value: %s", v)
	default:
		return 0, fmt.Errorf("unsupported value type: %T", value)
	}
}

// CountAggregator 计数聚合器
type CountAggregator struct {
	count int
	mu    sync.RWMutex
}

// NewCountAggregator 创建计数聚合器
func NewCountAggregator() *CountAggregator {
	return &CountAggregator{}
}

// Aggregate 聚合单个 FlowFile
func (ca *CountAggregator) Aggregate(flowFile *flowfile.FlowFile) error {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	ca.count++
	return nil
}

// GetResult 获取聚合结果
func (ca *CountAggregator) GetResult() interface{} {
	ca.mu.RLock()
	defer ca.mu.RUnlock()
	return ca.count
}

// Reset 重置聚合状态
func (ca *CountAggregator) Reset() {
	ca.mu.Lock()
	defer ca.mu.Unlock()
	ca.count = 0
}

// GetAggregationFunction 获取聚合函数类型
func (ca *CountAggregator) GetAggregationFunction() AggregationFunction {
	return AggregationFunctionCount
}

// FirstAggregator 第一个值聚合器
type FirstAggregator struct {
	first            interface{}
	aggregationField string
	initialized      bool
	mu               sync.RWMutex
}

// NewFirstAggregator 创建第一个值聚合器
func NewFirstAggregator(aggregationField string) *FirstAggregator {
	return &FirstAggregator{
		aggregationField: aggregationField,
	}
}

// Aggregate 聚合单个 FlowFile
func (fa *FirstAggregator) Aggregate(flowFile *flowfile.FlowFile) error {
	fa.mu.Lock()
	defer fa.mu.Unlock()

	if fa.initialized {
		return nil // 只取第一个值
	}

	value, err := fa.extractValue(flowFile)
	if err != nil {
		return fmt.Errorf("failed to extract value for aggregation: %w", err)
	}

	fa.first = value
	fa.initialized = true
	return nil
}

// GetResult 获取聚合结果
func (fa *FirstAggregator) GetResult() interface{} {
	fa.mu.RLock()
	defer fa.mu.RUnlock()
	return fa.first
}

// Reset 重置聚合状态
func (fa *FirstAggregator) Reset() {
	fa.mu.Lock()
	defer fa.mu.Unlock()

	fa.first = nil
	fa.initialized = false
}

// GetAggregationFunction 获取聚合函数类型
func (fa *FirstAggregator) GetAggregationFunction() AggregationFunction {
	return AggregationFunctionFirst
}

// extractValue 提取数值
func (fa *FirstAggregator) extractValue(flowFile *flowfile.FlowFile) (interface{}, error) {
	content := flowFile.Content
	var jsonData map[string]interface{}
	if err := json.Unmarshal(content, &jsonData); err != nil {
		// 尝试从属性中获取
		if valueStr := flowFile.Attributes[fa.aggregationField]; valueStr != "" {
			return valueStr, nil
		}
		return nil, fmt.Errorf("failed to parse JSON content")
	}

	value, exists := jsonData[fa.aggregationField]
	if !exists {
		return nil, fmt.Errorf("field %s not found", fa.aggregationField)
	}

	return value, nil
}

// LastAggregator 最后一个值聚合器
type LastAggregator struct {
	last             interface{}
	aggregationField string
	mu               sync.RWMutex
}

// NewLastAggregator 创建最后一个值聚合器
func NewLastAggregator(aggregationField string) *LastAggregator {
	return &LastAggregator{
		aggregationField: aggregationField,
	}
}

// Aggregate 聚合单个 FlowFile
func (la *LastAggregator) Aggregate(flowFile *flowfile.FlowFile) error {
	la.mu.Lock()
	defer la.mu.Unlock()

	value, err := la.extractValue(flowFile)
	if err != nil {
		return fmt.Errorf("failed to extract value for aggregation: %w", err)
	}

	la.last = value
	return nil
}

// GetResult 获取聚合结果
func (la *LastAggregator) GetResult() interface{} {
	la.mu.RLock()
	defer la.mu.RUnlock()
	return la.last
}

// Reset 重置聚合状态
func (la *LastAggregator) Reset() {
	la.mu.Lock()
	defer la.mu.Unlock()
	la.last = nil
}

// GetAggregationFunction 获取聚合函数类型
func (la *LastAggregator) GetAggregationFunction() AggregationFunction {
	return AggregationFunctionLast
}

// extractValue 提取数值
func (la *LastAggregator) extractValue(flowFile *flowfile.FlowFile) (interface{}, error) {
	content := flowFile.Content

	var jsonData map[string]interface{}
	if err := json.Unmarshal(content, &jsonData); err != nil {
		// 尝试从属性中获取
		if valueStr := flowFile.Attributes[la.aggregationField]; valueStr != "" {
			return valueStr, nil
		}
		return nil, fmt.Errorf("failed to parse JSON content")
	}

	value, exists := jsonData[la.aggregationField]
	if !exists {
		return nil, fmt.Errorf("field %s not found", la.aggregationField)
	}

	return value, nil
}

// MultiAggregationStrategy 多聚合策略
type MultiAggregationStrategy struct {
	aggregators []WindowAggregator
	mu          sync.RWMutex
}

func (mas *MultiAggregationStrategy) Aggregate(flowFile *flowfile.FlowFile) error {
	// TODO implement me
	panic("implement me")
}

func (mas *MultiAggregationStrategy) GetResult() interface{} {
	// TODO implement me
	panic("implement me")
}

func (mas *MultiAggregationStrategy) GetAggregationFunction() AggregationFunction {
	// TODO implement me
	panic("implement me")
}

// NewMultiAggregationStrategy 创建多聚合策略
func NewMultiAggregationStrategy() *MultiAggregationStrategy {
	return &MultiAggregationStrategy{
		aggregators: make([]WindowAggregator, 0),
	}
}

// AddAggregator 添加聚合器
func (mas *MultiAggregationStrategy) AddAggregator(aggregator WindowAggregator) {
	mas.mu.Lock()
	defer mas.mu.Unlock()

	mas.aggregators = append(mas.aggregators, aggregator)
}

// ComputeAggregations 计算聚合结果
func (mas *MultiAggregationStrategy) ComputeAggregations(flowFiles []*flowfile.FlowFile) (map[string]interface{}, error) {
	mas.mu.RLock()
	defer mas.mu.RUnlock()

	results := make(map[string]interface{})

	for _, aggregator := range mas.aggregators {
		// 重置聚合器
		aggregator.Reset()

		// 对每个 FlowFile 进行聚合
		for _, flowFile := range flowFiles {
			if err := aggregator.Aggregate(flowFile); err != nil {
				return nil, fmt.Errorf("failed to aggregate flow file: %w", err)
			}
		}

		// 获取聚合结果
		functionName := string(aggregator.GetAggregationFunction())
		results[functionName] = aggregator.GetResult()
	}

	return results, nil
}

// Reset 重置所有聚合器
func (mas *MultiAggregationStrategy) Reset() {
	mas.mu.Lock()
	defer mas.mu.Unlock()

	for _, aggregator := range mas.aggregators {
		aggregator.Reset()
	}
}

// AggregatorFactory 聚合器工厂
type AggregatorFactory struct{}

// NewAggregatorFactory 创建聚合器工厂
func NewAggregatorFactory() *AggregatorFactory {
	return &AggregatorFactory{}
}

// CreateAggregator 创建聚合器
func (af *AggregatorFactory) CreateAggregator(function AggregationFunction, field string) (WindowAggregator, error) {
	switch function {
	case AggregationFunctionCount:
		return NewCountAggregator(), nil
	case AggregationFunctionSum:
		return NewSumAggregator(field), nil
	case AggregationFunctionAvg:
		return NewAverageAggregator(field), nil
	case AggregationFunctionMax:
		return NewMaxAggregator(field), nil
	case AggregationFunctionMin:
		return NewMinAggregator(field), nil
	case AggregationFunctionFirst:
		return NewFirstAggregator(field), nil
	case AggregationFunctionLast:
		return NewLastAggregator(field), nil
	default:
		return nil, fmt.Errorf("unsupported aggregation function: %s", function)
	}
}
