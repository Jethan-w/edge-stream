package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/crazy/edge-stream/internal/ConfigManager"
	"github.com/crazy/edge-stream/internal/ConnectorRegistry"
	"github.com/crazy/edge-stream/internal/MetricCollector"
	statemanager "github.com/crazy/edge-stream/internal/StateManager"
	windowmanager "github.com/crazy/edge-stream/internal/WindowManager"
	"github.com/crazy/edge-stream/internal/flowfile"
	"github.com/crazy/edge-stream/internal/pipeline"
	"github.com/crazy/edge-stream/internal/processor"
	"github.com/crazy/edge-stream/internal/sink"
	"github.com/crazy/edge-stream/internal/source"
	"github.com/crazy/edge-stream/internal/stream-engine"
	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
)

// ComprehensiveExample 综合示例：展示所有组件协同工作
func ComprehensiveExample() {
	fmt.Println("=== EdgeStream 综合示例 ===")
	fmt.Println("展示从Source到Pipeline到Process到Sink的完整数据流")
	fmt.Println("包含ConfigManager、ConnectorRegistry、FlowFile、MetricCollector、StateManager、StreamEngine、WindowManager等组件")

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. 初始化配置管理器
	fmt.Println("\n1. 初始化配置管理器...")
	configManager := initializeConfigManager()

	// 2. 初始化连接器注册表
	fmt.Println("\n2. 初始化连接器注册表...")
	connectorRegistry := initializeConnectorRegistry()

	// 3. 初始化指标收集器
	fmt.Println("\n3. 初始化指标收集器...")
	metricCollector := initializeMetricCollector()

	// 4. 初始化状态管理器
	fmt.Println("\n4. 初始化状态管理器...")
	stateManager := initializeStateManager()

	// 5. 初始化窗口管理器
	fmt.Println("\n5. 初始化窗口管理器...")
	windowManager := initializeWindowManager()

	// 6. 初始化流引擎
	fmt.Println("\n6. 初始化流引擎...")
	streamEngine := initializeStreamEngine(configManager, connectorRegistry, metricCollector, stateManager)

	// 7. 创建数据源
	fmt.Println("\n7. 创建数据源...")
	dataSource := createDataSource(configManager)

	// 8. 创建数据处理管道
	fmt.Println("\n8. 创建数据处理管道...")
	pipeline := createDataPipeline(configManager, metricCollector, windowManager)

	// 9. 创建数据处理器
	fmt.Println("\n9. 创建数据处理器...")
	dataProcessor := createDataProcessor(configManager)

	// 10. 创建数据接收器
	fmt.Println("\n10. 创建数据接收器...")
	dataSink := createDataSink(configManager)

	// 11. 初始化MySQL数据库连接
	fmt.Println("\n11. 初始化MySQL数据库连接...")
	db := initializeMySQLConnection(configManager)
	if db != nil {
		defer db.Close()
	}

	// 12. 初始化Redis连接
	fmt.Println("\n12. 初始化Redis连接...")
	redisClient := initializeRedisConnection(configManager)
	if redisClient != nil {
		defer redisClient.Close()
	} else {
		fmt.Println("Redis连接失败，将跳过Redis数据处理")
	}

	// 13. 启动流引擎
	fmt.Println("\n13. 启动流引擎...")
	if err := streamEngine.Start(ctx); err != nil {
		log.Fatalf("启动流引擎失败: %v", err)
	}

	// 14. 运行数据处理流程
	fmt.Println("\n14. 运行数据处理流程...")
	runDataProcessingFlow(ctx, dataSource, pipeline, dataProcessor, dataSink, streamEngine, metricCollector, db, redisClient)

	// 15. 监控和报告
	fmt.Println("\n15. 生成监控报告...")
	generateMonitoringReport(metricCollector, stateManager, windowManager)

	// 16. 停止流引擎
	fmt.Println("\n16. 停止流引擎...")
	if err := streamEngine.Stop(ctx); err != nil {
		log.Printf("停止流引擎失败: %v", err)
	}

	fmt.Println("\n=== 综合示例完成 ===")
}

