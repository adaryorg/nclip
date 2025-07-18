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
	"bytes"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/adaryorg/nclip/internal/security"
	_ "github.com/mattn/go-sqlite3"
)

type ClipboardItem struct {
	ID          string    `json:"id"`
	Content     string    `json:"content"`
	ContentType string    `json:"content_type"`         // "text" or "image"
	ImageData   []byte    `json:"image_data,omitempty"` // Base64 encoded image data
	Timestamp   time.Time `json:"timestamp"`
	ThreatLevel string    `json:"threat_level"` // "none", "low", "medium", "high"
	SafeEntry   bool      `json:"safe_entry"`   // User-marked safe flag
	IsPinned    bool      `json:"is_pinned"`    // Whether item is pinned
	PinOrder    int       `json:"pin_order"`    // Order among pinned items (1-10)
}

// ClipboardItemMeta is a lightweight version of ClipboardItem without image data
// Used for memory-efficient listing when image data is not needed
type ClipboardItemMeta struct {
	ID          string    `json:"id"`
	Content     string    `json:"content"`
	ContentType string    `json:"content_type"`
	Timestamp   time.Time `json:"timestamp"`
	ThreatLevel string    `json:"threat_level"`
	SafeEntry   bool      `json:"safe_entry"`
	IsPinned    bool      `json:"is_pinned"`
	PinOrder    int       `json:"pin_order"`
}

type Storage struct {
	db         *sql.DB
	maxEntries int
}

func New(maxEntries int) (*Storage, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "nclip")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	dbPath := filepath.Join(configDir, "history.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	s := &Storage{
		db:         db,
		maxEntries: maxEntries,
	}

	if err := s.createTable(); err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return s, nil
}

func (s *Storage) createTable() error {
	// First create the table with original schema if it doesn't exist
	originalQuery := `
		CREATE TABLE IF NOT EXISTS clipboard_items (
			id TEXT PRIMARY KEY,
			content TEXT NOT NULL,
			timestamp DATETIME NOT NULL
		)
	`
	_, err := s.db.Exec(originalQuery)
	if err != nil {
		return err
	}

	// Then add new columns if they don't exist (for migration)
	s.db.Exec("ALTER TABLE clipboard_items ADD COLUMN content_type TEXT DEFAULT 'text'")
	s.db.Exec("ALTER TABLE clipboard_items ADD COLUMN image_data BLOB")
	s.db.Exec("ALTER TABLE clipboard_items ADD COLUMN threat_level TEXT DEFAULT 'none'")
	s.db.Exec("ALTER TABLE clipboard_items ADD COLUMN safe_entry BOOLEAN DEFAULT TRUE")
	s.db.Exec("ALTER TABLE clipboard_items ADD COLUMN is_pinned BOOLEAN DEFAULT FALSE")
	s.db.Exec("ALTER TABLE clipboard_items ADD COLUMN pin_order INTEGER DEFAULT 0")

	return nil
}

// normalizeContentForDeduplication normalizes content for deduplication comparison
// This helps identify duplicates that differ only in whitespace
func normalizeContentForDeduplication(content string) string {
	return strings.TrimSpace(content)
}

// calculateThreatLevel determines threat level based on security analysis
func calculateThreatLevel(content string, contentType string) (string, bool) {
	if contentType == "image" {
		return "none", true // Images don't have security threats
	}

	detector := security.NewSecurityDetector()
	threats := detector.DetectSecurity(content)

	if len(threats) == 0 {
		return "none", true
	}

	// Find highest confidence threat
	highestConfidence := 0.0
	for _, threat := range threats {
		if threat.Confidence > highestConfidence {
			highestConfidence = threat.Confidence
		}
	}

	// Categorize based on confidence levels
	if highestConfidence < 0.5 {
		return "none", true
	} else if highestConfidence < 0.6 {
		return "low", true
	} else if highestConfidence < 0.8 {
		return "medium", false
	} else {
		return "high", false
	}
}

func (s *Storage) Add(content string) error {
	return s.AddWithType(content, "text", nil)
}

func (s *Storage) AddImage(imageData []byte, description string) error {
	return s.AddWithType(description, "image", imageData)
}

