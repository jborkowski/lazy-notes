// Package doctor runs readiness checks for lazy-notes (deps, config, auth, watchers).
package doctor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jborkowski/lazy-notes/internal/config"
	"github.com/jborkowski/lazy-notes/internal/db"
	"github.com/jborkowski/lazy-notes/internal/hf"
	"github.com/jborkowski/lazy-notes/internal/paths"
	"github.com/jborkowski/lazy-notes/internal/publish"
	"github.com/jborkowski/lazy-notes/internal/superwhisper"
)

// Severity classifies a check outcome.
type Severity string

const (
	OK   Severity = "ok"
	Warn Severity = "warn"
	Fail Severity = "fail"
	Skip Severity = "skip"
)

// Check is one doctor row.
type Check struct {
	Name     string
	Severity Severity
	Detail   string
	Fix      string
}

// Report aggregates checks.
type Report struct {
	Checks []Check
}

// Failed reports whether any check failed.
func (r Report) Failed() bool {
	for _, c := range r.Checks {
		if c.Severity == Fail {
			return true
		}
	}
	return false
}

// Warned reports whether any check warned (and none failed).
func (r Report) Warned() bool {
	if r.Failed() {
		return false
	}
	for _, c := range r.Checks {
		if c.Severity == Warn {
			return true
		}
	}
	return false
}

// Options controls optional network / heavy checks.
type Options struct {
	// SkipHFAccess skips live Hugging Face dataset access probe.
	SkipHFAccess bool
	ConfigPath   string
}

// Run executes readiness checks.
func Run(ctx context.Context, opts Options) Report {
	var r Report
	configPath := opts.ConfigPath
	if configPath == "" {
		configPath = paths.ConfigPath()
	}

	r.Checks = append(r.Checks, checkConfigFile(configPath))

	cfg, err := config.Load(configPath)
	if err != nil {
		r.Checks = append(r.Checks, Check{
			Name:     "config.parse",
			Severity: Fail,
			Detail:   err.Error(),
			Fix:      fmt.Sprintf("fix %s or run: lazy-notes onboard", configPath),
		})
		// Still check binaries that do not need config.
		r.Checks = append(r.Checks, checkBinary("duckdb", "duckdb", "brew install duckdb")...)
		r.Checks = append(r.Checks, checkBinary("ffmpeg", "ffmpeg", "brew install ffmpeg")...)
		r.Checks = append(r.Checks, checkSuperwhisper())
		return r
	}

	r.Checks = append(r.Checks, Check{
		Name:     "config.parse",
		Severity: OK,
		Detail:   configPath,
	})

	r.Checks = append(r.Checks, checkNotesDir(cfg.NotesDir()))
	r.Checks = append(r.Checks, checkStateDB())
	r.Checks = append(r.Checks, checkBinary("duckdb", "duckdb", "brew install duckdb")...)
	r.Checks = append(r.Checks, checkBinary("ffmpeg", "ffmpeg", "brew install ffmpeg")...)
	r.Checks = append(r.Checks, checkBinary("hf", "hf", "brew install hf")...)
	r.Checks = append(r.Checks, checkSuperwhisper())
	r.Checks = append(r.Checks, checkModes(cfg))
	r.Checks = append(r.Checks, checkHFToken(cfg))

	if !opts.SkipHFAccess {
		r.Checks = append(r.Checks, checkHFAccess(ctx, cfg))
	} else {
		r.Checks = append(r.Checks, Check{
			Name:     "hf.access",
			Severity: Skip,
			Detail:   "skipped (--offline)",
		})
	}

	r.Checks = append(r.Checks, checkMemo(cfg))
	r.Checks = append(r.Checks, checkGog(cfg))
	r.Checks = append(r.Checks, checkWatch(cfg)...)

	return r
}

func checkConfigFile(path string) Check {
	if st, err := os.Stat(path); err != nil {
		return Check{
			Name:     "config.file",
			Severity: Fail,
			Detail:   err.Error(),
			Fix:      "lazy-notes onboard   # or: lazy-notes setup",
		}
	} else if st.IsDir() {
		return Check{
			Name:     "config.file",
			Severity: Fail,
			Detail:   path + " is a directory",
			Fix:      "remove it and run: lazy-notes onboard",
		}
	}
	return Check{Name: "config.file", Severity: OK, Detail: path}
}

