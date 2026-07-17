package db

import (
	"database/sql"
	"fmt"
)

// GetBySourceExternalID returns a recording by source + external_id, or nil.
func (d *DB) GetBySourceExternalID(source, externalID string) (*Recording, error) {
	if source == "" || externalID == "" {
		return nil, fmt.Errorf("source and external_id are required")
	}
	row := d.sql.QueryRow(
		`SELECT`+recordingSelectColumns+` FROM recordings WHERE source = ? AND external_id = ?`,
		source,
		externalID,
	)
	r, err := scanRecording(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get by source/external_id: %w", err)
	}
	return &r, nil
}

// AllocateVoiceMemoID returns the next negative recording_id for Voice Memos.
// IDs are -1, -2, … so they never collide with positive HF IDs.
func (d *DB) AllocateVoiceMemoID() (int64, error) {
	var minID sql.NullInt64
	err := d.sql.QueryRow(`SELECT MIN(recording_id) FROM recordings WHERE recording_id < 0`).Scan(&minID)
	if err != nil {
		return 0, fmt.Errorf("allocate voice memo id: %w", err)
	}
	if !minID.Valid {
		return -1, nil
	}
	return minID.Int64 - 1, nil
}

// ListPendingBySource returns pending rows for one source (e.g. voicememo).
func (d *DB) ListPendingBySource(source string) ([]Recording, error) {
	rows, err := d.sql.Query(
		`SELECT`+recordingSelectColumns+` FROM recordings WHERE status = ? AND COALESCE(source, ?) = ? ORDER BY recording_id DESC`,
		StatusPending,
		SourceHF,
		source,
	)
	if err != nil {
		return nil, fmt.Errorf("list pending source %q: %w", source, err)
	}
	defer rows.Close()
	return collectRecordings(rows, "pending:"+source)
}

// UpsertVoiceMemoPending inserts a new Voice Memo or updates title/path when still
// pending/error. It never resets submitted/harvested/published rows.
func (d *DB) UpsertVoiceMemoPending(r Recording) error {
	r.Source = SourceVoiceMemo
	if r.ExternalID == "" {
		return fmt.Errorf("voice memo external_id is required")
	}

	existing, err := d.GetBySourceExternalID(SourceVoiceMemo, r.ExternalID)
	if err != nil {
		return err
	}
	if existing != nil {
		switch existing.Status {
		case StatusSubmitted, StatusHarvested, StatusPublished:
			_, err := d.sql.Exec(`
UPDATE recordings SET title = ?,
  audio_path = CASE WHEN ? != '' THEN ? ELSE audio_path END
WHERE recording_id = ?`,
				r.Title, r.AudioPath, r.AudioPath, existing.RecordingID)
			if err != nil {
				return fmt.Errorf("refresh voice memo %q: %w", r.ExternalID, err)
			}
			return nil
		default:
			_, err := d.sql.Exec(`
UPDATE recordings
SET title = ?, audio_path = ?, language = ?, mode_key = ?, status = ?, submitted_at = NULL, error = ''
WHERE recording_id = ?`,
				r.Title,
				r.AudioPath,
				r.Language,
				r.ModeKey,
				StatusPending,
				existing.RecordingID,
			)
			if err != nil {
				return fmt.Errorf("requeue voice memo %q: %w", r.ExternalID, err)
			}
			return nil
		}
	}

	id, err := d.AllocateVoiceMemoID()
	if err != nil {
		return err
	}
	_, err = d.sql.Exec(`
INSERT INTO recordings (
  recording_id, source, external_id, created_at, title, audio_path, language, mode_key, status, submitted_at, error
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, '')`,
		id,
		SourceVoiceMemo,
		r.ExternalID,
		r.CreatedAt,
		r.Title,
		r.AudioPath,
		r.Language,
		r.ModeKey,
		StatusPending,
	)
	if err != nil {
		return fmt.Errorf("insert voice memo %q: %w", r.ExternalID, err)
	}
	return nil
}
