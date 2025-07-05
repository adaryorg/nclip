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

package logging

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestLoggingLevels(t *testing.T) {
	// Test that logging levels are defined correctly
	if DEBUG >= INFO {
		t.Error("DEBUG should be less than INFO")
	}
	if INFO >= WARN {
		t.Error("INFO should be less than WARN")
	}
	if WARN >= ERROR {
		t.Error("WARN should be less than ERROR")
	}
}

func TestSetAndGetLevel(t *testing.T) {
	// Save original level
	original := currentLevel
	defer func() { currentLevel = original }()

	// Test setting different levels
	tests := []struct {
		input    string
		expected string
	}{
		{"debug", "debug"},
		{"DEBUG", "debug"},
		{"info", "info"},
		{"INFO", "info"},
		{"warn", "warn"},
		{"WARN", "warn"},
		{"warning", "warn"}, // Alternative name
		{"error", "error"},
		{"ERROR", "error"},
		{"invalid", "error"}, // Should default to error
		{"", "error"},        // Should default to error
	}

	for _, test := range tests {
		SetLevel(test.input)
		if GetLevel() != test.expected {
			t.Errorf("SetLevel(%q): expected %q, got %q", test.input, test.expected, GetLevel())
		}
	}
}

func TestShouldLog(t *testing.T) {
	// Save original level
	original := currentLevel
	defer func() { currentLevel = original }()

	// Test at INFO level
	SetLevel("info")
	
	if shouldLog(DEBUG) {
		t.Error("DEBUG should not log when level is INFO")
	}
	if !shouldLog(INFO) {
		t.Error("INFO should log when level is INFO")
	}
	if !shouldLog(WARN) {
		t.Error("WARN should log when level is INFO")
	}
	if !shouldLog(ERROR) {
		t.Error("ERROR should log when level is INFO")
	}
}

func TestLogFunctions(t *testing.T) {
	// Save original level and log output
	original := currentLevel
	defer func() { currentLevel = original }()

	// Capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil) // Reset to default

	// Set to DEBUG level to test all log functions
	SetLevel("debug")

	tests := []struct {
		name     string
		logFunc  func(string, ...interface{})
		expected string
	}{
		{"Debug", Debug, "DEBUG:"},
		{"Info", Info, "INFO:"},
		{"Warn", Warn, "WARN:"},
		{"Error", Error, "ERROR:"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			buf.Reset()
			test.logFunc("test message %s", "arg")
			
			output := buf.String()
			if !strings.Contains(output, test.expected) {
				t.Errorf("Expected log output to contain '%s', got: %s", test.expected, output)
			}
			if !strings.Contains(output, "test message arg") {
				t.Errorf("Expected log output to contain formatted message, got: %s", output)
			}
		})
	}
}

func TestLogLevelFiltering(t *testing.T) {
	// Save original level and log output
	original := currentLevel
	defer func() { currentLevel = original }()

	// Capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil) // Reset to default

	// Set to WARN level
	SetLevel("warn")

	// Debug and Info should not log
	buf.Reset()
	Debug("debug message")
	if buf.String() != "" {
		t.Error("Debug should not log when level is WARN")
	}

	buf.Reset()
	Info("info message")
	if buf.String() != "" {
		t.Error("Info should not log when level is WARN")
	}

	// Warn and Error should log
	buf.Reset()
	Warn("warn message")
	if !strings.Contains(buf.String(), "WARN:") {
		t.Error("Warn should log when level is WARN")
	}

	buf.Reset()
	Error("error message")
	if !strings.Contains(buf.String(), "ERROR:") {
		t.Error("Error should log when level is WARN")
	}
}

func TestLevelCoverage(t *testing.T) {
	// Test that all level constants are properly defined
	levels := []Level{DEBUG, INFO, WARN, ERROR}
	
	for _, level := range levels {
		if _, exists := levelNames[level]; !exists {
			t.Errorf("Level %d not found in levelNames map", level)
		}
	}
	
	// Test that levelNames map has expected size
	if len(levelNames) != 4 {
		t.Errorf("Expected 4 levels in levelNames, got %d", len(levelNames))
	}
}