func checkNotesDir(dir string) Check {
	if dir == "" {
		return Check{
			Name:     "publish.notes_dir",
			Severity: Fail,
			Detail:   "empty",
			Fix:      "set publish.notes_dir in config.toml",
		}
	}
	if err := paths.EnsureDir(dir); err != nil {
		return Check{
			Name:     "publish.notes_dir",
			Severity: Fail,
			Detail:   err.Error(),
			Fix:      "fix permissions or publish.notes_dir path",
		}
	}
	probe := filepath.Join(dir, ".lazy-notes-doctor-write")
	if err := os.WriteFile(probe, []byte("ok"), 0o644); err != nil {
		return Check{
			Name:     "publish.notes_dir",
			Severity: Fail,
			Detail:   "not writable: " + err.Error(),
			Fix:      "chmod/chown " + dir,
		}
	}
	_ = os.Remove(probe)
	return Check{Name: "publish.notes_dir", Severity: OK, Detail: dir}
}

func checkStateDB() Check {
	database, err := db.Open(paths.DBPath())
	if err != nil {
		return Check{
			Name:     "state.db",
			Severity: Fail,
			Detail:   err.Error(),
			Fix:      "ensure LAZY_NOTES_STATE_DIR is writable",
		}
	}
	_ = database.Close()
	return Check{Name: "state.db", Severity: OK, Detail: paths.DBPath()}
}

func checkBinary(name, bin, fix string) []Check {
	path, err := exec.LookPath(bin)
	if err != nil {
		return []Check{{
			Name:     "bin." + name,
			Severity: Fail,
			Detail:   "not found on PATH",
			Fix:      fix,
		}}
	}
	return []Check{{Name: "bin." + name, Severity: OK, Detail: path}}
}

func checkSuperwhisper() Check {
	if path, ok := superwhisper.CLIPath(); ok {
		return Check{Name: "bin.superwhisper", Severity: OK, Detail: path}
	}
	return Check{
		Name:     "bin.superwhisper",
		Severity: Fail,
		Detail:   "not found",
		Fix:      "brew reinstall jborkowski/lazy-notes/lazy-notes   # or: lazy-notes setup",
	}
}

func checkModes(cfg *config.Config) Check {
	dir := filepath.Join(paths.ConfigDir(), "modes")
	// Modes may live under SuperWhisper's own config; we only verify prompts resolve.
	missing := 0
	for _, lang := range []string{"pl", "en", "es", "auto"} {
		if _, err := cfg.ResolvePrompt(lang, paths.ConfigDir()); err != nil {
			missing++
		}
	}
	if missing > 0 {
		return Check{
			Name:     "modes.prompts",
			Severity: Warn,
			Detail:   fmt.Sprintf("%d prompt(s) unresolved under %s", missing, paths.ConfigDir()),
			Fix:      "lazy-notes setup --force",
		}
	}
	_ = dir
	return Check{
		Name:     "modes.prompts",
		Severity: OK,
		Detail:   "pl/en/es/auto prompts OK",
	}
}

func checkHFToken(cfg *config.Config) Check {
	if token := hf.ResolveToken(cfg.HfTokenFile); token != "" {
		detail := hf.TokenSource()
		if detail == "" && cfg.HfTokenFile != "" {
			detail = "file:" + paths.Expand(cfg.HfTokenFile)
		}
		return Check{
			Name:     "hf.token",
			Severity: OK,
			Detail:   detail,
		}
	}
	return Check{
		Name:     "hf.token",
		Severity: Fail,
		Detail:   "no token found",
		Fix:      "echo hf_... > ~/.config/lazy-notes/hf_token && chmod 600 ~/.config/lazy-notes/hf_token",
	}
}

func checkHFAccess(ctx context.Context, cfg *config.Config) Check {
	token := hf.ResolveToken(cfg.HfTokenFile)
	client := hf.NewClient(cfg.Dataset, token, filepath.Join(paths.CacheDir(), "hf"))
	if err := client.VerifyAccess(ctx); err != nil {
		return Check{
			Name:     "hf.access",
			Severity: Fail,
			Detail:   err.Error(),
			Fix:      "hf auth login   # or fix ~/.config/lazy-notes/hf_token",
		}
	}
	return Check{Name: "hf.access", Severity: OK, Detail: cfg.Dataset}
}

