package example

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/crazy/edge-stream/internal/flowfile"
	"github.com/crazy/edge-stream/internal/processor"
)

// ExampleProcessor 示例处理器
func ExampleProcessor() {
	fmt.Println("=== EdgeStream Processor 示例 ===")

	// 创建处理器管理器
	manager := processor.NewProcessorManager()

	// 注册各种处理器
	registerProcessors(manager)

	// 运行示例
	runExamples(manager)
}

// registerProcessors 注册处理器
func registerProcessors(manager *processor.ProcessorManager) {
	// 注册数据转换处理器
	manager.RegisterProcessor("record_transformer", processor.NewRecordTransformer())
	manager.RegisterProcessor("content_modifier", processor.NewContentModifier())
	manager.RegisterProcessor("format_converter", processor.NewFormatConverter())

	// 注册路由处理器
	manager.RegisterProcessor("route_processor", processor.NewRouteProcessor())
	manager.RegisterProcessor("advanced_route_processor", processor.NewAdvancedRouteProcessor())

	// 注册语义提取处理器
	manager.RegisterProcessor("text_semantic_extractor", processor.NewTextSemanticExtractor())
	manager.RegisterProcessor("structured_semantic_extractor", processor.NewStructuredDataSemanticExtractor())
	manager.RegisterProcessor("advanced_semantic_extractor", processor.NewAdvancedSemanticExtractor())

	// 注册错误处理处理器
	manager.RegisterProcessor("error_handler", processor.NewErrorHandler())
	manager.RegisterProcessor("error_handling_processor", processor.NewErrorHandlingProcessor())

	// 注册脚本处理器
	manager.RegisterProcessor("script_processor", processor.NewScriptProcessor())
	manager.RegisterProcessor("custom_processor", processor.NewCustomProcessor())
	manager.RegisterProcessor("dynamic_processor", processor.NewDynamicProcessor())

	fmt.Printf("已注册 %d 个处理器\n", len(manager.ListProcessors()))
}

// runExamples 运行示例
func runExamples(manager *processor.ProcessorManager) {
	// 示例1：数据转换
	exampleDataTransformation(manager)

	// 示例2：路由处理
	exampleRouting(manager)

	// 示例3：语义提取
	exampleSemanticExtraction(manager)

	// 示例4：错误处理
	exampleErrorHandling(manager)

	// 示例5：脚本处理
	exampleScriptProcessing(manager)
}

// exampleDataTransformation 数据转换示例
func exampleDataTransformation(manager *processor.ProcessorManager) {
	fmt.Println("\n--- 数据转换示例 ---")

	// 创建测试数据
	testData := []map[string]interface{}{
		{
			"user_id":       "12345",
			"user_name":     "张三",
			"email_address": "zhangsan@example.com",
			"age":           30,
		},
		{
			"user_id":       "67890",
			"user_name":     "李四",
			"email_address": "lisi@example.com",
			"age":           25,
		},
	}

	jsonData, _ := json.Marshal(testData)

	flowFile := &flowfile.FlowFile{
		UUID:       "test_flowfile_001",
		Content:    jsonData,
		Attributes: make(map[string]string),
		Size:       int64(len(jsonData)),
		Timestamp:  time.Now(),
		LineageID:  "lineage_001",
	}

	// 使用记录转换器
	recordTransformer, _ := manager.GetProcessor("record_transformer")
	ctx := processor.ProcessContext{
		Context: context.Background(),
		Properties: map[string]string{
			"source.format":        "json",
			"target.format":        "json",
			"transformation.rules": "field_mapping",
		},
		Session: &processor.ProcessSession{},
		State:   make(map[string]interface{}),
	}

	result, err := recordTransformer.Process(ctx, flowFile)
	if err != nil {
		log.Printf("记录转换失败: %v", err)
	} else {
		fmt.Printf("记录转换成功，输出大小: %d bytes\n", result.Size)
		fmt.Printf("转换后属性: %v\n", result.Attributes)
	}
}

// exampleRouting 路由处理示例
func exampleRouting(manager *processor.ProcessorManager) {
	fmt.Println("\n--- 路由处理示例 ---")

	// 创建测试数据
	flowFile := &flowfile.FlowFile{
		UUID:    "test_flowfile_002",
		Content: []byte(`{"priority": "high", "category": "urgent", "content": "This is an urgent message"}`),
		Attributes: map[string]string{
			"priority":  "high",
			"category":  "urgent",
			"timestamp": time.Now().Format(time.RFC3339),
		},
		Size:      100,
		Timestamp: time.Now(),
		LineageID: "lineage_002",
	}

	// 使用高级路由处理器
	advancedRouter, _ := manager.GetProcessor("advanced_route_processor")
	ctx := processor.ProcessContext{
		Properties: map[string]string{
			"high_value.attribute":       "priority",
			"high_value.attribute_value": "high",
			"default.relationship":       "low_priority",
		},
		Session: &processor.ProcessSession{},
		State:   make(map[string]interface{}),
	}

	result, err := advancedRouter.Process(ctx, flowFile)
	if err != nil {
		log.Printf("路由处理失败: %v", err)
	} else {
		fmt.Printf("路由处理成功，目标队列: %s\n", result.Attributes["routing.queue"])
		fmt.Printf("路由属性: %v\n", result.Attributes)
	}
}

