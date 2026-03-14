//go:build !js

package logger

import (
	"fmt"
	"os"
	"sync"
	"time"
)

var (
	logFile     *os.File
	logFileOnce sync.Once
)

// initLogFile lazily opens the log file on first write.
func initLogFile() {
	logFileOnce.Do(func() {
		f, err := os.OpenFile("swarmsim.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[LOGGER] Failed to open swarmsim.log: %v\n", err)
			return
		}
		logFile = f
	})
}

// output writes to stdout and log file (native builds).
func output(level Level, tag, msg string) {
	ts := time.Now().Format("2006-01-02 15:04:05")
	line := fmt.Sprintf("[%s] [%s] [%s] %s\n", ts, LevelString(level), tag, msg)

	// Always print to stdout
	fmt.Print(line)

	// Also write to log file
	initLogFile()
	if logFile != nil {
		logFile.WriteString(line)
	}
}

// CloseLog closes the log file. Call on clean shutdown.
func CloseLog() {
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
}
