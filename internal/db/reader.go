package db

import (
	"database/sql"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// DB reads opencode session data from a SQLite database.
type DB struct {
	db *sql.DB
}

// Open opens a read-only connection to opencode.db.
func Open(path string) (*DB, error) {
	dsn := fmt.Sprintf("file:%s?mode=ro", filepath.ToSlash(path))
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &DB{db: db}, nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	if d == nil || d.db == nil {
		return nil
	}
	return d.db.Close()
}

// FindProjectByPath resolves a project path to a project ID and name.
func (d *DB) FindProjectByPath(path string) (id, name string, err error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", "", fmt.Errorf("resolve absolute path: %w", err)
	}

	if worktree, gitErr := gitRoot(absPath); gitErr == nil {
		id, name, err = d.findProjectByWorktree(worktree)
		if err == nil {
			return id, name, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return "", "", err
		}
	}

	id, name, err = d.findProjectByDirectory(absPath)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", fmt.Errorf("project not found for path %q", absPath)
		}
		return "", "", err
	}
	return id, name, nil
}

// ListSessions returns session IDs for the project containing projectPath.
func (d *DB) ListSessions(projectPath string) ([]string, error) {
	projectID, _, err := d.FindProjectByPath(projectPath)
	if err != nil {
		return nil, err
	}

	stmt, err := d.db.Prepare(`
		SELECT id
		FROM session
		WHERE project_id = ?
		ORDER BY time_updated DESC, id
	`)
	if err != nil {
		return nil, fmt.Errorf("prepare list sessions: %w", err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(projectID)
	if err != nil {
		return nil, fmt.Errorf("query sessions: %w", err)
	}
	defer rows.Close()

	var sessionIDs []string
	for rows.Next() {
		var sessionID string
		if err := rows.Scan(&sessionID); err != nil {
			return nil, fmt.Errorf("scan session id: %w", err)
		}
		sessionIDs = append(sessionIDs, sessionID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sessions: %w", err)
	}

	return sessionIDs, nil
}

// SessionRow holds raw session fields from the DB for multi-row queries.
// ProjectName is appended from the JOINed project.name column.
type SessionRow struct {
	ID, ParentID, Title, Version, ProjectName string
	TimeCreated, TimeUpdated                  int64
	Directory                                 string
	ShareURL                                  *string
	Agent                                     string
	ModelJSON                                 string
	Cost                                      float64
	TokensInput, TokensOutput                 int64
	TokensReasoning, TokensCacheRead          int64
	TokensCacheWrite                          int64
	Slug, WorkspaceID, Path                   string
}

// ListAllSessionsRows returns raw session rows joined with project metadata.
func (d *DB) ListAllSessionsRows() ([]SessionRow, error) {
	stmt, err := d.db.Prepare(`
		SELECT
			s.id,
			COALESCE(s.parent_id, ''),
			s.title,
			s.version,
			s.time_created,
			s.time_updated,
			s.directory,
			s.share_url,
			COALESCE(s.agent, ''),
			COALESCE(s.model, '{}'),
			s.cost,
			s.tokens_input,
			s.tokens_output,
			s.tokens_reasoning,
			s.tokens_cache_read,
			s.tokens_cache_write,
			s.slug,
			COALESCE(s.workspace_id, ''),
			COALESCE(s.path, ''),
			COALESCE(p.name, '')
		FROM session s
		JOIN project p ON p.id = s.project_id
		ORDER BY s.time_updated DESC, s.id
	`)
	if err != nil {
		return nil, fmt.Errorf("prepare list all sessions: %w", err)
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		return nil, fmt.Errorf("query all sessions: %w", err)
	}
	defer rows.Close()

	var sessions []SessionRow
	for rows.Next() {
		var row SessionRow
		if err := rows.Scan(
			&row.ID,
			&row.ParentID,
			&row.Title,
			&row.Version,
			&row.TimeCreated,
			&row.TimeUpdated,
			&row.Directory,
			&row.ShareURL,
			&row.Agent,
			&row.ModelJSON,
			&row.Cost,
			&row.TokensInput,
			&row.TokensOutput,
			&row.TokensReasoning,
			&row.TokensCacheRead,
			&row.TokensCacheWrite,
			&row.Slug,
			&row.WorkspaceID,
			&row.Path,
			&row.ProjectName,
		); err != nil {
			return nil, fmt.Errorf("scan session with project: %w", err)
		}
		sessions = append(sessions, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate all sessions: %w", err)
	}

	return sessions, nil
}

// MessageRow holds raw message fields from the DB.
type MessageRow struct {
	ID        string
	SessionID string
	CreatedAt int64
	UpdatedAt int64
	Data      string // raw JSON
}

// ReadMessagesRows reads raw message rows for a session, ordered by creation time.
func (d *DB) ReadMessagesRows(sessionID string) ([]MessageRow, error) {
	stmt, err := d.db.Prepare(`
		SELECT id, session_id, time_created, time_updated, data
		FROM message
		WHERE session_id = ?
		ORDER BY time_created, id
	`)
	if err != nil {
		return nil, fmt.Errorf("prepare read messages: %w", err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(sessionID)
	if err != nil {
		return nil, fmt.Errorf("query messages: %w", err)
	}
	defer rows.Close()

	var messages []MessageRow
	for rows.Next() {
		var msg MessageRow
		if err := rows.Scan(&msg.ID, &msg.SessionID, &msg.CreatedAt, &msg.UpdatedAt, &msg.Data); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		messages = append(messages, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate messages: %w", err)
	}

	return messages, nil
}

// PartRow holds raw part fields from the DB.
type PartRow struct {
	ID        string
	MessageID string
	SessionID string
	CreatedAt int64
	UpdatedAt int64
	Data      string // raw JSON
}

// ReadPartsRows reads raw part rows for a session, ordered by creation time.
func (d *DB) ReadPartsRows(sessionID string) ([]PartRow, error) {
	stmt, err := d.db.Prepare(`
		SELECT id, message_id, session_id, time_created, time_updated, data
		FROM part
		WHERE session_id = ?
		ORDER BY time_created, id
	`)
	if err != nil {
		return nil, fmt.Errorf("prepare read parts: %w", err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(sessionID)
	if err != nil {
		return nil, fmt.Errorf("query parts: %w", err)
	}
	defer rows.Close()

	var parts []PartRow
	for rows.Next() {
		var part PartRow
		if err := rows.Scan(&part.ID, &part.MessageID, &part.SessionID, &part.CreatedAt, &part.UpdatedAt, &part.Data); err != nil {
			return nil, fmt.Errorf("scan part: %w", err)
		}
		parts = append(parts, part)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate parts: %w", err)
	}

	return parts, nil
}

// SystemEventRow holds raw system event fields from the DB.
type SystemEventRow struct {
	ID        string
	SessionID string
	Type      string
	Seq       int
	Data      string // raw JSON
	CreatedAt int64
	UpdatedAt int64
}

// ReadSystemEventRows reads raw system event rows for a session, ordered by creation time.
func (d *DB) ReadSystemEventRows(sessionID string) ([]SystemEventRow, error) {
	stmt, err := d.db.Prepare(`
		SELECT id, session_id, type, seq, data, time_created, time_updated
		FROM session_message
		WHERE session_id = ?
		ORDER BY time_created, id
	`)
	if err != nil {
		return nil, fmt.Errorf("prepare read system events: %w", err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(sessionID)
	if err != nil {
		return nil, fmt.Errorf("query system events: %w", err)
	}
	defer rows.Close()

	var events []SystemEventRow
	for rows.Next() {
		var ev SystemEventRow
		if err := rows.Scan(&ev.ID, &ev.SessionID, &ev.Type, &ev.Seq, &ev.Data, &ev.CreatedAt, &ev.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan system event: %w", err)
		}
		events = append(events, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate system events: %w", err)
	}

	return events, nil
}

// SessionRow returns a single session row by ID.
func (d *DB) SessionRow(sessionID string) (*SessionRow, error) {
	stmt, err := d.db.Prepare(`
		SELECT
			id,
			COALESCE(parent_id, ''),
			title,
			version,
			time_created,
			time_updated,
			directory,
			share_url,
			COALESCE(agent, ''),
			COALESCE(model, '{}'),
			cost,
			tokens_input,
			tokens_output,
			tokens_reasoning,
			tokens_cache_read,
			tokens_cache_write,
			slug,
			COALESCE(workspace_id, ''),
			COALESCE(path, ''),
			''
		FROM session
		WHERE id = ?
	`)
	if err != nil {
		return nil, fmt.Errorf("prepare read session info: %w", err)
	}
	defer stmt.Close()

	var row SessionRow
	if err := stmt.QueryRow(sessionID).Scan(
		&row.ID,
		&row.ParentID,
		&row.Title,
		&row.Version,
		&row.TimeCreated,
		&row.TimeUpdated,
		&row.Directory,
		&row.ShareURL,
		&row.Agent,
		&row.ModelJSON,
		&row.Cost,
		&row.TokensInput,
		&row.TokensOutput,
		&row.TokensReasoning,
		&row.TokensCacheRead,
		&row.TokensCacheWrite,
		&row.Slug,
		&row.WorkspaceID,
		&row.Path,
		&row.ProjectName,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("session %q not found", sessionID)
		}
		return nil, fmt.Errorf("scan session info: %w", err)
	}

	return &row, nil
}

// SessionData holds all raw DB rows for a session.
type SessionData struct {
	Info   *SessionRow
	Msgs   []MessageRow
	Parts  []PartRow
	Events []SystemEventRow
}

// ReadSessionData reads all session data from the DB.
func (d *DB) ReadSessionData(sessionID string) (*SessionData, error) {
	row, err := d.SessionRow(sessionID)
	if err != nil {
		return nil, err
	}

	msgs, err := d.ReadMessagesRows(sessionID)
	if err != nil {
		return nil, err
	}

	parts, err := d.ReadPartsRows(sessionID)
	if err != nil {
		return nil, err
	}

	events, err := d.ReadSystemEventRows(sessionID)
	if err != nil {
		return nil, err
	}

	return &SessionData{
		Info:   row,
		Msgs:   msgs,
		Parts:  parts,
		Events: events,
	}, nil
}

func (d *DB) findProjectByWorktree(worktree string) (id, name string, err error) {
	stmt, err := d.db.Prepare(`
		SELECT id, COALESCE(name, '')
		FROM project
		WHERE worktree = ?
	`)
	if err != nil {
		return "", "", fmt.Errorf("prepare find project by worktree: %w", err)
	}
	defer stmt.Close()

	if err := stmt.QueryRow(worktree).Scan(&id, &name); err != nil {
		return "", "", err
	}
	return id, name, nil
}

func (d *DB) findProjectByDirectory(directory string) (id, name string, err error) {
	stmt, err := d.db.Prepare(`
		SELECT p.id, COALESCE(p.name, '')
		FROM project_directory pd
		JOIN project p ON p.id = pd.project_id
		WHERE pd.directory = ?
	`)
	if err != nil {
		return "", "", fmt.Errorf("prepare find project by directory: %w", err)
	}
	defer stmt.Close()

	if err := stmt.QueryRow(directory).Scan(&id, &name); err != nil {
		return "", "", err
	}
	return id, name, nil
}

func gitRoot(path string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
