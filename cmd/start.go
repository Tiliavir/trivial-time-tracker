package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Tiliavir/trivial-time-tracker/internal/model"
	"github.com/Tiliavir/trivial-time-tracker/internal/storage"
	"github.com/Tiliavir/trivial-time-tracker/internal/timecalc"
)

var (
	startTask    string
	startComment string
	startTags    string
)

var startCmd = &cobra.Command{
	Use:   "start <project>",
	Short: "Start a new time entry",
	Args:  cobra.ExactArgs(1),
	RunE:  runStart,
}

func init() {
	startCmd.Flags().StringVar(&startTask, "task", "", "Task description")
	startCmd.Flags().StringVar(&startComment, "comment", "", "Optional comment")
	startCmd.Flags().StringVar(&startTags, "tags", "", "Comma-separated tags")
}

func runStart(cmd *cobra.Command, args []string) error {
	project := args[0]
	now := time.Now()

	base, err := storage.BaseDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	// Check for an existing active timer and auto-stop it.
	active, activeDay, err := storage.FindActiveEntry(base)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if active != nil {
		fmt.Fprintf(os.Stderr, "Warning: auto-stopping active timer for project %q\n", active.Project)
		if err := stopEntry(base, active, activeDay, now, nil); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}

	// Build new entry.
	entry := model.Entry{
		ID:      timecalc.GenerateID(now),
		Project: project,
		Tags:    []string{},
		Start:   now,
		Source:  "manual",
	}
	if startTask != "" {
		entry.Task = &startTask
	}
	if startComment != "" {
		entry.Comment = &startComment
	}
	if startTags != "" {
		parts := strings.Split(startTags, ",")
		for i, p := range parts {
			parts[i] = strings.TrimSpace(p)
		}
		entry.Tags = parts
	}

	// Handle midnight crossover: if now is midnight exactly or start spans midnight,
	// we simply store on the current day as usual; crossover is handled at stop time.
	if err := storage.UpdateEntry(base, now, entry); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	fmt.Printf("Started timer for project %q at %s\n", project, now.Format("15:04:05"))
	return nil
}

// stopEntry closes an entry, handling midnight crossover by splitting if necessary.
func stopEntry(base string, entry *model.Entry, entryDay time.Time, stopTime time.Time, comment *string) error {
	if comment != nil && *comment != "" {
		if entry.Comment != nil {
			merged := *entry.Comment + "\n" + *comment
			entry.Comment = &merged
		} else {
			entry.Comment = comment
		}
	}

	// Check for midnight crossover.
	if !timecalc.SameDay(entry.Start, stopTime) {
		return splitAcrossMidnight(base, entry, entryDay, stopTime, comment)
	}

	end := stopTime
	dur := int64(stopTime.Sub(entry.Start).Seconds())
	entry.End = &end
	entry.DurationSeconds = &dur
	return storage.UpdateEntry(base, entryDay, *entry)
}

// splitAcrossMidnight splits a cross-midnight entry into two entries.
func splitAcrossMidnight(base string, entry *model.Entry, entryDay time.Time, stopTime time.Time, _ *string) error {
	// First segment ends at 23:59:59 of the start day.
	endOfFirst := timecalc.EndOfDay(entry.Start)
	dur1 := int64(endOfFirst.Sub(entry.Start).Seconds())
	entry.End = &endOfFirst
	entry.DurationSeconds = &dur1
	if err := storage.UpdateEntry(base, entryDay, *entry); err != nil {
		return err
	}

	// Second segment starts at 00:00:00 of the stop day.
	startOfSecond := timecalc.StartOfDay(stopTime)
	dur2 := int64(stopTime.Sub(startOfSecond).Seconds())
	second := model.Entry{
		ID:              timecalc.GenerateID(startOfSecond),
		Project:         entry.Project,
		Task:            entry.Task,
		Comment:         entry.Comment,
		Tags:            entry.Tags,
		Start:           startOfSecond,
		End:             &stopTime,
		DurationSeconds: &dur2,
		Source:          entry.Source,
	}
	return storage.UpdateEntry(base, stopTime, second)
}
