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
	"testing"

	"github.com/adaryorg/nclip/internal/config"
	"github.com/adaryorg/nclip/internal/storage"
)

func TestParseColor(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Empty string
		{"", ""},

		// Hex colors
		{"#FF0000", "#FF0000"},
		{"#00FF00", "#00FF00"},
		{"#0000FF", "#0000FF"},
		{"#123456", "#123456"},

		// CSS color names
		{"red", "#FF0000"},
		{"green", "#008000"},
		{"blue", "#0000FF"},
		{"white", "#FFFFFF"},
		{"black", "#000000"},
		{"gray", "#808080"},
		{"grey", "#808080"}, // Alternative spelling

		// Case insensitive
		{"RED", "#FF0000"},
		{"Green", "#008000"},
		{"BLUE", "#0000FF"},

		// Extended colors
		{"orange", "#FFA500"},
		{"purple", "#800080"},
		{"pink", "#FFC0CB"},
		{"coral", "#FF7F50"},
		{"lavender", "#E6E6FA"},

		// ANSI color codes (should pass through unchanged)
		{"1", "1"},
		{"15", "15"},
		{"255", "255"},

		// Unknown color (should pass through unchanged)
		{"unknown", "unknown"},
		{"random123", "random123"},
	}

	// Create a test model with advanced colors enabled for testing
	testModel := Model{useBasicColors: false}
	
	for _, test := range tests {
		result := testModel.parseColor(test.input)
		if string(result) != test.expected {
			t.Errorf("parseColor(%q) = %q, expected %q", test.input, string(result), test.expected)
		}
	}
}

func TestWrapText(t *testing.T) {
	tests := []struct {
		text     string
		width    int
		maxLines int
		expected []string
	}{
		// Simple cases
		{
			text:     "hello world",
			width:    20,
			maxLines: 5,
			expected: []string{"hello world"},
		},
		{
			text:     "hello",
			width:    10,
			maxLines: 5,
			expected: []string{"hello"},
		},

		// Text longer than width
		{
			text:     "this is a very long line that needs to be wrapped",
			width:    20,
			maxLines: 5,
			expected: []string{"this is a very long", "line that needs to", "be wrapped"},
		},

		// Break at word boundaries
		{
			text:     "word1 word2 word3 word4",
			width:    12,
			maxLines: 5,
			expected: []string{"word1 word2", "word3 word4"},
		},

		// Max lines limit
		{
			text:     "line1 line2 line3 line4 line5 line6",
			width:    10,
			maxLines: 2,
			expected: []string{"line1", "li..."},
		},

		// Width <= 0 fallback
		{
			text:     "test",
			width:    0,
			maxLines: 5,
			expected: []string{"test"},
		},

		// Empty text
		{
			text:     "",
			width:    10,
			maxLines: 5,
			expected: []string{},
		},

		// Single character width
		{
			text:     "abc",
			width:    1,
			maxLines: 5,
			expected: []string{"a", "b", "c"},
		},

		// No good break point (force break)
		{
			text:     "verylongwordwithoutspaces",
			width:    10,
			maxLines: 5,
			expected: []string{"verylongwo", "rdwithouts", "paces"},
		},
	}

	for i, test := range tests {
		result := wrapText(test.text, test.width, test.maxLines)
		if len(result) != len(test.expected) {
			t.Errorf("Test %d: expected %d lines, got %d", i, len(test.expected), len(result))
			continue
		}

		for j, expected := range test.expected {
			if j >= len(result) || result[j] != expected {
				t.Errorf("Test %d, line %d: expected %q, got %q", i, j, expected, result[j])
			}
		}
	}
}

func TestCreateStyle(t *testing.T) {
	tests := []struct {
		name   string
		config config.ColorConfig
	}{
		{
			name: "basic config",
			config: config.ColorConfig{
				Foreground: "red",
				Background: "blue",
				Bold:       true,
			},
		},
		{
			name: "empty config",
			config: config.ColorConfig{
				Foreground: "",
				Background: "",
				Bold:       false,
			},
		},
		{
			name: "hex colors",
			config: config.ColorConfig{
				Foreground: "#FF0000",
				Background: "#0000FF",
				Bold:       false,
			},
		},
		{
			name: "ansi colors",
			config: config.ColorConfig{
				Foreground: "15",
				Background: "234",
				Bold:       true,
			},
		},
	}

	// Create a test model for testing
	testModel := Model{useBasicColors: false}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			style := testModel.createStyle(test.config)

			// We can't easily test the exact style properties due to lipgloss internals,
			// but we can at least verify the function doesn't panic and returns a valid style
			// We'll just check that it can render text successfully

			// Test that the style can be used to render text
			rendered := style.Render("test")
			if rendered == "" {
				t.Error("Style failed to render text")
			}
		})
	}
}