func checkMemo(cfg *config.Config) Check {
	if !cfg.Publish.Enabled || !cfg.Publish.MemoEnabled {
		return Check{
			Name:     "bin.memo",
			Severity: Skip,
			Detail:   "memo publish disabled",
		}
	}
	path, err := publish.MemoBinPath(cfg.MemoBin())
	if err != nil {
		return Check{
			Name:     "bin.memo",
			Severity: Fail,
			Detail:   err.Error(),
			Fix:      "brew tap antoniorodr/memo && brew install antoniorodr/memo/memo",
		}
	}
	return Check{Name: "bin.memo", Severity: OK, Detail: path}
}

func checkGog(cfg *config.Config) Check {
	need := cfg.Publish.DriveEnabled ||
		(cfg.Watch.DriveEnabled && cfg.Watch.DriveFolderID != "")
	if !need {
		return Check{
			Name:     "bin.gog",
			Severity: Skip,
			Detail:   "Drive upload/watch via gog not enabled",
		}
	}
	bin := cfg.GogBin()
	path, err := exec.LookPath(bin)
	if err != nil {
		return Check{
			Name:     "bin.gog",
			Severity: Fail,
			Detail:   "not found on PATH",
			Fix:      "brew tap openclaw/tap && brew install openclaw/tap/gogcli",
		}
	}
	if cfg.Publish.DriveEnabled && cfg.Publish.DriveFolderID == "" {
		return Check{
			Name:     "bin.gog",
			Severity: Warn,
			Detail:   path + " (publish.drive_folder_id empty)",
			Fix:      "set publish.drive_folder_id in config.toml",
		}
	}
	return Check{Name: "bin.gog", Severity: OK, Detail: path}
}

func checkWatch(cfg *config.Config) []Check {
	var out []Check
	if !cfg.Watch.AppleNotesEnabled {
		out = append(out, Check{
			Name:     "watch.apple_notes",
			Severity: Skip,
			Detail:   "disabled",
		})
	} else {
		dbPath := cfg.AppleNotesDB()
		if st, err := os.Stat(dbPath); err != nil {
			out = append(out, Check{
				Name:     "watch.apple_notes",
				Severity: Fail,
				Detail:   err.Error(),
				Fix:      "open Apple Notes once, or set watch.apple_notes_db",
			})
		} else if st.IsDir() {
			out = append(out, Check{
				Name:     "watch.apple_notes",
				Severity: Fail,
				Detail:   "path is a directory",
				Fix:      "point watch.apple_notes_db at NoteStore.sqlite",
			})
		} else {
			out = append(out, Check{
				Name:     "watch.apple_notes",
				Severity: OK,
				Detail:   dbPath,
			})
		}
	}

	if !cfg.Watch.DriveEnabled {
		out = append(out, Check{
			Name:     "watch.drive",
			Severity: Skip,
			Detail:   "disabled",
		})
		return out
	}

	if cfg.Watch.DriveLocalDir == "" && cfg.Watch.DriveFolderID == "" {
		out = append(out, Check{
			Name:     "watch.drive",
			Severity: Fail,
			Detail:   "drive_enabled but neither drive_local_dir nor drive_folder_id set",
			Fix:      "set watch.drive_local_dir and/or watch.drive_folder_id",
		})
		return out
	}

	if dir := cfg.DriveLocalDir(); dir != "" {
		if st, err := os.Stat(dir); err != nil {
			out = append(out, Check{
				Name:     "watch.drive_local",
				Severity: Fail,
				Detail:   err.Error(),
				Fix:      "create the folder or fix watch.drive_local_dir",
			})
		} else if !st.IsDir() {
			out = append(out, Check{
				Name:     "watch.drive_local",
				Severity: Fail,
				Detail:   "not a directory",
				Fix:      "fix watch.drive_local_dir",
			})
		} else {
			out = append(out, Check{
				Name:     "watch.drive_local",
				Severity: OK,
				Detail:   dir,
			})
		}
	}

	if id := strings.TrimSpace(cfg.Watch.DriveFolderID); id != "" {
		out = append(out, Check{
			Name:     "watch.drive_folder",
			Severity: OK,
			Detail:   id,
		})
	}

	return out
}

