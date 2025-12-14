package logger

import (
	"log"
	"os"
	"strings"
)

// LogLevel represents the logging level
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

var currentLevel LogLevel = LevelInfo

func init() {
	// Set log format with date and time
	log.SetFlags(log.Ldate | log.Ltime)

	// Set log level from environment variable
	level := os.Getenv("LOG_LEVEL")
	switch strings.ToUpper(level) {
	case "DEBUG":
		currentLevel = LevelDebug
	case "WARN", "WARNING":
		currentLevel = LevelWarn
	case "ERROR":
		currentLevel = LevelError
	default:
		currentLevel = LevelInfo
	}
}

// Debug logs a debug message (only shown when LOG_LEVEL=DEBUG)
func Debug(format string, v ...interface{}) {
	if currentLevel <= LevelDebug {
		log.Printf("[DEBUG] "+format, v...)
	}
}

// Info logs an info message
func Info(format string, v ...interface{}) {
	if currentLevel <= LevelInfo {
		log.Printf("[INFO] "+format, v...)
	}
}

// Warn logs a warning message
func Warn(format string, v ...interface{}) {
	if currentLevel <= LevelWarn {
		log.Printf("[WARN] "+format, v...)
	}
}

// Error logs an error message
func Error(format string, v ...interface{}) {
	if currentLevel <= LevelError {
		log.Printf("[ERROR] "+format, v...)
	}
}

// Fatal logs a fatal message and exits the program
func Fatal(format string, v ...interface{}) {
	log.Fatalf("[FATAL] "+format, v...)
}
