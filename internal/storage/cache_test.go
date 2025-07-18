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
	"testing"
	"time"
)

func createTestStorageForCache(t *testing.T) (*Storage, string) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "nclip-cache-test")
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

func TestNewItemCache(t *testing.T) {
	storage, _ := createTestStorageForCache(t)

	// Test with default cache size
	cache := NewItemCache(storage, 0)
	if cache == nil {
		t.Fatal("Expected cache to be created")
	}

	stats := cache.GetCacheStats()
	if maxCache, ok := stats["max_image_cache"].(int); !ok || maxCache != 10 {
		t.Errorf("Expected default max_image_cache to be 10, got %v", maxCache)
	}

	// Test with custom cache size
	cache = NewItemCache(storage, 5)
	stats = cache.GetCacheStats()
	if maxCache, ok := stats["max_image_cache"].(int); !ok || maxCache != 5 {
		t.Errorf("Expected max_image_cache to be 5, got %v", maxCache)
	}
}

func TestItemCache_GetAllMeta(t *testing.T) {
	storage, _ := createTestStorageForCache(t)
	cache := NewItemCache(storage, 5)

	// Add some test data
	err := storage.Add("text content 1")
	if err != nil {
		t.Fatalf("Failed to add content: %v", err)
	}

	imageData := []byte("test image data")
	err = storage.AddImage(imageData, "test image")
	if err != nil {
		t.Fatalf("Failed to add image: %v", err)
	}

	// Refresh cache to pick up new items
	cache.refreshMetadata()

	// Get all metadata
	items := cache.GetAllMeta()
	if len(items) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(items))
	}

	// Verify order (most recent first)
	if items[0].Content != "test image" {
		t.Errorf("Expected first item to be 'test image', got '%s'", items[0].Content)
	}
	if items[0].ContentType != "image" {
		t.Errorf("Expected first item type to be 'image', got '%s'", items[0].ContentType)
	}

	if items[1].Content != "text content 1" {
		t.Errorf("Expected second item to be 'text content 1', got '%s'", items[1].Content)
	}
	if items[1].ContentType != "text" {
		t.Errorf("Expected second item type to be 'text', got '%s'", items[1].ContentType)
	}
}

func TestItemCache_GetItemCount(t *testing.T) {
	storage, _ := createTestStorageForCache(t)
	cache := NewItemCache(storage, 5)

	// Initially empty
	count := cache.GetItemCount()
	if count != 0 {
		t.Errorf("Expected count to be 0, got %d", count)
	}

	// Add some items
	for i := 0; i < 3; i++ {
		err := storage.Add(fmt.Sprintf("content %d", i))
		if err != nil {
			t.Fatalf("Failed to add content %d: %v", i, err)
		}
	}

	// Refresh cache and check count
	cache.refreshMetadata()
	count = cache.GetItemCount()
	if count != 3 {
		t.Errorf("Expected count to be 3, got %d", count)
	}
}

func TestItemCache_GetImageData(t *testing.T) {
	storage, _ := createTestStorageForCache(t)
	cache := NewItemCache(storage, 3)

	// Add test image
	imageData := []byte("test image data")
	err := storage.AddImage(imageData, "test image")
	if err != nil {
		t.Fatalf("Failed to add image: %v", err)
	}

	// Refresh cache to pick up new items
	cache.refreshMetadata()

	// Get the image ID
	items := cache.GetAllMeta()
	if len(items) == 0 {
		t.Fatal("No items found")
	}
	imageID := items[0].ID

	// Test GetImageData - first time (loads from storage)
	retrievedData := cache.GetImageData(imageID)
	if retrievedData == nil {
		t.Fatal("Expected image data, got nil")
	}
	if string(retrievedData) != "test image data" {
		t.Errorf("Expected 'test image data', got '%s'", string(retrievedData))
	}

	// Verify it's now cached
	stats := cache.GetCacheStats()
	if cachedImages, ok := stats["cached_images"].(int); !ok || cachedImages != 1 {
		t.Errorf("Expected 1 cached image, got %v", cachedImages)
	}

	// Test GetImageData - second time (loads from cache)
	retrievedData2 := cache.GetImageData(imageID)
	if string(retrievedData2) != "test image data" {
		t.Errorf("Expected cached data to match, got '%s'", string(retrievedData2))
	}

	// Test with non-existent ID
	nonExistentData := cache.GetImageData("non-existent-id")
	if nonExistentData != nil {
		t.Error("Expected nil for non-existent ID")
	}
}

