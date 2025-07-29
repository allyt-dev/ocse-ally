# ocse - opencode sessions exporter

Export your opencode sessions to clean, readable markdown files. Works on macOS, Linux, and Windows with no dependencies.

## Installation

Download the binary for your platform from [releases](https://github.com/sst/opencode-session-export/releases) or build from source:

```bash
git clone https://github.com/sst/opencode-session-export.git
cd opencode-session-export
make build
```

## Usage

```bash
# List available sessions
ocse list

# Export specific session
ocse export --session abc12345 --output session.md

# Export latest session
ocse export --latest --output latest.md

# Export all sessions
ocse export --all --output-dir ./exports/

# Export with costs and timing
ocse export --all --include-costs --include-timings --output-dir ./exports/
```

The exported markdown includes session metadata, conversation flow, tool executions with inputs/outputs, file attachments, and optional cost/timing information.

MIT License
