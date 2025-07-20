package ConfigManager

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ConfigValidator 配置验证器接口
type ConfigValidator interface {
	// Validate 验证配置
	Validate(configMap ConfigMap) (*ValidationResult, error)
}

// ValidationResult 验证结果
type ValidationResult struct {
	IsValid  bool
	Errors   []string
	Warnings []string
	mutex    sync.RWMutex
}

// NewValidationResult 创建新的验证结果
func NewValidationResult() *ValidationResult {
	return &ValidationResult{
		IsValid:  true,
		Errors:   make([]string, 0),
		Warnings: make([]string, 0),
	}
}

// AddError 添加错误
func (vr *ValidationResult) AddError(error string) {
	vr.mutex.Lock()
	defer vr.mutex.Unlock()

	vr.IsValid = false
	vr.Errors = append(vr.Errors, error)
}

// AddWarning 添加警告
func (vr *ValidationResult) AddWarning(warning string) {
	vr.mutex.Lock()
	defer vr.mutex.Unlock()

	vr.Warnings = append(vr.Warnings, warning)
}

// GetErrors 获取所有错误
func (vr *ValidationResult) GetErrors() string {
	vr.mutex.RLock()
	defer vr.mutex.RUnlock()

	return strings.Join(vr.Errors, "; ")
}

// GetWarnings 获取所有警告
func (vr *ValidationResult) GetWarnings() string {
	vr.mutex.RLock()
	defer vr.mutex.RUnlock()

	return strings.Join(vr.Warnings, "; ")
}

// Merge 合并验证结果
func (vr *ValidationResult) Merge(other *ValidationResult) {
	vr.mutex.Lock()
	defer vr.mutex.Unlock()

	if !other.IsValid {
		vr.IsValid = false
	}

	vr.Errors = append(vr.Errors, other.Errors...)
	vr.Warnings = append(vr.Warnings, other.Warnings...)
}

// ValidationStrategy 验证策略枚举
type ValidationStrategy string

const (
	ValidationStrategyStrict  ValidationStrategy = "STRICT"
	ValidationStrategyLenient ValidationStrategy = "LENIENT"
	ValidationStrategyIgnore  ValidationStrategy = "IGNORE"
)

// ConfigurationValidator 配置验证器
type ConfigurationValidator struct {
	validators []ConfigValidator
	strategy   ValidationStrategy
	mutex      sync.RWMutex
}

// NewConfigurationValidator 创建新的配置验证器
func NewConfigurationValidator(strategy ValidationStrategy) *ConfigurationValidator {
	return &ConfigurationValidator{
		validators: make([]ConfigValidator, 0),
		strategy:   strategy,
	}
}

// RegisterValidator 注册验证器
func (cv *ConfigurationValidator) RegisterValidator(validator ConfigValidator) {
	cv.mutex.Lock()
	defer cv.mutex.Unlock()

	cv.validators = append(cv.validators, validator)
}

// Validate 验证配置
func (cv *ConfigurationValidator) Validate(configMap ConfigMap) (*ValidationResult, error) {
	cv.mutex.RLock()
	defer cv.mutex.RUnlock()

	result := NewValidationResult()

	for _, validator := range cv.validators {
		validatorResult, err := validator.Validate(configMap)
		if err != nil {
			return nil, fmt.Errorf("验证器执行失败: %w", err)
		}

		result.Merge(validatorResult)

		// 根据策略决定是否继续验证
		if cv.strategy == ValidationStrategyStrict && !result.IsValid {
			break
		}
	}

	return result, nil
}

// SetStrategy 设置验证策略
func (cv *ConfigurationValidator) SetStrategy(strategy ValidationStrategy) {
	cv.mutex.Lock()
	defer cv.mutex.Unlock()

	cv.strategy = strategy
}

// GetStrategy 获取验证策略
func (cv *ConfigurationValidator) GetStrategy() ValidationStrategy {
	cv.mutex.RLock()
	defer cv.mutex.RUnlock()

	return cv.strategy
}