func TestItemCache_LRUEviction(t *testing.T) {
	storage, _ := createTestStorageForCache(t)
	cache := NewItemCache(storage, 2) // Small cache to test eviction

	// Add multiple images
	var imageIDs []string
	for i := 0; i < 4; i++ {
		imageData := []byte(fmt.Sprintf("image data %d", i))
		err := storage.AddImage(imageData, fmt.Sprintf("image %d", i))
		if err != nil {
			t.Fatalf("Failed to add image %d: %v", i, err)
		}
	}

	// Refresh cache to pick up new items
	cache.refreshMetadata()

	// Get all image IDs
	items := cache.GetAllMeta()
	for _, item := range items {
		if item.ContentType == "image" {
			imageIDs = append(imageIDs, item.ID)
		}
	}

	if len(imageIDs) != 4 {
		t.Fatalf("Expected 4 images, got %d", len(imageIDs))
	}

	// Load first two images into cache
	cache.GetImageData(imageIDs[0])
	cache.GetImageData(imageIDs[1])

	// Verify cache has 2 items
	stats := cache.GetCacheStats()
	if cachedImages, ok := stats["cached_images"].(int); !ok || cachedImages != 2 {
		t.Errorf("Expected 2 cached images, got %v", cachedImages)
	}

	// Load third image - should evict the oldest (first image)
	cache.GetImageData(imageIDs[2])

	// Cache should still have 2 items
	stats = cache.GetCacheStats()
	if cachedImages, ok := stats["cached_images"].(int); !ok || cachedImages != 2 {
		t.Errorf("Expected 2 cached images after eviction, got %v", cachedImages)
	}

	// Access second image again - makes it most recently used
	cache.GetImageData(imageIDs[1])

	// Load fourth image - should evict third image (since second was just accessed)
	cache.GetImageData(imageIDs[3])

	// Verify cache contents by checking which images can be retrieved quickly
	// (this is indirect testing since we can't directly inspect cache contents)
	stats = cache.GetCacheStats()
	if cachedImages, ok := stats["cached_images"].(int); !ok || cachedImages != 2 {
		t.Errorf("Expected 2 cached images after final eviction, got %v", cachedImages)
	}
}

func TestItemCache_GetFullItem(t *testing.T) {
	storage, _ := createTestStorageForCache(t)
	cache := NewItemCache(storage, 5)

	// Add text item
	err := storage.Add("test text content")
	if err != nil {
		t.Fatalf("Failed to add text: %v", err)
	}

	// Add image item
	imageData := []byte("test image data")
	err = storage.AddImage(imageData, "test image")
	if err != nil {
		t.Fatalf("Failed to add image: %v", err)
	}

	// Refresh cache to pick up new items
	cache.refreshMetadata()

	items := cache.GetAllMeta()
	if len(items) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(items))
	}

	// Test GetFullItem for image
	imageID := items[0].ID // Most recent (image)
	fullImageItem := cache.GetFullItem(imageID)
	if fullImageItem == nil {
		t.Fatal("Expected full image item, got nil")
	}
	if fullImageItem.Content != "test image" {
		t.Errorf("Expected content 'test image', got '%s'", fullImageItem.Content)
	}
	if string(fullImageItem.ImageData) != "test image data" {
		t.Errorf("Expected image data 'test image data', got '%s'", string(fullImageItem.ImageData))
	}

	// Test GetFullItem for text
	textID := items[1].ID
	fullTextItem := cache.GetFullItem(textID)
	if fullTextItem == nil {
		t.Fatal("Expected full text item, got nil")
	}
	if fullTextItem.Content != "test text content" {
		t.Errorf("Expected content 'test text content', got '%s'", fullTextItem.Content)
	}
	if fullTextItem.ImageData != nil {
		t.Error("Expected text item to have nil ImageData")
	}

	// Test with non-existent ID
	nonExistentItem := cache.GetFullItem("non-existent-id")
	if nonExistentItem != nil {
		t.Error("Expected nil for non-existent ID")
	}
}

