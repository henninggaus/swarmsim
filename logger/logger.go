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
	BotID   int // -1 = no specific bot
}

const maxEntries = 200

var (
	mu      sync.Mutex
	entries []LogEntry
)

// Info logs an informational message.
func Info(tag, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	addBot(LevelInfo, tag, msg, -1)
	output(LevelInfo, tag, msg)
}

// Warn logs a warning message.
func Warn(tag, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	addBot(LevelWarn, tag, msg, -1)
	output(LevelWarn, tag, msg)
}

// Error logs an error message.
func Error(tag, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	addBot(LevelError, tag, msg, -1)
	output(LevelError, tag, msg)
}

// InfoBot logs an info message associated with a specific bot.
func InfoBot(botID int, tag, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	addBot(LevelInfo, tag, msg, botID)
	output(LevelInfo, tag, msg)
}

// WarnBot logs a warning message associated with a specific bot.
func WarnBot(botID int, tag, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	addBot(LevelWarn, tag, msg, botID)
	output(LevelWarn, tag, msg)
}

// Entries returns a snapshot of recent log entries for the in-game console.
func Entries() []LogEntry {
	mu.Lock()
	defer mu.Unlock()
	cp := make([]LogEntry, len(entries))
	copy(cp, entries)
	return cp
}

// EntriesForBot returns entries filtered by bot ID.
// Returns entries with matching BotID or generic entries (BotID == -1).
func EntriesForBot(botID int) []LogEntry {
	mu.Lock()
	defer mu.Unlock()
	var result []LogEntry
	for _, e := range entries {
		if e.BotID == botID || e.BotID == -1 {
			result = append(result, e)
		}
	}
	return result
}

// addBot appends an entry to the ring buffer with an optional bot ID.
func addBot(level Level, tag, msg string, botID int) {
	mu.Lock()
	defer mu.Unlock()
	entry := LogEntry{
		Time:    time.Now().Format("15:04:05"),
		Level:   level,
		Tag:     tag,
		Message: msg,
		BotID:   botID,
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
