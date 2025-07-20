package source

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Source 数据源接口定义
type Source interface {
	// Initialize 初始化数据源
	Initialize(ctx context.Context) error

	// Start 启动数据源
	Start(ctx context.Context) error

	// Stop 停止数据源
	Stop(ctx context.Context) error

	// GetState 获取数据源状态
	GetState() SourceState

	// GetConfiguration 获取配置信息
	GetConfiguration() *SourceConfiguration

	// Validate 验证数据源配置
	Validate() *ValidationResult
}

// SourceState 数据源状态
type SourceState int

const (
	StateUnconfigured SourceState = iota
	StateConfiguring
	StateReady
	StateRunning
	StatePaused
	StateError
	StateStopped
)

// DataPacket 数据包接口
type DataPacket interface {
	GetType() DataType
	GetContent() []byte
	GetTimestamp() time.Time
	GetMetadata() map[string]string
	GetSize() int64
}

// DataType 数据类型
type DataType int

const (
	DataTypeVideo DataType = iota
	DataTypeText
	DataTypeBinary
	DataTypeAudio
	DataTypeCustom
)

// VideoPacket 视频数据包
type VideoPacket struct {
	FrameData  []byte
	Timestamp  time.Time
	Codec      VideoCodec
	Resolution Dimension
	FrameType  FrameType
	Metadata   map[string]string
}

// GetType 获取数据类型
func (vp *VideoPacket) GetType() DataType {
	return DataTypeVideo
}

// GetContent 获取内容
func (vp *VideoPacket) GetContent() []byte {
	return vp.FrameData
}

// GetTimestamp 获取时间戳
func (vp *VideoPacket) GetTimestamp() time.Time {
	return vp.Timestamp
}

// GetMetadata 获取元数据
func (vp *VideoPacket) GetMetadata() map[string]string {
	return vp.Metadata
}

// GetSize 获取大小
func (vp *VideoPacket) GetSize() int64 {
	return int64(len(vp.FrameData))
}

// VideoCodec 视频编码
type VideoCodec int

const (
	CodecH264 VideoCodec = iota
	CodecH265
	CodecVP8
	CodecVP9
)

// Dimension 分辨率
type Dimension struct {
	Width  int
	Height int
}

// FrameType 帧类型
type FrameType int

const (
	FrameTypeI FrameType = iota // I帧
	FrameTypeP                  // P帧
	FrameTypeB                  // B帧
)

// TextPacket 文本数据包
type TextPacket struct {
	Content   string
	Encoding  string // UTF-8/GBK
	Metadata  map[string]string
	Timestamp time.Time
}

// GetType 获取数据类型
func (tp *TextPacket) GetType() DataType {
	return DataTypeText
}

// GetContent 获取内容
func (tp *TextPacket) GetContent() []byte {
	return []byte(tp.Content)
}

// GetTimestamp 获取时间戳
func (tp *TextPacket) GetTimestamp() time.Time {
	return tp.Timestamp
}

// GetMetadata 获取元数据
func (tp *TextPacket) GetMetadata() map[string]string {
	return tp.Metadata
}

// GetSize 获取大小
func (tp *TextPacket) GetSize() int64 {
	return int64(len(tp.Content))
}

// BinaryPacket 二进制数据包
type BinaryPacket struct {
	Data      []byte
	Timestamp time.Time
	Metadata  map[string]string
}

// GetType 获取数据类型
func (bp *BinaryPacket) GetType() DataType {
	return DataTypeBinary
}

// GetContent 获取内容
func (bp *BinaryPacket) GetContent() []byte {
	return bp.Data
}

// GetTimestamp 获取时间戳
func (bp *BinaryPacket) GetTimestamp() time.Time {
	return bp.Timestamp
}

// GetMetadata 获取元数据
func (bp *BinaryPacket) GetMetadata() map[string]string {
	return bp.Metadata
}

// GetSize 获取大小
func (bp *BinaryPacket) GetSize() int64 {
	return int64(len(bp.Data))
}

