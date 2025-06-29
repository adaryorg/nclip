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
	"fmt"
	"strings"

	"github.com/adaryorg/nclip/internal/storage"
	"github.com/charmbracelet/lipgloss"
)

// buildMainContent builds the content area for the main window
func (m Model) buildMainContent(contentWidth, contentHeight int) string {
	var content strings.Builder

	// Create styles
	selectedStyle := m.createSelectedStyle(m.config.Theme.Selected)
	statusStyle := m.createStyle(m.config.Theme.Status)

	if len(m.filteredItems) == 0 {
		// Center "no items" message
		noItemsMsg := "No clipboard items found"
		padding := (contentHeight / 2)
		for i := 0; i < padding; i++ {
			content.WriteString("\n")
		}
		// Center the message horizontally
		msgPadding := (contentWidth - len(noItemsMsg)) / 2
		if msgPadding > 0 {
			content.WriteString(strings.Repeat(" ", msgPadding))
		}
		content.WriteString(statusStyle.Render(noItemsMsg))
		// Fill remaining space
		for i := padding + 1; i < contentHeight; i++ {
			content.WriteString("\n")
		}
		return content.String()
	}

	// Ensure cursor is within bounds
	if m.cursor >= len(m.filteredItems) {
		m.cursor = len(m.filteredItems) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}

	// Improved scrolling: calculate which items to show
	_, visibleItems := m.calculateVisibleItems(contentHeight, contentWidth)

	linesUsed := 0

	// Render visible items
	for i, itemIndex := range visibleItems {
		if linesUsed >= contentHeight {
			break
		}

		itemMeta := m.filteredItems[itemIndex]
		item := itemMeta.ToClipboardItem() // Convert to full item for display (no image data needed)
		displayLines := m.getItemDisplayLines(item, contentWidth)

		// Get colored security icon for non-selected items
		coloredSecurityIcon := m.getSecurityIcon(item)
		plainSecurityIcon := m.getPlainSecurityIcon(item)

		// Render the entry, showing as many lines as fit
		linesRendered := 0
		for lineIndex, line := range displayLines {
			if linesUsed >= contentHeight {
				break
			}

			displayLine := line
			
			// For non-selected items, replace plain icon with colored icon on first line
			if itemIndex != m.cursor && lineIndex == 0 && plainSecurityIcon != "" && coloredSecurityIcon != "" {
				// Replace the plain icon at the beginning with colored one
				if strings.HasPrefix(line, plainSecurityIcon+" ") {
					displayLine = strings.Replace(line, plainSecurityIcon+" ", coloredSecurityIcon+" ", 1)
				}
			}

			if itemIndex == m.cursor {
				// Selected item uses selected style with padding (keeps plain icon for consistent highlighting)
				content.WriteString(selectedStyle.Render(" " + line + " "))
			} else {
				// Non-selected items use colored icons
				content.WriteString("  " + displayLine)
			}
			content.WriteString("\n")
			linesUsed++
			linesRendered++
		}

		// Only add separator if we fully rendered this item and have space for more
		shouldAddSeparator := i < len(visibleItems)-1 && 
			linesRendered == len(displayLines) && 
			linesUsed < contentHeight

		if shouldAddSeparator {
			separatorChar := "─"
			separatorWidth := contentWidth - 4 // Account for padding
			if separatorWidth > 0 {
				separator := strings.Repeat(separatorChar, separatorWidth)
				// Use a subtle color for the separator
				separatorStyle := lipgloss.NewStyle().Foreground(m.parseColor("238")) // medium dark gray
				content.WriteString("  " + separatorStyle.Render(separator))
				content.WriteString("\n")
				linesUsed++
			}
		}
	}

	// Only fill remaining space with empty lines if we've reached the end of all items
	if len(visibleItems) > 0 {
		lastVisibleIndex := visibleItems[len(visibleItems)-1]
		if lastVisibleIndex == len(m.filteredItems)-1 {
			// We're showing the last item, so fill remaining space
			for linesUsed < contentHeight {
				content.WriteString("\n")
				linesUsed++
			}
		}
	}

	return content.String()
}

