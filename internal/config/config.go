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
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Unified config for backwards compatibility
type Config struct {
	Database DatabaseConfig `toml:"database"`
	Theme    ThemeConfig    `toml:"theme"`
	Editor   EditorConfig   `toml:"editor"`
	Logging  LoggingConfig  `toml:"logging"`
}

// TUI-specific configuration (nclip.toml)
type TUIConfig struct {
	Editor EditorConfig `toml:"editor"`
}

// Theme configuration (theme.toml)
type ThemeConfig struct {
	Header              ColorConfig `toml:"header"`
	Status              ColorConfig `toml:"status"`
	Search              ColorConfig `toml:"search"`
	Warning             ColorConfig `toml:"warning"`
	Selected            ColorConfig `toml:"selected"`
	AlternateBackground ColorConfig `toml:"alternate_background"`
	NormalBackground    ColorConfig `toml:"normal_background"`
	Frame               FrameConfig `toml:"frame"`
}

// Daemon-specific configuration (nclipd.toml)
type DaemonConfig struct {
	Database DatabaseConfig `toml:"database"`
	Logging  LoggingConfig  `toml:"logging"`
}

type DatabaseConfig struct {
	MaxEntries int `toml:"max_entries"`
}

type FrameConfig struct {
	Border     ColorConfig `toml:"border"`
	Background ColorConfig `toml:"background"`
}

type ColorConfig struct {
	Foreground string `toml:"foreground"`
	Background string `toml:"background"`
	Bold       bool   `toml:"bold"`
}

type EditorConfig struct {
	TextEditor  string `toml:"text_editor"`
	ImageEditor string `toml:"image_editor"`
	ImageViewer string `toml:"image_viewer"`
}

type LoggingConfig struct {
	Level string `toml:"level"`
}

// Load unified config (backwards compatibility)
func Load() (*Config, error) {
	tuiConfig, err := LoadTUIConfig()
	if err != nil {
		return nil, err
	}
	
	themeConfig, err := LoadThemeConfig()
	if err != nil {
		return nil, err
	}
	
	daemonConfig, err := LoadDaemonConfig()
	if err != nil {
		return nil, err
	}
	
	return &Config{
		Database: daemonConfig.Database,
		Theme:    *themeConfig,
		Editor:   tuiConfig.Editor,
		Logging:  daemonConfig.Logging,
	}, nil
}

// Load TUI-specific config (nclip.toml)
func LoadTUIConfig() (*TUIConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "nclip")
	configPath := filepath.Join(configDir, "nclip.toml")

	// Create default config if it doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := createDefaultTUIConfig(configPath); err != nil {
			return nil, fmt.Errorf("failed to create default TUI config: %w", err)
		}
	}

	var config TUIConfig
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return nil, fmt.Errorf("failed to decode TUI config file: %w", err)
	}

	// Set default editor values if not specified
	if config.Editor.TextEditor == "" {
		config.Editor.TextEditor = "nano"
	}
	if config.Editor.ImageEditor == "" {
		config.Editor.ImageEditor = "gimp"
	}
	if config.Editor.ImageViewer == "" {
		config.Editor.ImageViewer = "loupe"
	}

	return &config, nil
}

