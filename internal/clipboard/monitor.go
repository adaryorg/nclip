package clipboard

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	atotto "github.com/atotto/clipboard"
	"golang.design/x/clipboard"

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
	interval         time.Duration
	textCallback     ContentCallback
	imageCallback    ImageCallback
	securityCallback SecurityCallback
	detector         *security.SecurityDetector
	hashStore        *security.HashStore
}

func NewMonitor(callback func(string)) *Monitor {
	detector := security.NewSecurityDetector()
	hashStore, _ := security.NewHashStore() // Ignore error, will be nil if failed

	return &Monitor{
		interval:     500 * time.Millisecond,
		textCallback: callback,
		detector:     detector,
		hashStore:    hashStore,
	}
}

func NewMonitorWithImage(textCallback ContentCallback, imageCallback ImageCallback) *Monitor {
	detector := security.NewSecurityDetector()
	hashStore, _ := security.NewHashStore() // Ignore error, will be nil if failed

	return &Monitor{
		interval:      500 * time.Millisecond,
		textCallback:  textCallback,
		imageCallback: imageCallback,
		detector:      detector,
		hashStore:     hashStore,
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
	}
}

func (m *Monitor) Start(ctx context.Context) error {
	// Initialize clipboard
	if err := ensureInit(); err != nil {
		return err
	}

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Monitor text content
			content := string(clipboard.Read(clipboard.FmtText))
			if content != "" && content != m.lastContent {
				m.lastContent = content

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
								fmt.Printf("Skipping user-blocked security content (hash: %s)\n", contentHash[:8])
								continue
							}
						}

						// Log security detection but don't block - let TUI show indicators
						highestThreat := security.GetHighestThreat(threats)
						if highestThreat != nil {
							if security.IsHighRiskThreat(threats) {
								fmt.Printf("INFO: High-risk security content detected (%.0f%% confidence): %s - storing with security indicator\n",
									highestThreat.Confidence*100, highestThreat.Type)
							} else {
								fmt.Printf("INFO: Medium-risk security content detected (%.0f%% confidence): %s - storing with security indicator\n",
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
					m.textCallback(content)
				}
			}

			// Monitor image content if callback is set
			if m.imageCallback != nil {
				imageData := clipboard.Read(clipboard.FmtImage)
				if len(imageData) > 0 {
					// Create a description for the image
					description := fmt.Sprintf("Image (%d bytes)", len(imageData))
					fmt.Printf("DEBUG: Captured image data: %d bytes\n", len(imageData))
					m.imageCallback(imageData, description)
				}
			}
		}
	}
}

func ensureInit() error {
	initOnce.Do(func() {
		initErr = clipboard.Init()
	})
	return initErr
}

func Copy(content string) error {
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
	if m.hashStore != nil {
		return m.hashStore.Close()
	}
	return nil
}
