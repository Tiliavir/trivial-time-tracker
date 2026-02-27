package timecalc_test

import (
	"testing"
	"time"

	"github.com/Tiliavir/trivial-time-tracker/internal/timecalc"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds int64
		want    string
	}{
		{0, "0s"},
		{45, "45s"},
		{60, "1m"},
		{90, "1m"},
		{3600, "1h 0m"},
		{3661, "1h 1m"},
		{5400, "1h 30m"},
	}
	for _, tt := range tests {
		got := timecalc.FormatDuration(tt.seconds)
		if got != tt.want {
			t.Errorf("FormatDuration(%d) = %q, want %q", tt.seconds, got, tt.want)
		}
	}
}

func TestFormatDurationHHMMSS(t *testing.T) {
	tests := []struct {
		seconds int64
		want    string
	}{
		{0, "00:00:00"},
		{61, "00:01:01"},
		{3661, "01:01:01"},
	}
	for _, tt := range tests {
		got := timecalc.FormatDurationHHMMSS(tt.seconds)
		if got != tt.want {
			t.Errorf("FormatDurationHHMMSS(%d) = %q, want %q", tt.seconds, got, tt.want)
		}
	}
}

func TestWeekRange(t *testing.T) {
	// 2026-02-27 is a Friday (week 9).
	fri := time.Date(2026, 2, 27, 10, 0, 0, 0, time.UTC)
	monday, sunday := timecalc.WeekRange(fri)

	wantMonday := time.Date(2026, 2, 23, 0, 0, 0, 0, time.UTC)
	wantSunday := time.Date(2026, 3, 1, 23, 59, 59, 0, time.UTC)

	if !monday.Equal(wantMonday) {
		t.Errorf("WeekRange monday = %v, want %v", monday, wantMonday)
	}
	if !sunday.Equal(wantSunday) {
		t.Errorf("WeekRange sunday = %v, want %v", sunday, wantSunday)
	}
}

func TestISOWeekLabel(t *testing.T) {
	fri := time.Date(2026, 2, 27, 10, 0, 0, 0, time.UTC)
	got := timecalc.ISOWeekLabel(fri)
	if got != "2026-W09" {
		t.Errorf("ISOWeekLabel = %q, want %q", got, "2026-W09")
	}
}

func TestSameDay(t *testing.T) {
	a := time.Date(2026, 2, 27, 10, 0, 0, 0, time.UTC)
	b := time.Date(2026, 2, 27, 23, 59, 59, 0, time.UTC)
	c := time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)

	if !timecalc.SameDay(a, b) {
		t.Error("SameDay: expected same day for a and b")
	}
	if timecalc.SameDay(a, c) {
		t.Error("SameDay: expected different day for a and c")
	}
}

func TestGenerateID(t *testing.T) {
	ts := time.Date(2026, 2, 27, 8, 32, 10, 0, time.UTC)
	id := timecalc.GenerateID(ts)
	if len(id) != len("20260227-083210-xxxxx") {
		t.Errorf("GenerateID length = %d, want %d", len(id), len("20260227-083210-xxxxx"))
	}
	if id[:15] != "20260227-083210" {
		t.Errorf("GenerateID prefix = %q, want %q", id[:15], "20260227-083210")
	}
}
