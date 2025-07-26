// Copyright 2025 EdgeStream Team
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/crazy/edge-stream/internal/constants"
	"gopkg.in/yaml.v3"
)

// StandardConfigManager 标准配置管理器实现
type StandardConfigManager struct {
	mu        sync.RWMutex
	config    map[string]interface{}
	encryptor Encryptor
	sources   []ConfigSource
	watchers  []ConfigChangeCallback
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewStandardConfigManager 创建标准配置管理器
func NewStandardConfigManager(encryptionKey string) *StandardConfigManager {
	ctx, cancel := context.WithCancel(context.Background())

	var encryptor Encryptor
	if encryptionKey != "" {
		encryptor = NewAESEncryptor(encryptionKey)
	}

	return &StandardConfigManager{
		config:    make(map[string]interface{}),
		encryptor: encryptor,
		sources:   make([]ConfigSource, 0),
		watchers:  make([]ConfigChangeCallback, 0),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// LoadConfig 加载配置
func (m *StandardConfigManager) LoadConfig(source string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 判断配置源类型
	if strings.HasSuffix(source, ".yaml") || strings.HasSuffix(source, ".yml") {
		return m.loadYAMLFile(source)
	} else if strings.HasSuffix(source, ".json") {
		return m.loadJSONFile(source)
	}

	return fmt.Errorf("unsupported config file format: %s", source)
}

// loadYAMLFile 加载YAML配置文件
func (m *StandardConfigManager) loadYAMLFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", filename, err)
	}

	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse YAML config: %w", err)
	}

	// 展平配置并解密敏感数据
	flatConfig := m.flattenConfig(config, "")
	for key, value := range flatConfig {
		if strValue, ok := value.(string); ok && m.encryptor != nil {
			if decrypted, err := m.encryptor.Decrypt(strValue); err == nil {
				flatConfig[key] = decrypted
			}
		}
		m.config[key] = value
	}

	// 加载环境变量覆盖
	m.loadEnvironmentVariables()

	return nil
}

// loadJSONFile 加载JSON配置文件
func (m *StandardConfigManager) loadJSONFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", filename, err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse JSON config: %w", err)
	}

	// 展平配置并解密敏感数据
	flatConfig := m.flattenConfig(config, "")
	for key, value := range flatConfig {
		if strValue, ok := value.(string); ok && m.encryptor != nil {
			if decrypted, err := m.encryptor.Decrypt(strValue); err == nil {
				flatConfig[key] = decrypted
			}
		}
		m.config[key] = value
	}

	// 加载环境变量覆盖
	m.loadEnvironmentVariables()

	return nil
}

// flattenConfig 展平嵌套配置
func (m *StandardConfigManager) flattenConfig(config map[string]interface{}, prefix string) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range config {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		switch v := value.(type) {
		case map[string]interface{}:
			// 递归处理嵌套对象
			nested := m.flattenConfig(v, fullKey)
			for nestedKey, nestedValue := range nested {
				result[nestedKey] = nestedValue
			}
		default:
			result[fullKey] = value
		}
	}

	return result
}

// loadEnvironmentVariables 加载环境变量
func (m *StandardConfigManager) loadEnvironmentVariables() {
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", constants.EnvironmentVariableParts)
		if len(parts) != constants.EnvironmentVariableParts {
			continue
		}

		key := strings.ToLower(parts[0])
		value := parts[1]

		// 只处理以EDGE_STREAM_开头的环境变量
		if strings.HasPrefix(key, "edge_stream_") {
			configKey := strings.TrimPrefix(key, "edge_stream_")
			configKey = strings.ReplaceAll(configKey, "_", ".")
			m.config[configKey] = value
		}
	}
}

// GetString 获取字符串配置
func (m *StandardConfigManager) GetString(key string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if value, exists := m.config[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
		return fmt.Sprintf("%v", value)
	}
	return ""
}

// convertToInt 将任意类型的值转换为int
func convertToInt(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int8:
		return int(v)
	case int16:
		return int(v)
	case int32:
		return int(v)
	case int64:
		return convertInt64ToInt(v)
	case uint:
		return convertUintToInt(v)
	case uint8:
		return int(v)
	case uint16:
		return int(v)
	case uint32:
		return convertUint32ToInt(v)
	case uint64:
		return convertUint64ToInt(v)
	case float32:
		return int(v)
	case float64:
		return int(v)
	case string:
		return convertStringToInt(v)
	default:
		return 0
	}
}

// convertInt64ToInt 安全地将int64转换为int
func convertInt64ToInt(v int64) int {
	if v > constants.MaxInt32Value || v < constants.MinInt32Value {
		return 0 // 溢出时返回0
	}
	return int(v)
}

// convertUintToInt 安全地将uint转换为int
func convertUintToInt(v uint) int {
	if v > constants.MaxInt32Value {
		return 0 // 溢出时返回0
	}
	return int(v)
}

// convertUint32ToInt 安全地将uint32转换为int
func convertUint32ToInt(v uint32) int {
	if v > constants.MaxInt32Value {
		return 0 // 溢出时返回0
	}
	return int(v)
}

