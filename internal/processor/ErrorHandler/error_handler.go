package errorhandler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/edge-stream/internal/processor"
)

// ErrorHandler 错误处理器
type ErrorHandler struct {
	*processor.AbstractProcessor
	retryStrategy *processor.RetryStrategy
	errorRules    []ErrorRule
}

// NewErrorHandler 创建错误处理器
func NewErrorHandler() *ErrorHandler {
	eh := &ErrorHandler{
		AbstractProcessor: &processor.AbstractProcessor{},
		retryStrategy: &processor.RetryStrategy{
			MaxRetries: 3,
			BaseDelay:  time.Second,
		},
		errorRules: make([]ErrorRule, 0),
	}

	// 添加关系
	eh.AddRelationship("success", "处理成功")
	eh.AddRelationship("failure", "处理失败")
	eh.AddRelationship("retry", "重试")
	eh.AddRelationship("error", "错误")

	// 添加属性描述符
	eh.AddPropertyDescriptor("error.rules", "错误规则", false)
	eh.AddPropertyDescriptor("retry.max_attempts", "最大重试次数", false)
	eh.AddPropertyDescriptor("retry.base_delay", "基础重试延迟", false)
	eh.AddPropertyDescriptor("error.strategy", "错误处理策略", false)

	return eh
}

// Initialize 初始化处理器
func (eh *ErrorHandler) Initialize(ctx context.Context) error {
	if err := eh.AbstractProcessor.Initialize(ctx); err != nil {
		return err
	}

	// 解析错误规则
	errorRulesStr := eh.getProperty("error.rules")
	if errorRulesStr != "" {
		if err := eh.parseErrorRules(errorRulesStr); err != nil {
			return fmt.Errorf("failed to parse error rules: %w", err)
		}
	}

	// 配置重试策略
	if maxAttempts := eh.getProperty("retry.max_attempts"); maxAttempts != "" {
		// 解析最大重试次数
	}

	if baseDelay := eh.getProperty("retry.base_delay"); baseDelay != "" {
		// 解析基础重试延迟
	}

	return nil
}

// Process 处理数据
func (eh *ErrorHandler) Process(ctx processor.ProcessContext, flowFile *processor.FlowFile) (*processor.FlowFile, error) {
	eh.SetState(processor.StateRunning)
	defer eh.SetState(processor.StateReady)

	// 检查是否有错误信息
	if errorMsg, exists := flowFile.Attributes["error.message"]; exists {
		// 处理错误
		return eh.handleError(flowFile, errorMsg, ctx.Properties)
	}

	// 检查重试次数
	if retryCount := eh.getRetryCount(flowFile); retryCount > 0 {
		// 处理重试逻辑
		return eh.handleRetry(flowFile, retryCount, ctx.Properties)
	}

	// 正常处理
	return flowFile, nil
}

// handleError 处理错误
func (eh *ErrorHandler) handleError(flowFile *processor.FlowFile, errorMsg string, properties map[string]string) (*processor.FlowFile, error) {
	// 分析错误类型
	errorType := eh.classifyError(errorMsg)

	// 应用错误规则
	for _, rule := range eh.errorRules {
		if rule.Matches(errorType, errorMsg) {
			return eh.applyErrorRule(flowFile, rule, errorMsg)
		}
	}

	// 应用默认错误处理策略
	return eh.applyDefaultErrorStrategy(flowFile, errorType, errorMsg, properties)
}

// handleRetry 处理重试
func (eh *ErrorHandler) handleRetry(flowFile *processor.FlowFile, retryCount int, properties map[string]string) (*processor.FlowFile, error) {
	// 检查是否应该重试
	if !eh.retryStrategy.ShouldRetry(retryCount, fmt.Errorf("retry attempt %d", retryCount)) {
		// 超过最大重试次数，路由到失败
		flowFile.Attributes["error.final"] = "true"
		flowFile.Attributes["error.retry_exhausted"] = "true"
		return flowFile, nil
	}

	// 计算重试延迟
	delay := eh.retryStrategy.CalculateRetryDelay(retryCount)

	// 添加重试相关属性
	flowFile.Attributes["retry.count"] = fmt.Sprintf("%d", retryCount+1)
	flowFile.Attributes["retry.delay"] = delay.String()
	flowFile.Attributes["retry.timestamp"] = time.Now().Format(time.RFC3339)

	// 路由到重试队列
	return flowFile, nil
}

