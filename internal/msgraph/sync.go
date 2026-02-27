package msgraph

import (
	"fmt"
	"strings"
	"time"

	"github.com/Tiliavir/trivial-time-tracker/internal/model"
	"github.com/Tiliavir/trivial-time-tracker/internal/storage"
	"github.com/Tiliavir/trivial-time-tracker/internal/timecalc"
)

// SyncResult holds counters for a sync operation.
type SyncResult struct {
	Imported int
	Skipped  int
	Updated  int
	Errors   int
}

// SyncOptions configures a sync run.
type SyncOptions struct {
	Base    string
	From    time.Time
	To      time.Time
	DryRun  bool
	Project string
}

// parseGraphTime parses a Graph API dateTime string in the given timezone.
// Graph returns times like "2026-02-27T09:00:00.0000000" without a zone suffix
// when a Prefer: outlook.timezone header is set.
func parseGraphTime(dt, tz string) (time.Time, error) {
	// Try RFC3339 first (includes timezone offset).
	if t, err := time.Parse(time.RFC3339, dt); err == nil {
		return t, nil
	}
	// Try RFC3339Nano.
	if t, err := time.Parse(time.RFC3339Nano, dt); err == nil {
		return t, nil
	}

	loc := time.UTC
	if tz != "" {
		if l, err := time.LoadLocation(tz); err == nil {
			loc = l
		}
	}

	// Graph returns fractional seconds: "2026-02-27T09:00:00.0000000"
	for _, layout := range []string{
		"2006-01-02T15:04:05.0000000",
		"2006-01-02T15:04:05",
	} {
		if t, err := time.ParseInLocation(layout, dt, loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse graph time %q", dt)
}

// buildComment combines bodyPreview and location into a comment string.
func buildComment(event CalendarEvent) *string {
	parts := []string{}
	if event.BodyPreview != "" {
		parts = append(parts, event.BodyPreview)
	}
	if event.Location.DisplayName != "" {
		parts = append(parts, event.Location.DisplayName)
	}
	if len(parts) == 0 {
		return nil
	}
	s := strings.Join(parts, "\n")
	return &s
}

// shouldSkip returns true if the event should not be imported.
func shouldSkip(event CalendarEvent) bool {
	if event.IsCancelled {
		return true
	}
	if event.IsAllDay {
		return true
	}
	if event.Sensitivity == "private" {
		return true
	}
	if event.ShowAs == "free" {
		return true
	}
	if event.Start.DateTime == "" || event.End.DateTime == "" {
		return true
	}
	return false
}

// MapEventToEntry converts a Graph CalendarEvent into a ttt Entry.
func MapEventToEntry(event CalendarEvent, timezone, project string) (model.Entry, time.Time, error) {
	startTime, err := parseGraphTime(event.Start.DateTime, timezone)
	if err != nil {
		return model.Entry{}, time.Time{}, fmt.Errorf("parsing start time: %w", err)
	}
	endTime, err := parseGraphTime(event.End.DateTime, timezone)
	if err != nil {
		return model.Entry{}, time.Time{}, fmt.Errorf("parsing end time: %w", err)
	}

	dur := int64(endTime.Sub(startTime).Seconds())
	subject := event.Subject
	comment := buildComment(event)

	entry := model.Entry{
		ID:              timecalc.GenerateID(startTime),
		ExternalID:      event.ID,
		Project:         project,
		Task:            &subject,
		Comment:         comment,
		Tags:            []string{"outlook"},
		Start:           startTime,
		End:             &endTime,
		DurationSeconds: &dur,
		Source:          "outlook",
	}
	return entry, startTime, nil
}

// findByExternalID searches loaded entries for one with the given external_id.
func findByExternalID(entries []model.Entry, externalID string) *model.Entry {
	for i := range entries {
		if entries[i].ExternalID == externalID {
			return &entries[i]
		}
	}
	return nil
}

// SyncEvents processes a slice of Graph events and persists them to storage.
// It prints progress to stdout and returns a SyncResult.
func SyncEvents(events []CalendarEvent, opts SyncOptions, timezone string) (SyncResult, error) {
	var result SyncResult

	for _, event := range events {
		skip := shouldSkip(event)
		if skip {
			continue
		}

		entry, startTime, err := MapEventToEntry(event, timezone, opts.Project)
		if err != nil {
			fmt.Printf("  ! Error mapping event %q: %v\n", event.Subject, err)
			result.Errors++
			continue
		}

		// Load the day file to check for existing entry by external_id.
		existing, loadErr := storage.LoadDay(opts.Base, startTime)
		if loadErr != nil {
			fmt.Printf("  ! Error loading day for %q: %v\n", event.Subject, loadErr)
			result.Errors++
			continue
		}

		found := findByExternalID(existing.Entries, event.ID)
		if found != nil {
			// Already exists — check if it needs updating.
			if found.Task != nil && entry.Task != nil && *found.Task == *entry.Task &&
				found.Start.Equal(entry.Start) && found.End != nil && entry.End != nil && found.End.Equal(*entry.End) {
				fmt.Printf("  – Skipped:  %s (already exists)\n", event.Subject)
				result.Skipped++
				continue
			}
			// Update: preserve the original ID but update the content.
			entry.ID = found.ID
			if !opts.DryRun {
				if err := storage.UpdateEntry(opts.Base, startTime, entry); err != nil {
					fmt.Printf("  ! Error updating %q: %v\n", event.Subject, err)
					result.Errors++
					continue
				}
			}
			dur := ""
			if entry.DurationSeconds != nil {
				dur = fmt.Sprintf(" (%s)", timecalc.FormatDuration(*entry.DurationSeconds))
			}
			fmt.Printf("  ↑ Updated:  %s%s\n", event.Subject, dur)
			result.Updated++
			continue
		}

		// New entry.
		if !opts.DryRun {
			if err := storage.UpdateEntry(opts.Base, startTime, entry); err != nil {
				fmt.Printf("  ! Error saving %q: %v\n", event.Subject, err)
				result.Errors++
				continue
			}
		}
		dur := ""
		if entry.DurationSeconds != nil {
			dur = fmt.Sprintf(" (%s)", timecalc.FormatDuration(*entry.DurationSeconds))
		}
		fmt.Printf("  ✓ Imported: %s%s\n", event.Subject, dur)
		result.Imported++
	}

	return result, nil
}
