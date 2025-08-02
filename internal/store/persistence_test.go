package store

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestJSONFilePersistence_Save tests the Save method of JSON file persistence
func TestJSONFilePersistence_Save(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "persistence_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test_store.json")

	// Create a JSON file persistence instance
	persistence := NewJSONFilePersistence(testFile)

	// Create a test snapshot
	snapshot := &StoreSnapshot{
		Data: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		Stats: StoreStats{
			TotalKeys:      2,
			TotalRequests:  10,
			GetRequests:    4,
			SetRequests:    4,
			DeleteRequests: 2,
		},
		Version:   "1.0",
		Timestamp: time.Now().Unix(),
	}

	ctx := context.Background()

	// Save the snapshot - this should work without error
	err = persistence.Save(ctx, snapshot)
	if err != nil {
		t.Errorf("Save failed: %v", err)
	}

	// Verify that the file was created
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("Expected file to be created, but it doesn't exist")
	}
}

// TestJSONFilePersistence_Save_InvalidPath tests saving to an invalid path
func TestJSONFilePersistence_Save_InvalidPath(t *testing.T) {
	// Use an invalid path that should cause an error
	invalidPath := "/invalid/nonexistent/path/test.json"
	persistence := NewJSONFilePersistence(invalidPath)

	snapshot := &StoreSnapshot{
		Data:      map[string]string{"key": "value"},
		Stats:     StoreStats{TotalKeys: 1},
		Version:   "1.0",
		Timestamp: time.Now().Unix(),
	}

	ctx := context.Background()
	err := persistence.Save(ctx, snapshot)

	// Should return an error for invalid path
	if err == nil {
		t.Error("Expected error when saving to invalid path, but got nil")
	}
}

// TestJSONFilePersistence_Load tests the Load method of JSON file persistence
func TestJSONFilePersistence_Load(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "persistence_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test_store.json")
	persistence := NewJSONFilePersistence(testFile)

	// Create and save a test snapshot first
	originalSnapshot := &StoreSnapshot{
		Data: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		Stats: StoreStats{
			TotalKeys:      2,
			TotalRequests:  10,
			GetRequests:    4,
			SetRequests:    4,
			DeleteRequests: 2,
		},
		Version:   "1.0",
		Timestamp: 1234567890,
	}

	ctx := context.Background()

	// Save the snapshot first
	err = persistence.Save(ctx, originalSnapshot)
	if err != nil {
		t.Fatalf("Failed to save snapshot: %v", err)
	}

	// Now load it back
	loadedSnapshot, err := persistence.Load(ctx)
	if err != nil {
		t.Errorf("Load failed: %v", err)
	}

	if loadedSnapshot == nil {
		t.Fatal("Loaded snapshot is nil")
	}

	// Verify the loaded data matches the original
	if len(loadedSnapshot.Data) != len(originalSnapshot.Data) {
		t.Errorf("Expected %d keys, got %d", len(originalSnapshot.Data), len(loadedSnapshot.Data))
	}

	for key, expectedValue := range originalSnapshot.Data {
		if actualValue, exists := loadedSnapshot.Data[key]; !exists {
			t.Errorf("Key %s not found in loaded snapshot", key)
		} else if actualValue != expectedValue {
			t.Errorf("For key %s, expected %s, got %s", key, expectedValue, actualValue)
		}
	}

	// Verify metadata
	if loadedSnapshot.Version != originalSnapshot.Version {
		t.Errorf("Expected version %s, got %s", originalSnapshot.Version, loadedSnapshot.Version)
	}

	if loadedSnapshot.Timestamp != originalSnapshot.Timestamp {
		t.Errorf("Expected timestamp %d, got %d", originalSnapshot.Timestamp, loadedSnapshot.Timestamp)
	}
}

// TestJSONFilePersistence_Load_NonexistentFile tests loading from a file that doesn't exist
func TestJSONFilePersistence_Load_NonexistentFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "persistence_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	nonexistentFile := filepath.Join(tempDir, "nonexistent.json")
	persistence := NewJSONFilePersistence(nonexistentFile)

	ctx := context.Background()
	snapshot, err := persistence.Load(ctx)

	// Should return error for nonexistent file
	if err == nil {
		t.Error("Expected error when loading nonexistent file, but got nil")
	}

	if snapshot != nil {
		t.Error("Expected nil snapshot when loading nonexistent file")
	}
}

