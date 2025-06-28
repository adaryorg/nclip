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

package security

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func createTestHashStore(t *testing.T) (*HashStore, string) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "nclip-hashstore-test")
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

	store, err := NewHashStore()
	if err != nil {
		t.Fatalf("Failed to create hash store: %v", err)
	}

	t.Cleanup(func() {
		store.Close()
	})

	return store, tmpDir
}

func TestNewHashStore(t *testing.T) {
	store, _ := createTestHashStore(t)

	if store == nil {
		t.Fatal("Expected hash store to be created")
	}

	if store.db == nil {
		t.Fatal("Expected database connection to be established")
	}
}

func TestHashStoreAddAndGet(t *testing.T) {
	store, _ := createTestHashStore(t)

	threat := SecurityThreat{
		Type:       "jwt",
		Confidence: 0.95,
		Reason:     "JWT token detected",
	}

	hash := "test_hash_123"

	// Add hash
	err := store.AddHash(hash, threat)
	if err != nil {
		t.Fatalf("Failed to add hash: %v", err)
	}

	// Get hash
	entry, err := store.GetHash(hash)
	if err != nil {
		t.Fatalf("Failed to get hash: %v", err)
	}

	if entry == nil {
		t.Fatal("Expected hash entry to be found")
	}

	if entry.Hash != hash {
		t.Errorf("Expected hash '%s', got '%s'", hash, entry.Hash)
	}
	if entry.ThreatType != threat.Type {
		t.Errorf("Expected threat type '%s', got '%s'", threat.Type, entry.ThreatType)
	}
	if entry.Confidence != threat.Confidence {
		t.Errorf("Expected confidence %f, got %f", threat.Confidence, entry.Confidence)
	}
	if entry.Reason != threat.Reason {
		t.Errorf("Expected reason '%s', got '%s'", threat.Reason, entry.Reason)
	}
	if entry.Count != 1 {
		t.Errorf("Expected count 1, got %d", entry.Count)
	}
}

func TestHashStoreAddDuplicate(t *testing.T) {
	store, _ := createTestHashStore(t)

	threat := SecurityThreat{
		Type:       "api_key",
		Confidence: 0.8,
		Reason:     "API key detected",
	}

	hash := "duplicate_hash"

	// Add hash first time
	err := store.AddHash(hash, threat)
	if err != nil {
		t.Fatalf("Failed to add hash first time: %v", err)
	}

	// Add hash second time with different confidence
	updatedThreat := SecurityThreat{
		Type:       "api_key",
		Confidence: 0.9,
		Reason:     "Updated API key detection",
	}

	err = store.AddHash(hash, updatedThreat)
	if err != nil {
		t.Fatalf("Failed to add hash second time: %v", err)
	}

	// Get hash and verify it was updated
	entry, err := store.GetHash(hash)
	if err != nil {
		t.Fatalf("Failed to get hash: %v", err)
	}

	if entry.Count != 2 {
		t.Errorf("Expected count 2, got %d", entry.Count)
	}
	if entry.Confidence != 0.9 {
		t.Errorf("Expected confidence 0.9 (updated), got %f", entry.Confidence)
	}
	if entry.Reason != updatedThreat.Reason {
		t.Errorf("Expected updated reason, got '%s'", entry.Reason)
	}
}

func TestHashStoreHasHash(t *testing.T) {
	store, _ := createTestHashStore(t)

	hash := "test_exists_hash"
	threat := SecurityThreat{
		Type:       "password",
		Confidence: 0.7,
		Reason:     "Password detected",
	}

	// Should not exist initially
	exists, err := store.HasHash(hash)
	if err != nil {
		t.Fatalf("Failed to check hash existence: %v", err)
	}
	if exists {
		t.Error("Hash should not exist initially")
	}

	// Add hash
	err = store.AddHash(hash, threat)
	if err != nil {
		t.Fatalf("Failed to add hash: %v", err)
	}

	// Should exist now
	exists, err = store.HasHash(hash)
	if err != nil {
		t.Fatalf("Failed to check hash existence: %v", err)
	}
	if !exists {
		t.Error("Hash should exist after adding")
	}
}

func TestHashStoreGetNonExistent(t *testing.T) {
	store, _ := createTestHashStore(t)

	entry, err := store.GetHash("nonexistent_hash")
	if err != nil {
		t.Fatalf("Get hash should not error for non-existent hash: %v", err)
	}
	if entry != nil {
		t.Error("Expected nil for non-existent hash")
	}
}

