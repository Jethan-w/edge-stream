package provenance

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/crazy/edge-stream/internal/pipeline"
)

// ProvenanceTracker 数据溯源跟踪器
type ProvenanceTracker struct {
	repository *ProvenanceRepository
	mu         sync.RWMutex
}

// NewProvenanceTracker 创建数据溯源跟踪器
func NewProvenanceTracker() *ProvenanceTracker {
	return &ProvenanceTracker{
		repository: NewProvenanceRepository(),
	}
}

// Initialize 初始化溯源跟踪器
func (pt *ProvenanceTracker) Initialize(ctx context.Context) error {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	// 初始化溯源仓库
	if err := pt.repository.Initialize(ctx); err != nil {
		return fmt.Errorf("初始化溯源仓库失败: %w", err)
	}

	return nil
}

// RecordEvent 记录溯源事件
func (pt *ProvenanceTracker) RecordEvent(flowFile *pipeline.FlowFile, eventType ProvenanceEventType, processorID string, details map[string]string) error {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	// 创建溯源事件
	event := &ProvenanceEvent{
		EventID:     generateEventID(),
		FlowFileID:  flowFile.UUID,
		LineageID:   flowFile.LineageID,
		EventType:   eventType,
		ProcessorID: processorID,
		Timestamp:   time.Now(),
		Details:     details,
	}

	// 存储事件
	return pt.repository.Store(event)
}

// TraceFlowFile 追踪FlowFile的血缘关系
func (pt *ProvenanceTracker) TraceFlowFile(flowFileID string) (*DataLineage, error) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	// 查找FlowFile的所有事件
	events, err := pt.repository.FindEventsByFlowFileID(flowFileID)
	if err != nil {
		return nil, fmt.Errorf("查找FlowFile事件失败: %w", err)
	}

	// 构建血缘关系
	return pt.buildDataLineage(events)
}

// TraceLineage 追踪血缘ID的血缘关系
func (pt *ProvenanceTracker) TraceLineage(lineageID string) (*DataLineage, error) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	// 查找血缘ID的所有事件
	events, err := pt.repository.FindEventsByLineageID(lineageID)
	if err != nil {
		return nil, fmt.Errorf("查找血缘事件失败: %w", err)
	}

	// 构建血缘关系
	return pt.buildDataLineage(events)
}

// QueryEvents 查询溯源事件
func (pt *ProvenanceTracker) QueryEvents(query *ProvenanceQuery) ([]*ProvenanceEvent, error) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	return pt.repository.QueryEvents(query)
}

// GetEvent 获取指定事件
func (pt *ProvenanceTracker) GetEvent(eventID string) (*ProvenanceEvent, error) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	return pt.repository.GetEvent(eventID)
}

// buildDataLineage 构建数据血缘关系
func (pt *ProvenanceTracker) buildDataLineage(events []*ProvenanceEvent) (*DataLineage, error) {
	if len(events) == 0 {
		return nil, fmt.Errorf("没有找到溯源事件")
	}

	// 按时间戳排序事件
	sortedEvents := sortEventsByTimestamp(events)

	// 构建血缘路径
	lineage := &DataLineage{
		LineageID:  sortedEvents[0].LineageID,
		FlowFileID: sortedEvents[0].FlowFileID,
		Path:       make([]*LineageNode, 0),
		StartTime:  sortedEvents[0].Timestamp,
		EndTime:    sortedEvents[len(sortedEvents)-1].Timestamp,
	}

	// 构建血缘节点
	for _, event := range sortedEvents {
		node := &LineageNode{
			EventID:     event.EventID,
			ProcessorID: event.ProcessorID,
			EventType:   event.EventType,
			Timestamp:   event.Timestamp,
			Details:     event.Details,
		}
		lineage.Path = append(lineage.Path, node)
	}

	return lineage, nil
}

