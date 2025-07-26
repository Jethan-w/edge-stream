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
)

// Monitoring and metrics constants
const (
	// DefaultMetricsPort is the default port for metrics endpoint
	DefaultMetricsPort = 9090
	// DefaultHealthCheckPort is the default port for health check endpoint
	DefaultHealthCheckPort = 8081
)