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
	"strings"
	
	"github.com/charmbracelet/lipgloss"
	"github.com/adaryorg/nclip/internal/config"
)

// ThemeService provides styled components based on the theme configuration
type ThemeService struct {
	config *config.ThemeConfig
}

// NewThemeService creates a new theme service
func NewThemeService(themeConfig *config.ThemeConfig) *ThemeService {
	return &ThemeService{
		config: themeConfig,
	}
}

// MainViewStyles returns all styles for the main view
type MainViewStyles struct {
	Border              lipgloss.Style
	Header              lipgloss.Style
	HeaderSeparator     lipgloss.Style
	Text                lipgloss.Style
	HighlightedText     lipgloss.Style
	PinnedIndicator     lipgloss.Style
	HighRiskIndicator   lipgloss.Style
	MediumRiskIndicator lipgloss.Style
	FooterSeparator     lipgloss.Style
	FooterKey           lipgloss.Style
	FooterAction        lipgloss.Style
	FooterDivider       lipgloss.Style
	FilterIndicator     lipgloss.Style
	NormalBackground    lipgloss.Style
	AlternateBackground lipgloss.Style
	SelectedBackground  lipgloss.Style
	GlobalBackground    lipgloss.Style
}

// GetMainViewStyles returns styled components for the main view
func (ts *ThemeService) GetMainViewStyles() MainViewStyles {
	return MainViewStyles{
		Border:              colorConfigToStyle(ts.config.Main.Border),
		Header:              colorConfigToStyle(ts.config.Main.Header),
		HeaderSeparator:     colorConfigToStyle(ts.config.Main.HeaderSeparator),
		Text:                ts.colorConfigToStyleWithGlobalBg(ts.config.Main.Text),
		HighlightedText:     colorConfigToStyle(ts.config.Main.HighlightedText),
		PinnedIndicator:     colorConfigToStyleForegroundOnly(ts.config.Main.PinnedIndicator),
		HighRiskIndicator:   colorConfigToStyleForegroundOnly(ts.config.Main.HighRiskIndicator),
		MediumRiskIndicator: colorConfigToStyleForegroundOnly(ts.config.Main.MediumRiskIndicator),
		FooterSeparator:     ts.colorConfigToStyleWithGlobalBg(ts.config.Main.FooterSeparator),
		FooterKey:           ts.colorConfigToStyleWithGlobalBg(ts.config.Main.FooterKey),
		FooterAction:        ts.colorConfigToStyleWithGlobalBg(ts.config.Main.FooterAction),
		FooterDivider:       ts.colorConfigToStyleWithGlobalBg(ts.config.Main.FooterDivider),
		FilterIndicator:     ts.colorConfigToStyleWithGlobalBg(ts.config.Main.FilterIndicator),
		NormalBackground:    colorConfigToStyle(ts.config.Main.NormalBackground),
		AlternateBackground: colorConfigToStyle(ts.config.Main.AlternateBackground),
		SelectedBackground:  colorConfigToStyle(ts.config.Main.SelectedBackground),
		GlobalBackground:    colorConfigToStyle(ts.config.Main.GlobalBackground),
	}
}

// ViewStyles returns styles for any view with inheritance
type ViewStyles struct {
	Border          lipgloss.Style
	Header          lipgloss.Style
	HeaderIcon      lipgloss.Style
	HeaderInfo      lipgloss.Style
	HeaderRisk      lipgloss.Style
	HeaderSeparator lipgloss.Style
	Text            lipgloss.Style
	FooterSeparator lipgloss.Style
	FooterKey       lipgloss.Style
	FooterAction    lipgloss.Style
	FooterDivider   lipgloss.Style
	ImageInfo       lipgloss.Style
	Actions         lipgloss.Style
}