// ProvenanceEvent 溯源事件
type ProvenanceEvent struct {
	EventID     string
	FlowFileID  string
	LineageID   string
	EventType   ProvenanceEventType
	ProcessorID string
	Timestamp   time.Time
	Details     map[string]string
}

// ProvenanceEventType 溯源事件类型
type ProvenanceEventType int

const (
	EventTypeCreate ProvenanceEventType = iota
	EventTypeReceive
	EventTypeProcess
	EventTypeSend
	EventTypeDrop
	EventTypeExpire
	EventTypeFork
	EventTypeJoin
	EventTypeClone
	EventTypeContentModified
	EventTypeAttributeModified
)

// String 返回溯源事件类型的字符串表示
func (pet ProvenanceEventType) String() string {
	switch pet {
	case EventTypeCreate:
		return "CREATE"
	case EventTypeReceive:
		return "RECEIVE"
	case EventTypeProcess:
		return "PROCESS"
	case EventTypeSend:
		return "SEND"
	case EventTypeDrop:
		return "DROP"
	case EventTypeExpire:
		return "EXPIRE"
	case EventTypeFork:
		return "FORK"
	case EventTypeJoin:
		return "JOIN"
	case EventTypeClone:
		return "CLONE"
	case EventTypeContentModified:
		return "CONTENT_MODIFIED"
	case EventTypeAttributeModified:
		return "ATTRIBUTE_MODIFIED"
	default:
		return "UNKNOWN"
	}
}

// DataLineage 数据血缘关系
type DataLineage struct {
	LineageID  string
	FlowFileID string
	Path       []*LineageNode
	StartTime  time.Time
	EndTime    time.Time
}

// GetPathLength 获取血缘路径长度
func (dl *DataLineage) GetPathLength() int {
	return len(dl.Path)
}

// GetDuration 获取处理时长
func (dl *DataLineage) GetDuration() time.Duration {
	return dl.EndTime.Sub(dl.StartTime)
}

// GetProcessorCount 获取涉及的处理器数量
func (dl *DataLineage) GetProcessorCount() int {
	processors := make(map[string]bool)
	for _, node := range dl.Path {
		processors[node.ProcessorID] = true
	}
	return len(processors)
}

// LineageNode 血缘节点
type LineageNode struct {
	EventID     string
	ProcessorID string
	EventType   ProvenanceEventType
	Timestamp   time.Time
	Details     map[string]string
}

// ProvenanceRepository 溯源仓库
type ProvenanceRepository struct {
	events map[string]*ProvenanceEvent
	mu     sync.RWMutex
}

// NewProvenanceRepository 创建溯源仓库
func NewProvenanceRepository() *ProvenanceRepository {
	return &ProvenanceRepository{
		events: make(map[string]*ProvenanceEvent),
	}
}

// Initialize 初始化仓库
func (pr *ProvenanceRepository) Initialize(ctx context.Context) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	// 这里可以初始化仓库的存储后端
	// 例如：数据库、文件系统等

	return nil
}

// Store 存储事件
func (pr *ProvenanceRepository) Store(event *ProvenanceEvent) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	pr.events[event.EventID] = event

	// 这里应该持久化到存储后端
	fmt.Printf("存储溯源事件: %s -> %s (%s)\n", event.EventID, event.FlowFileID, event.EventType.String())

	return nil
}

// GetEvent 获取事件
func (pr *ProvenanceRepository) GetEvent(eventID string) (*ProvenanceEvent, error) {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	event, exists := pr.events[eventID]
	if !exists {
		return nil, fmt.Errorf("事件 %s 不存在", eventID)
	}

	return event, nil
}

// FindEventsByFlowFileID 根据FlowFile ID查找事件
func (pr *ProvenanceRepository) FindEventsByFlowFileID(flowFileID string) ([]*ProvenanceEvent, error) {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	events := make([]*ProvenanceEvent, 0)
	for _, event := range pr.events {
		if event.FlowFileID == flowFileID {
			events = append(events, event)
		}
	}

	return events, nil
}

