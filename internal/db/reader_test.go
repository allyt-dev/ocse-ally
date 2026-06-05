package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func createTestDB(t *testing.T) string {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")

	conn, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	defer conn.Close()

	schema := `
CREATE TABLE project (
	id TEXT PRIMARY KEY,
	worktree TEXT NOT NULL,
	vcs TEXT,
	name TEXT,
	time_created INTEGER NOT NULL,
	time_updated INTEGER NOT NULL
);

CREATE TABLE project_directory (
	project_id TEXT NOT NULL,
	directory TEXT NOT NULL,
	type TEXT NOT NULL,
	time_created INTEGER NOT NULL,
	PRIMARY KEY(project_id, directory)
);

CREATE TABLE session (
	id TEXT PRIMARY KEY,
	project_id TEXT NOT NULL,
	parent_id TEXT,
	slug TEXT NOT NULL,
	directory TEXT NOT NULL,
	title TEXT NOT NULL,
	version TEXT NOT NULL,
	share_url TEXT,
	time_created INTEGER NOT NULL,
	time_updated INTEGER NOT NULL,
	agent TEXT,
	model TEXT,
	cost REAL DEFAULT 0 NOT NULL,
	tokens_input INTEGER DEFAULT 0 NOT NULL,
	tokens_output INTEGER DEFAULT 0 NOT NULL,
	tokens_reasoning INTEGER DEFAULT 0 NOT NULL,
	tokens_cache_read INTEGER DEFAULT 0 NOT NULL,
	tokens_cache_write INTEGER DEFAULT 0 NOT NULL,
	workspace_id TEXT,
	path TEXT
);

CREATE INDEX session_project_idx ON session (project_id);

CREATE TABLE message (
	id TEXT PRIMARY KEY,
	session_id TEXT NOT NULL,
	time_created INTEGER NOT NULL,
	time_updated INTEGER NOT NULL,
	data TEXT NOT NULL
);

CREATE INDEX message_session_time_created_id_idx ON message (session_id, time_created, id);

CREATE TABLE part (
	id TEXT PRIMARY KEY,
	message_id TEXT NOT NULL,
	session_id TEXT NOT NULL,
	time_created INTEGER NOT NULL,
	time_updated INTEGER NOT NULL,
	data TEXT NOT NULL
);

CREATE INDEX part_session_idx ON part (session_id);

CREATE TABLE session_message (
	id TEXT PRIMARY KEY,
	session_id TEXT NOT NULL,
	type TEXT NOT NULL,
	time_created INTEGER NOT NULL,
	time_updated INTEGER NOT NULL,
	data TEXT NOT NULL,
	seq INTEGER NOT NULL
);
`
	if _, err := conn.Exec(schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	// Insert test data
	inserts := `
INSERT INTO project (id, worktree, name, time_created, time_updated) VALUES ('proj1', '/test/worktree1', 'Project One', 1, 2);
INSERT INTO project (id, worktree, name, time_created, time_updated) VALUES ('proj2', '', 'Project Two', 1, 2);
INSERT INTO project_directory (project_id, directory, type, time_created) VALUES ('proj2', '/test/project2', 'root', 1);
INSERT INTO project_directory (project_id, directory, type, time_created) VALUES ('proj1', '/test/worktree1', 'root', 1);

INSERT INTO session (id, project_id, parent_id, slug, directory, title, version, time_created, time_updated, agent, model, cost, tokens_input, tokens_output, tokens_reasoning, tokens_cache_read, tokens_cache_write, workspace_id, path) VALUES
	('sess1', 'proj1', 'parent1', 'test-sess', '/test/dir', 'Test Session', '1.0', 1000, 2000, 'build-medium', '{"providerID": "openai", "modelID": "gpt-5.5", "variant": "turbo"}', 0.0023, 100, 50, 10, 5, 2, 'ws1', '/test/dir');

INSERT INTO session (id, project_id, parent_id, slug, directory, title, version, time_created, time_updated, agent, model, cost, tokens_input, tokens_output, tokens_reasoning, tokens_cache_read, tokens_cache_write, workspace_id, path) VALUES
	('sess2', 'proj1', NULL, 'sess-2', '/test/dir', 'Second Session', '1.0', 3000, 4000, NULL, '{}', 0, 0, 0, 0, 0, 0, NULL, NULL);

INSERT INTO session (id, project_id, parent_id, slug, directory, title, version, time_created, time_updated, agent, model, cost, tokens_input, tokens_output, tokens_reasoning, tokens_cache_read, tokens_cache_write, workspace_id, path) VALUES
	('sess3', 'proj2', NULL, 'sess-3', '/test/dir', 'Project 2 Session', '1.0', 5000, 6000, NULL, '{}', 0, 0, 0, 0, 0, 0, NULL, NULL);

INSERT INTO message (id, session_id, time_created, time_updated, data) VALUES
	('msg1', 'sess1', 1000, 1000, '{"role": "user", "mode": "chat", "agent": "", "cost": 0, "tokens": {"total": 0, "input": 0, "output": 0, "reasoning": 0, "cacheRead": 0, "cacheWrite": 0}, "modelID": "", "providerID": "", "finish": "", "timeCreated": 1000, "timeCompleted": 0}');

INSERT INTO message (id, session_id, time_created, time_updated, data) VALUES
	('msg2', 'sess1', 2000, 2000, '{"role": "assistant", "mode": "build", "agent": "build-medium", "cost": 0.0023, "tokens": {"total": 150, "input": 100, "output": 50, "reasoning": 10, "cacheRead": 5, "cacheWrite": 2}, "modelID": "gpt-5.5", "providerID": "openai", "finish": "stop", "timeCreated": 2000, "timeCompleted": 2500}');

INSERT INTO part (id, message_id, session_id, time_created, time_updated, data) VALUES
	('part1', 'msg1', 'sess1', 1000, 1000, '{"type": "text", "text": "Hello", "tool": "", "callID": "", "state": null}');

INSERT INTO part (id, message_id, session_id, time_created, time_updated, data) VALUES
	('part2', 'msg2', 'sess1', 2000, 2000, '{"type": "text", "text": "Hello world", "tool": "", "callID": "", "state": null}');

INSERT INTO part (id, message_id, session_id, time_created, time_updated, data) VALUES
	('part3', 'msg2', 'sess1', 2100, 2100, '{"type": "tool", "text": "", "tool": "bash", "callID": "call123", "state": {"status": "completed", "input": {"command": "ls"}, "output": "file1", "metadata": null, "title": "List files", "timeStart": 2100, "timeEnd": 2200}}');

INSERT INTO session_message (id, session_id, type, time_created, time_updated, data, seq) VALUES
	('evt1', 'sess1', 'model-switched', 500, 500, '{"modelID": "gpt-4", "providerID": "openai"}', 0);

INSERT INTO session_message (id, session_id, type, time_created, time_updated, data, seq) VALUES
	('evt2', 'sess1', 'agent-switched', 600, 600, '{"agent": "build-medium"}', 1);
`
	if _, err := conn.Exec(inserts); err != nil {
		t.Fatalf("insert test data: %v", err)
	}

	return dbPath
}

func TestOpen(t *testing.T) {
	dbPath := createTestDB(t)
	d, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	if d == nil || d.db == nil {
		t.Fatal("expected non-nil DB")
	}
}

func TestFindProjectByPath_Worktree(t *testing.T) {
	dbPath := createTestDB(t)
	d, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	id, name, err := d.FindProjectByPath("/test/worktree1")
	if err != nil {
		t.Fatalf("FindProjectByPath: %v", err)
	}
	if id != "proj1" {
		t.Errorf("id = %q, want proj1", id)
	}
	if name != "Project One" {
		t.Errorf("name = %q, want Project One", name)
	}
}

func TestFindProjectByPath_DirectoryFallback(t *testing.T) {
	dbPath := createTestDB(t)
	d, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	id, name, err := d.FindProjectByPath("/test/project2")
	if err != nil {
		t.Fatalf("FindProjectByPath: %v", err)
	}
	if id != "proj2" {
		t.Errorf("id = %q, want proj2", id)
	}
	if name != "Project Two" {
		t.Errorf("name = %q, want Project Two", name)
	}
}

func TestFindProjectByPath_NotFound(t *testing.T) {
	dbPath := createTestDB(t)
	d, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	_, _, err = d.FindProjectByPath("/nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
}

func TestListSessions(t *testing.T) {
	dbPath := createTestDB(t)
	d, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	// Using worktree path for proj1
	ids, err := d.ListSessions("/test/worktree1")
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}

	if len(ids) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(ids))
	}
	// Ordered by time_updated DESC
	if ids[0] != "sess2" {
		t.Errorf("first session = %q, want sess2", ids[0])
	}
	if ids[1] != "sess1" {
		t.Errorf("second session = %q, want sess1", ids[1])
	}
}

