package sync

import (
	"context"
	"log/slog"
	"time"

	"github.com/jborkowski/lazy-notes/internal/config"
	"github.com/jborkowski/lazy-notes/internal/db"
	"github.com/jborkowski/lazy-notes/internal/publish"
	"github.com/jborkowski/lazy-notes/internal/superwhisper"
)

const (
	defaultHarvestWait  = 5 * time.Second
	defaultHarvestPolls = 12
	pendingNotePath     = ""
)

type harvestTarget struct {
	RecordingID int64
	Title       string
	Language    string
	ModeKey     string
	SubmittedAt time.Time
}

func processHarvestPublishBacklog(ctx context.Context, cfg *config.Config, database *db.DB, result *Result) {
	if !cfg.Publish.Enabled {
		return
	}

	max := cfg.Sync.MaxPerSync

	submitted, err := database.ListSubmittedAwaitingHarvest()
	if err != nil {
		slog.Error("list submitted awaiting harvest", "err", err)
		result.Errors++
	} else {
		for _, rec := range capRecordings(submitted, max) {
			if err := ctx.Err(); err != nil {
				return
			}
			target := recordingTarget(rec)
			harvestAndPublish(ctx, cfg, database, target, result)
		}
	}

	harvested, err := database.ListHarvestedAwaitingPublish()
	if err != nil {
		slog.Error("list harvested awaiting publish", "err", err)
		result.Errors++
		return
	}
	for _, rec := range capRecordings(harvested, max) {
		if err := ctx.Err(); err != nil {
			return
		}
		publishHarvested(ctx, cfg, database, rec, result)
	}
}

func harvestAndPublish(ctx context.Context, cfg *config.Config, database *db.DB, target harvestTarget, result *Result) {
	tr, err := superwhisper.WaitAndHarvest(ctx, target.ModeKey, target.SubmittedAt, defaultHarvestWait, defaultHarvestPolls)
	if err != nil {
		slog.Error("wait and harvest failed",
			"recording_id", target.RecordingID,
			"mode", target.ModeKey,
			"err", err,
		)
		result.Errors++
		return
	}
	if tr == nil {
		slog.Warn("harvest empty, will retry later",
			"recording_id", target.RecordingID,
			"mode", target.ModeKey,
		)
		return
	}

	body := tr.BestText()
	if body == "" {
		slog.Warn("harvest matched but text empty, will retry later",
			"recording_id", target.RecordingID,
			"sw_id", tr.ID,
		)
		return
	}

	if err := database.MarkHarvested(target.RecordingID, tr.ID, body, pendingNotePath); err != nil {
		slog.Error("mark harvested failed", "recording_id", target.RecordingID, "err", err)
		result.Errors++
		return
	}
	result.Harvested++

	rec := db.Recording{
		RecordingID: target.RecordingID,
		Title:       target.Title,
		Language:    target.Language,
		ModeKey:     target.ModeKey,
		Body:        body,
		SwID:        tr.ID,
	}
	publishHarvested(ctx, cfg, database, rec, result)
}

func publishHarvested(ctx context.Context, cfg *config.Config, database *db.DB, rec db.Recording, result *Result) {
	note := publish.Note{
		Title:       rec.Title,
		Body:        rec.Body,
		RecordingID: rec.RecordingID,
		Language:    rec.Language,
		ModeKey:     rec.ModeKey,
		SourceSW:    rec.SwID,
		Tag:         cfg.Publish.Tag,
	}

	notePath, err := publish.Publish(ctx, publish.Options{
		NotesDir:      cfg.Publish.NotesDir,
		MemoEnabled:   cfg.Publish.MemoEnabled,
		MemoBin:       cfg.Publish.MemoBin,
		MemoFolder:    cfg.Publish.MemoFolder,
		DriveEnabled:  cfg.Publish.DriveEnabled,
		DriveFolderID: cfg.Publish.DriveFolderID,
		GogBin:        cfg.GogBin(),
		GogAccount:    cfg.Publish.GogAccount,
	}, note)
	if err != nil {
		slog.Error("publish failed", "recording_id", rec.RecordingID, "err", err)
		result.Errors++
		return
	}

	if err := database.MarkHarvested(rec.RecordingID, rec.SwID, rec.Body, notePath); err != nil {
		slog.Error("update note path failed", "recording_id", rec.RecordingID, "err", err)
		result.Errors++
		return
	}
	if err := database.MarkPublished(rec.RecordingID); err != nil {
		slog.Error("mark published failed", "recording_id", rec.RecordingID, "err", err)
		result.Errors++
		return
	}

	result.Published++
	slog.Info("published recording",
		"recording_id", rec.RecordingID,
		"note_path", notePath,
		"sw_id", rec.SwID,
	)
}

func recordingTarget(rec db.Recording) harvestTarget {
	submittedAt := time.Now()
	if rec.SubmittedAt != nil {
		submittedAt = *rec.SubmittedAt
	}
	return harvestTarget{
		RecordingID: rec.RecordingID,
		Title:       rec.Title,
		Language:    rec.Language,
		ModeKey:     rec.ModeKey,
		SubmittedAt: submittedAt,
	}
}

func capRecordings(items []db.Recording, max int) []db.Recording {
	if max <= 0 || len(items) <= max {
		return items
	}
	return items[:max]
}
