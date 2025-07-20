package scriptprocessor

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/edge-stream/internal/processor"
)

// ScriptProcessor 脚本处理器
type ScriptProcessor struct {
	*processor.AbstractProcessor
	scriptEngine ScriptEngine
	scriptCache  map[string]CompiledScript
}

// NewScriptProcessor 创建脚本处理器
func NewScriptProcessor() *ScriptProcessor {
	sp := &ScriptProcessor{
		AbstractProcessor: &processor.AbstractProcessor{},
		scriptCache:       make(map[string]CompiledScript),
	}

	// 添加关系
	sp.AddRelationship("success", "脚本执行成功")
	sp.AddRelationship("error", "脚本执行失败")

	// 添加属性描述符
	sp.AddPropertyDescriptor("script.language", "脚本语言", true)
	sp.AddPropertyDescriptor("script.content", "脚本内容", true)
	sp.AddPropertyDescriptor("script.cache.enabled", "启用脚本缓存", false)
	sp.AddPropertyDescriptor("script.timeout", "脚本执行超时", false)

	return sp
}

// Initialize 初始化处理器
func (sp *ScriptProcessor) Initialize(ctx context.Context) error {
	if err := sp.AbstractProcessor.Initialize(ctx); err != nil {
		return err
	}

	// 初始化脚本引擎
	sp.scriptEngine = NewScriptEngine()

	return nil
}

// Process 处理数据
func (sp *ScriptProcessor) Process(ctx processor.ProcessContext, flowFile *processor.FlowFile) (*processor.FlowFile, error) {
	sp.SetState(processor.StateRunning)
	defer sp.SetState(processor.StateReady)

	scriptLanguage := ctx.Properties["script.language"]
	scriptContent := ctx.Properties["script.content"]
	cacheEnabled := ctx.Properties["script.cache.enabled"] == "true"

	// 检查脚本缓存
	var compiledScript CompiledScript
	var err error

	if cacheEnabled {
		cacheKey := fmt.Sprintf("%s:%s", scriptLanguage, scriptContent)
		if cached, exists := sp.scriptCache[cacheKey]; exists {
			compiledScript = cached
		} else {
			compiledScript, err = sp.scriptEngine.Compile(scriptLanguage, scriptContent)
			if err != nil {
				return nil, fmt.Errorf("failed to compile script: %w", err)
			}
			sp.scriptCache[cacheKey] = compiledScript
		}
	} else {
		compiledScript, err = sp.scriptEngine.Compile(scriptLanguage, scriptContent)
		if err != nil {
			return nil, fmt.Errorf("failed to compile script: %w", err)
		}
	}

	// 创建脚本执行上下文
	scriptContext := &ScriptExecutionContext{
		FlowFile:   flowFile,
		Session:    ctx.Session,
		Properties: ctx.Properties,
		State:      ctx.State,
	}

	// 执行脚本
	result, err := sp.executeScript(compiledScript, scriptContext, ctx.Properties)
	if err != nil {
		return nil, fmt.Errorf("failed to execute script: %w", err)
	}

	// 处理脚本执行结果
	return sp.handleScriptResult(result, flowFile, ctx.Properties)
}

// executeScript 执行脚本
func (sp *ScriptProcessor) executeScript(script CompiledScript, context *ScriptExecutionContext, properties map[string]string) (interface{}, error) {
	// 设置执行超时
	timeoutStr := properties["script.timeout"]
	var timeout time.Duration = 30 * time.Second // 默认30秒

	if timeoutStr != "" {
		if parsed, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = parsed
		}
	}

	// 创建带超时的执行上下文
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 执行脚本
	resultChan := make(chan interface{}, 1)
	errChan := make(chan error, 1)

	go func() {
		result, err := sp.scriptEngine.Execute(script, context)
		if err != nil {
			errChan <- err
		} else {
			resultChan <- result
		}
	}()

	// 等待执行结果或超时
	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errChan:
		return nil, err
	case <-ctx.Done():
		return nil, fmt.Errorf("script execution timeout after %v", timeout)
	}
}

