package MetricAggregator

import (
	"fmt"
	"sync"
	"time"
)

// MetricsAggregator 指标聚合器接口
type MetricsAggregator interface {
	// AggregateByProcessorType 按组件类型聚合
	AggregateByProcessorType() (map[string]*AggregatedMetrics, error)

	// AggregateByProcessGroup 按流程组聚合
	AggregateByProcessGroup() (map[string]*AggregatedMetrics, error)

	// AggregateByTimeWindow 按时间窗口聚合
	AggregateByTimeWindow(window time.Duration) ([]*TimeWindowMetrics, error)

	// AggregateByCluster 集群环境聚合
	AggregateByCluster() (*ClusterMetricsSnapshot, error)

	// GetAggregatedMetrics 获取聚合指标
	GetAggregatedMetrics(aggregationType string) (interface{}, error)
}

// AggregatedMetrics 聚合结果封装
type AggregatedMetrics struct {
	TotalProcessingTime   float64                `json:"totalProcessingTime"`
	TotalProcessedRecords int64                  `json:"totalProcessedRecords"`
	TotalErrorCount       int64                  `json:"totalErrorCount"`
	AverageProcessingTime float64                `json:"averageProcessingTime"`
	Throughput            float64                `json:"throughput"`
	AggregatedValues      map[string]interface{} `json:"aggregatedValues"`
	Timestamp             time.Time              `json:"timestamp"`
}

// NewAggregatedMetrics 创建新的聚合指标
func NewAggregatedMetrics() *AggregatedMetrics {
	return &AggregatedMetrics{
		AggregatedValues: make(map[string]interface{}),
		Timestamp:        time.Now(),
	}
}

// AddMetrics 添加指标
func (am *AggregatedMetrics) AddMetrics(metrics map[string]interface{}) {
	for name, metric := range metrics {
		switch m := metric.(type) {
		case map[string]interface{}:
			if processingTime, ok := m["processingTime"].(float64); ok {
				am.TotalProcessingTime += processingTime
			}
			if count, ok := m["count"].(int64); ok {
				am.TotalProcessedRecords += count
			}
			if errors, ok := m["errors"].(int64); ok {
				am.TotalErrorCount += errors
			}
		}

		am.AggregatedValues[name] = metric
	}

	// 计算平均值
	if am.TotalProcessedRecords > 0 {
		am.AverageProcessingTime = am.TotalProcessingTime / float64(am.TotalProcessedRecords)
	}
}

// TimeWindowMetrics 时间窗口指标
type TimeWindowMetrics struct {
	WindowStart time.Time          `json:"windowStart"`
	WindowEnd   time.Time          `json:"windowEnd"`
	Metrics     *AggregatedMetrics `json:"metrics"`
	RecordCount int64              `json:"recordCount"`
}

// NewTimeWindowMetrics 创建新的时间窗口指标
func NewTimeWindowMetrics(start, end time.Time) *TimeWindowMetrics {
	return &TimeWindowMetrics{
		WindowStart: start,
		WindowEnd:   end,
		Metrics:     NewAggregatedMetrics(),
	}
}

// ClusterMetricsSnapshot 集群指标快照
type ClusterMetricsSnapshot struct {
	ClusterAverages map[string]interface{} `json:"clusterAverages"`
	NodeMetrics     []*NodeMetrics         `json:"nodeMetrics"`
	Timestamp       time.Time              `json:"timestamp"`
}

// NewClusterMetricsSnapshot 创建新的集群指标快照
func NewClusterMetricsSnapshot() *ClusterMetricsSnapshot {
	return &ClusterMetricsSnapshot{
		ClusterAverages: make(map[string]interface{}),
		NodeMetrics:     make([]*NodeMetrics, 0),
		Timestamp:       time.Now(),
	}
}

// NodeMetrics 节点指标
type NodeMetrics struct {
	NodeID           string                 `json:"nodeId"`
	NodeName         string                 `json:"nodeName"`
	ProcessingTime   float64                `json:"processingTime"`
	ProcessedRecords int64                  `json:"processedRecords"`
	ErrorCount       int64                  `json:"errorCount"`
	Metrics          map[string]interface{} `json:"metrics"`
	Timestamp        time.Time              `json:"timestamp"`
}

