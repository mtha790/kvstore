package logger

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
)

func TestInit(t *testing.T) {
	// Reset global state for testing
	resetGlobalLogger()

	tempDir := t.TempDir()
	config := Config{
		Level:       LevelDebug,
		OutputFile:  tempDir + "/global_test.log",
		EnableJSON:  true,
		EnableColor: false,
	}

	err := Init(config)
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Test that default logger is initialized
	if defaultLogger == nil {
		t.Error("defaultLogger should not be nil after Init()")
	}

	// Test that level is set correctly
	if defaultLogger.GetLevel() != LevelDebug {
		t.Errorf("Expected level %v, got %v", LevelDebug, defaultLogger.GetLevel())
	}
}

func TestInitOnce(t *testing.T) {
	// Reset global state for testing
	resetGlobalLogger()

	config1 := Config{Level: LevelDebug}
	config2 := Config{Level: LevelError}

	// Initialize twice with different configs
	err1 := Init(config1)
	err2 := Init(config2)

	if err1 != nil {
		t.Errorf("First Init() failed: %v", err1)
	}
	if err2 != nil {
		t.Errorf("Second Init() failed: %v", err2)
	}

	// The level should be from the first initialization
	if defaultLogger.GetLevel() != LevelDebug {
		t.Errorf("Expected level from first Init(), got %v", defaultLogger.GetLevel())
	}
}

func TestDefault(t *testing.T) {
	// Reset global state for testing
	resetGlobalLogger()

	// Test Default() without Init()
	logger := Default()
	if logger == nil {
		t.Error("Default() should not return nil")
	}

	// Should create default configuration
	if logger.GetLevel() != LevelInfo {
		t.Errorf("Expected default level %v, got %v", LevelInfo, logger.GetLevel())
	}
}

func TestGlobalDebug(t *testing.T) {
	resetGlobalLogger()

	// Capture output by replacing the default logger with a test logger
	var buf bytes.Buffer
	config := Config{
		Level:       LevelDebug,
		EnableJSON:  false,
		EnableColor: false,
	}

	testLogger := createTestLogger(t, &buf, config)
	defaultLogger = testLogger

	Debug("global debug message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "global debug message") {
		t.Error("Global Debug message not found in output")
	}
}

func TestGlobalInfo(t *testing.T) {
	resetGlobalLogger()

	var buf bytes.Buffer
	config := Config{
		Level:       LevelInfo,
		EnableJSON:  false,
		EnableColor: false,
	}

	testLogger := createTestLogger(t, &buf, config)
	defaultLogger = testLogger

	Info("global info message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "global info message") {
		t.Error("Global Info message not found in output")
	}
}

func TestGlobalWarn(t *testing.T) {
	resetGlobalLogger()

	var buf bytes.Buffer
	config := Config{
		Level:       LevelWarn,
		EnableJSON:  false,
		EnableColor: false,
	}

	testLogger := createTestLogger(t, &buf, config)
	defaultLogger = testLogger

	Warn("global warn message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "global warn message") {
		t.Error("Global Warn message not found in output")
	}
}

func TestGlobalError(t *testing.T) {
	resetGlobalLogger()

	var buf bytes.Buffer
	config := Config{
		Level:       LevelError,
		EnableJSON:  false,
		EnableColor: false,
	}

	testLogger := createTestLogger(t, &buf, config)
	defaultLogger = testLogger

	Error("global error message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "global error message") {
		t.Error("Global Error message not found in output")
	}
}

func TestGlobalContextMethods(t *testing.T) {
	resetGlobalLogger()

	var buf bytes.Buffer
	config := Config{
		Level:       LevelDebug,
		EnableJSON:  false,
		EnableColor: false,
	}

	testLogger := createTestLogger(t, &buf, config)
	defaultLogger = testLogger

	ctx := context.Background()

	DebugContext(ctx, "global debug context")
	InfoContext(ctx, "global info context")
	WarnContext(ctx, "global warn context")
	ErrorContext(ctx, "global error context")

	output := buf.String()

	if !strings.Contains(output, "global debug context") {
		t.Error("Global DebugContext message not found")
	}
	if !strings.Contains(output, "global info context") {
		t.Error("Global InfoContext message not found")
	}
	if !strings.Contains(output, "global warn context") {
		t.Error("Global WarnContext message not found")
	}
	if !strings.Contains(output, "global error context") {
		t.Error("Global ErrorContext message not found")
	}
}

func TestGlobalWith(t *testing.T) {
	resetGlobalLogger()

	var buf bytes.Buffer
	config := Config{
		Level:       LevelInfo,
		EnableJSON:  true,
		EnableColor: false,
	}

	testLogger := createTestLogger(t, &buf, config)
	defaultLogger = testLogger

	childLogger := With("service", "global-test")
	childLogger.Info("test with global")

	output := buf.String()
	if !strings.Contains(output, "service") {
		t.Error("Expected 'service' attribute in global With() output")
	}
	if !strings.Contains(output, "global-test") {
		t.Error("Expected 'global-test' value in global With() output")
	}
}

func TestGlobalWithGroup(t *testing.T) {
	resetGlobalLogger()

	var buf bytes.Buffer
	config := Config{
		Level:       LevelInfo,
		EnableJSON:  true,
		EnableColor: false,
	}

	testLogger := createTestLogger(t, &buf, config)
	defaultLogger = testLogger

	groupLogger := WithGroup("api")
	groupLogger.Info("request processed", "status", 200)

	output := buf.String()
	if !strings.Contains(output, "api") {
		t.Error("Expected 'api' group in global WithGroup() output")
	}
}

func TestGlobalConcurrency(t *testing.T) {
	resetGlobalLogger()

	var buf bytes.Buffer
	config := Config{
		Level:       LevelInfo,
		EnableJSON:  false,
		EnableColor: false,
	}

	testLogger := createTestLogger(t, &buf, config)
	defaultLogger = testLogger

	const numGoroutines = 20
	const logsPerGoroutine = 5

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Test concurrent access to global logger functions
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < logsPerGoroutine; j++ {
				Info("concurrent global log", "goroutine", id, "iteration", j)
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

// Helper function to reset global logger state for testing
func resetGlobalLogger() {
	defaultLogger = nil
	once = sync.Once{}
}
