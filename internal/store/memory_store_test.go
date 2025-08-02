package store

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMemoryStore_Get(t *testing.T) {
	tests := []struct {
		name           string
		setupKeys      map[string]string
		key            Key
		expectedValue  Value
		expectedError  error
		contextTimeout time.Duration
	}{
		{
			name:          "get existing key",
			setupKeys:     map[string]string{"test-key": "test-value"},
			key:           Key("test-key"),
			expectedValue: Value{Data: "test-value", Version: 1},
			expectedError: nil,
		},
		{
			name:          "get non-existing key",
			setupKeys:     map[string]string{},
			key:           Key("non-existing"),
			expectedValue: Value{},
			expectedError: ErrKeyNotFound,
		},
		{
			name:          "get with empty key",
			setupKeys:     map[string]string{},
			key:           Key(""),
			expectedValue: Value{},
			expectedError: ErrInvalidKey,
		},
		{
			name:           "get with context timeout",
			setupKeys:      map[string]string{"test-key": "test-value"},
			key:            Key("test-key"),
			expectedValue:  Value{},
			expectedError:  context.DeadlineExceeded,
			contextTimeout: 1 * time.Nanosecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMemoryStore()

			// Setup initial data
			for k, v := range tt.setupKeys {
				ctx := context.Background()
				err := store.Set(ctx, Key(k), v)
				if err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			}

			// Create context with timeout if specified
			ctx := context.Background()
			if tt.contextTimeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tt.contextTimeout)
				defer cancel()
				time.Sleep(2 * time.Nanosecond) // Ensure timeout
			}

			value, err := store.Get(ctx, tt.key)

			if tt.expectedError != nil {
				if err != tt.expectedError {
					t.Errorf("expected error %v, got %v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if value.Data != tt.expectedValue.Data {
					t.Errorf("expected value data %s, got %s", tt.expectedValue.Data, value.Data)
				}
				if tt.expectedValue.Version > 0 && value.Version != tt.expectedValue.Version {
					t.Errorf("expected version %d, got %d", tt.expectedValue.Version, value.Version)
				}
				if !value.CreatedAt.IsZero() && value.UpdatedAt.Before(value.CreatedAt) {
					t.Error("UpdatedAt should not be before CreatedAt")
				}
			}
		})
	}
}

