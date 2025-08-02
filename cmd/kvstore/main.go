package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kvstore/internal/api"
	"kvstore/internal/config"
	"kvstore/internal/store"
	"kvstore/pkg/logger"
)

// Application holds all the application components
type Application struct {
	config      *config.Config
	logger      *logger.Logger
	store       store.Store
	persistence store.Persistence
	httpServer  *http.Server
}

// NewApplication creates a new application instance
func NewApplication(cfg *config.Config) (*Application, error) {
	if cfg == nil {
		return nil, errors.New("config cannot be nil")
	}

	// Create logger
	loggerConfig := logger.Config{
		Level:      mapLogLevel(cfg.LogLevel),
		OutputFile: "",
		EnableJSON: false,
	}

	log, err := logger.New(loggerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	// Initialize global logger
	if err := logger.Init(loggerConfig); err != nil {
		return nil, fmt.Errorf("failed to initialize global logger: %w", err)
	}

	// Create store
	memStore := store.NewMemoryStore()

	app := &Application{
		config: cfg,
		logger: log,
		store:  memStore,
	}

	return app, nil
}

// mapLogLevel maps config.LogLevel to logger.LogLevel
func mapLogLevel(configLevel config.LogLevel) logger.LogLevel {
	switch configLevel {
	case config.LogLevelDebug:
		return logger.LevelDebug
	case config.LogLevelInfo:
		return logger.LevelInfo
	case config.LogLevelWarn:
		return logger.LevelWarn
	case config.LogLevelError:
		return logger.LevelError
	default:
		return logger.LevelInfo
	}
}

// setupPersistence configures persistence based on config and wraps the store
func (app *Application) setupPersistence() error {
	app.logger.Info("setting up persistence",
		"type", app.config.PersistenceType,
		"path", app.config.PersistencePath)

	switch app.config.PersistenceType {
	case config.PersistenceMemory:
		// No persistence needed for memory-only mode
		app.logger.Info("using memory-only persistence")
		return nil
	case config.PersistenceFile:
		app.logger.Info("setting up file persistence", "path", app.config.PersistencePath)
		app.persistence = store.NewJSONFilePersistence(app.config.PersistencePath)

		// Configure PersistentStore with sensible defaults
		persistentConfig := store.PersistentStoreConfig{
			AutoSave:       true,
			SaveInterval:   30 * time.Second,
			SaveOnShutdown: true,
			RetryAttempts:  3,
			RetryDelay:     1 * time.Second,
		}

		// Wrap the memory store with persistence
		app.logger.Info("creating persistent store wrapper")
		persistentStore, err := store.NewPersistentStore(app.store, app.persistence, persistentConfig)
		if err != nil {
			return fmt.Errorf("failed to create persistent store: %w", err)
		}

		// Replace the store with the persistent store
		app.store = persistentStore
		app.logger.Info("persistent store configured successfully")

		return nil
	case config.PersistenceDB:
		return errors.New("unsupported persistence type")
	default:
		return fmt.Errorf("unknown persistence type: %s", app.config.PersistenceType)
	}
}

// setupHTTPServer creates and configures the HTTP server
func (app *Application) setupHTTPServer() *http.Server {
	// Setup API routes with all middleware (logging, CORS, recovery)
	handler := api.SetupRoutes(app.store, app.logger)

	server := &http.Server{
		Addr:    app.config.Address(),
		Handler: handler,
	}

	app.httpServer = server
	return server
}

// Shutdown gracefully shuts down the application
func (app *Application) Shutdown(ctx context.Context) error {
	app.logger.Info("shutting down application")

	// Shutdown HTTP server if it exists
	if app.httpServer != nil {
		if err := app.httpServer.Shutdown(ctx); err != nil {
			app.logger.Error("failed to shutdown HTTP server", "error", err)
			return err
		}
	}

	// Persistence is handled automatically by PersistentStore on Close()

	// Close store
	if app.store != nil {
		if err := app.store.Close(); err != nil {
			app.logger.Error("failed to close store", "error", err)
			return err
		}
	}

	app.logger.Info("application shutdown complete")
	return nil
}

// Run starts the application
func (app *Application) Run() error {
	// Setup persistence
	if err := app.setupPersistence(); err != nil {
		return fmt.Errorf("failed to setup persistence: %w", err)
	}

	// Setup HTTP server
	server := app.setupHTTPServer()

	// Start HTTP server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		app.logger.Info("starting HTTP server", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		app.logger.Info("received shutdown signal")
	case err := <-serverErr:
		app.logger.Error("HTTP server error", "error", err)
		return err
	}

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return app.Shutdown(ctx)
}

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Create application
	app, err := NewApplication(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create application: %v\n", err)
		os.Exit(1)
	}

	// Run application
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "application error: %v\n", err)
		os.Exit(1)
	}
}
