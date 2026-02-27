# trivial-time-tracker

**ttt** – Trivial Time Tracker: a single-binary, file-based command-line time tracker.

## Features

- Single static binary, no installation required
- File-based storage only (`~/.ttt/`) — no database
- No background daemon required
- Works offline
- Human-readable JSON storage

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
```

## Storage Layout

```
~/.ttt/
    2026/
        02/
            27.json
            28.json
```

Each daily file contains JSON entries:

```json
{
  "date": "2026-02-27",
  "entries": [
    {
      "id": "20260227-083210-x82ks",
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
