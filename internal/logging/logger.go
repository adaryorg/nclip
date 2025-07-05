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
	"log"
	"strings"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

var (
	currentLevel Level = ERROR
	levelNames = map[Level]string{
		DEBUG: "DEBUG",
		INFO:  "INFO",
		WARN:  "WARN",
		ERROR: "ERROR",
	}
)

func SetLevel(level string) {
	switch strings.ToLower(level) {
	case "debug":
		currentLevel = DEBUG
	case "info":
		currentLevel = INFO
	case "warn", "warning":
		currentLevel = WARN
	case "error":
		currentLevel = ERROR
	default:
		currentLevel = ERROR
	}
}

func GetLevel() string {
	return strings.ToLower(levelNames[currentLevel])
}

func shouldLog(level Level) bool {
	return level >= currentLevel
}

func Debug(format string, args ...interface{}) {
	if shouldLog(DEBUG) {
		log.Printf("DEBUG: "+format, args...)
	}
}

func Info(format string, args ...interface{}) {
	if shouldLog(INFO) {
		log.Printf("INFO: "+format, args...)
	}
}

func Warn(format string, args ...interface{}) {
	if shouldLog(WARN) {
		log.Printf("WARN: "+format, args...)
	}
}

func Error(format string, args ...interface{}) {
	if shouldLog(ERROR) {
		log.Printf("ERROR: "+format, args...)
	}
}

