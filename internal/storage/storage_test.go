package storage_test

import (
	"os"
	"testing"
	"time"

	"github.com/Tiliavir/trivial-time-tracker/internal/model"
	"github.com/Tiliavir/trivial-time-tracker/internal/storage"
)

func TestLoadDayNotExist(t *testing.T) {
	base := t.TempDir()
	day := time.Date(2026, 2, 27, 0, 0, 0, 0, time.UTC)
	df, err := storage.LoadDay(base, day)
	if err != nil {
		t.Fatalf("LoadDay on missing file: %v", err)
	}
	if df.Date != "2026-02-27" {
		t.Errorf("LoadDay date = %q, want %q", df.Date, "2026-02-27")
	}
	if len(df.Entries) != 0 {
		t.Errorf("LoadDay entries = %d, want 0", len(df.Entries))
	}
}

func TestSaveDayAndLoadDay(t *testing.T) {
	base := t.TempDir()
	day := time.Date(2026, 2, 27, 0, 0, 0, 0, time.UTC)

	project := "ECM"
	df := model.DayFile{
		Date: "2026-02-27",
		Entries: []model.Entry{
			{
				ID:      "test-id-1",
				Project: project,
				Tags:    []string{},
				Start:   day,
				Source:  "manual",
			},
		},
	}

	if err := storage.SaveDay(base, day, df); err != nil {
		t.Fatalf("SaveDay: %v", err)
	}

	loaded, err := storage.LoadDay(base, day)
	if err != nil {
		t.Fatalf("LoadDay after save: %v", err)
	}
	if len(loaded.Entries) != 1 {
		t.Fatalf("LoadDay entries = %d, want 1", len(loaded.Entries))
	}
	if loaded.Entries[0].Project != project {
		t.Errorf("LoadDay project = %q, want %q", loaded.Entries[0].Project, project)
	}
}

func TestSaveDayAtomicOnCorruptTmp(t *testing.T) {
	// Verify that a corrupt JSON file is backed up and returns an error.
	base := t.TempDir()
	day := time.Date(2026, 2, 27, 0, 0, 0, 0, time.UTC)

	// Write corrupt JSON directly to the path.
	path := base + "/2026/02/27.json"
	if err := os.MkdirAll(base+"/2026/02", 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{bad json"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := storage.LoadDay(base, day)
	if err == nil {
		t.Fatal("expected error for corrupt JSON, got nil")
	}

	// Backup file should exist.
	if _, err2 := os.Stat(path + ".corrupt"); os.IsNotExist(err2) {
		t.Error("expected backup file to exist after corrupt JSON")
	}
}

func TestUpdateEntry(t *testing.T) {
	base := t.TempDir()
	day := time.Date(2026, 2, 27, 0, 0, 0, 0, time.UTC)

	entry := model.Entry{
		ID:      "e1",
		Project: "P1",
		Tags:    []string{},
		Start:   day,
		Source:  "manual",
	}

	if err := storage.UpdateEntry(base, day, entry); err != nil {
		t.Fatalf("UpdateEntry (insert): %v", err)
	}

	// Update the same entry.
	task := "updated task"
	entry.Task = &task
	if err := storage.UpdateEntry(base, day, entry); err != nil {
		t.Fatalf("UpdateEntry (update): %v", err)
	}

	df, err := storage.LoadDay(base, day)
	if err != nil {
		t.Fatalf("LoadDay: %v", err)
	}
	if len(df.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(df.Entries))
	}
	if df.Entries[0].Task == nil || *df.Entries[0].Task != task {
		t.Errorf("task = %v, want %q", df.Entries[0].Task, task)
	}
}

func TestFindActiveEntry(t *testing.T) {
	base := t.TempDir()
	day := time.Now()

	// No entries — expect nil.
	active, _, err := storage.FindActiveEntry(base)
	if err != nil {
		t.Fatal(err)
	}
	if active != nil {
		t.Fatal("expected no active entry on empty storage")
	}

	// Add an open entry.
	entry := model.Entry{
		ID:      "active-1",
		Project: "Test",
		Tags:    []string{},
		Start:   day,
		Source:  "manual",
	}
	if err := storage.UpdateEntry(base, day, entry); err != nil {
		t.Fatal(err)
	}

	active, _, err = storage.FindActiveEntry(base)
	if err != nil {
		t.Fatal(err)
	}
	if active == nil {
		t.Fatal("expected active entry, got nil")
	}
	if active.ID != "active-1" {
		t.Errorf("active ID = %q, want %q", active.ID, "active-1")
	}
}
