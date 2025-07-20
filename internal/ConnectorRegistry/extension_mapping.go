package ConnectorRegistry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// ExtensionMapping 扩展映射结构
type ExtensionMapping struct {
	ComponentType string            `json:"componentType"`
	BundleID      string            `json:"bundleId"`
	ClassName     string            `json:"className"`
	Metadata      map[string]string `json:"metadata"`
}

// NewExtensionMapping 创建新的扩展映射
func NewExtensionMapping(componentType, bundleID, className string) *ExtensionMapping {
	return &ExtensionMapping{
		ComponentType: componentType,
		BundleID:      bundleID,
		ClassName:     className,
		Metadata:      make(map[string]string),
	}
}

// SetMetadata 设置元数据
func (em *ExtensionMapping) SetMetadata(key, value string) {
	if em.Metadata == nil {
		em.Metadata = make(map[string]string)
	}
	em.Metadata[key] = value
}

// GetMetadata 获取元数据
func (em *ExtensionMapping) GetMetadata(key string) (string, bool) {
	if em.Metadata == nil {
		return "", false
	}
	value, exists := em.Metadata[key]
	return value, exists
}

// ExtensionMappingProvider 扩展映射提供者接口
type ExtensionMappingProvider interface {
	// GetExtensionMapping 获取 NAR 包的扩展映射
	GetExtensionMapping(bundleID string) (*ExtensionMapping, error)

	// UpdateExtensionMapping 更新扩展映射
	UpdateExtensionMapping(bundleID string, mapping *ExtensionMapping) error

	// RegisterExtension 注册组件类型与 NAR 包的映射关系
	RegisterExtension(componentType, bundleID, className string) error

	// UnregisterExtension 注销扩展映射
	UnregisterExtension(bundleID string) error

	// ListExtensions 列出所有扩展映射
	ListExtensions() ([]*ExtensionMapping, error)

	// GetExtensionsByType 根据组件类型获取扩展映射
	GetExtensionsByType(componentType string) ([]*ExtensionMapping, error)
}

// JsonExtensionMappingProvider 基于 JSON 的扩展映射实现
type JsonExtensionMappingProvider struct {
	mappings           map[string]*ExtensionMapping
	mappingStoragePath string
	mutex              sync.RWMutex
}

// NewJsonExtensionMappingProvider 创建新的JSON扩展映射提供者
func NewJsonExtensionMappingProvider() *JsonExtensionMappingProvider {
	return &JsonExtensionMappingProvider{
		mappings:           make(map[string]*ExtensionMapping),
		mappingStoragePath: "extensions",
	}
}

// SetMappingStoragePath 设置映射存储路径
func (jemp *JsonExtensionMappingProvider) SetMappingStoragePath(path string) {
	jemp.mutex.Lock()
	defer jemp.mutex.Unlock()

	jemp.mappingStoragePath = path
}

// GetExtensionMapping 获取 NAR 包的扩展映射
func (jemp *JsonExtensionMappingProvider) GetExtensionMapping(bundleID string) (*ExtensionMapping, error) {
	jemp.mutex.RLock()
	defer jemp.mutex.RUnlock()

	mapping, exists := jemp.mappings[bundleID]
	if !exists {
		return nil, fmt.Errorf("扩展映射不存在: %s", bundleID)
	}

	return mapping, nil
}

// UpdateExtensionMapping 更新扩展映射
func (jemp *JsonExtensionMappingProvider) UpdateExtensionMapping(bundleID string, mapping *ExtensionMapping) error {
	jemp.mutex.Lock()
	defer jemp.mutex.Unlock()

	// 更新内存中的映射
	jemp.mappings[bundleID] = mapping

	// 持久化到文件
	return jemp.persistMappingToFile(mapping)
}

// RegisterExtension 注册组件类型与 NAR 包的映射关系
func (jemp *JsonExtensionMappingProvider) RegisterExtension(componentType, bundleID, className string) error {
	jemp.mutex.Lock()
	defer jemp.mutex.Unlock()

	mapping := NewExtensionMapping(componentType, bundleID, className)
	jemp.mappings[bundleID] = mapping

	// 持久化映射关系
	return jemp.persistMappingToFile(mapping)
}

// UnregisterExtension 注销扩展映射
func (jemp *JsonExtensionMappingProvider) UnregisterExtension(bundleID string) error {
	jemp.mutex.Lock()
	defer jemp.mutex.Unlock()

	// 从内存中移除
	delete(jemp.mappings, bundleID)

	// 删除映射文件
	mappingFile := filepath.Join(jemp.mappingStoragePath, bundleID+".json")
	return os.Remove(mappingFile)
}

