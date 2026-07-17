// Package onboard runs a numbered, step-by-step first-run flow.
package onboard

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/jborkowski/lazy-notes/internal/config"
	"github.com/jborkowski/lazy-notes/internal/doctor"
	"github.com/jborkowski/lazy-notes/internal/hf"
	"github.com/jborkowski/lazy-notes/internal/paths"
	"github.com/jborkowski/lazy-notes/internal/superwhisper"
)

// Options controls onboarding behaviour.
type Options struct {
	ForceModes   bool
	SkipHFAccess bool
	Stdout       io.Writer
	Stderr       io.Writer
}

// Result summarizes onboarding.
type Result struct {
	ConfigPath string
	Doctor     doctor.Report
}

// Run executes the step-by-step onboarding checklist.
func Run(ctx context.Context, opts Options) (*Result, error) {
	out := opts.Stdout
	if out == nil {
		out = os.Stdout
	}
	errOut := opts.Stderr
	if errOut == nil {
		errOut = os.Stderr
	}

	configPath := paths.ConfigPath()
	dataDir := paths.DataDir()
	step := 0
	next := func(title string) {
		step++
		fmt.Fprintf(out, "\n==> Step %d: %s\n", step, title)
	}

	fmt.Fprintln(out, "lazy-notes onboarding")
	fmt.Fprintln(out, "Follow each step; doctor runs at the end.")

	// 1. Config
	next("Config file")
	if err := config.EnsureExample(configPath, dataDir); err != nil {
		return nil, fmt.Errorf("ensure config: %w", err)
	}
	fmt.Fprintf(out, "    config: %s\n", configPath)
	fmt.Fprintf(out, "    data:   %s\n", dataDir)

	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	// 2. Directories
	next("State and notes directories")
	if err := paths.EnsureDir(paths.StateDir()); err != nil {
		return nil, fmt.Errorf("state dir: %w", err)
	}
	if err := paths.EnsureDir(paths.CacheDir()); err != nil {
		return nil, fmt.Errorf("cache dir: %w", err)
	}
	if err := paths.EnsureDir(cfg.NotesDir()); err != nil {
		return nil, fmt.Errorf("notes dir: %w", err)
	}
	fmt.Fprintf(out, "    state: %s\n", paths.StateDir())
	fmt.Fprintf(out, "    cache: %s\n", paths.CacheDir())
	fmt.Fprintf(out, "    notes: %s\n", cfg.NotesDir())

	// 3. SuperWhisper CLI
	next("SuperWhisper CLI")
	if err := superwhisper.EnsureCLI(ctx); err != nil {
		return nil, fmt.Errorf("superwhisper CLI: %w", err)
	}
	if path, ok := superwhisper.CLIPath(); ok {
		fmt.Fprintf(out, "    binary: %s\n", path)
	}

	// 4. Modes + prompts
	next("Note modes and prompts")
	installed, err := superwhisper.InstallModes(cfg, paths.ConfigDir(), opts.ForceModes)
	if err != nil {
		return nil, fmt.Errorf("install modes: %w", err)
	}
	if len(installed) > 0 {
		fmt.Fprintf(out, "    installed modes: %v\n", installed)
	} else {
		fmt.Fprintln(out, "    modes already present (use --force to overwrite)")
	}

	// 5. HF token
	next("Hugging Face auth")
	token := resolveToken(cfg)
	if src := hf.TokenSource(); src != "" {
		fmt.Fprintf(out, "    token source: %s\n", src)
	} else if token != "" {
		fmt.Fprintln(out, "    token source: config hf_token_file")
	} else {
		canonical := filepath.Join(paths.ConfigDir(), "hf_token")
		fmt.Fprintln(errOut, "    no HF token found yet")
		fmt.Fprintf(errOut, "    write one to: %s\n", canonical)
		fmt.Fprintln(errOut, "      mkdir -p ~/.config/lazy-notes && chmod 700 ~/.config/lazy-notes")
		fmt.Fprintln(errOut, "      echo 'hf_...' > ~/.config/lazy-notes/hf_token && chmod 600 ~/.config/lazy-notes/hf_token")
		fmt.Fprintln(errOut, "    or: hf auth login")
	}
	fmt.Fprintf(out, "    dataset: %s\n", cfg.Dataset)

	// 6. Publish targets
	next("Publish targets (Apple Notes / Drive)")
	fmt.Fprintf(out, "    publish.enabled:     %v\n", cfg.Publish.Enabled)
	fmt.Fprintf(out, "    memo_enabled:        %v  folder=%q\n", cfg.Publish.MemoEnabled, cfg.Publish.MemoFolder)
	fmt.Fprintf(out, "    drive_enabled:       %v  folder_id=%q\n", cfg.Publish.DriveEnabled, cfg.Publish.DriveFolderID)
	if cfg.Publish.DriveEnabled {
		fmt.Fprintln(out, "    Drive auth (once):")
		fmt.Fprintln(out, "      gog auth credentials set ~/Downloads/client_secret_….json")
		fmt.Fprintln(out, "      gog auth add you@example.com --services drive")
	} else {
		fmt.Fprintln(out, "    tip: set publish.drive_enabled + drive_folder_id to upload via gog")
	}

	// 7. Voice Memos inbox + optional wake watchers
	next("Voice Memos inbox and optional wake watchers")
	fmt.Fprintf(out, "    voice_memos.enabled: %v  export_dir=%q\n",
		cfg.VoiceMemos.Enabled, cfg.VoiceMemosExportDir())
	fmt.Fprintln(out, "    HF dataset + Voice Memos export inbox → SuperWhisper → notes")
	fmt.Fprintln(out, "    NoteStore / Drive watch = sync wake only (not Voice Memos ingest)")
	if cfg.VoiceMemos.Enabled {
		if err := paths.EnsureDir(cfg.VoiceMemosExportDir()); err != nil {
			fmt.Fprintf(errOut, "    warn: could not create inbox: %v\n", err)
		} else {
			fmt.Fprintf(out, "    drop finished .m4a files into: %s\n", cfg.VoiceMemosExportDir())
		}
	} else {
		fmt.Fprintln(out, "    tip: set voice_memos.enabled = true to ingest Voice Memos.app exports")
	}
	fmt.Fprintf(out, "    apple_notes_enabled: %v  (NoteStore wake, not Voice Memos)\n", cfg.Watch.AppleNotesEnabled)
	fmt.Fprintf(out, "    drive watch:         %v  local=%q folder_id=%q\n",
		cfg.Watch.DriveEnabled, cfg.Watch.DriveLocalDir, cfg.Watch.DriveFolderID)
	if !cfg.Watch.AppleNotesEnabled && !cfg.Watch.DriveEnabled && !cfg.VoiceMemos.Enabled {
		fmt.Fprintln(out, "    tip: enable voice_memos and/or watch.* for reactive sync")
	}

	// 8. Doctor
	next("Doctor (readiness check)")
	report := doctor.Run(ctx, doctor.Options{
		SkipHFAccess: opts.SkipHFAccess || token == "",
		ConfigPath:   configPath,
	})
	doctor.WriteReport(out, report)

	// 9. Next actions
	next("Next actions")
	fmt.Fprintf(out, "    1. Edit config if needed:  %s\n", configPath)
	if token == "" {
		fmt.Fprintln(out, "    2. Add HF token, then:      lazy-notes doctor")
		fmt.Fprintln(out, "    3. First sync:              lazy-notes sync   # or: make sync")
		fmt.Fprintln(out, "    4. Start daemon:            make start")
	} else if report.Failed() {
		fmt.Fprintln(out, "    2. Fix doctor failures, then: lazy-notes doctor")
		fmt.Fprintln(out, "    3. First sync:              lazy-notes sync")
		fmt.Fprintln(out, "    4. Start daemon:            make start")
	} else {
		fmt.Fprintln(out, "    2. First sync:              lazy-notes sync   # or: make sync")
		fmt.Fprintln(out, "    3. Publish backlog:         lazy-notes publish")
		fmt.Fprintln(out, "    4. Start daemon:            make start")
	}
	fmt.Fprintln(out, "    Re-check anytime:           lazy-notes doctor")

	return &Result{ConfigPath: configPath, Doctor: report}, nil
}

func resolveToken(cfg *config.Config) string {
	if cfg == nil {
		return hf.DefaultToken()
	}
	return hf.ResolveToken(cfg.HfTokenFile)
}
