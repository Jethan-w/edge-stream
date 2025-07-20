package processor

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/crazy/edge-stream/internal/flowfile"
)

// TextSemanticExtractor 文本语义提取器
type TextSemanticExtractor struct {
	*AbstractProcessor
	patterns map[string]*regexp.Regexp
}

func (tse *TextSemanticExtractor) Stop(ctx ProcessContext) error {
	// TODO implement me
	panic("implement me")
}

func (tse *TextSemanticExtractor) Start(ctx ProcessContext) error {
	// TODO implement me
	panic("implement me")
}

// NewTextSemanticExtractor 创建文本语义提取器
func NewTextSemanticExtractor() *TextSemanticExtractor {
	tse := &TextSemanticExtractor{
		AbstractProcessor: &AbstractProcessor{},
		patterns:          make(map[string]*regexp.Regexp),
	}

	// 初始化正则表达式模式
	tse.initializePatterns()

	// 添加关系
	tse.AddRelationship("extracted", "提取成功")
	tse.AddRelationship("error", "提取失败")

	// 添加属性描述符
	tse.AddPropertyDescriptor("extraction.patterns", "提取模式", false)
	tse.AddPropertyDescriptor("extraction.fields", "提取字段", false)

	return tse
}

// Initialize 初始化处理器
func (tse *TextSemanticExtractor) Initialize(ctx ProcessContext) error {
	if err := tse.AbstractProcessor.Initialize(ctx); err != nil {
		return err
	}

	// 加载自定义提取模式
	patternsStr := tse.getProperty("extraction.patterns")
	if patternsStr != "" {
		if err := tse.loadCustomPatterns(patternsStr); err != nil {
			return fmt.Errorf("failed to load custom patterns: %w", err)
		}
	}

	return nil
}

// Process 处理数据
func (tse *TextSemanticExtractor) Process(ctx ProcessContext, flowFile *flowfile.FlowFile) (*flowfile.FlowFile, error) {
	tse.SetState(StateRunning)
	defer tse.SetState(StateReady)

	// 提取文本语义
	text := string(flowFile.Content)
	semantics := tse.ExtractTextSemantics(text)

	// 将提取的语义信息添加到FlowFile属性中
	for key, value := range semantics {
		if value != "" {
			flowFile.Attributes["semantic."+key] = value
		}
	}

	// 添加提取相关属性
	flowFile.Attributes["semantic.extraction.timestamp"] = time.Now().Format(time.RFC3339)
	flowFile.Attributes["semantic.extraction.count"] = fmt.Sprintf("%d", len(semantics))

	return flowFile, nil
}

// initializePatterns 初始化正则表达式模式
func (tse *TextSemanticExtractor) initializePatterns() {
	// 身份证号模式
	tse.patterns["idCard"] = regexp.MustCompile(`\d{17}[\dX]`)

	// 手机号码模式
	tse.patterns["phoneNumber"] = regexp.MustCompile(`1[3-9]\d{9}`)

	// 邮箱模式
	tse.patterns["email"] = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)

	// 银行卡号模式
	tse.patterns["bankCard"] = regexp.MustCompile(`\d{16,19}`)

	// 日期模式
	tse.patterns["date"] = regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)

	// 时间模式
	tse.patterns["time"] = regexp.MustCompile(`\d{2}:\d{2}:\d{2}`)

	// 金额模式
	tse.patterns["amount"] = regexp.MustCompile(`¥?\d+(\.\d{2})?`)

	// URL模式
	tse.patterns["url"] = regexp.MustCompile(`https?://[^\s]+`)

	// IP地址模式
	tse.patterns["ipAddress"] = regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)
}

// ExtractTextSemantics 提取文本语义
func (tse *TextSemanticExtractor) ExtractTextSemantics(text string) map[string]string {
	semantics := make(map[string]string)

	// 提取各种语义信息
	semantics["idCard"] = tse.extractIdCard(text)
	semantics["phoneNumber"] = tse.extractPhoneNumber(text)
	semantics["email"] = tse.extractEmail(text)
	semantics["bankCard"] = tse.extractBankCard(text)
	semantics["date"] = tse.extractDate(text)
	semantics["time"] = tse.extractTime(text)
	semantics["amount"] = tse.extractAmount(text)
	semantics["url"] = tse.extractURL(text)
	semantics["ipAddress"] = tse.extractIPAddress(text)

	// 提取自定义模式
	tse.extractCustomPatterns(text, semantics)

	return semantics
}

