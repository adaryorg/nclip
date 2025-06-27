package ui

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"

	"github.com/adaryorg/nclip/internal/clipboard"
	"github.com/adaryorg/nclip/internal/config"
	"github.com/adaryorg/nclip/internal/security"
	"github.com/adaryorg/nclip/internal/storage"
)

type mode int

const (
	modeList mode = iota
	modeSearch
	modeConfirmDelete
	modeImageView
	modeSecurityWarning
	modeHelp
	modeTextView
)

type Model struct {
	storage         *storage.Storage
	config          *config.Config
	items           []storage.ClipboardItem
	filteredItems   []storage.ClipboardItem
	cursor          int
	searchQuery     string
	currentMode     mode
	width           int
	height          int
	deleteCandidate *storage.ClipboardItem
	visibleStart    int                    // Track first visible item index
	viewingImage    *storage.ClipboardItem // Track currently viewed image

	// Security warning state
	securityContent string
	securityThreats []security.SecurityThreat
	securityItem    *storage.ClipboardItem // Current item being analyzed
	hashStore       *security.HashStore

	// Terminal capabilities
	iconHelper *SecurityIconHelper

	// Help screen state
	helpScrollOffset int

	// Text viewer state
	viewingText      *storage.ClipboardItem
	textScrollOffset int
}

func NewModel(s *storage.Storage, cfg *config.Config) Model {
	items := s.GetAll()
	hashStore, _ := security.NewHashStore() // Initialize security hash store
	iconHelper := NewSecurityIconHelper()   // Initialize terminal detection

	return Model{
		storage:       s,
		config:        cfg,
		items:         items,
		filteredItems: items,
		cursor:        0,
		currentMode:   modeList,
		hashStore:     hashStore,
		iconHelper:    iconHelper,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
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
		"beige":     "#F5F5DC",
		"mint":      "#98FB98",
		"lavender":  "#E6E6FA",
	}

	// Convert to lowercase for case-insensitive matching
	lowerColor := strings.ToLower(colorStr)
	if hexColor, exists := cssColors[lowerColor]; exists {
		return lipgloss.Color(hexColor)
	}

	// Assume it's an ANSI color code
	return lipgloss.Color(colorStr)
}

