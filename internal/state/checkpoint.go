package state

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// FileCheckpointManager 文件检查点管理器
type FileCheckpointManager struct {
	basePath string
}

// NewFileCheckpointManager 创建文件检查点管理器
func NewFileCheckpointManager(basePath string) *FileCheckpointManager {
	return &FileCheckpointManager{
		basePath: basePath,
	}
}

// Save 保存检查点
func (fcm *FileCheckpointManager) Save(ctx context.Context, checkpoint *Checkpoint) error {
	// 确保目录存在
	if err := os.MkdirAll(fcm.basePath, 0755); err != nil {
		return fmt.Errorf("failed to create checkpoint directory: %w", err)
	}

	// 序列化检查点
	data, err := json.MarshalIndent(checkpoint, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	// 生成文件名
	filename := fmt.Sprintf("checkpoint_%s_%d.json",
		checkpoint.ID, checkpoint.Timestamp.Unix())
	filePath := filepath.Join(fcm.basePath, filename)

	// 写入文件
	if err := ioutil.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write checkpoint file: %w", err)
	}

	return nil
}

// Load 加载检查点
func (fcm *FileCheckpointManager) Load(ctx context.Context, checkpointID string) (*Checkpoint, error) {
	// 查找检查点文件
	files, err := ioutil.ReadDir(fcm.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read checkpoint directory: %w", err)
	}

	var targetFile string
	for _, file := range files {
		if strings.Contains(file.Name(), checkpointID) && strings.HasSuffix(file.Name(), ".json") {
			targetFile = file.Name()
			break
		}
	}

	if targetFile == "" {
		return nil, fmt.Errorf("checkpoint %s not found", checkpointID)
	}

	// 读取文件
	filePath := filepath.Join(fcm.basePath, targetFile)
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read checkpoint file: %w", err)
	}

	// 反序列化检查点
	var checkpoint Checkpoint
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		return nil, fmt.Errorf("failed to unmarshal checkpoint: %w", err)
	}

	return &checkpoint, nil
}

// List 列出检查点
func (fcm *FileCheckpointManager) List(ctx context.Context) ([]*CheckpointInfo, error) {
	files, err := ioutil.ReadDir(fcm.basePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*CheckpointInfo{}, nil
		}
		return nil, fmt.Errorf("failed to read checkpoint directory: %w", err)
	}

	var checkpoints []*CheckpointInfo
	for _, file := range files {
		if !strings.HasPrefix(file.Name(), "checkpoint_") || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		// 解析文件名获取ID和时间戳
		parts := strings.Split(strings.TrimSuffix(file.Name(), ".json"), "_")
		if len(parts) < 3 {
			continue
		}

		checkpointID := parts[1]

		// 读取文件获取详细信息
		filePath := filepath.Join(fcm.basePath, file.Name())
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			continue
		}

		var checkpoint Checkpoint
		if err := json.Unmarshal(data, &checkpoint); err != nil {
			continue
		}

		checkpointInfo := &CheckpointInfo{
			ID:        checkpointID,
			Timestamp: checkpoint.Timestamp,
			Size:      file.Size(),
			States:    len(checkpoint.States),
		}

		checkpoints = append(checkpoints, checkpointInfo)
	}

	// 按时间戳排序（最新的在前）
	sort.Slice(checkpoints, func(i, j int) bool {
		return checkpoints[i].Timestamp.After(checkpoints[j].Timestamp)
	})

	return checkpoints, nil
}

// Delete 删除检查点
func (fcm *FileCheckpointManager) Delete(ctx context.Context, checkpointID string) error {
	files, err := ioutil.ReadDir(fcm.basePath)
	if err != nil {
		return fmt.Errorf("failed to read checkpoint directory: %w", err)
	}

	var deleted bool
	for _, file := range files {
		if strings.Contains(file.Name(), checkpointID) && strings.HasSuffix(file.Name(), ".json") {
			filePath := filepath.Join(fcm.basePath, file.Name())
			if err := os.Remove(filePath); err != nil {
				return fmt.Errorf("failed to delete checkpoint file: %w", err)
			}
			deleted = true
			break
		}
	}

	if !deleted {
		return fmt.Errorf("checkpoint %s not found", checkpointID)
	}

	return nil
}

// Cleanup 清理过期检查点
func (fcm *FileCheckpointManager) Cleanup(ctx context.Context, retentionPeriod time.Duration) error {
	checkpoints, err := fcm.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list checkpoints: %w", err)
	}

	cutoffTime := time.Now().Add(-retentionPeriod)
	var deletedCount int

	for _, checkpoint := range checkpoints {
		if checkpoint.Timestamp.Before(cutoffTime) {
			if err := fcm.Delete(ctx, checkpoint.ID); err != nil {
				fmt.Printf("Failed to delete expired checkpoint %s: %v\n", checkpoint.ID, err)
			} else {
				deletedCount++
			}
		}
	}

	fmt.Printf("Cleaned up %d expired checkpoints\n", deletedCount)
	return nil
}

// GetLatest 获取最新的检查点
func (fcm *FileCheckpointManager) GetLatest(ctx context.Context) (*Checkpoint, error) {
	checkpoints, err := fcm.List(ctx)
	if err != nil {
		return nil, err
	}

	if len(checkpoints) == 0 {
		return nil, fmt.Errorf("no checkpoints found")
	}

	// 列表已经按时间排序，第一个就是最新的
	return fcm.Load(ctx, checkpoints[0].ID)
}

// GetSize 获取检查点目录大小
func (fcm *FileCheckpointManager) GetSize() (int64, error) {
	var totalSize int64

	err := filepath.Walk(fcm.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	return totalSize, err
}

// Validate 验证检查点文件完整性
func (fcm *FileCheckpointManager) Validate(ctx context.Context, checkpointID string) error {
	checkpoint, err := fcm.Load(ctx, checkpointID)
	if err != nil {
		return fmt.Errorf("failed to load checkpoint: %w", err)
	}

	// 基本验证
	if checkpoint.ID == "" {
		return fmt.Errorf("checkpoint ID is empty")
	}

	if checkpoint.Timestamp.IsZero() {
		return fmt.Errorf("checkpoint timestamp is zero")
	}

	if checkpoint.States == nil {
		return fmt.Errorf("checkpoint states is nil")
	}

	// 验证状态数据
	for stateName, stateData := range checkpoint.States {
		if stateName == "" {
			return fmt.Errorf("empty state name found")
		}
		if stateData == nil {
			return fmt.Errorf("state data for %s is nil", stateName)
		}
	}

	return nil
}
