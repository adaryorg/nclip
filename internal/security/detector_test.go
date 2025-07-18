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
	"strings"
	"testing"
)

func TestNewSecurityDetector(t *testing.T) {
	detector := NewSecurityDetector()
	if detector == nil {
		t.Fatal("Expected detector to be created")
	}

	if detector.patterns == nil {
		t.Fatal("Expected patterns to be initialized")
	}

	// Check that some patterns are compiled
	if len(detector.patterns) == 0 {
		t.Fatal("Expected some patterns to be compiled")
	}
}

func TestDetectSecurity_EmptyContent(t *testing.T) {
	detector := NewSecurityDetector()
	threats := detector.DetectSecurity("")
	if len(threats) != 0 {
		t.Errorf("Expected no threats for empty content, got %d", len(threats))
	}

	threats = detector.DetectSecurity("   ")
	if len(threats) != 0 {
		t.Errorf("Expected no threats for whitespace content, got %d", len(threats))
	}
}

func TestDetectSecurity_JWT(t *testing.T) {
	detector := NewSecurityDetector()

	// Valid JWT format
	jwtToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	threats := detector.DetectSecurity(jwtToken)

	if len(threats) == 0 {
		t.Error("Expected to detect JWT token")
	}

	found := false
	for _, threat := range threats {
		if threat.Type == "jwt" {
			found = true
			if threat.Confidence < 0.9 {
				t.Errorf("Expected high confidence for JWT, got %f", threat.Confidence)
			}
		}
	}
	if !found {
		t.Error("Expected JWT threat type")
	}
}

func TestDetectSecurity_GitHubToken(t *testing.T) {
	detector := NewSecurityDetector()

	// GitHub personal access token format
	githubToken := "ghp_abcdefghijklmnopqrstuvwxyz1234567890"
	threats := detector.DetectSecurity(githubToken)

	if len(threats) == 0 {
		t.Error("Expected to detect GitHub token")
	}

	found := false
	for _, threat := range threats {
		if threat.Type == "api_key" && strings.Contains(threat.Reason, "GitHub") {
			found = true
			if threat.Confidence < 0.9 {
				t.Errorf("Expected high confidence for GitHub token, got %f", threat.Confidence)
			}
		}
	}
	if !found {
		t.Error("Expected GitHub API key threat type")
	}
}

func TestDetectSecurity_SSHKey(t *testing.T) {
	detector := NewSecurityDetector()

	// SSH private key format
	sshKey := "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA..."
	threats := detector.DetectSecurity(sshKey)

	if len(threats) == 0 {
		t.Error("Expected to detect SSH key")
	}

	found := false
	for _, threat := range threats {
		if threat.Type == "ssh_key" {
			found = true
			if threat.Confidence < 0.9 {
				t.Errorf("Expected high confidence for SSH key, got %f", threat.Confidence)
			}
		}
	}
	if !found {
		t.Error("Expected SSH key threat type")
	}
}

func TestDetectSecurity_PasswordField(t *testing.T) {
	detector := NewSecurityDetector()

	// Password field pattern
	passwordField := "password=mySecretPassword123!"
	threats := detector.DetectSecurity(passwordField)

	if len(threats) == 0 {
		t.Error("Expected to detect password field")
	}

	found := false
	for _, threat := range threats {
		if threat.Type == "password" {
			found = true
		}
	}
	if !found {
		t.Error("Expected password threat type")
	}
}

func TestDetectSecurity_DatabaseURL(t *testing.T) {
	detector := NewSecurityDetector()

	// Database connection string
	dbURL := "postgresql://user:password@localhost:5432/mydb"
	threats := detector.DetectSecurity(dbURL)

	if len(threats) == 0 {
		t.Error("Expected to detect database URL")
	}

	found := false
	for _, threat := range threats {
		if threat.Type == "connection_string" {
			found = true
		}
	}
	if !found {
		t.Error("Expected connection string threat type")
	}
}

