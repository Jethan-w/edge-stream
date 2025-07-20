package processor

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/crazy/edge-stream/internal/flowfile"
)

// RouteProcessor 路由处理器
type RouteProcessor struct {
	*AbstractProcessor
	rules []RouteRule
}

func (rp *RouteProcessor) Stop(ctx ProcessContext) error {
	// TODO implement me
	panic("implement me")
}

func (rp *RouteProcessor) Start(ctx ProcessContext) error {
	// TODO implement me
	panic("implement me")
}

// NewRouteProcessor 创建路由处理器
func NewRouteProcessor() *RouteProcessor {
	rp := &RouteProcessor{
		AbstractProcessor: &AbstractProcessor{},
		rules:             make([]RouteRule, 0),
	}

	// 添加默认关系
	rp.AddRelationship("success", "路由成功")
	rp.AddRelationship("failure", "路由失败")
	rp.AddRelationship("unmatched", "未匹配")

	// 添加属性描述符
	rp.AddPropertyDescriptor("routing.rules", "路由规则", false)
	rp.AddPropertyDescriptor("routing.strategy", "路由策略", false)
	rp.AddPropertyDescriptor("default.relationship", "默认关系", false)

	return rp
}

// Initialize 初始化处理器
func (rp *RouteProcessor) Initialize(ctx ProcessContext) error {
	if err := rp.AbstractProcessor.Initialize(ctx); err != nil {
		return err
	}

	// 解析路由规则
	rulesStr := rp.getProperty("routing.rules")
	if rulesStr != "" {
		if err := rp.parseRoutingRules(rulesStr); err != nil {
			return fmt.Errorf("failed to parse routing rules: %w", err)
		}
	}

	return nil
}

// Process 处理数据
func (rp *RouteProcessor) Process(ctx ProcessContext, flowFile *flowfile.FlowFile) (*flowfile.FlowFile, error) {
	rp.SetState(StateRunning)
	defer rp.SetState(StateReady)

	// 应用路由规则
	targetRelationship, err := rp.routeFlowFile(flowFile, ctx.Properties)
	if err != nil {
		return nil, fmt.Errorf("failed to route flowfile: %w", err)
	}

	// 添加路由相关属性
	flowFile.Attributes["routing.target"] = targetRelationship
	flowFile.Attributes["routing.timestamp"] = time.Now().Format(time.RFC3339)

	return flowFile, nil
}

// routeFlowFile 路由FlowFile
func (rp *RouteProcessor) routeFlowFile(flowFile *flowfile.FlowFile, properties map[string]string) (string, error) {
	// 首先尝试基于属性的路由
	for _, rule := range rp.rules {
		if rule.Evaluate(flowFile) {
			return rule.TargetRelationship, nil
		}
	}

	// 尝试基于内容的路由
	if contentRule := rp.evaluateContentBasedRouting(flowFile, properties); contentRule != "" {
		return contentRule, nil
	}

	// 尝试多维度路由
	if multiRule := rp.evaluateMultiDimensionalRouting(flowFile, properties); multiRule != "" {
		return multiRule, nil
	}

	// 返回默认关系
	defaultRel := properties["default.relationship"]
	if defaultRel == "" {
		defaultRel = "unmatched"
	}

	return defaultRel, nil
}

// evaluateContentBasedRouting 基于内容的路由
func (rp *RouteProcessor) evaluateContentBasedRouting(flowFile *flowfile.FlowFile, properties map[string]string) string {
	content := string(flowFile.Content)

	// 检查是否包含特定关键词
	keywords := strings.Split(properties["content.keywords"], ",")
	for _, keyword := range keywords {
		keyword = strings.TrimSpace(keyword)
		if keyword != "" && strings.Contains(content, keyword) {
			return properties["keyword.relationship"]
		}
	}

	// 检查正则表达式匹配
	pattern := properties["content.pattern"]
	if pattern != "" {
		re, err := regexp.Compile(pattern)
		if err == nil && re.MatchString(content) {
			return properties["pattern.relationship"]
		}
	}

	// 检查JSON字段值
	if jsonField := properties["json.field"]; jsonField != "" {
		if value := rp.extractJsonField(content, jsonField); value != "" {
			expectedValue := properties["json.value"]
			if value == expectedValue {
				return properties["json.relationship"]
			}
		}
	}

	return ""
}

