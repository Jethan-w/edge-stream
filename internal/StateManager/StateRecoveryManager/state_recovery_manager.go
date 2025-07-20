package staterecoverymanager

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/edge-stream/internal/StateManager"
)

// StateCheckpointManager 检查点管理器
type StateCheckpointManager struct {
	checkpointDirectory string
	mu                  sync.RWMutex
}

// CheckpointInfo 检查点信息
type CheckpointInfo struct {
	ComponentID  string    `json:"component_id"`
	Timestamp    int64     `json:"timestamp"`
	Version      int       `json:"version"`
	FilePath     string    `json:"file_path"`
	Size         int64     `json:"size"`
	CreationTime time.Time `json:"creation_time"`
}

// NewStateCheckpointManager 创建检查点管理器
func NewStateCheckpointManager() *StateCheckpointManager {
	return &StateCheckpointManager{
		checkpointDirectory: "./checkpoints",
	}
}

// Initialize 初始化检查点管理器
func (scm *StateCheckpointManager) Initialize(checkpointDir string) error {
	scm.mu.Lock()
	defer scm.mu.Unlock()

	if checkpointDir != "" {
		scm.checkpointDirectory = checkpointDir
	}

	// 创建检查点目录
	if err := os.MkdirAll(scm.checkpointDirectory, 0755); err != nil {
		return fmt.Errorf("failed to create checkpoint directory: %w", err)
	}

	return nil
}

// CreateCheckpoint 创建检查点
func (scm *StateCheckpointManager) CreateCheckpoint(componentID string, stateMap *statemanager.StateMap) error {
	scm.mu.Lock()
	defer scm.mu.Unlock()

	// 生成检查点文件名
	timestamp := time.Now().UnixMilli()
	checkpointFileName := fmt.Sprintf("state_checkpoint_%s_%d.json", componentID, timestamp)
	checkpointPath := filepath.Join(scm.checkpointDirectory, checkpointFileName)

	// 序列化状态
	data, err := statemanager.SerializeStateMap(stateMap)
	if err != nil {
		return fmt.Errorf("failed to serialize state for checkpoint: %w", err)
	}

	// 创建检查点文件
	file, err := os.Create(checkpointPath)
	if err != nil {
		return fmt.Errorf("failed to create checkpoint file: %w", err)
	}
	defer file.Close()

	// 写入状态数据
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("failed to write checkpoint data: %w", err)
	}

	// 确保数据写入磁盘
	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to sync checkpoint file: %w", err)
	}

	// 创建检查点信息文件
	checkpointInfo := &CheckpointInfo{
		ComponentID:  componentID,
		Timestamp:    timestamp,
		Version:      stateMap.Version,
		FilePath:     checkpointPath,
		Size:         int64(len(data)),
		CreationTime: time.Now(),
	}

	infoFileName := fmt.Sprintf("state_checkpoint_%s_%d.info", componentID, timestamp)
	infoPath := filepath.Join(scm.checkpointDirectory, infoFileName)

	infoData, err := json.MarshalIndent(checkpointInfo, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint info: %w", err)
	}

	if err := os.WriteFile(infoPath, infoData, 0644); err != nil {
		return fmt.Errorf("failed to write checkpoint info: %w", err)
	}

	return nil
}

// RestoreLatestCheckpoint 恢复最新检查点
func (scm *StateCheckpointManager) RestoreLatestCheckpoint(componentID string) (*statemanager.StateMap, error) {
	scm.mu.RLock()
	defer scm.mu.RUnlock()

	// 查找组件的所有检查点
	checkpoints, err := scm.findCheckpointsForComponent(componentID)
	if err != nil {
		return nil, fmt.Errorf("failed to find checkpoints: %w", err)
	}

	if len(checkpoints) == 0 {
		return nil, fmt.Errorf("no checkpoints found for component: %s", componentID)
	}

	// 选择最新检查点
	latestCheckpoint := scm.selectLatestCheckpoint(checkpoints)

	// 读取检查点数据
	data, err := os.ReadFile(latestCheckpoint.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read checkpoint file: %w", err)
	}

	// 反序列化状态
	stateMap, err := statemanager.DeserializeStateMap(data)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize checkpoint state: %w", err)
	}

	return stateMap, nil
}

// findCheckpointsForComponent 查找组件的所有检查点
func (scm *StateCheckpointManager) findCheckpointsForComponent(componentID string) ([]*CheckpointInfo, error) {
	var checkpoints []*CheckpointInfo

	// 读取检查点目录
	entries, err := os.ReadDir(scm.checkpointDirectory)
	if err != nil {
		return nil, err
	}

	// 查找组件的检查点信息文件
	prefix := fmt.Sprintf("state_checkpoint_%s_", componentID)
	suffix := ".info"

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), prefix) && strings.HasSuffix(entry.Name(), suffix) {
			infoPath := filepath.Join(scm.checkpointDirectory, entry.Name())
			infoData, err := os.ReadFile(infoPath)
			if err != nil {
				continue
			}

			var checkpointInfo CheckpointInfo
			if err := json.Unmarshal(infoData, &checkpointInfo); err != nil {
				continue
			}

			checkpoints = append(checkpoints, &checkpointInfo)
		}
	}

	return checkpoints, nil
}

// selectLatestCheckpoint 选择最新检查点
func (scm *StateCheckpointManager) selectLatestCheckpoint(checkpoints []*CheckpointInfo) *CheckpointInfo {
	if len(checkpoints) == 0 {
		return nil
	}

	// 按时间戳排序，选择最新的
	sort.Slice(checkpoints, func(i, j int) bool {
		return checkpoints[i].Timestamp > checkpoints[j].Timestamp
	})

	return checkpoints[0]
}

