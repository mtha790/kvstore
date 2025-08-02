package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "basic console logger",
			config: Config{
				Level:       LevelInfo,
				EnableJSON:  false,
				EnableColor: true,
			},
			wantErr: false,
		},
		{
			name: "JSON logger with file output",
			config: Config{
				Level:      LevelDebug,
				OutputFile: filepath.Join(os.TempDir(), "test.log"),
				EnableJSON: true,
			},
			wantErr: false,
		},
		{
			name: "invalid file path",
			config: Config{
				Level:      LevelInfo,
				OutputFile: "/invalid/path/that/does/not/exist/test.log",
				EnableJSON: false,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := New(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("New() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("New() unexpected error: %v", err)
				return
			}

			if logger == nil {
				t.Error("New() returned nil logger")
				return
			}

			// Verify logger configuration
			if logger.config.Level != tt.config.Level {
				t.Errorf("New() config.Level = %v, want %v", logger.config.Level, tt.config.Level)
			}

			if logger.config.EnableJSON != tt.config.EnableJSON {
				t.Errorf("New() config.EnableJSON = %v, want %v", logger.config.EnableJSON, tt.config.EnableJSON)
			}

			// Clean up test files
			if tt.config.OutputFile != "" && !tt.wantErr {
				os.Remove(tt.config.OutputFile)
			}
		})
	}
}

func TestLogLevels(t *testing.T) {
	tests := []struct {
		name      string
		logLevel  LogLevel
		logMethod string
		shouldLog bool
	}{
		// Debug level tests
		{"debug level allows debug", LevelDebug, "debug", true},
		{"debug level allows info", LevelDebug, "info", true},
		{"debug level allows warn", LevelDebug, "warn", true},
		{"debug level allows error", LevelDebug, "error", true},

		// Info level tests
		{"info level blocks debug", LevelInfo, "debug", false},
		{"info level allows info", LevelInfo, "info", true},
		{"info level allows warn", LevelInfo, "warn", true},
		{"info level allows error", LevelInfo, "error", true},

		// Warn level tests
		{"warn level blocks debug", LevelWarn, "debug", false},
		{"warn level blocks info", LevelWarn, "info", false},
		{"warn level allows warn", LevelWarn, "warn", true},
		{"warn level allows error", LevelWarn, "error", true},

		// Error level tests
		{"error level blocks debug", LevelError, "debug", false},
		{"error level blocks info", LevelError, "info", false},
		{"error level blocks warn", LevelError, "warn", false},
		{"error level allows error", LevelError, "error", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			config := Config{
				Level:       tt.logLevel,
				EnableJSON:  false,
				EnableColor: false,
			}

			// Create logger with buffer output for testing
			logger := createTestLogger(t, &buf, config)

			// Execute the test method based on string
			switch tt.logMethod {
			case "debug":
				logger.Debug("test message")
			case "info":
				logger.Info("test message")
			case "warn":
				logger.Warn("test message")
			case "error":
				logger.Error("test message")
			}

			output := buf.String()

			if tt.shouldLog && output == "" {
				t.Errorf("Expected log output but got none")
			}

			if !tt.shouldLog && output != "" {
				t.Errorf("Expected no log output but got: %s", output)
			}
		})
	}
}

func TestLoggerMethods(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:       LevelDebug,
		EnableJSON:  false,
		EnableColor: false,
	}

	logger := createTestLogger(t, &buf, config)

	// Test basic logging methods
	logger.Debug("debug message", "key", "value")
	logger.Info("info message", "key", "value")
	logger.Warn("warn message", "key", "value")
	logger.Error("error message", "key", "value")

	output := buf.String()

	if !strings.Contains(output, "debug message") {
		t.Error("Debug message not found in output")
	}
	if !strings.Contains(output, "info message") {
		t.Error("Info message not found in output")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("Warn message not found in output")
	}
	if !strings.Contains(output, "error message") {
		t.Error("Error message not found in output")
	}
}

func TestLoggerContextMethods(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:       LevelDebug,
		EnableJSON:  false,
		EnableColor: false,
	}

	logger := createTestLogger(t, &buf, config)
	ctx := context.Background()

	// Test context-aware logging methods
	logger.DebugContext(ctx, "debug context message")
	logger.InfoContext(ctx, "info context message")
	logger.WarnContext(ctx, "warn context message")
	logger.ErrorContext(ctx, "error context message")

	output := buf.String()

	if !strings.Contains(output, "debug context message") {
		t.Error("Debug context message not found in output")
	}
	if !strings.Contains(output, "info context message") {
		t.Error("Info context message not found in output")
	}
	if !strings.Contains(output, "warn context message") {
		t.Error("Warn context message not found in output")
	}
	if !strings.Contains(output, "error context message") {
		t.Error("Error context message not found in output")
	}
}

func TestLoggerWith(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:       LevelInfo,
		EnableJSON:  true,
		EnableColor: false,
	}

	logger := createTestLogger(t, &buf, config)

	// Test With method
	childLogger := logger.With("component", "test", "version", "1.0")
	childLogger.Info("test message")

	output := buf.String()

	// Parse JSON output to verify attributes
	var logEntry map[string]interface{}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) > 0 {
		err := json.Unmarshal([]byte(lines[0]), &logEntry)
		if err != nil {
			t.Fatalf("Failed to parse JSON log output: %v", err)
		}

		if logEntry["component"] != "test" {
			t.Errorf("Expected component=test, got %v", logEntry["component"])
		}

		if logEntry["version"] != "1.0" {
			t.Errorf("Expected version=1.0, got %v", logEntry["version"])
		}

		if logEntry["msg"] != "test message" {
			t.Errorf("Expected msg='test message', got %v", logEntry["msg"])
		}
	}
}