func (s *Storage) AddWithType(content, contentType string, imageData []byte) error {
	if content == "" && len(imageData) == 0 {
		return nil
	}

	// For text content, check if duplicate exists and update timestamp if found
	if contentType == "text" {
		var existingID string
		normalizedContent := normalizeContentForDeduplication(content)
		
		// Check for existing entries with the same normalized content
		query := "SELECT id FROM clipboard_items WHERE TRIM(content) = ? AND content_type = ? LIMIT 1"
		err := s.db.QueryRow(query, normalizedContent, contentType).Scan(&existingID)
		if err == nil {
			// Duplicate found, update timestamp
			updateQuery := "UPDATE clipboard_items SET timestamp = ? WHERE id = ?"
			_, err := s.db.Exec(updateQuery, time.Now(), existingID)
			return err
		}
		if err != sql.ErrNoRows {
			return err
		}
	} else if contentType == "image" {
		// For images, we need to compare both content and image data
		// Use a simpler approach to avoid database locks
		query := "SELECT id, image_data FROM clipboard_items WHERE content = ? AND content_type = ?"
		rows, err := s.db.Query(query, content, contentType)
		if err != nil {
			return err
		}

		var existingID string
		for rows.Next() {
			var tempID string
			var tempImageData []byte
			if err := rows.Scan(&tempID, &tempImageData); err != nil {
				rows.Close()
				return err
			}
			// Compare image data
			if bytes.Equal(imageData, tempImageData) {
				existingID = tempID
				break
			}
		}
		rows.Close()

		if existingID != "" {
			// Duplicate found, update timestamp
			updateQuery := "UPDATE clipboard_items SET timestamp = ? WHERE id = ?"
			_, err := s.db.Exec(updateQuery, time.Now(), existingID)
			return err
		}
	}

	// No duplicate found, create new entry
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	timestamp := time.Now()

	// Calculate threat level and initial safe entry flag
	threatLevel, safeEntry := calculateThreatLevel(content, contentType)

	query := "INSERT INTO clipboard_items (id, content, content_type, image_data, timestamp, threat_level, safe_entry, is_pinned, pin_order) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)"
	_, err := s.db.Exec(query, id, content, contentType, imageData, timestamp, threatLevel, safeEntry, false, 0)
	if err != nil {
		return err
	}

	// Keep only the latest maxEntries items
	deleteQuery := `
		DELETE FROM clipboard_items 
		WHERE id NOT IN (
			SELECT id FROM clipboard_items 
			ORDER BY timestamp DESC 
			LIMIT ?
		)
	`
	_, err = s.db.Exec(deleteQuery, s.maxEntries)
	return err
}

func (s *Storage) GetAll() []ClipboardItem {
	query := "SELECT id, content, content_type, image_data, timestamp, threat_level, safe_entry, is_pinned, pin_order FROM clipboard_items ORDER BY is_pinned DESC, pin_order ASC, timestamp DESC"
	rows, err := s.db.Query(query)
	if err != nil {
		return []ClipboardItem{}
	}
	defer rows.Close()

	var items []ClipboardItem
	for rows.Next() {
		var item ClipboardItem
		var imageData []byte
		err := rows.Scan(&item.ID, &item.Content, &item.ContentType, &imageData, &item.Timestamp, &item.ThreatLevel, &item.SafeEntry, &item.IsPinned, &item.PinOrder)
		if err != nil {
			continue
		}
		item.ImageData = imageData
		items = append(items, item)
	}

	return items
}

// GetItemCount returns the total number of items in storage
func (s *Storage) GetItemCount() int {
	var count int
	query := "SELECT COUNT(*) FROM clipboard_items"
	err := s.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0
	}
	return count
}

// GetAllMeta returns lightweight metadata for all items (without image data)
func (s *Storage) GetAllMeta() []ClipboardItemMeta {
	query := "SELECT id, content, content_type, timestamp, threat_level, safe_entry, is_pinned, pin_order FROM clipboard_items ORDER BY is_pinned DESC, pin_order ASC, timestamp DESC"
	rows, err := s.db.Query(query)
	if err != nil {
		return []ClipboardItemMeta{}
	}
	defer rows.Close()

	var items []ClipboardItemMeta
	for rows.Next() {
		var item ClipboardItemMeta
		err := rows.Scan(&item.ID, &item.Content, &item.ContentType, &item.Timestamp, &item.ThreatLevel, &item.SafeEntry, &item.IsPinned, &item.PinOrder)
		if err != nil {
			continue
		}
		items = append(items, item)
	}

	return items
}