// classifyError 分类错误
func (eh *ErrorHandler) classifyError(errorMsg string) ErrorType {
	errorMsg = strings.ToLower(errorMsg)

	// 网络相关错误
	if strings.Contains(errorMsg, "connection") || strings.Contains(errorMsg, "network") || strings.Contains(errorMsg, "timeout") {
		return ErrorTypeNetwork
	}

	// 数据库相关错误
	if strings.Contains(errorMsg, "database") || strings.Contains(errorMsg, "sql") || strings.Contains(errorMsg, "db") {
		return ErrorTypeDatabase
	}

	// 权限相关错误
	if strings.Contains(errorMsg, "permission") || strings.Contains(errorMsg, "unauthorized") || strings.Contains(errorMsg, "forbidden") {
		return ErrorTypePermission
	}

	// 数据格式错误
	if strings.Contains(errorMsg, "format") || strings.Contains(errorMsg, "parse") || strings.Contains(errorMsg, "json") {
		return ErrorTypeDataFormat
	}

	// 资源不足错误
	if strings.Contains(errorMsg, "memory") || strings.Contains(errorMsg, "disk") || strings.Contains(errorMsg, "resource") {
		return ErrorTypeResource
	}

	// 业务逻辑错误
	if strings.Contains(errorMsg, "business") || strings.Contains(errorMsg, "validation") || strings.Contains(errorMsg, "invalid") {
		return ErrorTypeBusiness
	}

	// 默认未知错误
	return ErrorTypeUnknown
}

// applyErrorRule 应用错误规则
func (eh *ErrorHandler) applyErrorRule(flowFile *processor.FlowFile, rule ErrorRule, errorMsg string) (*processor.FlowFile, error) {
	// 添加错误处理相关属性
	flowFile.Attributes["error.rule.applied"] = rule.Name
	flowFile.Attributes["error.handling.timestamp"] = time.Now().Format(time.RFC3339)

	switch rule.Action {
	case ErrorActionRetry:
		// 路由到重试
		flowFile.Attributes["retry.count"] = "1"
		flowFile.Attributes["retry.timestamp"] = time.Now().Format(time.RFC3339)
		return flowFile, nil

	case ErrorActionRouteToFailure:
		// 路由到失败
		flowFile.Attributes["error.final"] = "true"
		return flowFile, nil

	case ErrorActionRouteToSuccess:
		// 路由到成功
		delete(flowFile.Attributes, "error.message")
		return flowFile, nil

	case ErrorActionDrop:
		// 丢弃
		return nil, fmt.Errorf("flowfile dropped due to error rule: %s", rule.Name)

	default:
		// 默认路由到错误
		return flowFile, nil
	}
}

// applyDefaultErrorStrategy 应用默认错误处理策略
func (eh *ErrorHandler) applyDefaultErrorStrategy(flowFile *processor.FlowFile, errorType ErrorType, errorMsg string, properties map[string]string) (*processor.FlowFile, error) {
	strategy := properties["error.strategy"]

	switch strategy {
	case "retry":
		// 默认重试策略
		if errorType == ErrorTypeNetwork || errorType == ErrorTypeResource {
			flowFile.Attributes["retry.count"] = "1"
			flowFile.Attributes["retry.timestamp"] = time.Now().Format(time.RFC3339)
			return flowFile, nil
		}

	case "fail_fast":
		// 快速失败策略
		flowFile.Attributes["error.final"] = "true"
		return flowFile, nil

	case "ignore":
		// 忽略错误策略
		delete(flowFile.Attributes, "error.message")
		return flowFile, nil

	default:
		// 默认策略：根据错误类型决定
		if errorType == ErrorTypeNetwork || errorType == ErrorTypeResource {
			// 可恢复错误，重试
			flowFile.Attributes["retry.count"] = "1"
			flowFile.Attributes["retry.timestamp"] = time.Now().Format(time.RFC3339)
		} else {
			// 不可恢复错误，失败
			flowFile.Attributes["error.final"] = "true"
		}
	}

	return flowFile, nil
}

// getRetryCount 获取重试次数
func (eh *ErrorHandler) getRetryCount(flowFile *processor.FlowFile) int {
	if retryCountStr, exists := flowFile.Attributes["retry.count"]; exists {
		// 解析重试次数
		var retryCount int
		fmt.Sscanf(retryCountStr, "%d", &retryCount)
		return retryCount
	}
	return 0
}

// parseErrorRules 解析错误规则
func (eh *ErrorHandler) parseErrorRules(rulesStr string) error {
	// 这里应该解析JSON格式的错误规则
	// 为了简化，使用硬编码的规则
	eh.errorRules = []ErrorRule{
		{
			Name:       "network_retry",
			ErrorType:  ErrorTypeNetwork,
			Pattern:    "connection.*timeout",
			Action:     ErrorActionRetry,
			MaxRetries: 3,
		},
		{
			Name:       "permission_fail",
			ErrorType:  ErrorTypePermission,
			Pattern:    "permission.*denied",
			Action:     ErrorActionRouteToFailure,
			MaxRetries: 0,
		},
		{
			Name:       "format_ignore",
			ErrorType:  ErrorTypeDataFormat,
			Pattern:    "invalid.*format",
			Action:     ErrorActionRouteToSuccess,
			MaxRetries: 0,
		},
	}

	return nil
}

