package store

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// mockPersistence is a mock implementation of Persistence for testing
type mockPersistence struct {
	mu             sync.RWMutex
	saveCount      int
	loadCount      int
	shouldFailSave bool
	shouldFailLoad bool
	snapshot       *StoreSnapshot
	saveCalls      []SaveCall
}

type SaveCall struct {
	Timestamp time.Time
	Snapshot  *StoreSnapshot
}

func newMockPersistence() *mockPersistence {
	return &mockPersistence{
		saveCalls: make([]SaveCall, 0),
	}
}

func (m *mockPersistence) Save(ctx context.Context, snapshot *StoreSnapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.saveCount++
	m.saveCalls = append(m.saveCalls, SaveCall{
		Timestamp: time.Now(),
		Snapshot:  snapshot,
	})

	if m.shouldFailSave {
		return errors.New("mock save error")
	}

	m.snapshot = snapshot
	return nil
}

func (m *mockPersistence) Load(ctx context.Context) (*StoreSnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.loadCount++

	if m.shouldFailLoad {
		return nil, errors.New("mock load error")
	}

	if m.snapshot == nil {
		return nil, ErrNoSnapshotFound
	}

	return m.snapshot, nil
}

func (m *mockPersistence) getSaveCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.saveCount
}

func (m *mockPersistence) getLoadCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.loadCount
}

func (m *mockPersistence) setFailSave(fail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFailSave = fail
}

func (m *mockPersistence) setFailLoad(fail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFailLoad = fail
}

// Test RED phase - these tests should fail initially
func TestPersistentStore_Creation(t *testing.T) {
	t.Run("create with default config", func(t *testing.T) {
		memStore := NewMemoryStore()
		persistence := newMockPersistence()

		// This should fail - PersistentStore doesn't exist yet
		store, err := NewPersistentStore(memStore, persistence, PersistentStoreConfig{})

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if store == nil {
			t.Fatal("expected store to be created")
		}
	})

	t.Run("create with custom save interval", func(t *testing.T) {
		memStore := NewMemoryStore()
		persistence := newMockPersistence()
		config := PersistentStoreConfig{
			SaveInterval: 10 * time.Second,
		}

		store, err := NewPersistentStore(memStore, persistence, config)

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if store == nil {
			t.Fatal("expected store to be created")
		}
	})
}

func TestPersistentStore_AutoSaveOnSet(t *testing.T) {
	t.Run("auto-save on Set operation", func(t *testing.T) {
		memStore := NewMemoryStore()
		persistence := newMockPersistence()
		config := PersistentStoreConfig{
			AutoSave: true,
		}

		store, err := NewPersistentStore(memStore, persistence, config)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer store.Close()

		ctx := context.Background()
		err = store.Set(ctx, Key("test-key"), "test-value")

		if err != nil {
			t.Fatalf("expected no error on Set, got %v", err)
		}

		// Give some time for async save
		time.Sleep(100 * time.Millisecond)

		saveCount := persistence.getSaveCount()
		if saveCount != 1 {
			t.Errorf("expected 1 save call, got %d", saveCount)
		}
	})
}

func TestPersistentStore_AutoSaveOnDelete(t *testing.T) {
	t.Run("auto-save on Delete operation", func(t *testing.T) {
		memStore := NewMemoryStore()
		persistence := newMockPersistence()
		config := PersistentStoreConfig{
			AutoSave: true,
		}

		store, err := NewPersistentStore(memStore, persistence, config)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer store.Close()

		ctx := context.Background()

		// First set a value
		err = store.Set(ctx, Key("test-key"), "test-value")
		if err != nil {
			t.Fatalf("failed to set value: %v", err)
		}

		time.Sleep(50 * time.Millisecond)

		// Then delete it
		_, err = store.Delete(ctx, Key("test-key"))
		if err != nil {
			t.Fatalf("expected no error on Delete, got %v", err)
		}

		time.Sleep(100 * time.Millisecond)

		saveCount := persistence.getSaveCount()
		if saveCount < 2 {
			t.Errorf("expected at least 2 save calls (Set + Delete), got %d", saveCount)
		}
	})
}