// handleScriptResult 处理脚本执行结果
func (sp *ScriptProcessor) handleScriptResult(result interface{}, flowFile *processor.FlowFile, properties map[string]string) (*processor.FlowFile, error) {
	if result == nil {
		return flowFile, nil
	}

	// 根据结果类型进行处理
	switch v := result.(type) {
	case *processor.FlowFile:
		// 直接返回新的FlowFile
		return v, nil

	case map[string]interface{}:
		// 更新FlowFile属性
		for key, value := range v {
			flowFile.Attributes[key] = fmt.Sprintf("%v", value)
		}
		return flowFile, nil

	case string:
		// 更新FlowFile内容
		flowFile.Content = []byte(v)
		flowFile.Size = int64(len(v))
		return flowFile, nil

	case []byte:
		// 更新FlowFile内容
		flowFile.Content = v
		flowFile.Size = int64(len(v))
		return flowFile, nil

	default:
		// 其他类型，转换为字符串
		flowFile.Attributes["script.result"] = fmt.Sprintf("%v", v)
		return flowFile, nil
	}
}

// ScriptEngine 脚本引擎接口
type ScriptEngine interface {
	Compile(language, content string) (CompiledScript, error)
	Execute(script CompiledScript, context *ScriptExecutionContext) (interface{}, error)
}

// CompiledScript 编译后的脚本
type CompiledScript interface {
	Language() string
	Content() string
}

// ScriptExecutionContext 脚本执行上下文
type ScriptExecutionContext struct {
	FlowFile   *processor.FlowFile
	Session    *processor.ProcessSession
	Properties map[string]string
	State      map[string]interface{}
}

// Background 返回后台上下文
func (sec *ScriptExecutionContext) Background() context.Context {
	return context.Background()
}

// DefaultScriptEngine 默认脚本引擎实现
type DefaultScriptEngine struct {
	engines map[string]ScriptLanguageEngine
}

// NewScriptEngine 创建脚本引擎
func NewScriptEngine() *DefaultScriptEngine {
	dse := &DefaultScriptEngine{
		engines: make(map[string]ScriptLanguageEngine),
	}

	// 注册支持的脚本语言
	dse.engines["javascript"] = &JavaScriptEngine{}
	dse.engines["python"] = &PythonEngine{}
	dse.engines["lua"] = &LuaEngine{}
	dse.engines["groovy"] = &GroovyEngine{}

	return dse
}

// Compile 编译脚本
func (dse *DefaultScriptEngine) Compile(language, content string) (CompiledScript, error) {
	engine, exists := dse.engines[strings.ToLower(language)]
	if !exists {
		return nil, fmt.Errorf("unsupported script language: %s", language)
	}

	return engine.Compile(content)
}

// Execute 执行脚本
func (dse *DefaultScriptEngine) Execute(script CompiledScript, context *ScriptExecutionContext) (interface{}, error) {
	engine, exists := dse.engines[strings.ToLower(script.Language())]
	if !exists {
		return nil, fmt.Errorf("unsupported script language: %s", script.Language())
	}

	return engine.Execute(script, context)
}

// ScriptLanguageEngine 脚本语言引擎接口
type ScriptLanguageEngine interface {
	Compile(content string) (CompiledScript, error)
	Execute(script CompiledScript, context *ScriptExecutionContext) (interface{}, error)
}

// JavaScriptEngine JavaScript引擎
type JavaScriptEngine struct{}

// Compile 编译JavaScript脚本
func (jse *JavaScriptEngine) Compile(content string) (CompiledScript, error) {
	// 这里应该使用实际的JavaScript引擎（如Otto、goja等）
	// 为了简化，返回一个简单的实现
	return &SimpleCompiledScript{
		language: "javascript",
		content:  content,
	}, nil
}