// ListExtensions 列出所有扩展映射
func (jemp *JsonExtensionMappingProvider) ListExtensions() ([]*ExtensionMapping, error) {
	jemp.mutex.RLock()
	defer jemp.mutex.RUnlock()

	mappings := make([]*ExtensionMapping, 0, len(jemp.mappings))
	for _, mapping := range jemp.mappings {
		mappings = append(mappings, mapping)
	}

	return mappings, nil
}

// GetExtensionsByType 根据组件类型获取扩展映射
func (jemp *JsonExtensionMappingProvider) GetExtensionsByType(componentType string) ([]*ExtensionMapping, error) {
	jemp.mutex.RLock()
	defer jemp.mutex.RUnlock()

	var mappings []*ExtensionMapping
	for _, mapping := range jemp.mappings {
		if mapping.ComponentType == componentType {
			mappings = append(mappings, mapping)
		}
	}

	return mappings, nil
}

// persistMappingToFile 持久化映射到文件
func (jemp *JsonExtensionMappingProvider) persistMappingToFile(mapping *ExtensionMapping) error {
	// 确保目录存在
	if err := os.MkdirAll(jemp.mappingStoragePath, 0755); err != nil {
		return fmt.Errorf("创建映射目录失败: %w", err)
	}

	// 序列化映射
	mappingBytes, err := json.MarshalIndent(mapping, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化映射失败: %w", err)
	}

	// 写入文件
	mappingFile := filepath.Join(jemp.mappingStoragePath, mapping.BundleID+".json")
	if err := os.WriteFile(mappingFile, mappingBytes, 0644); err != nil {
		return fmt.Errorf("写入映射文件失败: %w", err)
	}

	return nil
}

// LoadMappingsFromStorage 从存储中加载映射
func (jemp *JsonExtensionMappingProvider) LoadMappingsFromStorage() error {
	jemp.mutex.Lock()
	defer jemp.mutex.Unlock()

	// 检查目录是否存在
	if _, err := os.Stat(jemp.mappingStoragePath); os.IsNotExist(err) {
		return nil // 目录不存在，没有映射需要加载
	}

	// 读取目录中的所有JSON文件
	files, err := os.ReadDir(jemp.mappingStoragePath)
	if err != nil {
		return fmt.Errorf("读取映射目录失败: %w", err)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			mappingFile := filepath.Join(jemp.mappingStoragePath, file.Name())
			mappingBytes, err := os.ReadFile(mappingFile)
			if err != nil {
				fmt.Printf("读取映射文件失败: %s, 错误: %v\n", mappingFile, err)
				continue
			}

			var mapping ExtensionMapping
			if err := json.Unmarshal(mappingBytes, &mapping); err != nil {
				fmt.Printf("解析映射文件失败: %s, 错误: %v\n", mappingFile, err)
				continue
			}

			jemp.mappings[mapping.BundleID] = &mapping
		}
	}

	return nil
}

// DynamicComponentDiscovery 动态组件发现器
type DynamicComponentDiscovery struct {
	mappingProvider      ExtensionMappingProvider
	extensionClassLoader interface{} // 这里应该是类加载器，Go中简化处理
	mutex                sync.RWMutex
}

// NewDynamicComponentDiscovery 创建新的动态组件发现器
func NewDynamicComponentDiscovery(mappingProvider ExtensionMappingProvider) *DynamicComponentDiscovery {
	return &DynamicComponentDiscovery{
		mappingProvider: mappingProvider,
	}
}

// DiscoverComponents 发现组件
func (dcd *DynamicComponentDiscovery) DiscoverComponents(bundleID string) ([]string, error) {
	dcd.mutex.RLock()
	defer dcd.mutex.RUnlock()

	// 获取扩展映射
	mapping, err := dcd.mappingProvider.GetExtensionMapping(bundleID)
	if err != nil {
		return nil, fmt.Errorf("获取扩展映射失败: %w", err)
	}

	// 这里应该使用类加载器加载组件
	// 简化实现，返回组件类名
	components := []string{mapping.ClassName}

	return components, nil
}

// DiscoverComponentsByType 根据类型发现组件
func (dcd *DynamicComponentDiscovery) DiscoverComponentsByType(componentType string) ([]string, error) {
	dcd.mutex.RLock()
	defer dcd.mutex.RUnlock()

	// 获取指定类型的所有扩展映射
	mappings, err := dcd.mappingProvider.GetExtensionsByType(componentType)
	if err != nil {
		return nil, fmt.Errorf("获取扩展映射失败: %w", err)
	}

	var components []string
	for _, mapping := range mappings {
		components = append(components, mapping.ClassName)
	}

	return components, nil
}

