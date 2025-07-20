package timestampextractor

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/edge-stream/internal/WindowManager"
)

// TimestampExtractor 时间戳提取器接口
type TimestampExtractor interface {
	// ExtractTimestamp 从 FlowFile 提取事件时间戳
	ExtractTimestamp(flowFile *windowmanager.FlowFile) int64

	// GetTimestampFormat 获取时间戳格式
	GetTimestampFormat() string

	// AllowLateData 处理乱序数据
	AllowLateData(maxLateness int64) bool
}

// JsonTimestampExtractor JSON 时间戳提取器
type JsonTimestampExtractor struct {
	timestampField string
	dateFormat     string
	maxLateness    int64
}

// NewJsonTimestampExtractor 创建 JSON 时间戳提取器
func NewJsonTimestampExtractor(timestampField, dateFormat string) *JsonTimestampExtractor {
	return &JsonTimestampExtractor{
		timestampField: timestampField,
		dateFormat:     dateFormat,
		maxLateness:    300000, // 默认5分钟
	}
}

// ExtractTimestamp 从 FlowFile 提取事件时间戳
func (jte *JsonTimestampExtractor) ExtractTimestamp(flowFile *windowmanager.FlowFile) int64 {
	content := flowFile.GetContent()

	var jsonData map[string]interface{}
	if err := json.Unmarshal(content, &jsonData); err != nil {
		// JSON 解析失败，返回当前时间
		return time.Now().UnixMilli()
	}

	// 获取时间戳字段
	timestampValue, exists := jsonData[jte.timestampField]
	if !exists {
		// 字段不存在，返回当前时间
		return time.Now().UnixMilli()
	}

	// 根据字段类型处理时间戳
	switch v := timestampValue.(type) {
	case string:
		return jte.parseTimestampString(v)
	case float64:
		// 假设是 Unix 时间戳（秒）
		return int64(v * 1000)
	case int64:
		// 假设是 Unix 时间戳（毫秒）
		return v
	case int:
		// 假设是 Unix 时间戳（秒）
		return int64(v) * 1000
	default:
		// 未知类型，返回当前时间
		return time.Now().UnixMilli()
	}
}

// parseTimestampString 解析时间戳字符串
func (jte *JsonTimestampExtractor) parseTimestampString(timestampStr string) int64 {
	// 尝试解析为 Unix 时间戳
	if timestamp, err := strconv.ParseInt(timestampStr, 10, 64); err == nil {
		// 判断是秒还是毫秒
		if timestamp > 1000000000000 { // 毫秒时间戳
			return timestamp
		} else { // 秒时间戳
			return timestamp * 1000
		}
	}

	// 尝试解析为日期格式
	if jte.dateFormat != "" {
		if timestamp, err := time.Parse(jte.dateFormat, timestampStr); err == nil {
			return timestamp.UnixMilli()
		}
	}

	// 尝试常见的日期格式
	commonFormats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		time.RFC3339,
		time.RFC3339Nano,
	}

	for _, format := range commonFormats {
		if timestamp, err := time.Parse(format, timestampStr); err == nil {
			return timestamp.UnixMilli()
		}
	}

	// 解析失败，返回当前时间
	return time.Now().UnixMilli()
}

// GetTimestampFormat 获取时间戳格式
func (jte *JsonTimestampExtractor) GetTimestampFormat() string {
	return jte.dateFormat
}

// AllowLateData 处理乱序数据
func (jte *JsonTimestampExtractor) AllowLateData(maxLateness int64) bool {
	if maxLateness > 0 {
		jte.maxLateness = maxLateness
	}
	return true
}

// GetMaxLateness 获取最大延迟时间
func (jte *JsonTimestampExtractor) GetMaxLateness() int64 {
	return jte.maxLateness
}

// AttributeTimestampExtractor 属性时间戳提取器
type AttributeTimestampExtractor struct {
	timestampAttribute string
	dateFormat         string
	maxLateness        int64
}

// NewAttributeTimestampExtractor 创建属性时间戳提取器
func NewAttributeTimestampExtractor(timestampAttribute, dateFormat string) *AttributeTimestampExtractor {
	return &AttributeTimestampExtractor{
		timestampAttribute: timestampAttribute,
		dateFormat:         dateFormat,
		maxLateness:        300000, // 默认5分钟
	}
}

