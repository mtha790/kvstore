package api

import (
	"bytes"
	"net/http"
	"time"

	"kvstore/pkg/logger"
)

// responseWriter wraps http.ResponseWriter to capture status code and response size
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
	body       *bytes.Buffer
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // default status
		body:           new(bytes.Buffer),
	}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	// Capture response body for logging
	rw.body.Write(data)
	size, err := rw.ResponseWriter.Write(data)
	rw.size += size
	return size, err
}

// LoggingMiddleware logs HTTP requests and responses
func LoggingMiddleware(l *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Log incoming request
			l.InfoContext(r.Context(), "HTTP Request",
				"method", r.Method,
				"path", r.URL.Path,
				"query", r.URL.RawQuery,
				"remote_addr", r.RemoteAddr,
				"user_agent", r.Header.Get("User-Agent"),
				"content_type", r.Header.Get("Content-Type"),
			)

			// Wrap response writer to capture response details
			rw := newResponseWriter(w)

			// Process request
			next.ServeHTTP(rw, r)

			// Calculate duration
			duration := time.Since(start)

			// Log response
			l.InfoContext(r.Context(), "HTTP Response",
				"method", r.Method,
				"path", r.URL.Path,
				"status_code", rw.statusCode,
				"response_size", rw.size,
				"duration_ms", duration.Milliseconds(),
				"duration", duration.String(),
			)

			// Log response body for debugging (only for errors or debug level)
			if rw.statusCode >= 400 && l.Enabled(logger.LevelDebug) {
				l.DebugContext(r.Context(), "HTTP Error Response Body",
					"method", r.Method,
					"path", r.URL.Path,
					"status_code", rw.statusCode,
					"response_body", rw.body.String(),
				)
			}
		})
	}
}

// CORSMiddleware adds CORS headers for browser compatibility
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RecoveryMiddleware recovers from panics and logs them
func RecoveryMiddleware(l *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					l.ErrorContext(r.Context(), "HTTP Panic Recovery",
						"method", r.Method,
						"path", r.URL.Path,
						"panic", err,
					)
					writeError(w, http.StatusInternalServerError, "internal server error")
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
