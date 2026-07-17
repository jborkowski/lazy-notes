package db

import (
	"path/filepath"
	"testing"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "state.sqlite")
	d, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return d
}

func TestHFWatermarkIgnoresVoiceMemoIDs(t *testing.T) {
	d := openTestDB(t)

	if err := d.UpsertPending(Recording{
		RecordingID: 10,
		Source:      SourceHF,
		CreatedAt:   "2026-01-01T00:00:00Z",
		Title:       "hf",
		Status:      StatusPending,
	}); err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertVoiceMemoPending(Recording{
		ExternalID: "abc",
		CreatedAt:  "2026-01-02T00:00:00Z",
		Title:      "vm",
		AudioPath:  "/tmp/a.m4a",
		Language:   "auto",
		ModeKey:    "lazy-note-auto",
	}); err != nil {
		t.Fatal(err)
	}

	wm, err := d.Watermark()
	if err != nil {
		t.Fatal(err)
	}
	if wm != 10 {
		t.Fatalf("Watermark() = %d, want 10 (VM negatives must not poison)", wm)
	}

	known, err := d.KnownHFIDs()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := known[10]; !ok {
		t.Fatal("expected HF id 10 in KnownHFIDs")
	}
	if len(known) != 1 {
		t.Fatalf("KnownHFIDs len = %d, want 1", len(known))
	}
}

func TestAdvanceWatermarkIgnoresNonPositive(t *testing.T) {
	d := openTestDB(t)
	if err := d.AdvanceWatermark(5); err != nil {
		t.Fatal(err)
	}
	if err := d.AdvanceWatermark(-99); err != nil {
		t.Fatal(err)
	}
	got, err := d.EffectiveWatermark()
	if err != nil {
		t.Fatal(err)
	}
	if got != 5 {
		t.Fatalf("EffectiveWatermark() = %d, want 5", got)
	}
}

func TestUpsertVoiceMemoPendingDedupeAndNegativeIDs(t *testing.T) {
	d := openTestDB(t)

	rec := Recording{
		ExternalID: "file-hash-1",
		CreatedAt:  "2026-01-01T00:00:00Z",
		Title:      "Recording",
		AudioPath:  "/cache/a.m4a",
		Language:   "auto",
		ModeKey:    "lazy-note-auto",
	}
	if err := d.UpsertVoiceMemoPending(rec); err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertVoiceMemoPending(rec); err != nil {
		t.Fatal(err)
	}

	got, err := d.GetBySourceExternalID(SourceVoiceMemo, "file-hash-1")
	if err != nil || got == nil {
		t.Fatalf("GetBySourceExternalID: %v %#v", err, got)
	}
	if got.RecordingID != -1 {
		t.Fatalf("first VM id = %d, want -1", got.RecordingID)
	}
	if got.Status != StatusPending {
		t.Fatalf("status = %q, want pending", got.Status)
	}

	if err := d.MarkSubmitted(got.RecordingID, "auto", "lazy-note-auto", got.AudioPath); err != nil {
		t.Fatal(err)
	}
	// Re-upsert must not reset submitted → pending.
	rec.Title = "Renamed"
	if err := d.UpsertVoiceMemoPending(rec); err != nil {
		t.Fatal(err)
	}
	got, err = d.GetBySourceExternalID(SourceVoiceMemo, "file-hash-1")
	if err != nil || got == nil {
		t.Fatal(err)
	}
	if got.Status != StatusSubmitted {
		t.Fatalf("status after refresh = %q, want submitted", got.Status)
	}
	if got.Title != "Renamed" {
		t.Fatalf("title = %q, want Renamed", got.Title)
	}

	if err := d.UpsertVoiceMemoPending(Recording{
		ExternalID: "file-hash-2",
		CreatedAt:  "2026-01-02T00:00:00Z",
		Title:      "Second",
		AudioPath:  "/cache/b.m4a",
		Language:   "en",
		ModeKey:    "lazy-note-en",
	}); err != nil {
		t.Fatal(err)
	}
	second, err := d.GetBySourceExternalID(SourceVoiceMemo, "file-hash-2")
	if err != nil || second == nil {
		t.Fatal(err)
	}
	if second.RecordingID != -2 {
		t.Fatalf("second VM id = %d, want -2", second.RecordingID)
	}
}

func TestVoiceMemoWatermarkCursor(t *testing.T) {
	d := openTestDB(t)
	if err := d.AdvanceVoiceMemoWatermarkMTimeNS(100); err != nil {
		t.Fatal(err)
	}
	if err := d.AdvanceVoiceMemoWatermarkMTimeNS(50); err != nil {
		t.Fatal(err)
	}
	got, err := d.VoiceMemoWatermarkMTimeNS()
	if err != nil {
		t.Fatal(err)
	}
	if got != 100 {
		t.Fatalf("VoiceMemoWatermarkMTimeNS() = %d, want 100", got)
	}
}
