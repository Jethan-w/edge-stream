package source

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"
)

// ConfigManager 配置管理器
type ConfigManager struct {
	Configs         map[string]*SourceConfiguration
	ConfigParser    *SourceConfigParser
	ConfigValidator *DataSourceValidator
	ConfigNotifier  *ConfigChangeNotifier
	mu              sync.RWMutex
}

// NewConfigManager 创建配置管理器
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		Configs:         make(map[string]*SourceConfiguration),
		ConfigParser:    NewSourceConfigParser(),
		ConfigValidator: NewDataSourceValidator(),
		ConfigNotifier:  NewConfigChangeNotifier(),
	}
}

// Initialize 初始化配置管理器
func (cm *ConfigManager) Initialize(ctx context.Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 初始化配置解析器
	if err := cm.ConfigParser.Initialize(ctx); err != nil {
		return fmt.Errorf("初始化配置解析器失败: %w", err)
	}

	// 初始化配置验证器
	if err := cm.ConfigValidator.Initialize(ctx); err != nil {
		return fmt.Errorf("初始化配置验证器失败: %w", err)
	}

	// 初始化配置变更通知器
	if err := cm.ConfigNotifier.Initialize(ctx); err != nil {
		return fmt.Errorf("初始化配置变更通知器失败: %w", err)
	}

	return nil
}

// Start 启动配置管理器
func (cm *ConfigManager) Start(ctx context.Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 启动配置变更通知器
	if err := cm.ConfigNotifier.Start(ctx); err != nil {
		return fmt.Errorf("启动配置变更通知器失败: %w", err)
	}

	return nil
}

// Stop 停止配置管理器
func (cm *ConfigManager) Stop(ctx context.Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 停止配置变更通知器
	cm.ConfigNotifier.Stop(ctx)

	return nil
}

// LoadConfiguration 加载配置
func (cm *ConfigManager) LoadConfiguration(sourceID string, configData map[string]string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 解析配置
	config, err := cm.ConfigParser.Parse(configData)
	if err != nil {
		return fmt.Errorf("解析配置失败: %w", err)
	}

	// 验证配置
	result := cm.ConfigValidator.Validate(config)
	if !result.Valid {
		return fmt.Errorf("配置验证失败: %v", result.Errors)
	}

	// 设置配置ID
	config.ID = sourceID

	// 保存配置
	cm.Configs[sourceID] = config

	// 通知配置变更
	cm.ConfigNotifier.NotifyConfigChange(sourceID, config)

	return nil
}

// GetConfiguration 获取配置
func (cm *ConfigManager) GetConfiguration(sourceID string) (*SourceConfiguration, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	config, exists := cm.Configs[sourceID]
	return config, exists
}

// UpdateConfiguration 更新配置
func (cm *ConfigManager) UpdateConfiguration(sourceID string, newConfig map[string]string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 获取现有配置
	existingConfig, exists := cm.Configs[sourceID]
	if !exists {
		return fmt.Errorf("数据源 %s 不存在", sourceID)
	}

	// 更新配置
	existingConfig.Update(newConfig)

	// 验证更新后的配置
	result := cm.ConfigValidator.Validate(existingConfig)
	if !result.Valid {
		return fmt.Errorf("配置验证失败: %v", result.Errors)
	}

	// 通知配置变更
	cm.ConfigNotifier.NotifyConfigChange(sourceID, existingConfig)

	return nil
}

// RemoveConfiguration 移除配置
func (cm *ConfigManager) RemoveConfiguration(sourceID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.Configs[sourceID]; !exists {
		return fmt.Errorf("数据源 %s 不存在", sourceID)
	}

	delete(cm.Configs, sourceID)

	// 通知配置移除
	cm.ConfigNotifier.NotifyConfigRemoval(sourceID)

	return nil
}

// ListConfigurations 列出所有配置
func (cm *ConfigManager) ListConfigurations() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	ids := make([]string, 0, len(cm.Configs))
	for id := range cm.Configs {
		ids = append(ids, id)
	}
	return ids
}

// ValidateConfiguration 验证配置
func (cm *ConfigManager) ValidateConfiguration(sourceID string) *ValidationResult {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	config, exists := cm.Configs[sourceID]
	if !exists {
		result := &ValidationResult{Valid: false}
		result.AddError(fmt.Sprintf("数据源 %s 不存在", sourceID))
		return result
	}

	return cm.ConfigValidator.Validate(config)
}

