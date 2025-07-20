package example

import (
	"fmt"
	"log"

	"github.com/crazy/edge-stream/internal/ConnectorRegistry"
	"github.com/crazy/edge-stream/internal/ConnectorRegistry/ComponentRegistrar"
	"github.com/crazy/edge-stream/internal/ConnectorRegistry/NARPackageManager"
	"github.com/crazy/edge-stream/internal/ConnectorRegistry/SecurityValidator"
	"github.com/crazy/edge-stream/internal/ConnectorRegistry/VersionController"
)

// ExampleUsage 使用示例
func ExampleUsage() {
	// 创建连接器注册表
	registry := ConnectorRegistry.NewStandardConnectorRegistry()

	// 创建Bundle
	bundle := &NARPackageManager.Bundle{
		ID:       "org.apache.nifi:nifi-standard-nar",
		Group:    "org.apache.nifi",
		Artifact: "nifi-standard-nar",
		Version:  "1.20.0",
		File:     "nifi-standard-nar-1.20.0.nar",
		Dependencies: []*NARPackageManager.Dependency{
			{
				Group:    "org.apache.nifi",
				Artifact: "nifi-api-nar",
				Version:  "1.20.0",
				Scope:    "compile",
			},
		},
		Metadata: map[string]string{
			"description": "NiFi Standard Processors",
			"author":      "Apache NiFi Team",
		},
	}

	// 创建连接器描述符
	descriptor := ConnectorRegistry.NewNarBundleConnectorDescriptor(bundle, "NiFi Standard Processors", ConnectorRegistry.ComponentTypeProcessor)
	descriptor.SetDescription("Apache NiFi 标准处理器集合")
	descriptor.SetAuthor("Apache NiFi Team")
	descriptor.AddTag("processor")
	descriptor.AddTag("standard")
	descriptor.SetMetadata("category", "data-processing")

	// 注册连接器
	err := registry.RegisterConnector(descriptor)
	if err != nil {
		log.Printf("注册连接器失败: %v", err)
		return
	}

	// 列出所有连接器
	connectors := registry.ListConnectors()
	fmt.Printf("已注册连接器数量: %d\n", len(connectors))

	for _, connector := range connectors {
		fmt.Printf("连接器: %s (%s)\n", connector.GetName(), connector.GetIdentifier())
	}

	// 根据类型列出连接器
	processors := registry.ListConnectorsByType(ConnectorRegistry.ComponentTypeProcessor)
	fmt.Printf("处理器数量: %d\n", len(processors))

	// 搜索连接器
	results := registry.SearchConnectors("standard")
	fmt.Printf("搜索结果数量: %d\n", len(results))

	// 获取连接器版本
	versions, err := registry.GetConnectorVersions(descriptor.GetIdentifier())
	if err != nil {
		log.Printf("获取版本失败: %v", err)
	} else {
		fmt.Printf("版本数量: %d\n", len(versions))
	}
}

// ExampleNARPackageManagement NAR包管理示例
func ExampleNARPackageManagement() {
	// 创建Bundle仓库
	repository := NARPackageManager.NewFileSystemBundleRepository()
	repository.SetRepositoryPath("custom-bundles")

	// 创建Bundle
	bundle := &NARPackageManager.Bundle{
		ID:       "com.example:custom-processor-nar",
		Group:    "com.example",
		Artifact: "custom-processor-nar",
		Version:  "1.0.0",
		File:     "custom-processor-nar-1.0.0.nar",
		Metadata: map[string]string{
			"description": "自定义处理器",
			"author":      "Example Team",
		},
	}

	// 部署Bundle
	err := repository.DeployBundle(bundle)
	if err != nil {
		log.Printf("部署Bundle失败: %v", err)
		return
	}

	// 列出所有Bundle
	bundles, err := repository.ListBundles()
	if err != nil {
		log.Printf("列出Bundle失败: %v", err)
		return
	}

	fmt.Printf("已部署Bundle数量: %d\n", len(bundles))
	for _, b := range bundles {
		fmt.Printf("Bundle: %s:%s:%s\n", b.Group, b.Artifact, b.Version)
	}

	// 创建依赖解析器
	resolver := NARPackageManager.NewBundleDependencyResolver(repository)

	// 解析依赖
	err = resolver.ResolveDependencies(bundle)
	if err != nil {
		log.Printf("解析依赖失败: %v", err)
	}
}

