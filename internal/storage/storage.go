package storage

import (
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

	// Check if this content is already the most recent
	var lastContent string
	var lastType string
	err := s.db.QueryRow("SELECT content, content_type FROM clipboard_items ORDER BY timestamp DESC LIMIT 1").Scan(&lastContent, &lastType)
	if err == nil && lastContent == content && lastType == contentType {
		return nil
	}

	id := fmt.Sprintf("%d", time.Now().UnixNano())
	timestamp := time.Now()

	// Calculate threat level and initial safe entry flag
	threatLevel, safeEntry := calculateThreatLevel(content, contentType)

	query := "INSERT INTO clipboard_items (id, content, content_type, image_data, timestamp, threat_level, safe_entry) VALUES (?, ?, ?, ?, ?, ?, ?)"
	_, err = s.db.Exec(query, id, content, contentType, imageData, timestamp, threatLevel, safeEntry)
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

func (s *Storage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