func TestPersistentStore_AutoSaveOnClear(t *testing.T) {
	t.Run("auto-save on Clear operation", func(t *testing.T) {
		memStore := NewMemoryStore()
		persistence := newMockPersistence()
		config := PersistentStoreConfig{
			AutoSave: true,
		}

		store, err := NewPersistentStore(memStore, persistence, config)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer store.Close()

		ctx := context.Background()

		// Add some data first
		err = store.Set(ctx, Key("key1"), "value1")
		if err != nil {
			t.Fatalf("failed to set value: %v", err)
		}

		time.Sleep(50 * time.Millisecond)

		// Clear all data
		err = store.Clear(ctx)
		if err != nil {
			t.Fatalf("expected no error on Clear, got %v", err)
		}

		time.Sleep(100 * time.Millisecond)

		saveCount := persistence.getSaveCount()
		if saveCount < 2 {
			t.Errorf("expected at least 2 save calls (Set + Clear), got %d", saveCount)
		}
	})
}

func TestPersistentStore_PeriodicSave(t *testing.T) {
	t.Run("periodic background saves", func(t *testing.T) {
		memStore := NewMemoryStore()
		persistence := newMockPersistence()
		config := PersistentStoreConfig{
			SaveInterval: 100 * time.Millisecond,
		}

		store, err := NewPersistentStore(memStore, persistence, config)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer store.Close()

		ctx := context.Background()

		// Set some data
		err = store.Set(ctx, Key("test-key"), "test-value")
		if err != nil {
			t.Fatalf("failed to set value: %v", err)
		}

		// Wait for multiple save intervals
		time.Sleep(350 * time.Millisecond)

		saveCount := persistence.getSaveCount()
		if saveCount < 3 {
			t.Errorf("expected at least 3 periodic saves, got %d", saveCount)
		}
	})
}

func TestPersistentStore_LoadOnStartup(t *testing.T) {
	t.Run("load existing data on startup", func(t *testing.T) {
		persistence := newMockPersistence()

		// Prepare existing data
		existingData := map[string]string{
			"key1": "value1",
			"key2": "value2",
		}

		snapshot := &StoreSnapshot{
			Data:      existingData,
			Version:   "1.0",
			Timestamp: time.Now().Unix(),
		}
		persistence.snapshot = snapshot

		memStore := NewMemoryStore()
		config := PersistentStoreConfig{}

		store, err := NewPersistentStore(memStore, persistence, config)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer store.Close()

		// Check if data was loaded
		ctx := context.Background()
		value, err := store.Get(ctx, Key("key1"))
		if err != nil {
			t.Fatalf("expected to find key1, got error: %v", err)
		}

		if value.Data != "value1" {
			t.Errorf("expected value1, got %s", value.Data)
		}

		loadCount := persistence.getLoadCount()
		if loadCount != 1 {
			t.Errorf("expected 1 load call, got %d", loadCount)
		}
	})
}

func TestPersistentStore_ErrorHandling(t *testing.T) {
	t.Run("handle save errors gracefully", func(t *testing.T) {
		memStore := NewMemoryStore()
		persistence := newMockPersistence()
		persistence.setFailSave(true)

		config := PersistentStoreConfig{
			AutoSave: true,
		}

		store, err := NewPersistentStore(memStore, persistence, config)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer store.Close()

		ctx := context.Background()

		// This should not fail even if persistence fails
		err = store.Set(ctx, Key("test-key"), "test-value")
		if err != nil {
			t.Errorf("Set operation should not fail due to persistence error, got: %v", err)
		}

		// Verify the data is still in memory store
		value, err := store.Get(ctx, Key("test-key"))
		if err != nil {
			t.Fatalf("expected to find key in memory, got error: %v", err)
		}

		if value.Data != "test-value" {
			t.Errorf("expected test-value, got %s", value.Data)
		}
	})

	t.Run("handle load errors gracefully", func(t *testing.T) {
		persistence := newMockPersistence()
		persistence.setFailLoad(true)

		memStore := NewMemoryStore()
		config := PersistentStoreConfig{}

		// This should not fail even if loading fails
		store, err := NewPersistentStore(memStore, persistence, config)
		if err != nil {
			t.Fatalf("store creation should not fail due to load error, got: %v", err)
		}
		defer store.Close()

		// Store should be functional
		ctx := context.Background()
		err = store.Set(ctx, Key("test-key"), "test-value")
		if err != nil {
			t.Fatalf("store should be functional after load error: %v", err)
		}
	})
}

