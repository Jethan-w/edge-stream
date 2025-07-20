package pipeline

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/crazy/edge-stream/internal/processor"
	"github.com/crazy/edge-stream/internal/sink"
)

// Pipeline 数据处理流程接口
type Pipeline interface {
	// Initialize 初始化流程
	Initialize(ctx context.Context) error

	// Start 启动流程
	Start(ctx context.Context) error

	// Stop 停止流程
	Stop(ctx context.Context) error

	// Pause 暂停流程
	Pause(ctx context.Context) error

	// Resume 恢复流程
	Resume(ctx context.Context) error

	// GetState 获取流程状态
	GetState() PipelineState

	// GetConfiguration 获取流程配置
	GetConfiguration() *PipelineConfiguration

	// Validate 验证流程配置
	Validate() *ValidationResult

	// GetStatistics 获取流程统计信息
	GetStatistics() *PipelineStatistics
}

// PipelineState 流程状态
type PipelineState int

const (
	StateInitialized PipelineState = iota
	StateDeploying
	StateRunning
	StatePausing
	StatePaused
	StateResuming
	StateStopping
	StateStopped
)

// String 返回流程状态的字符串表示
func (ps PipelineState) String() string {
	switch ps {
	case StateInitialized:
		return "initialized"
	case StateDeploying:
		return "deploying"
	case StateRunning:
		return "running"
	case StatePausing:
		return "pausing"
	case StatePaused:
		return "paused"
	case StateResuming:
		return "resuming"
	case StateStopping:
		return "stopping"
	case StateStopped:
		return "stopped"
	default:
		return "unknown"
	}
}

// FlowFile 数据流转的基本单元
type FlowFile struct {
	UUID       string
	Attributes map[string]string
	Content    []byte
	Size       int64
	Timestamp  time.Time
	LineageID  string
	mu         sync.RWMutex
}

// NewFlowFile 创建新的FlowFile
func NewFlowFile() *FlowFile {
	return &FlowFile{
		UUID:       generateUUID(),
		Attributes: make(map[string]string),
		Content:    make([]byte, 0),
		Timestamp:  time.Now(),
		LineageID:  generateLineageID(),
	}
}

// AddAttribute 添加属性
func (ff *FlowFile) AddAttribute(key, value string) {
	ff.mu.Lock()
	defer ff.mu.Unlock()
	ff.Attributes[key] = value
}

// GetAttribute 获取属性
func (ff *FlowFile) GetAttribute(key string) string {
	ff.mu.RLock()
	defer ff.mu.RUnlock()
	return ff.Attributes[key]
}

// SetContent 设置内容
func (ff *FlowFile) SetContent(content []byte) {
	ff.mu.Lock()
	defer ff.mu.Unlock()
	ff.Content = content
	ff.Size = int64(len(content))
}

// GetContent 获取内容
func (ff *FlowFile) GetContent() []byte {
	ff.mu.RLock()
	defer ff.mu.RUnlock()
	return ff.Content
}

// GetSize 获取大小
func (ff *FlowFile) GetSize() int64 {
	ff.mu.RLock()
	defer ff.mu.RUnlock()
	return ff.Size
}

// GetTimestamp 获取时间戳
func (ff *FlowFile) GetTimestamp() time.Time {
	ff.mu.RLock()
	defer ff.mu.RUnlock()
	return ff.Timestamp
}

// GetLineageID 获取血缘ID
func (ff *FlowFile) GetLineageID() string {
	ff.mu.RLock()
	defer ff.mu.RUnlock()
	return ff.LineageID
}

// ProcessGroup 流程组
type ProcessGroup struct {
	ID            string
	Name          string
	Processors    map[string]*ProcessorNode
	Connections   map[string]*Connection
	ChildGroups   map[string]*ProcessGroup
	ParentGroup   *ProcessGroup
	Configuration *ProcessGroupConfiguration
	mu            sync.RWMutex
}