// SourceConfigParser 数据源配置解析器
type SourceConfigParser struct {
	mu sync.RWMutex
}

// NewSourceConfigParser 创建配置解析器
func NewSourceConfigParser() *SourceConfigParser {
	return &SourceConfigParser{}
}

// Initialize 初始化配置解析器
func (scp *SourceConfigParser) Initialize(ctx context.Context) error {
	scp.mu.Lock()
	defer scp.mu.Unlock()

	// 这里可以初始化解析器的相关组件
	// 例如：加载解析规则、初始化缓存等

	return nil
}

// Parse 解析配置
func (scp *SourceConfigParser) Parse(properties map[string]string) (*SourceConfiguration, error) {
	scp.mu.Lock()
	defer scp.mu.Unlock()

	config := NewSourceConfiguration()

	// 解析协议相关配置
	if protocol, exists := properties["protocol"]; exists {
		config.Protocol = protocol
	}

	if host, exists := properties["host"]; exists {
		config.Host = host
	}

	if portStr, exists := properties["port"]; exists {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("端口号格式错误: %s", portStr)
		}
		config.Port = port
	}

	// 解析安全配置
	if securityMode, exists := properties["security.mode"]; exists {
		config.SecurityMode = securityMode
	}

	// 解析高级配置
	if maxConnectionsStr, exists := properties["max.connections"]; exists {
		maxConnections, err := strconv.Atoi(maxConnectionsStr)
		if err != nil {
			return nil, fmt.Errorf("最大连接数格式错误: %s", maxConnectionsStr)
		}
		config.MaxConnections = maxConnections
	} else {
		config.MaxConnections = 10 // 默认值
	}

	if connectTimeoutStr, exists := properties["connect.timeout"]; exists {
		connectTimeout, err := strconv.Atoi(connectTimeoutStr)
		if err != nil {
			return nil, fmt.Errorf("连接超时格式错误: %s", connectTimeoutStr)
		}
		config.ConnectTimeout = time.Duration(connectTimeout) * time.Millisecond
	} else {
		config.ConnectTimeout = 5000 * time.Millisecond // 默认值
	}

	// 解析其他属性
	for key, value := range properties {
		if !scp.isReservedProperty(key) {
			config.SetProperty(key, value)
		}
	}

	return config, nil
}

// isReservedProperty 检查是否为保留属性
func (scp *SourceConfigParser) isReservedProperty(key string) bool {
	reservedKeys := []string{
		"protocol", "host", "port", "security.mode",
		"max.connections", "connect.timeout",
	}

	for _, reservedKey := range reservedKeys {
		if key == reservedKey {
			return true
		}
	}

	return false
}

// ParseFromFile 从文件解析配置
func (scp *SourceConfigParser) ParseFromFile(filePath string) (*SourceConfiguration, error) {
	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析JSON格式
	var configData map[string]interface{}
	if err := json.Unmarshal(content, &configData); err != nil {
		return nil, fmt.Errorf("解析JSON配置失败: %w", err)
	}

	// 转换为字符串映射
	properties := make(map[string]string)
	for key, value := range configData {
		properties[key] = fmt.Sprintf("%v", value)
	}

	return scp.Parse(properties)
}

// DataSourceValidator 数据源验证器
type DataSourceValidator struct {
	mu sync.RWMutex
}

// NewDataSourceValidator 创建数据源验证器
func NewDataSourceValidator() *DataSourceValidator {
	return &DataSourceValidator{}
}

// Initialize 初始化数据源验证器
func (dsv *DataSourceValidator) Initialize(ctx context.Context) error {
	dsv.mu.Lock()
	defer dsv.mu.Unlock()

	// 这里可以初始化验证器的相关组件
	// 例如：加载验证规则、初始化网络检查器等

	return nil
}

// Validate 验证数据源配置
func (dsv *DataSourceValidator) Validate(config *SourceConfiguration) *ValidationResult {
	dsv.mu.RLock()
	defer dsv.mu.RUnlock()

	result := &ValidationResult{Valid: true}

	// 连接性检查
	if err := dsv.checkConnectivity(config); err != nil {
		result.AddError(fmt.Sprintf("连接性检查失败: %v", err))
	}

	// 权限检查
	if err := dsv.checkPermissions(config); err != nil {
		result.AddError(fmt.Sprintf("权限检查失败: %v", err))
	}

	// 数据格式检查
	if err := dsv.validateDataFormat(config); err != nil {
		result.AddError(fmt.Sprintf("数据格式检查失败: %v", err))
	}

	// 配置完整性检查
	if err := dsv.checkConfigurationCompleteness(config); err != nil {
		result.AddError(fmt.Sprintf("配置完整性检查失败: %v", err))
	}

	return result
}