// GetViewStyles returns styled components for a specific view with inheritance
func (ts *ThemeService) GetViewStyles(viewName string) ViewStyles {
	viewTheme := ts.config.GetViewTheme(viewName)
	
	styles := ViewStyles{
		Border:          colorConfigToStyle(*viewTheme.Border),
		Header:          colorConfigToStyle(*viewTheme.Header),
		HeaderSeparator: colorConfigToStyle(*viewTheme.HeaderSeparator),
		Text:            ts.colorConfigToStyleWithGlobalBg(*viewTheme.Text),
		FooterSeparator: ts.colorConfigToStyleWithGlobalBg(*viewTheme.FooterSeparator),
		FooterKey:       ts.colorConfigToStyleWithGlobalBg(*viewTheme.FooterKey),
		FooterAction:    ts.colorConfigToStyleWithGlobalBg(*viewTheme.FooterAction),
		FooterDivider:   ts.colorConfigToStyleWithGlobalBg(*viewTheme.FooterDivider),
	}
	
	// Add view-specific styles if they exist
	if viewTheme.HeaderIcon != nil {
		styles.HeaderIcon = colorConfigToStyle(*viewTheme.HeaderIcon)
	} else {
		// Default to header style if not specified
		styles.HeaderIcon = styles.Header
	}
	
	if viewTheme.HeaderInfo != nil {
		styles.HeaderInfo = colorConfigToStyle(*viewTheme.HeaderInfo)
	} else {
		// Default to dimmed text
		styles.HeaderInfo = colorConfigToStyle(config.ColorConfig{Foreground: "8", Background: "", Bold: false})
	}
	
	if viewTheme.HeaderRisk != nil {
		styles.HeaderRisk = colorConfigToStyle(*viewTheme.HeaderRisk)
	} else {
		// Default to warning color
		styles.HeaderRisk = colorConfigToStyle(config.ColorConfig{Foreground: "9", Background: "", Bold: true})
	}
	
	if viewTheme.ImageInfo != nil {
		styles.ImageInfo = colorConfigToStyle(*viewTheme.ImageInfo)
	} else {
		// Default to info color
		styles.ImageInfo = colorConfigToStyle(config.ColorConfig{Foreground: "6", Background: "", Bold: false})
	}
	
	if viewTheme.Actions != nil {
		styles.Actions = colorConfigToStyle(*viewTheme.Actions)
	} else {
		// Default to normal text
		styles.Actions = styles.Text
	}
	
	return styles
}

// WarningStyles returns styles for warning dialogs
type WarningStyles struct {
	Border lipgloss.Style
	Title  lipgloss.Style
	Body   lipgloss.Style
	Prompt lipgloss.Style
}

// GetWarningStyles returns styled components for warning dialogs
func (ts *ThemeService) GetWarningStyles() WarningStyles {
	if ts.config.ScanningWarning != nil {
		return WarningStyles{
			Border: colorConfigToStyle(*ts.config.ScanningWarning.Border),
			Title:  colorConfigToStyle(*ts.config.ScanningWarning.Title),
			Body:   ts.colorConfigToStyleWithGlobalBg(*ts.config.ScanningWarning.Body),
			Prompt: ts.colorConfigToStyleWithGlobalBg(*ts.config.ScanningWarning.Prompt),
		}
	}
	
	// Default warning styles with global background inheritance
	return WarningStyles{
		Border: colorConfigToStyle(config.ColorConfig{Foreground: "9", Background: "", Bold: true}),
		Title:  colorConfigToStyle(config.ColorConfig{Foreground: "9", Background: "", Bold: true}),
		Body:   ts.colorConfigToStyleWithGlobalBg(config.ColorConfig{Foreground: "7", Background: "", Bold: false}),
		Prompt: ts.colorConfigToStyleWithGlobalBg(config.ColorConfig{Foreground: "8", Background: "", Bold: false}),
	}
}

