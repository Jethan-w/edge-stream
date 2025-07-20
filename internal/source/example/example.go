package example

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/edge-stream/internal/source"
	"github.com/edge-stream/internal/source/ConfigManager"
	"github.com/edge-stream/internal/source/MultiModalReceiv
	"github.com/edge-stream/internal/source/ProtocolAdapter"
r
)

// ExampleSourceUsage 展示Source模块的基本使用
func ExampleSourceUsage() {
	fmt.Println("=== Source模块使用示例 ===")

	// 创建上下文
	ctx := context.Background()

	// 1. 创建多模态数据接收器
	fmt.Println("\n1. 创建多模态数据接收器")
	multiModalReceiver := multimodalreceiver.NewMultiModalReceiver()

	// 初始化接收器
	if err := multiModalReceiver.Initialize(ctx); err != nil {
		log.Fatalf("初始化多模态接收器失败: %v", err)
	}

	// 启动接收器
	if err := multiModalReceiver.Start(ctx); err != nil {
		log.Fatalf("启动多模态接收器失败: %v", err)
	}

	// 接收不同类型的数据
	exampleReceiveData(multiModalReceiver)

	// 获取统计信息
	stats := multiModalReceiver.GetStatistics()
	fmt.Printf("接收统计: %+v\n", stats)

	// 停止接收器
	multiModalReceiver.Stop(ctx)

	// 2. 创建安全数据接收器
	fmt.Println("\n2. 创建安全数据接收器")
	secureReceiver := securereceiver.NewSecureReceiver()

	// 配置TLS
	tlsContext := &source.TLSContext{
		Protocol: "TLS",
		CertFile: "/path/to/cert.pem",
		KeyFile:  "/path/to/key.pem",
		CAFile:   "/path/to/ca.pem",
	}

	if err := secureReceiver.ConfigureTLS(tlsContext); err != nil {
		log.Printf("配置TLS失败: %v", err)
	}

	// 配置IP白名单
	allowedIPs := []string{"192.168.1.100", "192.168.1.101"}
	if err := secureReceiver.ConfigureIPWhitelist(allowedIPs); err != nil {
		log.Printf("配置IP白名单失败: %v", err)
	}

	// 初始化安全接收器
	if err := secureReceiver.Initialize(ctx); err != nil {
		log.Fatalf("初始化安全接收器失败: %v", err)
	}

	// 启动安全接收器
	if err := secureReceiver.Start(ctx); err != nil {
		log.Fatalf("启动安全接收器失败: %v", err)
	}

	// 停止安全接收器
	secureReceiver.Stop(ctx)

	// 3. 配置管理示例
	fmt.Println("\n3. 配置管理示例")
	exampleConfigManagement()

	// 4. 协议适配器示例
	fmt.Println("\n4. 协议适配器示例")
	exampleProtocolAdapters()

	// 5. 数据源管理器示例
	fmt.Println("\n5. 数据源管理器示例")
	exampleSourceManager()
}

// exampleReceiveData 示例接收数据
func exampleReceiveData(receiver *multimodalreceiver.MultiModalReceiver) {
	// 接收视频数据
	videoPacket := &source.VideoPacket{
		FrameData: []byte("video frame data"),
		Timestamp: time.Now(),
		Codec:     source.CodecH264,
		Resolution: source.Dimension{
			Width:  1920,
			Height: 1080,
		},
		FrameType: source.FrameTypeI,
		Metadata:  make(map[string]string),
	}

	if err := receiver.ReceiveVideo(videoPacket); err != nil {
		log.Printf("接收视频数据失败: %v", err)
	}

	// 接收文本数据
	textPacket := &source.TextPacket{
		Content:   "Hello, EdgeStream!",
		Encoding:  "UTF-8",
		Timestamp: time.Now(),
		Metadata:  make(map[string]string),
	}

	if err := receiver.ReceiveText(textPacket); err != nil {
		log.Printf("接收文本数据失败: %v", err)
	}

	// 接收二进制数据
	binaryPacket := &source.BinaryPacket{
		Data:      []byte{0x01, 0x02, 0x03, 0x04},
		Timestamp: time.Now(),
		Metadata:  make(map[string]string),
	}

	if err := receiver.ReceiveBinary(binaryPacket); err != nil {
		log.Printf("接收二进制数据失败: %v", err)
	}

	// 接收自定义数据
	customPacket := &source.CustomDataPacket{
		Data: map[string]interface{}{
			"sensor_id": "temp_001",
			"value":     25.6,
			"unit":      "celsius",
		},
		Type:      "sensor_data",
		Timestamp: time.Now(),
		Metadata:  make(map[string]string),
	}

	if err := receiver.ReceiveCustom(customPacket); err != nil {
		log.Printf("接收自定义数据失败: %v", err)
	}
}

