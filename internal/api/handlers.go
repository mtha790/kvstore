package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"kvstore/internal/store"
	"kvstore/pkg/logger"
)

// Handler holds the dependencies for API handlers
type Handler struct {
	store  store.Store
	logger *logger.Logger
}

// NewHandler creates a new Handler instance with dependencies
func NewHandler(s store.Store, l *logger.Logger) *Handler {
	return &Handler{
		store:  s,
		logger: l,
	}
}

// Response types for API handlers
type ErrorResponse struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

type GetResponse struct {
	Key   string      `json:"key"`
	Value store.Value `json:"value"`
}

type ListResponse struct {
	Keys []string `json:"keys"`
}

type SetRequest struct {
	Value string `json:"value"`
}

type SetResponse struct {
	Key     string      `json:"key"`
	Value   store.Value `json:"value"`
	Created bool        `json:"created"`
}

type DeleteResponse struct {
	Key     string      `json:"key"`
	Value   store.Value `json:"value"`
	Deleted bool        `json:"deleted"`
}

// extractKey extracts the key from URL path
func extractKey(path string) string {
	// Remove /api/kv/ prefix and get the key
	prefix := "/api/kv/"
	if strings.HasPrefix(path, prefix) {
		return strings.TrimPrefix(path, prefix)
	}
	return ""
}

// writeJSON writes JSON response with proper content type
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// writeError writes error response with proper content type
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Message: message})
}

// GetKey handles GET /api/kv/{key} - retrieve value
func (h *Handler) GetKey(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Extract key from URL
	key := extractKey(r.URL.Path)
	if key == "" {
		h.logger.WarnContext(ctx, "GetKey: invalid key", "path", r.URL.Path)
		writeError(w, http.StatusBadRequest, "invalid key")
		return
	}

	storeKey := store.Key(key)
	if err := storeKey.Validate(); err != nil {
		h.logger.WarnContext(ctx, "GetKey: key validation failed", "key", key, "error", err)
		writeError(w, http.StatusBadRequest, "invalid key")
		return
	}

	// Get value from store
	value, err := h.store.Get(ctx, storeKey)
	if err != nil {
		if err == store.ErrKeyNotFound {
			h.logger.InfoContext(ctx, "GetKey: key not found", "key", key)
			writeError(w, http.StatusNotFound, "key not found")
			return
		}
		h.logger.ErrorContext(ctx, "GetKey: store error", "key", key, "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	h.logger.InfoContext(ctx, "GetKey: success", "key", key)
	writeJSON(w, http.StatusOK, GetResponse{
		Key:   key,
		Value: value,
	})
}

// SetKey handles POST/PUT /api/kv/{key} - create/update value
func (h *Handler) SetKey(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Extract key from URL
	key := extractKey(r.URL.Path)
	if key == "" {
		h.logger.WarnContext(ctx, "SetKey: invalid key", "path", r.URL.Path)
		writeError(w, http.StatusBadRequest, "invalid key")
		return
	}

	storeKey := store.Key(key)
	if err := storeKey.Validate(); err != nil {
		h.logger.WarnContext(ctx, "SetKey: key validation failed", "key", key, "error", err)
		writeError(w, http.StatusBadRequest, "invalid key")
		return
	}

	// Parse request body
	var req SetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WarnContext(ctx, "SetKey: invalid JSON body", "key", key, "error", err)
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// Validate value
	if req.Value == "" {
		h.logger.WarnContext(ctx, "SetKey: empty value", "key", key)
		writeError(w, http.StatusBadRequest, "value cannot be empty")
		return
	}

	// Check if key exists to determine if this is a create or update
	exists, err := h.store.Exists(ctx, storeKey)
	if err != nil {
		h.logger.ErrorContext(ctx, "SetKey: store error checking existence", "key", key, "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Set value in store
	if err := h.store.Set(ctx, storeKey, req.Value); err != nil {
		h.logger.ErrorContext(ctx, "SetKey: store error", "key", key, "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Get the updated value to return
	value, err := h.store.Get(ctx, storeKey)
	if err != nil {
		h.logger.ErrorContext(ctx, "SetKey: error getting updated value", "key", key, "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	status := http.StatusOK
	created := false
	if !exists {
		status = http.StatusCreated
		created = true
	}

	h.logger.InfoContext(ctx, "SetKey: success", "key", key, "created", created)
	writeJSON(w, status, SetResponse{
		Key:     key,
		Value:   value,
		Created: created,
	})
}

// DeleteKey handles DELETE /api/kv/{key} - delete key
func (h *Handler) DeleteKey(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Extract key from URL
	key := extractKey(r.URL.Path)
	if key == "" {
		h.logger.WarnContext(ctx, "DeleteKey: invalid key", "path", r.URL.Path)
		writeError(w, http.StatusBadRequest, "invalid key")
		return
	}

	storeKey := store.Key(key)
	if err := storeKey.Validate(); err != nil {
		h.logger.WarnContext(ctx, "DeleteKey: key validation failed", "key", key, "error", err)
		writeError(w, http.StatusBadRequest, "invalid key")
		return
	}

	// Delete from store
	value, err := h.store.Delete(ctx, storeKey)
	if err != nil {
		if err == store.ErrKeyNotFound {
			h.logger.InfoContext(ctx, "DeleteKey: key not found", "key", key)
			writeError(w, http.StatusNotFound, "key not found")
			return
		}
		h.logger.ErrorContext(ctx, "DeleteKey: store error", "key", key, "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	h.logger.InfoContext(ctx, "DeleteKey: success", "key", key)
	writeJSON(w, http.StatusOK, DeleteResponse{
		Key:     key,
		Value:   value,
		Deleted: true,
	})
}

// ListKeys handles GET /api/kv - list all keys
func (h *Handler) ListKeys(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Get all keys from store
	keys, err := h.store.List(ctx)
	if err != nil {
		h.logger.ErrorContext(ctx, "ListKeys: store error", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Convert to string slice
	stringKeys := make([]string, len(keys))
	for i, key := range keys {
		stringKeys[i] = key.String()
	}

	h.logger.InfoContext(ctx, "ListKeys: success", "count", len(stringKeys))
	writeJSON(w, http.StatusOK, ListResponse{
		Keys: stringKeys,
	})
}
