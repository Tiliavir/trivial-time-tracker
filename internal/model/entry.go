package model

import "time"

// Entry represents a single tracked time entry.
type Entry struct {
	ID              string     `json:"id"`
	Project         string     `json:"project"`
	Task            *string    `json:"task"`
	Comment         *string    `json:"comment"`
	Tags            []string   `json:"tags"`
	Start           time.Time  `json:"start"`
	End             *time.Time `json:"end"`
	DurationSeconds *int64     `json:"duration_seconds"`
	Source          string     `json:"source"`
}

// DayFile is the top-level structure stored in each daily JSON file.
type DayFile struct {
	Date    string  `json:"date"`
	Entries []Entry `json:"entries"`
}