// Execute 执行JavaScript脚本
func (jse *JavaScriptEngine) Execute(script CompiledScript, context *ScriptExecutionContext) (interface{}, error) {
	// 这里应该使用实际的JavaScript引擎执行脚本
	// 为了简化，返回一个示例实现

	content := script.Content()

	// 简单的脚本处理示例
	if strings.Contains(content, "flowFile.content") {
		// 模拟修改FlowFile内容
		newContent := strings.ReplaceAll(string(context.FlowFile.Content), "old", "new")
		context.FlowFile.Content = []byte(newContent)
		context.FlowFile.Size = int64(len(newContent))
	}

	if strings.Contains(content, "flowFile.attributes") {
		// 模拟修改FlowFile属性
		context.FlowFile.Attributes["script.processed"] = "true"
		context.FlowFile.Attributes["script.timestamp"] = time.Now().Format(time.RFC3339)
	}

	return context.FlowFile, nil
}

// PythonEngine Python引擎
type PythonEngine struct{}

// Compile 编译Python脚本
func (pe *PythonEngine) Compile(content string) (CompiledScript, error) {
	return &SimpleCompiledScript{
		language: "python",
		content:  content,
	}, nil
}

// Execute 执行Python脚本
func (pe *PythonEngine) Execute(script CompiledScript, context *ScriptExecutionContext) (interface{}, error) {
	// 这里应该使用实际的Python引擎执行脚本
	// 为了简化，返回一个示例实现

	content := script.Content()

	// 简单的脚本处理示例
	if strings.Contains(content, "print") {
		// 模拟打印输出
		context.FlowFile.Attributes["python.output"] = "Hello from Python script"
	}

	return context.FlowFile, nil
}

// LuaEngine Lua引擎
type LuaEngine struct{}

// Compile 编译Lua脚本
func (le *LuaEngine) Compile(content string) (CompiledScript, error) {
	return &SimpleCompiledScript{
		language: "lua",
		content:  content,
	}, nil
}

// Execute 执行Lua脚本
func (le *LuaEngine) Execute(script CompiledScript, context *ScriptExecutionContext) (interface{}, error) {
	// 这里应该使用实际的Lua引擎执行脚本
	// 为了简化，返回一个示例实现

	content := script.Content()

	// 简单的脚本处理示例
	if strings.Contains(content, "return") {
		// 模拟返回值
		return map[string]interface{}{
			"lua.result": "Hello from Lua script",
		}, nil
	}

	return context.FlowFile, nil
}

// GroovyEngine Groovy引擎
type GroovyEngine struct{}

// Compile 编译Groovy脚本
func (ge *GroovyEngine) Compile(content string) (CompiledScript, error) {
	return &SimpleCompiledScript{
		language: "groovy",
		content:  content,
	}, nil
}

// Execute 执行Groovy脚本
func (ge *GroovyEngine) Execute(script CompiledScript, context *ScriptExecutionContext) (interface{}, error) {
	// 这里应该使用实际的Groovy引擎执行脚本
	// 为了简化，返回一个示例实现

	content := script.Content()

	// 简单的脚本处理示例
	if strings.Contains(content, "println") {
		// 模拟打印输出
		context.FlowFile.Attributes["groovy.output"] = "Hello from Groovy script"
	}

	return context.FlowFile, nil
}

// SimpleCompiledScript 简单的编译脚本实现
type SimpleCompiledScript struct {
	language string
	content  string
}

// Language 获取脚本语言
func (scs *SimpleCompiledScript) Language() string {
	return scs.language
}

// Content 获取脚本内容
func (scs *SimpleCompiledScript) Content() string {
	return scs.content
}

// CustomProcessor 自定义处理器
type CustomProcessor struct {
	*processor.AbstractProcessor
	scriptProcessor *ScriptProcessor
}

// NewCustomProcessor 创建自定义处理器
func NewCustomProcessor() *CustomProcessor {
	cp := &CustomProcessor{
		AbstractProcessor: &processor.AbstractProcessor{},
		scriptProcessor:   NewScriptProcessor(),
	}

	// 添加关系
	cp.AddRelationship("custom.success", "自定义处理成功")
	cp.AddRelationship("custom.error", "自定义处理失败")

	// 添加属性描述符
	cp.AddPropertyDescriptor("custom.script", "自定义脚本", true)
	cp.AddPropertyDescriptor("custom.language", "脚本语言", true)

	return cp
}