// createStyle creates a lipgloss style from a ColorConfig
func createStyle(colorCfg config.ColorConfig) lipgloss.Style {
	style := lipgloss.NewStyle()

	if colorCfg.Foreground != "" {
		style = style.Foreground(parseColor(colorCfg.Foreground))
	}
	if colorCfg.Background != "" {
		style = style.Background(parseColor(colorCfg.Background))
	}
	if colorCfg.Bold {
		style = style.Bold(true)
	}

	return style
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case editCompleteMsg:
		// Refresh items after editing
		oldCursor := m.cursor
		editedID := msg.editedItemID

		m.items = m.storage.GetAll()
		m.filterItems()

		// Try to find the edited item and position cursor on it
		for i, item := range m.filteredItems {
			if item.ID == editedID {
				m.cursor = i
				return m, nil
			}
		}

		// If not found in filtered items, try to restore old position
		if oldCursor < len(m.filteredItems) {
			m.cursor = oldCursor
		} else if len(m.filteredItems) > 0 {
			m.cursor = len(m.filteredItems) - 1
		} else {
			m.cursor = 0
		}

		return m, nil

	case textViewEditCompleteMsg:
		// Handle text view editing completion - stay in text view mode
		// The text content has already been updated in the editTextViewEntry function
		return m, nil


	case tea.KeyMsg:
		if m.currentMode == modeSecurityWarning {
			// In security warning mode
			switch msg.String() {
			case "s":
				// Mark as safe
				if m.securityItem != nil {
					m.storage.UpdateSafeEntry(m.securityItem.ID, true)
					// Refresh items list
					m.items = m.storage.GetAll()
					m.filterItems()
					// Update the security item with new status
					updatedItem := m.storage.GetByID(m.securityItem.ID)
					if updatedItem != nil {
						m.securityItem = updatedItem
					}
				}
				return m, nil
			case "u":
				// Mark as unsafe
				if m.securityItem != nil {
					m.storage.UpdateSafeEntry(m.securityItem.ID, false)
					// Refresh items list
					m.items = m.storage.GetAll()
					m.filterItems()
					// Update the security item with new status
					updatedItem := m.storage.GetByID(m.securityItem.ID)
					if updatedItem != nil {
						m.securityItem = updatedItem
					}
				}
				return m, nil
			case "y":
				// Remove from main database and add to security hash store
				if m.hashStore != nil {
					contentHash := security.CreateHash(m.securityContent)
					if len(m.securityThreats) > 0 {
						m.hashStore.AddHash(contentHash, m.securityThreats[0])
					}
				}

				// Find and remove the item from main database
				if m.securityItem != nil {
					m.storage.Delete(m.securityItem.ID)
				}

				// Refresh items list
				m.items = m.storage.GetAll()
				m.filterItems()
				if m.cursor >= len(m.filteredItems) && len(m.filteredItems) > 0 {
					m.cursor = len(m.filteredItems) - 1
				} else if len(m.filteredItems) == 0 {
					m.cursor = 0
				}

				m.currentMode = modeList
				m.securityContent = ""
				m.securityThreats = nil
				m.securityItem = nil
				return m, nil
			case "n", "enter":
				// Close without changes
				m.currentMode = modeList
				m.securityContent = ""
				m.securityThreats = nil
				m.securityItem = nil
				return m, nil
			case "ctrl+c", "q", "esc":
				return m, tea.Quit
			default:
				// Any other key goes back to list
				m.currentMode = modeList
				m.securityContent = ""
				m.securityThreats = nil
				m.securityItem = nil
				return m, nil
			}
		} else if m.currentMode == modeHelp {
			// In help mode - calculate scroll bounds first
			helpLines := m.generateHelpContent()
			dialogHeight := m.height - 2      // 1 char padding top and bottom
			contentHeight := dialogHeight - 4 // Border + header + footer
			if contentHeight < 5 {
				contentHeight = 5
			}
			maxScrollOffset := len(helpLines) - contentHeight
			if maxScrollOffset < 0 {
				maxScrollOffset = 0
			}

			switch msg.String() {
			case "ctrl+c", "q", "esc", "h":
				// Exit help mode
				m.currentMode = modeList
				return m, nil
			case "up", "k":
				if m.helpScrollOffset > 0 {
					m.helpScrollOffset--
				}
			case "down", "j":
				// Only allow scrolling down if within bounds
				if m.helpScrollOffset < maxScrollOffset {
					m.helpScrollOffset++
				}
			case "home":
				m.helpScrollOffset = 0
			case "end":
				// Set to actual max scroll
				m.helpScrollOffset = maxScrollOffset
			default:
				// Any other key exits help
				m.currentMode = modeList
				return m, nil
			}

			// Ensure scroll offset stays within bounds
			if m.helpScrollOffset < 0 {
				m.helpScrollOffset = 0
			}
			if m.helpScrollOffset > maxScrollOffset {
				m.helpScrollOffset = maxScrollOffset
			}
		} else if m.currentMode == modeTextView {
			// In text view mode - calculate scroll bounds first
			if m.viewingText == nil {
				m.currentMode = modeList
				return m, nil
			}

			textLines := m.getTextViewLines()
			dialogHeight := m.height - 2      // 1 char padding top and bottom
			contentHeight := dialogHeight - 4 // Border + header + footer
			if contentHeight < 5 {
				contentHeight = 5
			}
			maxScrollOffset := len(textLines) - contentHeight
			if maxScrollOffset < 0 {
				maxScrollOffset = 0
			}

			switch msg.String() {
			case "ctrl+c", "q", "esc", "v":
				// Exit text view mode
				m.currentMode = modeList
				m.viewingText = nil
				m.textScrollOffset = 0
				return m, nil
			case "up", "k":
				if m.textScrollOffset > 0 {
					m.textScrollOffset--
				}
			case "down", "j":
				// Only allow scrolling down if within bounds
				if m.textScrollOffset < maxScrollOffset {
					m.textScrollOffset++
				}
			case "home":
				m.textScrollOffset = 0
			case "end":
				// Set to actual max scroll
				m.textScrollOffset = maxScrollOffset
			case "enter":
				// Copy text to clipboard and exit
				if m.viewingText != nil {
					err := clipboard.Copy(m.viewingText.Content)
					if err == nil {
						return m, tea.Quit
					}
				}
				return m, nil
			case "e":
				// Edit text
				if m.viewingText != nil {
					return m, m.editTextViewEntry(*m.viewingText)
				}
				return m, nil
			default:
				// Any other key exits text view
				m.currentMode = modeList
				m.viewingText = nil
				m.textScrollOffset = 0
				return m, nil
			}

			// Ensure scroll offset stays within bounds
			if m.textScrollOffset < 0 {
				m.textScrollOffset = 0
			}
			if m.textScrollOffset > maxScrollOffset {
				m.textScrollOffset = maxScrollOffset
			}
		} else if m.currentMode == modeImageView {
			// In image view mode
			switch msg.String() {
			case "ctrl+c", "q", "esc":
				// Exit image view mode - clear Kitty graphics only
				// Let Bubble Tea handle terminal state management
				fmt.Print("\x1b_Ga=d;\x1b\\")  // Delete all Kitty images
				
				m.currentMode = modeList
				m.viewingImage = nil
				return m, nil
			case "enter":
				// Copy image to clipboard
				if m.viewingImage != nil && len(m.viewingImage.ImageData) > 0 {
					// Copy image data back to clipboard
					err := clipboard.CopyImage(m.viewingImage.ImageData)
					if err == nil {
						return m, tea.Quit
					}
				}
				return m, nil
			case "e":
				// Edit image
				if m.viewingImage != nil {
					return m, m.editImage(*m.viewingImage)
				}
				return m, nil
			case "d":
				// Debug - dump image info to file
				if m.viewingImage != nil {
					return m, m.dumpImageDebug(*m.viewingImage)
				}
				return m, nil
			default:
				// Any other key exits image view - clear Kitty graphics only
				// Let Bubble Tea handle terminal state management
				fmt.Print("\x1b_Ga=d;\x1b\\")  // Delete all Kitty images
				
				m.currentMode = modeList
				m.viewingImage = nil
				return m, nil
			}
		} else if m.currentMode == modeConfirmDelete {
			// In delete confirmation mode
			switch msg.String() {
			case "d":
				// Confirm delete by pressing 'd' again
				if m.deleteCandidate != nil {
					err := m.storage.Delete(m.deleteCandidate.ID)
					if err == nil {
						m.items = m.storage.GetAll()
						m.filterItems()
						// Adjust cursor if needed
						if m.cursor >= len(m.filteredItems) && len(m.filteredItems) > 0 {
							m.cursor = len(m.filteredItems) - 1
						} else if len(m.filteredItems) == 0 {
							m.cursor = 0
						}
					}
				}
				m.currentMode = modeList
				m.deleteCandidate = nil
				return m, nil
			case "ctrl+c":
				return m, tea.Quit
			default:
				// Any other key cancels delete
				m.currentMode = modeList
				m.deleteCandidate = nil
				return m, nil
			}
		} else if m.currentMode == modeSearch {
			// In search mode, handle filter input with real-time preview
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.currentMode = modeList
				m.searchQuery = ""
				m.filteredItems = m.items
				m.cursor = 0
				return m, nil
			case "enter":
				// Apply filter and return to list mode with all actions available
				m.currentMode = modeList
				m.cursor = 0
				return m, nil
			case "backspace":
				if len(m.searchQuery) > 0 {
					m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
					m.filterItems() // Update display in real-time
				}
			default:
				// Only add printable characters to search query
				if len(msg.String()) == 1 && msg.String() >= " " && msg.String() <= "~" {
					m.searchQuery += msg.String()
					m.filterItems() // Update display in real-time
				}
			}
		} else {
			// In list mode, handle all shortcuts
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit

			case "/":
				m.currentMode = modeSearch
				// Keep existing search query when re-entering search mode
				return m, nil

			case "c":
				// Clear filter
				if m.searchQuery != "" {
					m.searchQuery = ""
					m.filteredItems = m.items
					m.cursor = 0
				}
				return m, nil

			case "enter":
				if len(m.filteredItems) > 0 && m.cursor < len(m.filteredItems) {
					selectedItem := m.filteredItems[m.cursor]
					err := clipboard.Copy(selectedItem.Content)
					if err != nil {
						return m, nil
					}
					return m, tea.Quit
				}

			case "v":
				// View entry in full-screen (images or text)
				if len(m.filteredItems) > 0 && m.cursor < len(m.filteredItems) {
					selectedItem := m.filteredItems[m.cursor]
					if selectedItem.ContentType == "image" {
						m.viewingImage = &selectedItem
						m.currentMode = modeImageView
						return m, nil
					} else {
						m.viewingText = &selectedItem
						m.textScrollOffset = 0
						m.currentMode = modeTextView
						return m, nil
					}
				}

			case "e":
				if len(m.filteredItems) > 0 && m.cursor < len(m.filteredItems) {
					selectedItem := m.filteredItems[m.cursor]
					if selectedItem.ContentType == "image" {
						return m, m.editImage(selectedItem)
					} else {
						return m, m.editEntry(selectedItem)
					}
				}

			case "d":
				if len(m.filteredItems) > 0 && m.cursor < len(m.filteredItems) {
					selectedItem := m.filteredItems[m.cursor]
					m.deleteCandidate = &selectedItem
					m.currentMode = modeConfirmDelete
					return m, nil
				}

			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}

			case "down", "j":
				if m.cursor < len(m.filteredItems)-1 {
					m.cursor++
				}

			case "ctrl+s":
				// Scan current item for security issues
				if len(m.filteredItems) > 0 && m.cursor < len(m.filteredItems) {
					selectedItem := m.filteredItems[m.cursor]
					if selectedItem.ContentType != "image" {
						detector := security.NewSecurityDetector()
						threats := detector.DetectSecurity(selectedItem.Content)
						// Always show security dialog, even if no threats found
						m.ShowSecurityWarning(selectedItem, threats)
						return m, nil
					}
				}

			case "h":
				// Show help screen
				m.currentMode = modeHelp
				m.helpScrollOffset = 0
				return m, nil
			}
		}
	}

	return m, nil
}

