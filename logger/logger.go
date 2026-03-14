package logger

import (
	"fmt"
	"sync"
	"time"
)

// Level represents the severity of a log entry.
type Level int

const (
	LevelInfo  Level = iota // [INFO]
	LevelWarn               // [WARN]
	LevelError              // [ERROR]
)

// LogEntry is a single log message with metadata.
type LogEntry struct {
	Time    string
	Level   Level
	Tag     string // e.g. "SWARM", "GIF", "KEY"
	Message string
}

const maxEntries = 200

var (
	mu      sync.Mutex
	entries []LogEntry
)

// Info logs an informational message.
func Info(tag, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	add(LevelInfo, tag, msg)
	output(LevelInfo, tag, msg)
}

// Warn logs a warning message.
func Warn(tag, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	add(LevelWarn, tag, msg)
	output(LevelWarn, tag, msg)
}

// Error logs an error message.
func Error(tag, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	add(LevelError, tag, msg)
	output(LevelError, tag, msg)
}

// Entries returns a snapshot of recent log entries for the in-game console.
func Entries() []LogEntry {
	mu.Lock()
	defer mu.Unlock()
	cp := make([]LogEntry, len(entries))
	copy(cp, entries)
	return cp
}

// add appends an entry to the ring buffer.
func add(level Level, tag, msg string) {
	mu.Lock()
	defer mu.Unlock()
	entry := LogEntry{
		Time:    time.Now().Format("15:04:05"),
		Level:   level,
		Tag:     tag,
		Message: msg,
	}
	entries = append(entries, entry)
	if len(entries) > maxEntries {
		entries = entries[len(entries)-maxEntries:]
	}
}

// LevelString returns the display string for a log level.
func LevelString(l Level) string {
	switch l {
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "???"
	}
}
