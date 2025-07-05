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

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfig_DefaultValues(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "nclip-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Temporarily override home directory
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	// Load config (should create default)
	config, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test default database values
	if config.Database.MaxEntries != 1000 {
		t.Errorf("Expected MaxEntries to be 1000, got %d", config.Database.MaxEntries)
	}

	// Test default theme values
	if config.Theme.Header.Foreground != "8" {
		t.Errorf("Expected Header.Foreground to be '8', got '%s'", config.Theme.Header.Foreground)
	}
	if !config.Theme.Header.Bold {
		t.Error("Expected Header.Bold to be true")
	}

	if config.Theme.Status.Foreground != "8" {
		t.Errorf("Expected Status.Foreground to be '8', got '%s'", config.Theme.Status.Foreground)
	}

	if config.Theme.Search.Foreground != "141" {
		t.Errorf("Expected Search.Foreground to be '141', got '%s'", config.Theme.Search.Foreground)
	}
	if !config.Theme.Search.Bold {
		t.Error("Expected Search.Bold to be true")
	}

	if config.Theme.Warning.Foreground != "9" {
		t.Errorf("Expected Warning.Foreground to be '9', got '%s'", config.Theme.Warning.Foreground)
	}
	if !config.Theme.Warning.Bold {
		t.Error("Expected Warning.Bold to be true")
	}

	if config.Theme.Selected.Foreground != "15" {
		t.Errorf("Expected Selected.Foreground to be '15', got '%s'", config.Theme.Selected.Foreground)
	}
	if config.Theme.Selected.Background != "55" {
		t.Errorf("Expected Selected.Background to be '55' for visibility, got '%s'", config.Theme.Selected.Background)
	}

	if config.Theme.AlternateBackground.Background != "" {
		t.Errorf("Expected AlternateBackground.Background to be empty for syntax highlighting compatibility, got '%s'", config.Theme.AlternateBackground.Background)
	}

	// Test default frame values
	if config.Theme.Frame.Border.Foreground != "8" {
		t.Errorf("Expected Frame.Border.Foreground to be '8', got '%s'", config.Theme.Frame.Border.Foreground)
	}
	if config.Theme.Frame.Background.Background != "" {
		t.Errorf("Expected Frame.Background.Background to be empty for syntax highlighting compatibility, got '%s'", config.Theme.Frame.Background.Background)
	}

	// Test default editor values
	if config.Editor.TextEditor != "nano" {
		t.Errorf("Expected TextEditor to be 'nano', got '%s'", config.Editor.TextEditor)
	}
	if config.Editor.ImageEditor != "gimp" {
		t.Errorf("Expected ImageEditor to be 'gimp', got '%s'", config.Editor.ImageEditor)
	}
	if config.Editor.ImageViewer != "loupe" {
		t.Errorf("Expected ImageViewer to be 'loupe', got '%s'", config.Editor.ImageViewer)
	}
}

func TestConfig_CustomValues(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "nclip-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create config directory
	configDir := filepath.Join(tmpDir, ".config", "nclip")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Create custom daemon config file
	daemonConfigPath := filepath.Join(configDir, "nclipd.toml")
	daemonConfig := `[database]
max_entries = 500
`
	if err := os.WriteFile(daemonConfigPath, []byte(daemonConfig), 0644); err != nil {
		t.Fatalf("Failed to write daemon config file: %v", err)
	}

	// Create custom theme config file
	themeConfigPath := filepath.Join(configDir, "theme.toml")
	themeConfig := `[header]
foreground = "red"
background = "blue"
bold = false

[status]
foreground = "yellow"
`
	if err := os.WriteFile(themeConfigPath, []byte(themeConfig), 0644); err != nil {
		t.Fatalf("Failed to write theme config file: %v", err)
	}

	// Create custom TUI config file
	tuiConfigPath := filepath.Join(configDir, "nclip.toml")
	tuiConfig := `[editor]
text_editor = "vim"
image_editor = "gimp"
image_viewer = "feh"
`
	if err := os.WriteFile(tuiConfigPath, []byte(tuiConfig), 0644); err != nil {
		t.Fatalf("Failed to write TUI config file: %v", err)
	}

	// Temporarily override home directory
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	// Load config
	config, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test custom values are loaded
	if config.Database.MaxEntries != 500 {
		t.Errorf("Expected MaxEntries to be 500, got %d", config.Database.MaxEntries)
	}

	if config.Theme.Header.Foreground != "red" {
		t.Errorf("Expected Header.Foreground to be 'red', got '%s'", config.Theme.Header.Foreground)
	}
	if config.Theme.Header.Background != "blue" {
		t.Errorf("Expected Header.Background to be 'blue', got '%s'", config.Theme.Header.Background)
	}
	if config.Theme.Header.Bold {
		t.Error("Expected Header.Bold to be false")
	}

	if config.Theme.Status.Foreground != "yellow" {
		t.Errorf("Expected Status.Foreground to be 'yellow', got '%s'", config.Theme.Status.Foreground)
	}

	if config.Editor.TextEditor != "vim" {
		t.Errorf("Expected TextEditor to be 'vim', got '%s'", config.Editor.TextEditor)
	}
	if config.Editor.ImageViewer != "feh" {
		t.Errorf("Expected ImageViewer to be 'feh', got '%s'", config.Editor.ImageViewer)
	}

	// Test that missing values fall back to defaults
	if config.Theme.Search.Foreground != "141" {
		t.Errorf("Expected Search.Foreground to default to '141', got '%s'", config.Theme.Search.Foreground)
	}
}