func (m *Model) filterItems() {
	if m.searchQuery == "" {
		m.filteredItems = m.items
		m.cursor = 0
		return
	}

	// When filtering, only include text items (images can't be searched)
	var textItems []storage.ClipboardItem
	var searchTargets []string

	for _, item := range m.items {
		if item.ContentType != "image" {
			textItems = append(textItems, item)
			searchTargets = append(searchTargets, item.Content)
		}
	}

	matches := fuzzy.Find(m.searchQuery, searchTargets)

	// Filter out weak matches by checking if the search term actually appears in the content
	var filteredMatches []storage.ClipboardItem
	lowerQuery := strings.ToLower(m.searchQuery)

	for _, match := range matches {
		item := textItems[match.Index]
		// Only include if the search term is actually contained in the content
		if strings.Contains(strings.ToLower(item.Content), lowerQuery) {
			filteredMatches = append(filteredMatches, item)
		}
	}

	m.filteredItems = filteredMatches

	if m.cursor >= len(m.filteredItems) {
		m.cursor = 0
	}
}

type editCompleteMsg struct {
	editedItemID string
}

func (m *Model) editEntry(item storage.ClipboardItem) tea.Cmd {
	// Create temporary file with the content
	tmpFile, err := ioutil.TempFile("", "clip-edit-*.txt")
	if err != nil {
		return tea.Cmd(func() tea.Msg { return nil })
	}

	tmpFile.WriteString(item.Content)
	tmpFile.Close()
	tmpFilePath := tmpFile.Name()

	// Get editor from config, environment, or use default
	editor := m.config.Editor.TextEditor
	if envEditor := os.Getenv("EDITOR"); envEditor != "" {
		editor = envEditor
	}

	return tea.ExecProcess(exec.Command(editor, tmpFilePath), func(err error) tea.Msg {
		// After editing, read the file and add to storage
		defer os.Remove(tmpFilePath)

		content, readErr := ioutil.ReadFile(tmpFilePath)
		if readErr != nil {
			return editCompleteMsg{}
		}

		newContent := strings.TrimSpace(string(content))
		originalContent := strings.TrimSpace(item.Content)

		if newContent != originalContent && newContent != "" {
			// Update the existing entry instead of creating a new one
			m.storage.Update(item.ID, newContent)
		}

		return editCompleteMsg{editedItemID: item.ID}
	})
}

func (m *Model) editImage(item storage.ClipboardItem) tea.Cmd {
	if len(item.ImageData) == 0 {
		return tea.Cmd(func() tea.Msg { return nil })
	}

	return tea.Cmd(func() tea.Msg {
		// Create temporary image file
		tmpFile, err := ioutil.TempFile("", "nclip-image-*.png")
		if err != nil {
			// Debug: write error to file
			os.WriteFile("/tmp/nclip_editor_debug.txt", []byte(fmt.Sprintf("Failed to create temp file: %v\n", err)), 0644)
			return nil
		}

		// Write image data to temporary file
		_, writeErr := tmpFile.Write(item.ImageData)
		tmpFile.Close()
		tmpFilePath := tmpFile.Name()

		if writeErr != nil {
			os.Remove(tmpFilePath)
			os.WriteFile("/tmp/nclip_editor_debug.txt", []byte(fmt.Sprintf("Failed to write image data: %v\n", writeErr)), 0644)
			return nil
		}

		// Get image editor from config
		imageEditor := m.config.Editor.ImageEditor

		// Debug: log what we're trying to execute
		debugInfo := fmt.Sprintf("Attempting to launch: %s %s\nTemp file exists: %v\nFile size: %d bytes\n",
			imageEditor, tmpFilePath, fileExists(tmpFilePath), len(item.ImageData))

		// Launch GUI image editor in background (non-blocking)
		cmd := exec.Command(imageEditor, tmpFilePath)
		err = cmd.Start() // Use Start() instead of Run() to not block
		if err != nil {
			debugInfo += fmt.Sprintf("Launch failed: %v\n", err)
			os.WriteFile("/tmp/nclip_editor_debug.txt", []byte(debugInfo), 0644)
			os.Remove(tmpFilePath)
			return nil
		}

		debugInfo += "Launch successful!\n"
		os.WriteFile("/tmp/nclip_editor_debug.txt", []byte(debugInfo), 0644)

		// Note: We don't wait for the editor to close or monitor file changes
		// The user can manually add the edited image back to clipboard if needed
		// This keeps the TUI responsive and handles GUI apps properly

		return nil
	})
}

func (m *Model) dumpImageDebug(item storage.ClipboardItem) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		// Create debug info
		var debug strings.Builder
		debug.WriteString("=== IMAGE DEBUG INFO ===\n")
		debug.WriteString(fmt.Sprintf("Terminal: %s\n", os.Getenv("TERM_PROGRAM")))
		debug.WriteString(fmt.Sprintf("Kitty support: %v\n", detectKittySupport()))
		debug.WriteString(fmt.Sprintf("Image data size: %d bytes\n", len(item.ImageData)))
		debug.WriteString(fmt.Sprintf("Content type: %s\n", item.ContentType))
		debug.WriteString(fmt.Sprintf("Description: %s\n", item.Content))

		// Get image dimensions and format
		if len(item.ImageData) > 0 {
			width, height, format, err := getImageDimensions(item.ImageData)
			if err != nil {
				debug.WriteString(fmt.Sprintf("Image format detection failed: %v\n", err))
			} else {
				debug.WriteString(fmt.Sprintf("Image format: %s\n", format))
				debug.WriteString(fmt.Sprintf("Image dimensions: %dx%d pixels\n", width, height))
			}
		}

		debug.WriteString(fmt.Sprintf("Terminal size: %dx%d\n", m.width, m.height))
		debug.WriteString("========================\n")

		// Write to file
		debugFile := "/tmp/nclip_debug.txt"
		err := os.WriteFile(debugFile, []byte(debug.String()), 0644)
		if err == nil {
			fmt.Printf("\nDebug info written to %s\n", debugFile)
		} else {
			fmt.Printf("\nFailed to write debug file: %v\n", err)
		}

		return nil
	})
}

