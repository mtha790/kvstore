package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPRequest(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:       LevelInfo,
		EnableJSON:  true,
		EnableColor: false,
	}

	logger := createTestLogger(t, &buf, config)

	// Create a test HTTP request
	req := httptest.NewRequest("GET", "/api/users", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	req.Header.Set("User-Agent", "test-agent/1.0")

	duration := 150 * time.Millisecond
	statusCode := 200

	logger.HTTPRequest(req, statusCode, duration)

	output := buf.String()
	var logEntry map[string]interface{}

	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	if err != nil {
		t.Fatalf("Failed to parse JSON log output: %v", err)
	}

	// Verify all expected fields are present
	expectedFields := map[string]interface{}{
		"msg":         "HTTP request",
		"method":      "GET",
		"path":        "/api/users",
		"status":      float64(200), // JSON numbers are float64
		"duration_ms": float64(150),
		"remote_addr": "192.168.1.1:12345",
		"user_agent":  "test-agent/1.0",
	}

	for key, expectedValue := range expectedFields {
		if logEntry[key] != expectedValue {
			t.Errorf("Expected %s=%v, got %v", key, expectedValue, logEntry[key])
		}
	}
}

func TestHTTPError(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:       LevelError,
		EnableJSON:  true,
		EnableColor: false,
	}

	logger := createTestLogger(t, &buf, config)

	// Create a test HTTP request
	req := httptest.NewRequest("POST", "/api/users", nil)
	req.RemoteAddr = "10.0.0.1:54321"

	testError := errors.New("validation failed")
	statusCode := 400

	logger.HTTPError(req, testError, statusCode)

	output := buf.String()
	var logEntry map[string]interface{}

	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	if err != nil {
		t.Fatalf("Failed to parse JSON log output: %v", err)
	}

	// Verify all expected fields are present
	expectedFields := map[string]interface{}{
		"msg":         "HTTP error",
		"method":      "POST",
		"path":        "/api/users",
		"status":      float64(400),
		"error":       "validation failed",
		"remote_addr": "10.0.0.1:54321",
	}

	for key, expectedValue := range expectedFields {
		if logEntry[key] != expectedValue {
			t.Errorf("Expected %s=%v, got %v", key, expectedValue, logEntry[key])
		}
	}
}

func TestDatabaseOperation(t *testing.T) {
	tests := []struct {
		name          string
		operation     string
		table         string
		duration      time.Duration
		err           error
		expectedLevel string
		expectedMsg   string
	}{
		{
			name:          "successful operation",
			operation:     "SELECT",
			table:         "users",
			duration:      50 * time.Millisecond,
			err:           nil,
			expectedLevel: "DEBUG",
			expectedMsg:   "Database operation",
		},
		{
			name:          "failed operation",
			operation:     "INSERT",
			table:         "orders",
			duration:      200 * time.Millisecond,
			err:           errors.New("duplicate key"),
			expectedLevel: "ERROR",
			expectedMsg:   "Database operation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			config := Config{
				Level:       LevelDebug,
				EnableJSON:  true,
				EnableColor: false,
			}

			logger := createTestLogger(t, &buf, config)
			ctx := context.Background()

			logger.DatabaseOperation(ctx, tt.operation, tt.table, tt.duration, tt.err)

			output := buf.String()
			var logEntry map[string]interface{}

			err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
			if err != nil {
				t.Fatalf("Failed to parse JSON log output: %v", err)
			}

			if logEntry["msg"] != tt.expectedMsg {
				t.Errorf("Expected msg=%s, got %v", tt.expectedMsg, logEntry["msg"])
			}

			if logEntry["level"] != tt.expectedLevel {
				t.Errorf("Expected level=%s, got %v", tt.expectedLevel, logEntry["level"])
			}

			if logEntry["operation"] != tt.operation {
				t.Errorf("Expected operation=%s, got %v", tt.operation, logEntry["operation"])
			}

			if logEntry["table"] != tt.table {
				t.Errorf("Expected table=%s, got %v", tt.table, logEntry["table"])
			}

			if logEntry["duration_ms"] != float64(tt.duration.Milliseconds()) {
				t.Errorf("Expected duration_ms=%v, got %v", tt.duration.Milliseconds(), logEntry["duration_ms"])
			}

			if tt.err != nil {
				if logEntry["error"] != tt.err.Error() {
					t.Errorf("Expected error=%s, got %v", tt.err.Error(), logEntry["error"])
				}
			}
		})
	}
}

func TestStartupInfo(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:       LevelInfo,
		EnableJSON:  true,
		EnableColor: false,
	}

	logger := createTestLogger(t, &buf, config)

	appName := "test-app"
	version := "1.2.3"
	port := "8080"

	logger.StartupInfo(appName, version, port)

	output := buf.String()
	var logEntry map[string]interface{}

	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	if err != nil {
		t.Fatalf("Failed to parse JSON log output: %v", err)
	}

	expectedFields := map[string]interface{}{
		"msg":     "Application starting",
		"app":     appName,
		"version": version,
		"port":    port,
	}

	for key, expectedValue := range expectedFields {
		if logEntry[key] != expectedValue {
			t.Errorf("Expected %s=%v, got %v", key, expectedValue, logEntry[key])
		}
	}
}

