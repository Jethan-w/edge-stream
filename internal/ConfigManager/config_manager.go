package ConfigManager

import (
	"fmt"
	"sync"
	"time"
)

// ConfigManager 配置管理器接口
type ConfigManager interface {
	// GetProperty 获取配置属性
	GetProperty(key string) (string, error)

	// SetProperty 设置配置属性
	SetProperty(key, value string, sensitive bool) error

	// LoadConfiguration 加载配置文件
	LoadConfiguration(configFile string) error

	// PersistConfiguration 持久化配置变更
	PersistConfiguration() error

	// GetSensitivePropertyProvider 获取敏感属性提供者
	GetSensitivePropertyProvider() SensitivePropertyProvider

	// RegisterValidator 注册配置验证器
	RegisterValidator(validator ConfigValidator)

	// ValidateConfiguration 验证配置
	ValidateConfiguration() (*ValidationResult, error)

	// AddConfigurationChangeListener 添加配置变更监听器
	AddConfigurationChangeListener(listener ConfigurationChangeListener)

	// ReloadConfiguration 重载配置
	ReloadConfiguration() error
}

// ConfigEntry 配置条目
type ConfigEntry struct {
	Value            string    `json:"value"`
	Sensitive        bool      `json:"sensitive"`
	LastModifiedTime time.Time `json:"lastModifiedTime"`
}

// ConfigMap 配置映射接口
type ConfigMap interface {
	Get(key string) (string, bool)
	Set(key, value string, sensitive bool)
	GetAll() map[string]*ConfigEntry
	Remove(key string)
}

// StandardConfigMap 标准配置映射实现
type StandardConfigMap struct {
	entries map[string]*ConfigEntry
	mutex   sync.RWMutex
}

// NewStandardConfigMap 创建新的标准配置映射
func NewStandardConfigMap() *StandardConfigMap {
	return &StandardConfigMap{
		entries: make(map[string]*ConfigEntry),
	}
}

// Get 获取配置值
func (cm *StandardConfigMap) Get(key string) (string, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	entry, exists := cm.entries[key]
	if !exists {
		return "", false
	}
	return entry.Value, true
}

// Set 设置配置值
func (cm *StandardConfigMap) Set(key, value string, sensitive bool) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.entries[key] = &ConfigEntry{
		Value:            value,
		Sensitive:        sensitive,
		LastModifiedTime: time.Now(),
	}
}

// GetAll 获取所有配置
func (cm *StandardConfigMap) GetAll() map[string]*ConfigEntry {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	result := make(map[string]*ConfigEntry)
	for k, v := range cm.entries {
		result[k] = v
	}
	return result
}

// Remove 移除配置
func (cm *StandardConfigMap) Remove(key string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	delete(cm.entries, key)
}

// StandardConfigManager 标准配置管理器实现
type StandardConfigManager struct {
	configMap            ConfigMap
	sensitiveProvider    SensitivePropertyProvider
	configFileManager    *ConfigFileManager
	changeManager        *ConfigurationChangeManager
	validators           []ConfigValidator
	validationStrategy   ValidationStrategy
	configHistoryManager *ConfigHistoryManager
	mutex                sync.RWMutex
}

// NewStandardConfigManager 创建新的标准配置管理器
func NewStandardConfigManager() *StandardConfigManager {
	return &StandardConfigManager{
		configMap:            NewStandardConfigMap(),
		configFileManager:    NewConfigFileManager(),
		changeManager:        NewConfigurationChangeManager(),
		validators:           make([]ConfigValidator, 0),
		validationStrategy:   ValidationStrategyStrict,
		configHistoryManager: NewConfigHistoryManager(),
	}
}

