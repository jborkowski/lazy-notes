package watch

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestDebouncerCoalesces(t *testing.T) {
	d := NewDebouncer(40 * time.Millisecond)
	defer d.Stop()

	var n atomic.Int32
	for i := 0; i < 5; i++ {
		d.Trigger(func() { n.Add(1) })
		time.Sleep(5 * time.Millisecond)
	}

	time.Sleep(80 * time.Millisecond)
	if got := n.Load(); got != 1 {
		t.Fatalf("expected 1 fire, got %d", got)
	}
}

func TestIsNotesStoreName(t *testing.T) {
	cases := map[string]bool{
		"NoteStore.sqlite":     true,
		"NoteStore.sqlite-wal": true,
		"NoteStore.sqlite-shm": true,
		"/tmp/NoteStore.sqlite": true,
		"other.sqlite":         false,
		"Notes.sqlite":         false,
	}
	for name, want := range cases {
		if got := isNotesStoreName(name); got != want {
			t.Fatalf("%q: got %v want %v", name, got, want)
		}
	}
}
