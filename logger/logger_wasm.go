//go:build js

package logger

import (
	"fmt"
	"syscall/js"
)

// output writes to browser console (WASM builds).
func output(level Level, tag, msg string) {
	line := fmt.Sprintf("[%s] [%s] %s", LevelString(level), tag, msg)

	switch level {
	case LevelWarn:
		js.Global().Get("console").Call("warn", line)
	case LevelError:
		js.Global().Get("console").Call("error", line)
	default:
		js.Global().Get("console").Call("log", line)
	}
}

// CloseLog is a no-op on WASM.
func CloseLog() {}
