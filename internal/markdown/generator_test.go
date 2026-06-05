package markdown

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/byteowlz/ocse/internal/session"
)

func makeTestSession() *session.Session {
	shareURL := "https://example.com/share"
	parentID := "parent1"

	info := session.SessionInfo{
		ID:               "sess1",
		ParentID:         &parentID,
		Title:            "Test Session",
		Version:          "1.0",
		TimeCreated:      1700000000000,
		TimeUpdated:      1700000001000,
		Directory:        "/test/dir",
		ShareURL:         &shareURL,
		Agent:            "build-medium",
		Model:            session.ModelInfo{ProviderID: "openai", ModelID: "gpt-5.5", Variant: "turbo"},
		Cost:             0.0023,
		TokensInput:      100,
		TokensOutput:     50,
		TokensReasoning:  10,
		TokensCacheRead:  5,
		TokensCacheWrite: 2,
		Slug:             "test-sess",
		WorkspaceID:      "ws1",
		Path:             "/test/dir",
	}

	msg1Data := json.RawMessage(`{"role": "user", "mode": "chat", "agent": "", "cost": 0, "tokens": {"total": 0, "input": 0, "output": 0, "reasoning": 0, "cacheRead": 0, "cacheWrite": 0}, "modelID": "", "providerID": "", "finish": "", "timeCreated": 1700000001000, "timeCompleted": 0}`)
	msg2Data := json.RawMessage(`{"role": "assistant", "mode": "build", "agent": "build-medium", "cost": 0.0012, "tokens": {"total": 75, "input": 50, "output": 25, "reasoning": 5, "cacheRead": 3, "cacheWrite": 1}, "modelID": "gpt-5.5", "providerID": "openai", "finish": "stop", "timeCreated": 1700000002000, "timeCompleted": 1700000002500}`)

	messages := []session.Message{
		{ID: "msg1", SessionID: "sess1", CreatedAt: 1700000001000, UpdatedAt: 1700000001000, Data: msg1Data},
		{ID: "msg2", SessionID: "sess1", CreatedAt: 1700000002000, UpdatedAt: 1700000002000, Data: msg2Data},
	}

	part1Data := json.RawMessage(`{"type": "text", "text": "Hello, how can I help?"}`)
	part2Data := json.RawMessage(`{"type": "text", "text": "This is the assistant response."}`)
	part3Data := json.RawMessage(`{"type": "tool", "text": "", "tool": "bash", "callID": "call1", "state": {"status": "completed", "input": {"command": "ls -la"}, "output": "total 10\nfile1.txt", "metadata": {"exitCode": 0}, "title": "List files", "timeStart": 1700000002000, "timeEnd": 1700000002100}}`)
	part4Data := json.RawMessage(`{"type": "reasoning", "text": "Let me think about this step by step..."}`)
	part5Data := json.RawMessage(`{"type": "file", "text": "", "tool": "", "callID": "", "state": null}`)
	part6Data := json.RawMessage(`{"type": "step-start", "text": ""}`)
	part7Data := json.RawMessage(`{"type": "step-finish", "text": ""}`)

	parts := []session.MessagePart{
		{ID: "p1", MessageID: "msg1", SessionID: "sess1", TimeCreated: 1700000001000, TimeUpdated: 1700000001000, Data: part1Data},
		{ID: "p2", MessageID: "msg2", SessionID: "sess1", TimeCreated: 1700000002000, TimeUpdated: 1700000002000, Data: part2Data},
		{ID: "p3", MessageID: "msg2", SessionID: "sess1", TimeCreated: 1700000002100, TimeUpdated: 1700000002100, Data: part3Data},
		{ID: "p4", MessageID: "msg2", SessionID: "sess1", TimeCreated: 1700000002200, TimeUpdated: 1700000002200, Data: part4Data},
		{ID: "p5", MessageID: "msg1", SessionID: "sess1", TimeCreated: 1700000001300, TimeUpdated: 1700000001300, Data: part5Data},
		{ID: "p6", MessageID: "msg2", SessionID: "sess1", TimeCreated: 1700000002300, TimeUpdated: 1700000002300, Data: part6Data},
		{ID: "p7", MessageID: "msg2", SessionID: "sess1", TimeCreated: 1700000002400, TimeUpdated: 1700000002400, Data: part7Data},
	}

	events := []session.SystemEvent{
		{ID: "evt1", SessionID: "sess1", Type: "model-switched", Seq: 0, Data: json.RawMessage(`{"modelID": "gpt-4", "providerID": "openai"}`), TimeCreated: 1700000000500, TimeUpdated: 1700000000500},
		{ID: "evt2", SessionID: "sess1", Type: "agent-switched", Seq: 1, Data: json.RawMessage(`{"agent": "build-medium"}`), TimeCreated: 1700000000600, TimeUpdated: 1700000000600},
	}

	return &session.Session{
		Info:         info,
		Messages:     messages,
		Parts:        parts,
		SystemEvents: events,
	}
}