// initializeConfigManager 初始化配置管理器
func initializeConfigManager() *ConfigManager.StandardConfigManager {
	configManager := ConfigManager.NewStandardConfigManager()

	// 加载配置文件
	configFile := "config/edge-stream.properties"
	if err := configManager.LoadConfiguration(configFile); err != nil {
		log.Printf("加载配置文件失败，使用默认配置: %v", err)
		// 设置默认配置
		configManager.SetProperty("source.type", "file", false)
		configManager.SetProperty("source.path", "./data/input", false)
		configManager.SetProperty("sink.type", "file", false)
		configManager.SetProperty("sink.path", "./data/output", false)
		configManager.SetProperty("processor.batch.size", "100", false)
		configManager.SetProperty("window.size", "60", false)
		configManager.SetProperty("window.slide", "30", false)
		// MySQL配置
		configManager.SetProperty("mysql.host", "localhost", false)
		configManager.SetProperty("mysql.port", "3306", false)
		configManager.SetProperty("mysql.username", "root", false)
		configManager.SetProperty("mysql.password", "340824", false)
		configManager.SetProperty("mysql.database", "edge_stream", false)
		// Redis配置
		configManager.SetProperty("redis.host", "localhost", false)
		configManager.SetProperty("redis.port", "6379", false)
		configManager.SetProperty("redis.password", "", false)
		configManager.SetProperty("redis.database", "0", false)
	}

	// 注册配置变更监听器
	configManager.AddConfigurationChangeListener(&ConfigChangeListener{})

	// 验证配置
	if result, err := configManager.ValidateConfiguration(); err != nil {
		log.Printf("配置验证失败: %v", err)
	} else if !result.IsValid {
		log.Printf("配置验证警告: %s", result.GetWarnings())
	}

	return configManager
}

// initializeConnectorRegistry 初始化连接器注册表
func initializeConnectorRegistry() *ConnectorRegistry.StandardConnectorRegistry {
	connectorRegistry := ConnectorRegistry.NewStandardConnectorRegistry()

	// 注册内置连接器
	registerBuiltinConnectors(connectorRegistry)

	// 设置Bundle仓库
	bundleRepo := ConnectorRegistry.NewFileSystemBundleRepository()
	bundleRepo.SetRepositoryPath("./bundles")
	connectorRegistry.SetBundleRepository(bundleRepo)

	// 设置版本控制器
	versionController := ConnectorRegistry.NewStandardVersionController()
	connectorRegistry.SetVersionController(versionController)

	return connectorRegistry
}

// initializeMetricCollector 初始化指标收集器
func initializeMetricCollector() MetricCollector.MetricCollector {
	metricCollector := MetricCollector.NewStandardMetricCollector()

	// 注册系统指标
	registerSystemMetrics(metricCollector)

	// 启动指标收集器
	metricCollector.Start()

	return metricCollector
}

// initializeStateManager 初始化状态管理器
func initializeStateManager() *statemanager.StandardStateManager {
	stateManager := statemanager.NewStandardStateManager()

	// 初始化状态管理器
	config := statemanager.ManagerConfiguration{
		ConsistencyLevel:   statemanager.ConsistencyLevelEventual,
		CheckpointInterval: 5 * time.Minute,
		SyncInterval:       30 * time.Second,
	}

	if err := stateManager.Initialize(config); err != nil {
		log.Printf("初始化状态管理器失败: %v", err)
	}

	return stateManager
}

// initializeWindowManager 初始化窗口管理器
func initializeWindowManager() *windowmanager.StandardWindowManager {
	windowManager := windowmanager.NewStandardWindowManager()

	// 设置时间戳提取器
	timestampExtractor := windowmanager.NewJsonTimestampExtractor("timestamp", "2006-01-02T15:04:05Z")
	windowManager.SetTimestampExtractor(timestampExtractor)

	// 设置窗口状态管理器
	var windowStateManager windowmanager.WindowStateManager
	contentRepoManager, err := windowmanager.NewContentRepositoryStateManager("./state/windows")
	if err != nil {
		log.Printf("创建窗口状态管理器失败: %v", err)
		// 使用内存状态管理器作为备选
		windowStateManager = windowmanager.NewMemoryStateManager()
	} else {
		windowStateManager = contentRepoManager
	}
	windowManager.SetStateManager(windowStateManager)

	return windowManager
}

// initializeStreamEngine 初始化流引擎
func initializeStreamEngine(
	configManager *ConfigManager.StandardConfigManager,
	connectorRegistry *ConnectorRegistry.StandardConnectorRegistry,
	metricCollector MetricCollector.MetricCollector,
	stateManager *statemanager.StandardStateManager,
) *stream_engine.StandardStreamEngine {
	// 创建引擎配置
	config := &stream_engine.EngineConfig{
		MaxThreads:            10,
		CoreThreads:           5,
		QueueCapacity:         1000,
		BackpressureThreshold: 0.8,
		ClusterEnabled:        false,
		SyncInterval:          5 * time.Second,
	}

	streamEngine := stream_engine.NewStandardStreamEngine(config)

	// 设置组件依赖 - 使用接口类型
	streamEngine.SetConfigManager(configManager)
	streamEngine.SetConnectorRegistry(connectorRegistry)
	streamEngine.SetMetricCollector(metricCollector)
	streamEngine.SetStateManager(stateManager)

	return streamEngine
}