// checkConnectivity 检查连接性
func (dsv *DataSourceValidator) checkConnectivity(config *SourceConfiguration) error {
	// 这里实现实际的连接性检查逻辑
	// 例如：尝试建立网络连接、检查端口是否开放等

	// 示例：简单的地址格式检查
	if config.Host == "" {
		return fmt.Errorf("主机地址不能为空")
	}

	if config.Port <= 0 || config.Port > 65535 {
		return fmt.Errorf("端口号必须在1-65535之间")
	}

	// 这里可以添加实际的网络连接测试
	// 例如：使用net.DialTimeout测试连接

	return nil
}

// checkPermissions 检查权限
func (dsv *DataSourceValidator) checkPermissions(config *SourceConfiguration) error {
	// 这里实现实际的权限检查逻辑
	// 例如：检查文件权限、网络权限等

	// 示例：检查安全配置
	if config.SecurityMode != "" {
		validModes := []string{"tls", "oauth2", "tls_oauth2", "ip_whitelist", "none"}
		isValid := false
		for _, mode := range validModes {
			if config.SecurityMode == mode {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("不支持的安全模式: %s", config.SecurityMode)
		}
	}

	return nil
}

// validateDataFormat 验证数据格式
func (dsv *DataSourceValidator) validateDataFormat(config *SourceConfiguration) error {
	// 这里实现实际的数据格式验证逻辑
	// 例如：检查数据格式配置、验证编码设置等

	// 示例：检查协议配置
	if config.Protocol == "" {
		return fmt.Errorf("协议不能为空")
	}

	validProtocols := []string{"tcp", "udp", "http", "https", "tls", "ssl"}
	isValid := false
	for _, protocol := range validProtocols {
		if config.Protocol == protocol {
			isValid = true
			break
		}
	}
	if !isValid {
		return fmt.Errorf("不支持的协议: %s", config.Protocol)
	}

	return nil
}

// checkConfigurationCompleteness 检查配置完整性
func (dsv *DataSourceValidator) checkConfigurationCompleteness(config *SourceConfiguration) error {
	// 检查必需配置项
	if config.Host == "" {
		return fmt.Errorf("主机地址不能为空")
	}

	if config.Port <= 0 {
		return fmt.Errorf("端口号必须大于0")
	}

	if config.MaxConnections <= 0 {
		return fmt.Errorf("最大连接数必须大于0")
	}

	if config.ConnectTimeout <= 0 {
		return fmt.Errorf("连接超时必须大于0")
	}

	return nil
}

// ConfigChangeNotifier 配置变更通知器
type ConfigChangeNotifier struct {
	listeners map[string][]ConfigChangeListener
	mu        sync.RWMutex
}

// NewConfigChangeNotifier 创建配置变更通知器
func NewConfigChangeNotifier() *ConfigChangeNotifier {
	return &ConfigChangeNotifier{
		listeners: make(map[string][]ConfigChangeListener),
	}
}

// Initialize 初始化配置变更通知器
func (ccn *ConfigChangeNotifier) Initialize(ctx context.Context) error {
	ccn.mu.Lock()
	defer ccn.mu.Unlock()

	// 这里可以初始化通知器的相关组件
	// 例如：初始化事件队列、启动通知协程等

	return nil
}

// Start 启动配置变更通知器
func (ccn *ConfigChangeNotifier) Start(ctx context.Context) error {
	ccn.mu.Lock()
	defer ccn.mu.Unlock()

	// 这里可以启动通知器的相关组件
	// 例如：启动事件处理协程等

	return nil
}

// Stop 停止配置变更通知器
func (ccn *ConfigChangeNotifier) Stop(ctx context.Context) error {
	ccn.mu.Lock()
	defer ccn.mu.Unlock()

	// 这里可以停止通知器的相关组件

	return nil
}

// AddListener 添加监听器
func (ccn *ConfigChangeNotifier) AddListener(sourceID string, listener ConfigChangeListener) {
	ccn.mu.Lock()
	defer ccn.mu.Unlock()

	if ccn.listeners[sourceID] == nil {
		ccn.listeners[sourceID] = make([]ConfigChangeListener, 0)
	}

	ccn.listeners[sourceID] = append(ccn.listeners[sourceID], listener)
}

// RemoveListener 移除监听器
func (ccn *ConfigChangeNotifier) RemoveListener(sourceID string, listener ConfigChangeListener) {
	ccn.mu.Lock()
	defer ccn.mu.Unlock()

	listeners, exists := ccn.listeners[sourceID]
	if !exists {
		return
	}

	for i, l := range listeners {
		if l == listener {
			ccn.listeners[sourceID] = append(listeners[:i], listeners[i+1:]...)
			break
		}
	}
}

// NotifyConfigChange 通知配置变更
func (ccn *ConfigChangeNotifier) NotifyConfigChange(sourceID string, config *SourceConfiguration) {
	ccn.mu.RLock()
	defer ccn.mu.RUnlock()

	listeners, exists := ccn.listeners[sourceID]
	if !exists {
		return
	}

	event := &ConfigChangeEvent{
		SourceID: sourceID,
		Config:   config,
		Type:     ConfigChangeTypeUpdate,
		Time:     time.Now(),
	}

	for _, listener := range listeners {
		go listener.OnConfigChange(event)
	}
}

// NotifyConfigRemoval 通知配置移除
func (ccn *ConfigChangeNotifier) NotifyConfigRemoval(sourceID string) {
	ccn.mu.RLock()
	defer ccn.mu.RUnlock()

	listeners, exists := ccn.listeners[sourceID]
	if !exists {
		return
	}

	event := &ConfigChangeEvent{
		SourceID: sourceID,
		Config:   nil,
		Type:     ConfigChangeTypeRemove,
		Time:     time.Now(),
	}

	for _, listener := range listeners {
		go listener.OnConfigChange(event)
	}
}

// ConfigChangeListener 配置变更监听器接口
type ConfigChangeListener interface {
	OnConfigChange(event *ConfigChangeEvent)
}

// ConfigChangeEvent 配置变更事件
type ConfigChangeEvent struct {
	SourceID string
	Config   *SourceConfiguration
	Type     ConfigChangeType
	Time     time.Time
}

// ConfigChangeType 配置变更类型
type ConfigChangeType int

const (
	ConfigChangeTypeUpdate ConfigChangeType = iota
	ConfigChangeTypeRemove
)

// String 返回配置变更类型的字符串表示
func (cct ConfigChangeType) String() string {
	switch cct {
	case ConfigChangeTypeUpdate:
		return "update"
	case ConfigChangeTypeRemove:
		return "remove"
	default:
		return "unknown"
	}
}

// DynamicSourceConfigurator 动态数据源配置器
type DynamicSourceConfigurator struct {
	configManager *ConfigManager
	mu            sync.RWMutex
}

// NewDynamicSourceConfigurator 创建动态数据源配置器
func NewDynamicSourceConfigurator(configManager *ConfigManager) *DynamicSourceConfigurator {
	return &DynamicSourceConfigurator{
		configManager: configManager,
	}
}

// UpdateConfiguration 更新配置
func (dsc *DynamicSourceConfigurator) UpdateConfiguration(sourceID string, newConfig map[string]string) error {
	dsc.mu.Lock()
	defer dsc.mu.Unlock()

	// 获取当前配置
	currentConfig, exists := dsc.configManager.GetConfiguration(sourceID)
	if !exists {
		return fmt.Errorf("数据源 %s 不存在", sourceID)
	}

	// 更新配置
	currentConfig.Update(newConfig)

	// 验证更新后的配置
	result := dsc.configManager.ValidateConfiguration(sourceID)
	if !result.Valid {
		return fmt.Errorf("配置验证失败: %v", result.Errors)
	}

	// 通知相关组件配置已变更
	dsc.notifyConfigurationChange(sourceID, currentConfig)

	return nil
}

// notifyConfigurationChange 通知配置变更
func (dsc *DynamicSourceConfigurator) notifyConfigurationChange(sourceID string, config *SourceConfiguration) {
	// 这里可以通知相关的组件配置已变更
	// 例如：通知数据源重新加载配置、通知监控系统等

	// 示例：记录配置变更日志
	fmt.Printf("[DynamicSourceConfigurator] 数据源 %s 配置已更新: %+v\n", sourceID, config)
}