// TestJSONFilePersistence_AtomicWrites tests that saves are atomic (no partial writes)
func TestJSONFilePersistence_AtomicWrites(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "persistence_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test_store.json")

	// Write some initial valid JSON
	initialData := `{"data":{"initial":"value"},"stats":{"total_keys":1},"version":"1.0","timestamp":1234567890}`
	err = os.WriteFile(testFile, []byte(initialData), 0644)
	if err != nil {
		t.Fatalf("Failed to write initial data: %v", err)
	}

	persistence := NewJSONFilePersistence(testFile)

	// Create a test snapshot
	snapshot := &StoreSnapshot{
		Data: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		Stats: StoreStats{
			TotalKeys:      2,
			TotalRequests:  10,
			GetRequests:    4,
			SetRequests:    4,
			DeleteRequests: 2,
		},
		Version:   "1.0",
		Timestamp: time.Now().Unix(),
	}

	ctx := context.Background()

	// Save the snapshot
	err = persistence.Save(ctx, snapshot)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// The file should always contain valid JSON after save
	loadedSnapshot, err := persistence.Load(ctx)
	if err != nil {
		t.Errorf("Failed to load after save: %v", err)
	}

	if loadedSnapshot == nil {
		t.Fatal("Loaded snapshot is nil after atomic save")
	}

	// Verify the data was completely updated, not partially
	if len(loadedSnapshot.Data) != 2 {
		t.Errorf("Expected 2 keys after atomic save, got %d", len(loadedSnapshot.Data))
	}
}

// TestJSONFilePersistence_CorruptedFile tests handling of corrupted JSON files
func TestJSONFilePersistence_CorruptedFile(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "persistence_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "corrupted.json")

	// Write corrupted JSON data
	corruptedData := `{"data":{"key1":"value1"},corrupted json}`
	err = os.WriteFile(testFile, []byte(corruptedData), 0644)
	if err != nil {
		t.Fatalf("Failed to write corrupted data: %v", err)
	}

	persistence := NewJSONFilePersistence(testFile)
	ctx := context.Background()

	// Loading corrupted file should return an error
	snapshot, err := persistence.Load(ctx)
	if err == nil {
		t.Error("Expected error when loading corrupted file, but got nil")
	}

	if snapshot != nil {
		t.Error("Expected nil snapshot when loading corrupted file")
	}
}

// TestJSONFilePersistence_InvalidSnapshot tests handling of invalid snapshot data
func TestJSONFilePersistence_InvalidSnapshot(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "persistence_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "invalid.json")

	// Write JSON with missing required fields
	invalidData := `{"data":{"key1":"value1"}}` // Missing stats, version, timestamp
	err = os.WriteFile(testFile, []byte(invalidData), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid data: %v", err)
	}

	persistence := NewJSONFilePersistence(testFile)
	ctx := context.Background()

	// Loading invalid snapshot should return validation error
	snapshot, err := persistence.Load(ctx)
	if err == nil {
		t.Error("Expected error when loading invalid snapshot, but got nil")
	}

	if snapshot != nil {
		t.Error("Expected nil snapshot when loading invalid snapshot")
	}
}

// TestJSONFilePersistence_ConcurrentAccess tests concurrent save and load operations
func TestJSONFilePersistence_ConcurrentAccess(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "persistence_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "concurrent_test.json")
	persistence := NewJSONFilePersistence(testFile)

	ctx := context.Background()

	// Number of concurrent operations
	numOperations := 50
	var wg sync.WaitGroup
	errors := make(chan error, numOperations*2) // For saves and loads

	// Concurrent saves
	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			snapshot := &StoreSnapshot{
				Data: map[string]string{
					"key": "value_" + string(rune(i)),
				},
				Stats: StoreStats{
					TotalKeys:      1,
					TotalRequests:  i,
					GetRequests:    i / 3,
					SetRequests:    i / 3,
					DeleteRequests: i / 3,
				},
				Version:   "1.0",
				Timestamp: time.Now().Unix(),
			}

			if err := persistence.Save(ctx, snapshot); err != nil {
				errors <- err
			}
		}(i)
	}

	// Concurrent loads
	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			_, err := persistence.Load(ctx)
			// Don't treat "no snapshot found" as an error since saves might not have completed yet
			if err != nil && err.Error() != "persistence load error: no snapshot found" {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		t.Errorf("Concurrent operation failed: %v", err)
	}

	// Final verification: the file should be in a consistent state
	finalSnapshot, err := persistence.Load(ctx)
	if err != nil {
		t.Errorf("Failed to load final snapshot: %v", err)
	}

	if finalSnapshot == nil {
		t.Error("Final snapshot is nil after concurrent operations")
	}
}

