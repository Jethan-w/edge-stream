package windowmanager

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/crazy/edge-stream/internal/flowfile"
)

// ContentRepositoryStateManager 基于文件系统的窗口状态管理器
type ContentRepositoryStateManager struct {
	baseDir    string
	mu         sync.RWMutex
	windowData map[string]*WindowStateData
}

// WindowStateData 窗口状态数据
type WindowStateData struct {
	WindowID       string               `json:"window_id"`
	WindowType     WindowType           `json:"window_type"`
	StartTime      int64                `json:"start_time"`
	EndTime        int64                `json:"end_time"`
	FlowFiles      []*flowfile.FlowFile `json:"flow_files"`
	Metadata       *WindowMetadata      `json:"metadata"`
	LastUpdated    int64                `json:"last_updated"`
	SerializedData []byte               `json:"serialized_data,omitempty"`
}

// NewContentRepositoryStateManager 创建基于文件系统的状态管理器
func NewContentRepositoryStateManager(baseDir string) (*ContentRepositoryStateManager, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	return &ContentRepositoryStateManager{
		baseDir:    baseDir,
		windowData: make(map[string]*WindowStateData),
	}, nil
}

// PersistState 持久化窗口状态
func (crsm *ContentRepositoryStateManager) PersistState(window Window) error {
	crsm.mu.Lock()
	defer crsm.mu.Unlock()

	// 获取窗口数据
	flowFiles := window.GetFlowFiles()
	metadata := window.GetMetadata()

	// 序列化 FlowFile 数据
	serializedFlowFiles, err := json.Marshal(flowFiles)
	if err != nil {
		return fmt.Errorf("failed to serialize flow files: %w", err)
	}

	// 创建状态数据
	stateData := &WindowStateData{
		WindowID:       window.GetID(),
		WindowType:     window.GetWindowType(),
		StartTime:      window.GetStartTime(),
		EndTime:        window.GetEndTime(),
		FlowFiles:      flowFiles,
		Metadata:       metadata,
		LastUpdated:    time.Now().UnixMilli(),
		SerializedData: serializedFlowFiles,
	}

	// 保存到内存
	crsm.windowData[window.GetID()] = stateData

	// 保存到文件
	return crsm.saveToFile(window.GetID(), stateData)
}

// RestoreState 恢复窗口状态
func (crsm *ContentRepositoryStateManager) RestoreState(windowID string) (Window, error) {
	crsm.mu.RLock()
	defer crsm.mu.RUnlock()

	// 先从内存中查找
	if stateData, exists := crsm.windowData[windowID]; exists {
		return crsm.createWindowFromState(stateData)
	}

	// 从文件中加载
	stateData, err := crsm.loadFromFile(windowID)
	if err != nil {
		return nil, fmt.Errorf("failed to load window state from file: %w", err)
	}

	// 保存到内存
	crsm.windowData[windowID] = stateData

	return crsm.createWindowFromState(stateData)
}

// DeleteState 删除窗口状态
func (crsm *ContentRepositoryStateManager) DeleteState(windowID string) error {
	crsm.mu.Lock()
	defer crsm.mu.Unlock()

	// 从内存中删除
	delete(crsm.windowData, windowID)

	// 从文件中删除
	filePath := filepath.Join(crsm.baseDir, fmt.Sprintf("%s.json", windowID))
	return os.Remove(filePath)
}

// CleanupExpiredStates 清理过期状态
func (crsm *ContentRepositoryStateManager) CleanupExpiredStates(expirationTime int64) error {
	crsm.mu.Lock()
	defer crsm.mu.Unlock()

	var expiredWindows []string

	// 查找过期的窗口
	for windowID, stateData := range crsm.windowData {
		if stateData.LastUpdated < expirationTime {
			expiredWindows = append(expiredWindows, windowID)
		}
	}

	// 删除过期的窗口
	for _, windowID := range expiredWindows {
		delete(crsm.windowData, windowID)

		filePath := filepath.Join(crsm.baseDir, fmt.Sprintf("%s.json", windowID))
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			fmt.Printf("Warning: failed to delete expired window file %s: %v\n", filePath, err)
		}
	}

	return nil
}

