package api

import (
	"net/http"
	"path/filepath"
	"strings"

	"kvstore/internal/store"
	"kvstore/pkg/logger"
)

// Router holds the HTTP router and dependencies
type Router struct {
	handler *Handler
	logger  *logger.Logger
}

// NewRouter creates a new Router with dependencies
func NewRouter(store store.Store, logger *logger.Logger) *Router {
	return &Router{
		handler: NewHandler(store, logger),
		logger:  logger,
	}
}

// ServeHTTP implements http.Handler interface to route requests
func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Apply middleware chain
	handler := rt.applyMiddleware(rt.routeRequest())
	handler.ServeHTTP(w, r)
}

// applyMiddleware applies all middleware in the correct order
func (rt *Router) applyMiddleware(handler http.Handler) http.Handler {
	// Apply middleware in reverse order (last applied is executed first)
	handler = RecoveryMiddleware(rt.logger)(handler)
	handler = LoggingMiddleware(rt.logger)(handler)
	handler = CORSMiddleware(handler)
	return handler
}

// routeRequest routes the request to the appropriate handler
func (rt *Router) routeRequest() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Handle /api/kv routes
		if strings.HasPrefix(path, "/api/kv") {
			rt.handleKVRoutes(w, r)
			return
		}

		// Handle 404 for unknown routes
		writeError(w, http.StatusNotFound, "endpoint not found")
	})
}

// handleKVRoutes handles all /api/kv/* routes
func (rt *Router) handleKVRoutes(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// GET /api/kv - list all keys
	if path == "/api/kv" {
		if r.Method == http.MethodGet {
			rt.handler.ListKeys(w, r)
			return
		}
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Handle /api/kv/ (trailing slash with no key)
	if path == "/api/kv/" {
		if r.Method == http.MethodGet {
			rt.handler.ListKeys(w, r)
			return
		}
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Routes with key parameter: /api/kv/{key}
	if strings.HasPrefix(path, "/api/kv/") {
		key := extractKey(path)
		if key == "" {
			writeError(w, http.StatusBadRequest, "invalid key")
			return
		}

		switch r.Method {
		case http.MethodGet:
			rt.handler.GetKey(w, r)
		case http.MethodPost, http.MethodPut:
			rt.handler.SetKey(w, r)
		case http.MethodDelete:
			rt.handler.DeleteKey(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	// Unknown route
	writeError(w, http.StatusNotFound, "endpoint not found")
}

// Health check endpoint
func (rt *Router) HealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	response := map[string]interface{}{
		"status":  "healthy",
		"service": "key-value-store",
	}

	writeJSON(w, http.StatusOK, response)
}

// SetupRoutes creates a complete HTTP handler with all routes
func SetupRoutes(store store.Store, logger *logger.Logger) http.Handler {
	router := NewRouter(store, logger)

	// Create a new ServeMux for additional routes
	mux := http.NewServeMux()

	// Register main KV API routes
	mux.Handle("/api/kv", router)
	mux.Handle("/api/kv/", router)

	// Register health check
	mux.HandleFunc("/health", router.HealthCheck)

	// Register API documentation routes
	mux.HandleFunc("/api/docs", DocsHandler)
	mux.HandleFunc("/docs/openapi.yaml", OpenAPIHandler)

	// Serve static files from web/static directory
	staticDir := filepath.Join("web", "static")
	fileServer := http.FileServer(http.Dir(staticDir))
	mux.Handle("/", fileServer)

	// Apply global middleware
	return router.applyMiddleware(mux)
}