func TestDetectSecurity_InnocentContent(t *testing.T) {
	detector := NewSecurityDetector()

	innocentTexts := []string{
		"Hello world",
		"user@example.com",
		"https://example.com",
		"shopping list: milk, bread, eggs",
	}

	for _, text := range innocentTexts {
		threats := detector.DetectSecurity(text)
		if len(threats) > 0 {
			// Log what threats were detected for debugging
			for _, threat := range threats {
				t.Logf("Text '%s' detected threat: Type=%s, Confidence=%f, Reason=%s",
					text, threat.Type, threat.Confidence, threat.Reason)
			}
			t.Errorf("Expected no threats for innocent text '%s', got %d threats", text, len(threats))
		}
	}

	// Test some borderline cases that might trigger false positives
	borderlineCases := []string{
		"This is a normal message", // Might trigger random token detection
		"123456",                   // Short number, might trigger some patterns
	}

	for _, text := range borderlineCases {
		threats := detector.DetectSecurity(text)
		// Log but don't fail - these might have low-confidence detections
		if len(threats) > 0 {
			t.Logf("Borderline case '%s' detected %d threats (this may be expected)", text, len(threats))
			for _, threat := range threats {
				t.Logf("  - Type=%s, Confidence=%f, Reason=%s",
					threat.Type, threat.Confidence, threat.Reason)
			}
		}
	}
}

func TestLooksLikePassword(t *testing.T) {
	detector := NewSecurityDetector()

	tests := []struct {
		input    string
		expected bool
	}{
		{"Password123!", true},            // Has all 4 types (upper, lower, digit, special)
		{"Password123", true},             // Has 3 types (upper, lower, digit)
		{"password123!", true},            // Has 3 types (lower, digit, special)
		{"PASSWORD123!", true},            // Has 3 types (upper, digit, special)
		{"Password!@#", true},             // Has 3 types (upper, lower, special)
		{"password123", false},            // Only 2 types (lower, digit)
		{"PASSWORD123", false},            // Only 2 types (upper, digit)
		{"password!@#", false},            // Only 2 types (lower, special)
		{"password", false},               // Only lowercase
		{"12345678", false},               // Only digits
		{"!@#$%^&*", false},               // Only special chars
		{"P1!", false},                    // Too short (3 chars)
		{"", false},                       // Empty
		{strings.Repeat("a", 200), false}, // Too long
		{"Abc12345", true},                // Has 3 types (upper, lower, digit)
		{"abc!@#$%", false},               // Only 2 types (lower, special)
		{"Abc!@#$%", true},                // Has 3 types (upper, lower, special)
		{"MySecure123!", true},            // Has all 4 types
		{"Complex9*", true},               // Has all 4 types
		{"Test with space", false},        // Contains space
		{"Test\nNewline", false},          // Contains newline
		{"VeryLongPasswordThatExceedsTheFortyFiveCharacterLimit123!", false}, // Too long (>40 chars)
		{"Short7!", false},                // Too short (8 chars minimum)
		{"Perfect8!", true},               // Exactly 8 chars with all types
	}

	for _, test := range tests {
		result := detector.looksLikePassword(test.input)
		if result != test.expected {
			t.Errorf("looksLikePassword('%s') = %v, expected %v", test.input, result, test.expected)
		}
	}
}

func TestIsSensitiveEnvVar(t *testing.T) {
	detector := NewSecurityDetector()

	tests := []struct {
		input    string
		expected bool
	}{
		{"PASSWORD=secret123", true},
		{"API_KEY=abc123", true},
		{"SECRET_TOKEN=xyz789", true},
		{"DATABASE_URL=postgres://...", true},
		{"JWT_SECRET=mysecret", true},
		{"USERNAME=john", false},
		{"PORT=3000", false},
		{"DEBUG=true", false},
		{"normal text", false},
	}

	for _, test := range tests {
		result := detector.isSensitiveEnvVar(test.input)
		if result != test.expected {
			t.Errorf("isSensitiveEnvVar('%s') = %v, expected %v", test.input, result, test.expected)
		}
	}
}

