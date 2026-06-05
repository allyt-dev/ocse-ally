package config

import (
	"os"
	"path/filepath"
	"runtime"
)

// GetOpencodeDataDir returns the opencode data directory path
// following the same logic as the opencode binary
func GetOpencodeDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Check XDG_DATA_HOME first (Linux/Unix standard)
	if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
		return filepath.Join(xdgData, "opencode"), nil
	}

	// Platform-specific defaults
	switch runtime.GOOS {
	case "windows":
		// Windows: %USERPROFILE%\.local\share\opencode
		return filepath.Join(home, ".local", "share", "opencode"), nil
	default:
		// macOS/Linux: ~/.local/share/opencode
		return filepath.Join(home, ".local", "share", "opencode"), nil
	}
}

// GetDBPath returns the path to the SQLite database file
func GetDBPath() (string, error) {
	dataDir, err := GetOpencodeDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "opencode.db"), nil
}