// GetPage returns a page of lightweight metadata items (without image data)
func (s *Storage) GetPage(offset, limit int) []ClipboardItemMeta {
	query := "SELECT id, content, content_type, timestamp, threat_level, safe_entry, is_pinned, pin_order FROM clipboard_items ORDER BY is_pinned DESC, pin_order ASC, timestamp DESC LIMIT ? OFFSET ?"
	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return []ClipboardItemMeta{}
	}
	defer rows.Close()

	var items []ClipboardItemMeta
	for rows.Next() {
		var item ClipboardItemMeta
		err := rows.Scan(&item.ID, &item.Content, &item.ContentType, &item.Timestamp, &item.ThreatLevel, &item.SafeEntry, &item.IsPinned, &item.PinOrder)
		if err != nil {
			continue
		}
		items = append(items, item)
	}

	return items
}

// GetImageData returns just the image data for a specific item
func (s *Storage) GetImageData(id string) []byte {
	var imageData []byte
	query := "SELECT image_data FROM clipboard_items WHERE id = ?"
	err := s.db.QueryRow(query, id).Scan(&imageData)
	if err != nil {
		return nil
	}
	return imageData
}

// GetFullItem returns a complete ClipboardItem including image data for a specific ID
func (s *Storage) GetFullItem(id string) *ClipboardItem {
	query := "SELECT id, content, content_type, image_data, timestamp, threat_level, safe_entry, is_pinned, pin_order FROM clipboard_items WHERE id = ?"
	row := s.db.QueryRow(query, id)

	var item ClipboardItem
	var imageData []byte
	err := row.Scan(&item.ID, &item.Content, &item.ContentType, &imageData, &item.Timestamp, &item.ThreatLevel, &item.SafeEntry, &item.IsPinned, &item.PinOrder)
	if err != nil {
		return nil
	}
	item.ImageData = imageData

	return &item
}

// ToClipboardItem converts ClipboardItemMeta to ClipboardItem (without image data)
func (meta *ClipboardItemMeta) ToClipboardItem() ClipboardItem {
	return ClipboardItem{
		ID:          meta.ID,
		Content:     meta.Content,
		ContentType: meta.ContentType,
		ImageData:   nil, // Image data not included
		Timestamp:   meta.Timestamp,
		ThreatLevel: meta.ThreatLevel,
		SafeEntry:   meta.SafeEntry,
		IsPinned:    meta.IsPinned,
		PinOrder:    meta.PinOrder,
	}
}

// ToMeta converts ClipboardItem to ClipboardItemMeta (strips image data)
func (item *ClipboardItem) ToMeta() ClipboardItemMeta {
	return ClipboardItemMeta{
		ID:          item.ID,
		Content:     item.Content,
		ContentType: item.ContentType,
		Timestamp:   item.Timestamp,
		ThreatLevel: item.ThreatLevel,
		SafeEntry:   item.SafeEntry,
		IsPinned:    item.IsPinned,
		PinOrder:    item.PinOrder,
	}
}

func (s *Storage) GetByID(id string) *ClipboardItem {
	query := "SELECT id, content, content_type, image_data, timestamp, threat_level, safe_entry, is_pinned, pin_order FROM clipboard_items WHERE id = ?"
	row := s.db.QueryRow(query, id)

	var item ClipboardItem
	var imageData []byte
	err := row.Scan(&item.ID, &item.Content, &item.ContentType, &imageData, &item.Timestamp, &item.ThreatLevel, &item.SafeEntry, &item.IsPinned, &item.PinOrder)
	if err != nil {
		return nil
	}
	item.ImageData = imageData

	return &item
}

func (s *Storage) Update(id string, newContent string) error {
	// Get the item's current content type
	var contentType string
	err := s.db.QueryRow("SELECT content_type FROM clipboard_items WHERE id = ?", id).Scan(&contentType)
	if err != nil {
		return err
	}

	// Recalculate threat level and safe entry for the new content
	threatLevel, safeEntry := calculateThreatLevel(newContent, contentType)

	query := "UPDATE clipboard_items SET content = ?, threat_level = ?, safe_entry = ? WHERE id = ?"
	_, err = s.db.Exec(query, newContent, threatLevel, safeEntry, id)
	return err
}

// UpdateSafeEntry updates the safe_entry flag for a specific item
func (s *Storage) UpdateSafeEntry(id string, safeEntry bool) error {
	query := "UPDATE clipboard_items SET safe_entry = ? WHERE id = ?"
	_, err := s.db.Exec(query, safeEntry, id)
	return err
}

func (s *Storage) Delete(id string) error {
	query := "DELETE FROM clipboard_items WHERE id = ?"
	_, err := s.db.Exec(query, id)
	return err
}