// ExampleVersionControl 版本控制示例
func ExampleVersionControl() {
	// 创建版本控制器
	controller := VersionController.NewStandardVersionController()

	// 创建版本管理器
	manager := VersionController.NewVersionManager(controller)

	// 设置版本控制策略
	manager.SetStrategy(VersionController.VersionControlStrategySemantic)

	// 创建流程快照数据
	snapshotData := map[string]interface{}{
		"processors": []map[string]interface{}{
			{
				"id":   "processor-1",
				"type": "org.apache.nifi.processors.standard.GenerateFlowFile",
				"properties": map[string]string{
					"File Size":  "1KB",
					"Batch Size": "1",
				},
			},
		},
		"connections": []map[string]interface{}{
			{
				"id":           "connection-1",
				"source":       "processor-1",
				"destination":  "processor-2",
				"relationship": "success",
			},
		},
	}

	// 创建版本
	err := manager.CreateVersionWithStrategy(
		"flow-1",
		snapshotData,
		"admin",
		"初始版本",
	)
	if err != nil {
		log.Printf("创建版本失败: %v", err)
		return
	}

	// 列出版本
	versions, err := controller.ListVersions("flow-1")
	if err != nil {
		log.Printf("列出版本失败: %v", err)
		return
	}

	fmt.Printf("版本数量: %d\n", len(versions))
	for _, version := range versions {
		fmt.Printf("版本: %s\n", version)
	}

	// 获取最新版本
	latestVersion, err := controller.GetLatestVersion("flow-1")
	if err != nil {
		log.Printf("获取最新版本失败: %v", err)
		return
	}

	fmt.Printf("最新版本: %s, 创建时间: %s\n",
		latestVersion.Version,
		latestVersion.CreatedAt.Format("2006-01-02 15:04:05"))

	// 验证版本兼容性
	err = manager.ValidateVersionCompatibility("flow-1", "1.1.0")
	if err != nil {
		log.Printf("版本兼容性验证失败: %v", err)
	}
}

// ExampleExtensionMapping 扩展映射示例
func ExampleExtensionMapping() {
	// 创建扩展注册表
	extensionRegistry := ComponentRegistrar.NewExtensionRegistry()

	// 注册扩展
	err := extensionRegistry.RegisterExtension(
		"PROCESSOR",
		"org.apache.nifi:nifi-standard-nar",
		"org.apache.nifi.processors.standard.GenerateFlowFile",
	)
	if err != nil {
		log.Printf("注册扩展失败: %v", err)
		return
	}

	// 获取扩展
	extensions, err := extensionRegistry.GetExtensions("PROCESSOR")
	if err != nil {
		log.Printf("获取扩展失败: %v", err)
		return
	}

	fmt.Printf("处理器扩展数量: %d\n", len(extensions))
	for _, ext := range extensions {
		fmt.Printf("扩展: %s -> %s\n", ext.BundleID, ext.ClassName)
	}

	// 加载扩展
	err = extensionRegistry.LoadExtensions()
	if err != nil {
		log.Printf("加载扩展失败: %v", err)
	}
}

// ExampleSecurityValidation 安全验证示例
func ExampleSecurityValidation() {
	// 创建安全管理器
	securityManager := SecurityValidator.NewSecurityManager()

	// 创建安全策略
	policy := SecurityValidator.NewSecurityPolicy()
	policy.AddAllowedRepository("github.com/apache/nifi")
	policy.AddBlockedComponent("malicious-component")

	// 设置安全策略
	securityManager.SetSecurityPolicy(policy)

	// 创建模拟Bundle
	bundle := &NARPackageManager.Bundle{
		ID:       "org.apache.nifi:nifi-standard-nar",
		Group:    "org.apache.nifi",
		Artifact: "nifi-standard-nar",
		Version:  "1.20.0",
		File:     "nifi-standard-nar-1.20.0.nar",
	}

	// 使用策略验证组件
	err := securityManager.ValidateComponentWithPolicy(bundle, "github.com/apache/nifi")
	if err != nil {
		log.Printf("组件验证失败: %v", err)
		return
	}

	fmt.Println("组件验证通过")

	// 创建社区组件注册表
	communityRegistry := SecurityValidator.NewCommunityComponentRegistry()

	// 添加GitHub仓库
	githubRepo := SecurityValidator.NewGitHubComponentRepository(
		"https://api.github.com",
		"your-github-token",
	)
	communityRegistry.AddRepository(githubRepo)

	// 搜索组件
	results, err := communityRegistry.SearchComponents("processor")
	if err != nil {
		log.Printf("搜索组件失败: %v", err)
		return
	}

	fmt.Printf("搜索结果数量: %d\n", len(results))
}

