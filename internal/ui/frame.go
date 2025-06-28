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
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.parseColor(m.config.Theme.Frame.Border.Foreground)).
		Padding(0, 1).
		Width(width).
		Height(height)

	// Background colors removed to prevent interference with syntax highlighting

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
		BorderForeground(m.parseColor(m.config.Theme.Frame.Border.Foreground)).
		Padding(0, 1).
		Width(dialogWidth).
		Height(dialogHeight)

	// Background colors removed to prevent interference with syntax highlighting

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
	headerStyle := m.createStyle(m.config.Theme.Header)
	statusStyle := m.createStyle(m.config.Theme.Status)

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