// TestJSONFilePersistence_ConcurrentSaveLoad tests specific save-load race conditions
func TestJSONFilePersistence_ConcurrentSaveLoad(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "persistence_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "race_test.json")
	persistence := NewJSONFilePersistence(testFile)

	ctx := context.Background()

	// Create initial snapshot
	initialSnapshot := &StoreSnapshot{
		Data: map[string]string{
			"initial": "value",
		},
		Stats:     StoreStats{TotalKeys: 1},
		Version:   "1.0",
		Timestamp: time.Now().Unix(),
	}

	err = persistence.Save(ctx, initialSnapshot)
	if err != nil {
		t.Fatalf("Failed to save initial snapshot: %v", err)
	}

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	// Start multiple goroutines that save and load simultaneously
	for i := 0; i < 10; i++ {
		wg.Add(2)

		// Saver goroutine
		go func(i int) {
			defer wg.Done()

			for j := 0; j < 5; j++ {
				snapshot := &StoreSnapshot{
					Data: map[string]string{
						"key": "saver_" + string(rune(i)) + "_" + string(rune(j)),
					},
					Stats:     StoreStats{TotalKeys: 1},
					Version:   "1.0",
					Timestamp: time.Now().Unix(),
				}

				if err := persistence.Save(ctx, snapshot); err != nil {
					errors <- err
				}

				// Small delay to increase chance of race conditions
				time.Sleep(time.Microsecond)
			}
		}(i)

		// Loader goroutine
		go func() {
			defer wg.Done()

			for j := 0; j < 5; j++ {
				if _, err := persistence.Load(ctx); err != nil {
					errors <- err
				}

				// Small delay to increase chance of race conditions
				time.Sleep(time.Microsecond)
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		t.Errorf("Race condition detected: %v", err)
	}
}

// TestJSONFilePersistence_NilSnapshot tests saving a nil snapshot
func TestJSONFilePersistence_NilSnapshot(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "persistence_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "nil_test.json")
	persistence := NewJSONFilePersistence(testFile)

	ctx := context.Background()

	// Attempt to save nil snapshot
	err = persistence.Save(ctx, nil)
	if err == nil {
		t.Error("Expected error when saving nil snapshot, but got nil")
	}

	if err != nil && err.Error() != "snapshot is nil" {
		t.Errorf("Expected 'snapshot is nil' error, got: %v", err)
	}
}

// TestJSONFilePersistence_EmptyFilePath tests with empty file path
func TestJSONFilePersistence_EmptyFilePath(t *testing.T) {
	persistence := NewJSONFilePersistence("")

	snapshot := &StoreSnapshot{
		Data:      map[string]string{"key": "value"},
		Stats:     StoreStats{TotalKeys: 1},
		Version:   "1.0",
		Timestamp: time.Now().Unix(),
	}

	ctx := context.Background()

	// Save should fail with empty path
	err := persistence.Save(ctx, snapshot)
	if err == nil {
		t.Error("Expected error when saving with empty file path, but got nil")
	}
}

// TestJSONFilePersistence_ReadOnlyDirectory tests saving to read-only directory
func TestJSONFilePersistence_ReadOnlyDirectory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "persistence_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Make directory read-only
	err = os.Chmod(tempDir, 0444)
	if err != nil {
		t.Fatalf("Failed to make directory read-only: %v", err)
	}
	defer func() {
		_ = os.Chmod(tempDir, 0755) // Restore permissions for cleanup
	}()

	testFile := filepath.Join(tempDir, "readonly_test.json")
	persistence := NewJSONFilePersistence(testFile)

	snapshot := &StoreSnapshot{
		Data:      map[string]string{"key": "value"},
		Stats:     StoreStats{TotalKeys: 1},
		Version:   "1.0",
		Timestamp: time.Now().Unix(),
	}

	ctx := context.Background()

	// Save should fail in read-only directory
	err = persistence.Save(ctx, snapshot)
	if err == nil {
		t.Error("Expected error when saving to read-only directory, but got nil")
	}
}

// TestJSONFilePersistence_CancelledContext tests context cancellation
func TestJSONFilePersistence_CancelledContext(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "persistence_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "context_test.json")
	persistence := NewJSONFilePersistence(testFile)

	snapshot := &StoreSnapshot{
		Data:      map[string]string{"key": "value"},
		Stats:     StoreStats{TotalKeys: 1},
		Version:   "1.0",
		Timestamp: time.Now().Unix(),
	}

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Operations with cancelled context should still work in our current implementation
	// (since we don't check context during file operations)
	// But this tests the behavior
	err = persistence.Save(ctx, snapshot)
	if err != nil {
		t.Logf("Save with cancelled context returned: %v", err)
	}

	_, err = persistence.Load(ctx)
	if err != nil {
		t.Logf("Load with cancelled context returned: %v", err)
	}
}
