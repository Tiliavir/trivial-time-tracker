package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/Tiliavir/trivial-time-tracker/internal/storage"
	"github.com/Tiliavir/trivial-time-tracker/internal/timecalc"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current timer status",
	Args:  cobra.NoArgs,
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	now := time.Now()

	base, err := storage.BaseDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	active, _, err := storage.FindActiveEntry(base)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	if active != nil {
		elapsed := int64(now.Sub(active.Start).Seconds())
		fmt.Println("Running:")
		fmt.Printf("  Project: %s\n", active.Project)
		if active.Task != nil {
			fmt.Printf("  Task: %s\n", *active.Task)
		}
		fmt.Printf("  Since: %s\n", active.Start.Format("15:04"))
		fmt.Printf("  Elapsed: %s\n", timecalc.FormatDurationHHMMSS(elapsed))
		return nil
	}

	// Idle — show today's total.
	df, err := storage.LoadDay(base, now)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	var totalSeconds int64
	for _, e := range df.Entries {
		if e.DurationSeconds != nil {
			totalSeconds += *e.DurationSeconds
		}
	}

	fmt.Println("No active timer.")
	fmt.Printf("Today: %s logged.\n", timecalc.FormatDuration(totalSeconds))
	return nil
}
