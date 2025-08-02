package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kvstore/pkg/logger"
)

func TestLoggingMiddleware(t *testing.T) {
	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": "test response"}`))
	})

	// Create logger
	l := logger.Default()

	// Create middleware
	middleware := LoggingMiddleware(l)
	handler := middleware(testHandler)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/api/kv/test", nil)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("Content-Type", "application/json")

	// Create response recorder
	rec := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(rec, req)

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "test response") {
		t.Errorf("expected response to contain 'test response', got %s", rec.Body.String())
	}
}

func TestCORSMiddleware(t *testing.T) {
	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Create middleware
	handler := CORSMiddleware(testHandler)

	tests := []struct {
		name           string
		method         string
		expectedStatus int
	}{
		{
			name:           "GET request",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "OPTIONS request (preflight)",
			method:         http.MethodOptions,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST request",
			method:         http.MethodPost,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test request
			req := httptest.NewRequest(tt.method, "/api/kv/test", nil)

			// Create response recorder
			rec := httptest.NewRecorder()

			// Execute request
			handler.ServeHTTP(rec, req)

			// Verify status
			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			// Verify CORS headers
			expectedHeaders := map[string]string{
				"Access-Control-Allow-Origin":  "*",
				"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
				"Access-Control-Allow-Headers": "Content-Type, Authorization",
			}

			for header, expectedValue := range expectedHeaders {
				if value := rec.Header().Get(header); value != expectedValue {
					t.Errorf("expected header %s to be %s, got %s", header, expectedValue, value)
				}
			}
		})
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	// Create a test handler that panics
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	// Create logger
	l := logger.Default()

	// Create middleware
	middleware := RecoveryMiddleware(l)
	handler := middleware(panicHandler)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/api/kv/test", nil)

	// Create response recorder
	rec := httptest.NewRecorder()

	// Execute request (should not panic)
	handler.ServeHTTP(rec, req)

	// Verify response
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	// Verify error response format
	if !strings.Contains(rec.Body.String(), "internal server error") {
		t.Errorf("expected response to contain error message, got %s", rec.Body.String())
	}

	// Verify content type
	if contentType := rec.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}
}

func TestResponseWriter(t *testing.T) {
	// Create response recorder
	rec := httptest.NewRecorder()

	// Create wrapped response writer
	rw := newResponseWriter(rec)

	// Test writing
	data := []byte("test data")
	n, err := rw.Write(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != len(data) {
		t.Errorf("expected to write %d bytes, wrote %d", len(data), n)
	}

	// Test status code
	rw.WriteHeader(http.StatusNotFound)
	if rw.statusCode != http.StatusNotFound {
		t.Errorf("expected status code %d, got %d", http.StatusNotFound, rw.statusCode)
	}

	// Test captured body
	if rw.body.String() != string(data) {
		t.Errorf("expected body %s, got %s", string(data), rw.body.String())
	}

	// Test size tracking
	if rw.size != len(data) {
		t.Errorf("expected size %d, got %d", len(data), rw.size)
	}
}
