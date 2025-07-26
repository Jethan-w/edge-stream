// Copyright 2024 EdgeStream Team
// Licensed under the Apache License, Version 2.0

package constants

// Database connection constants
const (
	// DefaultMySQLPort is the default port for MySQL database connections
	DefaultMySQLPort = 3306
	// DefaultPostgreSQLPort is the default port for PostgreSQL database connections
	DefaultPostgreSQLPort = 5432
	// DefaultRedisPort is the default port for Redis connections
	DefaultRedisPort = 6379
	// DefaultRedisDB is the default Redis database number
	DefaultRedisDB = 0
	// DefaultRedisTimeoutSeconds is the default timeout for Redis operations
	DefaultRedisTimeoutSeconds = 5
	// DefaultHTTPPort is the default port for HTTP server
	DefaultHTTPPort = 8080
)

// Timeout and retry constants
const (
	// DefaultConnectionTimeoutSeconds is the default timeout for database connections
	DefaultConnectionTimeoutSeconds = 30
	// DefaultRetryAttempts is the default number of retry attempts
	DefaultRetryAttempts = 5
	// DefaultRetryDelaySeconds is the default delay between retries
	DefaultRetryDelaySeconds = 10
)

// Buffer and channel size constants
const (
	// DefaultChannelBufferSize is the default buffer size for channels
	DefaultChannelBufferSize = 100
	// DefaultErrorChannelSize is the default buffer size for error channels
	DefaultErrorChannelSize = 10
	// DefaultBatchSize is the default batch size for processing
	DefaultBatchSize = 20
)

// File and directory permission constants
const (
	// DefaultDirectoryPermission is the default permission for creating directories
	DefaultDirectoryPermission = 0750
	// DefaultFilePermission is the default permission for creating files
	DefaultFilePermission = 0644
)

// Time and duration constants
const (
	// DefaultCleanupIntervalSeconds is the default interval for cleanup operations
	DefaultCleanupIntervalSeconds = 30
	// DefaultTimeWindowSizeMultiplier is used for time window calculations
	DefaultTimeWindowSizeMultiplier = 4

	// State management constants
	DefaultCheckpointIntervalMinutes = 5
	DefaultMaxCheckpoints            = 10
	DefaultReplicationFactor         = 3
	DefaultAverageLatencyDivisor     = 2

	// Stream engine constants
	DefaultEngineEventChannelSize      = 100
	DefaultProcessorStopTimeoutSeconds = 5
	// DefaultServerReadTimeoutSeconds is the default read timeout for HTTP server
	DefaultServerReadTimeoutSeconds = 30
	// DefaultServerWriteTimeoutSeconds is the default write timeout for HTTP server
	DefaultServerWriteTimeoutSeconds = 30
	// DefaultServerIdleTimeoutSeconds is the default idle timeout for HTTP server
	DefaultServerIdleTimeoutSeconds = 60
	// DefaultMetricsIntervalSeconds is the default interval for metrics collection
	DefaultMetricsIntervalSeconds = 30
)

// Monitoring and metrics constants
const (
	// DefaultMetricsPort is the default port for metrics endpoint
	DefaultMetricsPort = 9090
	// DefaultHealthCheckPort is the default port for health check endpoint
	DefaultHealthCheckPort = 8081
)

// Encryption constants
const (
	// AESKeySize is the size of AES encryption key in bytes (256 bits)
	AESKeySize = 32
)

// Integer conversion constants
const (
	// MaxInt32Value is the maximum value for int32 type
	MaxInt32Value = 2147483647
	// MinInt32Value is the minimum value for int32 type
	MinInt32Value = -2147483648
	// EnvironmentVariableParts is the expected number of parts when splitting environment variables
	EnvironmentVariableParts = 2
)

// Connector constants
const (
	// DefaultConnectorEventChannelSize is the default buffer size for connector event channels
	DefaultConnectorEventChannelSize = 100
	// DefaultConnectorRestartDelayMilliseconds is the default delay before restarting a connector
	DefaultConnectorRestartDelayMilliseconds = 100
	// MinCheckpointFileNameParts is the minimum number of parts when parsing checkpoint file names
	MinCheckpointFileNameParts = 3
)