func TestMemoryStore_Set(t *testing.T) {
	tests := []struct {
		name          string
		key           Key
		value         string
		expectedError error
	}{
		{
			name:          "set new key",
			key:           Key("new-key"),
			value:         "new-value",
			expectedError: nil,
		},
		{
			name:          "set existing key (update)",
			key:           Key("existing-key"),
			value:         "updated-value",
			expectedError: nil,
		},
		{
			name:          "set with empty key",
			key:           Key(""),
			value:         "some-value",
			expectedError: ErrInvalidKey,
		},
		{
			name:          "set with empty value",
			key:           Key("valid-key"),
			value:         "",
			expectedError: nil, // Empty values should be allowed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMemoryStore()

			// Setup existing key if needed
			if tt.name == "set existing key (update)" {
				ctx := context.Background()
				err := store.Set(ctx, tt.key, "original-value")
				if err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			}

			ctx := context.Background()
			err := store.Set(ctx, tt.key, tt.value)

			if tt.expectedError != nil {
				if err != tt.expectedError {
					t.Errorf("expected error %v, got %v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				// Verify the value was set correctly
				value, getErr := store.Get(ctx, tt.key)
				if getErr != nil {
					t.Errorf("failed to get set value: %v", getErr)
				}
				if value.Data != tt.value {
					t.Errorf("expected value data %s, got %s", tt.value, value.Data)
				}
			}
		})
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	tests := []struct {
		name          string
		setupKeys     map[string]string
		key           Key
		expectedValue Value
		expectedError error
	}{
		{
			name:          "delete existing key",
			setupKeys:     map[string]string{"test-key": "test-value"},
			key:           Key("test-key"),
			expectedValue: Value{Data: "test-value", Version: 1},
			expectedError: nil,
		},
		{
			name:          "delete non-existing key",
			setupKeys:     map[string]string{},
			key:           Key("non-existing"),
			expectedValue: Value{},
			expectedError: ErrKeyNotFound,
		},
		{
			name:          "delete with empty key",
			setupKeys:     map[string]string{},
			key:           Key(""),
			expectedValue: Value{},
			expectedError: ErrInvalidKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMemoryStore()

			// Setup initial data
			for k, v := range tt.setupKeys {
				ctx := context.Background()
				err := store.Set(ctx, Key(k), v)
				if err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			}

			ctx := context.Background()
			value, err := store.Delete(ctx, tt.key)

			if tt.expectedError != nil {
				if err != tt.expectedError {
					t.Errorf("expected error %v, got %v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if value.Data != tt.expectedValue.Data {
					t.Errorf("expected deleted value data %s, got %s", tt.expectedValue.Data, value.Data)
				}

				// Verify key is actually deleted
				_, getErr := store.Get(ctx, tt.key)
				if getErr != ErrKeyNotFound {
					t.Error("expected key to be deleted")
				}
			}
		})
	}
}

func TestMemoryStore_List(t *testing.T) {
	tests := []struct {
		name         string
		setupKeys    map[string]string
		expectedKeys []Key
	}{
		{
			name:         "list empty store",
			setupKeys:    map[string]string{},
			expectedKeys: []Key{},
		},
		{
			name:         "list single key",
			setupKeys:    map[string]string{"key1": "value1"},
			expectedKeys: []Key{Key("key1")},
		},
		{
			name:         "list multiple keys",
			setupKeys:    map[string]string{"key1": "value1", "key2": "value2", "key3": "value3"},
			expectedKeys: []Key{Key("key1"), Key("key2"), Key("key3")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMemoryStore()

			// Setup initial data
			for k, v := range tt.setupKeys {
				ctx := context.Background()
				err := store.Set(ctx, Key(k), v)
				if err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			}

			ctx := context.Background()
			keys, err := store.List(ctx)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(keys) != len(tt.expectedKeys) {
				t.Errorf("expected %d keys, got %d", len(tt.expectedKeys), len(keys))
			}

			// Convert to map for easier comparison since order is not guaranteed
			keyMap := make(map[Key]bool)
			for _, key := range keys {
				keyMap[key] = true
			}

			for _, expectedKey := range tt.expectedKeys {
				if !keyMap[expectedKey] {
					t.Errorf("expected key %s not found in result", expectedKey)
				}
			}
		})
	}
}

func TestMemoryStore_ListEntries(t *testing.T) {
	tests := []struct {
		name            string
		setupKeys       map[string]string
		expectedEntries int
	}{
		{
			name:            "list entries empty store",
			setupKeys:       map[string]string{},
			expectedEntries: 0,
		},
		{
			name:            "list entries single entry",
			setupKeys:       map[string]string{"key1": "value1"},
			expectedEntries: 1,
		},
		{
			name:            "list entries multiple entries",
			setupKeys:       map[string]string{"key1": "value1", "key2": "value2"},
			expectedEntries: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMemoryStore()

			// Setup initial data
			for k, v := range tt.setupKeys {
				ctx := context.Background()
				err := store.Set(ctx, Key(k), v)
				if err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			}

			ctx := context.Background()
			entries, err := store.ListEntries(ctx)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(entries) != tt.expectedEntries {
				t.Errorf("expected %d entries, got %d", tt.expectedEntries, len(entries))
			}

			// Verify each entry has correct structure
			for _, entry := range entries {
				if entry.Key == "" {
					t.Error("entry key should not be empty")
				}
				if entry.Value.Data == "" && tt.setupKeys[string(entry.Key)] != "" {
					t.Error("entry value data should not be empty for non-empty setup value")
				}
			}
		})
	}
}

func TestMemoryStore_Size(t *testing.T) {
	tests := []struct {
		name         string
		setupKeys    map[string]string
		expectedSize int
	}{
		{
			name:         "size empty store",
			setupKeys:    map[string]string{},
			expectedSize: 0,
		},
		{
			name:         "size single entry",
			setupKeys:    map[string]string{"key1": "value1"},
			expectedSize: 1,
		},
		{
			name:         "size multiple entries",
			setupKeys:    map[string]string{"key1": "value1", "key2": "value2", "key3": "value3"},
			expectedSize: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMemoryStore()

			// Setup initial data
			for k, v := range tt.setupKeys {
				ctx := context.Background()
				err := store.Set(ctx, Key(k), v)
				if err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			}

			ctx := context.Background()
			size, err := store.Size(ctx)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if size != tt.expectedSize {
				t.Errorf("expected size %d, got %d", tt.expectedSize, size)
			}
		})
	}
}

func TestMemoryStore_Clear(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Setup initial data
	setupKeys := map[string]string{"key1": "value1", "key2": "value2", "key3": "value3"}
	for k, v := range setupKeys {
		err := store.Set(ctx, Key(k), v)
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}
	}

	// Verify store has data
	size, err := store.Size(ctx)
	if err != nil {
		t.Fatalf("failed to get size: %v", err)
	}
	if size != 3 {
		t.Fatalf("expected size 3 before clear, got %d", size)
	}

	// Clear the store
	err = store.Clear(ctx)
	if err != nil {
		t.Errorf("unexpected error during clear: %v", err)
	}

	// Verify store is empty
	size, err = store.Size(ctx)
	if err != nil {
		t.Errorf("failed to get size after clear: %v", err)
	}
	if size != 0 {
		t.Errorf("expected size 0 after clear, got %d", size)
	}

	// Verify keys are gone
	for k := range setupKeys {
		_, err := store.Get(ctx, Key(k))
		if err != ErrKeyNotFound {
			t.Errorf("expected key %s to be not found after clear", k)
		}
	}
}

func TestMemoryStore_Exists(t *testing.T) {
	tests := []struct {
		name           string
		setupKeys      map[string]string
		key            Key
		expectedExists bool
		expectedError  error
	}{
		{
			name:           "exists for existing key",
			setupKeys:      map[string]string{"test-key": "test-value"},
			key:            Key("test-key"),
			expectedExists: true,
			expectedError:  nil,
		},
		{
			name:           "exists for non-existing key",
			setupKeys:      map[string]string{},
			key:            Key("non-existing"),
			expectedExists: false,
			expectedError:  nil,
		},
		{
			name:           "exists with empty key",
			setupKeys:      map[string]string{},
			key:            Key(""),
			expectedExists: false,
			expectedError:  ErrInvalidKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMemoryStore()

			// Setup initial data
			for k, v := range tt.setupKeys {
				ctx := context.Background()
				err := store.Set(ctx, Key(k), v)
				if err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			}

			ctx := context.Background()
			exists, err := store.Exists(ctx, tt.key)

			if tt.expectedError != nil {
				if err != tt.expectedError {
					t.Errorf("expected error %v, got %v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if exists != tt.expectedExists {
					t.Errorf("expected exists %v, got %v", tt.expectedExists, exists)
				}
			}
		})
	}
}

