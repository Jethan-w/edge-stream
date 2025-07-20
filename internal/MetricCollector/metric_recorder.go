package MetricCollector

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// MetricsRepository 指标存储接口
type MetricsRepository interface {
	// StoreSnapshot 存储指标快照
	StoreSnapshot(snapshot *MetricsSnapshot) error

	// QueryHistoricalMetrics 查询历史指标
	QueryHistoricalMetrics(metricName string, startTime, endTime time.Time) ([]*MetricsSnapshot, error)

	// PurgeExpiredMetrics 清理过期指标
	PurgeExpiredMetrics(retentionPeriod time.Duration) error

	// GetMetricsByTimeRange 按时间范围获取指标
	GetMetricsByTimeRange(startTime, endTime time.Time) ([]*MetricsSnapshot, error)

	// GetMetricsByType 按类型获取指标
	GetMetricsByType(metricType string) ([]*MetricsSnapshot, error)
}

// MetricsSnapshot 指标快照
type MetricsSnapshot struct {
	Timestamp  time.Time                `json:"timestamp"`
	Gauges     map[string]float64       `json:"gauges"`
	Counters   map[string]int64         `json:"counters"`
	Histograms map[string]HistogramData `json:"histograms"`
	Timers     map[string]TimerData     `json:"timers"`
	Meters     map[string]MeterData     `json:"meters"`
}

// HistogramData 直方图数据
type HistogramData struct {
	Count int64   `json:"count"`
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
	Mean  float64 `json:"mean"`
	Sum   float64 `json:"sum"`
}

// TimerData 计时器数据
type TimerData struct {
	Count int64         `json:"count"`
	Min   time.Duration `json:"min"`
	Max   time.Duration `json:"max"`
	Mean  time.Duration `json:"mean"`
	Sum   time.Duration `json:"sum"`
}

// MeterData 计量器数据
type MeterData struct {
	Metric
	Count             int64   `json:"count"`
	MeanRate          float64 `json:"meanRate"`
	OneMinuteRate     float64 `json:"oneMinuteRate"`
	FiveMinuteRate    float64 `json:"fiveMinuteRate"`
	FifteenMinuteRate float64 `json:"fifteenMinuteRate"`
}

// NewMetricsSnapshot 创建新的指标快照
func NewMetricsSnapshot() *MetricsSnapshot {
	return &MetricsSnapshot{
		Timestamp:  time.Now(),
		Gauges:     make(map[string]float64),
		Counters:   make(map[string]int64),
		Histograms: make(map[string]HistogramData),
		Timers:     make(map[string]TimerData),
		Meters:     make(map[string]MeterData),
	}
}

// TimeSeriesMetricsRepository 基于时序数据库的实现
type TimeSeriesMetricsRepository struct {
	storagePath string
	mutex       sync.RWMutex
}

// NewTimeSeriesMetricsRepository 创建新的时序指标存储库
func NewTimeSeriesMetricsRepository(storagePath string) *TimeSeriesMetricsRepository {
	return &TimeSeriesMetricsRepository{
		storagePath: storagePath,
	}
}

// StoreSnapshot 存储指标快照
func (tsmr *TimeSeriesMetricsRepository) StoreSnapshot(snapshot *MetricsSnapshot) error {
	tsmr.mutex.Lock()
	defer tsmr.mutex.Unlock()

	// 确保存储目录存在
	if err := os.MkdirAll(tsmr.storagePath, 0755); err != nil {
		return fmt.Errorf("创建存储目录失败: %w", err)
	}

	// 生成文件名（基于时间戳）
	filename := fmt.Sprintf("metrics_%d.json", snapshot.Timestamp.Unix())
	filepath := filepath.Join(tsmr.storagePath, filename)

	// 序列化快照
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化指标快照失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("写入指标文件失败: %w", err)
	}

	return nil
}

// QueryHistoricalMetrics 查询历史指标
func (tsmr *TimeSeriesMetricsRepository) QueryHistoricalMetrics(
	metricName string,
	startTime, endTime time.Time,
) ([]*MetricsSnapshot, error) {
	tsmr.mutex.RLock()
	defer tsmr.mutex.RUnlock()

	var snapshots []*MetricsSnapshot

	// 读取存储目录中的所有文件
	files, err := os.ReadDir(tsmr.storagePath)
	if err != nil {
		return nil, fmt.Errorf("读取存储目录失败: %w", err)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			filepath := filepath.Join(tsmr.storagePath, file.Name())

			// 读取文件内容
			data, err := os.ReadFile(filepath)
			if err != nil {
				continue
			}

			// 反序列化快照
			var snapshot MetricsSnapshot
			if err := json.Unmarshal(data, &snapshot); err != nil {
				continue
			}

			// 检查时间范围
			if snapshot.Timestamp.After(startTime) && snapshot.Timestamp.Before(endTime) {
				// 检查是否包含指定的指标
				if tsmr.containsMetric(&snapshot, metricName) {
					snapshots = append(snapshots, &snapshot)
				}
			}
		}
	}

	return snapshots, nil
}