// ExtractTimestamp 从 FlowFile 属性提取事件时间戳
func (ate *AttributeTimestampExtractor) ExtractTimestamp(flowFile *windowmanager.FlowFile) int64 {
	timestampStr := flowFile.GetAttribute(ate.timestampAttribute)
	if timestampStr == "" {
		// 属性不存在，返回当前时间
		return time.Now().UnixMilli()
	}

	// 尝试解析为 Unix 时间戳
	if timestamp, err := strconv.ParseInt(timestampStr, 10, 64); err == nil {
		// 判断是秒还是毫秒
		if timestamp > 1000000000000 { // 毫秒时间戳
			return timestamp
		} else { // 秒时间戳
			return timestamp * 1000
		}
	}

	// 尝试解析为日期格式
	if ate.dateFormat != "" {
		if timestamp, err := time.Parse(ate.dateFormat, timestampStr); err == nil {
			return timestamp.UnixMilli()
		}
	}

	// 尝试常见的日期格式
	commonFormats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		time.RFC3339,
		time.RFC3339Nano,
	}

	for _, format := range commonFormats {
		if timestamp, err := time.Parse(format, timestampStr); err == nil {
			return timestamp.UnixMilli()
		}
	}

	// 解析失败，返回当前时间
	return time.Now().UnixMilli()
}

// GetTimestampFormat 获取时间戳格式
func (ate *AttributeTimestampExtractor) GetTimestampFormat() string {
	return ate.dateFormat
}

// AllowLateData 处理乱序数据
func (ate *AttributeTimestampExtractor) AllowLateData(maxLateness int64) bool {
	if maxLateness > 0 {
		ate.maxLateness = maxLateness
	}
	return true
}

// ProcessingTimeExtractor 处理时间提取器
type ProcessingTimeExtractor struct{}

// NewProcessingTimeExtractor 创建处理时间提取器
func NewProcessingTimeExtractor() *ProcessingTimeExtractor {
	return &ProcessingTimeExtractor{}
}

// ExtractTimestamp 提取处理时间戳
func (pte *ProcessingTimeExtractor) ExtractTimestamp(flowFile *windowmanager.FlowFile) int64 {
	// 直接返回当前处理时间
	return time.Now().UnixMilli()
}

// GetTimestampFormat 获取时间戳格式
func (pte *ProcessingTimeExtractor) GetTimestampFormat() string {
	return "processing_time"
}

// AllowLateData 处理乱序数据
func (pte *ProcessingTimeExtractor) AllowLateData(maxLateness int64) bool {
	// 处理时间不存在乱序问题
	return true
}

// CsvTimestampExtractor CSV 时间戳提取器
type CsvTimestampExtractor struct {
	timestampColumn int
	dateFormat      string
	delimiter       string
	maxLateness     int64
}

// NewCsvTimestampExtractor 创建 CSV 时间戳提取器
func NewCsvTimestampExtractor(timestampColumn int, dateFormat, delimiter string) *CsvTimestampExtractor {
	if delimiter == "" {
		delimiter = ","
	}

	return &CsvTimestampExtractor{
		timestampColumn: timestampColumn,
		dateFormat:      dateFormat,
		delimiter:       delimiter,
		maxLateness:     300000, // 默认5分钟
	}
}

// ExtractTimestamp 从 CSV 内容提取事件时间戳
func (cte *CsvTimestampExtractor) ExtractTimestamp(flowFile *windowmanager.FlowFile) int64 {
	content := string(flowFile.GetContent())
	lines := strings.Split(content, "\n")

	if len(lines) == 0 {
		return time.Now().UnixMilli()
	}

	// 取第一行数据
	firstLine := strings.TrimSpace(lines[0])
	if firstLine == "" {
		return time.Now().UnixMilli()
	}

	// 分割 CSV 行
	columns := strings.Split(firstLine, cte.delimiter)
	if len(columns) <= cte.timestampColumn {
		return time.Now().UnixMilli()
	}

	timestampStr := strings.TrimSpace(columns[cte.timestampColumn])
	if timestampStr == "" {
		return time.Now().UnixMilli()
	}

	// 尝试解析为 Unix 时间戳
	if timestamp, err := strconv.ParseInt(timestampStr, 10, 64); err == nil {
		// 判断是秒还是毫秒
		if timestamp > 1000000000000 { // 毫秒时间戳
			return timestamp
		} else { // 秒时间戳
			return timestamp * 1000
		}
	}

	// 尝试解析为日期格式
	if cte.dateFormat != "" {
		if timestamp, err := time.Parse(cte.dateFormat, timestampStr); err == nil {
			return timestamp.UnixMilli()
		}
	}

	// 尝试常见的日期格式
	commonFormats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		time.RFC3339,
		time.RFC3339Nano,
	}

	for _, format := range commonFormats {
		if timestamp, err := time.Parse(format, timestampStr); err == nil {
			return timestamp.UnixMilli()
		}
	}

	// 解析失败，返回当前时间
	return time.Now().UnixMilli()
}

