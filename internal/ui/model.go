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
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
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
	modeImageSecurityWarning
)

type Model struct {
	storage         *storage.Storage
	config          *config.Config
	cache           *storage.ItemCache     // Memory-efficient cache
	items           []storage.ClipboardItemMeta // Lightweight metadata only
	filteredItems   []storage.ClipboardItemMeta // Filtered lightweight metadata
	cursor          int
	searchQuery     string
	currentMode     mode
	
	// Content filtering
	filterMode      string // "", "images", "security-high", "security-medium", "security-safe"
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
	iconHelper          *SecurityIconHelper
	pinIconHelper       *PinIconHelper
	useBasicColors      bool // Track if we should use basic colors only

	// Syntax highlighting
	codeDetector *CodeDetector

	// Help screen state
	helpScrollOffset int
	helpViewport     viewport.Model
	helpViewportReady bool

	// Text viewer state
	viewingText        *storage.ClipboardItem
	textViewport       viewport.Model
	textViewportReady  bool
	textDeletePending  bool // Track if delete confirmation is pending in text view
	imageDeletePending bool // Track if delete confirmation is pending in image view

	// Security viewer state  
	securityDeletePending bool // Track if delete confirmation is pending in security view
	securityScrollOffset  int  // Track scroll position in security content
	securityViewport     viewport.Model
	securityViewportReady bool
	
	// Warning dialog state
	warningViewport      viewport.Model
	warningViewportReady bool
	
	// Theme service for comprehensive styling
	themeService *ThemeService
}


func NewModel(s *storage.Storage, cfg *config.Config, basicTerminal bool) Model {
	// Create memory-efficient cache (cache up to 20 images by default)
	cache := storage.NewItemCache(s, 20)
	items := cache.GetAllMeta()
	hashStore, _ := security.NewHashStore() // Initialize security hash store
	iconHelper := NewSecurityIconHelper(basicTerminal)   // Initialize terminal detection
	pinIconHelper := NewPinIconHelper(basicTerminal)     // Initialize pin icon helper
	useBasicColors := !iconHelper.GetCapabilities().SupportsColor
	codeDetector := NewCodeDetector() // Initialize syntax highlighting
	themeService := NewThemeService(&cfg.Theme) // Initialize theme service

	model := Model{
		storage:        s,
		config:         cfg,
		cache:          cache,
		items:          items,
		filteredItems:  items,
		cursor:         0,
		currentMode:    modeList,
		hashStore:      hashStore,
		iconHelper:     iconHelper,
		pinIconHelper:  pinIconHelper,
		useBasicColors: useBasicColors,
		codeDetector:   codeDetector,
		themeService:   themeService,
	}
	
	// Initial preload of images around cursor
	go model.preloadImagesAroundCursor()
	
	return model
}

func (m Model) Init() tea.Cmd {
	return nil
}

// getItemByIndex returns a full ClipboardItem for the given filtered index
func (m *Model) getItemByIndex(index int) *storage.ClipboardItem {
	if index < 0 || index >= len(m.filteredItems) {
		return nil
	}
	meta := m.filteredItems[index]
	return m.cache.GetFullItem(meta.ID)
}

// getCurrentItem returns the currently selected full ClipboardItem
func (m *Model) getCurrentItem() *storage.ClipboardItem {
	return m.getItemByIndex(m.cursor)
}

// refreshItems reloads the items list and applies current filters
func (m *Model) refreshItems() {
	m.items = m.cache.GetAllMeta()
	m.filterItems()
}

// applyFilters is an alias for filterItems to maintain compatibility
func (m *Model) applyFilters() {
	m.filterItems()
}

// getItemMeta returns metadata for the given filtered index  
func (m *Model) getItemMeta(index int) *storage.ClipboardItemMeta {
	if index < 0 || index >= len(m.filteredItems) {
		return nil
	}
	return &m.filteredItems[index]
}

// preloadImagesAroundCursor preloads image data for items around the current cursor
func (m *Model) preloadImagesAroundCursor() {
	const bufferSize = 10 // Preload ±10 items around cursor
	
	var imageIDs []string
	start := m.cursor - bufferSize
	end := m.cursor + bufferSize
	
	if start < 0 {
		start = 0
	}
	if end >= len(m.filteredItems) {
		end = len(m.filteredItems) - 1
	}
	
	// Collect image IDs in the buffer range
	for i := start; i <= end; i++ {
		if i < len(m.filteredItems) {
			item := &m.filteredItems[i]
			if item.ContentType == "image" {
				imageIDs = append(imageIDs, item.ID)
			}
		}
	}
	
	// Preload the images (this will cache them)
	if len(imageIDs) > 0 {
		m.cache.PreloadImageData(imageIDs)
	}
}

// evictImagesOutsideBuffer removes cached images that are far from cursor
func (m *Model) evictImagesOutsideBuffer() {
	const bufferSize = 10 // Keep ±10 items around cursor
	const evictionDistance = bufferSize * 2 // Evict items more than 20 positions away
	
	// Find items to evict (those far from cursor)
	for i, item := range m.filteredItems {
		if item.ContentType == "image" {
			distance := abs(i - m.cursor)
			if distance > evictionDistance {
				m.cache.EvictImageData(item.ID)
			}
		}
	}
}

// abs returns absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}


