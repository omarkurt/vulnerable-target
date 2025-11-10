package disk

import (
	"os"
	"path/filepath"
	"testing"

	bolt "go.etcd.io/bbolt"
)

// TestStruct is a simple struct for testing
type TestStruct struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// setupTestStore creates a test store with a temporary database
func setupTestStore(t *testing.T) (*Store[TestStruct], string, func()) {
	t.Helper()

	// Create a temporary directory for the test database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	config := NewConfig().
		WithFileName(dbPath).
		WithBucketName("testbucket")

	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	store := &Store[TestStruct]{
		db:     db,
		Config: config,
	}

	cleanup := func() {
		store.Close()
		os.RemoveAll(tempDir)
	}

	return store, dbPath, cleanup
}

func TestNewStorageStore(t *testing.T) {
	config := NewConfig().
		WithFileName("test_new_storage.db").
		WithBucketName("testbucket")

	store, err := NewStorageStore[TestStruct](config)
	if err != nil {
		t.Fatalf("NewStorageStore() failed: %v", err)
	}
	defer func() {
		store.Close()
		// Clean up the created database file
		userHomeDir, _ := os.UserHomeDir()
		vtHomeDir := filepath.Join(userHomeDir, ".vt")
		os.Remove(filepath.Join(vtHomeDir, "test_new_storage.db"))
	}()

	if store == nil {
		t.Fatal("NewStorageStore() returned nil store")
	}

	if store.Config != config {
		t.Error("Store config does not match provided config")
	}

	if store.db == nil {
		t.Error("Store database is nil")
	}
}

func TestNewStorageStore_CreatesVtDirectory(t *testing.T) {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("Cannot get user home directory: %v", err)
	}

	vtHomeDir := filepath.Join(userHomeDir, ".vt")

	config := NewConfig().
		WithFileName("test_directory_creation.db").
		WithBucketName("testbucket")

	store, err := NewStorageStore[TestStruct](config)
	if err != nil {
		t.Fatalf("NewStorageStore() failed: %v", err)
	}
	defer func() {
		store.Close()
		os.Remove(filepath.Join(vtHomeDir, "test_directory_creation.db"))
	}()

	// Check that the .vt directory exists
	if _, err := os.Stat(vtHomeDir); os.IsNotExist(err) {
		t.Errorf(".vt directory was not created at %s", vtHomeDir)
	}
}

func TestStore_Set(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	tests := []struct {
		name  string
		key   string
		value TestStruct
	}{
		{
			name: "simple set",
			key:  "key1",
			value: TestStruct{
				ID:   1,
				Name: "Test1",
			},
		},
		{
			name: "set with special characters in key",
			key:  "key-with-special_chars.123",
			value: TestStruct{
				ID:   2,
				Name: "Test2",
			},
		},
		{
			name: "set with empty name",
			key:  "key3",
			value: TestStruct{
				ID:   3,
				Name: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.Set(tt.key, tt.value)
			if err != nil {
				t.Errorf("Set() failed: %v", err)
			}

			// Verify the value was set
			retrieved, err := store.Get(tt.key)
			if err != nil {
				t.Errorf("Get() after Set() failed: %v", err)
			}

			if retrieved.ID != tt.value.ID || retrieved.Name != tt.value.Name {
				t.Errorf("Retrieved value mismatch: got %+v, want %+v", retrieved, tt.value)
			}
		})
	}
}

func TestStore_Set_Overwrite(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	key := "overwrite-key"
	value1 := TestStruct{ID: 1, Name: "First"}
	value2 := TestStruct{ID: 2, Name: "Second"}

	// Set initial value
	if err := store.Set(key, value1); err != nil {
		t.Fatalf("First Set() failed: %v", err)
	}

	// Overwrite with new value
	if err := store.Set(key, value2); err != nil {
		t.Fatalf("Second Set() failed: %v", err)
	}

	// Verify the value was overwritten
	retrieved, err := store.Get(key)
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if retrieved.ID != value2.ID || retrieved.Name != value2.Name {
		t.Errorf("Retrieved value should be overwritten: got %+v, want %+v", retrieved, value2)
	}
}

func TestStore_Get(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	// Set up test data
	testData := TestStruct{ID: 42, Name: "TestData"}
	key := "test-key"

	err := store.Set(key, testData)
	if err != nil {
		t.Fatalf("Set() failed: %v", err)
	}

	// Test getting the data
	retrieved, err := store.Get(key)
	if err != nil {
		t.Errorf("Get() failed: %v", err)
	}

	if retrieved.ID != testData.ID {
		t.Errorf("ID mismatch: got %d, want %d", retrieved.ID, testData.ID)
	}

	if retrieved.Name != testData.Name {
		t.Errorf("Name mismatch: got %q, want %q", retrieved.Name, testData.Name)
	}
}

func TestStore_Get_NonExistentKey(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	_, err := store.Get("nonexistent-key")
	if err == nil {
		t.Error("Get() should return error for non-existent key")
	}
}

func TestStore_Get_NonExistentBucket(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	// Change bucket name to one that doesn't exist
	originalBucket := store.Config.BucketName
	store.Config.BucketName = "nonexistent-bucket"

	_, err := store.Get("any-key")
	if err == nil {
		t.Error("Get() should return error when bucket doesn't exist")
	}

	// Restore original bucket name
	store.Config.BucketName = originalBucket
}