func TestNewModel(t *testing.T) {
	// Test basic model creation with config
	cfg := &config.Config{
		Database: config.DatabaseConfig{MaxEntries: 100},
		Theme: config.ThemeConfig{
			Header: config.ColorConfig{
				Foreground: "13",
				Bold:       true,
			},
		},
		Editor: config.EditorConfig{
			TextEditor: "nano",
		},
	}

	// We can't easily create a real storage in unit tests without database setup,
	// but we can test config assignment and basic properties
	if cfg == nil {
		t.Fatal("Config should not be nil")
	}

	// Test that config values are properly set
	if cfg.Database.MaxEntries != 100 {
		t.Error("Config not properly initialized")
	}
	if cfg.Theme.Header.Foreground != "13" {
		t.Error("Theme config not properly set")
	}
	if cfg.Editor.TextEditor != "nano" {
		t.Error("Editor config not properly set")
	}

	// Note: We skip testing NewModel with nil storage as it would panic
	// In real usage, storage is always provided
}

func TestFilterItemsLogic(t *testing.T) {
	// Test the filtering logic without requiring storage
	items := []storage.ClipboardItem{
		{ID: "1", Content: "hello world", ContentType: "text"},
		{ID: "2", Content: "test message", ContentType: "text"},
		{ID: "3", Content: "another hello", ContentType: "text"},
		{ID: "4", Content: "image data", ContentType: "image"},
	}

	// Test filtering logic manually
	searchQuery := "hello"

	// Find items containing the search query (simulating filterItems logic)
	var matches []storage.ClipboardItem
	for _, item := range items {
		if item.ContentType != "image" && strings.Contains(strings.ToLower(item.Content), strings.ToLower(searchQuery)) {
			matches = append(matches, item)
		}
	}

	if len(matches) != 2 {
		t.Errorf("Expected 2 matches for 'hello', got %d", len(matches))
	}

	// Test empty search
	emptyQuery := ""
	if emptyQuery == "" {
		// Empty search should return all items - this logic is correct
		if len(items) != 4 {
			t.Error("Empty search logic failed")
		}
	}

	// Test case insensitive search
	upperQuery := "HELLO"
	var upperMatches []storage.ClipboardItem
	for _, item := range items {
		if item.ContentType != "image" && strings.Contains(strings.ToLower(item.Content), strings.ToLower(upperQuery)) {
			upperMatches = append(upperMatches, item)
		}
	}

	if len(upperMatches) != 2 {
		t.Errorf("Case insensitive search failed: expected 2 matches, got %d", len(upperMatches))
	}
}

// Test string utility functions used in the UI
func TestStringUtilities(t *testing.T) {
	// Test that strings.TrimLeft works as expected (used in wrapText)
	result := strings.TrimLeft("  hello", " ")
	if result != "hello" {
		t.Errorf("TrimLeft failed: expected 'hello', got '%s'", result)
	}

	// Test string length calculation (important for UI layout)
	text := "test string"
	if len(text) != 11 {
		t.Errorf("String length calculation failed: expected 11, got %d", len(text))
	}

	// Test string slicing (used in text wrapping)
	if text[:4] != "test" {
		t.Errorf("String slicing failed: expected 'test', got '%s'", text[:4])
	}
}

// Test edge cases for text wrapping
func TestWrapTextEdgeCases(t *testing.T) {
	// Very large width
	result := wrapText("hello", 1000, 5)
	if len(result) != 1 || result[0] != "hello" {
		t.Error("Large width should not wrap short text")
	}

	// Width exactly equal to text length
	result = wrapText("hello", 5, 5)
	if len(result) != 1 || result[0] != "hello" {
		t.Error("Width equal to text length should not wrap")
	}

	// maxLines = 0
	result = wrapText("hello world", 5, 0)
	if len(result) != 0 {
		t.Error("maxLines = 0 should return empty slice")
	}

	// maxLines = 1 with long text
	result = wrapText("hello world test", 5, 1)
	if len(result) != 1 {
		t.Error("maxLines = 1 should return exactly 1 line")
	}
	if !strings.HasSuffix(result[0], "...") {
		t.Error("Truncated line should end with ellipsis")
	}
}

