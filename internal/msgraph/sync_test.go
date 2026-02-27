package msgraph_test

import (
	"testing"
	"time"

	"github.com/Tiliavir/trivial-time-tracker/internal/model"
	"github.com/Tiliavir/trivial-time-tracker/internal/msgraph"
	"github.com/Tiliavir/trivial-time-tracker/internal/storage"
)

func makeEvent(id, subject, start, end string) msgraph.CalendarEvent {
	return msgraph.CalendarEvent{
		ID:          id,
		Subject:     subject,
		BodyPreview: "",
		IsAllDay:    false,
		IsCancelled: false,
		Sensitivity: "normal",
		ShowAs:      "busy",
		Start: struct {
			DateTime string `json:"dateTime"`
			TimeZone string `json:"timeZone"`
		}{DateTime: start, TimeZone: "UTC"},
		End: struct {
			DateTime string `json:"dateTime"`
			TimeZone string `json:"timeZone"`
		}{DateTime: end, TimeZone: "UTC"},
	}
}

func TestMapEventToEntry(t *testing.T) {
	event := makeEvent("ext-id-1", "Sprint Planning", "2026-02-27T09:00:00", "2026-02-27T10:30:00")
	entry, startTime, err := msgraph.MapEventToEntry(event, "UTC", "Meetings")
	if err != nil {
		t.Fatalf("MapEventToEntry: %v", err)
	}
	if entry.ExternalID != "ext-id-1" {
		t.Errorf("ExternalID = %q, want %q", entry.ExternalID, "ext-id-1")
	}
	if entry.Task == nil || *entry.Task != "Sprint Planning" {
		t.Errorf("Task = %v, want %q", entry.Task, "Sprint Planning")
	}
	if entry.Project != "Meetings" {
		t.Errorf("Project = %q, want %q", entry.Project, "Meetings")
	}
	if entry.Source != "outlook" {
		t.Errorf("Source = %q, want %q", entry.Source, "outlook")
	}
	if entry.DurationSeconds == nil || *entry.DurationSeconds != 5400 {
		t.Errorf("DurationSeconds = %v, want 5400", entry.DurationSeconds)
	}
	if !startTime.Equal(entry.Start) {
		t.Errorf("startTime mismatch")
	}
	found := false
	for _, tag := range entry.Tags {
		if tag == "outlook" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'outlook' tag")
	}
}

func TestMapEventToEntry_WithLocation(t *testing.T) {
	event := makeEvent("ext-id-2", "Standup", "2026-02-27T10:00:00", "2026-02-27T10:15:00")
	event.BodyPreview = "Daily standup"
	event.Location.DisplayName = "Zoom"

	entry, _, err := msgraph.MapEventToEntry(event, "UTC", "Meetings")
	if err != nil {
		t.Fatalf("MapEventToEntry: %v", err)
	}
	if entry.Comment == nil {
		t.Fatal("expected comment, got nil")
	}
	if *entry.Comment != "Daily standup\nZoom" {
		t.Errorf("Comment = %q, want %q", *entry.Comment, "Daily standup\nZoom")
	}
}

func TestSyncEvents_Import(t *testing.T) {
	base := t.TempDir()
	events := []msgraph.CalendarEvent{
		makeEvent("ext-1", "Architecture Board", "2026-02-27T09:00:00", "2026-02-27T10:30:00"),
	}
	opts := msgraph.SyncOptions{
		Base:    base,
		From:    time.Date(2026, 2, 27, 0, 0, 0, 0, time.UTC),
		To:      time.Date(2026, 2, 27, 23, 59, 59, 0, time.UTC),
		DryRun:  false,
		Project: "Meetings",
	}

	result, err := msgraph.SyncEvents(events, opts, "UTC")
	if err != nil {
		t.Fatalf("SyncEvents: %v", err)
	}
	if result.Imported != 1 {
		t.Errorf("Imported = %d, want 1", result.Imported)
	}
	if result.Skipped != 0 {
		t.Errorf("Skipped = %d, want 0", result.Skipped)
	}

	// Verify persisted.
	day := time.Date(2026, 2, 27, 0, 0, 0, 0, time.UTC)
	df, err := storage.LoadDay(base, day)
	if err != nil {
		t.Fatalf("LoadDay: %v", err)
	}
	if len(df.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(df.Entries))
	}
	if df.Entries[0].ExternalID != "ext-1" {
		t.Errorf("ExternalID = %q, want %q", df.Entries[0].ExternalID, "ext-1")
	}
}