// getProperty 获取属性值
func (eh *ErrorHandler) getProperty(name string) string {
	// 这里应该从配置中获取属性值
	// 为了简化，返回空字符串
	return ""
}

// ErrorType 错误类型
type ErrorType int

const (
	ErrorTypeUnknown ErrorType = iota
	ErrorTypeNetwork
	ErrorTypeDatabase
	ErrorTypePermission
	ErrorTypeDataFormat
	ErrorTypeResource
	ErrorTypeBusiness
)

// ErrorAction 错误处理动作
type ErrorAction int

const (
	ErrorActionRetry ErrorAction = iota
	ErrorActionRouteToFailure
	ErrorActionRouteToSuccess
	ErrorActionDrop
)

// ErrorRule 错误规则
type ErrorRule struct {
	Name       string      `json:"name"`
	ErrorType  ErrorType   `json:"error_type"`
	Pattern    string      `json:"pattern"`
	Action     ErrorAction `json:"action"`
	MaxRetries int         `json:"max_retries"`
}

// Matches 检查错误是否匹配规则
func (er *ErrorRule) Matches(errorType ErrorType, errorMsg string) bool {
	// 检查错误类型
	if er.ErrorType != ErrorTypeUnknown && er.ErrorType != errorType {
		return false
	}

	// 检查错误消息模式
	if er.Pattern != "" {
		// 这里应该使用正则表达式匹配
		// 为了简化，使用字符串包含
		return strings.Contains(strings.ToLower(errorMsg), strings.ToLower(er.Pattern))
	}

	return true
}

// TransientException 临时异常
type TransientException struct {
	Message string
}

func (te *TransientException) Error() string {
	return te.Message
}

// PermanentException 永久异常
type PermanentException struct {
	Message string
}

func (pe *PermanentException) Error() string {
	return pe.Message
}

// ErrorHandlingProcessor 错误处理处理器
type ErrorHandlingProcessor struct {
	*ErrorHandler
}

// NewErrorHandlingProcessor 创建错误处理处理器
func NewErrorHandlingProcessor() *ErrorHandlingProcessor {
	ehp := &ErrorHandlingProcessor{
		ErrorHandler: NewErrorHandler(),
	}

	// 添加自定义关系
	ehp.AddRelationship("transient_error", "临时错误")
	ehp.AddRelationship("permanent_error", "永久错误")

	return ehp
}

// Process 处理数据
func (ehp *ErrorHandlingProcessor) Process(ctx processor.ProcessContext, flowFile *processor.FlowFile) (*processor.FlowFile, error) {
	ehp.SetState(processor.StateRunning)
	defer ehp.SetState(processor.StateReady)

	// 尝试处理FlowFile
	err := ehp.processFlowFile(flowFile)

	if err != nil {
		// 根据错误类型进行路由
		switch e := err.(type) {
		case *TransientException:
			// 临时错误，可以重试
			flowFile.Attributes["error.type"] = "transient"
			flowFile.Attributes["error.message"] = e.Message
			flowFile.Attributes["error.timestamp"] = time.Now().Format(time.RFC3339)
			return flowFile, nil

		case *PermanentException:
			// 永久错误，不能重试
			flowFile.Attributes["error.type"] = "permanent"
			flowFile.Attributes["error.message"] = e.Message
			flowFile.Attributes["error.timestamp"] = time.Now().Format(time.RFC3339)
			return flowFile, nil

		default:
			// 未知错误，默认为临时错误
			flowFile.Attributes["error.type"] = "unknown"
			flowFile.Attributes["error.message"] = err.Error()
			flowFile.Attributes["error.timestamp"] = time.Now().Format(time.RFC3339)
			return flowFile, nil
		}
	}

	// 处理成功
	return flowFile, nil
}

// processFlowFile 处理FlowFile
func (ehp *ErrorHandlingProcessor) processFlowFile(flowFile *processor.FlowFile) error {
	// 这里实现具体的处理逻辑
	// 为了演示，随机生成错误

	// 模拟处理逻辑
	content := string(flowFile.Content)

	// 检查内容是否包含错误关键词
	if strings.Contains(content, "network_error") {
		return &TransientException{Message: "Network connection timeout"}
	}

	if strings.Contains(content, "permission_error") {
		return &PermanentException{Message: "Permission denied"}
	}

	if strings.Contains(content, "format_error") {
		return &PermanentException{Message: "Invalid data format"}
	}

	// 正常处理
	return nil
}