// ExampleComponentDiscovery 组件发现示例
func ExampleComponentDiscovery() {
	// 创建扩展映射提供者
	mappingProvider := ComponentRegistrar.NewJsonExtensionMappingProvider()

	// 创建动态组件发现器
	discovery := ComponentRegistrar.NewDynamicComponentDiscovery(mappingProvider)

	// 注册扩展映射
	err := mappingProvider.RegisterExtension(
		"PROCESSOR",
		"org.apache.nifi:nifi-standard-nar",
		"org.apache.nifi.processors.standard.GenerateFlowFile",
	)
	if err != nil {
		log.Printf("注册扩展映射失败: %v", err)
		return
	}

	// 发现组件
	components, err := discovery.DiscoverComponents("org.apache.nifi:nifi-standard-nar")
	if err != nil {
		log.Printf("发现组件失败: %v", err)
		return
	}

	fmt.Printf("发现的组件: %v\n", components)

	// 根据类型发现组件
	processors, err := discovery.DiscoverComponentsByType("PROCESSOR")
	if err != nil {
		log.Printf("根据类型发现组件失败: %v", err)
		return
	}

	fmt.Printf("发现的处理器: %v\n", processors)
}

// ExampleBundleStructureValidation Bundle结构验证示例
func ExampleBundleStructureValidation() {
	// 创建NAR包结构验证器
	structure := NARPackageManager.NewNarBundleStructure()

	// 验证Bundle结构
	err := structure.ValidateStructure("bundles/org.apache.nifi/nifi-standard-nar/1.20.0")
	if err != nil {
		log.Printf("Bundle结构验证失败: %v", err)
		return
	}

	fmt.Println("Bundle结构验证通过")
}

// ExampleVersionCompatibility 版本兼容性示例
func ExampleVersionCompatibility() {
	// 创建版本兼容性检查器
	checker := VersionController.NewVersionCompatibilityChecker()

	// 检查版本兼容性
	isCompatible := checker.IsCompatible("1.0.0", "1.1.0")
	fmt.Printf("版本兼容性: %t\n", isCompatible)

	isCompatible = checker.IsCompatible("1.0.0", "2.0.0")
	fmt.Printf("版本兼容性: %t\n", isCompatible)
}

// ExamplePGPSignatureVerification PGP签名验证示例
func ExamplePGPSignatureVerification() {
	// 创建PGP签名验证器
	verifier := SecurityValidator.NewPGPSignatureVerifier()

	// 加载公钥环
	err := verifier.LoadPublicKeyRing("keys/public-keyring.gpg")
	if err != nil {
		log.Printf("加载公钥环失败: %v", err)
		return
	}

	// 验证文件签名
	err = verifier.VerifyFile("bundle.nar", "bundle.nar.sig")
	if err != nil {
		log.Printf("验证文件签名失败: %v", err)
		return
	}

	fmt.Println("文件签名验证通过")
}

// ExampleCommunityComponentIntegration 社区组件集成示例
func ExampleCommunityComponentIntegration() {
	// 创建社区组件注册表
	communityRegistry := SecurityValidator.NewCommunityComponentRegistry()

	// 添加多个仓库
	githubRepo := SecurityValidator.NewGitHubComponentRepository(
		"https://api.github.com",
		"your-github-token",
	)
	communityRegistry.AddRepository(githubRepo)

	// 搜索组件
	results, err := communityRegistry.SearchComponents("kafka")
	if err != nil {
		log.Printf("搜索组件失败: %v", err)
		return
	}

	fmt.Printf("找到 %d 个Kafka相关组件\n", len(results))

	// 下载并安装组件（如果有结果）
	if len(results) > 0 {
		err = communityRegistry.DownloadAndInstallComponent(results[0])
		if err != nil {
			log.Printf("下载并安装组件失败: %v", err)
			return
		}

		fmt.Println("组件下载并安装成功")
	}
}