// insertDirectly inserts data directly into the database bypassing deduplication (for testing)
func (s *Storage) insertDirectly(id, content, contentType string, imageData []byte, timestamp time.Time, threatLevel string, safeEntry bool) error {
	query := "INSERT INTO clipboard_items (id, content, content_type, image_data, timestamp, threat_level, safe_entry, is_pinned, pin_order) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)"
	_, err := s.db.Exec(query, id, content, contentType, imageData, timestamp, threatLevel, safeEntry, false, 0)
	return err
}

// DeduplicateExisting removes duplicate entries from the database, keeping the most recent one
func (s *Storage) DeduplicateExisting() (int, error) {
	// Get all items ordered by timestamp DESC (most recent first)
	items := s.GetAll()
	if len(items) <= 1 {
		return 0, nil // Nothing to deduplicate
	}

	// Track seen content and count of removed duplicates
	seenContent := make(map[string]string) // content+type -> ID of first (most recent) occurrence
	var toDelete []string
	removedCount := 0

	for _, item := range items {
		// Create a key for text content
		if item.ContentType == "text" {
			normalizedContent := normalizeContentForDeduplication(item.Content)
			key := fmt.Sprintf("text:%s", normalizedContent)
			if _, exists := seenContent[key]; exists {
				// This is a duplicate, mark for deletion
				toDelete = append(toDelete, item.ID)
				removedCount++
			} else {
				// First occurrence, remember it
				seenContent[key] = item.ID
			}
		} else if item.ContentType == "image" {
			// For images, we need to compare both content and image data
			// Create a key based on content + image data hash
			imageHash := fmt.Sprintf("%x", item.ImageData) // Simple hex representation
			key := fmt.Sprintf("image:%s:%s", item.Content, imageHash)
			if _, exists := seenContent[key]; exists {
				// This is a duplicate, mark for deletion
				toDelete = append(toDelete, item.ID)
				removedCount++
			} else {
				// First occurrence, remember it
				seenContent[key] = item.ID
			}
		}
	}

	// Delete all duplicate entries
	for _, id := range toDelete {
		if err := s.Delete(id); err != nil {
			return removedCount, fmt.Errorf("failed to delete duplicate entry %s: %w", id, err)
		}
	}

	return removedCount, nil
}

// PinItem pins an item to the top of the list
func (s *Storage) PinItem(id string) error {
	// Check if already pinned
	var isPinned bool
	err := s.db.QueryRow("SELECT is_pinned FROM clipboard_items WHERE id = ?", id).Scan(&isPinned)
	if err != nil {
		return err
	}
	if isPinned {
		return nil // Already pinned
	}

	// Check how many items are already pinned
	var pinnedCount int
	err = s.db.QueryRow("SELECT COUNT(*) FROM clipboard_items WHERE is_pinned = TRUE").Scan(&pinnedCount)
	if err != nil {
		return err
	}
	if pinnedCount >= 10 {
		return fmt.Errorf("maximum of 10 items can be pinned")
	}

	// Get the next pin order
	var maxOrder int
	err = s.db.QueryRow("SELECT COALESCE(MAX(pin_order), 0) FROM clipboard_items WHERE is_pinned = TRUE").Scan(&maxOrder)
	if err != nil {
		return err
	}

	// Pin the item
	query := "UPDATE clipboard_items SET is_pinned = TRUE, pin_order = ? WHERE id = ?"
	_, err = s.db.Exec(query, maxOrder+1, id)
	return err
}

// UnpinItem unpins an item from the top of the list
func (s *Storage) UnpinItem(id string) error {
	// Get the current pin order
	var pinOrder int
	var isPinned bool
	err := s.db.QueryRow("SELECT is_pinned, pin_order FROM clipboard_items WHERE id = ?", id).Scan(&isPinned, &pinOrder)
	if err != nil {
		return err
	}
	if !isPinned {
		return nil // Not pinned
	}

	// Unpin the item
	_, err = s.db.Exec("UPDATE clipboard_items SET is_pinned = FALSE, pin_order = 0 WHERE id = ?", id)
	if err != nil {
		return err
	}

	// Reorder remaining pinned items
	_, err = s.db.Exec("UPDATE clipboard_items SET pin_order = pin_order - 1 WHERE is_pinned = TRUE AND pin_order > ?", pinOrder)
	return err
}

