package sink

import (
	"time"
)

// OutputTargetType 输出目标类型
type OutputTargetType int

const (
	OutputTargetTypeFileSystem OutputTargetType = iota
	OutputTargetTypeDatabase
	OutputTargetTypeMessageQueue
	OutputTargetTypeSearchEngine
	OutputTargetTypeRESTAPI
	OutputTargetTypeCustom
)

func (t OutputTargetType) String() string {
	switch t {
	case OutputTargetTypeFileSystem:
		return "FileSystem"
	case OutputTargetTypeDatabase:
		return "Database"
	case OutputTargetTypeMessageQueue:
		return "MessageQueue"
	case OutputTargetTypeSearchEngine:
		return "SearchEngine"
	case OutputTargetTypeRESTAPI:
		return "RESTAPI"
	case OutputTargetTypeCustom:
		return "Custom"
	default:
		return "Unknown"
	}
}

// SecurityProtocol 安全传输协议
type SecurityProtocol int

const (
	SecurityProtocolTLS SecurityProtocol = iota
	SecurityProtocolHTTPS
	SecurityProtocolSFTP
	SecurityProtocolSSL
)

func (p SecurityProtocol) String() string {
	switch p {
	case SecurityProtocolTLS:
		return "TLS"
	case SecurityProtocolHTTPS:
		return "HTTPS"
	case SecurityProtocolSFTP:
		return "SFTP"
	case SecurityProtocolSSL:
		return "SSL"
	default:
		return "Unknown"
	}
}

// AuthenticationMethod 认证方法
type AuthenticationMethod int

const (
	AuthenticationMethodOAuth2 AuthenticationMethod = iota
	AuthenticationMethodClientCertificate
	AuthenticationMethodUsernamePassword
	AuthenticationMethodAPIKey
)

func (a AuthenticationMethod) String() string {
	switch a {
	case AuthenticationMethodOAuth2:
		return "OAuth2"
	case AuthenticationMethodClientCertificate:
		return "ClientCertificate"
	case AuthenticationMethodUsernamePassword:
		return "UsernamePassword"
	case AuthenticationMethodAPIKey:
		return "APIKey"
	default:
		return "Unknown"
	}
}

// EncryptionAlgorithm 加密算法
type EncryptionAlgorithm int

const (
	EncryptionAlgorithmAES256 EncryptionAlgorithm = iota
	EncryptionAlgorithmAES128
	EncryptionAlgorithmRSA
	EncryptionAlgorithmChaCha20
)

func (e EncryptionAlgorithm) String() string {
	switch e {
	case EncryptionAlgorithmAES256:
		return "AES256"
	case EncryptionAlgorithmAES128:
		return "AES128"
	case EncryptionAlgorithmRSA:
		return "RSA"
	case EncryptionAlgorithmChaCha20:
		return "ChaCha20"
	default:
		return "Unknown"
	}
}

// ReliabilityStrategy 可靠性策略
type ReliabilityStrategy int

const (
	ReliabilityStrategyAtMostOnce ReliabilityStrategy = iota
	ReliabilityStrategyAtLeastOnce
	ReliabilityStrategyExactlyOnce
)

func (r ReliabilityStrategy) String() string {
	switch r {
	case ReliabilityStrategyAtMostOnce:
		return "AtMostOnce"
	case ReliabilityStrategyAtLeastOnce:
		return "AtLeastOnce"
	case ReliabilityStrategyExactlyOnce:
		return "ExactlyOnce"
	default:
		return "Unknown"
	}
}

// ErrorRoutingStrategy 错误路由策略
type ErrorRoutingStrategy int

const (
	ErrorRoutingStrategyRetry ErrorRoutingStrategy = iota
	ErrorRoutingStrategyRouteToError
	ErrorRoutingStrategyDrop
	ErrorRoutingStrategyDeadLetter
)

func (e ErrorRoutingStrategy) String() string {
	switch e {
	case ErrorRoutingStrategyRetry:
		return "Retry"
	case ErrorRoutingStrategyRouteToError:
		return "RouteToError"
	case ErrorRoutingStrategyDrop:
		return "Drop"
	case ErrorRoutingStrategyDeadLetter:
		return "DeadLetter"
	default:
		return "Unknown"
	}
}

