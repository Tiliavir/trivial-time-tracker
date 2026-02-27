package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Tiliavir/trivial-time-tracker/internal/model"
)

// BaseDir returns the root data directory (~/.ttt).
func BaseDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".ttt"), nil
}

// dayFilePath returns the path for the given date's JSON file.
func dayFilePath(base string, t time.Time) string {
	return filepath.Join(base, t.Format("2006"), t.Format("01"), t.Format("02")+".json")
}

// LoadDay loads the DayFile for the given date. Returns an empty DayFile if not found.
func LoadDay(base string, t time.Time) (model.DayFile, error) {
	path := dayFilePath(base, t)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return model.DayFile{Date: t.Format("2006-01-02"), Entries: []model.Entry{}}, nil
	}
	if err != nil {
		return model.DayFile{}, fmt.Errorf("storage error reading %s: %w", path, err)
	}

	var df model.DayFile
	if err := json.Unmarshal(data, &df); err != nil {
		// Back up corrupt file and abort.
		backupPath := path + ".corrupt"
		_ = os.Rename(path, backupPath)
		return model.DayFile{}, fmt.Errorf("corrupt JSON in %s (backed up to %s): %w", path, backupPath, err)
	}
	return df, nil
}

// SaveDay atomically writes a DayFile for the given date.
func SaveDay(base string, t time.Time, df model.DayFile) error {
	path := dayFilePath(base, t)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("storage error creating directories: %w", err)
	}

	data, err := json.MarshalIndent(df, "", "  ")
	if err != nil {
		return fmt.Errorf("storage error marshalling JSON: %w", err)
	}

	// Atomic write: write to temp file then rename.
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("storage error writing temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("storage error renaming temp file: %w", err)
	}
	return nil
}

// FindActiveEntry searches all day files (most recent first) for an entry with end == nil.
// It returns the entry, the date it was found on, and an error if any.
func FindActiveEntry(base string) (*model.Entry, time.Time, error) {
	// Check today and the past few days to handle crash-recovery across midnight.
	now := time.Now()
	for i := 0; i < 7; i++ {
		day := now.AddDate(0, 0, -i)
		df, err := LoadDay(base, day)
		if err != nil {
			return nil, time.Time{}, err
		}
		for j := len(df.Entries) - 1; j >= 0; j-- {
			if df.Entries[j].End == nil {
				return &df.Entries[j], day, nil
		}
		}
	}
	return nil, time.Time{}, nil
}

// UpdateEntry replaces or appends an entry in the DayFile for the given date.
func UpdateEntry(base string, day time.Time, entry model.Entry) error {
	df, err := LoadDay(base, day)
	if err != nil {
		return err
	}
	for i, e := range df.Entries {
		if e.ID == entry.ID {
			df.Entries[i] = entry
			return SaveDay(base, day, df)
		}
	}
	df.Entries = append(df.Entries, entry)
	return SaveDay(base, day, df)
}

// LoadRange loads all entries in [from, to] inclusive.
func LoadRange(base string, from, to time.Time) ([]model.Entry, error) {
	var entries []model.Entry
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		df, err := LoadDay(base, d)
		if err != nil {
			return nil, err
		}
		entries = append(entries, df.Entries...)
	}
	return entries, nil
}
