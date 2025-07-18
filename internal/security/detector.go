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
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

// SecurityThreat represents the type and confidence level of detected security data
type SecurityThreat struct {
	Type       string  // "password", "jwt", "api_key", "ssh_key", "secret", "token"
	Confidence float64 // 0.0 to 1.0
	Reason     string  // Human readable explanation
}

// SecurityDetector contains patterns and logic for detecting sensitive information
type SecurityDetector struct {
	patterns map[string]*regexp.Regexp
}

// NewSecurityDetector creates a new security detector with predefined patterns
func NewSecurityDetector() *SecurityDetector {
	detector := &SecurityDetector{
		patterns: make(map[string]*regexp.Regexp),
	}

	// Compile all security patterns
	detector.compilePatterns()

	return detector
}

// compilePatterns initializes all the regex patterns for security detection
func (d *SecurityDetector) compilePatterns() {
	patterns := map[string]string{
		// JWT tokens (3 base64 parts separated by dots)
		"jwt": `^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$`,

		// API keys and tokens (various formats)
		"github_token":    `^(ghp_|gho_|ghu_|ghs_|ghr_)[A-Za-z0-9_]{36,}$`,
		"github_app":      `^(ghp_)[A-Za-z0-9_]{36}$`,
		"slack_token":     `^xox[baprs]-([0-9a-zA-Z]{10,48})?$`,
		"discord_token":   `^[MN][A-Za-z\d]{23}\.[A-Za-z\d-_]{6}\.[A-Za-z\d-_]{27}$`,
		"telegram_token":  `^\d{8,10}:[A-Za-z0-9_-]{35}$`,
		"aws_access_key":  `^AKIA[0-9A-Z]{16}$`,
		"aws_secret_key":  `^[A-Za-z0-9/+=]{40}$`,
		"google_api":      `^AIza[0-9A-Za-z_-]{35}$`,
		"stripe_key":      `^(sk_|pk_)(test_|live_)?[0-9a-zA-Z]{24}$`,
		"paypal_token":    `^access_token\$production\$[a-z0-9]{16}\$[a-f0-9]{32}$`,
		"square_token":    `^EAAA[a-zA-Z0-9_-]+$`,
		"twilio_sid":      `^AC[a-f0-9]{32}$`,
		"sendgrid_key":    `^SG\.[a-zA-Z0-9_-]{22}\.[a-zA-Z0-9_-]{43}$`,
		"mailgun_key":     `^key-[a-f0-9]{32}$`,
		"cloudinary_key":  `^[0-9]{15}$`,
		"firebase_token":  `^1//[A-Za-z0-9_-]{43,}$`,
		"heroku_api":      `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`,
		"shopify_token":   `^shpat_[a-fA-F0-9]{32}$`,
		"generic_api_key": `^[a-zA-Z0-9_-]{32,}$`,

		// SSH keys
		"ssh_private_key": `-----BEGIN (RSA |DSA |EC |OPENSSH )?PRIVATE KEY-----`,
		"ssh_public_key":  `^ssh-(rsa|dss|ed25519|ecdsa) [A-Za-z0-9+/]+=*( .*)?$`,

		// PGP keys
		"pgp_private_key": `-----BEGIN PGP PRIVATE KEY BLOCK-----`,
		"pgp_public_key":  `-----BEGIN PGP PUBLIC KEY BLOCK-----`,

		// Certificates
		"certificate":  `-----BEGIN CERTIFICATE-----`,
		"private_cert": `-----BEGIN PRIVATE KEY-----`,

		// Database connection strings
		"db_connection": `^(mysql|postgresql|mongodb|redis)://.*:.*@.*$`,
		"db_url":        `^(postgres|mysql|mongodb|redis)://[^:]+:[^@]+@[^/]+(/.*)?$`,

		// Generic tokens and secrets
		"bearer_token": `^Bearer [A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$`,
		"basic_auth":   `^Basic [A-Za-z0-9+/]+=*$`,
		"oauth_token":  `^[A-Za-z0-9_-]{43}$`,

		// Password-like patterns
		"password_field": `(?i)(password|passwd|pwd|secret|key|token)\s*[:=]\s*[^\s\n]{8,}`,
		"auth_header":    `(?i)authorization\s*:\s*(bearer|basic|token)\s+[a-zA-Z0-9_.-]+`,

		// Credit card numbers (basic pattern)
		"credit_card": `^[0-9]{4}[- ]?[0-9]{4}[- ]?[0-9]{4}[- ]?[0-9]{4}$`,

		// UUIDs (often used as secrets)
		"uuid": `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`,

		// Hash-like strings (common in tokens)
		"sha256_hash": `^[a-f0-9]{64}$`,
		"sha1_hash":   `^[a-f0-9]{40}$`,
		"md5_hash":    `^[a-f0-9]{32}$`,
	}

	// Compile all patterns
	for name, pattern := range patterns {
		if regex, err := regexp.Compile(pattern); err == nil {
			d.patterns[name] = regex
		}
	}
}

