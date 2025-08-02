package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

// LogLevel représente le niveau de logging
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

// Config contient la configuration du logger
type Config struct {
	Level       LogLevel
	OutputFile  string
	EnableJSON  bool
	EnableColor bool
}

// Logger encapsule slog avec des fonctionnalités supplémentaires
type Logger struct {
	logger *slog.Logger
	config Config
	mu     sync.RWMutex
}

var (
	defaultLogger *Logger
	once          sync.Once
)

// New crée une nouvelle instance de logger
func New(config Config) (*Logger, error) {
	var writers []io.Writer

	// Sortie console
	writers = append(writers, os.Stdout)

	// Sortie fichier si spécifiée
	if config.OutputFile != "" {
		// Créer le répertoire parent si nécessaire
		if err := os.MkdirAll(filepath.Dir(config.OutputFile), 0755); err != nil {
			return nil, err
		}

		file, err := os.OpenFile(config.OutputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, err
		}
		writers = append(writers, file)
	}

	// Créer un MultiWriter pour écrire dans tous les outputs
	multiWriter := io.MultiWriter(writers...)

	// Configurer les options slog
	opts := &slog.HandlerOptions{
		Level:     mapLogLevel(config.Level),
		AddSource: true,
	}

	var handler slog.Handler
	if config.EnableJSON {
		handler = slog.NewJSONHandler(multiWriter, opts)
	} else {
		handler = slog.NewTextHandler(multiWriter, opts)
	}

	logger := &Logger{
		logger: slog.New(handler),
		config: config,
	}

	return logger, nil
}

// Init initialise le logger par défaut
func Init(config Config) error {
	var err error
	once.Do(func() {
		defaultLogger, err = New(config)
	})
	return err
}

// Default retourne le logger par défaut
func Default() *Logger {
	if defaultLogger == nil {
		// Configuration par défaut si pas initialisé
		config := Config{
			Level:       LevelInfo,
			EnableJSON:  false,
			EnableColor: true,
		}
		defaultLogger, _ = New(config)
	}
	return defaultLogger
}

// mapLogLevel convertit notre LogLevel vers slog.Level
func mapLogLevel(level LogLevel) slog.Level {
	switch level {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Debug log un message de debug
func (l *Logger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

// Info log un message d'information
func (l *Logger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

// Warn log un message d'avertissement
func (l *Logger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

// Error log un message d'erreur
func (l *Logger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

// DebugContext log un message de debug avec contexte
func (l *Logger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.logger.DebugContext(ctx, msg, args...)
}

// InfoContext log un message d'information avec contexte
func (l *Logger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.logger.InfoContext(ctx, msg, args...)
}

// WarnContext log un message d'avertissement avec contexte
func (l *Logger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.logger.WarnContext(ctx, msg, args...)
}

// ErrorContext log un message d'erreur avec contexte
func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.logger.ErrorContext(ctx, msg, args...)
}

// With retourne un nouveau logger avec des attributs supplémentaires
func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		logger: l.logger.With(args...),
		config: l.config,
	}
}

// WithGroup retourne un nouveau logger avec un groupe d'attributs
func (l *Logger) WithGroup(name string) *Logger {
	return &Logger{
		logger: l.logger.WithGroup(name),
		config: l.config,
	}
}

// SetLevel change le niveau de logging
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.config.Level = level
}

// GetLevel retourne le niveau de logging actuel
func (l *Logger) GetLevel() LogLevel {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.config.Level
}

// Enabled vérifie si un niveau de log est activé
func (l *Logger) Enabled(level LogLevel) bool {
	return l.logger.Enabled(context.Background(), mapLogLevel(level))
}
