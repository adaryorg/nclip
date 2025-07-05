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
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"os"
	"strings"

	// Additional image format support
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"

	// For image resizing
	"golang.org/x/image/draw"
)

// detectKittySupport checks if the terminal supports Kitty image protocol
func detectKittySupport() bool {
	// Check specific terminal programs that support the protocol
	termProgram := os.Getenv("TERM_PROGRAM")
	if termProgram == "kitty" || termProgram == "ghostty" || termProgram == "WezTerm" ||
		termProgram == "Konsole" {
		return true
	}

	// Check if we're in a Kitty terminal by looking for KITTY_WINDOW_ID
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return true
	}

	// Check if we're in WezTerm by looking for any WezTerm environment variables
	if os.Getenv("WEZTERM_EXECUTABLE") != "" ||
		os.Getenv("WEZTERM_CONFIG_FILE") != "" ||
		os.Getenv("WEZTERM_PANE") != "" ||
		os.Getenv("WEZTERM_UNIX_SOCKET") != "" {
		return true
	}

	// Check for additional terminal-specific environment variables
	// Konsole
	if os.Getenv("KONSOLE_VERSION") != "" {
		return true
	}


	// Check TERM environment variable for known supporting terminals
	term := os.Getenv("TERM")
	if strings.Contains(term, "kitty") || strings.Contains(term, "wezterm") ||
		strings.Contains(term, "konsole") {
		return true
	}

	// Test actual protocol capability by sending a query
	return testKittyProtocol()
}

// testKittyProtocol sends a capability query to test if terminal supports graphics
func testKittyProtocol() bool {
	// Send a query to check if terminal supports graphics protocol
	// This is a non-invasive query that asks for terminal capabilities
	fmt.Print("\x1b_Gi=1,a=q;\x1b\\")

	// For now, we'll assume it works if we get here
	// In a full implementation, we'd read the response
	// but that requires more complex terminal handling
	return false // Conservative default for unknown terminals
}

// getImageDimensions extracts width, height, and format from image data
func getImageDimensions(imageData []byte) (int, int, string, error) {
	reader := bytes.NewReader(imageData)
	config, format, err := image.DecodeConfig(reader)
	if err != nil {
		return 0, 0, "", err
	}
	return config.Width, config.Height, format, nil
}

// resizeImageIfNeeded resizes very large images to prevent terminal buffer overflow
func resizeImageIfNeeded(imageData []byte, maxWidth, maxHeight int) ([]byte, error) {
	// Decode the image
	reader := bytes.NewReader(imageData)
	img, _, err := image.Decode(reader)
	if err != nil {
		return imageData, err // Return original if we can't decode
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Check if resizing is needed
	if width <= maxWidth && height <= maxHeight {
		return imageData, nil // No resizing needed
	}

	// Calculate new dimensions maintaining aspect ratio
	scaleX := float64(maxWidth) / float64(width)
	scaleY := float64(maxHeight) / float64(height)
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	newWidth := int(float64(width) * scale)
	newHeight := int(float64(height) * scale)

	// Create a new image with the target size
	dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// Resize the image using high-quality scaling
	draw.BiLinear.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

	// Encode as PNG
	var buf bytes.Buffer
	err = png.Encode(&buf, dst)
	if err != nil {
		return imageData, err // Return original if encoding fails
	}

	return buf.Bytes(), nil
}

// resizeImageToExactDimensions resizes an image to exact target dimensions
// Note: The target dimensions should already maintain aspect ratio from caller's calculation
func resizeImageToExactDimensions(imageData []byte, targetWidth, targetHeight int) ([]byte, error) {
	// Decode the image
	reader := bytes.NewReader(imageData)
	img, _, err := image.Decode(reader)
	if err != nil {
		return imageData, err // Return original if we can't decode
	}

	bounds := img.Bounds()
	currentWidth := bounds.Dx()
	currentHeight := bounds.Dy()

	// Check if resizing is actually needed
	if currentWidth == targetWidth && currentHeight == targetHeight {
		return imageData, nil // No resizing needed
	}

	// Create a new image with the exact target size
	dst := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))

	// Resize the image using high-quality scaling
	draw.BiLinear.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

	// Encode as PNG
	var buf bytes.Buffer
	err = png.Encode(&buf, dst)
	if err != nil {
		return imageData, err // Return original if encoding fails
	}

	return buf.Bytes(), nil
}

// renderKittyImage creates a Kitty terminal escape sequence to display an image
func renderKittyImage(imageData []byte, terminalWidth, terminalHeight int) string {
	if len(imageData) == 0 {
		return ""
	}

	// Calculate available terminal space (leave margins for UI)
	availableWidth := terminalWidth - 4   // Leave 4 columns margin
	availableHeight := terminalHeight - 6 // Leave 6 rows for UI elements

	// Ensure minimum reasonable dimensions
	if availableWidth < 10 {
		availableWidth = 10
	}
	if availableHeight < 5 {
		availableHeight = 5
	}

	// Let Kitty handle all the scaling using the 's' parameter
	// Pass original image data and let Kitty scale it to fit
	return renderKittyImageDirect(imageData, availableWidth, availableHeight)
}


