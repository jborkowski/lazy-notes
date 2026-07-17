package superwhisper

import (
	"fmt"
	"os/exec"
	"time"
)

// SetActiveMode sets SuperWhisper active and last-selected mode keys via defaults(1).
func SetActiveMode(modeKey string) error {
	for _, prefKey := range []string{"activeModeKey", "lastSelectedModeKey"} {
		cmd := exec.Command("defaults", "write", bundleID, prefKey, modeKey)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("defaults write %s: %w", prefKey, err)
		}
	}
	return nil
}

// OpenAudio opens an audio file in the SuperWhisper app.
func OpenAudio(path string) error {
	cmd := exec.Command("open", "-a", "Superwhisper", path)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("open audio in Superwhisper: %w", err)
	}
	return nil
}

// Submit selects modeKey, opens path in SuperWhisper, and waits delay between steps.
func Submit(path, modeKey string, delay time.Duration) error {
	if err := SetActiveMode(modeKey); err != nil {
		return err
	}

	half := delay / 2
	time.Sleep(half)

	if err := OpenAudio(path); err != nil {
		return err
	}

	time.Sleep(half)
	return nil
}
