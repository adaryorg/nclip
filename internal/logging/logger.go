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
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

var globalLogger zerolog.Logger

// InitLogger sets up logging with file rotation and dual output (file + stdout/stderr)
func InitLogger(logFile string, level string, maxAge, maxSize, maxBackups int) error {
	// Expand ~ to home directory if present
	if strings.HasPrefix(logFile, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		logFile = filepath.Join(homeDir, logFile[2:])
	}

	// Ensure log directory exists
	logDir := filepath.Dir(logFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	// Parse log level
	logLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		logLevel = zerolog.InfoLevel // Default to info if invalid level
	}

	// Set up lumberjack for log rotation
	fileWriter := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    maxSize,    // MB
		MaxAge:     maxAge,     // days
		MaxBackups: maxBackups, // number of backups
		LocalTime:  true,
		Compress:   true, // compress old log files
	}

	// Create console writer for stdout/stderr
	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "2006-01-02 15:04:05",
		NoColor:    false,
	}

	// Set up multi-writer to write to both file and console
	multiWriter := io.MultiWriter(fileWriter, consoleWriter)

	// Configure global logger
	globalLogger = zerolog.New(multiWriter).
		Level(logLevel).
		With().
		Timestamp().
		Caller().
		Logger()

	// Also set the global zerolog logger
	log.Logger = globalLogger

	return nil
}

// Debug logs a debug message
func Debug(format string, args ...interface{}) {
	globalLogger.Debug().Msgf(format, args...)
}

// Info logs an info message
func Info(format string, args ...interface{}) {
	globalLogger.Info().Msgf(format, args...)
}

// Warn logs a warning message
func Warn(format string, args ...interface{}) {
	globalLogger.Warn().Msgf(format, args...)
}

// Error logs an error message
func Error(format string, args ...interface{}) {
	globalLogger.Error().Msgf(format, args...)
}

// Fatal logs a fatal message and exits
func Fatal(format string, args ...interface{}) {
	globalLogger.Fatal().Msgf(format, args...)
}

// SetLevel changes the logging level (legacy compatibility)
func SetLevel(level string) {
	logLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}
	globalLogger = globalLogger.Level(logLevel)
	log.Logger = globalLogger
}

// GetLogger returns the configured logger instance
func GetLogger() zerolog.Logger {
	return globalLogger
}