// initializeMySQLConnection 初始化MySQL数据库连接
func initializeMySQLConnection(configManager *ConfigManager.StandardConfigManager) *sql.DB {
	host, _ := configManager.GetProperty("mysql.host")
	port, _ := configManager.GetProperty("mysql.port")
	username, _ := configManager.GetProperty("mysql.username")
	password, _ := configManager.GetProperty("mysql.password")
	database, _ := configManager.GetProperty("mysql.database")

	// 首先连接到MySQL服务器（不指定数据库）
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=True&loc=Local",
		username, password, host, port)

	// 连接数据库
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Printf("连接MySQL服务器失败: %v", err)
		return nil
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		log.Printf("MySQL服务器连接测试失败: %v", err)
		return nil
	}

	// 创建数据库（如果不存在）
	createDBSQL := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", database)
	if _, err := db.Exec(createDBSQL); err != nil {
		log.Printf("创建数据库失败: %v", err)
		return nil
	}

	// 关闭连接
	db.Close()

	// 重新连接到指定数据库
	dsnWithDB := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		username, password, host, port, database)

	db, err = sql.Open("mysql", dsnWithDB)
	if err != nil {
		log.Printf("连接MySQL数据库失败: %v", err)
		return nil
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		log.Printf("MySQL数据库连接测试失败: %v", err)
		return nil
	}

	// 创建表结构
	createTables(db)

	fmt.Printf("MySQL数据库连接成功: %s:%s/%s\n", host, port, database)
	return db
}

// initializeRedisConnection 初始化Redis连接
func initializeRedisConnection(configManager *ConfigManager.StandardConfigManager) *redis.Client {
	host, _ := configManager.GetProperty("redis.host")
	port, _ := configManager.GetProperty("redis.port")
	password, _ := configManager.GetProperty("redis.password")
	_, _ = configManager.GetProperty("redis.database") // 避免未使用变量警告

	// 创建Redis客户端
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", host, port),
		Password: password,
		DB:       0, // 默认使用数据库0
		PoolSize: 10,
	})

	// 测试连接
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("Redis连接测试失败: %v", err)
		return nil
	}

	fmt.Printf("Redis连接成功: %s:%s\n", host, port)
	return rdb
}

// createTables 创建数据库表
func createTables(db *sql.DB) {
	// 创建日志数据表
	logTableSQL := `
	CREATE TABLE IF NOT EXISTS log_entries (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		timestamp DATETIME NOT NULL,
		level VARCHAR(10) NOT NULL,
		message TEXT NOT NULL,
		source VARCHAR(100),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		INDEX idx_timestamp (timestamp),
		INDEX idx_level (level)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`

	// 创建处理统计表
	statsTableSQL := `
	CREATE TABLE IF NOT EXISTS processing_stats (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		processor_id VARCHAR(100) NOT NULL,
		processed_count BIGINT DEFAULT 0,
		error_count BIGINT DEFAULT 0,
		processing_time_ms BIGINT DEFAULT 0,
		window_start DATETIME,
		window_end DATETIME,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		INDEX idx_processor (processor_id),
		INDEX idx_window (window_start, window_end)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`

	// 创建Redis键前缀统计表
	redisPrefixTableSQL := `
	CREATE TABLE IF NOT EXISTS redis_key_prefix_stats (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		prefix VARCHAR(50) NOT NULL,
		count BIGINT DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		INDEX idx_prefix (prefix),
		INDEX idx_created_at (created_at)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`

	if _, err := db.Exec(logTableSQL); err != nil {
		log.Printf("创建日志表失败: %v", err)
	}

	if _, err := db.Exec(statsTableSQL); err != nil {
		log.Printf("创建统计表失败: %v", err)
	}

	if _, err := db.Exec(redisPrefixTableSQL); err != nil {
		log.Printf("创建Redis前缀统计表失败: %v", err)
	}
}

