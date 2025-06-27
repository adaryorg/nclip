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
		config.Theme.Header.Foreground = "13" // bright magenta/purple
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
		config.Theme.Selected.Background = "55" // darker purple
	}
	if config.Theme.AlternateBackground.Background == "" {
		config.Theme.AlternateBackground.Background = "234" // very dark purple/gray
	}

	// Set default frame values if not specified
	if config.Theme.Frame.Border.Foreground == "" {
		config.Theme.Frame.Border.Foreground = "39" // blue color (same as help window)
	}
	if config.Theme.Frame.Background.Background == "" {
		config.Theme.Frame.Background.Background = "235" // dark gray background
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
foreground = "13"
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
background = "55"
bold = false

[theme.alternate_background]
foreground = ""
background = "234"
bold = false

[theme.normal_background]
foreground = ""
background = ""
bold = false

[theme.frame.border]
foreground = "39"
background = ""
bold = false

[theme.frame.background]
foreground = ""
background = "235"
bold = false

[editor]
text_editor = "nano"
image_editor = "gimp"
`)

	return err
}
