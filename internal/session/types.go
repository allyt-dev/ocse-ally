package session

import (
	"encoding/json"
	"sync"
	"time"
)

// ModelInfo represents model information
// Used in SessionInfo.Model as a JSON object
type ModelInfo struct {
	ProviderID string `json:"providerID"`
	ModelID    string `json:"modelID"`
	Variant    string `json:"variant"`
}

// SessionInfo represents session metadata stored in the DB
// TimeCreated/TimeUpdated are int64 milliseconds (Unix epoch)
// Model is stored as JSON in the DB
type SessionInfo struct {
	ID               string    `json:"id"`
	ParentID         *string   `json:"parentID,omitempty"`
	Title            string    `json:"title"`
	Version          string    `json:"version"`
	TimeCreated      int64     `json:"timeCreated"`
	TimeUpdated      int64     `json:"timeUpdated"`
	Directory        string    `json:"directory"`
	ShareURL         *string   `json:"shareURL,omitempty"`
	Agent            string    `json:"agent"`
	Model            ModelInfo `json:"model"`
	Cost             float64   `json:"cost"`
	TokensInput      int64     `json:"tokensInput"`
	TokensOutput     int64     `json:"tokensOutput"`
	TokensReasoning  int64     `json:"tokensReasoning"`
	TokensCacheRead  int64     `json:"tokensCacheRead"`
	TokensCacheWrite int64     `json:"tokensCacheWrite"`
	Slug             string    `json:"slug"`
	WorkspaceID      string    `json:"workspaceID"`
	Path             string    `json:"path"`
}

// GetCreatedAt returns the creation time as time.Time
func (s *SessionInfo) GetCreatedAt() time.Time {
	return time.UnixMilli(s.TimeCreated)
}

// GetUpdatedAt returns the update time as time.Time
func (s *SessionInfo) GetUpdatedAt() time.Time {
	return time.UnixMilli(s.TimeUpdated)
}

// TokenInfo represents token usage information
type TokenInfo struct {
	Total      int64 `json:"total"`
	Input      int64 `json:"input"`
	Output     int64 `json:"output"`
	Reasoning  int64 `json:"reasoning"`
	CacheRead  int64 `json:"cacheRead"`
	CacheWrite int64 `json:"cacheWrite"`
}

// messageData is the parsed JSON structure for Message.Data
// Accessed lazily via sync.Once
type messageData struct {
	Role          string    `json:"role"`
	Mode          string    `json:"mode"`
	Agent         string    `json:"agent"`
	Cost          float64   `json:"cost"`
	Tokens        TokenInfo `json:"tokens"`
	ModelID       string    `json:"modelID"`
	ProviderID    string    `json:"providerID"`
	Finish        string    `json:"finish"`
	TimeCreated   int64     `json:"timeCreated"`
	TimeCompleted int64     `json:"timeCompleted"`
}

// Message represents a message in the session
// The Data field contains the raw JSON blob; parsed fields are accessed lazily
type Message struct {
	ID        string          `json:"id"`
	SessionID string          `json:"sessionID"`
	CreatedAt int64           `json:"timeCreated"`
	UpdatedAt int64           `json:"timeUpdated"`
	Data      json.RawMessage `json:"data"`

	// Lazy parsed fields
	once     sync.Once
	parsed   messageData
	parseErr error
}

func (m *Message) ensureParsed() {
	m.once.Do(func() {
		if len(m.Data) > 0 {
			m.parseErr = json.Unmarshal(m.Data, &m.parsed)
		}
	})
}

// Role returns the message role from the parsed data ("user" or "assistant")
func (m *Message) Role() string {
	m.ensureParsed()
	return m.parsed.Role
}

// Mode returns the mode from the parsed data
func (m *Message) Mode() string {
	m.ensureParsed()
	return m.parsed.Mode
}

// Agent returns the agent identifier from the parsed data
func (m *Message) Agent() string {
	m.ensureParsed()
	return m.parsed.Agent
}

// Cost returns the cost from the parsed data
func (m *Message) Cost() float64 {
	m.ensureParsed()
	return m.parsed.Cost
}

// Tokens returns the token usage info from the parsed data
func (m *Message) Tokens() TokenInfo {
	m.ensureParsed()
	return m.parsed.Tokens
}

// ModelID returns the model ID from the parsed data
func (m *Message) ModelID() string {
	m.ensureParsed()
	return m.parsed.ModelID
}

// ProviderID returns the provider ID from the parsed data
func (m *Message) ProviderID() string {
	m.ensureParsed()
	return m.parsed.ProviderID
}

// Finish returns the finish reason from the parsed data
func (m *Message) Finish() string {
	m.ensureParsed()
	return m.parsed.Finish
}

// TimeCreated returns the message creation time from the parsed data
func (m *Message) TimeCreated() int64 {
	m.ensureParsed()
	return m.parsed.TimeCreated
}

