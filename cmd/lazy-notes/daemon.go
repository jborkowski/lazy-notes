package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/jborkowski/lazy-notes/internal/config"
	"github.com/jborkowski/lazy-notes/internal/daemon"
	"github.com/jborkowski/lazy-notes/internal/db"
	lnsync "github.com/jborkowski/lazy-notes/internal/sync"
	"github.com/jborkowski/lazy-notes/internal/watch"
	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run sync on an interval until stopped",
	Long: `Runs sync immediately, then repeats every sync.interval_seconds from config until SIGINT or SIGTERM.

Optional watchers also trigger a sync pass when the Voice Memos export inbox,
Apple Notes NoteStore.sqlite (wake only), or a Google Drive directory/folder changes.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return withConfigDB(func(cfg *config.Config, database *db.DB) error {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			interval := daemon.IntervalFromSeconds(cfg.Sync.IntervalSeconds)

			var syncMu sync.Mutex
			syncFn := func(ctx context.Context) error {
				syncMu.Lock()
				defer syncMu.Unlock()
				_, err := lnsync.Run(ctx, cfg, database)
				return err
			}

			watchOpts := watch.OptionsFromConfig(cfg)
			if watchOpts.Enabled() {
				go func() {
					err := watch.Run(ctx, watchOpts, func(ctx context.Context, reason string) {
						if err := syncFn(ctx); err != nil {
							slog.Error("watch-triggered sync failed", "reason", reason, "err", err)
						}
					})
					if err != nil && ctx.Err() == nil {
						slog.Error("watchers stopped", "err", err)
					}
				}()
				fmt.Fprintf(os.Stdout, "daemon started (interval=%s watchers=on)\n", interval)
			} else {
				fmt.Fprintf(os.Stdout, "daemon started (interval=%s)\n", interval)
			}

			if err := daemon.Run(ctx, interval, syncFn); err != nil {
				return exitErr(fmt.Errorf("daemon: %w", err))
			}
			fmt.Fprintln(os.Stdout, "daemon stopped")
			return nil
		})
	},
}

func init() {
	rootCmd.AddCommand(daemonCmd)
}