// FindEventsByLineageID 根据血缘ID查找事件
func (pr *ProvenanceRepository) FindEventsByLineageID(lineageID string) ([]*ProvenanceEvent, error) {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	events := make([]*ProvenanceEvent, 0)
	for _, event := range pr.events {
		if event.LineageID == lineageID {
			events = append(events, event)
		}
	}

	return events, nil
}

// QueryEvents 查询事件
func (pr *ProvenanceRepository) QueryEvents(query *ProvenanceQuery) ([]*ProvenanceEvent, error) {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	events := make([]*ProvenanceEvent, 0)
	for _, event := range pr.events {
		if pr.matchesQuery(event, query) {
			events = append(events, event)
		}
	}

	return events, nil
}

// matchesQuery 检查事件是否匹配查询条件
func (pr *ProvenanceRepository) matchesQuery(event *ProvenanceEvent, query *ProvenanceQuery) bool {
	// 检查FlowFile ID
	if query.FlowFileID != "" && event.FlowFileID != query.FlowFileID {
		return false
	}

	// 检查血缘ID
	if query.LineageID != "" && event.LineageID != query.LineageID {
		return false
	}

	// 检查事件类型
	if query.EventType != nil && event.EventType != *query.EventType {
		return false
	}

	// 检查处理器ID
	if query.ProcessorID != "" && event.ProcessorID != query.ProcessorID {
		return false
	}

	// 检查时间范围
	if !query.StartTime.IsZero() && event.Timestamp.Before(query.StartTime) {
		return false
	}

	if !query.EndTime.IsZero() && event.Timestamp.After(query.EndTime) {
		return false
	}

	return true
}

// ProvenanceQuery 溯源查询
type ProvenanceQuery struct {
	FlowFileID  string
	LineageID   string
	EventType   *ProvenanceEventType
	ProcessorID string
	StartTime   time.Time
	EndTime     time.Time
	Limit       int
	Offset      int
}

// NewProvenanceQuery 创建溯源查询
func NewProvenanceQuery() *ProvenanceQuery {
	return &ProvenanceQuery{}
}

// SetFlowFileID 设置FlowFile ID
func (pq *ProvenanceQuery) SetFlowFileID(flowFileID string) *ProvenanceQuery {
	pq.FlowFileID = flowFileID
	return pq
}

// SetLineageID 设置血缘ID
func (pq *ProvenanceQuery) SetLineageID(lineageID string) *ProvenanceQuery {
	pq.LineageID = lineageID
	return pq
}

// SetEventType 设置事件类型
func (pq *ProvenanceQuery) SetEventType(eventType ProvenanceEventType) *ProvenanceQuery {
	pq.EventType = &eventType
	return pq
}

// SetProcessorID 设置处理器ID
func (pq *ProvenanceQuery) SetProcessorID(processorID string) *ProvenanceQuery {
	pq.ProcessorID = processorID
	return pq
}

// SetTimeRange 设置时间范围
func (pq *ProvenanceQuery) SetTimeRange(startTime, endTime time.Time) *ProvenanceQuery {
	pq.StartTime = startTime
	pq.EndTime = endTime
	return pq
}

// SetPagination 设置分页
func (pq *ProvenanceQuery) SetPagination(limit, offset int) *ProvenanceQuery {
	pq.Limit = limit
	pq.Offset = offset
	return pq
}

// DataLineageTracker 数据血缘跟踪器
type DataLineageTracker struct {
	tracker *ProvenanceTracker
	mu      sync.RWMutex
}

// NewDataLineageTracker 创建数据血缘跟踪器
func NewDataLineageTracker(tracker *ProvenanceTracker) *DataLineageTracker {
	return &DataLineageTracker{
		tracker: tracker,
	}
}