// NewNodeMetrics 创建新的节点指标
func NewNodeMetrics(nodeID, nodeName string) *NodeMetrics {
	return &NodeMetrics{
		NodeID:    nodeID,
		NodeName:  nodeName,
		Metrics:   make(map[string]interface{}),
		Timestamp: time.Now(),
	}
}

// StandardMetricsAggregator 标准指标聚合器实现
type StandardMetricsAggregator struct {
	metricCollector      MetricCollector
	processorRegistry    ProcessorRegistry
	processGroupRegistry ProcessGroupRegistry
	clusterManager       ClusterManager
	mutex                sync.RWMutex
}

// NewStandardMetricsAggregator 创建新的标准指标聚合器
func NewStandardMetricsAggregator(
	collector MetricCollector,
	processorRegistry ProcessorRegistry,
	processGroupRegistry ProcessGroupRegistry,
	clusterManager ClusterManager,
) *StandardMetricsAggregator {
	return &StandardMetricsAggregator{
		metricCollector:      collector,
		processorRegistry:    processorRegistry,
		processGroupRegistry: processGroupRegistry,
		clusterManager:       clusterManager,
	}
}

// AggregateByProcessorType 按组件类型聚合
func (sma *StandardMetricsAggregator) AggregateByProcessorType() (map[string]*AggregatedMetrics, error) {
	sma.mutex.RLock()
	defer sma.mutex.RUnlock()

	aggregations := make(map[string]*AggregatedMetrics)

	processors, err := sma.processorRegistry.GetAllProcessors()
	if err != nil {
		return nil, fmt.Errorf("获取处理器失败: %w", err)
	}

	for _, processor := range processors {
		processorType := processor.GetType()
		typeMetrics, exists := aggregations[processorType]
		if !exists {
			typeMetrics = NewAggregatedMetrics()
			aggregations[processorType] = typeMetrics
		}

		metrics := processor.GetMetrics()
		typeMetrics.AddMetrics(metrics)
	}

	return aggregations, nil
}

// AggregateByProcessGroup 按流程组聚合
func (sma *StandardMetricsAggregator) AggregateByProcessGroup() (map[string]*AggregatedMetrics, error) {
	sma.mutex.RLock()
	defer sma.mutex.RUnlock()

	aggregations := make(map[string]*AggregatedMetrics)

	groups, err := sma.processGroupRegistry.GetAllGroups()
	if err != nil {
		return nil, fmt.Errorf("获取流程组失败: %w", err)
	}

	for _, group := range groups {
		groupMetrics := NewAggregatedMetrics()

		processors, err := group.GetProcessors()
		if err != nil {
			continue
		}

		for _, processor := range processors {
			metrics := processor.GetMetrics()
			groupMetrics.AddMetrics(metrics)
		}

		aggregations[group.GetName()] = groupMetrics
	}

	return aggregations, nil
}

// AggregateByTimeWindow 按时间窗口聚合
func (sma *StandardMetricsAggregator) AggregateByTimeWindow(window time.Duration) ([]*TimeWindowMetrics, error) {
	sma.mutex.RLock()
	defer sma.mutex.RUnlock()

	// 获取当前时间
	now := time.Now()

	// 创建时间窗口
	var windows []*TimeWindowMetrics

	// 创建最近10个时间窗口
	for i := 9; i >= 0; i-- {
		windowEnd := now.Add(-time.Duration(i) * window)
		windowStart := windowEnd.Add(-window)

		windowMetrics := NewTimeWindowMetrics(windowStart, windowEnd)
		windows = append(windows, windowMetrics)
	}

	// 这里应该根据实际的时间序列数据填充窗口
	// 简化实现，返回空窗口
	return windows, nil
}

