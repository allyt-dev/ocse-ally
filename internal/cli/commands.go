package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/byteowlz/ocse/internal/markdown"
	"github.com/byteowlz/ocse/internal/session"
)

// Execute runs the CLI application
func Execute() error {
	if len(os.Args) < 2 {
		printUsage()
		return nil
	}

	command := os.Args[1]
	switch command {
	case "list":
		return runList(os.Args[2:])
	case "export":
		return runExport()
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		return fmt.Errorf("unknown command: %s", command)
	}
}

func printUsage() {
	fmt.Println(`opencode-session-export - Export opencode sessions to markdown

USAGE:
    opencode-session-export <command> [options]

COMMANDS:
    list [--all]            List available sessions (--all for all projects)
    export                  Export session(s) to markdown
    help                    Show this help message

LIST OPTIONS:
    --all                   List sessions from all projects, not just current repo

EXPORT OPTIONS:
    --session <id>          Export specific session by ID
    --latest                Export the most recent session
    --all                   Export all sessions
    --output <file>         Output file (default: stdout)
    --output-dir <dir>      Output directory for multiple sessions
    --project <path>        Project path (default: current directory)
    --include-costs         Include cost information in output
    --include-timings       Include timing information in output
    --include-snapshots     Include snapshot information in output
    --since <date>          Export sessions since date (YYYY-MM-DD)

EXAMPLES:
    opencode-session-export list
    opencode-session-export export --session abc123 --output session.md
    opencode-session-export export --latest --output latest.md
    opencode-session-export export --all --output-dir ./exports/
    opencode-session-export export --since 2024-01-01 --include-costs --output-dir ./exports/`)
}

func runList(args []string) error {
	// Parse list flags
	listFlags := flag.NewFlagSet("list", flag.ExitOnError)
	all := listFlags.Bool("all", false, "List sessions from all projects")
	listFlags.Parse(args)

	if *all {
		return runListAll()
	}

	// Default behavior: list sessions from current project only
	projectPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	reader, err := session.NewReader(projectPath)
	if err != nil {
		return fmt.Errorf("failed to create session reader: %w", err)
	}

	sessionIDs, err := reader.ListSessions()
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessionIDs) == 0 {
		fmt.Println("No sessions found in current project.")
		return nil
	}

	fmt.Printf("Found %d session(s) in current project:\n\n", len(sessionIDs))

	// Get session info for each session
	for _, sessionID := range sessionIDs {
		info, err := reader.ReadSessionInfo(sessionID)
		if err != nil {
			fmt.Printf("  %s (error reading info)\n", sessionID)
			continue
		}

		fmt.Printf("  %s - %s (%s)\n",
			sessionID[:8],
			info.Title,
			info.GetCreatedAt().Format("2006-01-02 15:04"))
	}

	return nil
}

func runListAll() error {
	reader, err := session.NewGlobalReader()
	if err != nil {
		return fmt.Errorf("failed to create global session reader: %w", err)
	}

	allSessions, err := reader.ListAllSessions()
	if err != nil {
		return fmt.Errorf("failed to list all sessions: %w", err)
	}

	if len(allSessions) == 0 {
		fmt.Println("No sessions found across all projects.")
		return nil
	}

	fmt.Printf("Found %d session(s) across all projects:\n\n", len(allSessions))

	// Group sessions by project
	projectSessions := make(map[string][]session.SessionWithProject)
	for _, sess := range allSessions {
		projectSessions[sess.ProjectName] = append(projectSessions[sess.ProjectName], sess)
	}

	// Display sessions grouped by project
	for projectName, sessions := range projectSessions {
		fmt.Printf("Project: %s\n", projectName)
		for _, sess := range sessions {
			fmt.Printf("  %s - %s (%s)\n",
				sess.SessionID[:8],
				sess.Info.Title,
				sess.Info.GetCreatedAt().Format("2006-01-02 15:04"))
		}
		fmt.Println()
	}

	return nil
}