func TestMemoryStore_CompareAndSwap(t *testing.T) {
	tests := []struct {
		name            string
		setupKey        string
		setupValue      string
		key             Key
		expectedVersion int64
		newValue        string
		expectedError   error
		shouldSucceed   bool
	}{
		{
			name:            "successful compare and swap",
			setupKey:        "test-key",
			setupValue:      "original-value",
			key:             Key("test-key"),
			expectedVersion: 1,
			newValue:        "new-value",
			expectedError:   nil,
			shouldSucceed:   true,
		},
		{
			name:            "compare and swap version mismatch",
			setupKey:        "test-key",
			setupValue:      "original-value",
			key:             Key("test-key"),
			expectedVersion: 999, // Wrong version
			newValue:        "new-value",
			expectedError:   ErrConcurrentModification,
			shouldSucceed:   false,
		},
		{
			name:            "compare and swap non-existing key",
			setupKey:        "",
			setupValue:      "",
			key:             Key("non-existing"),
			expectedVersion: 1,
			newValue:        "new-value",
			expectedError:   ErrKeyNotFound,
			shouldSucceed:   false,
		},
		{
			name:            "compare and swap empty key",
			setupKey:        "",
			setupValue:      "",
			key:             Key(""),
			expectedVersion: 1,
			newValue:        "new-value",
			expectedError:   ErrInvalidKey,
			shouldSucceed:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMemoryStore()
			ctx := context.Background()

			// Setup initial data if needed
			if tt.setupKey != "" {
				err := store.Set(ctx, Key(tt.setupKey), tt.setupValue)
				if err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			}

			value, err := store.CompareAndSwap(ctx, tt.key, tt.expectedVersion, tt.newValue)

			if tt.expectedError != nil {
				if err != tt.expectedError {
					t.Errorf("expected error %v, got %v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			if tt.shouldSucceed {
				if value.Data != tt.newValue {
					t.Errorf("expected new value %s, got %s", tt.newValue, value.Data)
				}
				if value.Version != tt.expectedVersion+1 {
					t.Errorf("expected version %d, got %d", tt.expectedVersion+1, value.Version)
				}

				// Verify the value was actually updated
				storedValue, getErr := store.Get(ctx, tt.key)
				if getErr != nil {
					t.Errorf("failed to get updated value: %v", getErr)
				}
				if storedValue.Data != tt.newValue {
					t.Errorf("stored value %s does not match expected %s", storedValue.Data, tt.newValue)
				}
			}
		})
	}
}

func TestMemoryStore_Close(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Add some data
	err := store.Set(ctx, Key("test-key"), "test-value")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Close the store
	err = store.Close()
	if err != nil {
		t.Errorf("unexpected error during close: %v", err)
	}

	// Verify operations return ErrStoreClosed
	_, err = store.Get(ctx, Key("test-key"))
	if err != ErrStoreClosed {
		t.Errorf("expected ErrStoreClosed, got %v", err)
	}

	err = store.Set(ctx, Key("new-key"), "new-value")
	if err != ErrStoreClosed {
		t.Errorf("expected ErrStoreClosed, got %v", err)
	}

	// Close should be idempotent
	err = store.Close()
	if err != nil {
		t.Errorf("close should be idempotent, got error: %v", err)
	}
}

func TestMemoryStore_ConcurrentAccess(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Test concurrent reads and writes
	numGoroutines := 100
	numOperations := 10 // Reduced for cleaner test

	// Use channels to coordinate goroutines
	done := make(chan bool, numGoroutines)
	errorChan := make(chan error, numGoroutines*numOperations)

	// Concurrent writes
	for i := 0; i < numGoroutines/2; i++ {
		go func(id int) {
			for j := 0; j < numOperations; j++ {
				key := Key(fmt.Sprintf("key-%d-%d", id, j))
				value := fmt.Sprintf("value-%d-%d", id, j)
				err := store.Set(ctx, key, value)
				if err != nil {
					errorChan <- fmt.Errorf("concurrent set failed: %v", err)
				}
			}
			done <- true
		}(i)
	}

	// Concurrent reads
	for i := numGoroutines / 2; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numOperations; j++ {
				key := Key(fmt.Sprintf("key-%d-%d", id-numGoroutines/2, j))
				_, err := store.Get(ctx, key)
				// Error is expected for non-existing keys in concurrent scenario
				if err != nil && err != ErrKeyNotFound {
					errorChan <- fmt.Errorf("concurrent get failed with unexpected error: %v", err)
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Check for any errors
	close(errorChan)
	for err := range errorChan {
		t.Error(err)
	}

	// Verify store is still functional
	err := store.Set(ctx, Key("final-test"), "final-value")
	if err != nil {
		t.Errorf("store not functional after concurrent access: %v", err)
	}

	value, err := store.Get(ctx, Key("final-test"))
	if err != nil {
		t.Errorf("failed to get final test value: %v", err)
	}
	if value.Data != "final-value" {
		t.Errorf("expected final-value, got %s", value.Data)
	}
}

// TestMemoryStore_ConcurrentReaderWriter tests multiple readers and writers under race conditions
func TestMemoryStore_ConcurrentReaderWriter(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Test parameters
	numReaders := 50
	numWriters := 20
	numKeys := 100
	testDuration := 5 * time.Second

	// Pre-populate store with some data
	for i := 0; i < numKeys/2; i++ {
		key := Key(fmt.Sprintf("key-%d", i))
		value := fmt.Sprintf("initial-value-%d", i)
		if err := store.Set(ctx, key, value); err != nil {
			t.Fatalf("Failed to setup initial data: %v", err)
		}
	}

	var wg sync.WaitGroup
	var readErrors, writeErrors int64
	var readOps, writeOps int64
	done := make(chan struct{})

	// Start readers
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			localReadOps := 0
			for {
				select {
				case <-done:
					atomic.AddInt64(&readOps, int64(localReadOps))
					return
				default:
					keyIndex := rand.Intn(numKeys)
					key := Key(fmt.Sprintf("key-%d", keyIndex))
					_, err := store.Get(ctx, key)
					if err != nil && err != ErrKeyNotFound {
						atomic.AddInt64(&readErrors, 1)
					}
					localReadOps++
				}
			}
		}(i)
	}

	// Start writers
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			localWriteOps := 0
			for {
				select {
				case <-done:
					atomic.AddInt64(&writeOps, int64(localWriteOps))
					return
				default:
					keyIndex := rand.Intn(numKeys)
					key := Key(fmt.Sprintf("key-%d", keyIndex))
					value := fmt.Sprintf("writer-%d-value-%d", writerID, localWriteOps)
					if err := store.Set(ctx, key, value); err != nil {
						atomic.AddInt64(&writeErrors, 1)
					}
					localWriteOps++
				}
			}
		}(i)
	}

	// Run test for specified duration
	time.Sleep(testDuration)
	close(done)
	wg.Wait()

	// Report results
	totalReadOps := atomic.LoadInt64(&readOps)
	totalWriteOps := atomic.LoadInt64(&writeOps)
	totalReadErrors := atomic.LoadInt64(&readErrors)
	totalWriteErrors := atomic.LoadInt64(&writeErrors)

	t.Logf("Read operations: %d, errors: %d", totalReadOps, totalReadErrors)
	t.Logf("Write operations: %d, errors: %d", totalWriteOps, totalWriteErrors)

	if totalReadErrors > 0 {
		t.Errorf("Unexpected read errors: %d", totalReadErrors)
	}
	if totalWriteErrors > 0 {
		t.Errorf("Unexpected write errors: %d", totalWriteErrors)
	}

	// Verify store is still functional
	if err := store.Set(ctx, Key("test"), "test"); err != nil {
		t.Errorf("Store not functional after concurrent test: %v", err)
	}
}

