//go:build !js

package render

import (
	"os"
	"path/filepath"
	"swarmsim/logger"
)

func tutorialDoneDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".swarmsim")
}

// IsTutorialDone checks if the tutorial has been completed before.
func IsTutorialDone() bool {
	dir := tutorialDoneDir()
	if dir == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(dir, "tutorial_done"))
	return err == nil
}

// MarkTutorialDone saves the tutorial-done flag.
func MarkTutorialDone() {
	dir := tutorialDoneDir()
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		logger.Warn("TUTORIAL", "Failed to create config dir: %v", err)
		return
	}
	if err := os.WriteFile(filepath.Join(dir, "tutorial_done"), []byte("done"), 0o644); err != nil {
		logger.Warn("TUTORIAL", "Failed to save tutorial state: %v", err)
	}
}