func TestLoggerWithGroup(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:       LevelInfo,
		EnableJSON:  true,
		EnableColor: false,
	}

	logger := createTestLogger(t, &buf, config)

	// Test WithGroup method
	groupLogger := logger.WithGroup("database")
	groupLogger.Info("query executed", "duration", "100ms")

	output := buf.String()

	// Parse JSON output to verify group structure
	var logEntry map[string]interface{}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) > 0 {
		err := json.Unmarshal([]byte(lines[0]), &logEntry)
		if err != nil {
			t.Fatalf("Failed to parse JSON log output: %v", err)
		}

		// Check if database group exists
		if database, ok := logEntry["database"].(map[string]interface{}); ok {
			if database["duration"] != "100ms" {
				t.Errorf("Expected database.duration=100ms, got %v", database["duration"])
			}
		} else {
			t.Error("Expected 'database' group not found in log output")
		}
	}
}

func TestSetGetLevel(t *testing.T) {
	logger, err := New(Config{
		Level:       LevelInfo,
		EnableJSON:  false,
		EnableColor: false,
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Test initial level
	if logger.GetLevel() != LevelInfo {
		t.Errorf("Expected initial level %v, got %v", LevelInfo, logger.GetLevel())
	}

	// Test setting level
	logger.SetLevel(LevelError)
	if logger.GetLevel() != LevelError {
		t.Errorf("Expected level %v after SetLevel, got %v", LevelError, logger.GetLevel())
	}
}

func TestEnabled(t *testing.T) {
	logger, err := New(Config{
		Level:       LevelWarn,
		EnableJSON:  false,
		EnableColor: false,
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	tests := []struct {
		level   LogLevel
		enabled bool
	}{
		{LevelDebug, false},
		{LevelInfo, false},
		{LevelWarn, true},
		{LevelError, true},
	}

	for _, tt := range tests {
		if logger.Enabled(tt.level) != tt.enabled {
			t.Errorf("Expected Enabled(%v) = %v, got %v", tt.level, tt.enabled, logger.Enabled(tt.level))
		}
	}
}

func TestConcurrentLogging(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:       LevelInfo,
		EnableJSON:  false,
		EnableColor: false,
	}

	logger := createTestLogger(t, &buf, config)

	const numGoroutines = 50
	const logsPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Launch multiple goroutines that log concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < logsPerGoroutine; j++ {
				logger.Info("concurrent log", "goroutine", id, "iteration", j)
			}
		}(i)
	}

	wg.Wait()

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	expectedLines := numGoroutines * logsPerGoroutine
	if len(lines) != expectedLines {
		t.Errorf("Expected %d log lines, got %d", expectedLines, len(lines))
	}
}

func TestFileOutput(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	config := Config{
		Level:       LevelInfo,
		OutputFile:  logFile,
		EnableJSON:  false,
		EnableColor: false,
	}

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger with file output: %v", err)
	}

	logger.Info("test file message", "key", "value")

	// Wait a bit for the write to complete
	time.Sleep(10 * time.Millisecond)

	// Read the file content
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), "test file message") {
		t.Error("Log message not found in file output")
	}
}

// Benchmark tests
func BenchmarkLoggerInfo(b *testing.B) {
	var buf bytes.Buffer
	config := Config{
		Level:       LevelInfo,
		EnableJSON:  false,
		EnableColor: false,
	}

	logger := createTestLogger(b, &buf, config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", "iteration", i, "timestamp", time.Now())
	}
}

func BenchmarkLoggerInfoJSON(b *testing.B) {
	var buf bytes.Buffer
	config := Config{
		Level:       LevelInfo,
		EnableJSON:  true,
		EnableColor: false,
	}

	logger := createTestLogger(b, &buf, config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", "iteration", i, "timestamp", time.Now())
	}
}

func BenchmarkLoggerWith(b *testing.B) {
	var buf bytes.Buffer
	config := Config{
		Level:       LevelInfo,
		EnableJSON:  false,
		EnableColor: false,
	}

	logger := createTestLogger(b, &buf, config)
	childLogger := logger.With("service", "benchmark", "version", "1.0")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		childLogger.Info("benchmark message", "iteration", i)
	}
}

func BenchmarkLoggerConcurrent(b *testing.B) {
	var buf bytes.Buffer
	config := Config{
		Level:       LevelInfo,
		EnableJSON:  false,
		EnableColor: false,
	}

	logger := createTestLogger(b, &buf, config)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			logger.Info("concurrent benchmark", "goroutine", i)
			i++
		}
	})
}

func BenchmarkGlobalLogger(b *testing.B) {
	resetGlobalLogger()

	var buf bytes.Buffer
	config := Config{
		Level:       LevelInfo,
		EnableJSON:  false,
		EnableColor: false,
	}

	testLogger := createTestLogger(b, &buf, config)
	defaultLogger = testLogger

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Info("global benchmark message", "iteration", i)
	}
}

// Helper function that works with both testing.T and testing.B
func createTestLogger(_ testing.TB, writer io.Writer, config Config) *Logger {
	opts := &slog.HandlerOptions{
		Level:     mapLogLevel(config.Level),
		AddSource: true,
	}

	var handler slog.Handler
	if config.EnableJSON {
		handler = slog.NewJSONHandler(writer, opts)
	} else {
		handler = slog.NewTextHandler(writer, opts)
	}

	return &Logger{
		logger: slog.New(handler),
		config: config,
	}
}