// SecurityConfiguration 安全配置
type SecurityConfiguration struct {
	Protocol             SecurityProtocol     `json:"protocol"`
	AuthenticationMethod AuthenticationMethod `json:"authentication_method"`
	EncryptionAlgorithm  EncryptionAlgorithm  `json:"encryption_algorithm"`
	TLSCertPath          string               `json:"tls_cert_path"`
	TLSKeyPath           string               `json:"tls_key_path"`
	CAFilePath           string               `json:"ca_file_path"`
	Username             string               `json:"username"`
	Password             string               `json:"password"`
	APIKey               string               `json:"api_key"`
	OAuth2Config         *OAuth2Config        `json:"oauth2_config"`
}

// OAuth2Config OAuth2 配置
type OAuth2Config struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	TokenURL     string `json:"token_url"`
	Scope        string `json:"scope"`
}

// ReliabilityConfig 可靠性配置
type ReliabilityConfig struct {
	Strategy           ReliabilityStrategy `json:"strategy"`
	MaxRetries         int                 `json:"max_retries"`
	BaseRetryDelay     time.Duration       `json:"base_retry_delay"`
	MaxRetryDelay      time.Duration       `json:"max_retry_delay"`
	RetryMultiplier    float64             `json:"retry_multiplier"`
	EnableTransaction  bool                `json:"enable_transaction"`
	TransactionTimeout time.Duration       `json:"transaction_timeout"`
}

// ErrorHandlingConfig 错误处理配置
type ErrorHandlingConfig struct {
	DefaultStrategy     ErrorRoutingStrategy `json:"default_strategy"`
	MaxRetryCount       int                  `json:"max_retry_count"`
	ErrorQueueName      string               `json:"error_queue_name"`
	DeadLetterQueueName string               `json:"dead_letter_queue_name"`
	LogErrors           bool                 `json:"log_errors"`
	NotifyOnError       bool                 `json:"notify_on_error"`
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries int           `json:"max_retries"`
	BaseDelay  time.Duration `json:"base_delay"`
	MaxDelay   time.Duration `json:"max_delay"`
	Multiplier float64       `json:"multiplier"`
	Jitter     bool          `json:"jitter"`
}

// OutputTargetMetadata 输出目标元数据
type OutputTargetMetadata struct {
	ID                   string                 `json:"id"`
	Type                 OutputTargetType       `json:"type"`
	ConnectionString     string                 `json:"connection_string"`
	AdditionalProperties map[string]string      `json:"additional_properties"`
	SecurityConfig       *SecurityConfiguration `json:"security_config"`
	CreatedAt            time.Time              `json:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at"`
}

// OutputOperation 输出操作
type OutputOperation struct {
	flowFile interface{} // *flowfile.FlowFile
	adapter  TargetAdapter
	sink     *Sink
}

// Execute 执行输出操作
func (o *OutputOperation) Execute() error {
	return o.adapter.Write(o.flowFile)
}

// TransientOutputError 临时输出错误
type TransientOutputError struct {
	Message string
	Cause   error
}

func (e *TransientOutputError) Error() string {
	return e.Message
}

func (e *TransientOutputError) Unwrap() error {
	return e.Cause
}

// NewTransientOutputError 创建临时输出错误
func NewTransientOutputError(message string, cause error) *TransientOutputError {
	return &TransientOutputError{
		Message: message,
		Cause:   cause,
	}
}

// UnsupportedProtocolError 不支持的协议错误
type UnsupportedProtocolError struct {
	Protocol string
}

func (e *UnsupportedProtocolError) Error() string {
	return "unsupported protocol: " + e.Protocol
}

// NewUnsupportedProtocolError 创建不支持的协议错误
func NewUnsupportedProtocolError(protocol string) *UnsupportedProtocolError {
	return &UnsupportedProtocolError{
		Protocol: protocol,
	}
}

// UnsupportedAuthenticationError 不支持的认证错误
type UnsupportedAuthenticationError struct {
	Method string
}

func (e *UnsupportedAuthenticationError) Error() string {
	return "unsupported authentication method: " + e.Method
}

// NewUnsupportedAuthenticationError 创建不支持的认证错误
func NewUnsupportedAuthenticationError(method string) *UnsupportedAuthenticationError {
	return &UnsupportedAuthenticationError{
		Method: method,
	}
}
