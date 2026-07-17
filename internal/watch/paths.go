package watch

import (
	"os"
	"path/filepath"
	"strings"
)

// DefaultAppleNotesDB is the standard macOS Apple Notes NoteStore path.
const DefaultAppleNotesDB = "~/Library/Group Containers/group.com.apple.notes/NoteStore.sqlite"

func expandHome(p string) string {
	if p == "" {
		return ""
	}
	if strings.HasPrefix(p, "~/") || p == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return p
		}
		if p == "~" {
			return home
		}
		return filepath.Join(home, p[2:])
	}
	return p
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
