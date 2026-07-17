package watch

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestParseDriveChangesArray(t *testing.T) {
	raw := []byte(`[
		{"fileId":"a","file":{"id":"a","parents":["folder1"]}},
		{"fileId":"b","removed":true}
	]`)
	changes, err := parseDriveChanges(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 2 {
		t.Fatalf("got %d changes", len(changes))
	}
	if !changeMatchesFolder(changes[0], "folder1") {
		t.Fatal("expected folder match")
	}
	if changeMatchesFolder(changes[1], "folder1") {
		t.Fatal("removed change without parents should not match folder filter")
	}
}

func TestParseDriveChangesEnvelope(t *testing.T) {
	raw := []byte(`{"changes":[{"fileId":"x","file":{"id":"x","parents":["p"]}}]}`)
	changes, err := parseDriveChanges(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 || changes[0].FileID != "x" {
		t.Fatalf("unexpected: %+v", changes)
	}
}

func TestWatchDriveGogMissingBinary(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := WatchDriveGog(ctx, DriveGogOptions{
		GogBin:    "gog-not-installed-for-test",
		StateFile: filepath.Join(t.TempDir(), "state.json"),
		LookPath: func(string) (string, error) {
			return "", exec.ErrNotFound
		},
	}, func(string) {})
	if err == nil {
		t.Fatal("expected error for missing gog binary")
	}
}

func TestPollDriveChangesOnceEmpty(t *testing.T) {
	dir := t.TempDir()
	state := filepath.Join(dir, "state.json")
	script := filepath.Join(dir, "fake-gog")
	body := "#!/bin/sh\n# ignore args\nexit 0\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}

	n, err := pollDriveChangesOnce(
		context.Background(),
		exec.CommandContext,
		script,
		"",
		"",
		state,
	)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("expected 0 changes, got %d", n)
	}
}

func TestOptionsEnabled(t *testing.T) {
	if (Options{}).Enabled() {
		t.Fatal("empty options should be disabled")
	}
	if !(Options{AppleNotesEnabled: true}).Enabled() {
		t.Fatal("apple notes should enable")
	}
	if (Options{DriveEnabled: true}).Enabled() {
		t.Fatal("drive enabled without targets should be disabled")
	}
	if !(Options{DriveEnabled: true, DriveFolderID: "abc"}).Enabled() {
		t.Fatal("drive folder should enable")
	}
}