// DetectSecurity analyzes text content for potential security threats
func (d *SecurityDetector) DetectSecurity(content string) []SecurityThreat {
	var threats []SecurityThreat

	content = strings.TrimSpace(content)
	if len(content) == 0 {
		return threats
	}

	// Check each pattern
	for patternName, regex := range d.patterns {
		if regex.MatchString(content) {
			threat := d.classifyThreat(patternName, content)
			if threat.Confidence > 0.5 { // Only include high-confidence matches
				threats = append(threats, threat)
			}
		}
	}

	// Additional heuristic checks
	threats = append(threats, d.heuristicChecks(content)...)

	return threats
}

// classifyThreat converts pattern matches into structured threat information
func (d *SecurityDetector) classifyThreat(patternName, content string) SecurityThreat {
	switch {
	case strings.Contains(patternName, "jwt"):
		return SecurityThreat{
			Type:       "jwt",
			Confidence: 0.95,
			Reason:     "JWT token detected (3-part base64 structure)",
		}
	case strings.Contains(patternName, "github"):
		return SecurityThreat{
			Type:       "api_key",
			Confidence: 0.98,
			Reason:     "GitHub personal access token detected",
		}
	case strings.Contains(patternName, "aws"):
		return SecurityThreat{
			Type:       "api_key",
			Confidence: 0.95,
			Reason:     "AWS access key detected",
		}
	case strings.Contains(patternName, "ssh"):
		return SecurityThreat{
			Type:       "ssh_key",
			Confidence: 0.99,
			Reason:     "SSH key detected",
		}
	case strings.Contains(patternName, "private"):
		return SecurityThreat{
			Type:       "private_key",
			Confidence: 0.99,
			Reason:     "Private key detected",
		}
	case strings.Contains(patternName, "password"):
		return SecurityThreat{
			Type:       "password",
			Confidence: 0.8,
			Reason:     "Password field detected",
		}
	case strings.Contains(patternName, "credit_card"):
		return SecurityThreat{
			Type:       "credit_card",
			Confidence: 0.7,
			Reason:     "Credit card number pattern detected",
		}
	case strings.Contains(patternName, "db_"):
		return SecurityThreat{
			Type:       "connection_string",
			Confidence: 0.9,
			Reason:     "Database connection string detected",
		}
	case strings.Contains(patternName, "token") || strings.Contains(patternName, "key"):
		return SecurityThreat{
			Type:       "token",
			Confidence: 0.8,
			Reason:     "API token or key detected",
		}
	case strings.Contains(patternName, "url"):
		return SecurityThreat{
			Type:       "url_with_params",
			Confidence: 0.65,
			Reason:     "URL with query parameters detected",
		}
	default:
		return SecurityThreat{
			Type:       "secret",
			Confidence: 0.6,
			Reason:     "Potential secret detected",
		}
	}
}

