package main

import (
	"fmt"

	"github.com/jborkowski/lazy-notes/internal/config"
	"github.com/jborkowski/lazy-notes/internal/db"
	lnsync "github.com/jborkowski/lazy-notes/internal/sync"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Run one sync pass from Hugging Face to SuperWhisper",
	RunE: func(cmd *cobra.Command, args []string) error {
		return withConfigDB(func(cfg *config.Config, database *db.DB) error {
			result, err := lnsync.Run(cmd.Context(), cfg, database)
			if err != nil {
				return exitErr(fmt.Errorf("sync: %w", err))
			}
			printSyncResult(result)
			return nil
		})
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
