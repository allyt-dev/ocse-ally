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