// TestMemoryStore_CompareAndSwapHighContention tests CompareAndSwap under high contention
func TestMemoryStore_CompareAndSwapHighContention(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Setup initial key
	key := Key("contended-key")
	initialValue := "initial"
	if err := store.Set(ctx, key, initialValue); err != nil {
		t.Fatalf("Failed to setup initial key: %v", err)
	}

	numGoroutines := 100
	maxAttempts := 50
	var successfulSwaps, failedSwaps int64
	var wg sync.WaitGroup

	// Start many goroutines trying to CAS the same key
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for attempt := 0; attempt < maxAttempts; attempt++ {
				// Get current value and version
				currentValue, err := store.Get(ctx, key)
				if err != nil {
					continue
				}

				newValue := fmt.Sprintf("goroutine-%d-attempt-%d", goroutineID, attempt)

				// Try to swap
				_, err = store.CompareAndSwap(ctx, key, currentValue.Version, newValue)
				if err == nil {
					atomic.AddInt64(&successfulSwaps, 1)
					// Small random delay to increase contention
					time.Sleep(time.Duration(rand.Intn(10)) * time.Microsecond)
				} else if err == ErrConcurrentModification {
					atomic.AddInt64(&failedSwaps, 1)
				} else {
					t.Errorf("Unexpected CAS error: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()

	totalSuccessful := atomic.LoadInt64(&successfulSwaps)
	totalFailed := atomic.LoadInt64(&failedSwaps)

	t.Logf("Successful CAS operations: %d", totalSuccessful)
	t.Logf("Failed CAS operations (concurrent modification): %d", totalFailed)

	// Should have some successful operations
	if totalSuccessful < 1 {
		t.Error("Expected at least one successful CAS operation")
	}

	// Should have many failed operations due to contention
	if totalFailed < int64(numGoroutines) {
		t.Error("Expected more failed CAS operations due to contention")
	}

	// Verify final state is consistent
	finalValue, err := store.Get(ctx, key)
	if err != nil {
		t.Errorf("Failed to get final value: %v", err)
	}

	// Version should be initial version + successful swaps
	expectedVersion := int64(1) + totalSuccessful
	if finalValue.Version != expectedVersion {
		t.Errorf("Expected final version %d, got %d", expectedVersion, finalValue.Version)
	}
}

// TestMemoryStore_MemoryConsistency tests memory consistency under concurrent access
func TestMemoryStore_MemoryConsistency(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	numKeys := 50
	numGoroutines := 20
	testDuration := 3 * time.Second

	// Initialize keys with known values
	for i := 0; i < numKeys; i++ {
		key := Key(fmt.Sprintf("key-%d", i))
		value := fmt.Sprintf("initial-%d", i)
		if err := store.Set(ctx, key, value); err != nil {
			t.Fatalf("Failed to initialize key %s: %v", key, err)
		}
	}

	var wg sync.WaitGroup
	var inconsistencies int64
	done := make(chan struct{})

	// Start goroutines that verify consistency
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for {
				select {
				case <-done:
					return
				default:
					// Perform operations that should maintain consistency
					keyIndex := rand.Intn(numKeys)
					key := Key(fmt.Sprintf("key-%d", keyIndex))

					// Read current value
					value1, err1 := store.Get(ctx, key)
					if err1 != nil && err1 != ErrKeyNotFound {
						continue
					}

					// Update value
					newValue := fmt.Sprintf("goroutine-%d-update-%d", goroutineID, rand.Int())
					if err := store.Set(ctx, key, newValue); err != nil {
						continue
					}

					// Read again - should see either old or new value, never inconsistent state
					value2, err2 := store.Get(ctx, key)
					if err2 != nil {
						atomic.AddInt64(&inconsistencies, 1)
						continue
					}

					// Check version consistency
					if err1 == nil && value2.Version <= value1.Version {
						// This should not happen - version should always increase
						atomic.AddInt64(&inconsistencies, 1)
					}

					// Check timestamp consistency
					if !value2.UpdatedAt.After(value2.CreatedAt) && !value2.UpdatedAt.Equal(value2.CreatedAt) {
						atomic.AddInt64(&inconsistencies, 1)
					}
				}
			}
		}(i)
	}

	// Run test
	time.Sleep(testDuration)
	close(done)
	wg.Wait()

	totalInconsistencies := atomic.LoadInt64(&inconsistencies)
	if totalInconsistencies > 0 {
		t.Errorf("Found %d memory consistency violations", totalInconsistencies)
	}
}

// TestMemoryStore_DeadlockDetection tests for potential deadlocks
func TestMemoryStore_DeadlockDetection(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	numGoroutines := 50
	numKeys := 10
	testDuration := 5 * time.Second

	// Initialize keys
	for i := 0; i < numKeys; i++ {
		key := Key(fmt.Sprintf("key-%d", i))
		if err := store.Set(ctx, key, fmt.Sprintf("value-%d", i)); err != nil {
			t.Fatalf("Failed to initialize key: %v", err)
		}
	}

	var wg sync.WaitGroup
	done := make(chan struct{})
	deadlockDetected := make(chan struct{}, 1)

	// Start goroutines performing various operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for {
				select {
				case <-done:
					return
				default:
					// Mix of operations that could potentially cause deadlocks
					switch rand.Intn(6) {
					case 0:
						// Single get
						key := Key(fmt.Sprintf("key-%d", rand.Intn(numKeys)))
						_, _ = store.Get(ctx, key)
					case 1:
						// Single set
						key := Key(fmt.Sprintf("key-%d", rand.Intn(numKeys)))
						value := fmt.Sprintf("goroutine-%d-value", goroutineID)
						_ = store.Set(ctx, key, value)
					case 2:
						// List operation (acquires read lock on entire store)
						_, _ = store.List(ctx)
					case 3:
						// Size operation
						_, _ = store.Size(ctx)
					case 4:
						// Clear operation (acquires write lock on entire store)
						if rand.Intn(100) < 5 { // Only 5% chance to avoid too much disruption
							_ = store.Clear(ctx)
						}
					case 5:
						// Compare and swap
						key := Key(fmt.Sprintf("key-%d", rand.Intn(numKeys)))
						if value, err := store.Get(ctx, key); err == nil {
							newValue := fmt.Sprintf("cas-%d", goroutineID)
							_, _ = store.CompareAndSwap(ctx, key, value.Version, newValue)
						}
					}
				}
			}
		}(i)
	}

	// Deadlock detection goroutine
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		lastSize := -1
		stuckCount := 0

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				// Try to perform a simple operation
				size, err := store.Size(ctx)
				if err != nil {
					continue
				}

				// If size hasn't changed for too long, might indicate deadlock
				if size == lastSize {
					stuckCount++
					if stuckCount > 50 { // 5 seconds of no progress
						select {
						case deadlockDetected <- struct{}{}:
						default:
						}
						return
					}
				} else {
					stuckCount = 0
					lastSize = size
				}
			}
		}
	}()

	// Run test with timeout
	testComplete := make(chan struct{})
	go func() {
		time.Sleep(testDuration)
		close(done)
		wg.Wait()
		close(testComplete)
	}()

	select {
	case <-testComplete:
		// Test completed successfully
		t.Log("Deadlock detection test completed without deadlocks")
	case <-deadlockDetected:
		t.Error("Potential deadlock detected - operations appear stuck")
	case <-time.After(testDuration + 2*time.Second):
		t.Error("Test timed out - possible deadlock")
	}
}

