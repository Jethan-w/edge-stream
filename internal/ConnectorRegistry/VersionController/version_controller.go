package VersionController

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// VersionController 版本控制器接口
type VersionController interface {
	// CreateVersion 创建新版本
	CreateVersion(componentID string, snapshot *FlowSnapshot) error

	// UpdateVersion 更新特定版本
	UpdateVersion(componentID, version string, snapshot *FlowSnapshot) error

	// GetVersion 获取特定版本
	GetVersion(componentID, version string) (*FlowSnapshot, error)

	// ListVersions 列出组件的所有版本
	ListVersions(componentID string) ([]string, error)

	// RollbackToVersion 回滚到指定版本
	RollbackToVersion(componentID, version string) error

	// DeleteVersion 删除特定版本
	DeleteVersion(componentID, version string) error
}

// FlowSnapshot 流程快照
type FlowSnapshot struct {
	ComponentID string                 `json:"componentId"`
	Version     string                 `json:"version"`
	Data        map[string]interface{} `json:"data"`
	CreatedAt   time.Time              `json:"createdAt"`
	CreatedBy   string                 `json:"createdBy"`
	Description string                 `json:"description"`
	Checksum    string                 `json:"checksum"`
}

// NewFlowSnapshot 创建新的流程快照
func NewFlowSnapshot(componentID string, data map[string]interface{}, createdBy, description string) *FlowSnapshot {
	snapshot := &FlowSnapshot{
		ComponentID: componentID,
		Data:        data,
		CreatedAt:   time.Now(),
		CreatedBy:   createdBy,
		Description: description,
	}

	// 计算校验和
	snapshot.Checksum = snapshot.calculateChecksum()

	// 生成版本号
	snapshot.Version = snapshot.generateVersion()

	return snapshot
}

// calculateChecksum 计算校验和
func (fs *FlowSnapshot) calculateChecksum() string {
	dataBytes, err := json.Marshal(fs.Data)
	if err != nil {
		return ""
	}

	hash := sha256.Sum256(dataBytes)
	return hex.EncodeToString(hash[:])
}

// generateVersion 生成版本号
func (fs *FlowSnapshot) generateVersion() string {
	// 使用时间戳和校验和的前8位生成版本号
	timestamp := fs.CreatedAt.Unix()
	shortChecksum := fs.Checksum[:8]
	return fmt.Sprintf("%d-%s", timestamp, shortChecksum)
}

// Serialize 序列化快照
func (fs *FlowSnapshot) Serialize() ([]byte, error) {
	return json.Marshal(fs)
}

// Deserialize 反序列化快照
func DeserializeSnapshot(data []byte) (*FlowSnapshot, error) {
	var snapshot FlowSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, err
	}
	return &snapshot, nil
}

// StandardVersionController 标准版本控制器实现
type StandardVersionController struct {
	versionHistory map[string]map[string]*FlowSnapshot
	mutex          sync.RWMutex
}

// NewStandardVersionController 创建新的标准版本控制器
func NewStandardVersionController() *StandardVersionController {
	return &StandardVersionController{
		versionHistory: make(map[string]map[string]*FlowSnapshot),
	}
}

// CreateVersion 创建新版本
func (svc *StandardVersionController) CreateVersion(componentID string, snapshot *FlowSnapshot) error {
	svc.mutex.Lock()
	defer svc.mutex.Unlock()

	// 初始化组件的版本历史
	if svc.versionHistory[componentID] == nil {
		svc.versionHistory[componentID] = make(map[string]*FlowSnapshot)
	}

	// 检查版本是否已存在
	if _, exists := svc.versionHistory[componentID][snapshot.Version]; exists {
		return fmt.Errorf("版本已存在: %s", snapshot.Version)
	}

	// 存储版本快照
	svc.versionHistory[componentID][snapshot.Version] = snapshot

	return nil
}

// UpdateVersion 更新特定版本
func (svc *StandardVersionController) UpdateVersion(componentID, version string, snapshot *FlowSnapshot) error {
	svc.mutex.Lock()
	defer svc.mutex.Unlock()

	// 检查组件是否存在
	if svc.versionHistory[componentID] == nil {
		return fmt.Errorf("组件不存在: %s", componentID)
	}

	// 检查版本是否存在
	if _, exists := svc.versionHistory[componentID][version]; !exists {
		return fmt.Errorf("版本不存在: %s", version)
	}

	// 更新版本快照
	snapshot.Version = version
	snapshot.CreatedAt = time.Now()
	svc.versionHistory[componentID][version] = snapshot

	return nil
}

