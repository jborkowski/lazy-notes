package db

import (
	"database/sql"
	"fmt"
	"strings"
)

// ClaimedSwIDs returns SuperWhisper transcript IDs already bound to a recording.
// Used so harvest cannot clamp a backlog onto one successful transcript.
func (d *DB) ClaimedSwIDs() (map[string]int64, error) {
	rows, err := d.sql.Query(`
SELECT sw_id, recording_id FROM recordings
WHERE sw_id IS NOT NULL AND TRIM(sw_id) != ''`)
	if err != nil {
		return nil, fmt.Errorf("query claimed sw_ids: %w", err)
	}
	defer rows.Close()

	out := make(map[string]int64)
	for rows.Next() {
		var swID string
		var recordingID int64
		if err := rows.Scan(&swID, &recordingID); err != nil {
			return nil, fmt.Errorf("scan claimed sw_id: %w", err)
		}
		swID = strings.TrimSpace(swID)
		if swID == "" {
			continue
		}
		out[swID] = recordingID
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate claimed sw_ids: %w", err)
	}
	return out, nil
}

// ResetForResubmit clears harvest/publish fields and returns the row to pending
// so the next sync re-extracts/re-submits audio to SuperWhisper (not just re-harvest).
func (d *DB) ResetForResubmit(id int64) error {
	res, err := d.sql.Exec(`
UPDATE recordings
SET status = ?,
    sw_id = NULL,
    body = '',
    note_path = '',
    published_at = NULL,
    submitted_at = NULL,
    error = ''
WHERE recording_id = ?`,
		StatusPending,
		id,
	)
	if err != nil {
		return fmt.Errorf("reset for resubmit %d: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("reset for resubmit rows affected %d: %w", id, err)
	}
	if n == 0 {
		return fmt.Errorf("reset for resubmit %d: recording not found", id)
	}
	return nil
}

// RepairPoisonedHarvestClaims undoes duplicate sw_id clamps and empty publishes.
// Returns how many rows were reset. Safe to run every sync — no-op when clean.
func (d *DB) RepairPoisonedHarvestClaims() (int, error) {
	reset := 0

	dupRows, err := d.sql.Query(`
SELECT recording_id FROM recordings
WHERE sw_id IS NOT NULL AND TRIM(sw_id) != ''
  AND sw_id IN (
    SELECT sw_id FROM recordings
    WHERE sw_id IS NOT NULL AND TRIM(sw_id) != ''
    GROUP BY sw_id
    HAVING COUNT(*) > 1
  )
ORDER BY recording_id`)
	if err != nil {
		return 0, fmt.Errorf("query duplicate sw_ids: %w", err)
	}
	var dupIDs []int64
	for dupRows.Next() {
		var id int64
		if err := dupRows.Scan(&id); err != nil {
			_ = dupRows.Close()
			return reset, fmt.Errorf("scan duplicate sw_id row: %w", err)
		}
		dupIDs = append(dupIDs, id)
	}
	if err := dupRows.Err(); err != nil {
		_ = dupRows.Close()
		return reset, fmt.Errorf("iterate duplicate sw_ids: %w", err)
	}
	_ = dupRows.Close()

	for _, id := range dupIDs {
		if err := d.ResetForResubmit(id); err != nil {
			return reset, err
		}
		reset++
	}

	emptyRows, err := d.sql.Query(`
SELECT recording_id FROM recordings
WHERE status IN (?, ?)
  AND (body IS NULL OR TRIM(body) = '')
ORDER BY recording_id`,
		StatusHarvested, StatusPublished,
	)
	if err != nil {
		return reset, fmt.Errorf("query empty harvested/published: %w", err)
	}
	var emptyIDs []int64
	for emptyRows.Next() {
		var id int64
		if err := emptyRows.Scan(&id); err != nil {
			_ = emptyRows.Close()
			return reset, fmt.Errorf("scan empty body row: %w", err)
		}
		emptyIDs = append(emptyIDs, id)
	}
	if err := emptyRows.Err(); err != nil {
		_ = emptyRows.Close()
		return reset, fmt.Errorf("iterate empty body rows: %w", err)
	}
	_ = emptyRows.Close()

	for _, id := range emptyIDs {
		if err := d.ResetForResubmit(id); err != nil {
			return reset, err
		}
		reset++
	}

	return reset, nil
}

// swIDClaimedByOther reports whether another recording already owns swID.
func (d *DB) swIDClaimedByOther(swID string, id int64) (int64, bool, error) {
	swID = strings.TrimSpace(swID)
	if swID == "" {
		return 0, false, nil
	}
	var other int64
	err := d.sql.QueryRow(`
SELECT recording_id FROM recordings
WHERE TRIM(sw_id) = ? AND recording_id != ?
LIMIT 1`, swID, id).Scan(&other)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return other, true, nil
}