// evaluateMultiDimensionalRouting 多维度路由
func (rp *RouteProcessor) evaluateMultiDimensionalRouting(flowFile *flowfile.FlowFile, properties map[string]string) string {
	// 评估高价值标准
	isHighValue := rp.evaluateHighValueCriteria(flowFile, properties)

	// 评估紧急程度标准
	isUrgent := rp.evaluateUrgencyCriteria(flowFile, properties)

	// 根据多维度条件进行路由
	if isHighValue && isUrgent {
		return "high_priority"
	} else if isHighValue {
		return "standard_priority"
	} else if isUrgent {
		return "urgent_processing"
	} else {
		return "low_priority"
	}
}

// evaluateHighValueCriteria 评估高价值标准
func (rp *RouteProcessor) evaluateHighValueCriteria(flowFile *flowfile.FlowFile, properties map[string]string) bool {
	// 检查文件大小
	if sizeThreshold := properties["high_value.size_threshold"]; sizeThreshold != "" {
		// 实现大小阈值检查逻辑
	}

	// 检查特定属性
	if attrName := properties["high_value.attribute"]; attrName != "" {
		if attrValue, exists := flowFile.Attributes[attrName]; exists {
			expectedValue := properties["high_value.attribute_value"]
			return attrValue == expectedValue
		}
	}

	// 检查内容特征
	content := string(flowFile.Content)
	if highValuePattern := properties["high_value.pattern"]; highValuePattern != "" {
		re, err := regexp.Compile(highValuePattern)
		if err == nil && re.MatchString(content) {
			return true
		}
	}

	return false
}

// evaluateUrgencyCriteria 评估紧急程度标准
func (rp *RouteProcessor) evaluateUrgencyCriteria(flowFile *flowfile.FlowFile, properties map[string]string) bool {
	// 检查时间戳
	if timestampStr, exists := flowFile.Attributes["timestamp"]; exists {
		if timestamp, err := time.Parse(time.RFC3339, timestampStr); err == nil {
			urgencyThreshold := time.Now().Add(-time.Hour) // 1小时内的数据认为是紧急的
			return timestamp.After(urgencyThreshold)
		}
	}

	// 检查紧急标识
	if urgentFlag, exists := flowFile.Attributes["urgent"]; exists {
		return urgentFlag == "true"
	}

	// 检查内容中的紧急关键词
	content := string(flowFile.Content)
	urgentKeywords := []string{"urgent", "critical", "emergency", "immediate"}
	for _, keyword := range urgentKeywords {
		if strings.Contains(strings.ToLower(content), keyword) {
			return true
		}
	}

	return false
}

// extractJsonField 提取JSON字段值
func (rp *RouteProcessor) extractJsonField(content, fieldPath string) string {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return ""
	}

	// 支持嵌套字段路径，如 "user.profile.name"
	fields := strings.Split(fieldPath, ".")
	current := data

	for _, field := range fields {
		if value, exists := current[field]; exists {
			if nested, ok := value.(map[string]interface{}); ok {
				current = nested
			} else {
				return fmt.Sprintf("%v", value)
			}
		} else {
			return ""
		}
	}

	return ""
}

// parseRoutingRules 解析路由规则
func (rp *RouteProcessor) parseRoutingRules(rulesStr string) error {
	var rules []RouteRule
	if err := json.Unmarshal([]byte(rulesStr), &rules); err != nil {
		return err
	}

	rp.rules = rules
	return nil
}

// getProperty 获取属性值
func (rp *RouteProcessor) getProperty(name string) string {
	// 这里应该从配置中获取属性值
	// 为了简化，返回空字符串
	return ""
}

// RouteRule 路由规则
type RouteRule struct {
	AttributeName      string `json:"attribute_name"`
	Condition          string `json:"condition"`
	TargetRelationship string `json:"target_relationship"`
	Operator           string `json:"operator,omitempty"`
	Value              string `json:"value,omitempty"`
}