// AggregateByCluster 集群环境聚合
func (sma *StandardMetricsAggregator) AggregateByCluster() (*ClusterMetricsSnapshot, error) {
	sma.mutex.RLock()
	defer sma.mutex.RUnlock()

	nodeMetricsList := make([]*NodeMetrics, 0)

	nodes, err := sma.clusterManager.GetAllNodes()
	if err != nil {
		return nil, fmt.Errorf("获取集群节点失败: %w", err)
	}

	for _, node := range nodes {
		nodeMetrics := sma.collectNodeMetrics(node)
		nodeMetricsList = append(nodeMetricsList, nodeMetrics)
	}

	clusterSnapshot := NewClusterMetricsSnapshot()
	clusterSnapshot.NodeMetrics = nodeMetricsList

	// 计算集群级别的平均指标
	clusterSnapshot.ClusterAverages = sma.calculateClusterAverages(nodeMetricsList)

	return clusterSnapshot, nil
}

// GetAggregatedMetrics 获取聚合指标
func (sma *StandardMetricsAggregator) GetAggregatedMetrics(aggregationType string) (interface{}, error) {
	switch aggregationType {
	case "processor_type":
		return sma.AggregateByProcessorType()
	case "process_group":
		return sma.AggregateByProcessGroup()
	case "cluster":
		return sma.AggregateByCluster()
	default:
		return nil, fmt.Errorf("不支持的聚合类型: %s", aggregationType)
	}
}

// collectNodeMetrics 收集节点指标
func (sma *StandardMetricsAggregator) collectNodeMetrics(node NiFiNode) *NodeMetrics {
	nodeMetrics := NewNodeMetrics(node.GetID(), node.GetName())

	// 收集节点级别的指标
	snapshot := sma.metricCollector.GetMetricsSnapshot()

	// 处理Gauge指标
	for name, gauge := range snapshot.Gauges {
		nodeMetrics.Metrics[name] = gauge.GetValue()
	}

	// 处理Counter指标
	for name, counter := range snapshot.Counters {
		nodeMetrics.Metrics[name] = counter.GetCount()
	}

	// 处理Timer指标
	for name, timer := range snapshot.Timers {
		nodeMetrics.Metrics[name] = map[string]interface{}{
			"count": timer.GetCount(),
			"mean":  timer.GetMean().String(),
			"min":   timer.GetMin().String(),
			"max":   timer.GetMax().String(),
		}
	}

	// 计算处理时间（简化实现）
	if processingTime, ok := nodeMetrics.Metrics["processing.time"].(float64); ok {
		nodeMetrics.ProcessingTime = processingTime
	}

	// 计算处理记录数（简化实现）
	if recordCount, ok := nodeMetrics.Metrics["processed.records"].(int64); ok {
		nodeMetrics.ProcessedRecords = recordCount
	}

	// 计算错误数（简化实现）
	if errorCount, ok := nodeMetrics.Metrics["error.count"].(int64); ok {
		nodeMetrics.ErrorCount = errorCount
	}

	return nodeMetrics
}

// calculateClusterAverages 计算集群级别的平均指标
func (sma *StandardMetricsAggregator) calculateClusterAverages(nodeMetricsList []*NodeMetrics) map[string]interface{} {
	clusterAverages := make(map[string]interface{})

	if len(nodeMetricsList) == 0 {
		return clusterAverages
	}

	// 计算平均处理时间
	totalProcessingTime := 0.0
	for _, nodeMetrics := range nodeMetricsList {
		totalProcessingTime += nodeMetrics.ProcessingTime
	}
	clusterAverages["cluster.avg.processing.time"] = totalProcessingTime / float64(len(nodeMetricsList))

	// 计算总处理记录数
	totalProcessedRecords := int64(0)
	for _, nodeMetrics := range nodeMetricsList {
		totalProcessedRecords += nodeMetrics.ProcessedRecords
	}
	clusterAverages["cluster.total.processed.records"] = totalProcessedRecords

	// 计算总错误数
	totalErrorCount := int64(0)
	for _, nodeMetrics := range nodeMetricsList {
		totalErrorCount += nodeMetrics.ErrorCount
	}
	clusterAverages["cluster.total.error.count"] = totalErrorCount

	// 计算吞吐量
	if len(nodeMetricsList) > 0 {
		clusterAverages["cluster.throughput"] = float64(totalProcessedRecords) / float64(len(nodeMetricsList))
	}

	return clusterAverages
}

// ClusterMetricsAggregator 集群指标聚合器
type ClusterMetricsAggregator struct {
	clusterManager ClusterManager
	mutex          sync.RWMutex
}