// CleanupOldCheckpoints 清理旧检查点
func (scm *StateCheckpointManager) CleanupOldCheckpoints(componentID string, keepCount int) error {
	scm.mu.Lock()
	defer scm.mu.Unlock()

	checkpoints, err := scm.findCheckpointsForComponent(componentID)
	if err != nil {
		return fmt.Errorf("failed to find checkpoints: %w", err)
	}

	if len(checkpoints) <= keepCount {
		return nil
	}

	// 按时间戳排序
	sort.Slice(checkpoints, func(i, j int) bool {
		return checkpoints[i].Timestamp > checkpoints[j].Timestamp
	})

	// 删除多余的检查点
	for i := keepCount; i < len(checkpoints); i++ {
		checkpoint := checkpoints[i]

		// 删除状态文件
		if err := os.Remove(checkpoint.FilePath); err != nil {
			fmt.Printf("Warning: failed to remove checkpoint file %s: %v\n", checkpoint.FilePath, err)
		}

		// 删除信息文件
		infoFileName := fmt.Sprintf("state_checkpoint_%s_%d.info", componentID, checkpoint.Timestamp)
		infoPath := filepath.Join(scm.checkpointDirectory, infoFileName)
		if err := os.Remove(infoPath); err != nil {
			fmt.Printf("Warning: failed to remove checkpoint info file %s: %v\n", infoPath, err)
		}
	}

	return nil
}

// WriteAheadLogManager 预写日志管理器
type WriteAheadLogManager struct {
	walDirectory string
	mu           sync.RWMutex
}

// StateUpdateRecord 状态更新记录
type StateUpdateRecord struct {
	Timestamp     int64                  `json:"timestamp"`
	ComponentID   string                 `json:"component_id"`
	PreviousState *statemanager.StateMap `json:"previous_state,omitempty"`
	NewState      *statemanager.StateMap `json:"new_state,omitempty"`
	Operation     string                 `json:"operation"`
}

// NewWriteAheadLogManager 创建预写日志管理器
func NewWriteAheadLogManager() *WriteAheadLogManager {
	return &WriteAheadLogManager{
		walDirectory: "./wal",
	}
}

// Initialize 初始化预写日志管理器
func (walm *WriteAheadLogManager) Initialize(walDir string) error {
	walm.mu.Lock()
	defer walm.mu.Unlock()

	if walDir != "" {
		walm.walDirectory = walDir
	}

	// 创建 WAL 目录
	if err := os.MkdirAll(walm.walDirectory, 0755); err != nil {
		return fmt.Errorf("failed to create WAL directory: %w", err)
	}

	return nil
}

// LogStateUpdate 记录状态更新
func (walm *WriteAheadLogManager) LogStateUpdate(componentID string, previousState, newState *statemanager.StateMap) error {
	walm.mu.Lock()
	defer walm.mu.Unlock()

	logFile := filepath.Join(walm.walDirectory, componentID+".wal")

	// 创建或打开日志文件
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open WAL file: %w", err)
	}
	defer file.Close()

	// 创建更新记录
	record := &StateUpdateRecord{
		Timestamp:   time.Now().UnixMilli(),
		ComponentID: componentID,
		Operation:   "UPDATE",
	}

	if previousState != nil {
		record.PreviousState = previousState
	}

	if newState != nil {
		record.NewState = newState
	}

	// 序列化记录
	recordData, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal update record: %w", err)
	}

	// 写入日志
	writer := bufio.NewWriter(file)
	if _, err := writer.Write(recordData); err != nil {
		return fmt.Errorf("failed to write to WAL: %w", err)
	}

	if _, err := writer.WriteString("\n"); err != nil {
		return fmt.Errorf("failed to write newline to WAL: %w", err)
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush WAL: %w", err)
	}

	// 确保数据写入磁盘
	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to sync WAL: %w", err)
	}

	return nil
}

// RecoverUpdates 恢复更新记录
func (walm *WriteAheadLogManager) RecoverUpdates(componentID string) ([]*StateUpdateRecord, error) {
	walm.mu.RLock()
	defer walm.mu.RUnlock()

	logFile := filepath.Join(walm.walDirectory, componentID+".wal")

	// 检查日志文件是否存在
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		return []*StateUpdateRecord{}, nil
	}

	// 读取日志文件
	file, err := os.Open(logFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open WAL file: %w", err)
	}
	defer file.Close()

	var recoveredUpdates []*StateUpdateRecord
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var record StateUpdateRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			fmt.Printf("Warning: failed to parse WAL record: %v\n", err)
			continue
		}

		recoveredUpdates = append(recoveredUpdates, &record)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read WAL file: %w", err)
	}

	return recoveredUpdates, nil
}

// TruncateLog 截断日志
func (walm *WriteAheadLogManager) TruncateLog(componentID string) error {
	walm.mu.Lock()
	defer walm.mu.Unlock()

	logFile := filepath.Join(walm.walDirectory, componentID+".wal")

	// 删除日志文件
	if err := os.Remove(logFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove WAL file: %w", err)
	}

	return nil
}

// GetLogSize 获取日志大小
func (walm *WriteAheadLogManager) GetLogSize(componentID string) (int64, error) {
	walm.mu.RLock()
	defer walm.mu.RUnlock()

	logFile := filepath.Join(walm.walDirectory, componentID+".wal")

	stat, err := os.Stat(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to stat WAL file: %w", err)
	}

	return stat.Size(), nil
}
