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
type Config struct{}

// defaultConfig returns an empty Config.
func defaultConfig() Config {
	return Config{}
}

// configTemplate is the annotated config written on first run.
// Lines whose trimmed content starts with // are stripped before JSON parsing,
// allowing human-readable documentation inside the file.
const configTemplate = `// ttt configuration – ~/.ttt/config.json
{}
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