// Evaluate 评估路由规则
func (rr *RouteRule) Evaluate(flowFile *flowfile.FlowFile) bool {
	attributeValue, exists := flowFile.Attributes[rr.AttributeName]
	if !exists {
		return false
	}

	// 根据操作符进行匹配
	switch rr.Operator {
	case "equals":
		return attributeValue == rr.Value
	case "contains":
		return strings.Contains(attributeValue, rr.Value)
	case "starts_with":
		return strings.HasPrefix(attributeValue, rr.Value)
	case "ends_with":
		return strings.HasSuffix(attributeValue, rr.Value)
	case "regex":
		re, err := regexp.Compile(rr.Value)
		if err != nil {
			return false
		}
		return re.MatchString(attributeValue)
	default:
		// 默认使用简单相等匹配
		return attributeValue == rr.Condition
	}
}

// AdvancedRouteProcessor 高级路由处理器
type AdvancedRouteProcessor struct {
	*RouteProcessor
	priorityQueues map[string]chan *flowfile.FlowFile
}

// NewAdvancedRouteProcessor 创建高级路由处理器
func NewAdvancedRouteProcessor() *AdvancedRouteProcessor {
	arp := &AdvancedRouteProcessor{
		RouteProcessor: NewRouteProcessor(),
		priorityQueues: make(map[string]chan *flowfile.FlowFile),
	}

	// 初始化优先级队列
	arp.priorityQueues["high_priority"] = make(chan *flowfile.FlowFile, 1000)
	arp.priorityQueues["standard_priority"] = make(chan *flowfile.FlowFile, 1000)
	arp.priorityQueues["low_priority"] = make(chan *flowfile.FlowFile, 1000)

	// 添加高级路由关系
	arp.AddRelationship("high_priority", "高优先级队列")
	arp.AddRelationship("standard_priority", "标准优先级队列")
	arp.AddRelationship("low_priority", "低优先级队列")

	return arp
}

// Process 处理数据
func (arp *AdvancedRouteProcessor) Process(ctx ProcessContext, flowFile *flowfile.FlowFile) (*flowfile.FlowFile, error) {
	arp.SetState(StateRunning)
	defer arp.SetState(StateReady)

	// 执行多维度路由决策
	targetQueue := arp.routeByMultipleCriteria(flowFile, ctx.Properties)

	// 将FlowFile发送到相应的优先级队列
	if queue, exists := arp.priorityQueues[targetQueue]; exists {
		select {
		case queue <- flowFile:
			// 成功发送到队列
		default:
			// 队列已满，降级到低优先级队列
			if targetQueue != "low_priority" {
				arp.priorityQueues["low_priority"] <- flowFile
			}
		}
	}

	// 添加路由相关属性
	flowFile.Attributes["routing.queue"] = targetQueue
	flowFile.Attributes["routing.timestamp"] = time.Now().Format(time.RFC3339)

	return flowFile, nil
}

// routeByMultipleCriteria 多维度路由决策
func (arp *AdvancedRouteProcessor) routeByMultipleCriteria(flowFile *flowfile.FlowFile, properties map[string]string) string {
	// 评估高价值标准
	isHighValue := arp.evaluateHighValueCriteria(flowFile, properties)

	// 评估紧急程度标准
	isUrgent := arp.evaluateUrgencyCriteria(flowFile, properties)

	// 根据多维度条件进行路由
	if isHighValue && isUrgent {
		return "high_priority"
	} else if isHighValue {
		return "standard_priority"
	} else {
		return "low_priority"
	}
}

// GetPriorityQueue 获取优先级队列
func (arp *AdvancedRouteProcessor) GetPriorityQueue(queueName string) (chan *flowfile.FlowFile, bool) {
	queue, exists := arp.priorityQueues[queueName]
	return queue, exists
}

// GetQueueStats 获取队列统计信息
func (arp *AdvancedRouteProcessor) GetQueueStats() map[string]int {
	stats := make(map[string]int)
	for queueName, queue := range arp.priorityQueues {
		stats[queueName] = len(queue)
	}
	return stats
}
