package session

import (
	"encoding/json"
	"testing"
	"time"
)

func TestMessageUnmarshal(t *testing.T) {
	msgJSON := `{
		"id": "msg1",
		"sessionID": "sess1",
		"timeCreated": 1700000000000,
		"timeUpdated": 1700000001000,
		"data": {
			"role": "assistant",
			"mode": "build",
			"agent": "build-medium",
			"cost": 0.0023,
			"tokens": {
				"total": 150,
				"input": 100,
				"output": 50,
				"reasoning": 10,
				"cacheRead": 5,
				"cacheWrite": 2
			},
			"modelID": "gpt-5.5",
			"providerID": "openai",
			"finish": "stop",
			"timeCreated": 1700000000000,
			"timeCompleted": 1700000000500
		}
	}`

	var msg Message
	if err := json.Unmarshal([]byte(msgJSON), &msg); err != nil {
		t.Fatalf("unmarshal message: %v", err)
	}

	if msg.ID != "msg1" {
		t.Errorf("ID = %q, want %q", msg.ID, "msg1")
	}
	if msg.SessionID != "sess1" {
		t.Errorf("SessionID = %q, want %q", msg.SessionID, "sess1")
	}
	if msg.CreatedAt != 1700000000000 {
		t.Errorf("CreatedAt = %d, want %d", msg.CreatedAt, 1700000000000)
	}
	if msg.UpdatedAt != 1700000001000 {
		t.Errorf("UpdatedAt = %d, want %d", msg.UpdatedAt, 1700000001000)
	}

	if got := msg.Role(); got != "assistant" {
		t.Errorf("Role() = %q, want %q", got, "assistant")
	}
	if got := msg.Mode(); got != "build" {
		t.Errorf("Mode() = %q, want %q", got, "build")
	}
	if got := msg.Agent(); got != "build-medium" {
		t.Errorf("Agent() = %q, want %q", got, "build-medium")
	}
	if got := msg.Cost(); got != 0.0023 {
		t.Errorf("Cost() = %v, want %v", got, 0.0023)
	}
	if got := msg.ModelID(); got != "gpt-5.5" {
		t.Errorf("ModelID() = %q, want %q", got, "gpt-5.5")
	}
	if got := msg.ProviderID(); got != "openai" {
		t.Errorf("ProviderID() = %q, want %q", got, "openai")
	}
	if got := msg.Finish(); got != "stop" {
		t.Errorf("Finish() = %q, want %q", got, "stop")
	}
	if got := msg.TimeCreated(); got != 1700000000000 {
		t.Errorf("TimeCreated() = %d, want %d", got, 1700000000000)
	}
	if got := msg.TimeCompleted(); got != 1700000000500 {
		t.Errorf("TimeCompleted() = %d, want %d", got, 1700000000500)
	}

	tokens := msg.Tokens()
	wantTokens := TokenInfo{Total: 150, Input: 100, Output: 50, Reasoning: 10, CacheRead: 5, CacheWrite: 2}
	if tokens != wantTokens {
		t.Errorf("Tokens() = %+v, want %+v", tokens, wantTokens)
	}

	wantCreated := time.UnixMilli(1700000000000)
	if got := msg.GetCreatedAt(); !got.Equal(wantCreated) {
		t.Errorf("GetCreatedAt() = %v, want %v", got, wantCreated)
	}
}