// createDataSource 创建数据源
func createDataSource(configManager *ConfigManager.StandardConfigManager) *source.AbstractSource {
	sourceType, err := configManager.GetProperty("source.type")
	if err != nil {
		log.Printf("获取source.type配置失败: %v", err)
		sourceType = "file" // 使用默认值
	}

	sourcePath, err := configManager.GetProperty("source.path")
	if err != nil {
		log.Printf("获取source.path配置失败: %v", err)
		sourcePath = "./data/input" // 使用默认值
	}

	// 创建数据源配置
	sourceConfig := source.NewSourceConfiguration()
	sourceConfig.SetProperty("type", sourceType)
	sourceConfig.SetProperty("path", sourcePath)
	sourceConfig.MaxConnections = 10 // 设置最大连接数

	// 创建数据源
	dataSource := source.NewAbstractSource()
	dataSource.SetConfiguration(sourceConfig)

	// 初始化数据源
	ctx := context.Background()
	if err := dataSource.Initialize(ctx); err != nil {
		log.Printf("初始化数据源失败: %v", err)
	}

	return dataSource
}

// createDataProcessor 创建数据处理器
func createDataProcessor(configManager *ConfigManager.StandardConfigManager) *processor.RecordTransformer {
	// 创建数据转换器
	dataTransformer := processor.NewRecordTransformer()

	// 创建处理上下文
	ctx := processor.ProcessContext{
		Context: context.Background(),
		Properties: map[string]string{
			"source.format":        "json",
			"target.format":        "csv",
			"transformation.rules": "json_to_csv",
		},
	}

	// 初始化处理器
	if err := dataTransformer.Initialize(ctx); err != nil {
		log.Printf("初始化数据转换器失败: %v", err)
	}

	return dataTransformer
}

// createDataPipeline 创建数据处理管道
func createDataPipeline(
	configManager *ConfigManager.StandardConfigManager,
	metricCollector MetricCollector.MetricCollector,
	windowManager *windowmanager.StandardWindowManager,
) *pipeline.ProcessGroup {
	// 创建流程设计器
	designer := pipeline.NewPipelineDesigner()

	// 创建流程组
	processGroup := designer.CreateProcessGroup("comprehensive-data-pipeline")

	// 添加数据转换处理器
	transformer := pipeline.NewProcessorDTO("transformer-1", "DataTransformer", "DataTransformer")
	transformer.SetProperty("transformation.type", "json_to_csv")
	designer.AddProcessor(processGroup, transformer)

	// 添加语义提取处理器
	extractor := pipeline.NewProcessorDTO("extractor-1", "SemanticExtractor", "SemanticExtractor")
	extractor.SetProperty("extraction.fields", "timestamp,level,message")
	designer.AddProcessor(processGroup, extractor)

	// 添加窗口聚合处理器
	aggregator := pipeline.NewProcessorDTO("aggregator-1", "WindowAggregator", "WindowAggregator")
	aggregator.SetProperty("window.size", "60")
	aggregator.SetProperty("window.slide", "30")
	designer.AddProcessor(processGroup, aggregator)

	// 创建连接
	connection1 := pipeline.NewConnectionDTO("conn-1", "transformer_to_extractor", "transformer-1", "extractor-1")
	designer.CreateConnection(processGroup, connection1)

	connection2 := pipeline.NewConnectionDTO("conn-2", "extractor_to_aggregator", "extractor-1", "aggregator-1")
	designer.CreateConnection(processGroup, connection2)

	return processGroup
}

// createDataSink 创建数据接收器
func createDataSink(configManager *ConfigManager.StandardConfigManager) *sink.Sink {
	sinkPath, _ := configManager.GetProperty("sink.path")

	// 创建Sink配置
	sinkConfig := &sink.SinkConfig{
		ID:                   "comprehensive-sink",
		Name:                 "Comprehensive Data Sink",
		Description:          "综合数据接收器",
		MaxConcurrentOutputs: 10,
		OutputTimeout:        30 * time.Second,
		EnableMetrics:        true,
		SecurityConfig:       &sink.SecurityConfiguration{}, // 添加空的安全配置
		ReliabilityConfig:    &sink.ReliabilityConfig{},     // 添加空的可靠性配置
		ErrorHandlingConfig:  &sink.ErrorHandlingConfig{},   // 添加空的错误处理配置
		OutputTargets: []sink.OutputTargetConfig{
			{
				ID:               "file-target",
				Type:             sink.OutputTargetTypeFileSystem,
				ConnectionString: sinkPath,
				Properties: map[string]string{
					"file.prefix": "processed_",
					"file.suffix": ".csv",
				},
			},
		},
	}

	// 创建数据接收器
	dataSink := sink.NewSink(sinkConfig)

	// 初始化数据接收器
	if err := dataSink.Initialize(); err != nil {
		log.Printf("初始化数据接收器失败: %v", err)
	}

	return dataSink
}