// exampleSemanticExtraction 语义提取示例
func exampleSemanticExtraction(manager *processor.ProcessorManager) {
	fmt.Println("\n--- 语义提取示例 ---")

	// 创建包含多种信息的测试数据
	testText := `
	用户信息：张三，身份证号：110101199001011234，手机号：13812345678
	邮箱：zhangsan@example.com，银行卡号：6222021234567890123
	交易时间：2024-01-15 14:30:25，交易金额：¥1234.56
	IP地址：192.168.1.100，网址：https://www.example.com
	`

	flowFile := &flowfile.FlowFile{
		UUID:       "test_flowfile_003",
		Content:    []byte(testText),
		Attributes: make(map[string]string),
		Size:       int64(len(testText)),
		Timestamp:  time.Now(),
		LineageID:  "lineage_003",
	}

	// 使用文本语义提取器
	textExtractor, _ := manager.GetProcessor("text_semantic_extractor")
	ctx := processor.ProcessContext{
		Properties: map[string]string{
			"extraction.patterns": "custom_patterns",
		},
		Session: &processor.ProcessSession{},
		State:   make(map[string]interface{}),
	}

	result, err := textExtractor.Process(ctx, flowFile)
	if err != nil {
		log.Printf("语义提取失败: %v", err)
	} else {
		fmt.Printf("语义提取成功，提取数量: %s\n", result.Attributes["semantic.extraction.count"])
		fmt.Printf("提取的语义信息:\n")
		for key, value := range result.Attributes {
			if key[:8] == "semantic" {
				fmt.Printf("  %s: %s\n", key, value)
			}
		}
	}
}

// exampleErrorHandling 错误处理示例
func exampleErrorHandling(manager *processor.ProcessorManager) {
	fmt.Println("\n--- 错误处理示例 ---")

	// 创建包含错误信息的测试数据
	flowFile := &flowfile.FlowFile{
		UUID:    "test_flowfile_004",
		Content: []byte(`{"data": "network_error", "message": "Connection timeout"}`),
		Attributes: map[string]string{
			"error.message": "Network connection timeout",
			"retry.count":   "2",
		},
		Size:      50,
		Timestamp: time.Now(),
		LineageID: "lineage_004",
	}

	// 使用错误处理处理器
	errorHandler, _ := manager.GetProcessor("error_handling_processor")
	ctx := processor.ProcessContext{
		Properties: map[string]string{
			"error.strategy": "retry",
		},
		Session: &processor.ProcessSession{},
		State:   make(map[string]interface{}),
	}

	result, err := errorHandler.Process(ctx, flowFile)
	if err != nil {
		log.Printf("错误处理失败: %v", err)
	} else {
		fmt.Printf("错误处理成功，错误类型: %s\n", result.Attributes["error.type"])
		fmt.Printf("错误信息: %s\n", result.Attributes["error.message"])
		fmt.Printf("处理时间: %s\n", result.Attributes["error.timestamp"])
	}
}

// exampleScriptProcessing 脚本处理示例
func exampleScriptProcessing(manager *processor.ProcessorManager) {
	fmt.Println("\n--- 脚本处理示例 ---")

	// 创建测试数据
	flowFile := &flowfile.FlowFile{
		UUID:       "test_flowfile_005",
		Content:    []byte(`{"name": "old_value", "status": "pending"}`),
		Attributes: make(map[string]string),
		Size:       30,
		Timestamp:  time.Now(),
		LineageID:  "lineage_005",
	}

	// 使用脚本处理器
	scriptProcessor, _ := manager.GetProcessor("script_processor")
	ctx := processor.ProcessContext{
		Properties: map[string]string{
			"script.language": "javascript",
			"script.content": `
				// 简单的JavaScript脚本示例
				flowFile.attributes["script.processed"] = "true";
				flowFile.attributes["script.timestamp"] = new Date().toISOString();
				flowFile.attributes["custom.field"] = "custom_value";
			`,
			"script.cache.enabled": "true",
			"script.timeout":       "10s",
		},
		Session: &processor.ProcessSession{},
		State:   make(map[string]interface{}),
	}

	result, err := scriptProcessor.Process(ctx, flowFile)
	if err != nil {
		log.Printf("脚本处理失败: %v", err)
	} else {
		fmt.Printf("脚本处理成功\n")
		fmt.Printf("脚本添加的属性:\n")
		for key, value := range result.Attributes {
			if key[:6] == "script" || key[:6] == "custom" {
				fmt.Printf("  %s: %s\n", key, value)
			}
		}
	}
}