// StressTestConfig holds configuration for stress tests
type StressTestConfig struct {
	NumGoroutines          int
	OperationsPerGoroutine int
	NumKeys                int
	ReadWriteRatio         float64 // 0.0 = all writes, 1.0 = all reads
	TestDuration           time.Duration
}

// TestMemoryStore_StressTest runs configurable stress tests
func TestMemoryStore_StressTest(t *testing.T) {
	configs := []StressTestConfig{
		{
			NumGoroutines:          100,
			OperationsPerGoroutine: 1000,
			NumKeys:                50,
			ReadWriteRatio:         0.7, // 70% reads, 30% writes
			TestDuration:           3 * time.Second,
		},
		{
			NumGoroutines:          500,
			OperationsPerGoroutine: 500,
			NumKeys:                100,
			ReadWriteRatio:         0.5, // 50/50 read/write
			TestDuration:           2 * time.Second,
		},
		{
			NumGoroutines:          1000,
			OperationsPerGoroutine: 100,
			NumKeys:                20,
			ReadWriteRatio:         0.9, // 90% reads, 10% writes
			TestDuration:           1 * time.Second,
		},
	}

	for _, config := range configs {
		t.Run(fmt.Sprintf("StressTest_%d_goroutines_%d_ops", config.NumGoroutines, config.OperationsPerGoroutine), func(t *testing.T) {
			runStressTest(t, config)
		})
	}
}