func TestGenerate_Basic(t *testing.T) {
	sess := makeTestSession()
	g := NewGenerator(Options{})

	md, err := g.Generate(sess)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Header
	if !strings.Contains(md, "# Session: Test Session") {
		t.Error("missing session title header")
	}
	if !strings.Contains(md, "**Session ID:** `sess1`") {
		t.Error("missing session ID")
	}
	if !strings.Contains(md, "**Created:** 2023-11-14") {
		t.Error("missing creation date")
	}
	if !strings.Contains(md, "**Duration:**") {
		t.Error("missing duration")
	}
	if !strings.Contains(md, "**Share URL:** https://example.com/share") {
		t.Error("missing share URL")
	}
	if !strings.Contains(md, "**Agent:** build-medium") {
		t.Error("missing agent")
	}
	if !strings.Contains(md, "**Model:** openai/gpt-5.5 (turbo)") {
		t.Error("missing model info")
	}
	// Cost not included with default Options{}
	if strings.Contains(md, "**Cost:** $0.0023") {
		t.Error("cost should not appear when IncludeCosts is false")
	}
	if !strings.Contains(md, "**Tokens:** total 167 (100 in, 50 out, 10 reasoning, 5 cache-read, 2 cache-write)") {
		t.Error("missing token breakdown in header")
	}

	// Messages
	if !strings.Contains(md, "## Message 1: User") {
		t.Error("missing user message")
	}
	if !strings.Contains(md, "## Message 2: Assistant") {
		t.Error("missing assistant message")
	}
	if !strings.Contains(md, "Hello, how can I help?") {
		t.Error("missing text part content")
	}
	if !strings.Contains(md, "This is the assistant response.") {
		t.Error("missing assistant response text")
	}

	// Tool parts
	if !strings.Contains(md, "#### ✅ bash") {
		t.Error("missing tool header")
	}
	if !strings.Contains(md, "List files") {
		t.Error("missing tool title")
	}
	if !strings.Contains(md, "**Status:** ✅ Completed") {
		t.Error("missing tool status")
	}
	if !strings.Contains(md, "**Input:**") {
		t.Error("missing tool input")
	}
	if !strings.Contains(md, "**Output:**") {
		t.Error("missing tool output")
	}

	// Reasoning part
	if !strings.Contains(md, "<details>") {
		t.Error("missing reasoning details block")
	}
	if !strings.Contains(md, "<summary>💭 Thinking</summary>") {
		t.Error("missing reasoning summary")
	}
	if !strings.Contains(md, "Let me think about this step by step...") {
		t.Error("missing reasoning text")
	}
	if !strings.Contains(md, "</details>") {
		t.Error("missing reasoning details close")
	}

	// File parts
	if !strings.Contains(md, "### Attachments") {
		t.Error("missing attachments section")
	}

	// System events should NOT be included by default (Options{} has false)
	if strings.Contains(md, "## System Events") {
		t.Error("system events should not be included by default")
	}

	// Step-start/finish should be skipped
	if strings.Contains(md, "step-start") {
		t.Error("step-start should be skipped")
	}
	if strings.Contains(md, "step-finish") {
		t.Error("step-finish should be skipped")
	}
}

