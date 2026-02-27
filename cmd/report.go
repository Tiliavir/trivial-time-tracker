package cmd

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"github.com/Tiliavir/trivial-time-tracker/internal/storage"
	"github.com/Tiliavir/trivial-time-tracker/internal/timecalc"
)

var (
	reportWeek   bool
	reportFormat string
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Show aggregated time report",
	Args:  cobra.NoArgs,
	RunE:  runReport,
}

func init() {
	reportCmd.Flags().BoolVar(&reportWeek, "week", false, "Report for this week (default)")
	reportCmd.Flags().StringVar(&reportFormat, "format", "md", "Output format: md, csv, json")
}

func runReport(cmd *cobra.Command, args []string) error {
	now := time.Now()

	base, err := storage.BaseDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	from, to := timecalc.WeekRange(now)
	label := timecalc.ISOWeekLabel(now)

	entries, err := storage.LoadRange(base, from, to)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	// Aggregate by project.
	totals := map[string]int64{}
	var order []string
	for _, e := range entries {
		if e.DurationSeconds == nil {
			continue
		}
		if _, seen := totals[e.Project]; !seen {
			order = append(order, e.Project)
		}
		totals[e.Project] += *e.DurationSeconds
	}
	sort.Strings(order)

	var grandTotal int64
	for _, sec := range totals {
		grandTotal += sec
	}

	switch reportFormat {
	case "csv":
		fmt.Println("project,duration_minutes")
		for _, p := range order {
			fmt.Printf("%s,%d\n", p, totals[p]/60)
		}
	case "json":
		fmt.Println("{")
		fmt.Printf("  \"week\": %q,\n", label)
		fmt.Println("  \"projects\": [")
		for i, p := range order {
			comma := ","
			if i == len(order)-1 {
				comma = ""
			}
			fmt.Printf("    {\"project\": %q, \"duration_minutes\": %d}%s\n",
				p, totals[p]/60, comma)
		}
		fmt.Println("  ],")
		fmt.Printf("  \"total_minutes\": %d\n", grandTotal/60)
		fmt.Println("}")
	default: // md
		fmt.Printf("Week %s\n", label)
		fmt.Println("--------------------------------")
		for _, p := range order {
			fmt.Printf("%-20s%s\n", p, timecalc.FormatDuration(totals[p]))
		}
		fmt.Println("--------------------------------")
		fmt.Printf("%-20s%s\n", "Total", timecalc.FormatDuration(grandTotal))
	}

	return nil
}
