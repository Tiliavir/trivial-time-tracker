package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/Tiliavir/trivial-time-tracker/internal/model"
	"github.com/Tiliavir/trivial-time-tracker/internal/storage"
	"github.com/Tiliavir/trivial-time-tracker/internal/timecalc"
)

var exportFormat string

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export time entries to stdout",
	Args:  cobra.NoArgs,
	RunE:  runExport,
}

func init() {
	exportCmd.Flags().StringVar(&exportFormat, "format", "csv", "Output format: csv, json, md")
}

func runExport(cmd *cobra.Command, args []string) error {
	now := time.Now()

	base, err := storage.BaseDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	from, to := timecalc.WeekRange(now)

	entries, err := storage.LoadRange(base, from, to)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	switch exportFormat {
	case "json":
		data, err := json.MarshalIndent(entries, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, "error encoding JSON:", err)
			os.Exit(2)
		}
		fmt.Println(string(data))
	case "md":
		printList(entries)
	default: // csv
		printCSV(entries)
	}

	return nil
}

func printCSV(entries []model.Entry) {
	fmt.Println("date,project,task,comment,start,end,duration_minutes")
	for _, e := range entries {
		date := e.Start.Format("2006-01-02")
		task := ""
		if e.Task != nil {
			task = *e.Task
		}
		comment := ""
		if e.Comment != nil {
			comment = *e.Comment
		}
		startStr := e.Start.Format(time.RFC3339)
		endStr := ""
		if e.End != nil {
			endStr = e.End.Format(time.RFC3339)
		}
		durMin := int64(0)
		if e.DurationSeconds != nil {
			durMin = *e.DurationSeconds / 60
		}
		fmt.Printf("%s,%s,%s,%s,%s,%s,%d\n",
			csvEscape(date),
			csvEscape(e.Project),
			csvEscape(task),
			csvEscape(comment),
			csvEscape(startStr),
			csvEscape(endStr),
			durMin,
		)
	}
}

// csvEscape wraps a field in quotes if it contains a comma, quote, or newline.
func csvEscape(s string) string {
	needsQuote := false
	for _, c := range s {
		if c == ',' || c == '"' || c == '\n' || c == '\r' {
			needsQuote = true
			break
		}
	}
	if !needsQuote {
		return s
	}
	// Escape internal double quotes by doubling them.
	escaped := ""
	for _, c := range s {
		if c == '"' {
			escaped += "\""
		}
		escaped += string(c)
	}
	return `"` + escaped + `"`
}
