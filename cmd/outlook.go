package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/Tiliavir/trivial-time-tracker/internal/msgraph"
	"github.com/Tiliavir/trivial-time-tracker/internal/storage"
	"github.com/Tiliavir/trivial-time-tracker/internal/timecalc"
)

var (
	outlookSyncFrom    string
	outlookSyncTo      string
	outlookSyncDate    string
	outlookSyncToday   bool
	outlookSyncDryRun  bool
	outlookSyncProject string
	outlookSyncTZ      string
)

var outlookCmd = &cobra.Command{
	Use:   "outlook",
	Short: "Outlook calendar integration",
}

var outlookSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync Outlook calendar events into ttt entries",
	Args:  cobra.NoArgs,
	RunE:  runOutlookSync,
}

func init() {
	outlookSyncCmd.Flags().StringVar(&outlookSyncFrom, "from", "", "Start date (YYYY-MM-DD); required when --to is specified")
	outlookSyncCmd.Flags().StringVar(&outlookSyncTo, "to", "", "End date (YYYY-MM-DD); defaults to today")
	outlookSyncCmd.Flags().StringVar(&outlookSyncDate, "date", "", "Sync a specific date (YYYY-MM-DD)")
	outlookSyncCmd.Flags().BoolVar(&outlookSyncToday, "today", false, "Sync only today (default)")
	outlookSyncCmd.Flags().BoolVar(&outlookSyncDryRun, "dry-run", false, "Print planned operations without writing")
	outlookSyncCmd.Flags().StringVar(&outlookSyncProject, "project", "Meetings", "Default project name for imported events")
	outlookSyncCmd.Flags().StringVar(&outlookSyncTZ, "timezone", "", "IANA timezone for event times (e.g. Europe/Berlin)")
	outlookCmd.AddCommand(outlookSyncCmd)
}

func runOutlookSync(cmd *cobra.Command, args []string) error {
	now := time.Now()
	var from, to time.Time

	switch {
	case outlookSyncDate != "":
		d, err := time.Parse("2006-01-02", outlookSyncDate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid --date value %q: %v\n", outlookSyncDate, err)
			os.Exit(1)
		}
		from = timecalc.StartOfDay(d)
		to = timecalc.EndOfDay(d)

	case outlookSyncFrom != "" || outlookSyncTo != "":
		if outlookSyncTo != "" && outlookSyncFrom == "" {
			fmt.Fprintln(os.Stderr, "--from is required when --to is specified")
			os.Exit(1)
		}
		var err error
		from, err = time.Parse("2006-01-02", outlookSyncFrom)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid --from value %q: %v\n", outlookSyncFrom, err)
			os.Exit(1)
		}
		from = timecalc.StartOfDay(from)

		if outlookSyncTo != "" {
			t, err := time.Parse("2006-01-02", outlookSyncTo)
			if err != nil {
				fmt.Fprintf(os.Stderr, "invalid --to value %q: %v\n", outlookSyncTo, err)
				os.Exit(1)
			}
			to = timecalc.EndOfDay(t)
		} else {
			to = timecalc.EndOfDay(now)
		}

	default:
		// Default: today.
		from = timecalc.StartOfDay(now)
		to = timecalc.EndOfDay(now)
	}

	base, err := storage.BaseDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	timezone := outlookSyncTZ

	dryTag := ""
	if outlookSyncDryRun {
		dryTag = " [dry-run]"
	}
	fmt.Printf("Syncing Outlook events (%s → %s)%s...\n",
		from.Format("2006-01-02"), to.Format("2006-01-02"), dryTag)
	fmt.Println()

	ctx := context.Background()

	tok, cfg, err := msgraph.GetHTTPClient(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Authentication failed: %v\n", err)
		os.Exit(1)
	}

	client := msgraph.NewClient(ctx, tok, cfg)

	events, err := client.GetCalendarView(ctx, from, to, timezone)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to fetch calendar events: %v\n", err)
		os.Exit(1)
	}

	opts := msgraph.SyncOptions{
		Base:    base,
		From:    from,
		To:      to,
		DryRun:  outlookSyncDryRun,
		Project: outlookSyncProject,
	}

	result, err := msgraph.SyncEvents(events, opts, timezone)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Sync error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("Summary:")
	fmt.Printf("  %d imported\n", result.Imported)
	fmt.Printf("  %d skipped\n", result.Skipped)
	fmt.Printf("  %d updated\n", result.Updated)
	if result.Errors > 0 {
		fmt.Printf("  %d errors\n", result.Errors)
		os.Exit(2)
	}
	return nil
}