// GetPinnedItems returns all pinned items in order
func (s *Storage) GetPinnedItems() []ClipboardItemMeta {
	query := "SELECT id, content, content_type, timestamp, threat_level, safe_entry, is_pinned, pin_order FROM clipboard_items WHERE is_pinned = TRUE ORDER BY pin_order ASC"
	rows, err := s.db.Query(query)
	if err != nil {
		return []ClipboardItemMeta{}
	}
	defer rows.Close()

	var items []ClipboardItemMeta
	for rows.Next() {
		var item ClipboardItemMeta
		err := rows.Scan(&item.ID, &item.Content, &item.ContentType, &item.Timestamp, &item.ThreatLevel, &item.SafeEntry, &item.IsPinned, &item.PinOrder)
		if err != nil {
			continue
		}
		items = append(items, item)
	}

	return items
}

// GetPinnedCount returns the number of pinned items
func (s *Storage) GetPinnedCount() int {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM clipboard_items WHERE is_pinned = TRUE").Scan(&count)
	if err != nil {
		return 0
	}
	return count
}

// RescanSecurityThreats re-scans all items and updates their threat levels
// Returns statistics about the changes made
func (s *Storage) RescanSecurityThreats() (map[string]int, error) {
	stats := map[string]int{
		"total_items":     0,
		"items_scanned":   0,
		"threats_before":  0,
		"threats_after":   0,
		"none_before":     0,
		"none_after":      0,
		"low_before":      0,
		"low_after":       0,
		"medium_before":   0,
		"medium_after":    0,
		"high_before":     0,
		"high_after":      0,
		"downgraded":      0,
		"upgraded":        0,
		"unchanged":       0,
	}

	// Get all items
	items := s.GetAll()
	stats["total_items"] = len(items)

	for _, item := range items {
		// Skip images
		if item.ContentType == "image" {
			continue
		}

		stats["items_scanned"]++
		
		// Record old threat level
		oldThreatLevel := item.ThreatLevel
		switch oldThreatLevel {
		case "none":
			stats["none_before"]++
		case "low":
			stats["low_before"]++
		case "medium":
			stats["medium_before"]++
		case "high":
			stats["high_before"]++
		}

		if oldThreatLevel != "none" {
			stats["threats_before"]++
		}

		// Calculate new threat level
		newThreatLevel, newSafeEntry := calculateThreatLevel(item.Content, item.ContentType)
		
		// Record new threat level
		switch newThreatLevel {
		case "none":
			stats["none_after"]++
		case "low":
			stats["low_after"]++
		case "medium":
			stats["medium_after"]++
		case "high":
			stats["high_after"]++
		}

		if newThreatLevel != "none" {
			stats["threats_after"]++
		}

		// Compare changes
		if oldThreatLevel != newThreatLevel {
			// Determine if it's an upgrade or downgrade
			oldSeverity := threatLevelToSeverity(oldThreatLevel)
			newSeverity := threatLevelToSeverity(newThreatLevel)
			
			if newSeverity > oldSeverity {
				stats["upgraded"]++
			} else {
				stats["downgraded"]++
			}

			// Update the database
			query := "UPDATE clipboard_items SET threat_level = ?, safe_entry = ? WHERE id = ?"
			_, err := s.db.Exec(query, newThreatLevel, newSafeEntry, item.ID)
			if err != nil {
				return stats, fmt.Errorf("failed to update item %s: %w", item.ID, err)
			}
		} else {
			stats["unchanged"]++
		}
	}

	return stats, nil
}

// threatLevelToSeverity converts threat level to numeric severity for comparison
func threatLevelToSeverity(level string) int {
	switch level {
	case "none":
		return 0
	case "low":
		return 1
	case "medium":
		return 2
	case "high":
		return 3
	default:
		return 0
	}
}

// PruneDatabase removes entries based on the provided criteria
func (s *Storage) PruneDatabase(pruneEmptyData, pruneSingleChar bool) (int, error) {
	if !pruneEmptyData && !pruneSingleChar {
		return 0, nil // Nothing to prune
	}

	var conditions []string
	var args []interface{}

	if pruneEmptyData {
		conditions = append(conditions, "content = ''")
	}

	if pruneSingleChar {
		conditions = append(conditions, "LENGTH(content) = 1")
	}

	whereClause := strings.Join(conditions, " OR ")
	query := fmt.Sprintf("DELETE FROM clipboard_items WHERE %s", whereClause)

	result, err := s.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to prune database: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rowsAffected), nil
}

func (s *Storage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
