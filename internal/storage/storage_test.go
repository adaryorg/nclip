package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func createTestStorage(t *testing.T) (*Storage, string) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "nclip-storage-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Temporarily override home directory
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)

	// Cleanup function will restore original HOME
	t.Cleanup(func() {
		os.Setenv("HOME", originalHome)
		os.RemoveAll(tmpDir)
	})

	storage, err := New(10) // Small max entries for testing
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	t.Cleanup(func() {
		storage.Close()
	})

	return storage, tmpDir
}

func TestNew(t *testing.T) {
	storage, _ := createTestStorage(t)

	if storage == nil {
		t.Fatal("Expected storage to be created")
	}

	if storage.maxEntries != 10 {
		t.Errorf("Expected maxEntries to be 10, got %d", storage.maxEntries)
	}
}

func TestAdd(t *testing.T) {
	storage, _ := createTestStorage(t)

	err := storage.Add("test content")
	if err != nil {
		t.Fatalf("Failed to add content: %v", err)
	}

	items := storage.GetAll()
	if len(items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(items))
	}

	item := items[0]
	if item.Content != "test content" {
		t.Errorf("Expected content 'test content', got '%s'", item.Content)
	}
	if item.ContentType != "text" {
		t.Errorf("Expected content type 'text', got '%s'", item.ContentType)
	}
	if item.ThreatLevel != "none" {
		t.Errorf("Expected threat level 'none', got '%s'", item.ThreatLevel)
	}
	if !item.SafeEntry {
		t.Error("Expected safe entry to be true")
	}
}

func TestAddDuplicate(t *testing.T) {
	storage, _ := createTestStorage(t)

	// Add same content twice
	err := storage.Add("duplicate content")
	if err != nil {
		t.Fatalf("Failed to add content: %v", err)
	}

	err = storage.Add("duplicate content")
	if err != nil {
		t.Fatalf("Failed to add duplicate content: %v", err)
	}

	// Should only have one item (duplicate prevention)
	items := storage.GetAll()
	if len(items) != 1 {
		t.Errorf("Expected 1 item after duplicate add, got %d", len(items))
	}
}

func TestAddEmpty(t *testing.T) {
	storage, _ := createTestStorage(t)

	err := storage.Add("")
	if err != nil {
		t.Fatalf("Add empty content should not error: %v", err)
	}

	// Should not add empty content
	items := storage.GetAll()
	if len(items) != 0 {
		t.Errorf("Expected 0 items after adding empty content, got %d", len(items))
	}
}

func TestAddImage(t *testing.T) {
	storage, _ := createTestStorage(t)

	imageData := []byte("fake image data")
	err := storage.AddImage(imageData, "test image")
	if err != nil {
		t.Fatalf("Failed to add image: %v", err)
	}

	items := storage.GetAll()
	if len(items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(items))
	}

	item := items[0]
	if item.Content != "test image" {
		t.Errorf("Expected content 'test image', got '%s'", item.Content)
	}
	if item.ContentType != "image" {
		t.Errorf("Expected content type 'image', got '%s'", item.ContentType)
	}
	if string(item.ImageData) != "fake image data" {
		t.Errorf("Expected image data 'fake image data', got '%s'", string(item.ImageData))
	}
	if item.ThreatLevel != "none" {
		t.Errorf("Expected threat level 'none' for image, got '%s'", item.ThreatLevel)
	}
}

func TestMaxEntries(t *testing.T) {
	storage, _ := createTestStorage(t)

	// Add more items than max entries (10)
	for i := 0; i < 15; i++ {
		err := storage.Add(fmt.Sprintf("content %d", i))
		if err != nil {
			t.Fatalf("Failed to add content %d: %v", i, err)
		}
	}

	items := storage.GetAll()
	if len(items) != 10 {
		t.Errorf("Expected max 10 items, got %d", len(items))
	}

	// Should have the most recent items (content 5-14)
	expectedContent := "content 14"
	if items[0].Content != expectedContent {
		t.Errorf("Expected most recent content '%s', got '%s'", expectedContent, items[0].Content)
	}
}

func TestGetByID(t *testing.T) {
	storage, _ := createTestStorage(t)

	err := storage.Add("test content")
	if err != nil {
		t.Fatalf("Failed to add content: %v", err)
	}

	items := storage.GetAll()
	if len(items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(items))
	}

	// Get by ID
	item := storage.GetByID(items[0].ID)
	if item == nil {
		t.Fatal("Expected item to be found")
	}

	if item.Content != "test content" {
		t.Errorf("Expected content 'test content', got '%s'", item.Content)
	}

	// Test non-existent ID
	nonExistent := storage.GetByID("non-existent-id")
	if nonExistent != nil {
		t.Error("Expected nil for non-existent ID")
	}
}