// calculateVisibleItems calculates which items should be visible with improved scrolling
func (m Model) calculateVisibleItems(contentHeight, contentWidth int) (int, []int) {
	if len(m.filteredItems) == 0 {
		return 0, []int{}
	}

	// Calculate lines needed for each item
	itemLines := make([]int, len(m.filteredItems))
	for i, itemMeta := range m.filteredItems {
		item := itemMeta.ToClipboardItem() // Convert for line calculation
		itemLines[i] = len(m.getItemDisplayLines(item, contentWidth)) + 1 // +1 for separator
	}

	// Find the optimal starting position
	// Strategy: Try to keep cursor visible and show as many complete items as possible

	var visibleItems []int
	var start int

	// If cursor is near the top, start from beginning
	if m.cursor < 3 {
		start = 0
	} else {
		// Try to position cursor in the middle of the screen
		targetCursorLine := contentHeight / 2

		// Work backwards from cursor to find start position
		linesFromCursor := 0
		start = m.cursor

		for start > 0 && linesFromCursor < targetCursorLine {
			start--
			linesFromCursor += itemLines[start]
		}

		// Adjust if we went too far back
		if linesFromCursor > targetCursorLine && start < m.cursor-1 {
			start++
		}
	}

	// Build list of visible items from start position
	linesUsed := 0
	for i := start; i < len(m.filteredItems) && linesUsed < contentHeight; i++ {
		// Always include the item, even if it will only partially fit
		visibleItems = append(visibleItems, i)
		linesUsed += itemLines[i]
		
		// If we've filled the available space, we can stop
		if linesUsed >= contentHeight {
			break
		}
	}

	// Ensure cursor is visible
	cursorVisible := false
	for _, itemIndex := range visibleItems {
		if itemIndex == m.cursor {
			cursorVisible = true
			break
		}
	}

	// If cursor is not visible, adjust the visible items
	if !cursorVisible {
		if m.cursor < start {
			// Cursor is above visible area, scroll up
			return m.calculateVisibleItemsFromCursor(contentHeight, contentWidth, true)
		} else {
			// Cursor is below visible area, scroll down
			return m.calculateVisibleItemsFromCursor(contentHeight, contentWidth, false)
		}
	}

	return start, visibleItems
}

// calculateVisibleItemsFromCursor calculates visible items ensuring cursor is visible
func (m Model) calculateVisibleItemsFromCursor(contentHeight, contentWidth int, scrollUp bool) (int, []int) {
	// Calculate lines needed for each item
	itemLines := make([]int, len(m.filteredItems))
	for i, itemMeta := range m.filteredItems {
		item := itemMeta.ToClipboardItem() // Convert for line calculation
		itemLines[i] = len(m.getItemDisplayLines(item, contentWidth)) + 1 // +1 for separator
	}

	var visibleItems []int
	var start int

	if scrollUp {
		// Position cursor at bottom of visible area
		start = m.cursor
		linesUsed := itemLines[m.cursor]
		visibleItems = []int{m.cursor}

		// Add items above cursor, allowing partial items
		for i := m.cursor - 1; i >= 0 && linesUsed < contentHeight; i-- {
			start = i
			visibleItems = append([]int{i}, visibleItems...)
			linesUsed += itemLines[i]
		}
	} else {
		// Position cursor at top of visible area
		start = m.cursor
		linesUsed := itemLines[m.cursor]
		visibleItems = []int{m.cursor}

		// Add items below cursor, allowing partial items
		for i := m.cursor + 1; i < len(m.filteredItems) && linesUsed < contentHeight; i++ {
			visibleItems = append(visibleItems, i)
			linesUsed += itemLines[i]
		}
	}

	return start, visibleItems
}

// getItemDisplayLines gets the display lines for an item
func (m Model) getItemDisplayLines(item storage.ClipboardItem, availableWidth int) []string {
	// Handle multiline content - limit to 5 lines max for better fit
	lines := strings.Split(item.Content, "\n")
	isMultiline := len(lines) > 1

	var displayLines []string
	maxLines := 5
	// Account for padding
	effectiveWidth := availableWidth - 4
	if effectiveWidth <= 0 {
		effectiveWidth = 20 // fallback
	}

	// Get plain security icon (without ANSI colors) for display line generation
	plainSecurityIcon := m.getPlainSecurityIcon(item)
	firstLineWidth := effectiveWidth
	if plainSecurityIcon != "" {
		firstLineWidth = effectiveWidth - 4 // Account for icon "[!] " or "[?] "
		if firstLineWidth <= 0 {
			firstLineWidth = 10 // fallback
		}
	}

	// Handle image items differently
	if item.ContentType == "image" {
		// Show descriptive text line for images (Content already includes size info)
		imageDesc := fmt.Sprintf("%s - Press 'v' to view, 'e' to edit", item.Content)
		if plainSecurityIcon != "" {
			imageDesc = plainSecurityIcon + " " + imageDesc
		}
		displayLines = wrapText(imageDesc, effectiveWidth, maxLines)
	} else {
		// Regular text content
		if isMultiline {
			// Process each line of multiline entries
			for j, line := range lines {
				if len(displayLines) >= maxLines {
					if j < len(lines) {
						displayLines = append(displayLines, "...")
					}
					break
				}

				// Add plain security icon to first line only
				lineContent := line
				if j == 0 && plainSecurityIcon != "" {
					lineContent = plainSecurityIcon + " " + line
				}

				// Wrap long lines within multiline entries
				wrappedLines := wrapText(lineContent, effectiveWidth, maxLines-len(displayLines))
				displayLines = append(displayLines, wrappedLines...)
			}
		} else {
			// Single line entry - wrap to show full content up to 5 lines
			contentWithIcon := item.Content
			if plainSecurityIcon != "" {
				contentWithIcon = plainSecurityIcon + " " + item.Content
			}
			displayLines = wrapText(contentWithIcon, effectiveWidth, maxLines)
		}
	}

	return displayLines
}
