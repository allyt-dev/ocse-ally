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
	includeCosts     bool
	includeTimings   bool
	includeSnapshots bool
}

// Options configures the markdown generator
type Options struct {
	IncludeCosts     bool
	IncludeTimings   bool
	IncludeSnapshots bool
}

// NewGenerator creates a new markdown generator
func NewGenerator(opts Options) *Generator {
	return &Generator{
		includeCosts:     opts.IncludeCosts,
		includeTimings:   opts.IncludeTimings,
		includeSnapshots: opts.IncludeSnapshots,
	}
}

// Generate creates markdown from a session
func (g *Generator) Generate(sess *session.Session) (string, error) {
	var md strings.Builder

	// Session header
	g.writeSessionHeader(&md, &sess.Info)

	// Group parts by message
	partsByMessage := g.groupPartsByMessage(sess.Parts)

	// Process messages in order
	for i, message := range sess.Messages {
		g.writeMessage(&md, &message, partsByMessage[message.ID], i+1)
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

	md.WriteString("\n---\n\n")
}

func (g *Generator) writeMessage(md *strings.Builder, msg *session.Message, parts []session.MessagePart, messageNum int) {
	// Message header
	role := strings.Title(msg.Role)
	md.WriteString(fmt.Sprintf("## Message %d: %s\n", messageNum, role))

	// Timestamp
	md.WriteString(fmt.Sprintf("**Timestamp:** %s", msg.GetCreatedAt().Format("15:04:05")))

	// Assistant metadata
	if msg.Role == "assistant" {
		if msg.Model != nil {
			md.WriteString(fmt.Sprintf(" | **Model:** %s", *msg.Model))
		}
		if g.includeCosts && msg.Cost != nil {
			md.WriteString(fmt.Sprintf(" | **Cost:** $%.4f", *msg.Cost))
		}
		if msg.InputTokens != nil && msg.OutputTokens != nil {
			md.WriteString(fmt.Sprintf(" | **Tokens:** %d in, %d out", *msg.InputTokens, *msg.OutputTokens))
		}
	}

	md.WriteString("\n\n")

	// Process parts
	g.writeParts(md, parts)

	md.WriteString("---\n\n")
}

func (g *Generator) writeParts(md *strings.Builder, parts []session.MessagePart) {
	var textParts []session.MessagePart
	var toolParts []session.MessagePart
	var fileParts []session.MessagePart
	var otherParts []session.MessagePart

	// Group parts by type
	for _, part := range parts {
		switch part.Type {
		case "text":
			textParts = append(textParts, part)
		case "tool":
			toolParts = append(toolParts, part)
		case "file":
			fileParts = append(fileParts, part)
		case "step-start", "step-finish":
			// Skip step metadata parts - these are internal processing markers
			continue
		default:
			otherParts = append(otherParts, part)
		}
	}

	// Write text parts first
	for _, part := range textParts {
		g.writeTextPart(md, part)
	}

	// Write file attachments
	if len(fileParts) > 0 {
		md.WriteString("### Attachments\n\n")
		for _, part := range fileParts {
			g.writeFilePart(md, part)
		}
		md.WriteString("\n")
	}

	// Write tool executions
	if len(toolParts) > 0 {
		md.WriteString("### Tool Executions\n\n")
		for _, part := range toolParts {
			g.writeToolPart(md, part)
		}
	}

	// Write other parts
	for _, part := range otherParts {
		g.writeOtherPart(md, part)
	}
}

func (g *Generator) writeTextPart(md *strings.Builder, part session.MessagePart) {
	// For text parts, the text is directly in the Text field
	if part.Text != nil {
		md.WriteString(*part.Text)
		md.WriteString("\n\n")
		return
	}

	// Fallback: try to parse from Data field
	var textData session.TextPartData
	if err := json.Unmarshal(part.Data, &textData); err != nil {
		md.WriteString(fmt.Sprintf("*[Error parsing text part: %v]*\n\n", err))
		return
	}

	md.WriteString(textData.Text)
	md.WriteString("\n\n")
}

func (g *Generator) writeToolPart(md *strings.Builder, part session.MessagePart) {
	// For tool parts, the data is directly in the part fields
	if part.Tool != nil && part.State != nil {
		g.writeToolPartDirect(md, part)
		return
	}

	// Fallback: try to parse from Data field
	var toolData session.ToolPartData
	if err := json.Unmarshal(part.Data, &toolData); err != nil {
		md.WriteString(fmt.Sprintf("*[Error parsing tool part: %v]*\n\n", err))
		return
	}

	g.writeToolPartFromData(md, toolData)
}

func (g *Generator) writeToolPartDirect(md *strings.Builder, part session.MessagePart) {
	var state session.ToolStateData
	if err := json.Unmarshal(part.State, &state); err != nil {
		md.WriteString(fmt.Sprintf("*[Error parsing tool state: %v]*\n\n", err))
		return
	}

	// Tool header with status
	statusIcon := g.getStatusIcon(state.Status)
	toolName := *part.Tool
	md.WriteString(fmt.Sprintf("#### %s %s", statusIcon, toolName))

	if state.Title != nil {
		md.WriteString(fmt.Sprintf(" - \"%s\"", *state.Title))
	}

	md.WriteString("\n")

	// Status and timing
	md.WriteString(fmt.Sprintf("**Status:** %s %s", statusIcon, strings.Title(state.Status)))

	if g.includeTimings && state.Time != nil {
		start := time.UnixMilli(state.Time.Start)
		end := time.UnixMilli(state.Time.End)
		duration := end.Sub(start)
		md.WriteString(fmt.Sprintf(" | **Duration:** %s", g.formatDuration(duration)))
	}

	md.WriteString("\n\n")

	// Tool input
	if state.Input != nil {
		md.WriteString("**Input:**\n")
		g.writeCodeBlock(md, state.Input, toolName)
		md.WriteString("\n")
	}

	// Tool output
	if state.Output != nil {
		md.WriteString("**Output:**\n")
		if outputStr, ok := state.Output.(string); ok {
			md.WriteString("```\n")
			md.WriteString(outputStr)
			md.WriteString("\n```\n\n")
		} else {
			// Try to marshal as JSON
			if outputJSON, err := json.MarshalIndent(state.Output, "", "  "); err == nil {
				md.WriteString("```json\n")
				md.WriteString(string(outputJSON))
				md.WriteString("\n```\n\n")
			} else {
				md.WriteString("```\n")
				md.WriteString(fmt.Sprintf("%v", state.Output))
				md.WriteString("\n```\n\n")
			}
		}
	}
}

func (g *Generator) writeToolPartFromData(md *strings.Builder, toolData session.ToolPartData) {
	// Tool header with status
	statusIcon := g.getStatusIcon(toolData.State.Status)
	md.WriteString(fmt.Sprintf("#### %s %s", statusIcon, toolData.Tool))

	if toolData.State.Title != nil {
		md.WriteString(fmt.Sprintf(" - \"%s\"", *toolData.State.Title))
	}

	md.WriteString("\n")

	// Status and timing
	md.WriteString(fmt.Sprintf("**Status:** %s %s", statusIcon, strings.Title(toolData.State.Status)))

	if g.includeTimings && toolData.State.Time != nil {
		start := time.UnixMilli(toolData.State.Time.Start)
		end := time.UnixMilli(toolData.State.Time.End)
		duration := end.Sub(start)
		md.WriteString(fmt.Sprintf(" | **Duration:** %s", g.formatDuration(duration)))
	}

	md.WriteString("\n\n")

	// Tool input
	if toolData.State.Input != nil {
		md.WriteString("**Input:**\n")
		g.writeCodeBlock(md, toolData.State.Input, toolData.Tool)
		md.WriteString("\n")
	}

	// Tool output
	if toolData.State.Output != nil {
		md.WriteString("**Output:**\n")
		if outputStr, ok := toolData.State.Output.(string); ok {
			md.WriteString("```\n")
			md.WriteString(outputStr)
			md.WriteString("\n```\n\n")
		} else {
			// Try to marshal as JSON
			if outputJSON, err := json.MarshalIndent(toolData.State.Output, "", "  "); err == nil {
				md.WriteString("```json\n")
				md.WriteString(string(outputJSON))
				md.WriteString("\n```\n\n")
			} else {
				md.WriteString("```\n")
				md.WriteString(fmt.Sprintf("%v", toolData.State.Output))
				md.WriteString("\n```\n\n")
			}
		}
	}
}

func (g *Generator) writeFilePart(md *strings.Builder, part session.MessagePart) {
	var fileData session.FilePartData
	if err := json.Unmarshal(part.Data, &fileData); err != nil {
		md.WriteString(fmt.Sprintf("*[Error parsing file part: %v]*\n", err))
		return
	}

	icon := g.getFileIcon(fileData.MimeType)
	md.WriteString(fmt.Sprintf("- %s `%s`", icon, fileData.Name))

	if fileData.Size != nil {
		md.WriteString(fmt.Sprintf(" (%s)", g.formatFileSize(*fileData.Size)))
	}

	md.WriteString(fmt.Sprintf(" (%s)", fileData.MimeType))
	md.WriteString("\n")
}

func (g *Generator) writeOtherPart(md *strings.Builder, part session.MessagePart) {
	md.WriteString(fmt.Sprintf("### %s Part\n\n", strings.Title(part.Type)))
	md.WriteString("```json\n")

	// Pretty print the JSON data
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

func (g *Generator) writeCodeBlock(md *strings.Builder, data json.RawMessage, toolName string) {
	// Try to determine language from tool name
	lang := g.getLanguageFromTool(toolName)

	md.WriteString(fmt.Sprintf("```%s\n", lang))

	// Try to pretty print if it's JSON
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
		// For non-JSON, try to extract string content
		var strData string
		if err := json.Unmarshal(data, &strData); err == nil {
			md.WriteString(strData)
		} else {
			md.WriteString(string(data))
		}
	}

	md.WriteString("\n```")
}

func (g *Generator) groupPartsByMessage(parts []session.MessagePart) map[string][]session.MessagePart {
	result := make(map[string][]session.MessagePart)
	for _, part := range parts {
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
	case "read", "write", "edit":
		return ""
	case "grep", "glob":
		return ""
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

func (g *Generator) formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
