package NARPackageManager

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Bundle NAR包结构
type Bundle struct {
	ID           string            `json:"id"`
	Group        string            `json:"group"`
	Artifact     string            `json:"artifact"`
	Version      string            `json:"version"`
	File         string            `json:"file"`
	Dependencies []*Dependency     `json:"dependencies"`
	Metadata     map[string]string `json:"metadata"`
}

// Dependency 依赖结构
type Dependency struct {
	Group    string `json:"group"`
	Artifact string `json:"artifact"`
	Version  string `json:"version"`
	Scope    string `json:"scope"` // compile, runtime, test
}

// BundleRepository Bundle仓库接口
type BundleRepository interface {
	// DeployBundle 部署 NAR 包
	DeployBundle(bundle *Bundle) error

	// UndeployBundle 卸载 NAR 包
	UndeployBundle(bundleID, version string) error

	// GetBundle 获取特定版本的 NAR 包
	GetBundle(bundleID, version string) (*Bundle, error)

	// ListBundles 列出所有已部署的 NAR 包
	ListBundles() ([]*Bundle, error)

	// GetBundlePath 获取Bundle路径
	GetBundlePath(bundleID, version string) (string, error)
}

// FileSystemBundleRepository 文件系统 NAR 包仓库实现
type FileSystemBundleRepository struct {
	repositoryPath string
	bundles        map[string]*Bundle
	mutex          sync.RWMutex
}

// NewFileSystemBundleRepository 创建新的文件系统Bundle仓库
func NewFileSystemBundleRepository() *FileSystemBundleRepository {
	return &FileSystemBundleRepository{
		repositoryPath: "bundles",
		bundles:        make(map[string]*Bundle),
	}
}

// SetRepositoryPath 设置仓库路径
func (fsbr *FileSystemBundleRepository) SetRepositoryPath(path string) {
	fsbr.mutex.Lock()
	defer fsbr.mutex.Unlock()

	fsbr.repositoryPath = path
}

// DeployBundle 部署 NAR 包
func (fsbr *FileSystemBundleRepository) DeployBundle(bundle *Bundle) error {
	fsbr.mutex.Lock()
	defer fsbr.mutex.Unlock()

	// 创建Bundle路径
	bundlePath := filepath.Join(fsbr.repositoryPath, bundle.Group, bundle.Artifact, bundle.Version)
	if err := os.MkdirAll(bundlePath, 0755); err != nil {
		return fmt.Errorf("创建Bundle目录失败: %w", err)
	}

	// 解压 NAR 包
	if err := fsbr.unzipNarBundle(bundle.File, bundlePath); err != nil {
		return fmt.Errorf("解压NAR包失败: %w", err)
	}

	// 生成元数据文件
	if err := fsbr.generateBundleMetadata(bundle, bundlePath); err != nil {
		return fmt.Errorf("生成Bundle元数据失败: %w", err)
	}

	// 验证Bundle结构
	if err := fsbr.validateBundleStructure(bundlePath); err != nil {
		return fmt.Errorf("验证Bundle结构失败: %w", err)
	}

	// 注册Bundle
	fsbr.bundles[bundle.ID] = bundle

	return nil
}

// UndeployBundle 卸载 NAR 包
func (fsbr *FileSystemBundleRepository) UndeployBundle(bundleID, version string) error {
	fsbr.mutex.Lock()
	defer fsbr.mutex.Unlock()

	// 查找Bundle
	bundle, exists := fsbr.bundles[bundleID]
	if !exists {
		return fmt.Errorf("Bundle不存在: %s", bundleID)
	}

	// 删除Bundle目录
	bundlePath := filepath.Join(fsbr.repositoryPath, bundle.Group, bundle.Artifact, bundle.Version)
	if err := os.RemoveAll(bundlePath); err != nil {
		return fmt.Errorf("删除Bundle目录失败: %w", err)
	}

	// 从注册表中移除
	delete(fsbr.bundles, bundleID)

	return nil
}

