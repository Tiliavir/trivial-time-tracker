package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/Tiliavir/trivial-time-tracker/internal/model"
	"github.com/Tiliavir/trivial-time-tracker/internal/storage"
	"github.com/Tiliavir/trivial-time-tracker/internal/timecalc"
)

var (
	listToday bool
	listWeek  bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List time entries",
	Args:  cobra.NoArgs,
	RunE:  runList,
}

func init() {
	listCmd.Flags().BoolVar(&listToday, "today", false, "Show today's entries")
	listCmd.Flags().BoolVar(&listWeek, "week", false, "Show this week's entries")
}

func runList(cmd *cobra.Command, args []string) error {
	now := time.Now()

	base, err := storage.BaseDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	var from, to time.Time
	switch {
	case listWeek:
		from, to = timecalc.WeekRange(now)
	default:
		// Default to today (covers --today and the bare command).
		from = timecalc.StartOfDay(now)
		to = timecalc.EndOfDay(now)
	}

	entries, err := storage.LoadRange(base, from, to)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	printList(entries)
	return nil
}

// printList groups entries by date and prints them.
func printList(entries []model.Entry) {
	if len(entries) == 0 {
		fmt.Println("No entries found.")
		return
	}

	var currentDay string
	for _, e := range entries {
		day := e.Start.Format("2006-01-02")
		if day != currentDay {
			fmt.Println(day)
			currentDay = day
		}

		startStr := e.Start.Format("15:04")
		endStr := "ongoing"
		durStr := ""
		if e.End != nil {
			endStr = e.End.Format("15:04")
		}
		if e.DurationSeconds != nil {
			durStr = fmt.Sprintf(" (%s)", timecalc.FormatDuration(*e.DurationSeconds))
		}

		task := ""
		if e.Task != nil {
			task = "  " + *e.Task
		}

		fmt.Printf("%s–%s  %s%s%s\n", startStr, endStr, e.Project, task, durStr)
	}
}