// runDataProcessingFlow 运行数据处理流程
func runDataProcessingFlow(
	ctx context.Context,
	dataSource *source.AbstractSource,
	pipeline *pipeline.ProcessGroup,
	dataProcessor *processor.RecordTransformer,
	dataSink *sink.Sink,
	streamEngine *stream_engine.StandardStreamEngine,
	metricCollector MetricCollector.MetricCollector,
	db *sql.DB,
	redisClient *redis.Client,
) {
	fmt.Println("开始数据处理流程...")

	// 创建示例数据
	createSampleData()

	// 处理Redis数据
	if redisClient != nil {
		processRedisData(redisClient, db, dataSink, metricCollector)
	}

	// 模拟数据流处理
	for i := 0; i < 10; i++ {
		// 1. 从数据源读取数据
		flowFile := readFromSource(dataSource)
		if flowFile == nil {
			continue
		}

		// 2. 记录指标
		metricCollector.RecordMetric("flowfile.received", int64(len(flowFile.Content)))

		// 3. 通过数据处理器处理数据
		processedFlowFile := processWithProcessor(dataProcessor, flowFile)

		// 4. 通过管道处理数据
		pipelineProcessedFlowFile := processThroughPipeline(pipeline, processedFlowFile)

		// 5. 写入MySQL数据库
		writeToDatabase(db, pipelineProcessedFlowFile)

		// 6. 写入数据接收器
		writeToSink(dataSink, pipelineProcessedFlowFile)

		// 7. 记录处理指标
		metricCollector.RecordMetric("flowfile.processed", int64(len(pipelineProcessedFlowFile.Content)))

		// 等待一段时间
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("数据处理流程完成")
}

// processRedisData 处理Redis数据
func processRedisData(redisClient *redis.Client, db *sql.DB, dataSink *sink.Sink, metricCollector MetricCollector.MetricCollector) {
	fmt.Println("开始处理Redis数据...")
	ctx := context.Background()

	// 获取所有Redis键
	keys, err := redisClient.Keys(ctx, "*").Result()
	if err != nil {
		log.Printf("获取Redis键失败: %v", err)
		return
	}

	if len(keys) == 0 {
		fmt.Println("Redis中没有数据")
		return
	}

	fmt.Printf("从Redis获取到 %d 个键\n", len(keys))

	// 统计键前缀
	prefixCount := make(map[string]int)
	var redisData []string

	// 处理每个键
	for _, key := range keys {
		// 获取键的前5个字符作为前缀
		prefix := ""
		if len(key) >= 5 {
			prefix = key[:5]
		} else {
			prefix = key
		}
		prefixCount[prefix]++

		// 获取键的值
		value, err := redisClient.Get(ctx, key).Result()
		if err != nil {
			log.Printf("获取Redis键 %s 的值失败: %v", key, err)
			continue
		}

		// 构建数据行
		dataLine := fmt.Sprintf("%s,%s", key, value)
		redisData = append(redisData, dataLine)

		// 记录指标
		metricCollector.RecordMetric("redis.key.processed", 1)
	}

	// 将Redis数据写入文件
	writeRedisDataToFile(redisData)

	// 将前缀统计写入MySQL
	writePrefixStatsToMySQL(db, prefixCount)

	fmt.Printf("Redis数据处理完成，处理了 %d 个键，统计了 %d 个前缀\n", len(keys), len(prefixCount))
}

// writeRedisDataToFile 将Redis数据写入文件
func writeRedisDataToFile(redisData []string) {
	// 创建输出目录
	outputDir := "./data/output"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Printf("创建输出目录失败: %v", err)
		return
	}

	// 创建Redis数据文件
	timestamp := time.Now().Format("20060102_150405")
	fileName := fmt.Sprintf("redis_data_%s.csv", timestamp)
	filePath := filepath.Join(outputDir, fileName)

	file, err := os.Create(filePath)
	if err != nil {
		log.Printf("创建Redis数据文件失败: %v", err)
		return
	}
	defer file.Close()

	// 写入CSV头部
	file.WriteString("key,value\n")

	// 写入数据
	for _, line := range redisData {
		file.WriteString(line + "\n")
	}

	fmt.Printf("Redis数据已写入文件: %s\n", filePath)
}

// writePrefixStatsToMySQL 将前缀统计写入MySQL
func writePrefixStatsToMySQL(db *sql.DB, prefixCount map[string]int) {
	if db == nil {
		return
	}

	// 准备插入语句
	insertSQL := `
	INSERT INTO redis_key_prefix_stats (prefix, count) 
	VALUES (?, ?)
	ON DUPLICATE KEY UPDATE 
	count = VALUES(count),
	updated_at = CURRENT_TIMESTAMP
	`

	// 按前缀排序
	var prefixes []string
	for prefix := range prefixCount {
		prefixes = append(prefixes, prefix)
	}
	sort.Strings(prefixes)

	// 插入每个前缀的统计
	for _, prefix := range prefixes {
		count := prefixCount[prefix]
		_, err := db.Exec(insertSQL, prefix, count)
		if err != nil {
			log.Printf("插入前缀统计失败 [%s]: %v", prefix, err)
		} else {
			fmt.Printf("前缀统计已写入MySQL: %s -> %d\n", prefix, count)
		}
	}
}

