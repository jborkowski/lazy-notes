package db

import (
	"database/sql"
	"fmt"
	"time"
)

const recordingSelectColumns = `
recording_id, created_at, title, audio_path, language, mode_key, status,
submitted_at, error, sw_id, note_path, published_at, body`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRecording(s rowScanner) (Recording, error) {
	var r Recording
	var submittedAt, publishedAt sql.NullString
	var swID, notePath, body sql.NullString
	if err := s.Scan(
		&r.RecordingID,
		&r.CreatedAt,
		&r.Title,
		&r.AudioPath,
		&r.Language,
		&r.ModeKey,
		&r.Status,
		&submittedAt,
		&r.Error,
		&swID,
		&notePath,
		&publishedAt,
		&body,
	); err != nil {
		return Recording{}, err
	}
	r.SubmittedAt = parseRFC3339Ptr(submittedAt)
	r.PublishedAt = parseRFC3339Ptr(publishedAt)
	r.SwID = swID.String
	r.NotePath = notePath.String
	r.Body = body.String
	return r, nil
}

func parseRFC3339Ptr(s sql.NullString) *time.Time {
	if !s.Valid || s.String == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s.String)
	if err != nil {
		return nil
	}
	return &t
}

func (d *DB) KnownIDs() (map[int64]Status, error) {
	rows, err := d.sql.Query(`SELECT recording_id, status FROM recordings`)
	if err != nil {
		return nil, fmt.Errorf("query known ids: %w", err)
	}
	defer rows.Close()

	out := make(map[int64]Status)
	for rows.Next() {
		var id int64
		var status Status
		if err := rows.Scan(&id, &status); err != nil {
			return nil, fmt.Errorf("scan known id: %w", err)
		}
		out[id] = status
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate known ids: %w", err)
	}
	return out, nil
}

func (d *DB) UpsertPending(r Recording) error {
	_, err := d.sql.Exec(`
INSERT INTO recordings (
  recording_id, created_at, title, audio_path, language, mode_key, status, submitted_at, error
) VALUES (?, ?, ?, ?, ?, ?, ?, NULL, '')
ON CONFLICT(recording_id) DO UPDATE SET
  created_at = excluded.created_at,
  title = excluded.title,
  audio_path = excluded.audio_path,
  language = excluded.language,
  mode_key = excluded.mode_key,
  status = ?,
  submitted_at = NULL,
  error = ''`,
		r.RecordingID,
		r.CreatedAt,
		r.Title,
		r.AudioPath,
		r.Language,
		r.ModeKey,
		StatusPending,
		StatusPending,
	)
	if err != nil {
		return fmt.Errorf("upsert pending %d: %w", r.RecordingID, err)
	}
	return nil
}

func (d *DB) MarkSubmitted(id int64, language, modeKey, audioPath string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := d.sql.Exec(`
UPDATE recordings
SET status = ?, language = ?, mode_key = ?, audio_path = ?, submitted_at = ?, error = ''
WHERE recording_id = ?`,
		StatusSubmitted,
		language,
		modeKey,
		audioPath,
		now,
		id,
	)
	if err != nil {
		return fmt.Errorf("mark submitted %d: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark submitted rows affected %d: %w", id, err)
	}
	if n == 0 {
		return fmt.Errorf("mark submitted %d: recording not found", id)
	}
	return nil
}

func (d *DB) MarkHarvested(id int64, swID, body, notePath string) error {
	res, err := d.sql.Exec(`
UPDATE recordings
SET status = ?, sw_id = ?, body = ?, note_path = ?, error = ''
WHERE recording_id = ?`,
		StatusHarvested,
		swID,
		body,
		notePath,
		id,
	)
	if err != nil {
		return fmt.Errorf("mark harvested %d: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark harvested rows affected %d: %w", id, err)
	}
	if n == 0 {
		return fmt.Errorf("mark harvested %d: recording not found", id)
	}
	return nil
}

func (d *DB) MarkPublished(id int64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := d.sql.Exec(`
UPDATE recordings
SET status = ?, published_at = ?
WHERE recording_id = ?`,
		StatusPublished,
		now,
		id,
	)
	if err != nil {
		return fmt.Errorf("mark published %d: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark published rows affected %d: %w", id, err)
	}
	if n == 0 {
		return fmt.Errorf("mark published %d: recording not found", id)
	}
	return nil
}

func (d *DB) GetByID(id int64) (*Recording, error) {
	row := d.sql.QueryRow(`SELECT`+recordingSelectColumns+` FROM recordings WHERE recording_id = ?`, id)
	r, err := scanRecording(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get recording %d: %w", id, err)
	}
	return &r, nil
}

func (d *DB) ListSubmittedAwaitingHarvest() ([]Recording, error) {
	return d.listByStatus(StatusSubmitted)
}

func (d *DB) ListHarvestedAwaitingPublish() ([]Recording, error) {
	return d.listByStatus(StatusHarvested)
}

func (d *DB) listByStatus(status Status) ([]Recording, error) {
	rows, err := d.sql.Query(`SELECT`+recordingSelectColumns+` FROM recordings WHERE status = ? ORDER BY recording_id`, status)
	if err != nil {
		return nil, fmt.Errorf("list recordings status %q: %w", status, err)
	}
	defer rows.Close()

	var out []Recording
	for rows.Next() {
		r, err := scanRecording(rows)
		if err != nil {
			return nil, fmt.Errorf("scan recording: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recordings status %q: %w", status, err)
	}
	return out, nil
}

func (d *DB) MarkError(id int64, errMsg string) error {
	res, err := d.sql.Exec(`
UPDATE recordings
SET status = ?, error = ?
WHERE recording_id = ?`,
		StatusError,
		errMsg,
		id,
	)
	if err != nil {
		return fmt.Errorf("mark error %d: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark error rows affected %d: %w", id, err)
	}
	if n == 0 {
		return fmt.Errorf("mark error %d: recording not found", id)
	}
	return nil
}

func (d *DB) StatusCounts() (map[Status]int, error) {
	rows, err := d.sql.Query(`SELECT status, COUNT(*) FROM recordings GROUP BY status`)
	if err != nil {
		return nil, fmt.Errorf("query status counts: %w", err)
	}
	defer rows.Close()

	out := make(map[Status]int)
	for rows.Next() {
		var status Status
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan status count: %w", err)
		}
		out[status] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate status counts: %w", err)
	}
	return out, nil
}

func (d *DB) Watermark() (int64, error) {
	var watermark sql.NullInt64
	err := d.sql.QueryRow(`SELECT MAX(recording_id) FROM recordings`).Scan(&watermark)
	if err != nil {
		return 0, fmt.Errorf("query watermark: %w", err)
	}
	if !watermark.Valid {
		return 0, nil
	}
	return watermark.Int64, nil
}

const metaWatermarkKey = "watermark"

// EffectiveWatermark returns the stored lazy cursor, else MAX(recording_id).
func (d *DB) EffectiveWatermark() (int64, error) {
	v, err := d.GetMeta(metaWatermarkKey)
	if err != nil {
		return 0, err
	}
	if v != "" {
		var id int64
		if _, err := fmt.Sscan(v, &id); err == nil {
			return id, nil
		}
	}
	return d.Watermark()
}

// AdvanceWatermark stores id when it is greater than the current cursor.
func (d *DB) AdvanceWatermark(id int64) error {
	cur, err := d.EffectiveWatermark()
	if err != nil {
		return err
	}
	if id <= cur {
		return nil
	}
	return d.SetMeta(metaWatermarkKey, fmt.Sprintf("%d", id))
}