func TestListSessions_EmptyProject(t *testing.T) {
	dbPath := createTestDB(t)
	d, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	ids, err := d.ListSessions("/test/project2")
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}

	if len(ids) != 1 {
		t.Fatalf("expected 1 session, got %d", len(ids))
	}
	if ids[0] != "sess3" {
		t.Errorf("session = %q, want sess3", ids[0])
	}
}

func TestListAllSessionsRows(t *testing.T) {
	dbPath := createTestDB(t)
	d, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	rows, err := d.ListAllSessionsRows()
	if err != nil {
		t.Fatalf("ListAllSessionsRows: %v", err)
	}

	if len(rows) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(rows))
	}

	// Ordered by time_updated DESC: sess3 (6000), sess2 (4000), sess1 (2000)
	if rows[0].ID != "sess3" {
		t.Errorf("first = %q, want sess3", rows[0].ID)
	}
	if rows[0].ProjectName != "Project Two" {
		t.Errorf("first project = %q, want Project Two", rows[0].ProjectName)
	}
	if rows[1].ID != "sess2" {
		t.Errorf("second = %q, want sess2", rows[1].ID)
	}
	if rows[2].ID != "sess1" {
		t.Errorf("third = %q, want sess1", rows[2].ID)
	}

	// Check sess1 details
	sess1 := rows[2]
	if sess1.Title != "Test Session" {
		t.Errorf("Title = %q", sess1.Title)
	}
	if sess1.Agent != "build-medium" {
		t.Errorf("Agent = %q", sess1.Agent)
	}
	if sess1.ModelJSON != `{"providerID": "openai", "modelID": "gpt-5.5", "variant": "turbo"}` {
		t.Errorf("ModelJSON = %s", sess1.ModelJSON)
	}
	if sess1.Cost != 0.0023 {
		t.Errorf("Cost = %v", sess1.Cost)
	}
	if sess1.TokensInput != 100 {
		t.Errorf("TokensInput = %d", sess1.TokensInput)
	}
	if sess1.TokensOutput != 50 {
		t.Errorf("TokensOutput = %d", sess1.TokensOutput)
	}
	if sess1.TokensReasoning != 10 {
		t.Errorf("TokensReasoning = %d", sess1.TokensReasoning)
	}
	if sess1.TokensCacheRead != 5 {
		t.Errorf("TokensCacheRead = %d", sess1.TokensCacheRead)
	}
	if sess1.TokensCacheWrite != 2 {
		t.Errorf("TokensCacheWrite = %d", sess1.TokensCacheWrite)
	}
	if sess1.WorkspaceID != "ws1" {
		t.Errorf("WorkspaceID = %q", sess1.WorkspaceID)
	}
	if sess1.Path != "/test/dir" {
		t.Errorf("Path = %q", sess1.Path)
	}
	if sess1.ParentID != "parent1" {
		t.Errorf("ParentID = %q", sess1.ParentID)
	}
	if sess1.Slug != "test-sess" {
		t.Errorf("Slug = %q", sess1.Slug)
	}
}