func TestIsRandomToken(t *testing.T) {
	detector := NewSecurityDetector()

	tests := []struct {
		input    string
		expected bool
	}{
		{"abcdefghijklmnopqrstuvwxyz123456", true}, // Long alphanumeric (32 chars)
		{"abc123def456ghi789jkl012345678901", true}, // Mixed alphanumeric (32 chars)
		{"token_with_underscores_123456789012", true}, // With underscores (32 chars)
		{"token-with-dashes-456789012345678", true}, // With dashes (32 chars)
		{strings.Repeat("a", 32), true},             // Exactly 32 chars
		{strings.Repeat("a", 256), true},            // Exactly 256 chars
		{"abcdefghijklmnopqrstuvwxyz", false},       // Too short (26 chars)
		{"abc123def456ghi789jkl", false},            // Too short (22 chars)
		{"token_with_underscores_123", false},       // Too short (28 chars)
		{"token-with-dashes-456", false},            // Too short (23 chars)
		{"short", false},                            // Too short
		{strings.Repeat("a", 600), false},           // Too long
		{"hello world test" + strings.Repeat("a", 20), false}, // Has spaces
		{"token@#$%^&*()+=abcdefghijklmnop", false}, // Too many special chars
		{"validtoken123" + strings.Repeat("a", 20), true}, // Valid token
		{"token with space" + strings.Repeat("a", 15), false}, // Contains space
		{"token\nwith\nnewline" + strings.Repeat("a", 10), false}, // Contains newline
	}

	for _, test := range tests {
		result := detector.isRandomToken(test.input)
		if result != test.expected {
			t.Errorf("isRandomToken('%s') = %v, expected %v", test.input, result, test.expected)
		}
	}
}

func TestIsUnsafeURL(t *testing.T) {
	detector := NewSecurityDetector()

	tests := []struct {
		input    string
		expected bool
	}{
		// Safe URLs (no query parameters)
		{"https://example.com", false},
		{"https://example.com/page", false},
		{"https://example.com/page/subpage", false},
		{"http://example.com", false},
		{"https://api.example.com/v1/users", false},
		
		// URLs with single parameter (no &)
		{"https://example.com?page=1", false},
		{"https://example.com?search=test", false},
		
		// Unsafe URLs (with query parameters containing &)
		{"https://example.com?page=1&limit=10", true},
		{"https://api.example.com/v1/data?token=abc123&format=json", true},
		{"https://example.com?api_key=secret&user=admin", true},
		{"https://oauth.example.com?access_token=xyz789&refresh_token=abc123", true},
		{"https://example.com?session=abc&csrf=def", true},
		
		// URLs with suspicious parameter names
		{"https://api.example.com?token=abc123&other=value", true},
		{"https://example.com?access_token=secret&page=1", true},
		{"https://example.com?api_key=key123&format=json", true},
		{"https://example.com?password=secret&user=admin", true},
		{"https://example.com?auth=bearer123&action=login", true},
		
		// Invalid cases
		{"ftp://example.com?token=abc&other=def", false},    // Wrong scheme
		{"https://example.com?token=abc other=def", false},  // Contains space
		{"https://example.com?token=abc\nother=def", false}, // Contains newline
		{"not a url", false},                                // Not a URL
		{"", false},                                         // Empty string
	}

	for _, test := range tests {
		result := detector.isUnsafeURL(test.input)
		if result != test.expected {
			t.Errorf("isUnsafeURL('%s') = %v, expected %v", test.input, result, test.expected)
		}
	}
}

func TestCreateHash(t *testing.T) {
	content := "test content"
	hash1 := CreateHash(content)
	hash2 := CreateHash(content)

	if hash1 != hash2 {
		t.Error("Same content should produce same hash")
	}

	if len(hash1) != 64 {
		t.Errorf("Expected SHA256 hash length 64, got %d", len(hash1))
	}

	// Different content should produce different hash
	differentHash := CreateHash("different content")
	if hash1 == differentHash {
		t.Error("Different content should produce different hash")
	}
}

func TestIsHighRiskThreat(t *testing.T) {
	tests := []struct {
		threats  []SecurityThreat
		expected bool
	}{
		{
			[]SecurityThreat{
				{Type: "jwt", Confidence: 0.95},
			},
			true,
		},
		{
			[]SecurityThreat{
				{Type: "api_key", Confidence: 0.98},
			},
			true,
		},
		{
			[]SecurityThreat{
				{Type: "password", Confidence: 0.7},
			},
			false,
		},
		{
			[]SecurityThreat{
				{Type: "jwt", Confidence: 0.5}, // Low confidence
			},
			false,
		},
		{
			[]SecurityThreat{},
			false,
		},
	}

	for i, test := range tests {
		result := IsHighRiskThreat(test.threats)
		if result != test.expected {
			t.Errorf("Test %d: IsHighRiskThreat = %v, expected %v", i, result, test.expected)
		}
	}
}

