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

	// Check for common password characteristics
	if d.looksLikePassword(content) {
		threats = append(threats, SecurityThreat{
			Type:       "password",
			Confidence: 0.7,
			Reason:     "Content has password-like characteristics",
		})
	}

	// Check for environment variables with sensitive names
	if d.isSensitiveEnvVar(content) {
		threats = append(threats, SecurityThreat{
			Type:       "env_secret",
			Confidence: 0.8,
			Reason:     "Environment variable with sensitive name",
		})
	}

	// Check for long random strings (potential tokens)
	if d.isRandomToken(content) {
		threats = append(threats, SecurityThreat{
			Type:       "token",
			Confidence: 0.6,
			Reason:     "Long random string detected (potential token)",
		})
	}

	return threats
}

// looksLikePassword checks if content has password-like characteristics
func (d *SecurityDetector) looksLikePassword(content string) bool {
	if len(content) < 8 || len(content) > 128 {
		return false
	}

	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false

	for _, char := range content {
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

	// Password-like if it has at least 3 of the 4 character types
	score := 0
	if hasUpper {
		score++
	}
	if hasLower {
		score++
	}
	if hasDigit {
		score++
	}
	if hasSpecial {
		score++
	}

	return score >= 3
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
func (d *SecurityDetector) isRandomToken(content string) bool {
	if len(content) < 20 || len(content) > 500 {
		return false
	}

	// Check if it's mostly alphanumeric with some symbols
	alphanumCount := 0
	for _, char := range content {
		if (char >= 'A' && char <= 'Z') ||
			(char >= 'a' && char <= 'z') ||
			(char >= '0' && char <= '9') ||
			char == '_' || char == '-' || char == '.' {
			alphanumCount++
		}
	}

	// Should be mostly alphanumeric
	ratio := float64(alphanumCount) / float64(len(content))
	return ratio > 0.8
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
