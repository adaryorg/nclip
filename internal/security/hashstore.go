package security

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// HashStore manages a separate database of security-related content hashes
type HashStore struct {
	db *sql.DB
}

// SecurityHash represents a stored security hash entry
type SecurityHash struct {
	Hash       string    `json:"hash"`
	ThreatType string    `json:"threat_type"`
	Confidence float64   `json:"confidence"`
	Reason     string    `json:"reason"`
	FirstSeen  time.Time `json:"first_seen"`
	LastSeen   time.Time `json:"last_seen"`
	Count      int       `json:"count"`
}

// NewHashStore creates a new security hash store
func NewHashStore() (*HashStore, error) {
	// Get user's config directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "nclip")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Open security hashes database
	dbPath := filepath.Join(configDir, "security_hashes.db")
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open security database: %w", err)
	}

	store := &HashStore{db: db}

	// Initialize database schema
	if err := store.initSchema(); err != nil {
		store.Close()
		return nil, fmt.Errorf("failed to initialize security database schema: %w", err)
	}

	return store, nil
}

// initSchema creates the security hashes table if it doesn't exist
func (s *HashStore) initSchema() error {
	query := `
	CREATE TABLE IF NOT EXISTS security_hashes (
		hash TEXT PRIMARY KEY,
		threat_type TEXT NOT NULL,
		confidence REAL NOT NULL,
		reason TEXT NOT NULL,
		first_seen DATETIME NOT NULL,
		last_seen DATETIME NOT NULL,
		count INTEGER NOT NULL DEFAULT 1
	);
	
	CREATE INDEX IF NOT EXISTS idx_threat_type ON security_hashes(threat_type);
	CREATE INDEX IF NOT EXISTS idx_confidence ON security_hashes(confidence);
	CREATE INDEX IF NOT EXISTS idx_last_seen ON security_hashes(last_seen);
	`

	_, err := s.db.Exec(query)
	return err
}

// AddHash stores a security hash in the database
func (s *HashStore) AddHash(hash string, threat SecurityThreat) error {
	now := time.Now()

	// Check if hash already exists
	var existingCount int
	var existingFirstSeen time.Time

	err := s.db.QueryRow("SELECT count, first_seen FROM security_hashes WHERE hash = ?", hash).
		Scan(&existingCount, &existingFirstSeen)

	if err == sql.ErrNoRows {
		// New hash, insert it
		_, err := s.db.Exec(`
			INSERT INTO security_hashes (hash, threat_type, confidence, reason, first_seen, last_seen, count)
			VALUES (?, ?, ?, ?, ?, ?, 1)
		`, hash, threat.Type, threat.Confidence, threat.Reason, now, now)
		return err
	} else if err != nil {
		return fmt.Errorf("failed to check existing hash: %w", err)
	}

	// Hash exists, update count and last_seen
	_, err = s.db.Exec(`
		UPDATE security_hashes 
		SET last_seen = ?, count = count + 1, confidence = MAX(confidence, ?), reason = ?
		WHERE hash = ?
	`, now, threat.Confidence, threat.Reason, hash)

	return err
}

// HasHash checks if a security hash exists in the database
func (s *HashStore) HasHash(hash string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT 1 FROM security_hashes WHERE hash = ? LIMIT 1", hash).Scan(&count)

	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to check hash existence: %w", err)
	}

	return true, nil
}

// GetHash retrieves a security hash entry from the database
func (s *HashStore) GetHash(hash string) (*SecurityHash, error) {
	var entry SecurityHash

	err := s.db.QueryRow(`
		SELECT hash, threat_type, confidence, reason, first_seen, last_seen, count
		FROM security_hashes WHERE hash = ?
	`, hash).Scan(
		&entry.Hash, &entry.ThreatType, &entry.Confidence,
		&entry.Reason, &entry.FirstSeen, &entry.LastSeen, &entry.Count,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get hash: %w", err)
	}

	return &entry, nil
}

// GetAllHashes retrieves all security hashes, optionally filtered by threat type
func (s *HashStore) GetAllHashes(threatType string) ([]SecurityHash, error) {
	var query string
	var args []interface{}

	if threatType != "" {
		query = `
			SELECT hash, threat_type, confidence, reason, first_seen, last_seen, count
			FROM security_hashes WHERE threat_type = ?
			ORDER BY last_seen DESC
		`
		args = []interface{}{threatType}
	} else {
		query = `
			SELECT hash, threat_type, confidence, reason, first_seen, last_seen, count
			FROM security_hashes
			ORDER BY last_seen DESC
		`
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query hashes: %w", err)
	}
	defer rows.Close()

	var hashes []SecurityHash
	for rows.Next() {
		var entry SecurityHash
		err := rows.Scan(
			&entry.Hash, &entry.ThreatType, &entry.Confidence,
			&entry.Reason, &entry.FirstSeen, &entry.LastSeen, &entry.Count,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan hash row: %w", err)
		}
		hashes = append(hashes, entry)
	}

	return hashes, rows.Err()
}

// RemoveHash removes a security hash from the database
func (s *HashStore) RemoveHash(hash string) error {
	_, err := s.db.Exec("DELETE FROM security_hashes WHERE hash = ?", hash)
	if err != nil {
		return fmt.Errorf("failed to remove hash: %w", err)
	}
	return nil
}

// CleanupOldHashes removes hashes older than the specified duration
func (s *HashStore) CleanupOldHashes(olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)

	result, err := s.db.Exec("DELETE FROM security_hashes WHERE last_seen < ?", cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old hashes: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}

	return rowsAffected, nil
}

// GetStats returns statistics about the security hash database
func (s *HashStore) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total count
	var totalCount int
	err := s.db.QueryRow("SELECT COUNT(*) FROM security_hashes").Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}
	stats["total_hashes"] = totalCount

	// Count by threat type
	rows, err := s.db.Query(`
		SELECT threat_type, COUNT(*) as count 
		FROM security_hashes 
		GROUP BY threat_type 
		ORDER BY count DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get threat type stats: %w", err)
	}
	defer rows.Close()

	threatCounts := make(map[string]int)
	for rows.Next() {
		var threatType string
		var count int
		if err := rows.Scan(&threatType, &count); err != nil {
			return nil, fmt.Errorf("failed to scan threat type row: %w", err)
		}
		threatCounts[threatType] = count
	}
	stats["threat_types"] = threatCounts

	// High confidence threats
	var highConfidenceCount int
	err = s.db.QueryRow("SELECT COUNT(*) FROM security_hashes WHERE confidence > 0.8").Scan(&highConfidenceCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get high confidence count: %w", err)
	}
	stats["high_confidence_count"] = highConfidenceCount

	return stats, nil
}

// Close closes the database connection
func (s *HashStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// IsKnownSecurityContent checks if content is already known to be security-related
func (s *HashStore) IsKnownSecurityContent(content string) (bool, *SecurityHash, error) {
	hash := CreateHash(content)
	exists, err := s.HasHash(hash)
	if err != nil {
		return false, nil, err
	}

	if !exists {
		return false, nil, nil
	}

	entry, err := s.GetHash(hash)
	if err != nil {
		return false, nil, err
	}

	return true, entry, nil
}
