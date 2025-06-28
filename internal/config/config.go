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

type Config struct {
	Database DatabaseConfig `toml:"database"`
	Theme    ThemeConfig    `toml:"theme"`
	Editor   EditorConfig   `toml:"editor"`
}

type DatabaseConfig struct {
	MaxEntries int `toml:"max_entries"`
}

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

func Load() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "nclip")
	configPath := filepath.Join(configDir, "config.toml")

	// Create default config if it doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := createDefaultConfig(configPath); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
	}

	var config Config
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	// Validate configuration
	if config.Database.MaxEntries <= 0 {
		config.Database.MaxEntries = 1000 // Default fallback
	}

	// Set default theme values if not specified
	if config.Theme.Header.Foreground == "" {
		config.Theme.Header.Foreground = "8" // grey (same as footer)
		config.Theme.Header.Bold = true
	}
	if config.Theme.Status.Foreground == "" {
		config.Theme.Status.Foreground = "8"
	}
	if config.Theme.Search.Foreground == "" {
		config.Theme.Search.Foreground = "141" // light purple
		config.Theme.Search.Bold = true
	}
	if config.Theme.Warning.Foreground == "" {
		config.Theme.Warning.Foreground = "9"
		config.Theme.Warning.Bold = true
	}
	if config.Theme.Selected.Foreground == "" {
		config.Theme.Selected.Foreground = "15" // bright white
	}
	// Set selected item background for visibility (only background we keep)
	if config.Theme.Selected.Background == "" {
		config.Theme.Selected.Background = "55" // dark purple for selected items
	}
	// Force clear other background colors for better syntax highlighting compatibility
	config.Theme.AlternateBackground.Background = ""
	config.Theme.NormalBackground.Background = ""
	config.Theme.Frame.Background.Background = ""

	// Set default frame values if not specified
	if config.Theme.Frame.Border.Foreground == "" {
		config.Theme.Frame.Border.Foreground = "8" // grey (same as footer)
	}
	// Frame background removed for better syntax highlighting compatibility

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

func createDefaultConfig(configPath string) error {
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
`)

	return err
}
