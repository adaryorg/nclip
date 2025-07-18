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
)

// buildMainContent builds the content area for the main window with fixed layout
func (m Model) buildMainContent(contentWidth, contentHeight int) string {
	// Calculate the exact number of lines available for content
	// Add 3 more lines to lower the footer by 3 rows
	availableContentLines := contentHeight
	
	var content strings.Builder
	linesRendered := 0

	// Get main view styles from theme service
	mainStyles := m.themeService.GetMainViewStyles()

	if len(m.filteredItems) == 0 {
		// Center "no items" message in the fixed content area
		noItemsMsg := "No clipboard items found"
		padding := (availableContentLines / 2)
		
		// Add padding lines before message
		for i := 0; i < padding && linesRendered < availableContentLines; i++ {
			content.WriteString("\n")
			linesRendered++
		}
		
		// Add the message if we have space
		if linesRendered < availableContentLines {
			// Center the message horizontally
			msgPadding := (contentWidth - len(noItemsMsg)) / 2
			if msgPadding > 0 {
				content.WriteString(strings.Repeat(" ", msgPadding))
			}
			content.WriteString(mainStyles.Text.Render(noItemsMsg))
			content.WriteString("\n")
			linesRendered++
		}
		
		// Fill remaining space to reach exact line count
		for linesRendered < availableContentLines {
			content.WriteString("\n")
			linesRendered++
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

	// Render items starting from cursor page, filling exactly availableContentLines
	pageStart := m.calculatePageStart(availableContentLines, contentWidth)
	
	// Render items from pageStart until we fill the content area
	for itemIndex := pageStart; itemIndex < len(m.filteredItems) && linesRendered < availableContentLines; itemIndex++ {
		itemMeta := m.filteredItems[itemIndex]
		item := itemMeta.ToClipboardItem()
		displayLines := m.getItemDisplayLines(item, contentWidth)

		// Render as many lines of this item as fit
		for lineIndex, line := range displayLines {
			if linesRendered >= availableContentLines {
				break
			}

			if itemIndex == m.cursor {
				// Selected item - build plain text first, then apply uniform selected background
				if lineIndex == 0 && (item.IsPinned || item.ThreatLevel != "none" || item.SafeEntry) {
					// First line with icons - build plain text line, then apply selected background uniformly
					plainLine := m.buildPlainLineWithIcons(item, line)
					content.WriteString("  " + mainStyles.SelectedBackground.Render(plainLine))
				} else {
					// Other lines - apply selected background to plain text
					content.WriteString("  " + mainStyles.SelectedBackground.Render(line))
				}
			} else {
				// Non-selected items
				if lineIndex == 0 && (item.IsPinned || item.ThreatLevel != "none" || item.SafeEntry) {
					// First line with icons - build properly styled line
					styledLine := m.buildStyledLineWithIcons(item, line, mainStyles)
					content.WriteString("  " + styledLine)
				} else {
					// Other lines - apply text styling
					content.WriteString("  " + mainStyles.Text.Render(line))
				}
			}
			content.WriteString("\n")
			linesRendered++
		}

		// Add separator if we have space and this isn't the last item we'll show
		if linesRendered < availableContentLines && itemIndex < len(m.filteredItems)-1 {
			separatorChar := "─"
			separatorWidth := contentWidth - 4 // Account for padding
			if separatorWidth > 0 {
				separator := strings.Repeat(separatorChar, separatorWidth)
				content.WriteString("  " + mainStyles.HeaderSeparator.Render(separator))
				content.WriteString("\n")
				linesRendered++
			}
		}
	}

	// Fill any remaining space to reach exact line count
	for linesRendered < availableContentLines {
		content.WriteString("\n")
		linesRendered++
	}

	return content.String()
}

// calculatePageStart calculates which item should be the first item on the current page
// Simple page-based scrolling: cursor moves within current page, then jumps to next page
func (m Model) calculatePageStart(availableContentLines, contentWidth int) int {
	if len(m.filteredItems) == 0 || m.cursor < 0 {
		return 0
	}

	// Calculate lines needed for each item (including separator)
	itemLines := make([]int, len(m.filteredItems))
	for i, itemMeta := range m.filteredItems {
		item := itemMeta.ToClipboardItem()
		lines := len(m.getItemDisplayLines(item, contentWidth))
		// Add 1 for separator (except for last item)
		if i < len(m.filteredItems)-1 {
			lines++
		}
		itemLines[i] = lines
	}

	// Simple approach: find which page contains the cursor
	// Build pages sequentially until we find the one with the cursor
	pageStart := 0
	currentPageLines := 0
	
	for i := 0; i < len(m.filteredItems); i++ {
		// Check if adding this item would exceed current page
		if currentPageLines + itemLines[i] > availableContentLines && currentPageLines > 0 {
			// This item starts a new page
			if m.cursor < i {
				// Cursor is on the previous page
				break
			}
			// Start new page from this item
			pageStart = i
			currentPageLines = 0
		}
		
		currentPageLines += itemLines[i]
		
		// If we've reached the cursor, we're on the right page
		if i == m.cursor {
			break
		}
	}
	
	return pageStart
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

	// Since icons are just Unicode characters, no special width calculation needed
	// The line styling will be handled by buildStyledLineWithIcons
	firstLineWidth := effectiveWidth

	// Handle image items differently
	if item.ContentType == "image" {
		// Show descriptive text line for images (Content already includes size info)
		imageDesc := fmt.Sprintf("%s - Press 'v' to view, 'e' to edit", item.Content)
		// Note: icons will be added later by buildStyledLineWithIcons
		displayLines = wrapText(imageDesc, firstLineWidth, maxLines)
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

				// Icons will be added later by buildStyledLineWithIcons
				lineContent := line

				// Wrap long lines within multiline entries
				// Use smaller width for first line to account for icons that will be added later
				wrapWidth := effectiveWidth
				if j == 0 {
					wrapWidth = firstLineWidth
				}
				wrappedLines := wrapText(lineContent, wrapWidth, maxLines-len(displayLines))
				displayLines = append(displayLines, wrappedLines...)
			}
		} else {
			// Single line entry - wrap to show full content up to 5 lines
			// Icons will be added later by buildStyledLineWithIcons
			displayLines = wrapText(item.Content, firstLineWidth, maxLines)
		}
	}

	return displayLines
}

