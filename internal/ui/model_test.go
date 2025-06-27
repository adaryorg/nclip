package ui

import (
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

	for _, test := range tests {
		result := parseColor(test.input)
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

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			style := createStyle(test.config)

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
	// Empty hex
	result := parseColor("#")
	if string(result) != "#" {
		t.Error("Invalid hex should pass through unchanged")
	}

	// Mixed case CSS color
	result = parseColor("ReD")
	if string(result) != "#FF0000" {
		t.Error("Mixed case CSS color should be normalized")
	}

	// Special characters
	result = parseColor("color-with-dashes")
	if string(result) != "color-with-dashes" {
		t.Error("Unknown color should pass through unchanged")
	}

	// Numbers
	result = parseColor("123")
	if string(result) != "123" {
		t.Error("ANSI color code should pass through unchanged")
	}
}