// NewProcessGroup 创建新的流程组
func NewProcessGroup(id, name string) *ProcessGroup {
	return &ProcessGroup{
		ID:            id,
		Name:          name,
		Processors:    make(map[string]*ProcessorNode),
		Connections:   make(map[string]*Connection),
		ChildGroups:   make(map[string]*ProcessGroup),
		Configuration: &ProcessGroupConfiguration{},
	}
}

// AddProcessor 添加处理器
func (pg *ProcessGroup) AddProcessor(processor *ProcessorNode) error {
	pg.mu.Lock()
	defer pg.mu.Unlock()

	if _, exists := pg.Processors[processor.ID]; exists {
		return fmt.Errorf("处理器 %s 已存在", processor.ID)
	}

	pg.Processors[processor.ID] = processor
	processor.SetParentGroup(pg)

	return nil
}

// RemoveProcessor 移除处理器
func (pg *ProcessGroup) RemoveProcessor(processorID string) error {
	pg.mu.Lock()
	defer pg.mu.Unlock()

	if _, exists := pg.Processors[processorID]; !exists {
		return fmt.Errorf("处理器 %s 不存在", processorID)
	}

	delete(pg.Processors, processorID)

	return nil
}

// GetProcessor 获取处理器
func (pg *ProcessGroup) GetProcessor(processorID string) (*ProcessorNode, bool) {
	pg.mu.RLock()
	defer pg.mu.RUnlock()

	processor, exists := pg.Processors[processorID]
	return processor, exists
}

// AddConnection 添加连接
func (pg *ProcessGroup) AddConnection(connection *Connection) error {
	pg.mu.Lock()
	defer pg.mu.Unlock()

	if _, exists := pg.Connections[connection.ID]; exists {
		return fmt.Errorf("连接 %s 已存在", connection.ID)
	}

	pg.Connections[connection.ID] = connection
	connection.SetParentGroup(pg)

	return nil
}

// RemoveConnection 移除连接
func (pg *ProcessGroup) RemoveConnection(connectionID string) error {
	pg.mu.Lock()
	defer pg.mu.Unlock()

	if _, exists := pg.Connections[connectionID]; !exists {
		return fmt.Errorf("连接 %s 不存在", connectionID)
	}

	delete(pg.Connections, connectionID)

	return nil
}

// GetConnection 获取连接
func (pg *ProcessGroup) GetConnection(connectionID string) (*Connection, bool) {
	pg.mu.RLock()
	defer pg.mu.RUnlock()

	connection, exists := pg.Connections[connectionID]
	return connection, exists
}

// AddChildGroup 添加子流程组
func (pg *ProcessGroup) AddChildGroup(childGroup *ProcessGroup) error {
	pg.mu.Lock()
	defer pg.mu.Unlock()

	if _, exists := pg.ChildGroups[childGroup.ID]; exists {
		return fmt.Errorf("子流程组 %s 已存在", childGroup.ID)
	}

	pg.ChildGroups[childGroup.ID] = childGroup
	childGroup.SetParentGroup(pg)

	return nil
}

// RemoveChildGroup 移除子流程组
func (pg *ProcessGroup) RemoveChildGroup(childGroupID string) error {
	pg.mu.Lock()
	defer pg.mu.Unlock()

	if _, exists := pg.ChildGroups[childGroupID]; !exists {
		return fmt.Errorf("子流程组 %s 不存在", childGroupID)
	}

	delete(pg.ChildGroups, childGroupID)

	return nil
}

// GetChildGroup 获取子流程组
func (pg *ProcessGroup) GetChildGroup(childGroupID string) (*ProcessGroup, bool) {
	pg.mu.RLock()
	defer pg.mu.RUnlock()

	childGroup, exists := pg.ChildGroups[childGroupID]
	return childGroup, exists
}

// SetParentGroup 设置父流程组
func (pg *ProcessGroup) SetParentGroup(parentGroup *ProcessGroup) {
	pg.mu.Lock()
	defer pg.mu.Unlock()
	pg.ParentGroup = parentGroup
}

