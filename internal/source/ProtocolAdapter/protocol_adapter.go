package protocoladapter

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/edge-stream/internal/source"
)

// ProtocolAdapter 协议适配器接口
type ProtocolAdapter interface {
	Initialize(config map[string]string) error
	Receive() (source.DataPacket, error)
	Close() error
	GetProtocolName() string
	GetStatus() *AdapterStatus
}

// AbstractProtocolAdapter 抽象协议适配器
type AbstractProtocolAdapter struct {
	config       map[string]string
	protocolName string
	status       *AdapterStatus
	mu           sync.RWMutex
}

// NewAbstractProtocolAdapter 创建抽象协议适配器
func NewAbstractProtocolAdapter(protocolName string) *AbstractProtocolAdapter {
	return &AbstractProtocolAdapter{
		config:       make(map[string]string),
		protocolName: protocolName,
		status:       &AdapterStatus{},
	}
}

// Initialize 初始化适配器
func (apa *AbstractProtocolAdapter) Initialize(config map[string]string) error {
	apa.mu.Lock()
	defer apa.mu.Unlock()

	apa.config = config
	apa.status.State = AdapterStateInitialized
	apa.status.LastActivity = time.Now()

	return nil
}

// GetConfig 获取配置
func (apa *AbstractProtocolAdapter) GetConfig() map[string]string {
	apa.mu.RLock()
	defer apa.mu.RUnlock()
	return apa.config
}

// GetConfigValue 获取配置值
func (apa *AbstractProtocolAdapter) GetConfigValue(key string) string {
	apa.mu.RLock()
	defer apa.mu.RUnlock()
	return apa.config[key]
}

// GetProtocolName 获取协议名称
func (apa *AbstractProtocolAdapter) GetProtocolName() string {
	apa.mu.RLock()
	defer apa.mu.RUnlock()
	return apa.protocolName
}

// GetStatus 获取适配器状态
func (apa *AbstractProtocolAdapter) GetStatus() *AdapterStatus {
	apa.mu.RLock()
	defer apa.mu.RUnlock()
	return apa.status
}

// LogAdapterActivity 记录适配器活动
func (apa *AbstractProtocolAdapter) LogAdapterActivity(message string) {
	apa.mu.Lock()
	defer apa.mu.Unlock()

	apa.status.LastActivity = time.Now()
	apa.status.MessageCount++

	// 这里应该使用实际的日志系统
	fmt.Printf("[%s] %s: %s\n", apa.protocolName, time.Now().Format(time.RFC3339), message)
}

// UpdateStatus 更新状态
func (apa *AbstractProtocolAdapter) UpdateStatus(state AdapterState, message string) {
	apa.mu.Lock()
	defer apa.mu.Unlock()

	apa.status.State = state
	apa.status.Message = message
	apa.status.LastActivity = time.Now()
}

// AdapterStatus 适配器状态
type AdapterStatus struct {
	State         AdapterState
	Message       string
	LastActivity  time.Time
	MessageCount  int64
	ErrorCount    int64
	BytesReceived int64
}

// AdapterState 适配器状态
type AdapterState int

const (
	AdapterStateUninitialized AdapterState = iota
	AdapterStateInitialized
	AdapterStateRunning
	AdapterStateStopped
	AdapterStateError
)

// String 返回适配器状态的字符串表示
func (as AdapterState) String() string {
	switch as {
	case AdapterStateUninitialized:
		return "uninitialized"
	case AdapterStateInitialized:
		return "initialized"
	case AdapterStateRunning:
		return "running"
	case AdapterStateStopped:
		return "stopped"
	case AdapterStateError:
		return "error"
	default:
		return "unknown"
	}
}

// TCPAdapter TCP协议适配器
type TCPAdapter struct {
	*AbstractProtocolAdapter
	listener net.Listener
	conn     net.Conn
}

// NewTCPAdapter 创建TCP适配器
func NewTCPAdapter() *TCPAdapter {
	return &TCPAdapter{
		AbstractProtocolAdapter: NewAbstractProtocolAdapter("tcp"),
	}
}

// Initialize 初始化TCP适配器
func (ta *TCPAdapter) Initialize(config map[string]string) error {
	if err := ta.AbstractProtocolAdapter.Initialize(config); err != nil {
		return err
	}

	// 获取TCP配置
	host := ta.GetConfigValue("host")
	if host == "" {
		host = "0.0.0.0"
	}

	portStr := ta.GetConfigValue("port")
	if portStr == "" {
		portStr = "8080"
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("端口号格式错误: %s", portStr)
	}

	// 创建TCP监听器
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return fmt.Errorf("创建TCP监听器失败: %w", err)
	}

	ta.listener = listener
	ta.UpdateStatus(AdapterStateRunning, "TCP监听器已启动")

	return nil
}

