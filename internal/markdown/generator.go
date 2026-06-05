package markdown

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/byteowlz/ocse/internal/session"
)

// Generator handles markdown generation from session data
type Generator struct {
	includeCosts        bool
	includeTimings      bool
	includeSnapshots    bool
	includeSystemEvents bool
}

// Options configures the markdown generator
type Options struct {
	IncludeCosts        bool
	IncludeTimings      bool
	IncludeSnapshots    bool
	IncludeSystemEvents bool
}

// NewGenerator creates a new markdown generator
func NewGenerator(opts Options) *Generator {
	return &Generator{
		includeCosts:        opts.IncludeCosts,
		includeTimings:      opts.IncludeTimings,
		includeSnapshots:    opts.IncludeSnapshots,
		includeSystemEvents: opts.IncludeSystemEvents,
	}
}

// Generate creates markdown from a session
func (g *Generator) Generate(sess *session.Session) (string, error) {
	var md strings.Builder

	g.writeSessionHeader(&md, &sess.Info)

	partsByMessage := g.groupPartsByMessage(sess.Parts)

	for i := range sess.Messages {
		msg := &sess.Messages[i]
		g.writeMessage(&md, msg, partsByMessage[msg.ID], i+1)
	}

	if g.includeSystemEvents && len(sess.SystemEvents) > 0 {
		g.writeSystemEvents(&md, sess.SystemEvents)
	}

	return md.String(), nil
}

func (g *Generator) writeSessionHeader(md *strings.Builder, info *session.SessionInfo) {
	md.WriteString(fmt.Sprintf("# Session: %s\n\n", info.Title))
	md.WriteString(fmt.Sprintf("**Session ID:** `%s`  \n", info.ID))
	md.WriteString(fmt.Sprintf("**Created:** %s  \n", info.GetCreatedAt().Format("2006-01-02 15:04:05")))

	createdAt := info.GetCreatedAt()
	updatedAt := info.GetUpdatedAt()
	if !updatedAt.IsZero() && !updatedAt.Equal(createdAt) {
		duration := updatedAt.Sub(createdAt)
		md.WriteString(fmt.Sprintf("**Duration:** %s  \n", g.formatDuration(duration)))
	}

	if info.ShareURL != nil {
		md.WriteString(fmt.Sprintf("**Share URL:** %s  \n", *info.ShareURL))
	}

	// Agent
	if info.Agent != "" {
		md.WriteString(fmt.Sprintf("**Agent:** %s  \n", info.Agent))
	}

	// Model info
	model := info.Model
	if model.ProviderID != "" || model.ModelID != "" {
		md.WriteString(fmt.Sprintf("**Model:** %s/%s", model.ProviderID, model.ModelID))
		if model.Variant != "" {
			md.WriteString(fmt.Sprintf(" (%s)", model.Variant))
		}
		md.WriteString("  \n")
	}

	// Cost
	if g.includeCosts && info.Cost > 0 {
		md.WriteString(fmt.Sprintf("**Cost:** $%.4f  \n", info.Cost))
	}

	// Token breakdown
	tokens := session.TokenInfo{
		Total:      info.TokensInput + info.TokensOutput + info.TokensReasoning + info.TokensCacheRead + info.TokensCacheWrite,
		Input:      info.TokensInput,
		Output:     info.TokensOutput,
		Reasoning:  info.TokensReasoning,
		CacheRead:  info.TokensCacheRead,
		CacheWrite: info.TokensCacheWrite,
	}
	g.writeTokenSummary(md, tokens, "  \n")

	md.WriteString("\n---\n\n")
}

func (g *Generator) writeTokenSummary(md *strings.Builder, tokens session.TokenInfo, suffix string) {
	if tokens.Total == 0 {
		return
	}
	md.WriteString(fmt.Sprintf("**Tokens:** %s", g.formatTokens(tokens)))
	md.WriteString(suffix)
}

func (g *Generator) formatTokens(tokens session.TokenInfo) string {
	var parts []string
	if tokens.Input > 0 {
		parts = append(parts, fmt.Sprintf("%d in", tokens.Input))
	}
	if tokens.Output > 0 {
		parts = append(parts, fmt.Sprintf("%d out", tokens.Output))
	}
	if tokens.Reasoning > 0 {
		parts = append(parts, fmt.Sprintf("%d reasoning", tokens.Reasoning))
	}
	if tokens.CacheRead > 0 {
		parts = append(parts, fmt.Sprintf("%d cache-read", tokens.CacheRead))
	}
	if tokens.CacheWrite > 0 {
		parts = append(parts, fmt.Sprintf("%d cache-write", tokens.CacheWrite))
	}
	return fmt.Sprintf("total %d (%s)", tokens.Total, strings.Join(parts, ", "))
}

