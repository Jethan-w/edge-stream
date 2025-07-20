package sink

import (
	"fmt"
	"strings"

	"github.com/edge-stream/internal/flowfile"
	"github.com/edge-stream/internal/sink/TargetAdapter"
)

// NewFileSystemOutputAdapter 创建文件系统输出适配器
func NewFileSystemOutputAdapter(config OutputTargetConfig) TargetAdapter.TargetAdapter {
	configMap := make(map[string]string)
	configMap["id"] = config.ID
	configMap["name"] = config.ID

	// 将配置转换为 map
	for key, value := range config.Properties {
		configMap[key] = value
	}

	return TargetAdapter.NewFileSystemOutputAdapter(configMap)
}

// NewDatabaseOutputAdapter 创建数据库输出适配器
func NewDatabaseOutputAdapter(config OutputTargetConfig) TargetAdapter.TargetAdapter {
	configMap := make(map[string]string)
	configMap["id"] = config.ID
	configMap["name"] = config.ID

	// 将配置转换为 map
	for key, value := range config.Properties {
		configMap[key] = value
	}

	return TargetAdapter.NewDatabaseOutputAdapter(configMap)
}

// NewMessageQueueOutputAdapter 创建消息队列输出适配器
func NewMessageQueueOutputAdapter(config OutputTargetConfig) TargetAdapter.TargetAdapter {
	// 创建消息队列适配器
	adapter := &MessageQueueOutputAdapter{
		AbstractTargetAdapter: TargetAdapter.NewAbstractTargetAdapter(
			config.ID,
			config.ID,
			"MessageQueue",
		),
		brokerURL:   config.ConnectionString,
		topicName:   config.Properties["topic_name"],
		queueName:   config.Properties["queue_name"],
		enableBatch: false,
		batchSize:   100,
	}

	// 应用配置
	adapter.ApplyConfig(config.Properties)

	return adapter
}

// NewSearchEngineOutputAdapter 创建搜索引擎输出适配器
func NewSearchEngineOutputAdapter(config OutputTargetConfig) TargetAdapter.TargetAdapter {
	// 创建搜索引擎适配器
	adapter := &SearchEngineOutputAdapter{
		AbstractTargetAdapter: TargetAdapter.NewAbstractTargetAdapter(
			config.ID,
			config.ID,
			"SearchEngine",
		),
		serverURL:    config.ConnectionString,
		indexName:    config.Properties["index_name"],
		documentType: config.Properties["document_type"],
	}

	// 应用配置
	adapter.ApplyConfig(config.Properties)

	return adapter
}

// NewRESTAPIOutputAdapter 创建 REST API 输出适配器
func NewRESTAPIOutputAdapter(config OutputTargetConfig) TargetAdapter.TargetAdapter {
	// 创建 REST API 适配器
	adapter := &RESTAPIOutputAdapter{
		AbstractTargetAdapter: TargetAdapter.NewAbstractTargetAdapter(
			config.ID,
			config.ID,
			"RESTAPI",
		),
		baseURL:     config.ConnectionString,
		endpoint:    config.Properties["endpoint"],
		method:      config.Properties["method"],
		contentType: config.Properties["content_type"],
		timeout:     30, // 默认30秒
	}

	// 应用配置
	adapter.ApplyConfig(config.Properties)

	return adapter
}

// MessageQueueOutputAdapter 消息队列输出适配器
type MessageQueueOutputAdapter struct {
	*TargetAdapter.AbstractTargetAdapter

	// 消息队列相关
	brokerURL   string
	topicName   string
	queueName   string
	enableBatch bool
	batchSize   int

	// 连接相关
	connection interface{} // 实际的连接对象
}

// ApplyConfig 应用配置
func (a *MessageQueueOutputAdapter) ApplyConfig(config map[string]string) {
	if broker, exists := config["broker_url"]; exists {
		a.brokerURL = broker
	}

	if topic, exists := config["topic_name"]; exists {
		a.topicName = topic
	}

	if queue, exists := config["queue_name"]; exists {
		a.queueName = queue
	}

	if enableBatch, exists := config["enable_batch"]; exists {
		a.enableBatch = strings.ToLower(enableBatch) == "true"
	}

	if batchSize, exists := config["batch_size"]; exists {
		// 解析批量大小
		if size, err := fmt.Sscanf(batchSize, "%d", &a.batchSize); err != nil {
			a.batchSize = 100 // 默认值
		}
	}
}

// doWrite 实现具体的消息队列写入逻辑
func (a *MessageQueueOutputAdapter) doWrite(flowFile interface{}) error {
	// 验证 FlowFile
	if err := a.ValidateFlowFile(flowFile); err != nil {
		return err
	}

	ff := flowFile.(*flowfile.FlowFile)

	// 获取内容
	content, err := ff.GetContent()
	if err != nil {
		return fmt.Errorf("failed to get flowfile content: %w", err)
	}

	// 发送消息到队列
	return a.sendMessage(content, ff.GetAttributes())
}

