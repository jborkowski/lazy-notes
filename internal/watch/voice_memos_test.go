package watch

import (
	"testing"

	"github.com/jborkowski/lazy-notes/internal/config"
)

func TestOptionsFromConfigVoiceMemos(t *testing.T) {
	cfg := config.Defaults()
	opts := OptionsFromConfig(&cfg)
	if opts.VoiceMemosEnabled {
		t.Fatal("default-off: VoiceMemosEnabled should be false")
	}
	if opts.Enabled() {
		// Drive/Apple Notes also default off
	}

	cfg.VoiceMemos.Enabled = true
	cfg.VoiceMemos.WatchEnabled = true
	opts = OptionsFromConfig(&cfg)
	if !opts.VoiceMemosEnabled {
		t.Fatal("expected VoiceMemosEnabled")
	}
	if !opts.Enabled() {
		t.Fatal("Options.Enabled should be true when VM watch is on")
	}

	cfg.VoiceMemos.WatchEnabled = false
	opts = OptionsFromConfig(&cfg)
	if opts.VoiceMemosEnabled {
		t.Fatal("watch_enabled=false should disable VM watcher")
	}
}

func TestEnsureVoiceMemosInbox(t *testing.T) {
	dir := t.TempDir() + "/inbox"
	if err := EnsureVoiceMemosInbox(dir); err != nil {
		t.Fatal(err)
	}
	if err := EnsureVoiceMemosInbox(dir); err != nil {
		t.Fatal(err)
	}
}