func TestSessionRow(t *testing.T) {
	dbPath := createTestDB(t)
	d, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	row, err := d.SessionRow("sess1")
	if err != nil {
		t.Fatalf("SessionRow: %v", err)
	}

	if row.ID != "sess1" {
		t.Errorf("ID = %q", row.ID)
	}
	if row.Title != "Test Session" {
		t.Errorf("Title = %q", row.Title)
	}
	if row.Version != "1.0" {
		t.Errorf("Version = %q", row.Version)
	}
	if row.TimeCreated != 1000 {
		t.Errorf("TimeCreated = %d", row.TimeCreated)
	}
	if row.TimeUpdated != 2000 {
		t.Errorf("TimeUpdated = %d", row.TimeUpdated)
	}
	if row.Directory != "/test/dir" {
		t.Errorf("Directory = %q", row.Directory)
	}
	if row.Agent != "build-medium" {
		t.Errorf("Agent = %q", row.Agent)
	}
	if row.Cost != 0.0023 {
		t.Errorf("Cost = %v", row.Cost)
	}
	if row.TokensInput != 100 {
		t.Errorf("TokensInput = %d", row.TokensInput)
	}
	if row.TokensOutput != 50 {
		t.Errorf("TokensOutput = %d", row.TokensOutput)
	}
	if row.TokensReasoning != 10 {
		t.Errorf("TokensReasoning = %d", row.TokensReasoning)
	}
	if row.TokensCacheRead != 5 {
		t.Errorf("TokensCacheRead = %d", row.TokensCacheRead)
	}
	if row.TokensCacheWrite != 2 {
		t.Errorf("TokensCacheWrite = %d", row.TokensCacheWrite)
	}
	if row.Slug != "test-sess" {
		t.Errorf("Slug = %q", row.Slug)
	}
	if row.WorkspaceID != "ws1" {
		t.Errorf("WorkspaceID = %q", row.WorkspaceID)
	}
	if row.Path != "/test/dir" {
		t.Errorf("Path = %q", row.Path)
	}
	if row.ParentID != "parent1" {
		t.Errorf("ParentID = %q", row.ParentID)
	}
}

