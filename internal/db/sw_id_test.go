package db

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func seedSubmitted(t *testing.T, d *DB, id int64) {
	t.Helper()
	if err := d.UpsertPending(Recording{
		RecordingID: id,
		CreatedAt:   "2026-07-18T12:00:00Z",
		Title:       "rec",
		Language:    "auto",
		ModeKey:     "lazy-note-auto",
		Status:      StatusPending,
	}); err != nil {
		t.Fatal(err)
	}
	if err := d.MarkSubmitted(id, "auto", "lazy-note-auto", "/tmp/a.wav"); err != nil {
		t.Fatal(err)
	}
}

func TestMarkHarvestedRejectsDuplicateSwID(t *testing.T) {
	d := openTestDB(t)
	seedSubmitted(t, d, 230)
	seedSubmitted(t, d, 231)

	const sw = "5262f3959be449a0aef6943f78cebf56"
	if err := d.MarkHarvested(230, sw, "reflash my ZMK keyboard", ""); err != nil {
		t.Fatal(err)
	}
	err := d.MarkHarvested(231, sw, "reflash my ZMK keyboard", "")
	if err == nil {
		t.Fatal("expected duplicate sw_id error")
	}
	if !strings.Contains(err.Error(), "already claimed") {
		t.Fatalf("error = %v, want already claimed", err)
	}
}

func TestMarkHarvestedRejectsEmptyBody(t *testing.T) {
	d := openTestDB(t)
	seedSubmitted(t, d, 1)
	err := d.MarkHarvested(1, "sw1", "   ", "")
	if err == nil || !strings.Contains(err.Error(), "empty body") {
		t.Fatalf("error = %v, want empty body", err)
	}
}

func TestRepairPoisonedHarvestClaimsResetsDuplicates(t *testing.T) {
	d := openTestDB(t)
	const sw = "5262f3959be449a0aef6943f78cebf56"
	body := "reflash my ZMK keyboard firmware"

	for _, id := range []int64{230, 231, 232} {
		seedSubmitted(t, d, id)
		if _, err := d.sql.Exec(`
UPDATE recordings SET status = ?, sw_id = ?, body = ?, published_at = ?
WHERE recording_id = ?`,
			StatusPublished, sw, body, time.Now().UTC().Format(time.RFC3339), id,
		); err != nil {
			t.Fatal(err)
		}
	}

	n, err := d.RepairPoisonedHarvestClaims()
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("reset = %d, want 3", n)
	}

	claimed, err := d.ClaimedSwIDs()
	if err != nil {
		t.Fatal(err)
	}
	if len(claimed) != 0 {
		t.Fatalf("claimed after repair = %#v, want empty", claimed)
	}

	for _, id := range []int64{230, 231, 232} {
		rec, err := d.GetByID(id)
		if err != nil || rec == nil {
			t.Fatalf("GetByID(%d): %v %#v", id, err, rec)
		}
		if rec.Status != StatusPending {
			t.Fatalf("id %d status = %q, want pending", id, rec.Status)
		}
		if rec.SwID != "" || rec.Body != "" {
			t.Fatalf("id %d still has harvest fields: sw_id=%q body=%q", id, rec.SwID, rec.Body)
		}
	}
}

func TestRepairPoisonedHarvestClaimsResetsEmptyPublished(t *testing.T) {
	d := openTestDB(t)
	seedSubmitted(t, d, 10)
	if _, err := d.sql.Exec(`
UPDATE recordings SET status = ?, sw_id = ?, body = ?, published_at = ?
WHERE recording_id = ?`,
		StatusPublished, "sw-empty", "  ", time.Now().UTC().Format(time.RFC3339), int64(10),
	); err != nil {
		t.Fatal(err)
	}

	n, err := d.RepairPoisonedHarvestClaims()
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("reset = %d, want 1", n)
	}
	rec, err := d.GetByID(10)
	if err != nil || rec == nil {
		t.Fatalf("GetByID: %v %#v", err, rec)
	}
	if rec.Status != StatusPending {
		t.Fatalf("status = %q, want pending", rec.Status)
	}
}

func TestClaimedSwIDs(t *testing.T) {
	d := openTestDB(t)
	seedSubmitted(t, d, 1)
	if err := d.MarkHarvested(1, "abc", "hello world", ""); err != nil {
		t.Fatal(err)
	}
	claimed, err := d.ClaimedSwIDs()
	if err != nil {
		t.Fatal(err)
	}
	if claimed["abc"] != 1 {
		t.Fatalf("claimed = %#v", claimed)
	}
}