// Receive 接收数据
func (ta *TCPAdapter) Receive() (source.DataPacket, error) {
	// 接受连接
	conn, err := ta.listener.Accept()
	if err != nil {
		ta.UpdateStatus(AdapterStateError, fmt.Sprintf("接受连接失败: %v", err))
		return nil, fmt.Errorf("接受连接失败: %w", err)
	}

	ta.conn = conn

	// 读取数据
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		ta.UpdateStatus(AdapterStateError, fmt.Sprintf("读取数据失败: %v", err))
		return nil, fmt.Errorf("读取数据失败: %w", err)
	}

	// 创建数据包
	packet := &source.BinaryPacket{
		Data:      buffer[:n],
		Timestamp: time.Now(),
		Metadata:  make(map[string]string),
	}

	// 添加元数据
	packet.Metadata["protocol"] = "tcp"
	packet.Metadata["remote_addr"] = conn.RemoteAddr().String()
	packet.Metadata["local_addr"] = conn.LocalAddr().String()

	ta.LogAdapterActivity(fmt.Sprintf("接收到 %d 字节数据", n))
	ta.GetStatus().BytesReceived += int64(n)

	return packet, nil
}

// Close 关闭适配器
func (ta *TCPAdapter) Close() error {
	ta.UpdateStatus(AdapterStateStopped, "TCP适配器已停止")

	if ta.conn != nil {
		ta.conn.Close()
	}

	if ta.listener != nil {
		return ta.listener.Close()
	}

	return nil
}

// UDPAdapter UDP协议适配器
type UDPAdapter struct {
	*AbstractProtocolAdapter
	conn *net.UDPConn
}

// NewUDPAdapter 创建UDP适配器
func NewUDPAdapter() *UDPAdapter {
	return &UDPAdapter{
		AbstractProtocolAdapter: NewAbstractProtocolAdapter("udp"),
	}
}

// Initialize 初始化UDP适配器
func (ua *UDPAdapter) Initialize(config map[string]string) error {
	if err := ua.AbstractProtocolAdapter.Initialize(config); err != nil {
		return err
	}

	// 获取UDP配置
	host := ua.GetConfigValue("host")
	if host == "" {
		host = "0.0.0.0"
	}

	portStr := ua.GetConfigValue("port")
	if portStr == "" {
		portStr = "8080"
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("端口号格式错误: %s", portStr)
	}

	// 创建UDP连接
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return fmt.Errorf("解析UDP地址失败: %w", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("创建UDP监听器失败: %w", err)
	}

	ua.conn = conn
	ua.UpdateStatus(AdapterStateRunning, "UDP监听器已启动")

	return nil
}

// Receive 接收数据
func (ua *UDPAdapter) Receive() (source.DataPacket, error) {
	// 读取UDP数据
	buffer := make([]byte, 1024)
	n, remoteAddr, err := ua.conn.ReadFromUDP(buffer)
	if err != nil {
		ua.UpdateStatus(AdapterStateError, fmt.Sprintf("读取UDP数据失败: %v", err))
		return nil, fmt.Errorf("读取UDP数据失败: %w", err)
	}

	// 创建数据包
	packet := &source.BinaryPacket{
		Data:      buffer[:n],
		Timestamp: time.Now(),
		Metadata:  make(map[string]string),
	}

	// 添加元数据
	packet.Metadata["protocol"] = "udp"
	packet.Metadata["remote_addr"] = remoteAddr.String()
	packet.Metadata["local_addr"] = ua.conn.LocalAddr().String()

	ua.LogAdapterActivity(fmt.Sprintf("接收到 %d 字节UDP数据", n))
	ua.GetStatus().BytesReceived += int64(n)

	return packet, nil
}

// Close 关闭适配器
func (ua *UDPAdapter) Close() error {
	ua.UpdateStatus(AdapterStateStopped, "UDP适配器已停止")

	if ua.conn != nil {
		return ua.conn.Close()
	}

	return nil
}

// HTTPAdapter HTTP协议适配器
type HTTPAdapter struct {
	*AbstractProtocolAdapter
	server *HTTPServer
}

// NewHTTPAdapter 创建HTTP适配器
func NewHTTPAdapter() *HTTPAdapter {
	return &HTTPAdapter{
		AbstractProtocolAdapter: NewAbstractProtocolAdapter("http"),
	}
}

// Initialize 初始化HTTP适配器
func (ha *HTTPAdapter) Initialize(config map[string]string) error {
	if err := ha.AbstractProtocolAdapter.Initialize(config); err != nil {
		return err
	}

	// 获取HTTP配置
	host := ha.GetConfigValue("host")
	if host == "" {
		host = "0.0.0.0"
	}

	portStr := ha.GetConfigValue("port")
	if portStr == "" {
		portStr = "8080"
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("端口号格式错误: %s", portStr)
	}

	// 创建HTTP服务器
	server := NewHTTPServer(host, port)
	ha.server = server

	ha.UpdateStatus(AdapterStateRunning, "HTTP服务器已启动")

	return nil
}