// GetParentGroup 获取父流程组
func (pg *ProcessGroup) GetParentGroup() *ProcessGroup {
	pg.mu.RLock()
	defer pg.mu.RUnlock()
	return pg.ParentGroup
}

// ListProcessors 列出所有处理器
func (pg *ProcessGroup) ListProcessors() []string {
	pg.mu.RLock()
	defer pg.mu.RUnlock()

	processors := make([]string, 0, len(pg.Processors))
	for id := range pg.Processors {
		processors = append(processors, id)
	}
	return processors
}

// ListConnections 列出所有连接
func (pg *ProcessGroup) ListConnections() []string {
	pg.mu.RLock()
	defer pg.mu.RUnlock()

	connections := make([]string, 0, len(pg.Connections))
	for id := range pg.Connections {
		connections = append(connections, id)
	}
	return connections
}

// ListChildGroups 列出所有子流程组
func (pg *ProcessGroup) ListChildGroups() []string {
	pg.mu.RLock()
	defer pg.mu.RUnlock()

	childGroups := make([]string, 0, len(pg.ChildGroups))
	for id := range pg.ChildGroups {
		childGroups = append(childGroups, id)
	}
	return childGroups
}

// ProcessGroupConfiguration 流程组配置
type ProcessGroupConfiguration struct {
	MaxConcurrentTasks int
	ExecutionEngine    string
	Properties         map[string]string
	mu                 sync.RWMutex
}

// NewProcessGroupConfiguration 创建流程组配置
func NewProcessGroupConfiguration() *ProcessGroupConfiguration {
	return &ProcessGroupConfiguration{
		MaxConcurrentTasks: 1,
		ExecutionEngine:    "default",
		Properties:         make(map[string]string),
	}
}

// SetProperty 设置属性
func (pgc *ProcessGroupConfiguration) SetProperty(key, value string) {
	pgc.mu.Lock()
	defer pgc.mu.Unlock()
	pgc.Properties[key] = value
}

// GetProperty 获取属性
func (pgc *ProcessGroupConfiguration) GetProperty(key string) string {
	pgc.mu.RLock()
	defer pgc.mu.RUnlock()
	return pgc.Properties[key]
}

// ProcessorNode 处理器节点
type ProcessorNode struct {
	ID            string
	Name          string
	Type          string
	Processor     processor.Processor
	State         ProcessorNodeState
	ParentGroup   *ProcessGroup
	Configuration *ProcessorNodeConfiguration
	mu            sync.RWMutex
}

// NewProcessorNode 创建新的处理器节点
func NewProcessorNode(id, name, processorType string, proc processor.Processor) *ProcessorNode {
	return &ProcessorNode{
		ID:            id,
		Name:          name,
		Type:          processorType,
		Processor:     proc,
		State:         ProcessorNodeStateStopped,
		Configuration: &ProcessorNodeConfiguration{},
	}
}

// SetParentGroup 设置父流程组
func (pn *ProcessorNode) SetParentGroup(parentGroup *ProcessGroup) {
	pn.mu.Lock()
	defer pn.mu.Unlock()
	pn.ParentGroup = parentGroup
}

// GetParentGroup 获取父流程组
func (pn *ProcessorNode) GetParentGroup() *ProcessGroup {
	pn.mu.RLock()
	defer pn.mu.RUnlock()
	return pn.ParentGroup
}

// SetState 设置状态
func (pn *ProcessorNode) SetState(state ProcessorNodeState) {
	pn.mu.Lock()
	defer pn.mu.Unlock()
	pn.State = state
}

// GetState 获取状态
func (pn *ProcessorNode) GetState() ProcessorNodeState {
	pn.mu.RLock()
	defer pn.mu.RUnlock()
	return pn.State
}