// extractIdCard 提取身份证号
func (tse *TextSemanticExtractor) extractIdCard(text string) string {
	if pattern, exists := tse.patterns["idCard"]; exists {
		matches := pattern.FindAllString(text, -1)
		if len(matches) > 0 {
			return matches[0]
		}
	}
	return ""
}

// extractPhoneNumber 提取手机号码
func (tse *TextSemanticExtractor) extractPhoneNumber(text string) string {
	if pattern, exists := tse.patterns["phoneNumber"]; exists {
		matches := pattern.FindAllString(text, -1)
		if len(matches) > 0 {
			return matches[0]
		}
	}
	return ""
}

// extractEmail 提取邮箱
func (tse *TextSemanticExtractor) extractEmail(text string) string {
	if pattern, exists := tse.patterns["email"]; exists {
		matches := pattern.FindAllString(text, -1)
		if len(matches) > 0 {
			return matches[0]
		}
	}
	return ""
}

// extractBankCard 提取银行卡号
func (tse *TextSemanticExtractor) extractBankCard(text string) string {
	if pattern, exists := tse.patterns["bankCard"]; exists {
		matches := pattern.FindAllString(text, -1)
		if len(matches) > 0 {
			return matches[0]
		}
	}
	return ""
}

// extractDate 提取日期
func (tse *TextSemanticExtractor) extractDate(text string) string {
	if pattern, exists := tse.patterns["date"]; exists {
		matches := pattern.FindAllString(text, -1)
		if len(matches) > 0 {
			return matches[0]
		}
	}
	return ""
}

// extractTime 提取时间
func (tse *TextSemanticExtractor) extractTime(text string) string {
	if pattern, exists := tse.patterns["time"]; exists {
		matches := pattern.FindAllString(text, -1)
		if len(matches) > 0 {
			return matches[0]
		}
	}
	return ""
}

// extractAmount 提取金额
func (tse *TextSemanticExtractor) extractAmount(text string) string {
	if pattern, exists := tse.patterns["amount"]; exists {
		matches := pattern.FindAllString(text, -1)
		if len(matches) > 0 {
			return matches[0]
		}
	}
	return ""
}

// extractURL 提取URL
func (tse *TextSemanticExtractor) extractURL(text string) string {
	if pattern, exists := tse.patterns["url"]; exists {
		matches := pattern.FindAllString(text, -1)
		if len(matches) > 0 {
			return matches[0]
		}
	}
	return ""
}

// extractIPAddress 提取IP地址
func (tse *TextSemanticExtractor) extractIPAddress(text string) string {
	if pattern, exists := tse.patterns["ipAddress"]; exists {
		matches := pattern.FindAllString(text, -1)
		if len(matches) > 0 {
			return matches[0]
		}
	}
	return ""
}

// extractCustomPatterns 提取自定义模式
func (tse *TextSemanticExtractor) extractCustomPatterns(text string, semantics map[string]string) {
	for name, pattern := range tse.patterns {
		if !isBuiltinPattern(name) {
			matches := pattern.FindAllString(text, -1)
			if len(matches) > 0 {
				semantics["custom."+name] = matches[0]
			}
		}
	}
}

// isBuiltinPattern 判断是否为内置模式
func isBuiltinPattern(name string) bool {
	builtinPatterns := []string{"idCard", "phoneNumber", "email", "bankCard", "date", "time", "amount", "url", "ipAddress"}
	for _, pattern := range builtinPatterns {
		if name == pattern {
			return true
		}
	}
	return false
}

// loadCustomPatterns 加载自定义模式
func (tse *TextSemanticExtractor) loadCustomPatterns(patternsStr string) error {
	var patterns map[string]string
	if err := json.Unmarshal([]byte(patternsStr), &patterns); err != nil {
		return err
	}

	for name, patternStr := range patterns {
		pattern, err := regexp.Compile(patternStr)
		if err != nil {
			return fmt.Errorf("invalid pattern %s: %w", name, err)
		}
		tse.patterns[name] = pattern
	}

	return nil
}