func TestItemCache_PreloadImageData(t *testing.T) {
	storage, _ := createTestStorageForCache(t)
	cache := NewItemCache(storage, 5)

	// Add multiple images
	var imageIDs []string
	for i := 0; i < 3; i++ {
		imageData := []byte(fmt.Sprintf("image data %d", i))
		err := storage.AddImage(imageData, fmt.Sprintf("image %d", i))
		if err != nil {
			t.Fatalf("Failed to add image %d: %v", i, err)
		}
	}

	// Refresh cache to pick up new items
	cache.refreshMetadata()

	// Get image IDs
	items := cache.GetAllMeta()
	for _, item := range items {
		if item.ContentType == "image" {
			imageIDs = append(imageIDs, item.ID)
		}
	}

	// Initially no images cached
	stats := cache.GetCacheStats()
	if cachedImages, ok := stats["cached_images"].(int); !ok || cachedImages != 0 {
		t.Errorf("Expected 0 cached images initially, got %v", cachedImages)
	}

	// Preload 2 images
	cache.PreloadImageData(imageIDs[:2])

	// Should have 2 images cached
	stats = cache.GetCacheStats()
	if cachedImages, ok := stats["cached_images"].(int); !ok || cachedImages != 2 {
		t.Errorf("Expected 2 cached images after preload, got %v", cachedImages)
	}

	// Preload already cached images (should not increase count)
	cache.PreloadImageData(imageIDs[:1])
	stats = cache.GetCacheStats()
	if cachedImages, ok := stats["cached_images"].(int); !ok || cachedImages != 2 {
		t.Errorf("Expected 2 cached images after preloading existing, got %v", cachedImages)
	}

	// Test preloading empty slice
	cache.PreloadImageData([]string{})
	stats = cache.GetCacheStats()
	if cachedImages, ok := stats["cached_images"].(int); !ok || cachedImages != 2 {
		t.Errorf("Expected 2 cached images after empty preload, got %v", cachedImages)
	}
}

func TestItemCache_EvictImageData(t *testing.T) {
	storage, _ := createTestStorageForCache(t)
	cache := NewItemCache(storage, 5)

	// Add test image
	imageData := []byte("test image data")
	err := storage.AddImage(imageData, "test image")
	if err != nil {
		t.Fatalf("Failed to add image: %v", err)
	}

	// Refresh cache to pick up new items
	cache.refreshMetadata()

	// Get image ID
	items := cache.GetAllMeta()
	imageID := items[0].ID

	// Load image into cache
	cache.GetImageData(imageID)

	// Verify it's cached
	stats := cache.GetCacheStats()
	if cachedImages, ok := stats["cached_images"].(int); !ok || cachedImages != 1 {
		t.Errorf("Expected 1 cached image, got %v", cachedImages)
	}

	// Evict the image
	cache.EvictImageData(imageID)

	// Verify it's no longer cached
	stats = cache.GetCacheStats()
	if cachedImages, ok := stats["cached_images"].(int); !ok || cachedImages != 0 {
		t.Errorf("Expected 0 cached images after eviction, got %v", cachedImages)
	}

	// Test evicting non-existent image (should not crash)
	cache.EvictImageData("non-existent-id")
	stats = cache.GetCacheStats()
	if cachedImages, ok := stats["cached_images"].(int); !ok || cachedImages != 0 {
		t.Errorf("Expected 0 cached images after non-existent eviction, got %v", cachedImages)
	}
}

func TestItemCache_GetCacheStats(t *testing.T) {
	storage, _ := createTestStorageForCache(t)
	cache := NewItemCache(storage, 3)

	// Add test data
	err := storage.Add("text content")
	if err != nil {
		t.Fatalf("Failed to add text: %v", err)
	}

	imageData := []byte("test image data")
	err = storage.AddImage(imageData, "test image")
	if err != nil {
		t.Fatalf("Failed to add image: %v", err)
	}

	// Refresh cache to pick up new items
	cache.refreshMetadata()

	// Get stats
	stats := cache.GetCacheStats()

	// Check required fields
	requiredFields := []string{"total_items", "cached_images", "max_image_cache", "last_refresh", "cache_hit_ratio"}
	for _, field := range requiredFields {
		if _, exists := stats[field]; !exists {
			t.Errorf("Expected stats to contain field '%s'", field)
		}
	}

	// Check specific values
	if totalItems, ok := stats["total_items"].(int); !ok || totalItems != 2 {
		t.Errorf("Expected total_items to be 2, got %v", totalItems)
	}

	if maxCache, ok := stats["max_image_cache"].(int); !ok || maxCache != 3 {
		t.Errorf("Expected max_image_cache to be 3, got %v", maxCache)
	}

	if cachedImages, ok := stats["cached_images"].(int); !ok || cachedImages != 0 {
		t.Errorf("Expected cached_images to be 0 initially, got %v", cachedImages)
	}

	// Load image and check stats again
	items := cache.GetAllMeta()
	imageID := items[0].ID
	cache.GetImageData(imageID)

	stats = cache.GetCacheStats()
	if cachedImages, ok := stats["cached_images"].(int); !ok || cachedImages != 1 {
		t.Errorf("Expected cached_images to be 1 after loading, got %v", cachedImages)
	}
}

