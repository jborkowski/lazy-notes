package superwhisper

import (
	"os"
	"path/filepath"
)

const (
	superwhisperDir = "Documents/superwhisper"
	bundleID        = "com.superduper.superwhisper"
)

func home() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return h
}

// ModesDir returns ~/Documents/superwhisper/modes/.
func ModesDir() string {
	return filepath.Join(home(), superwhisperDir, "modes")
}

// SettingsPath returns ~/Documents/superwhisper/settings/settings.json.
func SettingsPath() string {
	return filepath.Join(home(), superwhisperDir, "settings", "settings.json")
}