func TestHashStoreGetAllHashes(t *testing.T) {
	store, _ := createTestHashStore(t)

	// Add multiple hashes
	hashes := []struct {
		hash   string
		threat SecurityThreat
	}{
		{"hash1", SecurityThreat{Type: "jwt", Confidence: 0.95, Reason: "JWT 1"}},
		{"hash2", SecurityThreat{Type: "api_key", Confidence: 0.8, Reason: "API key 1"}},
		{"hash3", SecurityThreat{Type: "jwt", Confidence: 0.9, Reason: "JWT 2"}},
	}

	for _, h := range hashes {
		err := store.AddHash(h.hash, h.threat)
		if err != nil {
			t.Fatalf("Failed to add hash %s: %v", h.hash, err)
		}
		time.Sleep(time.Millisecond) // Ensure different timestamps
	}

	// Get all hashes
	allHashes, err := store.GetAllHashes("")
	if err != nil {
		t.Fatalf("Failed to get all hashes: %v", err)
	}

	if len(allHashes) != 3 {
		t.Errorf("Expected 3 hashes, got %d", len(allHashes))
	}

	// Get filtered by type
	jwtHashes, err := store.GetAllHashes("jwt")
	if err != nil {
		t.Fatalf("Failed to get JWT hashes: %v", err)
	}

	if len(jwtHashes) != 2 {
		t.Errorf("Expected 2 JWT hashes, got %d", len(jwtHashes))
	}

	for _, hash := range jwtHashes {
		if hash.ThreatType != "jwt" {
			t.Errorf("Expected JWT threat type, got '%s'", hash.ThreatType)
		}
	}
}

func TestHashStoreRemoveHash(t *testing.T) {
	store, _ := createTestHashStore(t)

	hash := "remove_test_hash"
	threat := SecurityThreat{
		Type:       "secret",
		Confidence: 0.6,
		Reason:     "Secret detected",
	}

	// Add hash
	err := store.AddHash(hash, threat)
	if err != nil {
		t.Fatalf("Failed to add hash: %v", err)
	}

	// Verify it exists
	exists, err := store.HasHash(hash)
	if err != nil {
		t.Fatalf("Failed to check hash existence: %v", err)
	}
	if !exists {
		t.Error("Hash should exist after adding")
	}

	// Remove hash
	err = store.RemoveHash(hash)
	if err != nil {
		t.Fatalf("Failed to remove hash: %v", err)
	}

	// Verify it's gone
	exists, err = store.HasHash(hash)
	if err != nil {
		t.Fatalf("Failed to check hash existence after removal: %v", err)
	}
	if exists {
		t.Error("Hash should not exist after removal")
	}

	// GetHash should return nil
	entry, err := store.GetHash(hash)
	if err != nil {
		t.Fatalf("Get hash should not error after removal: %v", err)
	}
	if entry != nil {
		t.Error("Expected nil for removed hash")
	}
}

func TestHashStoreCleanupOldHashes(t *testing.T) {
	store, _ := createTestHashStore(t)

	// Add some hashes
	threat := SecurityThreat{
		Type:       "test",
		Confidence: 0.5,
		Reason:     "Test threat",
	}

	hashes := []string{"old1", "old2", "recent"}
	for _, hash := range hashes {
		err := store.AddHash(hash, threat)
		if err != nil {
			t.Fatalf("Failed to add hash %s: %v", hash, err)
		}
	}

	// Manually update timestamps to make some hashes "old"
	// This is a bit hacky but necessary for testing cleanup
	oneHourAgo := time.Now().Add(-time.Hour)
	_, err := store.db.Exec("UPDATE security_hashes SET last_seen = ? WHERE hash IN (?, ?)",
		oneHourAgo, "old1", "old2")
	if err != nil {
		t.Fatalf("Failed to update timestamps: %v", err)
	}

	// Cleanup hashes older than 30 minutes
	count, err := store.CleanupOldHashes(30 * time.Minute)
	if err != nil {
		t.Fatalf("Failed to cleanup old hashes: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected to remove 2 hashes, removed %d", count)
	}

	// Verify only recent hash remains
	allHashes, err := store.GetAllHashes("")
	if err != nil {
		t.Fatalf("Failed to get remaining hashes: %v", err)
	}

	if len(allHashes) != 1 {
		t.Errorf("Expected 1 remaining hash, got %d", len(allHashes))
	}

	if len(allHashes) > 0 && allHashes[0].Hash != "recent" {
		t.Errorf("Expected 'recent' hash to remain, got '%s'", allHashes[0].Hash)
	}
}