// GetBundle 获取特定版本的 NAR 包
func (fsbr *FileSystemBundleRepository) GetBundle(bundleID, version string) (*Bundle, error) {
	fsbr.mutex.RLock()
	defer fsbr.mutex.RUnlock()

	bundle, exists := fsbr.bundles[bundleID]
	if !exists {
		return nil, fmt.Errorf("Bundle不存在: %s", bundleID)
	}

	if bundle.Version != version {
		return nil, fmt.Errorf("Bundle版本不匹配: 期望 %s, 实际 %s", version, bundle.Version)
	}

	return bundle, nil
}

// ListBundles 列出所有已部署的 NAR 包
func (fsbr *FileSystemBundleRepository) ListBundles() ([]*Bundle, error) {
	fsbr.mutex.RLock()
	defer fsbr.mutex.RUnlock()

	bundles := make([]*Bundle, 0, len(fsbr.bundles))
	for _, bundle := range fsbr.bundles {
		bundles = append(bundles, bundle)
	}

	return bundles, nil
}

// GetBundlePath 获取Bundle路径
func (fsbr *FileSystemBundleRepository) GetBundlePath(bundleID, version string) (string, error) {
	fsbr.mutex.RLock()
	defer fsbr.mutex.RUnlock()

	bundle, exists := fsbr.bundles[bundleID]
	if !exists {
		return "", fmt.Errorf("Bundle不存在: %s", bundleID)
	}

	if bundle.Version != version {
		return "", fmt.Errorf("Bundle版本不匹配: 期望 %s, 实际 %s", version, bundle.Version)
	}

	return filepath.Join(fsbr.repositoryPath, bundle.Group, bundle.Artifact, bundle.Version), nil
}

// unzipNarBundle 解压 NAR 包
func (fsbr *FileSystemBundleRepository) unzipNarBundle(narFile, targetPath string) error {
	reader, err := zip.OpenReader(narFile)
	if err != nil {
		return fmt.Errorf("打开NAR文件失败: %w", err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		filePath := filepath.Join(targetPath, file.Name)

		// 创建目录
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(filePath, file.Mode()); err != nil {
				return fmt.Errorf("创建目录失败: %w", err)
			}
			continue
		}

		// 创建文件
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return fmt.Errorf("创建文件目录失败: %w", err)
		}

		fileReader, err := file.Open()
		if err != nil {
			return fmt.Errorf("打开文件失败: %w", err)
		}

		fileWriter, err := os.Create(filePath)
		if err != nil {
			fileReader.Close()
			return fmt.Errorf("创建文件失败: %w", err)
		}

		if _, err := io.Copy(fileWriter, fileReader); err != nil {
			fileReader.Close()
			fileWriter.Close()
			return fmt.Errorf("复制文件失败: %w", err)
		}

		fileReader.Close()
		fileWriter.Close()
	}

	return nil
}

// generateBundleMetadata 生成Bundle元数据
func (fsbr *FileSystemBundleRepository) generateBundleMetadata(bundle *Bundle, bundlePath string) error {
	metadataFile := filepath.Join(bundlePath, "bundle.json")

	metadata := map[string]interface{}{
		"id":           bundle.ID,
		"group":        bundle.Group,
		"artifact":     bundle.Artifact,
		"version":      bundle.Version,
		"dependencies": bundle.Dependencies,
		"metadata":     bundle.Metadata,
		"deployedAt":   time.Now().Format(time.RFC3339),
	}

	metadataBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %w", err)
	}

	if err := os.WriteFile(metadataFile, metadataBytes, 0644); err != nil {
		return fmt.Errorf("写入元数据文件失败: %w", err)
	}

	return nil
}

// validateBundleStructure 验证Bundle结构
func (fsbr *FileSystemBundleRepository) validateBundleStructure(bundlePath string) error {
	requiredDirs := []string{
		"META-INF/services",
		"lib",
		"classes",
	}

	for _, dir := range requiredDirs {
		dirPath := filepath.Join(bundlePath, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			return fmt.Errorf("缺少必需目录: %s", dir)
		}
	}

	return nil
}

// BundleDependencyResolver Bundle依赖解析器
type BundleDependencyResolver struct {
	bundleRepository BundleRepository
	mutex            sync.RWMutex
}

