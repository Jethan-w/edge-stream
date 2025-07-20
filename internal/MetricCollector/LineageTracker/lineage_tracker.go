package LineageTracker

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// LineageService 数据溯源服务接口
type LineageService interface {
	// TraceFlowFile 追踪FlowFile
	TraceFlowFile(flowFileID string) (*LineageResult, error)

	// AddLineageEvent 添加溯源事件
	AddLineageEvent(event *LineageEvent) error

	// GetLineageEvents 获取溯源事件
	GetLineageEvents(flowFileID string) ([]*LineageEvent, error)

	// GetLineageEventsByTimeRange 按时间范围获取溯源事件
	GetLineageEventsByTimeRange(startTime, endTime time.Time) ([]*LineageEvent, error)

	// GetLineageEventsByProcessor 按处理器获取溯源事件
	GetLineageEventsByProcessor(processorID string) ([]*LineageEvent, error)

	// GetLineageSummary 获取溯源摘要
	GetLineageSummary(flowFileID string) (*LineageSummary, error)
}

// LineageEventType 溯源事件类型枚举
type LineageEventType string

const (
	LineageEventTypeCreate  LineageEventType = "CREATE"
	LineageEventTypeReceive LineageEventType = "RECEIVE"
	LineageEventTypeSend    LineageEventType = "SEND"
	LineageEventTypeRoute   LineageEventType = "ROUTE"
	LineageEventTypeClone   LineageEventType = "CLONE"
	LineageEventTypeModify  LineageEventType = "MODIFY"
	LineageEventTypeDrop    LineageEventType = "DROP"
	LineageEventTypeExpire  LineageEventType = "EXPIRE"
)

// LineageEvent 溯源事件
type LineageEvent struct {
	EventID       string            `json:"eventId"`
	FlowFileID    string            `json:"flowFileId"`
	EventType     LineageEventType  `json:"eventType"`
	ProcessorID   string            `json:"processorId"`
	ProcessorName string            `json:"processorName"`
	Timestamp     time.Time         `json:"timestamp"`
	Details       map[string]string `json:"details"`
	Attributes    map[string]string `json:"attributes"`
	Size          int64             `json:"size"`
	LineageStart  bool              `json:"lineageStart"`
	LineageEnd    bool              `json:"lineageEnd"`
}

// NewLineageEvent 创建新的溯源事件
func NewLineageEvent(
	flowFileID string,
	eventType LineageEventType,
	processorID, processorName string,
) *LineageEvent {
	return &LineageEvent{
		EventID:       generateEventID(),
		FlowFileID:    flowFileID,
		EventType:     eventType,
		ProcessorID:   processorID,
		ProcessorName: processorName,
		Timestamp:     time.Now(),
		Details:       make(map[string]string),
		Attributes:    make(map[string]string),
	}
}

// LineageResult 溯源结果
type LineageResult struct {
	FlowFileID     string           `json:"flowFileId"`
	Events         []*LineageEvent  `json:"events"`
	ProcessingPath []*ProcessorNode `json:"processingPath"`
	TotalEvents    int              `json:"totalEvents"`
	StartTime      time.Time        `json:"startTime"`
	EndTime        time.Time        `json:"endTime"`
	Duration       time.Duration    `json:"duration"`
	TotalSize      int64            `json:"totalSize"`
	FinalSize      int64            `json:"finalSize"`
}

// NewLineageResult 创建新的溯源结果
func NewLineageResult(flowFileID string) *LineageResult {
	return &LineageResult{
		FlowFileID:     flowFileID,
		Events:         make([]*LineageEvent, 0),
		ProcessingPath: make([]*ProcessorNode, 0),
	}
}

// ProcessorNode 处理器节点
type ProcessorNode struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Type           string            `json:"type"`
	InputCount     int64             `json:"inputCount"`
	OutputCount    int64             `json:"outputCount"`
	ErrorCount     int64             `json:"errorCount"`
	ProcessingTime time.Duration     `json:"processingTime"`
	Attributes     map[string]string `json:"attributes"`
}

// NewProcessorNode 创建新的处理器节点
func NewProcessorNode(id, name, processorType string) *ProcessorNode {
	return &ProcessorNode{
		ID:         id,
		Name:       name,
		Type:       processorType,
		Attributes: make(map[string]string),
	}
}

// LineageSummary 溯源摘要
type LineageSummary struct {
	FlowFileID       string                      `json:"flowFileId"`
	TotalEvents      int                         `json:"totalEvents"`
	UniqueProcessors int                         `json:"uniqueProcessors"`
	StartTime        time.Time                   `json:"startTime"`
	EndTime          time.Time                   `json:"endTime"`
	Duration         time.Duration               `json:"duration"`
	TotalSize        int64                       `json:"totalSize"`
	FinalSize        int64                       `json:"finalSize"`
	EventTypes       map[LineageEventType]int    `json:"eventTypes"`
	Processors       map[string]ProcessorSummary `json:"processors"`
}