func TestSyncEvents_Idempotent(t *testing.T) {
	base := t.TempDir()
	events := []msgraph.CalendarEvent{
		makeEvent("ext-1", "Architecture Board", "2026-02-27T09:00:00", "2026-02-27T10:30:00"),
	}
	opts := msgraph.SyncOptions{
		Base:    base,
		From:    time.Date(2026, 2, 27, 0, 0, 0, 0, time.UTC),
		To:      time.Date(2026, 2, 27, 23, 59, 59, 0, time.UTC),
		DryRun:  false,
		Project: "Meetings",
	}

	// First sync.
	r1, err := msgraph.SyncEvents(events, opts, "UTC")
	if err != nil {
		t.Fatalf("first SyncEvents: %v", err)
	}
	if r1.Imported != 1 {
		t.Errorf("first sync: Imported = %d, want 1", r1.Imported)
	}

	// Second sync — must not duplicate.
	r2, err := msgraph.SyncEvents(events, opts, "UTC")
	if err != nil {
		t.Fatalf("second SyncEvents: %v", err)
	}
	if r2.Imported != 0 {
		t.Errorf("second sync: Imported = %d, want 0 (idempotent)", r2.Imported)
	}
	if r2.Skipped != 1 {
		t.Errorf("second sync: Skipped = %d, want 1", r2.Skipped)
	}

	// Verify only one entry stored.
	day := time.Date(2026, 2, 27, 0, 0, 0, 0, time.UTC)
	df, err := storage.LoadDay(base, day)
	if err != nil {
		t.Fatalf("LoadDay: %v", err)
	}
	if len(df.Entries) != 1 {
		t.Fatalf("entries = %d after 2 syncs, want 1", len(df.Entries))
	}
}

func TestSyncEvents_Update(t *testing.T) {
	base := t.TempDir()
	event := makeEvent("ext-1", "Architecture Board", "2026-02-27T09:00:00", "2026-02-27T10:30:00")
	opts := msgraph.SyncOptions{
		Base:    base,
		From:    time.Date(2026, 2, 27, 0, 0, 0, 0, time.UTC),
		To:      time.Date(2026, 2, 27, 23, 59, 59, 0, time.UTC),
		DryRun:  false,
		Project: "Meetings",
	}

	// First sync.
	_, err := msgraph.SyncEvents([]msgraph.CalendarEvent{event}, opts, "UTC")
	if err != nil {
		t.Fatalf("first SyncEvents: %v", err)
	}

	// Modify the event subject.
	event.Subject = "Architecture Board (updated)"

	r2, err := msgraph.SyncEvents([]msgraph.CalendarEvent{event}, opts, "UTC")
	if err != nil {
		t.Fatalf("second SyncEvents: %v", err)
	}
	if r2.Updated != 1 {
		t.Errorf("Updated = %d, want 1", r2.Updated)
	}

	// Verify only one entry.
	day := time.Date(2026, 2, 27, 0, 0, 0, 0, time.UTC)
	df, err := storage.LoadDay(base, day)
	if err != nil {
		t.Fatalf("LoadDay: %v", err)
	}
	if len(df.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(df.Entries))
	}
	if df.Entries[0].Task == nil || *df.Entries[0].Task != "Architecture Board (updated)" {
		t.Errorf("Task = %v, want updated", df.Entries[0].Task)
	}
}

