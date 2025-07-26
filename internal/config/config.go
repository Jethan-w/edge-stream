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
	"time"

	"github.com/crazy/edge-stream/internal/constants"
)

// ConfigManager 配置管理器接口
type ConfigManager interface {
	// LoadConfig 加载配置
	LoadConfig(source string) error

	// GetString 获取字符串配置
	GetString(key string) string

	// GetInt 获取整数配置
	GetInt(key string) int

	// GetFloat64 获取浮点数配置
	GetFloat64(key string) float64

	// GetBool 获取布尔配置
	GetBool(key string) bool

	// GetDuration 获取时间间隔配置
	GetDuration(key string) time.Duration

	// Set 设置配置值
	Set(key string, value interface{}) error

	// Watch 监听配置变更
	Watch(ctx context.Context, callback ConfigChangeCallback) error

	// Validate 验证配置
	Validate() error

	// Reload 重新加载配置
	Reload() error

	// GetAll 获取所有配置
	GetAll() map[string]interface{}

	// IsEncrypted 检查是否为加密配置
	IsEncrypted(key string) bool

	// Decrypt 解密配置值
	Decrypt(encryptedValue string) (string, error)

	// Save 保存配置到文件
	Save(filename string) error
}

// ConfigChangeCallback 配置变更回调函数
type ConfigChangeCallback func(key string, oldValue, newValue interface{})

// Encryptor 加密器接口
type Encryptor interface {
	// Encrypt 加密数据
	Encrypt(plaintext string) (string, error)

	// Decrypt 解密数据
	Decrypt(ciphertext string) (string, error)

	// IsEncrypted 检查是否为加密数据
	IsEncrypted(data string) bool
}

// ConfigSource 配置源接口
type ConfigSource interface {
	// Load 加载配置数据
	Load() (map[string]interface{}, error)

	// Watch 监听配置变更
	Watch(ctx context.Context, callback func(map[string]interface{})) error

	// GetSourceType 获取配置源类型
	GetSourceType() string
}

// Config 配置结构体
type Config struct {
	// Database 数据库配置
	Database DatabaseConfig `yaml:"database" json:"database"`

	// Redis Redis配置
	Redis redisConfig `yaml:"redis" json:"redis"`

	// Server 服务器配置
	Server serverConfig `yaml:"server" json:"server"`

	// Logging 日志配置
	Logging loggingConfig `yaml:"logging" json:"logging"`

	// Metrics 指标配置
	Metrics metricsConfig `yaml:"metrics" json:"metrics"`

	// Security 安全配置
	Security securityConfig `yaml:"security" json:"security"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	MySQL      mysqlConfig      `yaml:"mysql" json:"mysql"`
	PostgreSQL postgreSQLConfig `yaml:"postgresql" json:"postgresql"`
}

// mysqlConfig MySQL配置
type mysqlConfig struct {
	Host     string        `yaml:"host" json:"host"`
	Port     int           `yaml:"port" json:"port"`
	Username string        `yaml:"username" json:"username"`
	Password string        `yaml:"password" json:"password" sensitive:"true"`
	Database string        `yaml:"database" json:"database"`
	Charset  string        `yaml:"charset" json:"charset"`
	Timeout  time.Duration `yaml:"timeout" json:"timeout"`
}

// postgreSQLConfig PostgreSQL配置
type postgreSQLConfig struct {
	Host     string        `yaml:"host" json:"host"`
	Port     int           `yaml:"port" json:"port"`
	Username string        `yaml:"username" json:"username"`
	Password string        `yaml:"password" json:"password" sensitive:"true"`
	Database string        `yaml:"database" json:"database"`
	SSLMode  string        `yaml:"sslmode" json:"sslmode"`
	Timeout  time.Duration `yaml:"timeout" json:"timeout"`
}

// redisConfig Redis配置
type redisConfig struct {
	Host     string        `yaml:"host" json:"host"`
	Port     int           `yaml:"port" json:"port"`
	Password string        `yaml:"password" json:"password" sensitive:"true"`
	DB       int           `yaml:"db" json:"db"`
	Timeout  time.Duration `yaml:"timeout" json:"timeout"`
}

// serverConfig 服务器配置
type serverConfig struct {
	Host         string        `yaml:"host" json:"host"`
	Port         int           `yaml:"port" json:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout" json:"idle_timeout"`
}

// loggingConfig 日志配置
type loggingConfig struct {
	Level  string `yaml:"level" json:"level"`
	Format string `yaml:"format" json:"format"`
	Output string `yaml:"output" json:"output"`
}

// metricsConfig 指标配置
type metricsConfig struct {
	Enabled  bool          `yaml:"enabled" json:"enabled"`
	Endpoint string        `yaml:"endpoint" json:"endpoint"`
	Interval time.Duration `yaml:"interval" json:"interval"`
}

// securityConfig 安全配置
type securityConfig struct {
	EncryptionKey string `yaml:"encryption_key" json:"encryption_key" sensitive:"true"`
	JWTSecret     string `yaml:"jwt_secret" json:"jwt_secret" sensitive:"true"`
	TLSEnabled    bool   `yaml:"tls_enabled" json:"tls_enabled"`
	CertFile      string `yaml:"cert_file" json:"cert_file"`
	KeyFile       string `yaml:"key_file" json:"key_file"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Database: DatabaseConfig{
			MySQL: mysqlConfig{
				Host:    "localhost",
				Port:    constants.DefaultMySQLPort,
				Charset: "utf8mb4",
				Timeout: constants.DefaultConnectionTimeoutSeconds * time.Second,
			},
			PostgreSQL: postgreSQLConfig{
				Host:    "localhost",
				Port:    constants.DefaultPostgreSQLPort,
				SSLMode: "disable",
				Timeout: constants.DefaultConnectionTimeoutSeconds * time.Second,
			},
		},
		Redis: redisConfig{
			Host:    "localhost",
			Port:    constants.DefaultRedisPort,
			DB:      constants.DefaultRedisDB,
			Timeout: constants.DefaultRedisTimeoutSeconds * time.Second,
		},
		Server: serverConfig{
			Host:         "localhost",
			Port:         constants.DefaultHTTPPort,
			ReadTimeout:  constants.DefaultServerReadTimeoutSeconds * time.Second,
			WriteTimeout: constants.DefaultServerWriteTimeoutSeconds * time.Second,
			IdleTimeout:  constants.DefaultServerIdleTimeoutSeconds * time.Second,
		},
		Logging: loggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
		Metrics: metricsConfig{
			Enabled:  true,
			Endpoint: "/metrics",
			Interval: constants.DefaultMetricsIntervalSeconds * time.Second,
		},
		Security: securityConfig{
			TLSEnabled: false,
		},
	}
}
