package watch

import (
	"path/filepath"
	"strings"

	"github.com/jborkowski/lazy-notes/internal/paths"
)

// DefaultAppleNotesDB is the standard macOS Apple Notes NoteStore path.
const DefaultAppleNotesDB = "~/Library/Group Containers/group.com.apple.notes/NoteStore.sqlite"

func expandHome(p string) string {
	return paths.Expand(p)
}

// isNotesStoreName reports whether name is NoteStore.sqlite or a WAL/SHM sibling.
func isNotesStoreName(name string) bool {
	base := filepath.Base(name)
	switch base {
	case "NoteStore.sqlite", "NoteStore.sqlite-wal", "NoteStore.sqlite-shm":
		return true
	}
	return strings.HasPrefix(base, "NoteStore.sqlite")
}
