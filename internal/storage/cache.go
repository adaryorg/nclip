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
	"container/list"
	"sync"
	"time"
)

// ItemCache provides memory-efficient caching for clipboard items with LRU eviction
type ItemCache struct {
	storage     *Storage
	
	// Metadata cache - always loaded for all items
	metaItems   []ClipboardItemMeta
	totalCount  int
	lastRefresh time.Time
	
	// Image data cache with LRU eviction
	imageCache     map[string][]byte      // id -> image data
	imageCacheList *list.List             // LRU list for image cache
	imageCacheMap  map[string]*list.Element // id -> list element for O(1) access
	maxImageCache  int                    // Maximum number of images to cache
	
	mu sync.RWMutex
}

type imageCacheEntry struct {
	id   string
	data []byte
}

// NewItemCache creates a new ItemCache
func NewItemCache(storage *Storage, maxImageCache int) *ItemCache {
	if maxImageCache <= 0 {
		maxImageCache = 10 // Default cache size
	}
	
	cache := &ItemCache{
		storage:        storage,
		imageCache:     make(map[string][]byte),
		imageCacheList: list.New(),
		imageCacheMap:  make(map[string]*list.Element),
		maxImageCache:  maxImageCache,
	}
	
	// Initial load of metadata
	cache.refreshMetadata()
	
	return cache
}

// refreshMetadata reloads all item metadata from storage
func (c *ItemCache) refreshMetadata() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.metaItems = c.storage.GetAllMeta()
	c.totalCount = len(c.metaItems)
	c.lastRefresh = time.Now()
}

// ForceRefresh forces an immediate refresh of the cache
func (c *ItemCache) ForceRefresh() {
	c.refreshMetadata()
}

// GetAllMeta returns all item metadata (lightweight)
func (c *ItemCache) GetAllMeta() []ClipboardItemMeta {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	// Refresh if data is older than 5 seconds (to catch new clipboard entries)
	if time.Since(c.lastRefresh) > 5*time.Second {
		c.mu.RUnlock()
		c.refreshMetadata()
		c.mu.RLock()
	}
	
	// Return a copy to prevent external modification
	result := make([]ClipboardItemMeta, len(c.metaItems))
	copy(result, c.metaItems)
	return result
}

// GetItemCount returns the total number of items
func (c *ItemCache) GetItemCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.totalCount
}

// GetImageData returns image data for an item, using cache when possible
func (c *ItemCache) GetImageData(id string) []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Check cache first
	if data, exists := c.imageCache[id]; exists {
		// Move to front of LRU list
		if elem, exists := c.imageCacheMap[id]; exists {
			c.imageCacheList.MoveToFront(elem)
		}
		return data
	}
	
	// Load from storage
	data := c.storage.GetImageData(id)
	if data != nil {
		c.putImageInCache(id, data)
	}
	
	return data
}

// putImageInCache adds image data to cache with LRU eviction
func (c *ItemCache) putImageInCache(id string, data []byte) {
	// Check if already in cache
	if elem, exists := c.imageCacheMap[id]; exists {
		// Update existing entry and move to front
		entry := elem.Value.(*imageCacheEntry)
		entry.data = data
		c.imageCacheList.MoveToFront(elem)
		c.imageCache[id] = data
		return
	}
	
	// Add new entry
	entry := &imageCacheEntry{id: id, data: data}
	elem := c.imageCacheList.PushFront(entry)
	c.imageCacheMap[id] = elem
	c.imageCache[id] = data
	
	// Evict if over capacity
	for c.imageCacheList.Len() > c.maxImageCache {
		c.evictOldestImage()
	}
}

// evictOldestImage removes the least recently used image from cache
func (c *ItemCache) evictOldestImage() {
	if c.imageCacheList.Len() == 0 {
		return
	}
	
	oldest := c.imageCacheList.Back()
	if oldest != nil {
		entry := oldest.Value.(*imageCacheEntry)
		c.imageCacheList.Remove(oldest)
		delete(c.imageCacheMap, entry.id)
		delete(c.imageCache, entry.id)
	}
}

// GetFullItem returns a complete ClipboardItem, loading image data if needed
func (c *ItemCache) GetFullItem(id string) *ClipboardItem {
	c.mu.RLock()
	
	// Find metadata
	var meta *ClipboardItemMeta
	for i := range c.metaItems {
		if c.metaItems[i].ID == id {
			meta = &c.metaItems[i]
			break
		}
	}
	c.mu.RUnlock()
	
	if meta == nil {
		return nil
	}
	
	// Convert to ClipboardItem
	item := meta.ToClipboardItem()
	
	// Load image data if it's an image
	if meta.ContentType == "image" {
		item.ImageData = c.GetImageData(id)
	}
	
	return &item
}

// PreloadImageData preloads image data for specific items
func (c *ItemCache) PreloadImageData(ids []string) {
	for _, id := range ids {
		// Check if already cached
		c.mu.RLock()
		_, exists := c.imageCache[id]
		c.mu.RUnlock()
		
		if !exists {
			// Trigger loading (this will cache it)
			c.GetImageData(id)
		}
	}
}

// EvictImageData removes specific image data from cache
func (c *ItemCache) EvictImageData(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if elem, exists := c.imageCacheMap[id]; exists {
		c.imageCacheList.Remove(elem)
		delete(c.imageCacheMap, id)
		delete(c.imageCache, id)
	}
}

// GetCacheStats returns statistics about the cache
func (c *ItemCache) GetCacheStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	return map[string]interface{}{
		"total_items":        c.totalCount,
		"cached_images":      len(c.imageCache),
		"max_image_cache":    c.maxImageCache,
		"last_refresh":       c.lastRefresh,
		"cache_hit_ratio":    float64(len(c.imageCache)) / float64(c.maxImageCache),
	}
}