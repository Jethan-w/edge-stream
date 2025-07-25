package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/crazy/edge-stream/internal/flowfile"
)

func main() {
	log.Println("=== Edge Stream 模拟数据处理演示 ===")

	// 演示MySQL到文件的流式处理（模拟）
	demonstrateStreamProcessing()

	// 等待一段时间
	time.Sleep(2 * time.Second)

	// 演示Redis聚合处理（模拟）
	demonstrateAggregationProcessing()

	log.Println("演示完成")
}

// demonstrateStreamProcessing 演示流式处理
func demonstrateStreamProcessing() {
	log.Println("\n=== 演示MySQL到文件的流式处理（模拟） ===")

	// 创建模拟数据源
	source := &MockMySQLSource{
		data: []map[string]interface{}{
			{"id": 1, "name": "张三", "email": "zhangsan@example.com", "created_at": time.Now()},
			{"id": 2, "name": "李四", "email": "lisi@example.com", "created_at": time.Now()},
			{"id": 3, "name": "王五", "email": "wangwu@example.com", "created_at": time.Now()},
			{"id": 4, "name": "赵六", "email": "zhaoliu@example.com", "created_at": time.Now()},
			{"id": 5, "name": "钱七", "email": "qianqi@example.com", "created_at": time.Now()},
		},
		index: 0,
	}

	// 创建CSV处理器
	processor := &CSVProcessor{}

	// 创建文件输出器
	outputDir := "./data/output"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Printf("创建输出目录失败: %v", err)
		return
	}

	timestamp := time.Now().Format("20060102_150405")
	fileName := fmt.Sprintf("mysql_export_%s.csv", timestamp)
	filePath := filepath.Join(outputDir, fileName)

	sink := &FileSink{filePath: filePath}

	// 执行流式处理
	processedCount := 0
	log.Println("开始流式处理...")

	// 写入CSV头部
	headerFlow := flowfile.NewFlowFile()
	headerFlow.Content = []byte("id,name,email,created_at\n")
	if err := sink.Write(headerFlow); err != nil {
		log.Printf("写入头部失败: %v", err)
		return
	}

	// 处理数据
	for {
		flowFile, err := source.Read()
		if err != nil {
			if err.Error() == "没有更多数据" {
				break
			}
			log.Printf("读取数据失败: %v", err)
			continue
		}

		// 处理数据
		processedFlow, err := processor.Process(flowFile)
		if err != nil {
			log.Printf("处理数据失败: %v", err)
			continue
		}

		// 写入文件
		if err := sink.Write(processedFlow); err != nil {
			log.Printf("写入文件失败: %v", err)
			continue
		}

		processedCount++
		log.Printf("已处理 %d 条记录: %s", processedCount, string(processedFlow.Content))
	}

	sink.Close()
	log.Printf("MySQL到文件处理完成，共处理 %d 条记录，输出文件: %s", processedCount, filePath)
}