func TestPersistentStore_SaveOnShutdown(t *testing.T) {
	t.Run("save data on shutdown", func(t *testing.T) {
		memStore := NewMemoryStore()
		persistence := newMockPersistence()
		config := PersistentStoreConfig{
			SaveOnShutdown: true,
		}

		store, err := NewPersistentStore(memStore, persistence, config)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}

		ctx := context.Background()
		err = store.Set(ctx, Key("test-key"), "test-value")
		if err != nil {
			t.Fatalf("failed to set value: %v", err)
		}

		initialSaveCount := persistence.getSaveCount()

		// Close the store
		err = store.Close()
		if err != nil {
			t.Fatalf("failed to close store: %v", err)
		}

		finalSaveCount := persistence.getSaveCount()
		if finalSaveCount <= initialSaveCount {
			t.Errorf("expected save on shutdown, save count didn't increase: %d -> %d",
				initialSaveCount, finalSaveCount)
		}
	})
}

func TestPersistentStore_AtomicOperations(t *testing.T) {
	t.Run("operations are atomic with persistence", func(t *testing.T) {
		memStore := NewMemoryStore()
		persistence := newMockPersistence()
		config := PersistentStoreConfig{
			AutoSave: true,
		}

		store, err := NewPersistentStore(memStore, persistence, config)
		if err != nil {
			t.Fatalf("failed to create store: %v", err)
		}
		defer store.Close()

		ctx := context.Background()

		// Perform multiple operations concurrently
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				key := Key("key" + string(rune('0'+i)))
				value := "value" + string(rune('0'+i))
				err := store.Set(ctx, key, value)
				if err != nil {
					t.Errorf("failed to set key %s: %v", key, err)
				}
			}(i)
		}

		wg.Wait()

		// Give time for all saves to complete
		time.Sleep(200 * time.Millisecond)

		// Verify all keys are present
		size, err := store.Size(ctx)
		if err != nil {
			t.Fatalf("failed to get size: %v", err)
		}

		if size != 10 {
			t.Errorf("expected 10 keys, got %d", size)
		}

		// Verify persistence was called
		saveCount := persistence.getSaveCount()
		if saveCount == 0 {
			t.Error("expected at least one save call")
		}
	})
}

// Integration test with real file persistence
func TestPersistentStore_Integration(t *testing.T) {
	t.Run("integration with JSONFilePersistence", func(t *testing.T) {
		// Create temporary file
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test_store.json")

		memStore := NewMemoryStore()
		persistence := NewJSONFilePersistence(filePath)
		config := PersistentStoreConfig{
			AutoSave:       true,
			SaveOnShutdown: true,
		}

		// First store instance
		store1, err := NewPersistentStore(memStore, persistence, config)
		if err != nil {
			t.Fatalf("failed to create first store: %v", err)
		}

		ctx := context.Background()
		err = store1.Set(ctx, Key("test-key"), "test-value")
		if err != nil {
			t.Fatalf("failed to set value: %v", err)
		}

		// Close first store (should save)
		err = store1.Close()
		if err != nil {
			t.Fatalf("failed to close first store: %v", err)
		}

		// Verify file was created
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Fatal("expected persistence file to be created")
		}

		// Create second store instance (should load data)
		memStore2 := NewMemoryStore()
		store2, err := NewPersistentStore(memStore2, persistence, config)
		if err != nil {
			t.Fatalf("failed to create second store: %v", err)
		}
		defer store2.Close()

		// Verify data was loaded
		value, err := store2.Get(ctx, Key("test-key"))
		if err != nil {
			t.Fatalf("expected to find key in loaded store: %v", err)
		}

		if value.Data != "test-value" {
			t.Errorf("expected test-value, got %s", value.Data)
		}
	})
}