// CustomDataPacket 自定义数据包
type CustomDataPacket struct {
	Data      interface{}
	Type      string
	Timestamp time.Time
	Metadata  map[string]string
}

// GetType 获取数据类型
func (cp *CustomDataPacket) GetType() DataType {
	return DataTypeCustom
}

// GetContent 获取内容
func (cp *CustomDataPacket) GetContent() []byte {
	// 将自定义数据转换为字节数组
	// 这里需要根据具体的数据类型进行序列化
	return []byte(fmt.Sprintf("%v", cp.Data))
}

// GetTimestamp 获取时间戳
func (cp *CustomDataPacket) GetTimestamp() time.Time {
	return cp.Timestamp
}

// GetMetadata 获取元数据
func (cp *CustomDataPacket) GetMetadata() map[string]string {
	return cp.Metadata
}

// GetSize 获取大小
func (cp *CustomDataPacket) GetSize() int64 {
	return int64(len(cp.GetContent()))
}

// MultiModalSource 多模态数据源接口
type MultiModalSource interface {
	Source

	// ReceiveVideo 接收视频数据
	ReceiveVideo(packet *VideoPacket) error

	// ReceiveText 接收文本数据
	ReceiveText(packet *TextPacket) error

	// ReceiveBinary 接收二进制数据
	ReceiveBinary(packet *BinaryPacket) error

	// ReceiveCustom 接收自定义数据
	ReceiveCustom(packet *CustomDataPacket) error
}

// SecureSource 安全数据源接口
type SecureSource interface {
	Source

	// ConfigureTLS 配置TLS
	ConfigureTLS(context *TLSContext) error

	// Authenticate 认证
	Authenticate(token *OAuth2Token) error

	// ConfigureIPWhitelist 配置IP白名单
	ConfigureIPWhitelist(allowedIPs []string) error
}

// SourceConfiguration 数据源配置
type SourceConfiguration struct {
	ID             string
	Type           string
	Protocol       string
	Host           string
	Port           int
	SecurityMode   string
	MaxConnections int
	ConnectTimeout time.Duration
	Properties     map[string]string
	SecurityConfig *SecurityConfiguration
	mu             sync.RWMutex
}

// NewSourceConfiguration 创建数据源配置
func NewSourceConfiguration() *SourceConfiguration {
	return &SourceConfiguration{
		Properties:     make(map[string]string),
		SecurityConfig: &SecurityConfiguration{},
	}
}

// SetProperty 设置属性
func (sc *SourceConfiguration) SetProperty(key, value string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.Properties[key] = value
}

// GetProperty 获取属性
func (sc *SourceConfiguration) GetProperty(key string) string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.Properties[key]
}

// Update 更新配置
func (sc *SourceConfiguration) Update(newConfig map[string]string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	for key, value := range newConfig {
		sc.Properties[key] = value
	}
}

// SecurityConfiguration 安全配置
type SecurityConfiguration struct {
	TLSContext   *TLSContext
	OAuth2Config *OAuth2Config
	IPWhitelist  []string
	Enabled      bool
}

// TLSContext TLS上下文
type TLSContext struct {
	Protocol   string
	KeyStore   string
	TrustStore string
	CertFile   string
	KeyFile    string
	CAFile     string
}

// OAuth2Config OAuth2配置
type OAuth2Config struct {
	ClientID     string
	ClientSecret string
	TokenURL     string
	Scopes       []string
}

