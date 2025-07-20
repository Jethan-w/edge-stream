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

// ValidateFlowFile 验证 FlowFile
func (f *FlowFile) ValidateFlowFile() error {
	if f == nil {
		return fmt.Errorf("flowFile cannot be nil")
	}
	return nil
}