func (g *Generator) writeMessage(md *strings.Builder, msg *session.Message, parts []*session.MessagePart, messageNum int) {
	role := strings.Title(msg.Role())
	md.WriteString(fmt.Sprintf("## Message %d: %s\n", messageNum, role))

	md.WriteString(fmt.Sprintf("**Timestamp:** %s", msg.GetCreatedAt().Format("15:04:05")))

	if msg.Role() == "assistant" {
		modelID := msg.ModelID()
		if modelID != "" {
			md.WriteString(fmt.Sprintf(" | **Model:** %s", modelID))
		}
		if g.includeCosts {
			cost := msg.Cost()
			if cost > 0 {
				md.WriteString(fmt.Sprintf(" | **Cost:** $%.4f", cost))
			}
		}
		tokens := msg.Tokens()
		if tokens.Total > 0 || tokens.Input > 0 || tokens.Output > 0 {
			md.WriteString(fmt.Sprintf(" | **Tokens:** %s", g.formatTokens(tokens)))
		}
	}

	md.WriteString("\n\n")

	g.writeParts(md, parts)

	md.WriteString("---\n\n")
}

func (g *Generator) writeParts(md *strings.Builder, parts []*session.MessagePart) {
	var textParts, reasoningParts, toolParts, fileParts, otherParts []*session.MessagePart

	for i := range parts {
		part := parts[i]
		switch part.Type() {
		case "text":
			textParts = append(textParts, part)
		case "reasoning":
			reasoningParts = append(reasoningParts, part)
		case "tool":
			toolParts = append(toolParts, part)
		case "file":
			fileParts = append(fileParts, part)
		case "step-start", "step-finish":
			continue
		default:
			otherParts = append(otherParts, part)
		}
	}

	for _, part := range textParts {
		g.writeTextPart(md, part)
	}

	for _, part := range reasoningParts {
		g.writeReasoningPart(md, part)
	}

	if len(fileParts) > 0 {
		md.WriteString("### Attachments\n\n")
		for _, part := range fileParts {
			g.writeFilePart(md, part)
		}
		md.WriteString("\n")
	}

	if len(toolParts) > 0 {
		md.WriteString("### Tool Executions\n\n")
		for _, part := range toolParts {
			g.writeToolPart(md, part)
		}
	}

	for _, part := range otherParts {
		g.writeOtherPart(md, part)
	}
}

func (g *Generator) writeTextPart(md *strings.Builder, part *session.MessagePart) {
	text := part.Text()
	if text != "" {
		md.WriteString(text)
		md.WriteString("\n\n")
		return
	}

	var textData struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(part.Data, &textData); err != nil {
		md.WriteString(fmt.Sprintf("*[Error parsing text part: %v]*\n\n", err))
		return
	}

	md.WriteString(textData.Text)
	md.WriteString("\n\n")
}

func (g *Generator) writeReasoningPart(md *strings.Builder, part *session.MessagePart) {
	text := part.Text()
	if text == "" {
		var textData struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(part.Data, &textData); err == nil {
			text = textData.Text
		}
	}

	if text == "" {
		md.WriteString("*[Error: empty reasoning part]*\n\n")
		return
	}

	md.WriteString("<details>\n")
	md.WriteString("<summary>💭 Thinking</summary>\n\n")
	md.WriteString(text)
	md.WriteString("\n\n</details>\n\n")
}

func (g *Generator) writeToolPart(md *strings.Builder, part *session.MessagePart) {
	tool := part.Tool()
	stateJSON := part.State()

	if tool != "" && len(stateJSON) > 0 {
		g.writeToolPartFromState(md, tool, stateJSON)
		return
	}

	md.WriteString(fmt.Sprintf("*[Error: tool part missing data for part %s]*\n\n", part.ID))
}

func (g *Generator) writeToolPartFromState(md *strings.Builder, toolName string, stateJSON json.RawMessage) {
	var state session.ToolState
	if err := json.Unmarshal(stateJSON, &state); err != nil {
		md.WriteString(fmt.Sprintf("*[Error parsing tool state: %v]*\n\n", err))
		return
	}

	statusIcon := g.getStatusIcon(state.Status)
	md.WriteString(fmt.Sprintf("#### %s %s", statusIcon, toolName))

	if state.Title != "" {
		md.WriteString(fmt.Sprintf(" - \"%s\"", state.Title))
	}

	md.WriteString("\n")
	md.WriteString(fmt.Sprintf("**Status:** %s %s", statusIcon, strings.Title(state.Status)))

	if g.includeTimings && state.TimeStart > 0 && state.TimeEnd > 0 {
		start := time.UnixMilli(state.TimeStart)
		end := time.UnixMilli(state.TimeEnd)
		duration := end.Sub(start)
		md.WriteString(fmt.Sprintf(" | **Duration:** %s", g.formatDuration(duration)))
	}

	md.WriteString("\n\n")

	if len(state.Input) > 0 {
		md.WriteString("**Input:**\n")
		g.writeCodeBlock(md, state.Input, toolName)
		md.WriteString("\n")
	}

	if len(state.Output) > 0 {
		md.WriteString("**Output:**\n")
		g.writeJSONOutput(md, state.Output)
	}
}

