package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/jborkowski/lazy-notes/internal/daemon"
	lnsync "github.com/jborkowski/lazy-notes/internal/sync"
	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run sync on an interval until stopped",
	Long:  "Runs sync immediately, then repeats every sync.interval_seconds from config until SIGINT or SIGTERM.",
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

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		interval := daemon.IntervalFromSeconds(cfg.Sync.IntervalSeconds)
		syncFn := func(ctx context.Context) error {
			_, err := lnsync.Run(ctx, cfg, database)
			return err
		}

		fmt.Fprintf(os.Stdout, "daemon started (interval=%s)\n", interval)
		if err := daemon.Run(ctx, interval, syncFn); err != nil {
			return exitErr(fmt.Errorf("daemon: %w", err))
		}
		fmt.Fprintln(os.Stdout, "daemon stopped")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(daemonCmd)
}
