package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"kvstore/internal/store"
	"kvstore/pkg/logger"
)

// mockStore implements store.Store interface for testing
type mockStore struct {
	data map[store.Key]store.Value
	err  error
}

func newMockStore() *mockStore {
	return &mockStore{
		data: make(map[store.Key]store.Value),
	}
}

func (m *mockStore) Get(ctx context.Context, key store.Key) (store.Value, error) {
	if m.err != nil {
		return store.Value{}, m.err
	}
	val, exists := m.data[key]
	if !exists {
		return store.Value{}, store.ErrKeyNotFound
	}
	return val, nil
}

func (m *mockStore) Set(ctx context.Context, key store.Key, value string) error {
	if m.err != nil {
		return m.err
	}
	if err := key.Validate(); err != nil {
		return err
	}

	now := time.Now()
	existing, exists := m.data[key]
	if exists {
		m.data[key] = store.Value{
			Data:      value,
			CreatedAt: existing.CreatedAt,
			UpdatedAt: now,
			Version:   existing.Version + 1,
		}
	} else {
		m.data[key] = store.Value{
			Data:      value,
			CreatedAt: now,
			UpdatedAt: now,
			Version:   1,
		}
	}
	return nil
}

func (m *mockStore) Delete(ctx context.Context, key store.Key) (store.Value, error) {
	if m.err != nil {
		return store.Value{}, m.err
	}
	val, exists := m.data[key]
	if !exists {
		return store.Value{}, store.ErrKeyNotFound
	}
	delete(m.data, key)
	return val, nil
}

func (m *mockStore) List(ctx context.Context) ([]store.Key, error) {
	if m.err != nil {
		return nil, m.err
	}
	keys := make([]store.Key, 0, len(m.data))
	for key := range m.data {
		keys = append(keys, key)
	}
	return keys, nil
}

func (m *mockStore) ListEntries(ctx context.Context) ([]store.Entry, error) {
	if m.err != nil {
		return nil, m.err
	}
	entries := make([]store.Entry, 0, len(m.data))
	for key, value := range m.data {
		entries = append(entries, store.Entry{Key: key, Value: value})
	}
	return entries, nil
}

func (m *mockStore) Size(ctx context.Context) (int, error) {
	if m.err != nil {
		return 0, m.err
	}
	return len(m.data), nil
}

func (m *mockStore) Clear(ctx context.Context) error {
	if m.err != nil {
		return m.err
	}
	m.data = make(map[store.Key]store.Value)
	return nil
}

func (m *mockStore) Exists(ctx context.Context, key store.Key) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	_, exists := m.data[key]
	return exists, nil
}

func (m *mockStore) CompareAndSwap(ctx context.Context, key store.Key, expectedVersion int64, newValue string) (store.Value, error) {
	if m.err != nil {
		return store.Value{}, m.err
	}
	val, exists := m.data[key]
	if !exists {
		return store.Value{}, store.ErrKeyNotFound
	}
	if val.Version != expectedVersion {
		return val, store.ErrConcurrentModification
	}

	now := time.Now()
	newVal := store.Value{
		Data:      newValue,
		CreatedAt: val.CreatedAt,
		UpdatedAt: now,
		Version:   val.Version + 1,
	}
	m.data[key] = newVal
	return newVal, nil
}

func (m *mockStore) Close() error {
	return nil
}

// Test response types are defined in handlers.go