func TestStore_GetAll(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	// Test empty bucket
	results, err := store.GetAll()
	if err != nil {
		t.Errorf("GetAll() on empty bucket failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results from empty bucket, got %d", len(results))
	}

	// Add test data
	testData := []struct {
		key   string
		value TestStruct
	}{
		{"key1", TestStruct{ID: 1, Name: "First"}},
		{"key2", TestStruct{ID: 2, Name: "Second"}},
		{"key3", TestStruct{ID: 3, Name: "Third"}},
	}

	for _, td := range testData {
		if err := store.Set(td.key, td.value); err != nil {
			t.Fatalf("Set() failed: %v", err)
		}
	}

	// Get all items
	results, err = store.GetAll()
	if err != nil {
		t.Fatalf("GetAll() failed: %v", err)
	}

	if len(results) != len(testData) {
		t.Errorf("Expected %d results, got %d", len(testData), len(results))
	}

	// Verify all items are present (order may vary)
	found := make(map[int]bool)
	for _, result := range results {
		found[result.ID] = true
	}

	for _, td := range testData {
		if !found[td.value.ID] {
			t.Errorf("Item with ID %d not found in results", td.value.ID)
		}
	}
}

func TestStore_GetAll_NonExistentBucket(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	// Change to non-existent bucket
	store.Config.BucketName = "nonexistent-bucket"

	results, err := store.GetAll()
	if err != nil {
		t.Errorf("GetAll() should not error on non-existent bucket: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results from non-existent bucket, got %d", len(results))
	}
}

func TestStore_Delete(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	// Set up test data
	key := "delete-key"
	value := TestStruct{ID: 99, Name: "ToDelete"}

	err := store.Set(key, value)
	if err != nil {
		t.Fatalf("Set() failed: %v", err)
	}

	// Verify it exists
	_, err = store.Get(key)
	if err != nil {
		t.Fatalf("Get() before delete failed: %v", err)
	}

	// Delete the item
	err = store.Delete(key)
	if err != nil {
		t.Errorf("Delete() failed: %v", err)
	}

	// Verify it's gone
	_, err = store.Get(key)
	if err == nil {
		t.Error("Get() after Delete() should return error")
	}
}

func TestStore_Delete_NonExistentKey(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	// Set up bucket first
	store.Set("dummy", TestStruct{ID: 1, Name: "Dummy"})

	// Delete non-existent key should not error (BoltDB behavior)
	err := store.Delete("nonexistent-key")
	if err != nil {
		t.Errorf("Delete() on non-existent key returned error: %v", err)
	}
}

func TestStore_Delete_NonExistentBucket(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	// Change to non-existent bucket
	store.Config.BucketName = "nonexistent-bucket"

	err := store.Delete("any-key")
	if err == nil {
		t.Error("Delete() should return error when bucket doesn't exist")
	}
}

func TestStore_Close(t *testing.T) {
	store, dbPath, _ := setupTestStore(t)

	// Close the store
	err := store.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	// Try to use the store after closing (should fail)
	err = store.Set("key", TestStruct{ID: 1, Name: "Test"})
	if err == nil {
		t.Error("Set() should fail after Close()")
	}

	// Clean up
	os.Remove(dbPath)
}

func TestStore_ConcurrentOperations(t *testing.T) {
	store, _, cleanup := setupTestStore(t)
	defer cleanup()

	// Test concurrent writes
	done := make(chan bool)
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			key := filepath.Join("concurrent", string(rune(id)))
			value := TestStruct{ID: id, Name: filepath.Join("Concurrent", string(rune(id)))}
			if err := store.Set(key, value); err != nil {
				errors <- err
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent operation failed: %v", err)
	}
}

func TestStore_WithDifferentTypes(t *testing.T) {
	// Test with string type
	t.Run("string type", func(t *testing.T) {
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "string_test.db")

		config := NewConfig().
			WithFileName(dbPath).
			WithBucketName("stringbucket")

		db, err := bolt.Open(dbPath, 0600, nil)
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}
		defer db.Close()

		store := &Store[string]{
			db:     db,
			Config: config,
		}

		err = store.Set("key1", "value1")
		if err != nil {
			t.Errorf("Set() failed for string: %v", err)
		}

		value, err := store.Get("key1")
		if err != nil {
			t.Errorf("Get() failed for string: %v", err)
		}

		if value != "value1" {
			t.Errorf("Expected 'value1', got %q", value)
		}
	})

	// Test with int type
	t.Run("int type", func(t *testing.T) {
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "int_test.db")

		config := NewConfig().
			WithFileName(dbPath).
			WithBucketName("intbucket")

		db, err := bolt.Open(dbPath, 0600, nil)
		if err != nil {
			t.Fatalf("Failed to open database: %v", err)
		}
		defer db.Close()

		store := &Store[int]{
			db:     db,
			Config: config,
		}

		err = store.Set("key1", 42)
		if err != nil {
			t.Errorf("Set() failed for int: %v", err)
		}

		value, err := store.Get("key1")
		if err != nil {
			t.Errorf("Get() failed for int: %v", err)
		}

		if value != 42 {
			t.Errorf("Expected 42, got %d", value)
		}
	})
}