func runStressTest(t *testing.T, config StressTestConfig) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Pre-populate store
	for i := 0; i < config.NumKeys/2; i++ {
		key := Key(fmt.Sprintf("key-%d", i))
		value := fmt.Sprintf("initial-value-%d", i)
		if err := store.Set(ctx, key, value); err != nil {
			t.Fatalf("Failed to setup initial data: %v", err)
		}
	}

	var wg sync.WaitGroup
	var readOps, writeOps, errors int64
	done := make(chan struct{})

	startTime := time.Now()

	// Start goroutines
	for i := 0; i < config.NumGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			operations := 0
			for operations < config.OperationsPerGoroutine {
				select {
				case <-done:
					return
				default:
					keyIndex := rand.Intn(config.NumKeys)
					key := Key(fmt.Sprintf("key-%d", keyIndex))

					if rand.Float64() < config.ReadWriteRatio {
						// Read operation
						_, err := store.Get(ctx, key)
						if err != nil && err != ErrKeyNotFound {
							atomic.AddInt64(&errors, 1)
						}
						atomic.AddInt64(&readOps, 1)
					} else {
						// Write operation
						value := fmt.Sprintf("goroutine-%d-op-%d", goroutineID, operations)
						if err := store.Set(ctx, key, value); err != nil {
							atomic.AddInt64(&errors, 1)
						}
						atomic.AddInt64(&writeOps, 1)
					}
					operations++
				}
			}
		}(i)
	}

	// Wait for completion or timeout
	testDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(testDone)
	}()

	select {
	case <-testDone:
		// Completed normally
	case <-time.After(config.TestDuration):
		close(done)
		wg.Wait()
	}

	duration := time.Since(startTime)
	totalReadOps := atomic.LoadInt64(&readOps)
	totalWriteOps := atomic.LoadInt64(&writeOps)
	totalErrors := atomic.LoadInt64(&errors)
	totalOps := totalReadOps + totalWriteOps

	// Report performance metrics
	opsPerSecond := float64(totalOps) / duration.Seconds()
	t.Logf("Duration: %v", duration)
	t.Logf("Total operations: %d (reads: %d, writes: %d)", totalOps, totalReadOps, totalWriteOps)
	t.Logf("Operations per second: %.2f", opsPerSecond)
	t.Logf("Errors: %d", totalErrors)

	if totalErrors > 0 {
		t.Errorf("Stress test failed with %d errors", totalErrors)
	}

	// Verify store is still functional
	if err := store.Set(ctx, Key("post-stress-test"), "test-value"); err != nil {
		t.Errorf("Store not functional after stress test: %v", err)
	}
}

// TestMemoryStore_EdgeCases tests edge cases like near capacity and version overflow
func TestMemoryStore_EdgeCases(t *testing.T) {
	t.Run("HighVersionNumbers", func(t *testing.T) {
		store := NewMemoryStore()
		ctx := context.Background()

		key := Key("version-test-key")

		// Set initial value
		if err := store.Set(ctx, key, "initial"); err != nil {
			t.Fatalf("Failed to set initial value: %v", err)
		}

		// Update many times to get high version numbers
		for i := 0; i < 10000; i++ {
			value := fmt.Sprintf("update-%d", i)
			if err := store.Set(ctx, key, value); err != nil {
				t.Fatalf("Failed to update at iteration %d: %v", i, err)
			}
		}

		// Verify final state
		finalValue, err := store.Get(ctx, key)
		if err != nil {
			t.Fatalf("Failed to get final value: %v", err)
		}

		expectedVersion := int64(10001) // Initial + 10000 updates
		if finalValue.Version != expectedVersion {
			t.Errorf("Expected version %d, got %d", expectedVersion, finalValue.Version)
		}
	})

	t.Run("LargeNumberOfKeys", func(t *testing.T) {
		store := NewMemoryStore()
		ctx := context.Background()

		numKeys := 100000

		// Add many keys
		for i := 0; i < numKeys; i++ {
			key := Key(fmt.Sprintf("key-%06d", i))
			value := fmt.Sprintf("value-%d", i)
			if err := store.Set(ctx, key, value); err != nil {
				t.Fatalf("Failed to set key %d: %v", i, err)
			}

			// Check progress periodically
			if i%10000 == 0 {
				size, err := store.Size(ctx)
				if err != nil {
					t.Fatalf("Failed to get size at %d keys: %v", i, err)
				}
				if size != i+1 {
					t.Errorf("Expected size %d, got %d", i+1, size)
				}
			}
		}

		// Verify final size
		finalSize, err := store.Size(ctx)
		if err != nil {
			t.Fatalf("Failed to get final size: %v", err)
		}
		if finalSize != numKeys {
			t.Errorf("Expected final size %d, got %d", numKeys, finalSize)
		}

		// Test random access
		for i := 0; i < 1000; i++ {
			keyIndex := rand.Intn(numKeys)
			key := Key(fmt.Sprintf("key-%06d", keyIndex))
			expectedValue := fmt.Sprintf("value-%d", keyIndex)

			value, err := store.Get(ctx, key)
			if err != nil {
				t.Errorf("Failed to get key %s: %v", key, err)
				continue
			}
			if value.Data != expectedValue {
				t.Errorf("Key %s: expected %s, got %s", key, expectedValue, value.Data)
			}
		}
	})
}