// OAuth2Token OAuth2令牌
type OAuth2Token struct {
	AccessToken  string
	TokenType    string
	ExpiresIn    int
	RefreshToken string
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

// AbstractSource 抽象数据源基类
type AbstractSource struct {
	configuration *SourceConfiguration
	state         SourceState
	mu            sync.RWMutex
}

// NewAbstractSource 创建抽象数据源
func NewAbstractSource() *AbstractSource {
	return &AbstractSource{
		configuration: NewSourceConfiguration(),
		state:         StateUnconfigured,
	}
}

// Initialize 初始化抽象数据源
func (as *AbstractSource) Initialize(ctx context.Context) error {
	as.mu.Lock()
	defer as.mu.Unlock()

	as.state = StateConfiguring

	// 验证配置
	result := as.Validate()
	if !result.Valid {
		as.state = StateError
		return fmt.Errorf("配置验证失败: %v", result.Errors)
	}

	as.state = StateReady
	return nil
}

// Start 启动数据源
func (as *AbstractSource) Start(ctx context.Context) error {
	as.mu.Lock()
	defer as.mu.Unlock()

	if as.state != StateReady {
		return fmt.Errorf("数据源状态不正确，当前状态: %v", as.state)
	}

	as.state = StateRunning
	return nil
}

// Stop 停止数据源
func (as *AbstractSource) Stop(ctx context.Context) error {
	as.mu.Lock()
	defer as.mu.Unlock()

	as.state = StateStopped
	return nil
}

// GetState 获取数据源状态
func (as *AbstractSource) GetState() SourceState {
	as.mu.RLock()
	defer as.mu.RUnlock()
	return as.state
}

// GetConfiguration 获取配置信息
func (as *AbstractSource) GetConfiguration() *SourceConfiguration {
	as.mu.RLock()
	defer as.mu.RUnlock()
	return as.configuration
}

// SetConfiguration 设置配置
func (as *AbstractSource) SetConfiguration(config *SourceConfiguration) {
	as.mu.Lock()
	defer as.mu.Unlock()
	as.configuration = config
}

// Validate 验证数据源配置
func (as *AbstractSource) Validate() *ValidationResult {
	result := &ValidationResult{Valid: true}

	config := as.GetConfiguration()

	// 检查基本配置
	if config.Host == "" {
		result.AddError("主机地址不能为空")
	}

	if config.Port <= 0 || config.Port > 65535 {
		result.AddError("端口号必须在1-65535之间")
	}

	if config.MaxConnections <= 0 {
		result.AddError("最大连接数必须大于0")
	}

	return result
}

// SetState 设置状态
func (as *AbstractSource) SetState(state SourceState) {
	as.mu.Lock()
	defer as.mu.Unlock()
	as.state = state
}

// SourceManager 数据源管理器
type SourceManager struct {
	sources map[string]Source
	mu      sync.RWMutex
}

// NewSourceManager 创建数据源管理器
func NewSourceManager() *SourceManager {
	return &SourceManager{
		sources: make(map[string]Source),
	}
}

// RegisterSource 注册数据源
func (sm *SourceManager) RegisterSource(id string, source Source) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.sources[id]; exists {
		return fmt.Errorf("数据源 %s 已存在", id)
	}

	sm.sources[id] = source
	return nil
}

// GetSource 获取数据源
func (sm *SourceManager) GetSource(id string) (Source, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	source, exists := sm.sources[id]
	return source, exists
}

// ListSources 列出所有数据源
func (sm *SourceManager) ListSources() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	ids := make([]string, 0, len(sm.sources))
	for id := range sm.sources {
		ids = append(ids, id)
	}
	return ids
}

// RemoveSource 移除数据源
func (sm *SourceManager) RemoveSource(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.sources[id]; !exists {
		return fmt.Errorf("数据源 %s 不存在", id)
	}

	delete(sm.sources, id)
	return nil
}

// StartAllSources 启动所有数据源
func (sm *SourceManager) StartAllSources(ctx context.Context) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for id, source := range sm.sources {
		if err := source.Start(ctx); err != nil {
			return fmt.Errorf("启动数据源 %s 失败: %w", id, err)
		}
	}

	return nil
}

// StopAllSources 停止所有数据源
func (sm *SourceManager) StopAllSources(ctx context.Context) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for id, source := range sm.sources {
		if err := source.Stop(ctx); err != nil {
			return fmt.Errorf("停止数据源 %s 失败: %w", id, err)
		}
	}

	return nil
}