// parseColor converts various color formats to lipgloss.Color with basic terminal fallback
func (m Model) parseColor(colorStr string) lipgloss.Color {
	if colorStr == "" {
		return lipgloss.Color("")
	}

	// If using basic colors, map advanced colors to basic ANSI colors
	if m.useBasicColors {
		return m.mapToBasicColor(colorStr)
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

// mapToBasicColor maps advanced colors to basic ANSI colors for limited terminals
func (m Model) mapToBasicColor(colorStr string) lipgloss.Color {
	// Return empty color if not specified
	if colorStr == "" {
		return lipgloss.Color("")
	}

	colorStr = strings.ToLower(colorStr)

	// Map hex colors and complex colors to basic ANSI colors
	basicColorMap := map[string]string{
		// Direct ANSI colors (keep as-is if they're basic)
		"0": "0", "1": "1", "2": "2", "3": "3", "4": "4", "5": "5", "6": "6", "7": "7",
		"10": "10", "11": "11", "12": "12", "14": "14",

		// CSS color names to basic ANSI
		"black":     "0",
		"red":       "1",
		"green":     "2",
		"yellow":    "3",
		"blue":      "4",
		"magenta":   "5",
		"cyan":      "6",
		"white":     "7",
		"gray":      "8",
		"grey":      "8",
		"darkred":   "1",
		"darkgreen": "2",
		"darkblue":  "4",
		"orange":    "3", // Map to yellow
		"purple":    "5", // Map to magenta
		"pink":      "5", // Map to magenta
		"brown":     "1", // Map to red
		"lime":      "2", // Map to green
		"navy":      "4", // Map to blue
		"maroon":    "1", // Map to red
		"olive":     "3", // Map to yellow
		"teal":      "6", // Map to cyan
		"silver":    "7", // Map to white
		"gold":      "3", // Map to yellow

		// Complex color codes - map to closest basic color
		"234": "0", "235": "0", "236": "8", "237": "8", "238": "8", "239": "8",
		"13": "5",   // Default header color -> magenta
		"8": "8",    // Default status color -> gray  
		"141": "5",  // Default search color -> magenta
		"9": "1",    // Default warning color -> red
		"15": "7",   // Default selected foreground -> white
		"55": "4",   // Default selected background -> blue
		"39": "6",   // Default frame border -> cyan
	}

	// Check if it's a hex color and map to nearest basic color
	if strings.HasPrefix(colorStr, "#") {
		return m.mapHexToBasicColor(colorStr)
	}

	// Look up in basic color map
	if basicColor, exists := basicColorMap[colorStr]; exists {
		return lipgloss.Color(basicColor)
	}

	// For unknown colors, return empty to use terminal default
	return lipgloss.Color("")
}

// mapHexToBasicColor maps hex colors to the nearest basic ANSI color
func (m Model) mapHexToBasicColor(hexColor string) lipgloss.Color {
	// Simple mapping of common hex colors to basic ANSI
	hexToBasic := map[string]string{
		"#000000": "0", // black
		"#800000": "1", // red
		"#008000": "2", // green
		"#808000": "3", // yellow
		"#000080": "4", // blue
		"#800080": "5", // magenta
		"#008080": "6", // cyan
		"#c0c0c0": "7", // white
		"#808080": "8", // gray
		"#ff0000": "1", // bright red
		"#00ff00": "2", // bright green
		"#ffff00": "3", // bright yellow
		"#0000ff": "4", // bright blue
		"#ff00ff": "5", // bright magenta
		"#00ffff": "6", // bright cyan
		"#ffffff": "7", // bright white
	}

	if basicColor, exists := hexToBasic[strings.ToLower(hexColor)]; exists {
		return lipgloss.Color(basicColor)
	}

	// For unknown hex colors, try to determine closest basic color
	// This is a simplified approach - just return empty for terminal default
	return lipgloss.Color("")
}

// createStyle creates a lipgloss style from a ColorConfig without background
func (m Model) createStyle(colorCfg config.ColorConfig) lipgloss.Style {
	style := lipgloss.NewStyle()

	if colorCfg.Foreground != "" {
		style = style.Foreground(m.parseColor(colorCfg.Foreground))
	}
	// Background colors removed to prevent interference with syntax highlighting
	if colorCfg.Bold {
		style = style.Bold(true)
	}

	return style
}

// createSelectedStyle creates a lipgloss style for selected items (with background)
func (m Model) createSelectedStyle(colorCfg config.ColorConfig) lipgloss.Style {
	style := lipgloss.NewStyle()

	if colorCfg.Foreground != "" {
		style = style.Foreground(m.parseColor(colorCfg.Foreground))
	}
	if colorCfg.Background != "" {
		style = style.Background(m.parseColor(colorCfg.Background))
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
		
		// Initialize viewports based on current mode
		if m.currentMode == modeTextView && m.viewingText != nil {
			m.initTextViewport()
		} else if m.currentMode == modeHelp {
			m.initHelpViewport()
		} else if m.currentMode == modeSecurityWarning {
			m.initSecurityViewport()
		} else if m.currentMode == modeImageSecurityWarning {
			m.initWarningViewport()
		}
		
		return m, nil

	case editCompleteMsg:
		// Refresh items after editing
		oldCursor := m.cursor
		editedID := msg.editedItemID

		m.cache.ForceRefresh()
		m.items = m.cache.GetAllMeta()
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
			// In security warning mode - use same logic as text view
			contentLines := m.getSecurityViewLines()
			_, _, _, contentHeight := m.calculateDialogDimensions()
			if contentHeight < 5 {
				contentHeight = 5
			}
			maxScrollOffset := len(contentLines) - contentHeight
			if maxScrollOffset < 0 {
				maxScrollOffset = 0
			}

			switch msg.String() {
			case "up", "k", "down", "j", "pgup", "pgdown", "home", "end":
				// Let viewport handle scrolling
				if !m.securityDeletePending && m.securityViewportReady {
					m.securityViewport, _ = m.securityViewport.Update(msg)
				}
				return m, nil
			case "s":
				// Mark as safe
				if m.securityItem != nil {
					m.storage.UpdateSafeEntry(m.securityItem.ID, true)
					// Refresh items list
					m.refreshItems()
					// Update the security item with new status
					updatedItem := m.storage.GetByID(m.securityItem.ID)
					if updatedItem != nil {
						m.securityItem = updatedItem
					}
				}
				// Exit security view after marking
				m.currentMode = modeList
				m.securityContent = ""
				m.securityThreats = nil
				m.securityItem = nil
				m.securityDeletePending = false
				m.securityViewportReady = false
				return m, nil
			case "u":
				// Mark as unsafe
				if m.securityItem != nil {
					m.storage.UpdateSafeEntry(m.securityItem.ID, false)
					// Refresh items list
					m.refreshItems()
					// Update the security item with new status
					updatedItem := m.storage.GetByID(m.securityItem.ID)
					if updatedItem != nil {
						m.securityItem = updatedItem
					}
				}
				// Exit security view after marking
				m.currentMode = modeList
				m.securityContent = ""
				m.securityThreats = nil
				m.securityItem = nil
				m.securityDeletePending = false
				m.securityViewportReady = false
				return m, nil
			case "x":
				if m.securityDeletePending {
					// Confirm deletion
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
					m.refreshItems()
					if m.cursor >= len(m.filteredItems) && len(m.filteredItems) > 0 {
						m.cursor = len(m.filteredItems) - 1
					} else if len(m.filteredItems) == 0 {
						m.cursor = 0
					}

					m.currentMode = modeList
					m.securityContent = ""
					m.securityThreats = nil
					m.securityItem = nil
					m.securityDeletePending = false
					m.securityViewportReady = false
					return m, nil
				} else {
					// First 'x' press - enter delete confirmation mode
					m.securityDeletePending = true
					return m, nil
				}
			case "ctrl+c":
				return m, tea.Quit
			case "q":
				// q should exit view, not application 
				m.currentMode = modeList
				m.securityContent = ""
				m.securityThreats = nil
				m.securityItem = nil
				m.securityDeletePending = false
				m.securityViewportReady = false
				return m, nil
			default:
				// Any other key cancels delete confirmation or exits view
				if m.securityDeletePending {
					m.securityDeletePending = false
					return m, nil
				} else {
					// Exit security view only if not a scroll key for short content
					m.currentMode = modeList
					m.securityContent = ""
					m.securityThreats = nil
					m.securityItem = nil
					m.securityDeletePending = false
					m.securityViewportReady = false
					return m, nil
				}
			}
		} else if m.currentMode == modeHelp {
			// In help mode - calculate scroll bounds first
			helpLines := m.generateHelpContent()
			_, _, _, contentHeight := m.calculateDialogDimensions()
			if contentHeight < 5 {
				contentHeight = 5
			}
			maxScrollOffset := len(helpLines) - contentHeight
			if maxScrollOffset < 0 {
				maxScrollOffset = 0
			}

			switch msg.String() {
			case "ctrl+c", "q", "esc", "?":
				// Exit help mode
				m.currentMode = modeList
				m.helpViewportReady = false
				return m, nil
			case "up", "k", "down", "j", "pgup", "pgdown", "home", "end":
				// Let viewport handle scrolling
				if m.helpViewportReady {
					m.helpViewport, _ = m.helpViewport.Update(msg)
				}
			default:
				// Any other key exits help
				m.currentMode = modeList
				m.helpViewportReady = false
				return m, nil
			}
		} else if m.currentMode == modeTextView {
			// In text view mode - calculate scroll bounds first
			if m.viewingText == nil {
				m.currentMode = modeList
				return m, nil
			}

			textLines := m.getTextViewLines()
			_, _, _, contentHeight := m.calculateDialogDimensions()
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
				m.textViewportReady = false
				m.textDeletePending = false
				return m, nil
			case "up", "k", "down", "j", "pgup", "pgdown", "home", "end":
				// Let viewport handle scrolling
				if m.textViewportReady {
					m.textViewport, _ = m.textViewport.Update(msg)
				}
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
			case "x":
				// Delete text from database with confirmation
				if m.viewingText != nil {
					if m.textDeletePending {
						// Second press - confirm deletion
						err := m.storage.Delete(m.viewingText.ID)
						if err == nil {
							m.refreshItems()
							// Adjust cursor if needed
							if m.cursor >= len(m.filteredItems) && len(m.filteredItems) > 0 {
								m.cursor = len(m.filteredItems) - 1
							} else if len(m.filteredItems) == 0 {
								m.cursor = 0
							}
						}
						// Exit text view after deletion
						m.currentMode = modeList
						m.viewingText = nil
						m.textViewportReady = false
						m.textDeletePending = false
						return m, nil
					} else {
						// First press - show confirmation
						m.textDeletePending = true
						return m, nil
					}
				}
				return m, nil
			case "s":
				// Mark as safe - only available for items with security warnings
				if m.viewingText != nil && (m.viewingText.ThreatLevel == "high" || m.viewingText.ThreatLevel == "medium") {
					err := m.storage.UpdateSafeEntry(m.viewingText.ID, true)
					if err == nil {
						// Update the current viewing item
						m.viewingText.SafeEntry = true
						m.viewingText.ThreatLevel = "none" // Clear threat level when marked as safe
						
						// Update the main items list
						m.refreshItems()
					}
				}
				return m, nil
			default:
				// Any other key exits text view (or cancels delete confirmation)
				m.currentMode = modeList
				m.viewingText = nil
				m.textViewportReady = false
				m.textDeletePending = false
				return m, nil
			}
		} else if m.currentMode == modeImageView {
			// In image view mode
			switch msg.String() {
			case "ctrl+c", "q", "esc":
				// Exit image view mode - clear Kitty graphics only
				// Let Bubble Tea handle terminal state management
				fmt.Print("\x1b_Ga=d;\x1b\\") // Delete all Kitty images

				m.currentMode = modeList
				m.viewingImage = nil
				m.imageDeletePending = false
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
			case "o":
				// Open in external image viewer
				if m.viewingImage != nil {
					return m, m.openImageInViewer(*m.viewingImage)
				}
				return m, nil
			case "x":
				// Delete image from database with confirmation
				if m.viewingImage != nil {
					if m.imageDeletePending {
						// Second press - confirm deletion
						err := m.storage.Delete(m.viewingImage.ID)
						if err == nil {
							m.refreshItems()
							// Adjust cursor if needed
							if m.cursor >= len(m.filteredItems) && len(m.filteredItems) > 0 {
								m.cursor = len(m.filteredItems) - 1
							} else if len(m.filteredItems) == 0 {
								m.cursor = 0
							}
						}
						// Clear Kitty graphics and exit image view after deletion
						fmt.Print("\x1b_Ga=d;\x1b\\") // Delete all Kitty images
						m.currentMode = modeList
						m.viewingImage = nil
						m.imageDeletePending = false
						return m, nil
					} else {
						// First press - show confirmation
						m.imageDeletePending = true
						return m, nil
					}
				}
				return m, nil
			default:
				// Any other key exits image view (or cancels delete confirmation)
				// Let Bubble Tea handle terminal state management
				fmt.Print("\x1b_Ga=d;\x1b\\") // Delete all Kitty images

				m.currentMode = modeList
				m.viewingImage = nil
				m.imageDeletePending = false
				return m, nil
			}
		} else if m.currentMode == modeImageSecurityWarning {
			// In image security warning mode - allow scrolling or dismiss
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "up", "k", "down", "j", "pgup", "pgdown", "home", "end":
				// Let viewport handle scrolling
				if m.warningViewportReady {
					m.warningViewport, _ = m.warningViewport.Update(msg)
				}
			default:
				// Any other key dismisses the warning and returns to list mode
				m.currentMode = modeList
				m.warningViewportReady = false
				return m, nil
			}
		} else if m.currentMode == modeConfirmDelete {
			// In delete confirmation mode
			switch msg.String() {
			case "x":
				// Confirm delete by pressing 'x' again
				if m.deleteCandidate != nil {
					err := m.storage.Delete(m.deleteCandidate.ID)
					if err == nil {
						m.refreshItems()
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
					selectedItem := m.getCurrentItem()
					if selectedItem == nil {
						return m, nil
					}
					err := clipboard.Copy(selectedItem.Content)
					if err != nil {
						return m, nil
					}
					return m, tea.Quit
				}

			case "v":
				// View entry in full-screen (images or text)
				if len(m.filteredItems) > 0 && m.cursor < len(m.filteredItems) {
					selectedItem := m.getCurrentItem()
					if selectedItem == nil {
						return m, nil
					}
					if selectedItem.ContentType == "image" {
						m.viewingImage = selectedItem
						m.currentMode = modeImageView
						return m, nil
					} else {
						m.viewingText = selectedItem
						m.textViewportReady = false
						m.currentMode = modeTextView
						return m, nil
					}
				}

			case "e":
				if len(m.filteredItems) > 0 && m.cursor < len(m.filteredItems) {
					selectedItem := m.getCurrentItem()
					if selectedItem == nil {
						return m, nil
					}
					if selectedItem.ContentType == "image" {
						return m, m.editImage(*selectedItem)
					} else {
						return m, m.editEntry(*selectedItem)
					}
				}

			case "x":
				if len(m.filteredItems) > 0 && m.cursor < len(m.filteredItems) {
					selectedItem := m.getCurrentItem()
					if selectedItem == nil {
						return m, nil
					}
					m.deleteCandidate = selectedItem
					m.currentMode = modeConfirmDelete
					return m, nil
				}

			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
					// Preload images around new cursor position
					go m.preloadImagesAroundCursor()
				}

			case "down", "j":
				if m.cursor < len(m.filteredItems)-1 {
					m.cursor++
					// Preload images around new cursor position
					go m.preloadImagesAroundCursor()
				}

			case "pgup":
				// Page up - jump to item roughly one screen height up
				if len(m.filteredItems) > 0 {
					// Calculate content area height
					_, _, _, contentHeight := m.calculateDialogDimensions()
					if contentHeight < 5 {
						contentHeight = 5
					}

					// Calculate how many items typically fit on screen
					// Conservative estimate: assume each item takes 2-3 lines on average
					itemsPerPage := contentHeight / 3
					if itemsPerPage < 1 {
						itemsPerPage = 1
					}

					newCursor := m.cursor - itemsPerPage
					if newCursor < 0 {
						newCursor = 0
					}
					m.cursor = newCursor
					// Preload images around new cursor position
					go m.preloadImagesAroundCursor()
				}

			case "pgdown":
				// Page down - jump to item roughly one screen height down
				if len(m.filteredItems) > 0 {
					// Calculate content area height
					_, _, _, contentHeight := m.calculateDialogDimensions()
					if contentHeight < 5 {
						contentHeight = 5
					}

					// Calculate how many items typically fit on screen
					// Conservative estimate: assume each item takes 2-3 lines on average
					itemsPerPage := contentHeight / 3
					if itemsPerPage < 1 {
						itemsPerPage = 1
					}

					newCursor := m.cursor + itemsPerPage
					if newCursor >= len(m.filteredItems) {
						newCursor = len(m.filteredItems) - 1
					}
					m.cursor = newCursor
					// Preload images around new cursor position
					go m.preloadImagesAroundCursor()
				}

			case "g":
				// Go to first item
				if len(m.filteredItems) > 0 {
					m.cursor = 0
					// Preload images around new cursor position
					go m.preloadImagesAroundCursor()
				}

			case "G":
				// Go to last item
				if len(m.filteredItems) > 0 {
					m.cursor = len(m.filteredItems) - 1
					// Preload images around new cursor position
					go m.preloadImagesAroundCursor()
				}

			case "ctrl+s":
				// Scan current item for security issues
				if len(m.filteredItems) > 0 && m.cursor < len(m.filteredItems) {
					selectedItem := m.getCurrentItem()
					if selectedItem == nil {
						return m, nil
					}
					if selectedItem.ContentType != "image" {
						detector := security.NewSecurityDetector()
						threats := detector.DetectSecurity(selectedItem.Content)
						// Always show security dialog, even if no threats found
						m.ShowSecurityWarning(*selectedItem, threats)
						return m, nil
					} else {
						// Show warning that security scanning is not available for images
						m.currentMode = modeImageSecurityWarning
						m.warningViewportReady = false // Reset viewport state
						return m, nil
					}
				} else {
					// For testing - show image warning even if no items
					m.currentMode = modeImageSecurityWarning
					m.warningViewportReady = false // Reset viewport state
					return m, nil
				}

			case "i":
				// Toggle image filter
				if m.filterMode == "images" {
					m.filterMode = "" // Clear filter
				} else {
					m.filterMode = "images" // Show only images
				}
				m.filterItems()
				return m, nil
				
			case "h":
				// Toggle high-risk security filter
				if m.filterMode == "security-high" {
					m.filterMode = "" // Clear filter
				} else {
					m.filterMode = "security-high" // Show only high-risk items
				}
				m.filterItems()
				return m, nil
				
			case "m":
				// Toggle medium-risk security filter
				if m.filterMode == "security-medium" {
					m.filterMode = "" // Clear filter
				} else {
					m.filterMode = "security-medium" // Show only medium-risk items
				}
				m.filterItems()
				return m, nil
				
			case "s":
				// Toggle safe security filter
				if m.filterMode == "security-safe" {
					m.filterMode = "" // Clear filter
				} else {
					m.filterMode = "security-safe" // Show only safe items
				}
				m.filterItems()
				return m, nil

			case "p":
				// Toggle pin/unpin for current item
				if len(m.filteredItems) > 0 && m.cursor < len(m.filteredItems) {
					selectedItem := m.getCurrentItem()
					if selectedItem == nil {
						return m, nil
					}
					
					if selectedItem.IsPinned {
						// Unpin the item
						err := m.storage.UnpinItem(selectedItem.ID)
						if err == nil {
							// Force cache refresh and update UI
							m.cache.ForceRefresh()
							m.items = m.cache.GetAllMeta()
							m.filterItems()
						}
					} else {
						// Pin the item
						err := m.storage.PinItem(selectedItem.ID)
						if err == nil {
							// Force cache refresh and update UI
							m.cache.ForceRefresh()
							m.items = m.cache.GetAllMeta()
							m.filterItems()
						}
					}
				}
				return m, nil

			case "1", "2", "3", "4", "5", "6", "7", "8", "9", "0":
				// Quick copy pinned items
				pinIndex := 0
				switch msg.String() {
				case "1":
					pinIndex = 1
				case "2":
					pinIndex = 2
				case "3":
					pinIndex = 3
				case "4":
					pinIndex = 4
				case "5":
					pinIndex = 5
				case "6":
					pinIndex = 6
				case "7":
					pinIndex = 7
				case "8":
					pinIndex = 8
				case "9":
					pinIndex = 9
				case "0":
					pinIndex = 10
				}
				
				pinnedItems := m.storage.GetPinnedItems()
				if pinIndex > 0 && pinIndex <= len(pinnedItems) {
					selectedItem := pinnedItems[pinIndex-1]
					fullItem := m.storage.GetByID(selectedItem.ID)
					if fullItem != nil {
						var err error
						if fullItem.ContentType == "image" && len(fullItem.ImageData) > 0 {
							err = clipboard.CopyImage(fullItem.ImageData)
						} else {
							err = clipboard.Copy(fullItem.Content)
						}
						if err == nil {
							return m, tea.Quit
						}
					}
				}
				return m, nil

			case "?":
				// Show help screen
				m.currentMode = modeHelp
				m.helpViewportReady = false
				return m, nil
			}
		}
	}

	return m, nil
}