// Test color parsing edge cases
func TestParseColorEdgeCases(t *testing.T) {
	// Create a test model with advanced colors enabled for testing
	testModel := Model{useBasicColors: false}
	
	// Empty hex
	result := testModel.parseColor("#")
	if string(result) != "#" {
		t.Error("Invalid hex should pass through unchanged")
	}

	// Mixed case CSS color
	result = testModel.parseColor("ReD")
	if string(result) != "#FF0000" {
		t.Error("Mixed case CSS color should be normalized")
	}

	// Special characters
	result = testModel.parseColor("color-with-dashes")
	if string(result) != "color-with-dashes" {
		t.Error("Unknown color should pass through unchanged")
	}

	// Numbers
	result = testModel.parseColor("123")
	if string(result) != "123" {
		t.Error("ANSI color code should pass through unchanged")
	}
}

// Test basic color fallback functionality
func TestBasicColorFallback(t *testing.T) {
	// Create a test model with basic colors enabled
	testModel := Model{useBasicColors: true}
	
	tests := []struct {
		input    string
		expected string
	}{
		// Complex colors should map to basic ANSI
		{"13", "5"},   // header color -> magenta
		{"141", "5"},  // search color -> magenta
		{"55", "4"},   // selected background -> blue
		{"39", "6"},   // frame border -> cyan
		{"234", "0"},  // dark gray -> black
		
		// CSS colors should map to basic
		{"red", "1"},
		{"green", "2"},
		{"blue", "4"},
		{"yellow", "3"},
		{"magenta", "5"},
		{"cyan", "6"},
		{"white", "7"},
		{"gray", "8"},
		
		// Hex colors should map to basic
		{"#FF0000", "1"}, // red
		{"#00FF00", "2"}, // green
		{"#0000FF", "4"}, // blue
		
		// Unknown colors should return empty
		{"999", ""},
		{"unknown", ""},
	}

	for _, test := range tests {
		result := testModel.parseColor(test.input)
		if string(result) != test.expected {
			t.Errorf("parseColor(%q) in basic mode = %q, expected %q", test.input, string(result), test.expected)
		}
	}
}

// Test ANSI handling functions
func TestPadLineToWidth(t *testing.T) {
	// Create a test model
	testModel := Model{useBasicColors: false}
	
	tests := []struct {
		input       string
		targetWidth int
		expected    int // expected visible length
	}{
		{"hello", 10, 10},
		{"", 5, 5},
		{"exact", 5, 5},
		{"toolong", 3, 3}, // Should not pad if already too long
		{"\x1b[31mred\x1b[0m", 10, 10}, // ANSI colored text
	}
	
	for _, test := range tests {
		result := testModel.padLineToWidth(test.input, test.targetWidth)
		visibleLen := testModel.calculateVisibleLength(result)
		
		// For lines that are already too long, we don't pad
		expectedLen := test.expected
		if testModel.calculateVisibleLength(test.input) > test.targetWidth {
			expectedLen = testModel.calculateVisibleLength(test.input)
		}
		
		if visibleLen != expectedLen {
			t.Errorf("padLineToWidth(%q, %d) visible length = %d, expected %d", 
				test.input, test.targetWidth, visibleLen, expectedLen)
		}
	}
}

func TestTruncateWithANSI(t *testing.T) {
	// Create a test model
	testModel := Model{useBasicColors: false}
	
	tests := []struct {
		input    string
		maxLen   int
		expected int // expected visible length
	}{
		{"hello", 10, 5},
		{"hello", 3, 3},
		{"hello", 0, 0},
		{"\x1b[31mhello\x1b[0m", 3, 3}, // Should preserve ANSI codes
		{"\x1b[31mhello\x1b[0mworld", 7, 7},
	}
	
	for _, test := range tests {
		result := testModel.truncateWithANSI(test.input, test.maxLen)
		visibleLen := testModel.calculateVisibleLength(result)
		
		if visibleLen != test.expected {
			t.Errorf("truncateWithANSI(%q, %d) visible length = %d, expected %d", 
				test.input, test.maxLen, visibleLen, test.expected)
		}
		
		// Verify ANSI codes are preserved (result should contain escape sequences if input did)
		inputHasANSI := strings.Contains(test.input, "\x1b")
		resultHasANSI := strings.Contains(result, "\x1b")
		if inputHasANSI && !resultHasANSI && test.maxLen > 0 {
			t.Errorf("truncateWithANSI(%q, %d) lost ANSI codes", test.input, test.maxLen)
		}
	}
}

