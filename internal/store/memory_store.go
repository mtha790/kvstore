// Package store implements a concurrent in-memory key-value store
package store

import (
	"context"
	"sync"
	"time"
)

// MemoryStore implements the Store interface using an in-memory map with concurrent access control
type MemoryStore struct {
	// data holds the key-value pairs with their metadata
	data map[string]Value

	// mutex provides thread-safe access to the data map
	// Using RWMutex to allow multiple concurrent reads while ensuring exclusive writes
	mutex sync.RWMutex

	// stats tracks store statistics
	stats StoreStats

	// statsMutex protects the stats from concurrent access
	statsMutex sync.RWMutex

	// closed indicates if the store has been closed
	closed bool
}

// NewMemoryStore creates and returns a new instance of MemoryStore
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data:   make(map[string]Value),
		stats:  StoreStats{},
		closed: false,
	}
}

// NewMemoryStoreWithCapacity creates a new MemoryStore with a pre-allocated map capacity
// This can improve performance when the expected number of keys is known in advance
func NewMemoryStoreWithCapacity(capacity int) *MemoryStore {
	return &MemoryStore{
		data:   make(map[string]Value, capacity),
		stats:  StoreStats{},
		closed: false,
	}
}

// GetStats returns a copy of the current store statistics
func (ms *MemoryStore) GetStats() StoreStats {
	ms.statsMutex.RLock()
	defer ms.statsMutex.RUnlock()

	// Return a copy to prevent external modification
	return StoreStats{
		TotalKeys:      ms.stats.TotalKeys,
		TotalRequests:  ms.stats.TotalRequests,
		GetRequests:    ms.stats.GetRequests,
		SetRequests:    ms.stats.SetRequests,
		DeleteRequests: ms.stats.DeleteRequests,
	}
}

// StatType represents the type of operation for statistics
type StatType int

const (
	StatGet StatType = iota
	StatSet
	StatDelete
)

// incrementStat safely increments a specific statistic counter
func (ms *MemoryStore) incrementStat(statType StatType) {
	ms.statsMutex.Lock()
	defer ms.statsMutex.Unlock()

	ms.stats.TotalRequests++
	switch statType {
	case StatGet:
		ms.stats.GetRequests++
	case StatSet:
		ms.stats.SetRequests++
	case StatDelete:
		ms.stats.DeleteRequests++
	}
}

// updateKeyCount updates the total key count in statistics
func (ms *MemoryStore) updateKeyCount() {
	ms.statsMutex.Lock()
	defer ms.statsMutex.Unlock()

	ms.stats.TotalKeys = len(ms.data)
}

// Get retrieves the value associated with the given key
func (ms *MemoryStore) Get(ctx context.Context, key Key) (Value, error) {
	// Validate key first (before any locks)
	if err := key.Validate(); err != nil {
		return Value{}, err
	}

	// Check for context cancellation or timeout
	select {
	case <-ctx.Done():
		return Value{}, ctx.Err()
	default:
	}

	ms.mutex.RLock()
	defer ms.mutex.RUnlock()

	// Check if store is closed
	if ms.closed {
		return Value{}, ErrStoreClosed
	}

	ms.incrementStat(StatGet)

	value, exists := ms.data[string(key)]
	if !exists {
		return Value{}, ErrKeyNotFound
	}

	return value, nil
}

// Set stores a key-value pair in the store
func (ms *MemoryStore) Set(ctx context.Context, key Key, value string) error {
	// Validate key first (before any locks)
	if err := key.Validate(); err != nil {
		return err
	}

	// Check for context cancellation or timeout
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	// Check if store is closed
	if ms.closed {
		return ErrStoreClosed
	}

	ms.incrementStat(StatSet)

	now := time.Now()
	existingValue, exists := ms.data[string(key)]

	var newValue Value
	if exists {
		// Update existing value
		newValue = Value{
			Data:      value,
			CreatedAt: existingValue.CreatedAt,
			UpdatedAt: now,
			Version:   existingValue.Version + 1,
		}
	} else {
		// Create new value
		newValue = Value{
			Data:      value,
			CreatedAt: now,
			UpdatedAt: now,
			Version:   1,
		}
	}

	ms.data[string(key)] = newValue
	ms.updateKeyCount()

	return nil
}