// demonstrateAggregationProcessing 演示聚合处理
func demonstrateAggregationProcessing() {
	log.Println("\n=== 演示Redis到数据库的聚合处理（模拟） ===")

	// 模拟Redis数据
	redisData := map[string]string{
		"user:1001:profile": "张三的个人资料",
		"user:1002:profile": "李四的个人资料",
		"user:1003:profile": "王五的个人资料",
		"prod:2001:info":    "产品A信息",
		"prod:2002:info":    "产品B信息",
		"prod:2003:info":    "产品C信息",
		"order:3001:data":   "订单1数据",
		"order:3002:data":   "订单2数据",
		"order:3003:data":   "订单3数据",
		"cache:4001:temp":   "临时缓存1",
		"cache:4002:temp":   "临时缓存2",
		"session:5001:auth": "会话认证1",
		"session:5002:auth": "会话认证2",
		"config:6001:app":   "应用配置1",
		"config:6002:app":   "应用配置2",
	}

	log.Printf("从Redis获取到 %d 个键", len(redisData))

	// 统计键前缀（前4位）
	prefixCount := make(map[string]int)
	processedKeys := 0
	startTime := time.Now()

	log.Println("开始聚合处理...")

	// 模拟每5秒统计一次
	ticker := time.NewTicker(2 * time.Second) // 为了演示，使用2秒
	defer ticker.Stop()

	keysSlice := make([]string, 0, len(redisData))
	for key := range redisData {
		keysSlice = append(keysSlice, key)
	}

	go func() {
		for range ticker.C {
			if processedKeys >= len(keysSlice) {
				return
			}
			duration := time.Since(startTime)
			keysPerSecond := float64(processedKeys) / duration.Seconds()
			log.Printf("处理统计: 已处理 %d 个键，处理速度: %.2f 键/秒", processedKeys, keysPerSecond)
		}
	}()

	// 处理每个键
	for _, key := range keysSlice {
		value := redisData[key]

		// 获取键的前4个字符作为前缀
		prefix := ""
		if len(key) >= 4 {
			prefix = key[:4]
		} else {
			prefix = key
		}
		prefixCount[prefix]++

		log.Printf("处理键: %s -> %s (前缀: %s)", key, value, prefix)
		processedKeys++

		// 模拟处理延迟
		time.Sleep(200 * time.Millisecond)
	}

	// 显示聚合结果
	log.Println("\n=== 前缀统计结果 ===")
	for prefix, count := range prefixCount {
		log.Printf("前缀 '%s': %d 个键", prefix, count)
	}

	// 模拟写入数据库
	log.Println("\n=== 模拟写入数据库 ===")
	log.Printf("成功写入 %d 条Redis数据到数据库", len(redisData))
	log.Printf("成功写入 %d 个前缀统计到数据库", len(prefixCount))

	totalDuration := time.Since(startTime)
	log.Printf("Redis数据处理完成，处理了 %d 个键，统计了 %d 个前缀，总耗时: %v", len(redisData), len(prefixCount), totalDuration)
}

// MockMySQLSource 模拟MySQL数据源
type MockMySQLSource struct {
	data  []map[string]interface{}
	index int
}

func (s *MockMySQLSource) Read() (*flowfile.FlowFile, error) {
	if s.index >= len(s.data) {
		return nil, fmt.Errorf("没有更多数据")
	}

	row := s.data[s.index]
	s.index++

	// 构建CSV内容
	content := fmt.Sprintf("%v,%v,%v,%v",
		row["id"], row["name"], row["email"], row["created_at"].(time.Time).Format("2006-01-02 15:04:05"))

	// 创建FlowFile
	flowFile := flowfile.NewFlowFile()
	flowFile.Content = []byte(content)
	flowFile.Attributes["source"] = "mock-mysql"
	flowFile.Attributes["record.index"] = fmt.Sprintf("%d", s.index-1)
	flowFile.Size = int64(len(content))

	return flowFile, nil
}

func (s *MockMySQLSource) GetName() string {
	return "mock-mysql-source"
}

// CSVProcessor CSV处理器
type CSVProcessor struct{}

func (p *CSVProcessor) Process(flowFile *flowfile.FlowFile) (*flowfile.FlowFile, error) {
	// 简单地添加换行符
	processedContent := string(flowFile.Content) + "\n"

	processedFlow := flowfile.NewFlowFile()
	processedFlow.Content = []byte(processedContent)
	processedFlow.Attributes = flowFile.Attributes
	processedFlow.Attributes["processor"] = "csv-processor"
	processedFlow.Size = int64(len(processedContent))

	return processedFlow, nil
}

func (p *CSVProcessor) GetName() string {
	return "csv-processor"
}

// FileSink 文件输出器
type FileSink struct {
	filePath string
	file     *os.File
}

func (s *FileSink) Write(flowFile *flowfile.FlowFile) error {
	if s.file == nil {
		file, err := os.Create(s.filePath)
		if err != nil {
			return fmt.Errorf("创建文件失败: %v", err)
		}
		s.file = file
	}

	_, err := s.file.Write(flowFile.Content)
	return err
}

func (s *FileSink) GetName() string {
	return "file-sink"
}

func (s *FileSink) Close() error {
	if s.file != nil {
		return s.file.Close()
	}
	return nil
}