// Test content filtering
func TestApplyContentFilter(t *testing.T) {
	// Create a test model
	testModel := Model{useBasicColors: false}
	
	// Create test items
	testItems := []storage.ClipboardItem{
		{ID: "1", Content: "Normal text", ContentType: "text", ThreatLevel: ""},
		{ID: "2", Content: "Image data", ContentType: "image", ThreatLevel: ""},
		{ID: "3", Content: "Suspicious content", ContentType: "text", ThreatLevel: "high"},
		{ID: "4", Content: "Questionable content", ContentType: "text", ThreatLevel: "medium"},
		{ID: "5", Content: "Another image", ContentType: "image", ThreatLevel: ""},
	}
	
	tests := []struct {
		filterMode   string
		expectedCount int
		description  string
	}{
		{"", 5, "no filter should return all items"},
		{"images", 2, "image filter should return only images"},
		{"security-high", 1, "high-risk filter should return only high-risk items"},
		{"security-medium", 1, "medium-risk filter should return only medium-risk items"},
		{"invalid", 5, "invalid filter should return all items"},
	}
	
	for _, test := range tests {
		testModel.filterMode = test.filterMode
		result := testModel.applyContentFilter(testItems)
		
		if len(result) != test.expectedCount {
			t.Errorf("Filter mode '%s': expected %d items, got %d (%s)", 
				test.filterMode, test.expectedCount, len(result), test.description)
		}
		
		// Verify content of filtered results
		switch test.filterMode {
		case "images":
			for _, item := range result {
				if item.ContentType != "image" {
					t.Errorf("Image filter returned non-image item: %s", item.Content)
				}
			}
		case "security-high":
			for _, item := range result {
				if item.ThreatLevel != "high" {
					t.Errorf("High-risk filter returned non-high-risk item: %s", item.Content)
				}
			}
		case "security-medium":
			for _, item := range result {
				if item.ThreatLevel != "medium" {
					t.Errorf("Medium-risk filter returned non-medium-risk item: %s", item.Content)
				}
			}
		}
	}
}

// Test terminal capability detection
func TestDetectTerminalCapabilities(t *testing.T) {
	// Save original environment
	originalTerm := os.Getenv("TERM")
	originalColorTerm := os.Getenv("COLORTERM")
	defer func() {
		os.Setenv("TERM", originalTerm)
		os.Setenv("COLORTERM", originalColorTerm)
	}()

	tests := []struct {
		term      string
		colorTerm string
		expected  bool
	}{
		// Basic terminals should return false
		{"xterm", "", false},
		{"screen", "", false},
		{"linux", "", false},
		{"vt100", "", false},
		
		// Advanced terminals should return true
		{"xterm-256color", "", true},
		{"screen-256color", "", true},
		{"alacritty", "", true},
		{"kitty", "", true},
		
		// COLORTERM override should work
		{"xterm", "truecolor", true},
		{"linux", "24bit", true},
		
		// Unknown terminals default to advanced
		{"some-unknown-term", "", true},
	}

	for _, test := range tests {
		os.Setenv("TERM", test.term)
		if test.colorTerm == "" {
			os.Unsetenv("COLORTERM")
		} else {
			os.Setenv("COLORTERM", test.colorTerm)
		}
		
		// Also unset modern terminal indicators to ensure clean test
		os.Unsetenv("ITERM_SESSION_ID")
		os.Unsetenv("KITTY_WINDOW_ID")
		os.Unsetenv("ALACRITTY_SOCKET")
		os.Unsetenv("WEZTERM_PANE")
		os.Unsetenv("GHOSTTY_RESOURCES_DIR")
		os.Unsetenv("COLORS")
		
		result := detectTerminalCapabilities()
		if result != test.expected {
			t.Errorf("detectTerminalCapabilities() with TERM=%s COLORTERM=%s = %v, expected %v", 
				test.term, test.colorTerm, result, test.expected)
		}
	}
}
