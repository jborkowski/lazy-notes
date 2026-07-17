// Package config loads lazy-notes TOML configuration, resolves models and prompts,
// and builds SuperWhisper custom mode specs.
//
// Requires github.com/pelletier/go-toml/v2 (add to go.mod: go get github.com/pelletier/go-toml/v2).
package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/jborkowski/lazy-notes/internal/paths"
	"github.com/pelletier/go-toml/v2"
)

// Config is the root TOML configuration.
type Config struct {
	Dataset     string        `toml:"dataset"`
	AudioPrefer string        `toml:"audio_prefer"`
	// HfTokenFile optional path to a file containing the HF token (private datasets).
	// Env HF_TOKEN / hf auth login still take precedence via hf.DefaultToken unless
	// only this file is set — sync passes file contents when DefaultToken is empty.
	HfTokenFile string        `toml:"hf_token_file"`
	Languages   []string      `toml:"languages"`
	Sync        SyncConfig    `toml:"sync"`
	Modes       ModesConfig   `toml:"modes"`
	Model       ModelConfig   `toml:"model"`
	Prompts     PromptsConfig `toml:"prompts"`
	Publish     PublishConfig `toml:"publish"`
	Watch       WatchConfig   `toml:"watch"`
}

// SyncConfig controls background sync timing and filters.
type SyncConfig struct {
	IntervalSeconds    int     `toml:"interval_seconds"`
	MinDurationSeconds float64 `toml:"min_duration_seconds"`
	SubmitDelaySeconds float64 `toml:"submit_delay_seconds"`
	// LazyStart: on first run (no watermark), jump to the newest recording_id
	// without backfilling history. Default true.
	LazyStart bool `toml:"lazy_start"`
	// MaxPerSync caps how many clips are submitted in one pass (0 = unlimited).
	MaxPerSync int `toml:"max_per_sync"`
	// KeepAudio keeps extracted files after SuperWhisper submit. Default false.
	KeepAudio bool `toml:"keep_audio"`
	// KeepShards keeps downloaded parquet shards on disk. Default false.
	KeepShards bool `toml:"keep_shards"`
}

// ModesConfig maps language codes to SuperWhisper mode keys.
type ModesConfig struct {
	PL       string `toml:"pl"`
	EN       string `toml:"en"`
	ES       string `toml:"es"`
	Auto     string `toml:"auto"`
	Fallback string `toml:"fallback"`
}

// ModelConfig holds global and per-language model IDs.
type ModelConfig struct {
	VoiceModelID    string            `toml:"voice_model_id"`
	LanguageModelID string            `toml:"language_model_id"`
	PL              LangModelOverride `toml:"pl,omitempty"`
	EN              LangModelOverride `toml:"en,omitempty"`
	ES              LangModelOverride `toml:"es,omitempty"`
}

// LangModelOverride overrides voice and/or language model for one language.
type LangModelOverride struct {
	VoiceModelID    string `toml:"voice_model_id,omitempty"`
	LanguageModelID string `toml:"language_model_id,omitempty"`
}

// PromptsConfig holds per-language prompt file or inline text.
type PromptsConfig struct {
	PL   PromptSpec `toml:"pl,omitempty"`
	EN   PromptSpec `toml:"en,omitempty"`
	ES   PromptSpec `toml:"es,omitempty"`
	Auto PromptSpec `toml:"auto,omitempty"`
}

// PromptSpec references a prompt file or inline markdown text.
type PromptSpec struct {
	File string `toml:"file,omitempty"`
	Text string `toml:"text,omitempty"`
}

// PublishConfig controls post-processing and Apple Notes / Drive export.
type PublishConfig struct {
	Enabled     bool   `toml:"enabled"`
	NotesDir    string `toml:"notes_dir"`
	MemoEnabled bool   `toml:"memo_enabled"`
	MemoFolder  string `toml:"memo_folder"`
	MemoBin     string `toml:"memo_bin"`
	// Tag is appended to note bodies (e.g. "#lazy-notes") for search.
	Tag string `toml:"tag"`
	// DriveEnabled uploads each published markdown note to Google Drive via gog-cli.
	DriveEnabled  bool   `toml:"drive_enabled"`
	DriveFolderID string `toml:"drive_folder_id"`
	GogBin        string `toml:"gog_bin"`
	GogAccount    string `toml:"gog_account"`
}

