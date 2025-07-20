package ConfigManager

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"text/template"
	"time"
)

// EnvironmentInjector 环境变量注入器
type EnvironmentInjector struct {
	envPattern *regexp.Regexp
}

// NewEnvironmentInjector 创建新的环境变量注入器
func NewEnvironmentInjector() *EnvironmentInjector {
	return &EnvironmentInjector{
		envPattern: regexp.MustCompile(`\$\{ENV:([^}]+)\}`),
	}
}

// InjectEnvironmentVariables 注入环境变量
func (ei *EnvironmentInjector) InjectEnvironmentVariables(configValue string) string {
	if configValue == "" {
		return configValue
	}

	// 使用正则表达式查找环境变量引用
	result := ei.envPattern.ReplaceAllStringFunc(configValue, func(match string) string {
		// 提取环境变量名
		submatches := ei.envPattern.FindStringSubmatch(match)
		if len(submatches) != 2 {
			return match // 保留原始表达式
		}

		envKey := submatches[1]
		envValue := os.Getenv(envKey)

		if envValue == "" {
			// 未找到环境变量，保留原始表达式
			return match
		}

		return envValue
	})

	return result
}

// ExpressionLanguageProcessor 表达式语言处理器
type ExpressionLanguageProcessor struct {
	templateFuncs template.FuncMap
}

// NewExpressionLanguageProcessor 创建新的表达式语言处理器
func NewExpressionLanguageProcessor() *ExpressionLanguageProcessor {
	return &ExpressionLanguageProcessor{
		templateFuncs: make(template.FuncMap),
	}
}

// EvaluateExpression 计算表达式
func (elp *ExpressionLanguageProcessor) EvaluateExpression(expression string) (string, error) {
	if expression == "" {
		return "", nil
	}

	// 检查是否为简单的环境变量表达式
	if strings.HasPrefix(expression, "${ENV:") && strings.HasSuffix(expression, "}") {
		envKey := strings.TrimPrefix(strings.TrimSuffix(expression, "}"), "${ENV:")
		return os.Getenv(envKey), nil
	}

	// 使用Go模板引擎处理复杂表达式
	tmpl, err := template.New("expression").Funcs(elp.templateFuncs).Parse(expression)
	if err != nil {
		// 表达式解析失败，返回原始表达式
		return expression, nil
	}

	// 创建模板数据上下文
	data := elp.createTemplateData()

	var result strings.Builder
	err = tmpl.Execute(&result, data)
	if err != nil {
		// 表达式执行失败，返回原始表达式
		return expression, nil
	}

	return result.String(), nil
}

// createTemplateData 创建模板数据上下文
func (elp *ExpressionLanguageProcessor) createTemplateData() map[string]interface{} {
	data := make(map[string]interface{})

	// 注入环境变量
	envVars := make(map[string]string)
	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) == 2 {
			envVars[pair[0]] = pair[1]
		}
	}
	data["env"] = envVars

	// 注入系统属性（简化实现）
	systemProps := make(map[string]string)
	// 这里可以添加更多系统属性
	systemProps["os"] = "linux"   // 示例
	systemProps["arch"] = "amd64" // 示例
	data["props"] = systemProps

	return data
}

// RegisterFunction 注册自定义函数
func (elp *ExpressionLanguageProcessor) RegisterFunction(name string, fn interface{}) {
	elp.templateFuncs[name] = fn
}

// ConfigValueProcessor 配置值处理器
type ConfigValueProcessor struct {
	envInjector   *EnvironmentInjector
	exprProcessor *ExpressionLanguageProcessor
}

// NewConfigValueProcessor 创建新的配置值处理器
func NewConfigValueProcessor() *ConfigValueProcessor {
	return &ConfigValueProcessor{
		envInjector:   NewEnvironmentInjector(),
		exprProcessor: NewExpressionLanguageProcessor(),
	}
}

// ProcessConfigValue 处理配置值
func (cvp *ConfigValueProcessor) ProcessConfigValue(configValue string) (string, error) {
	if configValue == "" {
		return configValue, nil
	}

	// 第一步：注入环境变量
	processedValue := cvp.envInjector.InjectEnvironmentVariables(configValue)

	// 第二步：处理表达式语言
	result, err := cvp.exprProcessor.EvaluateExpression(processedValue)
	if err != nil {
		return "", fmt.Errorf("处理配置值失败: %w", err)
	}

	return result, nil
}

// ProcessConfigMap 处理整个配置映射
func (cvp *ConfigValueProcessor) ProcessConfigMap(configMap map[string]string) (map[string]string, error) {
	result := make(map[string]string)

	for key, value := range configMap {
		processedValue, err := cvp.ProcessConfigValue(value)
		if err != nil {
			return nil, fmt.Errorf("处理配置键 '%s' 失败: %w", key, err)
		}
		result[key] = processedValue
	}

	return result, nil
}

// VariableResolver 变量解析器
type VariableResolver struct {
	resolvers map[string]func() (string, error)
}

// NewVariableResolver 创建新的变量解析器
func NewVariableResolver() *VariableResolver {
	return &VariableResolver{
		resolvers: make(map[string]func() (string, error)),
	}
}

// RegisterResolver 注册变量解析器
func (vr *VariableResolver) RegisterResolver(name string, resolver func() (string, error)) {
	vr.resolvers[name] = resolver
}

// ResolveVariable 解析变量
func (vr *VariableResolver) ResolveVariable(name string) (string, error) {
	resolver, exists := vr.resolvers[name]
	if !exists {
		return "", fmt.Errorf("未找到变量解析器: %s", name)
	}

	return resolver()
}

// ResolveVariables 解析字符串中的所有变量
func (vr *VariableResolver) ResolveVariables(input string) (string, error) {
	// 使用正则表达式查找变量引用
	pattern := regexp.MustCompile(`\$\{([^}]+)\}`)

	result := pattern.ReplaceAllStringFunc(input, func(match string) string {
		// 提取变量名
		submatches := pattern.FindStringSubmatch(match)
		if len(submatches) != 2 {
			return match // 保留原始表达式
		}

		varName := submatches[1]
		resolvedValue, err := vr.ResolveVariable(varName)
		if err != nil {
			// 解析失败，保留原始表达式
			return match
		}

		return resolvedValue
	})

	return result, nil
}

// DefaultVariableResolvers 默认变量解析器
func DefaultVariableResolvers() *VariableResolver {
	resolver := NewVariableResolver()

	// 注册环境变量解析器
	resolver.RegisterResolver("ENV", func() (string, error) {
		// 这里可以返回环境变量的JSON表示
		return "{}", nil
	})

	// 注册系统属性解析器
	resolver.RegisterResolver("PROPS", func() (string, error) {
		// 这里可以返回系统属性的JSON表示
		return "{}", nil
	})

	// 注册时间解析器
	resolver.RegisterResolver("TIMESTAMP", func() (string, error) {
		return fmt.Sprintf("%d", time.Now().Unix()), nil
	})

	return resolver
}