// Trace 追踪FlowFile的血缘关系
func (dlt *DataLineageTracker) Trace(flowFile *pipeline.FlowFile) (*DataLineage, error) {
	dlt.mu.RLock()
	defer dlt.mu.RUnlock()

	return dlt.tracker.TraceFlowFile(flowFile.UUID)
}

// TraceByLineageID 根据血缘ID追踪
func (dlt *DataLineageTracker) TraceByLineageID(lineageID string) (*DataLineage, error) {
	dlt.mu.RLock()
	defer dlt.mu.RUnlock()

	return dlt.tracker.TraceLineage(lineageID)
}

// GetLineageStatistics 获取血缘统计信息
func (dlt *DataLineageTracker) GetLineageStatistics(lineageID string) (*LineageStatistics, error) {
	dlt.mu.RLock()
	defer dlt.mu.RUnlock()

	lineage, err := dlt.tracker.TraceLineage(lineageID)
	if err != nil {
		return nil, err
	}

	stats := &LineageStatistics{
		LineageID:      lineageID,
		PathLength:     lineage.GetPathLength(),
		Duration:       lineage.GetDuration(),
		ProcessorCount: lineage.GetProcessorCount(),
		StartTime:      lineage.StartTime,
		EndTime:        lineage.EndTime,
	}

	// 统计事件类型
	eventTypeCounts := make(map[ProvenanceEventType]int)
	for _, node := range lineage.Path {
		eventTypeCounts[node.EventType]++
	}
	stats.EventTypeCounts = eventTypeCounts

	return stats, nil
}

// LineageStatistics 血缘统计信息
type LineageStatistics struct {
	LineageID       string
	PathLength      int
	Duration        time.Duration
	ProcessorCount  int
	StartTime       time.Time
	EndTime         time.Time
	EventTypeCounts map[ProvenanceEventType]int
}

// ProvenanceMonitor 溯源监控器
type ProvenanceMonitor struct {
	tracker *ProvenanceTracker
	mu      sync.RWMutex
}

// NewProvenanceMonitor 创建溯源监控器
func NewProvenanceMonitor(tracker *ProvenanceTracker) *ProvenanceMonitor {
	return &ProvenanceMonitor{
		tracker: tracker,
	}
}

// GetEventCount 获取事件数量
func (pm *ProvenanceMonitor) GetEventCount() (int, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	query := NewProvenanceQuery()
	events, err := pm.tracker.QueryEvents(query)
	if err != nil {
		return 0, err
	}

	return len(events), nil
}

// GetEventCountByType 根据类型获取事件数量
func (pm *ProvenanceMonitor) GetEventCountByType(eventType ProvenanceEventType) (int, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	query := NewProvenanceQuery().SetEventType(eventType)
	events, err := pm.tracker.QueryEvents(query)
	if err != nil {
		return 0, err
	}

	return len(events), nil
}

// GetRecentEvents 获取最近的事件
func (pm *ProvenanceMonitor) GetRecentEvents(limit int) ([]*ProvenanceEvent, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	query := NewProvenanceQuery().SetPagination(limit, 0)
	return pm.tracker.QueryEvents(query)
}

// GetEventsInTimeRange 获取时间范围内的事件
func (pm *ProvenanceMonitor) GetEventsInTimeRange(startTime, endTime time.Time) ([]*ProvenanceEvent, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	query := NewProvenanceQuery().SetTimeRange(startTime, endTime)
	return pm.tracker.QueryEvents(query)
}

// 工具函数
func generateEventID() string {
	return fmt.Sprintf("event-%d", time.Now().UnixNano())
}

func sortEventsByTimestamp(events []*ProvenanceEvent) []*ProvenanceEvent {
	// 这里实现按时间戳排序的逻辑
	// 可以使用sort包

	// 简单实现：按时间戳排序
	sorted := make([]*ProvenanceEvent, len(events))
	copy(sorted, events)

	// 冒泡排序（实际应用中应该使用更高效的排序算法）
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j].Timestamp.After(sorted[j+1].Timestamp) {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	return sorted
}
