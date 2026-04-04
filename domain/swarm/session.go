package swarm

import (
	"os"
	"path/filepath"
)

// sessionPath returns the path to the session save file.
func sessionPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "swarmsim", "session.txt"), nil
}

// SaveSession persists the current swarm program text to disk.
func SaveSession(program string) {
	path, err := sessionPath()
	if err != nil {
		return
	}
	os.MkdirAll(filepath.Dir(path), 0755)
	os.WriteFile(path, []byte(program), 0644)
}

// LoadSession reads the last saved swarm program from disk.
// Returns empty string if no session exists or on error.
func LoadSession() string {
	path, err := sessionPath()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}
