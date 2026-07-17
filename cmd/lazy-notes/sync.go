package main

import (
	"context"
	"fmt"
	"os"

	lnsync "github.com/jborkowski/lazy-notes/internal/sync"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Run one sync pass from Hugging Face to SuperWhisper",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return exitErr(err)
		}

		database, err := openDB()
		if err != nil {
			return exitErr(err)
		}
		defer database.Close()

		result, err := lnsync.Run(context.Background(), cfg, database)
		if err != nil {
			return exitErr(fmt.Errorf("sync: %w", err))
		}

		fmt.Fprintf(os.Stdout, "scanned=%d submitted=%d skipped=%d errors=%d watermark=%d\n",
			result.Scanned, result.Submitted, result.Skipped, result.Errors, result.Watermark)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
