package main

import (
	"fmt"
	"os"

	"github.com/jborkowski/lazy-notes/internal/db"
	"github.com/jborkowski/lazy-notes/internal/paths"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show sync state and config location",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintf(os.Stdout, "config: %s\n", paths.ConfigPath())
		fmt.Fprintf(os.Stdout, "database: %s\n", paths.DBPath())

		database, err := openDB()
		if err != nil {
			return exitErr(err)
		}
		defer database.Close()

		counts, err := database.StatusCounts()
		if err != nil {
			return exitErr(fmt.Errorf("status counts: %w", err))
		}

		watermark, err := database.EffectiveWatermark()
		if err != nil {
			return exitErr(fmt.Errorf("watermark: %w", err))
		}

		fmt.Fprintf(os.Stdout, "watermark: %d\n", watermark)
		printStatusCounts(counts)
		return nil
	},
}

func printStatusCounts(counts map[db.Status]int) {
	order := []db.Status{db.StatusPending, db.StatusSubmitted, db.StatusError}
	for _, status := range order {
		fmt.Fprintf(os.Stdout, "%s: %d\n", status, counts[status])
	}
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