// TimeCompleted returns the message completion time from the parsed data
func (m *Message) TimeCompleted() int64 {
	m.ensureParsed()
	return m.parsed.TimeCompleted
}

// GetCreatedAt returns the DB record creation time as time.Time
func (m *Message) GetCreatedAt() time.Time {
	return time.UnixMilli(m.CreatedAt)
}

// GetUpdatedAt returns the DB record update time as time.Time
func (m *Message) GetUpdatedAt() time.Time {
	return time.UnixMilli(m.UpdatedAt)
}

// messagePartData is the parsed JSON structure for MessagePart.Data
type messagePartData struct {
	Type   string          `json:"type"`
	Text   string          `json:"text"`
	Tool   string          `json:"tool"`
	CallID string          `json:"callID"`
	State  json.RawMessage `json:"state"`
}

// MessagePart represents a part of a message
// The Data field contains the raw JSON blob; parsed fields are accessed lazily
type MessagePart struct {
	ID          string          `json:"id"`
	MessageID   string          `json:"messageID"`
	SessionID   string          `json:"sessionID"`
	TimeCreated int64           `json:"timeCreated"`
	TimeUpdated int64           `json:"timeUpdated"`
	Data        json.RawMessage `json:"data"`

	// Lazy parsed fields
	once     sync.Once
	parsed   messagePartData
	parseErr error
}

func (p *MessagePart) ensureParsed() {
	p.once.Do(func() {
		if len(p.Data) > 0 {
			p.parseErr = json.Unmarshal(p.Data, &p.parsed)
		}
	})
}

// Type returns the part type from the parsed data ("text", "tool", "file", etc.)
func (p *MessagePart) Type() string {
	p.ensureParsed()
	return p.parsed.Type
}

// Text returns the text content from the parsed data
func (p *MessagePart) Text() string {
	p.ensureParsed()
	return p.parsed.Text
}

// Tool returns the tool name from the parsed data
func (p *MessagePart) Tool() string {
	p.ensureParsed()
	return p.parsed.Tool
}

// CallID returns the tool call ID from the parsed data
func (p *MessagePart) CallID() string {
	p.ensureParsed()
	return p.parsed.CallID
}

// State returns the raw tool state JSON from the parsed data
func (p *MessagePart) State() json.RawMessage {
	p.ensureParsed()
	return p.parsed.State
}

// GetCreatedAt returns the DB record creation time as time.Time
func (p *MessagePart) GetCreatedAt() time.Time {
	return time.UnixMilli(p.TimeCreated)
}

// GetUpdatedAt returns the DB record update time as time.Time
func (p *MessagePart) GetUpdatedAt() time.Time {
	return time.UnixMilli(p.TimeUpdated)
}

// SystemEvent represents a system event in the session
// Type is one of: "model-switched", "agent-switched"
// Data contains the raw JSON blob; parsed fields are accessed lazily
type SystemEvent struct {
	ID          string          `json:"id"`
	SessionID   string          `json:"sessionID"`
	Type        string          `json:"type"`
	Seq         int             `json:"seq"`
	Data        json.RawMessage `json:"data"`
	TimeCreated int64           `json:"timeCreated"`
	TimeUpdated int64           `json:"timeUpdated"`

	// Lazy parsed fields
	once     sync.Once
	parsed   map[string]interface{}
	parseErr error
}

func (e *SystemEvent) ensureParsed() {
	e.once.Do(func() {
		if len(e.Data) > 0 {
			e.parseErr = json.Unmarshal(e.Data, &e.parsed)
		}
	})
}

// GetCreatedAt returns the DB record creation time as time.Time
func (e *SystemEvent) GetCreatedAt() time.Time {
	return time.UnixMilli(e.TimeCreated)
}

// GetUpdatedAt returns the DB record update time as time.Time
func (e *SystemEvent) GetUpdatedAt() time.Time {
	return time.UnixMilli(e.TimeUpdated)
}

// ToolState represents the state of a tool execution
type ToolState struct {
	Status    string          `json:"status"`
	Input     json.RawMessage `json:"input,omitempty"`
	Output    json.RawMessage `json:"output,omitempty"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
	Title     string          `json:"title"`
	TimeStart int64           `json:"timeStart"`
	TimeEnd   int64           `json:"timeEnd"`
}

// FilePartData represents file attachment data
// Fields changed from the old Name/MimeType/Size/URL structure
type FilePartData struct {
	URL      string `json:"url"`
	Filename string `json:"filename"`
	Mime     string `json:"mime"`
}

// Session represents a complete session with all its data
type Session struct {
	Info         SessionInfo   `json:"info"`
	Messages     []Message     `json:"messages"`
	Parts        []MessagePart `json:"parts"`
	SystemEvents []SystemEvent `json:"systemEvents"`
}
