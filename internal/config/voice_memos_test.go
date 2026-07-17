package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVoiceMemosDefaults(t *testing.T) {
	cfg := Defaults()
	if cfg.VoiceMemos.Enabled {
		t.Fatal("voice_memos.enabled should default false")
	}
	if cfg.VoiceMemos.Source != "export_inbox" {
		t.Fatalf("source = %q", cfg.VoiceMemos.Source)
	}
	if !cfg.VoiceMemos.WatchEnabled {
		t.Fatal("watch_enabled should default true")
	}
	if cfg.VoiceMemos.ExportDir == "" {
		t.Fatal("export_dir empty")
	}
}

func TestLoadVoiceMemosSection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	body := `
dataset = "x/y"
[voice_memos]
enabled = true
export_dir = "~/vm-inbox"
language = "en"
extensions = [".m4a", ".wav"]
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.VoiceMemos.Enabled {
		t.Fatal("expected enabled")
	}
	if cfg.VoiceMemos.Language != "en" {
		t.Fatalf("language = %q", cfg.VoiceMemos.Language)
	}
	if len(cfg.VoiceMemos.Extensions) != 2 {
		t.Fatalf("extensions = %#v", cfg.VoiceMemos.Extensions)
	}
	got := cfg.VoiceMemosExportDir()
	if !strings.Contains(got, "vm-inbox") {
		t.Fatalf("VoiceMemosExportDir = %q", got)
	}
	if strings.HasPrefix(got, "~") {
		t.Fatalf("export dir not expanded: %q", got)
	}
}
