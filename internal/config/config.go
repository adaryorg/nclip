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
	Mouse    MouseConfig    `toml:"mouse"`
}

// TUI-specific configuration (nclip.toml)
type TUIConfig struct {
	Editor EditorConfig `toml:"editor"`
	Mouse  MouseConfig  `toml:"mouse"`
}

type MouseConfig struct {
	Enable bool `toml:"enable"`
}

// Theme configuration (theme.toml)
type ThemeConfig struct {
	// Main view elements (serve as defaults for other views)
	Main MainViewTheme `toml:"main"`
	
	// View-specific overrides (inherit from main if not specified)
	TextView         *ViewTheme      `toml:"text_view"`
	ImageView        *ViewTheme      `toml:"image_view"`
	ImageUnsupported *ViewTheme      `toml:"image_unsupported"`
	ScanningWarning  *WarningTheme   `toml:"scanning_warning"`
	
	// Code syntax highlighting theme (special case)
	CodeHighlight CodeHighlightTheme `toml:"code_highlight"`
	
	// Chroma syntax highlighting configuration
	Chroma ChromaConfig `toml:"chroma"`
	
	// Legacy fields for backward compatibility
	Header              ColorConfig `toml:"header"`
	Status              ColorConfig `toml:"status"`
	Search              ColorConfig `toml:"search"`
	Warning             ColorConfig `toml:"warning"`
	Selected            ColorConfig `toml:"selected"`
	AlternateBackground ColorConfig `toml:"alternate_background"`
	NormalBackground    ColorConfig `toml:"normal_background"`
	Frame               FrameConfig `toml:"frame"`
}

type MainViewTheme struct {
	Border              ColorConfig `toml:"border"`
	Header              ColorConfig `toml:"header"`
	HeaderSeparator     ColorConfig `toml:"header_separator"`
	Text                ColorConfig `toml:"text"`
	HighlightedText     ColorConfig `toml:"highlighted_text"`
	PinnedIndicator     ColorConfig `toml:"pinned_indicator"`
	HighRiskIndicator   ColorConfig `toml:"high_risk_indicator"`
	MediumRiskIndicator ColorConfig `toml:"medium_risk_indicator"`
	FooterSeparator     ColorConfig `toml:"footer_separator"`
	FooterKey           ColorConfig `toml:"footer_key"`
	FooterAction        ColorConfig `toml:"footer_action"`
	FooterDivider       ColorConfig `toml:"footer_divider"`
	FilterIndicator     ColorConfig `toml:"filter_indicator"`
	
	// Row backgrounds
	NormalBackground    ColorConfig `toml:"normal_background"`
	AlternateBackground ColorConfig `toml:"alternate_background"`
	SelectedBackground  ColorConfig `toml:"selected_background"`
	
	// Global background
	GlobalBackground    ColorConfig `toml:"global_background"`
}

type ViewTheme struct {
	Border          *ColorConfig `toml:"border"`
	Header          *ColorConfig `toml:"header"`
	HeaderIcon      *ColorConfig `toml:"header_icon"`
	HeaderInfo      *ColorConfig `toml:"header_info"`
	HeaderRisk      *ColorConfig `toml:"header_risk"`
	HeaderSeparator *ColorConfig `toml:"header_separator"`
	Text            *ColorConfig `toml:"text"`
	FooterSeparator *ColorConfig `toml:"footer_separator"`
	FooterKey       *ColorConfig `toml:"footer_key"`
	FooterAction    *ColorConfig `toml:"footer_action"`
	FooterDivider   *ColorConfig `toml:"footer_divider"`
	
	// Image view specific
	ImageInfo       *ColorConfig `toml:"image_info"`
	Actions         *ColorConfig `toml:"actions"`
}

type WarningTheme struct {
	Border    *ColorConfig `toml:"border"`
	Title     *ColorConfig `toml:"title"`
	Body      *ColorConfig `toml:"body"`
	Prompt    *ColorConfig `toml:"prompt"`
}

type CodeHighlightTheme struct {
	// Will be expanded later for comprehensive syntax highlighting
	Keyword   ColorConfig `toml:"keyword"`
	String    ColorConfig `toml:"string"`
	Comment   ColorConfig `toml:"comment"`
	Number    ColorConfig `toml:"number"`
	Function  ColorConfig `toml:"function"`
	Type      ColorConfig `toml:"type"`
	Operator  ColorConfig `toml:"operator"`
	// ... more to be added
}