func TestShutdownInfo(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:       LevelInfo,
		EnableJSON:  true,
		EnableColor: false,
	}

	logger := createTestLogger(t, &buf, config)

	appName := "test-app"
	duration := 500 * time.Millisecond

	logger.ShutdownInfo(appName, duration)

	output := buf.String()
	var logEntry map[string]interface{}

	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	if err != nil {
		t.Fatalf("Failed to parse JSON log output: %v", err)
	}

	expectedFields := map[string]interface{}{
		"msg":                  "Application shutdown",
		"app":                  appName,
		"shutdown_duration_ms": float64(500),
	}

	for key, expectedValue := range expectedFields {
		if logEntry[key] != expectedValue {
			t.Errorf("Expected %s=%v, got %v", key, expectedValue, logEntry[key])
		}
	}
}

func TestUserAction(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:       LevelInfo,
		EnableJSON:  true,
		EnableColor: false,
	}

	logger := createTestLogger(t, &buf, config)
	ctx := context.Background()

	userID := "user123"
	action := "login"
	metadata := map[string]any{
		"ip_address": "192.168.1.100",
		"success":    true,
		"attempts":   1,
	}

	logger.UserAction(ctx, userID, action, metadata)

	output := buf.String()
	var logEntry map[string]interface{}

	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	if err != nil {
		t.Fatalf("Failed to parse JSON log output: %v", err)
	}

	if logEntry["msg"] != "User action" {
		t.Errorf("Expected msg='User action', got %v", logEntry["msg"])
	}

	if logEntry["user_id"] != userID {
		t.Errorf("Expected user_id=%s, got %v", userID, logEntry["user_id"])
	}

	if logEntry["action"] != action {
		t.Errorf("Expected action=%s, got %v", action, logEntry["action"])
	}

	// Verify metadata fields
	if logEntry["ip_address"] != "192.168.1.100" {
		t.Errorf("Expected ip_address=192.168.1.100, got %v", logEntry["ip_address"])
	}

	if logEntry["success"] != true {
		t.Errorf("Expected success=true, got %v", logEntry["success"])
	}

	if logEntry["attempts"] != float64(1) {
		t.Errorf("Expected attempts=1, got %v", logEntry["attempts"])
	}
}

func TestSecurityEvent(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:       LevelWarn,
		EnableJSON:  true,
		EnableColor: false,
	}

	logger := createTestLogger(t, &buf, config)
	ctx := context.Background()

	event := "failed_login_attempt"
	userID := "user456"
	ipAddress := "192.168.1.200"
	severity := "high"

	logger.SecurityEvent(ctx, event, userID, ipAddress, severity)

	output := buf.String()
	var logEntry map[string]interface{}

	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	if err != nil {
		t.Fatalf("Failed to parse JSON log output: %v", err)
	}

	expectedFields := map[string]interface{}{
		"msg":        "Security event",
		"event":      event,
		"user_id":    userID,
		"ip_address": ipAddress,
		"severity":   severity,
	}

	for key, expectedValue := range expectedFields {
		if logEntry[key] != expectedValue {
			t.Errorf("Expected %s=%v, got %v", key, expectedValue, logEntry[key])
		}
	}
}

func TestPerformance(t *testing.T) {
	tests := []struct {
		name          string
		operation     string
		duration      time.Duration
		metadata      map[string]any
		expectedLevel string
		expectedMsg   string
	}{
		{
			name:          "fast operation",
			operation:     "cache_lookup",
			duration:      50 * time.Millisecond,
			metadata:      map[string]any{"cache_hit": true},
			expectedLevel: "DEBUG",
			expectedMsg:   "Performance metric",
		},
		{
			name:          "slow operation",
			operation:     "database_query",
			duration:      1500 * time.Millisecond,
			metadata:      map[string]any{"rows_affected": 1000},
			expectedLevel: "WARN",
			expectedMsg:   "Slow operation detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			config := Config{
				Level:       LevelDebug,
				EnableJSON:  true,
				EnableColor: false,
			}

			logger := createTestLogger(t, &buf, config)
			ctx := context.Background()

			logger.Performance(ctx, tt.operation, tt.duration, tt.metadata)

			output := buf.String()
			var logEntry map[string]interface{}

			err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
			if err != nil {
				t.Fatalf("Failed to parse JSON log output: %v", err)
			}

			if logEntry["msg"] != tt.expectedMsg {
				t.Errorf("Expected msg=%s, got %v", tt.expectedMsg, logEntry["msg"])
			}

			if logEntry["level"] != tt.expectedLevel {
				t.Errorf("Expected level=%s, got %v", tt.expectedLevel, logEntry["level"])
			}

			if logEntry["operation"] != tt.operation {
				t.Errorf("Expected operation=%s, got %v", tt.operation, logEntry["operation"])
			}

			expectedDuration := float64(tt.duration.Milliseconds())
			if logEntry["duration_ms"] != expectedDuration {
				t.Errorf("Expected duration_ms=%v, got %v", expectedDuration, logEntry["duration_ms"])
			}

			// Verify metadata (handle type conversions for JSON)
			for key, expectedValue := range tt.metadata {
				actualValue := logEntry[key]

				// Handle numeric type conversions for JSON
				switch expected := expectedValue.(type) {
				case int:
					if actual, ok := actualValue.(float64); ok {
						if float64(expected) != actual {
							t.Errorf("Expected %s=%v, got %v", key, expected, actual)
						}
					} else if actualValue != expectedValue {
						t.Errorf("Expected %s=%v, got %v", key, expectedValue, actualValue)
					}
				default:
					if actualValue != expectedValue {
						t.Errorf("Expected %s=%v, got %v", key, expectedValue, actualValue)
					}
				}
			}
		})
	}
}

