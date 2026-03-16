//go:build !js

package render

// ClipboardWrite is a no-op on native builds.
func ClipboardWrite(text string) {}

// ClipboardRead is a no-op on native builds.
func ClipboardRead(callback func(string)) {
	callback("")
}
