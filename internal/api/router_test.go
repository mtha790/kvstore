package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"kvstore/internal/store"
	"kvstore/pkg/logger"
)

func TestRouter(t *testing.T) {
	// Setup
	mockStore := newMockStore()
	logger := logger.Default()
	router := NewRouter(mockStore, logger)

	// Test data
	testKey := "test-key"
	testValue := "test-value"

	// Add test data to store
	mockStore.data[store.Key(testKey)] = store.Value{
		Data:      testValue,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Version:   1,
	}

	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "GET existing key",
			method:         http.MethodGet,
			path:           "/api/kv/" + testKey,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "GET non-existing key",
			method:         http.MethodGet,
			path:           "/api/kv/non-existing",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "POST new key",
			method:         http.MethodPost,
			path:           "/api/kv/new-key",
			body:           `{"value": "new-value"}`,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "PUT update key",
			method:         http.MethodPut,
			path:           "/api/kv/" + testKey,
			body:           `{"value": "updated-value"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "DELETE existing key",
			method:         http.MethodDelete,
			path:           "/api/kv/" + testKey,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "GET list keys",
			method:         http.MethodGet,
			path:           "/api/kv",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid method on list endpoint",
			method:         http.MethodPost,
			path:           "/api/kv",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Invalid method on key endpoint",
			method:         http.MethodPatch,
			path:           "/api/kv/" + testKey,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Unknown endpoint",
			method:         http.MethodGet,
			path:           "/api/unknown",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "List keys with trailing slash",
			method:         http.MethodGet,
			path:           "/api/kv/",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body *bytes.Buffer
			if tt.body != "" {
				body = bytes.NewBufferString(tt.body)
			} else {
				body = bytes.NewBuffer(nil)
			}

			req := httptest.NewRequest(tt.method, tt.path, body)
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			// Verify content type is JSON
			if contentType := rec.Header().Get("Content-Type"); contentType != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", contentType)
			}
		})
	}
}

func TestSetupRoutes(t *testing.T) {
	// Setup
	mockStore := newMockStore()
	logger := logger.Default()
	handler := SetupRoutes(mockStore, logger)

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "Health check",
			method:         http.MethodGet,
			path:           "/health",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "KV API route",
			method:         http.MethodGet,
			path:           "/api/kv",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "KV API with key",
			method:         http.MethodGet,
			path:           "/api/kv/test-key",
			expectedStatus: http.StatusNotFound, // Key doesn't exist
		},
		{
			name:           "Invalid health check method",
			method:         http.MethodPost,
			path:           "/health",
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestHealthCheck(t *testing.T) {
	// Setup
	mockStore := newMockStore()
	logger := logger.Default()
	router := NewRouter(mockStore, logger)

	// Test GET /health
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	router.HealthCheck(rec, req)

	// Verify status
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Verify content type
	if contentType := rec.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}

	// Verify response body
	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if status, ok := response["status"]; !ok || status != "healthy" {
		t.Errorf("expected status 'healthy', got %v", status)
	}

	if service, ok := response["service"]; !ok || service != "key-value-store" {
		t.Errorf("expected service 'key-value-store', got %v", service)
	}
}

func TestRouterMiddlewareChain(t *testing.T) {
	// Setup
	mockStore := newMockStore()
	logger := logger.Default()
	router := NewRouter(mockStore, logger)

	// Test request
	req := httptest.NewRequest(http.MethodGet, "/api/kv", nil)
	rec := httptest.NewRecorder()

	// Execute
	router.ServeHTTP(rec, req)

	// Verify CORS headers are present (showing middleware chain works)
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

	// Verify successful response
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}