// NewBundleDependencyResolver 创建新的Bundle依赖解析器
func NewBundleDependencyResolver(repository BundleRepository) *BundleDependencyResolver {
	return &BundleDependencyResolver{
		bundleRepository: repository,
	}
}

// ResolveDependencies 解析依赖
func (bdr *BundleDependencyResolver) ResolveDependencies(bundle *Bundle) error {
	bdr.mutex.Lock()
	defer bdr.mutex.Unlock()

	for _, dependency := range bundle.Dependencies {
		// 检查依赖是否已部署
		if !bdr.isBundleDeployed(dependency) {
			// 自动下载并部署依赖
			if err := bdr.downloadAndDeployDependency(dependency); err != nil {
				return fmt.Errorf("下载并部署依赖失败: %w", err)
			}
		}

		// 验证依赖版本兼容性
		if err := bdr.validateDependencyVersion(dependency); err != nil {
			return fmt.Errorf("验证依赖版本失败: %w", err)
		}
	}

	return nil
}

// isBundleDeployed 检查Bundle是否已部署
func (bdr *BundleDependencyResolver) isBundleDeployed(dependency *Dependency) bool {
	bundleID := fmt.Sprintf("%s:%s", dependency.Group, dependency.Artifact)
	_, err := bdr.bundleRepository.GetBundle(bundleID, dependency.Version)
	return err == nil
}

// downloadAndDeployDependency 下载并部署依赖
func (bdr *BundleDependencyResolver) downloadAndDeployDependency(dependency *Dependency) error {
	// 这里简化实现，实际应该从远程仓库下载
	fmt.Printf("下载依赖: %s:%s:%s\n", dependency.Group, dependency.Artifact, dependency.Version)

	// 创建虚拟Bundle（实际应该下载）
	bundle := &Bundle{
		ID:       fmt.Sprintf("%s:%s", dependency.Group, dependency.Artifact),
		Group:    dependency.Group,
		Artifact: dependency.Artifact,
		Version:  dependency.Version,
		File:     fmt.Sprintf("%s-%s.nar", dependency.Artifact, dependency.Version),
	}

	return bdr.bundleRepository.DeployBundle(bundle)
}

// validateDependencyVersion 验证依赖版本兼容性
func (bdr *BundleDependencyResolver) validateDependencyVersion(dependency *Dependency) error {
	bundleID := fmt.Sprintf("%s:%s", dependency.Group, dependency.Artifact)
	existingBundle, err := bdr.bundleRepository.GetBundle(bundleID, dependency.Version)
	if err != nil {
		return fmt.Errorf("获取现有Bundle失败: %w", err)
	}

	if !bdr.isVersionCompatible(existingBundle.Version, dependency.Version) {
		return fmt.Errorf("版本不兼容：%s vs %s", existingBundle.Version, dependency.Version)
	}

	return nil
}

// isVersionCompatible 检查版本兼容性
func (bdr *BundleDependencyResolver) isVersionCompatible(currentVersion, newVersion string) bool {
	// 简化的版本兼容性检查
	// 实际应该使用语义化版本比较
	return currentVersion == newVersion
}

// NarBundleStructure NAR包结构定义
type NarBundleStructure struct {
	RequiredDirectories []string
	MetadataFiles       []string
}

// NewNarBundleStructure 创建新的NAR包结构
func NewNarBundleStructure() *NarBundleStructure {
	return &NarBundleStructure{
		RequiredDirectories: []string{
			"META-INF/services",
			"lib",
			"classes",
		},
		MetadataFiles: []string{
			"bundle.json",
			"MANIFEST.MF",
			"dependencies.properties",
		},
	}
}

// ValidateStructure 验证NAR包结构
func (nbs *NarBundleStructure) ValidateStructure(bundlePath string) error {
	// 验证必需目录
	for _, dir := range nbs.RequiredDirectories {
		dirPath := filepath.Join(bundlePath, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			return fmt.Errorf("缺少必需目录: %s", dir)
		}
	}

	// 验证元数据文件（可选）
	for _, file := range nbs.MetadataFiles {
		filePath := filepath.Join(bundlePath, file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			// 元数据文件不是必需的，只记录警告
			fmt.Printf("警告: 缺少元数据文件: %s\n", file)
		}
	}

	return nil
}