func TestSessionRow_NotFound(t *testing.T) {
	dbPath := createTestDB(t)
	d, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	_, err = d.SessionRow("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}
}

func TestReadMessagesRows(t *testing.T) {
	dbPath := createTestDB(t)
	d, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	msgs, err := d.ReadMessagesRows("sess1")
	if err != nil {
		t.Fatalf("ReadMessagesRows: %v", err)
	}

	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}

	if msgs[0].ID != "msg1" {
		t.Errorf("first msg ID = %q", msgs[0].ID)
	}
	if msgs[0].SessionID != "sess1" {
		t.Errorf("first msg SessionID = %q", msgs[0].SessionID)
	}
	if msgs[0].CreatedAt != 1000 {
		t.Errorf("first msg CreatedAt = %d", msgs[0].CreatedAt)
	}
	if msgs[1].ID != "msg2" {
		t.Errorf("second msg ID = %q", msgs[1].ID)
	}
	if msgs[1].CreatedAt != 2000 {
		t.Errorf("second msg CreatedAt = %d", msgs[1].CreatedAt)
	}

	// Check data JSON
	if msgs[1].Data == "" {
		t.Error("msg2 data is empty")
	}
	if msgs[0].Data == "" {
		t.Error("msg1 data is empty")
	}
}

func TestReadMessagesRows_EmptySession(t *testing.T) {
	dbPath := createTestDB(t)
	d, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	msgs, err := d.ReadMessagesRows("sess2")
	if err != nil {
		t.Fatalf("ReadMessagesRows: %v", err)
	}

	if len(msgs) != 0 {
		t.Errorf("expected 0 messages, got %d", len(msgs))
	}
}

