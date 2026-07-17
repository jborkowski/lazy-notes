package hf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTokenFileCandidatesPrefersLazyNotesPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		t.Skip("no home dir")
	}
	t.Setenv("HF_HOME", filepath.Join(home, ".cache", "huggingface"))

	cands := tokenFileCandidates()
	if len(cands) == 0 {
		t.Fatal("expected candidates")
	}
	want := filepath.Join(home, ".config", "lazy-notes", "hf_token")
	if cands[0] != want {
		t.Fatalf("first candidate = %q, want %q", cands[0], want)
	}
	joined := strings.Join(cands, "\n")
	if !strings.Contains(joined, filepath.Join(home, ".cache", "huggingface", "token")) {
		t.Fatalf("missing HF_HOME token fallback in %v", cands)
	}
}

func TestDefaultTokenUsesHFTokenPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hf_token")
	if err := os.WriteFile(path, []byte("hf_test_token_value\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HF_TOKEN", "")
	t.Setenv("HUGGING_FACE_HUB_TOKEN", "")
	t.Setenv("HF_TOKEN_PATH", path)
	t.Setenv("HF_HOME", "")

	got := DefaultToken()
	if got != "hf_test_token_value" {
		t.Fatalf("DefaultToken() = %q", got)
	}
	if src := TokenSource(); src != "file:"+path {
		t.Fatalf("TokenSource() = %q", src)
	}
}
