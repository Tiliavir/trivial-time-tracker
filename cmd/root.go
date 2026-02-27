package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/Tiliavir/trivial-time-tracker/internal/config"
)

var rootCmd = &cobra.Command{
	Use:   "ttt",
	Short: "Trivial Time Tracker – a minimal CLI time tracker",
	Long: `ttt is a single-binary, file-based command-line time tracker.
All data is stored as human-readable JSON files in ~/.ttt/.`,
	// PersistentPreRunE runs before every subcommand, ensuring ~/.ttt/config.json
	// is created with annotated defaults on the very first invocation.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if _, err := config.Load(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: config error: %v\n", err)
		}
		return nil
	},
}

// Execute is the entry point called from main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(reportCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(outlookCmd)
}
