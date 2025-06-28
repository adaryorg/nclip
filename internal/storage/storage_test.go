/*
MIT License

Copyright (c) 2025 Yuval Adar <adary@adary.org>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

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

func TestComprehensiveDeduplication(t *testing.T) {
	storage, _ := createTestStorage(t)

	// Add initial content
	err := storage.Add("test content")
	if err != nil {
		t.Fatalf("Failed to add initial content: %v", err)
	}

	// Add different content
	err = storage.Add("different content")
	if err != nil {
		t.Fatalf("Failed to add different content: %v", err)
	}

	// Add another different content
	err = storage.Add("another content")
	if err != nil {
		t.Fatalf("Failed to add another content: %v", err)
	}

	// Now add duplicate of the first content (not the most recent)
	originalItems := storage.GetAll()
	if len(originalItems) != 3 {
		t.Fatalf("Expected 3 items before duplicate, got %d", len(originalItems))
	}

	// Find the original "test content" item
	var originalItem *ClipboardItem
	for _, item := range originalItems {
		if item.Content == "test content" {
			originalItem = &item
			break
		}
	}
	if originalItem == nil {
		t.Fatal("Could not find original 'test content' item")
	}

	// Add the duplicate
	err = storage.Add("test content")
	if err != nil {
		t.Fatalf("Failed to add duplicate content: %v", err)
	}

	// Should still have 3 items (no new item created)
	items := storage.GetAll()
	if len(items) != 3 {
		t.Errorf("Expected 3 items after duplicate add, got %d", len(items))
	}

	// The "test content" should now be the most recent (first in the list)
	if items[0].Content != "test content" {
		t.Errorf("Expected 'test content' to be most recent, got '%s'", items[0].Content)
	}

	// The ID should be the same as the original (item was updated, not recreated)
	if items[0].ID != originalItem.ID {
		t.Errorf("Expected same ID for updated item, got %s instead of %s", items[0].ID, originalItem.ID)
	}

	// The timestamp should be newer than the original
	if !items[0].Timestamp.After(originalItem.Timestamp) {
		t.Error("Expected updated timestamp to be newer than original")
	}
}

func TestImageDeduplication(t *testing.T) {
	storage, _ := createTestStorage(t)

	imageData1 := []byte("fake image data 1")
	imageSameAsFirst := []byte("fake image data 1") // Same as imageData1

	// Add first image
	err := storage.AddImage(imageData1, "image 1")
	if err != nil {
		t.Fatalf("Failed to add first image: %v", err)
	}

	items := storage.GetAll()
	if len(items) != 1 {
		t.Fatalf("Expected 1 item after adding first image, got %d", len(items))
	}

	// Record the original timestamp
	originalTimestamp := items[0].Timestamp
	originalID := items[0].ID

	// Add some delay to ensure different timestamp
	time.Sleep(2 * time.Millisecond)

	// Now add image with same data and description - should be deduplicated
	err = storage.AddImage(imageSameAsFirst, "image 1")
	if err != nil {
		t.Fatalf("Failed to add duplicate image: %v", err)
	}

	// Should still have 1 item (duplicate was not added)
	items = storage.GetAll()
	if len(items) != 1 {
		t.Errorf("Expected 1 item after duplicate image add, got %d", len(items))
	}

	// Verify the item is the same but with updated timestamp
	if items[0].ID != originalID {
		t.Errorf("Expected same ID for updated item, got %s instead of %s", items[0].ID, originalID)
	}

	if !items[0].Timestamp.After(originalTimestamp) {
		t.Error("Expected updated image timestamp to be newer")
	}
}

func TestMixedContentDeduplication(t *testing.T) {
	storage, _ := createTestStorage(t)

	// Add text content
	err := storage.Add("mixed content")
	if err != nil {
		t.Fatalf("Failed to add text content: %v", err)
	}

	// Add image with same description - should not be deduplicated (different types)
	imageData := []byte("image data")
	err = storage.AddImage(imageData, "mixed content")
	if err != nil {
		t.Fatalf("Failed to add image content: %v", err)
	}

	items := storage.GetAll()
	if len(items) != 2 {
		t.Errorf("Expected 2 items for mixed content types, got %d", len(items))
	}

	// Verify we have both text and image entries
	hasText := false
	hasImage := false
	for _, item := range items {
		if item.Content == "mixed content" {
			if item.ContentType == "text" {
				hasText = true
			} else if item.ContentType == "image" {
				hasImage = true
			}
		}
	}
	if !hasText {
		t.Error("Expected to find text content type")
	}
	if !hasImage {
		t.Error("Expected to find image content type")
	}
}

func TestDeduplicationWithTimestampUpdate(t *testing.T) {
	storage, _ := createTestStorage(t)

	// Add content and record timestamp
	err := storage.Add("timestamp test")
	if err != nil {
		t.Fatalf("Failed to add content: %v", err)
	}

	items := storage.GetAll()
	if len(items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(items))
	}
	originalTimestamp := items[0].Timestamp

	// Wait a small amount to ensure timestamp difference
	time.Sleep(2 * time.Millisecond)

	// Add duplicate content
	err = storage.Add("timestamp test")
	if err != nil {
		t.Fatalf("Failed to add duplicate content: %v", err)
	}

	// Should still have 1 item with updated timestamp
	items = storage.GetAll()
	if len(items) != 1 {
		t.Errorf("Expected 1 item after duplicate, got %d", len(items))
	}

	if !items[0].Timestamp.After(originalTimestamp) {
		t.Error("Expected timestamp to be updated for duplicate entry")
	}
}

func TestDeduplicateExisting(t *testing.T) {
	storage, _ := createTestStorage(t)

	// Manually insert duplicates by bypassing the new deduplication logic
	id1 := fmt.Sprintf("%d", time.Now().UnixNano())
	id2 := fmt.Sprintf("%d", time.Now().UnixNano()+1)
	id3 := fmt.Sprintf("%d", time.Now().UnixNano()+2)
	id4 := fmt.Sprintf("%d", time.Now().UnixNano()+3)

	baseTime := time.Now()

	// Insert duplicate text entries using direct insert to simulate old behavior
	err := storage.insertDirectly(id1, "duplicate text", "text", nil, baseTime.Add(1*time.Second), "none", true)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	err = storage.insertDirectly(id2, "unique text", "text", nil, baseTime.Add(2*time.Second), "none", true)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	err = storage.insertDirectly(id3, "duplicate text", "text", nil, baseTime.Add(3*time.Second), "none", true)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Insert duplicate image entries
	imageData := []byte("test image data")
	err = storage.insertDirectly(id4, "test image", "image", imageData, baseTime.Add(4*time.Second), "none", true)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Verify we have 4 items initially
	items := storage.GetAll()
	if len(items) != 4 {
		t.Fatalf("Expected 4 items before deduplication, got %d", len(items))
	}

	// Run deduplication
	removedCount, err := storage.DeduplicateExisting()
	if err != nil {
		t.Fatalf("Failed to deduplicate: %v", err)
	}

	if removedCount != 1 {
		t.Errorf("Expected to remove 1 duplicate, removed %d", removedCount)
	}

	// Verify we now have 3 unique items
	items = storage.GetAll()
	if len(items) != 3 {
		t.Errorf("Expected 3 items after deduplication, got %d", len(items))
	}

	// Verify the most recent "duplicate text" entry was kept
	foundDuplicateText := false
	for _, item := range items {
		if item.Content == "duplicate text" {
			foundDuplicateText = true
			if item.ID != id3 { // id3 is more recent (baseTime + 3s)
				t.Errorf("Expected to keep the most recent duplicate (ID: %s), but found ID: %s", id3, item.ID)
			}
		}
	}
	if !foundDuplicateText {
		t.Error("Expected to find 'duplicate text' entry after deduplication")
	}
}

func TestDeduplicateExistingImages(t *testing.T) {
	storage, _ := createTestStorage(t)

	// Manually insert duplicate image entries
	imageData1 := []byte("image data 1")
	imageData2 := []byte("image data 2")

	id1 := fmt.Sprintf("%d", time.Now().UnixNano())
	id2 := fmt.Sprintf("%d", time.Now().UnixNano()+1)
	id3 := fmt.Sprintf("%d", time.Now().UnixNano()+2)
	id4 := fmt.Sprintf("%d", time.Now().UnixNano()+3)

	baseTime := time.Now()

	// Insert same description but different image data - should not be deduplicated
	err := storage.insertDirectly(id1, "image", "image", imageData1, baseTime.Add(1*time.Second), "none", true)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	err = storage.insertDirectly(id2, "image", "image", imageData2, baseTime.Add(2*time.Second), "none", true)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Insert same description AND same image data - should be deduplicated
	err = storage.insertDirectly(id3, "image", "image", imageData1, baseTime.Add(3*time.Second), "none", true)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Insert different content type but same content - should not be deduplicated
	err = storage.insertDirectly(id4, "image", "text", nil, baseTime.Add(4*time.Second), "none", true)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Verify we have 4 items initially
	items := storage.GetAll()
	if len(items) != 4 {
		t.Fatalf("Expected 4 items before deduplication, got %d", len(items))
	}

	// Run deduplication
	removedCount, err := storage.DeduplicateExisting()
	if err != nil {
		t.Fatalf("Failed to deduplicate: %v", err)
	}

	if removedCount != 1 {
		t.Errorf("Expected to remove 1 duplicate, removed %d", removedCount)
	}

	// Verify we now have 3 items
	items = storage.GetAll()
	if len(items) != 3 {
		t.Errorf("Expected 3 items after deduplication, got %d", len(items))
	}

	// Count items by type to verify correct deduplication
	imageCount := 0
	textCount := 0
	for _, item := range items {
		if item.ContentType == "image" {
			imageCount++
		} else if item.ContentType == "text" {
			textCount++
		}
	}

	if imageCount != 2 {
		t.Errorf("Expected 2 image items after deduplication, got %d", imageCount)
	}
	if textCount != 1 {
		t.Errorf("Expected 1 text item after deduplication, got %d", textCount)
	}
}

func TestDeduplicateExistingEmptyDatabase(t *testing.T) {
	storage, _ := createTestStorage(t)

	// Run deduplication on empty database
	removedCount, err := storage.DeduplicateExisting()
	if err != nil {
		t.Fatalf("Failed to deduplicate empty database: %v", err)
	}

	if removedCount != 0 {
		t.Errorf("Expected to remove 0 items from empty database, removed %d", removedCount)
	}
}

func TestDeduplicateExistingNoDuplicates(t *testing.T) {
	storage, _ := createTestStorage(t)

	// Add unique items using the normal Add method
	err := storage.Add("unique text 1")
	if err != nil {
		t.Fatalf("Failed to add content: %v", err)
	}

	err = storage.Add("unique text 2")
	if err != nil {
		t.Fatalf("Failed to add content: %v", err)
	}

	imageData := []byte("unique image data")
	err = storage.AddImage(imageData, "unique image")
	if err != nil {
		t.Fatalf("Failed to add image: %v", err)
	}

	// Run deduplication
	removedCount, err := storage.DeduplicateExisting()
	if err != nil {
		t.Fatalf("Failed to deduplicate: %v", err)
	}

	if removedCount != 0 {
		t.Errorf("Expected to remove 0 items (no duplicates), removed %d", removedCount)
	}

	// Verify all items are still there
	items := storage.GetAll()
	if len(items) != 3 {
		t.Errorf("Expected 3 items after deduplication, got %d", len(items))
	}
}
