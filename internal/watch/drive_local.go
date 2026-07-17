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
	if dir == "" {
		return fmt.Errorf("drive local dir is empty")
	}
	if st, err := os.Stat(dir); err != nil {
		return fmt.Errorf("drive local dir %q: %w", dir, err)
	} else if !st.IsDir() {
		return fmt.Errorf("drive local dir %q is not a directory", dir)
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

	for {
		select {
		case <-ctx.Done():
			return nil
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			slog.Warn("drive local watch error", "err", err)
		case ev, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if ev.Op&fsnotify.Create != 0 {
				if st, err := os.Stat(ev.Name); err == nil && st.IsDir() {
					_ = watcher.Add(ev.Name)
				}
			}
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Remove) == 0 {
				continue
			}
			onChange("drive-local:" + filepath.Base(ev.Name))
		}
	}
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