// Load theme config (theme.toml)
func LoadThemeConfig() (*ThemeConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "nclip")
	configPath := filepath.Join(configDir, "theme.toml")

	// Create default config if it doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := createDefaultThemeConfig(configPath); err != nil {
			return nil, fmt.Errorf("failed to create default theme config: %w", err)
		}
	}

	var config ThemeConfig
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return nil, fmt.Errorf("failed to decode theme config file: %w", err)
	}

	// Set default theme values if not specified
	if config.Header.Foreground == "" {
		config.Header.Foreground = "8" // grey (same as footer)
		config.Header.Bold = true
	}
	if config.Status.Foreground == "" {
		config.Status.Foreground = "8"
	}
	if config.Search.Foreground == "" {
		config.Search.Foreground = "141" // light purple
		config.Search.Bold = true
	}
	if config.Warning.Foreground == "" {
		config.Warning.Foreground = "9"
		config.Warning.Bold = true
	}
	if config.Selected.Foreground == "" {
		config.Selected.Foreground = "15" // bright white
	}
	// Set selected item background for visibility (only background we keep)
	if config.Selected.Background == "" {
		config.Selected.Background = "55" // dark purple for selected items
	}
	// Force clear other background colors for better syntax highlighting compatibility
	config.AlternateBackground.Background = ""
	config.NormalBackground.Background = ""
	config.Frame.Background.Background = ""

	// Set default frame values if not specified
	if config.Frame.Border.Foreground == "" {
		config.Frame.Border.Foreground = "8" // grey (same as footer)
	}

	return &config, nil
}

// Load daemon config (nclipd.toml)
func LoadDaemonConfig() (*DaemonConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "nclip")
	configPath := filepath.Join(configDir, "nclipd.toml")

	// Create default config if it doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := createDefaultDaemonConfig(configPath); err != nil {
			return nil, fmt.Errorf("failed to create default daemon config: %w", err)
		}
	}

	var config DaemonConfig
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return nil, fmt.Errorf("failed to decode daemon config file: %w", err)
	}

	// Validate configuration
	if config.Database.MaxEntries <= 0 {
		config.Database.MaxEntries = 1000 // Default fallback
	}

	// Set default logging values if not specified
	if config.Logging.Level == "" {
		config.Logging.Level = "error"
	}

	return &config, nil
}

func createDefaultTUIConfig(configPath string) error {
	// Ensure config directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	file, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(`[editor]
text_editor = "nano"
image_editor = "gimp"
image_viewer = "loupe"
`)

	return err
}

func createDefaultThemeConfig(configPath string) error {
	// Ensure config directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	file, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(`[header]
foreground = "8"
background = ""
bold = true

[status]
foreground = "8"
background = ""
bold = false

[search]
foreground = "141"
background = ""
bold = true

[warning]
foreground = "9"
background = ""
bold = true

[selected]
foreground = "15"
background = "55"  # Only background kept for selected item visibility
bold = false

[alternate_background]
foreground = ""
background = ""  # Background removed for better syntax highlighting
bold = false

[normal_background]
foreground = ""
background = ""
bold = false

[frame.border]
foreground = "8"
background = ""
bold = false

[frame.background]
foreground = ""
background = ""  # Background removed for better syntax highlighting
bold = false
`)

	return err
}

func createDefaultDaemonConfig(configPath string) error {
	// Ensure config directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	file, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(`[database]
max_entries = 1000

[logging]
level = "error"  # Options: debug, info, warn, error
`)

	return err
}

// Legacy function for backwards compatibility
func createDefaultConfig(configPath string) error {
	// This function creates the old unified config.toml
	// Ensure config directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	file, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(`[database]
max_entries = 1000

[theme.header]
foreground = "8"
background = ""
bold = true

[theme.status]
foreground = "8"
background = ""
bold = false

[theme.search]
foreground = "141"
background = ""
bold = true

[theme.warning]
foreground = "9"
background = ""
bold = true

[theme.selected]
foreground = "15"
background = "55"  # Only background kept for selected item visibility
bold = false

[theme.alternate_background]
foreground = ""
background = ""  # Background removed for better syntax highlighting
bold = false

[theme.normal_background]
foreground = ""
background = ""
bold = false

[theme.frame.border]
foreground = "8"
background = ""
bold = false

[theme.frame.background]
foreground = ""
background = ""  # Background removed for better syntax highlighting
bold = false

[editor]
text_editor = "nano"
image_editor = "gimp"
image_viewer = "loupe"

[logging]
level = "error"  # Options: debug, info, warn, error
`)

	return err
}