// NewClusterMetricsAggregator 创建新的集群指标聚合器
func NewClusterMetricsAggregator(clusterManager ClusterManager) *ClusterMetricsAggregator {
	return &ClusterMetricsAggregator{
		clusterManager: clusterManager,
	}
}

// AggregateClusterMetrics 聚合集群指标
func (cma *ClusterMetricsAggregator) AggregateClusterMetrics() (*ClusterMetricsSnapshot, error) {
	cma.mutex.RLock()
	defer cma.mutex.RUnlock()

	nodeMetricsList := make([]*NodeMetrics, 0)

	nodes, err := cma.clusterManager.GetAllNodes()
	if err != nil {
		return nil, fmt.Errorf("获取集群节点失败: %w", err)
	}

	for _, node := range nodes {
		nodeMetrics := cma.collectNodeMetrics(node)
		nodeMetricsList = append(nodeMetricsList, nodeMetrics)
	}

	return &ClusterMetricsSnapshot{
		ClusterAverages: cma.calculateClusterAverages(nodeMetricsList),
		NodeMetrics:     nodeMetricsList,
		Timestamp:       time.Now(),
	}, nil
}

// collectNodeMetrics 收集节点指标
func (cma *ClusterMetricsAggregator) collectNodeMetrics(node NiFiNode) *NodeMetrics {
	nodeMetrics := NewNodeMetrics(node.GetID(), node.GetName())

	// 这里应该从节点收集实际的指标数据
	// 简化实现，使用模拟数据
	nodeMetrics.ProcessingTime = 100.0 + float64(len(node.GetID())*10)
	nodeMetrics.ProcessedRecords = 1000 + int64(len(node.GetID())*100)
	nodeMetrics.ErrorCount = int64(len(node.GetID()))

	return nodeMetrics
}

// calculateClusterAverages 计算集群平均指标
func (cma *ClusterMetricsAggregator) calculateClusterAverages(nodeMetricsList []*NodeMetrics) map[string]interface{} {
	clusterAverages := make(map[string]interface{})

	if len(nodeMetricsList) == 0 {
		return clusterAverages
	}

	// 计算平均处理时间
	totalProcessingTime := 0.0
	for _, nodeMetrics := range nodeMetricsList {
		totalProcessingTime += nodeMetrics.ProcessingTime
	}
	clusterAverages["cluster.avg.processing.time"] = totalProcessingTime / float64(len(nodeMetricsList))

	// 计算总处理记录数
	totalProcessedRecords := int64(0)
	for _, nodeMetrics := range nodeMetricsList {
		totalProcessedRecords += nodeMetrics.ProcessedRecords
	}
	clusterAverages["cluster.total.processed.records"] = totalProcessedRecords

	return clusterAverages
}

// 接口定义（简化版本）

// MetricCollector 指标收集器接口
type MetricCollector interface {
	GetMetricsSnapshot() *MetricsSnapshot
}

// MetricsSnapshot 指标快照
type MetricsSnapshot struct {
	Gauges   map[string]Gauge
	Counters map[string]Counter
	Timers   map[string]Timer
}

// Gauge 瞬时值指标接口
type Gauge interface {
	GetValue() float64
}

// Counter 累计值指标接口
type Counter interface {
	GetCount() int64
}

// Timer 耗时统计指标接口
type Timer interface {
	GetCount() int64
	GetMean() time.Duration
	GetMin() time.Duration
	GetMax() time.Duration
}

// ProcessorRegistry 处理器注册表接口
type ProcessorRegistry interface {
	GetAllProcessors() ([]Processor, error)
}

// Processor 处理器接口
type Processor interface {
	GetType() string
	GetMetrics() map[string]interface{}
}

// ProcessGroupRegistry 流程组注册表接口
type ProcessGroupRegistry interface {
	GetAllGroups() ([]ProcessGroup, error)
}

// ProcessGroup 流程组接口
type ProcessGroup interface {
	GetName() string
	GetProcessors() ([]Processor, error)
}

// ClusterManager 集群管理器接口
type ClusterManager interface {
	GetAllNodes() ([]NiFiNode, error)
}

// NiFiNode NiFi节点接口
type NiFiNode interface {
	GetID() string
	GetName() string
}