func (m *Model) filterItems() {
	// Start with all items
	items := m.items
	
	// Apply content filtering first
	if m.filterMode != "" {
		items = m.applyContentFilter(items)
	}
	
	// Apply search query if present
	if m.searchQuery == "" {
		m.filteredItems = items
	} else {
		m.filteredItems = m.applySearchFilter(items)
	}
	
	// Reset cursor if it's out of bounds
	if m.cursor >= len(m.filteredItems) {
		m.cursor = 0
	}
}

// applyContentFilter filters items based on content type or security status
func (m *Model) applyContentFilter(items []storage.ClipboardItemMeta) []storage.ClipboardItemMeta {
	var filtered []storage.ClipboardItemMeta
	
	switch m.filterMode {
	case "images":
		// Show only images
		for _, item := range items {
			if item.ContentType == "image" {
				filtered = append(filtered, item)
			}
		}
	case "security-high":
		// Show only high-risk security items
		for _, item := range items {
			if item.ThreatLevel == "high" {
				filtered = append(filtered, item)
			}
		}
	case "security-medium":
		// Show only medium-risk security items
		for _, item := range items {
			if item.ThreatLevel == "medium" {
				filtered = append(filtered, item)
			}
		}
	case "security-safe":
		// Show only safe items (marked as safe but with threat level)
		for _, item := range items {
			if item.SafeEntry && item.ThreatLevel != "none" {
				filtered = append(filtered, item)
			}
		}
	default:
		return items
	}
	
	return filtered
}

