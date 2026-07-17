package publish

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteMarkdown(t *testing.T) {
	dir := t.TempDir()
	n := Note{
		Body:        "First line title\n\nBody paragraph.",
		RecordingID: 42,
		Language:    "en",
		ModeKey:     "lazy-note-en",
		SourceSW:    "sw-abc",
	}

	path, err := WriteMarkdown(dir, n)
	if err != nil {
		t.Fatalf("WriteMarkdown: %v", err)
	}

	wantName := "42-first-line-title.md"
	if filepath.Base(path) != wantName {
		t.Fatalf("filename = %q, want %q", filepath.Base(path), wantName)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	text := string(data)
	for _, want := range []string{
		"id: 42",
		`language: "en"`,
		`mode: "lazy-note-en"`,
		`sw_id: "sw-abc"`,
		"created:",
		"Body paragraph.",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in:\n%s", want, text)
		}
	}
}

func TestResolveTitle(t *testing.T) {
	tests := []struct {
		name string
		n    Note
		want string
	}{
		{
			name: "explicit title",
			n:    Note{Title: "Custom", Body: "ignored"},
			want: "Custom",
		},
		{
			name: "first line",
			n:    Note{Body: "Hello world\nrest"},
			want: "Hello world",
		},
		{
			name: "fallback id",
			n:    Note{RecordingID: 7},
			want: "Voice note 7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveTitle(tt.n); got != tt.want {
				t.Fatalf("ResolveTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSlugTitle(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"Hello, World!", "hello-world"},
		{"   spaced   out   ", "spaced-out"},
		{"!!!", "note"},
	}
	for _, tt := range tests {
		if got := slugTitle(tt.in); got != tt.want {
			t.Fatalf("slugTitle(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestWriteMarkdownExpandsTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip(err)
	}
	dir := filepath.Join(home, ".lazy-notes-test-publish")
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	n := Note{RecordingID: 1, Body: "Note body"}
	path, err := WriteMarkdown("~/.lazy-notes-test-publish", n)
	if err != nil {
		t.Fatalf("WriteMarkdown: %v", err)
	}
	if !strings.HasPrefix(path, dir) {
		t.Fatalf("path %q not under %q", path, dir)
	}
}