// Start 启动处理器节点
func (pn *ProcessorNode) Start(ctx context.Context) error {
	pn.mu.Lock()
	defer pn.mu.Unlock()

	if pn.State == ProcessorNodeStateRunning {
		return fmt.Errorf("处理器 %s 已在运行", pn.ID)
	}

	if err := pn.Processor.Start(ctx); err != nil {
		return fmt.Errorf("启动处理器失败: %w", err)
	}

	pn.State = ProcessorNodeStateRunning
	return nil
}

// Stop 停止处理器节点
func (pn *ProcessorNode) Stop(ctx context.Context) error {
	pn.mu.Lock()
	defer pn.mu.Unlock()

	if pn.State == ProcessorNodeStateStopped {
		return nil
	}

	if err := pn.Processor.Stop(ctx); err != nil {
		return fmt.Errorf("停止处理器失败: %w", err)
	}

	pn.State = ProcessorNodeStateStopped
	return nil
}

// Process 处理FlowFile
func (pn *ProcessorNode) Process(ctx context.Context, flowFile *FlowFile) error {
	pn.mu.RLock()
	defer pn.mu.RUnlock()

	if pn.State != ProcessorNodeStateRunning {
		return fmt.Errorf("处理器 %s 未运行", pn.ID)
	}

	// 这里需要将FlowFile转换为processor.Processor可以处理的格式
	// 具体实现取决于processor.Processor的接口定义

	return nil
}

// ProcessorNodeState 处理器节点状态
type ProcessorNodeState int

const (
	ProcessorNodeStateStopped ProcessorNodeState = iota
	ProcessorNodeStateRunning
	ProcessorNodeStateDisabled
	ProcessorNodeStateInvalid
)

// String 返回处理器节点状态的字符串表示
func (pns ProcessorNodeState) String() string {
	switch pns {
	case ProcessorNodeStateStopped:
		return "stopped"
	case ProcessorNodeStateRunning:
		return "running"
	case ProcessorNodeStateDisabled:
		return "disabled"
	case ProcessorNodeStateInvalid:
		return "invalid"
	default:
		return "unknown"
	}
}

// ProcessorNodeConfiguration 处理器节点配置
type ProcessorNodeConfiguration struct {
	Properties map[string]string
	Scheduling *SchedulingConfiguration
	mu         sync.RWMutex
}

// NewProcessorNodeConfiguration 创建处理器节点配置
func NewProcessorNodeConfiguration() *ProcessorNodeConfiguration {
	return &ProcessorNodeConfiguration{
		Properties: make(map[string]string),
		Scheduling: &SchedulingConfiguration{},
	}
}

// SetProperty 设置属性
func (pnc *ProcessorNodeConfiguration) SetProperty(key, value string) {
	pnc.mu.Lock()
	defer pnc.mu.Unlock()
	pnc.Properties[key] = value
}

// GetProperty 获取属性
func (pnc *ProcessorNodeConfiguration) GetProperty(key string) string {
	pnc.mu.RLock()
	defer pnc.mu.RUnlock()
	return pnc.Properties[key]
}

// SchedulingConfiguration 调度配置
type SchedulingConfiguration struct {
	Strategy           string
	Execution          string
	Period             time.Duration
	MaxConcurrentTasks int
}

// Connection 连接
type Connection struct {
	ID            string
	Name          string
	Source        *ProcessorNode
	Destination   *ProcessorNode
	Queue         *ConnectionQueue
	Prioritizer   QueuePrioritizer
	ParentGroup   *ProcessGroup
	Configuration *ConnectionConfiguration
	mu            sync.RWMutex
}

// NewConnection 创建新的连接
func NewConnection(id, name string, source, destination *ProcessorNode) *Connection {
	return &Connection{
		ID:            id,
		Name:          name,
		Source:        source,
		Destination:   destination,
		Queue:         NewConnectionQueue(),
		Prioritizer:   NewOldestFirstPrioritizer(),
		Configuration: &ConnectionConfiguration{},
	}
}