func (m Model) View() string {
	// Handle special modes with their own rendering
	if m.currentMode == modeImageView && m.viewingImage != nil {
		return m.renderImageView()
	}

	if m.currentMode == modeSecurityWarning {
		return m.renderSecurityWarning()
	}

	if m.currentMode == modeHelp {
		return m.renderHelp()
	}

	if m.currentMode == modeTextView {
		return m.renderTextView()
	}

	// Render main window with frame
	return m.renderMainWindow()
}

// renderMainWindow renders the main clipboard window with frame
func (m Model) renderMainWindow() string {
	// Calculate frame dimensions for main window
	dialogWidth := m.width - 2        // 1 char padding left and right
	dialogHeight := m.height - 2      // 1 char padding top and bottom
	contentWidth := dialogWidth - 4   // Border + internal padding
	contentHeight := dialogHeight - 4 // Border + header + footer

	if contentWidth < 20 {
		contentWidth = 20
	}
	if contentHeight < 5 {
		contentHeight = 5
	}

	// Create header text
	var headerText string
	if m.searchQuery != "" {
		if m.currentMode == modeSearch {
			headerText = "Clipboard Manager - Filter: " + m.searchQuery + "█"
		} else {
			headerText = "Clipboard Manager - Filter: " + m.searchQuery + " (press 'c' to clear)"
		}
	} else {
		headerText = "Clipboard Manager"
	}

	// Add delete confirmation to header if needed
	if m.currentMode == modeConfirmDelete && m.deleteCandidate != nil {
		preview := m.deleteCandidate.Content
		if len(preview) > 30 {
			preview = preview[:27] + "..."
		}
		preview = strings.ReplaceAll(preview, "\n", " ")
		headerText += " - Delete: " + preview
	}

	// Build main content
	mainContent := m.buildMainContent(contentWidth, contentHeight)

	// Create footer text
	var footerText string
	switch m.currentMode {
	case modeConfirmDelete:
		footerText = "Press 'd' again to delete, any other key to cancel"
	case modeSearch:
		footerText = "type filter text • enter: apply filter • esc: cancel"
	default:
		if m.searchQuery != "" {
			footerText = "↑/↓: navigate • /: edit filter • c: clear • enter: copy • v: view • e: edit • h: help • q: quit"
		} else {
			footerText = "↑/↓: navigate • /: search • enter: copy • v: view • e: edit • h: help • q: quit"
		}
	}

	// Build frame content using shared function
	frameContent := m.buildFrameContent(headerText, mainContent, footerText, contentWidth)

	// Create framed main window
	return m.createMainFrameDialog(frameContent)
}

// wrapText wraps text to fit within the given width, up to maxLines
func wrapText(text string, width int, maxLines int) []string {
	if width <= 0 {
		width = 80 // fallback width
	}

	var lines []string
	remaining := text

	for len(remaining) > 0 && len(lines) < maxLines {
		if len(remaining) <= width {
			lines = append(lines, remaining)
			break
		}

		// Find the best break point within the width
		breakPoint := width

		// Try to break at a space if possible
		for i := width - 1; i >= width/2; i-- {
			if i < len(remaining) && remaining[i] == ' ' {
				breakPoint = i
				break
			}
		}

		lines = append(lines, remaining[:breakPoint])
		remaining = strings.TrimLeft(remaining[breakPoint:], " ")
	}

	// If there's still text remaining and we've hit maxLines, add ellipsis
	if len(remaining) > 0 && len(lines) == maxLines {
		if len(lines) > 0 {
			lastLine := lines[len(lines)-1]
			if len(lastLine) > 3 {
				lines[len(lines)-1] = lastLine[:len(lastLine)-3] + "..."
			}
		}
	}

	return lines
}


// renderImageView renders the image viewer using manual positioning
func (m Model) renderImageView() string {
	// Use the new implementation that properly handles Kitty graphics
	return m.renderImageViewNew()
}

// renderImageDebugView shows debug info about why image support wasn't detected
func (m Model) renderImageDebugView() string {

	debugContent := fmt.Sprintf(`Image Viewer Debug Information

Terminal Detection Results:
• TERM_PROGRAM: %q
• TERM: %q  
• KITTY_WINDOW_ID: %q
• WEZTERM_EXECUTABLE: %q
• WEZTERM_PANE: %q
• KONSOLE_VERSION: %q
• FOOT_PID: %q

Detection Logic:
✗ Not detected as supporting Kitty graphics protocol

Supported terminals with Kitty graphics protocol:
- Kitty: TERM_PROGRAM="kitty" or KITTY_WINDOW_ID set
- Ghostty: TERM_PROGRAM="ghostty"  
- WezTerm: TERM_PROGRAM="WezTerm" or WEZTERM_* vars set
- Konsole: TERM_PROGRAM="Konsole" or KONSOLE_VERSION set
- foot: TERM_PROGRAM="foot" or FOOT_PID set
- Or TERM containing: kitty, wezterm, foot, konsole

Image info: %s (%d bytes)

Press any key to return to list`, 
		os.Getenv("TERM_PROGRAM"),
		os.Getenv("TERM"),
		os.Getenv("KITTY_WINDOW_ID"), 
		os.Getenv("WEZTERM_EXECUTABLE"),
		os.Getenv("WEZTERM_PANE"),
		os.Getenv("KONSOLE_VERSION"),
		os.Getenv("FOOT_PID"),
		m.viewingImage.Content,
		len(m.viewingImage.ImageData))

	// Calculate frame dimensions
	dialogWidth := m.width - 2   
	dialogHeight := m.height - 2 
	contentWidth := dialogWidth - 4   

	frameContent := m.buildFrameContent("Image Viewer - Debug Mode", debugContent, "v/esc/q: close", contentWidth)
	return m.createFramedDialog(dialogWidth, dialogHeight, frameContent)
}

// renderSimpleImageView for terminals that don't support Kitty graphics
func (m Model) renderSimpleImageView() string {
	// Get icon
	viewIcon := "?"
	if m.iconHelper.GetCapabilities().SupportsUnicode {
		viewIcon = "🖼️"
	}

	headerText := fmt.Sprintf("%s Image View - Terminal Not Supported", viewIcon)

	unsupportedContent := fmt.Sprintf(`%s

[Terminal does not support image display]

Your terminal does not support the Kitty graphics protocol.
Supported terminals: Kitty, Ghostty, WezTerm, Konsole, foot

Available actions:
• Press 'o' to open in external viewer (%s)
• Press Enter to copy image to clipboard  
• Press 'e' to edit in external editor
• Press 'd' to save debug info to file

Image info: %d bytes`, 
		m.viewingImage.Content,
		m.config.Editor.ImageViewer,
		len(m.viewingImage.ImageData))

	footerText := "o: open • v/esc/q: close • enter: copy • e: edit • d: debug"

	// Calculate frame dimensions
	dialogWidth := m.width - 2   
	dialogHeight := m.height - 2 
	contentWidth := dialogWidth - 4   

	frameContent := m.buildFrameContent(headerText, unsupportedContent, footerText, contentWidth)
	return m.createFramedDialog(dialogWidth, dialogHeight, frameContent)
}

