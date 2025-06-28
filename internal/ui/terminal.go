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

package ui

import (
	"os"
	"strings"
)

// TerminalCapabilities holds information about what the terminal can display
type TerminalCapabilities struct {
	SupportsUnicode bool
	SupportsEmoji   bool
	SupportsColor   bool
}

// DetectTerminalCapabilities analyzes the current terminal's capabilities
func DetectTerminalCapabilities() TerminalCapabilities {
	caps := TerminalCapabilities{
		SupportsUnicode: true,  // Default assumption
		SupportsEmoji:   false, // Conservative default
		SupportsColor:   true,  // Most terminals support basic colors
	}

	// Check environment variables for terminal capabilities
	term := strings.ToLower(os.Getenv("TERM"))
	termProgram := strings.ToLower(os.Getenv("TERM_PROGRAM"))

	// Detect Unicode/Emoji support based on terminal type
	caps.SupportsUnicode = detectUnicodeSupport(term, termProgram)
	caps.SupportsEmoji = detectEmojiSupport(term, termProgram)
	caps.SupportsColor = detectColorSupport(term)

	return caps
}

// detectUnicodeSupport checks if terminal supports Unicode characters
func detectUnicodeSupport(term, termProgram string) bool {
	// Terminals known to support Unicode well
	unicodeTerminals := []string{
		"xterm-256color", "screen-256color", "tmux-256color",
		"alacritty", "kitty", "iterm2", "vscode",
		"gnome-terminal", "konsole", "terminology",
	}

	for _, supportedTerm := range unicodeTerminals {
		if strings.Contains(term, supportedTerm) || strings.Contains(termProgram, supportedTerm) {
			return true
		}
	}

	// Check for UTF-8 locale
	lang := os.Getenv("LANG")
	lcAll := os.Getenv("LC_ALL")
	if strings.Contains(strings.ToUpper(lang), "UTF-8") ||
		strings.Contains(strings.ToUpper(lcAll), "UTF-8") {
		return true
	}

	// Conservative fallback for unknown terminals
	if term == "" || strings.Contains(term, "dumb") || strings.Contains(term, "linux") {
		return false
	}

	return true // Default to assuming Unicode support
}

// detectEmojiSupport checks if terminal supports emoji rendering
func detectEmojiSupport(term, termProgram string) bool {
	// Modern terminals with good emoji support
	emojiTerminals := []string{
		"kitty", "alacritty", "iterm2", "terminal.app",
		"gnome-terminal", "konsole", "terminology",
		"hyper", "wezterm", "ghostty",
	}

	for _, emojiTerm := range emojiTerminals {
		if strings.Contains(termProgram, emojiTerm) || strings.Contains(term, emojiTerm) {
			return true
		}
	}

	// VS Code integrated terminal
	if strings.Contains(termProgram, "vscode") {
		return true
	}

	// Check for modern terminal indicators
	if strings.Contains(term, "256color") || strings.Contains(term, "truecolor") {
		// Modern color terminals often support emoji
		return true
	}

	// Conservative default - many terminals don't render emoji well
	return false
}

// detectColorSupport checks if terminal supports ANSI colors
func detectColorSupport(term string) bool {
	// Terminals that don't support color
	noColorTerminals := []string{"dumb", "unknown"}

	for _, noColorTerm := range noColorTerminals {
		if strings.Contains(term, noColorTerm) {
			return false
		}
	}

	// Check for explicit color support indicators
	if strings.Contains(term, "color") ||
		strings.Contains(term, "xterm") ||
		strings.Contains(term, "screen") ||
		strings.Contains(term, "tmux") {
		return true
	}

	// Check NO_COLOR environment variable
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Default to supporting color
	return true
}

// SecurityIndicators holds the visual indicators for security threats
type SecurityIndicators struct {
	HighRisk   string
	MediumRisk string
	Clean      string
}

// GetSecurityIndicators returns appropriate security indicators based on terminal capabilities
func GetSecurityIndicators(caps TerminalCapabilities) SecurityIndicators {
	// Use simple ASCII characters for all terminals
	return SecurityIndicators{
		HighRisk:   "[!]",
		MediumRisk: "[?]",
		Clean:      "",
	}
}

// GetColorizedSecurityIndicator returns a security indicator with appropriate coloring
func GetColorizedSecurityIndicator(indicator string, riskLevel string, caps TerminalCapabilities) string {
	if !caps.SupportsColor {
		return indicator
	}

	// Apply colors based on risk level
	switch riskLevel {
	case "high":
		// Red for high risk indicators
		return "\033[91m" + indicator + "\033[0m" // Bright red
	case "medium":
		// Yellow for medium risk indicators
		return "\033[93m" + indicator + "\033[0m" // Bright yellow
	default:
		return indicator
	}
}

// SecurityIconHelper provides easy access to security indicators
type SecurityIconHelper struct {
	caps       TerminalCapabilities
	indicators SecurityIndicators
}

// NewSecurityIconHelper creates a new security icon helper
func NewSecurityIconHelper() *SecurityIconHelper {
	caps := DetectTerminalCapabilities()
	indicators := GetSecurityIndicators(caps)

	return &SecurityIconHelper{
		caps:       caps,
		indicators: indicators,
	}
}

// GetHighRiskIcon returns the high-risk security indicator
func (s *SecurityIconHelper) GetHighRiskIcon() string {
	return GetColorizedSecurityIndicator(s.indicators.HighRisk, "high", s.caps)
}

// GetMediumRiskIcon returns the medium-risk security indicator
func (s *SecurityIconHelper) GetMediumRiskIcon() string {
	return GetColorizedSecurityIndicator(s.indicators.MediumRisk, "medium", s.caps)
}

// GetCapabilities returns the detected terminal capabilities
func (s *SecurityIconHelper) GetCapabilities() TerminalCapabilities {
	return s.caps
}

// GetIndicatorDescription returns a human-readable description of the indicators
func (s *SecurityIconHelper) GetIndicatorDescription() string {
	return "Icons: [!]=high risk [?]=medium risk"
}
