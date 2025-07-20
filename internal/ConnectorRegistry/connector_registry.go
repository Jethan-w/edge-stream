package ConnectorRegistry

import (
	"fmt"
	"sync"
)

// ConnectorRegistry 连接器注册表接口
type ConnectorRegistry interface {
	// RegisterConnector 注册连接器
	RegisterConnector(descriptor ConnectorDescriptor) error

	// UnregisterConnector 注销连接器
	UnregisterConnector(identifier string) error

	// GetConnector 获取连接器
	GetConnector(identifier string) (ConnectorDescriptor, error)

	// ListConnectors 列出所有连接器
	ListConnectors() []ConnectorDescriptor

	// ListConnectorsByType 根据类型列出连接器
	ListConnectorsByType(componentType ComponentType) []ConnectorDescriptor

	// SearchConnectors 搜索连接器
	SearchConnectors(keyword string) []ConnectorDescriptor

	// GetConnectorVersions 获取连接器版本
	GetConnectorVersions(identifier string) ([]string, error)

	// UpdateConnector 更新连接器
	UpdateConnector(descriptor ConnectorDescriptor) error
}

// ComponentType 组件类型枚举
type ComponentType string

const (
	ComponentTypeProcessor         ComponentType = "PROCESSOR"
	ComponentTypeControllerService ComponentType = "CONTROLLER_SERVICE"
	ComponentTypeReportingTask     ComponentType = "REPORTING_TASK"
	ComponentTypeFlowController    ComponentType = "FLOW_CONTROLLER"
	ComponentTypeSource            ComponentType = "SOURCE"
	ComponentTypeSink              ComponentType = "SINK"
)

// ConnectorDescriptor 连接器描述符接口
type ConnectorDescriptor interface {
	// GetIdentifier 获取组件唯一标识
	GetIdentifier() string

	// GetName 获取组件名称
	GetName() string

	// GetType 获取组件类型
	GetType() ComponentType

	// GetVersion 获取组件版本
	GetVersion() string

	// GetDescription 获取组件描述
	GetDescription() string

	// GetAuthor 获取组件作者
	GetAuthor() string

	// GetTags 获取组件标签
	GetTags() []string

	// GetMetadata 获取组件元数据
	GetMetadata() map[string]string
}

// NarBundleConnectorDescriptor NAR包形式的连接器描述符
type NarBundleConnectorDescriptor struct {
	Bundle      *Bundle
	Identifier  string
	Name        string
	Type        ComponentType
	Version     string
	Description string
	Author      string
	Tags        []string
	Metadata    map[string]string
}

// NewNarBundleConnectorDescriptor 创建新的NAR包连接器描述符
func NewNarBundleConnectorDescriptor(bundle *Bundle, name string, componentType ComponentType) *NarBundleConnectorDescriptor {
	return &NarBundleConnectorDescriptor{
		Bundle:      bundle,
		Identifier:  fmt.Sprintf("%s:%s:%s", bundle.Group, bundle.Artifact, bundle.Version),
		Name:        name,
		Type:        componentType,
		Version:     bundle.Version,
		Description: "",
		Author:      "",
		Tags:        make([]string, 0),
		Metadata:    make(map[string]string),
	}
}

// GetIdentifier 获取组件唯一标识
func (n *NarBundleConnectorDescriptor) GetIdentifier() string {
	return n.Identifier
}

// GetName 获取组件名称
func (n *NarBundleConnectorDescriptor) GetName() string {
	return n.Name
}

// GetType 获取组件类型
func (n *NarBundleConnectorDescriptor) GetType() ComponentType {
	return n.Type
}

// GetVersion 获取组件版本
func (n *NarBundleConnectorDescriptor) GetVersion() string {
	return n.Version
}

// GetDescription 获取组件描述
func (n *NarBundleConnectorDescriptor) GetDescription() string {
	return n.Description
}

// GetAuthor 获取组件作者
func (n *NarBundleConnectorDescriptor) GetAuthor() string {
	return n.Author
}

// GetTags 获取组件标签
func (n *NarBundleConnectorDescriptor) GetTags() []string {
	return n.Tags
}

// GetMetadata 获取组件元数据
func (n *NarBundleConnectorDescriptor) GetMetadata() map[string]string {
	return n.Metadata
}

// SetDescription 设置组件描述
func (n *NarBundleConnectorDescriptor) SetDescription(description string) {
	n.Description = description
}

// SetAuthor 设置组件作者
func (n *NarBundleConnectorDescriptor) SetAuthor(author string) {
	n.Author = author
}

// AddTag 添加标签
func (n *NarBundleConnectorDescriptor) AddTag(tag string) {
	n.Tags = append(n.Tags, tag)
}

// SetMetadata 设置元数据
func (n *NarBundleConnectorDescriptor) SetMetadata(key, value string) {
	if n.Metadata == nil {
		n.Metadata = make(map[string]string)
	}
	n.Metadata[key] = value
}

