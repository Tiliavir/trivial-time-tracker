package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/Tiliavir/trivial-time-tracker/internal/storage"
)

var stopComment string

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the currently running timer",
	Args:  cobra.NoArgs,
	RunE:  runStop,
}

func init() {
	stopCmd.Flags().StringVar(&stopComment, "comment", "", "Append a comment to the entry")
}

func runStop(cmd *cobra.Command, args []string) error {
	now := time.Now()

	base, err := storage.BaseDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	active, activeDay, err := storage.FindActiveEntry(base)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if active == nil {
		fmt.Fprintln(os.Stderr, "No active timer to stop.")
		os.Exit(1)
	}

	var comment *string
	if stopComment != "" {
		comment = &stopComment
	}

	if err := stopEntry(base, active, activeDay, now, comment); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	elapsed := int64(now.Sub(active.Start).Seconds())
	fmt.Printf("Stopped timer for project %q. Elapsed: %s\n",
		active.Project, formatElapsed(elapsed))
	return nil
}

func formatElapsed(seconds int64) string {
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