// Receive 接收数据
func (ha *HTTPAdapter) Receive() (source.DataPacket, error) {
	// HTTP适配器通过回调接收数据
	// 这里返回一个空的数据包，实际数据通过HTTP处理器接收
	return &source.BinaryPacket{
		Data:      []byte{},
		Timestamp: time.Now(),
		Metadata:  make(map[string]string),
	}, nil
}

// Close 关闭适配器
func (ha *HTTPAdapter) Close() error {
	ha.UpdateStatus(AdapterStateStopped, "HTTP适配器已停止")

	if ha.server != nil {
		return ha.server.Stop()
	}

	return nil
}

// HTTPServer HTTP服务器
type HTTPServer struct {
	host   string
	port   int
	server *net.Server
	mu     sync.RWMutex
}

// NewHTTPServer 创建HTTP服务器
func NewHTTPServer(host string, port int) *HTTPServer {
	return &HTTPServer{
		host: host,
		port: port,
	}
}

// Start 启动HTTP服务器
func (hs *HTTPServer) Start() error {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	// 这里实现HTTP服务器的启动逻辑
	// 可以使用标准库的net/http包或第三方HTTP框架

	return nil
}

// Stop 停止HTTP服务器
func (hs *HTTPServer) Stop() error {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	if hs.server != nil {
		return hs.server.Close()
	}

	return nil
}

// ModbusAdapter Modbus协议适配器
type ModbusAdapter struct {
	*AbstractProtocolAdapter
	// 这里可以添加Modbus相关的字段
	// 例如：ModbusMaster、连接配置等
}

// NewModbusAdapter 创建Modbus适配器
func NewModbusAdapter() *ModbusAdapter {
	return &ModbusAdapter{
		AbstractProtocolAdapter: NewAbstractProtocolAdapter("modbus-rtu"),
	}
}

// Initialize 初始化Modbus适配器
func (ma *ModbusAdapter) Initialize(config map[string]string) error {
	if err := ma.AbstractProtocolAdapter.Initialize(config); err != nil {
		return err
	}

	// 获取Modbus配置
	serialPort := ma.GetConfigValue("serial.port")
	if serialPort == "" {
		return fmt.Errorf("串口不能为空")
	}

	baudRateStr := ma.GetConfigValue("baud.rate")
	if baudRateStr == "" {
		baudRateStr = "9600"
	}

	baudRate, err := strconv.Atoi(baudRateStr)
	if err != nil {
		return fmt.Errorf("波特率格式错误: %s", baudRateStr)
	}

	// 这里实现Modbus连接初始化
	// 可以使用第三方Modbus库

	ma.UpdateStatus(AdapterStateRunning, "Modbus适配器已启动")

	return nil
}

// Receive 接收数据
func (ma *ModbusAdapter) Receive() (source.DataPacket, error) {
	// 这里实现Modbus数据读取逻辑
	// 例如：读取传感器寄存器数据

	// 示例：创建模拟的Modbus数据包
	packet := &source.CustomDataPacket{
		Data: map[string]interface{}{
			"register": 0,
			"value":    123.45,
		},
		Type:      "modbus",
		Timestamp: time.Now(),
		Metadata:  make(map[string]string),
	}

	packet.Metadata["protocol"] = "modbus-rtu"
	packet.Metadata["register_address"] = "0"

	ma.LogAdapterActivity("接收到Modbus数据")

	return packet, nil
}

// Close 关闭适配器
func (ma *ModbusAdapter) Close() error {
	ma.UpdateStatus(AdapterStateStopped, "Modbus适配器已停止")

	// 这里实现Modbus连接关闭逻辑

	return nil
}

// CustomProtocolRegistry 自定义协议注册表
type CustomProtocolRegistry struct {
	registeredProtocols map[string]func() ProtocolAdapter
	mu                  sync.RWMutex
}

// NewCustomProtocolRegistry 创建协议注册表
func NewCustomProtocolRegistry() *CustomProtocolRegistry {
	return &CustomProtocolRegistry{
		registeredProtocols: make(map[string]func() ProtocolAdapter),
	}
}

// RegisterProtocol 注册协议
func (cpr *CustomProtocolRegistry) RegisterProtocol(protocolName string, adapterFactory func() ProtocolAdapter) {
	cpr.mu.Lock()
	defer cpr.mu.Unlock()

	cpr.registeredProtocols[protocolName] = adapterFactory
}