// applySearchFilter applies fuzzy search filtering to items
func (m *Model) applySearchFilter(items []storage.ClipboardItemMeta) []storage.ClipboardItemMeta {
	// When searching, only include text items (images can't be searched)
	var textItems []storage.ClipboardItemMeta
	var searchTargets []string

	for _, item := range items {
		if item.ContentType != "image" {
			textItems = append(textItems, item)
			searchTargets = append(searchTargets, item.Content)
		}
	}

	matches := fuzzy.Find(m.searchQuery, searchTargets)

	// Filter out weak matches by checking if the search term actually appears in the content
	var filteredMatches []storage.ClipboardItemMeta
	lowerQuery := strings.ToLower(m.searchQuery)

	for _, match := range matches {
		item := textItems[match.Index]
		// Only include if the search term is actually contained in the content
		if strings.Contains(strings.ToLower(item.Content), lowerQuery) {
			filteredMatches = append(filteredMatches, item)
		}
	}

	return filteredMatches
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

func (m *Model) openImageInViewer(item storage.ClipboardItem) tea.Cmd {
	if len(item.ImageData) == 0 {
		return tea.Cmd(func() tea.Msg { return nil })
	}

	return tea.Cmd(func() tea.Msg {
		// Create temporary image file
		tmpFile, err := ioutil.TempFile("", "nclip-view-*.png")
		if err != nil {
			return nil
		}

		// Write image data to temporary file
		_, writeErr := tmpFile.Write(item.ImageData)
		tmpFile.Close()
		tmpFilePath := tmpFile.Name()

		if writeErr != nil {
			os.Remove(tmpFilePath)
			return nil
		}

		// Get image viewer from config
		imageViewer := m.config.Editor.ImageViewer

		// Launch image viewer in background (non-blocking)
		cmd := exec.Command(imageViewer, tmpFilePath)
		err = cmd.Start() // Use Start() instead of Run() to not block
		if err != nil {
			os.Remove(tmpFilePath)
			return nil
		}

		// Start a background goroutine to clean up the temp file after viewer exits
		go func() {
			cmd.Wait()             // Wait for viewer to exit
			os.Remove(tmpFilePath) // Clean up temp file
		}()

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

	if m.currentMode == modeImageSecurityWarning {
		return m.renderImageSecurityWarning()
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
	// Use standard dialog dimensions (consistent with all other views)
	dialogWidth, dialogHeight, contentWidth, contentHeight := m.calculateDialogDimensions()

	// Create header text
	var headerText string
	if m.currentMode == modeSearch {
		// In search mode, always show filter with cursor
		headerText = "Clipboard Manager - Filter: " + m.searchQuery + "█"
	} else if m.searchQuery != "" {
		// Has active filter but not in search mode
		headerText = "Clipboard Manager - Filter: " + m.searchQuery + " (press 'c' to clear)"
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

	// Build main content area (scrolling content only)
	mainContent := m.buildMainContent(contentWidth, contentHeight)

	// Create footer text
	var footerText string
	switch m.currentMode {
	case modeConfirmDelete:
		footerText = "Press 'x' again to delete, any other key to cancel"
	case modeSearch:
		footerText = "type filter text | enter: apply filter | esc: cancel"
	default:
		// Build base footer text
		baseFooter := "enter: copy | x: delete | v: view | e: edit | ?: help"
		
		// Add filter status if active using proper formatting
		var filterIndicator string
		switch m.filterMode {
		case "images":
			filterIndicator = "[IMAGES ONLY]"
		case "security-high":
			filterIndicator = "[HIGH RISK ONLY]"
		case "security-medium":
			filterIndicator = "[MEDIUM RISK ONLY]"
		case "security-safe":
			filterIndicator = "[SAFE ITEMS ONLY]"
		}
		
		// Build footer text properly
		if filterIndicator != "" {
			footerText = baseFooter + " | " + filterIndicator
		} else {
			footerText = baseFooter
		}
	}

	// Build frame content using shared function (consistent with all other views)
	frameContent := m.buildFrameContent(headerText, mainContent, footerText, contentWidth)

	// Create framed dialog using standard function (consistent with all other views)
	return m.createFramedDialog(dialogWidth, dialogHeight, frameContent)
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


// renderSimpleImageView for terminals that don't support Kitty graphics
func (m Model) renderSimpleImageView() string {
	headerText := "Image View - Not Supported"

	// Use standard dialog dimensions (consistent with all other views)
	dialogWidth, dialogHeight, contentWidth, contentHeight := m.calculateDialogDimensions()

	// Build content lines
	contentLines := []string{
		"Your terminal does not support the Kitty graphics protocol.",
		"Supported terminals: Kitty, Ghostty, WezTerm, Konsole",
		"",
		"Available actions:",
		fmt.Sprintf("'o' open image in external viewer (%s)", m.config.Editor.ImageViewer),
		"'enter' copy image to clipboard",
		"'e' edit in external editor",
		"'d' delete image from database",
		"",
		fmt.Sprintf("Image: %d bytes", len(m.viewingImage.ImageData)),
	}

	// Build content and pad to fill content area (like help view does)
	var contentBuilder strings.Builder
	for _, line := range contentLines {
		contentBuilder.WriteString(line)
		contentBuilder.WriteString("\n")
	}

	// Pad to fill content area - add one extra line to push footer to bottom
	currentLines := len(contentLines)
	targetLines := contentHeight // Fill completely to push footer to very bottom
	for currentLines < targetLines {
		contentBuilder.WriteString("\n")
		currentLines++
	}

	// Create footer text based on delete confirmation state
	var footerText string
	if m.imageDeletePending {
		footerText = "Press 'x' again to confirm deletion, any other key to cancel"
	} else {
		footerText = "enter: copy | x: delete | e: edit | o: open"
	}

	// Build frame content using standard function (like help view)
	frameContent := m.buildFrameContent(headerText, contentBuilder.String(), footerText, contentWidth)
	return m.createFramedDialog(dialogWidth, dialogHeight, frameContent)
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
	m.securityDeletePending = false // Reset delete confirmation state
	m.securityViewportReady = false     // Reset scroll position
	m.currentMode = modeSecurityWarning
}

// highlightSecurityThreats adds highlighting to content where security threats are detected
func (m Model) highlightSecurityThreats(content string) string {
	if len(m.securityThreats) == 0 {
		return content // No threats to highlight
	}

	// Get the highest threat for highlighting
	threat := security.GetHighestThreat(m.securityThreats)
	if threat == nil {
		return content
	}

	// Define highlighting colors
	warningStyle := "\x1b[91m\x1b[1m" // Bold red
	resetStyle := "\x1b[0m"

	// Try to find and highlight specific patterns within the content
	highlighted := m.findAndHighlightPatterns(content, threat.Type, warningStyle, resetStyle)
	if highlighted != content {
		return highlighted // Found and highlighted specific patterns
	}

	// Fallback: if content is short and looks like a single credential, highlight it all
	trimmed := strings.TrimSpace(content)
	if len(strings.Fields(trimmed)) <= 3 && len(trimmed) >= 8 && len(trimmed) <= 100 {
		return warningStyle + content + resetStyle
	}

	return content
}

// findAndHighlightPatterns finds specific security patterns within content and highlights them
func (m Model) findAndHighlightPatterns(content, threatType, warningStyle, resetStyle string) string {
	switch threatType {
	case "jwt":
		// Find JWT tokens (3 base64 parts separated by dots)
		re := regexp.MustCompile(`[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`)
		return re.ReplaceAllStringFunc(content, func(match string) string {
			return warningStyle + match + resetStyle
		})

	case "password":
		// Look for password=value or password: value patterns first
		re := regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*([^\s\n]+)`)
		if re.MatchString(content) {
			return re.ReplaceAllStringFunc(content, func(match string) string {
				parts := re.FindStringSubmatch(match)
				if len(parts) >= 3 {
					separator := ":"
					if strings.Contains(match, "=") {
						separator = "="
					}
					return parts[1] + separator + " " + warningStyle + parts[2] + resetStyle
				}
				return match
			})
		}
		// Look for potential passwords (8+ chars with mixed character types)
		words := strings.Fields(content)
		for _, word := range words {
			if m.looksLikePassword(word) {
				content = strings.Replace(content, word, warningStyle+word+resetStyle, 1)
				return content
			}
		}

	case "api_key", "token":
		// Look for API key patterns
		patterns := []string{
			`[A-Za-z0-9_-]{32,}`,     // Generic long alphanumeric strings
			`[A-Za-z0-9+/]{40,}=*`,   // Base64-like strings
			`sk_[a-zA-Z0-9_-]{24,}`,  // Stripe-like keys
			`pk_[a-zA-Z0-9_-]{24,}`,  // Public keys
		}
		for _, pattern := range patterns {
			re := regexp.MustCompile(pattern)
			matches := re.FindAllString(content, -1)
			for _, match := range matches {
				if len(match) >= 20 { // Only highlight reasonably long tokens
					content = strings.Replace(content, match, warningStyle+match+resetStyle, 1)
					return content
				}
			}
		}

	case "ssh_key", "private_key":
		// Highlight key blocks
		patterns := []string{
			`-----BEGIN [A-Z ]+-----[^-]+-----END [A-Z ]+-----`,
			`ssh-(rsa|dss|ed25519|ecdsa) [A-Za-z0-9+/]+=*`,
		}
		for _, pattern := range patterns {
			re := regexp.MustCompile(pattern)
			return re.ReplaceAllStringFunc(content, func(match string) string {
				return warningStyle + match + resetStyle
			})
		}

	case "connection_string":
		// Highlight password in connection strings
		re := regexp.MustCompile(`://([^:]+):([^@]+)@`)
		return re.ReplaceAllStringFunc(content, func(match string) string {
			parts := re.FindStringSubmatch(match)
			if len(parts) >= 3 {
				return "://" + parts[1] + ":" + warningStyle + parts[2] + resetStyle + "@"
			}
			return match
		})
	}

	return content
}

// looksLikePassword checks if a word has password-like characteristics
func (m Model) looksLikePassword(word string) bool {
	if len(word) < 8 || len(word) > 128 {
		return false
	}

	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false

	for _, char := range word {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasDigit = true
		case strings.ContainsRune("!@#$%^&*()_+-=[]{}|;:,.<>?", char):
			hasSpecial = true
		}
	}

	// Password-like if it has at least 3 of the 4 character types
	score := 0
	if hasUpper { score++ }
	if hasLower { score++ }
	if hasDigit { score++ }
	if hasSpecial { score++ }

	return score >= 3
}

// getSecurityViewLines returns the content lines for scrolling (like getTextViewLines)
func (m Model) getSecurityViewLines() []string {
	// This should ONLY return the content that scrolls, not static headers/footers
	// Follow the exact pattern of getTextViewLines()
	
	content := m.securityContent
	if content == "" {
		return []string{"[No content available]"}
	}

	// Apply security highlighting if threats are detected
	if len(m.securityThreats) > 0 {
		content = m.highlightSecurityThreats(content)
	}

	// Split content into lines and apply theme styling
	rawLines := strings.Split(content, "\n")
	warningStyles := m.themeService.GetWarningStyles()
	lines := make([]string, len(rawLines))
	for i, line := range rawLines {
		lines[i] = warningStyles.Body.Render(line)
	}

	// Use standard dialog dimensions for consistent content width
	_, _, contentWidth, _ := m.calculateDialogDimensions()
	if contentWidth < 10 {
		contentWidth = 10
	}

	// Wrap long lines (same logic as text view)
	var wrappedLines []string
	for _, line := range lines {
		// Calculate visible length (excluding ANSI escape codes)
		visibleLen := m.calculateVisibleLength(line)
		
		if visibleLen <= contentWidth {
			wrappedLines = append(wrappedLines, line)
		} else {
			// Wrap long lines for regular text (same as text view)
			wrappedLines = append(wrappedLines, m.wrapLongLine(line, contentWidth)...)
		}
	}

	return wrappedLines
}

// renderSecurityWarning renders the security warning dialog using exact text view pattern
func (m Model) renderSecurityWarning() string {
	// Ensure minimum terminal size - same as text view
	if m.width < 10 || m.height < 8 {
		return "Terminal too small for security viewer"
	}

	if m.securityItem == nil {
		return "No security item to view"
	}

	// Use standard dialog dimensions (consistent with all other views)
	dialogWidth, dialogHeight, _, _ := m.calculateDialogDimensions()

	// Calculate content area within the dialog - same as text view
	contentWidth := dialogWidth - 4   // Border + internal padding
	contentHeight := dialogHeight - 4 // Border + header + footer

	if contentWidth < 10 {
		contentWidth = 10
	}
	if contentHeight < 5 {
		contentHeight = 5
	}

	// Initialize viewport if not ready
	if !m.securityViewportReady {
		m.initSecurityViewport()
	}

	// Create header with ALL security info - same pattern as text view
	lineCount := len(strings.Split(m.securityContent, "\n"))
	charCount := len(m.securityContent)
	
	// Build header with security status info (like text view puts security info in header)
	var headerText string
	if len(m.securityThreats) > 0 {
		threat := security.GetHighestThreat(m.securityThreats)
		if threat != nil {
			headerText = fmt.Sprintf("SECURITY WARNING - %s (%.0f%% confidence) - %s (%d lines, %d chars)",
				strings.ToUpper(threat.Type), threat.Confidence*100,
				strings.ToUpper(m.securityItem.ThreatLevel), lineCount, charCount)
		}
	} else {
		headerText = fmt.Sprintf("SECURITY SCAN - No threats detected (%d lines, %d chars)", lineCount, charCount)
	}

	// Add safe status to header
	if m.securityItem.SafeEntry {
		headerText += " - MARKED SAFE"
	} else {
		headerText += " - MARKED UNSAFE"
	}

	// Build content - same as text view
	var textContent strings.Builder
	if m.securityViewportReady {
		textContent.WriteString(m.securityViewport.View())
	} else {
		// Fallback to basic content display
		contentLines := m.getSecurityViewLines()
		for i := 0; i < contentHeight && i < len(contentLines); i++ {
			line := contentLines[i]
			// Use visible length calculation for proper truncation with ANSI codes
			visibleLen := m.calculateVisibleLength(line)
			if visibleLen > contentWidth {
				line = m.truncateWithANSI(line, contentWidth-3) + "..."
			}
			
			// Apply background styling with proper width to handle both content and padding
			warningStyles := m.themeService.GetWarningStyles()
			styledLine := warningStyles.Body.Width(contentWidth).Render(line)
			
			textContent.WriteString(styledLine)
			textContent.WriteString("\n")
		}
	}

	// Create footer text - same pattern as text view
	var scrollInfo string
	if m.securityViewportReady {
		// Get scroll position from viewport
		totalLines := len(m.getSecurityViewLines())
		if totalLines > contentHeight {
			scrollInfo = fmt.Sprintf(" - %d%%", int(float64(m.securityViewport.YOffset)/float64(totalLines-contentHeight)*100))
		}
	}

	var footerText string
	if m.securityDeletePending {
		footerText = "Press 'x' again to confirm deletion, any other key to cancel"
	} else {
		baseFooter := "enter: copy | x: delete | s: mark safe | u: mark unsafe"
		footerText = baseFooter + scrollInfo
	}

	// Build frame content using shared function - same as text view
	frameContent := m.buildFrameContent(headerText, textContent.String(), footerText, contentWidth)

	// Create framed dialog using shared function - same as text view
	positioned := m.createFramedDialog(dialogWidth, dialogHeight, frameContent)

	return positioned
}

// renderImageSecurityWarning renders a small warning when trying to scan images using viewport
func (m Model) renderImageSecurityWarning() string {
	// Ensure minimum terminal size
	if m.width < 10 || m.height < 8 {
		return "Terminal too small for warning dialog"
	}

	// Use smaller dialog dimensions for warning
	_, _, fullContentWidth, _ := m.calculateDialogDimensions()
	
	// Scale down for warning dialog
	warningWidth := min(50, fullContentWidth+4)
	warningHeight := min(10, 12) 
	warningContentWidth := warningWidth - 4
	
	// Initialize viewport if not ready (following same pattern as renderHelp)
	if !m.warningViewportReady {
		m.initWarningViewport()
	}

	// Create header text
	headerText := "Content Scanning Not Available"

	// Build warning content using viewport
	var warningContent strings.Builder
	if m.warningViewportReady {
		viewportContent := m.warningViewport.View()
		warningContent.WriteString(viewportContent)
		// Ensure content ends with newline for proper frame rendering
		if !strings.HasSuffix(viewportContent, "\n") {
			warningContent.WriteString("\n")
		}
	} else {
		// Fallback to basic warning display if viewport initialization fails
		return m.renderBasicWarning(warningWidth, warningHeight, warningContentWidth)
	}

	// Create footer text with scroll information if needed
	var scrollInfo string
	if m.warningViewportReady {
		// Simple footer for warning dialog
		scrollInfo = ""
	}
	footerText := "any key: continue" + scrollInfo

	// Build frame content using proper system
	frameContent := m.buildFrameContent(headerText, warningContent.String(), footerText, warningContentWidth)
	
	// Create framed dialog
	return m.createFramedDialog(warningWidth, warningHeight, frameContent)
}

// renderBasicWarning provides fallback rendering when viewport is not ready
func (m Model) renderBasicWarning(warningWidth, warningHeight, warningContentWidth int) string {
	// Get warning styles from theme service
	warningStyles := m.themeService.GetWarningStyles()
	
	// Build content with proper styling
	var contentLines []string
	contentLines = append(contentLines, "Security scanning is only available for text")
	contentLines = append(contentLines, "content. Images cannot be analyzed for")
	contentLines = append(contentLines, "security threats.")
	contentLines = append(contentLines, "")
	contentLines = append(contentLines, "Press any key to continue")
	
	// Apply styling to content lines
	var styledContent strings.Builder
	for i, line := range contentLines {
		if i < len(contentLines)-1 {
			// Regular content lines
			styledLine := warningStyles.Body.Width(warningContentWidth).Render(line)
			styledContent.WriteString(styledLine)
		} else {
			// Instructions line
			styledLine := warningStyles.Prompt.Width(warningContentWidth).Render(line)
			styledContent.WriteString(styledLine)
		}
		styledContent.WriteString("\n")
	}
	
	// Build frame content using proper system
	headerText := "Content Scanning Not Available"
	footerText := "any key: continue"
	frameContent := m.buildFrameContent(headerText, styledContent.String(), footerText, warningContentWidth)
	
	// Create framed dialog
	return m.createFramedDialog(warningWidth, warningHeight, frameContent)
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
		itemMeta := m.filteredItems[i]
		item := itemMeta.ToClipboardItem()
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
		itemMeta := m.filteredItems[i]
		item := itemMeta.ToClipboardItem()
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

	// Show safe marker if item has been marked as safe but has a threat level
	if item.SafeEntry && item.ThreatLevel != "none" {
		return m.iconHelper.GetSafeIcon()
	}

	// Don't show security warnings if item has been marked as safe with no threat
	if item.SafeEntry {
		return ""
	}

	// Use stored threat level for display
	switch item.ThreatLevel {
	case "high":
		return m.iconHelper.GetHighRiskIcon()
	case "medium":
		return m.iconHelper.GetMediumRiskIcon()
	case "low":
		// Show low risk with a simple icon
		return "[m]" // Simple low risk indicator
	default: // "none"
		return ""
	}
}

// getThemedSecurityIcon returns appropriate themed security icon for content
func (m Model) getThemedSecurityIcon(item storage.ClipboardItem) string {
	if item.ContentType == "image" {
		return ""
	}

	mainStyles := m.themeService.GetMainViewStyles()

	// Show safe marker if item has been marked as safe but has a threat level
	if item.SafeEntry && item.ThreatLevel != "none" {
		return m.iconHelper.GetThemedSafeIcon(mainStyles)
	}

	// Don't show security warnings if item has been marked as safe with no threat
	if item.SafeEntry {
		return ""
	}

	// Use stored threat level for display
	switch item.ThreatLevel {
	case "high":
		return m.iconHelper.GetThemedHighRiskIcon(mainStyles)
	case "medium":
		return m.iconHelper.GetThemedMediumRiskIcon(mainStyles)
	case "low":
		// Show low risk with medium risk styling
		return m.iconHelper.GetThemedMediumRiskIcon(mainStyles)
	default: // "none"
		return ""
	}
}



func (m Model) getPinIcon(item storage.ClipboardItem) string {
	if item.IsPinned {
		return m.pinIconHelper.GetColorizedPinIcon(item.PinOrder)
	}
	return ""
}

// getThemedPinIcon returns the appropriate themed pin icon for an item
func (m Model) getThemedPinIcon(item storage.ClipboardItem) string {
	if item.IsPinned {
		mainStyles := m.themeService.GetMainViewStyles()
		return m.pinIconHelper.GetThemedPinIcon(item.PinOrder, mainStyles)
	}
	return ""
}

// getPlainSecurityIconText returns plain Unicode security icon text for selected items
func (m Model) getPlainSecurityIconText(item storage.ClipboardItem) string {
	if item.ContentType == "image" {
		return ""
	}

	// Show safe marker if item has been marked as safe but has a threat level
	if item.SafeEntry && item.ThreatLevel != "none" {
		return m.iconHelper.indicators.Safe
	}

	// Don't show security warnings if item has been marked as safe with no threat
	if item.SafeEntry {
		return ""
	}

	// Use stored threat level for display - return plain Unicode characters
	switch item.ThreatLevel {
	case "high":
		return m.iconHelper.indicators.HighRisk
	case "medium":
		return m.iconHelper.indicators.MediumRisk
	case "low":
		return m.iconHelper.indicators.MediumRisk
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

	// Use standard dialog dimensions (consistent with all other views)
	dialogWidth, dialogHeight, contentWidth, contentHeight := m.calculateDialogDimensions()

	// Initialize viewport if not ready
	if !m.helpViewportReady {
		m.initHelpViewport()
	}

	// Create header text
	headerText := "NClip Help"

	// Build help content
	var helpContent strings.Builder
	if m.helpViewportReady {
		helpContent.WriteString(m.helpViewport.View())
	} else {
		// Fallback to basic help display
		helpLines := m.generateHelpContent()
		helpViewStyles := m.themeService.GetViewStyles("help")
		for i := 0; i < contentHeight && i < len(helpLines); i++ {
			line := helpLines[i]
			if len(line) > contentWidth {
				line = line[:contentWidth-3] + "..."
			}
			// Apply proper styling with width for consistency
			styledLine := helpViewStyles.Text.Width(contentWidth).Render(line)
			helpContent.WriteString(styledLine)
			helpContent.WriteString("\n")
		}
	}

	// Create footer text
	var scrollInfo string
	if m.helpViewportReady {
		// Get scroll position from viewport
		totalLines := len(m.generateHelpContent())
		if totalLines > contentHeight {
			scrollInfo = fmt.Sprintf(" - %d%%", int(float64(m.helpViewport.YOffset)/float64(totalLines-contentHeight)*100))
		}
	}
	footerText := "?: close" + scrollInfo

	// Build frame content using shared function
	frameContent := m.buildFrameContent(headerText, helpContent.String(), footerText, contentWidth)

	// Create framed dialog using shared function
	positioned := m.createFramedDialog(dialogWidth, dialogHeight, frameContent)

	return positioned
}

// getTextViewLines splits the text content into lines for viewing with syntax highlighting
func (m Model) getTextViewLines() []string {
	if m.viewingText == nil {
		return []string{}
	}

	content := m.viewingText.Content

	// Detect if this is source code and apply syntax highlighting
	language, isCode := m.codeDetector.DetectLanguage(content)
	
	var lines []string
	if isCode {
		// Apply syntax highlighting with theme
		themeName := m.themeService.config.Chroma.Theme
		// Debug: ensure theme is being passed correctly
		if themeName == "" {
			themeName = "monokai" // fallback to ensure syntax highlighting works
		}
		highlightedLines, err := m.codeDetector.HighlightCodeWithTheme(content, language, m.useBasicColors, themeName)
		if err == nil {
			lines = highlightedLines
		} else {
			// Fallback to original content if highlighting fails (styling will be applied later after padding)
			lines = strings.Split(content, "\n")
		}
	} else {
		// Use original content for non-code text (styling will be applied later after padding)
		lines = strings.Split(content, "\n")
	}

	// Use standard dialog dimensions for consistent content width
	_, _, contentWidth, _ := m.calculateDialogDimensions()
	if contentWidth < 10 {
		contentWidth = 10
	}

	// Wrap long lines (considering ANSI codes for highlighted text)
	var wrappedLines []string
	for _, line := range lines {
		// Calculate visible length (excluding ANSI escape codes)
		visibleLen := m.calculateVisibleLength(line)
		
		if visibleLen <= contentWidth {
			wrappedLines = append(wrappedLines, line)
		} else {
			// For syntax-highlighted code, prefer not to wrap to preserve formatting
			// Instead, truncate with indication
			if isCode {
				truncated := m.truncateWithANSI(line, contentWidth-3) + "..."
				wrappedLines = append(wrappedLines, truncated)
			} else {
				// Wrap long lines for regular text
				wrappedLines = append(wrappedLines, m.wrapLongLine(line, contentWidth)...)
			}
		}
	}

	return wrappedLines
}

// calculateVisibleLength calculates the visible length of a string excluding ANSI escape codes
func (m Model) calculateVisibleLength(s string) int {
	// Simple ANSI escape sequence removal for length calculation
	// This regex matches basic ANSI escape sequences
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	clean := ansiRegex.ReplaceAllString(s, "")
	return len(clean)
}

// truncateWithANSI truncates a string with ANSI codes while preserving color formatting
func (m Model) truncateWithANSI(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	
	visibleLen := m.calculateVisibleLength(s)
	if visibleLen <= maxLen {
		return s
	}
	
	// Parse the string character by character, tracking ANSI sequences
	var result strings.Builder
	var visibleCount int
	inEscape := false
	
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		
		// Start of ANSI escape sequence
		if r == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			inEscape = true
		}
		
		// Always include escape sequences (they don't count toward visible length)
		if inEscape {
			result.WriteRune(r)
			// End of escape sequence
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		
		// Regular character - check if we have room
		if visibleCount >= maxLen {
			break
		}
		
		result.WriteRune(r)
		visibleCount++
	}
	
	return result.String()
}

// wrapLongLine wraps a long line at word boundaries
func (m Model) wrapLongLine(line string, contentWidth int) []string {
	var wrapped []string
	
	for len(line) > 0 {
		if len(line) <= contentWidth {
			wrapped = append(wrapped, line)
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

		wrapped = append(wrapped, line[:breakPoint])
		line = strings.TrimLeft(line[breakPoint:], " ")
	}
	
	return wrapped
}

// padLineToWidth pads a line with spaces to ensure consistent frame width
func (m Model) padLineToWidth(line string, targetWidth int) string {
	visibleLen := m.calculateVisibleLength(line)
	if visibleLen >= targetWidth {
		return line
	}
	
	// Add spaces to reach target width
	padding := targetWidth - visibleLen
	return line + strings.Repeat(" ", padding)
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

	// Use standard dialog dimensions (consistent with all other views)
	dialogWidth, dialogHeight, contentWidth, contentHeight := m.calculateDialogDimensions()

	// Initialize viewport if not ready
	if !m.textViewportReady {
		m.initTextViewport()
	}

	// Get text content from viewport
	var textContent strings.Builder
	if m.textViewportReady {
		textContent.WriteString(m.textViewport.View())
	} else {
		// Fallback to basic text display
		textLines := m.getTextViewLines()
		textViewStyles := m.themeService.GetViewStyles("text")
		for i := 0; i < contentHeight && i < len(textLines); i++ {
			line := textLines[i]
			// Use visible length calculation for proper truncation with ANSI codes
			visibleLen := m.calculateVisibleLength(line)
			if visibleLen > contentWidth {
				line = m.truncateWithANSI(line, contentWidth-3) + "..."
			}
			// Apply proper styling with width for consistency
			styledLine := textViewStyles.Text.Width(contentWidth).Render(line)
			textContent.WriteString(styledLine)
			textContent.WriteString("\n")
		}
	}

	// Create title with security marking, length info and syntax highlighting status
	lineCount := len(strings.Split(m.viewingText.Content, "\n"))
	charCount := len(m.viewingText.Content)
	
	// Build header with proper Lipgloss composition
	var headerText string
	
	// Get security icon and build header parts
	securityIcon := m.getThemedSecurityIcon(*m.viewingText)
	
	// Check if syntax highlighting was applied
	language, isCode := m.codeDetector.DetectLanguage(m.viewingText.Content)
	
	// Build the text part of the header
	var headerTextPart string
	if isCode {
		headerTextPart = fmt.Sprintf("Text View - %s (%d lines, %d chars)", strings.ToUpper(language), lineCount, charCount)
	} else {
		headerTextPart = fmt.Sprintf("Text View (%d lines, %d chars)", lineCount, charCount)
	}
	
	// Compose header with proper styling
	if securityIcon != "" {
		// Build composed header with icon and styled space
		mainStyles := m.themeService.GetMainViewStyles()
		styledSpace := mainStyles.Header.Render(" ")
		styledText := mainStyles.Header.Render(headerTextPart)
		headerText = securityIcon + styledSpace + styledText
	} else {
		// No icon, just the text (styling will be applied in buildFrameContent)
		headerText = headerTextPart
	}
	
	// Add security information to the header if item has security warnings
	if m.viewingText.ThreatLevel == "high" || m.viewingText.ThreatLevel == "medium" {
		var threatDesc string
		if m.viewingText.ThreatLevel == "high" {
			threatDesc = "HIGH RISK - Contains potentially dangerous content"
		} else {
			threatDesc = "MEDIUM RISK - Contains potentially sensitive content"
		}
		headerText += " - " + threatDesc
	}

	// Create footer text
	var scrollInfo string
	if m.textViewportReady {
		// Get scroll position from viewport
		totalLines := len(m.getTextViewLines())
		if totalLines > contentHeight {
			scrollInfo = fmt.Sprintf(" - %d%%", int(float64(m.textViewport.YOffset)/float64(totalLines-contentHeight)*100))
		}
	}
	// Create footer text based on delete confirmation state and security warnings
	var footerText string
	if m.textDeletePending {
		footerText = "Press 'x' again to confirm deletion, any other key to cancel"
	} else {
		baseFooter := "enter: copy | x: delete | e: edit"
		// Add security actions if this item has security warnings
		if m.viewingText.ThreatLevel == "high" || m.viewingText.ThreatLevel == "medium" {
			baseFooter += " | s: mark as safe"
		}
		footerText = baseFooter + scrollInfo
	}

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
			m.cache.ForceRefresh()
			m.items = m.cache.GetAllMeta()
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
	searchStyle := m.createStyle(m.config.Theme.Search)
	warningStyle := m.createStyle(m.config.Theme.Warning)
	mainStyles := m.themeService.GetMainViewStyles()

	var lines []string

	// Basic Navigation
	lines = append(lines, searchStyle.Render("BASIC NAVIGATION"))
	lines = append(lines, "")
	lines = append(lines, "  j/k or up/down   Navigate up/down through clipboard items")
	lines = append(lines, "  g                Go to first item")
	lines = append(lines, "  G                Go to last item")
	lines = append(lines, "  pgup/pgdown      Page up/down through items")
	lines = append(lines, "  Enter        Copy selected item to clipboard and exit")
	lines = append(lines, "  q / Ctrl+C   Quit the application")
	lines = append(lines, "  ?            Show this help screen")
	lines = append(lines, "")

	// Search and Filtering
	lines = append(lines, searchStyle.Render("SEARCH & FILTERING"))
	lines = append(lines, "")
	lines = append(lines, "  Basic Filtering:")
	lines = append(lines, "    /            Enter search mode for fuzzy filtering")
	lines = append(lines, "    c            Clear current search filter")
	lines = append(lines, "")
	lines = append(lines, "  Content Filters:")
	lines = append(lines, "    i            Show only images")
	lines = append(lines, "    h            Show only high-risk security items")
	lines = append(lines, "    m            Show only medium-risk security items")
	lines = append(lines, "    s            Show only safe security items")
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
	lines = append(lines, "  Basic content operations:")
	lines = append(lines, "    v            View text/image in full-screen viewer")
	lines = append(lines, "    e            Edit selected item in external editor")
	lines = append(lines, "    x            Delete item (press 'x' again to confirm)")
	lines = append(lines, "    p            Pin/unpin item to top of list")
	lines = append(lines, "")
	lines = append(lines, "  Quick access to pinned items:")
	// Show pin icons based on terminal capabilities
	pin1 := m.pinIconHelper.GetThemedPinIcon(1, mainStyles)
	pin9 := m.pinIconHelper.GetThemedPinIcon(9, mainStyles)
	pin10 := m.pinIconHelper.GetThemedPinIcon(10, mainStyles)
	lines = append(lines, fmt.Sprintf("    1-9          Copy pinned item %s-%s to clipboard and exit", pin1, pin9))
	lines = append(lines, fmt.Sprintf("    0            Copy pinned item %s to clipboard and exit", pin10))
	lines = append(lines, "")
	lines = append(lines, "  In text view mode:")
	lines = append(lines, "    up/down      Scroll through text content")
	lines = append(lines, "    Enter        Copy text to clipboard and exit")
	lines = append(lines, "    e            Edit text (returns to viewer after editing)")
	lines = append(lines, "    x            Delete text from database")
	lines = append(lines, "    s            Mark security-flagged item as safe")
	lines = append(lines, "    any other key Exit text viewer and return to list")
	lines = append(lines, "")
	lines = append(lines, "  In image view mode:")
	lines = append(lines, "    Enter        Copy image to clipboard and exit")
	lines = append(lines, "    o            Open image in external viewer")
	lines = append(lines, "    e            Edit image in external editor")
	lines = append(lines, "    x            Delete image from database")
	lines = append(lines, "    any other key Exit image viewer and return to list")
	lines = append(lines, "")

	// Security Features
	lines = append(lines, warningStyle.Render("SECURITY FEATURES"))
	lines = append(lines, "")
	lines = append(lines, "  Initialize content scanning:")
	lines = append(lines, "    ctrl+s       Analyze current item for security threats")
	lines = append(lines, "")
	lines = append(lines, "  Security indicators automatically detect:")
	lines = append(lines, "    - JWT tokens, API keys, SSH keys")
	lines = append(lines, "    - Database connection strings")
	lines = append(lines, "    - Password-like patterns")
	lines = append(lines, "    - Credit card numbers")
	lines = append(lines, "")
	lines = append(lines, "  Security visual indicators:")
	lines = append(lines, fmt.Sprintf("    %s          High-risk security content", m.iconHelper.GetThemedHighRiskIcon(mainStyles)))
	lines = append(lines, fmt.Sprintf("    %s          Medium-risk security content", m.iconHelper.GetThemedMediumRiskIcon(mainStyles)))
	lines = append(lines, fmt.Sprintf("    %s          User-marked safe content", m.iconHelper.GetThemedSafeIcon(mainStyles)))
	lines = append(lines, "")
	lines = append(lines, "  Pin indicators:")
	// Show examples of pin icons based on terminal capabilities
	examplePin1 := m.pinIconHelper.GetThemedPinIcon(1, mainStyles)
	examplePin2 := m.pinIconHelper.GetThemedPinIcon(2, mainStyles)
	lines = append(lines, fmt.Sprintf("    %s %s ...     Pinned items (appear at top of list)", examplePin1, examplePin2))
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
	lines = append(lines, "")

	return lines
}

// initTextViewport initializes the text viewport with proper dimensions
func (m *Model) initTextViewport() {
	_, _, contentWidth, contentHeight := m.calculateDialogDimensions()
	if contentHeight < 5 {
		contentHeight = 5
	}
	
	// Create a new viewport with proper dimensions
	m.textViewport = viewport.New(contentWidth, contentHeight)
	m.textViewport.YPosition = 0
	
	// Set the content for the viewport
	if m.viewingText != nil {
		textLines := m.getTextViewLines()
		
		// Apply styling to each line before setting content
		_, isCode := m.codeDetector.DetectLanguage(m.viewingText.Content)
		textViewStyles := m.themeService.GetViewStyles("text")
		
		var styledLines []string
		for _, line := range textLines {
			// Use visible length calculation for proper truncation with ANSI codes
			visibleLen := m.calculateVisibleLength(line)
			if visibleLen > contentWidth {
				line = m.truncateWithANSI(line, contentWidth-3) + "..."
			}
			
			// Apply styling based on content type
			if !isCode {
				// For non-code content, apply styling with proper width
				line = textViewStyles.Text.Width(contentWidth).Render(line)
			} else {
				// For code content, preserve Chroma syntax highlighting
				// Apply background styling that doesn't interfere with Chroma colors
				if textViewStyles.Text.GetBackground() != lipgloss.Color("") {
					// Create a style that only sets background, preserving existing colors
					bgStyle := lipgloss.NewStyle().Background(textViewStyles.Text.GetBackground())
					// Apply background and padding together to avoid ANSI interference
					line = bgStyle.Width(contentWidth).Render(line)
				} else {
					// No background styling needed, just pad
					line = m.padLineToWidth(line, contentWidth)
				}
			}
			
			styledLines = append(styledLines, line)
		}
		
		content := strings.Join(styledLines, "\n")
		m.textViewport.SetContent(content)
	}
	
	m.textViewportReady = true
}

// initHelpViewport initializes the help viewport with proper dimensions
func (m *Model) initHelpViewport() {
	_, _, contentWidth, contentHeight := m.calculateDialogDimensions()
	if contentHeight < 5 {
		contentHeight = 5
	}
	
	// Create a new viewport with proper dimensions
	m.helpViewport = viewport.New(contentWidth, contentHeight)
	m.helpViewport.YPosition = 0
	
	// Set the content for the viewport
	helpLines := m.generateHelpContent()
	helpViewStyles := m.themeService.GetViewStyles("help")
	
	var styledLines []string
	for _, line := range helpLines {
		// Use visible length calculation for proper truncation with ANSI codes
		visibleLen := m.calculateVisibleLength(line)
		if visibleLen > contentWidth {
			line = m.truncateWithANSI(line, contentWidth-3) + "..."
		}
		// Apply styling with proper width to handle both content and padding
		styledLine := helpViewStyles.Text.Width(contentWidth).Render(line)
		styledLines = append(styledLines, styledLine)
	}
	
	content := strings.Join(styledLines, "\n")
	m.helpViewport.SetContent(content)
	
	m.helpViewportReady = true
}

// initSecurityViewport initializes the security viewport with proper dimensions
func (m *Model) initSecurityViewport() {
	_, _, contentWidth, contentHeight := m.calculateDialogDimensions()
	if contentHeight < 5 {
		contentHeight = 5
	}
	
	// Create a new viewport with proper dimensions
	m.securityViewport = viewport.New(contentWidth, contentHeight)
	m.securityViewport.YPosition = 0
	
	// Set the content for the viewport
	securityLines := m.getSecurityViewLines()
	securityViewStyles := m.themeService.GetViewStyles("security")
	
	var styledLines []string
	for _, line := range securityLines {
		// Use visible length calculation for proper truncation with ANSI codes
		visibleLen := m.calculateVisibleLength(line)
		if visibleLen > contentWidth {
			line = m.truncateWithANSI(line, contentWidth-3) + "..."
		}
		// Apply styling with proper width to handle both content and padding
		styledLine := securityViewStyles.Text.Width(contentWidth).Render(line)
		styledLines = append(styledLines, styledLine)
	}
	
	content := strings.Join(styledLines, "\n")
	m.securityViewport.SetContent(content)
	
	m.securityViewportReady = true
}

// initWarningViewport initializes the warning viewport with proper dimensions
func (m *Model) initWarningViewport() {
	// Use exact same dimension calculation as renderImageSecurityWarning
	_, _, fullContentWidth, _ := m.calculateDialogDimensions()
	
	// Scale down for warning dialog (match renderImageSecurityWarning exactly)
	warningWidth := min(50, fullContentWidth+4)
	warningHeight := min(10, 12) 
	contentWidth := warningWidth - 4
	contentHeight := warningHeight - 4
	
	if contentHeight < 3 {
		contentHeight = 3
	}
	
	// Create a new viewport with proper dimensions
	m.warningViewport = viewport.New(contentWidth, contentHeight)
	m.warningViewport.YPosition = 0
	
	// Get warning styles from theme service
	warningStyles := m.themeService.GetWarningStyles()
	
	// Build warning content
	var contentLines []string
	contentLines = append(contentLines, "Security scanning is only available for text")
	contentLines = append(contentLines, "content. Images cannot be analyzed for")
	contentLines = append(contentLines, "security threats.")
	contentLines = append(contentLines, "")
	contentLines = append(contentLines, "Press any key to continue")
	
	// Apply styling to content lines
	var styledLines []string
	for i, line := range contentLines {
		// Use visible length calculation for proper truncation with ANSI codes
		visibleLen := m.calculateVisibleLength(line)
		if visibleLen > contentWidth {
			line = m.truncateWithANSI(line, contentWidth-3) + "..."
		}
		
		// Apply styling with proper width to handle both content and padding
		var styledLine string
		if i < len(contentLines)-1 {
			// Regular content lines
			styledLine = warningStyles.Body.Width(contentWidth).Render(line)
		} else {
			// Instructions line
			styledLine = warningStyles.Prompt.Width(contentWidth).Render(line)
		}
		styledLines = append(styledLines, styledLine)
	}
	
	// Join content and ensure it fills the viewport height
	content := strings.Join(styledLines, "\n")
	
	// Pad content to fill viewport height if needed
	contentLineCount := len(styledLines)
	if contentLineCount < contentHeight {
		// Add empty lines with proper background styling to fill the viewport
		warningStyles := m.themeService.GetWarningStyles()
		for i := contentLineCount; i < contentHeight; i++ {
			emptyLine := warningStyles.Body.Width(contentWidth).Render("")
			content += "\n" + emptyLine
		}
	}
	
	m.warningViewport.SetContent(content)
	
	m.warningViewportReady = true
}

// fileExists checks if a file exists
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}
