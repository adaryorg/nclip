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
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// CodeDetector detects if text content is source code and determines the language using Chroma
type CodeDetector struct {
	// No fields needed - Chroma handles lexer caching internally
}

// NewCodeDetector creates a new code detector
func NewCodeDetector() *CodeDetector {
	return &CodeDetector{}
}

// DetectLanguage analyzes text content and returns the detected programming language using Chroma
func (cd *CodeDetector) DetectLanguage(content string) (string, bool) {
	if len(content) == 0 {
		return "", false
	}

	// Clean the content for analysis
//	trimmed := strings.TrimSpace(content)
//	if len(trimmed) < 10 {
//		return "", false // Too short to reliably detect
//	}

	// Use Chroma's built-in content analysis
	lexer := lexers.Analyse(content)
	if lexer != nil {
		config := lexer.Config()
		if config != nil && len(config.Aliases) > 0 {
			return config.Aliases[0], true
		}
	}

	// No language detected - assume plain text
	return "", false
}


// HighlightCode applies syntax highlighting to the provided code
func (cd *CodeDetector) HighlightCode(content, language string, useBasicColors bool) ([]string, error) {
	return cd.HighlightCodeWithTheme(content, language, useBasicColors, "")
}

// HighlightCodeWithTheme applies syntax highlighting with a custom theme
func (cd *CodeDetector) HighlightCodeWithTheme(content, language string, useBasicColors bool, themeName string) ([]string, error) {
	if language == "" {
		// Return original content split into lines
		return strings.Split(content, "\n"), nil
	}

	// Get lexer for the language
	lexer := lexers.Get(language)
	if lexer == nil {
		// Fallback to plain text
		return strings.Split(content, "\n"), nil
	}

	// Choose style based on theme configuration and terminal capabilities
	var baseStyle *chroma.Style
	if themeName != "" {
		// Use the specified theme
		baseStyle = styles.Get(themeName)
	}
	
	// Fallback to default behavior if no theme specified or theme not found
	if baseStyle == nil {
		if useBasicColors {
			// Use a style that works well with basic colors
			baseStyle = styles.Get("bw") // Black and white style
			if baseStyle == nil {
				baseStyle = styles.Fallback
			}
		} else {
			// Use a colorful style for advanced terminals
			baseStyle = styles.Get("monokai")
			if baseStyle == nil {
				baseStyle = styles.Fallback
			}
		}
	}
	
	// Use the base style as is
	// Background override will be handled in the text view rendering
	style := baseStyle

	// Create terminal formatter with no background colors
	var formatter chroma.Formatter
	if useBasicColors {
		// Use 8-color formatter for basic terminals with no background
		formatter = formatters.Get("terminal")
	} else {
		// Use 256-color formatter for advanced terminals with no background
		formatter = formatters.Get("terminal256")
	}
	
	if formatter == nil {
		formatter = formatters.Fallback
	}

	// Tokenize the code
	iterator, err := lexer.Tokenise(nil, content)
	if err != nil {
		return strings.Split(content, "\n"), err
	}

	// Format the tokens
	var buf bytes.Buffer
	err = formatter.Format(&buf, style, iterator)
	if err != nil {
		return strings.Split(content, "\n"), err
	}

	// Split into lines and return
	highlighted := buf.String()
	return strings.Split(highlighted, "\n"), nil
}

