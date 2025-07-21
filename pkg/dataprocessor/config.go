package dataprocessor

import (
	"time"
)

// DataSourceType 数据源类型
type DataSourceType string

const (
	DataSourceRedis DataSourceType = "redis"
	DataSourceMySQL DataSourceType = "mysql"
	DataSourceFile  DataSourceType = "file"
	DataSourceKafka DataSourceType = "kafka"
)

// DataSinkType 数据接收器类型
type DataSinkType string

const (
	DataSinkMySQL DataSinkType = "mysql"
	DataSinkFile  DataSinkType = "file"
	DataSinkRedis DataSinkType = "redis"
	DataSinkKafka DataSinkType = "kafka"
)

// ProcessorType 处理器类型
type ProcessorType string

const (
	ProcessorRecordTransformer ProcessorType = "record_transformer"
	ProcessorScript            ProcessorType = "script"
	ProcessorRoute             ProcessorType = "route"
	ProcessorSemantic          ProcessorType = "semantic"
)

// WindowType 窗口类型
type WindowType string

const (
	WindowTime    WindowType = "time"
	WindowCount   WindowType = "count"
	WindowSession WindowType = "session"
)

// DataProcessorConfig 数据处理配置
type DataProcessorConfig struct {
	// 基础配置
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description" yaml:"description"`
	Version     string `json:"version" yaml:"version"`

	// 数据源配置
	Source SourceConfig `json:"source" yaml:"source"`

	// 处理器配置
	Processor ProcessorConfig `json:"processor" yaml:"processor"`

	// 数据接收器配置
	Sink SinkConfig `json:"sink" yaml:"sink"`

	// 窗口配置
	Window WindowConfig `json:"window" yaml:"window"`

	// 流引擎配置
	StreamEngine StreamEngineConfig `json:"stream_engine" yaml:"stream_engine"`

	// 指标收集配置
	Metrics MetricsConfig `json:"metrics" yaml:"metrics"`

	// 状态管理配置
	State StateConfig `json:"state" yaml:"state"`
}

// SourceConfig 数据源配置
type SourceConfig struct {
	Type     DataSourceType `json:"type" yaml:"type"`
	Host     string         `json:"host" yaml:"host"`
	Port     int            `json:"port" yaml:"port"`
	Username string         `json:"username" yaml:"username"`
	Password string         `json:"password" yaml:"password"`
	Database string         `json:"database" yaml:"database"`
	Path     string         `json:"path" yaml:"path"`

	// Redis特定配置
	RedisKeyPattern string `json:"redis_key_pattern" yaml:"redis_key_pattern"`
	RedisBatchSize  int    `json:"redis_batch_size" yaml:"redis_batch_size"`

	// MySQL特定配置
	MySQLTable     string `json:"mysql_table" yaml:"mysql_table"`
	MySQLQuery     string `json:"mysql_query" yaml:"mysql_query"`
	MySQLBatchSize int    `json:"mysql_batch_size" yaml:"mysql_batch_size"`

	// 通用配置
	ConnectionTimeout time.Duration `json:"connection_timeout" yaml:"connection_timeout"`
	RetryCount        int           `json:"retry_count" yaml:"retry_count"`
	RetryInterval     time.Duration `json:"retry_interval" yaml:"retry_interval"`
}

// ProcessorConfig 处理器配置
type ProcessorConfig struct {
	Type ProcessorType `json:"type" yaml:"type"`

	// RecordTransformer配置
	TransformRules []TransformRule `json:"transform_rules" yaml:"transform_rules"`
	OutputFormat   string          `json:"output_format" yaml:"output_format"`

	// Script配置
	ScriptLanguage string `json:"script_language" yaml:"script_language"`
	ScriptCode     string `json:"script_code" yaml:"script_code"`

	// Route配置
	RouteRules []RouteRule `json:"route_rules" yaml:"route_rules"`

	// 通用配置
	BatchSize int `json:"batch_size" yaml:"batch_size"`
}

// TransformRule 转换规则
type TransformRule struct {
	SourceField string `json:"source_field" yaml:"source_field"`
	TargetField string `json:"target_field" yaml:"target_field"`
	Transform   string `json:"transform" yaml:"transform"` // 转换函数名
}

// RouteRule 路由规则
type RouteRule struct {
	Condition string `json:"condition" yaml:"condition"`
	Target    string `json:"target" yaml:"target"`
}

