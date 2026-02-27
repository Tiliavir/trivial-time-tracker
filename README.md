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
- Outlook calendar sync via Microsoft Graph API (device code flow, no daemon)

## Installation

Install directly as `ttt`:

```bash
go install github.com/Tiliavir/trivial-time-tracker/cmd/ttt@latest
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

## Configuration

On the first run ttt creates `~/.ttt/config.json` with annotated defaults:

```jsonc
// ttt configuration – ~/.ttt/config.json
{
  // ── Microsoft Graph / Outlook calendar sync ──────────────────────────────
  "outlook": {
    // Azure AD tenant ID.
    // • "common"  – personal Microsoft accounts and any organisation (default)
    // • Your organisation's tenant GUID, e.g. "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
    "tenant_id": "common",

    // Azure application (client) ID used for the OAuth2 device code flow.
    // The built-in value is the public Azure CLI app – no app registration needed.
    // Replace with your own Azure app registration for single-tenant deployments.
    "client_id": "04b07795-8542-4c4a-95af-30b2c573d5ab",

    // Default project name assigned to imported Outlook calendar events.
    // Can be overridden per-sync with: ttt outlook sync --project <name>
    "default_project": "Meetings",

    // IANA timezone for interpreting calendar event times, e.g. "Europe/Berlin".
    // Leave empty to use UTC. Can be overridden with: ttt outlook sync --timezone <tz>
    "timezone": ""
  }
}
```

The file supports `//` line comments. All fields are optional — built-in defaults apply for any omitted field.

### Config field reference

| Field | Default | Description |
|---|---|---|
| `outlook.tenant_id` | `"common"` | Azure AD tenant ID. Use `"common"` or your org's GUID. |
| `outlook.client_id` | *(Azure CLI app)* | Azure app client ID for OAuth2 device code flow. |
| `outlook.default_project` | `"Meetings"` | Project assigned to imported calendar events. |
| `outlook.timezone` | `""` (UTC) | IANA timezone for event times, e.g. `"Europe/Berlin"`. |

## Outlook Sync

`ttt outlook sync` imports Outlook calendar events into local ttt entries using the Microsoft Graph API.

**Authentication — first run** — a device code prompt is shown:

```
To sign in, use a web browser to open the page:
  https://microsoft.com/devicelogin
Enter the code: ABCD-1234
```

Tokens are stored in `~/.ttt/auth/msgraph_tokens.json` and refreshed automatically. Re-authentication is only required when the refresh token expires.

**Event filtering** — the following events are skipped:

- Cancelled (`isCancelled: true`)
- All-day events
- Private events (`sensitivity: private`)
- Free slots (`showAs: free`)

**Example output:**

```text
Syncing Outlook events (2026-02-27 → 2026-02-27)...

  ✓ Imported: Architecture Board (1h 30m)
  ✓ Imported: 1:1 with CTO (30m)
  – Skipped:  Weekly Sync (already exists)
  ↑ Updated:  Design Review (30m → 1h)

Summary:
  2 imported
  1 skipped
  1 updated
```

Re-running sync is idempotent — existing entries are matched by `external_id` and skipped (or updated if the subject or time window changed). Manual entries are never touched.

**Flag reference:**

| Flag | Description |
|---|---|
| `--date YYYY-MM-DD` | Sync a single day |
| `--from YYYY-MM-DD` | Range start (required when `--to` is set) |
| `--to YYYY-MM-DD` | Range end (defaults to today) |
| `--today` | Sync today (default when no date flag is given) |
| `--dry-run` | Print planned operations without writing |
| `--project NAME` | Override default project (falls back to config) |
| `--timezone TZ` | Override IANA timezone (falls back to config) |

## Storage Layout

```
~/.ttt/
    config.json          ← created on first run with annotated defaults
    2026/
        02/
            27.json
            28.json
    auth/
        msgraph_tokens.json   ← OAuth2 tokens (mode 0600)
```

Each daily file contains JSON entries:

```json
{
  "date": "2026-02-27",
  "entries": [
    {
      "id": "20260227-083210-x82ks",
      "external_id": "AAMkAGI2T...",
      "project": "Meetings",
      "task": "Architecture Board",
      "comment": "Zoom\nQuarterly roadmap discussion",
      "tags": ["outlook"],
      "start": "2026-02-27T09:00:00+01:00",
      "end": "2026-02-27T10:30:00+01:00",
      "duration_seconds": 5400,
      "source": "outlook"
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