// heuristicChecks performs additional security checks based on content analysis
func (d *SecurityDetector) heuristicChecks(content string) []SecurityThreat {
	var threats []SecurityThreat

	// Determine content type for context-aware detection
	if d.isSourceCode(content) {
		// For source code, only scan for tokens and secrets, skip password detection
		if d.isRandomToken(content) {
			threats = append(threats, SecurityThreat{
				Type:       "token",
				Confidence: 0.75,
				Reason:     "Long alphanumeric string detected in source code (potential token)",
			})
		}
	} else {
		// For non-source code content, apply revised password detection logic
		if d.looksLikePasswordRevised(content) {
			threats = append(threats, SecurityThreat{
				Type:       "password",
				Confidence: 0.85,
				Reason:     "Potential password detected based on character complexity",
			})
		}
	}

	// Check for environment variables with sensitive names
	if d.isSensitiveEnvVar(content) {
		threats = append(threats, SecurityThreat{
			Type:       "env_secret",
			Confidence: 0.8,
			Reason:     "Environment variable with sensitive name",
		})
	}

	// Check for long random strings (potential tokens) - only if not source code
	if !d.isSourceCode(content) && d.isRandomToken(content) {
		threats = append(threats, SecurityThreat{
			Type:       "token",
			Confidence: 0.75,
			Reason:     "Long alphanumeric string detected (potential token)",
		})
	}

	// Check for URLs with query parameters (potential tokens in URLs)
	if d.isUnsafeURL(content) {
		threats = append(threats, SecurityThreat{
			Type:       "url_with_params",
			Confidence: 0.65,
			Reason:     "URL with query parameters detected (may contain tokens)",
		})
	}

	return threats
}

// looksLikePasswordRevised implements the revised detection algorithm:
// 1. Single word: 3 of 4 character types = unsafe
// 2. Multi-word: only if a single word has 4 of 4 character types = unsafe
// 3. Excludes URLs without parameters
func (d *SecurityDetector) looksLikePasswordRevised(content string) bool {
	// First check if this is a simple URL without parameters (safe)
	if d.isSimpleURL(content) {
		return false
	}
	
	// Split content into words
	words := strings.Fields(content)
	if len(words) == 0 {
		return false
	}
	
	// Single word: flag if it has 3 of 4 character types
	if len(words) == 1 {
		return d.wordHasCharacterTypes(words[0], 3)
	}
	
	// Multi-word: only flag if any single word has ALL 4 character types
	for _, word := range words {
		if d.wordHasCharacterTypes(word, 4) {
			return true
		}
	}
	
	return false
}

// wordHasCharacterTypes checks if a word has at least the specified number of character types
func (d *SecurityDetector) wordHasCharacterTypes(word string, minTypes int) bool {
	// Length must be between 8-40 characters (typical password range)
	if len(word) < 8 || len(word) > 40 {
		return false
	}

	// Exclude URLs (common false positive)
	if d.isURL(word) {
		return false
	}
	
	// Exclude file paths (common false positive)
	if d.isFilePath(word) {
		return false
	}
	
	// Exclude common code patterns (hex values, version numbers, etc.)
	if d.isCodePattern(word) {
		return false
	}

	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false

	for _, char := range word {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasDigit = true
		case strings.ContainsRune("!@#$%^&*()_+-=[]{}|;:,.<>?", char):
			hasSpecial = true
		}
	}

	// Count how many character types are present
	charTypes := 0
	if hasUpper {
		charTypes++
	}
	if hasLower {
		charTypes++
	}
	if hasDigit {
		charTypes++
	}
	if hasSpecial {
		charTypes++
	}

	// Check if we have at least the required number of character types
	return charTypes >= minTypes
}

// isSimpleURL checks if content is a simple URL without query parameters (should not be flagged)
func (d *SecurityDetector) isSimpleURL(content string) bool {
	// Must be single word - no spaces, tabs, or newlines
	if strings.ContainsAny(content, " \t\n\r") {
		return false
	}
	
	// Must start with a valid URL scheme
	if !strings.HasPrefix(content, "http://") && !strings.HasPrefix(content, "https://") {
		return false
	}
	
	// Must NOT contain query parameters (? and &) to be considered "simple"
	if strings.Contains(content, "?") {
		return false
	}
	
	return true
}

