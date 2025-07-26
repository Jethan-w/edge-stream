// Copyright 2025 EdgeStream Team
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sink

import (
	"fmt"

	"github.com/crazy/edge-stream/internal/flowfile"
)

// Sink 数据输出接口定义
type Sink interface {
	// Write 写入数据
	Write(flowFile *flowfile.FlowFile) error

	// GetName 获取输出器名称
	GetName() string
}

// SimpleSink 简单数据输出实现
type SimpleSink struct {
	name string
	data []*flowfile.FlowFile
}

// NewSimpleSink 创建简单数据输出器
func NewSimpleSink(name string) *SimpleSink {
	return &SimpleSink{
		name: name,
		data: make([]*flowfile.FlowFile, 0),
	}
}

// Write 写入数据
func (ss *SimpleSink) Write(flowFile *flowfile.FlowFile) error {
	if flowFile == nil {
		return fmt.Errorf("flowFile cannot be nil")
	}

	ss.data = append(ss.data, flowFile)
	return nil
}

// GetName 获取输出器名称
func (ss *SimpleSink) GetName() string {
	return ss.name
}

// GetData 获取所有数据
func (ss *SimpleSink) GetData() []*flowfile.FlowFile {
	return ss.data
}

// Clear 清空数据
func (ss *SimpleSink) Clear() {
	ss.data = make([]*flowfile.FlowFile, 0)
}

// ConsoleSink 控制台输出器
type ConsoleSink struct {
	name string
}

// NewConsoleSink 创建控制台输出器
func NewConsoleSink(name string) *ConsoleSink {
	return &ConsoleSink{
		name: name,
	}
}

// Write 写入数据到控制台
func (cs *ConsoleSink) Write(flowFile *flowfile.FlowFile) error {
	if flowFile == nil {
		return fmt.Errorf("flowFile cannot be nil")
	}

	fmt.Printf("[%s] FlowFile ID: %s, Size: %d bytes, Content: %s\n",
		cs.name, flowFile.UUID, flowFile.Size, string(flowFile.Content))
	return nil
}

// GetName 获取输出器名称
func (cs *ConsoleSink) GetName() string {
	return cs.name
}
