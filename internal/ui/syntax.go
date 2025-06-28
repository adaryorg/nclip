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
	"regexp"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// CodeDetector detects if text content is source code and determines the language
type CodeDetector struct {
	// Cache for lexer analysis
	lexerCache map[string]*chroma.Lexer
}

// NewCodeDetector creates a new code detector
func NewCodeDetector() *CodeDetector {
	return &CodeDetector{
		lexerCache: make(map[string]*chroma.Lexer),
	}
}

// DetectLanguage analyzes text content and returns the detected programming language
func (cd *CodeDetector) DetectLanguage(content string) (string, bool) {
	if len(content) == 0 {
		return "", false
	}

	// Clean the content for analysis
	trimmed := strings.TrimSpace(content)
	if len(trimmed) < 10 {
		return "", false // Too short to reliably detect
	}

	// Try to detect by analyzing the content
	if lang := cd.detectByContent(trimmed); lang != "" {
		return lang, true
	}

	// Try Chroma's built-in analysis
	lexer := lexers.Analyse(trimmed)
	if lexer != nil {
		config := lexer.Config()
		if config != nil && len(config.Aliases) > 0 {
			return config.Aliases[0], true
		}
	}

	return "", false
}

// detectByContent analyzes content patterns to determine language
func (cd *CodeDetector) detectByContent(content string) string {
	lines := strings.Split(content, "\n")
	
	// Remove empty lines and comments for analysis
	var codeLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) > 0 && !strings.HasPrefix(trimmed, "//") && !strings.HasPrefix(trimmed, "#") {
			codeLines = append(codeLines, trimmed)
		}
	}

	if len(codeLines) == 0 {
		return ""
	}

	// Join lines for pattern matching
	codeText := strings.Join(codeLines, "\n")

	// Language-specific patterns (ordered by specificity)
	patterns := []struct {
		lang    string
		regexes []*regexp.Regexp
	}{
		// Go language patterns
		{
			lang: "go",
			regexes: []*regexp.Regexp{
				regexp.MustCompile(`(?m)^package\s+\w+`),
				regexp.MustCompile(`import\s*\(`),
				regexp.MustCompile(`func\s+\w+\s*\(`),
				regexp.MustCompile(`var\s+\w+\s+\w+`),
				regexp.MustCompile(`type\s+\w+\s+(struct|interface)`),
				regexp.MustCompile(`fmt\.Print`),
				regexp.MustCompile(`make\(`),
				regexp.MustCompile(`range\s+\w+`),
			},
		},
		// JavaScript/TypeScript patterns
		{
			lang: "javascript",
			regexes: []*regexp.Regexp{
				regexp.MustCompile(`(?m)^(const|let|var)\s+\w+`),
				regexp.MustCompile(`function\s+\w+\s*\(`),
				regexp.MustCompile(`=>\s*{`),
				regexp.MustCompile(`console\.log`),
				regexp.MustCompile(`require\(`),
				regexp.MustCompile(`import\s+.+\s+from`),
				regexp.MustCompile(`export\s+(default|const|function)`),
			},
		},
		// Python patterns
		{
			lang: "python",
			regexes: []*regexp.Regexp{
				regexp.MustCompile(`(?m)^def\s+\w+\s*\(`),
				regexp.MustCompile(`(?m)^class\s+\w+`),
				regexp.MustCompile(`(?m)^import\s+\w+`),
				regexp.MustCompile(`(?m)^from\s+\w+\s+import`),
				regexp.MustCompile(`print\(`),
				regexp.MustCompile(`if\s+__name__\s*==\s*['""]__main__['""]`),
				regexp.MustCompile(`self\.\w+`),
			},
		},
		// Rust patterns
		{
			lang: "rust",
			regexes: []*regexp.Regexp{
				regexp.MustCompile(`(?m)^fn\s+\w+\s*\(`),
				regexp.MustCompile(`(?m)^use\s+\w+`),
				regexp.MustCompile(`(?m)^struct\s+\w+`),
				regexp.MustCompile(`(?m)^impl\s+\w+`),
				regexp.MustCompile(`println!\(`),
				regexp.MustCompile(`let\s+(mut\s+)?\w+`),
				regexp.MustCompile(`match\s+\w+\s*{`),
			},
		},
		// C/C++ patterns
		{
			lang: "cpp",
			regexes: []*regexp.Regexp{
				regexp.MustCompile(`#include\s*<.+>`),
				regexp.MustCompile(`(?m)^(int|void|char|float|double)\s+\w+\s*\(`),
				regexp.MustCompile(`std::`),
				regexp.MustCompile(`cout\s*<<`),
				regexp.MustCompile(`cin\s*>>`),
				regexp.MustCompile(`(?m)^class\s+\w+`),
				regexp.MustCompile(`(?m)^namespace\s+\w+`),
			},
		},
		// Java patterns
		{
			lang: "java",
			regexes: []*regexp.Regexp{
				regexp.MustCompile(`(?m)^public\s+class\s+\w+`),
				regexp.MustCompile(`(?m)^package\s+\w+`),
				regexp.MustCompile(`(?m)^import\s+\w+`),
				regexp.MustCompile(`public\s+static\s+void\s+main`),
				regexp.MustCompile(`System\.out\.print`),
				regexp.MustCompile(`@Override`),
				regexp.MustCompile(`extends\s+\w+`),
			},
		},
		// Shell script patterns
		{
			lang: "bash",
			regexes: []*regexp.Regexp{
				regexp.MustCompile(`(?m)^#!/bin/(ba)?sh`),
				regexp.MustCompile(`\$\{\w+\}`),
				regexp.MustCompile(`\$\w+`),
				regexp.MustCompile(`(?m)^(if|while|for)\s+\[`),
				regexp.MustCompile(`echo\s+`),
				regexp.MustCompile(`(?m)^function\s+\w+`),
			},
		},
		// JSON patterns
		{
			lang: "json",
			regexes: []*regexp.Regexp{
				regexp.MustCompile(`(?s)^\s*\{.*\}\s*$`),
				regexp.MustCompile(`(?s)^\s*\[.*\]\s*$`),
				regexp.MustCompile(`"\w+"\s*:`),
			},
		},
		// YAML patterns
		{
			lang: "yaml",
			regexes: []*regexp.Regexp{
				regexp.MustCompile(`(?m)^\w+:\s*$`),
				regexp.MustCompile(`(?m)^\s+-\s+\w+`),
				regexp.MustCompile(`(?m)^---\s*$`),
			},
		},
		// XML/HTML patterns
		{
			lang: "xml",
			regexes: []*regexp.Regexp{
				regexp.MustCompile(`<\w+[^>]*>`),
				regexp.MustCompile(`</\w+>`),
				regexp.MustCompile(`<\?xml`),
			},
		},
		// SQL patterns
		{
			lang: "sql",
			regexes: []*regexp.Regexp{
				regexp.MustCompile(`(?i)(SELECT|INSERT|UPDATE|DELETE|CREATE|ALTER|DROP)\s+`),
				regexp.MustCompile(`(?i)FROM\s+\w+`),
				regexp.MustCompile(`(?i)WHERE\s+\w+`),
				regexp.MustCompile(`(?i)JOIN\s+\w+`),
			},
		},
	}

	// Score each language based on pattern matches
	scores := make(map[string]int)
	for _, pattern := range patterns {
		score := 0
		for _, regex := range pattern.regexes {
			if regex.MatchString(codeText) {
				score++
			}
		}
		if score > 0 {
			scores[pattern.lang] = score
		}
	}

	// Return the language with the highest score
	bestLang := ""
	bestScore := 0
	for lang, score := range scores {
		if score > bestScore {
			bestScore = score
			bestLang = lang
		}
	}

	// Require at least 2 pattern matches to be confident
	if bestScore >= 2 {
		return bestLang
	}

	return ""
}

// HighlightCode applies syntax highlighting to the provided code
func (cd *CodeDetector) HighlightCode(content, language string, useBasicColors bool) ([]string, error) {
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

	// Choose style based on terminal capabilities
	var style *chroma.Style
	if useBasicColors {
		// Use a style that works well with basic colors
		style = styles.Get("bw") // Black and white style
		if style == nil {
			style = styles.Fallback
		}
	} else {
		// Use a colorful style for advanced terminals
		style = styles.Get("monokai")
		if style == nil {
			style = styles.Fallback
		}
	}

	// Create terminal formatter
	var formatter chroma.Formatter
	if useBasicColors {
		// Use 8-color formatter for basic terminals
		formatter = formatters.Get("terminal")
	} else {
		// Use 256-color formatter for advanced terminals
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