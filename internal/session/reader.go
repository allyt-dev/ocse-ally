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
	storageDir string
}

// SessionWithProject represents a session with its associated project information
type SessionWithProject struct {
	SessionID   string
	ProjectName string
	Info        SessionInfo
}

// NewReader creates a new session reader
func NewReader(projectPath string) (*Reader, error) {
	storageDir, err := config.GetStorageDir(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage directory: %w", err)
	}

	return &Reader{
		storageDir: storageDir,
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

	// Sort sessions by creation time (newest first)
	sort.Slice(allSessions, func(i, j int) bool {
		return allSessions[i].Info.GetCreatedAt().After(allSessions[j].Info.GetCreatedAt())
	})

	return allSessions, nil
}

// ReadSessionInfo reads session metadata
func (r *Reader) ReadSessionInfo(sessionID string) (*SessionInfo, error) {
	infoPath := filepath.Join(r.storageDir, "session", "info", sessionID+".json")

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

// ReadMessages reads all messages for a session
func (r *Reader) ReadMessages(sessionID string) ([]Message, error) {
	messageDir := filepath.Join(r.storageDir, "session", "message", sessionID)

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
	partDir := filepath.Join(r.storageDir, "session", "part", sessionID, messageID)

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