// ProcessorSummary 处理器摘要
type ProcessorSummary struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	Type           string        `json:"type"`
	EventCount     int           `json:"eventCount"`
	InputCount     int64         `json:"inputCount"`
	OutputCount    int64         `json:"outputCount"`
	ErrorCount     int64         `json:"errorCount"`
	ProcessingTime time.Duration `json:"processingTime"`
	FirstSeen      time.Time     `json:"firstSeen"`
	LastSeen       time.Time     `json:"lastSeen"`
}

// NewLineageSummary 创建新的溯源摘要
func NewLineageSummary(flowFileID string) *LineageSummary {
	return &LineageSummary{
		FlowFileID: flowFileID,
		EventTypes: make(map[LineageEventType]int),
		Processors: make(map[string]ProcessorSummary),
	}
}

// StandardLineageService 标准数据溯源服务实现
type StandardLineageService struct {
	provenanceRepository ProvenanceRepository
	mutex                sync.RWMutex
}

// NewStandardLineageService 创建新的标准数据溯源服务
func NewStandardLineageService(repository ProvenanceRepository) *StandardLineageService {
	return &StandardLineageService{
		provenanceRepository: repository,
	}
}

// TraceFlowFile 追踪FlowFile
func (sls *StandardLineageService) TraceFlowFile(flowFileID string) (*LineageResult, error) {
	sls.mutex.RLock()
	defer sls.mutex.RUnlock()

	// 获取FlowFile的所有事件
	events, err := sls.provenanceRepository.FindEventsByFlowFileID(flowFileID)
	if err != nil {
		return nil, fmt.Errorf("获取FlowFile事件失败: %w", err)
	}

	if len(events) == 0 {
		return nil, fmt.Errorf("未找到FlowFile: %s", flowFileID)
	}

	// 创建溯源结果
	result := NewLineageResult(flowFileID)
	result.Events = events
	result.TotalEvents = len(events)

	// 计算处理路径
	result.ProcessingPath = sls.calculateProcessingPath(events)

	// 计算时间范围
	result.StartTime = events[0].Timestamp
	result.EndTime = events[len(events)-1].Timestamp
	result.Duration = result.EndTime.Sub(result.StartTime)

	// 计算大小信息
	result.TotalSize = sls.calculateTotalSize(events)
	result.FinalSize = sls.calculateFinalSize(events)

	return result, nil
}

// AddLineageEvent 添加溯源事件
func (sls *StandardLineageService) AddLineageEvent(event *LineageEvent) error {
	sls.mutex.Lock()
	defer sls.mutex.Unlock()

	return sls.provenanceRepository.StoreEvent(event)
}

// GetLineageEvents 获取溯源事件
func (sls *StandardLineageService) GetLineageEvents(flowFileID string) ([]*LineageEvent, error) {
	sls.mutex.RLock()
	defer sls.mutex.RUnlock()

	return sls.provenanceRepository.FindEventsByFlowFileID(flowFileID)
}

// GetLineageEventsByTimeRange 按时间范围获取溯源事件
func (sls *StandardLineageService) GetLineageEventsByTimeRange(
	startTime, endTime time.Time,
) ([]*LineageEvent, error) {
	sls.mutex.RLock()
	defer sls.mutex.RUnlock()

	return sls.provenanceRepository.FindEventsByTimeRange(startTime, endTime)
}

// GetLineageEventsByProcessor 按处理器获取溯源事件
func (sls *StandardLineageService) GetLineageEventsByProcessor(processorID string) ([]*LineageEvent, error) {
	sls.mutex.RLock()
	defer sls.mutex.RUnlock()

	return sls.provenanceRepository.FindEventsByProcessorID(processorID)
}

