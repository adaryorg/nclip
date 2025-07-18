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
	"fmt"
	"os"
	"strings"
)

// TerminalCapabilities holds information about what the terminal can display
type TerminalCapabilities struct {
	SupportsUnicode bool
	SupportsColor   bool
}

// DetectTerminalCapabilities analyzes the current terminal's capabilities
func DetectTerminalCapabilities(basicTerminal bool) TerminalCapabilities {
	// If basic terminal mode is requested, disable all advanced features
	if basicTerminal {
		return TerminalCapabilities{
			SupportsUnicode: false,
			SupportsColor:   false,
		}
	}

	// Check if we're in a TTY (no graphical environment)
	if isTTY() {
		return TerminalCapabilities{
			SupportsUnicode: false,
			SupportsColor:   true, // TTY supports basic colors
		}
	}

	// Default: assume modern terminal with Unicode support
	return TerminalCapabilities{
		SupportsUnicode: true,
		SupportsColor:   true,
	}
}

// isTTY checks if we're running in a TTY (no graphical environment)
func isTTY() bool {
	// Check XDG_SESSION_TYPE environment variable
	sessionType := strings.ToLower(os.Getenv("XDG_SESSION_TYPE"))
	
	// If XDG_SESSION_TYPE is "tty", we're in a TTY
	if sessionType == "tty" {
		return true
	}
	
	// If XDG_SESSION_TYPE is present and not "tty", assume graphical
	if sessionType != "" {
		return false
	}
	
	// If XDG_SESSION_TYPE is not set, check for other TTY indicators
	term := strings.ToLower(os.Getenv("TERM"))
	
	// Check for known TTY terminal types
	if term == "linux" || term == "console" || strings.HasPrefix(term, "tty") {
		return true
	}
	
	// If no clear indicators, assume we're in a graphical environment
	return false
}

// SecurityIndicators holds the visual indicators for security threats
type SecurityIndicators struct {
	HighRisk   string
	MediumRisk string
	Clean      string
	Safe       string
}

// GetSecurityIndicators returns appropriate security indicators based on terminal capabilities
func GetSecurityIndicators(caps TerminalCapabilities) SecurityIndicators {
	if caps.SupportsUnicode {
		// Use Unicode symbols that are more widely supported
		return SecurityIndicators{
			HighRisk:   "⚠", // Warning sign (U+26A0)
			MediumRisk: "⚡", // High voltage sign (U+26A1)
			Clean:      "",
			Safe:       "✓", // Check mark (U+2713)
		}
	}
	
	// Fallback to simple ASCII characters
	return SecurityIndicators{
		HighRisk:   "[h]",
		MediumRisk: "[m]",
		Clean:      "",
		Safe:       "[s]",
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
	case "safe":
		// Green for safe indicators
		return "\033[92m" + indicator + "\033[0m" // Bright green
	default:
		return indicator
	}
}

// GetThemedSecurityIndicator returns a security indicator with themed coloring
func GetThemedSecurityIndicator(indicator string, riskLevel string, caps TerminalCapabilities, mainStyles MainViewStyles) string {
	if !caps.SupportsColor {
		return indicator
	}

	// Apply themed colors based on risk level
	switch riskLevel {
	case "high":
		return mainStyles.HighRiskIndicator.Render(indicator)
	case "medium":
		return mainStyles.MediumRiskIndicator.Render(indicator)
	case "safe":
		return mainStyles.PinnedIndicator.Render(indicator) // Use pinned indicator color for safe items
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
func NewSecurityIconHelper(basicTerminal bool) *SecurityIconHelper {
	caps := DetectTerminalCapabilities(basicTerminal)
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

// GetSafeIcon returns the safe security indicator
func (s *SecurityIconHelper) GetSafeIcon() string {
	return GetColorizedSecurityIndicator(s.indicators.Safe, "safe", s.caps)
}

// GetThemedHighRiskIcon returns the high-risk security indicator with themed colors
func (s *SecurityIconHelper) GetThemedHighRiskIcon(mainStyles MainViewStyles) string {
	return GetThemedSecurityIndicator(s.indicators.HighRisk, "high", s.caps, mainStyles)
}

// GetThemedMediumRiskIcon returns the medium-risk security indicator with themed colors
func (s *SecurityIconHelper) GetThemedMediumRiskIcon(mainStyles MainViewStyles) string {
	return GetThemedSecurityIndicator(s.indicators.MediumRisk, "medium", s.caps, mainStyles)
}

// GetThemedSafeIcon returns the safe security indicator with themed colors
func (s *SecurityIconHelper) GetThemedSafeIcon(mainStyles MainViewStyles) string {
	return GetThemedSecurityIndicator(s.indicators.Safe, "safe", s.caps, mainStyles)
}

// GetCapabilities returns the detected terminal capabilities
func (s *SecurityIconHelper) GetCapabilities() TerminalCapabilities {
	return s.caps
}

// GetIndicatorDescription returns a human-readable description of the indicators
func (s *SecurityIconHelper) GetIndicatorDescription() string {
	if s.caps.SupportsUnicode {
		return "Icons: ⚠=high risk ⚡=medium risk ✓=safe"
	}
	return "Icons: [h]=high risk [m]=medium risk [s]=safe"
}

// PinIndicators holds the visual indicators for pinned items
type PinIndicators struct {
	PinFormat string // Format string with %d placeholder for pin number
}

// GetPinIndicators returns appropriate pin indicators based on terminal capabilities
func GetPinIndicators(caps TerminalCapabilities) PinIndicators {
	if caps.SupportsUnicode {
		// Use pin icon with number
		return PinIndicators{
			PinFormat: " %d", // Pin icon (U+EBA0) with number
		}
	}
	
	// Fallback to simple ASCII format
	return PinIndicators{
		PinFormat: "[%d]",
	}
}

// PinIconHelper provides easy access to pin indicators
type PinIconHelper struct {
	caps       TerminalCapabilities
	indicators PinIndicators
}

// NewPinIconHelper creates a new pin icon helper
func NewPinIconHelper(basicTerminal bool) *PinIconHelper {
	caps := DetectTerminalCapabilities(basicTerminal)
	indicators := GetPinIndicators(caps)

	return &PinIconHelper{
		caps:       caps,
		indicators: indicators,
	}
}

// GetPinIcon returns the formatted pin icon for a given pin number
func (p *PinIconHelper) GetPinIcon(pinNumber int) string {
	return fmt.Sprintf(p.indicators.PinFormat, pinNumber)
}

// GetColorizedPinIcon returns a pin icon with appropriate coloring
func (p *PinIconHelper) GetColorizedPinIcon(pinNumber int) string {
	icon := p.GetPinIcon(pinNumber)
	
	if !p.caps.SupportsColor {
		return icon
	}
	
	// Use bright blue for pin icons
	return "\033[94m" + icon + "\033[0m"
}

// GetThemedPinIcon returns a pin icon with themed coloring
func (p *PinIconHelper) GetThemedPinIcon(pinNumber int, mainStyles MainViewStyles) string {
	icon := p.GetPinIcon(pinNumber)
	
	if !p.caps.SupportsColor {
		return icon
	}
	
	// Use themed pin indicator color
	return mainStyles.PinnedIndicator.Render(icon)
}
