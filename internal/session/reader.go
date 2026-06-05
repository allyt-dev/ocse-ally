package session

import (
	"encoding/json"

	"github.com/byteowlz/ocse/internal/config"
	"github.com/byteowlz/ocse/internal/db"
)

// SessionWithProject represents a session with its associated project information.
type SessionWithProject struct {
	SessionID   string
	ProjectName string
	Info        SessionInfo
}

// Reader wraps db.DB with an optional project scope.
// If projectPath was provided at construction, reads are scoped to that project.
type Reader struct {
	db          *db.DB
	projectPath string
}

// NewReader opens opencode.db via config.GetDBPath() and scopes reads
// to the project containing projectPath.
func NewReader(projectPath string) (*Reader, error) {
	dbPath, err := config.GetDBPath()
	if err != nil {
		return nil, err
	}

	d, err := db.Open(dbPath)
	if err != nil {
		return nil, err
	}

	if projectPath != "" {
		_, _, err := d.FindProjectByPath(projectPath)
		if err != nil {
			d.Close()
			return nil, err
		}
	}

	return &Reader{db: d, projectPath: projectPath}, nil
}

// NewGlobalReader opens opencode.db without a project filter.
func NewGlobalReader() (*Reader, error) {
	dbPath, err := config.GetDBPath()
	if err != nil {
		return nil, err
	}

	d, err := db.Open(dbPath)
	if err != nil {
		return nil, err
	}

	return &Reader{db: d}, nil
}

// Close closes the underlying database connection.
func (r *Reader) Close() error {
	if r == nil || r.db == nil {
		return nil
	}
	return r.db.Close()
}

// ListSessions returns session IDs for the scoped project.
func (r *Reader) ListSessions() ([]string, error) {
	if r.projectPath == "" {
		return nil, nil // allow empty projectPath in global reader context
	}
	return r.db.ListSessions(r.projectPath)
}

// ListAllSessions returns every session with project metadata.
func (r *Reader) ListAllSessions() ([]SessionWithProject, error) {
	rows, err := r.db.ListAllSessionsRows()
	if err != nil {
		return nil, err
	}

	sessions := make([]SessionWithProject, len(rows))
	for i, row := range rows {
		sessions[i] = SessionWithProject{
			SessionID:   row.ID,
			ProjectName: row.ProjectName,
			Info:        sessionRowToInfo(row),
		}
	}
	return sessions, nil
}

// ReadSessionInfo reads session metadata.
func (r *Reader) ReadSessionInfo(sessionID string) (*SessionInfo, error) {
	row, err := r.db.SessionRow(sessionID)
	if err != nil {
		return nil, err
	}
	info := sessionRowToInfo(*row)
	return &info, nil
}

// ReadMessages reads all messages for a session, ordered by creation time.
func (r *Reader) ReadMessages(sessionID string) ([]Message, error) {
	rows, err := r.db.ReadMessagesRows(sessionID)
	if err != nil {
		return nil, err
	}

	messages := make([]Message, len(rows))
	for i, row := range rows {
		messages[i] = Message{
			ID:        row.ID,
			SessionID: row.SessionID,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
			Data:      json.RawMessage(row.Data),
		}
	}
	return messages, nil
}

// ReadMessageParts reads all parts for a session, ordered by creation time.
// The messageID parameter is accepted for API compatibility but ignored
// (the DB reader returns all session parts in a single query).
func (r *Reader) ReadMessageParts(sessionID, messageID string) ([]MessagePart, error) {
	rows, err := r.db.ReadPartsRows(sessionID)
	if err != nil {
		return nil, err
	}

	parts := make([]MessagePart, len(rows))
	for i, row := range rows {
		parts[i] = MessagePart{
			ID:          row.ID,
			MessageID:   row.MessageID,
			SessionID:   row.SessionID,
			TimeCreated: row.CreatedAt,
			TimeUpdated: row.UpdatedAt,
			Data:        json.RawMessage(row.Data),
		}
	}
	return parts, nil
}

// ReadSession reads a complete session with all its data.
func (r *Reader) ReadSession(sessionID string) (*Session, error) {
	data, err := r.db.ReadSessionData(sessionID)
	if err != nil {
		return nil, err
	}

	messages := make([]Message, len(data.Msgs))
	for i, row := range data.Msgs {
		messages[i] = Message{
			ID:        row.ID,
			SessionID: row.SessionID,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
			Data:      json.RawMessage(row.Data),
		}
	}

	parts := make([]MessagePart, len(data.Parts))
	for i, row := range data.Parts {
		parts[i] = MessagePart{
			ID:          row.ID,
			MessageID:   row.MessageID,
			SessionID:   row.SessionID,
			TimeCreated: row.CreatedAt,
			TimeUpdated: row.UpdatedAt,
			Data:        json.RawMessage(row.Data),
		}
	}

	events := make([]SystemEvent, len(data.Events))
	for i, row := range data.Events {
		events[i] = SystemEvent{
			ID:          row.ID,
			SessionID:   row.SessionID,
			Type:        row.Type,
			Seq:         row.Seq,
			Data:        json.RawMessage(row.Data),
			TimeCreated: row.CreatedAt,
			TimeUpdated: row.UpdatedAt,
		}
	}

	info := sessionRowToInfo(*data.Info)
	return &Session{
		Info:         info,
		Messages:     messages,
		Parts:        parts,
		SystemEvents: events,
	}, nil
}

// sessionRowToInfo converts a db.SessionRow to a SessionInfo.
func sessionRowToInfo(row db.SessionRow) SessionInfo {
	return SessionInfo{
		ID:               row.ID,
		ParentID:         strPtr(row.ParentID),
		Title:            row.Title,
		Version:          row.Version,
		TimeCreated:      row.TimeCreated,
		TimeUpdated:      row.TimeUpdated,
		Directory:        row.Directory,
		ShareURL:         row.ShareURL,
		Agent:            row.Agent,
		Model:            parseModelJSON(row.ModelJSON),
		Cost:             row.Cost,
		TokensInput:      row.TokensInput,
		TokensOutput:     row.TokensOutput,
		TokensReasoning:  row.TokensReasoning,
		TokensCacheRead:  row.TokensCacheRead,
		TokensCacheWrite: row.TokensCacheWrite,
		Slug:             row.Slug,
		WorkspaceID:      row.WorkspaceID,
		Path:             row.Path,
	}
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func parseModelJSON(raw string) ModelInfo {
	var m ModelInfo
	if raw == "" || raw == "{}" {
		return m
	}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return ModelInfo{}
	}
	return m
}