// TestMemoryStore_ContextCancellation tests context cancellation under load
func TestMemoryStore_ContextCancellation(t *testing.T) {
	store := NewMemoryStore()

	t.Run("CancelledContextDuringOperations", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		// Start some background operations
		var wg sync.WaitGroup
		numGoroutines := 50

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				for j := 0; j < 1000; j++ {
					key := Key(fmt.Sprintf("cancel-test-%d-%d", goroutineID, j))

					// This should work initially
					if err := store.Set(ctx, key, "test-value"); err != nil {
						if err == context.Canceled {
							return // Expected when context is cancelled
						}
						t.Errorf("Unexpected error: %v", err)
						return
					}

					// Small delay to allow cancellation to take effect
					time.Sleep(time.Microsecond)
				}
			}(i)
		}

		// Cancel context after short delay
		time.Sleep(10 * time.Millisecond)
		cancel()

		wg.Wait()

		// Verify store is still functional with new context
		newCtx := context.Background()
		if err := store.Set(newCtx, Key("post-cancel-test"), "test-value"); err != nil {
			t.Errorf("Store not functional after context cancellation: %v", err)
		}
	})

	t.Run("TimeoutDuringHighLoad", func(t *testing.T) {
		// Create context with very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// Ensure timeout has passed
		time.Sleep(1 * time.Millisecond)

		// Operations should fail with deadline exceeded
		err := store.Set(ctx, Key("timeout-test"), "value")
		if err != context.DeadlineExceeded {
			t.Errorf("Expected context.DeadlineExceeded, got %v", err)
		}

		_, err = store.Get(ctx, Key("timeout-test"))
		if err != context.DeadlineExceeded {
			t.Errorf("Expected context.DeadlineExceeded, got %v", err)
		}
	})
}

// Benchmark tests for performance analysis

// BenchmarkMemoryStore_Read tests read performance
func BenchmarkMemoryStore_Read(b *testing.B) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Setup data
	numKeys := 1000
	for i := 0; i < numKeys; i++ {
		key := Key(fmt.Sprintf("benchmark-key-%d", i))
		value := fmt.Sprintf("benchmark-value-%d", i)
		if err := store.Set(ctx, key, value); err != nil {
			b.Fatalf("Failed to setup data: %v", err)
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			keyIndex := rand.Intn(numKeys)
			key := Key(fmt.Sprintf("benchmark-key-%d", keyIndex))
			_, err := store.Get(ctx, key)
			if err != nil {
				b.Errorf("Read failed: %v", err)
			}
		}
	})
}

// BenchmarkMemoryStore_Write tests write performance
func BenchmarkMemoryStore_Write(b *testing.B) {
	store := NewMemoryStore()
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		goroutineID := runtime.GOMAXPROCS(0) // Use as approximation for goroutine ID
		counter := 0
		for pb.Next() {
			key := Key(fmt.Sprintf("write-bench-%d-%d", goroutineID, counter))
			value := fmt.Sprintf("value-%d-%d", goroutineID, counter)
			if err := store.Set(ctx, key, value); err != nil {
				b.Errorf("Write failed: %v", err)
			}
			counter++
		}
	})
}

// BenchmarkMemoryStore_MixedWorkload tests mixed read/write performance
func BenchmarkMemoryStore_MixedWorkload(b *testing.B) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Pre-populate with some data
	numInitialKeys := 1000
	for i := 0; i < numInitialKeys; i++ {
		key := Key(fmt.Sprintf("mixed-key-%d", i))
		value := fmt.Sprintf("mixed-value-%d", i)
		if err := store.Set(ctx, key, value); err != nil {
			b.Fatalf("Failed to setup data: %v", err)
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		goroutineID := runtime.GOMAXPROCS(0)
		counter := 0
		for pb.Next() {
			if rand.Float64() < 0.7 { // 70% reads, 30% writes
				// Read operation
				keyIndex := rand.Intn(numInitialKeys)
				key := Key(fmt.Sprintf("mixed-key-%d", keyIndex))
				_, err := store.Get(ctx, key)
				if err != nil && err != ErrKeyNotFound {
					b.Errorf("Read failed: %v", err)
				}
			} else {
				// Write operation
				key := Key(fmt.Sprintf("mixed-write-%d-%d", goroutineID, counter))
				value := fmt.Sprintf("value-%d-%d", goroutineID, counter)
				if err := store.Set(ctx, key, value); err != nil {
					b.Errorf("Write failed: %v", err)
				}
			}
			counter++
		}
	})
}

// BenchmarkMemoryStore_CompareAndSwap tests CAS performance
func BenchmarkMemoryStore_CompareAndSwap(b *testing.B) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Setup initial keys
	numKeys := 100
	for i := 0; i < numKeys; i++ {
		key := Key(fmt.Sprintf("cas-key-%d", i))
		value := fmt.Sprintf("initial-value-%d", i)
		if err := store.Set(ctx, key, value); err != nil {
			b.Fatalf("Failed to setup data: %v", err)
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		goroutineID := runtime.GOMAXPROCS(0)
		counter := 0
		for pb.Next() {
			keyIndex := rand.Intn(numKeys)
			key := Key(fmt.Sprintf("cas-key-%d", keyIndex))

			// Get current value
			currentValue, err := store.Get(ctx, key)
			if err != nil {
				continue
			}

			// Try CAS
			newValue := fmt.Sprintf("cas-value-%d-%d", goroutineID, counter)
			_, err = store.CompareAndSwap(ctx, key, currentValue.Version, newValue)
			// Don't fail on concurrent modification as it's expected in high contention
			if err != nil && err != ErrConcurrentModification {
				b.Errorf("CAS failed: %v", err)
			}
			counter++
		}
	})
}

