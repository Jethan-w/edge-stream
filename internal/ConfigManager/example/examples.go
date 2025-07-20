package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/crazy/edge-stream/internal/ConfigManager"
)

// ExampleUsage 使用示例
func ExampleUsage() {
	// 创建配置管理器
	configManager := ConfigManager.NewStandardConfigManager()

	// 创建密钥管理器
	keyManager := ConfigManager.NewKeyStoreManager("keystore.p12", "nifi-key", "nifi-secret")
	secretKey, err := keyManager.GenerateOrLoadKey()
	if err != nil {
		log.Fatalf("生成密钥失败: %v", err)
	}

	// 创建敏感属性提供者
	sensitiveProvider := ConfigManager.NewAESSensitivePropertyProvider(secretKey)
	configManager.SetSensitivePropertyProvider(sensitiveProvider)

	// 注册配置验证器
	portValidator := ConfigManager.NewPortRangeValidator(1024, 65535)
	configManager.RegisterValidator(portValidator)

	dbValidator := ConfigManager.NewDatabaseConfigValidator()
	configManager.RegisterValidator(dbValidator)

	// 添加配置变更监听器
	listener := &ExampleConfigChangeListener{}
	configManager.AddConfigurationChangeListener(listener)

	// 设置配置属性
	err = configManager.SetProperty("nifi.web.http.port", "8080", false)
	if err != nil {
		log.Printf("设置HTTP端口失败: %v", err)
	}

	err = configManager.SetProperty("db.password", "secret123", true)
	if err != nil {
		log.Printf("设置数据库密码失败: %v", err)
	}

	// 获取配置属性
	httpPort, err := configManager.GetProperty("nifi.web.http.port")
	if err != nil {
		log.Printf("获取HTTP端口失败: %v", err)
	} else {
		fmt.Printf("HTTP端口: %s\n", httpPort)
	}

	// 验证配置
	result, err := configManager.ValidateConfiguration()
	if err != nil {
		log.Printf("配置验证失败: %v", err)
	} else {
		if result.IsValid {
			fmt.Println("配置验证通过")
		} else {
			fmt.Printf("配置验证失败: %s\n", result.GetErrors())
		}
	}

	// 持久化配置
	err = configManager.PersistConfiguration()
	if err != nil {
		log.Printf("持久化配置失败: %v", err)
	}
}

// ExampleConfigChangeListener 示例配置变更监听器
type ExampleConfigChangeListener struct{}

// OnConfigurationChanged 配置变更回调
func (ecl *ExampleConfigChangeListener) OnConfigurationChanged(event *ConfigManager.ConfigurationChangeEvent) {
	fmt.Printf("配置变更通知: 键=%s, 值=%s, 时间=%s\n",
		event.Key, event.Value, event.Timestamp.Format("2006-01-02 15:04:05"))
}

// ExampleDynamicConfiguration 动态配置示例
func ExampleDynamicConfiguration() {
	// 创建配置管理器
	ConfigManager.NewStandardConfigManager()

	// 创建环境变量注入器
	ConfigManager.NewEnvironmentInjector()

	// 创建配置值处理器
	valueProcessor := ConfigManager.NewConfigValueProcessor()

	// 模拟环境变量
	os.Setenv("NIFI_DB_HOST", "localhost")
	os.Setenv("NIFI_DB_PORT", "5432")

	// 处理包含环境变量的配置值
	configValue := "jdbc:postgresql://${ENV:NIFI_DB_HOST}:${ENV:NIFI_DB_PORT}/nifi"
	processedValue, err := valueProcessor.ProcessConfigValue(configValue)
	if err != nil {
		log.Printf("处理配置值失败: %v", err)
	} else {
		fmt.Printf("原始值: %s\n", configValue)
		fmt.Printf("处理后: %s\n", processedValue)
	}
}

// ExampleConfigurationValidation 配置验证示例
func ExampleConfigurationValidation() {
	// 创建配置验证器
	validator := ConfigManager.NewConfigurationValidator(ConfigManager.ValidationStrategyStrict)

	// 注册多个验证器
	validator.RegisterValidator(ConfigManager.NewPortRangeValidator(1024, 65535))
	validator.RegisterValidator(ConfigManager.NewDatabaseConfigValidator())

	// 创建测试配置
	configMap := ConfigManager.NewStandardConfigMap()
	configMap.Set("nifi.web.http.port", "8080", false)
	configMap.Set("nifi.web.https.port", "8443", false)
	configMap.Set("db.url", "jdbc:postgresql://localhost:5432/nifi", false)
	configMap.Set("db.username", "nifi", false)
	configMap.Set("db.password", "ENC{encrypted_password}", false)

	// 验证配置
	result, err := validator.Validate(configMap)
	if err != nil {
		log.Printf("验证失败: %v", err)
		return
	}

	if result.IsValid {
		fmt.Println("配置验证通过")
	} else {
		fmt.Printf("配置验证失败:\n")
		fmt.Printf("错误: %s\n", result.GetErrors())
		fmt.Printf("警告: %s\n", result.GetWarnings())
	}
}

