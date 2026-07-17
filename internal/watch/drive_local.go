package watch

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

// WatchDriveLocal watches a local Google Drive desktop sync directory.
func WatchDriveLocal(ctx context.Context, dir string, onChange func(reason string)) error {
	dir = expandHome(dir)
	if err := requireDir(dir, "drive local dir"); err != nil {
		return err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("fsnotify: %w", err)
	}
	defer watcher.Close()

	if err := addRecursive(watcher, dir); err != nil {
		return err
	}

	slog.Info("watching Google Drive local directory", "dir", dir)

	return runWatcher(ctx, watcher, "drive local", func(ev fsnotify.Event) {
		if ev.Op&fsnotify.Create != 0 {
			if st, err := os.Stat(ev.Name); err == nil && st.IsDir() {
				_ = watcher.Add(ev.Name)
			}
		}
		if ev.Op&mutateEvents == 0 {
			return
		}
		onChange("drive-local:" + filepath.Base(ev.Name))
	})
}

func addRecursive(w *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if err := w.Add(path); err != nil {
				return fmt.Errorf("watch %q: %w", path, err)
			}
		}
		return nil
	})
}
