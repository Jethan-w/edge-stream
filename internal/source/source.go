package source

import (
	"fmt"

	"github.com/crazy/edge-stream/internal/flowfile"
)

// Source 数据源接口定义
type Source interface {
	// Read 读取数据
	Read() (*flowfile.FlowFile, error)

	// GetName 获取数据源名称
	GetName() string

	// HasNext 是否还有数据
	HasNext() bool
}

// SimpleSource 简单数据源实现
type SimpleSource struct {
	name  string
	data  []*flowfile.FlowFile
	index int
}

// NewSimpleSource 创建简单数据源
func NewSimpleSource(name string, data []*flowfile.FlowFile) *SimpleSource {
	return &SimpleSource{
		name:  name,
		data:  data,
		index: 0,
	}
}

// Read 读取数据
func (ss *SimpleSource) Read() (*flowfile.FlowFile, error) {
	if ss.index >= len(ss.data) {
		return nil, nil // 没有更多数据
	}

	flowFile := ss.data[ss.index]
	ss.index++
	return flowFile, nil
}

// GetName 获取数据源名称
func (ss *SimpleSource) GetName() string {
	return ss.name
}

// HasNext 是否还有数据
func (ss *SimpleSource) HasNext() bool {
	return ss.index < len(ss.data)
}

// Reset 重置数据源
func (ss *SimpleSource) Reset() {
	ss.index = 0
}

// StringSource 字符串数据源
type StringSource struct {
	name  string
	data  []string
	index int
}

// NewStringSource 创建字符串数据源
func NewStringSource(name string, data []string) *StringSource {
	return &StringSource{
		name:  name,
		data:  data,
		index: 0,
	}
}

// Read 读取数据
func (ss *StringSource) Read() (*flowfile.FlowFile, error) {
	if ss.index >= len(ss.data) {
		return nil, nil // 没有更多数据
	}

	flowFile := flowfile.NewFlowFile()
	flowFile.Content = []byte(ss.data[ss.index])
	flowFile.Size = int64(len(flowFile.Content))
	flowFile.Attributes["source"] = ss.name
	flowFile.Attributes["index"] = fmt.Sprintf("%d", ss.index)

	ss.index++
	return flowFile, nil
}

// GetName 获取数据源名称
func (ss *StringSource) GetName() string {
	return ss.name
}

// HasNext 是否还有数据
func (ss *StringSource) HasNext() bool {
	return ss.index < len(ss.data)
}

// Reset 重置数据源
func (ss *StringSource) Reset() {
	ss.index = 0
}