// isSourceCode checks if content appears to be source code
func (d *SecurityDetector) isSourceCode(content string) bool {
	// Check for common source code patterns
	codeIndicators := []string{
		"function ", "class ", "import ", "export ", "const ", "let ", "var ",
		"def ", "if ", "else ", "for ", "while ", "return ", "package ",
		"#include", "import ", "from ", "struct ", "interface ", "type ",
		"public ", "private ", "protected ", "static ", "void ", "int ",
		"string ", "bool ", "float ", "double ", "char ", "println(",
		"console.log(", "System.out", "print(", "fmt.Print", "echo ",
		"SELECT ", "INSERT ", "UPDATE ", "DELETE ", "CREATE ", "ALTER ",
		"<?php", "?>", "<!DOCTYPE", "<html", "<head", "<body", "<div",
		"#!/bin/", "#!/usr/bin/", "use strict", "use warnings",
	}
	
	// Convert to lowercase for case-insensitive matching
	lowerContent := strings.ToLower(content)
	
	// Count how many code indicators are present
	indicators := 0
	for _, indicator := range codeIndicators {
		if strings.Contains(lowerContent, strings.ToLower(indicator)) {
			indicators++
		}
	}
	
	// Also check for common code patterns
	if strings.Contains(content, "()") || strings.Contains(content, "{}") ||
		strings.Contains(content, "[];") || strings.Contains(content, "//") ||
		strings.Contains(content, "/*") || strings.Contains(content, "*/") ||
		strings.Contains(content, "<!--") || strings.Contains(content, "-->") {
		indicators++
	}
	
	// If we have multiple indicators or the content has typical code structure, consider it source code
	return indicators >= 2 || (indicators >= 1 && (strings.Contains(content, "\n") || len(content) > 100))
}

// isURL checks if a word is a URL
func (d *SecurityDetector) isURL(word string) bool {
	// Simple URL detection
	return strings.HasPrefix(word, "http://") || 
		   strings.HasPrefix(word, "https://") || 
		   strings.HasPrefix(word, "ftp://") ||
		   strings.HasPrefix(word, "ftps://") ||
		   (strings.Contains(word, ".") && (strings.Contains(word, "/") || strings.Contains(word, "?")))
}

// isFilePath checks if a word looks like a file path
func (d *SecurityDetector) isFilePath(word string) bool {
	// Common file path patterns
	return strings.HasPrefix(word, "/") ||
		   strings.HasPrefix(word, "./") ||
		   strings.HasPrefix(word, "../") ||
		   strings.HasPrefix(word, "~/") ||
		   (strings.Contains(word, "/") && (strings.Contains(word, ".") || strings.Contains(word, "bin") || strings.Contains(word, "usr"))) ||
		   strings.Contains(word, "\\") // Windows paths
}

// isCodePattern checks if a word looks like a common code pattern that shouldn't be flagged
func (d *SecurityDetector) isCodePattern(word string) bool {
	// Git commit hashes (40 hex characters)
	if len(word) == 40 && d.isHexString(word) {
		return true
	}
	
	// SHA256 hashes (64 hex characters)
	if len(word) == 64 && d.isHexString(word) {
		return true
	}
	
	// Version numbers with dots and numbers
	if strings.Count(word, ".") >= 2 && d.hasDigits(word) {
		return true
	}
	
	// Package names (e.g., com.example.package)
	if strings.Count(word, ".") >= 2 && strings.ToLower(word) == word {
		return true
	}
	
	// UUIDs
	if len(word) == 36 && strings.Count(word, "-") == 4 {
		return true
	}
	
	return false
}

// isHexString checks if a string contains only hexadecimal characters
func (d *SecurityDetector) isHexString(s string) bool {
	for _, char := range s {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return false
		}
	}
	return true
}

// hasDigits checks if a string contains any digits
func (d *SecurityDetector) hasDigits(s string) bool {
	for _, char := range s {
		if char >= '0' && char <= '9' {
			return true
		}
	}
	return false
}