// GetVersion 获取特定版本
func (svc *StandardVersionController) GetVersion(componentID, version string) (*FlowSnapshot, error) {
	svc.mutex.RLock()
	defer svc.mutex.RUnlock()

	// 检查组件是否存在
	componentVersions, exists := svc.versionHistory[componentID]
	if !exists {
		return nil, fmt.Errorf("组件不存在: %s", componentID)
	}

	// 获取特定版本
	snapshot, exists := componentVersions[version]
	if !exists {
		return nil, fmt.Errorf("版本不存在: %s", version)
	}

	return snapshot, nil
}

// ListVersions 列出组件的所有版本
func (svc *StandardVersionController) ListVersions(componentID string) ([]string, error) {
	svc.mutex.RLock()
	defer svc.mutex.RUnlock()

	// 检查组件是否存在
	componentVersions, exists := svc.versionHistory[componentID]
	if !exists {
		return nil, fmt.Errorf("组件不存在: %s", componentID)
	}

	// 收集所有版本
	versions := make([]string, 0, len(componentVersions))
	for version := range componentVersions {
		versions = append(versions, version)
	}

	return versions, nil
}

// RollbackToVersion 回滚到指定版本
func (svc *StandardVersionController) RollbackToVersion(componentID, version string) error {
	svc.mutex.Lock()
	defer svc.mutex.Unlock()

	// 检查组件是否存在
	componentVersions, exists := svc.versionHistory[componentID]
	if !exists {
		return fmt.Errorf("组件不存在: %s", componentID)
	}

	// 检查版本是否存在
	if _, exists := componentVersions[version]; !exists {
		return fmt.Errorf("版本不存在: %s", version)
	}

	// 这里应该实现回滚逻辑
	// 简化实现，只记录回滚操作
	fmt.Printf("回滚组件 %s 到版本 %s\n", componentID, version)

	return nil
}

// DeleteVersion 删除特定版本
func (svc *StandardVersionController) DeleteVersion(componentID, version string) error {
	svc.mutex.Lock()
	defer svc.mutex.Unlock()

	// 检查组件是否存在
	componentVersions, exists := svc.versionHistory[componentID]
	if !exists {
		return fmt.Errorf("组件不存在: %s", componentID)
	}

	// 检查版本是否存在
	if _, exists := componentVersions[version]; !exists {
		return fmt.Errorf("版本不存在: %s", version)
	}

	// 删除版本
	delete(componentVersions, version)

	// 如果组件没有版本了，删除组件记录
	if len(componentVersions) == 0 {
		delete(svc.versionHistory, componentID)
	}

	return nil
}

// GetLatestVersion 获取最新版本
func (svc *StandardVersionController) GetLatestVersion(componentID string) (*FlowSnapshot, error) {
	svc.mutex.RLock()
	defer svc.mutex.RUnlock()

	// 检查组件是否存在
	componentVersions, exists := svc.versionHistory[componentID]
	if !exists {
		return nil, fmt.Errorf("组件不存在: %s", componentID)
	}

	// 找到最新版本
	var latestSnapshot *FlowSnapshot
	var latestTime time.Time

	for _, snapshot := range componentVersions {
		if latestSnapshot == nil || snapshot.CreatedAt.After(latestTime) {
			latestSnapshot = snapshot
			latestTime = snapshot.CreatedAt
		}
	}

	if latestSnapshot == nil {
		return nil, fmt.Errorf("没有找到版本")
	}

	return latestSnapshot, nil
}

// VersionCompatibilityChecker 版本兼容性检查器
type VersionCompatibilityChecker struct {
	mutex sync.RWMutex
}

// NewVersionCompatibilityChecker 创建新的版本兼容性检查器
func NewVersionCompatibilityChecker() *VersionCompatibilityChecker {
	return &VersionCompatibilityChecker{}
}

// IsCompatible 检查版本兼容性
func (vcc *VersionCompatibilityChecker) IsCompatible(currentVersion, newVersion string) bool {
	vcc.mutex.RLock()
	defer vcc.mutex.RUnlock()

	// 解析语义化版本
	current := vcc.parseSemanticVersion(currentVersion)
	newVer := vcc.parseSemanticVersion(newVersion)

	// 主版本号不同，不兼容
	if current.Major != newVer.Major {
		return false
	}

	// 次版本号更高，兼容
	return newVer.Minor >= current.Minor
}

// SemanticVersion 语义化版本结构
type SemanticVersion struct {
	Major int
	Minor int
	Patch int
	Pre   string
	Build string
}

// parseSemanticVersion 解析语义化版本
func (vcc *VersionCompatibilityChecker) parseSemanticVersion(version string) *SemanticVersion {
	// 简化的语义化版本解析
	// 实际应该使用更完整的解析逻辑
	return &SemanticVersion{
		Major: 1,
		Minor: 0,
		Patch: 0,
	}
}