// PortRangeValidator 端口范围验证器
type PortRangeValidator struct {
	minPort int
	maxPort int
}

// NewPortRangeValidator 创建新的端口范围验证器
func NewPortRangeValidator(minPort, maxPort int) *PortRangeValidator {
	return &PortRangeValidator{
		minPort: minPort,
		maxPort: maxPort,
	}
}

// Validate 验证端口范围
func (prv *PortRangeValidator) Validate(configMap ConfigMap) (*ValidationResult, error) {
	result := NewValidationResult()

	// 验证HTTP端口
	httpPort, exists := configMap.Get("nifi.web.http.port")
	if exists {
		if err := prv.validatePort(httpPort, "HTTP端口"); err != nil {
			result.AddError(err.Error())
		}
	}

	// 验证HTTPS端口
	httpsPort, exists := configMap.Get("nifi.web.https.port")
	if exists {
		if err := prv.validatePort(httpsPort, "HTTPS端口"); err != nil {
			result.AddError(err.Error())
		}
	}

	return result, nil
}

// validatePort 验证单个端口
func (prv *PortRangeValidator) validatePort(portStr, portName string) error {
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("%s必须是有效的整数", portName)
	}

	if port < prv.minPort || port > prv.maxPort {
		return fmt.Errorf("%s必须在%d-%d范围内", portName, prv.minPort, prv.maxPort)
	}

	return nil
}

// DatabaseConfigValidator 数据库配置验证器
type DatabaseConfigValidator struct{}

// NewDatabaseConfigValidator 创建新的数据库配置验证器
func NewDatabaseConfigValidator() *DatabaseConfigValidator {
	return &DatabaseConfigValidator{}
}

// Validate 验证数据库配置
func (dcv *DatabaseConfigValidator) Validate(configMap ConfigMap) (*ValidationResult, error) {
	result := NewValidationResult()

	dbUrl, exists := configMap.Get("db.url")
	if exists {
		if !dcv.isValidJdbcUrl(dbUrl) {
			result.AddError("无效的数据库连接URL格式")
		}
	}

	username, exists := configMap.Get("db.username")
	if exists {
		if username == "" {
			result.AddError("数据库用户名不能为空")
		}
	}

	password, exists := configMap.Get("db.password")
	if exists {
		if !dcv.isSensitivePropertyEncrypted(password) {
			result.AddWarning("建议对数据库密码进行加密")
		}
	}

	return result, nil
}

// isValidJdbcUrl 验证JDBC URL格式
func (dcv *DatabaseConfigValidator) isValidJdbcUrl(url string) bool {
	// 简化的JDBC URL验证
	pattern := regexp.MustCompile(`^jdbc:[a-zA-Z]+://.+$`)
	return pattern.MatchString(url)
}

// isSensitivePropertyEncrypted 检查敏感属性是否已加密
func (dcv *DatabaseConfigValidator) isSensitivePropertyEncrypted(value string) bool {
	// 简化的加密检测逻辑
	// 实际应该检查是否使用了加密前缀或格式
	return strings.HasPrefix(value, "ENC{") && strings.HasSuffix(value, "}")
}

// ConfigSnapshot 配置快照
type ConfigSnapshot struct {
	ConfigMap ConfigMap
	Timestamp time.Time
}

// NewConfigSnapshot 创建新的配置快照
func NewConfigSnapshot(configMap ConfigMap) *ConfigSnapshot {
	return &ConfigSnapshot{
		ConfigMap: configMap,
		Timestamp: time.Now(),
	}
}

// ConfigHistoryManager 配置历史管理器
type ConfigHistoryManager struct {
	configHistory  []*ConfigSnapshot
	maxHistorySize int
	mutex          sync.RWMutex
}

