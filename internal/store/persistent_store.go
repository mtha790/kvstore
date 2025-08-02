// Package store implements a persistent wrapper around the Store interface
package store

import (
	"context"
	"fmt"
	"sync"
	"time"

	"kvstore/pkg/logger"
)

// PersistentStoreConfig holds configuration for the PersistentStore
type PersistentStoreConfig struct {
	// AutoSave enables automatic saving after each modification operation
	AutoSave bool

	// SaveInterval specifies the interval for periodic background saves
	// Default is 30 seconds
	SaveInterval time.Duration

	// SaveOnShutdown enables saving data when the store is closed
	SaveOnShutdown bool

	// RetryAttempts specifies how many times to retry failed save operations
	// Default is 3
	RetryAttempts int

	// RetryDelay specifies the delay between retry attempts
	// Default is 1 second
	RetryDelay time.Duration
}

// DefaultPersistentStoreConfig returns a configuration with sensible defaults
func DefaultPersistentStoreConfig() PersistentStoreConfig {
	return PersistentStoreConfig{
		AutoSave:       true,
		SaveInterval:   30 * time.Second,
		SaveOnShutdown: true,
		RetryAttempts:  3,
		RetryDelay:     1 * time.Second,
	}
}

// PersistentStore wraps a Store implementation and adds persistence capabilities
type PersistentStore struct {
	// Embedded store provides the core Store interface
	store Store

	// persistence handles saving and loading of store snapshots
	persistence Persistence

	// config holds the persistence configuration
	config PersistentStoreConfig

	// saveChannel is used for asynchronous save operations
	saveChannel chan struct{}

	// periodicSaveTimer handles periodic saves
	periodicSaveTimer *time.Timer

	// shutdownOnce ensures shutdown operations are performed only once
	shutdownOnce sync.Once

	// closed indicates whether the store has been closed
	closed bool

	// mutex protects the closed flag and other internal state
	mutex sync.RWMutex

	// wg tracks active goroutines for graceful shutdown
	wg sync.WaitGroup
}

// NewPersistentStore creates a new PersistentStore wrapping the given store
func NewPersistentStore(store Store, persistence Persistence, config PersistentStoreConfig) (*PersistentStore, error) {
	if store == nil {
		return nil, fmt.Errorf("store cannot be nil")
	}
	if persistence == nil {
		return nil, fmt.Errorf("persistence cannot be nil")
	}

	// Apply defaults
	if config.SaveInterval == 0 {
		config.SaveInterval = 30 * time.Second
	}
	if config.RetryAttempts == 0 {
		config.RetryAttempts = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 1 * time.Second
	}

	ps := &PersistentStore{
		store:       store,
		persistence: persistence,
		config:      config,
		saveChannel: make(chan struct{}, 100), // Buffered to prevent blocking
	}

	// Try to load existing data
	if err := ps.loadData(); err != nil {
		logger.Warn("failed to load existing data", "error", err)
	}

	// Start background save processor
	ps.wg.Add(1)
	go ps.saveProcessor()

	// Start periodic save timer if configured
	if config.SaveInterval > 0 {
		ps.startPeriodicSave()
	}

	return ps, nil
}

// loadData attempts to load existing data from persistence into the store
func (ps *PersistentStore) loadData() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	snapshot, err := ps.persistence.Load(ctx)
	if err != nil {
		if err == ErrNoSnapshotFound {
			logger.Debug("no existing snapshot found, starting with empty store")
			return nil
		}
		return fmt.Errorf("failed to load snapshot: %w", err)
	}

	// Load data into the store
	for key, value := range snapshot.Data {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := ps.store.Set(ctx, Key(key), value)
		cancel()
		if err != nil {
			logger.Error("failed to load key into store", "key", key, "error", err)
		}
	}

	logger.Info("loaded data from persistence", "entries", len(snapshot.Data))
	return nil
}