func TestGetHandler(t *testing.T) {
	tests := []struct {
		name           string
		key            string
		setupStore     func(*mockStore)
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name: "get existing key",
			key:  "test-key",
			setupStore: func(ms *mockStore) {
				ms.data[store.Key("test-key")] = store.Value{
					Data:      "test-value",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
					Version:   1,
				}
			},
			expectedStatus: http.StatusOK,
			expectedBody: GetResponse{
				Key: "test-key",
				Value: store.Value{
					Data:    "test-value",
					Version: 1,
				},
			},
		},
		{
			name:           "get non-existing key",
			key:            "non-existing",
			setupStore:     func(ms *mockStore) {},
			expectedStatus: http.StatusNotFound,
			expectedBody: ErrorResponse{
				Message: "key not found",
			},
		},
		{
			name:           "get with invalid key",
			key:            "",
			setupStore:     func(ms *mockStore) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: ErrorResponse{
				Message: "invalid key",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockStore := newMockStore()
			tt.setupStore(mockStore)

			handler := NewHandler(mockStore, logger.Default())

			// Create request
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/kv/%s", tt.key), nil)
			rec := httptest.NewRecorder()

			// Execute
			handler.GetKey(rec, req)

			// Assert status code
			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			// Assert content type
			expectedContentType := "application/json"
			if contentType := rec.Header().Get("Content-Type"); contentType != expectedContentType {
				t.Errorf("expected Content-Type %s, got %s", expectedContentType, contentType)
			}

			// Assert response body based on expected type
			switch expected := tt.expectedBody.(type) {
			case GetResponse:
				var actual GetResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &actual); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if actual.Key != expected.Key {
					t.Errorf("expected key %s, got %s", expected.Key, actual.Key)
				}
				if actual.Value.Data != expected.Value.Data {
					t.Errorf("expected data %s, got %s", expected.Value.Data, actual.Value.Data)
				}
			case ErrorResponse:
				var actual ErrorResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &actual); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if actual.Message != expected.Message {
					t.Errorf("expected message %s, got %s", expected.Message, actual.Message)
				}
			}
		})
	}
}

func TestSetHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		key            string
		requestBody    SetRequest
		setupStore     func(*mockStore)
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name:   "create new key with POST",
			method: http.MethodPost,
			key:    "new-key",
			requestBody: SetRequest{
				Value: "new-value",
			},
			setupStore:     func(ms *mockStore) {},
			expectedStatus: http.StatusCreated,
			expectedBody: SetResponse{
				Key: "new-key",
				Value: store.Value{
					Data:    "new-value",
					Version: 1,
				},
				Created: true,
			},
		},
		{
			name:   "update existing key with PUT",
			method: http.MethodPut,
			key:    "existing-key",
			requestBody: SetRequest{
				Value: "updated-value",
			},
			setupStore: func(ms *mockStore) {
				ms.data[store.Key("existing-key")] = store.Value{
					Data:      "old-value",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
					Version:   1,
				}
			},
			expectedStatus: http.StatusOK,
			expectedBody: SetResponse{
				Key: "existing-key",
				Value: store.Value{
					Data:    "updated-value",
					Version: 2,
				},
				Created: false,
			},
		},
		{
			name:           "invalid JSON body",
			method:         http.MethodPost,
			key:            "test-key",
			setupStore:     func(ms *mockStore) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: ErrorResponse{
				Message: "invalid JSON body",
			},
		},
		{
			name:   "empty value",
			method: http.MethodPost,
			key:    "test-key",
			requestBody: SetRequest{
				Value: "",
			},
			setupStore:     func(ms *mockStore) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: ErrorResponse{
				Message: "value cannot be empty",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockStore := newMockStore()
			tt.setupStore(mockStore)

			handler := NewHandler(mockStore, logger.Default())

			// Create request body
			var body *bytes.Buffer
			if tt.name == "invalid JSON body" {
				body = bytes.NewBufferString("invalid json")
			} else {
				bodyBytes, _ := json.Marshal(tt.requestBody)
				body = bytes.NewBuffer(bodyBytes)
			}

			// Create request
			req := httptest.NewRequest(tt.method, fmt.Sprintf("/api/kv/%s", tt.key), body)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			// Execute
			handler.SetKey(rec, req)

			// Assert status code
			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			// Assert content type
			expectedContentType := "application/json"
			if contentType := rec.Header().Get("Content-Type"); contentType != expectedContentType {
				t.Errorf("expected Content-Type %s, got %s", expectedContentType, contentType)
			}

			// Assert response body based on expected type
			switch expected := tt.expectedBody.(type) {
			case SetResponse:
				var actual SetResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &actual); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if actual.Key != expected.Key {
					t.Errorf("expected key %s, got %s", expected.Key, actual.Key)
				}
				if actual.Value.Data != expected.Value.Data {
					t.Errorf("expected data %s, got %s", expected.Value.Data, actual.Value.Data)
				}
				if actual.Created != expected.Created {
					t.Errorf("expected created %v, got %v", expected.Created, actual.Created)
				}
			case ErrorResponse:
				var actual ErrorResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &actual); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if actual.Message != expected.Message {
					t.Errorf("expected message %s, got %s", expected.Message, actual.Message)
				}
			}
		})
	}
}

