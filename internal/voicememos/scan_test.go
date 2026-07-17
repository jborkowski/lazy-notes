package voicememos

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanFiltersExtensionsAndDuration(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "keep.m4a"), "audio-a")
	mustWrite(t, filepath.Join(dir, "skip.txt"), "nope")
	mustWrite(t, filepath.Join(dir, "short.m4a"), "audio-b")
	mustWrite(t, filepath.Join(dir, ".hidden.m4a"), "hidden")

	items, err := Scan(ScanOptions{
		Dir:                dir,
		Extensions:         []string{".m4a"},
		MinDurationSeconds: 1.0,
		DurationOf: func(path string) (float64, error) {
			if filepath.Base(path) == "short.m4a" {
				return 0.2, nil
			}
			return 3.0, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Title != "keep" {
		t.Fatalf("title = %q, want keep", items[0].Title)
	}
	if items[0].ExternalID == "" {
		t.Fatal("expected external_id")
	}
}

func TestContentHashStable(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.m4a")
	b := filepath.Join(dir, "b.m4a")
	mustWrite(t, a, "same-bytes")
	mustWrite(t, b, "same-bytes")
	ha, err := ContentHash(a)
	if err != nil {
		t.Fatal(err)
	}
	hb, err := ContentHash(b)
	if err != nil {
		t.Fatal(err)
	}
	if ha != hb {
		t.Fatalf("hashes differ: %s vs %s", ha, hb)
	}
}

func TestStage(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	src := filepath.Join(srcDir, "rec.m4a")
	mustWrite(t, src, "payload")
	dest, err := Stage(src, dstDir, "abcd", ".m4a")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "payload" {
		t.Fatalf("staged = %q", data)
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
