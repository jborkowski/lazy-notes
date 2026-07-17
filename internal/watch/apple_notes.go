package watch

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

// WatchAppleNotes watches the Apple Notes NoteStore.sqlite directory for writes.
// SQLite updates often land on -wal/-shm siblings, so the parent directory is watched.
func WatchAppleNotes(ctx context.Context, dbPath string, onChange func(reason string)) error {
	dbPath = expandHome(dbPath)
	if dbPath == "" {
		return fmt.Errorf("apple notes db path is empty")
	}

	dir := filepath.Dir(dbPath)
	if st, err := os.Stat(dir); err != nil {
		return fmt.Errorf("apple notes dir %q: %w", dir, err)
	} else if !st.IsDir() {
		return fmt.Errorf("apple notes dir %q is not a directory", dir)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("fsnotify: %w", err)
	}
	defer watcher.Close()

	if err := watcher.Add(dir); err != nil {
		return fmt.Errorf("watch %q: %w", dir, err)
	}

	slog.Info("watching Apple Notes SQLite", "db", dbPath, "dir", dir)

	for {
		select {
		case <-ctx.Done():
			return nil
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			slog.Warn("apple notes watch error", "err", err)
		case ev, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if !isNotesStoreName(ev.Name) {
				continue
			}
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Remove) == 0 {
				continue
			}
			onChange("apple-notes:" + filepath.Base(ev.Name))
		}
	}
}