// GetLineageSummary 获取溯源摘要
func (sls *StandardLineageService) GetLineageSummary(flowFileID string) (*LineageSummary, error) {
	sls.mutex.RLock()
	defer sls.mutex.RUnlock()

	// 获取FlowFile的所有事件
	events, err := sls.provenanceRepository.FindEventsByFlowFileID(flowFileID)
	if err != nil {
		return nil, fmt.Errorf("获取FlowFile事件失败: %w", err)
	}

	if len(events) == 0 {
		return nil, fmt.Errorf("未找到FlowFile: %s", flowFileID)
	}

	// 创建溯源摘要
	summary := NewLineageSummary(flowFileID)
	summary.TotalEvents = len(events)
	summary.StartTime = events[0].Timestamp
	summary.EndTime = events[len(events)-1].Timestamp
	summary.Duration = summary.EndTime.Sub(summary.StartTime)

	// 统计事件类型
	for _, event := range events {
		summary.EventTypes[event.EventType]++
	}

	// 统计处理器信息
	processorMap := make(map[string]*ProcessorSummary)
	for _, event := range events {
		processor, exists := processorMap[event.ProcessorID]
		if !exists {
			processor = &ProcessorSummary{
				ID:        event.ProcessorID,
				Name:      event.ProcessorName,
				FirstSeen: event.Timestamp,
			}
			processorMap[event.ProcessorID] = processor
		}

		processor.EventCount++
		processor.LastSeen = event.Timestamp

		// 根据事件类型更新统计信息
		switch event.EventType {
		case LineageEventTypeReceive:
			processor.InputCount += event.Size
		case LineageEventTypeSend:
			processor.OutputCount += event.Size
		case LineageEventTypeDrop:
			processor.ErrorCount += event.Size
		}
	}

	// 转换处理器映射
	for _, processor := range processorMap {
		summary.Processors[processor.ID] = *processor
	}

	summary.UniqueProcessors = len(processorMap)
	summary.TotalSize = sls.calculateTotalSize(events)
	summary.FinalSize = sls.calculateFinalSize(events)

	return summary, nil
}

// calculateProcessingPath 计算处理路径
func (sls *StandardLineageService) calculateProcessingPath(events []*LineageEvent) []*ProcessorNode {
	processorMap := make(map[string]*ProcessorNode)

	for _, event := range events {
		processor, exists := processorMap[event.ProcessorID]
		if !exists {
			processor = NewProcessorNode(event.ProcessorID, event.ProcessorName, "Unknown")
			processorMap[event.ProcessorID] = processor
		}

		// 根据事件类型更新统计信息
		switch event.EventType {
		case LineageEventTypeReceive:
			processor.InputCount += event.Size
		case LineageEventTypeSend:
			processor.OutputCount += event.Size
		case LineageEventTypeDrop:
			processor.ErrorCount += event.Size
		}

		// 复制属性
		for key, value := range event.Attributes {
			processor.Attributes[key] = value
		}
	}

	// 转换为切片
	var processors []*ProcessorNode
	for _, processor := range processorMap {
		processors = append(processors, processor)
	}

	return processors
}

// calculateTotalSize 计算总大小
func (sls *StandardLineageService) calculateTotalSize(events []*LineageEvent) int64 {
	var totalSize int64
	for _, event := range events {
		totalSize += event.Size
	}
	return totalSize
}

// calculateFinalSize 计算最终大小
func (sls *StandardLineageService) calculateFinalSize(events []*LineageEvent) int64 {
	if len(events) == 0 {
		return 0
	}
	return events[len(events)-1].Size
}

// ProvenanceRepository 溯源存储库接口
type ProvenanceRepository interface {
	// StoreEvent 存储事件
	StoreEvent(event *LineageEvent) error

	// FindEventsByFlowFileID 根据FlowFile ID查找事件
	FindEventsByFlowFileID(flowFileID string) ([]*LineageEvent, error)

	// FindEventsByTimeRange 根据时间范围查找事件
	FindEventsByTimeRange(startTime, endTime time.Time) ([]*LineageEvent, error)

	// FindEventsByProcessorID 根据处理器ID查找事件
	FindEventsByProcessorID(processorID string) ([]*LineageEvent, error)

	// PurgeExpiredEvents 清理过期事件
	PurgeExpiredEvents(retentionPeriod time.Duration) error
}

// InMemoryProvenanceRepository 内存溯源存储库实现
type InMemoryProvenanceRepository struct {
	events map[string]*LineageEvent
	mutex  sync.RWMutex
}

// NewInMemoryProvenanceRepository 创建新的内存溯源存储库
func NewInMemoryProvenanceRepository() *InMemoryProvenanceRepository {
	return &InMemoryProvenanceRepository{
		events: make(map[string]*LineageEvent),
	}
}

// StoreEvent 存储事件
func (impr *InMemoryProvenanceRepository) StoreEvent(event *LineageEvent) error {
	impr.mutex.Lock()
	defer impr.mutex.Unlock()

	impr.events[event.EventID] = event
	return nil
}

// FindEventsByFlowFileID 根据FlowFile ID查找事件
func (impr *InMemoryProvenanceRepository) FindEventsByFlowFileID(flowFileID string) ([]*LineageEvent, error) {
	impr.mutex.RLock()
	defer impr.mutex.RUnlock()

	var events []*LineageEvent
	for _, event := range impr.events {
		if event.FlowFileID == flowFileID {
			events = append(events, event)
		}
	}

	return events, nil
}

// FindEventsByTimeRange 根据时间范围查找事件
func (impr *InMemoryProvenanceRepository) FindEventsByTimeRange(
	startTime, endTime time.Time,
) ([]*LineageEvent, error) {
	impr.mutex.RLock()
	defer impr.mutex.RUnlock()

	var events []*LineageEvent
	for _, event := range impr.events {
		if event.Timestamp.After(startTime) && event.Timestamp.Before(endTime) {
			events = append(events, event)
		}
	}

	return events, nil
}