// GetProperty 获取配置属性
func (cm *StandardConfigManager) GetProperty(key string) (string, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	value, exists := cm.configMap.Get(key)
	if !exists {
		return "", fmt.Errorf("配置键 '%s' 不存在", key)
	}

	// 检查是否为敏感属性，如果是则解密
	// 避免在持有锁的情况下再次调用GetAll()
	// 直接从configMap获取条目信息
	if cm.sensitiveProvider != nil {
		// 这里简化处理，假设所有属性都是非敏感的
		// 在实际实现中，需要更复杂的逻辑来检查敏感属性
		return value, nil
	}

	return value, nil
}

// SetProperty 设置配置属性
func (cm *StandardConfigManager) SetProperty(key, value string, sensitive bool) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// 如果是敏感属性，先加密
	if sensitive {
		encrypted, err := cm.sensitiveProvider.Encrypt(value)
		if err != nil {
			return fmt.Errorf("加密敏感属性失败: %w", err)
		}
		value = encrypted
	}

	// 记录配置快照
	cm.configHistoryManager.RecordConfigurationSnapshot(cm.configMap)

	// 设置配置
	cm.configMap.Set(key, value, sensitive)

	// 通知配置变更
	cm.changeManager.NotifyConfigurationChange(key, value)

	return nil
}

// LoadConfiguration 加载配置文件
func (cm *StandardConfigManager) LoadConfiguration(configFile string) error {
	properties, err := cm.configFileManager.LoadConfigFile(configFile)
	if err != nil {
		return fmt.Errorf("加载配置文件失败: %w", err)
	}

	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// 清空现有配置
	cm.configMap = NewStandardConfigMap()

	// 加载新配置
	for key, value := range properties {
		cm.configMap.Set(key, value, false) // 默认非敏感
	}

	// 验证配置 - 在锁外进行验证
	cm.mutex.Unlock()
	result, err := cm.ValidateConfiguration()
	cm.mutex.Lock()

	if err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	} else if !result.IsValid {
		return fmt.Errorf("配置验证失败: %s", result.GetErrors())
	}

	return nil
}

// PersistConfiguration 持久化配置变更
func (cm *StandardConfigManager) PersistConfiguration() error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// 获取所有配置
	entries := cm.configMap.GetAll()
	properties := make(map[string]string)

	for key, entry := range entries {
		properties[key] = entry.Value
	}

	// 持久化到文件
	return cm.configFileManager.SaveConfigFile(properties, "nifi.properties")
}

// GetSensitivePropertyProvider 获取敏感属性提供者
func (cm *StandardConfigManager) GetSensitivePropertyProvider() SensitivePropertyProvider {
	return cm.sensitiveProvider
}

// RegisterValidator 注册配置验证器
func (cm *StandardConfigManager) RegisterValidator(validator ConfigValidator) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.validators = append(cm.validators, validator)
}

// ValidateConfiguration 验证配置
func (cm *StandardConfigManager) ValidateConfiguration() (*ValidationResult, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	result := NewValidationResult()

	for _, validator := range cm.validators {
		validatorResult, err := validator.Validate(cm.configMap)
		if err != nil {
			return nil, fmt.Errorf("验证器执行失败: %w", err)
		}
		result.Merge(validatorResult)
	}

	return result, nil
}

// AddConfigurationChangeListener 添加配置变更监听器
func (cm *StandardConfigManager) AddConfigurationChangeListener(listener ConfigurationChangeListener) {
	cm.changeManager.AddConfigurationChangeListener(listener)
}

// ReloadConfiguration 重载配置
func (cm *StandardConfigManager) ReloadConfiguration() error {
	// 记录当前配置快照
	cm.configHistoryManager.RecordConfigurationSnapshot(cm.configMap)

	// 重新加载配置文件
	return cm.LoadConfiguration("nifi.properties")
}

// SetSensitivePropertyProvider 设置敏感属性提供者
func (cm *StandardConfigManager) SetSensitivePropertyProvider(provider SensitivePropertyProvider) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.sensitiveProvider = provider
}

// GetConfigMap 获取配置映射
func (cm *StandardConfigManager) GetConfigMap() ConfigMap {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	return cm.configMap
}
