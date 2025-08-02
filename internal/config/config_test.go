package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear environment variables
	clearEnv()

	config, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Check default values
	if config.HTTPPort != 8080 {
		t.Errorf("Expected HTTPPort to be 8080, got %d", config.HTTPPort)
	}

	if config.HTTPHost != "localhost" {
		t.Errorf("Expected HTTPHost to be 'localhost', got %s", config.HTTPHost)
	}

	if config.LogLevel != LogLevelInfo {
		t.Errorf("Expected LogLevel to be 'info', got %s", config.LogLevel)
	}

	if config.PersistenceType != PersistenceMemory {
		t.Errorf("Expected PersistenceType to be 'memory', got %s", config.PersistenceType)
	}

	if config.PersistencePath != "./kvstore.json" {
		t.Errorf("Expected PersistencePath to be './kvstore.json', got %s", config.PersistencePath)
	}
}

func TestLoad_FromEnvironment(t *testing.T) {
	// Clear environment variables
	clearEnv()

	// Set environment variables
	os.Setenv("KVSTORE_HTTP_PORT", "9000")
	os.Setenv("KVSTORE_HTTP_HOST", "0.0.0.0")
	os.Setenv("KVSTORE_LOG_LEVEL", "debug")
	os.Setenv("KVSTORE_PERSISTENCE_TYPE", "file")
	os.Setenv("KVSTORE_PERSISTENCE_PATH", "/tmp/kvstore.json")
	defer clearEnv()

	config, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if config.HTTPPort != 9000 {
		t.Errorf("Expected HTTPPort to be 9000, got %d", config.HTTPPort)
	}

	if config.HTTPHost != "0.0.0.0" {
		t.Errorf("Expected HTTPHost to be '0.0.0.0', got %s", config.HTTPHost)
	}

	if config.LogLevel != LogLevelDebug {
		t.Errorf("Expected LogLevel to be 'debug', got %s", config.LogLevel)
	}

	if config.PersistenceType != PersistenceFile {
		t.Errorf("Expected PersistenceType to be 'file', got %s", config.PersistenceType)
	}

	if config.PersistencePath != "/tmp/kvstore.json" {
		t.Errorf("Expected PersistencePath to be '/tmp/kvstore.json', got %s", config.PersistencePath)
	}
}

func TestLoad_InvalidPort(t *testing.T) {
	clearEnv()
	os.Setenv("KVSTORE_HTTP_PORT", "invalid")
	defer clearEnv()

	_, err := Load()
	if err == nil {
		t.Error("Expected error for invalid port, got nil")
	}
}

func TestLoad_InvalidLogLevel(t *testing.T) {
	clearEnv()
	os.Setenv("KVSTORE_LOG_LEVEL", "invalid")
	defer clearEnv()

	_, err := Load()
	if err == nil {
		t.Error("Expected error for invalid log level, got nil")
	}
}

func TestLoad_InvalidPersistenceType(t *testing.T) {
	clearEnv()
	os.Setenv("KVSTORE_PERSISTENCE_TYPE", "invalid")
	defer clearEnv()

	_, err := Load()
	if err == nil {
		t.Error("Expected error for invalid persistence type, got nil")
	}
}

func TestValidate_InvalidPort(t *testing.T) {
	config := &Config{
		HTTPPort:        0,
		HTTPHost:        "localhost",
		LogLevel:        LogLevelInfo,
		PersistenceType: PersistenceMemory,
	}

	err := config.Validate()
	if err == nil {
		t.Error("Expected validation error for invalid port, got nil")
	}
}

func TestValidate_EmptyHost(t *testing.T) {
	config := &Config{
		HTTPPort:        8080,
		HTTPHost:        "",
		LogLevel:        LogLevelInfo,
		PersistenceType: PersistenceMemory,
	}

	err := config.Validate()
	if err == nil {
		t.Error("Expected validation error for empty host, got nil")
	}
}

func TestValidate_FilePersistenceWithoutPath(t *testing.T) {
	config := &Config{
		HTTPPort:        8080,
		HTTPHost:        "localhost",
		LogLevel:        LogLevelInfo,
		PersistenceType: PersistenceFile,
		PersistencePath: "",
	}

	err := config.Validate()
	if err == nil {
		t.Error("Expected validation error for file persistence without path, got nil")
	}
}

func TestValidate_DatabasePersistenceWithoutURL(t *testing.T) {
	config := &Config{
		HTTPPort:        8080,
		HTTPHost:        "localhost",
		LogLevel:        LogLevelInfo,
		PersistenceType: PersistenceDB,
		DatabaseURL:     "",
	}

	err := config.Validate()
	if err == nil {
		t.Error("Expected validation error for database persistence without URL, got nil")
	}
}

func TestAddress(t *testing.T) {
	config := &Config{
		HTTPPort: 9000,
		HTTPHost: "0.0.0.0",
	}

	expected := "0.0.0.0:9000"
	if config.Address() != expected {
		t.Errorf("Expected address to be %s, got %s", expected, config.Address())
	}
}

func TestIsDebugEnabled(t *testing.T) {
	config := &Config{LogLevel: LogLevelDebug}
	if !config.IsDebugEnabled() {
		t.Error("Expected IsDebugEnabled to return true for debug level")
	}

	config.LogLevel = LogLevelInfo
	if config.IsDebugEnabled() {
		t.Error("Expected IsDebugEnabled to return false for info level")
	}
}

// clearEnv clears all KVSTORE-related environment variables
func clearEnv() {
	envVars := []string{
		"KVSTORE_HTTP_PORT",
		"KVSTORE_HTTP_HOST",
		"KVSTORE_LOG_LEVEL",
		"KVSTORE_PERSISTENCE_TYPE",
		"KVSTORE_PERSISTENCE_PATH",
		"KVSTORE_DATABASE_URL",
	}

	for _, env := range envVars {
		os.Unsetenv(env)
	}
}
