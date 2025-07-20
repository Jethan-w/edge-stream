package statemanager

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// FileSystemStateProvider 文件系统状态提供者实现
type FileSystemStateProvider struct {
	stateDirectory string
	mu             sync.RWMutex
}

// NewFileSystemStateProvider 创建文件系统状态提供者
func NewFileSystemStateProvider() *FileSystemStateProvider {
	return &FileSystemStateProvider{}
}

// Initialize 初始化提供者
func (fsp *FileSystemStateProvider) Initialize(config ProviderConfiguration) error {
	fsp.mu.Lock()
	defer fsp.mu.Unlock()

	// 获取状态目录路径
	stateDir, ok := config.Properties["state.directory"]
	if !ok {
		stateDir = "./state" // 默认状态目录
	}

	fsp.stateDirectory = stateDir

	// 创建状态目录
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	return nil
}

// LoadState 加载组件状态
func (fsp *FileSystemStateProvider) LoadState(componentID string) (*StateMap, error) {
	fsp.mu.RLock()
	defer fsp.mu.RUnlock()

	componentStatePath := filepath.Join(fsp.stateDirectory, componentID+".state")

	// 检查文件是否存在
	if _, err := os.Stat(componentStatePath); os.IsNotExist(err) {
		// 文件不存在，返回空状态
		return NewStateMap(nil, StateScopeLocal), nil
	}

	// 读取状态文件
	file, err := os.Open(componentStatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open state file: %w", err)
	}
	defer file.Close()

	// 尝试解析为 JSON 格式
	var stateMap StateMap
	if err := json.NewDecoder(file).Decode(&stateMap); err == nil {
		return &stateMap, nil
	}

	// 如果不是 JSON 格式，尝试解析为键值对格式
	file.Seek(0, 0)
	scanner := bufio.NewScanner(file)
	stateValues := make(map[string]string)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			stateValues[parts[0]] = parts[1]
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	return NewStateMap(stateValues, StateScopeLocal), nil
}

// PersistState 持久化组件状态
func (fsp *FileSystemStateProvider) PersistState(componentID string, stateMap *StateMap) error {
	fsp.mu.Lock()
	defer fsp.mu.Unlock()

	componentStatePath := filepath.Join(fsp.stateDirectory, componentID+".state")

	// 创建临时文件
	tempPath := componentStatePath + ".tmp"
	file, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temp state file: %w", err)
	}
	defer file.Close()

	// 序列化为 JSON 格式
	if err := json.NewEncoder(file).Encode(stateMap); err != nil {
		return fmt.Errorf("failed to encode state: %w", err)
	}

	// 确保数据写入磁盘
	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to sync state file: %w", err)
	}

	// 原子性地重命名文件
	if err := os.Rename(tempPath, componentStatePath); err != nil {
		return fmt.Errorf("failed to rename temp state file: %w", err)
	}

	return nil
}

// Shutdown 关闭资源
func (fsp *FileSystemStateProvider) Shutdown() error {
	// 文件系统提供者不需要特殊清理
	return nil
}

// MemoryStateProvider 内存状态提供者实现（用于测试和临时状态）
type MemoryStateProvider struct {
	states map[string]*StateMap
	mu     sync.RWMutex
}

// NewMemoryStateProvider 创建内存状态提供者
func NewMemoryStateProvider() *MemoryStateProvider {
	return &MemoryStateProvider{
		states: make(map[string]*StateMap),
	}
}

// Initialize 初始化提供者
func (msp *MemoryStateProvider) Initialize(config ProviderConfiguration) error {
	// 内存提供者不需要特殊初始化
	return nil
}

// LoadState 加载组件状态
func (msp *MemoryStateProvider) LoadState(componentID string) (*StateMap, error) {
	msp.mu.RLock()
	defer msp.mu.RUnlock()

	state, exists := msp.states[componentID]
	if !exists {
		// 状态不存在，返回空状态
		return NewStateMap(nil, StateScopeLocal), nil
	}

	return state, nil
}

// PersistState 持久化组件状态
func (msp *MemoryStateProvider) PersistState(componentID string, stateMap *StateMap) error {
	msp.mu.Lock()
	defer msp.mu.Unlock()

	// 创建状态的副本
	stateCopy := &StateMap{
		StateValues: make(map[string]string),
		Timestamp:   stateMap.Timestamp,
		Version:     stateMap.Version,
		Scope:       stateMap.Scope,
	}

	for k, v := range stateMap.StateValues {
		stateCopy.StateValues[k] = v
	}

	msp.states[componentID] = stateCopy
	return nil
}

// Shutdown 关闭资源
func (msp *MemoryStateProvider) Shutdown() error {
	msp.mu.Lock()
	defer msp.mu.Unlock()

	// 清空内存中的状态
	msp.states = make(map[string]*StateMap)
	return nil
}

// GetStateCount 获取状态数量（用于测试）
func (msp *MemoryStateProvider) GetStateCount() int {
	msp.mu.RLock()
	defer msp.mu.RUnlock()

	return len(msp.states)
}

// ClearStates 清空所有状态（用于测试）
func (msp *MemoryStateProvider) ClearStates() {
	msp.mu.Lock()
	defer msp.mu.Unlock()

	msp.states = make(map[string]*StateMap)
}
