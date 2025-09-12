package session

import (
	"encoding/json"
	"time"
)

// SessionInfo represents the session metadata
type SessionInfo struct {
	ID       string   `json:"id"`
	ParentID *string  `json:"parentID,omitempty"`
	Title    string   `json:"title"`
	Version  string   `json:"version"`
	Time     TimeInfo `json:"time"`
	ShareURL *string  `json:"shareUrl,omitempty"`
}

// TimeInfo represents the time information in session
type TimeInfo struct {
	Created int64 `json:"created"`
	Updated int64 `json:"updated"`
}

// GetCreatedAt returns the creation time as a time.Time
func (s *SessionInfo) GetCreatedAt() time.Time {
	return time.UnixMilli(s.Time.Created)
}

// GetUpdatedAt returns the update time as a time.Time
func (s *SessionInfo) GetUpdatedAt() time.Time {
	return time.UnixMilli(s.Time.Updated)
}

// Message represents a message in the session
type Message struct {
	ID        string    `json:"id"`
	SessionID string    `json:"sessionID"`
	Role      string    `json:"role"` // "user" or "assistant"
	Time      *TimeInfo `json:"time,omitempty"`

	// Assistant-specific fields
	Model        *string  `json:"model,omitempty"`
	Provider     *string  `json:"provider,omitempty"`
	Cost         *float64 `json:"cost,omitempty"`
	InputTokens  *int     `json:"inputTokens,omitempty"`
	OutputTokens *int     `json:"outputTokens,omitempty"`
	CompletedAt  *int64   `json:"completedAt,omitempty"`
}

// GetCreatedAt returns the creation time as a time.Time
func (m *Message) GetCreatedAt() time.Time {
	if m.Time == nil {
		return time.Time{}
	}
	return time.UnixMilli(m.Time.Created)
}

// MessagePart represents a part of a message
type MessagePart struct {
	ID        string          `json:"id"`
	MessageID string          `json:"messageID"`
	SessionID string          `json:"sessionID"`
	Type      string          `json:"type"`
	Text      *string         `json:"text,omitempty"`   // For text parts
	Tool      *string         `json:"tool,omitempty"`   // For tool parts
	CallID    *string         `json:"callID,omitempty"` // For tool parts
	State     json.RawMessage `json:"state,omitempty"`  // For tool parts
	Data      json.RawMessage `json:"data,omitempty"`   // For other parts
	Time      *PartTimeData   `json:"time,omitempty"`   // Time can be object or int64
}

// GetCreatedAt returns the creation time as a time.Time
func (p *MessagePart) GetCreatedAt() time.Time {
	if p.Time == nil {
		return time.Time{}
	}
	return time.UnixMilli(p.Time.Start)
}

// TextPartData represents text content
type TextPartData struct {
	Text string `json:"text"`
}

// ToolPartData represents tool execution (new structure)
type ToolPartData struct {
	Tool   string        `json:"tool"`
	CallID string        `json:"callID"`
	State  ToolStateData `json:"state"`
}

// ToolStateData represents the state of a tool execution
type ToolStateData struct {
	Status   string          `json:"status"`
	Input    json.RawMessage `json:"input,omitempty"`
	Output   interface{}     `json:"output,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
	Title    *string         `json:"title,omitempty"`
	Time     *ToolTimeData   `json:"time,omitempty"`
}

// ToolTimeData represents timing information for tool execution
type ToolTimeData struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

// PartTimeData represents timing information for message parts
type PartTimeData struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

// FilePartData represents file attachment
type FilePartData struct {
	Name     string  `json:"name"`
	MimeType string  `json:"mimeType"`
	Size     *int64  `json:"size,omitempty"`
	URL      *string `json:"url,omitempty"`
}

// Session represents a complete session with all its data
type Session struct {
	Info     SessionInfo   `json:"info"`
	Messages []Message     `json:"messages"`
	Parts    []MessagePart `json:"parts"`
}