// isSensitiveEnvVar checks if content looks like a sensitive environment variable
func (d *SecurityDetector) isSensitiveEnvVar(content string) bool {
	sensitiveEnvNames := []string{
		"PASSWORD", "PASSWD", "PWD", "SECRET", "KEY", "TOKEN", "API_KEY",
		"ACCESS_KEY", "PRIVATE_KEY", "CLIENT_SECRET", "AUTH_TOKEN",
		"DATABASE_URL", "DB_PASSWORD", "REDIS_PASSWORD", "JWT_SECRET",
	}

	upper := strings.ToUpper(content)
	for _, name := range sensitiveEnvNames {
		if strings.Contains(upper, name+"=") {
			return true
		}
	}

	return false
}

// isRandomToken checks if content looks like a random token
// More conservative approach to reduce false positives
func (d *SecurityDetector) isRandomToken(content string) bool {
	// Must be single word - no spaces, tabs, or newlines
	if strings.ContainsAny(content, " \t\n\r") {
		return false
	}
	
	// Tokens are typically longer than passwords but not too long
	if len(content) < 32 || len(content) > 256 {
		return false
	}

	// Check if it's mostly alphanumeric with some allowed symbols
	alphanumCount := 0
	symbolCount := 0
	for _, char := range content {
		if (char >= 'A' && char <= 'Z') ||
			(char >= 'a' && char <= 'z') ||
			(char >= '0' && char <= '9') {
			alphanumCount++
		} else if char == '_' || char == '-' || char == '.' {
			symbolCount++
		}
	}

	// Must be mostly alphanumeric with minimal punctuation
	totalValidChars := alphanumCount + symbolCount
	if totalValidChars != len(content) {
		return false // Contains invalid characters
	}
	
	// Should be at least 90% alphanumeric
	ratio := float64(alphanumCount) / float64(len(content))
	return ratio > 0.9
}

// isUnsafeURL checks if content is a URL with query parameters that may contain tokens
// Only URLs with query parameters (containing ? and &) are considered unsafe
func (d *SecurityDetector) isUnsafeURL(content string) bool {
	// Must be single word - no spaces, tabs, or newlines
	if strings.ContainsAny(content, " \t\n\r") {
		return false
	}
	
	// Must contain query parameters (? followed by parameters)
	if !strings.Contains(content, "?") || !strings.Contains(content, "&") {
		return false
	}
	
	// Must start with a valid URL scheme
	if !strings.HasPrefix(content, "http://") && !strings.HasPrefix(content, "https://") {
		return false
	}
	
	// Check for common patterns that indicate tokens in URLs
	// Look for parameter names that suggest sensitive data
	lowerContent := strings.ToLower(content)
	suspiciousParams := []string{
		"token=", "access_token=", "api_key=", "apikey=", "key=",
		"secret=", "password=", "auth=", "authorization=", "bearer=",
		"client_secret=", "refresh_token=", "session=", "sid=",
	}
	
	for _, param := range suspiciousParams {
		if strings.Contains(lowerContent, param) {
			return true
		}
	}
	
	// If URL has query parameters but no obviously sensitive parameter names,
	// still flag it as potentially unsafe since tokens might be in non-standard parameter names
	return true
}

// CreateHash creates a SHA256 hash of the content for tracking purposes
func CreateHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// IsHighRiskThreat determines if a threat is high risk and should be flagged
func IsHighRiskThreat(threats []SecurityThreat) bool {
	for _, threat := range threats {
		if threat.Confidence > 0.8 &&
			(threat.Type == "jwt" || threat.Type == "api_key" ||
				threat.Type == "ssh_key" || threat.Type == "private_key") {
			return true
		}
	}
	return false
}

// GetHighestThreat returns the threat with the highest confidence level
func GetHighestThreat(threats []SecurityThreat) *SecurityThreat {
	if len(threats) == 0 {
		return nil
	}

	highest := &threats[0]
	for i := 1; i < len(threats); i++ {
		if threats[i].Confidence > highest.Confidence {
			highest = &threats[i]
		}
	}

	return highest
}