// NewConfigHistoryManager 创建新的配置历史管理器
func NewConfigHistoryManager() *ConfigHistoryManager {
	return &ConfigHistoryManager{
		configHistory:  make([]*ConfigSnapshot, 0),
		maxHistorySize: 10,
	}
}

// RecordConfigurationSnapshot 记录配置快照
func (chm *ConfigHistoryManager) RecordConfigurationSnapshot(configMap ConfigMap) {
	chm.mutex.Lock()
	defer chm.mutex.Unlock()

	snapshot := NewConfigSnapshot(configMap)
	chm.configHistory = append(chm.configHistory, snapshot)

	// 限制历史记录大小
	if len(chm.configHistory) > chm.maxHistorySize {
		chm.configHistory = chm.configHistory[1:]
	}
}

// RollbackToPreviousConfiguration 回滚到上一个配置
func (chm *ConfigHistoryManager) RollbackToPreviousConfiguration() (ConfigMap, error) {
	chm.mutex.Lock()
	defer chm.mutex.Unlock()

	if len(chm.configHistory) <= 1 {
		return nil, fmt.Errorf("没有可用的历史配置")
	}

	// 移除当前配置
	chm.configHistory = chm.configHistory[:len(chm.configHistory)-1]

	// 返回上一个配置
	previousSnapshot := chm.configHistory[len(chm.configHistory)-1]
	return previousSnapshot.ConfigMap, nil
}

// GetConfigurationHistory 获取配置历史
func (chm *ConfigHistoryManager) GetConfigurationHistory() []*ConfigSnapshot {
	chm.mutex.RLock()
	defer chm.mutex.RUnlock()

	result := make([]*ConfigSnapshot, len(chm.configHistory))
	copy(result, chm.configHistory)
	return result
}

// SetMaxHistorySize 设置最大历史记录大小
func (chm *ConfigHistoryManager) SetMaxHistorySize(size int) {
	chm.mutex.Lock()
	defer chm.mutex.Unlock()

	chm.maxHistorySize = size

	// 如果当前历史记录超过新的大小限制，截断它
	if len(chm.configHistory) > size {
		chm.configHistory = chm.configHistory[len(chm.configHistory)-size:]
	}
}

// GetMaxHistorySize 获取最大历史记录大小
func (chm *ConfigHistoryManager) GetMaxHistorySize() int {
	chm.mutex.RLock()
	defer chm.mutex.RUnlock()

	return chm.maxHistorySize
}

// ClearHistory 清空历史记录
func (chm *ConfigHistoryManager) ClearHistory() {
	chm.mutex.Lock()
	defer chm.mutex.Unlock()

	chm.configHistory = make([]*ConfigSnapshot, 0)
}

// ConfigMetadata 配置元数据
type ConfigMetadata struct {
	ID                    string            `json:"id"`
	Type                  string            `json:"type"`
	CreationTimestamp     time.Time         `json:"creationTimestamp"`
	LastModifiedTimestamp time.Time         `json:"lastModifiedTimestamp"`
	Source                string            `json:"source"`
	AdditionalMetadata    map[string]string `json:"additionalMetadata"`
}

// NewConfigMetadata 创建新的配置元数据
func NewConfigMetadata(id, configType, source string) *ConfigMetadata {
	now := time.Now()
	return &ConfigMetadata{
		ID:                    id,
		Type:                  configType,
		CreationTimestamp:     now,
		LastModifiedTimestamp: now,
		Source:                source,
		AdditionalMetadata:    make(map[string]string),
	}
}

// UpdateLastModified 更新最后修改时间
func (cm *ConfigMetadata) UpdateLastModified() {
	cm.LastModifiedTimestamp = time.Now()
}

// AddMetadata 添加元数据
func (cm *ConfigMetadata) AddMetadata(key, value string) {
	cm.AdditionalMetadata[key] = value
}

// GetMetadata 获取元数据
func (cm *ConfigMetadata) GetMetadata(key string) (string, bool) {
	value, exists := cm.AdditionalMetadata[key]
	return value, exists
}
