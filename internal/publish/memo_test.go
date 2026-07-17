package publish

import (
	"os"
	"strings"
	"testing"
)

func TestBuildMemoMarkdown(t *testing.T) {
	got := buildMemoMarkdown("Title", "Body text", "#lazy-notes")
	want := "# Title\n\nBody text\n\n#lazy-notes\n"
	if got != want {
		t.Fatalf("buildMemoMarkdown() = %q, want %q", got, want)
	}

	got = buildMemoMarkdown("Title", "# Already headed\n\nBody", "#lazy-notes")
	if !strings.HasPrefix(got, "# Already headed") {
		t.Fatalf("expected body heading preserved, got %q", got)
	}
	if !strings.Contains(got, "#lazy-notes") {
		t.Fatalf("expected tag in memo body, got %q", got)
	}
}

func TestWriteMemoEditorScript(t *testing.T) {
	path, cleanup, err := writeMemoEditorScript()
	if err != nil {
		t.Fatalf("writeMemoEditorScript: %v", err)
	}
	defer cleanup()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat editor script: %v", err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("editor script is not executable")
	}
}
