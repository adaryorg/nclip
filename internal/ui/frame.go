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
)

// createFramedDialog creates a framed dialog using consistent styling
func (m Model) createFramedDialog(width, height int, content string) string {
	// Get main view styles from theme service
	mainStyles := m.themeService.GetMainViewStyles()
	
	// Create dialog style with proper background inheritance
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(mainStyles.Border.GetForeground()).
		Padding(0, 1).
		Width(width).
		Height(height)

	// Apply global background to both content and border areas
	if mainStyles.GlobalBackground.GetBackground() != lipgloss.Color("") {
		dialogStyle = dialogStyle.
			Background(mainStyles.GlobalBackground.GetBackground()).
			BorderBackground(mainStyles.GlobalBackground.GetBackground())
	}

	// Render the dialog with its styling
	dialog := dialogStyle.Render(content)

	// Create a full-screen background style
	screenStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height)
	
	// Apply global background if configured
	if mainStyles.GlobalBackground.GetBackground() != lipgloss.Color("") {
		screenStyle = screenStyle.Background(mainStyles.GlobalBackground.GetBackground())
	}

	// Use Lipgloss Align to center the dialog while preserving background
	centeredDialog := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center)
	
	// Apply global background to the centering container
	if mainStyles.GlobalBackground.GetBackground() != lipgloss.Color("") {
		centeredDialog = centeredDialog.Background(mainStyles.GlobalBackground.GetBackground())
	}

	// Render the centered dialog with background
	return centeredDialog.Render(dialog)
}

// calculateDialogDimensions returns standard dialog dimensions for consistent sizing across all views
func (m Model) calculateDialogDimensions() (dialogWidth, dialogHeight, contentWidth, contentHeight int) {
	// Standard dialog sizing used by all views
	dialogWidth = m.width - 2   // 1 char padding left and right
	dialogHeight = m.height - 2 // 1 char padding top and bottom
	contentWidth = dialogWidth - 4   // Border + internal padding
	contentHeight = dialogHeight - 4 // Border + header + footer
	
	// Apply minimum constraints
	if contentWidth < 20 {
		contentWidth = 20
	}
	if contentHeight < 5 {
		contentHeight = 5
	}
	
	return dialogWidth, dialogHeight, contentWidth, contentHeight
}

// buildFrameContent builds content for a framed dialog with header, content area, and footer
func (m Model) buildFrameContent(headerText, contentText, footerText string, contentWidth int) string {
	// Get main view styles from theme service
	mainStyles := m.themeService.GetMainViewStyles()

	var content strings.Builder

	// Header - check if it's already styled (contains ANSI codes)
	if strings.Contains(headerText, "\x1b[") {
		// Header is already styled, don't apply additional styling
		content.WriteString(headerText)
	} else {
		// Header needs styling
		content.WriteString(mainStyles.Header.Render(headerText))
	}
	content.WriteString("\n")
	content.WriteString(mainStyles.HeaderSeparator.Render(strings.Repeat("─", contentWidth)))
	content.WriteString("\n")

	// Content area - just add the content as-is since buildMainContent already sized it correctly
	content.WriteString(contentText)

	// Footer separator and controls  
	content.WriteString(mainStyles.FooterSeparator.Render(strings.Repeat("─", contentWidth)))
	content.WriteString("\n")

	// Footer - parse and style key-action pairs
	if len(footerText) > contentWidth {
		footerText = footerText[:contentWidth-3] + "..."
	}
	
	// Parse and style the footer text
	items, filterIndicator := m.parseFooterText(footerText)
	styledFooter := m.buildStyledFooter(items, filterIndicator, mainStyles)
	content.WriteString(styledFooter)

	return content.String()
}

// FooterItem represents a key-action pair in the footer
type FooterItem struct {
	Key    string
	Action string
}

// buildStyledFooter builds a styled footer from structured data
func (m Model) buildStyledFooter(items []FooterItem, filterIndicator string, mainStyles MainViewStyles) string {
	var parts []string
	
	// Build key-action pairs
	for i, item := range items {
		if i > 0 {
			parts = append(parts, mainStyles.FooterDivider.Render(" | "))
		}
		
		styledItem := mainStyles.FooterKey.Render(item.Key) + 
			mainStyles.FooterDivider.Render(": ") + 
			mainStyles.FooterAction.Render(item.Action)
		
		parts = append(parts, styledItem)
	}
	
	// Add filter indicator if present
	if filterIndicator != "" {
		parts = append(parts, mainStyles.FooterDivider.Render(" | "))
		parts = append(parts, mainStyles.FilterIndicator.Render(filterIndicator))
	}
	
	return strings.Join(parts, "")
}

// parseFooterText parses footer text into structured data
func (m Model) parseFooterText(footerText string) (items []FooterItem, filterIndicator string) {
	// Check for filter indicator
	filterStart := strings.Index(footerText, "[")
	if filterStart != -1 {
		filterIndicator = footerText[filterStart:]
		footerText = footerText[:filterStart]
	}
	
	// Parse key-action pairs
	parts := strings.Split(footerText, " | ")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		
		colonIndex := strings.Index(part, ":")
		if colonIndex > 0 {
			items = append(items, FooterItem{
				Key:    strings.TrimSpace(part[:colonIndex]),
				Action: strings.TrimSpace(part[colonIndex+1:]),
			})
		}
	}
	
	return items, filterIndicator
}