type ChromaConfig struct {
	Theme string `toml:"theme"`
}

// GetViewTheme returns the effective theme for a specific view with inheritance
func (t *ThemeConfig) GetViewTheme(viewName string) ViewTheme {
	var viewTheme *ViewTheme
	
	switch viewName {
	case "text":
		viewTheme = t.TextView
	case "image":
		viewTheme = t.ImageView
	case "image_unsupported":
		viewTheme = t.ImageUnsupported
	default:
		// Return main theme elements as ViewTheme
		return t.mainAsViewTheme()
	}
	
	// If no view-specific theme, return main theme
	if viewTheme == nil {
		return t.mainAsViewTheme()
	}
	
	// Apply inheritance: use main theme values where view theme doesn't override
	return t.mergeWithMain(viewTheme)
}

// mainAsViewTheme converts main theme to ViewTheme format
func (t *ThemeConfig) mainAsViewTheme() ViewTheme {
	return ViewTheme{
		Border:          &t.Main.Border,
		Header:          &t.Main.Header,
		HeaderSeparator: &t.Main.HeaderSeparator,
		Text:            &t.Main.Text,
		FooterSeparator: &t.Main.FooterSeparator,
		FooterKey:       &t.Main.FooterKey,
		FooterAction:    &t.Main.FooterAction,
		FooterDivider:   &t.Main.FooterDivider,
	}
}

// mergeWithMain applies inheritance from main theme
func (t *ThemeConfig) mergeWithMain(view *ViewTheme) ViewTheme {
	result := t.mainAsViewTheme()
	
	// Override with view-specific values if they exist
	if view.Border != nil {
		result.Border = view.Border
	}
	if view.Header != nil {
		result.Header = view.Header
	}
	if view.HeaderIcon != nil {
		result.HeaderIcon = view.HeaderIcon
	}
	if view.HeaderInfo != nil {
		result.HeaderInfo = view.HeaderInfo
	}
	if view.HeaderRisk != nil {
		result.HeaderRisk = view.HeaderRisk
	}
	if view.HeaderSeparator != nil {
		result.HeaderSeparator = view.HeaderSeparator
	}
	if view.Text != nil {
		result.Text = view.Text
	}
	if view.FooterSeparator != nil {
		result.FooterSeparator = view.FooterSeparator
	}
	if view.FooterKey != nil {
		result.FooterKey = view.FooterKey
	}
	if view.FooterAction != nil {
		result.FooterAction = view.FooterAction
	}
	if view.FooterDivider != nil {
		result.FooterDivider = view.FooterDivider
	}
	if view.ImageInfo != nil {
		result.ImageInfo = view.ImageInfo
	}
	if view.Actions != nil {
		result.Actions = view.Actions
	}
	
	return result
}

// MigrateFromLegacy populates the new theme structure from legacy fields if new structure is empty
func (t *ThemeConfig) MigrateFromLegacy() {
	// If main theme is not configured, populate from legacy fields
	if t.Main.Border.Foreground == "" && t.Frame.Border.Foreground != "" {
		t.Main.Border = t.Frame.Border
	}
	if t.Main.Header.Foreground == "" && t.Header.Foreground != "" {
		t.Main.Header = t.Header
		t.Main.HeaderSeparator = ColorConfig{Foreground: "8", Background: "", Bold: false}
	}
	if t.Main.Text.Foreground == "" {
		t.Main.Text = ColorConfig{Foreground: "", Background: "", Bold: false}
	}
	if t.Main.HighlightedText.Foreground == "" && t.Search.Foreground != "" {
		t.Main.HighlightedText = t.Search
	}
	if t.Main.NormalBackground.Background == "" && t.NormalBackground.Background != "" {
		t.Main.NormalBackground = t.NormalBackground
	}
	if t.Main.AlternateBackground.Background == "" && t.AlternateBackground.Background != "" {
		t.Main.AlternateBackground = t.AlternateBackground
	}
	if t.Main.SelectedBackground.Background == "" && t.Selected.Background != "" {
		t.Main.SelectedBackground = t.Selected
	}
	
	// Set default indicator colors if not set
	if t.Main.PinnedIndicator.Foreground == "" {
		t.Main.PinnedIndicator = ColorConfig{Foreground: "11", Background: "", Bold: true}
	}
	if t.Main.HighRiskIndicator.Foreground == "" {
		t.Main.HighRiskIndicator = ColorConfig{Foreground: "9", Background: "", Bold: true}
	}
	if t.Main.MediumRiskIndicator.Foreground == "" {
		t.Main.MediumRiskIndicator = ColorConfig{Foreground: "214", Background: "", Bold: true}
	}
	
	// Set default footer colors if not set
	if t.Main.FooterSeparator.Foreground == "" {
		t.Main.FooterSeparator = ColorConfig{Foreground: "8", Background: "", Bold: false}
	}
	if t.Main.FooterKey.Foreground == "" {
		t.Main.FooterKey = ColorConfig{Foreground: "15", Background: "", Bold: true}
	}
	if t.Main.FooterAction.Foreground == "" {
		t.Main.FooterAction = ColorConfig{Foreground: "8", Background: "", Bold: false}
	}
	if t.Main.FooterDivider.Foreground == "" {
		t.Main.FooterDivider = ColorConfig{Foreground: "8", Background: "", Bold: false}
	}
	if t.Main.FilterIndicator.Foreground == "" && t.Search.Foreground != "" {
		t.Main.FilterIndicator = t.Search
	}
}

