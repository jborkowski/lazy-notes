package watch

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"github.com/jborkowski/lazy-notes/internal/config"
	"github.com/jborkowski/lazy-notes/internal/paths"
)

// TriggerFunc is invoked (debounced) when a watched source changes.
type TriggerFunc func(ctx context.Context, reason string)

// Options configures which sources to watch.
type Options struct {
	Debounce              time.Duration
	AppleNotesEnabled     bool
	AppleNotesDB          string
	DriveEnabled          bool
	DriveLocalDir         string
	DriveFolderID         string
	DrivePollInterval     time.Duration
	GogBin                string
	GogAccount            string
	DriveChangesStateFile string

	// Voice Memos export inbox (not NoteStore / not Group Container recordings).
	VoiceMemosEnabled    bool
	VoiceMemosExportDir  string
	VoiceMemosExtensions []string
}

// OptionsFromConfig builds watch Options from lazy-notes config.
func OptionsFromConfig(cfg *config.Config) Options {
	if cfg == nil {
		return Options{}
	}
	stateFile := filepath.Join(paths.StateDir(), "drive-changes.json")
	return Options{
		Debounce:              time.Duration(cfg.Watch.DebounceMs) * time.Millisecond,
		AppleNotesEnabled:     cfg.Watch.AppleNotesEnabled,
		AppleNotesDB:          cfg.AppleNotesDB(),
		DriveEnabled:          cfg.Watch.DriveEnabled,
		DriveLocalDir:         cfg.DriveLocalDir(),
		DriveFolderID:         cfg.Watch.DriveFolderID,
		DrivePollInterval:     time.Duration(cfg.Watch.DrivePollIntervalSecs) * time.Second,
		GogBin:                cfg.GogBin(),
		GogAccount:            cfg.Publish.GogAccount,
		DriveChangesStateFile: stateFile,
		VoiceMemosEnabled:     cfg.VoiceMemos.Enabled && cfg.VoiceMemos.WatchEnabled,
		VoiceMemosExportDir:   cfg.VoiceMemosExportDir(),
		VoiceMemosExtensions:  cfg.VoiceMemos.Extensions,
	}
}

// Enabled reports whether any watcher is turned on.
func (o Options) Enabled() bool {
	if o.AppleNotesEnabled {
		return true
	}
	if o.VoiceMemosEnabled && o.VoiceMemosExportDir != "" {
		return true
	}
	if !o.DriveEnabled {
		return false
	}
	return o.DriveLocalDir != "" || o.DriveFolderID != ""
}

// Run starts configured watchers and blocks until ctx is cancelled.
// onTrigger is debounced across all sources.
func Run(ctx context.Context, opts Options, onTrigger TriggerFunc) error {
	if onTrigger == nil {
		return fmt.Errorf("watch trigger is nil")
	}
	if !opts.Enabled() {
		<-ctx.Done()
		return nil
	}

	debouncer := NewDebouncer(opts.Debounce)
	defer debouncer.Stop()

	fire := func(reason string) {
		debouncer.Trigger(func() {
			if err := ctx.Err(); err != nil {
				return
			}
			slog.Info("watch trigger", "reason", reason)
			onTrigger(ctx, reason)
		})
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 4)

	start := func(name string, fn func() error) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := fn(); err != nil && ctx.Err() == nil {
				slog.Error("watcher stopped", "name", name, "err", err)
				errCh <- fmt.Errorf("%s: %w", name, err)
			}
		}()
	}

	if opts.AppleNotesEnabled {
		db := opts.AppleNotesDB
		if db == "" {
			db = expandHome(DefaultAppleNotesDB)
		}
		start("apple-notes", func() error {
			return WatchAppleNotes(ctx, db, fire)
		})
	}

	if opts.VoiceMemosEnabled && opts.VoiceMemosExportDir != "" {
		dir := opts.VoiceMemosExportDir
		if err := EnsureVoiceMemosInbox(dir); err != nil {
			return fmt.Errorf("voice memos inbox: %w", err)
		}
		exts := opts.VoiceMemosExtensions
		start("voice-memos-inbox", func() error {
			return WatchVoiceMemosInbox(ctx, dir, exts, fire)
		})
	}

	if opts.DriveEnabled && opts.DriveLocalDir != "" {
		dir := opts.DriveLocalDir
		start("drive-local", func() error {
			return WatchDriveLocal(ctx, dir, fire)
		})
	}

	if opts.DriveEnabled && opts.DriveFolderID != "" {
		gogOpts := DriveGogOptions{
			GogBin:    opts.GogBin,
			Account:   opts.GogAccount,
			FolderID:  opts.DriveFolderID,
			StateFile: opts.DriveChangesStateFile,
			Interval:  opts.DrivePollInterval,
		}
		start("drive-gog", func() error {
			return WatchDriveGog(ctx, gogOpts, fire)
		})
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		<-done
		return nil
	case err := <-errCh:
		return err
	case <-done:
		return nil
	}
}
