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

	return nil
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
		query := "SELECT id FROM clipboard_items WHERE content = ? AND content_type = ? LIMIT 1"
		err := s.db.QueryRow(query, content, contentType).Scan(&existingID)
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

	query := "INSERT INTO clipboard_items (id, content, content_type, image_data, timestamp, threat_level, safe_entry) VALUES (?, ?, ?, ?, ?, ?, ?)"
	_, err := s.db.Exec(query, id, content, contentType, imageData, timestamp, threatLevel, safeEntry)
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
	query := "SELECT id, content, content_type, image_data, timestamp, threat_level, safe_entry FROM clipboard_items ORDER BY timestamp DESC"
	rows, err := s.db.Query(query)
	if err != nil {
		return []ClipboardItem{}
	}
	defer rows.Close()

	var items []ClipboardItem
	for rows.Next() {
		var item ClipboardItem
		var imageData []byte
		err := rows.Scan(&item.ID, &item.Content, &item.ContentType, &imageData, &item.Timestamp, &item.ThreatLevel, &item.SafeEntry)
		if err != nil {
			continue
		}
		item.ImageData = imageData
		items = append(items, item)
	}

	return items
}

func (s *Storage) GetByID(id string) *ClipboardItem {
	query := "SELECT id, content, content_type, image_data, timestamp, threat_level, safe_entry FROM clipboard_items WHERE id = ?"
	row := s.db.QueryRow(query, id)

	var item ClipboardItem
	var imageData []byte
	err := row.Scan(&item.ID, &item.Content, &item.ContentType, &imageData, &item.Timestamp, &item.ThreatLevel, &item.SafeEntry)
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
	query := "INSERT INTO clipboard_items (id, content, content_type, image_data, timestamp, threat_level, safe_entry) VALUES (?, ?, ?, ?, ?, ?, ?)"
	_, err := s.db.Exec(query, id, content, contentType, imageData, timestamp, threatLevel, safeEntry)
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
			key := fmt.Sprintf("text:%s", item.Content)
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

func (s *Storage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