// CreateAdapter 创建适配器
func (cpr *CustomProtocolRegistry) CreateAdapter(protocolName string) (ProtocolAdapter, error) {
	cpr.mu.RLock()
	defer cpr.mu.RUnlock()

	adapterFactory, exists := cpr.registeredProtocols[protocolName]
	if !exists {
		return nil, fmt.Errorf("不支持的协议: %s", protocolName)
	}

	return adapterFactory(), nil
}

// ListProtocols 列出所有协议
func (cpr *CustomProtocolRegistry) ListProtocols() []string {
	cpr.mu.RLock()
	defer cpr.mu.RUnlock()

	protocols := make([]string, 0, len(cpr.registeredProtocols))
	for protocol := range cpr.registeredProtocols {
		protocols = append(protocols, protocol)
	}
	return protocols
}

// UnregisterProtocol 注销协议
func (cpr *CustomProtocolRegistry) UnregisterProtocol(protocolName string) {
	cpr.mu.Lock()
	defer cpr.mu.Unlock()

	delete(cpr.registeredProtocols, protocolName)
}

// ProtocolExtension 协议扩展接口
type ProtocolExtension interface {
	GetName() string
	Register(registry *CustomProtocolRegistry)
	CreateAdapter() ProtocolAdapter
}

// ModbusProtocolExtension Modbus协议扩展
type ModbusProtocolExtension struct{}

// NewModbusProtocolExtension 创建Modbus协议扩展
func NewModbusProtocolExtension() *ModbusProtocolExtension {
	return &ModbusProtocolExtension{}
}

// GetName 获取协议名称
func (mpe *ModbusProtocolExtension) GetName() string {
	return "modbus-rtu"
}

// Register 注册协议
func (mpe *ModbusProtocolExtension) Register(registry *CustomProtocolRegistry) {
	registry.RegisterProtocol(mpe.GetName(), mpe.CreateAdapter)
}

// CreateAdapter 创建适配器
func (mpe *ModbusProtocolExtension) CreateAdapter() ProtocolAdapter {
	return NewModbusAdapter()
}

// ProtocolManager 协议管理器
type ProtocolManager struct {
	registry *CustomProtocolRegistry
	adapters map[string]ProtocolAdapter
	mu       sync.RWMutex
}

// NewProtocolManager 创建协议管理器
func NewProtocolManager() *ProtocolManager {
	return &ProtocolManager{
		registry: NewCustomProtocolRegistry(),
		adapters: make(map[string]ProtocolAdapter),
	}
}

// Initialize 初始化协议管理器
func (pm *ProtocolManager) Initialize(ctx context.Context) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 注册内置协议
	pm.registerBuiltinProtocols()

	return nil
}

// registerBuiltinProtocols 注册内置协议
func (pm *ProtocolManager) registerBuiltinProtocols() {
	// 注册TCP协议
	pm.registry.RegisterProtocol("tcp", func() ProtocolAdapter {
		return NewTCPAdapter()
	})

	// 注册UDP协议
	pm.registry.RegisterProtocol("udp", func() ProtocolAdapter {
		return NewUDPAdapter()
	})

	// 注册HTTP协议
	pm.registry.RegisterProtocol("http", func() ProtocolAdapter {
		return NewHTTPAdapter()
	})

	// 注册Modbus协议
	modbusExtension := NewModbusProtocolExtension()
	modbusExtension.Register(pm.registry)
}

// CreateAdapter 创建适配器
func (pm *ProtocolManager) CreateAdapter(protocolName string, config map[string]string) (ProtocolAdapter, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 创建适配器
	adapter, err := pm.registry.CreateAdapter(protocolName)
	if err != nil {
		return nil, err
	}

	// 初始化适配器
	if err := adapter.Initialize(config); err != nil {
		return nil, fmt.Errorf("初始化适配器失败: %w", err)
	}

	// 保存适配器
	pm.adapters[protocolName] = adapter

	return adapter, nil
}

// GetAdapter 获取适配器
func (pm *ProtocolManager) GetAdapter(protocolName string) (ProtocolAdapter, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	adapter, exists := pm.adapters[protocolName]
	return adapter, exists
}

// ListAdapters 列出所有适配器
func (pm *ProtocolManager) ListAdapters() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	adapters := make([]string, 0, len(pm.adapters))
	for name := range pm.adapters {
		adapters = append(adapters, name)
	}
	return adapters
}

// RemoveAdapter 移除适配器
func (pm *ProtocolManager) RemoveAdapter(protocolName string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	adapter, exists := pm.adapters[protocolName]
	if !exists {
		return fmt.Errorf("适配器 %s 不存在", protocolName)
	}

	// 关闭适配器
	if err := adapter.Close(); err != nil {
		return fmt.Errorf("关闭适配器失败: %w", err)
	}

	delete(pm.adapters, protocolName)

	return nil
}

// RegisterExtension 注册协议扩展
func (pm *ProtocolManager) RegisterExtension(extension ProtocolExtension) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	extension.Register(pm.registry)
}