// ExampleUserProfileEnrichment 用户画像丰富示例
func ExampleUserProfileEnrichment() {
	fmt.Println("\n=== 用户画像丰富示例 ===")

	// 创建用户画像丰富处理器
	enrichmentProcessor := &UserProfileEnrichmentProcessor{
		AbstractProcessor: &processor.AbstractProcessor{},
	}

	// 初始化处理器
	enrichmentProcessor.Initialize(processor.ProcessContext{})

	// 创建测试数据
	userData := map[string]interface{}{
		"user_id": "12345",
		"name":    "张三",
		"email":   "zhangsan@example.com",
	}

	jsonData, _ := json.Marshal(userData)

	flowFile := &flowfile.FlowFile{
		UUID:       "user_profile_001",
		Content:    jsonData,
		Attributes: map[string]string{"user.id": "12345"},
		Size:       int64(len(jsonData)),
		Timestamp:  time.Now(),
		LineageID:  "user_lineage_001",
	}

	// 处理数据
	ctx := processor.ProcessContext{
		Properties: map[string]string{
			"api.endpoint": "https://api.example.com/user/profile",
		},
		Session: &processor.ProcessSession{},
		State:   make(map[string]interface{}),
	}

	result, err := enrichmentProcessor.Process(ctx, flowFile)
	if err != nil {
		log.Printf("用户画像丰富失败: %v", err)
	} else {
		fmt.Printf("用户画像丰富成功\n")
		fmt.Printf("丰富后的属性:\n")
		for key, value := range result.Attributes {
			if key[:8] == "profile." {
				fmt.Printf("  %s: %s\n", key, value)
			}
		}
	}
}

// UserProfileEnrichmentProcessor 用户画像丰富处理器
type UserProfileEnrichmentProcessor struct {
	*processor.AbstractProcessor
}

// Initialize 初始化处理器
func (upep *UserProfileEnrichmentProcessor) Initialize(ctx processor.ProcessContext) error {
	if err := upep.AbstractProcessor.Initialize(ctx); err != nil {
		return err
	}

	// 添加关系
	upep.AddRelationship("enriched", "用户画像丰富成功")
	upep.AddRelationship("error", "用户画像丰富失败")

	// 添加属性描述符
	upep.AddPropertyDescriptor("api.endpoint", "用户画像API端点", true)
	upep.AddPropertyDescriptor("api.timeout", "API超时时间", false)

	return nil
}

// Process 处理数据
func (upep *UserProfileEnrichmentProcessor) Process(ctx processor.ProcessContext, flowFile *flowfile.FlowFile) (*flowfile.FlowFile, error) {
	upep.SetState(processor.StateRunning)
	defer upep.SetState(processor.StateReady)

	// 读取用户ID
	userID := flowFile.Attributes["user.id"]
	if userID == "" {
		return nil, fmt.Errorf("user.id attribute is required")
	}

	// 模拟调用外部API获取用户详细信息
	profile := upep.fetchUserProfile(userID)

	// 将用户信息写入FlowFile
	profileData, _ := json.Marshal(profile)
	flowFile.Content = profileData
	flowFile.Size = int64(len(profileData))

	// 添加用户画像属性
	flowFile.Attributes["profile.age"] = fmt.Sprintf("%d", profile.Age)
	flowFile.Attributes["profile.interests"] = strings.Join(profile.Interests, ",")
	flowFile.Attributes["profile.location"] = profile.Location
	flowFile.Attributes["profile.vip_level"] = fmt.Sprintf("%d", profile.VIPLevel)

	return flowFile, nil
}

// fetchUserProfile 获取用户画像
func (upep *UserProfileEnrichmentProcessor) fetchUserProfile(userID string) *UserProfile {
	// 模拟API调用
	// 在实际应用中，这里应该调用真实的API
	return &UserProfile{
		UserID:    userID,
		Name:      "张三",
		Age:       30,
		Location:  "北京",
		Interests: []string{"技术", "阅读", "旅行"},
		VIPLevel:  2,
	}
}

// UserProfile 用户画像
type UserProfile struct {
	UserID    string   `json:"user_id"`
	Name      string   `json:"name"`
	Age       int      `json:"age"`
	Location  string   `json:"location"`
	Interests []string `json:"interests"`
	VIPLevel  int      `json:"vip_level"`
}
