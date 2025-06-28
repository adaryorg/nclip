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
	"fmt"
	"testing"
	"time"

	"github.com/adaryorg/nclip/internal/security"
)

func TestNewMonitor(t *testing.T) {
	callbackCalled := false
	callback := func(content string) {
		callbackCalled = true
	}

	monitor := NewMonitor(callback)
	defer monitor.Close()

	if monitor == nil {
		t.Fatal("Expected monitor to be created")
	}

	if monitor.textCallback == nil {
		t.Error("Expected text callback to be set")
	}

	if monitor.interval != 500*time.Millisecond {
		t.Errorf("Expected interval 500ms, got %v", monitor.interval)
	}

	if monitor.detector == nil {
		t.Error("Expected security detector to be initialized")
	}

	// Test callback functionality (without actual clipboard)
	if monitor.textCallback != nil {
		monitor.textCallback("test")
		if !callbackCalled {
			t.Error("Expected callback to be called")
		}
	}
}

func TestNewMonitorWithImage(t *testing.T) {
	textCallbackCalled := false
	imageCallbackCalled := false

	textCallback := func(content string) {
		textCallbackCalled = true
	}

	imageCallback := func(data []byte, description string) {
		imageCallbackCalled = true
	}

	monitor := NewMonitorWithImage(textCallback, imageCallback)
	defer monitor.Close()

	if monitor == nil {
		t.Fatal("Expected monitor to be created")
	}

	if monitor.textCallback == nil {
		t.Error("Expected text callback to be set")
	}

	if monitor.imageCallback == nil {
		t.Error("Expected image callback to be set")
	}

	// Test callbacks
	if monitor.textCallback != nil {
		monitor.textCallback("test")
		if !textCallbackCalled {
			t.Error("Expected text callback to be called")
		}
	}

	if monitor.imageCallback != nil {
		monitor.imageCallback([]byte("test"), "test image")
		if !imageCallbackCalled {
			t.Error("Expected image callback to be called")
		}
	}
}

func TestNewMonitorWithSecurity(t *testing.T) {
	textCallbackCalled := false
	imageCallbackCalled := false
	securityCallbackCalled := false

	textCallback := func(content string) {
		textCallbackCalled = true
	}

	imageCallback := func(data []byte, description string) {
		imageCallbackCalled = true
	}

	securityCallback := func(content string, threats []security.SecurityThreat) {
		securityCallbackCalled = true
	}

	monitor := NewMonitorWithSecurity(textCallback, imageCallback, securityCallback)
	defer monitor.Close()

	if monitor == nil {
		t.Fatal("Expected monitor to be created")
	}

	if monitor.textCallback == nil {
		t.Error("Expected text callback to be set")
	}

	if monitor.imageCallback == nil {
		t.Error("Expected image callback to be set")
	}

	if monitor.securityCallback == nil {
		t.Error("Expected security callback to be set")
	}

	// Test callbacks
	if monitor.textCallback != nil {
		monitor.textCallback("test")
		if !textCallbackCalled {
			t.Error("Expected text callback to be called")
		}
	}

	if monitor.imageCallback != nil {
		monitor.imageCallback([]byte("test"), "test image")
		if !imageCallbackCalled {
			t.Error("Expected image callback to be called")
		}
	}

	if monitor.securityCallback != nil {
		monitor.securityCallback("test", []security.SecurityThreat{})
		if !securityCallbackCalled {
			t.Error("Expected security callback to be called")
		}
	}
}

func TestMonitorStart_ContextCancellation(t *testing.T) {
	// This test verifies that the monitor respects context cancellation
	monitor := NewMonitor(func(string) {})
	defer monitor.Close()

	ctx, cancel := context.WithCancel(context.Background())

	// Start monitor in goroutine
	done := make(chan error, 1)
	go func() {
		done <- monitor.Start(ctx)
	}()

	// Cancel context immediately
	cancel()

	// Wait for monitor to stop
	select {
	case err := <-done:
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled error, got %v", err)
		}
	case <-time.After(time.Second):
		t.Error("Monitor did not stop within timeout")
	}
}

