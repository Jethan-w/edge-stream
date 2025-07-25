package processor

import (
	"github.com/crazy/edge-stream/internal/flowfile"
)

// Processor 处理器接口定义
type Processor interface {
	// Process 处理数据
	Process(flowFile *flowfile.FlowFile) (*flowfile.FlowFile, error)

	// GetName 获取处理器名称
	GetName() string
}

// SimpleProcessor 简单处理器实现
type SimpleProcessor struct {
	name string
}

// NewSimpleProcessor 创建简单处理器
func NewSimpleProcessor(name string) *SimpleProcessor {
	return &SimpleProcessor{
		name: name,
	}
}

// Process 处理数据 - 默认直接返回原数据
func (sp *SimpleProcessor) Process(flowFile *flowfile.FlowFile) (*flowfile.FlowFile, error) {
	// 简单的处理逻辑 - 可以被子类重写
	return flowFile, nil
}

// GetName 获取处理器名称
func (sp *SimpleProcessor) GetName() string {
	return sp.name
}

// TransformProcessor 数据转换处理器
type TransformProcessor struct {
	*SimpleProcessor
	transformFunc func(*flowfile.FlowFile) (*flowfile.FlowFile, error)
}

// NewTransformProcessor 创建转换处理器
func NewTransformProcessor(name string, transformFunc func(*flowfile.FlowFile) (*flowfile.FlowFile, error)) *TransformProcessor {
	return &TransformProcessor{
		SimpleProcessor: NewSimpleProcessor(name),
		transformFunc:   transformFunc,
	}
}

// Process 处理数据
func (tp *TransformProcessor) Process(flowFile *flowfile.FlowFile) (*flowfile.FlowFile, error) {
	if tp.transformFunc != nil {
		return tp.transformFunc(flowFile)
	}
	return tp.SimpleProcessor.Process(flowFile)
}
