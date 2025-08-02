package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// LogLevel represents the logging level
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// PersistenceType represents the type of persistence to use
type PersistenceType string

const (
	PersistenceMemory PersistenceType = "memory"
	PersistenceFile   PersistenceType = "file"
	PersistenceDB     PersistenceType = "database"
)

// Config holds the application configuration
type Config struct {
	// HTTP server configuration
	HTTPPort int    `json:"http_port"`
	HTTPHost string `json:"http_host"`

	// Logging configuration
	LogLevel LogLevel `json:"log_level"`

	// Persistence configuration
	PersistenceType PersistenceType `json:"persistence_type"`
	PersistencePath string          `json:"persistence_path"`

	// Database configuration (when using database persistence)
	DatabaseURL string `json:"database_url"`
}

// Load loads configuration from environment variables with sensible defaults
func Load() (*Config, error) {
	config := &Config{
		// Default values
		HTTPPort:        8080,
		HTTPHost:        "localhost",
		LogLevel:        LogLevelInfo,
		PersistenceType: PersistenceMemory,
		PersistencePath: "./kvstore.json",
		DatabaseURL:     "",
	}

	// Load from environment variables
	if port := os.Getenv("KVSTORE_HTTP_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err != nil {
			return nil, fmt.Errorf("invalid KVSTORE_HTTP_PORT: %w", err)
		} else {
			config.HTTPPort = p
		}
	}

	if host := os.Getenv("KVSTORE_HTTP_HOST"); host != "" {
		config.HTTPHost = host
	}

	if logLevel := os.Getenv("KVSTORE_LOG_LEVEL"); logLevel != "" {
		level := LogLevel(strings.ToLower(logLevel))
		if !isValidLogLevel(level) {
			return nil, fmt.Errorf("invalid KVSTORE_LOG_LEVEL: %s (must be debug, info, warn, or error)", logLevel)
		}
		config.LogLevel = level
	}

	if persistenceType := os.Getenv("KVSTORE_PERSISTENCE_TYPE"); persistenceType != "" {
		pType := PersistenceType(strings.ToLower(persistenceType))
		if !isValidPersistenceType(pType) {
			return nil, fmt.Errorf("invalid KVSTORE_PERSISTENCE_TYPE: %s (must be memory, file, or database)", persistenceType)
		}
		config.PersistenceType = pType
	}

	if persistencePath := os.Getenv("KVSTORE_PERSISTENCE_PATH"); persistencePath != "" {
		config.PersistencePath = persistencePath
	}

	if dbURL := os.Getenv("KVSTORE_DATABASE_URL"); dbURL != "" {
		config.DatabaseURL = dbURL
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate HTTP port
	if c.HTTPPort < 1 || c.HTTPPort > 65535 {
		return fmt.Errorf("http_port must be between 1 and 65535, got %d", c.HTTPPort)
	}

	// Validate HTTP host
	if c.HTTPHost == "" {
		return fmt.Errorf("http_host cannot be empty")
	}

	// Validate log level
	if !isValidLogLevel(c.LogLevel) {
		return fmt.Errorf("invalid log_level: %s", c.LogLevel)
	}

	// Validate persistence type
	if !isValidPersistenceType(c.PersistenceType) {
		return fmt.Errorf("invalid persistence_type: %s", c.PersistenceType)
	}

	// Validate persistence-specific configuration
	switch c.PersistenceType {
	case PersistenceFile:
		if c.PersistencePath == "" {
			return fmt.Errorf("persistence_path is required when using file persistence")
		}
	case PersistenceDB:
		if c.DatabaseURL == "" {
			return fmt.Errorf("database_url is required when using database persistence")
		}
	}

	return nil
}

// Address returns the HTTP server address
func (c *Config) Address() string {
	return fmt.Sprintf("%s:%d", c.HTTPHost, c.HTTPPort)
}

// IsDebugEnabled returns true if debug logging is enabled
func (c *Config) IsDebugEnabled() bool {
	return c.LogLevel == LogLevelDebug
}

// isValidLogLevel checks if the log level is valid
func isValidLogLevel(level LogLevel) bool {
	switch level {
	case LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError:
		return true
	default:
		return false
	}
}

// isValidPersistenceType checks if the persistence type is valid
func isValidPersistenceType(pType PersistenceType) bool {
	switch pType {
	case PersistenceMemory, PersistenceFile, PersistenceDB:
		return true
	default:
		return false
	}
}
