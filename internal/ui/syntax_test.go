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
	"strings"
	"testing"
)

func TestCodeDetector_DetectLanguage(t *testing.T) {
	detector := NewCodeDetector()

	tests := []struct {
		name        string
		content     string
		expectedLang string
		expectedCode bool
	}{
		{
			name: "Go code",
			content: `package main

import (
	"fmt"
)

func main() {
	fmt.Println("Hello, World!")
}`,
			expectedLang: "go",
			expectedCode: true,
		},
		{
			name: "Python code",
			content: `def fibonacci(n):
    """Generate Fibonacci sequence."""
    if n <= 0:
        return []
    
    fib = [0, 1]
    for i in range(2, n):
        fib.append(fib[i-1] + fib[i-2])
    
    return fib

if __name__ == "__main__":
    print(fibonacci(10))`,
			expectedLang: "python",
			expectedCode: true,
		},
		{
			name: "JavaScript code",
			content: `const express = require('express');
const app = express();

app.get('/', (req, res) => {
    res.json({ message: 'Hello, World!' });
});

app.listen(3000, () => {
    console.log('Server running');
});`,
			expectedLang: "javascript",
			expectedCode: true,
		},
		{
			name: "JSON content",
			content: `{
  "name": "test-project",
  "version": "1.0.0",
  "dependencies": {
    "express": "^4.17.1"
  }
}`,
			expectedLang: "json",
			expectedCode: true,
		},
		{
			name: "Regular text",
			content: `This is just regular text content.
It doesn't contain any programming code.
Just some normal sentences for testing.`,
			expectedLang: "",
			expectedCode: false,
		},
		{
			name: "Empty string",
			content: "",
			expectedLang: "",
			expectedCode: false,
		},
		{
			name: "Very short text",
			content: "Hi",
			expectedLang: "",
			expectedCode: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			language, isCode := detector.DetectLanguage(test.content)
			
			if isCode != test.expectedCode {
				t.Errorf("Expected isCode=%v, got %v", test.expectedCode, isCode)
			}
			
			if language != test.expectedLang {
				t.Errorf("Expected language=%q, got %q", test.expectedLang, language)
			}
		})
	}
}

func TestCodeDetector_HighlightCode(t *testing.T) {
	detector := NewCodeDetector()

	tests := []struct {
		name        string
		content     string
		language    string
		basicColors bool
	}{
		{
			name: "Go code with basic colors",
			content: `package main

func main() {
	fmt.Println("Hello")
}`,
			language:    "go",
			basicColors: true,
		},
		{
			name: "Go code with advanced colors",
			content: `package main

func main() {
	fmt.Println("Hello")
}`,
			language:    "go",
			basicColors: false,
		},
		{
			name: "Unknown language",
			content: "Some random content",
			language: "unknown",
			basicColors: true,
		},
		{
			name: "Empty language",
			content: "Some content",
			language: "",
			basicColors: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lines, err := detector.HighlightCode(test.content, test.language, test.basicColors)
			
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			
			if len(lines) == 0 {
				t.Error("Expected at least one line of output")
			}
			
			// Verify that content is preserved (join lines and check)
			result := strings.Join(lines, "\n")
			
			// For unknown/empty languages, content should be unchanged
			if test.language == "unknown" || test.language == "" {
				if result != test.content {
					t.Errorf("Expected unchanged content for unknown language, got different content")
				}
			}
		})
	}
}

func TestCodeDetector_detectByContent(t *testing.T) {
	detector := NewCodeDetector()

	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name: "Go package declaration",
			content: `package main
import "fmt"
func main() {
	fmt.Println("Hello")
}`,
			expected: "go",
		},
		{
			name: "Python function",
			content: `def hello():
    print("Hello")`,
			expected: "python",
		},
		{
			name: "JavaScript const",
			content: `const app = express();
console.log("Starting");`,
			expected: "javascript",
		},
		{
			name: "C++ include",
			content: `#include <iostream>
std::cout << "Hello";`,
			expected: "cpp",
		},
		{
			name: "SQL query",
			content: `SELECT name, email FROM users
WHERE active = 1`,
			expected: "sql",
		},
		{
			name: "Regular text",
			content: `This is just normal text.
No programming patterns here.`,
			expected: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := detector.detectByContent(test.content)
			if result != test.expected {
				t.Errorf("Expected %q, got %q", test.expected, result)
			}
		})
	}
}