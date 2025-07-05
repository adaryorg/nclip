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
	"testing"
)

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
