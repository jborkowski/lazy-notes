package sync

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jborkowski/lazy-notes/internal/config"
	"github.com/jborkowski/lazy-notes/internal/db"
	"github.com/jborkowski/lazy-notes/internal/hf"
	"github.com/jborkowski/lazy-notes/internal/language"
	"github.com/jborkowski/lazy-notes/internal/paths"
	"github.com/jborkowski/lazy-notes/internal/superwhisper"
)

// Result summarizes one sync pass.
type Result struct {
	Scanned   int
	Submitted int
	Harvested int
	Published int
	Skipped   int
	Errors    int
	Watermark int64
}

type metaJob struct {
	meta    hf.Meta
	parquet string
}

// Run performs one lazy incremental sync: only recordings newer than the
// watermark, newest shards first, optional caps, delete audio/shards after use.
func Run(ctx context.Context, cfg *config.Config, database *db.DB) (*Result, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	if database == nil {
		return nil, fmt.Errorf("database is nil")
	}

	result := &Result{}
	preferOriginal := cfg.AudioPrefer == "original"
	minDuration := cfg.Sync.MinDurationSeconds
	submitDelay := time.Duration(cfg.Sync.SubmitDelaySeconds * float64(time.Second))

	client := hf.NewClient(cfg.Dataset, tokenFromConfig(cfg), filepath.Join(paths.CacheDir(), "hf"))
	if err := client.VerifyAccess(ctx); err != nil {
		return nil, err
	}

	watermark, err := database.EffectiveWatermark()
	if err != nil {
		return nil, fmt.Errorf("watermark: %w", err)
	}
	result.Watermark = watermark

	// First run: jump to latest id, do not backfill history onto disk.
	if watermark == 0 && cfg.Sync.LazyStart {
		latest, err := bootstrapLatest(ctx, client, cfg.Sync.KeepShards)
		if err != nil {
			return nil, fmt.Errorf("lazy start: %w", err)
		}
		if err := database.AdvanceWatermark(latest); err != nil {
			return nil, err
		}
		result.Watermark = latest
		slog.Info("lazy start: watermark set to latest, no backfill", "watermark", latest)
		return result, nil
	}

	known, err := database.KnownIDs()
	if err != nil {
		return nil, fmt.Errorf("known ids: %w", err)
	}

	jobs, skipped, maxSeen, err := listNewMetas(ctx, client, minDuration, watermark, known, preferOriginal, cfg.Sync.KeepShards, cfg.Sync.MaxPerSync)
	if err != nil {
		return nil, err
	}
	result.Skipped = skipped

	if err := paths.EnsureDir(paths.AudioCacheDir()); err != nil {
		return nil, fmt.Errorf("ensure audio cache: %w", err)
	}

	fallback := fallbackLang(cfg)

	for _, job := range jobs {
		if err := ctx.Err(); err != nil {
			return result, err
		}

		result.Scanned++
		meta := job.meta

		ext := ".flac"
		if preferOriginal {
			ext = ".m4a"
		}
		audioPath := filepath.Join(paths.AudioCacheDir(), fmt.Sprintf("%d%s", meta.RecordingID, ext))

		rec := db.Recording{
			RecordingID: meta.RecordingID,
			CreatedAt:   meta.CreatedAt,
			Title:       meta.Title,
			AudioPath:   audioPath,
			Status:      db.StatusPending,
		}
		if err := database.UpsertPending(rec); err != nil {
			slog.Error("upsert pending failed", "recording_id", meta.RecordingID, "err", err)
			result.Errors++
			continue
		}

		if err := client.ExtractAudio(ctx, job.parquet, meta.RecordingID, preferOriginal, audioPath); err != nil {
			slog.Error("extract audio failed", "recording_id", meta.RecordingID, "err", err)
			_ = database.MarkError(meta.RecordingID, err.Error())
			result.Errors++
			continue
		}

		lang := language.Detect(meta.Transcription, audioPath, cfg.Languages, fallback)
		modeKey := cfg.ModeKey(lang)

		if err := superwhisper.Submit(audioPath, modeKey, submitDelay); err != nil {
			slog.Error("submit failed", "recording_id", meta.RecordingID, "mode", modeKey, "err", err)
			_ = database.MarkError(meta.RecordingID, err.Error())
			result.Errors++
			continue
		}

		if err := database.MarkSubmitted(meta.RecordingID, lang, modeKey, audioPath); err != nil {
			slog.Error("mark submitted failed", "recording_id", meta.RecordingID, "err", err)
			result.Errors++
			continue
		}

		if cfg.Publish.Enabled {
			harvestAndPublish(ctx, cfg, database, harvestTarget{
				RecordingID: meta.RecordingID,
				Title:       meta.Title,
				Language:    lang,
				ModeKey:     modeKey,
				SubmittedAt: time.Now(),
			}, result)
		}

		if !cfg.Sync.KeepAudio {
			_ = os.Remove(audioPath)
		}

		result.Submitted++
		if meta.RecordingID > maxSeen {
			maxSeen = meta.RecordingID
		}
		if err := database.AdvanceWatermark(meta.RecordingID); err != nil {
			slog.Error("advance watermark", "err", err)
		}
		result.Watermark = meta.RecordingID

		slog.Info("submitted recording",
			"recording_id", meta.RecordingID,
			"language", lang,
			"mode", modeKey,
		)
	}

	if maxSeen > watermark {
		_ = database.AdvanceWatermark(maxSeen)
		result.Watermark = maxSeen
	}

	if !cfg.Sync.KeepShards {
		cleanupParquets(jobs)
	}

	processHarvestPublishBacklog(ctx, cfg, database, result)

	slog.Info("sync pass finished",
		"scanned", result.Scanned,
		"submitted", result.Submitted,
		"harvested", result.Harvested,
		"published", result.Published,
		"skipped", result.Skipped,
		"errors", result.Errors,
		"watermark", result.Watermark,
	)

	return result, nil
}