func TestGenerate_WithSystemEvents(t *testing.T) {
	sess := makeTestSession()
	g := NewGenerator(Options{IncludeSystemEvents: true})

	md, err := g.Generate(sess)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if !strings.Contains(md, "## System Events") {
		t.Error("missing system events section")
	}
	if !strings.Contains(md, "| Time | Event | Details |") {
		t.Error("missing system events table header")
	}
	if !strings.Contains(md, "| model-switched | openai/gpt-4 |") {
		t.Error("missing model-switched event")
	}
	if !strings.Contains(md, "| agent-switched | build-medium |") {
		t.Error("missing agent-switched event")
	}
}

func TestGenerate_WithCosts(t *testing.T) {
	sess := makeTestSession()
	g := NewGenerator(Options{IncludeCosts: true})

	md, err := g.Generate(sess)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Header should have cost
	if !strings.Contains(md, "**Cost:** $0.0023") {
		t.Error("missing cost in header")
	}
	// Message 2 (assistant) should have cost
	if !strings.Contains(md, "**Cost:** $0.0012") {
		t.Error("missing message cost")
	}
}

func TestGenerate_WithTimings(t *testing.T) {
	sess := makeTestSession()
	g := NewGenerator(Options{IncludeTimings: true})

	md, err := g.Generate(sess)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Tool part should have duration
	if !strings.Contains(md, "**Duration:**") {
		t.Error("missing duration in tool output")
	}
	// The duration should be 100ms which formats as "0.1s"
	if !strings.Contains(md, "0.1s") {
		t.Error("missing duration value")
	}
}

func TestGenerate_WithAllOptions(t *testing.T) {
	sess := makeTestSession()
	g := NewGenerator(Options{IncludeCosts: true, IncludeTimings: true, IncludeSystemEvents: true})

	md, err := g.Generate(sess)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if !strings.Contains(md, "**Cost:** $0.0023") {
		t.Error("missing cost")
	}
	if !strings.Contains(md, "**Duration:**") {
		t.Error("missing duration")
	}
	if !strings.Contains(md, "## System Events") {
		t.Error("missing system events")
	}
}

func TestGenerate_ToolPartError(t *testing.T) {
	sess := makeTestSession()
	// Replace tool part with bad data
	sess.Parts[2].Data = json.RawMessage(`{"type": "tool"}`) // missing tool name and state

	g := NewGenerator(Options{})
	md, err := g.Generate(sess)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if !strings.Contains(md, "*[Error: tool part missing data") {
		t.Error("missing tool error message")
	}
}

func TestGenerate_FilePart(t *testing.T) {
	sess := makeTestSession()
	// Add a file part with actual file data
	fileData := json.RawMessage(`{"url": "https://example.com/test.txt", "filename": "test.txt", "mime": "text/plain"}`)
	sess.Parts = append(sess.Parts, session.MessagePart{
		ID: "p8", MessageID: "msg1", SessionID: "sess1",
		TimeCreated: 1700000001400, TimeUpdated: 1700000001400,
		Data: fileData,
	})

	g := NewGenerator(Options{})
	md, err := g.Generate(sess)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if !strings.Contains(md, "test.txt") {
		t.Error("missing filename")
	}
	if !strings.Contains(md, "text/plain") {
		t.Error("missing mime type")
	}
}

func TestGenerate_OtherPart(t *testing.T) {
	sess := makeTestSession()
	otherData := json.RawMessage(`{"type": "other", "key": "value", "number": 42}`)
	sess.Parts = append(sess.Parts, session.MessagePart{
		ID: "p9", MessageID: "msg2", SessionID: "sess1",
		TimeCreated: 1700000002500, TimeUpdated: 1700000002500,
		Data: otherData,
	})

	g := NewGenerator(Options{})
	md, err := g.Generate(sess)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if !strings.Contains(md, "Other Part") {
		t.Error("missing other part header")
	}
	if !strings.Contains(md, "```json") {
		t.Error("missing JSON code block")
	}
	if !strings.Contains(md, `"key": "value"`) {
		t.Error("missing JSON content")
	}
}

