package watch

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/fsnotify/fsnotify"
)

func requireDir(path, label string) error {
	if path == "" {
		return fmt.Errorf("%s is empty", label)
	}
	st, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%s %q: %w", label, path, err)
	}
	if !st.IsDir() {
		return fmt.Errorf("%s %q is not a directory", label, path)
	}
	return nil
}

func newDirWatcher(dir string) (*fsnotify.Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("fsnotify: %w", err)
	}
	if err := watcher.Add(dir); err != nil {
		_ = watcher.Close()
		return nil, fmt.Errorf("watch %q: %w", dir, err)
	}
	return watcher, nil
}

// mutateEvents is Write|Create|Rename|Remove.
const mutateEvents = fsnotify.Write | fsnotify.Create | fsnotify.Rename | fsnotify.Remove

func runWatcher(ctx context.Context, watcher *fsnotify.Watcher, name string, handle func(fsnotify.Event)) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			slog.Warn(name+" watch error", "err", err)
		case ev, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			handle(ev)
		}
	}
}