// GetLegacyStyles returns legacy styled components for backward compatibility
func (ts *ThemeService) GetLegacyStyles() map[string]lipgloss.Style {
	return map[string]lipgloss.Style{
		"header":              colorConfigToStyle(ts.config.Header),
		"status":              colorConfigToStyle(ts.config.Status),
		"search":              colorConfigToStyle(ts.config.Search),
		"warning":             colorConfigToStyle(ts.config.Warning),
		"selected":            colorConfigToStyle(ts.config.Selected),
		"alternateBackground": colorConfigToStyle(ts.config.AlternateBackground),
		"normalBackground":    colorConfigToStyle(ts.config.NormalBackground),
		"frameBorder":         colorConfigToStyle(ts.config.Frame.Border),
		"frameBackground":     colorConfigToStyle(ts.config.Frame.Background),
	}
}

// parseColor converts various color formats to lipgloss.Color
func parseColor(colorStr string) lipgloss.Color {
	if colorStr == "" {
		return lipgloss.Color("")
	}

	// Check if it's a hex color
	if strings.HasPrefix(colorStr, "#") {
		return lipgloss.Color(colorStr)
	}

	// Check if it's a CSS color name and convert to hex
	cssColors := map[string]string{
		"black":     "#000000",
		"red":       "#FF0000",
		"green":     "#008000",
		"yellow":    "#FFFF00",
		"blue":      "#0000FF",
		"magenta":   "#FF00FF",
		"cyan":      "#00FFFF",
		"white":     "#FFFFFF",
		"gray":      "#808080",
		"grey":      "#808080",
		"darkred":   "#8B0000",
		"darkgreen": "#006400",
		"darkblue":  "#00008B",
		"orange":    "#FFA500",
		"purple":    "#800080",
		"pink":      "#FFC0CB",
		"brown":     "#A52A2A",
		"lime":      "#00FF00",
		"navy":      "#000080",
		"maroon":    "#800000",
		"olive":     "#808000",
		"teal":      "#008080",
		"silver":    "#C0C0C0",
		"gold":      "#FFD700",
		"violet":    "#EE82EE",
		"indigo":    "#4B0082",
		"coral":     "#FF7F50",
		"salmon":    "#FA8072",
		"khaki":     "#F0E68C",
		"plum":      "#DDA0DD",
		"orchid":    "#DA70D6",
		"tan":       "#D2B48C",
	}

	if hexColor, exists := cssColors[strings.ToLower(colorStr)]; exists {
		return lipgloss.Color(hexColor)
	}

	// Otherwise, treat as ANSI color code
	return lipgloss.Color(colorStr)
}

// colorConfigToStyle converts a ColorConfig to a lipgloss Style
func colorConfigToStyle(cc config.ColorConfig) lipgloss.Style {
	style := lipgloss.NewStyle()
	
	if cc.Foreground != "" {
		style = style.Foreground(parseColor(cc.Foreground))
	}
	
	if cc.Background != "" {
		style = style.Background(parseColor(cc.Background))
	}
	
	if cc.Bold {
		style = style.Bold(true)
	}
	
	return style
}

// colorConfigToStyleWithGlobalBg converts a ColorConfig to a lipgloss Style with global background inheritance
func (ts *ThemeService) colorConfigToStyleWithGlobalBg(cc config.ColorConfig) lipgloss.Style {
	style := lipgloss.NewStyle()
	
	if cc.Foreground != "" {
		style = style.Foreground(parseColor(cc.Foreground))
	}
	
	if cc.Background != "" {
		style = style.Background(parseColor(cc.Background))
	} else if ts.config.Main.GlobalBackground.Background != "" {
		// Inherit global background if no specific background is set
		style = style.Background(parseColor(ts.config.Main.GlobalBackground.Background))
	}
	
	if cc.Bold {
		style = style.Bold(true)
	}
	
	return style
}

// colorConfigToStyleForegroundOnly converts a ColorConfig to a lipgloss Style with only foreground and bold
// This is used for icons that should inherit background from parent styles
func colorConfigToStyleForegroundOnly(cc config.ColorConfig) lipgloss.Style {
	style := lipgloss.NewStyle()
	
	if cc.Foreground != "" {
		style = style.Foreground(parseColor(cc.Foreground))
	}
	
	// Never set background - let parent style handle it
	
	if cc.Bold {
		style = style.Bold(true)
	}
	
	return style
}