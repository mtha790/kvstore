package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"kvstore/internal/config"
	"kvstore/pkg/logger"
)

// Test mapping function from config LogLevel to logger LogLevel
func TestMapLogLevel(t *testing.T) {
	tests := []struct {
		name        string
		configLevel config.LogLevel
		want        logger.LogLevel
	}{
		{
			name:        "debug level",
			configLevel: config.LogLevelDebug,
			want:        logger.LevelDebug,
		},
		{
			name:        "info level",
			configLevel: config.LogLevelInfo,
			want:        logger.LevelInfo,
		},
		{
			name:        "warn level",
			configLevel: config.LogLevelWarn,
			want:        logger.LevelWarn,
		},
		{
			name:        "error level",
			configLevel: config.LogLevelError,
			want:        logger.LevelError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapLogLevel(tt.configLevel)
			if got != tt.want {
				t.Errorf("mapLogLevel(%v) = %v, want %v", tt.configLevel, got, tt.want)
			}
		})
	}
}

// Test Application creation and initialization
func TestNewApplication(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.Config
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config",
			config: &config.Config{
				HTTPPort:        8080,
				HTTPHost:        "localhost",
				LogLevel:        config.LogLevelInfo,
				PersistenceType: config.PersistenceMemory,
			},
			wantErr: false,
		},
		{
			name:        "nil config",
			config:      nil,
			wantErr:     true,
			errContains: "config cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := NewApplication(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
					return
				}
				if tt.errContains != "" && err.Error() != tt.errContains {
					t.Errorf("expected error to contain %q, got %q", tt.errContains, err.Error())
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if app == nil {
				t.Error("expected non-nil application")
				return
			}
			if app.config != tt.config {
				t.Error("config not properly set")
			}
			if app.store == nil {
				t.Error("store not initialized")
			}
		})
	}
}

// Test persistence setup based on config
func TestApplication_setupPersistence(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := fmt.Sprintf("%s/test.json", tempDir)

	tests := []struct {
		name            string
		persistenceType config.PersistenceType
		persistencePath string
		wantErr         bool
		errContains     string
	}{
		{
			name:            "memory persistence",
			persistenceType: config.PersistenceMemory,
			wantErr:         false,
		},
		{
			name:            "file persistence",
			persistenceType: config.PersistenceFile,
			persistencePath: tempFile,
			wantErr:         false,
		},
		{
			name:            "unsupported persistence",
			persistenceType: config.PersistenceDB,
			wantErr:         true,
			errContains:     "unsupported persistence type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &config.Config{
				HTTPPort:        8080,
				HTTPHost:        "localhost",
				LogLevel:        config.LogLevelInfo,
				PersistenceType: tt.persistenceType,
				PersistencePath: tt.persistencePath,
			}

			app, err := NewApplication(config)
			if err != nil {
				t.Fatalf("failed to create application: %v", err)
			}

			err = app.setupPersistence()
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
					return
				}
				if tt.errContains != "" && err.Error() != tt.errContains {
					t.Errorf("expected error to contain %q, got %q", tt.errContains, err.Error())
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// Test graceful shutdown
func TestApplication_Shutdown(t *testing.T) {
	config := &config.Config{
		HTTPPort:        8080,
		HTTPHost:        "localhost",
		LogLevel:        config.LogLevelInfo,
		PersistenceType: config.PersistenceMemory,
	}

	app, err := NewApplication(config)
	if err != nil {
		t.Fatalf("failed to create application: %v", err)
	}

	// Test shutdown with context timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = app.Shutdown(ctx)
	if err != nil {
		t.Errorf("unexpected error during shutdown: %v", err)
	}
}

// Test HTTP server setup
func TestApplication_setupHTTPServer(t *testing.T) {
	config := &config.Config{
		HTTPPort:        8080,
		HTTPHost:        "localhost",
		LogLevel:        config.LogLevelInfo,
		PersistenceType: config.PersistenceMemory,
	}

	app, err := NewApplication(config)
	if err != nil {
		t.Fatalf("failed to create application: %v", err)
	}

	server := app.setupHTTPServer()
	if server == nil {
		t.Error("expected non-nil HTTP server")
		return
	}

	if server.Addr != config.Address() {
		t.Errorf("expected server address %s, got %s", config.Address(), server.Addr)
	}
}