// buildStyledLineWithIcons builds a properly styled line with icons
func (m Model) buildStyledLineWithIcons(item storage.ClipboardItem, line string, mainStyles MainViewStyles) string {
	// Get the icons to use
	pinIcon := ""
	securityIcon := ""
	
	if item.IsPinned {
		pinIcon = m.getThemedPinIcon(item)
	}
	
	// Get security icon (includes safe marker)
	securityIcon = m.getThemedSecurityIcon(item)
	
	// The line parameter now contains only text content (icons are added here)
	textContent := line
	
	// Build the styled line by composing styled parts
	var parts []string
	
	if pinIcon != "" {
		parts = append(parts, pinIcon)
	}
	if securityIcon != "" {
		parts = append(parts, securityIcon)
	}
	
	// Add the text content with proper styling
	styledText := mainStyles.Text.Render(textContent)
	parts = append(parts, styledText)
	
	// Join with styled spaces that inherit the text background
	if len(parts) == 1 {
		return parts[0]
	}
	
	// Create a styled space using the text style for background consistency
	styledSpace := mainStyles.Text.Render(" ")
	return strings.Join(parts, styledSpace)
}

// buildPlainLineWithIcons builds a plain text line with icons for selected items
func (m Model) buildPlainLineWithIcons(item storage.ClipboardItem, line string) string {
	// Get plain icon text
	var iconParts []string
	
	if item.IsPinned {
		iconParts = append(iconParts, m.pinIconHelper.GetPinIcon(item.PinOrder))
	}
	
	// Add security icon using the same logic as getThemedSecurityIcon
	securityIcon := ""
	
	// Show safe marker if item has been marked as safe but has a threat level
	if item.SafeEntry && item.ThreatLevel != "none" {
		if m.iconHelper.GetCapabilities().SupportsUnicode {
			securityIcon = "✓"
		} else {
			securityIcon = "[s]"
		}
	} else if item.SafeEntry {
		// Don't show security warnings if item has been marked as safe with no threat
		securityIcon = ""
	} else {
		// Use stored threat level for display
		switch item.ThreatLevel {
		case "high":
			if m.iconHelper.GetCapabilities().SupportsUnicode {
				securityIcon = "⚠"
			} else {
				securityIcon = "[h]"
			}
		case "medium":
			if m.iconHelper.GetCapabilities().SupportsUnicode {
				securityIcon = "⚡"
			} else {
				securityIcon = "[m]"
			}
		default:
			securityIcon = ""
		}
	}
	
	if securityIcon != "" {
		iconParts = append(iconParts, securityIcon)
	}
	
	// Build plain text line
	if len(iconParts) > 0 {
		return strings.Join(iconParts, " ") + " " + line
	}
	return line
}

