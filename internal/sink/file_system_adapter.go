package sink

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/crazy/edge-stream/internal/flowfile"
)

// FileSystemOutputAdapter 文件系统输出适配器
type FileSystemOutputAdapter struct {
	*AbstractTargetAdapter

	// 文件系统相关
	outputDirectory   string
	filePermissions   os.FileMode
	createDirectories bool
	overwriteExisting bool
	appendMode        bool
	fileExtension     string
	dateFormat        string
	compression       string
}

// NewFileSystemOutputAdapter 创建文件系统输出适配器
func NewFileSystemOutputAdapter(config map[string]string) *FileSystemOutputAdapter {
	adapter := &FileSystemOutputAdapter{
		AbstractTargetAdapter: NewAbstractTargetAdapter(
			config["id"],
			config["name"],
			"FileSystem",
		),
		filePermissions:   0644,
		createDirectories: true,
		overwriteExisting: false,
		appendMode:        false,
		dateFormat:        "2006-01-02",
		compression:       "none",
	}

	// 应用配置
	adapter.ApplyConfig(config)

	return adapter
}

// ApplyConfig 应用配置
func (a *FileSystemOutputAdapter) ApplyConfig(config map[string]string) {
	if outputDir, exists := config["output.directory"]; exists {
		a.outputDirectory = outputDir
	}

	if perms, exists := config["file.permissions"]; exists {
		if parsed, err := parseFilePermissions(perms); err == nil {
			a.filePermissions = parsed
		}
	}

	if createDirs, exists := config["create.directories"]; exists {
		a.createDirectories = strings.ToLower(createDirs) == "true"
	}

	if overwrite, exists := config["overwrite.existing"]; exists {
		a.overwriteExisting = strings.ToLower(overwrite) == "true"
	}

	if append, exists := config["append.mode"]; exists {
		a.appendMode = strings.ToLower(append) == "true"
	}

	if ext, exists := config["file.extension"]; exists {
		a.fileExtension = ext
	}

	if dateFmt, exists := config["date.format"]; exists {
		a.dateFormat = dateFmt
	}

	if comp, exists := config["compression"]; exists {
		a.compression = comp
	}
}

// GetRequiredConfig 获取必需的配置项
func (a *FileSystemOutputAdapter) GetRequiredConfig() []string {
	return []string{"output.directory"}
}

