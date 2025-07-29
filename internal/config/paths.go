package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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

// GetProjectDataDir returns the project-specific data directory
func GetProjectDataDir(projectPath string) (string, error) {
	dataDir, err := GetOpencodeDataDir()
	if err != nil {
		return "", err
	}

	// Try to detect git repository
	gitRoot, err := findGitRoot(projectPath)
	if err != nil {
		// Not a git repository, use global
		return filepath.Join(dataDir, "project", "global"), nil
	}

	// Generate project directory name from git root path
	// This mimics opencode's logic of sanitizing the path
	projectDirName := sanitizeProjectPath(gitRoot)
	return filepath.Join(dataDir, "project", projectDirName), nil
}

// GetStorageDir returns the storage directory for sessions
func GetStorageDir(projectPath string) (string, error) {
	projectDir, err := GetProjectDataDir(projectPath)
	if err != nil {
		return "", err
	}
	return filepath.Join(projectDir, "storage"), nil
}

// findGitRoot finds the git repository root starting from the given path
func findGitRoot(startPath string) (string, error) {
	// Make path absolute
	absPath, err := filepath.Abs(startPath)
	if err != nil {
		return "", err
	}

	// Try git command first (most reliable)
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = absPath
	output, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(output)), nil
	}

	// Fallback: manually search for .git directory
	currentPath := absPath
	for {
		gitPath := filepath.Join(currentPath, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			return currentPath, nil
		}

		parent := filepath.Dir(currentPath)
		if parent == currentPath {
			// Reached filesystem root
			break
		}
		currentPath = parent
	}

	return "", fmt.Errorf("not a git repository")
}

// sanitizeProjectPath converts a file path to a safe directory name
// This replicates opencode's path sanitization logic
func sanitizeProjectPath(path string) string {
	// Convert to absolute path and clean it
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	// Replace path separators and other problematic characters
	result := strings.ReplaceAll(absPath, string(filepath.Separator), "-")
	result = strings.ReplaceAll(result, ":", "")
	result = strings.ReplaceAll(result, " ", "-")

	// Remove leading dashes
	result = strings.TrimLeft(result, "-")

	return result
}