func TestMessageUnmarshalEmptyData(t *testing.T) {
	msgJSON := `{"id": "msg1", "sessionID": "sess1", "timeCreated": 0, "timeUpdated": 0, "data": ""}`
	var msg Message
	if err := json.Unmarshal([]byte(msgJSON), &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if msg.Role() != "" {
		t.Errorf("Role() = %q, want empty", msg.Role())
	}
	if msg.Cost() != 0 {
		t.Errorf("Cost() = %v, want 0", msg.Cost())
	}
}

func TestMessagePartUnmarshal(t *testing.T) {
	partJSON := `{
		"id": "part1",
		"messageID": "msg1",
		"sessionID": "sess1",
		"timeCreated": 1700000000000,
		"timeUpdated": 1700000001000,
		"data": {
			"type": "text",
			"text": "Hello world"
		}
	}`

	var part MessagePart
	if err := json.Unmarshal([]byte(partJSON), &part); err != nil {
		t.Fatalf("unmarshal part: %v", err)
	}

	if part.ID != "part1" {
		t.Errorf("ID = %q, want %q", part.ID, "part1")
	}
	if part.Type() != "text" {
		t.Errorf("Type() = %q, want %q", part.Type(), "text")
	}
	if part.Text() != "Hello world" {
		t.Errorf("Text() = %q, want %q", part.Text(), "Hello world")
	}
	if part.Tool() != "" {
		t.Errorf("Tool() = %q, want empty", part.Tool())
	}
	if part.CallID() != "" {
		t.Errorf("CallID() = %q, want empty", part.CallID())
	}
	if len(part.State()) != 0 {
		t.Errorf("State() = %v, want empty", part.State())
	}

	wantTime := time.UnixMilli(1700000000000)
	if got := part.GetCreatedAt(); !got.Equal(wantTime) {
		t.Errorf("GetCreatedAt() = %v, want %v", got, wantTime)
	}
}

func TestMessagePartToolUnmarshal(t *testing.T) {
	partJSON := `{
		"id": "part2",
		"messageID": "msg1",
		"sessionID": "sess1",
		"timeCreated": 1700000000000,
		"timeUpdated": 1700000001000,
		"data": {
			"type": "tool",
			"tool": "bash",
			"callID": "call123",
			"state": {"status": "completed"}
		}
	}`

	var part MessagePart
	if err := json.Unmarshal([]byte(partJSON), &part); err != nil {
		t.Fatalf("unmarshal tool part: %v", err)
	}

	if part.Type() != "tool" {
		t.Errorf("Type() = %q, want %q", part.Type(), "tool")
	}
	if part.Tool() != "bash" {
		t.Errorf("Tool() = %q, want %q", part.Tool(), "bash")
	}
	if part.CallID() != "call123" {
		t.Errorf("CallID() = %q, want %q", part.CallID(), "call123")
	}
	if string(part.State()) != `{"status": "completed"}` {
		t.Errorf("State() = %s, want %s", part.State(), `{"status": "completed"}`)
	}
}

func TestSystemEventUnmarshal(t *testing.T) {
	jsonBlob := `{
		"id": "evt1",
		"sessionID": "sess1",
		"type": "model-switched",
		"seq": 1,
		"data": {"modelID": "gpt-4", "providerID": "openai"},
		"timeCreated": 1700000000000,
		"timeUpdated": 1700000001000
	}`

	var event SystemEvent
	if err := json.Unmarshal([]byte(jsonBlob), &event); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if event.ID != "evt1" {
		t.Errorf("ID = %q, want %q", event.ID, "evt1")
	}
	if event.Type != "model-switched" {
		t.Errorf("Type = %q, want %q", event.Type, "model-switched")
	}
	if event.Seq != 1 {
		t.Errorf("Seq = %d, want %d", event.Seq, 1)
	}
	if string(event.Data) != `{"modelID": "gpt-4", "providerID": "openai"}` {
		t.Errorf("Data = %s", event.Data)
	}

	wantTime := time.UnixMilli(1700000000000)
	if got := event.GetCreatedAt(); !got.Equal(wantTime) {
		t.Errorf("GetCreatedAt() = %v, want %v", got, wantTime)
	}
}

func TestSessionInfoUnmarshal(t *testing.T) {
	jsonBlob := `{
		"id": "sess1",
		"parentID": "parent1",
		"title": "Test Session",
		"version": "1.0",
		"timeCreated": 1700000000000,
		"timeUpdated": 1700000001000,
		"directory": "/test/dir",
		"shareURL": "https://example.com/share",
		"agent": "build-medium",
		"model": {"providerID": "openai", "modelID": "gpt-5.5", "variant": "turbo"},
		"cost": 0.0023,
		"tokensInput": 100,
		"tokensOutput": 50,
		"tokensReasoning": 10,
		"tokensCacheRead": 5,
		"tokensCacheWrite": 2,
		"slug": "test-session",
		"workspaceID": "ws1",
		"path": "/test/dir"
	}`

	var info SessionInfo
	if err := json.Unmarshal([]byte(jsonBlob), &info); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if info.ID != "sess1" {
		t.Errorf("ID = %q, want %q", info.ID, "sess1")
	}
	if info.ParentID == nil || *info.ParentID != "parent1" {
		t.Errorf("ParentID = %v, want parent1", info.ParentID)
	}
	if info.Title != "Test Session" {
		t.Errorf("Title = %q, want %q", info.Title, "Test Session")
	}
	if info.Version != "1.0" {
		t.Errorf("Version = %q, want %q", info.Version, "1.0")
	}
	if info.TimeCreated != 1700000000000 {
		t.Errorf("TimeCreated = %d, want %d", info.TimeCreated, 1700000000000)
	}
	if info.Directory != "/test/dir" {
		t.Errorf("Directory = %q, want %q", info.Directory, "/test/dir")
	}
	if info.ShareURL == nil || *info.ShareURL != "https://example.com/share" {
		t.Errorf("ShareURL = %v", info.ShareURL)
	}
	if info.Agent != "build-medium" {
		t.Errorf("Agent = %q, want %q", info.Agent, "build-medium")
	}
	if info.Model.ProviderID != "openai" {
		t.Errorf("Model.ProviderID = %q", info.Model.ProviderID)
	}
	if info.Model.ModelID != "gpt-5.5" {
		t.Errorf("Model.ModelID = %q", info.Model.ModelID)
	}
	if info.Model.Variant != "turbo" {
		t.Errorf("Model.Variant = %q", info.Model.Variant)
	}
	if info.Cost != 0.0023 {
		t.Errorf("Cost = %v, want %v", info.Cost, 0.0023)
	}
	if info.TokensInput != 100 {
		t.Errorf("TokensInput = %d, want 100", info.TokensInput)
	}
	if info.TokensOutput != 50 {
		t.Errorf("TokensOutput = %d, want 50", info.TokensOutput)
	}
	if info.TokensReasoning != 10 {
		t.Errorf("TokensReasoning = %d, want 10", info.TokensReasoning)
	}
	if info.TokensCacheRead != 5 {
		t.Errorf("TokensCacheRead = %d, want 5", info.TokensCacheRead)
	}
	if info.TokensCacheWrite != 2 {
		t.Errorf("TokensCacheWrite = %d, want 2", info.TokensCacheWrite)
	}
	if info.Slug != "test-session" {
		t.Errorf("Slug = %q, want %q", info.Slug, "test-session")
	}
	if info.WorkspaceID != "ws1" {
		t.Errorf("WorkspaceID = %q, want %q", info.WorkspaceID, "ws1")
	}
	if info.Path != "/test/dir" {
		t.Errorf("Path = %q, want %q", info.Path, "/test/dir")
	}

	wantCreated := time.UnixMilli(1700000000000)
	if got := info.GetCreatedAt(); !got.Equal(wantCreated) {
		t.Errorf("GetCreatedAt() = %v, want %v", got, wantCreated)
	}
	wantUpdated := time.UnixMilli(1700000001000)
	if got := info.GetUpdatedAt(); !got.Equal(wantUpdated) {
		t.Errorf("GetUpdatedAt() = %v, want %v", got, wantUpdated)
	}
}

func TestModelInfoUnmarshal(t *testing.T) {
	jsonBlob := `{"providerID": "openai", "modelID": "gpt-4", "variant": "32k"}`
	var model ModelInfo
	if err := json.Unmarshal([]byte(jsonBlob), &model); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if model.ProviderID != "openai" {
		t.Errorf("ProviderID = %q", model.ProviderID)
	}
	if model.ModelID != "gpt-4" {
		t.Errorf("ModelID = %q", model.ModelID)
	}
	if model.Variant != "32k" {
		t.Errorf("Variant = %q", model.Variant)
	}
}

func TestToolStateUnmarshal(t *testing.T) {
	jsonBlob := `{
		"status": "completed",
		"input": {"command": "ls -la"},
		"output": "file1\nfile2",
		"metadata": {"exitCode": 0},
		"title": "List files",
		"timeStart": 1700000000000,
		"timeEnd": 1700000000500
	}`

	var state ToolState
	if err := json.Unmarshal([]byte(jsonBlob), &state); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if state.Status != "completed" {
		t.Errorf("Status = %q", state.Status)
	}
	if string(state.Input) != `{"command": "ls -la"}` {
		t.Errorf("Input = %s", state.Input)
	}
	if string(state.Output) != `"file1\nfile2"` {
		t.Errorf("Output = %s", state.Output)
	}
	if string(state.Metadata) != `{"exitCode": 0}` {
		t.Errorf("Metadata = %s", state.Metadata)
	}
	if state.Title != "List files" {
		t.Errorf("Title = %q", state.Title)
	}
	if state.TimeStart != 1700000000000 {
		t.Errorf("TimeStart = %d", state.TimeStart)
	}
	if state.TimeEnd != 1700000000500 {
		t.Errorf("TimeEnd = %d", state.TimeEnd)
	}
}

func TestFilePartDataUnmarshal(t *testing.T) {
	jsonBlob := `{"url": "https://example.com/file.txt", "filename": "file.txt", "mime": "text/plain"}`
	var data FilePartData
	if err := json.Unmarshal([]byte(jsonBlob), &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if data.URL != "https://example.com/file.txt" {
		t.Errorf("URL = %q", data.URL)
	}
	if data.Filename != "file.txt" {
		t.Errorf("Filename = %q", data.Filename)
	}
	if data.Mime != "text/plain" {
		t.Errorf("Mime = %q", data.Mime)
	}
}

func TestSessionUnmarshal(t *testing.T) {
	jsonBlob := `{
		"info": {"id": "sess1", "title": "Test", "version": "1", "timeCreated": 0, "timeUpdated": 0, "directory": "/test", "agent": "", "model": {}, "cost": 0, "tokensInput": 0, "tokensOutput": 0, "tokensReasoning": 0, "tokensCacheRead": 0, "tokensCacheWrite": 0, "slug": "", "workspaceID": "", "path": ""},
		"messages": [{"id": "m1", "sessionID": "sess1", "timeCreated": 0, "timeUpdated": 0, "data": {"role": "user"}}],
		"parts": [{"id": "p1", "messageID": "m1", "sessionID": "sess1", "timeCreated": 0, "timeUpdated": 0, "data": {"type": "text", "text": "hi"}}],
		"systemEvents": [{"id": "e1", "sessionID": "sess1", "type": "model-switched", "seq": 0, "data": {}, "timeCreated": 0, "timeUpdated": 0}]
	}`

	var sess Session
	if err := json.Unmarshal([]byte(jsonBlob), &sess); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if sess.Info.ID != "sess1" {
		t.Errorf("Info.ID = %q", sess.Info.ID)
	}
	if len(sess.Messages) != 1 || sess.Messages[0].ID != "m1" {
		t.Errorf("Messages = %+v", sess.Messages)
	}
	if len(sess.Parts) != 1 || sess.Parts[0].ID != "p1" {
		t.Errorf("Parts = %+v", sess.Parts)
	}
	if len(sess.SystemEvents) != 1 || sess.SystemEvents[0].ID != "e1" {
		t.Errorf("SystemEvents = %+v", sess.SystemEvents)
	}
}
