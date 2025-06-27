package ui

import (
	"fmt"
	"strings"
)

// renderImageViewNew renders the image viewer using the same approach as help window
func (m Model) renderImageViewNew() string {
	if m.viewingImage == nil || len(m.viewingImage.ImageData) == 0 {
		return "No image to display"
	}

	// Ensure minimum terminal size
	if m.width < 20 || m.height < 10 {
		return "Terminal too small for image viewer"
	}

	if !detectKittySupport() {
		// Fallback for non-Kitty terminals
		return m.renderSimpleImageView()
	}

	// Get image dimensions for display info
	imageWidth, imageHeight, format, err := getImageDimensions(m.viewingImage.ImageData)

	// Calculate dialog dimensions - same as help window
	dialogWidth := m.width - 2   // 1 char padding left and right
	dialogHeight := m.height - 2 // 1 char padding top and bottom

	// Calculate content area within the dialog
	contentWidth := dialogWidth - 4   // Border + internal padding
	contentHeight := dialogHeight - 4 // Border + header + footer

	if contentWidth < 10 {
		contentWidth = 10
	}
	if contentHeight < 5 {
		contentHeight = 5
	}

	// Create header text
	var headerText string
	if err == nil && format != "" {
		headerText = fmt.Sprintf("Image View (%dx%d %s, %d bytes)",
			imageWidth, imageHeight, strings.ToUpper(format), len(m.viewingImage.ImageData))
	} else {
		headerText = fmt.Sprintf("Image View (%d bytes)", len(m.viewingImage.ImageData))
	}

	// Reserve space for image
	imageAreaLines := contentHeight - 3 // -3 for header, separator, footer
	var imageSpaceContent strings.Builder
	for i := 0; i < imageAreaLines; i++ {
		imageSpaceContent.WriteString("\n")
	}

	// Footer
	// Create footer text based on delete confirmation state
	var footerText string
	if m.imageDeletePending {
		footerText = "Press 'd' again to confirm deletion, any other key to cancel"
	} else {
		footerText = "o: open • enter: copy • e: edit • d: delete • any other key: close"
	}

	// Build frame content using shared function
	frameContent := m.buildFrameContent(headerText, imageSpaceContent.String(), footerText, contentWidth)

	// Create framed dialog using shared function
	positioned := m.createFramedDialog(dialogWidth, dialogHeight, frameContent)

	// Now add the image on top of the positioned frame
	var result strings.Builder
	result.WriteString(positioned)

	// Calculate where to place the image within the positioned frame
	// The dialog is centered, so we need to find its actual position
	dialogStartY := (m.height-dialogHeight)/2 + 1 // +1 for 1-based coordinates
	dialogStartX := (m.width-dialogWidth)/2 + 1   // +1 for 1-based coordinates

	// Image position: inside the dialog border + after header + separator
	imageX := dialogStartX + 2 // Dialog border + padding
	imageY := dialogStartY + 3 // Dialog border + header + separator

	// Position cursor and render image
	result.WriteString(fmt.Sprintf("\x1b[%d;%dH", imageY, imageX))

	// Calculate available space for image scaling
	availableWidth := contentWidth
	availableHeight := imageAreaLines

	// Render image with size constraints
	imageData := m.viewingImage.ImageData
	if err == nil && (imageWidth > availableWidth*8 || imageHeight > availableHeight*16) {
		// Need to scale down
		imageData = m.scaleImageForFrame(imageData, imageWidth, imageHeight, availableWidth, availableHeight)
	}

	imageDisplay := renderSimpleKittyImage(imageData)
	result.WriteString(imageDisplay)

	return result.String()
}

// drawSimpleFrame draws a minimal frame using ANSI positioning
func (m Model) drawSimpleFrame(startX, startY, width, height int, format string, imgWidth, imgHeight int) string {
	var result strings.Builder

	// Create title
	var title string
	if format != "" {
		title = fmt.Sprintf("Image View (%dx%d %s, %d bytes)",
			imgWidth, imgHeight, strings.ToUpper(format), len(m.viewingImage.ImageData))
	} else {
		title = fmt.Sprintf("Image View (%d bytes)", len(m.viewingImage.ImageData))
	}

	// Ensure title fits
	if len(title) > width-2 {
		title = title[:width-5] + "..."
	}

	// Top border
	result.WriteString(fmt.Sprintf("\x1b[%d;%dH", startY, startX))
	result.WriteString("╭" + strings.Repeat("─", width-2) + "╮")

	// Title line (header)
	result.WriteString(fmt.Sprintf("\x1b[%d;%dH", startY+1, startX))
	titlePadding := (width - 2 - len(title)) / 2
	result.WriteString("│" + strings.Repeat(" ", titlePadding) + title +
		strings.Repeat(" ", width-2-titlePadding-len(title)) + "│")

	// Side borders for content area (start from row 2, right after header)
	for y := startY + 2; y < startY+height-2; y++ {
		result.WriteString(fmt.Sprintf("\x1b[%d;%dH│", y, startX))
		result.WriteString(fmt.Sprintf("\x1b[%d;%dH│", y, startX+width-1))
	}

	// Bottom area with footer
	result.WriteString(fmt.Sprintf("\x1b[%d;%dH", startY+height-2, startX))
	// Create footer text based on delete confirmation state
	var footerText string
	if m.imageDeletePending {
		footerText = "Press 'd' again to confirm deletion, any other key to cancel"
	} else {
		footerText = "o: open • enter: copy • e: edit • d: delete • any other key: close"
	}
	if len(footerText) > width-2 {
		footerText = footerText[:width-5] + "..."
	}
	footerPadding := (width - 2 - len(footerText)) / 2
	result.WriteString("│" + strings.Repeat(" ", footerPadding) + footerText +
		strings.Repeat(" ", width-2-footerPadding-len(footerText)) + "│")

	// Bottom border
	result.WriteString(fmt.Sprintf("\x1b[%d;%dH", startY+height-1, startX))
	result.WriteString("╰" + strings.Repeat("─", width-2) + "╯")

	return result.String()
}