func TestUpdate(t *testing.T) {
	storage, _ := createTestStorage(t)

	err := storage.Add("original content")
	if err != nil {
		t.Fatalf("Failed to add content: %v", err)
	}

	items := storage.GetAll()
	if len(items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(items))
	}

	// Update content
	err = storage.Update(items[0].ID, "updated content")
	if err != nil {
		t.Fatalf("Failed to update content: %v", err)
	}

	// Verify update
	updatedItem := storage.GetByID(items[0].ID)
	if updatedItem == nil {
		t.Fatal("Expected item to still exist after update")
	}

	if updatedItem.Content != "updated content" {
		t.Errorf("Expected updated content 'updated content', got '%s'", updatedItem.Content)
	}
}

func TestUpdateSafeEntry(t *testing.T) {
	storage, _ := createTestStorage(t)

	err := storage.Add("test content")
	if err != nil {
		t.Fatalf("Failed to add content: %v", err)
	}

	items := storage.GetAll()
	if len(items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(items))
	}

	// Update safe entry flag
	err = storage.UpdateSafeEntry(items[0].ID, false)
	if err != nil {
		t.Fatalf("Failed to update safe entry: %v", err)
	}

	// Verify update
	updatedItem := storage.GetByID(items[0].ID)
	if updatedItem == nil {
		t.Fatal("Expected item to still exist after update")
	}

	if updatedItem.SafeEntry {
		t.Error("Expected safe entry to be false")
	}
}

func TestDelete(t *testing.T) {
	storage, _ := createTestStorage(t)

	err := storage.Add("test content")
	if err != nil {
		t.Fatalf("Failed to add content: %v", err)
	}

	items := storage.GetAll()
	if len(items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(items))
	}

	// Delete item
	err = storage.Delete(items[0].ID)
	if err != nil {
		t.Fatalf("Failed to delete item: %v", err)
	}

	// Verify deletion
	itemsAfterDelete := storage.GetAll()
	if len(itemsAfterDelete) != 0 {
		t.Errorf("Expected 0 items after delete, got %d", len(itemsAfterDelete))
	}

	// Verify GetByID returns nil
	deletedItem := storage.GetByID(items[0].ID)
	if deletedItem != nil {
		t.Error("Expected deleted item to return nil")
	}
}

func TestCalculateThreatLevel(t *testing.T) {
	tests := []struct {
		content     string
		contentType string
		expected    string
		safeness    bool
	}{
		{"normal text", "text", "none", true},
		{"", "image", "none", true},
		{"password123", "text", "none", true}, // Not a real threat pattern
	}

	for _, test := range tests {
		level, safe := calculateThreatLevel(test.content, test.contentType)
		if level != test.expected {
			t.Errorf("For content '%s' type '%s', expected threat level '%s', got '%s'",
				test.content, test.contentType, test.expected, level)
		}
		if safe != test.safeness {
			t.Errorf("For content '%s' type '%s', expected safeness %v, got %v",
				test.content, test.contentType, test.safeness, safe)
		}
	}
}

func TestGetAllOrdering(t *testing.T) {
	storage, _ := createTestStorage(t)

	// Add items with small delays to ensure different timestamps
	contents := []string{"first", "second", "third"}
	for _, content := range contents {
		err := storage.Add(content)
		if err != nil {
			t.Fatalf("Failed to add content '%s': %v", content, err)
		}
		time.Sleep(time.Millisecond) // Ensure different timestamps
	}

	items := storage.GetAll()
	if len(items) != 3 {
		t.Fatalf("Expected 3 items, got %d", len(items))
	}

	// Should be ordered by timestamp DESC (most recent first)
	expected := []string{"third", "second", "first"}
	for i, expectedContent := range expected {
		if items[i].Content != expectedContent {
			t.Errorf("Expected item %d to be '%s', got '%s'", i, expectedContent, items[i].Content)
		}
	}
}

func TestDatabaseCreation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nclip-storage-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Override home directory
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	storage, err := New(10)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	// Check if database file was created
	dbPath := filepath.Join(tmpDir, ".config", "nclip", "history.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}
}

func TestInvalidHomeDirectory(t *testing.T) {
	// Save original HOME
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	// Unset HOME to trigger error
	os.Unsetenv("HOME")

	_, err := New(10)
	if err == nil {
		// On some systems this might not fail, so we skip the test
		t.Skip("UserHomeDir didn't fail on this system")
	}
	if !strings.Contains(err.Error(), "failed to get user home directory") {
		t.Errorf("Expected home directory error, got: %v", err)
	}
}

func TestClose(t *testing.T) {
	storage, _ := createTestStorage(t)

	err := storage.Close()
	if err != nil {
		t.Errorf("Close should not error: %v", err)
	}

	// Multiple closes should be safe
	err = storage.Close()
	if err != nil {
		t.Errorf("Multiple closes should not error: %v", err)
	}
}