// StandardConnectorRegistry 标准连接器注册表实现
type StandardConnectorRegistry struct {
	registeredConnectors map[string]ConnectorDescriptor
	bundleRepository     BundleRepository
	versionController    VersionController
	extensionMapper      ExtensionMappingProvider
	securityValidator    SecurityValidator
	mutex                sync.RWMutex
}

// NewStandardConnectorRegistry 创建新的标准连接器注册表
func NewStandardConnectorRegistry() *StandardConnectorRegistry {
	return &StandardConnectorRegistry{
		registeredConnectors: make(map[string]ConnectorDescriptor),
		bundleRepository:     NewFileSystemBundleRepository(),
		versionController:    NewStandardVersionController(),
		extensionMapper:      NewJsonExtensionMappingProvider(),
		securityValidator:    NewComponentSecurityValidator(),
	}
}

// RegisterConnector 注册连接器
func (scr *StandardConnectorRegistry) RegisterConnector(descriptor ConnectorDescriptor) error {
	scr.mutex.Lock()
	defer scr.mutex.Unlock()

	// 验证连接器是否已存在
	if _, exists := scr.registeredConnectors[descriptor.GetIdentifier()]; exists {
		return fmt.Errorf("连接器已存在: %s", descriptor.GetIdentifier())
	}

	// 如果是 NAR 包形式的连接器，进行安全验证
	if narDescriptor, ok := descriptor.(*NarBundleConnectorDescriptor); ok {
		if err := scr.securityValidator.ValidateComponent(narDescriptor.Bundle); err != nil {
			return fmt.Errorf("组件安全验证失败: %w", err)
		}

		// 部署 NAR 包到仓库
		if err := scr.bundleRepository.DeployBundle(narDescriptor.Bundle); err != nil {
			return fmt.Errorf("部署NAR包失败: %w", err)
		}

		// 注册扩展映射
		scr.extensionMapper.RegisterExtension(string(descriptor.GetType()), narDescriptor.Bundle.ID, descriptor.GetName())
	}

	// 注册连接器
	scr.registeredConnectors[descriptor.GetIdentifier()] = descriptor

	// 触发注册事件
	scr.notifyConnectorRegistered(descriptor)

	return nil
}

// UnregisterConnector 注销连接器
func (scr *StandardConnectorRegistry) UnregisterConnector(identifier string) error {
	scr.mutex.Lock()
	defer scr.mutex.Unlock()

	descriptor, exists := scr.registeredConnectors[identifier]
	if !exists {
		return fmt.Errorf("连接器不存在: %s", identifier)
	}

	// 如果是 NAR 包形式的连接器，从仓库卸载
	if narDescriptor, ok := descriptor.(*NarBundleConnectorDescriptor); ok {
		if err := scr.bundleRepository.UndeployBundle(narDescriptor.Bundle.ID, narDescriptor.Bundle.Version); err != nil {
			return fmt.Errorf("卸载NAR包失败: %w", err)
		}
	}

	// 注销连接器
	delete(scr.registeredConnectors, identifier)

	// 触发注销事件
	scr.notifyConnectorUnregistered(descriptor)

	return nil
}

// GetConnector 获取连接器
func (scr *StandardConnectorRegistry) GetConnector(identifier string) (ConnectorDescriptor, error) {
	scr.mutex.RLock()
	defer scr.mutex.RUnlock()

	descriptor, exists := scr.registeredConnectors[identifier]
	if !exists {
		return nil, fmt.Errorf("连接器不存在: %s", identifier)
	}

	return descriptor, nil
}

// ListConnectors 列出所有连接器
func (scr *StandardConnectorRegistry) ListConnectors() []ConnectorDescriptor {
	scr.mutex.RLock()
	defer scr.mutex.RUnlock()

	connectors := make([]ConnectorDescriptor, 0, len(scr.registeredConnectors))
	for _, descriptor := range scr.registeredConnectors {
		connectors = append(connectors, descriptor)
	}

	return connectors
}

// ListConnectorsByType 根据类型列出连接器
func (scr *StandardConnectorRegistry) ListConnectorsByType(componentType ComponentType) []ConnectorDescriptor {
	scr.mutex.RLock()
	defer scr.mutex.RUnlock()

	var connectors []ConnectorDescriptor
	for _, descriptor := range scr.registeredConnectors {
		if descriptor.GetType() == componentType {
			connectors = append(connectors, descriptor)
		}
	}

	return connectors
}

// SearchConnectors 搜索连接器
func (scr *StandardConnectorRegistry) SearchConnectors(keyword string) []ConnectorDescriptor {
	scr.mutex.RLock()
	defer scr.mutex.RUnlock()

	var results []ConnectorDescriptor
	for _, descriptor := range scr.registeredConnectors {
		// 简单的关键词匹配
		if scr.matchesKeyword(descriptor, keyword) {
			results = append(results, descriptor)
		}
	}

	return results
}

// GetConnectorVersions 获取连接器版本
func (scr *StandardConnectorRegistry) GetConnectorVersions(identifier string) ([]string, error) {
	scr.mutex.RLock()
	defer scr.mutex.RUnlock()

	_, exists := scr.registeredConnectors[identifier]
	if !exists {
		return nil, fmt.Errorf("连接器不存在: %s", identifier)
	}

	// 从版本控制器获取版本历史
	return scr.versionController.ListVersions(identifier)
}