func TestDeleteHandler(t *testing.T) {
	tests := []struct {
		name           string
		key            string
		setupStore     func(*mockStore)
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name: "delete existing key",
			key:  "existing-key",
			setupStore: func(ms *mockStore) {
				ms.data[store.Key("existing-key")] = store.Value{
					Data:      "test-value",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
					Version:   1,
				}
			},
			expectedStatus: http.StatusOK,
			expectedBody: DeleteResponse{
				Key: "existing-key",
				Value: store.Value{
					Data:    "test-value",
					Version: 1,
				},
				Deleted: true,
			},
		},
		{
			name:           "delete non-existing key",
			key:            "non-existing",
			setupStore:     func(ms *mockStore) {},
			expectedStatus: http.StatusNotFound,
			expectedBody: ErrorResponse{
				Message: "key not found",
			},
		},
		{
			name:           "delete with invalid key",
			key:            "",
			setupStore:     func(ms *mockStore) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: ErrorResponse{
				Message: "invalid key",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockStore := newMockStore()
			tt.setupStore(mockStore)

			handler := NewHandler(mockStore, logger.Default())

			// Create request
			req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/kv/%s", tt.key), nil)
			rec := httptest.NewRecorder()

			// Execute
			handler.DeleteKey(rec, req)

			// Assert status code
			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			// Assert content type
			expectedContentType := "application/json"
			if contentType := rec.Header().Get("Content-Type"); contentType != expectedContentType {
				t.Errorf("expected Content-Type %s, got %s", expectedContentType, contentType)
			}

			// Assert response body based on expected type
			switch expected := tt.expectedBody.(type) {
			case DeleteResponse:
				var actual DeleteResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &actual); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if actual.Key != expected.Key {
					t.Errorf("expected key %s, got %s", expected.Key, actual.Key)
				}
				if actual.Value.Data != expected.Value.Data {
					t.Errorf("expected data %s, got %s", expected.Value.Data, actual.Value.Data)
				}
				if actual.Deleted != expected.Deleted {
					t.Errorf("expected deleted %v, got %v", expected.Deleted, actual.Deleted)
				}
			case ErrorResponse:
				var actual ErrorResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &actual); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if actual.Message != expected.Message {
					t.Errorf("expected message %s, got %s", expected.Message, actual.Message)
				}
			}
		})
	}
}

func TestListHandler(t *testing.T) {
	tests := []struct {
		name           string
		setupStore     func(*mockStore)
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name: "list keys with data",
			setupStore: func(ms *mockStore) {
				ms.data[store.Key("key1")] = store.Value{Data: "value1", Version: 1}
				ms.data[store.Key("key2")] = store.Value{Data: "value2", Version: 1}
				ms.data[store.Key("key3")] = store.Value{Data: "value3", Version: 1}
			},
			expectedStatus: http.StatusOK,
			expectedBody: ListResponse{
				Keys: []string{"key1", "key2", "key3"},
			},
		},
		{
			name:           "list keys with empty store",
			setupStore:     func(ms *mockStore) {},
			expectedStatus: http.StatusOK,
			expectedBody: ListResponse{
				Keys: []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockStore := newMockStore()
			tt.setupStore(mockStore)

			handler := NewHandler(mockStore, logger.Default())

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/api/kv", nil)
			rec := httptest.NewRecorder()

			// Execute
			handler.ListKeys(rec, req)

			// Assert status code
			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			// Assert content type
			expectedContentType := "application/json"
			if contentType := rec.Header().Get("Content-Type"); contentType != expectedContentType {
				t.Errorf("expected Content-Type %s, got %s", expectedContentType, contentType)
			}

			// Assert response body
			expected := tt.expectedBody.(ListResponse)
			var actual ListResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &actual); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			// For empty lists, ensure both are empty
			if len(expected.Keys) == 0 {
				if len(actual.Keys) != 0 {
					t.Errorf("expected empty keys list, got %v", actual.Keys)
				}
			} else {
				// Check that all expected keys are present (order doesn't matter for maps)
				if len(actual.Keys) != len(expected.Keys) {
					t.Errorf("expected %d keys, got %d", len(expected.Keys), len(actual.Keys))
				}

				expectedMap := make(map[string]bool)
				for _, key := range expected.Keys {
					expectedMap[key] = true
				}

				for _, key := range actual.Keys {
					if !expectedMap[key] {
						t.Errorf("unexpected key in response: %s", key)
					}
				}
			}
		})
	}
}