// saveToFile 保存状态到文件
func (crsm *ContentRepositoryStateManager) saveToFile(windowID string, stateData *WindowStateData) error {
	filePath := filepath.Join(crsm.baseDir, fmt.Sprintf("%s.json", windowID))

	// 序列化状态数据
	data, err := json.MarshalIndent(stateData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state data: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// loadFromFile 从文件加载状态
func (crsm *ContentRepositoryStateManager) loadFromFile(windowID string) (*WindowStateData, error) {
	filePath := filepath.Join(crsm.baseDir, fmt.Sprintf("%s.json", windowID))

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var stateData WindowStateData
	if err := json.Unmarshal(data, &stateData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state data: %w", err)
	}

	return &stateData, nil
}

// createWindowFromState 从状态数据创建窗口
func (crsm *ContentRepositoryStateManager) createWindowFromState(stateData *WindowStateData) (Window, error) {
	switch stateData.WindowType {
	case WindowTypeTumbling:
		return crsm.createTimeWindowFromState(stateData)
	case WindowTypeCountBased:
		return crsm.createCountWindowFromState(stateData)
	case WindowTypeSession:
		return crsm.createSessionWindowFromState(stateData)
	case WindowTypeSliding:
		return crsm.createSlidingWindowFromState(stateData)
	default:
		return nil, fmt.Errorf("unsupported window type: %s", stateData.WindowType)
	}
}

// createTimeWindowFromState 从状态创建时间窗口
func (crsm *ContentRepositoryStateManager) createTimeWindowFromState(stateData *WindowStateData) (Window, error) {
	// 这里需要导入 WindowDefiner 包来创建具体的窗口实现
	// 为了演示，我们创建一个简单的实现
	return nil, fmt.Errorf("time window restoration not implemented")
}

// createCountWindowFromState 从状态创建计数窗口
func (crsm *ContentRepositoryStateManager) createCountWindowFromState(stateData *WindowStateData) (Window, error) {
	return nil, fmt.Errorf("count window restoration not implemented")
}

// createSessionWindowFromState 从状态创建会话窗口
func (crsm *ContentRepositoryStateManager) createSessionWindowFromState(stateData *WindowStateData) (Window, error) {
	return nil, fmt.Errorf("session window restoration not implemented")
}

// createSlidingWindowFromState 从状态创建滑动窗口
func (crsm *ContentRepositoryStateManager) createSlidingWindowFromState(stateData *WindowStateData) (Window, error) {
	return nil, fmt.Errorf("sliding window restoration not implemented")
}

// GetWindowState 获取窗口状态
func (crsm *ContentRepositoryStateManager) GetWindowState(windowID string) (*WindowStateData, error) {
	crsm.mu.RLock()
	defer crsm.mu.RUnlock()

	if stateData, exists := crsm.windowData[windowID]; exists {
		return stateData, nil
	}

	return crsm.loadFromFile(windowID)
}

// ListWindowStates 列出所有窗口状态
func (crsm *ContentRepositoryStateManager) ListWindowStates() ([]*WindowStateData, error) {
	crsm.mu.RLock()
	defer crsm.mu.RUnlock()

	var states []*WindowStateData
	for _, stateData := range crsm.windowData {
		states = append(states, stateData)
	}

	return states, nil
}

// MemoryStateManager 内存状态管理器（用于测试）
type MemoryStateManager struct {
	windowData map[string]*WindowStateData
	mu         sync.RWMutex
}

// NewMemoryStateManager 创建内存状态管理器
func NewMemoryStateManager() *MemoryStateManager {
	return &MemoryStateManager{
		windowData: make(map[string]*WindowStateData),
	}
}

// PersistState 持久化窗口状态
func (msm *MemoryStateManager) PersistState(window Window) error {
	msm.mu.Lock()
	defer msm.mu.Unlock()

	// 获取窗口数据
	flowFiles := window.GetFlowFiles()
	metadata := window.GetMetadata()

	// 创建状态数据
	stateData := &WindowStateData{
		WindowID:    window.GetID(),
		WindowType:  window.GetWindowType(),
		StartTime:   window.GetStartTime(),
		EndTime:     window.GetEndTime(),
		FlowFiles:   flowFiles,
		Metadata:    metadata,
		LastUpdated: time.Now().UnixMilli(),
	}

	// 保存到内存
	msm.windowData[window.GetID()] = stateData

	return nil
}

// RestoreState 恢复窗口状态
func (msm *MemoryStateManager) RestoreState(windowID string) (Window, error) {
	msm.mu.RLock()
	defer msm.mu.RUnlock()

	_, exists := msm.windowData[windowID]
	if !exists {
		return nil, fmt.Errorf("window state not found: %s", windowID)
	}

	// 这里需要根据窗口类型创建具体的窗口实现
	// 为了演示，我们返回一个错误
	return nil, fmt.Errorf("window restoration not implemented")
}

// DeleteState 删除窗口状态
func (msm *MemoryStateManager) DeleteState(windowID string) error {
	msm.mu.Lock()
	defer msm.mu.Unlock()

	delete(msm.windowData, windowID)
	return nil
}

// CleanupExpiredStates 清理过期状态
func (msm *MemoryStateManager) CleanupExpiredStates(expirationTime int64) error {
	msm.mu.Lock()
	defer msm.mu.Unlock()

	var expiredWindows []string

	// 查找过期的窗口
	for windowID, stateData := range msm.windowData {
		if stateData.LastUpdated < expirationTime {
			expiredWindows = append(expiredWindows, windowID)
		}
	}

	// 删除过期的窗口
	for _, windowID := range expiredWindows {
		delete(msm.windowData, windowID)
	}

	return nil
}

// GetWindowState 获取窗口状态
func (msm *MemoryStateManager) GetWindowState(windowID string) (*WindowStateData, error) {
	msm.mu.RLock()
	defer msm.mu.RUnlock()

	stateData, exists := msm.windowData[windowID]
	if !exists {
		return nil, fmt.Errorf("window state not found: %s", windowID)
	}

	return stateData, nil
}

// ListWindowStates 列出所有窗口状态
func (msm *MemoryStateManager) ListWindowStates() ([]*WindowStateData, error) {
	msm.mu.RLock()
	defer msm.mu.RUnlock()

	var states []*WindowStateData
	for _, stateData := range msm.windowData {
		states = append(states, stateData)
	}

	return states, nil
}