// UpdateConnector 更新连接器
func (scr *StandardConnectorRegistry) UpdateConnector(descriptor ConnectorDescriptor) error {
	scr.mutex.Lock()
	defer scr.mutex.Unlock()

	identifier := descriptor.GetIdentifier()
	if _, exists := scr.registeredConnectors[identifier]; !exists {
		return fmt.Errorf("连接器不存在: %s", identifier)
	}

	// 更新连接器
	scr.registeredConnectors[identifier] = descriptor

	// 触发更新事件
	scr.notifyConnectorUpdated(descriptor)

	return nil
}

// matchesKeyword 检查连接器是否匹配关键词
func (scr *StandardConnectorRegistry) matchesKeyword(descriptor ConnectorDescriptor, keyword string) bool {
	// 检查名称
	if scr.containsIgnoreCase(descriptor.GetName(), keyword) {
		return true
	}

	// 检查描述
	if scr.containsIgnoreCase(descriptor.GetDescription(), keyword) {
		return true
	}

	// 检查标签
	for _, tag := range descriptor.GetTags() {
		if scr.containsIgnoreCase(tag, keyword) {
			return true
		}
	}

	// 检查作者
	if scr.containsIgnoreCase(descriptor.GetAuthor(), keyword) {
		return true
	}

	return false
}

// containsIgnoreCase 忽略大小写的字符串包含检查
func (scr *StandardConnectorRegistry) containsIgnoreCase(s, substr string) bool {
	// 这里简化实现，实际应该使用更高效的字符串匹配
	return len(s) >= len(substr) &&
		(len(substr) == 0 || scr.indexOfIgnoreCase(s, substr) >= 0)
}

// indexOfIgnoreCase 忽略大小写的字符串查找
func (scr *StandardConnectorRegistry) indexOfIgnoreCase(s, substr string) int {
	// 这里简化实现，实际应该使用更高效的算法
	for i := 0; i <= len(s)-len(substr); i++ {
		if scr.equalsIgnoreCase(s[i:i+len(substr)], substr) {
			return i
		}
	}
	return -1
}

// equalsIgnoreCase 忽略大小写的字符串比较
func (scr *StandardConnectorRegistry) equalsIgnoreCase(s1, s2 string) bool {
	if len(s1) != len(s2) {
		return false
	}
	for i := 0; i < len(s1); i++ {
		c1 := s1[i]
		c2 := s2[i]
		if c1 != c2 && scr.toLower(c1) != scr.toLower(c2) {
			return false
		}
	}
	return true
}

// toLower 字符转小写
func (scr *StandardConnectorRegistry) toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	return c
}

// notifyConnectorRegistered 通知连接器注册事件
func (scr *StandardConnectorRegistry) notifyConnectorRegistered(descriptor ConnectorDescriptor) {
	// 这里可以添加事件通知逻辑
	fmt.Printf("连接器已注册: %s (%s)\n", descriptor.GetName(), descriptor.GetIdentifier())
}

// notifyConnectorUnregistered 通知连接器注销事件
func (scr *StandardConnectorRegistry) notifyConnectorUnregistered(descriptor ConnectorDescriptor) {
	// 这里可以添加事件通知逻辑
	fmt.Printf("连接器已注销: %s (%s)\n", descriptor.GetName(), descriptor.GetIdentifier())
}

// notifyConnectorUpdated 通知连接器更新事件
func (scr *StandardConnectorRegistry) notifyConnectorUpdated(descriptor ConnectorDescriptor) {
	// 这里可以添加事件通知逻辑
	fmt.Printf("连接器已更新: %s (%s)\n", descriptor.GetName(), descriptor.GetIdentifier())
}

// SetBundleRepository 设置Bundle仓库
func (scr *StandardConnectorRegistry) SetBundleRepository(repository BundleRepository) {
	scr.mutex.Lock()
	defer scr.mutex.Unlock()

	scr.bundleRepository = repository
}

// SetVersionController 设置版本控制器
func (scr *StandardConnectorRegistry) SetVersionController(controller VersionController) {
	scr.mutex.Lock()
	defer scr.mutex.Unlock()

	scr.versionController = controller
}

// SetExtensionMapper 设置扩展映射提供者
func (scr *StandardConnectorRegistry) SetExtensionMapper(mapper ExtensionMappingProvider) {
	scr.mutex.Lock()
	defer scr.mutex.Unlock()

	scr.extensionMapper = mapper
}

// SetSecurityValidator 设置安全验证器
func (scr *StandardConnectorRegistry) SetSecurityValidator(validator SecurityValidator) {
	scr.mutex.Lock()
	defer scr.mutex.Unlock()

	scr.securityValidator = validator
}

// GetRegisteredConnectorsCount 获取已注册连接器数量
func (scr *StandardConnectorRegistry) GetRegisteredConnectorsCount() int {
	scr.mutex.RLock()
	defer scr.mutex.RUnlock()

	return len(scr.registeredConnectors)
}
