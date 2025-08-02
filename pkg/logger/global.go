package logger

import "context"

// Fonctions de convenance pour utiliser le logger par défaut

// Debug log un message de debug avec le logger par défaut
func Debug(msg string, args ...any) {
	Default().Debug(msg, args...)
}

// Info log un message d'information avec le logger par défaut
func Info(msg string, args ...any) {
	Default().Info(msg, args...)
}

// Warn log un message d'avertissement avec le logger par défaut
func Warn(msg string, args ...any) {
	Default().Warn(msg, args...)
}

// Error log un message d'erreur avec le logger par défaut
func Error(msg string, args ...any) {
	Default().Error(msg, args...)
}

// DebugContext log un message de debug avec contexte avec le logger par défaut
func DebugContext(ctx context.Context, msg string, args ...any) {
	Default().DebugContext(ctx, msg, args...)
}

// InfoContext log un message d'information avec contexte avec le logger par défaut
func InfoContext(ctx context.Context, msg string, args ...any) {
	Default().InfoContext(ctx, msg, args...)
}

// WarnContext log un message d'avertissement avec contexte avec le logger par défaut
func WarnContext(ctx context.Context, msg string, args ...any) {
	Default().WarnContext(ctx, msg, args...)
}

// ErrorContext log un message d'erreur avec contexte avec le logger par défaut
func ErrorContext(ctx context.Context, msg string, args ...any) {
	Default().ErrorContext(ctx, msg, args...)
}

// With retourne un nouveau logger avec des attributs supplémentaires
func With(args ...any) *Logger {
	return Default().With(args...)
}

// WithGroup retourne un nouveau logger avec un groupe d'attributs
func WithGroup(name string) *Logger {
	return Default().WithGroup(name)
}