// getProperty 获取属性值
func (tse *TextSemanticExtractor) getProperty(name string) string {
	// 这里应该从配置中获取属性值
	// 为了简化，返回空字符串
	return ""
}

// StructuredDataSemanticExtractor 结构化数据语义提取器
type StructuredDataSemanticExtractor struct {
	*AbstractProcessor
}

func (sdse *StructuredDataSemanticExtractor) Stop(ctx ProcessContext) error {
	// TODO implement me
	panic("implement me")
}

func (sdse *StructuredDataSemanticExtractor) Start(ctx ProcessContext) error {
	// TODO implement me
	panic("implement me")
}

// NewStructuredDataSemanticExtractor 创建结构化数据语义提取器
func NewStructuredDataSemanticExtractor() *StructuredDataSemanticExtractor {
	sdse := &StructuredDataSemanticExtractor{
		AbstractProcessor: &AbstractProcessor{},
	}

	// 添加关系
	sdse.AddRelationship("extracted", "提取成功")
	sdse.AddRelationship("error", "提取失败")

	// 添加属性描述符
	sdse.AddPropertyDescriptor("data.format", "数据格式", true)
	sdse.AddPropertyDescriptor("extraction.fields", "提取字段", false)

	return sdse
}

// Process 处理数据
func (sdse *StructuredDataSemanticExtractor) Process(ctx ProcessContext, flowFile *flowfile.FlowFile) (*flowfile.FlowFile, error) {
	sdse.SetState(StateRunning)
	defer sdse.SetState(StateReady)

	dataFormat := ctx.Properties["data.format"]
	content := string(flowFile.Content)

	var semantics map[string]interface{}
	var err error

	switch dataFormat {
	case "json":
		semantics, err = sdse.ExtractJsonSemantics(content)
	case "xml":
		semantics, err = sdse.ExtractXmlSemantics(content)
	default:
		return nil, fmt.Errorf("unsupported data format: %s", dataFormat)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to extract semantics: %w", err)
	}

	// 将提取的语义信息添加到FlowFile属性中
	for key, value := range semantics {
		if value != nil {
			flowFile.Attributes["semantic."+key] = fmt.Sprintf("%v", value)
		}
	}

	// 添加提取相关属性
	flowFile.Attributes["semantic.extraction.timestamp"] = time.Now().Format(time.RFC3339)
	flowFile.Attributes["semantic.extraction.format"] = dataFormat
	flowFile.Attributes["semantic.extraction.count"] = fmt.Sprintf("%d", len(semantics))

	return flowFile, nil
}

// ExtractJsonSemantics 提取JSON语义
func (sdse *StructuredDataSemanticExtractor) ExtractJsonSemantics(jsonContent string) (map[string]interface{}, error) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonContent), &data); err != nil {
		return nil, err
	}

	semantics := make(map[string]interface{})

	// 提取常用字段
	semantics["userId"] = sdse.extractNestedValue(data, "user.id")
	semantics["userName"] = sdse.extractNestedValue(data, "user.name")
	semantics["userEmail"] = sdse.extractNestedValue(data, "user.email")
	semantics["transactionAmount"] = sdse.extractNestedValue(data, "transaction.amount")
	semantics["transactionTimestamp"] = sdse.extractNestedValue(data, "transaction.timestamp")
	semantics["orderId"] = sdse.extractNestedValue(data, "order.id")
	semantics["orderStatus"] = sdse.extractNestedValue(data, "order.status")

	// 提取自定义字段
	extractionFields := sdse.getProperty("extraction.fields")
	if extractionFields != "" {
		var fields []string
		if err := json.Unmarshal([]byte(extractionFields), &fields); err == nil {
			for _, field := range fields {
				if value := sdse.extractNestedValue(data, field); value != nil {
					semantics["custom."+field] = value
				}
			}
		}
	}

	return semantics, nil
}

