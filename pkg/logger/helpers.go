package logger

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// HTTPRequest log une requête HTTP entrante
func (l *Logger) HTTPRequest(r *http.Request, statusCode int, duration time.Duration) {
	l.Info("HTTP request",
		"method", r.Method,
		"path", r.URL.Path,
		"status", statusCode,
		"duration_ms", duration.Milliseconds(),
		"remote_addr", r.RemoteAddr,
		"user_agent", r.UserAgent(),
	)
}

// HTTPError log une erreur HTTP avec détails
func (l *Logger) HTTPError(r *http.Request, err error, statusCode int) {
	l.Error("HTTP error",
		"method", r.Method,
		"path", r.URL.Path,
		"status", statusCode,
		"error", err.Error(),
		"remote_addr", r.RemoteAddr,
	)
}

// DatabaseOperation log une opération de base de données
func (l *Logger) DatabaseOperation(ctx context.Context, operation, table string, duration time.Duration, err error) {
	if err != nil {
		l.ErrorContext(ctx, "Database operation failed",
			"operation", operation,
			"table", table,
			"duration_ms", duration.Milliseconds(),
			"error", err.Error(),
		)
	} else {
		l.DebugContext(ctx, "Database operation",
			"operation", operation,
			"table", table,
			"duration_ms", duration.Milliseconds(),
		)
	}
}

// StartupInfo log les informations de démarrage de l'application
func (l *Logger) StartupInfo(appName, version, port string) {
	l.Info("Application starting",
		"app", appName,
		"version", version,
		"port", port,
	)
}

// ShutdownInfo log les informations d'arrêt de l'application
func (l *Logger) ShutdownInfo(appName string, duration time.Duration) {
	l.Info("Application shutdown",
		"app", appName,
		"shutdown_duration_ms", duration.Milliseconds(),
	)
}

// UserAction log une action utilisateur
func (l *Logger) UserAction(ctx context.Context, userID, action string, metadata map[string]any) {
	args := []any{
		"user_id", userID,
		"action", action,
	}

	// Ajouter les métadonnées
	for k, v := range metadata {
		args = append(args, k, v)
	}

	l.InfoContext(ctx, "User action", args...)
}

// SecurityEvent log un événement de sécurité
func (l *Logger) SecurityEvent(ctx context.Context, event, userID, ipAddress string, severity string) {
	l.WarnContext(ctx, "Security event",
		"event", event,
		"user_id", userID,
		"ip_address", ipAddress,
		"severity", severity,
	)
}

// Performance log des métriques de performance
func (l *Logger) Performance(ctx context.Context, operation string, duration time.Duration, metadata map[string]any) {
	args := []any{
		"operation", operation,
		"duration_ms", duration.Milliseconds(),
	}

	// Ajouter les métadonnées
	for k, v := range metadata {
		args = append(args, k, v)
	}

	if duration > 1000*time.Millisecond {
		l.WarnContext(ctx, "Slow operation detected", args...)
	} else {
		l.DebugContext(ctx, "Performance metric", args...)
	}
}

// Recovery log la récupération après une panique
func (l *Logger) Recovery(r any, stack []byte) {
	l.Error("Panic recovered",
		"panic", fmt.Sprintf("%v", r),
		"stack", string(stack),
	)
}

// Fonctions de convenance pour le logger par défaut

// HTTPRequestDefault log une requête HTTP avec le logger par défaut
func HTTPRequestDefault(r *http.Request, statusCode int, duration time.Duration) {
	Default().HTTPRequest(r, statusCode, duration)
}

// HTTPErrorDefault log une erreur HTTP avec le logger par défaut
func HTTPErrorDefault(r *http.Request, err error, statusCode int) {
	Default().HTTPError(r, err, statusCode)
}

// DatabaseOperationDefault log une opération de base de données avec le logger par défaut
func DatabaseOperationDefault(ctx context.Context, operation, table string, duration time.Duration, err error) {
	Default().DatabaseOperation(ctx, operation, table, duration, err)
}

// StartupInfoDefault log les informations de démarrage avec le logger par défaut
func StartupInfoDefault(appName, version, port string) {
	Default().StartupInfo(appName, version, port)
}

// ShutdownInfoDefault log les informations d'arrêt avec le logger par défaut
func ShutdownInfoDefault(appName string, duration time.Duration) {
	Default().ShutdownInfo(appName, duration)
}

// UserActionDefault log une action utilisateur avec le logger par défaut
func UserActionDefault(ctx context.Context, userID, action string, metadata map[string]any) {
	Default().UserAction(ctx, userID, action, metadata)
}

// SecurityEventDefault log un événement de sécurité avec le logger par défaut
func SecurityEventDefault(ctx context.Context, event, userID, ipAddress string, severity string) {
	Default().SecurityEvent(ctx, event, userID, ipAddress, severity)
}

// PerformanceDefault log des métriques de performance avec le logger par défaut
func PerformanceDefault(ctx context.Context, operation string, duration time.Duration, metadata map[string]any) {
	Default().Performance(ctx, operation, duration, metadata)
}

// RecoveryDefault log la récupération après une panique avec le logger par défaut
func RecoveryDefault(r any, stack []byte) {
	Default().Recovery(r, stack)
}
