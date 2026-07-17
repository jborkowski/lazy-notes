package publish

import (
	"context"

	"github.com/jborkowski/lazy-notes/internal/config"
)

// Options controls local markdown, Apple Notes (memo), and Google Drive export.
type Options struct {
	NotesDir string

	MemoEnabled bool
	MemoBin     string
	MemoFolder  string

	DriveEnabled  bool
	DriveFolderID string
	GogBin        string
	GogAccount    string
}

// OptionsFromConfig builds publish Options from lazy-notes config.
func OptionsFromConfig(cfg *config.Config) Options {
	if cfg == nil {
		return Options{}
	}
	return Options{
		NotesDir:      cfg.NotesDir(),
		MemoEnabled:   cfg.Publish.MemoEnabled,
		MemoBin:       cfg.MemoBin(),
		MemoFolder:    cfg.Publish.MemoFolder,
		DriveEnabled:  cfg.Publish.DriveEnabled,
		DriveFolderID: cfg.Publish.DriveFolderID,
		GogBin:        cfg.GogBin(),
		GogAccount:    cfg.Publish.GogAccount,
	}
}

// Publish always writes a local markdown file and optionally pushes to Apple Notes
// and/or Google Drive (via gog-cli).
func Publish(ctx context.Context, opts Options, n Note) (notePath string, err error) {
	notePath, err = WriteMarkdown(opts.NotesDir, n)
	if err != nil {
		return "", err
	}

	if opts.MemoEnabled {
		if err := PushMemo(ctx, opts.MemoBin, opts.MemoFolder, n); err != nil {
			return notePath, err
		}
	}

	if opts.DriveEnabled {
		if err := PushDrive(ctx, opts.GogBin, opts.GogAccount, opts.DriveFolderID, notePath); err != nil {
			return notePath, err
		}
	}

	return notePath, nil
}