// PurgeExpiredMetrics 清理过期指标
func (tsmr *TimeSeriesMetricsRepository) PurgeExpiredMetrics(retentionPeriod time.Duration) error {
	tsmr.mutex.Lock()
	defer tsmr.mutex.Unlock()

	cutoffTime := time.Now().Add(-retentionPeriod)

	// 读取存储目录中的所有文件
	files, err := os.ReadDir(tsmr.storagePath)
	if err != nil {
		return fmt.Errorf("读取存储目录失败: %w", err)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			filepath := filepath.Join(tsmr.storagePath, file.Name())

			// 读取文件内容
			data, err := os.ReadFile(filepath)
			if err != nil {
				continue
			}

			// 反序列化快照
			var snapshot MetricsSnapshot
			if err := json.Unmarshal(data, &snapshot); err != nil {
				continue
			}

			// 检查是否过期
			if snapshot.Timestamp.Before(cutoffTime) {
				// 删除过期文件
				if err := os.Remove(filepath); err != nil {
					fmt.Printf("删除过期文件失败: %s, 错误: %v\n", filepath, err)
				}
			}
		}
	}

	return nil
}

// GetMetricsByTimeRange 按时间范围获取指标
func (tsmr *TimeSeriesMetricsRepository) GetMetricsByTimeRange(
	startTime, endTime time.Time,
) ([]*MetricsSnapshot, error) {
	tsmr.mutex.RLock()
	defer tsmr.mutex.RUnlock()

	var snapshots []*MetricsSnapshot

	// 读取存储目录中的所有文件
	files, err := os.ReadDir(tsmr.storagePath)
	if err != nil {
		return nil, fmt.Errorf("读取存储目录失败: %w", err)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			filepath := filepath.Join(tsmr.storagePath, file.Name())

			// 读取文件内容
			data, err := os.ReadFile(filepath)
			if err != nil {
				continue
			}

			// 反序列化快照
			var snapshot MetricsSnapshot
			if err := json.Unmarshal(data, &snapshot); err != nil {
				continue
			}

			// 检查时间范围
			if snapshot.Timestamp.After(startTime) && snapshot.Timestamp.Before(endTime) {
				snapshots = append(snapshots, &snapshot)
			}
		}
	}

	return snapshots, nil
}

// GetMetricsByType 按类型获取指标
func (tsmr *TimeSeriesMetricsRepository) GetMetricsByType(metricType string) ([]*MetricsSnapshot, error) {
	tsmr.mutex.RLock()
	defer tsmr.mutex.RUnlock()

	var snapshots []*MetricsSnapshot

	// 读取存储目录中的所有文件
	files, err := os.ReadDir(tsmr.storagePath)
	if err != nil {
		return nil, fmt.Errorf("读取存储目录失败: %w", err)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			filepath := filepath.Join(tsmr.storagePath, file.Name())

			// 读取文件内容
			data, err := os.ReadFile(filepath)
			if err != nil {
				continue
			}

			// 反序列化快照
			var snapshot MetricsSnapshot
			if err := json.Unmarshal(data, &snapshot); err != nil {
				continue
			}

			// 检查是否包含指定类型的指标
			if tsmr.containsMetricType(&snapshot, metricType) {
				snapshots = append(snapshots, &snapshot)
			}
		}
	}

	return snapshots, nil
}

// containsMetric 检查快照是否包含指定指标
func (tsmr *TimeSeriesMetricsRepository) containsMetric(snapshot *MetricsSnapshot, metricName string) bool {
	// 检查Gauge指标
	if _, exists := snapshot.Gauges[metricName]; exists {
		return true
	}

	// 检查Counter指标
	if _, exists := snapshot.Counters[metricName]; exists {
		return true
	}

	// 检查Histogram指标
	if _, exists := snapshot.Histograms[metricName]; exists {
		return true
	}

	// 检查Timer指标
	if _, exists := snapshot.Timers[metricName]; exists {
		return true
	}

	// 检查Meter指标
	if _, exists := snapshot.Meters[metricName]; exists {
		return true
	}

	return false
}

// containsMetricType 检查快照是否包含指定类型的指标
func (tsmr *TimeSeriesMetricsRepository) containsMetricType(snapshot *MetricsSnapshot, metricType string) bool {
	switch metricType {
	case "gauge":
		return len(snapshot.Gauges) > 0
	case "counter":
		return len(snapshot.Counters) > 0
	case "histogram":
		return len(snapshot.Histograms) > 0
	case "timer":
		return len(snapshot.Timers) > 0
	case "meter":
		return len(snapshot.Meters) > 0
	default:
		return false
	}
}