// drawImageFrame draws a frame around the image area
func (m Model) drawImageFrame(startX, startY, width, height int, format string, imgWidth, imgHeight int) string {
	var result strings.Builder

	headerStyle := createStyle(m.config.Theme.Header).Bold(true).Foreground(lipgloss.Color("35"))
	statusStyle := createStyle(m.config.Theme.Status).Foreground(lipgloss.Color("248"))

	// Top border
	result.WriteString(fmt.Sprintf("\x1b[%d;%dH", startY, startX))
	result.WriteString("╭" + strings.Repeat("─", width-2) + "╮")

	// Header line
	result.WriteString(fmt.Sprintf("\x1b[%d;%dH", startY+1, startX))
	viewIcon := "?"
	if m.iconHelper.GetCapabilities().SupportsUnicode {
		viewIcon = "🖼️"
	}

	var title string
	if format != "" {
		title = fmt.Sprintf(" %s Image View (%dx%d %s, %d bytes) ", viewIcon, imgWidth, imgHeight, strings.ToUpper(format), len(m.viewingImage.ImageData))
	} else {
		title = fmt.Sprintf(" %s Image View (%d bytes) ", viewIcon, len(m.viewingImage.ImageData))
	}

	// Truncate title if too long
	maxTitleWidth := width - 2
	if len(title) > maxTitleWidth {
		title = title[:maxTitleWidth-3] + "..."
	}

	// Pad title to frame width
	padding := width - 2 - len(title)
	leftPad := padding / 2
	rightPad := padding - leftPad

	result.WriteString("│" + strings.Repeat(" ", leftPad) + headerStyle.Render(title) + strings.Repeat(" ", rightPad) + "│")

	// Separator line
	result.WriteString(fmt.Sprintf("\x1b[%d;%dH", startY+2, startX))
	result.WriteString("│" + strings.Repeat("─", width-2) + "│")

	// Side borders (for image area)
	for y := startY + 3; y < startY+height-3; y++ {
		result.WriteString(fmt.Sprintf("\x1b[%d;%dH", y, startX))
		result.WriteString("│")
		result.WriteString(fmt.Sprintf("\x1b[%d;%dH", y, startX+width-1))
		result.WriteString("│")
	}

	// Bottom separator line
	result.WriteString(fmt.Sprintf("\x1b[%d;%dH", startY+height-3, startX))
	result.WriteString("│" + strings.Repeat("─", width-2) + "│")

	// Footer line
	result.WriteString(fmt.Sprintf("\x1b[%d;%dH", startY+height-2, startX))
	footerText := " v/esc/q: close • enter: copy • e: edit • d: debug "
	if len(footerText) > width-2 {
		footerText = footerText[:width-5] + "... "
	}

	// Pad footer to frame width
	footerPadding := width - 2 - len(footerText)
	footerLeftPad := footerPadding / 2
	footerRightPad := footerPadding - footerLeftPad

	result.WriteString("│" + strings.Repeat(" ", footerLeftPad) + statusStyle.Render(footerText) + strings.Repeat(" ", footerRightPad) + "│")

	// Bottom border
	result.WriteString(fmt.Sprintf("\x1b[%d;%dH", startY+height-1, startX))
	result.WriteString("╰" + strings.Repeat("─", width-2) + "╯")

	return result.String()
}

// scaleImageForFrame scales an image to fit within the frame if necessary
func (m Model) scaleImageForFrame(imageData []byte, imgWidth, imgHeight, frameWidth, frameHeight int) []byte {
	// Convert frame dimensions to approximate pixels
	frameWidthPixels := frameWidth * 8    // Conservative estimate
	frameHeightPixels := frameHeight * 16 // Conservative estimate

	// Check if scaling is needed
	if imgWidth <= frameWidthPixels && imgHeight <= frameHeightPixels {
		return imageData // No scaling needed
	}

	// Calculate scale factor
	scaleX := float64(frameWidthPixels) / float64(imgWidth)
	scaleY := float64(frameHeightPixels) / float64(imgHeight)

	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	targetWidth := int(float64(imgWidth) * scale)
	targetHeight := int(float64(imgHeight) * scale)

	// Resize the image
	resizedData, err := resizeImageToExactDimensions(imageData, targetWidth, targetHeight)
	if err != nil {
		return imageData // Return original if resize fails
	}

	return resizedData
}

// ShowSecurityWarning sets up the security warning dialog
func (m *Model) ShowSecurityWarning(item storage.ClipboardItem, threats []security.SecurityThreat) {
	m.securityContent = item.Content
	m.securityThreats = threats
	m.securityItem = &item
	m.currentMode = modeSecurityWarning
}

// renderSecurityWarning renders the security warning dialog
func (m Model) renderSecurityWarning() string {
	headerStyle := createStyle(m.config.Theme.Header).Padding(0, 1)
	warningStyle := createStyle(m.config.Theme.Warning).Padding(0, 1)
	statusStyle := createStyle(m.config.Theme.Status).Padding(0, 1)

	var content strings.Builder

	// Clear screen
	content.WriteString("\x1b[2J\x1b[H")

	// Header
	if len(m.securityThreats) > 0 {
		warningIcon := m.iconHelper.GetMediumRiskIcon()
		if warningIcon == "" {
			warningIcon = "!"
		}
		content.WriteString(headerStyle.Render(warningIcon + "  SECURITY WARNING"))
	} else {
		// Use a simple magnifying glass alternative for scan mode
		scanIcon := "?"
		if m.iconHelper.GetCapabilities().SupportsUnicode {
			scanIcon = "🔍"
		}
		content.WriteString(headerStyle.Render(scanIcon + " SECURITY SCAN"))
	}
	content.WriteString("\n\n")

	// Show current threat level and safe entry status
	if m.securityItem != nil {
		infoStyle := createStyle(m.config.Theme.Status)
		content.WriteString(infoStyle.Render(fmt.Sprintf("Current threat level: %s", strings.ToUpper(m.securityItem.ThreatLevel))))
		content.WriteString("\n")
		safeStatus := "UNSAFE"
		if m.securityItem.SafeEntry {
			safeStatus = "SAFE"
		}
		content.WriteString(infoStyle.Render(fmt.Sprintf("Currently marked as: %s", safeStatus)))
		content.WriteString("\n\n")
	}

	// Warning message
	if len(m.securityThreats) > 0 {
		threat := security.GetHighestThreat(m.securityThreats)
		if threat != nil {
			content.WriteString(warningStyle.Render(fmt.Sprintf("Live scan detected: %s (%.0f%% confidence)",
				strings.ToUpper(threat.Type), threat.Confidence*100)))
			content.WriteString("\n")
			content.WriteString(warningStyle.Render(fmt.Sprintf("Reason: %s", threat.Reason)))
			content.WriteString("\n\n")
		}
	} else {
		statusStyle := createStyle(m.config.Theme.Status)
		content.WriteString(statusStyle.Render("✅ Live scan: No security threats detected"))
		content.WriteString("\n\n")
	}

	// Content preview (first 200 chars)
	preview := m.securityContent
	if len(preview) > 200 {
		preview = preview[:200] + "..."
	}
	// Replace newlines with spaces for better display
	preview = strings.ReplaceAll(preview, "\n", " ")

	content.WriteString("Content preview:\n")
	content.WriteString(fmt.Sprintf("\"%s\"\n\n", preview))

	// Security guidance
	if len(m.securityThreats) > 0 {
		content.WriteString("Live scan found potential security content.\n")
		content.WriteString("Review the analysis and decide how to mark this entry.\n\n")
	} else {
		content.WriteString("Live scan found no security threats.\n")
		content.WriteString("You can still manually mark this entry as safe or unsafe.\n\n")
	}

	// Options
	content.WriteString("Options:\n\n")
	content.WriteString("• Press 's' to mark as SAFE\n")
	content.WriteString("• Press 'u' to mark as UNSAFE\n")
	content.WriteString("• Press 'y' to REMOVE this entry from clipboard history\n")
	content.WriteString("• Press 'n' or Enter to close without changes\n")
	content.WriteString("• Press 'q' to quit application\n\n")

	// Footer
	footerText := "s: mark safe • u: mark unsafe • y: remove • n/enter: close • q: quit"

	// Calculate padding to push footer to bottom
	currentLines := strings.Count(content.String(), "\n")
	remainingLines := m.height - currentLines - 2

	for i := 0; i < remainingLines && i >= 0; i++ {
		content.WriteString("\n")
	}

	content.WriteString(statusStyle.Render(footerText))

	return content.String()
}