func TestGenerate_EmptySession(t *testing.T) {
	sess := &session.Session{
		Info: session.SessionInfo{
			ID:          "empty",
			Title:       "Empty Session",
			Version:     "1.0",
			TimeCreated: 0,
			TimeUpdated: 0,
			Directory:   "/test",
			Agent:       "",
			Model:       session.ModelInfo{},
			Slug:        "empty",
			WorkspaceID: "",
			Path:        "/test",
		},
		Messages:     []session.Message{},
		Parts:        []session.MessagePart{},
		SystemEvents: []session.SystemEvent{},
	}

	g := NewGenerator(Options{})
	md, err := g.Generate(sess)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if !strings.Contains(md, "# Session: Empty Session") {
		t.Error("missing session title")
	}
	if !strings.Contains(md, "**Session ID:** `empty`") {
		t.Error("missing session ID")
	}
	// No cost or tokens should be shown when zero
	if strings.Contains(md, "**Cost:**") {
		t.Error("cost should not appear for zero cost")
	}
	if strings.Contains(md, "**Tokens:**") {
		t.Error("tokens should not appear for zero tokens")
	}
}

func TestGenerate_SessionInfoWithoutShareURL(t *testing.T) {
	sess := makeTestSession()
	sess.Info.ShareURL = nil

	g := NewGenerator(Options{})
	md, err := g.Generate(sess)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if strings.Contains(md, "**Share URL:**") {
		t.Error("share URL should not appear when nil")
	}
}

func TestGenerate_SessionInfoWithoutModel(t *testing.T) {
	sess := makeTestSession()
	sess.Info.Model = session.ModelInfo{}

	g := NewGenerator(Options{})
	md, err := g.Generate(sess)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Check only the header section (before first `---`)
	headerParts := strings.Split(md, "\n---\n")
	if len(headerParts) == 0 {
		t.Fatal("no header section found")
	}
	header := headerParts[0]
	if strings.Contains(header, "**Model:**") {
		t.Error("model should not appear in header when empty")
	}
}

func TestGenerate_AssistantMessageWithoutModel(t *testing.T) {
	sess := makeTestSession()
	// Update message 2 to have empty model info
	msg2Data := json.RawMessage(`{"role": "assistant", "mode": "build", "agent": "", "cost": 0.0012, "tokens": {"total": 75, "input": 50, "output": 25, "reasoning": 5, "cacheRead": 3, "cacheWrite": 1}, "modelID": "", "providerID": "", "finish": "stop", "timeCreated": 1700000002000, "timeCompleted": 1700000002500}`)
	sess.Messages[1].Data = msg2Data

	g := NewGenerator(Options{IncludeCosts: true})
	md, err := g.Generate(sess)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Should not have model line for assistant message
	msg2Section := strings.Split(md, "## Message 2: Assistant")[1]
	msg2Section = strings.Split(msg2Section, "---")[0]
	if strings.Contains(msg2Section, "**Model:**") {
		t.Error("model should not appear when empty in message")
	}
	// But should still have cost
	if !strings.Contains(msg2Section, "**Cost:** $0.0012") {
		t.Error("missing message cost")
	}
}

func TestGenerate_ToolWithErrorState(t *testing.T) {
	sess := makeTestSession()
	toolData := json.RawMessage(`{"type": "tool", "text": "", "tool": "bash", "callID": "call2", "state": {"status": "error", "input": {"command": "false"}, "output": "exit 1", "metadata": null, "title": "", "timeStart": 1700000002000, "timeEnd": 1700000002100}}`)
	parts := []session.MessagePart{
		{ID: "p_err", MessageID: "msg2", SessionID: "sess1", TimeCreated: 1700000002600, TimeUpdated: 1700000002600, Data: toolData},
	}
	sess.Parts = parts

	g := NewGenerator(Options{})
	md, err := g.Generate(sess)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if !strings.Contains(md, "❌ bash") {
		t.Error("missing error status icon")
	}
	if !strings.Contains(md, "**Status:** ❌ Error") {
		t.Error("missing error status text")
	}
}