func runExport() error {
	// Parse export flags
	exportFlags := flag.NewFlagSet("export", flag.ExitOnError)

	sessionID := exportFlags.String("session", "", "Session ID to export")
	latest := exportFlags.Bool("latest", false, "Export latest session")
	all := exportFlags.Bool("all", false, "Export all sessions")
	output := exportFlags.String("output", "", "Output file (default: stdout)")
	outputDir := exportFlags.String("output-dir", "", "Output directory for multiple sessions")
	projectPath := exportFlags.String("project", "", "Project path (default: current directory)")
	includeCosts := exportFlags.Bool("include-costs", false, "Include cost information")
	includeTimings := exportFlags.Bool("include-timings", false, "Include timing information")
	includeSnapshots := exportFlags.Bool("include-snapshots", false, "Include snapshot information")
	since := exportFlags.String("since", "", "Export sessions since date (YYYY-MM-DD)")

	exportFlags.Parse(os.Args[2:])

	// Determine project path
	if *projectPath == "" {
		var err error
		*projectPath, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	reader, err := session.NewReader(*projectPath)
	if err != nil {
		return fmt.Errorf("failed to create session reader: %w", err)
	}

	// Create markdown generator
	generator := markdown.NewGenerator(markdown.Options{
		IncludeCosts:     *includeCosts,
		IncludeTimings:   *includeTimings,
		IncludeSnapshots: *includeSnapshots,
	})

	// Determine which sessions to export
	var sessionsToExport []string

	if *sessionID != "" {
		sessionsToExport = []string{*sessionID}
	} else if *latest {
		sessions, err := reader.ListSessions()
		if err != nil {
			return fmt.Errorf("failed to list sessions: %w", err)
		}
		if len(sessions) == 0 {
			return fmt.Errorf("no sessions found")
		}

		// Find the latest session by reading creation times
		var latestSession string
		var latestTime time.Time

		for _, sid := range sessions {
			info, err := reader.ReadSessionInfo(sid)
			if err != nil {
				continue
			}
			if info.GetCreatedAt().After(latestTime) {
				latestTime = info.GetCreatedAt()
				latestSession = sid
			}
		}

		if latestSession == "" {
			return fmt.Errorf("no valid sessions found")
		}

		sessionsToExport = []string{latestSession}
	} else if *all {
		sessionsToExport, err = reader.ListSessions()
		if err != nil {
			return fmt.Errorf("failed to list sessions: %w", err)
		}
	} else if *since != "" {
		sinceTime, err := time.Parse("2006-01-02", *since)
		if err != nil {
			return fmt.Errorf("invalid date format (use YYYY-MM-DD): %w", err)
		}

		allSessions, err := reader.ListSessions()
		if err != nil {
			return fmt.Errorf("failed to list sessions: %w", err)
		}

		for _, sid := range allSessions {
			info, err := reader.ReadSessionInfo(sid)
			if err != nil {
				continue
			}
			if info.GetCreatedAt().After(sinceTime) {
				sessionsToExport = append(sessionsToExport, sid)
			}
		}
	} else {
		return fmt.Errorf("must specify --session, --latest, --all, or --since")
	}

	if len(sessionsToExport) == 0 {
		fmt.Println("No sessions to export.")
		return nil
	}

	// Export sessions
	if len(sessionsToExport) == 1 && *outputDir == "" {
		// Single session export
		return exportSingleSession(reader, generator, sessionsToExport[0], *output)
	} else {
		// Multiple sessions export
		if *outputDir == "" {
			*outputDir = "./exports"
		}
		return exportMultipleSessions(reader, generator, sessionsToExport, *outputDir)
	}
}

func exportSingleSession(reader *session.Reader, generator *markdown.Generator, sessionID, outputFile string) error {
	sess, err := reader.ReadSession(sessionID)
	if err != nil {
		return fmt.Errorf("failed to read session: %w", err)
	}

	markdown, err := generator.Generate(sess)
	if err != nil {
		return fmt.Errorf("failed to generate markdown: %w", err)
	}

	if outputFile == "" {
		fmt.Print(markdown)
	} else {
		if err := os.WriteFile(outputFile, []byte(markdown), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Printf("Exported session %s to %s\n", sessionID[:8], outputFile)
	}

	return nil
}

func exportMultipleSessions(reader *session.Reader, generator *markdown.Generator, sessionIDs []string, outputDir string) error {
	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	fmt.Printf("Exporting %d session(s) to %s...\n", len(sessionIDs), outputDir)

	for i, sessionID := range sessionIDs {
		sess, err := reader.ReadSession(sessionID)
		if err != nil {
			fmt.Printf("Warning: failed to read session %s: %v\n", sessionID[:8], err)
			continue
		}

		markdown, err := generator.Generate(sess)
		if err != nil {
			fmt.Printf("Warning: failed to generate markdown for session %s: %v\n", sessionID[:8], err)
			continue
		}

		// Create filename from session title and ID
		filename := fmt.Sprintf("%s_%s.md",
			sanitizeFilename(sess.Info.Title),
			sessionID[:8])

		outputFile := filepath.Join(outputDir, filename)

		if err := os.WriteFile(outputFile, []byte(markdown), 0644); err != nil {
			fmt.Printf("Warning: failed to write %s: %v\n", filename, err)
			continue
		}

		fmt.Printf("  [%d/%d] %s -> %s\n", i+1, len(sessionIDs), sessionID[:8], filename)
	}

	fmt.Printf("Export complete! Files saved to %s\n", outputDir)
	return nil
}

func sanitizeFilename(name string) string {
	// Replace invalid filename characters
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := name

	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}

	// Limit length and trim spaces
	if len(result) > 50 {
		result = result[:50]
	}

	return strings.TrimSpace(result)
}