func bootstrapLatest(ctx context.Context, client *hf.Client, keepShard bool) (int64, error) {
	shards, err := client.ListShards(ctx)
	if err != nil {
		return 0, err
	}
	if len(shards) == 0 {
		return 0, nil
	}
	// Newest shard only.
	local, err := client.EnsureShard(ctx, shards[0].Path)
	if err != nil {
		return 0, err
	}
	maxID, err := client.MaxRecordingID(ctx, local)
	if err != nil {
		return 0, err
	}
	if !keepShard {
		_ = os.Remove(local)
	}
	return maxID, nil
}

func cleanupParquets(jobs []metaJob) {
	seen := make(map[string]struct{})
	for _, j := range jobs {
		if j.parquet == "" {
			continue
		}
		if _, ok := seen[j.parquet]; ok {
			continue
		}
		seen[j.parquet] = struct{}{}
		_ = os.Remove(j.parquet)
	}
}

// listNewMetas downloads newest shards only until ids fall below the watermark.
func listNewMetas(
	ctx context.Context,
	client *hf.Client,
	minDuration float64,
	watermark int64,
	known map[int64]db.Status,
	preferOriginal bool,
	keepShards bool,
	maxPerSync int,
) ([]metaJob, int, int64, error) {
	knownFilter := make(map[int64]struct{})
	for id, status := range known {
		switch status {
		case db.StatusPending, db.StatusSubmitted, db.StatusHarvested, db.StatusPublished:
			knownFilter[id] = struct{}{}
		}
	}

	shards, err := client.ListShards(ctx)
	if err != nil {
		return nil, 0, watermark, err
	}

	var jobs []metaJob
	seen := make(map[int64]struct{})
	skipped := 0
	maxSeen := watermark
	downloaded := make([]string, 0)

	for _, shard := range shards {
		local, err := client.EnsureShard(ctx, shard.Path)
		if err != nil {
			return nil, 0, maxSeen, fmt.Errorf("%s: %w", shard.Path, err)
		}
		downloaded = append(downloaded, local)

		maxID, err := client.MaxRecordingID(ctx, local)
		if err != nil {
			return nil, 0, maxSeen, fmt.Errorf("max %s: %w", shard.Path, err)
		}
		if maxID <= watermark {
			// Older shards cannot contain newer ids (newest-first order).
			break
		}

		rows, err := client.ScanShardMeta(ctx, local, minDuration, watermark)
		if err != nil {
			return nil, 0, maxSeen, fmt.Errorf("scan %s: %w", shard.Path, err)
		}

		for _, m := range rows {
			if m.RecordingID > maxSeen {
				maxSeen = m.RecordingID
			}
			if _, skip := knownFilter[m.RecordingID]; skip {
				skipped++
				continue
			}
			if _, dup := seen[m.RecordingID]; dup {
				continue
			}
			seen[m.RecordingID] = struct{}{}
			m.PreferOriginal = preferOriginal
			jobs = append(jobs, metaJob{meta: m, parquet: local})
			if maxPerSync > 0 && len(jobs) >= maxPerSync {
				goto done
			}
		}
	}

done:
	if !keepShards {
		needed := make(map[string]struct{})
		for _, j := range jobs {
			needed[j.parquet] = struct{}{}
		}
		for _, p := range downloaded {
			if _, ok := needed[p]; !ok {
				_ = os.Remove(p)
			}
		}
	}

	return jobs, skipped, maxSeen, nil
}

func fallbackLang(cfg *config.Config) string {
	fb := cfg.Modes.Fallback
	switch fb {
	case cfg.Modes.PL:
		return "pl"
	case cfg.Modes.EN:
		return "en"
	case cfg.Modes.ES:
		return "es"
	}
	for _, lang := range []string{"pl", "en", "es"} {
		if strings.Contains(fb, lang) {
			return lang
		}
	}
	return "en"
}

func tokenFromConfig(cfg *config.Config) string {
	if cfg == nil {
		return hf.DefaultToken()
	}
	return hf.ResolveToken(cfg.HfTokenFile)
}