// convertUint64ToInt 安全地将uint64转换为int
func convertUint64ToInt(v uint64) int {
	if v > constants.MaxInt32Value {
		return 0 // 溢出时返回0
	}
	return int(v)
}

// convertStringToInt 将字符串转换为int
func convertStringToInt(v string) int {
	if i, err := strconv.Atoi(v); err == nil {
		return i
	}
	return 0
}

// GetInt 获取整数配置
func (m *StandardConfigManager) GetInt(key string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if value, exists := m.config[key]; exists {
		return convertToInt(value)
	}
	return 0
}

// GetBool 获取布尔配置
func (m *StandardConfigManager) GetBool(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if value, exists := m.config[key]; exists {
		switch v := value.(type) {
		case bool:
			return v
		case string:
			if b, err := strconv.ParseBool(v); err == nil {
				return b
			}
		}
	}
	return false
}

// GetDuration 获取时间间隔配置
func (m *StandardConfigManager) GetDuration(key string) time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if value, exists := m.config[key]; exists {
		switch v := value.(type) {
		case time.Duration:
			return v
		case string:
			if d, err := time.ParseDuration(v); err == nil {
				return d
			}
		case int:
			return time.Duration(v) * time.Second
		case int64:
			return time.Duration(v) * time.Second
		case float64:
			return time.Duration(v) * time.Second
		}
	}
	return 0
}

// GetFloat64 获取浮点数配置
func (m *StandardConfigManager) GetFloat64(key string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if value, exists := m.config[key]; exists {
		switch v := value.(type) {
		case float64:
			return v
		case float32:
			return float64(v)
		case int:
			return float64(v)
		case int64:
			return float64(v)
		case string:
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return f
			}
		}
	}
	return 0.0
}

// Set 设置配置值
func (m *StandardConfigManager) Set(key string, value interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	oldValue := m.config[key]
	m.config[key] = value

	// 通知监听器
	for _, watcher := range m.watchers {
		go watcher(key, oldValue, value)
	}

	return nil
}

// Watch 监听配置变更
func (m *StandardConfigManager) Watch(ctx context.Context, callback ConfigChangeCallback) error {
	m.mu.Lock()
	m.watchers = append(m.watchers, callback)
	m.mu.Unlock()

	return nil
}

// Validate 验证配置
func (m *StandardConfigManager) Validate() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 基本验证逻辑
	requiredKeys := []string{
		"database.mysql.host",
		"database.mysql.port",
		"redis.host",
		"redis.port",
	}

	for _, key := range requiredKeys {
		if _, exists := m.config[key]; !exists {
			return fmt.Errorf("required configuration key missing: %s", key)
		}
	}

	return nil
}

// Reload 重新加载配置
func (m *StandardConfigManager) Reload() error {
	// 这里可以实现配置重新加载逻辑
	// 为了简化，暂时返回nil
	return nil
}

// Save 保存配置到文件
func (m *StandardConfigManager) Save(filename string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 将扁平化的配置还原为嵌套结构
	config := m.unflattenConfig(m.config)

	// 根据文件扩展名选择格式
	if strings.HasSuffix(filename, ".json") {
		data, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal config to JSON: %w", err)
		}
		return os.WriteFile(filename, data, constants.DefaultFilePermission)
	} else {
		// 默认使用YAML格式
		data, err := yaml.Marshal(config)
		if err != nil {
			return fmt.Errorf("failed to marshal config to YAML: %w", err)
		}
		return os.WriteFile(filename, data, constants.DefaultFilePermission)
	}
}

// GetAll 获取所有配置
func (m *StandardConfigManager) GetAll() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]interface{})
	for key, value := range m.config {
		result[key] = value
	}
	return result
}

// IsEncrypted 检查是否为加密配置
func (m *StandardConfigManager) IsEncrypted(key string) bool {
	if m.encryptor == nil {
		return false
	}

	value := m.GetString(key)
	return m.encryptor.IsEncrypted(value)
}

// Decrypt 解密配置值
func (m *StandardConfigManager) Decrypt(encryptedValue string) (string, error) {
	if m.encryptor == nil {
		return encryptedValue, nil
	}

	return m.encryptor.Decrypt(encryptedValue)
}

// Close 关闭配置管理器
func (m *StandardConfigManager) Close() error {
	if m.cancel != nil {
		m.cancel()
	}
	return nil
}

// AddSource 添加配置源
func (m *StandardConfigManager) AddSource(source ConfigSource) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sources = append(m.sources, source)
}

// GetConfigStruct 将配置映射到结构体
func (m *StandardConfigManager) GetConfigStruct(v interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 将map转换为JSON再反序列化到结构体
	data, err := json.Marshal(m.unflattenConfig(m.config))
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}

// unflattenConfig 将扁平化的配置还原为嵌套结构
func (m *StandardConfigManager) unflattenConfig(flat map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range flat {
		parts := strings.Split(key, ".")
		current := result

		for i, part := range parts {
			if i == len(parts)-1 {
				// 最后一个部分，设置值
				current[part] = value
			} else {
				// 中间部分，创建嵌套map
				if _, exists := current[part]; !exists {
					current[part] = make(map[string]interface{})
				}
				if nested, ok := current[part].(map[string]interface{}); ok {
					current = nested
				}
			}
		}
	}

	return result
}
