package main

import (
	"fmt"
	"os"

	"github.com/jborkowski/lazy-notes/internal/config"
	"github.com/jborkowski/lazy-notes/internal/db"
	lnsync "github.com/jborkowski/lazy-notes/internal/sync"
	"github.com/spf13/cobra"
)

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Harvest SuperWhisper transcripts and publish notes",
	Long:  "Processes the backlog: harvest text for submitted recordings, then publish harvested notes to notes_dir, Apple Notes via memo, and optionally Google Drive via gog.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return withConfigDB(func(cfg *config.Config, database *db.DB) error {
			if !cfg.Publish.Enabled {
				fmt.Fprintln(os.Stdout, "publish disabled in config (publish.enabled = false)")
				return nil
			}
			result, err := lnsync.ProcessHarvestPublish(cmd.Context(), cfg, database)
			if err != nil {
				return exitErr(err)
			}
			fmt.Fprintf(os.Stdout, "harvested=%d published=%d errors=%d\n",
				result.Harvested, result.Published, result.Errors)
			return nil
		})
	},
}

func init() {
	rootCmd.AddCommand(publishCmd)
}