// calculateOptimalStart determines the best starting item index to keep cursor visible
func (m Model) calculateOptimalStart(contentHeight int) int {
	if len(m.filteredItems) == 0 || contentHeight < 1 {
		return 0
	}

	// If cursor can fit from the beginning, start at 0
	linesNeededFromStart := m.calculateLinesUpToCursor()
	if linesNeededFromStart <= contentHeight {
		return 0
	}

	// Find the latest starting position that still shows the cursor
	for start := m.cursor; start >= 0; start-- {
		if m.canFitCursorFromStart(start, contentHeight) {
			return start
		}
	}

	// Fallback: start from cursor position
	return m.cursor
}

// calculateLinesUpToCursor counts how many lines are needed from start to show cursor
func (m Model) calculateLinesUpToCursor() int {
	totalLines := 0
	availableWidth := m.width - 4
	if availableWidth <= 0 {
		availableWidth = 80
	}

	for i := 0; i <= m.cursor && i < len(m.filteredItems); i++ {
		item := m.filteredItems[i]
		itemLines := m.calculateItemLines(item, availableWidth)
		totalLines += itemLines

		// Add separator line (except for last item)
		if i < len(m.filteredItems)-1 {
			totalLines++
		}
	}

	return totalLines
}

// canFitCursorFromStart checks if cursor is visible when starting from given index
func (m Model) canFitCursorFromStart(start int, contentHeight int) bool {
	linesUsed := 0
	availableWidth := m.width - 4
	if availableWidth <= 0 {
		availableWidth = 80
	}

	for i := start; i < len(m.filteredItems) && linesUsed < contentHeight; i++ {
		item := m.filteredItems[i]
		itemLines := m.calculateItemLines(item, availableWidth)

		// Check if this item would fit
		if linesUsed+itemLines > contentHeight {
			// This item won't fit, so cursor won't be visible if it's this item or later
			return i > m.cursor
		}

		linesUsed += itemLines

		// If we've reached the cursor and it fits, we're good
		if i == m.cursor {
			return true
		}

		// Add separator line (except for last item)
		if i < len(m.filteredItems)-1 && linesUsed < contentHeight {
			linesUsed++
		}
	}

	// If we got here, cursor wasn't reached, so it's not visible
	return false
}

// calculateItemLines calculates how many lines an item will take
func (m Model) calculateItemLines(item storage.ClipboardItem, availableWidth int) int {
	if item.ContentType == "image" {
		return 1 // Images always take 1 line
	}

	// Account for security icon
	securityIcon := m.getSecurityIcon(item)
	if securityIcon != "" {
		availableWidth -= 3 // Account for icon and space
	}

	// Handle multiline content
	lines := strings.Split(item.Content, "\n")
	isMultiline := len(lines) > 1
	maxLines := 5

	var displayLines []string
	if isMultiline {
		for j, line := range lines {
			if len(displayLines) >= maxLines {
				if j < len(lines) {
					displayLines = append(displayLines, "...")
				}
				break
			}

			// Add icon prefix only to first line
			lineContent := line
			if j == 0 && securityIcon != "" {
				lineContent = securityIcon + " " + line
			}

			wrappedLines := wrapText(lineContent, availableWidth, maxLines-len(displayLines))
			displayLines = append(displayLines, wrappedLines...)
		}
	} else {
		contentWithIcon := item.Content
		if securityIcon != "" {
			contentWithIcon = securityIcon + " " + item.Content
		}
		displayLines = wrapText(contentWithIcon, availableWidth, maxLines)
	}

	return len(displayLines)
}

// isHighRiskSecurityContent checks if content is high-risk security content
func (m Model) isHighRiskSecurityContent(item storage.ClipboardItem) bool {
	if item.ContentType == "image" {
		return false
	}

	detector := security.NewSecurityDetector()
	threats := detector.DetectSecurity(item.Content)
	return security.IsHighRiskThreat(threats)
}

// getSecurityIcon returns appropriate security icon for content
func (m Model) getSecurityIcon(item storage.ClipboardItem) string {
	if item.ContentType == "image" {
		return ""
	}

	// Use stored threat level for display
	switch item.ThreatLevel {
	case "high":
		return m.iconHelper.GetHighRiskIcon()
	case "medium":
		return m.iconHelper.GetMediumRiskIcon()
	case "low":
		// Show low risk with a dimmer icon
		if m.iconHelper.GetCapabilities().SupportsEmoji {
			return "🔓" // Unlocked padlock for low risk
		} else if m.iconHelper.GetCapabilities().SupportsUnicode {
			return "◦" // Small circle for low risk
		} else {
			return "[L]" // [L] for low risk
		}
	default: // "none"
		return ""
	}
}