// doWrite 实现具体的文件写入逻辑
func (a *FileSystemOutputAdapter) doWrite(flowFile interface{}) error {
	// 验证 FlowFile
	if err := a.ValidateFlowFile(flowFile); err != nil {
		return err
	}

	ff := flowFile.(*flowfile.FlowFile)

	// 生成输出文件路径
	outputPath, err := a.generateOutputPath(ff)
	if err != nil {
		return fmt.Errorf("failed to generate output path: %w", err)
	}

	// 确保输出目录存在
	if a.createDirectories {
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// 检查文件是否已存在
	if !a.overwriteExisting {
		if _, err := os.Stat(outputPath); err == nil {
			return fmt.Errorf("file already exists and overwrite is disabled: %s", outputPath)
		}
	}

	// 确定文件打开模式
	flag := os.O_CREATE | os.O_WRONLY
	if a.appendMode {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}

	// 打开文件
	file, err := os.OpenFile(outputPath, flag, a.filePermissions)
	if err != nil {
		return fmt.Errorf("failed to open output file: %w", err)
	}
	defer file.Close()

	// 写入数据
	content := ff.Content

	// 根据压缩设置处理数据
	processedContent, err := a.processContent(content)
	if err != nil {
		return fmt.Errorf("failed to process content: %w", err)
	}

	// 写入文件
	written, err := file.Write(processedContent)
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	// 更新统计信息
	a.stats.mu.Lock()
	a.stats.TotalBytes += int64(written)
	a.stats.mu.Unlock()

	// 记录输出活动
	a.LogOutputActivity(fmt.Sprintf("Successfully wrote %d bytes to %s", written, outputPath))

	return nil
}

// generateOutputPath 生成输出文件路径
func (a *FileSystemOutputAdapter) generateOutputPath(ff *flowfile.FlowFile) (string, error) {
	// 获取文件名
	filename := ff.Attributes["filename"]
	if filename == "" {
		// 生成默认文件名
		timestamp := time.Now().Format("20060102_150405")
		filename = fmt.Sprintf("flowfile_%s", timestamp)
	}

	// 添加文件扩展名
	if a.fileExtension != "" && !strings.HasSuffix(filename, "."+a.fileExtension) {
		filename += "." + a.fileExtension
	}

	// 处理日期格式
	if strings.Contains(filename, "${date}") {
		dateStr := time.Now().Format(a.dateFormat)
		filename = strings.ReplaceAll(filename, "${date}", dateStr)
	}

	// 处理时间戳格式
	if strings.Contains(filename, "${timestamp}") {
		timestamp := time.Now().Format("20060102_150405")
		filename = strings.ReplaceAll(filename, "${timestamp}", timestamp)
	}

	// 处理 UUID 格式
	if strings.Contains(filename, "${uuid}") {
		uuid := ff.Attributes["uuid"]
		if uuid != "" {
			filename = strings.ReplaceAll(filename, "${uuid}", uuid)
		}
	}

	// 组合完整路径
	outputPath := filepath.Join(a.outputDirectory, filename)

	// 确保路径是绝对路径
	if !filepath.IsAbs(outputPath) {
		absPath, err := filepath.Abs(outputPath)
		if err != nil {
			return "", fmt.Errorf("failed to get absolute path: %w", err)
		}
		outputPath = absPath
	}

	return outputPath, nil
}

// processContent 处理内容（压缩等）
func (a *FileSystemOutputAdapter) processContent(content []byte) ([]byte, error) {
	switch strings.ToLower(a.compression) {
	case "none", "":
		return content, nil
	case "gzip":
		return a.compressGzip(content)
	case "zlib":
		return a.compressZlib(content)
	default:
		return nil, fmt.Errorf("unsupported compression: %s", a.compression)
	}
}

// compressGzip Gzip 压缩
func (a *FileSystemOutputAdapter) compressGzip(content []byte) ([]byte, error) {
	// 这里应该实现 Gzip 压缩
	// 为了简化，暂时返回原内容
	return content, nil
}

// compressZlib Zlib 压缩
func (a *FileSystemOutputAdapter) compressZlib(content []byte) ([]byte, error) {
	// 这里应该实现 Zlib 压缩
	// 为了简化，暂时返回原内容
	return content, nil
}

// performHealthCheck 执行健康检查
func (a *FileSystemOutputAdapter) performHealthCheck() bool {
	// 检查输出目录是否存在且可写
	if a.outputDirectory == "" {
		return false
	}

	// 检查目录是否存在
	if _, err := os.Stat(a.outputDirectory); os.IsNotExist(err) {
		// 如果目录不存在且允许创建，尝试创建
		if a.createDirectories {
			if err := os.MkdirAll(a.outputDirectory, 0755); err != nil {
				return false
			}
		} else {
			return false
		}
	}

	// 检查目录是否可写
	testFile := filepath.Join(a.outputDirectory, ".health_check")
	file, err := os.Create(testFile)
	if err != nil {
		return false
	}
	file.Close()
	os.Remove(testFile)

	return true
}

// GetOutputDirectory 获取输出目录
func (a *FileSystemOutputAdapter) GetOutputDirectory() string {
	return a.outputDirectory
}

// SetOutputDirectory 设置输出目录
func (a *FileSystemOutputAdapter) SetOutputDirectory(dir string) {
	a.outputDirectory = dir
	a.metadata.UpdatedAt = time.Now()
}

// parseFilePermissions 解析文件权限
func parseFilePermissions(perms string) (os.FileMode, error) {
	// 支持八进制格式，如 "0644"
	if len(perms) == 4 && perms[0] == '0' {
		var mode uint32
		_, err := fmt.Sscanf(perms, "%o", &mode)
		if err != nil {
			return 0, err
		}
		return os.FileMode(mode), nil
	}

	// 支持十进制格式
	var mode uint32
	_, err := fmt.Sscanf(perms, "%d", &mode)
	if err != nil {
		return 0, err
	}
	return os.FileMode(mode), nil
}

// CopyFile 复制文件（用于备份等场景）
func (a *FileSystemOutputAdapter) CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// GetFileInfo 获取文件信息
func (a *FileSystemOutputAdapter) GetFileInfo(filepath string) (os.FileInfo, error) {
	return os.Stat(filepath)
}
