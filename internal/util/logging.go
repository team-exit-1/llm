package util

import (
	"encoding/json"
	"log"
)

// Logger provides consistent logging across services
type Logger struct {
	prefix string
}

// NewLogger creates a new logger with a prefix
func NewLogger(prefix string) *Logger {
	return &Logger{prefix: prefix}
}

// Start logs the start of a process
func (l *Logger) Start(name string) {
	log.Printf("\n"+LogStart+"\n", l.prefix+": "+name)
}

// End logs the end of a process
func (l *Logger) End(name string) {
	log.Printf(LogEnd, l.prefix+": "+name)
}

// Section logs a section header
func (l *Logger) Section(name string) {
	log.Printf("\n"+LogSection+"\n", name)
}

// Error logs an error message
func (l *Logger) Error(msg string, err error) {
	log.Printf(LogError+": %s - %v\n", msg, err)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, err error) {
	log.Printf(LogWarning+": %s - %v\n", msg, err)
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	log.Printf(format+"\n", args...)
}

// JSON logs data as formatted JSON
func (l *Logger) JSON(label string, data interface{}) {
	jsonBytes, _ := json.MarshalIndent(data, "", "  ")
	log.Printf("%s:\n%s\n", label, string(jsonBytes))
}

// Success logs a success message
func (l *Logger) Success(msg string) {
	log.Printf("✓ %s\n", msg)
}

// KeyValue logs key-value pairs
func (l *Logger) KeyValue(pairs ...interface{}) {
	for i := 0; i < len(pairs)-1; i += 2 {
		key, val := pairs[i], pairs[i+1]
		log.Printf("%s: %v\n", key, val)
	}
}