package daemon

import (
	"context"
	"log/slog"
	"time"
)

const defaultInterval = 15 * time.Minute

// SyncFunc is one sync pass.
type SyncFunc func(ctx context.Context) error

// IntervalFromSeconds converts seconds to a duration. Non-positive values use 15 minutes.
func IntervalFromSeconds(sec int) time.Duration {
	if sec <= 0 {
		return defaultInterval
	}
	return time.Duration(sec) * time.Second
}

// Run loops: call sync, sleep interval, until ctx cancelled.
func Run(ctx context.Context, interval time.Duration, syncFn SyncFunc) error {
	if err := syncFn(ctx); err != nil {
		slog.Error("sync failed", "err", err)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := syncFn(ctx); err != nil {
				slog.Error("sync failed", "err", err)
			}
		}
	}
}