// ExampleConfigurationHistory 配置历史示例
func ExampleConfigurationHistory() {
	// 创建配置历史管理器
	historyManager := ConfigManager.NewConfigHistoryManager()

	// 创建初始配置
	initialConfig := ConfigManager.NewStandardConfigMap()
	initialConfig.Set("nifi.web.http.port", "8080", false)
	initialConfig.Set("nifi.web.https.port", "8443", false)

	// 记录配置快照
	historyManager.RecordConfigurationSnapshot(initialConfig)

	// 修改配置
	updatedConfig := ConfigManager.NewStandardConfigMap()
	updatedConfig.Set("nifi.web.http.port", "9090", false)
	updatedConfig.Set("nifi.web.https.port", "9443", false)

	// 记录新的配置快照
	historyManager.RecordConfigurationSnapshot(updatedConfig)

	// 获取配置历史
	history := historyManager.GetConfigurationHistory()
	fmt.Printf("配置历史记录数量: %d\n", len(history))

	for i, snapshot := range history {
		fmt.Printf("快照 %d: %s\n", i+1, snapshot.Timestamp.Format("2006-01-02 15:04:05"))
		entries := snapshot.ConfigMap.GetAll()
		for key, entry := range entries {
			fmt.Printf("  %s = %s\n", key, entry.Value)
		}
	}

	// 回滚到上一个配置
	previousConfig, err := historyManager.RollbackToPreviousConfiguration()
	if err != nil {
		log.Printf("回滚失败: %v", err)
		return
	}

	fmt.Println("回滚后的配置:")
	entries := previousConfig.GetAll()
	for key, entry := range entries {
		fmt.Printf("  %s = %s\n", key, entry.Value)
	}
}

// ExampleHotReload 热重载示例
func ExampleHotReload() {
	// 创建配置管理器
	configManager := ConfigManager.NewStandardConfigManager()

	// 创建热重载管理器
	hotReloadManager := ConfigManager.NewHotReloadManager(configManager)

	// 模拟处理器
	processor := &ExampleProcessor{}

	// 重载处理器配置
	err := hotReloadManager.ReloadProcessorConfiguration(processor)
	if err != nil {
		log.Printf("重载处理器配置失败: %v", err)
	}
}

// ExampleProcessor 示例处理器
type ExampleProcessor struct {
	config map[string]string
}

// OnConfigurationReloaded 配置重载回调
func (ep *ExampleProcessor) OnConfigurationReloaded() {
	fmt.Println("处理器配置已重载")
}

// Reinitialize 重新初始化
func (ep *ExampleProcessor) Reinitialize() error {
	fmt.Println("重新初始化处理器资源")
	return nil
}

// RequiresRestart 检查是否需要重启
func (ep *ExampleProcessor) RequiresRestart() bool {
	return false
}

// GracefulRestart 优雅重启
func (ep *ExampleProcessor) GracefulRestart() error {
	fmt.Println("执行优雅重启")
	return nil
}

// ExampleNotificationManager 通知管理器示例
func ExampleNotificationManager() {
	// 创建通知管理器
	notificationManager := ConfigManager.NewNotificationManager(ConfigManager.NotificationTypeLocal)

	// 创建配置变更事件
	event := ConfigManager.NewConfigurationChangeEvent("nifi.web.http.port", "9090", "Example", "UPDATE")

	// 发送通知
	err := notificationManager.SendNotification(event)
	if err != nil {
		log.Printf("发送通知失败: %v", err)
	}

	// 切换到分布式通知
	notificationManager.SetNotificationType(ConfigManager.NotificationTypeDistributed)

	// 发送分布式通知
	err = notificationManager.SendNotification(event)
	if err != nil {
		log.Printf("发送分布式通知失败: %v", err)
	}
}

// ExampleVariableResolver 变量解析器示例
func ExampleVariableResolver() {
	// 创建变量解析器
	resolver := ConfigManager.NewVariableResolver()

	// 注册自定义解析器
	resolver.RegisterResolver("CURRENT_TIME", func() (string, error) {
		return time.Now().Format("2006-01-02 15:04:05"), nil
	})

	resolver.RegisterResolver("APP_VERSION", func() (string, error) {
		return "1.0.0", nil
	})

	// 解析包含变量的字符串
	input := "应用版本: ${APP_VERSION}, 当前时间: ${CURRENT_TIME}"
	resolved, err := resolver.ResolveVariables(input)
	if err != nil {
		log.Printf("解析变量失败: %v", err)
		return
	}

	fmt.Printf("原始字符串: %s\n", input)
	fmt.Printf("解析后: %s\n", resolved)
}

func main() {
	// 创建配置管理器
	configManager := ConfigManager.NewStandardConfigManager()

	// 创建密钥管理器
	keyManager := ConfigManager.NewKeyStoreManager("keystore.p12", "nifi-key", "nifi-secret")
	secretKey, err := keyManager.GenerateOrLoadKey()
	if err != nil {
		log.Fatalf("生成密钥失败: %v", err)
	}

	// 设置敏感属性提供者
	sensitiveProvider := ConfigManager.NewAESSensitivePropertyProvider(secretKey)
	configManager.SetSensitivePropertyProvider(sensitiveProvider)

	// 设置配置属性
	err = configManager.SetProperty("nifi.web.http.port", "8080", false)
	if err != nil {
		log.Printf("设置HTTP端口失败: %v", err)
	}

	// 设置敏感配置
	err = configManager.SetProperty("db.password", "secret123", true)
	if err != nil {
		log.Printf("设置数据库密码失败: %v", err)
	}

	// 获取配置
	httpPort, err := configManager.GetProperty("nifi.web.http.port")
	if err != nil {
		log.Printf("获取HTTP端口失败: %v", err)
	} else {
		fmt.Printf("HTTP端口: %s\n", httpPort)
	}
	// 注册配置验证器
	portValidator := ConfigManager.NewPortRangeValidator(1024, 65535)
	configManager.RegisterValidator(portValidator)

	dbValidator := ConfigManager.NewDatabaseConfigValidator()
	configManager.RegisterValidator(dbValidator)

	// 验证配置
	result, err := configManager.ValidateConfiguration()
	if err != nil {
		log.Printf("配置验证失败: %v", err)
	} else {
		if result.IsValid {
			fmt.Println("配置验证通过")
		} else {
			fmt.Printf("配置验证失败: %s\n", result.GetErrors())
		}
	}

}