// sendMessage 发送消息
func (a *MessageQueueOutputAdapter) sendMessage(content []byte, attributes map[string]string) error {
	// 这里应该实现实际的消息发送逻辑
	// 为了简化，暂时只记录日志
	a.LogOutputActivity(fmt.Sprintf("Would send message to %s/%s: %d bytes", a.brokerURL, a.topicName, len(content)))
	return nil
}

// performHealthCheck 执行健康检查
func (a *MessageQueueOutputAdapter) performHealthCheck() bool {
	// 检查连接是否有效
	return a.brokerURL != "" && (a.topicName != "" || a.queueName != "")
}

// SearchEngineOutputAdapter 搜索引擎输出适配器
type SearchEngineOutputAdapter struct {
	*TargetAdapter.AbstractTargetAdapter

	// 搜索引擎相关
	serverURL    string
	indexName    string
	documentType string

	// 连接相关
	client interface{} // 实际的客户端对象
}

// ApplyConfig 应用配置
func (a *SearchEngineOutputAdapter) ApplyConfig(config map[string]string) {
	if server, exists := config["server_url"]; exists {
		a.serverURL = server
	}

	if index, exists := config["index_name"]; exists {
		a.indexName = index
	}

	if docType, exists := config["document_type"]; exists {
		a.documentType = docType
	}
}

// doWrite 实现具体的搜索引擎写入逻辑
func (a *SearchEngineOutputAdapter) doWrite(flowFile interface{}) error {
	// 验证 FlowFile
	if err := a.ValidateFlowFile(flowFile); err != nil {
		return err
	}

	ff := flowFile.(*flowfile.FlowFile)

	// 获取内容
	content, err := ff.GetContent()
	if err != nil {
		return fmt.Errorf("failed to get flowfile content: %w", err)
	}

	// 索引文档
	return a.indexDocument(content, ff.GetAttributes())
}

// indexDocument 索引文档
func (a *SearchEngineOutputAdapter) indexDocument(content []byte, attributes map[string]string) error {
	// 这里应该实现实际的文档索引逻辑
	// 为了简化，暂时只记录日志
	a.LogOutputActivity(fmt.Sprintf("Would index document to %s/%s: %d bytes", a.serverURL, a.indexName, len(content)))
	return nil
}

// performHealthCheck 执行健康检查
func (a *SearchEngineOutputAdapter) performHealthCheck() bool {
	// 检查配置是否有效
	return a.serverURL != "" && a.indexName != ""
}

// RESTAPIOutputAdapter REST API 输出适配器
type RESTAPIOutputAdapter struct {
	*TargetAdapter.AbstractTargetAdapter

	// REST API 相关
	baseURL     string
	endpoint    string
	method      string
	contentType string
	timeout     int

	// HTTP 客户端
	client interface{} // 实际的 HTTP 客户端对象
}

// ApplyConfig 应用配置
func (a *RESTAPIOutputAdapter) ApplyConfig(config map[string]string) {
	if baseURL, exists := config["base_url"]; exists {
		a.baseURL = baseURL
	}

	if endpoint, exists := config["endpoint"]; exists {
		a.endpoint = endpoint
	}

	if method, exists := config["method"]; exists {
		a.method = method
	} else {
		a.method = "POST" // 默认方法
	}

	if contentType, exists := config["content_type"]; exists {
		a.contentType = contentType
	} else {
		a.contentType = "application/json" // 默认内容类型
	}

	if timeout, exists := config["timeout"]; exists {
		if t, err := fmt.Sscanf(timeout, "%d", &a.timeout); err != nil {
			a.timeout = 30 // 默认30秒
		}
	}
}

// doWrite 实现具体的 REST API 写入逻辑
func (a *RESTAPIOutputAdapter) doWrite(flowFile interface{}) error {
	// 验证 FlowFile
	if err := a.ValidateFlowFile(flowFile); err != nil {
		return err
	}

	ff := flowFile.(*flowfile.FlowFile)

	// 获取内容
	content, err := ff.GetContent()
	if err != nil {
		return fmt.Errorf("failed to get flowfile content: %w", err)
	}

	// 发送 HTTP 请求
	return a.sendHTTPRequest(content, ff.GetAttributes())
}

// sendHTTPRequest 发送 HTTP 请求
func (a *RESTAPIOutputAdapter) sendHTTPRequest(content []byte, attributes map[string]string) error {
	// 这里应该实现实际的 HTTP 请求逻辑
	// 为了简化，暂时只记录日志
	url := a.baseURL
	if a.endpoint != "" {
		url += "/" + a.endpoint
	}

	a.LogOutputActivity(fmt.Sprintf("Would send %s request to %s: %d bytes", a.method, url, len(content)))
	return nil
}

// performHealthCheck 执行健康检查
func (a *RESTAPIOutputAdapter) performHealthCheck() bool {
	// 检查配置是否有效
	return a.baseURL != "" && a.method != ""
}