// Process 处理数据
func (cp *CustomProcessor) Process(ctx processor.ProcessContext, flowFile *processor.FlowFile) (*processor.FlowFile, error) {
	cp.SetState(processor.StateRunning)
	defer cp.SetState(processor.StateReady)

	// 获取自定义脚本
	scriptContent := ctx.Properties["custom.script"]
	scriptLanguage := ctx.Properties["custom.language"]

	if scriptContent == "" {
		return nil, fmt.Errorf("custom script is required")
	}

	// 创建脚本处理上下文
	scriptContext := processor.ProcessContext{
		Properties: map[string]string{
			"script.language": scriptLanguage,
			"script.content":  scriptContent,
		},
		Session: ctx.Session,
		State:   ctx.State,
	}

	// 执行自定义脚本
	result, err := cp.scriptProcessor.Process(scriptContext, flowFile)
	if err != nil {
		return nil, fmt.Errorf("failed to execute custom script: %w", err)
	}

	// 添加自定义处理标记
	result.Attributes["custom.processed"] = "true"
	result.Attributes["custom.timestamp"] = time.Now().Format(time.RFC3339)

	return result, nil
}

// DynamicProcessor 动态处理器
type DynamicProcessor struct {
	*processor.AbstractProcessor
	scriptEngine *DefaultScriptEngine
}

// NewDynamicProcessor 创建动态处理器
func NewDynamicProcessor() *DynamicProcessor {
	dp := &DynamicProcessor{
		AbstractProcessor: &processor.AbstractProcessor{},
		scriptEngine:      NewScriptEngine(),
	}

	// 添加关系
	dp.AddRelationship("dynamic.success", "动态处理成功")
	dp.AddRelationship("dynamic.error", "动态处理失败")

	// 添加属性描述符
	dp.AddPropertyDescriptor("dynamic.script", "动态脚本", true)
	dp.AddPropertyDescriptor("dynamic.language", "脚本语言", true)
	dp.AddPropertyDescriptor("dynamic.context", "执行上下文", false)

	return dp
}

// Process 处理数据
func (dp *DynamicProcessor) Process(ctx processor.ProcessContext, flowFile *processor.FlowFile) (*processor.FlowFile, error) {
	dp.SetState(processor.StateRunning)
	defer dp.SetState(processor.StateReady)

	scriptContent := ctx.Properties["dynamic.script"]
	scriptLanguage := ctx.Properties["dynamic.language"]

	// 编译脚本
	compiledScript, err := dp.scriptEngine.Compile(scriptLanguage, scriptContent)
	if err != nil {
		return nil, fmt.Errorf("failed to compile dynamic script: %w", err)
	}

	// 创建执行上下文
	scriptContext := &ScriptExecutionContext{
		FlowFile:   flowFile,
		Session:    ctx.Session,
		Properties: ctx.Properties,
		State:      ctx.State,
	}

	// 执行脚本
	result, err := dp.scriptEngine.Execute(compiledScript, scriptContext)
	if err != nil {
		return nil, fmt.Errorf("failed to execute dynamic script: %w", err)
	}

	// 处理结果
	if resultFlowFile, ok := result.(*processor.FlowFile); ok {
		resultFlowFile.Attributes["dynamic.processed"] = "true"
		resultFlowFile.Attributes["dynamic.timestamp"] = time.Now().Format(time.RFC3339)
		return resultFlowFile, nil
	}

	// 如果结果不是FlowFile，创建新的FlowFile
	newFlowFile := &processor.FlowFile{
		ID:         fmt.Sprintf("dynamic_%d", time.Now().UnixNano()),
		Content:    []byte(fmt.Sprintf("%v", result)),
		Attributes: flowFile.Attributes,
		Size:       int64(len(fmt.Sprintf("%v", result))),
		Timestamp:  time.Now(),
		LineageID:  flowFile.LineageID,
	}

	newFlowFile.Attributes["dynamic.processed"] = "true"
	newFlowFile.Attributes["dynamic.timestamp"] = time.Now().Format(time.RFC3339)
	newFlowFile.Attributes["dynamic.result.type"] = reflect.TypeOf(result).String()

	return newFlowFile, nil
}