// SinkConfig 数据接收器配置
type SinkConfig struct {
	Type     DataSinkType `json:"type" yaml:"type"`
	Host     string       `json:"host" yaml:"host"`
	Port     int          `json:"port" yaml:"port"`
	Username string       `json:"username" yaml:"username"`
	Password string       `json:"password" yaml:"password"`
	Database string       `json:"database" yaml:"database"`
	Path     string       `json:"path" yaml:"path"`

	// MySQL特定配置
	MySQLTable     string `json:"mysql_table" yaml:"mysql_table"`
	MySQLUpsert    bool   `json:"mysql_upsert" yaml:"mysql_upsert"`
	MySQLBatchSize int    `json:"mysql_batch_size" yaml:"mysql_batch_size"`

	// 文件特定配置
	FileFormat   string `json:"file_format" yaml:"file_format"` // csv, json, xml
	FileEncoding string `json:"file_encoding" yaml:"file_encoding"`
	FileCompress bool   `json:"file_compress" yaml:"file_compress"`

	// 通用配置
	ConnectionTimeout time.Duration `json:"connection_timeout" yaml:"connection_timeout"`
	RetryCount        int           `json:"retry_count" yaml:"retry_count"`
	RetryInterval     time.Duration `json:"retry_interval" yaml:"retry_interval"`
}

// WindowConfig 窗口配置
type WindowConfig struct {
	Type    WindowType    `json:"type" yaml:"type"`
	Size    time.Duration `json:"size" yaml:"size"`
	Slide   time.Duration `json:"slide" yaml:"slide"`
	Count   int           `json:"count" yaml:"count"`
	Session time.Duration `json:"session" yaml:"session"`
	Trigger string        `json:"trigger" yaml:"trigger"` // 触发条件
}

// StreamEngineConfig 流引擎配置
type StreamEngineConfig struct {
	ThreadPoolSize    int           `json:"thread_pool_size" yaml:"thread_pool_size"`
	BackpressureLimit int           `json:"backpressure_limit" yaml:"backpressure_limit"`
	ProcessingTimeout time.Duration `json:"processing_timeout" yaml:"processing_timeout"`
	EnableMetrics     bool          `json:"enable_metrics" yaml:"enable_metrics"`
	EnableState       bool          `json:"enable_state" yaml:"enable_state"`
}

// MetricsConfig 指标配置
type MetricsConfig struct {
	Enabled          bool          `json:"enabled" yaml:"enabled"`
	ExportType       string        `json:"export_type" yaml:"export_type"` // prometheus, opentsdb, graphite
	ExportPort       int           `json:"export_port" yaml:"export_port"`
	Retention        time.Duration `json:"retention" yaml:"retention"`
	SnapshotInterval time.Duration `json:"snapshot_interval" yaml:"snapshot_interval"`
}

// StateConfig 状态配置
type StateConfig struct {
	Enabled      bool          `json:"enabled" yaml:"enabled"`
	ProviderType string        `json:"provider_type" yaml:"provider_type"` // local, distributed
	StoragePath  string        `json:"storage_path" yaml:"storage_path"`
	SyncInterval time.Duration `json:"sync_interval" yaml:"sync_interval"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *DataProcessorConfig {
	return &DataProcessorConfig{
		Name:        "default-processor",
		Description: "Default data processor configuration",
		Version:     "1.0.0",
		Source: SourceConfig{
			Type:              DataSourceRedis,
			Host:              "localhost",
			Port:              6379,
			RedisKeyPattern:   "*",
			RedisBatchSize:    1000,
			ConnectionTimeout: 30 * time.Second,
			RetryCount:        3,
			RetryInterval:     5 * time.Second,
		},
		Processor: ProcessorConfig{
			Type:         ProcessorRecordTransformer,
			BatchSize:    100,
			OutputFormat: "csv",
		},
		Sink: SinkConfig{
			Type:              DataSinkFile,
			Path:              "./data/output",
			FileFormat:        "csv",
			FileEncoding:      "utf-8",
			ConnectionTimeout: 30 * time.Second,
			RetryCount:        3,
			RetryInterval:     5 * time.Second,
		},
		Window: WindowConfig{
			Type:  WindowTime,
			Size:  60 * time.Second,
			Slide: 30 * time.Second,
		},
		StreamEngine: StreamEngineConfig{
			ThreadPoolSize:    4,
			BackpressureLimit: 1000,
			ProcessingTimeout: 60 * time.Second,
			EnableMetrics:     true,
			EnableState:       true,
		},
		Metrics: MetricsConfig{
			Enabled:          true,
			ExportType:       "prometheus",
			ExportPort:       9090,
			Retention:        24 * time.Hour,
			SnapshotInterval: 1 * time.Minute,
		},
		State: StateConfig{
			Enabled:      true,
			ProviderType: "local",
			StoragePath:  "./state",
			SyncInterval: 30 * time.Second,
		},
	}
}