func TestHashStoreGetStats(t *testing.T) {
	store, _ := createTestHashStore(t)

	// Add various hashes
	testData := []struct {
		hash   string
		threat SecurityThreat
	}{
		{"jwt1", SecurityThreat{Type: "jwt", Confidence: 0.95, Reason: "JWT 1"}},
		{"jwt2", SecurityThreat{Type: "jwt", Confidence: 0.9, Reason: "JWT 2"}},
		{"api1", SecurityThreat{Type: "api_key", Confidence: 0.85, Reason: "API 1"}},
		{"pwd1", SecurityThreat{Type: "password", Confidence: 0.7, Reason: "Password 1"}},
	}

	for _, data := range testData {
		err := store.AddHash(data.hash, data.threat)
		if err != nil {
			t.Fatalf("Failed to add hash %s: %v", data.hash, err)
		}
	}

	// Get stats
	stats, err := store.GetStats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	// Check total count
	totalCount, ok := stats["total_hashes"].(int)
	if !ok {
		t.Fatal("Expected total_hashes to be int")
	}
	if totalCount != 4 {
		t.Errorf("Expected total count 4, got %d", totalCount)
	}

	// Check threat type counts
	threatTypes, ok := stats["threat_types"].(map[string]int)
	if !ok {
		t.Fatal("Expected threat_types to be map[string]int")
	}

	if threatTypes["jwt"] != 2 {
		t.Errorf("Expected 2 JWT threats, got %d", threatTypes["jwt"])
	}
	if threatTypes["api_key"] != 1 {
		t.Errorf("Expected 1 API key threat, got %d", threatTypes["api_key"])
	}
	if threatTypes["password"] != 1 {
		t.Errorf("Expected 1 password threat, got %d", threatTypes["password"])
	}

	// Check high confidence count
	highConfidenceCount, ok := stats["high_confidence_count"].(int)
	if !ok {
		t.Fatal("Expected high_confidence_count to be int")
	}
	if highConfidenceCount != 3 { // jwt1, jwt2, api1 have >0.8 confidence
		t.Errorf("Expected 3 high confidence threats, got %d", highConfidenceCount)
	}
}

func TestHashStoreIsKnownSecurityContent(t *testing.T) {
	store, _ := createTestHashStore(t)

	content := "test security content"
	threat := SecurityThreat{
		Type:       "test",
		Confidence: 0.8,
		Reason:     "Test content",
	}

	// Should not be known initially
	known, entry, err := store.IsKnownSecurityContent(content)
	if err != nil {
		t.Fatalf("Failed to check known content: %v", err)
	}
	if known {
		t.Error("Content should not be known initially")
	}
	if entry != nil {
		t.Error("Entry should be nil for unknown content")
	}

	// Add the content hash
	hash := CreateHash(content)
	err = store.AddHash(hash, threat)
	if err != nil {
		t.Fatalf("Failed to add hash: %v", err)
	}

	// Should be known now
	known, entry, err = store.IsKnownSecurityContent(content)
	if err != nil {
		t.Fatalf("Failed to check known content: %v", err)
	}
	if !known {
		t.Error("Content should be known after adding")
	}
	if entry == nil {
		t.Fatal("Entry should not be nil for known content")
	}
	if entry.ThreatType != threat.Type {
		t.Errorf("Expected threat type '%s', got '%s'", threat.Type, entry.ThreatType)
	}
}

func TestHashStoreClose(t *testing.T) {
	store, _ := createTestHashStore(t)

	err := store.Close()
	if err != nil {
		t.Errorf("Close should not error: %v", err)
	}

	// Multiple closes should be safe
	err = store.Close()
	if err != nil {
		t.Errorf("Multiple closes should not error: %v", err)
	}
}

func TestHashStoreDatabaseCreation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nclip-hashstore-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Override home directory
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	store, err := NewHashStore()
	if err != nil {
		t.Fatalf("Failed to create hash store: %v", err)
	}
	defer store.Close()

	// Check if database file was created
	dbPath := filepath.Join(tmpDir, ".config", "nclip", "security_hashes.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Security database file was not created")
	}
}

func TestHashStoreInvalidHomeDirectory(t *testing.T) {
	// Save original HOME
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	// Unset HOME to trigger error
	os.Unsetenv("HOME")

	_, err := NewHashStore()
	if err == nil {
		// On some systems this might not fail, so we skip the test
		t.Skip("UserHomeDir didn't fail on this system")
	}
	if !strings.Contains(err.Error(), "failed to get user home directory") {
		t.Errorf("Expected home directory error, got: %v", err)
	}
}