// MetricsArchivalStrategy 指标归档策略
type MetricsArchivalStrategy struct {
	repository MetricsRepository
	mutex      sync.RWMutex
}

// NewMetricsArchivalStrategy 创建新的指标归档策略
func NewMetricsArchivalStrategy(repository MetricsRepository) *MetricsArchivalStrategy {
	return &MetricsArchivalStrategy{
		repository: repository,
	}
}

// ArchiveMetrics 归档指标
func (mas *MetricsArchivalStrategy) ArchiveMetrics() error {
	mas.mutex.Lock()
	defer mas.mutex.Unlock()

	// 按不同精度归档
	if err := mas.archiveHighResolutionMetrics(); err != nil {
		return fmt.Errorf("归档高精度指标失败: %w", err)
	}

	if err := mas.archiveLowResolutionMetrics(); err != nil {
		return fmt.Errorf("归档低精度指标失败: %w", err)
	}

	return nil
}

// archiveHighResolutionMetrics 归档高精度指标
func (mas *MetricsArchivalStrategy) archiveHighResolutionMetrics() error {
	// 保留最近7天的高精度指标
	retentionPeriod := 7 * 24 * time.Hour
	return mas.repository.PurgeExpiredMetrics(retentionPeriod)
}

// archiveLowResolutionMetrics 归档低精度指标
func (mas *MetricsArchivalStrategy) archiveLowResolutionMetrics() error {
	// 长期保留低精度聚合指标（30天）
	retentionPeriod := 30 * 24 * time.Hour
	return mas.repository.PurgeExpiredMetrics(retentionPeriod)
}

// MetricsStorageManager 指标存储管理器
type MetricsStorageManager struct {
	repository       MetricsRepository
	archivalStrategy *MetricsArchivalStrategy
	mutex            sync.RWMutex
}

// NewMetricsStorageManager 创建新的指标存储管理器
func NewMetricsStorageManager(repository MetricsRepository) *MetricsStorageManager {
	return &MetricsStorageManager{
		repository:       repository,
		archivalStrategy: NewMetricsArchivalStrategy(repository),
	}
}

// StoreMetrics 存储指标
func (msm *MetricsStorageManager) StoreMetrics(snapshot *MetricsSnapshot) error {
	msm.mutex.Lock()
	defer msm.mutex.Unlock()

	return msm.repository.StoreSnapshot(snapshot)
}

// QueryMetrics 查询指标
func (msm *MetricsStorageManager) QueryMetrics(
	metricName string,
	startTime, endTime time.Time,
) ([]*MetricsSnapshot, error) {
	msm.mutex.RLock()
	defer msm.mutex.RUnlock()

	return msm.repository.QueryHistoricalMetrics(metricName, startTime, endTime)
}

// ArchiveMetrics 归档指标
func (msm *MetricsStorageManager) ArchiveMetrics() error {
	msm.mutex.Lock()
	defer msm.mutex.Unlock()

	return msm.archivalStrategy.ArchiveMetrics()
}

// GetMetricsSummary 获取指标摘要
func (msm *MetricsStorageManager) GetMetricsSummary(
	startTime, endTime time.Time,
) (*MetricsSummary, error) {
	msm.mutex.RLock()
	defer msm.mutex.RUnlock()

	snapshots, err := msm.repository.GetMetricsByTimeRange(startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("获取指标失败: %w", err)
	}

	summary := NewMetricsSummary()

	for _, snapshot := range snapshots {
		summary.AddSnapshot(snapshot)
	}

	return summary, nil
}

// MetricsSummary 指标摘要
type MetricsSummary struct {
	TotalSnapshots   int64                       `json:"totalSnapshots"`
	TimeRange        TimeRange                   `json:"timeRange"`
	GaugeSummary     map[string]GaugeSummary     `json:"gaugeSummary"`
	CounterSummary   map[string]CounterSummary   `json:"counterSummary"`
	HistogramSummary map[string]HistogramSummary `json:"histogramSummary"`
	TimerSummary     map[string]TimerSummary     `json:"timerSummary"`
	MeterSummary     map[string]MeterSummary     `json:"meterSummary"`
}

// TimeRange 时间范围
type TimeRange struct {
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
}

// GaugeSummary Gauge指标摘要
type GaugeSummary struct {
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	Average float64 `json:"average"`
	Count   int64   `json:"count"`
}