// WatchConfig enables reactive daemon triggers beyond the sync interval.
type WatchConfig struct {
	// DebounceMs coalesces bursts of filesystem / Drive events (default 1500).
	DebounceMs int `toml:"debounce_ms"`

	// Apple Notes NoteStore.sqlite filesystem watch (macOS Group Container).
	AppleNotesEnabled bool   `toml:"apple_notes_enabled"`
	AppleNotesDB      string `toml:"apple_notes_db"`

	// Google Drive: local synced directory (fsnotify) and/or folder via gog-cli poll.
	DriveEnabled          bool   `toml:"drive_enabled"`
	DriveLocalDir         string `toml:"drive_local_dir"`
	DriveFolderID         string `toml:"drive_folder_id"`
	DrivePollIntervalSecs int    `toml:"drive_poll_interval_seconds"`
}

// Defaults returns the shipped default configuration.
func Defaults() Config {
	return Config{
		Dataset:     "j14i/voice-memories",
		AudioPrefer: "original",
		Languages:   []string{"pl", "en", "es"},
		Sync: SyncConfig{
			IntervalSeconds:    900,
			MinDurationSeconds: 1.0,
			SubmitDelaySeconds: 3.0,
			LazyStart:          true,
			MaxPerSync:         5,
			KeepAudio:          false,
			KeepShards:         false,
		},
		Modes: ModesConfig{
			PL:       "lazy-note-pl",
			EN:       "lazy-note-en",
			ES:       "lazy-note-es",
			Auto:     "lazy-note-auto",
			Fallback: "lazy-note-auto",
		},
		Model: ModelConfig{
			VoiceModelID:    "sw-elevenlabs-scribe",
			LanguageModelID: "claude-sonnet-4-6",
			PL: LangModelOverride{
				LanguageModelID: "claude-sonnet-4-6",
			},
			EN: LangModelOverride{
				VoiceModelID:    "nvidia_parakeet-v2_476MB",
				LanguageModelID: "gemini-3.1-flash-lite-preview",
			},
		},
		Prompts: PromptsConfig{
			PL:   PromptSpec{File: "prompts/note.pl.md"},
			EN:   PromptSpec{File: "prompts/note.en.md"},
			ES:   PromptSpec{File: "prompts/note.es.md"},
			Auto: PromptSpec{File: "prompts/note.auto.md"},
		},
		Publish: PublishConfig{
			Enabled:     true,
			NotesDir:    "~/.local/share/lazy-notes/notes",
			MemoEnabled: true,
			MemoFolder:  "Lazy Notes",
			MemoBin:     "memo",
			Tag:         "#lazy-notes",
			GogBin:      "gog",
		},
		Watch: WatchConfig{
			DebounceMs:            1500,
			AppleNotesDB:          "~/Library/Group Containers/group.com.apple.notes/NoteStore.sqlite",
			DrivePollIntervalSecs: 60,
		},
	}
}

// Load reads and parses a TOML config file. Empty path uses paths.ConfigPath().
func Load(path string) (*Config, error) {
	if path == "" {
		path = paths.ConfigPath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}

	cfg := Defaults()
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}

	cfg.applyMissingDefaults()
	return &cfg, nil
}