// createSnapshot creates a snapshot of the current store state
func (ps *PersistentStore) createSnapshot() (*StoreSnapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	entries, err := ps.store.ListEntries(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list entries: %w", err)
	}

	data := make(map[string]string)
	for _, entry := range entries {
		data[string(entry.Key)] = entry.Value.Data
	}

	snapshot := &StoreSnapshot{
		Data:      data,
		Version:   "1.0",
		Timestamp: time.Now().Unix(),
		Stats:     StoreStats{TotalKeys: len(data)},
	}

	return snapshot, nil
}

// saveWithRetry attempts to save a snapshot with retry logic
func (ps *PersistentStore) saveWithRetry(snapshot *StoreSnapshot) error {
	var lastErr error

	for attempt := 0; attempt <= ps.config.RetryAttempts; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := ps.persistence.Save(ctx, snapshot)
		cancel()

		if err == nil {
			if attempt > 0 {
				logger.Info("save succeeded after retry", "attempt", attempt)
			}
			return nil
		}

		lastErr = err
		logger.Warn("save attempt failed", "attempt", attempt, "error", err)

		if attempt < ps.config.RetryAttempts {
			time.Sleep(ps.config.RetryDelay)
		}
	}

	return fmt.Errorf("failed to save after %d attempts: %w", ps.config.RetryAttempts+1, lastErr)
}

// triggerSave requests an asynchronous save operation
func (ps *PersistentStore) triggerSave() {
	ps.mutex.RLock()
	closed := ps.closed
	ps.mutex.RUnlock()

	if closed {
		return
	}

	// Non-blocking send to save channel
	select {
	case ps.saveChannel <- struct{}{}:
	default:
		// Channel is full, save is already pending
	}
}

// saveProcessor handles asynchronous save operations
func (ps *PersistentStore) saveProcessor() {
	defer ps.wg.Done()

	for range ps.saveChannel {
		ps.mutex.RLock()
		closed := ps.closed
		ps.mutex.RUnlock()

		if closed {
			return
		}

		snapshot, err := ps.createSnapshot()
		if err != nil {
			logger.Error("failed to create snapshot", "error", err)
			continue
		}

		if err := ps.saveWithRetry(snapshot); err != nil {
			logger.Error("failed to save snapshot", "error", err)
		} else {
			logger.Debug("snapshot saved successfully", "entries", len(snapshot.Data))
		}
	}
}

// startPeriodicSave starts the periodic save timer
func (ps *PersistentStore) startPeriodicSave() {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	if ps.closed {
		return
	}

	ps.periodicSaveTimer = time.AfterFunc(ps.config.SaveInterval, func() {
		ps.triggerSave()

		ps.mutex.RLock()
		closed := ps.closed
		ps.mutex.RUnlock()

		if !closed {
			ps.startPeriodicSave() // Schedule next save
		}
	})
}

// Store interface implementation with persistence hooks

// Get retrieves a value from the underlying store
func (ps *PersistentStore) Get(ctx context.Context, key Key) (Value, error) {
	ps.mutex.RLock()
	closed := ps.closed
	ps.mutex.RUnlock()

	if closed {
		return Value{}, ErrStoreClosed
	}

	return ps.store.Get(ctx, key)
}

// Set stores a key-value pair and triggers save if auto-save is enabled
func (ps *PersistentStore) Set(ctx context.Context, key Key, value string) error {
	logger.Info("PersistentStore.Set called", "key", key, "autosave", ps.config.AutoSave)
	ps.mutex.RLock()
	closed := ps.closed
	ps.mutex.RUnlock()

	if closed {
		return ErrStoreClosed
	}

	err := ps.store.Set(ctx, key, value)
	if err != nil {
		return err
	}

	if ps.config.AutoSave {
		logger.Info("Triggering save due to Set operation")
		ps.triggerSave()
	}

	return nil
}

