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

package flowfile

import (
	"fmt"
	"sync"
	"time"
)

// FlowFile 数据流转的基本单元
type FlowFile struct {
	UUID       string
	Attributes map[string]string
	Content    []byte
	Size       int64
	Timestamp  time.Time
	LineageID  string
	mu         sync.RWMutex
}

// NewFlowFile 创建新的FlowFile
func NewFlowFile() *FlowFile {
	return &FlowFile{
		UUID:       GenerateUUID(),
		Attributes: make(map[string]string),
		Content:    make([]byte, 0),
		Timestamp:  time.Now(),
		LineageID:  GenerateLineageID(),
	}
}

// 工具函数
func GenerateUUID() string {
	// 这里应该使用实际的UUID生成库
	// 例如：github.com/google/uuid
	return fmt.Sprintf("flowfile-%d", time.Now().UnixNano())
}

func GenerateLineageID() string {
	// 这里应该使用实际的LineageID生成逻辑
	return fmt.Sprintf("lineage-%d", time.Now().UnixNano())
}

// SetAttribute 设置属性
func (f *FlowFile) SetAttribute(key, value string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Attributes[key] = value
}

// GetAttribute 获取属性
func (f *FlowFile) GetAttribute(key string) (string, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	value, exists := f.Attributes[key]
	return value, exists
}

// ValidateFlowFile 验证 FlowFile
func (f *FlowFile) ValidateFlowFile() error {
	if f == nil {
		return fmt.Errorf("flowFile cannot be nil")
	}
	return nil
}
