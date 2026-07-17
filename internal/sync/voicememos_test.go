package sync

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jborkowski/lazy-notes/internal/config"
	"github.com/jborkowski/lazy-notes/internal/db"
	"github.com/jborkowski/lazy-notes/internal/paths"
	"github.com/jborkowski/lazy-notes/internal/voicememos"
)

func TestRunVoiceMemosDisabledNoop(t *testing.T) {
	cfg := config.Defaults()
	database, err := db.Open(filepath.Join(t.TempDir(), "state.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })

	result := &Result{}
	if err := RunVoiceMemos(context.Background(), &cfg, database, result); err != nil {
		t.Fatal(err)
	}
	if result.Scanned != 0 || result.Submitted != 0 {
		t.Fatalf("disabled should noop: %+v", result)
	}
}

func TestRunVoiceMemosUpsertsPendingWithoutSubmitWhenNoSW(t *testing.T) {
	// Staging + DB upsert path; SuperWhisper submit will fail and mark error —
	// still proves inbox → pending/error identity without poisoning HF watermark.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("LAZY_NOTES_STATE_DIR", filepath.Join(home, "state"))
	t.Setenv("LAZY_NOTES_CACHE_DIR", filepath.Join(home, "cache"))

	inbox := filepath.Join(home, "inbox")
	if err := os.MkdirAll(inbox, 0o755); err != nil {
		t.Fatal(err)
	}
	audio := filepath.Join(inbox, "hello.m4a")
	if err := os.WriteFile(audio, []byte("fake-m4a-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	cfg.VoiceMemos.Enabled = true
	cfg.VoiceMemos.ExportDir = inbox
	cfg.VoiceMemos.MinDurationSeconds = 0 // skip ffprobe filter
	cfg.Sync.SubmitDelaySeconds = 0
	cfg.Publish.Enabled = false

	database, err := db.Open(paths.DBPath())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })

	// Seed HF row so we can assert watermark stays HF-scoped.
	if err := database.UpsertPending(db.Recording{
		RecordingID: 99,
		Source:      db.SourceHF,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		Title:       "hf",
		Status:      db.StatusPending,
	}); err != nil {
		t.Fatal(err)
	}
	if err := database.AdvanceWatermark(99); err != nil {
		t.Fatal(err)
	}

	result := &Result{}
	_ = RunVoiceMemos(context.Background(), &cfg, database, result)

	got, err := database.GetBySourceExternalID(db.SourceVoiceMemo, mustHash(t, audio))
	if err != nil || got == nil {
		t.Fatalf("expected voice memo row: %v", err)
	}
	if got.RecordingID >= 0 {
		t.Fatalf("VM id should be negative, got %d", got.RecordingID)
	}

	wm, err := database.EffectiveWatermark()
	if err != nil {
		t.Fatal(err)
	}
	if wm != 99 {
		t.Fatalf("HF watermark poisoned: %d", wm)
	}
}

func mustHash(t *testing.T, path string) string {
	t.Helper()
	sum, err := voicememos.ContentHash(path)
	if err != nil {
		t.Fatal(err)
	}
	return sum
}