func TestGetHighestThreat(t *testing.T) {
	// Empty slice
	result := GetHighestThreat([]SecurityThreat{})
	if result != nil {
		t.Error("Expected nil for empty threat slice")
	}

	// Single threat
	threats := []SecurityThreat{
		{Type: "password", Confidence: 0.7},
	}
	result = GetHighestThreat(threats)
	if result == nil {
		t.Fatal("Expected threat to be returned")
	}
	if result.Confidence != 0.7 {
		t.Errorf("Expected confidence 0.7, got %f", result.Confidence)
	}

	// Multiple threats
	threats = []SecurityThreat{
		{Type: "password", Confidence: 0.7},
		{Type: "jwt", Confidence: 0.95},
		{Type: "api_key", Confidence: 0.8},
	}
	result = GetHighestThreat(threats)
	if result == nil {
		t.Fatal("Expected threat to be returned")
	}
	if result.Type != "jwt" {
		t.Errorf("Expected highest threat to be 'jwt', got '%s'", result.Type)
	}
	if result.Confidence != 0.95 {
		t.Errorf("Expected confidence 0.95, got %f", result.Confidence)
	}
}

func TestClassifyThreat(t *testing.T) {
	detector := NewSecurityDetector()

	tests := []struct {
		patternName   string
		content       string
		expectedType  string
		minConfidence float64
	}{
		{"jwt", "eyJ...", "jwt", 0.9},
		{"github_token", "ghp_...", "api_key", 0.9},
		{"aws_access_key", "AKIA...", "api_key", 0.9},
		{"ssh_private_key", "-----BEGIN...", "ssh_key", 0.9},
		{"password_field", "password=...", "password", 0.7},
		{"credit_card", "4111111111111111", "credit_card", 0.6},
		{"db_connection", "postgres://...", "connection_string", 0.8},
		{"unknown_pattern", "something", "secret", 0.5},
	}

	for _, test := range tests {
		threat := detector.classifyThreat(test.patternName, test.content)
		if threat.Type != test.expectedType {
			t.Errorf("Pattern '%s': expected type '%s', got '%s'",
				test.patternName, test.expectedType, threat.Type)
		}
		if threat.Confidence < test.minConfidence {
			t.Errorf("Pattern '%s': expected confidence >= %f, got %f",
				test.patternName, test.minConfidence, threat.Confidence)
		}
		if threat.Reason == "" {
			t.Errorf("Pattern '%s': expected non-empty reason", test.patternName)
		}
	}
}

func TestHeuristicChecks(t *testing.T) {
	detector := NewSecurityDetector()

	// Test password-like content
	threats := detector.heuristicChecks("MyPassword123!")
	found := false
	for _, threat := range threats {
		if threat.Type == "password" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to detect password-like characteristics")
	}

	// Test environment variable
	threats = detector.heuristicChecks("SECRET_KEY=abcdef123")
	found = false
	for _, threat := range threats {
		if threat.Type == "env_secret" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to detect sensitive environment variable")
	}

	// Test random token
	threats = detector.heuristicChecks("abcdefghijklmnopqrstuvwxyz123456789")
	found = false
	for _, threat := range threats {
		if threat.Type == "token" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to detect random token-like string")
	}

	// Test innocent content
	threats = detector.heuristicChecks("hello world")
	if len(threats) > 0 {
		t.Error("Expected no threats for innocent content")
	}
}

func TestPatternCompilation(t *testing.T) {
	detector := NewSecurityDetector()

	// Test that common patterns are compiled
	expectedPatterns := []string{
		"jwt", "github_token", "aws_access_key", "ssh_private_key",
		"password_field", "credit_card", "uuid",
	}

	for _, pattern := range expectedPatterns {
		if _, exists := detector.patterns[pattern]; !exists {
			t.Errorf("Expected pattern '%s' to be compiled", pattern)
		}
	}
}