// CounterSummary Counter指标摘要
type CounterSummary struct {
	StartValue int64 `json:"startValue"`
	EndValue   int64 `json:"endValue"`
	Total      int64 `json:"total"`
}

// HistogramSummary Histogram指标摘要
type HistogramSummary struct {
	TotalCount int64   `json:"totalCount"`
	Min        float64 `json:"min"`
	Max        float64 `json:"max"`
	Average    float64 `json:"average"`
	P95        float64 `json:"p95"`
	P99        float64 `json:"p99"`
}

// TimerSummary Timer指标摘要
type TimerSummary struct {
	TotalCount int64         `json:"totalCount"`
	Min        time.Duration `json:"min"`
	Max        time.Duration `json:"max"`
	Average    time.Duration `json:"average"`
	P95        time.Duration `json:"p95"`
	P99        time.Duration `json:"p99"`
}

// MeterSummary Meter指标摘要
type MeterSummary struct {
	TotalCount int64   `json:"totalCount"`
	MeanRate   float64 `json:"meanRate"`
	MaxRate    float64 `json:"maxRate"`
}

// NewMetricsSummary 创建新的指标摘要
func NewMetricsSummary() *MetricsSummary {
	return &MetricsSummary{
		GaugeSummary:     make(map[string]GaugeSummary),
		CounterSummary:   make(map[string]CounterSummary),
		HistogramSummary: make(map[string]HistogramSummary),
		TimerSummary:     make(map[string]TimerSummary),
		MeterSummary:     make(map[string]MeterSummary),
	}
}

// AddSnapshot 添加快照到摘要
func (ms *MetricsSummary) AddSnapshot(snapshot *MetricsSnapshot) {
	ms.TotalSnapshots++

	// 更新时间范围
	if ms.TimeRange.StartTime.IsZero() || snapshot.Timestamp.Before(ms.TimeRange.StartTime) {
		ms.TimeRange.StartTime = snapshot.Timestamp
	}
	if ms.TimeRange.EndTime.IsZero() || snapshot.Timestamp.After(ms.TimeRange.EndTime) {
		ms.TimeRange.EndTime = snapshot.Timestamp
	}

	// 处理Gauge指标
	for name, value := range snapshot.Gauges {
		summary, exists := ms.GaugeSummary[name]
		if !exists {
			summary = GaugeSummary{
				Min: value,
				Max: value,
			}
		}

		if value < summary.Min {
			summary.Min = value
		}
		if value > summary.Max {
			summary.Max = value
		}
		summary.Average = (summary.Average*float64(summary.Count) + value) / float64(summary.Count+1)
		summary.Count++

		ms.GaugeSummary[name] = summary
	}

	// 处理Counter指标
	for name, value := range snapshot.Counters {
		summary, exists := ms.CounterSummary[name]
		if !exists {
			summary.StartValue = value
		}
		summary.EndValue = value
		summary.Total = summary.EndValue - summary.StartValue

		ms.CounterSummary[name] = summary
	}

	// 处理Histogram指标
	for name, data := range snapshot.Histograms {
		summary, exists := ms.HistogramSummary[name]
		if !exists {
			summary = HistogramSummary{
				Min: data.Min,
				Max: data.Max,
			}
		}

		summary.TotalCount += data.Count
		if data.Min < summary.Min {
			summary.Min = data.Min
		}
		if data.Max > summary.Max {
			summary.Max = data.Max
		}
		summary.Average = (summary.Average*float64(summary.TotalCount-data.Count) + data.Sum) / float64(summary.TotalCount)

		ms.HistogramSummary[name] = summary
	}

	// 处理Timer指标
	for name, data := range snapshot.Timers {
		summary, exists := ms.TimerSummary[name]
		if !exists {
			summary = TimerSummary{
				Min: data.Min,
				Max: data.Max,
			}
		}

		summary.TotalCount += data.Count
		if data.Min < summary.Min {
			summary.Min = data.Min
		}
		if data.Max > summary.Max {
			summary.Max = data.Max
		}
		summary.Average = (summary.Average*time.Duration(summary.TotalCount-data.Count) + data.Sum) / time.Duration(summary.TotalCount)

		ms.TimerSummary[name] = summary
	}

	// 处理Meter指标
	for name, data := range snapshot.Meters {
		summary, exists := ms.MeterSummary[name]
		if !exists {
			summary = MeterSummary{
				MaxRate: data.MeanRate,
			}
		}

		summary.TotalCount += data.Count
		summary.MeanRate = (summary.MeanRate*float64(summary.TotalCount-data.Count) + data.MeanRate) / float64(summary.TotalCount)
		if data.MeanRate > summary.MaxRate {
			summary.MaxRate = data.MeanRate
		}

		ms.MeterSummary[name] = summary
	}
}