// FindEventsByProcessorID 根据处理器ID查找事件
func (impr *InMemoryProvenanceRepository) FindEventsByProcessorID(processorID string) ([]*LineageEvent, error) {
	impr.mutex.RLock()
	defer impr.mutex.RUnlock()

	var events []*LineageEvent
	for _, event := range impr.events {
		if event.ProcessorID == processorID {
			events = append(events, event)
		}
	}

	return events, nil
}

// PurgeExpiredEvents 清理过期事件
func (impr *InMemoryProvenanceRepository) PurgeExpiredEvents(retentionPeriod time.Duration) error {
	impr.mutex.Lock()
	defer impr.mutex.Unlock()

	cutoffTime := time.Now().Add(-retentionPeriod)

	for eventID, event := range impr.events {
		if event.Timestamp.Before(cutoffTime) {
			delete(impr.events, eventID)
		}
	}

	return nil
}

// LineageQueryController 溯源查询控制器
type LineageQueryController struct {
	lineageService LineageService
	mutex          sync.RWMutex
}

// NewLineageQueryController 创建新的溯源查询控制器
func NewLineageQueryController(lineageService LineageService) *LineageQueryController {
	return &LineageQueryController{
		lineageService: lineageService,
	}
}

// TraceFlowFile 追踪FlowFile
func (lqc *LineageQueryController) TraceFlowFile(flowFileID string) (*LineageResult, error) {
	lqc.mutex.RLock()
	defer lqc.mutex.RUnlock()

	return lqc.lineageService.TraceFlowFile(flowFileID)
}

// GetLineageSummary 获取溯源摘要
func (lqc *LineageQueryController) GetLineageSummary(flowFileID string) (*LineageSummary, error) {
	lqc.mutex.RLock()
	defer lqc.mutex.RUnlock()

	return lqc.lineageService.GetLineageSummary(flowFileID)
}

// GetLineageEvents 获取溯源事件
func (lqc *LineageQueryController) GetLineageEvents(flowFileID string) ([]*LineageEvent, error) {
	lqc.mutex.RLock()
	defer lqc.mutex.RUnlock()

	return lqc.lineageService.GetLineageEvents(flowFileID)
}

// GetLineageEventsByTimeRange 按时间范围获取溯源事件
func (lqc *LineageQueryController) GetLineageEventsByTimeRange(
	startTime, endTime time.Time,
) ([]*LineageEvent, error) {
	lqc.mutex.RLock()
	defer lqc.mutex.RUnlock()

	return lqc.lineageService.GetLineageEventsByTimeRange(startTime, endTime)
}

// GetLineageEventsByProcessor 按处理器获取溯源事件
func (lqc *LineageQueryController) GetLineageEventsByProcessor(processorID string) ([]*LineageEvent, error) {
	lqc.mutex.RLock()
	defer lqc.mutex.RUnlock()

	return lqc.lineageService.GetLineageEventsByProcessor(processorID)
}

// LineageManager 溯源管理器
type LineageManager struct {
	lineageService LineageService
	repository     ProvenanceRepository
	mutex          sync.RWMutex
}

// NewLineageManager 创建新的溯源管理器
func NewLineageManager(repository ProvenanceRepository) *LineageManager {
	return &LineageManager{
		lineageService: NewStandardLineageService(repository),
		repository:     repository,
	}
}

// AddEvent 添加事件
func (lm *LineageManager) AddEvent(event *LineageEvent) error {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	return lm.lineageService.AddLineageEvent(event)
}

// TraceFlowFile 追踪FlowFile
func (lm *LineageManager) TraceFlowFile(flowFileID string) (*LineageResult, error) {
	lm.mutex.RLock()
	defer lm.mutex.RUnlock()

	return lm.lineageService.TraceFlowFile(flowFileID)
}

// GetSummary 获取摘要
func (lm *LineageManager) GetSummary(flowFileID string) (*LineageSummary, error) {
	lm.mutex.RLock()
	defer lm.mutex.RUnlock()

	return lm.lineageService.GetLineageSummary(flowFileID)
}

// PurgeExpiredEvents 清理过期事件
func (lm *LineageManager) PurgeExpiredEvents(retentionPeriod time.Duration) error {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	return lm.repository.PurgeExpiredEvents(retentionPeriod)
}

// 辅助函数

// generateEventID 生成事件ID
func generateEventID() string {
	return fmt.Sprintf("event_%d", time.Now().UnixNano())
}

// ConvertToJSON 转换为JSON
func ConvertToJSON(data interface{}) ([]byte, error) {
	return json.MarshalIndent(data, "", "  ")
}