// Delete removes a key-value pair and triggers save if auto-save is enabled
func (ps *PersistentStore) Delete(ctx context.Context, key Key) (Value, error) {
	ps.mutex.RLock()
	closed := ps.closed
	ps.mutex.RUnlock()

	if closed {
		return Value{}, ErrStoreClosed
	}

	value, err := ps.store.Delete(ctx, key)
	if err != nil {
		return value, err
	}

	if ps.config.AutoSave {
		ps.triggerSave()
	}

	return value, nil
}

// List returns all keys from the underlying store
func (ps *PersistentStore) List(ctx context.Context) ([]Key, error) {
	ps.mutex.RLock()
	closed := ps.closed
	ps.mutex.RUnlock()

	if closed {
		return nil, ErrStoreClosed
	}

	return ps.store.List(ctx)
}

// ListEntries returns all entries from the underlying store
func (ps *PersistentStore) ListEntries(ctx context.Context) ([]Entry, error) {
	ps.mutex.RLock()
	closed := ps.closed
	ps.mutex.RUnlock()

	if closed {
		return nil, ErrStoreClosed
	}

	return ps.store.ListEntries(ctx)
}

// Size returns the number of entries in the underlying store
func (ps *PersistentStore) Size(ctx context.Context) (int, error) {
	ps.mutex.RLock()
	closed := ps.closed
	ps.mutex.RUnlock()

	if closed {
		return 0, ErrStoreClosed
	}

	return ps.store.Size(ctx)
}

// Clear removes all entries and triggers save if auto-save is enabled
func (ps *PersistentStore) Clear(ctx context.Context) error {
	ps.mutex.RLock()
	closed := ps.closed
	ps.mutex.RUnlock()

	if closed {
		return ErrStoreClosed
	}

	err := ps.store.Clear(ctx)
	if err != nil {
		return err
	}

	if ps.config.AutoSave {
		ps.triggerSave()
	}

	return nil
}

// Exists checks if a key exists in the underlying store
func (ps *PersistentStore) Exists(ctx context.Context, key Key) (bool, error) {
	ps.mutex.RLock()
	closed := ps.closed
	ps.mutex.RUnlock()

	if closed {
		return false, ErrStoreClosed
	}

	return ps.store.Exists(ctx, key)
}

// CompareAndSwap performs an atomic compare-and-swap and triggers save if auto-save is enabled
func (ps *PersistentStore) CompareAndSwap(ctx context.Context, key Key, expectedVersion int64, newValue string) (Value, error) {
	ps.mutex.RLock()
	closed := ps.closed
	ps.mutex.RUnlock()

	if closed {
		return Value{}, ErrStoreClosed
	}

	value, err := ps.store.CompareAndSwap(ctx, key, expectedVersion, newValue)
	if err != nil {
		return value, err
	}

	if ps.config.AutoSave {
		ps.triggerSave()
	}

	return value, nil
}

// Close closes the persistent store and performs final save if configured
func (ps *PersistentStore) Close() error {
	var closeErr error

	ps.shutdownOnce.Do(func() {
		ps.mutex.Lock()
		ps.closed = true
		// Stop periodic save timer while holding the lock
		if ps.periodicSaveTimer != nil {
			ps.periodicSaveTimer.Stop()
		}
		ps.mutex.Unlock()

		// Perform final save if configured
		if ps.config.SaveOnShutdown {
			snapshot, err := ps.createSnapshot()
			if err != nil {
				logger.Error("failed to create final snapshot", "error", err)
			} else {
				if err := ps.saveWithRetry(snapshot); err != nil {
					logger.Error("failed to save final snapshot", "error", err)
					closeErr = err
				} else {
					logger.Info("final snapshot saved on shutdown")
				}
			}
		}

		// Close the save channel
		close(ps.saveChannel)

		// Wait for save processor to finish
		ps.wg.Wait()

		// Close the underlying store
		if err := ps.store.Close(); err != nil {
			logger.Error("failed to close underlying store", "error", err)
			if closeErr == nil {
				closeErr = err
			}
		}
	})

	return closeErr
}