// ExtractXmlSemantics 提取XML语义
func (sdse *StructuredDataSemanticExtractor) ExtractXmlSemantics(xmlContent string) (map[string]interface{}, error) {
	// 简化的XML解析示例
	// 在实际应用中，应该使用proper XML解析库
	semantics := make(map[string]interface{})

	// 使用正则表达式提取XML标签内容
	patterns := map[string]*regexp.Regexp{
		"userId":    regexp.MustCompile(`<userId[^>]*>(.*?)</userId>`),
		"userName":  regexp.MustCompile(`<userName[^>]*>(.*?)</userName>`),
		"userEmail": regexp.MustCompile(`<userEmail[^>]*>(.*?)</userEmail>`),
		"amount":    regexp.MustCompile(`<amount[^>]*>(.*?)</amount>`),
		"timestamp": regexp.MustCompile(`<timestamp[^>]*>(.*?)</timestamp>`),
	}

	for key, pattern := range patterns {
		matches := pattern.FindStringSubmatch(xmlContent)
		if len(matches) > 1 {
			semantics[key] = matches[1]
		}
	}

	return semantics, nil
}

// extractNestedValue 提取嵌套值
func (sdse *StructuredDataSemanticExtractor) extractNestedValue(data map[string]interface{}, path string) interface{} {
	fields := strings.Split(path, ".")
	current := data

	for _, field := range fields {
		if value, exists := current[field]; exists {
			if nested, ok := value.(map[string]interface{}); ok {
				current = nested
			} else {
				return value
			}
		} else {
			return nil
		}
	}

	return current
}

// getProperty 获取属性值
func (sdse *StructuredDataSemanticExtractor) getProperty(name string) string {
	// 这里应该从配置中获取属性值
	// 为了简化，返回空字符串
	return ""
}

// AdvancedSemanticExtractor 高级语义提取器
type AdvancedSemanticExtractor struct {
	*AbstractProcessor
	textExtractor       *TextSemanticExtractor
	structuredExtractor *StructuredDataSemanticExtractor
}

func (ase *AdvancedSemanticExtractor) Stop(ctx ProcessContext) error {
	// TODO implement me
	panic("implement me")
}

func (ase *AdvancedSemanticExtractor) Start(ctx ProcessContext) error {
	// TODO implement me
	panic("implement me")
}

// NewAdvancedSemanticExtractor 创建高级语义提取器
func NewAdvancedSemanticExtractor() *AdvancedSemanticExtractor {
	ase := &AdvancedSemanticExtractor{
		AbstractProcessor:   &AbstractProcessor{},
		textExtractor:       NewTextSemanticExtractor(),
		structuredExtractor: NewStructuredDataSemanticExtractor(),
	}

	// 添加关系
	ase.AddRelationship("extracted", "提取成功")
	ase.AddRelationship("error", "提取失败")

	// 添加属性描述符
	ase.AddPropertyDescriptor("extraction.mode", "提取模式", true)
	ase.AddPropertyDescriptor("confidence.threshold", "置信度阈值", false)

	return ase
}

// Process 处理数据
func (ase *AdvancedSemanticExtractor) Process(ctx ProcessContext, flowFile *flowfile.FlowFile) (*flowfile.FlowFile, error) {
	ase.SetState(StateRunning)
	defer ase.SetState(StateReady)

	extractionMode := ctx.Properties["extraction.mode"]

	switch extractionMode {
	case "text":
		return ase.textExtractor.Process(ctx, flowFile)
	case "structured":
		return ase.structuredExtractor.Process(ctx, flowFile)
	case "hybrid":
		return ase.hybridExtraction(ctx, flowFile)
	default:
		return nil, fmt.Errorf("unsupported extraction mode: %s", extractionMode)
	}
}

// hybridExtraction 混合提取
func (ase *AdvancedSemanticExtractor) hybridExtraction(ctx ProcessContext, flowFile *flowfile.FlowFile) (*flowfile.FlowFile, error) {
	// 首先尝试结构化提取
	result, err := ase.structuredExtractor.Process(ctx, flowFile)
	if err != nil {
		// 如果结构化提取失败，尝试文本提取
		return ase.textExtractor.Process(ctx, flowFile)
	}

	// 如果结构化提取成功，再尝试文本提取补充
	textResult, err := ase.textExtractor.Process(ctx, result)
	if err != nil {
		// 文本提取失败不影响结构化提取结果
		return result, nil
	}

	return textResult, nil
}
