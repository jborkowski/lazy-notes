package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jborkowski/lazy-notes/internal/config"
	"github.com/jborkowski/lazy-notes/internal/db"
	lnpublish "github.com/jborkowski/lazy-notes/internal/publish"
	"github.com/jborkowski/lazy-notes/internal/superwhisper"
	"github.com/spf13/cobra"
)

const (
	publishHarvestWait  = 5 * time.Second
	publishHarvestPolls = 12
)

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Harvest SuperWhisper transcripts and publish notes",
	Long:  "Processes the backlog: harvest text for submitted recordings, then publish harvested notes to notes_dir, Apple Notes via memo, and optionally Google Drive via gog.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return exitErr(err)
		}

		if !cfg.Publish.Enabled {
			fmt.Fprintln(os.Stdout, "publish disabled in config (publish.enabled = false)")
			return nil
		}

		database, err := openDB()
		if err != nil {
			return exitErr(err)
		}
		defer database.Close()

		result, err := runPublishBacklog(cmd.Context(), cfg, database)
		if err != nil {
			return exitErr(err)
		}
		printPublishResult(result)
		return nil
	},
}

type publishResult struct {
	Harvested int
	Published int
	Errors    int
}

type publishTarget struct {
	RecordingID int64
	Title       string
	Language    string
	ModeKey     string
	SubmittedAt time.Time
}

func runPublishBacklog(ctx context.Context, cfg *config.Config, database *db.DB) (*publishResult, error) {
	result := &publishResult{}
	max := cfg.Sync.MaxPerSync

	submitted, err := database.ListSubmittedAwaitingHarvest()
	if err != nil {
		return nil, fmt.Errorf("list submitted: %w", err)
	}
	for _, rec := range capPublishRecordings(submitted, max) {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		harvestAndPublishRecording(ctx, cfg, database, publishTargetFromRecording(rec), result)
	}

	harvested, err := database.ListHarvestedAwaitingPublish()
	if err != nil {
		return nil, fmt.Errorf("list harvested: %w", err)
	}
	for _, rec := range capPublishRecordings(harvested, max) {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		publishRecording(ctx, cfg, database, rec, result)
	}

	return result, nil
}

func harvestAndPublishRecording(ctx context.Context, cfg *config.Config, database *db.DB, target publishTarget, result *publishResult) {
	tr, err := superwhisper.WaitAndHarvest(ctx, target.ModeKey, target.SubmittedAt, publishHarvestWait, publishHarvestPolls)
	if err != nil {
		result.Errors++
		return
	}
	if tr == nil {
		return
	}

	body := tr.BestText()
	if body == "" {
		return
	}

	if err := database.MarkHarvested(target.RecordingID, tr.ID, body, ""); err != nil {
		result.Errors++
		return
	}
	result.Harvested++

	publishRecording(ctx, cfg, database, db.Recording{
		RecordingID: target.RecordingID,
		Title:       target.Title,
		Language:    target.Language,
		ModeKey:     target.ModeKey,
		Body:        body,
		SwID:        tr.ID,
	}, result)
}

func publishRecording(ctx context.Context, cfg *config.Config, database *db.DB, rec db.Recording, result *publishResult) {
	notePath, err := lnpublish.Publish(ctx, lnpublish.Options{
		NotesDir:      cfg.NotesDir(),
		MemoEnabled:   cfg.Publish.MemoEnabled,
		MemoBin:       cfg.MemoBin(),
		MemoFolder:    cfg.Publish.MemoFolder,
		DriveEnabled:  cfg.Publish.DriveEnabled,
		DriveFolderID: cfg.Publish.DriveFolderID,
		GogBin:        cfg.GogBin(),
		GogAccount:    cfg.Publish.GogAccount,
	}, lnpublish.Note{
		Title:       rec.Title,
		Body:        rec.Body,
		RecordingID: rec.RecordingID,
		Language:    rec.Language,
		ModeKey:     rec.ModeKey,
		SourceSW:    rec.SwID,
		Tag:         cfg.Publish.Tag,
	})
	if err != nil {
		result.Errors++
		return
	}

	if err := database.MarkHarvested(rec.RecordingID, rec.SwID, rec.Body, notePath); err != nil {
		result.Errors++
		return
	}
	if err := database.MarkPublished(rec.RecordingID); err != nil {
		result.Errors++
		return
	}
	result.Published++
}

func publishTargetFromRecording(rec db.Recording) publishTarget {
	submittedAt := time.Now()
	if rec.SubmittedAt != nil {
		submittedAt = *rec.SubmittedAt
	}
	return publishTarget{
		RecordingID: rec.RecordingID,
		Title:       rec.Title,
		Language:    rec.Language,
		ModeKey:     rec.ModeKey,
		SubmittedAt: submittedAt,
	}
}

func capPublishRecordings(items []db.Recording, max int) []db.Recording {
	if max <= 0 || len(items) <= max {
		return items
	}
	return items[:max]
}

func printPublishResult(result *publishResult) {
	fmt.Fprintf(os.Stdout, "harvested=%d published=%d errors=%d\n",
		result.Harvested, result.Published, result.Errors)
}

func init() {
	rootCmd.AddCommand(publishCmd)
}