func (g *Generator) writeJSONOutput(md *strings.Builder, data json.RawMessage) {
	var prettyData interface{}
	if err := json.Unmarshal(data, &prettyData); err == nil {
		if prettyJSON, err := json.MarshalIndent(prettyData, "", "  "); err == nil {
			md.WriteString("```json\n")
			md.WriteString(string(prettyJSON))
			md.WriteString("\n```\n\n")
			return
		}
	}
	var strData string
	if err := json.Unmarshal(data, &strData); err == nil {
		md.WriteString("```\n")
		md.WriteString(strData)
		md.WriteString("\n```\n\n")
	} else {
		md.WriteString("```\n")
		md.WriteString(string(data))
		md.WriteString("\n```\n\n")
	}
}

func (g *Generator) writeFilePart(md *strings.Builder, part *session.MessagePart) {
	var fileData session.FilePartData
	if err := json.Unmarshal(part.Data, &fileData); err != nil {
		md.WriteString(fmt.Sprintf("*[Error parsing file part: %v]*\n", err))
		return
	}

	icon := g.getFileIcon(fileData.Mime)
	md.WriteString(fmt.Sprintf("- %s `%s` (%s)\n", icon, fileData.Filename, fileData.Mime))
}

func (g *Generator) writeOtherPart(md *strings.Builder, part *session.MessagePart) {
	md.WriteString(fmt.Sprintf("### %s Part\n\n", strings.Title(part.Type())))
	md.WriteString("```json\n")

	var prettyData interface{}
	if err := json.Unmarshal(part.Data, &prettyData); err == nil {
		if prettyJSON, err := json.MarshalIndent(prettyData, "", "  "); err == nil {
			md.WriteString(string(prettyJSON))
		} else {
			md.WriteString(string(part.Data))
		}
	} else {
		md.WriteString(string(part.Data))
	}

	md.WriteString("\n```\n\n")
}

func (g *Generator) writeSystemEvents(md *strings.Builder, events []session.SystemEvent) {
	md.WriteString("## System Events\n\n")
	md.WriteString("| Time | Event | Details |\n")
	md.WriteString("|------|-------|---------|\n")

	for i := range events {
		g.writeSystemEventRow(md, &events[i])
	}

	md.WriteString("\n")
}

func (g *Generator) writeSystemEventRow(md *strings.Builder, event *session.SystemEvent) {
	timestamp := event.GetCreatedAt().Format("15:04:05")
	var details string

	switch event.Type {
	case "model-switched":
		var data struct {
			ModelID    string `json:"modelID"`
			ProviderID string `json:"providerID"`
		}
		if err := json.Unmarshal(event.Data, &data); err == nil {
			details = fmt.Sprintf("%s/%s", data.ProviderID, data.ModelID)
		} else {
			details = "unknown model"
		}
	case "agent-switched":
		var data struct {
			Agent string `json:"agent"`
		}
		if err := json.Unmarshal(event.Data, &data); err == nil {
			details = data.Agent
		} else {
			details = "unknown agent"
		}
	default:
		var raw interface{}
		if err := json.Unmarshal(event.Data, &raw); err == nil {
			if b, err := json.Marshal(raw); err == nil {
				details = string(b)
			} else {
				details = string(event.Data)
			}
		} else {
			details = string(event.Data)
		}
	}

	md.WriteString(fmt.Sprintf("| %s | %s | %s |\n", timestamp, event.Type, details))
}

func (g *Generator) writeCodeBlock(md *strings.Builder, data json.RawMessage, toolName string) {
	lang := g.getLanguageFromTool(toolName)
	md.WriteString(fmt.Sprintf("```%s\n", lang))

	if lang == "json" {
		var prettyData interface{}
		if err := json.Unmarshal(data, &prettyData); err == nil {
			if prettyJSON, err := json.MarshalIndent(prettyData, "", "  "); err == nil {
				md.WriteString(string(prettyJSON))
			} else {
				md.WriteString(string(data))
			}
		} else {
			md.WriteString(string(data))
		}
	} else {
		var strData string
		if err := json.Unmarshal(data, &strData); err == nil {
			md.WriteString(strData)
		} else {
			md.WriteString(string(data))
		}
	}

	md.WriteString("\n```")
}

func (g *Generator) groupPartsByMessage(parts []session.MessagePart) map[string][]*session.MessagePart {
	result := make(map[string][]*session.MessagePart)
	for i := range parts {
		part := &parts[i]
		result[part.MessageID] = append(result[part.MessageID], part)
	}
	return result
}

func (g *Generator) getStatusIcon(state string) string {
	switch state {
	case "completed":
		return "✅"
	case "error":
		return "❌"
	case "running":
		return "🔄"
	case "pending":
		return "⏳"
	default:
		return "❓"
	}
}

func (g *Generator) getFileIcon(mimeType string) string {
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return "🖼️"
	case strings.HasPrefix(mimeType, "text/"):
		return "📄"
	case strings.Contains(mimeType, "json"):
		return "📋"
	case strings.Contains(mimeType, "pdf"):
		return "📕"
	default:
		return "📎"
	}
}

func (g *Generator) getLanguageFromTool(toolName string) string {
	switch toolName {
	case "bash", "shell":
		return "bash"
	default:
		return ""
	}
}

func (g *Generator) formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}