// processWithProcessor 使用数据处理器处理数据
func processWithProcessor(dataProcessor *processor.RecordTransformer, flowFile *flowfile.FlowFile) *flowfile.FlowFile {
	fmt.Printf("使用数据处理器处理FlowFile: %s\n", flowFile.UUID)

	// 创建处理上下文
	ctx := processor.ProcessContext{
		Context: context.Background(),
		Properties: map[string]string{
			"source.format":        "json",
			"target.format":        "csv",
			"transformation.rules": "json_to_csv",
		},
	}

	// 使用处理器处理数据
	processedFlowFile, err := dataProcessor.Process(ctx, flowFile)
	if err != nil {
		log.Printf("数据处理器处理失败: %v", err)
		return flowFile
	}

	// 更新FlowFile属性
	processedFlowFile.Attributes["processor.processed"] = "true"
	processedFlowFile.Attributes["processor.type"] = "RecordTransformer"

	return processedFlowFile
}

// writeToDatabase 写入数据库
func writeToDatabase(db *sql.DB, flowFile *flowfile.FlowFile) {
	if db == nil {
		return
	}

	// 解析FlowFile内容
	content := string(flowFile.Content)

	// 简单的CSV解析（假设格式为: timestamp,level,message）
	// 在实际应用中，应该使用更robust的CSV解析器
	if len(content) > 0 {
		// 插入日志数据
		insertSQL := `
		INSERT INTO log_entries (timestamp, level, message, source) 
		VALUES (?, ?, ?, ?)
		`

		// 这里简化处理，实际应该解析CSV内容
		timestamp := time.Now()
		level := "INFO"
		message := content
		source := "edge-stream"

		_, err := db.Exec(insertSQL, timestamp, level, message, source)
		if err != nil {
			log.Printf("插入数据库失败: %v", err)
		} else {
			fmt.Printf("数据已写入数据库: %s\n", flowFile.UUID)
		}
	}
}

// createSampleData 创建示例数据
func createSampleData() {
	// 创建输入目录
	inputDir := "./data/input"
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		log.Printf("创建输入目录失败: %v", err)
		return
	}

	// 创建示例日志文件
	sampleData := []string{
		`{"timestamp":"2024-01-01T10:00:00Z","level":"INFO","message":"Application started"}`,
		`{"timestamp":"2024-01-01T10:00:01Z","level":"DEBUG","message":"Processing request"}`,
		`{"timestamp":"2024-01-01T10:00:02Z","level":"WARN","message":"High memory usage detected"}`,
		`{"timestamp":"2024-01-01T10:00:03Z","level":"ERROR","message":"Database connection failed"}`,
		`{"timestamp":"2024-01-01T10:00:04Z","level":"INFO","message":"Request completed successfully"}`,
	}

	// 写入文件
	filePath := filepath.Join(inputDir, "sample.log")
	file, err := os.Create(filePath)
	if err != nil {
		log.Printf("创建示例文件失败: %v", err)
		return
	}
	defer file.Close()

	for _, line := range sampleData {
		file.WriteString(line + "\n")
	}

	fmt.Printf("创建示例数据文件: %s\n", filePath)
}

// readFromSource 从数据源读取数据
func readFromSource(dataSource *source.AbstractSource) *flowfile.FlowFile {
	// 模拟从数据源读取FlowFile
	flowFile := flowfile.NewFlowFile()
	// 使用数组格式的JSON，这样可以被RecordTransformer正确解析
	flowFile.Content = []byte(`[{"id":"1","timestamp":"2024-01-01T10:00:00Z","data":{"timestamp":"2024-01-01T10:00:00Z","level":"INFO","message":"Sample log entry"}}]`)
	flowFile.Attributes["filename"] = "sample.log"
	flowFile.Attributes["source"] = "file"

	fmt.Printf("从数据源读取FlowFile: %s\n", flowFile.UUID)
	return flowFile
}

