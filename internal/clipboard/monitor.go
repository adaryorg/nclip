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

package clipboard

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	atotto "github.com/atotto/clipboard"
	"golang.design/x/clipboard"

	"github.com/adaryorg/nclip/internal/logging"
	"github.com/adaryorg/nclip/internal/security"
)

var (
	initOnce sync.Once
	initErr  error
)

type ContentCallback func(string)
type ImageCallback func([]byte, string)
type SecurityCallback func(string, []security.SecurityThreat)

type Monitor struct {
	lastContent      string
	lastImageHash    string
	interval         time.Duration
	textCallback     ContentCallback
	imageCallback    ImageCallback
	securityCallback SecurityCallback
	detector         *security.SecurityDetector
	hashStore        *security.HashStore
	useWayland       bool
	
	// Anti-bump fields
	pendingContent   string
	stabilizeTimer   *time.Timer
	stabilizeTimeout time.Duration
}

func NewMonitor(callback func(string)) *Monitor {
	detector := security.NewSecurityDetector()
	hashStore, _ := security.NewHashStore() // Ignore error, will be nil if failed

	return &Monitor{
		interval:         500 * time.Millisecond,
		textCallback:     callback,
		detector:         detector,
		hashStore:        hashStore,
		useWayland:       isWaylandSession(),
		stabilizeTimeout: 500 * time.Millisecond, // Wait 500ms for selection to stabilize
	}
}

func NewMonitorWithImage(textCallback ContentCallback, imageCallback ImageCallback) *Monitor {
	detector := security.NewSecurityDetector()
	hashStore, _ := security.NewHashStore() // Ignore error, will be nil if failed

	return &Monitor{
		interval:         500 * time.Millisecond,
		textCallback:     textCallback,
		imageCallback:    imageCallback,
		detector:         detector,
		hashStore:        hashStore,
		useWayland:       isWaylandSession(),
		stabilizeTimeout: 500 * time.Millisecond, // Wait 500ms for selection to stabilize
	}
}

func NewMonitorWithSecurity(textCallback ContentCallback, imageCallback ImageCallback, securityCallback SecurityCallback) *Monitor {
	detector := security.NewSecurityDetector()
	hashStore, _ := security.NewHashStore() // Ignore error, will be nil if failed

	return &Monitor{
		interval:         500 * time.Millisecond,
		textCallback:     textCallback,
		imageCallback:    imageCallback,
		securityCallback: securityCallback,
		detector:         detector,
		hashStore:        hashStore,
		useWayland:       isWaylandSession(),
		stabilizeTimeout: 500 * time.Millisecond, // Wait 500ms for selection to stabilize
	}
}

func (m *Monitor) Start(ctx context.Context) error {
	if m.useWayland {
		return m.startWaylandMonitor(ctx)
	}
	return m.startX11Monitor(ctx)
}

func (m *Monitor) startWaylandMonitor(ctx context.Context) error {
	// Use wl-paste --watch for real-time clipboard monitoring
	cmd := exec.CommandContext(ctx, "wl-paste", "--watch", "echo", "CLIPBOARD_CHANGED")
	cmd.Stdout = nil
	cmd.Stderr = nil
	
	// Start the wl-paste watcher
	err := cmd.Start()
	if err != nil {
		// Fallback to polling if wl-paste watch fails
		logging.Warn("wl-paste watch failed, falling back to polling: %v", err)
		return m.startPollingMonitor(ctx)
	}

	go func() {
		cmd.Wait()
	}()

	// Monitor clipboard changes
	ticker := time.NewTicker(100 * time.Millisecond) // Faster polling for Wayland
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			cmd.Process.Kill()
			return ctx.Err()
		case <-ticker.C:
			content, err := m.getWaylandClipboardContent()
			if err != nil {
				continue
			}
			
			if content != "" && content != m.lastContent {
				m.handleClipboardChange(content)
				m.lastContent = content
			}

			// Monitor image content if callback is set
			if m.imageCallback != nil {
				imageData, err := m.getWaylandClipboardImage()
				if err == nil && len(imageData) > 16 { // Ignore very small images (likely empty/invalid)
					imageHash := fmt.Sprintf("%x", sha256.Sum256(imageData))
					if imageHash != m.lastImageHash {
						m.lastImageHash = imageHash
						description := fmt.Sprintf("Image (%d bytes)", len(imageData))
						logging.Debug("Captured image data: %d bytes", len(imageData))
						m.imageCallback(imageData, description)
					}
				}
			}
		}
	}
}