func TestSyncEvents_SkipFiltered(t *testing.T) {
	base := t.TempDir()
	opts := msgraph.SyncOptions{
		Base:    base,
		Project: "Meetings",
	}

	tests := []struct {
		name  string
		event msgraph.CalendarEvent
	}{
		{
			name: "cancelled",
			event: func() msgraph.CalendarEvent {
				e := makeEvent("c1", "Cancelled", "2026-02-27T09:00:00", "2026-02-27T10:00:00")
				e.IsCancelled = true
				return e
			}(),
		},
		{
			name: "all-day",
			event: func() msgraph.CalendarEvent {
				e := makeEvent("c2", "All Day", "2026-02-27T00:00:00", "2026-02-28T00:00:00")
				e.IsAllDay = true
				return e
			}(),
		},
		{
			name: "private",
			event: func() msgraph.CalendarEvent {
				e := makeEvent("c3", "Private", "2026-02-27T09:00:00", "2026-02-27T10:00:00")
				e.Sensitivity = "private"
				return e
			}(),
		},
		{
			name: "free",
			event: func() msgraph.CalendarEvent {
				e := makeEvent("c4", "Free Block", "2026-02-27T09:00:00", "2026-02-27T10:00:00")
				e.ShowAs = "free"
				return e
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := msgraph.SyncEvents([]msgraph.CalendarEvent{tt.event}, opts, "UTC")
			if err != nil {
				t.Fatalf("SyncEvents: %v", err)
			}
			if r.Imported != 0 {
				t.Errorf("expected 0 imported for %s event, got %d", tt.name, r.Imported)
			}
		})
	}
}

func TestSyncEvents_DryRun(t *testing.T) {
	base := t.TempDir()
	events := []msgraph.CalendarEvent{
		makeEvent("ext-dry", "Dry Run Event", "2026-02-27T09:00:00", "2026-02-27T10:00:00"),
	}
	opts := msgraph.SyncOptions{
		Base:    base,
		Project: "Meetings",
		DryRun:  true,
	}

	result, err := msgraph.SyncEvents(events, opts, "UTC")
	if err != nil {
		t.Fatalf("SyncEvents dry-run: %v", err)
	}
	if result.Imported != 1 {
		t.Errorf("dry-run Imported = %d, want 1", result.Imported)
	}

	// Nothing should be persisted.
	day := time.Date(2026, 2, 27, 0, 0, 0, 0, time.UTC)
	df, err := storage.LoadDay(base, day)
	if err != nil {
		t.Fatalf("LoadDay: %v", err)
	}
	if len(df.Entries) != 0 {
		t.Errorf("dry-run wrote %d entries, want 0", len(df.Entries))
	}
}

func TestSyncEvents_ExternalIDPreservesManualEntries(t *testing.T) {
	base := t.TempDir()
	day := time.Date(2026, 2, 27, 9, 0, 0, 0, time.UTC)
	end := day.Add(time.Hour)
	dur := int64(3600)

	// Pre-existing manual entry on the same day.
	manual := model.Entry{
		ID:              "manual-1",
		Project:         "Work",
		Tags:            []string{},
		Start:           day,
		End:             &end,
		DurationSeconds: &dur,
		Source:          "manual",
	}
	if err := storage.UpdateEntry(base, day, manual); err != nil {
		t.Fatalf("inserting manual entry: %v", err)
	}

	events := []msgraph.CalendarEvent{
		makeEvent("ext-1", "Meeting", "2026-02-27T11:00:00", "2026-02-27T12:00:00"),
	}
	opts := msgraph.SyncOptions{
		Base:    base,
		Project: "Meetings",
		DryRun:  false,
	}

	_, err := msgraph.SyncEvents(events, opts, "UTC")
	if err != nil {
		t.Fatalf("SyncEvents: %v", err)
	}

	df, err := storage.LoadDay(base, day)
	if err != nil {
		t.Fatalf("LoadDay: %v", err)
	}
	if len(df.Entries) != 2 {
		t.Fatalf("entries = %d, want 2 (manual + imported)", len(df.Entries))
	}

	// Verify manual entry is untouched.
	var found bool
	for _, e := range df.Entries {
		if e.ID == "manual-1" {
			found = true
			if e.Source != "manual" {
				t.Errorf("manual entry source changed to %q", e.Source)
			}
		}
	}
	if !found {
		t.Error("manual entry not found after sync")
	}
}
