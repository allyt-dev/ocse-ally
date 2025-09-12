package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/byteowlz/ocse/internal/config"
)

// Reader handles reading session data from the filesystem
type Reader struct {
	storageDir  string
	projectPath string // For filtering sessions in new format
}

// SessionWithProject represents a session with its associated project information
type SessionWithProject struct {
	SessionID   string
	ProjectName string
	Info        SessionInfo
}

// NewReader creates a new session reader
func NewReader(projectPath string) (*Reader, error) {
	// First try the old format (project-based storage)
	storageDir, err := config.GetStorageDir(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage directory: %w", err)
	}

	// Check if old format exists
	if _, err := os.Stat(filepath.Join(storageDir, "session")); err == nil {
		return &Reader{
			storageDir: storageDir,
		}, nil
	}

	// Try new format (hash-based storage in main storage directory)
	dataDir, err := config.GetOpencodeDataDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get opencode data directory: %w", err)
	}

	mainStorageDir := filepath.Join(dataDir, "storage")
	return &Reader{
		storageDir:  mainStorageDir,
		projectPath: projectPath,
	}, nil
}

// NewGlobalReader creates a new session reader for accessing all projects
func NewGlobalReader() (*Reader, error) {
	return &Reader{
		storageDir: "", // Will be set dynamically for each project
	}, nil
}

// ListSessions returns all available session IDs
func (r *Reader) ListSessions() ([]string, error) {
	// Check if this is the new format (hash-based storage)
	if r.projectPath != "" {
		return r.listSessionsNewFormat()
	}

	// Old format (project-based storage)
	sessionInfoDir := filepath.Join(r.storageDir, "session", "info")

	entries, err := os.ReadDir(sessionInfoDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read session info directory: %w", err)
	}

	var sessionIDs []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".json") {
			sessionID := strings.TrimSuffix(entry.Name(), ".json")
			sessionIDs = append(sessionIDs, sessionID)
		}
	}

	return sessionIDs, nil
}

// listSessionsNewFormat lists sessions in the new hash-based format
func (r *Reader) listSessionsNewFormat() ([]string, error) {
	sessionDir := filepath.Join(r.storageDir, "session")

	// Read all project directories (hashes)
	projectDirs, err := os.ReadDir(sessionDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read session directory: %w", err)
	}

	absProjectPath, _ := filepath.Abs(r.projectPath)

	var sessionIDs []string
	for _, projectDir := range projectDirs {
		if !projectDir.IsDir() {
			continue
		}

		// Read session files in this project directory
		projectSessionDir := filepath.Join(sessionDir, projectDir.Name())
		sessionFiles, err := os.ReadDir(projectSessionDir)
		if err != nil {
			continue
		}

		// Check each session file to see if it belongs to our project
		for _, sessionFile := range sessionFiles {
			if !strings.HasSuffix(sessionFile.Name(), ".json") {
				continue
			}

			// Read the session metadata to check the directory
			sessionPath := filepath.Join(projectSessionDir, sessionFile.Name())
			data, err := os.ReadFile(sessionPath)
			if err != nil {
				continue
			}

			var sessionMeta struct {
				ID        string `json:"id"`
				Directory string `json:"directory"`
			}
			if err := json.Unmarshal(data, &sessionMeta); err != nil {
				continue
			}

			// Check if this session belongs to our project
			if sessionMeta.Directory == absProjectPath {
				sessionIDs = append(sessionIDs, sessionMeta.ID)
			}
		}
	}

	return sessionIDs, nil
}

// ListAllSessions returns all sessions from all projects
func (r *Reader) ListAllSessions() ([]SessionWithProject, error) {
	dataDir, err := config.GetOpencodeDataDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get opencode data directory: %w", err)
	}

	projectsDir := filepath.Join(dataDir, "project")

	// Check if projects directory exists
	if _, err := os.Stat(projectsDir); os.IsNotExist(err) {
		return []SessionWithProject{}, nil
	}

	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read projects directory: %w", err)
	}

	var allSessions []SessionWithProject

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectName := entry.Name()
		storageDir := filepath.Join(projectsDir, projectName, "storage")

		// Create a temporary reader for this project
		projectReader := &Reader{storageDir: storageDir}

		sessionIDs, err := projectReader.ListSessions()
		if err != nil {
			continue // Skip projects with errors
		}

		for _, sessionID := range sessionIDs {
			info, err := projectReader.ReadSessionInfo(sessionID)
			if err != nil {
				continue // Skip sessions with errors
			}

			allSessions = append(allSessions, SessionWithProject{
				SessionID:   sessionID,
				ProjectName: projectName,
				Info:        *info,
			})
		}
	}

	// Sort sessions by update time (most recently active first)
	sort.Slice(allSessions, func(i, j int) bool {
		return allSessions[i].Info.GetUpdatedAt().After(allSessions[j].Info.GetUpdatedAt())
	})

	return allSessions, nil
}