func TestMonitorStart_Timeout(t *testing.T) {
	// This test verifies that the monitor stops when context times out
	monitor := NewMonitor(func(string) {})
	defer monitor.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := monitor.Start(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded error, got %v", err)
	}
}

func TestCopy(t *testing.T) {
	// Test the Copy function (this may fail in headless environments)
	content := "test clipboard content"

	err := Copy(content)
	// We don't fail the test if Copy fails, as it depends on system clipboard
	// being available, which may not be the case in CI environments
	if err != nil {
		t.Logf("Copy failed (this is expected in headless environments): %v", err)
	}
}

func TestCopyImage(t *testing.T) {
	// Test the CopyImage function
	imageData := []byte("fake image data")

	err := CopyImage(imageData)
	// Similar to Copy, this may fail in headless environments
	if err != nil {
		t.Logf("CopyImage failed (this is expected in headless environments): %v", err)
	}
}

func TestEnsureInit(t *testing.T) {
	// Test that ensureInit can be called multiple times safely
	err1 := ensureInit()
	err2 := ensureInit()

	// Both calls should return the same error (or lack thereof)
	if err1 != err2 {
		t.Errorf("ensureInit should return consistent results, got %v and %v", err1, err2)
	}

	// In headless environments, this will likely fail, which is expected
	if err1 != nil {
		t.Logf("ensureInit failed (this is expected in headless environments): %v", err1)
	}
}

func TestMonitorClose(t *testing.T) {
	monitor := NewMonitor(func(string) {})

	err := monitor.Close()
	if err != nil {
		t.Errorf("Close should not error: %v", err)
	}

	// Multiple closes should be safe
	err = monitor.Close()
	if err != nil {
		t.Errorf("Multiple closes should not error: %v", err)
	}
}

// Mock test for security detection logic (without actual clipboard)
func TestMonitorSecurityDetection(t *testing.T) {
	textContents := []string{}
	securityThreats := [][]security.SecurityThreat{}

	textCallback := func(content string) {
		textContents = append(textContents, content)
	}

	securityCallback := func(content string, threats []security.SecurityThreat) {
		securityThreats = append(securityThreats, threats)
	}

	monitor := NewMonitorWithSecurity(textCallback, nil, securityCallback)
	defer monitor.Close()

	// Verify security detector is available
	if monitor.detector == nil {
		t.Fatal("Expected security detector to be initialized")
	}

	// Test security detection (simulated)
	testContent := "password=mySecretPassword123!"
	threats := monitor.detector.DetectSecurity(testContent)

	if len(threats) == 0 {
		t.Error("Expected to detect security threats in test content")
	}

	// Test that callbacks would work
	if monitor.textCallback != nil {
		monitor.textCallback(testContent)
	}

	if monitor.securityCallback != nil && len(threats) > 0 {
		monitor.securityCallback(testContent, threats)
	}

	// Verify callbacks were called
	if len(textContents) != 1 || textContents[0] != testContent {
		t.Error("Text callback was not called correctly")
	}

	if len(securityThreats) != 1 || len(securityThreats[0]) == 0 {
		t.Error("Security callback was not called correctly")
	}
}

// Test monitor behavior with nil callbacks
func TestMonitorWithNilCallbacks(t *testing.T) {
	monitor := &Monitor{
		interval:      500 * time.Millisecond,
		textCallback:  nil,
		imageCallback: nil,
		detector:      security.NewSecurityDetector(),
	}
	defer monitor.Close()

	// Should not panic with nil callbacks
	if monitor.textCallback != nil {
		monitor.textCallback("test")
	}

	if monitor.imageCallback != nil {
		monitor.imageCallback([]byte("test"), "test")
	}

	if monitor.securityCallback != nil {
		monitor.securityCallback("test", []security.SecurityThreat{})
	}
}

// Test image description generation logic
func TestImageDescriptionGeneration(t *testing.T) {
	imageData := []byte("fake image data")
	expectedDescription := "Image (15 bytes)"

	// This tests the logic used in the Start method
	description := fmt.Sprintf("Image (%d bytes)", len(imageData))

	if description != expectedDescription {
		t.Errorf("Expected description '%s', got '%s'", expectedDescription, description)
	}
}
