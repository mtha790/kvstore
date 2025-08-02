// Package store defines persistence interfaces and types for store serialization
package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Persistence defines the interface for persisting and loading store data
// Implementations are responsible for serializing store snapshots to/from
// external storage (files, databases, etc.)
type Persistence interface {
	// Save persists the given store snapshot to external storage
	// Returns an error if the save operation fails
	Save(ctx context.Context, snapshot *StoreSnapshot) error

	// Load retrieves a store snapshot from external storage
	// Returns the snapshot and nil error on success
	// Returns nil snapshot and error if loading fails or no data exists
	Load(ctx context.Context) (*StoreSnapshot, error)
}

// StoreSnapshot represents a complete snapshot of the store's state
// This type is used for serialization and deserialization during persistence operations
type StoreSnapshot struct {
	// Data contains all key-value pairs in the store
	Data map[string]string `json:"data"`

	// Stats contains store statistics at the time of snapshot
	Stats StoreStats `json:"stats"`

	// Version identifies the snapshot format version for future compatibility
	Version string `json:"version"`

	// Timestamp records when the snapshot was created (Unix timestamp)
	Timestamp int64 `json:"timestamp"`
}

// PersistenceConfig holds configuration for persistence operations
type PersistenceConfig struct {
	// Path specifies the location for persistence (file path, database connection, etc.)
	Path string `json:"path"`

	// AutoSave enables automatic saving of snapshots on store modifications
	AutoSave bool `json:"auto_save"`

	// SaveInterval specifies the interval for periodic saves (in seconds)
	// Only used when AutoSave is true
	SaveInterval int `json:"save_interval"`

	// BackupEnabled enables creation of backup files before overwriting
	BackupEnabled bool `json:"backup_enabled"`

	// MaxBackups specifies the maximum number of backup files to keep
	MaxBackups int `json:"max_backups"`
}

// Persistence-specific errors
var (
	// ErrPersistenceNotConfigured indicates that persistence is not properly configured
	ErrPersistenceNotConfigured = errors.New("persistence not configured")

	// ErrSnapshotCorrupted indicates that the loaded snapshot data is corrupted or invalid
	ErrSnapshotCorrupted = errors.New("snapshot data is corrupted")

	// ErrUnsupportedVersion indicates that the snapshot version is not supported
	ErrUnsupportedVersion = errors.New("unsupported snapshot version")

	// ErrSaveOperationFailed indicates that the save operation failed
	ErrSaveOperationFailed = errors.New("save operation failed")

	// ErrLoadOperationFailed indicates that the load operation failed
	ErrLoadOperationFailed = errors.New("load operation failed")

	// ErrNoSnapshotFound indicates that no snapshot data was found
	ErrNoSnapshotFound = errors.New("no snapshot found")
)

// NewPersistenceError creates a wrapped persistence error with additional context
func NewPersistenceError(operation string, err error) error {
	return fmt.Errorf("persistence %s error: %w", operation, err)
}

// ValidateSnapshot validates that a snapshot contains required fields and valid data
func ValidateSnapshot(snapshot *StoreSnapshot) error {
	if snapshot == nil {
		return fmt.Errorf("snapshot is nil")
	}

	if snapshot.Data == nil {
		return fmt.Errorf("snapshot data is nil")
	}

	if snapshot.Version == "" {
		return fmt.Errorf("snapshot version is empty")
	}

	if snapshot.Timestamp <= 0 {
		return fmt.Errorf("snapshot timestamp is invalid")
	}

	return nil
}

// SnapshotSize returns the number of key-value pairs in the snapshot
func (s *StoreSnapshot) SnapshotSize() int {
	if s.Data == nil {
		return 0
	}
	return len(s.Data)
}

// IsEmpty returns true if the snapshot contains no data
func (s *StoreSnapshot) IsEmpty() bool {
	return s.SnapshotSize() == 0
}

// JSONFilePersistence implements file-based persistence using JSON format
// It provides thread-safe operations using a mutex to coordinate access
type JSONFilePersistence struct {
	filePath string
	mutex    sync.RWMutex // Protects file operations for thread safety
}

// NewJSONFilePersistence creates a new JSON file persistence instance
func NewJSONFilePersistence(filePath string) *JSONFilePersistence {
	return &JSONFilePersistence{
		filePath: filePath,
	}
}

// generateTempFileName creates a unique temporary file name to avoid conflicts
// This is critical for concurrent operations to prevent file collisions
func (j *JSONFilePersistence) generateTempFileName() (string, error) {
	// Generate a random hex string for uniqueness across concurrent operations
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	randomHex := hex.EncodeToString(randomBytes)
	return j.filePath + ".tmp." + randomHex, nil
}

// Save saves the store snapshot to a JSON file using atomic write operations
// This method is thread-safe and uses write locks to prevent concurrent modifications
func (j *JSONFilePersistence) Save(ctx context.Context, snapshot *StoreSnapshot) error {
	if snapshot == nil {
		return fmt.Errorf("snapshot is nil")
	}

	// Validate snapshot before expensive operations
	if err := ValidateSnapshot(snapshot); err != nil {
		return NewPersistenceError("save", err)
	}

	// Marshal snapshot to JSON before acquiring lock (optimization)
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return NewPersistenceError("save", fmt.Errorf("failed to marshal snapshot: %w", err))
	}

	// Use write lock to ensure only one save operation at a time
	j.mutex.Lock()
	defer j.mutex.Unlock()

	// Create directory if it doesn't exist
	dir := filepath.Dir(j.filePath)
	if dir != "." && dir != "/" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return NewPersistenceError("save", fmt.Errorf("failed to create directory: %w", err))
		}
	}

	// Generate unique temporary file name to avoid conflicts
	tempFile, err := j.generateTempFileName()
	if err != nil {
		return NewPersistenceError("save", fmt.Errorf("failed to generate temp filename: %w", err))
	}

	// Ensure we clean up temp file on any failure
	defer func() {
		if _, err := os.Stat(tempFile); err == nil {
			os.Remove(tempFile)
		}
	}()

	// Write to temporary file first
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return NewPersistenceError("save", fmt.Errorf("failed to write temp file: %w", err))
	}

	// Atomically replace the original file with the temporary file
	if err := os.Rename(tempFile, j.filePath); err != nil {
		return NewPersistenceError("save", fmt.Errorf("failed to rename temp file: %w", err))
	}

	return nil
}

// Load loads the store snapshot from a JSON file
// This method is thread-safe and uses read locks to allow concurrent reads
func (j *JSONFilePersistence) Load(ctx context.Context) (*StoreSnapshot, error) {
	// Use read lock to allow concurrent reads but exclude writes
	j.mutex.RLock()
	defer j.mutex.RUnlock()

	// Check if file exists
	if _, err := os.Stat(j.filePath); os.IsNotExist(err) {
		return nil, NewPersistenceError("load", ErrNoSnapshotFound)
	}

	// Read file contents
	data, err := os.ReadFile(j.filePath)
	if err != nil {
		return nil, NewPersistenceError("load", fmt.Errorf("failed to read file: %w", err))
	}

	// Early validation: check if we have any data
	if len(data) == 0 {
		return nil, NewPersistenceError("load", fmt.Errorf("file is empty"))
	}

	// Unmarshal JSON data
	var snapshot StoreSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, NewPersistenceError("load", fmt.Errorf("failed to unmarshal snapshot: %w", err))
	}

	// Validate loaded snapshot
	if err := ValidateSnapshot(&snapshot); err != nil {
		return nil, NewPersistenceError("load", ErrSnapshotCorrupted)
	}

	return &snapshot, nil
}
