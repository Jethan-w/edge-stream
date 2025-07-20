package sink

import (
	"fmt"
	"log"
	"sync"
)

// CustomOutputProtocolRegistry 自定义输出协议注册表
type CustomOutputProtocolRegistry struct {
	mu sync.RWMutex

	// 注册的协议
	registeredProtocols map[string]ProtocolAdapterFactory

	// 协议元数据
	protocolMetadata map[string]*ProtocolMetadata

	// 统计信息
	stats *ProtocolRegistryStats
}

// ProtocolAdapterFactory 协议适配器工厂函数
type ProtocolAdapterFactory func(config map[string]string) (TargetAdapter, error)

// ProtocolMetadata 协议元数据
type ProtocolMetadata struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Author      string            `json:"author"`
	URL         string            `json:"url"`
	License     string            `json:"license"`
	Properties  map[string]string `json:"properties"`
	CreatedAt   string            `json:"created_at"`
}

// ProtocolRegistryStats 协议注册表统计信息
type ProtocolRegistryStats struct {
	TotalProtocols      int64 `json:"total_protocols"`
	ActiveProtocols     int64 `json:"active_protocols"`
	TotalAdapters       int64 `json:"total_adapters"`
	FailedRegistrations int64 `json:"failed_registrations"`
	mu                  sync.RWMutex
}

// NewCustomOutputProtocolRegistry 创建自定义输出协议注册表
func NewCustomOutputProtocolRegistry() *CustomOutputProtocolRegistry {
	return &CustomOutputProtocolRegistry{
		registeredProtocols: make(map[string]ProtocolAdapterFactory),
		protocolMetadata:    make(map[string]*ProtocolMetadata),
		stats:               &ProtocolRegistryStats{},
	}
}

// RegisterProtocol 注册协议
func (r *CustomOutputProtocolRegistry) RegisterProtocol(name string, factory ProtocolAdapterFactory) error {
	return r.RegisterProtocolWithMetadata(name, factory, nil)
}

// RegisterProtocolWithMetadata 注册协议（带元数据）
func (r *CustomOutputProtocolRegistry) RegisterProtocolWithMetadata(
	name string,
	factory ProtocolAdapterFactory,
	metadata *ProtocolMetadata,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 检查协议名是否已存在
	if _, exists := r.registeredProtocols[name]; exists {
		return fmt.Errorf("protocol already registered: %s", name)
	}

	// 验证协议名
	if err := r.validateProtocolName(name); err != nil {
		return fmt.Errorf("invalid protocol name: %w", err)
	}

	// 注册协议
	r.registeredProtocols[name] = factory

	// 设置默认元数据
	if metadata == nil {
		metadata = &ProtocolMetadata{
			Name:        name,
			Version:     "1.0.0",
			Description: fmt.Sprintf("Custom protocol: %s", name),
			Author:      "Unknown",
			License:     "MIT",
			Properties:  make(map[string]string),
		}
	}
	r.protocolMetadata[name] = metadata

	// 更新统计信息
	r.stats.mu.Lock()
	r.stats.TotalProtocols++
	r.stats.ActiveProtocols++
	r.stats.mu.Unlock()

	log.Printf("Registered custom protocol: %s", name)
	return nil
}

// UnregisterProtocol 注销协议
func (r *CustomOutputProtocolRegistry) UnregisterProtocol(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.registeredProtocols[name]; !exists {
		return fmt.Errorf("protocol not found: %s", name)
	}

	// 注销协议
	delete(r.registeredProtocols, name)
	delete(r.protocolMetadata, name)

	// 更新统计信息
	r.stats.mu.Lock()
	r.stats.ActiveProtocols--
	r.stats.mu.Unlock()

	log.Printf("Unregistered custom protocol: %s", name)
	return nil
}

// CreateAdapter 创建适配器
func (r *CustomOutputProtocolRegistry) CreateAdapter(protocolName string) (TargetAdapter, error) {
	return r.CreateAdapterWithConfig(protocolName, nil)
}