func TestConfig_InvalidMaxEntries(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "nclip-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create config directory
	configDir := filepath.Join(tmpDir, ".config", "nclip")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Create config with invalid max_entries
	configPath := filepath.Join(configDir, "config.toml")
	invalidConfig := `[database]
max_entries = -10
`
	if err := os.WriteFile(configPath, []byte(invalidConfig), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Temporarily override home directory
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	// Load config
	config, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Should fallback to default value
	if config.Database.MaxEntries != 1000 {
		t.Errorf("Expected MaxEntries to fallback to 1000, got %d", config.Database.MaxEntries)
	}
}

func TestConfig_MalformedToml(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "nclip-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create config directory
	configDir := filepath.Join(tmpDir, ".config", "nclip")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Create malformed daemon config file (missing closing bracket)
	configPath := filepath.Join(configDir, "nclipd.toml")
	malformedConfig := `[database
max_entries = 500
invalid syntax here
`
	if err := os.WriteFile(configPath, []byte(malformedConfig), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Temporarily override home directory
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	// Load config should fail
	_, err = Load()
	if err == nil {
		t.Error("Expected error when loading malformed config")
	}
	if !strings.Contains(err.Error(), "failed to decode daemon config file") {
		t.Errorf("Expected daemon config decode error, got: %v", err)
	}
}

func TestCreateDefaultConfig(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "nclip-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.toml")

	// Create default config
	err = createDefaultConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to create default config: %v", err)
	}

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Default config file was not created")
	}

	// Read and verify content
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	contentStr := string(content)

	// Check for expected sections
	expectedSections := []string{
		"[database]",
		"max_entries = 1000",
		"[theme.header]",
		"foreground = \"8\"",
		"[editor]",
		"text_editor = \"nano\"",
		"image_editor = \"gimp\"",
	}

	for _, expected := range expectedSections {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("Expected config to contain '%s'", expected)
		}
	}
}

func TestCreateDefaultConfig_DirectoryCreation(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "nclip-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Use nested directory that doesn't exist
	configPath := filepath.Join(tmpDir, "nonexistent", "config.toml")

	// Create default config - should create directory
	err = createDefaultConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to create default config: %v", err)
	}

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Default config file was not created")
	}

	// Check if directory was created
	if _, err := os.Stat(filepath.Dir(configPath)); os.IsNotExist(err) {
		t.Error("Config directory was not created")
	}
}

func TestLoad_HomeDirectoryError(t *testing.T) {
	// Temporarily set invalid HOME to trigger UserHomeDir error
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Unsetenv("HOME")

	// This should fail on systems where UserHomeDir depends on HOME
	_, err := Load()
	if err == nil {
		// On some systems this might not fail, so we skip the test
		t.Skip("UserHomeDir didn't fail on this system")
	}
	if !strings.Contains(err.Error(), "failed to get user home directory") {
		t.Errorf("Expected home directory error, got: %v", err)
	}
}