func TestRecovery(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:       LevelError,
		EnableJSON:  true,
		EnableColor: false,
	}

	logger := createTestLogger(t, &buf, config)

	panicValue := "unexpected nil pointer"
	stackTrace := []byte("goroutine 1 [running]:\nmain.main()\n\t/app/main.go:10 +0x64")

	logger.Recovery(panicValue, stackTrace)

	output := buf.String()
	var logEntry map[string]interface{}

	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	if err != nil {
		t.Fatalf("Failed to parse JSON log output: %v", err)
	}

	if logEntry["msg"] != "Panic recovered" {
		t.Errorf("Expected msg='Panic recovered', got %v", logEntry["msg"])
	}

	if logEntry["panic"] != panicValue {
		t.Errorf("Expected panic=%s, got %v", panicValue, logEntry["panic"])
	}

	if logEntry["stack"] != string(stackTrace) {
		t.Errorf("Expected stack trace in log output")
	}
}

// Test global helper functions
func TestGlobalHelperFunctions(t *testing.T) {
	resetGlobalLogger()

	var buf bytes.Buffer
	config := Config{
		Level:       LevelDebug, // Use Debug level to capture all logs
		EnableJSON:  false,
		EnableColor: false,
	}

	testLogger := createTestLogger(t, &buf, config)
	defaultLogger = testLogger

	// Test HTTP request helper
	req := httptest.NewRequest("GET", "/test", nil)
	HTTPRequestDefault(req, 200, 100*time.Millisecond)

	if !strings.Contains(buf.String(), "HTTP request") {
		t.Error("HTTPRequestDefault did not log")
	}

	buf.Reset()

	// Test HTTP error helper
	HTTPErrorDefault(req, errors.New("test error"), 500)

	if !strings.Contains(buf.String(), "HTTP error") {
		t.Error("HTTPErrorDefault did not log")
	}

	buf.Reset()

	// Test startup info helper
	StartupInfoDefault("test-app", "1.0.0", "8080")

	if !strings.Contains(buf.String(), "Application starting") {
		t.Error("StartupInfoDefault did not log")
	}

	buf.Reset()

	// Test shutdown info helper
	ShutdownInfoDefault("test-app", 100*time.Millisecond)

	if !strings.Contains(buf.String(), "Application shutdown") {
		t.Error("ShutdownInfoDefault did not log")
	}

	buf.Reset()

	// Test database operation helper
	ctx := context.Background()
	DatabaseOperationDefault(ctx, "SELECT", "users", 50*time.Millisecond, nil)

	if !strings.Contains(buf.String(), "Database operation") {
		t.Error("DatabaseOperationDefault did not log")
	}

	buf.Reset()

	// Test user action helper
	UserActionDefault(ctx, "user123", "login", map[string]any{"success": true})

	if !strings.Contains(buf.String(), "User action") {
		t.Error("UserActionDefault did not log")
	}

	buf.Reset()

	// Test security event helper
	SecurityEventDefault(ctx, "failed_login", "user123", "192.168.1.1", "medium")

	if !strings.Contains(buf.String(), "Security event") {
		t.Error("SecurityEventDefault did not log")
	}

	buf.Reset()

	// Test performance helper
	PerformanceDefault(ctx, "api_call", 150*time.Millisecond, map[string]any{"endpoint": "/users"})

	if !strings.Contains(buf.String(), "Performance metric") {
		t.Error("PerformanceDefault did not log")
	}

	buf.Reset()

	// Test recovery helper
	RecoveryDefault("test panic", []byte("stack trace"))

	if !strings.Contains(buf.String(), "Panic recovered") {
		t.Error("RecoveryDefault did not log")
	}
}