func (m *Monitor) startX11Monitor(ctx context.Context) error {
	// Initialize clipboard
	if err := ensureInit(); err != nil {
		return err
	}

	return m.startPollingMonitor(ctx)
}

func (m *Monitor) startPollingMonitor(ctx context.Context) error {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			var content string
			var err error
			
			if m.useWayland {
				content, err = m.getWaylandClipboardContent()
				if err != nil {
					continue
				}
			} else {
				content = string(clipboard.Read(clipboard.FmtText))
			}
			
			if content != "" && content != m.lastContent {
				m.handleClipboardChange(content)
				m.lastContent = content
			}

			// Monitor image content if callback is set
			if m.imageCallback != nil {
				var imageData []byte
				var err error
				
				if m.useWayland {
					imageData, err = m.getWaylandClipboardImage()
					if err != nil {
						continue
					}
				} else {
					imageData = clipboard.Read(clipboard.FmtImage)
				}
				
				if len(imageData) > 16 { // Ignore very small images (likely empty/invalid)
					imageHash := fmt.Sprintf("%x", sha256.Sum256(imageData))
					if imageHash != m.lastImageHash {
						m.lastImageHash = imageHash
						description := fmt.Sprintf("Image (%d bytes)", len(imageData))
						logging.Debug("Captured image data: %d bytes", len(imageData))
						m.imageCallback(imageData, description)
					}
				}
			}
		}
	}
}

func (m *Monitor) handleClipboardChange(content string) {
	// Check if this content is a substring expansion of the last content
	if m.isSubstringExpansion(content) {
		logging.Debug("Substring expansion detected - starting stabilize timer: len=%d, content=%s...", 
			len(content), m.truncateForLog(content))
		
		// Store the content as pending and start/reset stabilization timer
		m.pendingContent = content
		
		// Cancel existing timer if any
		if m.stabilizeTimer != nil {
			m.stabilizeTimer.Stop()
		}
		
		// Start stabilization timer
		m.stabilizeTimer = time.AfterFunc(m.stabilizeTimeout, func() {
			// Timer expired, store the final stabilized content
			logging.Debug("Selection stabilized, storing final content: len=%d, content=%s...", 
				len(m.pendingContent), m.truncateForLog(m.pendingContent))
			m.processClipboardContent(m.pendingContent)
			m.pendingContent = ""
		})
		return
	}
	
	// This is new content (not a substring expansion)
	// Cancel any pending stabilization and store immediately
	if m.stabilizeTimer != nil {
		logging.Debug("Canceling stabilization timer, storing new content immediately")
		m.stabilizeTimer.Stop()
		m.stabilizeTimer = nil
		m.pendingContent = ""
	}
	
	logging.Debug("Storing new clipboard content: len=%d, content=%s...", 
		len(content), m.truncateForLog(content))
	m.processClipboardContent(content)
}

func (m *Monitor) isSubstringExpansion(newContent string) bool {
	// Only check against last processed content
	if m.lastContent == "" {
		return false
	}
	
	// Check if the previous content is a substring of the new content
	// This handles expanding selections: "test" -> "test debug" -> "test debug logging"
	if strings.Contains(newContent, m.lastContent) && len(newContent) > len(m.lastContent) {
		logging.Debug("Substring expansion detected: %q expanded to %q", 
			m.truncateForLog(m.lastContent), m.truncateForLog(newContent))
		return true
	}
	
	// Check the reverse case - new content is substring of previous content  
	// This handles shrinking selections: "test debug logging" -> "test debug" -> "test"
	if strings.Contains(m.lastContent, newContent) && len(m.lastContent) > len(newContent) {
		logging.Debug("Substring contraction detected: %q contracted to %q", 
			m.truncateForLog(m.lastContent), m.truncateForLog(newContent))
		return true
	}
	
	return false
}


func (m *Monitor) truncateForLog(content string) string {
	const maxLen = 50
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen-3] + "..."
}