// exampleConfigManagement 配置管理示例
func exampleConfigManagement() {
	// 创建配置管理器
	configManager := configmanager.NewConfigManager()

	// 初始化配置管理器
	ctx := context.Background()
	if err := configManager.Initialize(ctx); err != nil {
		log.Fatalf("初始化配置管理器失败: %v", err)
	}

	// 启动配置管理器
	if err := configManager.Start(ctx); err != nil {
		log.Fatalf("启动配置管理器失败: %v", err)
	}

	// 加载配置
	configData := map[string]string{
		"protocol":        "tcp",
		"host":            "0.0.0.0",
		"port":            "8080",
		"security.mode":   "tls",
		"max.connections": "100",
		"connect.timeout": "5000",
		"custom.property": "custom_value",
	}

	if err := configManager.LoadConfiguration("example_source", configData); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 获取配置
	config, exists := configManager.GetConfiguration("example_source")
	if exists {
		fmt.Printf("配置信息: %+v\n", config)
	}

	// 更新配置
	updateData := map[string]string{
		"port":            "8081",
		"max.connections": "200",
	}

	if err := configManager.UpdateConfiguration("example_source", updateData); err != nil {
		log.Printf("更新配置失败: %v", err)
	}

	// 验证配置
	result := configManager.ValidateConfiguration("example_source")
	if result.Valid {
		fmt.Println("配置验证通过")
	} else {
		fmt.Printf("配置验证失败: %v\n", result.Errors)
	}

	// 列出所有配置
	configs := configManager.ListConfigurations()
	fmt.Printf("所有配置: %v\n", configs)

	// 停止配置管理器
	configManager.Stop(ctx)
}

// exampleProtocolAdapters 协议适配器示例
func exampleProtocolAdapters() {
	// 创建协议管理器
	protocolManager := protocoladapter.NewProtocolManager()

	// 初始化协议管理器
	ctx := context.Background()
	if err := protocolManager.Initialize(ctx); err != nil {
		log.Fatalf("初始化协议管理器失败: %v", err)
	}

	// 创建TCP适配器
	tcpConfig := map[string]string{
		"host": "0.0.0.0",
		"port": "8080",
	}

	tcpAdapter, err := protocolManager.CreateAdapter("tcp", tcpConfig)
	if err != nil {
		log.Printf("创建TCP适配器失败: %v", err)
	} else {
		fmt.Printf("TCP适配器状态: %+v\n", tcpAdapter.GetStatus())

		// 接收数据（在实际应用中，这应该在协程中进行）
		// packet, err := tcpAdapter.Receive()
		// if err != nil {
		//     log.Printf("接收TCP数据失败: %v", err)
		// } else {
		//     fmt.Printf("接收到TCP数据: %+v\n", packet)
		// }

		// 关闭适配器
		tcpAdapter.Close()
	}

	// 创建UDP适配器
	udpConfig := map[string]string{
		"host": "0.0.0.0",
		"port": "8081",
	}

	udpAdapter, err := protocolManager.CreateAdapter("udp", udpConfig)
	if err != nil {
		log.Printf("创建UDP适配器失败: %v", err)
	} else {
		fmt.Printf("UDP适配器状态: %+v\n", udpAdapter.GetStatus())
		udpAdapter.Close()
	}

	// 创建Modbus适配器
	modbusConfig := map[string]string{
		"serial.port": "COM1",
		"baud.rate":   "9600",
	}

	modbusAdapter, err := protocolManager.CreateAdapter("modbus-rtu", modbusConfig)
	if err != nil {
		log.Printf("创建Modbus适配器失败: %v", err)
	} else {
		fmt.Printf("Modbus适配器状态: %+v\n", modbusAdapter.GetStatus())

		// 接收Modbus数据
		packet, err := modbusAdapter.Receive()
		if err != nil {
			log.Printf("接收Modbus数据失败: %v", err)
		} else {
			fmt.Printf("接收到Modbus数据: %+v\n", packet)
		}

		modbusAdapter.Close()
	}

	// 列出所有适配器
	adapters := protocolManager.ListAdapters()
	fmt.Printf("所有适配器: %v\n", adapters)
}

// exampleSourceManager 数据源管理器示例
func exampleSourceManager() {
	// 创建数据源管理器
	sourceManager := source.NewSourceManager()

	// 创建多模态数据接收器
	multiModalReceiver := multimodalreceiver.NewMultiModalReceiver()

	// 注册数据源
	if err := sourceManager.RegisterSource("multi_modal", multiModalReceiver); err != nil {
		log.Fatalf("注册数据源失败: %v", err)
	}

	// 创建安全数据接收器
	secureReceiver := securereceiver.NewSecureReceiver()

	// 注册数据源
	if err := sourceManager.RegisterSource("secure", secureReceiver); err != nil {
		log.Fatalf("注册数据源失败: %v", err)
	}

	// 列出所有数据源
	sources := sourceManager.ListSources()
	fmt.Printf("所有数据源: %v\n", sources)

	// 获取数据源
	if source, exists := sourceManager.GetSource("multi_modal"); exists {
		fmt.Printf("数据源状态: %v\n", source.GetState())
	}

	// 启动所有数据源
	ctx := context.Background()
	if err := sourceManager.StartAllSources(ctx); err != nil {
		log.Printf("启动所有数据源失败: %v", err)
	}

	// 等待一段时间
	time.Sleep(2 * time.Second)

	// 停止所有数据源
	if err := sourceManager.StopAllSources(ctx); err != nil {
		log.Printf("停止所有数据源失败: %v", err)
	}

	// 移除数据源
	if err := sourceManager.RemoveSource("multi_modal"); err != nil {
		log.Printf("移除数据源失败: %v", err)
	}
}