// renderHelp renders the help screen as a framed modal dialog
func (m Model) renderHelp() string {
	// Ensure minimum terminal size
	if m.width < 10 || m.height < 8 {
		return "Terminal too small for help dialog"
	}

	// Calculate dialog dimensions - leave only 1 character padding on each side
	dialogWidth := m.width - 2   // 1 char padding left and right
	dialogHeight := m.height - 2 // 1 char padding top and bottom

	// Generate help content
	helpLines := m.generateHelpContent()

	// Calculate content area within the dialog (account for border + padding)
	contentWidth := dialogWidth - 4   // Border (2) + internal padding (2)
	contentHeight := dialogHeight - 4 // Border (2) + header + footer

	if contentWidth < 10 {
		contentWidth = 10
	}
	if contentHeight < 5 {
		contentHeight = 5
	}

	// Apply scroll limits
	maxScrollOffset := len(helpLines) - contentHeight
	if maxScrollOffset < 0 {
		maxScrollOffset = 0
	}
	if m.helpScrollOffset > maxScrollOffset {
		m.helpScrollOffset = maxScrollOffset
	}
	if m.helpScrollOffset < 0 {
		m.helpScrollOffset = 0
	}

	// Get visible lines
	visibleLines := make([]string, 0)
	start := m.helpScrollOffset
	end := start + contentHeight
	if end > len(helpLines) {
		end = len(helpLines)
	}

	for i := start; i < end; i++ {
		line := helpLines[i]
		if len(line) > contentWidth {
			line = line[:contentWidth-3] + "..."
		}
		visibleLines = append(visibleLines, line)
	}

	// Create header text
	helpIcon := "?"
	if m.iconHelper.GetCapabilities().SupportsUnicode {
		helpIcon = "📖"
	}
	headerText := helpIcon + " NClip Help"

	// Build help content
	var helpContent strings.Builder
	for _, line := range visibleLines {
		helpContent.WriteString(line)
		helpContent.WriteString("\n")
	}

	// Pad to fill content area
	currentLines := len(visibleLines)
	for currentLines < contentHeight-3 { // -3 for header, separator, footer
		helpContent.WriteString("\n")
		currentLines++
	}

	// Create footer text
	scrollInfo := ""
	if maxScrollOffset > 0 {
		scrollInfo = fmt.Sprintf(" • %d-%d/%d", start+1, end, len(helpLines))
	}
	footerText := "h/esc/q: close • ↑/↓: scroll" + scrollInfo

	// Build frame content using shared function
	frameContent := m.buildFrameContent(headerText, helpContent.String(), footerText, contentWidth)

	// Create framed dialog using shared function
	positioned := m.createFramedDialog(dialogWidth, dialogHeight, frameContent)

	return positioned
}

// getTextViewLines splits the text content into lines for viewing
func (m Model) getTextViewLines() []string {
	if m.viewingText == nil {
		return []string{}
	}

	// Split content into lines, preserving empty lines
	lines := strings.Split(m.viewingText.Content, "\n")

	// Calculate available width for wrapping
	dialogWidth := m.width - 2      // 1 char padding left and right
	contentWidth := dialogWidth - 4 // Border + internal padding
	if contentWidth < 10 {
		contentWidth = 10
	}

	// Wrap long lines
	var wrappedLines []string
	for _, line := range lines {
		if len(line) <= contentWidth {
			wrappedLines = append(wrappedLines, line)
		} else {
			// Wrap long lines
			for len(line) > 0 {
				if len(line) <= contentWidth {
					wrappedLines = append(wrappedLines, line)
					break
				}

				// Find best break point
				breakPoint := contentWidth
				for i := contentWidth - 1; i >= contentWidth/2 && i < len(line); i-- {
					if line[i] == ' ' {
						breakPoint = i
						break
					}
				}

				wrappedLines = append(wrappedLines, line[:breakPoint])
				line = strings.TrimLeft(line[breakPoint:], " ")
			}
		}
	}

	return wrappedLines
}

// renderTextView renders the text viewer dialog
func (m Model) renderTextView() string {
	// Ensure minimum terminal size
	if m.width < 10 || m.height < 8 {
		return "Terminal too small for text viewer"
	}

	if m.viewingText == nil {
		return "No text to view"
	}

	// Calculate dialog dimensions - leave only 1 character padding on each side
	dialogWidth := m.width - 2   // 1 char padding left and right
	dialogHeight := m.height - 2 // 1 char padding top and bottom

	// Get text lines
	textLines := m.getTextViewLines()

	// Calculate content area within the dialog
	contentWidth := dialogWidth - 4   // Border + internal padding
	contentHeight := dialogHeight - 4 // Border + header + footer

	if contentWidth < 10 {
		contentWidth = 10
	}
	if contentHeight < 5 {
		contentHeight = 5
	}

	// Apply scroll limits
	maxScrollOffset := len(textLines) - contentHeight
	if maxScrollOffset < 0 {
		maxScrollOffset = 0
	}
	if m.textScrollOffset > maxScrollOffset {
		m.textScrollOffset = maxScrollOffset
	}
	if m.textScrollOffset < 0 {
		m.textScrollOffset = 0
	}

	// Get visible lines
	visibleLines := make([]string, 0)
	start := m.textScrollOffset
	end := start + contentHeight
	if end > len(textLines) {
		end = len(textLines)
	}

	for i := start; i < end; i++ {
		line := textLines[i]
		if len(line) > contentWidth {
			line = line[:contentWidth-3] + "..."
		}
		visibleLines = append(visibleLines, line)
	}

	// Create header with icon and item info
	viewIcon := "?"
	if m.iconHelper.GetCapabilities().SupportsUnicode {
		viewIcon = "📄"
	}

	// Show security icon if present
	securityIcon := m.getSecurityIcon(*m.viewingText)
	if securityIcon != "" {
		viewIcon = securityIcon + " " + viewIcon
	}

	// Create title with length info
	lineCount := len(strings.Split(m.viewingText.Content, "\n"))
	charCount := len(m.viewingText.Content)
	headerText := fmt.Sprintf("%s Text View (%d lines, %d chars)", viewIcon, lineCount, charCount)

	// Build text content
	var textContent strings.Builder
	for _, line := range visibleLines {
		textContent.WriteString(line)
		textContent.WriteString("\n")
	}

	// Pad to fill content area
	currentLines := len(visibleLines)
	for currentLines < contentHeight-3 { // -3 for header, separator, footer
		textContent.WriteString("\n")
		currentLines++
	}

	// Create footer text
	scrollInfo := ""
	if maxScrollOffset > 0 {
		scrollInfo = fmt.Sprintf(" • %d-%d/%d", start+1, end, len(textLines))
	}
	footerText := "v/esc/q: close • ↑/↓: scroll • enter: copy • e: edit" + scrollInfo

	// Build frame content using shared function
	frameContent := m.buildFrameContent(headerText, textContent.String(), footerText, contentWidth)

	// Create framed dialog using shared function
	positioned := m.createFramedDialog(dialogWidth, dialogHeight, frameContent)

	return positioned
}

