package watch

import (
	"context"
	"fmt"
	"log/slog"
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
	if err := requireDir(dir, "apple notes dir"); err != nil {
		return err
	}

	watcher, err := newDirWatcher(dir)
	if err != nil {
		return err
	}
	defer watcher.Close()

	slog.Info("watching Apple Notes SQLite", "db", dbPath, "dir", dir)

	return runWatcher(ctx, watcher, "apple notes", func(ev fsnotify.Event) {
		if !isNotesStoreName(ev.Name) {
			return
		}
		if ev.Op&mutateEvents == 0 {
			return
		}
		onChange("apple-notes:" + filepath.Base(ev.Name))
	})
}