// ReadSessionInfo reads session metadata
func (r *Reader) ReadSessionInfo(sessionID string) (*SessionInfo, error) {
	var infoPath string

	if r.projectPath != "" {
		// New format: find the session file
		sessionPath, err := r.findSessionFile(sessionID)
		if err != nil {
			return nil, err
		}
		infoPath = sessionPath
	} else {
		// Old format
		infoPath = filepath.Join(r.storageDir, "session", "info", sessionID+".json")
	}

	data, err := os.ReadFile(infoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read session info: %w", err)
	}

	var info SessionInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("failed to parse session info: %w", err)
	}

	return &info, nil
}

// findSessionFile finds a session file in the new format
func (r *Reader) findSessionFile(sessionID string) (string, error) {
	sessionDir := filepath.Join(r.storageDir, "session")

	// Search through all project directories
	projectDirs, err := os.ReadDir(sessionDir)
	if err != nil {
		return "", fmt.Errorf("failed to read session directory: %w", err)
	}

	for _, projectDir := range projectDirs {
		if !projectDir.IsDir() {
			continue
		}

		// Look for the session file
		sessionPath := filepath.Join(sessionDir, projectDir.Name(), sessionID+".json")
		if _, err := os.Stat(sessionPath); err == nil {
			return sessionPath, nil
		}
	}

	return "", fmt.Errorf("session %s not found", sessionID)
}

// ReadMessages reads all messages for a session
func (r *Reader) ReadMessages(sessionID string) ([]Message, error) {
	var messageDir string

	if r.projectPath != "" {
		// New format: messages are in session-specific subdirectory
		messageDir = filepath.Join(r.storageDir, "message", sessionID)
	} else {
		// Old format
		messageDir = filepath.Join(r.storageDir, "session", "message", sessionID)
	}

	entries, err := os.ReadDir(messageDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Message{}, nil
		}
		return nil, fmt.Errorf("failed to read message directory: %w", err)
	}

	var messages []Message
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".json") {
			messagePath := filepath.Join(messageDir, entry.Name())
			data, err := os.ReadFile(messagePath)
			if err != nil {
				continue // Skip corrupted files
			}

			var message Message
			if err := json.Unmarshal(data, &message); err != nil {
				continue // Skip corrupted files
			}

			messages = append(messages, message)
		}
	}

	// Sort messages by creation time
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].GetCreatedAt().Before(messages[j].GetCreatedAt())
	})

	return messages, nil
}

// ReadMessageParts reads all parts for a specific message
func (r *Reader) ReadMessageParts(sessionID, messageID string) ([]MessagePart, error) {
	var partDir string

	if r.projectPath != "" {
		// New format: parts are in top-level part directory under messageID
		partDir = filepath.Join(r.storageDir, "part", messageID)
	} else {
		// Old format
		partDir = filepath.Join(r.storageDir, "session", "part", sessionID, messageID)
	}

	entries, err := os.ReadDir(partDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []MessagePart{}, nil
		}
		return nil, fmt.Errorf("failed to read part directory: %w", err)
	}

	var parts []MessagePart
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".json") {
			partPath := filepath.Join(partDir, entry.Name())
			data, err := os.ReadFile(partPath)
			if err != nil {
				continue // Skip corrupted files
			}

			var part MessagePart
			if err := json.Unmarshal(data, &part); err != nil {
				continue // Skip corrupted files
			}

			parts = append(parts, part)
		}
	}

	// Sort parts by creation time
	sort.Slice(parts, func(i, j int) bool {
		return parts[i].GetCreatedAt().Before(parts[j].GetCreatedAt())
	})

	return parts, nil
}

// ReadSession reads a complete session with all its data
func (r *Reader) ReadSession(sessionID string) (*Session, error) {
	info, err := r.ReadSessionInfo(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to read session info: %w", err)
	}

	messages, err := r.ReadMessages(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to read messages: %w", err)
	}

	var allParts []MessagePart
	for _, message := range messages {
		parts, err := r.ReadMessageParts(sessionID, message.ID)
		if err != nil {
			continue // Skip messages with corrupted parts
		}
		allParts = append(allParts, parts...)
	}

	return &Session{
		Info:     *info,
		Messages: messages,
		Parts:    allParts,
	}, nil
}
