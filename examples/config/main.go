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

package main

import (
	"fmt"
	"os"

	"github.com/crazy/edge-stream/examples/common"
	"github.com/crazy/edge-stream/internal/config"
)

func main() {
	common.ComponentExample[config.ConfigManager]{
		Name:        "Edge Stream 配置管理器示例",
		Description: "演示配置管理器的加载、验证、加密和环境变量覆盖功能",
		CreateFunc:  createConfigManager,
		ConfigFunc:  configureManager,
		RunFunc:     runConfigExamples,
	}.Run()
}

func createConfigManager() (config.ConfigManager, error) {
	encryptionKey := "my-secret-encryption-key-32-bytes!"
	return config.NewStandardConfigManager(encryptionKey), nil
}

func configureManager(configManager config.ConfigManager) error {
	// 加载配置文件
	configFile := "config.yaml"
	if len(os.Args) > 1 {
		configFile = os.Args[1]
	}

	common.PrintInfo(fmt.Sprintf("加载配置文件: %s", configFile))
	if err := configManager.LoadConfig(configFile); err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	// 验证配置
	if err := configManager.Validate(); err != nil {
		common.PrintError(fmt.Errorf("配置验证警告: %w", err))
	}

	return nil
}

func runConfigExamples(configManager config.ConfigManager) error {
	// 1. 配置读取示例
	if err := demonstrateConfigReading(configManager); err != nil {
		return err
	}

	// 2. 加密功能演示
	if err := demonstrateEncryption(); err != nil {
		return err
	}

	// 3. 敏感配置检测
	demonstrateSensitiveDetection(configManager)

	// 4. 环境变量覆盖演示
	if err := demonstrateEnvOverride(configManager); err != nil {
		return err
	}

	// 5. 配置结构映射
	if err := demonstrateStructMapping(configManager); err != nil {
		return err
	}

	// 6. 显示所有配置
	demonstrateAllConfigs(configManager)

	return nil
}

func demonstrateConfigReading(configManager config.ConfigManager) error {
	common.PrintSection("配置读取示例")

	// 数据库配置
	mysqlHost := configManager.GetString("database.mysql.host")
	mysqlPort := configManager.GetInt("database.mysql.port")
	mysqlPassword := configManager.GetString("database.mysql.password")
	common.PrintResult("MySQL配置", fmt.Sprintf("%s:%d, 密码: %s", mysqlHost, mysqlPort, maskPassword(mysqlPassword)))

	// Redis配置
	redisHost := configManager.GetString("redis.host")
	redisPort := configManager.GetInt("redis.port")
	redisPassword := configManager.GetString("redis.password")
	common.PrintResult("Redis配置", fmt.Sprintf("%s:%d, 密码: %s", redisHost, redisPort, maskPassword(redisPassword)))

	// 服务器配置
	serverHost := configManager.GetString("server.host")
	serverPort := configManager.GetInt("server.port")
	readTimeout := configManager.GetDuration("server.read_timeout")
	common.PrintResult("服务器配置", fmt.Sprintf("%s:%d, 读取超时: %v", serverHost, serverPort, readTimeout))

	// 布尔配置
	metricsEnabled := configManager.GetBool("metrics.enabled")
	tlsEnabled := configManager.GetBool("security.tls_enabled")
	common.PrintResult("功能开关", fmt.Sprintf("指标启用: %t, TLS启用: %t", metricsEnabled, tlsEnabled))

	return nil
}

func demonstrateEncryption() error {
	common.PrintSection("加密功能演示")

	encryptionKey := "my-secret-encryption-key-32-bytes!"
	encryptor := config.NewAESEncryptor(encryptionKey)

	// 加密示例
	plaintext := "my-secret-password"
	encrypted, err := encryptor.Encrypt(plaintext)
	if err != nil {
		return fmt.Errorf("加密失败: %w", err)
	}

	common.PrintResult("原文", plaintext)
	common.PrintResult("密文", encrypted)

	// 解密验证
	decrypted, err := encryptor.Decrypt(encrypted)
	if err != nil {
		return fmt.Errorf("解密失败: %w", err)
	}

	common.PrintResult("解密", decrypted)
	common.PrintResult("解密成功", plaintext == decrypted)

	return nil
}

func demonstrateSensitiveDetection(configManager config.ConfigManager) {
	common.PrintSection("敏感配置检测")

	sensitiveKeys := []string{
		"database.mysql.password",
		"redis.password",
		"security.encryption_key",
		"security.jwt_secret",
	}

	for _, key := range sensitiveKeys {
		isEncrypted := configManager.IsEncrypted(key)
		isSensitive := config.IsSensitiveKey(key)
		common.PrintResult(fmt.Sprintf("配置项 %s", key), fmt.Sprintf("敏感=%t, 已加密=%t", isSensitive, isEncrypted))
	}
}

func demonstrateEnvOverride(configManager config.ConfigManager) error {
	common.PrintSection("环境变量覆盖演示")

	os.Setenv("EDGE_STREAM_SERVER_PORT", "9090")
	os.Setenv("EDGE_STREAM_LOGGING_LEVEL", "debug")

	// 重新加载配置以应用环境变量
	configFile := "config.yaml"
	if len(os.Args) > 1 {
		configFile = os.Args[1]
	}

	if err := configManager.LoadConfig(configFile); err != nil {
		return fmt.Errorf("重新加载配置失败: %w", err)
	}

	newPort := configManager.GetInt("server.port")
	newLogLevel := configManager.GetString("logging.level")
	common.PrintResult("环境变量覆盖后", fmt.Sprintf("端口: %d, 日志级别: %s", newPort, newLogLevel))

	return nil
}

func demonstrateStructMapping(configManager config.ConfigManager) error {
	common.PrintSection("配置结构映射")

	// 演示通过键值获取配置信息
	mysqlDB := configManager.GetString("database.mysql.database")
	redisDB := configManager.GetInt("redis.db")
	serverPort := configManager.GetInt("server.port")
	logLevel := configManager.GetString("logging.level")

	common.PrintResult("MySQL数据库", mysqlDB)
	common.PrintResult("Redis数据库", redisDB)
	common.PrintResult("服务器端口", serverPort)
	common.PrintResult("日志级别", logLevel)

	return nil
}

func demonstrateAllConfigs(configManager config.ConfigManager) {
	common.PrintSection("所有配置项")

	allConfigs := configManager.GetAll()
	for key, value := range allConfigs {
		if config.IsSensitiveKey(key) {
			common.PrintResult(key, maskPassword(fmt.Sprintf("%v", value)))
		} else {
			common.PrintResult(key, value)
		}
	}
}

// maskPassword 遮蔽密码显示
func maskPassword(password string) string {
	if len(password) <= 4 {
		return "****"
	}
	return password[:2] + "****" + password[len(password)-2:]
}