// editTextViewEntry edits the text entry and returns to text view mode
func (m *Model) editTextViewEntry(item storage.ClipboardItem) tea.Cmd {
	// Create temporary file with the content
	tmpFile, err := ioutil.TempFile("", "clip-textview-edit-*.txt")
	if err != nil {
		return tea.Cmd(func() tea.Msg { return nil })
	}

	tmpFile.WriteString(item.Content)
	tmpFile.Close()
	tmpFilePath := tmpFile.Name()

	// Get editor from config, environment, or use default
	editor := m.config.Editor.TextEditor
	if envEditor := os.Getenv("EDITOR"); envEditor != "" {
		editor = envEditor
	}

	return tea.ExecProcess(exec.Command(editor, tmpFilePath), func(err error) tea.Msg {
		// After editing, read the file and update storage
		defer os.Remove(tmpFilePath)

		content, readErr := ioutil.ReadFile(tmpFilePath)
		if readErr != nil {
			return textViewEditCompleteMsg{editedItemID: item.ID, success: false}
		}

		newContent := strings.TrimSpace(string(content))
		originalContent := strings.TrimSpace(item.Content)

		if newContent != originalContent && newContent != "" {
			// Update the existing entry
			m.storage.Update(item.ID, newContent)

			// Update the viewing text to show new content
			updatedItem := item
			updatedItem.Content = newContent
			m.viewingText = &updatedItem

			// Refresh main items list
			m.items = m.storage.GetAll()
			m.filterItems()
		}

		return textViewEditCompleteMsg{editedItemID: item.ID, success: true}
	})
}

type textViewEditCompleteMsg struct {
	editedItemID string
	success      bool
}

// generateHelpContent creates the help text lines
func (m Model) generateHelpContent() []string {
	searchStyle := createStyle(m.config.Theme.Search)
	warningStyle := createStyle(m.config.Theme.Warning)

	var lines []string

	// Basic Navigation
	lines = append(lines, searchStyle.Render("BASIC NAVIGATION"))
	lines = append(lines, "")
	lines = append(lines, "  j/k or ↑/↓   Navigate up/down through clipboard items")
	lines = append(lines, "  Enter        Copy selected item to clipboard and exit")
	lines = append(lines, "  q / Ctrl+C   Quit the application")
	lines = append(lines, "  h            Show this help screen")
	lines = append(lines, "")

	// Search and Filtering
	lines = append(lines, searchStyle.Render("SEARCH & FILTERING"))
	lines = append(lines, "")
	lines = append(lines, "  /            Enter search mode for fuzzy filtering")
	lines = append(lines, "  c            Clear current search filter")
	lines = append(lines, "")
	lines = append(lines, "  In search mode:")
	lines = append(lines, "    Type         Filter items in real-time")
	lines = append(lines, "    Enter        Apply filter and return to list")
	lines = append(lines, "    Esc          Cancel search and clear filter")
	lines = append(lines, "    Backspace    Delete characters from search")
	lines = append(lines, "")

	// Content Operations
	lines = append(lines, searchStyle.Render("CONTENT OPERATIONS"))
	lines = append(lines, "")
	lines = append(lines, "  v            View text/image in full-screen viewer")
	lines = append(lines, "  e            Edit selected item in external editor")
	lines = append(lines, "  d            Delete item (press 'd' again to confirm)")
	lines = append(lines, "")
	lines = append(lines, "  In text view mode:")
	lines = append(lines, "    v/esc/q      Exit text viewer and return to list")
	lines = append(lines, "    ↑/↓          Scroll through text content")
	lines = append(lines, "    Enter        Copy text to clipboard and exit")
	lines = append(lines, "    e            Edit text (returns to viewer after editing)")
	lines = append(lines, "")
	lines = append(lines, "  In image view mode:")
	lines = append(lines, "    Enter        Copy image to clipboard and exit")
	lines = append(lines, "    e            Edit image in external editor")
	lines = append(lines, "    d            Save debug info to file")
	lines = append(lines, "    v/esc/q      Exit image viewer and return to list")
	lines = append(lines, "")

	// Security Features
	lines = append(lines, warningStyle.Render("SECURITY FEATURES"))
	lines = append(lines, "")
	lines = append(lines, "  Ctrl+S       Analyze current item for security threats")
	lines = append(lines, "")
	lines = append(lines, "  Security indicators automatically detect:")
	lines = append(lines, "    • JWT tokens, API keys, SSH keys")
	lines = append(lines, "    • Database connection strings")
	lines = append(lines, "    • Password-like patterns")
	lines = append(lines, "    • Credit card numbers")
	lines = append(lines, "")

	// Security Icons
	caps := m.iconHelper.GetCapabilities()
	lines = append(lines, "  Security visual indicators:")
	if caps.SupportsEmoji {
		lines = append(lines, "    🔒           High-risk security content")
		lines = append(lines, "    ⚠️           Medium-risk security content")
	} else if caps.SupportsUnicode {
		lines = append(lines, "    ⚡           High-risk security content")
		lines = append(lines, "    ⚪           Medium-risk security content")
	} else if caps.SupportsColor {
		lines = append(lines, "    ! (red)      High-risk security content")
		lines = append(lines, "    ? (yellow)   Medium-risk security content")
	} else {
		lines = append(lines, "    [H]          High-risk security content")
		lines = append(lines, "    [M]          Medium-risk security content")
	}
	lines = append(lines, "")

	// Security Workflow
	lines = append(lines, "  Security workflow:")
	lines = append(lines, "    1. Daemon detects security content automatically")
	lines = append(lines, "    2. Content is stored with visual indicators")
	lines = append(lines, "    3. Press Ctrl+S to view detailed analysis")
	lines = append(lines, "    4. Choose to remove suspicious content")
	lines = append(lines, "    5. Removed content is blocked from future collection")
	lines = append(lines, "")

	// Mouse Support
	lines = append(lines, searchStyle.Render("MOUSE SUPPORT"))
	lines = append(lines, "")
	lines = append(lines, "  Click        Select and copy item directly")
	lines = append(lines, "")

	// Configuration
	lines = append(lines, searchStyle.Render("CONFIGURATION"))
	lines = append(lines, "")
	lines = append(lines, "  Config file: ~/.config/nclip/config.toml")
	lines = append(lines, "  Themes:      See THEME.md for customization")
	lines = append(lines, "  Editors:     Configure text_editor and image_editor")
	lines = append(lines, "")

	// Command Line Options
	lines = append(lines, searchStyle.Render("COMMAND LINE OPTIONS"))
	lines = append(lines, "")
	lines = append(lines, "  nclip                              Normal operation")
	lines = append(lines, "  nclip --help                       Show command help")
	lines = append(lines, "  nclip --remove-security-information Clear security hashes")
	lines = append(lines, "")

	// Data Storage
	lines = append(lines, searchStyle.Render("DATA STORAGE"))
	lines = append(lines, "")
	lines = append(lines, "  Clipboard history: ~/.config/nclip/history.db")
	lines = append(lines, "  Security hashes:   ~/.config/nclip/security_hashes.db")
	lines = append(lines, "")

	// Terminal Compatibility
	lines = append(lines, searchStyle.Render("TERMINAL COMPATIBILITY"))
	lines = append(lines, "")
	lines = append(lines, "  Image display: Only supported in terminals with Kitty graphics protocol")
	lines = append(lines, "  Security icons: Only displayed in Unicode-capable terminals")
	lines = append(lines, "")

	return lines
}

// fileExists checks if a file exists
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}
