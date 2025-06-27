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
	selectedStyle := createStyle(m.config.Theme.Selected)
	statusStyle := createStyle(m.config.Theme.Status)

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

		item := m.filteredItems[itemIndex]
		displayLines := m.getItemDisplayLines(item, contentWidth)

		// Check if we have room for this entire entry
		if linesUsed+len(displayLines) > contentHeight {
			break
		}

		// Render the entry
		for _, line := range displayLines {
			if itemIndex == m.cursor {
				// Selected item uses selected style with padding
				content.WriteString(selectedStyle.Render(" " + line + " "))
			} else {
				// Non-selected items use consistent styling
				content.WriteString("  " + line)
			}
			content.WriteString("\n")
			linesUsed++
		}

		// Add separator line between entries (except for the last entry)
		if i < len(visibleItems)-1 && linesUsed < contentHeight {
			separatorChar := "─"
			separatorWidth := contentWidth - 4 // Account for padding
			if separatorWidth > 0 {
				separator := strings.Repeat(separatorChar, separatorWidth)
				// Use a subtle color for the separator
				separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("238")) // medium dark gray
				content.WriteString("  " + separatorStyle.Render(separator))
				content.WriteString("\n")
				linesUsed++
			}
		}
	}

	// Fill remaining space with empty lines
	for linesUsed < contentHeight {
		content.WriteString("\n")
		linesUsed++
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
	for i, item := range m.filteredItems {
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
		if linesUsed+itemLines[i] <= contentHeight {
			visibleItems = append(visibleItems, i)
			linesUsed += itemLines[i]
		} else {
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
	for i, item := range m.filteredItems {
		itemLines[i] = len(m.getItemDisplayLines(item, contentWidth)) + 1 // +1 for separator
	}

	var visibleItems []int
	var start int

	if scrollUp {
		// Position cursor at bottom of visible area
		start = m.cursor
		linesUsed := itemLines[m.cursor]
		visibleItems = []int{m.cursor}

		// Add items above cursor
		for i := m.cursor - 1; i >= 0 && linesUsed+itemLines[i] <= contentHeight; i-- {
			start = i
			visibleItems = append([]int{i}, visibleItems...)
			linesUsed += itemLines[i]
		}
	} else {
		// Position cursor at top of visible area
		start = m.cursor
		linesUsed := itemLines[m.cursor]
		visibleItems = []int{m.cursor}

		// Add items below cursor
		for i := m.cursor + 1; i < len(m.filteredItems) && linesUsed+itemLines[i] <= contentHeight; i++ {
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

	// Get security icon for this item
	securityIcon := m.getSecurityIcon(item)
	iconPrefix := ""
	if securityIcon != "" {
		iconPrefix = securityIcon + " "
		effectiveWidth -= 3 // Account for icon and space
	}

	// Handle image items differently
	if item.ContentType == "image" {
		// Show descriptive text line for images (Content already includes size info)
		displayLines = []string{fmt.Sprintf("%s%s - Press 'v' to view, 'e' to edit", iconPrefix, item.Content)}
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

				// Add icon prefix only to first line
				lineContent := line
				if j == 0 {
					lineContent = iconPrefix + line
				}

				// Wrap long lines within multiline entries
				wrappedLines := wrapText(lineContent, effectiveWidth, maxLines-len(displayLines))
				displayLines = append(displayLines, wrappedLines...)
			}
		} else {
			// Single line entry - wrap to show full content up to 5 lines
			contentWithIcon := iconPrefix + item.Content
			displayLines = wrapText(contentWithIcon, effectiveWidth, maxLines)
		}
	}

	return displayLines
}
