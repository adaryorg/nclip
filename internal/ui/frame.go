package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// createFramedDialog creates a framed dialog using consistent styling
func (m Model) createFramedDialog(width, height int, content string) string {
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(parseColor(m.config.Theme.Frame.Border.Foreground)).
		Background(parseColor(m.config.Theme.Frame.Background.Background)).
		Padding(0, 1).
		Width(width).
		Height(height)

	dialog := dialogStyle.Render(content)

	positioned := lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		dialog,
	)

	return positioned
}

// createMainFrameDialog creates a framed dialog for the main window (full screen)
func (m Model) createMainFrameDialog(content string) string {
	// Use almost full terminal size for main window
	dialogWidth := m.width - 2   // 1 char padding left and right
	dialogHeight := m.height - 2 // 1 char padding top and bottom

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(parseColor(m.config.Theme.Frame.Border.Foreground)).
		Background(parseColor(m.config.Theme.Frame.Background.Background)).
		Padding(0, 1).
		Width(dialogWidth).
		Height(dialogHeight)

	dialog := dialogStyle.Render(content)

	// Position with minimal padding (not centered like other dialogs)
	positioned := lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		dialog,
	)

	return positioned
}

// buildFrameContent builds content for a framed dialog with header, content area, and footer
func (m Model) buildFrameContent(headerText, contentText, footerText string, contentWidth int) string {
	headerStyle := createStyle(m.config.Theme.Header)
	statusStyle := createStyle(m.config.Theme.Status)

	var content strings.Builder

	// Header
	content.WriteString(headerStyle.Render(headerText))
	content.WriteString("\n")
	content.WriteString(strings.Repeat("─", contentWidth))
	content.WriteString("\n")

	// Content
	content.WriteString(contentText)

	// Footer separator and controls
	content.WriteString(strings.Repeat("─", contentWidth))
	content.WriteString("\n")

	// Footer
	if len(footerText) > contentWidth {
		footerText = footerText[:contentWidth-3] + "..."
	}
	content.WriteString(statusStyle.Render(footerText))

	return content.String()
}