func (c *Config) applyMissingDefaults() {
	def := Defaults()
	if c.Dataset == "" {
		c.Dataset = def.Dataset
	}
	if c.AudioPrefer == "" {
		c.AudioPrefer = def.AudioPrefer
	}
	if len(c.Languages) == 0 {
		c.Languages = def.Languages
	}
	if c.Sync.IntervalSeconds == 0 {
		c.Sync.IntervalSeconds = def.Sync.IntervalSeconds
	}
	if c.Sync.MinDurationSeconds == 0 {
		c.Sync.MinDurationSeconds = def.Sync.MinDurationSeconds
	}
	if c.Sync.SubmitDelaySeconds == 0 {
		c.Sync.SubmitDelaySeconds = def.Sync.SubmitDelaySeconds
	}
	if c.Sync.MaxPerSync == 0 {
		c.Sync.MaxPerSync = def.Sync.MaxPerSync
	}
	if c.Modes.PL == "" {
		c.Modes.PL = def.Modes.PL
	}
	if c.Modes.EN == "" {
		c.Modes.EN = def.Modes.EN
	}
	if c.Modes.ES == "" {
		c.Modes.ES = def.Modes.ES
	}
	if c.Modes.Auto == "" {
		c.Modes.Auto = def.Modes.Auto
	}
	if c.Modes.Fallback == "" {
		c.Modes.Fallback = def.Modes.Fallback
	}
	if c.Model.VoiceModelID == "" {
		c.Model.VoiceModelID = def.Model.VoiceModelID
	}
	if c.Model.LanguageModelID == "" {
		c.Model.LanguageModelID = def.Model.LanguageModelID
	}
	if c.Prompts.PL.File == "" && c.Prompts.PL.Text == "" {
		c.Prompts.PL = def.Prompts.PL
	}
	if c.Prompts.EN.File == "" && c.Prompts.EN.Text == "" {
		c.Prompts.EN = def.Prompts.EN
	}
	if c.Prompts.ES.File == "" && c.Prompts.ES.Text == "" {
		c.Prompts.ES = def.Prompts.ES
	}
	if c.Prompts.Auto.File == "" && c.Prompts.Auto.Text == "" {
		c.Prompts.Auto = def.Prompts.Auto
	}
	if c.Publish.NotesDir == "" {
		c.Publish.NotesDir = def.Publish.NotesDir
	}
	if c.Publish.MemoFolder == "" {
		c.Publish.MemoFolder = def.Publish.MemoFolder
	}
	if c.Publish.MemoBin == "" {
		c.Publish.MemoBin = def.Publish.MemoBin
	}
	if c.Publish.Tag == "" {
		c.Publish.Tag = def.Publish.Tag
	}
	if c.Publish.GogBin == "" {
		c.Publish.GogBin = def.Publish.GogBin
	}
	if c.Watch.DebounceMs == 0 {
		c.Watch.DebounceMs = def.Watch.DebounceMs
	}
	if c.Watch.AppleNotesDB == "" {
		c.Watch.AppleNotesDB = def.Watch.AppleNotesDB
	}
	if c.Watch.DrivePollIntervalSecs == 0 {
		c.Watch.DrivePollIntervalSecs = def.Watch.DrivePollIntervalSecs
	}
}

// EnsureExample copies config.example.toml from dataDir to dest when dest is missing,
// and ensures prompts/ are present next to the config file.
func EnsureExample(dest string, dataDir string) error {
	if dest == "" {
		dest = paths.ConfigPath()
	}
	configDir := filepath.Dir(dest)
	if err := paths.EnsureDir(configDir); err != nil {
		return fmt.Errorf("ensure config dir: %w", err)
	}

	if _, err := os.Stat(dest); err == nil {
		// config exists; still seed prompts if absent
	} else if os.IsNotExist(err) {
		src := filepath.Join(dataDir, paths.ExampleConfigName())
		in, err := os.Open(src)
		if err != nil {
			return fmt.Errorf("open example config %q: %w", src, err)
		}
		defer in.Close()

		out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
		if err != nil {
			return fmt.Errorf("create config %q: %w", dest, err)
		}
		defer out.Close()

		if _, err := io.Copy(out, in); err != nil {
			return fmt.Errorf("copy example to %q: %w", dest, err)
		}
	} else {
		return fmt.Errorf("stat config %q: %w", dest, err)
	}

	return copyPrompts(dataDir, configDir)
}

func copyPrompts(dataDir, configDir string) error {
	srcDir := filepath.Join(dataDir, "prompts")
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read prompts dir %q: %w", srcDir, err)
	}
	dstDir := filepath.Join(configDir, "prompts")
	if err := paths.EnsureDir(dstDir); err != nil {
		return fmt.Errorf("ensure prompts dir: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		dst := filepath.Join(dstDir, e.Name())
		if _, err := os.Stat(dst); err == nil {
			continue
		}
		src := filepath.Join(srcDir, e.Name())
		data, err := os.ReadFile(src)
		if err != nil {
			return fmt.Errorf("read prompt %q: %w", src, err)
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return fmt.Errorf("write prompt %q: %w", dst, err)
		}
	}
	return nil
}