// Daemon-specific configuration (nclipd.toml)
type DaemonConfig struct {
	Database    DatabaseConfig    `toml:"database"`
	Logging     LoggingConfig     `toml:"logging"`
	Maintenance MaintenanceConfig `toml:"maintenance"`
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
	Level          string `toml:"level"`
	LogFile        string `toml:"log_file"`
	MaxAge         int    `toml:"max_age_days"`
	MaxSize        int    `toml:"max_size_mb"`
	MaxBackups     int    `toml:"max_backups"`
}

type MaintenanceConfig struct {
	AutoDedupe      bool `toml:"auto_dedupe"`
	DedupeInterval  int  `toml:"dedupe_interval_minutes"`
	AutoPrune       bool `toml:"auto_prune"`
	PruneInterval   int  `toml:"prune_interval_minutes"`
	PruneEmptyData  bool `toml:"prune_empty_data"`
	PruneSingleChar bool `toml:"prune_single_char"`
}

// Load unified config (backwards compatibility)
func Load() (*Config, error) {
	return LoadWithCustomTheme("")
}

// LoadWithCustomTheme loads unified config with optional custom theme file
func LoadWithCustomTheme(customThemeFile string) (*Config, error) {
	tuiConfig, err := LoadTUIConfig()
	if err != nil {
		return nil, err
	}
	
	themeConfig, err := LoadThemeConfigFromFile(customThemeFile)
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
		Mouse:    tuiConfig.Mouse,
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

	// Set default mouse configuration (disabled by default to allow text selection)
	// Note: Mouse support can interfere with terminal text selection

	return &config, nil
}

// Load theme config (theme.toml)
func LoadThemeConfig() (*ThemeConfig, error) {
	return LoadThemeConfigFromFile("")
}

