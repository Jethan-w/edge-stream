package example

import (
	"fmt"
	"log"
	"time"

	"github.com/edge-stream/internal/flowfile"
	"github.com/edge-stream/internal/sink"
	"github.com/edge-stream/internal/sink/TargetAdapter"
)

// ExampleSinkUsage Sink 使用示例
func ExampleSinkUsage() {
	// 1. 创建 Sink 配置
	sinkConfig := &sink.SinkConfig{
		ID:          "example-sink",
		Name:        "Example Sink",
		Description: "示例 Sink 配置",
		OutputTargets: []sink.OutputTargetConfig{
			// 文件系统输出目标
			{
				ID:               "file-output",
				Type:             sink.OutputTargetTypeFileSystem,
				ConnectionString: "/tmp/output",
				Properties: map[string]string{
					"output.directory":   "/tmp/output",
					"file.permissions":   "0644",
					"create.directories": "true",
					"overwrite.existing": "false",
					"append.mode":        "false",
					"file.extension":     "txt",
					"date.format":        "2006-01-02",
					"compression":        "none",
				},
			},
			// 数据库输出目标
			{
				ID:               "db-output",
				Type:             sink.OutputTargetTypeDatabase,
				ConnectionString: "user:password@tcp(localhost:3306)/testdb",
				Properties: map[string]string{
					"driver_name":        "mysql",
					"dsn":                "user:password@tcp(localhost:3306)/testdb",
					"table_name":         "flowfiles",
					"batch_size":         "100",
					"enable_batch":       "false",
					"enable_transaction": "true",
					"max_open_conns":     "10",
					"max_idle_conns":     "5",
					"conn_max_lifetime":  "1h",
				},
			},
			// 消息队列输出目标
			{
				ID:               "mq-output",
				Type:             sink.OutputTargetTypeMessageQueue,
				ConnectionString: "localhost:9092",
				Properties: map[string]string{
					"broker_url":   "localhost:9092",
					"topic_name":   "flowfiles",
					"queue_name":   "",
					"enable_batch": "false",
					"batch_size":   "100",
				},
			},
		},
		SecurityConfig: &sink.SecurityConfiguration{
			Protocol:             sink.SecurityProtocolTLS,
			AuthenticationMethod: sink.AuthenticationMethodUsernamePassword,
			EncryptionAlgorithm:  sink.EncryptionAlgorithmAES256,
			Username:             "user",
			Password:             "password",
		},
		ReliabilityConfig: &sink.ReliabilityConfig{
			Strategy:           sink.ReliabilityStrategyAtLeastOnce,
			MaxRetries:         3,
			BaseRetryDelay:     time.Second,
			MaxRetryDelay:      30 * time.Second,
			RetryMultiplier:    2.0,
			EnableTransaction:  true,
			TransactionTimeout: 30 * time.Second,
		},
		ErrorHandlingConfig: &sink.ErrorHandlingConfig{
			DefaultStrategy:     sink.ErrorRoutingStrategyRetry,
			MaxRetryCount:       3,
			ErrorQueueName:      "error_queue",
			DeadLetterQueueName: "dead_letter_queue",
			LogErrors:           true,
			NotifyOnError:       false,
		},
		MaxConcurrentOutputs: 10,
		OutputTimeout:        30 * time.Second,
		EnableMetrics:        true,
	}

	// 2. 创建 Sink 实例
	sinkInstance := sink.NewSink(sinkConfig)

	// 3. 初始化 Sink
	if err := sinkInstance.Initialize(); err != nil {
		log.Fatalf("Failed to initialize sink: %v", err)
	}

	// 4. 启动 Sink
	if err := sinkInstance.Start(); err != nil {
		log.Fatalf("Failed to start sink: %v", err)
	}

	// 5. 创建示例 FlowFile
	flowFile := createExampleFlowFile()

	// 6. 写入数据到不同的目标
	targets := []string{"file-output", "db-output", "mq-output"}
	for _, targetID := range targets {
		if err := sinkInstance.Write(flowFile, targetID); err != nil {
			log.Printf("Failed to write to target %s: %v", targetID, err)
		} else {
			log.Printf("Successfully wrote to target: %s", targetID)
		}
	}

	// 7. 获取状态信息
	log.Printf("Sink state: %s", sinkInstance.GetState())

	// 8. 停止 Sink
	if err := sinkInstance.Stop(); err != nil {
		log.Printf("Failed to stop sink: %v", err)
	}
}

// ExampleCustomProtocol 自定义协议示例
func ExampleCustomProtocol() {
	// 1. 创建 Sink 实例
	sinkConfig := &sink.SinkConfig{
		ID:   "custom-protocol-sink",
		Name: "Custom Protocol Sink",
		OutputTargets: []sink.OutputTargetConfig{
			{
				ID:   "redis-output",
				Type: sink.OutputTargetTypeCustom,
				Properties: map[string]string{
					"protocol": "redis",
				},
			},
		},
	}

	sinkInstance := sink.NewSink(sinkConfig)

	// 2. 注册自定义协议
	sinkInstance.RegisterCustomProtocol("redis", &RedisOutputAdapter{
		AbstractTargetAdapter: TargetAdapter.NewAbstractTargetAdapter(
			"redis-adapter",
			"Redis Output Adapter",
			"Redis",
		),
		host:     "localhost",
		port:     6379,
		database: 0,
	})

	// 3. 初始化和启动
	if err := sinkInstance.Initialize(); err != nil {
		log.Fatalf("Failed to initialize sink: %v", err)
	}

	if err := sinkInstance.Start(); err != nil {
		log.Fatalf("Failed to start sink: %v", err)
	}

	// 4. 使用自定义协议
	flowFile := createExampleFlowFile()
	if err := sinkInstance.Write(flowFile, "redis-output"); err != nil {
		log.Printf("Failed to write to Redis: %v", err)
	}

	// 5. 停止
	sinkInstance.Stop()
}