func (m *Monitor) processClipboardContent(content string) {
	// Check for security threats
	if m.detector != nil {
		threats := m.detector.DetectSecurity(content)

		if len(threats) > 0 {
			// Security content detected
			contentHash := security.CreateHash(content)

			// Check if this content is already known to be blocked by user
			if m.hashStore != nil {
				isKnown, _, err := m.hashStore.IsKnownSecurityContent(content)
				if err == nil && isKnown {
					// Skip only content user has explicitly chosen to block
					logging.Info("Skipping user-blocked security content (hash: %s)", contentHash[:8])
					return
				}
			}

			// Log security detection but don't block - let TUI show indicators
			highestThreat := security.GetHighestThreat(threats)
			if highestThreat != nil {
				if security.IsHighRiskThreat(threats) {
					logging.Warn("High-risk security content detected (%.0f%% confidence): %s - storing with security indicator",
						highestThreat.Confidence*100, highestThreat.Type)
				} else {
					logging.Info("Medium-risk security content detected (%.0f%% confidence): %s - storing with security indicator",
						highestThreat.Confidence*100, highestThreat.Type)
				}

				// Call security callback for logging if available
				if m.securityCallback != nil {
					m.securityCallback(content, threats)
				}
			}

			// Note: We don't add to hash store here - only when user explicitly removes via TUI
		}
	}

	// Store content normally
	if m.textCallback != nil {
		logging.Debug("Storing clipboard content (len=%d): %s...", len(content), m.truncateForLog(content))
		m.textCallback(content)
	}
}

func ensureInit() error {
	initOnce.Do(func() {
		initErr = clipboard.Init()
	})
	return initErr
}

func isWaylandSession() bool {
	return os.Getenv("WAYLAND_DISPLAY") != "" || os.Getenv("XDG_SESSION_TYPE") == "wayland"
}

func (m *Monitor) getWaylandClipboardContent() (string, error) {
	cmd := exec.Command("wl-paste", "--no-newline")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func (m *Monitor) getWaylandClipboardImage() ([]byte, error) {
	// First check if image data is available
	checkCmd := exec.Command("wl-paste", "--list-types")
	types, err := checkCmd.Output()
	if err != nil {
		return nil, err
	}
	
	// Look for image MIME types
	typesStr := string(types)
	if !strings.Contains(typesStr, "image/") {
		return nil, fmt.Errorf("no image data available")
	}
	
	// Try different image formats
	formats := []string{"image/png", "image/jpeg", "image/gif", "image/bmp"}
	for _, format := range formats {
		if strings.Contains(typesStr, format) {
			cmd := exec.Command("wl-paste", "--type", format)
			output, err := cmd.Output()
			if err == nil && len(output) > 16 {
				return output, nil
			}
		}
	}
	
	return nil, fmt.Errorf("no valid image data found")
}

func Copy(content string) error {
	if isWaylandSession() {
		return copyWayland(content)
	}
	return copyX11(content)
}

func copyWayland(content string) error {
	// Use wl-copy for Wayland clipboard
	cmd := exec.Command("wl-copy")
	cmd.Stdin = strings.NewReader(content)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("wl-copy failed: %v", err)
	}
	return nil
}

func copyX11(content string) error {
	// Copy to both CLIPBOARD and PRIMARY selections for full X11 compatibility
	// CLIPBOARD: Ctrl+C/Ctrl+V (browsers, GUI apps)
	// PRIMARY: Text selection and Shift+Insert (terminals)

	// Copy to CLIPBOARD using atotto (reliable for GUI apps)
	err := atotto.WriteAll(content)
	if err != nil {
		return err
	}

	// Copy to PRIMARY selection using xclip directly (most reliable)
	cmd := exec.Command("xclip", "-selection", "primary")
	cmd.Stdin = strings.NewReader(content)
	cmd.Run() // Ignore errors as PRIMARY is optional

	return nil
}

func CopyImage(imageData []byte) error {
	if isWaylandSession() {
		return copyImageWayland(imageData)
	}
	return copyImageX11(imageData)
}

func copyImageWayland(imageData []byte) error {
	// Use wl-copy for Wayland image clipboard
	cmd := exec.Command("wl-copy", "--type", "image/png")
	cmd.Stdin = strings.NewReader(string(imageData))
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("wl-copy image failed: %v", err)
	}
	return nil
}

func copyImageX11(imageData []byte) error {
	// Initialize clipboard if needed
	if err := ensureInit(); err != nil {
		return err
	}

	// Copy image data to clipboard
	clipboard.Write(clipboard.FmtImage, imageData)

	return nil
}

// Close cleans up resources used by the monitor
func (m *Monitor) Close() error {
	// Cancel any pending stabilization timer
	if m.stabilizeTimer != nil {
		m.stabilizeTimer.Stop()
		m.stabilizeTimer = nil
	}
	
	if m.hashStore != nil {
		return m.hashStore.Close()
	}
	return nil
}
