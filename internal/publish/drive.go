package publish

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PushDrive uploads localPath into the Drive folder identified by folderID using gog-cli.
func PushDrive(ctx context.Context, gogBin, account, folderID, localPath string) error {
	if strings.TrimSpace(folderID) == "" {
		return fmt.Errorf("drive folder id is empty")
	}
	if strings.TrimSpace(localPath) == "" {
		return fmt.Errorf("drive upload path is empty")
	}
	if st, err := os.Stat(localPath); err != nil {
		return fmt.Errorf("drive upload %q: %w", localPath, err)
	} else if st.IsDir() {
		return fmt.Errorf("drive upload %q is a directory", localPath)
	}

	bin := strings.TrimSpace(gogBin)
	if bin == "" {
		bin = "gog"
	}
	if !filepath.IsAbs(bin) {
		resolved, err := exec.LookPath(bin)
		if err != nil {
			return fmt.Errorf("gog binary %q not found on PATH (brew install gogcli): %w", bin, err)
		}
		bin = resolved
	}

	args := []string{"drive", "upload", localPath, "--parent", folderID, "--json"}
	if account = strings.TrimSpace(account); account != "" {
		args = append([]string{"--account", account}, args...)
	}

	cmd := exec.CommandContext(ctx, bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if msg == "" {
			return fmt.Errorf("gog drive upload: %w", err)
		}
		return fmt.Errorf("gog drive upload: %w: %s", err, msg)
	}
	return nil
}