// RedisOutputAdapter Redis 输出适配器示例
type RedisOutputAdapter struct {
	*TargetAdapter.AbstractTargetAdapter
	host     string
	port     int
	database int
}

// doWrite 实现 Redis 写入逻辑
func (r *RedisOutputAdapter) doWrite(flowFile interface{}) error {
	// 验证 FlowFile
	if err := r.ValidateFlowFile(flowFile); err != nil {
		return err
	}

	ff := flowFile.(*flowfile.FlowFile)

	// 获取内容
	content, err := ff.GetContent()
	if err != nil {
		return fmt.Errorf("failed to get flowfile content: %w", err)
	}

	// 获取 Redis 键
	key := ff.GetAttribute("redis.key")
	if key == "" {
		key = ff.GetAttribute("uuid") // 使用 UUID 作为默认键
	}

	// 获取数据类型
	dataType := ff.GetAttribute("redis.data.type")
	if dataType == "" {
		dataType = "string" // 默认类型
	}

	// 这里应该实现实际的 Redis 写入逻辑
	// 为了简化，只记录日志
	r.LogOutputActivity(fmt.Sprintf("Would write to Redis %s:%d, key: %s, type: %s, data: %d bytes",
		r.host, r.port, key, dataType, len(content)))

	return nil
}

// performHealthCheck 执行健康检查
func (r *RedisOutputAdapter) performHealthCheck() bool {
	// 这里应该实现实际的 Redis 连接检查
	// 为了简化，只检查配置
	return r.host != "" && r.port > 0
}

// createExampleFlowFile 创建示例 FlowFile
func createExampleFlowFile() *flowfile.FlowFile {
	// 创建 FlowFile
	ff := flowfile.NewFlowFile()

	// 设置属性
	ff.SetAttribute("filename", "example.txt")
	ff.SetAttribute("uuid", "12345678-1234-1234-1234-123456789abc")
	ff.SetAttribute("timestamp", time.Now().Format(time.RFC3339))
	ff.SetAttribute("source", "example")
	ff.SetAttribute("redis.key", "flowfile:example")
	ff.SetAttribute("redis.data.type", "string")

	// 设置内容
	content := []byte(`{
		"message": "Hello, World!",
		"timestamp": "2024-01-01T00:00:00Z",
		"level": "INFO",
		"source": "example"
	}`)
	ff.SetContent(content)

	return ff
}

// ExampleErrorHandling 错误处理示例
func ExampleErrorHandling() {
	// 创建配置了错误处理的 Sink
	sinkConfig := &sink.SinkConfig{
		ID:   "error-handling-sink",
		Name: "Error Handling Sink",
		OutputTargets: []sink.OutputTargetConfig{
			{
				ID:   "error-prone-output",
				Type: sink.OutputTargetTypeFileSystem,
				Properties: map[string]string{
					"output.directory": "/invalid/path", // 故意使用无效路径
				},
			},
		},
		ErrorHandlingConfig: &sink.ErrorHandlingConfig{
			DefaultStrategy:     sink.ErrorRoutingStrategyRetry,
			MaxRetryCount:       3,
			ErrorQueueName:      "error_queue",
			DeadLetterQueueName: "dead_letter_queue",
			LogErrors:           true,
			NotifyOnError:       true,
		},
	}

	sinkInstance := sink.NewSink(sinkConfig)
	sinkInstance.Initialize()
	sinkInstance.Start()

	// 尝试写入（会失败）
	flowFile := createExampleFlowFile()
	err := sinkInstance.Write(flowFile, "error-prone-output")
	if err != nil {
		log.Printf("Expected error occurred: %v", err)
	}

	// 获取错误统计
	errorHandler := sinkInstance.GetErrorHandler()
	if errorHandler != nil {
		stats := errorHandler.GetStats()
		log.Printf("Error handling stats: %+v", stats)
	}

	sinkInstance.Stop()
}

// ExampleRetryMechanism 重试机制示例
func ExampleRetryMechanism() {
	// 创建配置了重试机制的 Sink
	sinkConfig := &sink.SinkConfig{
		ID:   "retry-sink",
		Name: "Retry Mechanism Sink",
		OutputTargets: []sink.OutputTargetConfig{
			{
				ID:   "retry-output",
				Type: sink.OutputTargetTypeRESTAPI,
				Properties: map[string]string{
					"base_url":     "http://unreachable-server.com",
					"endpoint":     "api/data",
					"method":       "POST",
					"content_type": "application/json",
					"timeout":      "5",
				},
			},
		},
		ReliabilityConfig: &sink.ReliabilityConfig{
			Strategy:        sink.ReliabilityStrategyAtLeastOnce,
			MaxRetries:      5,
			BaseRetryDelay:  1 * time.Second,
			MaxRetryDelay:   30 * time.Second,
			RetryMultiplier: 2.0,
		},
	}

	sinkInstance := sink.NewSink(sinkConfig)
	sinkInstance.Initialize()
	sinkInstance.Start()

	// 尝试写入（会重试）
	flowFile := createExampleFlowFile()
	err := sinkInstance.Write(flowFile, "retry-output")
	if err != nil {
		log.Printf("Final error after retries: %v", err)
	}

	// 获取重试统计
	retryMgr := sinkInstance.GetRetryManager()
	if retryMgr != nil {
		stats := retryMgr.GetStats()
		log.Printf("Retry stats: %+v", stats)
	}

	sinkInstance.Stop()
}