// ComponentDiscoveryListener 组件发现监听器接口
type ComponentDiscoveryListener interface {
	// OnComponentDiscovered 组件发现回调
	OnComponentDiscovered(componentType, className, bundleID string)

	// OnComponentRemoved 组件移除回调
	OnComponentRemoved(componentType, className, bundleID string)
}

// ComponentDiscoveryStrategy 组件发现策略接口
type ComponentDiscoveryStrategy interface {
	// DiscoverComponents 发现组件
	DiscoverComponents(loader interface{}) ([]string, error)

	// RegisterDiscoveryListener 注册发现监听器
	RegisterDiscoveryListener(listener ComponentDiscoveryListener)
}

// ServiceLoaderDiscoveryStrategy 基于服务加载器的发现策略
type ServiceLoaderDiscoveryStrategy struct {
	listeners []ComponentDiscoveryListener
	mutex     sync.RWMutex
}

// NewServiceLoaderDiscoveryStrategy 创建新的服务加载器发现策略
func NewServiceLoaderDiscoveryStrategy() *ServiceLoaderDiscoveryStrategy {
	return &ServiceLoaderDiscoveryStrategy{
		listeners: make([]ComponentDiscoveryListener, 0),
	}
}

// DiscoverComponents 发现组件
func (slds *ServiceLoaderDiscoveryStrategy) DiscoverComponents(loader interface{}) ([]string, error) {
	slds.mutex.RLock()
	defer slds.mutex.RUnlock()

	// 这里应该使用服务加载器发现组件
	// 简化实现，返回空列表
	components := []string{}

	// 通知监听器
	for _, listener := range slds.listeners {
		for _, component := range components {
			listener.OnComponentDiscovered("PROCESSOR", component, "unknown")
		}
	}

	return components, nil
}

// RegisterDiscoveryListener 注册发现监听器
func (slds *ServiceLoaderDiscoveryStrategy) RegisterDiscoveryListener(listener ComponentDiscoveryListener) {
	slds.mutex.Lock()
	defer slds.mutex.Unlock()

	slds.listeners = append(slds.listeners, listener)
}

// ExtensionRegistry 扩展注册表
type ExtensionRegistry struct {
	mappingProvider ExtensionMappingProvider
	discovery       *DynamicComponentDiscovery
	mutex           sync.RWMutex
}

// NewExtensionRegistry 创建新的扩展注册表
func NewExtensionRegistry() *ExtensionRegistry {
	mappingProvider := NewJsonExtensionMappingProvider()
	discovery := NewDynamicComponentDiscovery(mappingProvider)

	return &ExtensionRegistry{
		mappingProvider: mappingProvider,
		discovery:       discovery,
	}
}

// RegisterExtension 注册扩展
func (er *ExtensionRegistry) RegisterExtension(componentType, bundleID, className string) error {
	er.mutex.Lock()
	defer er.mutex.Unlock()

	// 注册扩展映射
	if err := er.mappingProvider.RegisterExtension(componentType, bundleID, className); err != nil {
		return fmt.Errorf("注册扩展映射失败: %w", err)
	}

	// 发现组件
	components, err := er.discovery.DiscoverComponents(bundleID)
	if err != nil {
		return fmt.Errorf("发现组件失败: %w", err)
	}

	fmt.Printf("发现组件: %v\n", components)

	return nil
}

// GetExtensions 获取扩展
func (er *ExtensionRegistry) GetExtensions(componentType string) ([]*ExtensionMapping, error) {
	er.mutex.RLock()
	defer er.mutex.RUnlock()

	return er.mappingProvider.GetExtensionsByType(componentType)
}

// UnregisterExtension 注销扩展
func (er *ExtensionRegistry) UnregisterExtension(bundleID string) error {
	er.mutex.Lock()
	defer er.mutex.Unlock()

	return er.mappingProvider.UnregisterExtension(bundleID)
}

// LoadExtensions 加载扩展
func (er *ExtensionRegistry) LoadExtensions() error {
	er.mutex.Lock()
	defer er.mutex.Unlock()

	// 从存储中加载映射
	if jsonProvider, ok := er.mappingProvider.(*JsonExtensionMappingProvider); ok {
		return jsonProvider.LoadMappingsFromStorage()
	}

	return nil
}
