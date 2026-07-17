package superwhisper

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const installScript = "curl -fsSL https://superwhisper.com/install-cli.sh | bash"

// CLIPath returns the superwhisper binary path when found.
func CLIPath() (string, bool) {
	if path, err := exec.LookPath("superwhisper"); err == nil {
		return path, true
	}
	// Homebrew cellar (lazy-notes formula installs the CLI into the same prefix).
	candidates := []string{
		"/opt/homebrew/bin/superwhisper",
		"/usr/local/bin/superwhisper",
		filepath.Join(os.Getenv("HOME"), ".local", "bin", "superwhisper"),
	}
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, true
		}
	}
	return "", false
}

// EnsureCLI installs the SuperWhisper CLI when it is not already available.
func EnsureCLI(ctx context.Context) error {
	if _, ok := CLIPath(); ok {
		return nil
	}

	cmd := exec.CommandContext(ctx, "bash", "-c", installScript)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("install superwhisper cli: %w (or: brew reinstall jborkowski/lazy-notes/lazy-notes)", err)
	}

	if _, ok := CLIPath(); !ok {
		return fmt.Errorf("superwhisper CLI not found on PATH after install")
	}
	return nil
}