func TestGenerate_ToolRunningState(t *testing.T) {
	sess := makeTestSession()
	toolData := json.RawMessage(`{"type": "tool", "text": "", "tool": "bash", "callID": "call3", "state": {"status": "running", "input": {"command": "sleep 10"}, "output": "", "metadata": null, "title": "Sleep", "timeStart": 1700000002000, "timeEnd": 0}}`)
	parts := []session.MessagePart{
		{ID: "p_run", MessageID: "msg2", SessionID: "sess1", TimeCreated: 1700000002700, TimeUpdated: 1700000002700, Data: toolData},
	}
	sess.Parts = parts

	g := NewGenerator(Options{IncludeTimings: true})
	md, err := g.Generate(sess)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if !strings.Contains(md, "🔄 bash") {
		t.Error("missing running status icon")
	}
	if !strings.Contains(md, "**Status:** 🔄 Running") {
		t.Error("missing running status text")
	}
	// Duration should not be shown since timeEnd is 0
	msg2Section := strings.Split(md, "## Message 2: Assistant")[1]
	msg2Section = strings.Split(msg2Section, "---")[0]
	if strings.Contains(msg2Section, "**Duration:**") {
		t.Error("duration should not appear when timeEnd is 0")
	}
}

func TestGenerate_UnknownStatus(t *testing.T) {
	sess := makeTestSession()
	toolData := json.RawMessage(`{"type": "tool", "text": "", "tool": "unknown", "callID": "call4", "state": {"status": "unknown", "input": {}, "output": "", "metadata": null, "title": "", "timeStart": 1700000002000, "timeEnd": 1700000002100}}`)
	parts := []session.MessagePart{
		{ID: "p_unk", MessageID: "msg2", SessionID: "sess1", TimeCreated: 1700000002800, TimeUpdated: 1700000002800, Data: toolData},
	}
	sess.Parts = parts

	g := NewGenerator(Options{})
	md, err := g.Generate(sess)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if !strings.Contains(md, "❓ unknown") {
		t.Error("missing unknown status icon")
	}
}

func TestGenerate_TextPartFallback(t *testing.T) {
	sess := makeTestSession()
	// Text part without text field, should fallback to parsing Data
	textData := json.RawMessage(`{"type": "text", "text": "fallback text"}`)
	parts := []session.MessagePart{
		{ID: "p_fb", MessageID: "msg1", SessionID: "sess1", TimeCreated: 1700000001500, TimeUpdated: 1700000001500, Data: textData},
	}
	sess.Parts = parts
	sess.Messages = []session.Message{
		{ID: "msg1", SessionID: "sess1", CreatedAt: 1700000001000, UpdatedAt: 1700000001000,
			Data: json.RawMessage(`{"role": "user", "mode": "chat", "agent": "", "cost": 0, "tokens": {"total": 0, "input": 0, "output": 0, "reasoning": 0, "cacheRead": 0, "cacheWrite": 0}, "modelID": "", "providerID": "", "finish": "", "timeCreated": 1700000001000, "timeCompleted": 0}`)},
	}

	g := NewGenerator(Options{})
	md, err := g.Generate(sess)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if !strings.Contains(md, "fallback text") {
		t.Error("missing fallback text")
	}
}

func TestGenerate_FileIcons(t *testing.T) {
	tests := []struct {
		mime  string
		icon  string
	}{
		{"image/png", "🖼️"},
		{"text/plain", "📄"},
		{"application/json", "📋"},
		{"application/pdf", "📕"},
		{"application/octet-stream", "📎"},
	}

	for _, tc := range tests {
		sess := makeTestSession()
		fileData := json.RawMessage(fmt.Sprintf(`{"type": "file", "url": "", "filename": "file.%s", "mime": "%s"}`, tc.mime, tc.mime))
		sess.Parts = []session.MessagePart{
			{ID: "pf", MessageID: "msg1", SessionID: "sess1", TimeCreated: 0, TimeUpdated: 0, Data: fileData},
		}
		sess.Messages = []session.Message{
			{ID: "msg1", SessionID: "sess1", CreatedAt: 0, UpdatedAt: 0,
				Data: json.RawMessage(`{"role": "user", "mode": "chat", "agent": "", "cost": 0, "tokens": {"total": 0, "input": 0, "output": 0, "reasoning": 0, "cacheRead": 0, "cacheWrite": 0}, "modelID": "", "providerID": "", "finish": "", "timeCreated": 0, "timeCompleted": 0}`)},
		}

		g := NewGenerator(Options{})
		md, err := g.Generate(sess)
		if err != nil {
			t.Fatalf("Generate: %v", err)
		}

		if !strings.Contains(md, tc.icon) {
			t.Errorf("missing icon %s for mime %s", tc.icon, tc.mime)
		}
	}
}
