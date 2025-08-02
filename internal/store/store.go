// Package store provides interfaces and types for concurrent key-value storage
package store

import (
	"context"
	"errors"
	"time"
)

// Common errors for the key-value store
var (
	// ErrKeyNotFound is returned when a requested key does not exist in the store
	ErrKeyNotFound = errors.New("key not found")

	// ErrInvalidKey is returned when an invalid key is provided
	ErrInvalidKey = errors.New("invalid key")

	// ErrInvalidValue is returned when an invalid value is provided
	ErrInvalidValue = errors.New("invalid value")

	// ErrStoreClosed is returned when attempting to operate on a closed store
	ErrStoreClosed = errors.New("store is closed")

	// ErrTimeout is returned when an operation times out
	ErrTimeout = errors.New("operation timed out")

	// ErrConcurrentModification is returned when a concurrent modification conflict occurs
	ErrConcurrentModification = errors.New("concurrent modification detected")
)

// Key represents a store key with validation
type Key string

// Value represents a store value with metadata
type Value struct {
	Data      string    `json:"data"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Version   int64     `json:"version"`
}

// Entry represents a complete key-value entry
type Entry struct {
	Key   Key   `json:"key"`
	Value Value `json:"value"`
}

// Store defines the interface for a concurrent key-value store
// All methods are designed to be safe for concurrent use by multiple goroutines
type Store interface {
	// Get retrieves the value associated with the given key
	// Returns the value and nil error if the key exists
	// Returns zero value and ErrKeyNotFound if the key does not exist
	// Context can be used for cancellation and timeout
	Get(ctx context.Context, key Key) (Value, error)

	// Set stores a key-value pair in the store
	// If the key already exists, it updates the value and increments the version
	// Returns an error if the operation fails
	// Context can be used for cancellation and timeout
	Set(ctx context.Context, key Key, value string) error

	// Delete removes a key-value pair from the store
	// Returns the deleted value and nil error if the key existed and was deleted
	// Returns zero value and ErrKeyNotFound if the key didn't exist
	// Context can be used for cancellation and timeout
	Delete(ctx context.Context, key Key) (Value, error)

	// List returns all keys currently stored in the key-value store
	// The returned slice is a snapshot at the time of the call
	// Returns an empty slice if no keys exist
	// Context can be used for cancellation and timeout
	List(ctx context.Context) ([]Key, error)

	// ListEntries returns all key-value entries currently stored
	// The returned slice is a snapshot at the time of the call
	// Returns an empty slice if no entries exist
	// Context can be used for cancellation and timeout
	ListEntries(ctx context.Context) ([]Entry, error)

	// Size returns the current number of key-value pairs in the store
	// This is a snapshot count at the time of the call
	// Context can be used for cancellation and timeout
	Size(ctx context.Context) (int, error)

	// Clear removes all key-value pairs from the store
	// This operation is atomic - either all entries are removed or none
	// Context can be used for cancellation and timeout
	Clear(ctx context.Context) error

	// Exists checks if a key exists in the store without retrieving the value
	// Returns true if the key exists, false otherwise
	// Context can be used for cancellation and timeout
	Exists(ctx context.Context, key Key) (bool, error)

	// CompareAndSwap atomically compares and swaps a value
	// Updates the value only if the current version matches expectedVersion
	// Returns the new value and nil error on success
	// Returns the current value and ErrConcurrentModification if versions don't match
	// Returns zero value and ErrKeyNotFound if the key doesn't exist
	CompareAndSwap(ctx context.Context, key Key, expectedVersion int64, newValue string) (Value, error)

	// Close closes the store and releases any resources
	// After calling Close, all other operations will return ErrStoreClosed
	// This method should be idempotent - safe to call multiple times
	Close() error
}

// TransactionalStore extends Store with transaction support
// Transactions provide ACID properties for multiple operations
type TransactionalStore interface {
	Store

	// BeginTx starts a new transaction
	// The returned Transaction must be committed or rolled back
	// Context can be used for cancellation and timeout
	BeginTx(ctx context.Context) (Transaction, error)
}

// Transaction represents a database transaction
// All operations within a transaction are atomic
type Transaction interface {
	// Get retrieves a value within the transaction context
	Get(ctx context.Context, key Key) (Value, error)

	// Set stores a key-value pair within the transaction context
	Set(ctx context.Context, key Key, value string) error

	// Delete removes a key-value pair within the transaction context
	Delete(ctx context.Context, key Key) (Value, error)

	// Commit commits all operations in the transaction
	// After commit, all changes become visible to other transactions
	Commit(ctx context.Context) error

	// Rollback rolls back all operations in the transaction
	// After rollback, no changes from this transaction are applied
	Rollback(ctx context.Context) error
}

// StoreConfig holds configuration options for store implementations
type StoreConfig struct {
	// MaxKeys limits the maximum number of keys that can be stored
	// 0 means no limit
	MaxKeys int `json:"max_keys"`

	// MaxValueSize limits the maximum size of a value in bytes
	// 0 means no limit
	MaxValueSize int `json:"max_value_size"`

	// DefaultTimeout is the default timeout for operations
	DefaultTimeout time.Duration `json:"default_timeout"`

	// EnableMetrics enables collection of store metrics
	EnableMetrics bool `json:"enable_metrics"`

	// EnableCompression enables value compression
	EnableCompression bool `json:"enable_compression"`
}

// Metrics represents store performance metrics
type Metrics struct {
	// TotalOperations is the total number of operations performed
	TotalOperations int64 `json:"total_operations"`

	// GetOperations is the number of Get operations
	GetOperations int64 `json:"get_operations"`

	// SetOperations is the number of Set operations
	SetOperations int64 `json:"set_operations"`

	// DeleteOperations is the number of Delete operations
	DeleteOperations int64 `json:"delete_operations"`

	// AverageResponseTime is the average response time in nanoseconds
	AverageResponseTime time.Duration `json:"average_response_time"`

	// ErrorCount is the total number of errors encountered
	ErrorCount int64 `json:"error_count"`

	// ConcurrentConnections is the current number of concurrent operations
	ConcurrentConnections int32 `json:"concurrent_connections"`
}

// MetricsStore extends Store with metrics collection capabilities
type MetricsStore interface {
	Store

	// GetMetrics returns the current store metrics
	GetMetrics() Metrics

	// ResetMetrics resets all metrics counters to zero
	ResetMetrics()
}

// Validate validates a key according to store rules
func (k Key) Validate() error {
	if len(k) == 0 {
		return ErrInvalidKey
	}
	if len(k) > 255 {
		return ErrInvalidKey
	}
	// Keys cannot contain null bytes
	for _, b := range []byte(k) {
		if b == 0 {
			return ErrInvalidKey
		}
	}
	return nil
}

// String returns the string representation of the key
func (k Key) String() string {
	return string(k)
}

// IsEmpty checks if the value is empty (zero value)
func (v Value) IsEmpty() bool {
	return v.Data == "" && v.CreatedAt.IsZero() && v.UpdatedAt.IsZero() && v.Version == 0
}