// GetTimestampFormat 获取时间戳格式
func (cte *CsvTimestampExtractor) GetTimestampFormat() string {
	return cte.dateFormat
}

// AllowLateData 处理乱序数据
func (cte *CsvTimestampExtractor) AllowLateData(maxLateness int64) bool {
	if maxLateness > 0 {
		cte.maxLateness = maxLateness
	}
	return true
}

// LateDataStrategy 延迟数据处理策略
type LateDataStrategy struct {
	maxLateness     int64
	handling        LateDataHandling
	sideOutputQueue chan *windowmanager.FlowFile
}

// LateDataHandling 延迟数据处理方式
type LateDataHandling string

const (
	LateDataHandlingDrop       LateDataHandling = "DROP"        // 丢弃
	LateDataHandlingSideOutput LateDataHandling = "SIDE_OUTPUT" // 输出到侧边流
	LateDataHandlingInclude    LateDataHandling = "INCLUDE"     // 包含在窗口计算
)

// NewLateDataStrategy 创建延迟数据处理策略
func NewLateDataStrategy(maxLateness int64, handling LateDataHandling) *LateDataStrategy {
	return &LateDataStrategy{
		maxLateness:     maxLateness,
		handling:        handling,
		sideOutputQueue: make(chan *windowmanager.FlowFile, 1000),
	}
}

// IsLateData 判断是否为延迟数据
func (lds *LateDataStrategy) IsLateData(eventTime, watermark int64) bool {
	return eventTime < watermark-lds.maxLateness
}

// HandleLateData 处理延迟数据
func (lds *LateDataStrategy) HandleLateData(flowFile *windowmanager.FlowFile, eventTime, watermark int64) bool {
	if !lds.IsLateData(eventTime, watermark) {
		return true // 不是延迟数据，正常处理
	}

	switch lds.handling {
	case LateDataHandlingDrop:
		// 丢弃延迟数据
		return false
	case LateDataHandlingSideOutput:
		// 输出到侧边流
		select {
		case lds.sideOutputQueue <- flowFile:
			return false
		default:
			// 队列满，丢弃
			return false
		}
	case LateDataHandlingInclude:
		// 包含在窗口计算中
		return true
	default:
		// 默认丢弃
		return false
	}
}

// GetSideOutputQueue 获取侧边输出队列
func (lds *LateDataStrategy) GetSideOutputQueue() <-chan *windowmanager.FlowFile {
	return lds.sideOutputQueue
}

// GetMaxLateness 获取最大延迟时间
func (lds *LateDataStrategy) GetMaxLateness() int64 {
	return lds.maxLateness
}

// GetHandling 获取处理方式
func (lds *LateDataStrategy) GetHandling() LateDataHandling {
	return lds.handling
}

// TimestampExtractorFactory 时间戳提取器工厂
type TimestampExtractorFactory struct{}

// NewTimestampExtractorFactory 创建时间戳提取器工厂
func NewTimestampExtractorFactory() *TimestampExtractorFactory {
	return &TimestampExtractorFactory{}
}

// CreateExtractor 创建时间戳提取器
func (tef *TimestampExtractorFactory) CreateExtractor(extractorType string, config map[string]string) (TimestampExtractor, error) {
	switch extractorType {
	case "json":
		timestampField := config["timestamp_field"]
		dateFormat := config["date_format"]
		return NewJsonTimestampExtractor(timestampField, dateFormat), nil
	case "attribute":
		timestampAttribute := config["timestamp_attribute"]
		dateFormat := config["date_format"]
		return NewAttributeTimestampExtractor(timestampAttribute, dateFormat), nil
	case "processing_time":
		return NewProcessingTimeExtractor(), nil
	case "csv":
		timestampColumn, _ := strconv.Atoi(config["timestamp_column"])
		dateFormat := config["date_format"]
		delimiter := config["delimiter"]
		return NewCsvTimestampExtractor(timestampColumn, dateFormat, delimiter), nil
	default:
		return nil, fmt.Errorf("unsupported timestamp extractor type: %s", extractorType)
	}
}