// SetParentGroup 设置父流程组
func (c *Connection) SetParentGroup(parentGroup *ProcessGroup) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ParentGroup = parentGroup
}

// GetParentGroup 获取父流程组
func (c *Connection) GetParentGroup() *ProcessGroup {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ParentGroup
}

// Enqueue 入队FlowFile
func (c *Connection) Enqueue(flowFile *FlowFile) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.Queue.Enqueue(flowFile)
}

// Dequeue 出队FlowFile
func (c *Connection) Dequeue() (*FlowFile, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.Queue.Dequeue(c.Prioritizer)
}

// GetQueueSize 获取队列大小
func (c *Connection) GetQueueSize() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.Queue.Size()
}

// SetPrioritizer 设置优先级策略
func (c *Connection) SetPrioritizer(prioritizer QueuePrioritizer) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Prioritizer = prioritizer
}

// GetPrioritizer 获取优先级策略
func (c *Connection) GetPrioritizer() QueuePrioritizer {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Prioritizer
}

// ConnectionConfiguration 连接配置
type ConnectionConfiguration struct {
	MaxQueueSize int
	Properties   map[string]string
	mu           sync.RWMutex
}

// NewConnectionConfiguration 创建连接配置
func NewConnectionConfiguration() *ConnectionConfiguration {
	return &ConnectionConfiguration{
		MaxQueueSize: 1000,
		Properties:   make(map[string]string),
	}
}

// SetProperty 设置属性
func (cc *ConnectionConfiguration) SetProperty(key, value string) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.Properties[key] = value
}

// GetProperty 获取属性
func (cc *ConnectionConfiguration) GetProperty(key string) string {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return cc.Properties[key]
}

// ConnectionQueue 连接队列
type ConnectionQueue struct {
	queue []*FlowFile
	mu    sync.RWMutex
}

// NewConnectionQueue 创建连接队列
func NewConnectionQueue() *ConnectionQueue {
	return &ConnectionQueue{
		queue: make([]*FlowFile, 0),
	}
}

// Enqueue 入队
func (cq *ConnectionQueue) Enqueue(flowFile *FlowFile) error {
	cq.mu.Lock()
	defer cq.mu.Unlock()

	cq.queue = append(cq.queue, flowFile)
	return nil
}

// Dequeue 出队
func (cq *ConnectionQueue) Dequeue(prioritizer QueuePrioritizer) (*FlowFile, error) {
	cq.mu.Lock()
	defer cq.mu.Unlock()

	if len(cq.queue) == 0 {
		return nil, fmt.Errorf("队列为空")
	}

	selectedFlowFile := prioritizer.SelectNext(cq.queue)
	if selectedFlowFile == nil {
		return nil, fmt.Errorf("无法选择FlowFile")
	}

	// 从队列中移除选中的FlowFile
	for i, ff := range cq.queue {
		if ff.UUID == selectedFlowFile.UUID {
			cq.queue = append(cq.queue[:i], cq.queue[i+1:]...)
			break
		}
	}

	return selectedFlowFile, nil
}

// Size 获取队列大小
func (cq *ConnectionQueue) Size() int {
	cq.mu.RLock()
	defer cq.mu.RUnlock()
	return len(cq.queue)
}

// QueuePrioritizer 队列优先级策略接口
type QueuePrioritizer interface {
	SelectNext(queue []*FlowFile) *FlowFile
}

// OldestFirstPrioritizer 先进先出优先级策略
type OldestFirstPrioritizer struct{}

// NewOldestFirstPrioritizer 创建先进先出优先级策略
func NewOldestFirstPrioritizer() *OldestFirstPrioritizer {
	return &OldestFirstPrioritizer{}
}

// SelectNext 选择下一个FlowFile
func (ofp *OldestFirstPrioritizer) SelectNext(queue []*FlowFile) *FlowFile {
	if len(queue) == 0 {
		return nil
	}

	// 选择时间戳最早的FlowFile
	oldest := queue[0]
	for _, ff := range queue[1:] {
		if ff.Timestamp.Before(oldest.Timestamp) {
			oldest = ff
		}
	}

	return oldest
}