// CreateAdapterWithConfig 创建适配器（带配置）
func (r *CustomOutputProtocolRegistry) CreateAdapterWithConfig(
	protocolName string,
	config map[string]string,
) (TargetAdapter, error) {
	r.mu.RLock()
	factory, exists := r.registeredProtocols[protocolName]
	r.mu.RUnlock()

	if !exists {
		return nil, NewUnsupportedProtocolError(protocolName)
	}

	// 创建适配器
	adapter, err := factory(config)
	if err != nil {
		// 更新统计信息
		r.stats.mu.Lock()
		r.stats.FailedRegistrations++
		r.stats.mu.Unlock()

		return nil, fmt.Errorf("failed to create adapter for protocol %s: %w", protocolName, err)
	}

	// 更新统计信息
	r.stats.mu.Lock()
	r.stats.TotalAdapters++
	r.stats.mu.Unlock()

	return adapter, nil
}

// GetRegisteredProtocols 获取已注册的协议列表
func (r *CustomOutputProtocolRegistry) GetRegisteredProtocols() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	protocols := make([]string, 0, len(r.registeredProtocols))
	for name := range r.registeredProtocols {
		protocols = append(protocols, name)
	}

	return protocols
}

// GetProtocolMetadata 获取协议元数据
func (r *CustomOutputProtocolRegistry) GetProtocolMetadata(protocolName string) (*ProtocolMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metadata, exists := r.protocolMetadata[protocolName]
	if !exists {
		return nil, fmt.Errorf("protocol not found: %s", protocolName)
	}

	return metadata, nil
}

// GetAllProtocolMetadata 获取所有协议元数据
func (r *CustomOutputProtocolRegistry) GetAllProtocolMetadata() map[string]*ProtocolMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metadata := make(map[string]*ProtocolMetadata)
	for name, meta := range r.protocolMetadata {
		metadata[name] = meta
	}

	return metadata
}

// IsProtocolRegistered 检查协议是否已注册
func (r *CustomOutputProtocolRegistry) IsProtocolRegistered(protocolName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.registeredProtocols[protocolName]
	return exists
}

// GetStats 获取统计信息
func (r *CustomOutputProtocolRegistry) GetStats() *ProtocolRegistryStats {
	r.stats.mu.RLock()
	defer r.stats.mu.RUnlock()

	stats := *r.stats
	return &stats
}

// validateProtocolName 验证协议名
func (r *CustomOutputProtocolRegistry) validateProtocolName(name string) error {
	if name == "" {
		return fmt.Errorf("protocol name cannot be empty")
	}

	// 检查长度
	if len(name) > 50 {
		return fmt.Errorf("protocol name too long (max 50 characters)")
	}

	// 检查是否包含非法字符
	for _, char := range name {
		if !isValidProtocolNameChar(char) {
			return fmt.Errorf("protocol name contains invalid character: %c", char)
		}
	}

	// 检查是否为保留名称
	reservedNames := []string{
		"file", "filesystem", "fs",
		"database", "db", "sql",
		"queue", "mq", "message",
		"api", "rest", "http",
		"custom", "unknown",
	}

	for _, reserved := range reservedNames {
		if name == reserved {
			return fmt.Errorf("protocol name is reserved: %s", name)
		}
	}

	return nil
}

// isValidProtocolNameChar 检查字符是否为有效的协议名字符
func isValidProtocolNameChar(char rune) bool {
	return (char >= 'a' && char <= 'z') ||
		(char >= 'A' && char <= 'Z') ||
		(char >= '0' && char <= '9') ||
		char == '-' || char == '_'
}

// Clear 清空注册表
func (r *CustomOutputProtocolRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.registeredProtocols = make(map[string]ProtocolAdapterFactory)
	r.protocolMetadata = make(map[string]*ProtocolMetadata)

	// 重置统计信息
	r.stats.mu.Lock()
	r.stats = &ProtocolRegistryStats{}
	r.stats.mu.Unlock()

	log.Printf("Cleared custom protocol registry")
}

// GetProtocolCount 获取协议数量
func (r *CustomOutputProtocolRegistry) GetProtocolCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.registeredProtocols)
}

// GetAdapterCount 获取适配器数量
func (r *CustomOutputProtocolRegistry) GetAdapterCount() int64 {
	r.stats.mu.RLock()
	defer r.stats.mu.RUnlock()

	return r.stats.TotalAdapters
}
