package sync

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jborkowski/lazy-notes/internal/config"
	"github.com/jborkowski/lazy-notes/internal/db"
	"github.com/jborkowski/lazy-notes/internal/paths"
	"github.com/jborkowski/lazy-notes/internal/superwhisper"
	"github.com/jborkowski/lazy-notes/internal/voicememos"
)

// RunVoiceMemos scans the export inbox, upserts pending rows, and submits to SuperWhisper.
// It is a no-op when voice_memos.enabled is false.
func RunVoiceMemos(ctx context.Context, cfg *config.Config, database *db.DB, result *Result) error {
	if cfg == nil || database == nil {
		return fmt.Errorf("config and database are required")
	}
	if result == nil {
		result = &Result{}
	}
	if !cfg.VoiceMemos.Enabled {
		return nil
	}

	src := cfg.VoiceMemos.Source
	if src == "" {
		src = "export_inbox"
	}
	if src != "export_inbox" {
		return fmt.Errorf("unsupported voice_memos.source %q (MVP supports export_inbox only)", src)
	}

	dir := cfg.VoiceMemosExportDir()
	if err := paths.EnsureDir(dir); err != nil {
		return fmt.Errorf("ensure voice memos inbox: %w", err)
	}
	if err := paths.EnsureDir(paths.AudioCacheDir()); err != nil {
		return fmt.Errorf("ensure audio cache: %w", err)
	}

	items, err := voicememos.Scan(voicememos.ScanOptions{
		Dir:                dir,
		Extensions:         cfg.VoiceMemos.Extensions,
		MinDurationSeconds: cfg.VoiceMemos.MinDurationSeconds,
		DurationOf:         voicememos.ProbeDuration,
	})
	if err != nil {
		return err
	}

	lang := cfg.VoiceMemos.Language
	if lang == "" {
		lang = "auto"
	}
	modeKey := cfg.ModeKey(lang)
	submitDelay := time.Duration(cfg.Sync.SubmitDelaySeconds * float64(time.Second))
	max := cfg.Sync.MaxPerSync
	submitted := 0
	var maxMtimeNS int64

	for _, item := range items {
		if err := ctx.Err(); err != nil {
			return err
		}
		ns := item.ModTime.UTC().UnixNano()
		if ns > maxMtimeNS {
			maxMtimeNS = ns
		}

		existing, err := database.GetBySourceExternalID(db.SourceVoiceMemo, item.ExternalID)
		if err != nil {
			slog.Error("voice memo lookup failed", "err", err)
			result.Errors++
			continue
		}
		if existing != nil {
			switch existing.Status {
			case db.StatusSubmitted, db.StatusHarvested, db.StatusPublished:
				result.Skipped++
				continue
			}
		}

		if max > 0 && submitted >= max {
			result.Skipped++
			continue
		}

		staged, err := voicememos.Stage(item.Path, paths.AudioCacheDir(), item.ExternalID, item.Ext)
		if err != nil {
			slog.Error("stage voice memo failed", "err", err)
			result.Errors++
			continue
		}

		rec := db.Recording{
			CreatedAt:  item.ModTime.UTC().Format(time.RFC3339),
			Title:      item.Title,
			AudioPath:  staged,
			Language:   lang,
			ModeKey:    modeKey,
			ExternalID: item.ExternalID,
			Status:     db.StatusPending,
		}
		if err := database.UpsertVoiceMemoPending(rec); err != nil {
			slog.Error("upsert voice memo failed", "err", err)
			result.Errors++
			continue
		}

		stored, err := database.GetBySourceExternalID(db.SourceVoiceMemo, item.ExternalID)
		if err != nil || stored == nil {
			slog.Error("reload voice memo failed", "err", err)
			result.Errors++
			continue
		}
		if stored.Status != db.StatusPending && stored.Status != db.StatusError {
			result.Skipped++
			continue
		}

		result.Scanned++
		if err := submitVoiceMemo(ctx, cfg, database, stored, staged, lang, modeKey, submitDelay, result); err != nil {
			continue
		}
		submitted++
	}

	if maxMtimeNS > 0 {
		if err := database.AdvanceVoiceMemoWatermarkMTimeNS(maxMtimeNS); err != nil {
			slog.Error("advance voice memo watermark", "err", err)
		}
	}

	// Retry pending rows that still have an audio path (e.g. prior submit failure).
	if max == 0 || submitted < max {
		pending, err := database.ListPendingBySource(db.SourceVoiceMemo)
		if err != nil {
			slog.Error("list pending voice memos", "err", err)
			result.Errors++
		} else {
			for _, rec := range pending {
				if err := ctx.Err(); err != nil {
					return err
				}
				if max > 0 && submitted >= max {
					break
				}
				if rec.AudioPath == "" {
					continue
				}
				if _, err := os.Stat(rec.AudioPath); err != nil {
					continue
				}
				recLang := rec.Language
				if recLang == "" {
					recLang = lang
				}
				recMode := rec.ModeKey
				if recMode == "" {
					recMode = modeKey
				}
				result.Scanned++
				if err := submitVoiceMemo(ctx, cfg, database, &rec, rec.AudioPath, recLang, recMode, submitDelay, result); err != nil {
					continue
				}
				submitted++
			}
		}
	}

	return nil
}

func submitVoiceMemo(
	ctx context.Context,
	cfg *config.Config,
	database *db.DB,
	rec *db.Recording,
	audioPath, lang, modeKey string,
	submitDelay time.Duration,
	result *Result,
) error {
	if err := superwhisper.Submit(audioPath, modeKey, submitDelay); err != nil {
		slog.Error("voice memo submit failed", "recording_id", rec.RecordingID, "mode", modeKey, "err", err)
		_ = database.MarkError(rec.RecordingID, err.Error())
		result.Errors++
		return err
	}
	if err := database.MarkSubmitted(rec.RecordingID, lang, modeKey, audioPath); err != nil {
		slog.Error("mark voice memo submitted failed", "recording_id", rec.RecordingID, "err", err)
		result.Errors++
		return err
	}
	if cfg.Publish.Enabled {
		harvestAndPublish(ctx, cfg, database, harvestTarget{
			RecordingID: rec.RecordingID,
			Title:       rec.Title,
			Language:    lang,
			ModeKey:     modeKey,
			SubmittedAt: time.Now(),
		}, result)
	}
	if !cfg.Sync.KeepAudio {
		_ = os.Remove(audioPath)
	}
	result.Submitted++
	slog.Info("submitted voice memo",
		"recording_id", rec.RecordingID,
		"language", lang,
		"mode", modeKey,
	)
	return nil
}