func TestReadPartsRows(t *testing.T) {
	dbPath := createTestDB(t)
	d, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	parts, err := d.ReadPartsRows("sess1")
	if err != nil {
		t.Fatalf("ReadPartsRows: %v", err)
	}

	if len(parts) != 3 {
		t.Fatalf("expected 3 parts, got %d", len(parts))
	}

	if parts[0].ID != "part1" {
		t.Errorf("first part ID = %q", parts[0].ID)
	}
	if parts[0].MessageID != "msg1" {
		t.Errorf("first part MessageID = %q", parts[0].MessageID)
	}
	if parts[0].SessionID != "sess1" {
		t.Errorf("first part SessionID = %q", parts[0].SessionID)
	}
	if parts[0].CreatedAt != 1000 {
		t.Errorf("first part CreatedAt = %d", parts[0].CreatedAt)
	}
	if parts[1].ID != "part2" {
		t.Errorf("second part ID = %q", parts[1].ID)
	}
	if parts[2].ID != "part3" {
		t.Errorf("third part ID = %q", parts[2].ID)
	}
	if parts[2].MessageID != "msg2" {
		t.Errorf("third part MessageID = %q", parts[2].MessageID)
	}
}

func TestReadSystemEventRows(t *testing.T) {
	dbPath := createTestDB(t)
	d, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	events, err := d.ReadSystemEventRows("sess1")
	if err != nil {
		t.Fatalf("ReadSystemEventRows: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	if events[0].ID != "evt1" {
		t.Errorf("first event ID = %q", events[0].ID)
	}
	if events[0].Type != "model-switched" {
		t.Errorf("first event Type = %q", events[0].Type)
	}
	if events[0].Seq != 0 {
		t.Errorf("first event Seq = %d", events[0].Seq)
	}
	if events[0].SessionID != "sess1" {
		t.Errorf("first event SessionID = %q", events[0].SessionID)
	}
	if events[0].CreatedAt != 500 {
		t.Errorf("first event CreatedAt = %d", events[0].CreatedAt)
	}
	if events[1].ID != "evt2" {
		t.Errorf("second event ID = %q", events[1].ID)
	}
	if events[1].Type != "agent-switched" {
		t.Errorf("second event Type = %q", events[1].Type)
	}
	if events[1].Seq != 1 {
		t.Errorf("second event Seq = %d", events[1].Seq)
	}

	if events[0].Data == "" {
		t.Error("first event data is empty")
	}
	if events[1].Data == "" {
		t.Error("second event data is empty")
	}
}

func TestReadSessionData(t *testing.T) {
	dbPath := createTestDB(t)
	d, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	data, err := d.ReadSessionData("sess1")
	if err != nil {
		t.Fatalf("ReadSessionData: %v", err)
	}

	if data.Info == nil {
		t.Fatal("Info is nil")
	}
	if data.Info.ID != "sess1" {
		t.Errorf("Info.ID = %q", data.Info.ID)
	}
	if len(data.Msgs) != 2 {
		t.Errorf("expected 2 messages, got %d", len(data.Msgs))
	}
	if len(data.Parts) != 3 {
		t.Errorf("expected 3 parts, got %d", len(data.Parts))
	}
	if len(data.Events) != 2 {
		t.Errorf("expected 2 events, got %d", len(data.Events))
	}
}

func TestClose(t *testing.T) {
	dbPath := createTestDB(t)
	d, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	if err := d.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Double close should not error
	if err := d.Close(); err != nil {
		t.Fatalf("double Close: %v", err)
	}
}

func TestOpen_InvalidFile(t *testing.T) {
	_, err := Open("/nonexistent/file.db")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestOpen_NonDBFile(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "notadb.txt")
	if err := os.WriteFile(tmp, []byte("not a database"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Open(tmp)
	if err == nil {
		t.Fatal("expected error for non-db file")
	}
}
