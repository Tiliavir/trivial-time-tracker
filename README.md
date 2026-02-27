# trivial-time-tracker

[![CI](https://github.com/Tiliavir/trivial-time-tracker/actions/workflows/ci.yml/badge.svg)](https://github.com/Tiliavir/trivial-time-tracker/actions/workflows/ci.yml)
[![CodeQL](https://github.com/Tiliavir/trivial-time-tracker/actions/workflows/codeql.yml/badge.svg)](https://github.com/Tiliavir/trivial-time-tracker/actions/workflows/codeql.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

**ttt** – Trivial Time Tracker: a single-binary, file-based command-line time tracker.

**[▶ Try the interactive demo](https://tiliavir.github.io/trivial-time-tracker/)**

## Features

- Single static binary, no installation required
- File-based storage only (`~/.ttt/`) — no database
- No background daemon required
- Works offline
- Human-readable JSON storage
- Outlook calendar sync via Microsoft Graph API

## Installation

```bash
go install github.com/Tiliavir/trivial-time-tracker@latest
```

Or build from source:

```bash
go build -o ttt .
```

## Usage

```bash
# Start a timer
ttt start ECM --task "REST refactor" --comment "Investigating mapping issue" --tags backend,api

# Check current status
ttt status

# Stop the active timer
ttt stop --comment "Wrapped up the fix"

# List entries
ttt list --today
ttt list --week

# Weekly report
ttt report --week
ttt report --week --format csv
ttt report --week --format json

# Export data to stdout
ttt export --format csv
ttt export --format json
ttt export --format md

# Sync Outlook calendar events (today by default)
ttt outlook sync
ttt outlook sync --date 2026-02-27
ttt outlook sync --from 2026-02-20 --to 2026-02-27
ttt outlook sync --dry-run
ttt outlook sync --project Meetings --timezone Europe/Berlin
```

## Outlook Sync

`ttt outlook sync` imports Outlook calendar events into local ttt entries using the Microsoft Graph API.

**First run** — a device code authentication prompt is shown:

```
To sign in, use a web browser to open the page:
  https://microsoft.com/devicelogin
Enter the code: ABCD-1234
```

Tokens are stored in `~/.ttt/auth/msgraph_tokens.json` and refreshed automatically.

**Example output:**

```text
Syncing Outlook events (2026-02-27 → 2026-02-27)...

  ✓ Imported: Architecture Board (1h 30m)
  ✓ Imported: 1:1 with CTO (30m)
  – Skipped:  Weekly Sync (already exists)

Summary:
  2 imported
  1 skipped
  0 updated
```

Re-running sync is safe — existing entries are detected by `external_id` and skipped (or updated if changed).

## Storage Layout

```
~/.ttt/
    2026/
        02/
            27.json
            28.json
    auth/
        msgraph_tokens.json
```

Each daily file contains JSON entries:

```json
{
  "date": "2026-02-27",
  "entries": [
    {
      "id": "20260227-083210-x82ks",
      "external_id": "AAMkAGI2T...",
      "project": "ECM",
      "task": "REST refactor",
      "comment": "Investigating mapping issue",
      "tags": [],
      "start": "2026-02-27T08:32:10+01:00",
      "end": "2026-02-27T10:12:00+01:00",
      "duration_seconds": 5990,
      "source": "manual"
    }
  ]
}
```

## Exit Codes

| Code | Meaning      |
|------|--------------|
| `0`  | Success      |
| `1`  | User error   |
| `2`  | Storage error|

## Development

```bash
# Build
go build -o ttt .

# Run tests
go test ./...

# Vet
go vet ./...
```

## Contributing

Pull requests are welcome. Please open an issue first to discuss what you would like to change.