// VersionControlStrategy 版本控制策略枚举
type VersionControlStrategy string

const (
	VersionControlStrategySemantic  VersionControlStrategy = "SEMANTIC_VERSIONING"
	VersionControlStrategyHash      VersionControlStrategy = "HASH_BASED"
	VersionControlStrategyTimestamp VersionControlStrategy = "TIMESTAMP_BASED"
)

// VersionManager 版本管理器
type VersionManager struct {
	controller VersionController
	checker    *VersionCompatibilityChecker
	strategy   VersionControlStrategy
	mutex      sync.RWMutex
}

// NewVersionManager 创建新的版本管理器
func NewVersionManager(controller VersionController) *VersionManager {
	return &VersionManager{
		controller: controller,
		checker:    NewVersionCompatibilityChecker(),
		strategy:   VersionControlStrategySemantic,
	}
}

// SetStrategy 设置版本控制策略
func (vm *VersionManager) SetStrategy(strategy VersionControlStrategy) {
	vm.mutex.Lock()
	defer vm.mutex.Unlock()

	vm.strategy = strategy
}

// GetStrategy 获取版本控制策略
func (vm *VersionManager) GetStrategy() VersionControlStrategy {
	vm.mutex.RLock()
	defer vm.mutex.RUnlock()

	return vm.strategy
}

// CreateVersionWithStrategy 使用指定策略创建版本
func (vm *VersionManager) CreateVersionWithStrategy(componentID string, data map[string]interface{}, createdBy, description string) error {
	vm.mutex.Lock()
	defer vm.mutex.Unlock()

	var snapshot *FlowSnapshot

	switch vm.strategy {
	case VersionControlStrategySemantic:
		snapshot = vm.createSemanticVersion(componentID, data, createdBy, description)
	case VersionControlStrategyHash:
		snapshot = vm.createHashBasedVersion(componentID, data, createdBy, description)
	case VersionControlStrategyTimestamp:
		snapshot = vm.createTimestampBasedVersion(componentID, data, createdBy, description)
	default:
		return fmt.Errorf("不支持的版本控制策略: %s", vm.strategy)
	}

	return vm.controller.CreateVersion(componentID, snapshot)
}

// createSemanticVersion 创建语义化版本
func (vm *VersionManager) createSemanticVersion(componentID string, data map[string]interface{}, createdBy, description string) *FlowSnapshot {
	// 获取当前版本
	currentVersions, err := vm.controller.ListVersions(componentID)
	if err != nil {
		// 如果没有版本，创建初始版本
		return NewFlowSnapshot(componentID, data, createdBy, description)
	}
	currentVersion := currentVersions[0]
	// 解析当前版本号
	current := vm.checker.parseSemanticVersion(currentVersion)

	// 创建新版本号
	newVersion := &SemanticVersion{
		Major: current.Major,
		Minor: current.Minor + 1,
		Patch: 0,
	}

	snapshot := NewFlowSnapshot(componentID, data, createdBy, description)
	snapshot.Version = fmt.Sprintf("%d.%d.%d", newVersion.Major, newVersion.Minor, newVersion.Patch)

	return snapshot
}

// createHashBasedVersion 创建基于哈希的版本
func (vm *VersionManager) createHashBasedVersion(componentID string, data map[string]interface{}, createdBy, description string) *FlowSnapshot {
	snapshot := NewFlowSnapshot(componentID, data, createdBy, description)
	// 使用校验和作为版本号
	snapshot.Version = snapshot.Checksum[:16]
	return snapshot
}

// createTimestampBasedVersion 创建基于时间戳的版本
func (vm *VersionManager) createTimestampBasedVersion(componentID string, data map[string]interface{}, createdBy, description string) *FlowSnapshot {
	snapshot := NewFlowSnapshot(componentID, data, createdBy, description)
	// 使用时间戳作为版本号
	snapshot.Version = fmt.Sprintf("%d", time.Now().Unix())
	return snapshot
}

// ValidateVersionCompatibility 验证版本兼容性
func (vm *VersionManager) ValidateVersionCompatibility(componentID, newVersion string) error {
	vm.mutex.RLock()
	defer vm.mutex.RUnlock()

	// 获取当前版本
	currentSnapshots, err := vm.controller.ListVersions(componentID)
	if err != nil {
		// 如果没有当前版本，新版本总是兼容的
		return nil
	}
	currentSnapshot := currentSnapshots[0]
	// 检查兼容性
	if !vm.checker.IsCompatible(currentSnapshot, newVersion) {
		return fmt.Errorf("版本不兼容: 当前版本 %s, 新版本 %s", currentSnapshot, newVersion)
	}

	return nil
}
