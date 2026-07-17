package watch

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

// WatchVoiceMemosInbox watches the Voice Memos export inbox directory (non-recursive).
// This is separate from Apple Notes NoteStore wake — it only observes export_dir.
func WatchVoiceMemosInbox(ctx context.Context, dir string, extensions []string, onChange func(reason string)) error {
	dir = expandHome(dir)
	if err := requireDir(dir, "voice memos export dir"); err != nil {
		return err
	}

	extOK := make(map[string]bool)
	if len(extensions) == 0 {
		extOK[".m4a"] = true
	} else {
		for _, e := range extensions {
			e = strings.ToLower(strings.TrimSpace(e))
			if e == "" {
				continue
			}
			if !strings.HasPrefix(e, ".") {
				e = "." + e
			}
			extOK[e] = true
		}
	}

	watcher, err := newDirWatcher(dir)
	if err != nil {
		return err
	}
	defer watcher.Close()

	slog.Info("watching Voice Memos export inbox", "dir", dir)

	return runWatcher(ctx, watcher, "voice-memos-inbox", func(ev fsnotify.Event) {
		if ev.Op&mutateEvents == 0 {
			return
		}
		base := filepath.Base(ev.Name)
		if strings.HasPrefix(base, ".") {
			return
		}
		ext := strings.ToLower(filepath.Ext(base))
		if !extOK[ext] {
			return
		}
		onChange("voice-memos-inbox:" + base)
	})
}

// EnsureVoiceMemosInbox creates the export inbox when missing (daemon start helper).
func EnsureVoiceMemosInbox(dir string) error {
	dir = expandHome(dir)
	if dir == "" {
		return fmt.Errorf("voice memos export dir is empty")
	}
	return os.MkdirAll(dir, 0o755)
}