func TestItemCache_RefreshMetadata(t *testing.T) {
	storage, _ := createTestStorageForCache(t)
	cache := NewItemCache(storage, 5)

	// Initially empty
	items := cache.GetAllMeta()
	if len(items) != 0 {
		t.Errorf("Expected 0 items initially, got %d", len(items))
	}

	// Add data directly to storage (bypassing cache)
	err := storage.Add("new content")
	if err != nil {
		t.Fatalf("Failed to add content: %v", err)
	}

	// Cache should still show old data
	items = cache.GetAllMeta()
	if len(items) != 0 {
		t.Errorf("Expected cache to still show 0 items, got %d", len(items))
	}

	// Manually refresh
	cache.refreshMetadata()

	// Now cache should show new data
	items = cache.GetAllMeta()
	if len(items) != 1 {
		t.Errorf("Expected 1 item after refresh, got %d", len(items))
	}

	if items[0].Content != "new content" {
		t.Errorf("Expected 'new content', got '%s'", items[0].Content)
	}
}

func TestItemCache_AutoRefresh(t *testing.T) {
	storage, _ := createTestStorageForCache(t)
	cache := NewItemCache(storage, 5)

	// Manually set last refresh to old time to trigger auto-refresh
	cache.lastRefresh = time.Now().Add(-10 * time.Second)

	// Add data to storage
	err := storage.Add("auto refresh test")
	if err != nil {
		t.Fatalf("Failed to add content: %v", err)
	}

	// GetAllMeta should trigger auto-refresh due to old timestamp
	items := cache.GetAllMeta()
	if len(items) != 1 {
		t.Errorf("Expected 1 item after auto-refresh, got %d", len(items))
	}

	if items[0].Content != "auto refresh test" {
		t.Errorf("Expected 'auto refresh test', got '%s'", items[0].Content)
	}

	// Verify lastRefresh was updated
	if time.Since(cache.lastRefresh) > time.Second {
		t.Error("Expected lastRefresh to be updated")
	}
}

func TestItemCache_ConcurrentAccess(t *testing.T) {
	storage, _ := createTestStorageForCache(t)
	cache := NewItemCache(storage, 10)

	// Add some test images
	var imageIDs []string
	for i := 0; i < 5; i++ {
		imageData := []byte(fmt.Sprintf("concurrent test image %d", i))
		err := storage.AddImage(imageData, fmt.Sprintf("concurrent image %d", i))
		if err != nil {
			t.Fatalf("Failed to add image %d: %v", i, err)
		}
	}

	// Refresh cache to pick up new items
	cache.refreshMetadata()

	items := cache.GetAllMeta()
	for _, item := range items {
		if item.ContentType == "image" {
			imageIDs = append(imageIDs, item.ID)
		}
	}

	// Test concurrent access
	done := make(chan bool, 10)

	// Start multiple goroutines accessing cache concurrently
	for i := 0; i < 10; i++ {
		go func(goroutineID int) {
			defer func() { done <- true }()

			// Each goroutine performs multiple cache operations
			for j := 0; j < 20; j++ {
				// Get metadata
				_ = cache.GetAllMeta()

				// Get image data
				if len(imageIDs) > 0 {
					_ = cache.GetImageData(imageIDs[j%len(imageIDs)])
				}

				// Get stats
				_ = cache.GetCacheStats()

				// Get full item
				if len(imageIDs) > 0 {
					_ = cache.GetFullItem(imageIDs[j%len(imageIDs)])
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent operations")
		}
	}

	// Verify cache is still functional
	stats := cache.GetCacheStats()
	if totalItems, ok := stats["total_items"].(int); !ok || totalItems != 5 {
		t.Errorf("Expected 5 total items after concurrent access, got %v", totalItems)
	}
}

func TestItemCache_ForceRefresh(t *testing.T) {
	storage, _ := createTestStorageForCache(t)
	cache := NewItemCache(storage, 5)

	// Add initial item
	err := storage.Add("initial content")
	if err != nil {
		t.Fatalf("Failed to add initial item: %v", err)
	}

	// Cache should not see the new item without refresh
	items := cache.GetAllMeta()
	if len(items) != 0 {
		t.Errorf("Expected 0 items in cache before refresh, got %d", len(items))
	}

	// Force refresh should pick up new item
	cache.ForceRefresh()
	items = cache.GetAllMeta()
	if len(items) != 1 {
		t.Errorf("Expected 1 item in cache after force refresh, got %d", len(items))
	}
	if items[0].Content != "initial content" {
		t.Errorf("Expected content 'initial content', got '%s'", items[0].Content)
	}

	// Add another item and force refresh again
	err = storage.Add("second content")
	if err != nil {
		t.Fatalf("Failed to add second item: %v", err)
	}

	cache.ForceRefresh()
	items = cache.GetAllMeta()
	if len(items) != 2 {
		t.Errorf("Expected 2 items in cache after second force refresh, got %d", len(items))
	}
}