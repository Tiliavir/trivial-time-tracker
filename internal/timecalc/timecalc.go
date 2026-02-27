package timecalc

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

// GenerateID creates a unique entry ID based on timestamp and random suffix.
func GenerateID(t time.Time) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	suffix := make([]byte, 5)
	for i := range suffix {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		suffix[i] = chars[n.Int64()]
	}
	return fmt.Sprintf("%s-%s", t.Format("20060102-150405"), string(suffix))
}

// FormatDuration formats seconds as a human-readable string like "1h 40m" or "45m" or "30s".
func FormatDuration(seconds int64) string {
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%ds", s)
}

// FormatDurationHHMMSS formats seconds as HH:MM:SS.
func FormatDurationHHMMSS(seconds int64) string {
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

// WeekRange returns the Monday and Sunday of the ISO week containing t.
func WeekRange(t time.Time) (time.Time, time.Time) {
	// Go's weekday: Sunday=0, Monday=1, …, Saturday=6
	wd := int(t.Weekday())
	if wd == 0 {
		wd = 7 // treat Sunday as 7 (ISO)
	}
	monday := t.AddDate(0, 0, -(wd - 1))
	monday = time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, t.Location())
	sunday := monday.AddDate(0, 0, 6)
	sunday = time.Date(sunday.Year(), sunday.Month(), sunday.Day(), 23, 59, 59, 0, t.Location())
	return monday, sunday
}

// ISOWeekLabel returns a label like "2026-W09".
func ISOWeekLabel(t time.Time) string {
	year, week := t.ISOWeek()
	return fmt.Sprintf("%d-W%02d", year, week)
}

// Midnight returns the start of the next day (midnight) in the same location.
func Midnight(t time.Time) time.Time {
	next := t.AddDate(0, 0, 1)
	return time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, t.Location())
}

// StartOfDay returns 00:00:00 of the same day.
func StartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// EndOfDay returns 23:59:59 of the same day.
func EndOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, t.Location())
}

// SameDay reports whether two times fall on the same calendar day.
func SameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