// LoadThemeConfigFromFile loads theme config from a specific file, or default if empty
func LoadThemeConfigFromFile(customThemeFile string) (*ThemeConfig, error) {
	var configPath string
	
	if customThemeFile != "" {
		// Use custom theme file
		configPath = customThemeFile
		// Check if file exists
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("custom theme file not found: %s", configPath)
		}
	} else {
		// Use default theme file
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}

		configDir := filepath.Join(homeDir, ".config", "nclip")
		configPath = filepath.Join(configDir, "theme.toml")

		// Create default config if it doesn't exist
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			if err := createDefaultThemeConfig(configPath); err != nil {
				return nil, fmt.Errorf("failed to create default theme config: %w", err)
			}
		}
	}

	var config ThemeConfig
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return nil, fmt.Errorf("failed to decode theme config file '%s': %w", configPath, err)
	}
	
	// Migrate from legacy theme structure if needed
	config.MigrateFromLegacy()

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
		config.Logging.Level = "info"
	}
	if config.Logging.LogFile == "" {
		homeDir, _ := os.UserHomeDir()
		config.Logging.LogFile = filepath.Join(homeDir, ".local", "log", "nclipd.log")
	}
	if config.Logging.MaxAge <= 0 {
		config.Logging.MaxAge = 10 // 10 days
	}
	if config.Logging.MaxSize <= 0 {
		config.Logging.MaxSize = 10 // 10 MB
	}
	if config.Logging.MaxBackups <= 0 {
		config.Logging.MaxBackups = 10 // 10 backups
	}

	// Set default maintenance values if not specified
	if config.Maintenance.DedupeInterval <= 0 {
		config.Maintenance.DedupeInterval = 10 // Default 10 minutes
	}
	if config.Maintenance.PruneInterval <= 0 {
		config.Maintenance.PruneInterval = 60 // Default 60 minutes
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

[mouse]
# Enable mouse support in the TUI (default: false)
# Note: Enabling mouse support may interfere with terminal text selection
# Disable this (false) to allow normal text selection with mouse
enable = false
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

	_, err = file.WriteString(`# NClip Theme Configuration
# This file supports comprehensive theming with inheritance
# Views inherit from [main] theme unless overridden

# Main view theme (serves as default for all views)
[main]

[main.border]
foreground = "8"
background = ""
bold = false

[main.header]
foreground = "8"
background = ""
bold = true

[main.header_separator]
foreground = "8"
background = ""
bold = false

[main.text]
foreground = ""
background = ""
bold = false

[main.highlighted_text]
foreground = "141"
background = ""
bold = true

[main.pinned_indicator]
foreground = "11"  # Yellow
background = ""
bold = true

[main.high_risk_indicator]
foreground = "9"   # Red
background = ""
bold = true

[main.medium_risk_indicator]
foreground = "214"  # Orange
background = ""
bold = true

[main.footer_separator]
foreground = "8"
background = ""
bold = false

[main.footer_key]
foreground = "15"
background = ""
bold = true

[main.footer_action]
foreground = "8"
background = ""
bold = false

[main.footer_divider]
foreground = "8"
background = ""
bold = false

[main.filter_indicator]
foreground = "141"
background = ""
bold = true

[main.normal_background]
foreground = ""
background = ""
bold = false

[main.alternate_background]
foreground = ""
background = ""
bold = false

[main.selected_background]
foreground = "15"
background = "55"
bold = false

# Text view overrides (optional - inherits from main if not specified)
# [text_view.header]
# foreground = "10"
# background = ""
# bold = true

# [text_view.header_info]
# foreground = "8"
# background = ""
# bold = false

# Image view overrides (optional)
# [image_view.image_info]
# foreground = "6"
# background = ""
# bold = false

# Code syntax highlighting theme
[code_highlight]

[code_highlight.keyword]
foreground = "141"
background = ""
bold = true

[code_highlight.string]
foreground = "10"
background = ""
bold = false

[code_highlight.comment]
foreground = "8"
background = ""
bold = false

[code_highlight.number]
foreground = "14"
background = ""
bold = false

[code_highlight.function]
foreground = "12"
background = ""
bold = false

[code_highlight.type]
foreground = "13"
background = ""
bold = false

[code_highlight.operator]
foreground = "9"
background = ""
bold = false

# Scanning warning theme
[scanning_warning.border]
foreground = "9"
background = ""
bold = true

[scanning_warning.title]
foreground = "9"
background = ""
bold = true

[scanning_warning.body]
foreground = "7"
background = ""
bold = false

[scanning_warning.prompt]
foreground = "8"
background = ""
bold = false

# Legacy theme fields (kept for backward compatibility)
[header]
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
background = "55"
bold = false

[alternate_background]
foreground = ""
background = ""
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
background = ""
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
level = "info"                             # Options: debug, info, warn, error
log_file = "~/.local/log/nclipd.log"       # Log file location
max_age_days = 10                          # Maximum age of log files in days
max_size_mb = 10                           # Maximum size of each log file in MB
max_backups = 10                           # Number of backup log files to keep

[maintenance]
# Automatic database maintenance tasks
auto_dedupe = true               # Enable automatic deduplication
dedupe_interval_minutes = 10     # Run deduplication every 10 minutes
auto_prune = true                # Enable automatic pruning
prune_interval_minutes = 60      # Run pruning every 60 minutes
prune_empty_data = true          # Remove entries with no data
prune_single_char = true         # Remove entries with single character data
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

[mouse]
# Enable mouse support in the TUI (default: false)
# Note: Enabling mouse support may interfere with terminal text selection
# Disable this (false) to allow normal text selection with mouse
enable = false

[logging]
level = "error"  # Options: debug, info, warn, error
`)

	return err
}