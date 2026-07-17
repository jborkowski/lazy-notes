package main

import (
	"context"
	"fmt"

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

		printSyncResult(result)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