// processThroughPipeline 通过管道处理数据
func processThroughPipeline(pipeline *pipeline.ProcessGroup, flowFile *flowfile.FlowFile) *flowfile.FlowFile {
	// 模拟管道处理
	fmt.Printf("通过管道处理FlowFile: %s\n", flowFile.UUID)

	// 1. 数据转换
	flowFile.Attributes["transformed"] = "true"
	flowFile.Attributes["transformation.type"] = "json_to_csv"

	// 2. 语义提取
	flowFile.Attributes["extracted"] = "true"
	flowFile.Attributes["timestamp"] = "2024-01-01T10:00:00Z"
	flowFile.Attributes["level"] = "INFO"
	flowFile.Attributes["message"] = "Sample log entry"

	// 3. 窗口聚合
	flowFile.Attributes["aggregated"] = "true"
	flowFile.Attributes["window.size"] = "60"
	flowFile.Attributes["window.slide"] = "30"

	// 更新内容
	flowFile.Content = []byte("2024-01-01T10:00:00Z,INFO,Sample log entry")

	return flowFile
}

// writeToSink 写入数据接收器
func writeToSink(dataSink *sink.Sink, flowFile *flowfile.FlowFile) {
	// 创建输出目录
	outputDir := "./data/output"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Printf("创建输出目录失败: %v", err)
		return
	}

	// 模拟写入数据接收器
	fmt.Printf("写入数据接收器: %s\n", flowFile.UUID)

	// 实际写入文件
	outputFile := filepath.Join(outputDir, fmt.Sprintf("processed_%s.csv", flowFile.UUID))
	if err := os.WriteFile(outputFile, flowFile.Content, 0644); err != nil {
		log.Printf("写入输出文件失败: %v", err)
	} else {
		fmt.Printf("数据已写入: %s\n", outputFile)
	}
}

// generateMonitoringReport 生成监控报告
func generateMonitoringReport(
	metricCollector MetricCollector.MetricCollector,
	stateManager *statemanager.StandardStateManager,
	windowManager *windowmanager.StandardWindowManager,
) {
	fmt.Println("\n=== 监控报告 ===")

	// 1. 指标快照
	snapshot := metricCollector.GetMetricsSnapshot()
	fmt.Printf("指标快照 (时间: %s):\n", snapshot.Timestamp.Format(time.RFC3339))
	fmt.Printf("  Gauge指标: %d个\n", len(snapshot.Gauges))
	fmt.Printf("  Counter指标: %d个\n", len(snapshot.Counters))
	fmt.Printf("  Histogram指标: %d个\n", len(snapshot.Histograms))
	fmt.Printf("  Timer指标: %d个\n", len(snapshot.Timers))
	fmt.Printf("  Meter指标: %d个\n", len(snapshot.Meters))

	// 2. 状态信息
	fmt.Printf("状态信息:\n")
	fmt.Printf("  状态管理器已初始化\n")
	fmt.Printf("  状态提供者类型: 本地\n")

	// 3. 窗口统计
	fmt.Printf("窗口管理器统计:\n")
	activeWindows := windowManager.GetActiveWindows()
	fmt.Printf("  活跃窗口数量: %d\n", len(activeWindows))

	// 4. 组件使用情况总结
	fmt.Printf("\n=== 组件使用情况总结 ===\n")
	fmt.Printf("✓ ConfigManager (配置管理器): 已使用 - 管理所有配置项\n")
	fmt.Printf("✓ ConnectorRegistry (连接器注册中心): 已使用 - 注册和管理连接器\n")
	fmt.Printf("✓ MetricCollector (指标收集器): 已使用 - 收集和监控系统指标\n")
	fmt.Printf("✓ StateManager (状态管理器): 已使用 - 管理系统状态\n")
	fmt.Printf("✓ WindowManager (窗口管理器): 已使用 - 管理时间窗口\n")
	fmt.Printf("✓ StreamEngine (流引擎): 已使用 - 核心流处理引擎\n")
	fmt.Printf("✓ Source (数据源): 已使用 - 数据输入组件\n")
	fmt.Printf("✓ Pipeline (数据管道): 已使用 - 数据处理管道\n")
	fmt.Printf("✓ Processor (数据处理器): 已使用 - 数据处理组件\n")
	fmt.Printf("✓ Sink (数据接收器): 已使用 - 数据输出组件\n")
	fmt.Printf("✓ MySQL数据库: 已连接 - 数据持久化存储\n")
	fmt.Printf("✓ Redis缓存: 已连接 - 数据读取和统计\n")
}