// NewestFirstPrioritizer 后进先出优先级策略
type NewestFirstPrioritizer struct{}

// NewNewestFirstPrioritizer 创建后进先出优先级策略
func NewNewestFirstPrioritizer() *NewestFirstPrioritizer {
	return &NewestFirstPrioritizer{}
}

// SelectNext 选择下一个FlowFile
func (nfp *NewestFirstPrioritizer) SelectNext(queue []*FlowFile) *FlowFile {
	if len(queue) == 0 {
		return nil
	}

	// 选择时间戳最新的FlowFile
	newest := queue[0]
	for _, ff := range queue[1:] {
		if ff.Timestamp.After(newest.Timestamp) {
			newest = ff
		}
	}

	return newest
}

// PipelineConfiguration 流程配置
type PipelineConfiguration struct {
	ID          string
	Name        string
	Description string
	Version     string
	Properties  map[string]string
	mu          sync.RWMutex
}

// NewPipelineConfiguration 创建流程配置
func NewPipelineConfiguration() *PipelineConfiguration {
	return &PipelineConfiguration{
		Properties: make(map[string]string),
	}
}

// SetProperty 设置属性
func (pc *PipelineConfiguration) SetProperty(key, value string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.Properties[key] = value
}

// GetProperty 获取属性
func (pc *PipelineConfiguration) GetProperty(key string) string {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return pc.Properties[key]
}

// ValidationResult 验证结果
type ValidationResult struct {
	Valid    bool
	Errors   []string
	Warnings []string
}

// AddError 添加错误
func (vr *ValidationResult) AddError(error string) {
	vr.Valid = false
	vr.Errors = append(vr.Errors, error)
}

// AddWarning 添加警告
func (vr *ValidationResult) AddWarning(warning string) {
	vr.Warnings = append(vr.Warnings, warning)
}

// PipelineStatistics 流程统计信息
type PipelineStatistics struct {
	TotalFlowFilesProcessed int64
	TotalBytesProcessed     int64
	AverageProcessingTime   time.Duration
	LastProcessedTime       time.Time
	ActiveConnections       int
	ActiveProcessors        int
	mu                      sync.RWMutex
}

// NewPipelineStatistics 创建流程统计信息
func NewPipelineStatistics() *PipelineStatistics {
	return &PipelineStatistics{}
}

// IncrementFlowFilesProcessed 增加处理的FlowFile数量
func (ps *PipelineStatistics) IncrementFlowFilesProcessed() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.TotalFlowFilesProcessed++
}

// AddBytesProcessed 增加处理的字节数
func (ps *PipelineStatistics) AddBytesProcessed(bytes int64) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.TotalBytesProcessed += bytes
}

// SetAverageProcessingTime 设置平均处理时间
func (ps *PipelineStatistics) SetAverageProcessingTime(duration time.Duration) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.AverageProcessingTime = duration
}

// SetLastProcessedTime 设置最后处理时间
func (ps *PipelineStatistics) SetLastProcessedTime(t time.Time) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.LastProcessedTime = t
}

// SetActiveConnections 设置活跃连接数
func (ps *PipelineStatistics) SetActiveConnections(count int) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.ActiveConnections = count
}

// SetActiveProcessors 设置活跃处理器数
func (ps *PipelineStatistics) SetActiveProcessors(count int) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.ActiveProcessors = count
}

// 工具函数
func generateUUID() string {
	// 这里应该使用实际的UUID生成库
	// 例如：github.com/google/uuid
	return fmt.Sprintf("flowfile-%d", time.Now().UnixNano())
}

func generateLineageID() string {
	// 这里应该使用实际的LineageID生成逻辑
	return fmt.Sprintf("lineage-%d", time.Now().UnixNano())
}