// BenchmarkMemoryStore_HighContentionRead tests read performance under high contention
func BenchmarkMemoryStore_HighContentionRead(b *testing.B) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Setup limited number of keys to create high contention
	numKeys := 10
	for i := 0; i < numKeys; i++ {
		key := Key(fmt.Sprintf("contention-key-%d", i))
		value := fmt.Sprintf("contention-value-%d", i)
		if err := store.Set(ctx, key, value); err != nil {
			b.Fatalf("Failed to setup data: %v", err)
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			keyIndex := rand.Intn(numKeys)
			key := Key(fmt.Sprintf("contention-key-%d", keyIndex))
			_, err := store.Get(ctx, key)
			if err != nil {
				b.Errorf("Read failed: %v", err)
			}
		}
	})
}

// BenchmarkMemoryStore_HighContentionWrite tests write performance under high contention
func BenchmarkMemoryStore_HighContentionWrite(b *testing.B) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Setup limited number of keys to create high contention
	numKeys := 10
	for i := 0; i < numKeys; i++ {
		key := Key(fmt.Sprintf("contention-write-key-%d", i))
		value := fmt.Sprintf("initial-value-%d", i)
		if err := store.Set(ctx, key, value); err != nil {
			b.Fatalf("Failed to setup data: %v", err)
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		goroutineID := runtime.GOMAXPROCS(0)
		counter := 0
		for pb.Next() {
			keyIndex := rand.Intn(numKeys)
			key := Key(fmt.Sprintf("contention-write-key-%d", keyIndex))
			value := fmt.Sprintf("updated-value-%d-%d", goroutineID, counter)
			if err := store.Set(ctx, key, value); err != nil {
				b.Errorf("Write failed: %v", err)
			}
			counter++
		}
	})
}

// BenchmarkMemoryStore_List tests list operation performance
func BenchmarkMemoryStore_List(b *testing.B) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Setup data with varying number of keys
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			// Clear store first
			_ = store.Clear(ctx)

			// Add keys
			for i := 0; i < size; i++ {
				key := Key(fmt.Sprintf("list-key-%d", i))
				value := fmt.Sprintf("list-value-%d", i)
				if err := store.Set(ctx, key, value); err != nil {
					b.Fatalf("Failed to setup data: %v", err)
				}
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := store.List(ctx)
				if err != nil {
					b.Errorf("List failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkMemoryStore_Size tests size operation performance
func BenchmarkMemoryStore_Size(b *testing.B) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Setup data
	numKeys := 10000
	for i := 0; i < numKeys; i++ {
		key := Key(fmt.Sprintf("size-key-%d", i))
		value := fmt.Sprintf("size-value-%d", i)
		if err := store.Set(ctx, key, value); err != nil {
			b.Fatalf("Failed to setup data: %v", err)
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := store.Size(ctx)
			if err != nil {
				b.Errorf("Size failed: %v", err)
			}
		}
	})
}

// BenchmarkMemoryStore_Exists tests exists operation performance
func BenchmarkMemoryStore_Exists(b *testing.B) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Setup data
	numKeys := 1000
	for i := 0; i < numKeys; i++ {
		key := Key(fmt.Sprintf("exists-key-%d", i))
		value := fmt.Sprintf("exists-value-%d", i)
		if err := store.Set(ctx, key, value); err != nil {
			b.Fatalf("Failed to setup data: %v", err)
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			keyIndex := rand.Intn(numKeys * 2) // Half will be non-existent
			key := Key(fmt.Sprintf("exists-key-%d", keyIndex))
			_, err := store.Exists(ctx, key)
			if err != nil {
				b.Errorf("Exists failed: %v", err)
			}
		}
	})
}

// BenchmarkMemoryStore_ScalabilityTest tests performance with different goroutine counts
func BenchmarkMemoryStore_ScalabilityTest(b *testing.B) {
	goroutineCounts := []int{1, 2, 4, 8, 16, 32, 64, 128}

	for _, count := range goroutineCounts {
		b.Run(fmt.Sprintf("goroutines_%d", count), func(b *testing.B) {
			store := NewMemoryStore()
			ctx := context.Background()

			// Setup data
			numKeys := 1000
			for i := 0; i < numKeys; i++ {
				key := Key(fmt.Sprintf("scale-key-%d", i))
				value := fmt.Sprintf("scale-value-%d", i)
				if err := store.Set(ctx, key, value); err != nil {
					b.Fatalf("Failed to setup data: %v", err)
				}
			}

			// Set GOMAXPROCS to control parallelism
			oldMaxProcs := runtime.GOMAXPROCS(count)
			defer runtime.GOMAXPROCS(oldMaxProcs)

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					keyIndex := rand.Intn(numKeys)
					key := Key(fmt.Sprintf("scale-key-%d", keyIndex))

					if rand.Float64() < 0.8 { // 80% reads
						_, err := store.Get(ctx, key)
						if err != nil {
							b.Errorf("Read failed: %v", err)
						}
					} else { // 20% writes
						value := fmt.Sprintf("updated-value-%d", rand.Int())
						if err := store.Set(ctx, key, value); err != nil {
							b.Errorf("Write failed: %v", err)
						}
					}
				}
			})
		})
	}
}