// renderSimpleKittyImage renders image with basic Kitty protocol
func renderSimpleKittyImage(imageData []byte) string {
	if len(imageData) == 0 {
		return ""
	}

	// Encode image data as base64
	encoded := base64.StdEncoding.EncodeToString(imageData)

	// Use simple transmission protocol
	const chunkSize = 4096

	if len(encoded) <= chunkSize {
		// Single chunk - basic transmission
		return fmt.Sprintf("\x1b_Ga=T,f=100;%s\x1b\\", encoded)
	}

	// Multi-chunk transmission
	var result strings.Builder
	chunks := []string{}

	for i := 0; i < len(encoded); i += chunkSize {
		end := i + chunkSize
		if end > len(encoded) {
			end = len(encoded)
		}
		chunks = append(chunks, encoded[i:end])
	}

	// First chunk
	result.WriteString(fmt.Sprintf("\x1b_Ga=T,f=100,m=1;%s\x1b\\", chunks[0]))

	// Middle chunks
	for i := 1; i < len(chunks)-1; i++ {
		result.WriteString(fmt.Sprintf("\x1b_Gm=1;%s\x1b\\", chunks[i]))
	}

	// Final chunk
	result.WriteString(fmt.Sprintf("\x1b_Gm=0;%s\x1b\\", chunks[len(chunks)-1]))

	return result.String()
}

// renderKittyImageDirect renders image data directly with Kitty protocol
func renderKittyImageDirect(imageData []byte, displayCols, displayRows int) string {
	if len(imageData) == 0 {
		return ""
	}

	// Encode image data as base64
	encoded := base64.StdEncoding.EncodeToString(imageData)

	// For large images, we need to split into chunks
	const chunkSize = 4096

	var result strings.Builder

	if len(encoded) <= chunkSize {
		// Single chunk with explicit size constraints
		// a=T: transmit and display immediately
		// f=100: PNG format
		// s=<width>,<height>: Scale image to fit within these pixel dimensions
		// Calculate max pixel dimensions based on terminal cells
		maxPixelWidth := displayCols * 10  // ~10 pixels per column
		maxPixelHeight := displayRows * 18 // ~18 pixels per row
		command := fmt.Sprintf("\x1b_Ga=T,f=100,s=%d,%d;%s\x1b\\", maxPixelWidth, maxPixelHeight, encoded)
		result.WriteString(command)
		return result.String()
	}

	// Multi-chunk transmission
	chunks := []string{}

	for i := 0; i < len(encoded); i += chunkSize {
		end := i + chunkSize
		if end > len(encoded) {
			end = len(encoded)
		}
		chunks = append(chunks, encoded[i:end])
	}

	// First chunk with format specification and size constraints
	maxPixelWidth := displayCols * 10  // ~10 pixels per column
	maxPixelHeight := displayRows * 18 // ~18 pixels per row
	result.WriteString(fmt.Sprintf("\x1b_Ga=T,f=100,s=%d,%d,m=1;%s\x1b\\", maxPixelWidth, maxPixelHeight, chunks[0]))

	// Middle chunks
	for i := 1; i < len(chunks)-1; i++ {
		result.WriteString(fmt.Sprintf("\x1b_Gm=1;%s\x1b\\", chunks[i]))
	}

	// Final chunk - this triggers the display
	result.WriteString(fmt.Sprintf("\x1b_Gm=0;%s\x1b\\", chunks[len(chunks)-1]))

	return result.String()
}

// renderKittyImageWithPlacement renders image using Kitty's placement parameters for better positioning
func renderKittyImageWithPlacement(imageData []byte, cellWidth, cellHeight int) string {
	if len(imageData) == 0 {
		return ""
	}

	// Encode image data as base64
	encoded := base64.StdEncoding.EncodeToString(imageData)

	// Use Kitty's placement parameters to position and scale the image
	// C=1: use cell units for placement
	// c=cols,rows: scale image to fit within specified cell dimensions
	const chunkSize = 4096

	if len(encoded) <= chunkSize {
		// Single chunk with placement parameters
		// a=T: transmit and display immediately
		// f=100: PNG format
		// C=1: use cell units
		// c=<cols>,<rows>: scale to fit within specified cell dimensions
		command := fmt.Sprintf("\x1b_Ga=T,f=100,C=1,c=%d,%d;%s\x1b\\", cellWidth, cellHeight, encoded)
		return command
	}

	// Multi-chunk transmission with placement
	var result strings.Builder
	chunks := []string{}

	for i := 0; i < len(encoded); i += chunkSize {
		end := i + chunkSize
		if end > len(encoded) {
			end = len(encoded)
		}
		chunks = append(chunks, encoded[i:end])
	}

	// First chunk with placement parameters
	result.WriteString(fmt.Sprintf("\x1b_Ga=T,f=100,C=1,c=%d,%d,m=1;%s\x1b\\", cellWidth, cellHeight, chunks[0]))

	// Middle chunks
	for i := 1; i < len(chunks)-1; i++ {
		result.WriteString(fmt.Sprintf("\x1b_Gm=1;%s\x1b\\", chunks[i]))
	}

	// Final chunk
	result.WriteString(fmt.Sprintf("\x1b_Gm=0;%s\x1b\\", chunks[len(chunks)-1]))

	return result.String()
}