// ExampleCustomProtocol 自定义协议示例
func ExampleCustomProtocol() {
	fmt.Println("\n=== 自定义协议示例 ===")

	// 创建协议管理器
	protocolManager := protocoladapter.NewProtocolManager()

	// 注册自定义协议
	customExtension := &CustomProtocolExtension{}
	protocolManager.RegisterExtension(customExtension)

	// 创建自定义适配器
	config := map[string]string{
		"custom.param": "custom_value",
	}

	adapter, err := protocolManager.CreateAdapter("custom-protocol", config)
	if err != nil {
		log.Printf("创建自定义适配器失败: %v", err)
		return
	}

	fmt.Printf("自定义适配器状态: %+v\n", adapter.GetStatus())

	// 接收数据
	packet, err := adapter.Receive()
	if err != nil {
		log.Printf("接收自定义数据失败: %v", err)
	} else {
		fmt.Printf("接收到自定义数据: %+v\n", packet)
	}

	// 关闭适配器
	adapter.Close()
}

// CustomProtocolExtension 自定义协议扩展
type CustomProtocolExtension struct{}

// GetName 获取协议名称
func (cpe *CustomProtocolExtension) GetName() string {
	return "custom-protocol"
}

// Register 注册协议
func (cpe *CustomProtocolExtension) Register(registry *protocoladapter.CustomProtocolRegistry) {
	registry.RegisterProtocol(cpe.GetName(), cpe.CreateAdapter)
}

// CreateAdapter 创建适配器
func (cpe *CustomProtocolExtension) CreateAdapter() protocoladapter.ProtocolAdapter {
	return &CustomProtocolAdapter{
		AbstractProtocolAdapter: protocoladapter.NewAbstractProtocolAdapter("custom-protocol"),
	}
}

// CustomProtocolAdapter 自定义协议适配器
type CustomProtocolAdapter struct {
	*protocoladapter.AbstractProtocolAdapter
}

// Initialize 初始化自定义适配器
func (cpa *CustomProtocolAdapter) Initialize(config map[string]string) error {
	if err := cpa.AbstractProtocolAdapter.Initialize(config); err != nil {
		return err
	}

	cpa.UpdateStatus(protocoladapter.AdapterStateRunning, "自定义适配器已启动")
	return nil
}

// Receive 接收数据
func (cpa *CustomProtocolAdapter) Receive() (source.DataPacket, error) {
	// 模拟接收自定义数据
	packet := &source.CustomDataPacket{
		Data: map[string]interface{}{
			"custom_field": "custom_value",
			"timestamp":    time.Now().Unix(),
		},
		Type:      "custom",
		Timestamp: time.Now(),
		Metadata:  make(map[string]string),
	}

	packet.Metadata["protocol"] = "custom-protocol"
	packet.Metadata["source"] = "custom_adapter"

	cpa.LogAdapterActivity("接收到自定义数据")

	return packet, nil
}

// Close 关闭适配器
func (cpa *CustomProtocolAdapter) Close() error {
	cpa.UpdateStatus(protocoladapter.AdapterStateStopped, "自定义适配器已停止")
	return nil
}

// ExampleConfigurationListener 配置变更监听器示例
func ExampleConfigurationListener() {
	fmt.Println("\n=== 配置变更监听器示例 ===")

	// 创建配置管理器
	configManager := configmanager.NewConfigManager()

	// 创建配置变更监听器
	listener := &ExampleConfigChangeListener{}

	// 添加监听器
	configManager.configNotifier.AddListener("example_source", listener)

	// 加载配置
	configData := map[string]string{
		"protocol": "tcp",
		"host":     "0.0.0.0",
		"port":     "8080",
	}

	if err := configManager.LoadConfiguration("example_source", configData); err != nil {
		log.Printf("加载配置失败: %v", err)
	}

	// 更新配置
	updateData := map[string]string{
		"port": "8081",
	}

	if err := configManager.UpdateConfiguration("example_source", updateData); err != nil {
		log.Printf("更新配置失败: %v", err)
	}

	// 等待配置变更通知
	time.Sleep(1 * time.Second)
}

// ExampleConfigChangeListener 示例配置变更监听器
type ExampleConfigChangeListener struct{}

// OnConfigChange 配置变更回调
func (eccl *ExampleConfigChangeListener) OnConfigChange(event *configmanager.ConfigChangeEvent) {
	fmt.Printf("配置变更事件: 数据源=%s, 类型=%s, 时间=%s\n",
		event.SourceID, event.Type.String(), event.Time.Format(time.RFC3339))

	if event.Config != nil {
		fmt.Printf("配置信息: %+v\n", event.Config)
	}
}
