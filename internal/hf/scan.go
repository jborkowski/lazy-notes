package hf

import (
	"context"
	"fmt"
)

type scanRow struct {
	RecordingID     int64   `json:"recording_id"`
	Title           string  `json:"title"`
	CreatedAt       string  `json:"created_at"`
	DurationSeconds float64 `json:"duration_seconds"`
	Transcription   string  `json:"transcription"`
	AudioPath       string  `json:"audio_path"`
	OriginalPath    string  `json:"original_path"`
}

// ScanShardMeta reads metadata rows from a local parquet shard using DuckDB.
//
// SQL:
//
//	SELECT recording_id, title, created_at, duration_seconds,
//	       coalesce(transcription, '') AS transcription,
//	       audio.path AS audio_path, audio_original.path AS original_path
//	FROM read_parquet(...)
//	WHERE duration_seconds >= ? AND recording_id > ?
func (c *Client) ScanShardMeta(ctx context.Context, localParquet string, minDuration float64, afterID int64) ([]Meta, error) {
	sql := fmt.Sprintf(`SELECT recording_id, title, created_at, duration_seconds,
coalesce(transcription, '') AS transcription,
audio.path AS audio_path, audio_original.path AS original_path
FROM read_parquet(%s)
WHERE duration_seconds >= %g AND recording_id > %d
ORDER BY recording_id`,
		sqlStringLiteral(localParquet), minDuration, afterID)

	var rows []scanRow
	if err := duckdbQueryRows(ctx, sql, &rows); err != nil {
		return nil, err
	}

	metas := make([]Meta, 0, len(rows))
	for _, r := range rows {
		audioPath := r.AudioPath
		preferOriginal := false
		if r.OriginalPath != "" {
			audioPath = r.OriginalPath
			preferOriginal = true
		}
		metas = append(metas, Meta{
			RecordingID:     r.RecordingID,
			Title:           r.Title,
			CreatedAt:       r.CreatedAt,
			DurationSeconds: r.DurationSeconds,
			Transcription:   r.Transcription,
			AudioPath:       audioPath,
			PreferOriginal:  preferOriginal,
		})
	}
	return metas, nil
}

// MaxRecordingID returns MAX(recording_id) in a local parquet shard.
func (c *Client) MaxRecordingID(ctx context.Context, localParquet string) (int64, error) {
	sql := fmt.Sprintf(`SELECT coalesce(max(recording_id), 0) AS max_id FROM read_parquet(%s)`,
		sqlStringLiteral(localParquet))
	var rows []struct {
		MaxID int64 `json:"max_id"`
	}
	if err := duckdbQueryRows(ctx, sql, &rows); err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	return rows[0].MaxID, nil
}

// ListNewMetas lists shards newest-first, ensures each is cached, scans for rows
// with recording_id > watermark and duration_seconds >= minDuration, skipping IDs
// present in any of the known maps.
func (c *Client) ListNewMetas(ctx context.Context, minDuration float64, watermark int64, known ...map[int64]struct{}) ([]Meta, error) {
	knownSet := make(map[int64]struct{})
	for _, m := range known {
		for id := range m {
			knownSet[id] = struct{}{}
		}
	}

	shards, err := c.ListShards(ctx)
	if err != nil {
		return nil, err
	}

	var out []Meta
	seen := make(map[int64]struct{})
	for _, shard := range shards {
		local, err := c.EnsureShard(ctx, shard.Path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", shard.Path, err)
		}
		maxID, err := c.MaxRecordingID(ctx, local)
		if err != nil {
			return nil, fmt.Errorf("max %s: %w", shard.Path, err)
		}
		if maxID <= watermark {
			// Newer shards sorted first; older ones cannot contain new IDs.
			break
		}
		rows, err := c.ScanShardMeta(ctx, local, minDuration, watermark)
		if err != nil {
			return nil, fmt.Errorf("scan %s: %w", shard.Path, err)
		}
		for _, m := range rows {
			if _, skip := knownSet[m.RecordingID]; skip {
				continue
			}
			if _, dup := seen[m.RecordingID]; dup {
				continue
			}
			seen[m.RecordingID] = struct{}{}
			out = append(out, m)
		}
	}
	return out, nil
}