// Delete removes a key-value pair from the store
func (ms *MemoryStore) Delete(ctx context.Context, key Key) (Value, error) {
	// Validate key first (before any locks)
	if err := key.Validate(); err != nil {
		return Value{}, err
	}

	// Check for context cancellation or timeout
	select {
	case <-ctx.Done():
		return Value{}, ctx.Err()
	default:
	}

	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	// Check if store is closed
	if ms.closed {
		return Value{}, ErrStoreClosed
	}

	ms.incrementStat(StatDelete)

	value, exists := ms.data[string(key)]
	if !exists {
		return Value{}, ErrKeyNotFound
	}

	delete(ms.data, string(key))
	ms.updateKeyCount()

	return value, nil
}

// List returns all keys currently stored in the key-value store
func (ms *MemoryStore) List(ctx context.Context) ([]Key, error) {
	// Check for context cancellation or timeout
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	ms.mutex.RLock()
	defer ms.mutex.RUnlock()

	// Check if store is closed
	if ms.closed {
		return nil, ErrStoreClosed
	}

	keys := make([]Key, 0, len(ms.data))
	for key := range ms.data {
		keys = append(keys, Key(key))
	}

	return keys, nil
}

// ListEntries returns all key-value entries currently stored
func (ms *MemoryStore) ListEntries(ctx context.Context) ([]Entry, error) {
	// Check for context cancellation or timeout
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	ms.mutex.RLock()
	defer ms.mutex.RUnlock()

	// Check if store is closed
	if ms.closed {
		return nil, ErrStoreClosed
	}

	entries := make([]Entry, 0, len(ms.data))
	for key, value := range ms.data {
		entries = append(entries, Entry{
			Key:   Key(key),
			Value: value,
		})
	}

	return entries, nil
}

// Size returns the current number of key-value pairs in the store
func (ms *MemoryStore) Size(ctx context.Context) (int, error) {
	// Check for context cancellation or timeout
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	ms.mutex.RLock()
	defer ms.mutex.RUnlock()

	// Check if store is closed
	if ms.closed {
		return 0, ErrStoreClosed
	}

	return len(ms.data), nil
}

// Clear removes all key-value pairs from the store
func (ms *MemoryStore) Clear(ctx context.Context) error {
	// Check for context cancellation or timeout
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	// Check if store is closed
	if ms.closed {
		return ErrStoreClosed
	}

	// Clear all data atomically
	ms.data = make(map[string]Value)
	ms.updateKeyCount()

	return nil
}

// Exists checks if a key exists in the store without retrieving the value
func (ms *MemoryStore) Exists(ctx context.Context, key Key) (bool, error) {
	// Validate key first (before any locks)
	if err := key.Validate(); err != nil {
		return false, err
	}

	// Check for context cancellation or timeout
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	ms.mutex.RLock()
	defer ms.mutex.RUnlock()

	// Check if store is closed
	if ms.closed {
		return false, ErrStoreClosed
	}

	_, exists := ms.data[string(key)]
	return exists, nil
}

// CompareAndSwap atomically compares and swaps a value
func (ms *MemoryStore) CompareAndSwap(ctx context.Context, key Key, expectedVersion int64, newValue string) (Value, error) {
	// Validate key first (before any locks)
	if err := key.Validate(); err != nil {
		return Value{}, err
	}

	// Check for context cancellation or timeout
	select {
	case <-ctx.Done():
		return Value{}, ctx.Err()
	default:
	}

	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	// Check if store is closed
	if ms.closed {
		return Value{}, ErrStoreClosed
	}

	currentValue, exists := ms.data[string(key)]
	if !exists {
		return Value{}, ErrKeyNotFound
	}

	if currentValue.Version != expectedVersion {
		return currentValue, ErrConcurrentModification
	}

	now := time.Now()
	updatedValue := Value{
		Data:      newValue,
		CreatedAt: currentValue.CreatedAt,
		UpdatedAt: now,
		Version:   currentValue.Version + 1,
	}

	ms.data[string(key)] = updatedValue
	return updatedValue, nil
}

// Close closes the store and releases any resources
func (ms *MemoryStore) Close() error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	ms.closed = true
	return nil
}
