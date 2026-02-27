package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config is the root configuration for ttt, stored in ~/.ttt/config.json.
// The file supports single-line // comments for documentation purposes.
type Config struct {
	Outlook OutlookConfig `json:"outlook"`
}

// OutlookConfig holds Microsoft Graph / Outlook calendar sync settings.
type OutlookConfig struct {
	// TenantID is the Azure AD tenant. Use "common" for personal/multi-tenant accounts.
	TenantID string `json:"tenant_id"`
	// ClientID is the Azure app (client) ID for the OAuth2 device code flow.
	ClientID string `json:"client_id"`
	// DefaultProject is the project name assigned to imported Outlook events.
	DefaultProject string `json:"default_project"`
	// Timezone is the IANA timezone for event times (e.g. "Europe/Berlin"). Empty = UTC.
	Timezone string `json:"timezone"`
}

const (
	// DefaultTenantID is the Microsoft "common" tenant (supports personal and
	// multi-tenant organisational accounts without additional registration).
	DefaultTenantID = "common"
	// DefaultClientID is the well-known public Azure CLI app ID.
	// It supports device code flow without a client secret and requires no
	// app registration. Replace with your own registered app ID for
	// organisational or production deployments.
	DefaultClientID = "04b07795-8542-4c4a-95af-30b2c573d5ab"
	// DefaultProject is the project name used when none is specified.
	DefaultProject = "Meetings"
)

// defaultConfig returns a Config pre-filled with sensible defaults.
func defaultConfig() Config {
	return Config{
		Outlook: OutlookConfig{
			TenantID:       DefaultTenantID,
			ClientID:       DefaultClientID,
			DefaultProject: DefaultProject,
			Timezone:       "",
		},
	}
}

// configTemplate is the annotated config written on first run.
// Lines whose trimmed content starts with // are stripped before JSON parsing,
// allowing human-readable documentation inside the file.
const configTemplate = `// ttt configuration – ~/.ttt/config.json
//
// All settings are optional; the built-in defaults shown below work out of
// the box for personal Microsoft accounts and most organisations.
// Edit this file to customise ttt behaviour.
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
`

// configFilePath returns the path to ~/.ttt/config.json.
func configFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".ttt", "config.json"), nil
}

// stripLineComments removes lines whose leading non-whitespace content starts
// with //. Only full-line comments are handled; inline comments are not stripped.
func stripLineComments(data []byte) []byte {
	var out []byte
	for _, line := range bytes.Split(data, []byte("\n")) {
		if bytes.HasPrefix(bytes.TrimLeft(line, " \t"), []byte("//")) {
			continue
		}
		out = append(out, line...)
		out = append(out, '\n')
	}
	return out
}

// Load reads ~/.ttt/config.json, creating it with annotated defaults on first
// run. Lines starting with // are treated as comments and stripped before
// JSON parsing.
func Load() (Config, error) {
	path, err := configFilePath()
	if err != nil {
		return defaultConfig(), err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		// First run: write the annotated template so users can discover options.
		if writeErr := writeDefault(path); writeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not create config file %s: %v\n", path, writeErr)
		}
		return defaultConfig(), nil
	}
	if err != nil {
		return defaultConfig(), fmt.Errorf("reading config file %s: %w", path, err)
	}

	cleaned := stripLineComments(data)
	var cfg Config
	if err := json.Unmarshal(cleaned, &cfg); err != nil {
		return defaultConfig(), fmt.Errorf("parsing config file %s: %w\nTip: delete the file to regenerate defaults", path, err)
	}

	// Fill zero-value fields with built-in defaults so callers always get
	// a usable Config even if the user only partially fills in the file.
	if cfg.Outlook.TenantID == "" {
		cfg.Outlook.TenantID = DefaultTenantID
	}
	if cfg.Outlook.ClientID == "" {
		cfg.Outlook.ClientID = DefaultClientID
	}
	if cfg.Outlook.DefaultProject == "" {
		cfg.Outlook.DefaultProject = DefaultProject
	}

	return cfg, nil
}

// writeDefault creates the config directory and writes the annotated default
// config template.
func writeDefault(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(configTemplate), 0o600); err != nil {
		return fmt.Errorf("writing default config: %w", err)
	}
	return nil
}