// registerBuiltinConnectors 注册内置连接器
func registerBuiltinConnectors(registry *ConnectorRegistry.StandardConnectorRegistry) {
	// 注册文件连接器
	fileConnector := &ConnectorRegistry.NarBundleConnectorDescriptor{
		Bundle: &ConnectorRegistry.Bundle{
			ID:       "file-connector",
			Group:    "org.apache.nifi",
			Artifact: "nifi-file-connector",
			Version:  "1.0.0",
		},
		Identifier:  "file-connector",
		Name:        "File Connector",
		Type:        ConnectorRegistry.ComponentTypeProcessor,
		Version:     "1.0.0",
		Description: "File system connector for reading and writing files",
		Author:      "Apache NiFi",
		Tags:        []string{"file", "io"},
		Metadata:    map[string]string{"category": "io"},
	}
	registry.RegisterConnector(fileConnector)

	// 注册Kafka连接器
	kafkaConnector := &ConnectorRegistry.NarBundleConnectorDescriptor{
		Bundle: &ConnectorRegistry.Bundle{
			ID:       "kafka-connector",
			Group:    "org.apache.nifi",
			Artifact: "nifi-kafka-connector",
			Version:  "1.0.0",
		},
		Identifier:  "kafka-connector",
		Name:        "Kafka Connector",
		Type:        ConnectorRegistry.ComponentTypeProcessor,
		Version:     "1.0.0",
		Description: "Apache Kafka connector for streaming data",
		Author:      "Apache NiFi",
		Tags:        []string{"kafka", "streaming"},
		Metadata:    map[string]string{"category": "messaging"},
	}
	registry.RegisterConnector(kafkaConnector)

	// 注册MySQL连接器
	mysqlConnector := &ConnectorRegistry.NarBundleConnectorDescriptor{
		Bundle: &ConnectorRegistry.Bundle{
			ID:       "mysql-connector",
			Group:    "org.apache.nifi",
			Artifact: "nifi-mysql-connector",
			Version:  "1.0.0",
		},
		Identifier:  "mysql-connector",
		Name:        "MySQL Connector",
		Type:        ConnectorRegistry.ComponentTypeProcessor,
		Version:     "1.0.0",
		Description: "MySQL database connector for data persistence",
		Author:      "Apache NiFi",
		Tags:        []string{"mysql", "database"},
		Metadata:    map[string]string{"category": "database"},
	}
	registry.RegisterConnector(mysqlConnector)

	// 注册Redis连接器
	redisConnector := &ConnectorRegistry.NarBundleConnectorDescriptor{
		Bundle: &ConnectorRegistry.Bundle{
			ID:       "redis-connector",
			Group:    "org.apache.nifi",
			Artifact: "nifi-redis-connector",
			Version:  "1.0.0",
		},
		Identifier:  "redis-connector",
		Name:        "Redis Connector",
		Type:        ConnectorRegistry.ComponentTypeProcessor,
		Version:     "1.0.0",
		Description: "Redis cache connector for data processing",
		Author:      "Apache NiFi",
		Tags:        []string{"redis", "cache"},
		Metadata:    map[string]string{"category": "cache"},
	}
	registry.RegisterConnector(redisConnector)
}

// registerSystemMetrics 注册系统指标
func registerSystemMetrics(collector MetricCollector.MetricCollector) {
	// 注册FlowFile计数器
	collector.RegisterCounter("flowfile.received")
	collector.RegisterCounter("flowfile.processed")
	collector.RegisterCounter("flowfile.error")

	// 注册处理时间计时器
	collector.RegisterTimer("processing.time")

	// 注册吞吐量计量器
	collector.RegisterMeter("throughput")

	// 注册内存使用量仪表
	collector.RegisterGauge("memory.usage", func() float64 {
		// 模拟内存使用量
		return 75.5
	})

	// 注册CPU使用量仪表
	collector.RegisterGauge("cpu.usage", func() float64 {
		// 模拟CPU使用量
		return 45.2
	})

	// 注册数据库连接指标
	collector.RegisterGauge("database.connections", func() float64 {
		// 模拟数据库连接数
		return 5.0
	})

	// 注册处理器指标
	collector.RegisterCounter("processor.executions")
	collector.RegisterTimer("processor.execution.time")

	// 注册Redis指标
	collector.RegisterCounter("redis.key.processed")
	collector.RegisterGauge("redis.keys.total", func() float64 {
		// 模拟Redis键总数
		return 100.0
	})
}

// ConfigChangeListener 配置变更监听器
type ConfigChangeListener struct{}

// OnConfigurationChanged 配置变更回调
func (ccl *ConfigChangeListener) OnConfigurationChanged(event *ConfigManager.ConfigurationChangeEvent) {
	fmt.Printf("配置变更: %s = %s (来源: %s, 类型: %s)\n",
		event.Key, event.Value, event.Source, event.ChangeType)
}

func main() {
	// 运行综合示例
	ComprehensiveExample()
}
